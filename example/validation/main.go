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
	v "github.com/mmorton/bunquery/v"
)

type securityCtxKey struct{}
type SecurityCtx struct {
	UserID int64
}

func NewSecurityContext(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, securityCtxKey{}, &SecurityCtx{UserID: userID})
}

func GetCurrentUserID(ctx context.Context) (int64, error) {
	if ctx == nil {
		return 0, fmt.Errorf("context is nil")
	}
	if ctxVal := ctx.Value(securityCtxKey{}); ctxVal != nil {
		if ctxVal, ok := ctxVal.(*SecurityCtx); ok {
			return ctxVal.UserID, nil
		}
	}
	return 0, fmt.Errorf("user id not found")
}

type SecurityBinder struct {
}

var _ bunquery.QueryBinder = (*SecurityBinder)(nil)

func (sb *SecurityBinder) Kind() string { return "security" }
func (sb *SecurityBinder) Bind(ctx context.Context, db bun.IDB, qry bunquery.QueryBuilderEx, args ...any) {
	currentUserID, err := GetCurrentUserID(ctx)
	if err != nil {
		qry.Err(err)
		return
	}

	qry.Where("author_id = ?", currentUserID)
}

type GetMyStoriesArgs struct {
}

var GetMyStories = bunquery.CreateQuery(func(ctx context.Context, db bunquery.QueryDB, args GetMyStoriesArgs) ([]Story, error) {
	stories := make([]Story, 0)
	if err := db.NewSelect().Model(&stories).Relation("Author").OrderExpr("story.id ASC").Scan(ctx); err != nil {
		return nil, err
	}
	return stories, nil
})

type CreateMyStoryArgs struct {
	Title string
}

var CreateMyStory = bunquery.CreateQueryMutationV(v.Args(func(set *v.Set[CreateMyStoryArgs]) {
	v.String(set, func(args CreateMyStoryArgs) string { return args.Title }).Min(1).Max(10)
}), func(ctx context.Context, db bunquery.MutationDB, args CreateMyStoryArgs) (*Story, error) {
	currentUserID, err := GetCurrentUserID(ctx)
	if err != nil {
		return nil, err
	}

	story := Story{
		Title:    args.Title,
		AuthorID: currentUserID,
	}

	if _, err := db.NewInsert().Model(&story).Exec(ctx); err != nil {
		return nil, err
	}

	return &story, nil
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

	currentUserID := int64(2)
	ctx = NewSecurityContext(ctx, currentUserID)
	ctx = bunquery.NewContext(ctx, db, &SecurityBinder{})

	if stories, err := GetMyStories(ctx, GetMyStoriesArgs{}); err != nil {
		panic(err)
	} else {
		fmt.Printf("all stories: %v\n\n", stories)
	}

	fmt.Printf("Creating a story with title: %s\n", "My Story")
	if _, err := CreateMyStory(ctx, CreateMyStoryArgs{Title: "My Story"}); err != nil {
		panic(err)
	}

	fmt.Printf("Creating a story with title: %s\n", "")
	if _, err := CreateMyStory(ctx, CreateMyStoryArgs{Title: ""}); err != nil {
		fmt.Printf("Expected error: %v\n", err)
	}

	fmt.Printf("Creating a story with title: %s\n", "")
	if _, err := CreateMyStory(ctx, CreateMyStoryArgs{Title: "This is a much longer title than is allowed."}); err != nil {
		fmt.Printf("Expected error: %v\n", err)
	}

	if stories, err := GetMyStories(ctx, GetMyStoriesArgs{}); err != nil {
		panic(err)
	} else {
		fmt.Printf("all stories: %v\n\n", stories)
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

type Story struct {
	ID       int64 `bun:",pk,autoincrement"`
	Title    string
	AuthorID int64
	Author   *User `bun:"rel:belongs-to,join:author_id=id"`
}

func resetSchema(ctx context.Context, db *bun.DB) error {
	if err := db.ResetModel(ctx, (*User)(nil), (*Story)(nil)); err != nil {
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

	stories := []Story{
		{
			Title:    "Admin's cool story 1",
			AuthorID: users[0].ID,
		},
		{
			Title:    "Admin's cool story 2",
			AuthorID: users[0].ID,
		},
		{
			Title:    "Root's cool story",
			AuthorID: users[1].ID,
		},
	}
	if _, err := db.NewInsert().Model(&stories).Exec(ctx); err != nil {
		return err
	}

	return nil
}
