package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/xid"

	apigen "github.com/KevinMcHugh/inventory/internal/api/gen"
	"github.com/KevinMcHugh/inventory/internal/auth"
	dbgen "github.com/KevinMcHugh/inventory/internal/db/gen"
	invmcp "github.com/KevinMcHugh/inventory/internal/mcp"
	"github.com/KevinMcHugh/inventory/internal/oauth"
	"github.com/KevinMcHugh/inventory/internal/server"
	authmw "github.com/KevinMcHugh/inventory/internal/server/middleware"
)

func main() {
	var err error
	switch cmd := firstArg(); cmd {
	case "", "serve":
		err = runServer()
	case "bootstrap":
		err = runBootstrap(os.Args[2:])
	case "keys":
		err = runKeys(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q; use one of: serve, bootstrap, keys", cmd)
	}
	if err != nil {
		slog.Error("command failed", "err", err)
		os.Exit(1)
	}
}

func firstArg() string {
	if len(os.Args) < 2 {
		return ""
	}
	return os.Args[1]
}

// -----------------------------------------------------------------------------
// serve
// -----------------------------------------------------------------------------

func runServer() error {
	ctx := context.Background()
	pool, err := openPool(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("pgxpool.Ping: %w", err)
	}

	issuer := os.Getenv("PUBLIC_URL")
	if issuer == "" {
		issuer = "http://localhost:8080"
	}
	issuer = strings.TrimRight(issuer, "/")

	q := dbgen.New(pool)
	srv := server.New(q)
	mcpServer := invmcp.NewServer(q)
	oauthHandler := &oauth.Handler{Issuer: issuer, Q: q}

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	// If a web bundle is present, browser navigation to any URL that is not
	// explicitly server-rendered by the app itself gets the SPA index.html
	// even when an API route would otherwise match. JSON fetches (default
	// Accept: */*) fall through to the API as normal.
	dist := webDistDir()
	var spa http.HandlerFunc
	if dist != "" {
		slog.Info("serving web UI", "dist", dist)
		spa = spaHandler(dist)
		r.Use(spaOrAPI(spa))
	}

	// Public: OAuth discovery + flow endpoints do their own credential validation.
	oauthHandler.Mount(r)

	// Auth-protected: REST + MCP.
	mcpHandler := mcpsdk.NewStreamableHTTPHandler(
		func(*http.Request) *mcpsdk.Server { return mcpServer },
		nil,
	)
	r.Group(func(r chi.Router) {
		r.Use(authmw.Auth(q, issuer))
		apigen.HandlerFromMux(apigen.NewStrictHandler(srv, nil), r)
		// The bare /mcp path is intercepted by the fly/sprite proxy layer
		// (POST hangs before reaching us). Mounting under /mcp/rpc dodges
		// that; subpaths reach the SDK handler fine.
		r.Handle("/mcp/rpc", mcpHandler)
		r.Handle("/mcp/rpc/*", mcpHandler)
	})

	// Static asset fallback: browser GET for /assets/*.js etc. lands here
	// because JS asset requests do not send Accept: text/html.
	if spa != nil {
		r.NotFound(spa)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	slog.Info("starting server", "port", port)
	return http.ListenAndServe(":"+port, r)
}

// -----------------------------------------------------------------------------
// bootstrap
// -----------------------------------------------------------------------------

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
	pool, err := openPool(ctx)
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

	raw, err := mintKey(ctx, q, tenant.ID, *keyName)
	if err != nil {
		return err
	}

	fmt.Printf("tenant_id: %s\n", tenant.ID)
	printKey(raw)
	return nil
}

// -----------------------------------------------------------------------------
// keys
// -----------------------------------------------------------------------------

func runKeys(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: keys <list|create|rotate> ...")
	}
	switch args[0] {
	case "list":
		return runKeysList(args[1:])
	case "create":
		return runKeysCreate(args[1:])
	case "rotate":
		return runKeysRotate(args[1:])
	default:
		return fmt.Errorf("unknown keys subcommand %q", args[0])
	}
}

func runKeysList(args []string) error {
	fs := flag.NewFlagSet("keys list", flag.ExitOnError)
	tenantID := fs.String("tenant", "", "tenant xid (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *tenantID == "" {
		return errors.New("--tenant is required")
	}

	ctx := context.Background()
	pool, err := openPool(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

	q := dbgen.New(pool)
	rows, err := q.ListAPIKeysByTenant(ctx, *tenantID)
	if err != nil {
		return err
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tCREATED\tLAST USED")
	for _, r := range rows {
		last := "-"
		if r.LastUsedAt.Valid {
			last = r.LastUsedAt.Time.Format("2006-01-02 15:04:05Z")
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			r.ID, r.Name,
			r.CreatedAt.Time.Format("2006-01-02 15:04:05Z"),
			last,
		)
	}
	return tw.Flush()
}

func runKeysCreate(args []string) error {
	fs := flag.NewFlagSet("keys create", flag.ExitOnError)
	tenantID := fs.String("tenant", "", "tenant xid (required)")
	keyName := fs.String("name", "", "label for the created api key (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *tenantID == "" || *keyName == "" {
		return errors.New("--tenant and --name are required")
	}

	ctx := context.Background()
	pool, err := openPool(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

	raw, err := mintKey(ctx, dbgen.New(pool), *tenantID, *keyName)
	if err != nil {
		return err
	}
	printKey(raw)
	return nil
}

// runKeysRotate soft-deletes an existing key and mints a replacement under the
// same tenant, carrying its label forward. The old key stops working
// immediately; the new raw key is printed once.
func runKeysRotate(args []string) error {
	fs := flag.NewFlagSet("keys rotate", flag.ExitOnError)
	keyID := fs.String("key-id", "", "api key xid to rotate (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *keyID == "" {
		return errors.New("--key-id is required")
	}

	ctx := context.Background()
	pool, err := openPool(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

	q := dbgen.New(pool)
	old, err := q.GetAPIKey(ctx, *keyID)
	if err != nil {
		return fmt.Errorf("look up key: %w", err)
	}

	if err := q.DeleteAPIKey(ctx, old.ID); err != nil {
		return fmt.Errorf("delete old key: %w", err)
	}

	raw, err := mintKey(ctx, q, old.TenantID, old.Name)
	if err != nil {
		return err
	}
	fmt.Printf("rotated key %s (tenant %s, name %q)\n", old.ID, old.TenantID, old.Name)
	printKey(raw)
	return nil
}

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

func openPool(ctx context.Context) (*pgxpool.Pool, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, errors.New("DATABASE_URL required")
	}
	return pgxpool.New(ctx, dsn)
}

func mintKey(ctx context.Context, q dbgen.Querier, tenantID, name string) (string, error) {
	raw, err := auth.GenerateKey()
	if err != nil {
		return "", err
	}
	if _, err := q.CreateAPIKey(ctx, dbgen.CreateAPIKeyParams{
		ID:       xid.New().String(),
		TenantID: tenantID,
		Name:     name,
		KeyHash:  auth.Hash(raw),
	}); err != nil {
		return "", fmt.Errorf("create api key: %w", err)
	}
	return raw, nil
}

func printKey(raw string) {
	fmt.Printf("api_key:   %s\n\n", raw)
	fmt.Println("Save the key now — it will never be shown again.")
}

// webDistDir returns the directory holding the built SPA, or "" if none is
// configured or present. Order: $WEB_DIST, then ./web/dist relative to CWD.
func webDistDir() string {
	if v := os.Getenv("WEB_DIST"); v != "" {
		if info, err := os.Stat(v); err == nil && info.IsDir() {
			return v
		}
		return ""
	}
	const def = "./web/dist"
	if info, err := os.Stat(def); err == nil && info.IsDir() {
		return def
	}
	return ""
}

// spaHandler serves static assets from dir, falling back to index.html for
// any path that does not match a file. That fallback is what lets client-side
// routes (e.g. /callback) work without server config.
func spaHandler(dir string) http.HandlerFunc {
	index := filepath.Join(dir, "index.html")
	return func(w http.ResponseWriter, r *http.Request) {
		clean := path.Clean("/" + r.URL.Path)
		full := filepath.Join(dir, clean)
		if info, err := os.Stat(full); err == nil && !info.IsDir() {
			http.ServeFile(w, r, full)
			return
		}
		http.ServeFile(w, r, index)
	}
}

// spaOrAPI is chi middleware that intercepts browser navigation and serves
// the SPA before chi's own routing runs. It applies only to GET requests
// whose Accept header prefers text/html — which is what browsers send when
// following a link or typing a URL, but not what fetch() sends by default.
//
// Server-rendered pages (the OAuth login form, health, well-known discovery)
// are exempt so they render on the server as intended.
func spaOrAPI(spa http.HandlerFunc) func(http.Handler) http.Handler {
	serverRendered := func(p string) bool {
		if p == "/health" {
			return true
		}
		return strings.HasPrefix(p, "/oauth/") || strings.HasPrefix(p, "/.well-known/")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || serverRendered(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			if strings.Contains(r.Header.Get("Accept"), "text/html") {
				spa(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
