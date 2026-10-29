package pagination

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// GroupedFind is the one implementation of a grouped page for every Mongo-backed
// list endpoint. Modules supply their own
// `$match` and whitelist; everything below is identical across them.
//
// Paging is over RAW RECORDS, exactly as when ungrouped: `page`/`perPage` window
// the row set and the returned total is the record count. A group therefore
// spans page boundaries and the client repeats its header.

// groupedDoc wraps the original document alongside its group keys. The level
// fields are fixed at MaxGroupDepth rather than dynamic because a BSON struct
// tag cannot be computed — three named fields is the price of decoding straight
// into T with the collection's own registry (which is what keeps UUID fields
// working; a re-marshal through the default registry would corrupt them).
type groupedDoc[T any] struct {
	Doc T   `bson:"doc"`
	L0  any `bson:"l0"`
	L1  any `bson:"l1"`
	L2  any `bson:"l2"`
}

func (g groupedDoc[T]) level(i int) any {
	switch i {
	case 0:
		return g.L0
	case 1:
		return g.L1
	default:
		return g.L2
	}
}

// GroupedFind returns one page of records ordered so each group is contiguous
// and tagged with the group it belongs to, plus the full size of every group
// present in the window.
//
// `defaultSort` is the order applied to rows within a group when the query
// carries no `sortBy`. It is a full bson.E (key AND direction), not just a
// field, because some lists default to descending — some lists are
// newest-first — and defaulting everything to ascending would silently reverse
// them the moment grouping was switched on.
func GroupedFind[T any](
	ctx context.Context,
	col *mongo.Collection,
	filter bson.M,
	levels []GroupField,
	q Query,
	defaultSort bson.E,
) ([]GroupedRow, int64, []GroupCount, error) {
	if len(levels) == 0 {
		return nil, 0, nil, ErrInvalidGroupBy
	}
	if len(levels) > MaxGroupDepth {
		return nil, 0, nil, ErrInvalidGroupBy
	}

	total, err := col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, nil, err
	}

	// Group fields lead the sort, ALWAYS ascending. This is what makes a group
	// contiguous in the window; without it one group's rows scatter across the
	// page and there is nothing to render a header over. The caller's own sort
	// then orders rows *within* a group, which is why group order is
	// deliberately independent of sortDir.
	sort := bson.D{}
	for _, lv := range levels {
		sort = append(sort, bson.E{Key: lv.Path, Value: 1})
	}
	rowSort := defaultSort
	if q.SortBy != "" {
		dir := 1
		if q.SortDir == "desc" {
			dir = -1
		}
		rowSort = bson.E{Key: q.SortBy, Value: dir}
	}
	sort = append(sort, rowSort)

	// $replaceRoot (not $replaceWith — wider server support) lifts the whole
	// document under `doc` and projects each level's value beside it, so the row
	// and its group path arrive in ONE query rather than two passes over the
	// same window.
	newRoot := bson.M{"doc": "$$ROOT"}
	for i, lv := range levels {
		newRoot[fmt.Sprintf("l%d", i)] = "$" + lv.Path
	}
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: filter}},
		bson.D{{Key: "$sort", Value: sort}},
		bson.D{{Key: "$skip", Value: q.Skip()}},
		bson.D{{Key: "$limit", Value: q.Limit()}},
		bson.D{{Key: "$replaceRoot", Value: bson.M{"newRoot": newRoot}}},
	}
	cur, err := col.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, nil, err
	}
	defer cur.Close(ctx)
	var wrapped []groupedDoc[T]
	if err := cur.All(ctx, &wrapped); err != nil {
		return nil, 0, nil, err
	}

	rows := make([]GroupedRow, 0, len(wrapped))
	topKeys := make([]any, 0, len(wrapped))
	seenTop := map[string]struct{}{}
	for _, w := range wrapped {
		path := make([]any, 0, len(levels))
		for i, lv := range levels {
			path = append(path, DecodeGroupKey(w.level(i), lv.Kind))
		}
		rows = append(rows, GroupedRow{GroupPath: path, Row: w.Doc})

		// Raw (undecoded) top-level value: it is matched back against the
		// collection below, where the stored BSON type is what counts.
		raw := w.level(0)
		if k := fmt.Sprint(raw); true {
			if _, dup := seenTop[k]; !dup {
				seenTop[k] = struct{}{}
				topKeys = append(topKeys, raw)
			}
		}
	}

	counts, err := groupCounts(ctx, col, filter, levels, topKeys)
	if err != nil {
		return nil, 0, nil, err
	}
	return rows, total, counts, nil
}

// groupCounts returns the FULL size of every group and sub-group reachable from
// the rows in the current window — full, so a header can read "(12 items)" while
// showing 10.
//
// Restricted to the top-level keys actually returned, which bounds both the work
// and the response by perPage: grouping a high-cardinality column must not ship
// counts for groups the page never shows.
func groupCounts(
	ctx context.Context, col *mongo.Collection, filter bson.M, levels []GroupField, topKeys []any,
) ([]GroupCount, error) {
	out := make([]GroupCount, 0)
	if len(topKeys) == 0 {
		return out, nil
	}

	scoped := bson.M{}
	for k, v := range filter {
		scoped[k] = v
	}
	// APPEND to any existing `$and`, never replace it. Some modules express their
	// whole filter as one (a module's filter builder wraps scope visibility AND
	// the search term that way), so assigning here silently dropped tenant scope
	// and search from the count — every group then reported a total spanning
	// every tenant, which looks like a plausible number and is not.
	//
	// A separate clause rather than a direct `scoped[path]` for the same reason:
	// the level's path may already be constrained by the module's own filter (a
	// status column that is both a filter param and a group level). A nil among
	// the values matches null and missing alike — the "no value" group.
	existing, _ := scoped["$and"].(bson.A)
	clauses := make(bson.A, 0, len(existing)+1)
	clauses = append(clauses, existing...)
	clauses = append(clauses, bson.M{levels[0].Path: bson.M{"$in": topKeys}})
	scoped["$and"] = clauses

	id := bson.M{}
	for i, lv := range levels {
		id[fmt.Sprintf("l%d", i)] = "$" + lv.Path
	}
	cur, err := col.Aggregate(ctx, mongo.Pipeline{
		bson.D{{Key: "$match", Value: scoped}},
		bson.D{{Key: "$group", Value: bson.M{"_id": id, "count": bson.M{"$sum": 1}}}},
	})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []bson.M
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}

	// One aggregation yields leaf counts; every shorter prefix is a roll-up of
	// those, summed here rather than costing another round trip per level.
	totals := map[string]int64{}
	paths := map[string][]any{}
	for _, d := range docs {
		key, _ := d["_id"].(bson.M)
		n := GroupCountOf(d["count"])
		full := make([]any, 0, len(levels))
		for i, lv := range levels {
			full = append(full, DecodeGroupKey(key[fmt.Sprintf("l%d", i)], lv.Kind))
		}
		for depth := 1; depth <= len(full); depth++ {
			prefix := full[:depth]
			pid := fmt.Sprint(prefix)
			totals[pid] += n
			paths[pid] = prefix
		}
	}
	for pid, p := range paths {
		out = append(out, GroupCount{Path: p, Count: totals[pid]})
	}
	return out, nil
}
