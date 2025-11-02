package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/uptrace/bun/extra/bundebug"

	"github.com/mmorton/bunquery"
)

type GetUsersArgs struct {
	Limit int
}

var GetUsers = bunquery.CreateQuery(bunquery.Query[*GetUsersArgs, []*User]{
	Handler: func(ctx context.Context, db bunquery.QueryDB, args *GetUsersArgs) ([]*User, error) {
		users := make([]*User, args.Limit)
		if err := db.NewSelect().Model(&users).OrderExpr("id ASC").Limit(args.Limit).Scan(ctx); err != nil {
			return nil, err
		}
		return users, nil
	},
})

func getUserByID(ctx context.Context, db bunquery.QueryCommon, id int64) (*User, error) {
	user := new(User)
	if err := db.NewSelect().Model(user).Where("id = ?", id).Scan(ctx); err != nil {
		return nil, err
	}
	return user, nil
}

type GetUserArgs struct {
	ID int64
}

var GetUser = bunquery.CreateQuery(bunquery.Query[*GetUserArgs, *User]{
	Handler: func(ctx context.Context, db bunquery.QueryDB, args *GetUserArgs) (*User, error) {
		user := new(User)
		if err := db.NewSelect().Model(user).Where("id = ?", args.ID).Scan(ctx); err != nil {
			return nil, err
		}
		return user, nil
	},
})

type UpdateUserArgs struct {
	ID   int64
	Name string
}

var UpdateUser = bunquery.CreateMutation(bunquery.Mutation[*UpdateUserArgs]{
	Handler: func(ctx context.Context, db bunquery.MutationDB, args *UpdateUserArgs) error {
		user := new(User)
		if err := db.NewSelect().Model(user).Where("id = ?", args.ID).Scan(ctx); err != nil {
			return err
		}
		user.Name = args.Name
		if _, err := db.NewUpdate().Model(user).WherePK().Exec(ctx); err != nil {
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

	// Select all users.
	if users, err := GetUsers(ctx, &GetUsersArgs{Limit: 10}); err != nil {
		panic(err)
	} else {
		fmt.Printf("all users: %v\n\n", users)
	}

	// Select one user by primary key.
	if user, err := GetUser(ctx, &GetUserArgs{ID: 1}); err != nil {
		panic(err)
	} else {
		fmt.Printf("user1: %v\n\n", user)
	}

	if err := UpdateUser(ctx, &UpdateUserArgs{ID: 1, Name: "admin_new_name"}); err != nil {
		panic(err)
	} else if user, err := GetUser(ctx, &GetUserArgs{ID: 1}); err != nil {
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
