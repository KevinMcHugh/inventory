package oauth

import (
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/KevinMcHugh/inventory/internal/auth"
	dbgen "github.com/KevinMcHugh/inventory/internal/db/gen"
)

// authorizeParams holds the OAuth authorize-request fields we care about.
type authorizeParams struct {
	ClientID            string
	RedirectURI         string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	Scope               string
}

func parseAuthorizeParams(vals url.Values) (authorizeParams, string) {
	if vals.Get("response_type") != "code" {
		return authorizeParams{}, "response_type must be 'code'"
	}
	p := authorizeParams{
		ClientID:            vals.Get("client_id"),
		RedirectURI:         vals.Get("redirect_uri"),
		State:               vals.Get("state"),
		CodeChallenge:       vals.Get("code_challenge"),
		CodeChallengeMethod: vals.Get("code_challenge_method"),
		Scope:               vals.Get("scope"),
	}
	if p.ClientID == "" {
		return p, "client_id is required"
	}
	if p.RedirectURI == "" {
		return p, "redirect_uri is required"
	}
	if p.CodeChallenge == "" {
		return p, "code_challenge is required (PKCE)"
	}
	if p.CodeChallengeMethod == "" {
		p.CodeChallengeMethod = "plain"
	}
	if p.CodeChallengeMethod != "S256" {
		return p, "only S256 code_challenge_method is supported"
	}
	return p, ""
}

// authorizeGET renders a small HTML form asking the user to paste their
// inv_... api key to grant the requesting client access.
func (h *Handler) authorizeGET(w http.ResponseWriter, r *http.Request) {
	p, errMsg := parseAuthorizeParams(r.URL.Query())
	if errMsg != "" {
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	uris, err := h.loadClientRedirectURIs(r, p.ClientID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "unknown client_id", http.StatusBadRequest)
			return
		}
		http.Error(w, "client lookup failed", http.StatusInternalServerError)
		return
	}
	if !stringsIncludes(uris, p.RedirectURI) {
		http.Error(w, "redirect_uri not registered for client", http.StatusBadRequest)
		return
	}

	view := authorizeView{Params: p, GoogleEnabled: h.idp != nil && h.idp.GoogleEnabled()}
	if err := authorizeTmpl.Execute(w, view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// authorizePOST validates the pasted api key, mints a one-time auth code
// bound to (client, tenant, redirect_uri, code_challenge), and redirects the
// user-agent back to redirect_uri?code=...&state=...
func (h *Handler) authorizePOST(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	p, errMsg := parseAuthorizeParams(r.Form)
	if errMsg != "" {
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	uris, err := h.loadClientRedirectURIs(r, p.ClientID)
	if err != nil {
		http.Error(w, "unknown client_id", http.StatusBadRequest)
		return
	}
	if !stringsIncludes(uris, p.RedirectURI) {
		http.Error(w, "redirect_uri not registered for client", http.StatusBadRequest)
		return
	}

	rawKey := r.FormValue("api_key")
	if rawKey == "" {
		renderAuthorizeError(w, p, "please paste your api key")
		return
	}

	row, err := h.Q.GetAPIKeyByHash(r.Context(), auth.Hash(rawKey))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			renderAuthorizeError(w, p, "that api key is not recognized")
			return
		}
		http.Error(w, "auth lookup failed", http.StatusInternalServerError)
		return
	}

	rawCode, err := randToken(AuthCodePrefix)
	if err != nil {
		http.Error(w, "code generation failed", http.StatusInternalServerError)
		return
	}
	if _, err := h.Q.CreateOAuthCode(r.Context(), dbgen.CreateOAuthCodeParams{
		CodeHash:            hashToken(rawCode),
		ClientID:            &p.ClientID,
		TenantID:            row.TenantID,
		RedirectUri:         p.RedirectURI,
		CodeChallenge:       p.CodeChallenge,
		CodeChallengeMethod: p.CodeChallengeMethod,
		Scope:               nullable(p.Scope),
		ExpiresAt:           timestamptzFrom(time.Now().Add(AuthCodeTTL)),
	}); err != nil {
		http.Error(w, "code persist failed", http.StatusInternalServerError)
		return
	}

	redirect := p.RedirectURI
	sep := "?"
	if hasQuery(redirect) {
		sep = "&"
	}
	redirect += sep + "code=" + url.QueryEscape(rawCode)
	if p.State != "" {
		redirect += "&state=" + url.QueryEscape(p.State)
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func hasQuery(u string) bool {
	for i := 0; i < len(u); i++ {
		if u[i] == '?' {
			return true
		}
	}
	return false
}

func renderAuthorizeError(w http.ResponseWriter, p authorizeParams, msg string) {
	w.WriteHeader(http.StatusBadRequest)
	_ = authorizeTmpl.Execute(w, authorizeView{Params: p, Error: msg})
}

type authorizeView struct {
	Params        authorizeParams
	Error         string
	GoogleEnabled bool
}

var authorizeTmpl = template.Must(template.New("authorize").Parse(`<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Inventory · Authorize</title>
  <style>
    :root {
      color-scheme: light dark;
      --bg: #fdfdfd; --fg: #1a1a1a; --muted: #555; --input-bg: #ffffff;
      --input-border: #bbb; --btn-bg: #222; --btn-fg: #ffffff;
      --err-bg: #ffe8ec; --err-fg: #b00020; --code-bg: #f4f4f4;
    }
    @media (prefers-color-scheme: dark) {
      :root {
        --bg: #14161a; --fg: #e8e8e6; --muted: #8b8f95; --input-bg: #1a1d22;
        --input-border: #3a3f45; --btn-bg: #f0f0ee; --btn-fg: #14161a;
        --err-bg: #3a1a1f; --err-fg: #ff9aa8; --code-bg: #1e2227;
      }
    }
    html, body { background: var(--bg); color: var(--fg); }
    body { font-family: system-ui, -apple-system, sans-serif; max-width: 480px; margin: 4rem auto; padding: 0 1rem; }
    h1 { font-size: 1.25rem; margin-bottom: 0.5rem; }
    p  { color: var(--muted); line-height: 1.5; }
    form { display: grid; gap: 1rem; margin-top: 1.5rem; }
    input[type=text] {
      padding: 0.6rem 0.8rem; font-family: ui-monospace, Menlo, Consolas, monospace;
      background: var(--input-bg); color: var(--fg);
      border: 1px solid var(--input-border); border-radius: 6px;
    }
    button {
      padding: 0.6rem 1rem; background: var(--btn-bg); color: var(--btn-fg);
      border: 0; border-radius: 6px; cursor: pointer; font: inherit;
    }
    .err { color: var(--err-fg); background: var(--err-bg); padding: 0.6rem 0.8rem; border-radius: 6px; }
    code { background: var(--code-bg); padding: 0 0.25rem; border-radius: 3px; }
    a.google {
      display: block; text-align: center;
      padding: 0.7rem 1rem; margin-top: 1rem;
      background: var(--btn-bg); color: var(--btn-fg);
      border-radius: 6px; text-decoration: none; font-weight: 600;
    }
    a.google:hover { opacity: 0.9; }
    .or {
      text-align: center; color: var(--muted); font-size: 0.85rem;
      margin: 1rem 0 0.5rem 0;
    }
  </style>
</head>
<body>
  <h1>Sign in to grant access</h1>
  {{if .Error}}<div class="err">{{.Error}}</div>{{end}}

  {{if .GoogleEnabled}}
  <p>Choose how to sign in.</p>
  <a href="/auth/google?client_id={{.Params.ClientID}}&redirect_uri={{.Params.RedirectURI}}&state={{.Params.State}}&code_challenge={{.Params.CodeChallenge}}&code_challenge_method={{.Params.CodeChallengeMethod}}&scope={{.Params.Scope}}"
     class="google">
    Sign in with Google
  </a>
  <div class="or">or</div>
  {{end}}

  <p>Paste your Inventory api key (starts with <code>inv_</code>) to grant access.</p>
  <form method="POST">
    <input type="text" name="api_key" autocomplete="off" autofocus placeholder="inv_...">
    <input type="hidden" name="response_type" value="code">
    <input type="hidden" name="client_id" value="{{.Params.ClientID}}">
    <input type="hidden" name="redirect_uri" value="{{.Params.RedirectURI}}">
    <input type="hidden" name="state" value="{{.Params.State}}">
    <input type="hidden" name="code_challenge" value="{{.Params.CodeChallenge}}">
    <input type="hidden" name="code_challenge_method" value="{{.Params.CodeChallengeMethod}}">
    <input type="hidden" name="scope" value="{{.Params.Scope}}">
    <button type="submit">Allow</button>
  </form>
</body>
</html>`))
