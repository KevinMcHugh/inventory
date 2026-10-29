// Command sprite-deploy redeploys inventory onto its fly.io sprite. It is
// what the GitHub Actions deploy job runs.
//
// It takes a sprite checkpoint (so a bad migration can be rolled back with
// `sprite restore`), fast-forwards the sprite checkout to the given commit,
// and runs deploy/redeploy.sh there, streaming its output and exiting with
// its exit code.
//
// This is its own Go module so the sprites SDK and its websocket deps stay
// out of the server binary.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"time"

	sprites "github.com/superfly/sprites-go"
)

var shaRE = regexp.MustCompile(`^[0-9a-f]{7,40}$`)

func main() {
	spriteName := flag.String("sprite", envOr("SPRITE_NAME", "inventory"), "sprite to deploy to")
	sha := flag.String("sha", os.Getenv("GITHUB_SHA"), "commit to deploy (must be on origin/main)")
	dir := flag.String("dir", "/home/sprite/inventory", "checkout path on the sprite")
	checkpoint := flag.Bool("checkpoint", true, "take a sprite checkpoint before deploying")
	timeout := flag.Duration("timeout", 15*time.Minute, "overall deploy timeout")
	flag.Parse()

	// Trim so a secret pasted with a trailing newline still makes a valid
	// Authorization header.
	token := strings.TrimSpace(os.Getenv("SPRITES_TOKEN"))
	if token == "" {
		log.Fatal("SPRITES_TOKEN is not set")
	}
	if !shaRE.MatchString(*sha) {
		log.Fatalf("-sha %q is not a git commit sha", *sha)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, *timeout)
	defer cancel()

	client := sprites.New(token)
	defer client.Close()
	sprite := client.Sprite(*spriteName)

	if *checkpoint {
		if err := takeCheckpoint(ctx, sprite, "pre-deploy "+*sha); err != nil {
			log.Fatalf("checkpoint: %v", err)
		}
	}

	script := fmt.Sprintf(
		"set -euo pipefail; cd %s && git fetch --quiet origin main && git merge --ff-only %s && exec bash deploy/redeploy.sh",
		*dir, *sha,
	)
	cmd := sprite.CommandContext(ctx, "bash", "-lc", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *sprites.ExitError
		if errors.As(err, &exitErr) {
			log.Printf("deploy failed: %v", err)
			os.Exit(exitErr.ExitCode())
		}
		log.Fatalf("deploy: %v", err)
	}
}

func takeCheckpoint(ctx context.Context, sprite *sprites.Sprite, comment string) error {
	log.Printf("creating checkpoint %q", comment)
	stream, err := sprite.CreateCheckpointWithComment(ctx, comment)
	if err != nil {
		return err
	}
	return stream.ProcessAll(func(m *sprites.StreamMessage) error {
		if m.Type == "error" {
			return errors.New(m.Error)
		}
		if m.Data != "" {
			log.Printf("checkpoint: %s", m.Data)
		}
		return nil
	})
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
