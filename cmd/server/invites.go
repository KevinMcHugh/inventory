package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/xid"

	dbgen "github.com/KevinMcHugh/inventory/internal/db/gen"
	"github.com/KevinMcHugh/inventory/internal/idp"
)

func runInvites(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: invites <create|list> ...")
	}
	switch args[0] {
	case "create":
		return runInvitesCreate(args[1:])
	case "list":
		return runInvitesList(args[1:])
	default:
		return fmt.Errorf("unknown invites subcommand %q", args[0])
	}
}

func runInvitesCreate(args []string) error {
	fs := flag.NewFlagSet("invites create", flag.ExitOnError)
	tenantID := fs.String("tenant", "", "tenant xid to link into (blank means the invite mints a fresh tenant on redeem)")
	note := fs.String("note", "", "human-readable note stored on the invite for later review")
	expires := fs.Duration("expires-in", 0, "expiry duration (default: never, e.g. 720h for 30 days)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx := context.Background()
	pool, err := openPool(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()
	q := dbgen.New(pool)

	raw, err := idp.GenerateInvite()
	if err != nil {
		return err
	}
	arg := dbgen.CreateInviteParams{
		ID:       xid.New().String(),
		CodeHash: idp.HashInvite(raw),
	}
	if *tenantID != "" {
		arg.TenantID = tenantID
	}
	if *note != "" {
		arg.Note = note
	}
	if *expires > 0 {
		arg.ExpiresAt = pgtype.Timestamptz{Time: time.Now().Add(*expires), Valid: true}
	}
	if _, err := q.CreateInvite(ctx, arg); err != nil {
		return fmt.Errorf("create invite: %w", err)
	}

	fmt.Printf("invite:    %s\n\n", raw)
	if *tenantID != "" {
		fmt.Printf("Redeem at: https://<host>/auth/google?invite=%s (links into tenant %s)\n", raw, *tenantID)
	} else {
		fmt.Printf("Redeem at: https://<host>/auth/google?invite=%s (creates a fresh tenant)\n", raw)
	}
	fmt.Println("Single-use. Save the code now — it will never be shown again.")
	return nil
}

func runInvitesList(args []string) error {
	fs := flag.NewFlagSet("invites list", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx := context.Background()
	pool, err := openPool(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()
	q := dbgen.New(pool)

	rows, err := q.ListActiveInvites(ctx)
	if err != nil {
		return err
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tTENANT\tNOTE\tCREATED\tEXPIRES")
	for _, r := range rows {
		tenant := "(new tenant)"
		if r.TenantID != nil {
			tenant = *r.TenantID
		}
		note := ""
		if r.Note != nil {
			note = *r.Note
		}
		exp := "never"
		if r.ExpiresAt.Valid {
			exp = r.ExpiresAt.Time.Format("2006-01-02 15:04Z")
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			r.ID, tenant, note,
			r.CreatedAt.Time.Format("2006-01-02 15:04Z"),
			exp,
		)
	}
	return tw.Flush()
}
