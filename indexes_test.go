package hades_test

import (
	"context"
	"testing"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"github.com/itchio/hades"
	"github.com/stretchr/testify/assert"
)

func indexNames(t *testing.T, c *hades.Context, conn *sqlite.Conn, tableName string) []string {
	var names []string
	ordie(c.ExecRaw(conn,
		"SELECT name FROM sqlite_master WHERE type = 'index' AND tbl_name = ? AND name NOT LIKE 'sqlite_%' ORDER BY name",
		func(stmt *sqlite.Stmt) error {
			names = append(names, stmt.ColumnText(0))
			return nil
		},
		tableName,
	))
	return names
}

func Test_DeclareIndexValidation(t *testing.T) {
	type Article struct {
		ID       int64
		AuthorID int64
		Title    string
	}
	type Unregistered struct {
		ID int64
	}

	c, err := hades.NewContext(&Article{})
	ordie(err)

	assert.NoError(t, c.DeclareIndex(&Article{}, "author_id"))
	assert.NoError(t, c.DeclareIndex(&Article{}, "author_id", "title"))

	// value models are accepted wherever NewContext accepts them
	assert.NoError(t, c.DeclareIndex(Article{}, "title"), "value model")

	// only the type is consulted, so a typed nil pointer is a valid model
	// descriptor and must not panic
	var nilArticle *Article
	assert.NoError(t, c.DeclareIndex(nilArticle, "author_id"), "typed nil pointer")
	assert.Error(t, c.DeclareIndex(nil, "author_id"), "untyped nil")

	assert.Error(t, c.DeclareIndex(&Unregistered{}, "id"), "unregistered model")
	assert.Error(t, c.DeclareIndex(&Article{}), "no columns")
	assert.Error(t, c.DeclareIndex(&Article{}, "nonexistent"), "unknown column")
	assert.Error(t, c.DeclareIndex(&Article{}, "author_id", "author_id"), "duplicate column")

	// a distinct type mapping to the same table name must not pass
	// registration, or its columns would be validated against the wrong
	// struct and CREATE INDEX would fail at AutoMigrate time
	shadow := func() any {
		type Article struct {
			ID       int64
			Impostor string
		}
		return &Article{}
	}()
	assert.Error(t, c.DeclareIndex(shadow, "impostor"), "shadow type with same table name")
}

func Test_DeclareIndexClonesColumns(t *testing.T) {
	dbpool, err := sqlitex.Open("file:memory:?mode=memory", 0, 10)
	ordie(err)
	defer dbpool.Close()

	conn := dbpool.Get(context.Background())
	defer dbpool.Put(conn)

	type Article struct {
		ID       int64
		AuthorID int64
	}

	c, err := hades.NewContext(&Article{})
	ordie(err)
	c.Logger = testLogger(t)

	// mutating the caller's slice after declaring must not affect the
	// declaration that passed validation
	cols := []string{"author_id"}
	ordie(c.DeclareIndex(&Article{}, cols...))
	cols[0] = "missing"

	ordie(c.AutoMigrate(conn))
	assert.EqualValues(t, []string{"hades_idx_articles__author_id"}, indexNames(t, c, conn, "articles"))
}

// A failed AutoMigrate run must not strand previously-synced tables without
// their indexes: table rebuilds drop indexes, re-creation happens at the end
// of the run, and map iteration order means any table may sync first. The
// transaction enclosing the whole run rolls everything back instead.
func Test_AutoMigrateRollsBackOnFailure(t *testing.T) {
	dbpool, err := sqlitex.Open("file:memory:?mode=memory", 0, 10)
	ordie(err)
	defer dbpool.Close()

	conn := dbpool.Get(context.Background())
	defer dbpool.Put(conn)

	{
		type Article struct {
			ID       int64
			AuthorID int64
		}
		c, err := hades.NewContext(&Article{})
		ordie(err)
		c.Logger = testLogger(t)
		ordie(c.DeclareIndex(&Article{}, "author_id"))
		ordie(c.AutoMigrate(conn))
		assert.EqualValues(t, []string{"hades_idx_articles__author_id"}, indexNames(t, c, conn, "articles"))
	}

	// second run: articles gains a column (forcing a rebuild that drops its
	// index), but another model fails to sync; everything must roll back
	{
		type Article struct {
			ID       int64
			AuthorID int64
			Title    string
		}
		type Bad struct {
			ID int64
			Fn func() // unsupported column type: sync fails
		}
		c, err := hades.NewContext(&Article{}, &Bad{})
		ordie(err)
		c.Logger = testLogger(t)
		ordie(c.DeclareIndex(&Article{}, "author_id"))

		assert.Error(t, c.AutoMigrateEx(conn, &hades.AutoMigrateStats{}))

		assert.EqualValues(t, []string{"hades_idx_articles__author_id"},
			indexNames(t, c, conn, "articles"), "index survives failed run")
		pti, err := c.PragmaTableInfo(conn, "articles")
		ordie(err)
		assert.Len(t, pti, 2, "rebuild rolled back: no title column")
	}

	// a subsequent successful run converges normally
	{
		type Article struct {
			ID       int64
			AuthorID int64
			Title    string
		}
		c, err := hades.NewContext(&Article{})
		ordie(err)
		c.Logger = testLogger(t)
		ordie(c.DeclareIndex(&Article{}, "author_id"))
		ordie(c.AutoMigrate(conn))

		assert.EqualValues(t, []string{"hades_idx_articles__author_id"}, indexNames(t, c, conn, "articles"))
		pti, err := c.PragmaTableInfo(conn, "articles")
		ordie(err)
		assert.Len(t, pti, 3, "title column added")
	}
}

// index names read from sqlite_master are interpolated into DROP INDEX
// and must be escaped, or a hand-created name with special characters makes
// every AutoMigrate fail
func Test_AutoMigratePrunesQuotedNames(t *testing.T) {
	dbpool, err := sqlitex.Open("file:memory:?mode=memory", 0, 10)
	ordie(err)
	defer dbpool.Close()

	conn := dbpool.Get(context.Background())
	defer dbpool.Put(conn)

	type Article struct {
		ID       int64
		AuthorID int64
	}

	c, err := hades.NewContext(&Article{})
	ordie(err)
	c.Logger = testLogger(t)
	ordie(c.AutoMigrate(conn))
	ordie(c.ExecRaw(conn, `CREATE INDEX "hades_idx_articles__author id; --" ON articles (author_id)`, nil))

	stats := &hades.AutoMigrateStats{}
	ordie(c.AutoMigrateEx(conn, stats))
	assert.EqualValues(t, 1, stats.NumIndexesDropped, "quoted-name index pruned, not a syntax error")
	assert.Empty(t, indexNames(t, c, conn, "articles"))
}

func Test_DeclareIndexNameCollision(t *testing.T) {
	// "__" is both the name separator and a legal identifier substring, so
	// (a__b) and (a, b) encode to the same index name; the second
	// declaration must be rejected instead of silently ignored
	type Weird struct {
		ID   int64
		A__B string
		A    string
		B    string
	}

	c, err := hades.NewContext(&Weird{})
	ordie(err)

	assert.NoError(t, c.DeclareIndex(&Weird{}, "a__b"))
	assert.NoError(t, c.DeclareIndex(&Weird{}, "a__b"), "identical duplicate declaration is harmless")
	assert.Error(t, c.DeclareIndex(&Weird{}, "a", "b"), "same name, different definition")
}

func Test_AutoMigrateIndexes(t *testing.T) {
	dbpool, err := sqlitex.Open("file:memory:?mode=memory", 0, 10)
	ordie(err)
	defer dbpool.Close()

	conn := dbpool.Get(context.Background())
	defer dbpool.Put(conn)

	type Article struct {
		ID       int64
		AuthorID int64
	}

	// fresh database: declared index is created
	{
		c, err := hades.NewContext(&Article{})
		ordie(err)
		c.Logger = testLogger(t)
		ordie(c.DeclareIndex(&Article{}, "author_id"))

		stats := &hades.AutoMigrateStats{}
		ordie(c.AutoMigrateEx(conn, stats))
		assert.EqualValues(t, 1, stats.NumIndexesCreated)
		assert.EqualValues(t, []string{"hades_idx_articles__author_id"}, indexNames(t, c, conn, "articles"))

		// second run is a no-op
		stats = &hades.AutoMigrateStats{}
		ordie(c.AutoMigrateEx(conn, stats))
		assert.EqualValues(t, 0, stats.NumIndexesCreated)
		assert.EqualValues(t, 0, stats.NumIndexesDropped)
	}

	// adding a column triggers a table rebuild, which drops all indexes;
	// the same AutoMigrate run must re-create the declared index
	{
		type Article struct {
			ID       int64
			AuthorID int64
			Title    string
		}

		c, err := hades.NewContext(&Article{})
		ordie(err)
		c.Logger = testLogger(t)
		ordie(c.DeclareIndex(&Article{}, "author_id"))

		stats := &hades.AutoMigrateStats{}
		ordie(c.AutoMigrateEx(conn, stats))
		assert.EqualValues(t, 1, stats.NumMigrated, "table was rebuilt")
		assert.EqualValues(t, 1, stats.NumIndexesCreated, "index re-created after rebuild")
		assert.EqualValues(t, []string{"hades_idx_articles__author_id"}, indexNames(t, c, conn, "articles"))
	}

	// changing the declaration retires the old index and creates the new one
	{
		type Article struct {
			ID       int64
			AuthorID int64
			Title    string
		}

		c, err := hades.NewContext(&Article{})
		ordie(err)
		c.Logger = testLogger(t)
		ordie(c.DeclareIndex(&Article{}, "author_id", "title"))

		stats := &hades.AutoMigrateStats{}
		ordie(c.AutoMigrateEx(conn, stats))
		assert.EqualValues(t, 1, stats.NumIndexesCreated)
		assert.EqualValues(t, 1, stats.NumIndexesDropped)
		assert.EqualValues(t, []string{"hades_idx_articles__author_id__title"}, indexNames(t, c, conn, "articles"))
	}

	// a context that doesn't manage a table must not prune managed indexes
	// on it: partial contexts sharing a database own only their own tables
	{
		type Comment struct {
			ID   int64
			Body string
		}

		c, err := hades.NewContext(&Comment{})
		ordie(err)
		c.Logger = testLogger(t)

		stats := &hades.AutoMigrateStats{}
		ordie(c.AutoMigrateEx(conn, stats))
		assert.EqualValues(t, 0, stats.NumIndexesDropped, "articles index untouched by comments-only context")
		assert.EqualValues(t, []string{"hades_idx_articles__author_id__title"}, indexNames(t, c, conn, "articles"))
	}

	// indexes not managed by hades are left alone
	{
		type Article struct {
			ID       int64
			AuthorID int64
			Title    string
		}

		c, err := hades.NewContext(&Article{})
		ordie(err)
		c.Logger = testLogger(t)
		ordie(c.ExecRaw(conn, "CREATE INDEX custom_idx ON articles (title)", nil))

		stats := &hades.AutoMigrateStats{}
		ordie(c.AutoMigrateEx(conn, stats))
		assert.EqualValues(t, 1, stats.NumIndexesDropped, "previously declared index dropped")
		assert.EqualValues(t, []string{"custom_idx"}, indexNames(t, c, conn, "articles"))
	}

	// an existing index whose name matches a declaration but whose actual
	// definition differs is rebuilt to match: convergence trusts the
	// indexed columns, not the (non-injective) name
	{
		type Article struct {
			ID       int64
			AuthorID int64
			Title    string
		}

		c, err := hades.NewContext(&Article{})
		ordie(err)
		c.Logger = testLogger(t)
		ordie(c.ExecRaw(conn, "CREATE INDEX hades_idx_articles__author_id ON articles (title)", nil))
		ordie(c.DeclareIndex(&Article{}, "author_id"))

		stats := &hades.AutoMigrateStats{}
		ordie(c.AutoMigrateEx(conn, stats))
		assert.EqualValues(t, 1, stats.NumIndexesDropped, "drifted index dropped")
		assert.EqualValues(t, 1, stats.NumIndexesCreated, "declared definition re-created")

		var cols []string
		ordie(c.ExecRaw(conn, "PRAGMA index_info(hades_idx_articles__author_id)",
			func(stmt *sqlite.Stmt) error {
				cols = append(cols, stmt.ColumnText(2))
				return nil
			}))
		assert.EqualValues(t, []string{"author_id"}, cols)
	}

	// index names are database-global: a matching name on a different
	// table must be rebuilt on the declared table
	{
		type Article struct {
			ID       int64
			AuthorID int64
			Title    string
		}
		type Comment struct {
			ID       int64
			Body     string
			AuthorID int64
		}

		c, err := hades.NewContext(&Article{}, &Comment{})
		ordie(err)
		c.Logger = testLogger(t)

		// sync schema first so the rogue index survives table rebuilds
		ordie(c.AutoMigrate(conn))
		ordie(c.ExecRaw(conn, "CREATE INDEX hades_idx_articles__author_id ON comments (author_id)", nil))

		ordie(c.DeclareIndex(&Article{}, "author_id"))
		stats := &hades.AutoMigrateStats{}
		ordie(c.AutoMigrateEx(conn, stats))
		assert.EqualValues(t, 1, stats.NumIndexesDropped, "index on wrong table dropped")
		assert.EqualValues(t, 1, stats.NumIndexesCreated, "declared index created on the declared table")
		assert.EqualValues(t, []string{"custom_idx", "hades_idx_articles__author_id"}, indexNames(t, c, conn, "articles"))
		assert.Empty(t, indexNames(t, c, conn, "comments"))
	}
}
