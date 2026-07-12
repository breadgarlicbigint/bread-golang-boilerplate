# ID Strategy Migration Guide

The project supports two MongoDB `_id` strategies. This document covers how to
switch between them and the trade-offs of each.

---

## Strategies

| Strategy | Type | BSON | JSON | Size | Sortable |
|---|---|---|---|---|---|
| `uuid` (default) | `uuid.UUID` | Binary subtype 4 | `"550e8400-e29b-..."` | 16 bytes | No (random) |
| `objectid` | `primitive.ObjectID` | ObjectID | `"6507c3a1..."` | 12 bytes | Yes (time-prefixed) |

**Choose UUID when:**
- You generate IDs at the application layer before inserting
- You need IDs to be unpredictable (security)
- You want readability in logs and API responses

**Choose ObjectID when:**
- You want time-sortable IDs out of the box
- You prefer smaller storage (4 bytes smaller per ID)
- You are migrating from an existing MongoDB-native app

---

## Current default: UUID

Set in `.env`:
```env
DB_ID_TYPE=uuid
```

Entity field type:
```go
type User struct {
    ID     uuid.UUID `bson:"_id" json:"id"`
    RoleID uuid.UUID `bson:"roleId" json:"roleId"`
}
```

Repository generate:
```go
u := &entity.User{ID: uuid.New(), ...}
```

Repository query:
```go
uid, err := uuid.Parse(idString)
filter := bson.M{"_id": uid}
```

---

## Switching to ObjectID

### Step 1 — Change `.env`
```env
DB_ID_TYPE=objectid
```

### Step 2 — Update entity field types
Change every `uuid.UUID` field used as an `_id` or foreign key reference:

```go
// Before
type User struct {
    ID     uuid.UUID `bson:"_id"    json:"id"`
    RoleID uuid.UUID `bson:"roleId" json:"roleId"`
}

// After
type User struct {
    ID     primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    RoleID primitive.ObjectID `bson:"roleId"        json:"roleId"`
}
```

### Step 3 — Update repository generate / parse
```go
// Before (uuid)
u := &entity.User{ID: dbid.NewUUID()}
uid, err := dbid.ParseUUID(c.Param("id"))

// After (objectid)
u := &entity.User{ID: dbid.NewObjectID()}
oid, err := dbid.ParseObjectID(c.Param("id"))
```

### Step 4 — Update JWT helper calls
In `modules/auth/service/auth.service.go`, `issueTokenPair` uses `u.ID.String()`.
For ObjectID:
```go
// uuid.UUID
u.ID.String()              // "550e8400-e29b-..."

// primitive.ObjectID
u.ID.Hex()                 // "6507c3a1f4a3b2c1d0e9f8a7"
```

### Step 5 — Migrate existing data (if any)
```js
// MongoDB shell migration script
// Run ONCE on your existing collection
db.users.find({}).forEach(doc => {
    const newId = UUID();
    db.users.insertOne({...doc, _id: newId});
    db.users.deleteOne({_id: doc._id});
});
```

> ⚠️ Always take a backup before running a migration.

---

## Files that need updating when switching

| File | What to change |
|---|---|
| `modules/*/entity/*.go` | `uuid.UUID` → `primitive.ObjectID` on `_id` and FK fields |
| `modules/*/repository/*.go` | `uuid.New()` → `primitive.NewObjectID()`, `uuid.Parse()` → `primitive.ObjectIDFromHex()` |
| `modules/*/handler/*.go` | `uuid.Parse(c.Param("id"))` → `primitive.ObjectIDFromHex(c.Param("id"))` |
| `modules/auth/service/auth.service.go` | `.String()` → `.Hex()` in `issueTokenPair` |
| `shared/middleware/auth.go` | `mustUserID` helper (if it calls `uuid.Parse`) |
| `scripts/seed/*.go` | `uuid.New()` → `primitive.NewObjectID()` (mainly `role.go`, `user.go`, `featureflag.go`, `appversion.go`) |

---

## BSON codec registration

`pkg/dbid/dbid.go` → `NewRegistry(strategy string)` returns the right BSON codec:
- `"uuid"` → registers the UUID Binary-4 codec from `pkg/uuidbson`
- `"objectid"` or `""` → returns MongoDB's default codec (ObjectID native)

The registry is applied in `shared/database/mongo.go`:
```go
opts := options.MergeClientOptions(
    options.Client().ApplyURI(cfg.URI),
    options.Client().SetRegistry(dbid.NewRegistry(cfg.IDType)),
)
```
