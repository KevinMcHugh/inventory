package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	apigen "github.com/kevinmchugh/inventory/internal/api/gen"
	dbgen "github.com/kevinmchugh/inventory/internal/db/gen"
	invmcp "github.com/kevinmchugh/inventory/internal/mcp"
	"github.com/kevinmchugh/inventory/internal/server"
)

func main() {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		slog.Error("DATABASE_URL required")
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		slog.Error("pgxpool.New", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("pgxpool.Ping", "err", err)
		os.Exit(1)
	}

	q := dbgen.New(pool)
	srv := server.New(q)
	mcpServer := invmcp.NewServer(q)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

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
	if err := http.ListenAndServe(":"+port, r); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
