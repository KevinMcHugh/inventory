package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	dbgen "github.com/KevinMcHugh/inventory/internal/db/gen"
	"github.com/KevinMcHugh/inventory/internal/indexedfields"
	"github.com/KevinMcHugh/inventory/internal/kindschema"
)

// runReindex rebuilds indexed_fields for every model of a kind. Writes only
// index the fields their schema marks indexed at write time, so after a new
// kind version marks an existing field indexed, this backfills the models
// written before it. It runs out-of-band so a large kind is not bound by an
// HTTP request timeout.
//
// Every model is indexed against the kind's latest schema, whichever version
// it is pinned to. Model rows themselves (body, kind_version_id, updated_at)
// are left untouched.
func runReindex(args []string) error {
	fs := flag.NewFlagSet("reindex", flag.ExitOnError)
	tenantID := fs.String("tenant", "", "tenant xid (required)")
	kindID := fs.String("kind", "", "kind xid to reindex (required)")
	batchSize := fs.Int("batch-size", 500, "models fetched per page")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *tenantID == "" || *kindID == "" {
		return errors.New("--tenant and --kind are required")
	}
	if *batchSize <= 0 {
		return errors.New("--batch-size must be positive")
	}

	ctx := context.Background()
	pool, err := openPool(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()
	q := dbgen.New(pool)

	kind, err := q.GetKind(ctx, dbgen.GetKindParams{ID: *kindID, TenantID: *tenantID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("kind %s not found in tenant %s", *kindID, *tenantID)
		}
		return fmt.Errorf("look up kind: %w", err)
	}
	schema, err := kindschema.LoadLatest(ctx, q, kind.ID)
	if err != nil {
		return fmt.Errorf("load latest schema: %w", err)
	}

	var indexed []string
	for _, f := range schema.Fields {
		if f.Indexed {
			indexed = append(indexed, f.Key)
		}
	}
	fmt.Printf("reindexing kind %s (%s); indexed fields: %v\n", kind.ID, kind.Name, indexed)

	var done, failed int
	after := ""
	for {
		page, err := q.ListModelsByKindAfterID(ctx, dbgen.ListModelsByKindAfterIDParams{
			TenantID: kind.TenantID,
			KindID:   kind.ID,
			AfterID:  after,
			PageSize: int32(*batchSize),
		})
		if err != nil {
			return fmt.Errorf("list models after %q: %w", after, err)
		}
		for _, m := range page {
			if err := reindexModel(ctx, pool, schema, m); err != nil {
				slog.Error("reindex model", "modelId", m.ID, "slug", m.Slug, "err", err)
				failed++
				continue
			}
			done++
		}
		if len(page) < *batchSize {
			break
		}
		after = page[len(page)-1].ID
		fmt.Printf("  %d models reindexed so far\n", done+failed)
	}

	fmt.Printf("reindexed %d models, %d failed\n", done, failed)
	if failed > 0 {
		return fmt.Errorf("%d models failed to reindex", failed)
	}
	return nil
}

// reindexModel rebuilds one model's rows in its own transaction, so a
// concurrent search never sees the model half-indexed and a failure leaves
// its previous rows in place.
func reindexModel(ctx context.Context, pool *pgxpool.Pool, schema *kindschema.Schema, m dbgen.Model) error {
	var body map[string]any
	if err := json.Unmarshal(m.Body, &body); err != nil {
		return fmt.Errorf("decode body: %w", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := indexedfields.Rebuild(ctx, dbgen.New(tx), schema, m.TenantID, m.KindID, m.ID, body); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
