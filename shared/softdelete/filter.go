// Package softdelete holds the one expression of "live rows" versus "removed
// rows" that every soft-deleting repository filters on.
//
// A leaf package with no dependency beyond bson: the clause is needed by many
// repositories, and hand-written copies are where the reveal flag gets honoured
// on the list and quietly forgotten on the count beside it.
package softdelete

import "go.mongodb.org/mongo-driver/bson"

// Clause returns the `deletedAt` filter value for a list read.
//
// deletedOnly=false is the default everywhere and yields exactly the `nil` a
// caller wrote by hand before — so an untouched caller behaves identically.
// true swaps the read to the removed rows ALONE (never a union: a mixed table
// makes the pager total mean two things at once).
func Clause(deletedOnly bool) interface{} {
	if deletedOnly {
		return bson.M{"$ne": nil}
	}
	return nil
}

// Active is Clause for collections that express soft delete as `active: false`
// rather than a `deletedAt` stamp. Same contract, opposite polarity.
func Active(deletedOnly bool) bool { return !deletedOnly }
