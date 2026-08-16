package api

import (
	"context"
	"testing"

	"github.com/magalab/tracy/internal/storage/meta"
	sqlitestore "github.com/magalab/tracy/internal/storage/sqlite"
)

func newTestMetaStore(t *testing.T) (context.Context, *meta.Store) {
	t.Helper()
	ctx := context.Background()
	db, err := sqlitestore.Open(ctx, t.TempDir()+"/meta.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	store := meta.NewStore(db)
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	return ctx, store
}
