package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/samber/lo"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/uptrace/bun/extra/bundebug"

	"github.com/mmorton/bunquery"
)

var GetUser = bunquery.CreateQuery(bunquery.Query[int64, *User]{
	Handler: func(ctx context.Context, db bunquery.QueryDB, id int64) (*User, error) {
		user := new(User)
		if err := db.NewSelect().Model(user).Where("id = ?", id).Scan(ctx); err != nil {
			return nil, err
		}
		return user, nil
	},
})

var UpdateUser = bunquery.CreateMutation(bunquery.Mutation[*UserPatch]{
	Handler: func(ctx context.Context, db bunquery.MutationDB, patch *UserPatch) error {
		if _, err := db.NewUpdate().Apply(patch.Compile()).Exec(ctx); err != nil {
			return err
		}
		return nil
	},
})

func main() {
	ctx := context.Background()

	sqlite, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	if err != nil {
		panic(err)
	}
	sqlite.SetMaxOpenConns(1)

	db := bun.NewDB(sqlite, sqlitedialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(true),
		bundebug.FromEnv("BUNDEBUG"),
	))

	if err := resetSchema(ctx, db); err != nil {
		panic(err)
	}

	ctx = bunquery.NewContext(ctx, db)

	// Select one user by primary key.
	user, err := GetUser(ctx, 1)
	if err != nil {
		panic(err)
	} else {
		fmt.Printf("user1: %v\n\n", user)
	}

	patch := NewUserPatch(user)
	patch.Name = lo.ToPtr("NEW NAME")

	if err := UpdateUser(ctx, patch); err != nil {
		panic(err)
	} else if user, err := GetUser(ctx, 1); err != nil {
		panic(err)
	} else {
		fmt.Printf("user1: %v\n\n", user)
	}
}

type User struct {
	ID     int64 `bun:",pk,autoincrement"`
	Name   string
	Emails []string
}

func (u User) String() string {
	return fmt.Sprintf("User<%d %s %v>", u.ID, u.Name, u.Emails)
}

type UserPatch struct {
	bunquery.Patch[User, UserPatch]

	Name   *string
	Emails *[]string
}

func NewUserPatch(user *User) *UserPatch {
	res := &UserPatch{}
	res.Patch = bunquery.CreatePatch(user, res)
	return res
}

func resetSchema(ctx context.Context, db *bun.DB) error {
	if err := db.ResetModel(ctx, (*User)(nil)); err != nil {
		return err
	}

	users := []User{
		{
			Name:   "admin",
			Emails: []string{"admin1@admin", "admin2@admin"},
		},
		{
			Name:   "root",
			Emails: []string{"root1@root", "root2@root"},
		},
	}
	if _, err := db.NewInsert().Model(&users).Exec(ctx); err != nil {
		return err
	}

	return nil
}
