package meta

import (
	"context"
	"testing"
	"time"

	"github.com/panda/tracy/internal/annotation"
	sqlite "github.com/panda/tracy/internal/storage/sqlite"
)

func TestAnnotationsAreProjectScoped(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(ctx, t.TempDir()+"/meta.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := NewStore(db)
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := store.CreateProject(ctx, Project{ID: "p1", Name: "one", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	score := .75
	item := annotation.Annotation{ID: "ann-1", ProjectID: "p1", TraceID: "trace", Key: "quality", Score: &score, Label: "good", Comment: "works", CreatedBy: "key", CreatedAt: now, UpdatedAt: now}
	if err := store.CreateAnnotation(ctx, item); err != nil {
		t.Fatal(err)
	}
	items, err := store.ListAnnotations(ctx, "p1", "trace")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Score == nil || *items[0].Score != score {
		t.Fatalf("items=%+v", items)
	}
	if _, err := store.ListAnnotations(ctx, "p2", "trace"); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteAnnotation(ctx, "p2", "ann-1"); err != ErrNotFound {
		t.Fatalf("cross-project delete error=%v", err)
	}
	if err := store.DeleteAnnotation(ctx, "p1", "ann-1"); err != nil {
		t.Fatal(err)
	}
}
