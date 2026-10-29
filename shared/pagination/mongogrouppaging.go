package pagination

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// The Mongo half of the group-paging mode. GroupedFind's counterpart: same filter, same
// whitelist, same ordering — a different window.
//
// Two aggregations per page, the same order of magnitude as the grouping contract's three:
//
//  1. one `$group` over the filtered set for the whole group tree (bounded, since
//     group keys are a whitelist of low-cardinality columns), and
//  2. one row read that is the grouping contract's pipeline plus an `$or` restricting it to the
//     leaf groups that are open.
//
// There is deliberately NO per-group query. an earlier version fetched a group's rows on
// demand and that design is not coming back; the planner exists precisely so the
// page's rows are one contiguous slice of the open-group row set.

// EncodeGroupKey is DecodeGroupKey's inverse: it turns a group key as it travels
// on the wire back into the value the collection actually stores, so it can be
// matched. Only UUIDs need real work — they are stored as BSON Binary subtype 4
// (pkg/uuidbson) and arrive as canonical text.
//
// A nil key matches null AND a missing field, which is exactly the "no value"
// group Mongo's own ascending sort puts first.
func EncodeGroupKey(v any, kind GroupKind) any {
	if v == nil {
		return nil
	}
	switch kind {
	case GroupUUID:
		s, ok := v.(string)
		if !ok {
			return nil
		}
		id, err := uuid.Parse(s)
		if err != nil {
			return nil
		}
		return id
	case GroupBool:
		b, ok := v.(bool)
		if !ok {
			return nil
		}
		return b
	default:
		s, ok := v.(string)
		if !ok {
			return nil
		}
		return s
	}
}

// WithClause appends one clause to a filter's `$and`, creating it if needed.
//
// APPEND, never assign. Some modules express their whole filter as a single
// `$and` (master data wraps scope visibility and the search term that way), so
// writing `filter["$and"] = …` silently drops tenant scope — which produced
// counts spanning every tenant, a plausible-looking number that is wrong.
// Likewise a direct `filter[path] = value` can collide with a constraint the
// module already placed on that same path (a status column that is both a filter
// param and a group level).
func WithClause(filter bson.M, clause bson.M) bson.M {
	out := bson.M{}
	for k, v := range filter {
		out[k] = v
	}
	existing, _ := out["$and"].(bson.A)
	clauses := make(bson.A, 0, len(existing)+1)
	clauses = append(clauses, existing...)
	clauses = append(clauses, clause)
	out["$and"] = clauses
	return out
}

// OpenLeafClause restricts a row read to the leaf groups a plan says are open.
// Returns nil when everything is open (no restriction needed) — see
// WindowPlan.AllOpen for why that case is not merely an optimisation.
func OpenLeafClause(levels []GroupField, plan WindowPlan) bson.M {
	if plan.AllOpen {
		return nil
	}
	clauses := make(bson.A, 0, len(plan.OpenLeaves))
	for _, path := range plan.OpenLeaves {
		m := bson.M{}
		for i, lv := range levels {
			if i >= len(path) {
				break
			}
			m[lv.Path] = EncodeGroupKey(path[i], lv.Kind)
		}
		clauses = append(clauses, m)
	}
	if len(clauses) == 0 {
		// Nothing is open: match nothing. `$or: []` is an error in Mongo, and
		// omitting the clause would return every row — the loudest possible
		// version of this bug, since a page of closed headers would come back
		// full of rows.
		return bson.M{"_id": bson.M{"$in": bson.A{}}}
	}
	return bson.M{"$or": clauses}
}

// GroupSort is the ordering every grouped read shares: group fields first and
// ALWAYS ascending, so a group is contiguous, then the caller's own sort within
// the group. Group order is deliberately independent of sortDir.
func GroupSort(levels []GroupField, q Query, defaultSort bson.E) bson.D {
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
	return append(sort, rowSort)
}

// GroupLeavesFromDocs decodes the `$group` stage's output into the planner's
// input. Shared with the three endpoints that cannot use GroupPagedFind, so the
// key decoding and the roll-up cannot drift between them.
func GroupLeavesFromDocs(docs []bson.M, levels []GroupField, summary GroupSummary) []GroupLeaf {
	out := make([]GroupLeaf, 0, len(docs))
	for _, d := range docs {
		key, _ := d["_id"].(bson.M)
		path := make([]any, 0, len(levels))
		for i, lv := range levels {
			path = append(path, DecodeGroupKey(key[fmt.Sprintf("l%d", i)], lv.Kind))
		}
		leaf := GroupLeaf{Path: path, Count: GroupCountOf(d["count"])}
		if summary.Declared() {
			leaf.Summary = GroupCountOf(d["sum"])
		}
		out = append(out, leaf)
	}
	return out
}

// GroupLeafStage builds the `$group` stage that produces one row per distinct
// full group path, with its record count and — where the endpoint declares one —
// its summed money. One stage, so the summary costs nothing beyond the count
// aggregation that had to run anyway.
func GroupLeafStage(levels []GroupField, summary GroupSummary) bson.D {
	id := bson.M{}
	for i, lv := range levels {
		id[fmt.Sprintf("l%d", i)] = "$" + lv.Path
	}
	group := bson.M{"_id": id, "count": bson.M{"$sum": 1}}
	if summary.Declared() {
		group["sum"] = bson.M{"$sum": "$" + summary.Path}
	}
	return bson.D{{Key: "$group", Value: group}}
}

// GroupPagedFind returns one page of the VISIBLE sequence: group headers at
// every level interleaved with the rows of open groups, windowed by
// page/perPage over that sequence rather than over records.
//
// `total` is still the RECORD count and is returned separately from
// `visibleTotal`, which is what the pager counts. They are two numbers and never
// one.
func GroupPagedFind[T any](
	ctx context.Context,
	col *mongo.Collection,
	filter bson.M,
	levels []GroupField,
	summary GroupSummary,
	q Query,
	defaultSort bson.E,
) (items []any, total int64, visibleTotal int64, err error) {
	if len(levels) == 0 || len(levels) > MaxGroupDepth {
		return nil, 0, 0, ErrInvalidGroupBy
	}

	if total, err = col.CountDocuments(ctx, filter); err != nil {
		return nil, 0, 0, err
	}

	leafDocs, err := aggregateM(ctx, col, mongo.Pipeline{
		bson.D{{Key: "$match", Value: filter}},
		GroupLeafStage(levels, summary),
	})
	if err != nil {
		return nil, 0, 0, err
	}

	tree := BuildGroupTree(GroupLeavesFromDocs(leafDocs, levels, summary), len(levels))
	plan := PlanWindow(tree, len(levels), NewExpansionSet(q), summary.Declared(), q)

	var rows []T
	if plan.RowLimit > 0 {
		rowFilter := filter
		if clause := OpenLeafClause(levels, plan); clause != nil {
			rowFilter = WithClause(filter, clause)
		}
		newRoot := bson.M{"doc": "$$ROOT"}
		for i, lv := range levels {
			newRoot[fmt.Sprintf("l%d", i)] = "$" + lv.Path
		}
		cur, aggErr := col.Aggregate(ctx, mongo.Pipeline{
			bson.D{{Key: "$match", Value: rowFilter}},
			bson.D{{Key: "$sort", Value: GroupSort(levels, q, defaultSort)}},
			bson.D{{Key: "$skip", Value: plan.RowSkip}},
			bson.D{{Key: "$limit", Value: plan.RowLimit}},
			bson.D{{Key: "$replaceRoot", Value: bson.M{"newRoot": newRoot}}},
		})
		if aggErr != nil {
			return nil, 0, 0, aggErr
		}
		defer cur.Close(ctx)
		var wrapped []groupedDoc[T]
		if err = cur.All(ctx, &wrapped); err != nil {
			return nil, 0, 0, err
		}
		rows = make([]T, 0, len(wrapped))
		for _, w := range wrapped {
			rows = append(rows, w.Doc)
		}
	}

	return AssembleWindow(plan, rows), total, plan.VisibleTotal, nil
}

// AssembleWindow splices the fetched rows into the plan's slots, in order. It is
// generic over the row type and shared with the three bespoke endpoints, so the
// one thing that would silently mis-pair a row with a header — the order — has a
// single implementation.
//
// A short row slice (a row deleted between the two aggregations) leaves the
// remaining slots out rather than pairing a row with the wrong header.
func AssembleWindow[T any](plan WindowPlan, rows []T) []any {
	out := make([]any, 0, len(plan.Slots))
	next := 0
	for _, slot := range plan.Slots {
		if slot.Header != nil {
			out = append(out, slot.Header)
			continue
		}
		if next >= len(rows) {
			continue
		}
		out = append(out, GroupPagedRow{Kind: "row", GroupPath: slot.GroupPath, Row: rows[next]})
		next++
	}
	return out
}

// aggregateM runs a pipeline and decodes into bson.M — which is mandatory, not a
// convenience: the application's BSON registry (pkg/uuidbson.NewRegistry)
// registers the default codecs but not bsoncodec's primitive ones, so neither
// bson.Raw nor bson.RawValue has a decoder here.
func aggregateM(ctx context.Context, col *mongo.Collection, pipeline mongo.Pipeline) ([]bson.M, error) {
	cur, err := col.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []bson.M
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}
