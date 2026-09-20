package tests

import (
	"context"
	"github.com/mwangaben/graphqltester/pkg/adapters/database"
)

// UserResolver wraps EntUser for GraphQL.
//
// The schema's User type maps to this resolver type, not to EntUser.
//type UserResolver struct {
//	user *EntUser
//}

// ID resolves the `id` field as int32.
//func (r *UserResolver) ID() int32 {
//	return int32(r.user.ID)
//}

// Name resolves the `name` field.
//func (r *UserResolver) Name() string {
//	return r.user.Name
//}

// Email resolves the `email` field.
//func (r *UserResolver) Email() string {
//	return r.user.Email
//}
//
//// Age resolves the `age` field as int32.
//func (r *UserResolver) Age() int32 {
//	return int32(r.user.Age)
//}
//
//// Role resolves the `role` field.
//func (r *UserResolver) Role() string {
//	return r.user.Role
//}

// ============================================================
// Resolver for GraphQL queries
// ============================================================

type TestResolver struct {
	adapter *database.EntAdapter
}

// User resolves the `user` query.
func (r *TestResolver) User(ctx context.Context, args struct{ ID int32 }) (*UserResolver, error) {
	record, err := r.adapter.GetRecord(ctx, "ent_factory_users", map[string]interface{}{
		"id": args.ID,
	})
	if err != nil {
		return nil, err
	}

	// Convert the DB record into an EntUser
	entUser := &EntUser{
		Name:  toString(record["name"]),
		Email: toString(record["email"]),
		Role:  toString(record["role"]),
	}

	// IDs come back as int64 from MySQL
	if v, ok := record["id"].(int64); ok {
		entUser.ID = int(v)
	}
	if v, ok := record["age"].(int64); ok {
		entUser.Age = int(v)
	}

	return &UserResolver{user: entUser}, nil
}

// toString safely converts interface{} to string.
func toString(v interface{}) string {
	switch s := v.(type) {
	case string:
		return s
	case []byte:
		return string(s)
	default:
		return ""
	}
}
