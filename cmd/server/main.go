package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/xid"

	apigen "github.com/KevinMcHugh/inventory/internal/api/gen"
	"github.com/KevinMcHugh/inventory/internal/auth"
	dbgen "github.com/KevinMcHugh/inventory/internal/db/gen"
	invmcp "github.com/KevinMcHugh/inventory/internal/mcp"
	"github.com/KevinMcHugh/inventory/internal/server"
	authmw "github.com/KevinMcHugh/inventory/internal/server/middleware"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "bootstrap" {
		if err := runBootstrap(os.Args[2:]); err != nil {
			slog.Error("bootstrap failed", "err", err)
			os.Exit(1)
		}
		return
	}

	if err := runServer(); err != nil {
		slog.Error("server exited", "err", err)
		os.Exit(1)
	}
}

func runServer() error {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL required")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("pgxpool.New: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("pgxpool.Ping: %w", err)
	}

	q := dbgen.New(pool)
	srv := server.New(q)
	mcpServer := invmcp.NewServer(q)

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(authmw.Auth(q))

	apigen.HandlerFromMux(apigen.NewStrictHandler(srv, nil), r)

	mcpHandler := mcpsdk.NewStreamableHTTPHandler(
		func(*http.Request) *mcpsdk.Server { return mcpServer },
		nil,
	)
	r.Handle("/mcp", mcpHandler)
	r.Handle("/mcp/*", mcpHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	slog.Info("starting server", "port", port)
	return http.ListenAndServe(":"+port, r)
}

// runBootstrap creates a tenant and its first api key, printing the raw key
// exactly once. This is the only way to create a new tenant.
func runBootstrap(args []string) error {
	fs := flag.NewFlagSet("bootstrap", flag.ExitOnError)
	tenantName := fs.String("tenant", "", "tenant display name (required)")
	keyName := fs.String("key-name", "default", "label for the created api key")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *tenantName == "" {
		return errors.New("--tenant is required")
	}

	ctx := context.Background()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL required")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()

	q := dbgen.New(pool)

	tenant, err := q.CreateTenant(ctx, dbgen.CreateTenantParams{
		ID:   xid.New().String(),
		Name: *tenantName,
	})
	if err != nil {
		return fmt.Errorf("create tenant: %w", err)
	}

	rawKey, err := auth.GenerateKey()
	if err != nil {
		return err
	}
	if _, err := q.CreateAPIKey(ctx, dbgen.CreateAPIKeyParams{
		ID:       xid.New().String(),
		TenantID: tenant.ID,
		Name:     *keyName,
		KeyHash:  auth.Hash(rawKey),
	}); err != nil {
		return fmt.Errorf("create api key: %w", err)
	}

	fmt.Printf("tenant_id: %s\n", tenant.ID)
	fmt.Printf("api_key:   %s\n", rawKey)
	fmt.Println()
	fmt.Println("Save the key now — it will never be shown again.")
	return nil
}
