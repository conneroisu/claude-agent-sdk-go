From: https://github.com/modelcontextprotocol/go-sdk
<files>
<file path="auth/auth.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package auth

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"
)

// TokenInfo holds information from a bearer token.
type TokenInfo struct {
	Scopes     []string
	Expiration time.Time
	// TODO: add standard JWT fields
	Extra map[string]any
}

// The error that a TokenVerifier should return if the token cannot be verified.
var ErrInvalidToken = errors.New("invalid token")

// The error that a TokenVerifier should return for OAuth-specific protocol errors.
var ErrOAuth = errors.New("oauth error")

// A TokenVerifier checks the validity of a bearer token, and extracts information
// from it. If verification fails, it should return an error that unwraps to ErrInvalidToken.
// The HTTP request is provided in case verifying the token involves checking it.
type TokenVerifier func(ctx context.Context, token string, req *http.Request) (*TokenInfo, error)

// RequireBearerTokenOptions are options for [RequireBearerToken].
type RequireBearerTokenOptions struct {
	// The URL for the resource server metadata OAuth flow, to be returned as part
	// of the WWW-Authenticate header.
	ResourceMetadataURL string
	// The required scopes.
	Scopes []string
}

type tokenInfoKey struct{}

// TokenInfoFromContext returns the [TokenInfo] stored in ctx, or nil if none.
func TokenInfoFromContext(ctx context.Context) *TokenInfo {
	ti := ctx.Value(tokenInfoKey{})
	if ti == nil {
		return nil
	}
	return ti.(*TokenInfo)
}

// RequireBearerToken returns a piece of middleware that verifies a bearer token using the verifier.
// If verification succeeds, the [TokenInfo] is added to the request's context and the request proceeds.
// If verification fails, the request fails with a 401 Unauthenticated, and the WWW-Authenticate header
// is populated to enable [protected resource metadata].
//
// [protected resource metadata]: https://datatracker.ietf.org/doc/rfc9728
func RequireBearerToken(verifier TokenVerifier, opts *RequireBearerTokenOptions) func(http.Handler) http.Handler {
	// Based on typescript-sdk/src/server/auth/middleware/bearerAuth.ts.

	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenInfo, errmsg, code := verify(r, verifier, opts)
			if code != 0 {
				if code == http.StatusUnauthorized || code == http.StatusForbidden {
					if opts != nil && opts.ResourceMetadataURL != "" {
						w.Header().Add("WWW-Authenticate", "Bearer resource_metadata="+opts.ResourceMetadataURL)
					}
				}
				http.Error(w, errmsg, code)
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), tokenInfoKey{}, tokenInfo))
			handler.ServeHTTP(w, r)
		})
	}
}

func verify(req *http.Request, verifier TokenVerifier, opts *RequireBearerTokenOptions) (_ *TokenInfo, errmsg string, code int) {
	// Extract bearer token.
	authHeader := req.Header.Get("Authorization")
	fields := strings.Fields(authHeader)
	if len(fields) != 2 || strings.ToLower(fields[0]) != "bearer" {
		return nil, "no bearer token", http.StatusUnauthorized
	}

	// Verify the token and get information from it.
	tokenInfo, err := verifier(req.Context(), fields[1], req)
	if err != nil {
		if errors.Is(err, ErrInvalidToken) {
			return nil, err.Error(), http.StatusUnauthorized
		}
		if errors.Is(err, ErrOAuth) {
			return nil, err.Error(), http.StatusBadRequest
		}
		return nil, err.Error(), http.StatusInternalServerError
	}

	// Check scopes. All must be present.
	if opts != nil {
		// Note: quadratic, but N is small.
		for _, s := range opts.Scopes {
			if !slices.Contains(tokenInfo.Scopes, s) {
				return nil, "insufficient scope", http.StatusForbidden
			}
		}
	}

	// Check expiration.
	if tokenInfo.Expiration.IsZero() {
		return nil, "token missing expiration", http.StatusUnauthorized
	}
	if tokenInfo.Expiration.Before(time.Now()) {
		return nil, "token expired", http.StatusUnauthorized
	}
	return tokenInfo, "", 0
}
</content>
</file>
<file path="auth/auth_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package auth

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestVerify(t *testing.T) {
	verifier := func(_ context.Context, token string, _ *http.Request) (*TokenInfo, error) {
		switch token {
		case "valid":
			return &TokenInfo{Expiration: time.Now().Add(time.Hour)}, nil
		case "invalid":
			return nil, ErrInvalidToken
		case "oauth":
			return nil, ErrOAuth
		case "noexp":
			return &TokenInfo{}, nil
		case "expired":
			return &TokenInfo{Expiration: time.Now().Add(-time.Hour)}, nil
		default:
			return nil, errors.New("unknown")
		}
	}

	for _, tt := range []struct {
		name     string
		opts     *RequireBearerTokenOptions
		header   string
		wantMsg  string
		wantCode int
	}{
		{
			"valid", nil, "Bearer valid",
			"", 0,
		},
		{
			"bad header", nil, "Barer valid",
			"no bearer token", 401,
		},
		{
			"invalid", nil, "bearer invalid",
			"invalid token", 401,
		},
		{
			"oauth error", nil, "Bearer oauth",
			"oauth error", 400,
		},
		{
			"no expiration", nil, "Bearer noexp",
			"token missing expiration", 401,
		},
		{
			"expired", nil, "Bearer expired",
			"token expired", 401,
		},
		{
			"missing scope", &RequireBearerTokenOptions{Scopes: []string{"s1"}}, "Bearer valid",
			"insufficient scope", 403,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, gotMsg, gotCode := verify(&http.Request{
				Header: http.Header{"Authorization": {tt.header}},
			}, verifier, tt.opts)
			if gotMsg != tt.wantMsg || gotCode != tt.wantCode {
				t.Errorf("got (%q, %d), want (%q, %d)", gotMsg, gotCode, tt.wantMsg, tt.wantCode)
			}
		})
	}
}
</content>
</file>
<file path="auth/client.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

//go:build mcp_go_client_oauth

package auth

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/internal/oauthex"
	"golang.org/x/oauth2"
)

// An OAuthHandler conducts an OAuth flow and returns a [oauth2.TokenSource] if the authorization
// is approved, or an error if not.
type OAuthHandler func(context.Context, OAuthHandlerArgs) (oauth2.TokenSource, error)

// OAuthHandlerArgs are arguments to an [OAuthHandler].
type OAuthHandlerArgs struct {
	// The URL to fetch protected resource metadata, extracted from the WWW-Authenticate header.
	// Empty if not present or there was an error obtaining it.
	ResourceMetadataURL string
}

// HTTPTransport is an [http.RoundTripper] that follows the MCP
// OAuth protocol when it encounters a 401 Unauthorized response.
type HTTPTransport struct {
	handler OAuthHandler
	mu      sync.Mutex // protects opts.Base
	opts    HTTPTransportOptions
}

// NewHTTPTransport returns a new [*HTTPTransport].
// The handler is invoked when an HTTP request results in a 401 Unauthorized status.
// It is called only once per transport. Once a TokenSource is obtained, it is used
// for the lifetime of the transport; subsequent 401s are not processed.
func NewHTTPTransport(handler OAuthHandler, opts *HTTPTransportOptions) (*HTTPTransport, error) {
	if handler == nil {
		return nil, errors.New("handler cannot be nil")
	}
	t := &HTTPTransport{
		handler: handler,
	}
	if opts != nil {
		t.opts = *opts
	}
	if t.opts.Base == nil {
		t.opts.Base = http.DefaultTransport
	}
	return t, nil
}

// HTTPTransportOptions are options to [NewHTTPTransport].
type HTTPTransportOptions struct {
	// Base is the [http.RoundTripper] to use.
	// If nil, [http.DefaultTransport] is used.
	Base http.RoundTripper
}

func (t *HTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	base := t.opts.Base
	t.mu.Unlock()

	resp, err := base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}
	if _, ok := base.(*oauth2.Transport); ok {
		// We failed to authorize even with a token source; give up.
		return resp, nil
	}

	resp.Body.Close()
	// Try to authorize.
	t.mu.Lock()
	defer t.mu.Unlock()
	// If we don't have a token source, get one by following the OAuth flow.
	// (We may have obtained one while t.mu was not held above.)
	// TODO: We hold the lock for the entire OAuth flow. This could be a long
	// time. Is there a better way?
	if _, ok := t.opts.Base.(*oauth2.Transport); !ok {
		authHeaders := resp.Header[http.CanonicalHeaderKey("WWW-Authenticate")]
		ts, err := t.handler(req.Context(), OAuthHandlerArgs{
			ResourceMetadataURL: extractResourceMetadataURL(authHeaders),
		})
		if err != nil {
			return nil, err
		}
		t.opts.Base = &oauth2.Transport{Base: t.opts.Base, Source: ts}
	}
	return t.opts.Base.RoundTrip(req.Clone(req.Context()))
}

func extractResourceMetadataURL(authHeaders []string) string {
	cs, err := oauthex.ParseWWWAuthenticate(authHeaders)
	if err != nil {
		return ""
	}
	return oauthex.ResourceMetadataURL(cs)
}
</content>
</file>
<file path="auth/client_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

//go:build mcp_go_client_oauth

package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/oauth2"
)

// TestHTTPTransport validates the OAuth HTTPTransport.
func TestHTTPTransport(t *testing.T) {
	const testToken = "test-token-123"
	fakeTokenSource := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: testToken,
		TokenType:   "Bearer",
	})

	// authServer simulates a resource that requires OAuth.
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == fmt.Sprintf("Bearer %s", testToken) {
			w.WriteHeader(http.StatusOK)
			return
		}

		w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="http://metadata.example.com"`)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer authServer.Close()

	t.Run("successful auth flow", func(t *testing.T) {
		var handlerCalls int
		handler := func(ctx context.Context, args OAuthHandlerArgs) (oauth2.TokenSource, error) {
			handlerCalls++
			if args.ResourceMetadataURL != "http://metadata.example.com" {
				t.Errorf("handler got metadata URL %q, want %q", args.ResourceMetadataURL, "http://metadata.example.com")
			}
			return fakeTokenSource, nil
		}

		transport, err := NewHTTPTransport(handler, nil)
		if err != nil {
			t.Fatalf("NewHTTPTransport() failed: %v", err)
		}
		client := &http.Client{Transport: transport}

		resp, err := client.Get(authServer.URL)
		if err != nil {
			t.Fatalf("client.Get() failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if handlerCalls != 1 {
			t.Errorf("handler was called %d times, want 1", handlerCalls)
		}

		// Second request should reuse the token and not call the handler again.
		resp2, err := client.Get(authServer.URL)
		if err != nil {
			t.Fatalf("second client.Get() failed: %v", err)
		}
		defer resp2.Body.Close()

		if resp2.StatusCode != http.StatusOK {
			t.Errorf("second request got status %d, want %d", resp2.StatusCode, http.StatusOK)
		}
		if handlerCalls != 1 {
			t.Errorf("handler should still be called only once, but was %d", handlerCalls)
		}
	})

	t.Run("handler returns error", func(t *testing.T) {
		handlerErr := errors.New("user rejected auth")
		handler := func(ctx context.Context, args OAuthHandlerArgs) (oauth2.TokenSource, error) {
			return nil, handlerErr
		}

		transport, err := NewHTTPTransport(handler, nil)
		if err != nil {
			t.Fatalf("NewHTTPTransport() failed: %v", err)
		}
		client := &http.Client{Transport: transport}

		_, err = client.Get(authServer.URL)
		if !errors.Is(err, handlerErr) {
			t.Errorf("client.Get() returned error %v, want %v", err, handlerErr)
		}
	})
}
</content>
</file>
<file path="copyright_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package gosdk

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestCopyrightHeaders(t *testing.T) {
	var re = regexp.MustCompile(`Copyright \d{4} The Go MCP SDK Authors. All rights reserved.
Use of this source code is governed by an MIT-style
license that can be found in the LICENSE file.`)

	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories starting with "." or "_", and testdata directories.
		if d.IsDir() && d.Name() != "." &&
			(strings.HasPrefix(d.Name(), ".") ||
				strings.HasPrefix(d.Name(), "_") ||
				filepath.Base(d.Name()) == "testdata") {

			return filepath.SkipDir
		}

		// Skip non-go files.
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Check the copyright header.
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments|parser.PackageClauseOnly)
		if err != nil {
			return fmt.Errorf("parsing %s: %v", path, err)
		}
		if len(f.Comments) == 0 {
			t.Errorf("File %s must start with a copyright header matching %s", path, re)
		} else if !re.MatchString(f.Comments[0].Text()) {
			t.Errorf("Header comment for %s does not match expected copyright header.\ngot:\n%s\nwant matching:%s", path, f.Comments[0].Text(), re)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
</content>
</file>
<file path="examples/client/listfeatures/main.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// The listfeatures command lists all features of a stdio MCP server.
//
// Usage: listfeatures <command> [<args>]
//
// For example:
//
//	listfeatures go run github.com/modelcontextprotocol/go-sdk/examples/server/hello
//
// or
//
//	listfeatures npx @modelcontextprotocol/server-everything
package main

import (
	"context"
	"flag"
	"fmt"
	"iter"
	"log"
	"os"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	endpoint = flag.String("http", "", "if set, connect to this streamable endpoint rather than running a stdio server")
)

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 && *endpoint == "" {
		fmt.Fprintln(os.Stderr, "Usage: listfeatures <command> [<args>]")
		fmt.Fprintln(os.Stderr, "Usage: listfeatures --http=\"https://example.com/server/mcp\"")
		fmt.Fprintln(os.Stderr, "List all features for a stdio MCP server")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Example:\n\tlistfeatures npx @modelcontextprotocol/server-everything")
		os.Exit(2)
	}

	var (
		ctx       = context.Background()
		transport mcp.Transport
	)
	if *endpoint != "" {
		transport = &mcp.StreamableClientTransport{
			Endpoint: *endpoint,
		}
	} else {
		cmd := exec.Command(args[0], args[1:]...)
		transport = &mcp.CommandTransport{Command: cmd}
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-client", Version: "v1.0.0"}, nil)
	cs, err := client.Connect(ctx, transport, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer cs.Close()

	if cs.InitializeResult().Capabilities.Tools != nil {
		printSection("tools", cs.Tools(ctx, nil), func(t *mcp.Tool) string { return t.Name })
	}
	if cs.InitializeResult().Capabilities.Resources != nil {
		printSection("resources", cs.Resources(ctx, nil), func(r *mcp.Resource) string { return r.Name })
		printSection("resource templates", cs.ResourceTemplates(ctx, nil), func(r *mcp.ResourceTemplate) string { return r.Name })
	}
	if cs.InitializeResult().Capabilities.Prompts != nil {
		printSection("prompts", cs.Prompts(ctx, nil), func(p *mcp.Prompt) string { return p.Name })
	}
}

func printSection[T any](name string, features iter.Seq2[T, error], featName func(T) string) {
	fmt.Printf("%s:\n", name)
	for feat, err := range features {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("\t%s\n", featName(feat))
	}
	fmt.Println()
}
</content>
</file>
<file path="examples/client/loadtest/main.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// The load command load tests a streamable MCP server
//
// Usage: loadtest <URL>
//
// For example:
//
//	loadtest -tool=greet -args='{"name": "foo"}' http://localhost:8080
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	duration = flag.Duration("duration", 1*time.Minute, "duration of the load test")
	tool     = flag.String("tool", "", "tool to call")
	jsonArgs = flag.String("args", "", "JSON arguments to pass")
	workers  = flag.Int("workers", 10, "number of concurrent workers")
	timeout  = flag.Duration("timeout", 1*time.Second, "request timeout")
	qps      = flag.Int("qps", 100, "tool calls per second, per worker")
	verbose  = flag.Bool("v", false, "if set, enable verbose logging")
)

func main() {
	flag.Usage = func() {
		out := flag.CommandLine.Output()
		fmt.Fprintf(out, "Usage: loadtest [flags] <URL>")
		fmt.Fprintf(out, "Load test a streamable HTTP server (CTRL-C to end early)")
		fmt.Fprintln(out)
		fmt.Fprintf(out, "Example: loadtest -tool=greet -args='{\"name\": \"foo\"}' http://localhost:8080\n")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Flags:")
		flag.PrintDefaults()
	}
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 || *tool == "" {
		flag.Usage()
		os.Exit(2)
	}

	parentCtx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()
	parentCtx, stop := signal.NotifyContext(parentCtx, os.Interrupt)
	defer stop()

	var (
		start   = time.Now()
		success atomic.Int64
		failure atomic.Int64
	)

	// Run the test.
	var wg sync.WaitGroup
	for range *workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := mcp.NewClient(&mcp.Implementation{Name: "mcp-client", Version: "v1.0.0"}, nil)
			cs, err := client.Connect(parentCtx, &mcp.StreamableClientTransport{Endpoint: args[0]}, nil)
			if err != nil {
				log.Fatal(err)
			}
			defer cs.Close()

			ticker := time.NewTicker(1 * time.Second / time.Duration(*qps))
			defer ticker.Stop()

			for range ticker.C {
				ctx, cancel := context.WithTimeout(parentCtx, *timeout)
				defer cancel()

				res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: *tool, Arguments: json.RawMessage(*jsonArgs)})
				if err != nil {
					if parentCtx.Err() != nil {
						return // test ended
					}
					failure.Add(1)
					if *verbose {
						log.Printf("FAILURE: %v", err)
					}
				} else {
					success.Add(1)
					if *verbose {
						data, err := json.Marshal(res)
						if err != nil {
							log.Fatalf("marshalling result: %v", err)
						}
						log.Printf("SUCCESS: %s", string(data))
					}
				}
			}
		}()
	}
	wg.Wait()
	stop() // restore the interrupt signal

	// Print stats.
	var (
		dur  = time.Since(start)
		succ = success.Load()
		fail = failure.Load()
	)
	fmt.Printf("Results (in %s):\n", dur)
	fmt.Printf("\tsuccess: %d (%g QPS)\n", succ, float64(succ)/dur.Seconds())
	fmt.Printf("\tfailure: %d (%g QPS)\n", fail, float64(fail)/dur.Seconds())
}
</content>
</file>
<file path="examples/client/middleware/main.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"sync/atomic"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var nextProgressToken atomic.Int64

// This middleware function adds a progress token to every outgoing request
// from the client.
func main() {
	c := mcp.NewClient(&mcp.Implementation{Name: "test"}, nil)
	c.AddSendingMiddleware(addProgressToken)
}

func addProgressToken(h mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (result mcp.Result, err error) {
		if rp, ok := req.GetParams().(mcp.RequestParams); ok {
			rp.SetProgressToken(nextProgressToken.Add(1))
		}
		return h(ctx, method, req)
	}
}
</content>
</file>
<file path="examples/http/logging_middleware.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func loggingHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code.
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Log request details.
		log.Printf("[REQUEST] %s | %s | %s %s",
			start.Format(time.RFC3339),
			r.RemoteAddr,
			r.Method,
			r.URL.Path)

		// Call the actual handler.
		handler.ServeHTTP(wrapped, r)

		// Log response details.
		duration := time.Since(start)
		log.Printf("[RESPONSE] %s | %s | %s %s | Status: %d | Duration: %v",
			time.Now().Format(time.RFC3339),
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			duration)
	})
}
</content>
</file>
<file path="examples/http/main.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	host  = flag.String("host", "localhost", "host to connect to/listen on")
	port  = flag.Int("port", 8000, "port number to connect to/listen on")
	proto = flag.String("proto", "http", "if set, use as proto:// part of URL (ignored for server)")
)

func main() {
	out := flag.CommandLine.Output()
	flag.Usage = func() {
		fmt.Fprintf(out, "Usage: %s <client|server> [-proto <http|https>] [-port <port] [-host <host>]\n\n", os.Args[0])
		fmt.Fprintf(out, "This program demonstrates MCP over HTTP using the streamable transport.\n")
		fmt.Fprintf(out, "It can run as either a server or client.\n\n")
		fmt.Fprintf(out, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(out, "\nExamples:\n")
		fmt.Fprintf(out, "  Run as server:  %s server\n", os.Args[0])
		fmt.Fprintf(out, "  Run as client:  %s client\n", os.Args[0])
		fmt.Fprintf(out, "  Custom host/port: %s -port 9000 -host 0.0.0.0 server\n", os.Args[0])
		os.Exit(1)
	}
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintf(out, "Error: Must specify 'client' or 'server' as first argument\n")
		flag.Usage()
	}
	mode := flag.Arg(0)

	switch mode {
	case "server":
		if *proto != "http" {
			log.Fatalf("Server only works with 'http' (you passed proto=%s)", *proto)
		}
		runServer(fmt.Sprintf("%s:%d", *host, *port))
	case "client":
		runClient(fmt.Sprintf("%s://%s:%d", *proto, *host, *port))
	default:
		fmt.Fprintf(os.Stderr, "Error: Invalid mode '%s'. Must be 'client' or 'server'\n\n", mode)
		flag.Usage()
	}
}

// GetTimeParams defines the parameters for the cityTime tool.
type GetTimeParams struct {
	City string `json:"city" jsonschema:"City to get time for (nyc, sf, or boston)"`
}

// getTime implements the tool that returns the current time for a given city.
func getTime(ctx context.Context, req *mcp.CallToolRequest, params *GetTimeParams) (*mcp.CallToolResult, any, error) {
	// Define time zones for each city
	locations := map[string]string{
		"nyc":    "America/New_York",
		"sf":     "America/Los_Angeles",
		"boston": "America/New_York",
	}

	city := params.City
	if city == "" {
		city = "nyc" // Default to NYC
	}

	// Get the timezone.
	tzName, ok := locations[city]
	if !ok {
		return nil, nil, fmt.Errorf("unknown city: %s", city)
	}

	// Load the location.
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load timezone: %w", err)
	}

	// Get current time in that location.
	now := time.Now().In(loc)

	// Format the response.
	cityNames := map[string]string{
		"nyc":    "New York City",
		"sf":     "San Francisco",
		"boston": "Boston",
	}

	response := fmt.Sprintf("The current time in %s is %s",
		cityNames[city],
		now.Format(time.RFC3339))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: response},
		},
	}, nil, nil
}

func runServer(url string) {
	// Create an MCP server.
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "time-server",
		Version: "1.0.0",
	}, nil)

	// Add the cityTime tool.
	mcp.AddTool(server, &mcp.Tool{
		Name:        "cityTime",
		Description: "Get the current time in NYC, San Francisco, or Boston",
	}, getTime)

	// Create the streamable HTTP handler.
	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, nil)

	handlerWithLogging := loggingHandler(handler)

	log.Printf("MCP server listening on %s", url)
	log.Printf("Available tool: cityTime (cities: nyc, sf, boston)")

	// Start the HTTP server with logging handler.
	if err := http.ListenAndServe(url, handlerWithLogging); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func runClient(url string) {
	ctx := context.Background()

	// Create the URL for the server.
	log.Printf("Connecting to MCP server at %s", url)

	// Create an MCP client.
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "time-client",
		Version: "1.0.0",
	}, nil)

	// Connect to the server.
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: url}, nil)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer session.Close()

	log.Printf("Connected to server (session ID: %s)", session.ID())

	// First, list available tools.
	log.Println("Listing available tools...")
	toolsResult, err := session.ListTools(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}

	for _, tool := range toolsResult.Tools {
		log.Printf("  - %s: %s\n", tool.Name, tool.Description)
	}

	// Call the cityTime tool for each city.
	cities := []string{"nyc", "sf", "boston"}

	log.Println("Getting time for each city...")
	for _, city := range cities {
		// Call the tool.
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "cityTime",
			Arguments: map[string]any{
				"city": city,
			},
		})
		if err != nil {
			log.Printf("Failed to get time for %s: %v\n", city, err)
			continue
		}

		// Print the result.
		for _, content := range result.Content {
			if textContent, ok := content.(*mcp.TextContent); ok {
				log.Printf("  %s", textContent.Text)
			}
		}
	}

	log.Println("Client completed successfully")
}
</content>
</file>
<file path="examples/server/auth-middleware/main.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// This example demonstrates how to integrate auth.RequireBearerToken middleware
// with an MCP server to provide authenticated access to MCP tools and resources.

var httpAddr = flag.String("http", ":8080", "HTTP address to listen on")

// JWTClaims represents the claims in our JWT tokens.
// In a real application, you would include additional claims like issuer, audience, etc.
type JWTClaims struct {
	UserID string   `json:"user_id"` // User identifier
	Scopes []string `json:"scopes"`  // Permissions/roles for the user
	jwt.RegisteredClaims
}

// APIKey represents an API key with associated scopes.
// In production, this would be stored in a database with additional metadata.
type APIKey struct {
	Key    string   `json:"key"`     // The actual API key value
	UserID string   `json:"user_id"` // User identifier
	Scopes []string `json:"scopes"`  // Permissions/roles for this key
}

// In-memory storage for API keys (in production, use a database).
// This is for demonstration purposes only.
var apiKeys = map[string]*APIKey{
	"sk-1234567890abcdef": {
		Key:    "sk-1234567890abcdef",
		UserID: "user1",
		Scopes: []string{"read", "write"},
	},
	"sk-abcdef1234567890": {
		Key:    "sk-abcdef1234567890",
		UserID: "user2",
		Scopes: []string{"read"},
	},
}

// JWT secret (in production, use environment variables).
// This should be a strong, randomly generated secret in real applications.
var jwtSecret = []byte("your-secret-key")

// generateToken creates a JWT token for testing purposes.
// In a real application, this would be handled by your authentication service.
func generateToken(userID string, scopes []string, expiresIn time.Duration) (string, error) {
	// Create JWT claims with user information and scopes.
	claims := JWTClaims{
		UserID: userID,
		Scopes: scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)), // Token expiration
			IssuedAt:  jwt.NewNumericDate(time.Now()),                // Token issuance time
			NotBefore: jwt.NewNumericDate(time.Now()),                // Token validity start time
		},
	}

	// Create and sign the JWT token.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// verifyJWT verifies JWT tokens and returns TokenInfo for the auth middleware.
// This function implements the TokenVerifier interface required by auth.RequireBearerToken.
func verifyJWT(ctx context.Context, tokenString string, _ *http.Request) (*auth.TokenInfo, error) {
	// Parse and validate the JWT token.
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (any, error) {
		// Verify the signing method is HMAC.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})
	if err != nil {
		// Return standard error for invalid tokens.
		return nil, fmt.Errorf("%w: %v", auth.ErrInvalidToken, err)
	}

	// Extract claims and verify token validity.
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return &auth.TokenInfo{
			Scopes:     claims.Scopes,         // User permissions
			Expiration: claims.ExpiresAt.Time, // Token expiration time
		}, nil
	}

	return nil, fmt.Errorf("%w: invalid token claims", auth.ErrInvalidToken)
}

// verifyAPIKey verifies API keys and returns TokenInfo for the auth middleware.
// This function implements the TokenVerifier interface required by auth.RequireBearerToken.
func verifyAPIKey(ctx context.Context, apiKey string, _ *http.Request) (*auth.TokenInfo, error) {
	// Look up the API key in our storage.
	key, exists := apiKeys[apiKey]
	if !exists {
		return nil, fmt.Errorf("%w: API key not found", auth.ErrInvalidToken)
	}

	// API keys don't expire in this example, but you could add expiration logic here.
	// For demonstration, we set a 24-hour expiration.
	return &auth.TokenInfo{
		Scopes:     key.Scopes,                     // User permissions
		Expiration: time.Now().Add(24 * time.Hour), // 24 hour expiration
	}, nil
}

// MCP Tool Arguments
type getUserInfoArgs struct {
	UserID string `json:"user_id" jsonschema:"the user ID to get information for"`
}

type createResourceArgs struct {
	Name        string `json:"name" jsonschema:"the name of the resource"`
	Description string `json:"description" jsonschema:"the description of the resource"`
	Content     string `json:"content" jsonschema:"the content of the resource"`
}

// SayHi is a simple MCP tool that requires authentication
func SayHi(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, any, error) {
	// Extract user information from request (v0.3.0+)
	userInfo := req.Extra.TokenInfo

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Hello! You have scopes: %v", userInfo.Scopes)},
		},
	}, nil, nil
}

// GetUserInfo is an MCP tool that requires read scope
func GetUserInfo(ctx context.Context, req *mcp.CallToolRequest, args getUserInfoArgs) (*mcp.CallToolResult, any, error) {
	// Extract user information from request (v0.3.0+)
	userInfo := req.Extra.TokenInfo

	// Check if user has read scope.
	if !slices.Contains(userInfo.Scopes, "read") {
		return nil, nil, fmt.Errorf("insufficient permissions: read scope required")
	}

	userData := map[string]any{
		"requested_user_id": args.UserID,
		"your_scopes":       userInfo.Scopes,
		"message":           "User information retrieved successfully",
	}

	userDataJSON, err := json.Marshal(userData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal user data: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(userDataJSON)},
		},
	}, nil, nil
}

// CreateResource is an MCP tool that requires write scope
func CreateResource(ctx context.Context, req *mcp.CallToolRequest, args createResourceArgs) (*mcp.CallToolResult, any, error) {
	// Extract user information from request (v0.3.0+)
	userInfo := req.Extra.TokenInfo

	// Check if user has write scope.
	if !slices.Contains(userInfo.Scopes, "write") {
		return nil, nil, fmt.Errorf("insufficient permissions: write scope required")
	}

	resourceInfo := map[string]any{
		"name":        args.Name,
		"description": args.Description,
		"content":     args.Content,
		"created_by":  "authenticated_user",
		"created_at":  time.Now().Format(time.RFC3339),
	}

	resourceInfoJSON, err := json.Marshal(resourceInfo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal resource info: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Resource created successfully: %s", string(resourceInfoJSON))},
		},
	}, nil, nil
}

// createMCPServer creates an MCP server with authentication-aware tools
func createMCPServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "authenticated-mcp-server"}, nil)

	// Add tools that require authentication.
	mcp.AddTool(server, &mcp.Tool{
		Name:        "say_hi",
		Description: "A simple greeting tool that requires authentication",
	}, SayHi)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_user_info",
		Description: "Get user information (requires read scope)",
	}, GetUserInfo)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_resource",
		Description: "Create a new resource (requires write scope)",
	}, CreateResource)

	return server
}

func main() {
	flag.Parse()

	// Create the MCP server.
	server := createMCPServer()

	// Create authentication middleware.
	jwtAuth := auth.RequireBearerToken(verifyJWT, &auth.RequireBearerTokenOptions{
		Scopes: []string{"read"}, // Require "read" permission
	})

	apiKeyAuth := auth.RequireBearerToken(verifyAPIKey, &auth.RequireBearerTokenOptions{
		Scopes: []string{"read"}, // Require "read" permission
	})

	// Create HTTP handler with authentication.
	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, nil)

	// Apply authentication middleware to the MCP handler.
	authenticatedHandler := jwtAuth(handler)
	apiKeyHandler := apiKeyAuth(handler)

	// Create router for different authentication methods.
	http.HandleFunc("/mcp/jwt", authenticatedHandler.ServeHTTP)
	http.HandleFunc("/mcp/apikey", apiKeyHandler.ServeHTTP)

	// Add utility endpoints for token generation.
	http.HandleFunc("/generate-token", func(w http.ResponseWriter, r *http.Request) {
		// Get user ID from query parameters (default: "test-user").
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			userID = "test-user"
		}

		// Get scopes from query parameters (default: ["read", "write"]).
		scopes := strings.Split(r.URL.Query().Get("scopes"), ",")
		if len(scopes) == 1 && scopes[0] == "" {
			scopes = []string{"read", "write"}
		}

		// Get expiration time from query parameters (default: 1 hour).
		expiresIn := 1 * time.Hour
		if expStr := r.URL.Query().Get("expires_in"); expStr != "" {
			if exp, err := time.ParseDuration(expStr); err == nil {
				expiresIn = exp
			}
		}

		// Generate the JWT token.
		token, err := generateToken(userID, scopes, expiresIn)
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		// Return the generated token.
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"token": token,
			"type":  "Bearer",
		})
	})

	http.HandleFunc("/generate-api-key", func(w http.ResponseWriter, r *http.Request) {
		// Generate a random API key using cryptographically secure random bytes.
		bytes := make([]byte, 16)
		if _, err := rand.Read(bytes); err != nil {
			http.Error(w, "Failed to generate random bytes", http.StatusInternalServerError)
			return
		}
		apiKey := "sk-" + base64.URLEncoding.EncodeToString(bytes)

		// Get user ID from query parameters (default: "test-user").
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			userID = "test-user"
		}

		// Get scopes from query parameters (default: ["read"]).
		scopes := strings.Split(r.URL.Query().Get("scopes"), ",")
		if len(scopes) == 1 && scopes[0] == "" {
			scopes = []string{"read"}
		}

		// Store the new API key in our in-memory storage.
		// In production, this would be stored in a database.
		apiKeys[apiKey] = &APIKey{
			Key:    apiKey,
			UserID: userID,
			Scopes: scopes,
		}

		// Return the generated API key.
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"api_key": apiKey,
			"type":    "Bearer",
		})
	})

	// Health check endpoint.
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// Start the HTTP server.
	log.Println("Authenticated MCP Server")
	log.Println("========================")
	log.Println("Server starting on", *httpAddr)
	log.Println()
	log.Println("Available endpoints:")
	log.Println("  GET  /health                    - Health check (no auth)")
	log.Println("  GET  /generate-token            - Generate JWT token")
	log.Println("  POST /generate-api-key          - Generate API key")
	log.Println("  POST /mcp/jwt                   - MCP endpoint (JWT auth)")
	log.Println("  POST /mcp/apikey                - MCP endpoint (API key auth)")
	log.Println()
	log.Println("Available MCP Tools:")
	log.Println("  - say_hi                        - Simple greeting (any auth)")
	log.Println("  - get_user_info                 - Get user info (read scope)")
	log.Println("  - create_resource               - Create resource (write scope)")
	log.Println()
	log.Println("Example usage:")
	log.Println("  # Generate a token")
	log.Println("  curl 'http://localhost:8080/generate-token?user_id=alice&scopes=read,write'")
	log.Println()
	log.Println("  # Use MCP with JWT authentication")
	log.Println("  curl -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' \\")
	log.Println("       -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"say_hi\",\"arguments\":{}}}' \\")
	log.Println("       http://localhost:8080/mcp/jwt")
	log.Println()
	log.Println("  # Generate an API key")
	log.Println("  curl -X POST 'http://localhost:8080/generate-api-key?user_id=bob&scopes=read'")
	log.Println()
	log.Println("  # Use MCP with API key authentication")
	log.Println("  curl -H 'Authorization: Bearer <api_key>' -H 'Content-Type: application/json' \\")
	log.Println("       -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"get_user_info\",\"arguments\":{\"user_id\":\"test\"}}}' \\")
	log.Println("       http://localhost:8080/mcp/apikey")

	log.Fatal(http.ListenAndServe(*httpAddr, nil))
}
</content>
</file>
<file path="examples/server/basic/main.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type SayHiParams struct {
	Name string `json:"name"`
}

func SayHi(ctx context.Context, req *mcp.CallToolRequest, args SayHiParams) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Hi " + args.Name},
		},
	}, nil, nil
}

func main() {
	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	server := mcp.NewServer(&mcp.Implementation{Name: "greeter", Version: "v0.0.1"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "greet", Description: "say hi"}, SayHi)

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		log.Fatal(err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "client"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		log.Fatal(err)
	}

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "greet",
		Arguments: map[string]any{"name": "user"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res.Content[0].(*mcp.TextContent).Text)

	clientSession.Close()
	serverSession.Wait()

	// Output: Hi user
}
</content>
</file>
<file path="examples/server/completion/main.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// This example demonstrates the minimal code to declare and assign
// a CompletionHandler to an MCP Server's options.
func main() {
	// Define your custom CompletionHandler logic.
	// !+completionhandler
	myCompletionHandler := func(_ context.Context, req *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
		// In a real application, you'd implement actual completion logic here.
		// For this example, we return a fixed set of suggestions.
		var suggestions []string
		switch req.Params.Ref.Type {
		case "ref/prompt":
			suggestions = []string{"suggestion1", "suggestion2", "suggestion3"}
		case "ref/resource":
			suggestions = []string{"suggestion4", "suggestion5", "suggestion6"}
		default:
			return nil, fmt.Errorf("unrecognized content type %s", req.Params.Ref.Type)
		}

		return &mcp.CompleteResult{
			Completion: mcp.CompletionResultDetails{
				HasMore: false,
				Total:   len(suggestions),
				Values:  suggestions,
			},
		}, nil
	}

	// Create the MCP Server instance and assign the handler.
	// No server running, just showing the configuration.
	_ = mcp.NewServer(&mcp.Implementation{Name: "server"}, &mcp.ServerOptions{
		CompletionHandler: myCompletionHandler,
	})
	// !-completionhandler

	log.Println("MCP Server instance created with a CompletionHandler assigned (but not running).")
	log.Println("This example demonstrates configuration, not live interaction.")
}
</content>
</file>
<file path="examples/server/custom-transport/main.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// IOTransport is a simplified implementation of a transport that communicates using
// newline-delimited JSON over an io.Reader and io.Writer. It is similar to ioTransport
// in transport.go and serves as a demonstration of how to implement a custom transport.
type IOTransport struct {
	r *bufio.Reader
	w io.Writer
}

// NewIOTransport creates a new IOTransport with the given io.Reader and io.Writer.
func NewIOTransport(r io.Reader, w io.Writer) *IOTransport {
	return &IOTransport{
		r: bufio.NewReader(r),
		w: w,
	}
}

// ioConn is a connection that uses newlines to delimit messages. It implements [mcp.Connection].
type ioConn struct {
	r *bufio.Reader
	w io.Writer
}

// Connect implements [mcp.Transport.Connect] by creating a new ioConn.
func (t *IOTransport) Connect(ctx context.Context) (mcp.Connection, error) {
	return &ioConn{
		r: t.r,
		w: t.w,
	}, nil
}

// Read implements [mcp.Connection.Read], assuming messages are newline-delimited JSON.
func (t *ioConn) Read(context.Context) (jsonrpc.Message, error) {
	data, err := t.r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	return jsonrpc.DecodeMessage(data[:len(data)-1])
}

// Write implements [mcp.Connection.Write], appending a newline delimiter after the message.
func (t *ioConn) Write(_ context.Context, msg jsonrpc.Message) error {
	data, err := jsonrpc.EncodeMessage(msg)
	if err != nil {
		return err
	}

	_, err1 := t.w.Write(data)
	_, err2 := t.w.Write([]byte{'\n'})
	return errors.Join(err1, err2)
}

// Close implements [mcp.Connection.Close]. Since this is a simplified example, it is a no-op.
func (t *ioConn) Close() error {
	return nil
}

// SessionID implements [mcp.Connection.SessionID]. Since this is a simplified example,
// it returns an empty session ID.
func (t *ioConn) SessionID() string {
	return ""
}

// HiArgs is the argument type for the SayHi tool.
type HiArgs struct {
	Name string `json:"name" mcp:"the name to say hi to"`
}

// SayHi is a tool handler that responds with a greeting.
func SayHi(ctx context.Context, req *mcp.CallToolRequest, args HiArgs) (*mcp.CallToolResult, struct{}, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Hi " + args.Name},
		},
	}, struct{}{}, nil
}

func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: "greeter"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "greet", Description: "say hi"}, SayHi)

	// Run the server with a custom IOTransport using stdio as the io.Reader and io.Writer.
	transport := &IOTransport{
		r: bufio.NewReader(os.Stdin),
		w: os.Stdout,
	}
	err := server.Run(context.Background(), transport)
	if err != nil {
		log.Println("[ERROR]: Failed to run server:", err)
	}
}
</content>
</file>
<file path="examples/server/distributed/main.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// The distributed command is an example of a distributed MCP server.
//
// It forks multiple child processes (according to the -child_ports flag), each
// of which is a streamable HTTP MCP server with the 'inc' tool, and proxies
// incoming http requests to them.
//
// Distributed MCP servers must be stateless, because there's no guarantee that
// subsequent requests for a session land on the same backend. However, they
// may still have logical session IDs, as can be seen with verbose logging
// (-v).
//
// Example:
//
//	./distributed -http=localhost:8080 -child_ports=8081,8082
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const childPortVar = "MCP_CHILD_PORT"

var (
	httpAddr   = flag.String("http", "", "if set, use streamable HTTP at this address, instead of stdin/stdout")
	childPorts = flag.String("child_ports", "", "comma-separated child ports to distribute to")
	verbose    = flag.Bool("v", false, "if set, enable verbose logging")
)

func main() {
	// This command runs as either a parent or a child, depending on whether
	// childPortVar is set (a.k.a. the fork-and-exec trick).
	//
	// Each child is a streamable HTTP server, and the parent is a reverse proxy.
	flag.Parse()
	if v := os.Getenv(childPortVar); v != "" {
		child(v)
	} else {
		parent()
	}
}

func parent() {
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}

	if *httpAddr == "" {
		log.Fatal("must provide -http")
	}
	if *childPorts == "" {
		log.Fatal("must provide -child_ports")
	}

	// Ensure that children are cleaned up on CTRL-C
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Start the child processes.
	ports := strings.Split(*childPorts, ",")
	var wg sync.WaitGroup
	childURLs := make([]*url.URL, len(ports))
	for i, port := range ports {
		childURL := fmt.Sprintf("http://localhost:%s", port)
		childURLs[i], err = url.Parse(childURL)
		if err != nil {
			log.Fatal(err)
		}
		cmd := exec.CommandContext(ctx, exe, os.Args[1:]...)
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", childPortVar, port))
		cmd.Stderr = os.Stderr

		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Printf("starting child %d at %s", i, childURL)
			if err := cmd.Run(); err != nil && ctx.Err() == nil {
				log.Printf("child %d failed: %v", i, err)
			} else {
				log.Printf("child %d exited gracefully", i)
			}
		}()
	}

	// Start a reverse proxy that round-robin's requests to each backend.
	var nextBackend atomic.Int64
	server := http.Server{
		Addr: *httpAddr,
		Handler: &httputil.ReverseProxy{
			Rewrite: func(r *httputil.ProxyRequest) {
				child := int(nextBackend.Add(1)) % len(childURLs)
				if *verbose {
					log.Printf("dispatching to localhost:%s", ports[child])
				}
				r.SetURL(childURLs[child])
			},
		},
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.ListenAndServe(); err != nil && ctx.Err() == nil {
			log.Printf("Server failed: %v", err)
		}
	}()

	log.Printf("Serving at %s (CTRL-C to cancel)", *httpAddr)

	<-ctx.Done()
	stop() // restore the interrupt signal

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt the graceful shutdown.
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	// Wait for the subprocesses and http server to stop.
	wg.Wait()

	log.Println("Server shutdown gracefully.")
}

func child(port string) {
	// Create a server with a single tool that increments a counter and sends a notification.
	server := mcp.NewServer(&mcp.Implementation{Name: "counter"}, nil)

	var count atomic.Int64
	inc := func(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, struct{ Count int64 }, error) {
		n := count.Add(1)
		if *verbose {
			log.Printf("request %d (session %s)", n, req.Session.ID())
		}
		// Send a notification in the context of the request
		// Hint: in stateless mode, at least log level 'info' is required to send notifications
		req.Session.Log(ctx, &mcp.LoggingMessageParams{Data: fmt.Sprintf("request %d (session %s)", n, req.Session.ID()), Level: "info"})
		return nil, struct{ Count int64 }{n}, nil
	}
	mcp.AddTool(server, &mcp.Tool{Name: "inc"}, inc)

	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		Stateless: true,
	})
	log.Printf("child listening on localhost:%s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("localhost:%s", port), handler))
}
</content>
</file>
<file path="examples/server/elicitation/main.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	// Create server
	server := mcp.NewServer(&mcp.Implementation{Name: "config-server", Version: "v1.0.0"}, nil)

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Create client with elicitation handler
	// Note: Never use elicitation for sensitive data like API keys or passwords
	client := mcp.NewClient(&mcp.Implementation{Name: "config-client", Version: "v1.0.0"}, &mcp.ClientOptions{
		ElicitationHandler: func(ctx context.Context, request *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
			fmt.Printf("Server requests: %s\n", request.Params.Message)

			// In a real application, this would prompt the user for input
			// Here we simulate user providing configuration data
			return &mcp.ElicitResult{
				Action: "accept",
				Content: map[string]any{
					"serverEndpoint": "https://api.example.com",
					"maxRetries":     float64(3),
					"enableLogs":     true,
				},
			}, nil
		},
	})

	_, err = client.Connect(ctx, clientTransport, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Server requests user configuration via elicitation
	configSchema := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"serverEndpoint": {Type: "string", Description: "Server endpoint URL"},
			"maxRetries":     {Type: "number", Minimum: ptr(1.0), Maximum: ptr(10.0)},
			"enableLogs":     {Type: "boolean", Description: "Enable debug logging"},
		},
		Required: []string{"serverEndpoint"},
	}

	result, err := serverSession.Elicit(ctx, &mcp.ElicitParams{
		Message:         "Please provide your configuration settings",
		RequestedSchema: configSchema,
	})
	if err != nil {
		log.Fatal(err)
	}

	if result.Action == "accept" {
		fmt.Printf("Configuration received: Endpoint: %v, Max Retries: %.0f, Logs: %v\n",
			result.Content["serverEndpoint"],
			result.Content["maxRetries"],
			result.Content["enableLogs"])
	}

	// Output:
	// Server requests: Please provide your configuration settings
	// Configuration received: Endpoint: https://api.example.com, Max Retries: 3, Logs: true
}

// ptr is a helper function to create pointers for schema constraints
func ptr[T any](v T) *T {
	return &v
}
</content>
</file>
<file path="examples/server/everything/main.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// The everything server implements all supported features of an MCP server.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"runtime"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	httpAddr  = flag.String("http", "", "if set, use streamable HTTP at this address, instead of stdin/stdout")
	pprofAddr = flag.String("pprof", "", "if set, host the pprof debugging server at this address")
)

func main() {
	flag.Parse()

	if *pprofAddr != "" {
		// For debugging memory leaks, add an endpoint to trigger a few garbage
		// collection cycles and ensure the /debug/pprof/heap endpoint only reports
		// reachable memory.
		http.DefaultServeMux.Handle("/gc", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			for range 3 {
				runtime.GC()
			}
			fmt.Fprintln(w, "GC'ed")
		}))
		go func() {
			// DefaultServeMux was mutated by the /debug/pprof import.
			http.ListenAndServe(*pprofAddr, http.DefaultServeMux)
		}()
	}

	opts := &mcp.ServerOptions{
		Instructions:      "Use this server!",
		CompletionHandler: complete, // support completions by setting this handler
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "everything"}, opts)

	// Add tools that exercise different features of the protocol.
	mcp.AddTool(server, &mcp.Tool{Name: "greet", Description: "say hi"}, contentTool)
	mcp.AddTool(server, &mcp.Tool{Name: "greet (structured)"}, structuredTool) // returns structured output
	mcp.AddTool(server, &mcp.Tool{Name: "ping"}, pingingTool)                  // performs a ping
	mcp.AddTool(server, &mcp.Tool{Name: "log"}, loggingTool)                   // performs a log
	mcp.AddTool(server, &mcp.Tool{Name: "sample"}, samplingTool)               // performs sampling
	mcp.AddTool(server, &mcp.Tool{Name: "elicit"}, elicitingTool)              // performs elicitation
	mcp.AddTool(server, &mcp.Tool{Name: "roots"}, rootsTool)                   // lists roots

	// Add a basic prompt.
	server.AddPrompt(&mcp.Prompt{Name: "greet"}, prompt)

	// Add an embedded resource.
	server.AddResource(&mcp.Resource{
		Name:     "info",
		MIMEType: "text/plain",
		URI:      "embedded:info",
	}, embeddedResource)

	// Serve over stdio, or streamable HTTP if -http is set.
	if *httpAddr != "" {
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return server
		}, nil)
		log.Printf("MCP handler listening at %s", *httpAddr)
		if *pprofAddr != "" {
			log.Printf("pprof listening at http://%s/debug/pprof", *pprofAddr)
		}
		log.Fatal(http.ListenAndServe(*httpAddr, handler))
	} else {
		t := &mcp.LoggingTransport{Transport: &mcp.StdioTransport{}, Writer: os.Stderr}
		if err := server.Run(context.Background(), t); err != nil {
			log.Printf("Server failed: %v", err)
		}
	}
}

func prompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return &mcp.GetPromptResult{
		Description: "Hi prompt",
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: "Say hi to " + req.Params.Arguments["name"]},
			},
		},
	}, nil
}

var embeddedResources = map[string]string{
	"info": "This is the hello example server.",
}

func embeddedResource(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	u, err := url.Parse(req.Params.URI)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "embedded" {
		return nil, fmt.Errorf("wrong scheme: %q", u.Scheme)
	}
	key := u.Opaque
	text, ok := embeddedResources[key]
	if !ok {
		return nil, fmt.Errorf("no embedded resource named %q", key)
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{URI: req.Params.URI, MIMEType: "text/plain", Text: text},
		},
	}, nil
}

type args struct {
	Name string `json:"name" jsonschema:"the name to say hi to"`
}

// contentTool is a tool that returns unstructured content.
//
// Since its output type is 'any', no output schema is created.
func contentTool(ctx context.Context, req *mcp.CallToolRequest, args args) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Hi " + args.Name},
		},
	}, nil, nil
}

type result struct {
	Message string `json:"message" jsonschema:"the message to convey"`
}

// structuredTool returns a structured result.
func structuredTool(ctx context.Context, req *mcp.CallToolRequest, args *args) (*mcp.CallToolResult, *result, error) {
	return nil, &result{Message: "Hi " + args.Name}, nil
}

func pingingTool(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	if err := req.Session.Ping(ctx, nil); err != nil {
		return nil, nil, fmt.Errorf("ping failed")
	}
	return nil, nil, nil
}

func loggingTool(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	if err := req.Session.Log(ctx, &mcp.LoggingMessageParams{
		Data:  "something happened!",
		Level: "error",
	}); err != nil {
		return nil, nil, fmt.Errorf("log failed")
	}
	return nil, nil, nil
}

func rootsTool(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	res, err := req.Session.ListRoots(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("listing roots failed: %v", err)
	}
	var allroots []string
	for _, r := range res.Roots {
		allroots = append(allroots, fmt.Sprintf("%s:%s", r.Name, r.URI))
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: strings.Join(allroots, ",")},
		},
	}, nil, nil
}

func samplingTool(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	res, err := req.Session.CreateMessage(ctx, new(mcp.CreateMessageParams))
	if err != nil {
		return nil, nil, fmt.Errorf("sampling failed: %v", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			res.Content,
		},
	}, nil, nil
}

func elicitingTool(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	res, err := req.Session.Elicit(ctx, &mcp.ElicitParams{
		Message: "provide a random string",
		RequestedSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"random": {Type: "string"},
			},
		},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("eliciting failed: %v", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: res.Content["random"].(string)},
		},
	}, nil, nil
}

func complete(ctx context.Context, req *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	return &mcp.CompleteResult{
		Completion: mcp.CompletionResultDetails{
			Total:  1,
			Values: []string{req.Params.Argument.Value + "x"},
		},
	}, nil
}
</content>
</file>
<file path="examples/server/hello/main.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// The hello server contains a single tool that says hi to the user.
//
// It runs over the stdio transport.
package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	// Create a server with a single tool that says "Hi".
	server := mcp.NewServer(&mcp.Implementation{Name: "greeter"}, nil)

	// Using the generic AddTool automatically populates the the input and output
	// schema of the tool.
	//
	// The schema considers 'json' and 'jsonschema' struct tags to get argument
	// names and descriptions.
	type args struct {
		Name string `json:"name" jsonschema:"the person to greet"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "greet",
		Description: "say hi",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args args) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "Hi " + args.Name},
			},
		}, nil, nil
	})

	// server.Run runs the server on the given transport.
	//
	// In this case, the server communicates over stdin/stdout.
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Printf("Server failed: %v", err)
	}
}
</content>
</file>
<file path="examples/server/memory/kb.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Entity represents a knowledge graph node with observations.
type Entity struct {
	Name         string   `json:"name"`
	EntityType   string   `json:"entityType"`
	Observations []string `json:"observations"`
}

// Relation represents a directed edge between two entities.
type Relation struct {
	From         string `json:"from"`
	To           string `json:"to"`
	RelationType string `json:"relationType"`
}

// Observation contains facts about an entity.
type Observation struct {
	EntityName string   `json:"entityName"`
	Contents   []string `json:"contents"`

	Observations []string `json:"observations,omitempty"` // Used for deletion operations
}

// KnowledgeGraph represents the complete graph structure.
type KnowledgeGraph struct {
	Entities  []Entity   `json:"entities"`
	Relations []Relation `json:"relations"`
}

// store provides persistence interface for knowledge base data.
type store interface {
	Read() ([]byte, error)
	Write(data []byte) error
}

// memoryStore implements in-memory storage that doesn't persist across restarts.
type memoryStore struct {
	data []byte
}

// Read returns the in-memory data.
func (ms *memoryStore) Read() ([]byte, error) {
	return ms.data, nil
}

// Write stores data in memory.
func (ms *memoryStore) Write(data []byte) error {
	ms.data = data
	return nil
}

// fileStore implements file-based storage for persistent knowledge base.
type fileStore struct {
	path string
}

// Read loads data from file, returning empty slice if file doesn't exist.
func (fs *fileStore) Read() ([]byte, error) {
	data, err := os.ReadFile(fs.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read file %s: %w", fs.path, err)
	}
	return data, nil
}

// Write saves data to file with 0600 permissions.
func (fs *fileStore) Write(data []byte) error {
	if err := os.WriteFile(fs.path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write file %s: %w", fs.path, err)
	}
	return nil
}

// knowledgeBase manages entities and relations with persistent storage.
type knowledgeBase struct {
	s store
}

// kbItem represents a single item in persistent storage (entity or relation).
type kbItem struct {
	Type string `json:"type"`

	// Entity fields (when Type == "entity")
	Name         string   `json:"name,omitempty"`
	EntityType   string   `json:"entityType,omitempty"`
	Observations []string `json:"observations,omitempty"`

	// Relation fields (when Type == "relation")
	From         string `json:"from,omitempty"`
	To           string `json:"to,omitempty"`
	RelationType string `json:"relationType,omitempty"`
}

// loadGraph deserializes the knowledge graph from storage.
func (k knowledgeBase) loadGraph() (KnowledgeGraph, error) {
	data, err := k.s.Read()
	if err != nil {
		return KnowledgeGraph{}, fmt.Errorf("failed to read from store: %w", err)
	}

	if len(data) == 0 {
		return KnowledgeGraph{}, nil
	}

	var items []kbItem
	if err := json.Unmarshal(data, &items); err != nil {
		return KnowledgeGraph{}, fmt.Errorf("failed to unmarshal from store: %w", err)
	}

	graph := KnowledgeGraph{}

	for _, item := range items {
		switch item.Type {
		case "entity":
			graph.Entities = append(graph.Entities, Entity{
				Name:         item.Name,
				EntityType:   item.EntityType,
				Observations: item.Observations,
			})
		case "relation":
			graph.Relations = append(graph.Relations, Relation{
				From:         item.From,
				To:           item.To,
				RelationType: item.RelationType,
			})
		}
	}

	return graph, nil
}

// saveGraph serializes and persists the knowledge graph to storage.
func (k knowledgeBase) saveGraph(graph KnowledgeGraph) error {
	items := make([]kbItem, 0, len(graph.Entities)+len(graph.Relations))

	for _, entity := range graph.Entities {
		items = append(items, kbItem{
			Type:         "entity",
			Name:         entity.Name,
			EntityType:   entity.EntityType,
			Observations: entity.Observations,
		})
	}

	for _, relation := range graph.Relations {
		items = append(items, kbItem{
			Type:         "relation",
			From:         relation.From,
			To:           relation.To,
			RelationType: relation.RelationType,
		})
	}

	itemsJSON, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("failed to marshal items: %w", err)
	}

	if err := k.s.Write(itemsJSON); err != nil {
		return fmt.Errorf("failed to write to store: %w", err)
	}
	return nil
}

// createEntities adds new entities to the graph, skipping duplicates by name.
// It returns the new entities that were actually added.
func (k knowledgeBase) createEntities(entities []Entity) ([]Entity, error) {
	graph, err := k.loadGraph()
	if err != nil {
		return nil, err
	}

	var newEntities []Entity
	for _, entity := range entities {
		if !slices.ContainsFunc(graph.Entities, func(e Entity) bool { return e.Name == entity.Name }) {
			newEntities = append(newEntities, entity)
			graph.Entities = append(graph.Entities, entity)
		}
	}

	if err := k.saveGraph(graph); err != nil {
		return nil, err
	}

	return newEntities, nil
}

// createRelations adds new relations to the graph, skipping exact duplicates.
// It returns the new relations that were actually added.
func (k knowledgeBase) createRelations(relations []Relation) ([]Relation, error) {
	graph, err := k.loadGraph()
	if err != nil {
		return nil, err
	}

	var newRelations []Relation
	for _, relation := range relations {
		exists := slices.ContainsFunc(graph.Relations, func(r Relation) bool {
			return r.From == relation.From &&
				r.To == relation.To &&
				r.RelationType == relation.RelationType
		})
		if !exists {
			newRelations = append(newRelations, relation)
			graph.Relations = append(graph.Relations, relation)
		}
	}

	if err := k.saveGraph(graph); err != nil {
		return nil, err
	}

	return newRelations, nil
}

// addObservations appends new observations to existing entities.
// It returns the new observations that were actually added.
func (k knowledgeBase) addObservations(observations []Observation) ([]Observation, error) {
	graph, err := k.loadGraph()
	if err != nil {
		return nil, err
	}

	var results []Observation

	for _, obs := range observations {
		entityIndex := slices.IndexFunc(graph.Entities, func(e Entity) bool { return e.Name == obs.EntityName })
		if entityIndex == -1 {
			return nil, fmt.Errorf("entity with name %s not found", obs.EntityName)
		}

		var newObservations []string
		for _, content := range obs.Contents {
			if !slices.Contains(graph.Entities[entityIndex].Observations, content) {
				newObservations = append(newObservations, content)
				graph.Entities[entityIndex].Observations = append(graph.Entities[entityIndex].Observations, content)
			}
		}

		results = append(results, Observation{
			EntityName: obs.EntityName,
			Contents:   newObservations,
		})
	}

	if err := k.saveGraph(graph); err != nil {
		return nil, err
	}

	return results, nil
}

// deleteEntities removes entities and their associated relations.
func (k knowledgeBase) deleteEntities(entityNames []string) error {
	graph, err := k.loadGraph()
	if err != nil {
		return err
	}

	// Create map for quick lookup
	entitiesToDelete := make(map[string]bool)
	for _, name := range entityNames {
		entitiesToDelete[name] = true
	}

	// Filter entities using slices.DeleteFunc
	graph.Entities = slices.DeleteFunc(graph.Entities, func(entity Entity) bool {
		return entitiesToDelete[entity.Name]
	})

	// Filter relations using slices.DeleteFunc
	graph.Relations = slices.DeleteFunc(graph.Relations, func(relation Relation) bool {
		return entitiesToDelete[relation.From] || entitiesToDelete[relation.To]
	})

	return k.saveGraph(graph)
}

// deleteObservations removes specific observations from entities.
func (k knowledgeBase) deleteObservations(deletions []Observation) error {
	graph, err := k.loadGraph()
	if err != nil {
		return err
	}

	for _, deletion := range deletions {
		entityIndex := slices.IndexFunc(graph.Entities, func(e Entity) bool {
			return e.Name == deletion.EntityName
		})
		if entityIndex == -1 {
			continue
		}

		// Create a map for quick lookup
		observationsToDelete := make(map[string]bool)
		for _, observation := range deletion.Observations {
			observationsToDelete[observation] = true
		}

		// Filter observations using slices.DeleteFunc
		graph.Entities[entityIndex].Observations = slices.DeleteFunc(graph.Entities[entityIndex].Observations, func(observation string) bool {
			return observationsToDelete[observation]
		})
	}

	return k.saveGraph(graph)
}

// deleteRelations removes specific relations from the graph.
func (k knowledgeBase) deleteRelations(relations []Relation) error {
	graph, err := k.loadGraph()
	if err != nil {
		return err
	}

	// Filter relations using slices.DeleteFunc and slices.ContainsFunc
	graph.Relations = slices.DeleteFunc(graph.Relations, func(existingRelation Relation) bool {
		return slices.ContainsFunc(relations, func(relationToDelete Relation) bool {
			return existingRelation.From == relationToDelete.From &&
				existingRelation.To == relationToDelete.To &&
				existingRelation.RelationType == relationToDelete.RelationType
		})
	})
	return k.saveGraph(graph)
}

// searchNodes filters entities and relations matching the query string.
func (k knowledgeBase) searchNodes(query string) (KnowledgeGraph, error) {
	graph, err := k.loadGraph()
	if err != nil {
		return KnowledgeGraph{}, err
	}

	queryLower := strings.ToLower(query)
	var filteredEntities []Entity

	// Filter entities
	for _, entity := range graph.Entities {
		if strings.Contains(strings.ToLower(entity.Name), queryLower) ||
			strings.Contains(strings.ToLower(entity.EntityType), queryLower) {
			filteredEntities = append(filteredEntities, entity)
			continue
		}

		// Check observations
		for _, observation := range entity.Observations {
			if strings.Contains(strings.ToLower(observation), queryLower) {
				filteredEntities = append(filteredEntities, entity)
				break
			}
		}
	}

	// Create map for quick entity lookup
	filteredEntityNames := make(map[string]bool)
	for _, entity := range filteredEntities {
		filteredEntityNames[entity.Name] = true
	}

	// Filter relations
	var filteredRelations []Relation
	for _, relation := range graph.Relations {
		if filteredEntityNames[relation.From] && filteredEntityNames[relation.To] {
			filteredRelations = append(filteredRelations, relation)
		}
	}

	return KnowledgeGraph{
		Entities:  filteredEntities,
		Relations: filteredRelations,
	}, nil
}

// openNodes returns entities with specified names and their interconnecting relations.
func (k knowledgeBase) openNodes(names []string) (KnowledgeGraph, error) {
	graph, err := k.loadGraph()
	if err != nil {
		return KnowledgeGraph{}, err
	}

	// Create map for quick name lookup
	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}

	// Filter entities
	var filteredEntities []Entity
	for _, entity := range graph.Entities {
		if nameSet[entity.Name] {
			filteredEntities = append(filteredEntities, entity)
		}
	}

	// Create map for quick entity lookup
	filteredEntityNames := make(map[string]bool)
	for _, entity := range filteredEntities {
		filteredEntityNames[entity.Name] = true
	}

	// Filter relations
	var filteredRelations []Relation
	for _, relation := range graph.Relations {
		if filteredEntityNames[relation.From] && filteredEntityNames[relation.To] {
			filteredRelations = append(filteredRelations, relation)
		}
	}

	return KnowledgeGraph{
		Entities:  filteredEntities,
		Relations: filteredRelations,
	}, nil
}

func (k knowledgeBase) CreateEntities(ctx context.Context, req *mcp.CallToolRequest, args CreateEntitiesArgs) (*mcp.CallToolResult, CreateEntitiesResult, error) {
	var res mcp.CallToolResult

	entities, err := k.createEntities(args.Entities)
	if err != nil {
		return nil, CreateEntitiesResult{}, err
	}

	res.Content = []mcp.Content{
		&mcp.TextContent{Text: "Entities created successfully"},
	}
	return &res, CreateEntitiesResult{Entities: entities}, nil
}

func (k knowledgeBase) CreateRelations(ctx context.Context, req *mcp.CallToolRequest, args CreateRelationsArgs) (*mcp.CallToolResult, CreateRelationsResult, error) {
	var res mcp.CallToolResult

	relations, err := k.createRelations(args.Relations)
	if err != nil {
		return nil, CreateRelationsResult{}, err
	}

	res.Content = []mcp.Content{
		&mcp.TextContent{Text: "Relations created successfully"},
	}

	return &res, CreateRelationsResult{Relations: relations}, nil
}

func (k knowledgeBase) AddObservations(ctx context.Context, req *mcp.CallToolRequest, args AddObservationsArgs) (*mcp.CallToolResult, AddObservationsResult, error) {
	var res mcp.CallToolResult

	observations, err := k.addObservations(args.Observations)
	if err != nil {
		return nil, AddObservationsResult{}, err
	}

	res.Content = []mcp.Content{
		&mcp.TextContent{Text: "Observations added successfully"},
	}

	return &res, AddObservationsResult{
		Observations: observations,
	}, nil
}

func (k knowledgeBase) DeleteEntities(ctx context.Context, req *mcp.CallToolRequest, args DeleteEntitiesArgs) (*mcp.CallToolResult, any, error) {
	var res mcp.CallToolResult

	err := k.deleteEntities(args.EntityNames)
	if err != nil {
		return nil, nil, err
	}

	res.Content = []mcp.Content{
		&mcp.TextContent{Text: "Entities deleted successfully"},
	}

	return &res, nil, nil
}

func (k knowledgeBase) DeleteObservations(ctx context.Context, req *mcp.CallToolRequest, args DeleteObservationsArgs) (*mcp.CallToolResult, any, error) {
	var res mcp.CallToolResult

	err := k.deleteObservations(args.Deletions)
	if err != nil {
		return nil, nil, err
	}

	res.Content = []mcp.Content{
		&mcp.TextContent{Text: "Observations deleted successfully"},
	}

	return &res, nil, nil
}

func (k knowledgeBase) DeleteRelations(ctx context.Context, req *mcp.CallToolRequest, args DeleteRelationsArgs) (*mcp.CallToolResult, struct{}, error) {
	var res mcp.CallToolResult

	err := k.deleteRelations(args.Relations)
	if err != nil {
		return nil, struct{}{}, err
	}

	res.Content = []mcp.Content{
		&mcp.TextContent{Text: "Relations deleted successfully"},
	}

	return &res, struct{}{}, nil
}

func (k knowledgeBase) ReadGraph(ctx context.Context, req *mcp.CallToolRequest, args any) (*mcp.CallToolResult, KnowledgeGraph, error) {
	var res mcp.CallToolResult

	graph, err := k.loadGraph()
	if err != nil {
		return nil, KnowledgeGraph{}, err
	}

	res.Content = []mcp.Content{
		&mcp.TextContent{Text: "Graph read successfully"},
	}

	return &res, graph, nil
}

func (k knowledgeBase) SearchNodes(ctx context.Context, req *mcp.CallToolRequest, args SearchNodesArgs) (*mcp.CallToolResult, KnowledgeGraph, error) {
	var res mcp.CallToolResult

	graph, err := k.searchNodes(args.Query)
	if err != nil {
		return nil, KnowledgeGraph{}, err
	}

	res.Content = []mcp.Content{
		&mcp.TextContent{Text: "Nodes searched successfully"},
	}

	return &res, graph, nil
}

func (k knowledgeBase) OpenNodes(ctx context.Context, req *mcp.CallToolRequest, args OpenNodesArgs) (*mcp.CallToolResult, KnowledgeGraph, error) {
	var res mcp.CallToolResult

	graph, err := k.openNodes(args.Names)
	if err != nil {
		return nil, KnowledgeGraph{}, err
	}

	res.Content = []mcp.Content{
		&mcp.TextContent{Text: "Nodes opened successfully"},
	}
	return &res, graph, nil
}
</content>
</file>
<file path="examples/server/memory/kb_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// stores provides test factories for both storage implementations.
func stores() map[string]func(t *testing.T) store {
	return map[string]func(t *testing.T) store{
		"file": func(t *testing.T) store {
			tempDir, err := os.MkdirTemp("", "kb-test-file-*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			t.Cleanup(func() { os.RemoveAll(tempDir) })
			return &fileStore{path: filepath.Join(tempDir, "test-memory.json")}
		},
		"memory": func(t *testing.T) store {
			return &memoryStore{}
		},
	}
}

// TestKnowledgeBaseOperations verifies CRUD operations work correctly.
func TestKnowledgeBaseOperations(t *testing.T) {
	for name, newStore := range stores() {
		t.Run(name, func(t *testing.T) {
			s := newStore(t)
			kb := knowledgeBase{s: s}

			// Verify empty graph loads correctly
			graph, err := kb.loadGraph()
			if err != nil {
				t.Fatalf("failed to load empty graph: %v", err)
			}
			if len(graph.Entities) != 0 || len(graph.Relations) != 0 {
				t.Errorf("expected empty graph, got %+v", graph)
			}

			// Create and verify entities
			testEntities := []Entity{
				{
					Name:         "Alice",
					EntityType:   "Person",
					Observations: []string{"Likes coffee"},
				},
				{
					Name:         "Bob",
					EntityType:   "Person",
					Observations: []string{"Likes tea"},
				},
			}

			createdEntities, err := kb.createEntities(testEntities)
			if err != nil {
				t.Fatalf("failed to create entities: %v", err)
			}
			if len(createdEntities) != 2 {
				t.Errorf("expected 2 created entities, got %d", len(createdEntities))
			}

			// Verify entities persist
			graph, err = kb.loadGraph()
			if err != nil {
				t.Fatalf("failed to read graph: %v", err)
			}
			if len(graph.Entities) != 2 {
				t.Errorf("expected 2 entities, got %d", len(graph.Entities))
			}

			// Create and verify relations
			testRelations := []Relation{
				{
					From:         "Alice",
					To:           "Bob",
					RelationType: "friend",
				},
			}

			createdRelations, err := kb.createRelations(testRelations)
			if err != nil {
				t.Fatalf("failed to create relations: %v", err)
			}
			if len(createdRelations) != 1 {
				t.Errorf("expected 1 created relation, got %d", len(createdRelations))
			}

			// Add observations to entities
			testObservations := []Observation{
				{
					EntityName: "Alice",
					Contents:   []string{"Works as developer", "Lives in New York"},
				},
			}

			addedObservations, err := kb.addObservations(testObservations)
			if err != nil {
				t.Fatalf("failed to add observations: %v", err)
			}
			if len(addedObservations) != 1 || len(addedObservations[0].Contents) != 2 {
				t.Errorf("expected 1 observation with 2 contents, got %+v", addedObservations)
			}

			// Search nodes by content
			searchResult, err := kb.searchNodes("developer")
			if err != nil {
				t.Fatalf("failed to search nodes: %v", err)
			}
			if len(searchResult.Entities) != 1 || searchResult.Entities[0].Name != "Alice" {
				t.Errorf("expected to find Alice when searching for 'developer', got %+v", searchResult)
			}

			// Retrieve specific nodes
			openResult, err := kb.openNodes([]string{"Bob"})
			if err != nil {
				t.Fatalf("failed to open nodes: %v", err)
			}
			if len(openResult.Entities) != 1 || openResult.Entities[0].Name != "Bob" {
				t.Errorf("expected to find Bob when opening 'Bob', got %+v", openResult)
			}

			// Remove specific observations
			deleteObs := []Observation{
				{
					EntityName:   "Alice",
					Observations: []string{"Works as developer"},
				},
			}
			err = kb.deleteObservations(deleteObs)
			if err != nil {
				t.Fatalf("failed to delete observations: %v", err)
			}

			// Confirm observation removal
			graph, _ = kb.loadGraph()
			aliceIndex := slices.IndexFunc(graph.Entities, func(e Entity) bool {
				return e.Name == "Alice"
			})
			if aliceIndex == -1 {
				t.Errorf("entity 'Alice' not found after deleting observation")
			} else {
				alice := graph.Entities[aliceIndex]
				if slices.Contains(alice.Observations, "Works as developer") {
					t.Errorf("observation 'Works as developer' should have been deleted")
				}
			}

			// Remove relations
			err = kb.deleteRelations(testRelations)
			if err != nil {
				t.Fatalf("failed to delete relations: %v", err)
			}

			// Confirm relation removal
			graph, _ = kb.loadGraph()
			if len(graph.Relations) != 0 {
				t.Errorf("expected 0 relations after deletion, got %d", len(graph.Relations))
			}

			// Remove entities
			err = kb.deleteEntities([]string{"Alice"})
			if err != nil {
				t.Fatalf("failed to delete entities: %v", err)
			}

			// Confirm entity removal
			graph, _ = kb.loadGraph()
			if len(graph.Entities) != 1 || graph.Entities[0].Name != "Bob" {
				t.Errorf("expected only Bob to remain after deleting Alice, got %+v", graph.Entities)
			}
		})
	}
}

// TestSaveAndLoadGraph ensures data persists correctly across save/load cycles.
func TestSaveAndLoadGraph(t *testing.T) {
	for name, newStore := range stores() {
		t.Run(name, func(t *testing.T) {
			s := newStore(t)
			kb := knowledgeBase{s: s}

			// Setup test data
			testGraph := KnowledgeGraph{
				Entities: []Entity{
					{
						Name:         "Charlie",
						EntityType:   "Person",
						Observations: []string{"Likes hiking"},
					},
				},
				Relations: []Relation{
					{
						From:         "Charlie",
						To:           "Mountains",
						RelationType: "enjoys",
					},
				},
			}

			// Persist to storage
			err := kb.saveGraph(testGraph)
			if err != nil {
				t.Fatalf("failed to save graph: %v", err)
			}

			// Reload from storage
			loadedGraph, err := kb.loadGraph()
			if err != nil {
				t.Fatalf("failed to load graph: %v", err)
			}

			// Verify data integrity
			if !reflect.DeepEqual(testGraph, loadedGraph) {
				t.Errorf("loaded graph does not match saved graph.\nExpected: %+v\nGot: %+v", testGraph, loadedGraph)
			}

			// Test malformed data handling
			if fs, ok := s.(*fileStore); ok {
				err := os.WriteFile(fs.path, []byte("invalid json"), 0o600)
				if err != nil {
					t.Fatalf("failed to write invalid json: %v", err)
				}

				_, err = kb.loadGraph()
				if err == nil {
					t.Errorf("expected error when loading invalid JSON, got nil")
				}
			}
		})
	}
}

// TestDuplicateEntitiesAndRelations verifies duplicate prevention logic.
func TestDuplicateEntitiesAndRelations(t *testing.T) {
	for name, newStore := range stores() {
		t.Run(name, func(t *testing.T) {
			s := newStore(t)
			kb := knowledgeBase{s: s}

			// Setup initial state
			initialEntities := []Entity{
				{
					Name:         "Dave",
					EntityType:   "Person",
					Observations: []string{"Plays guitar"},
				},
			}

			_, err := kb.createEntities(initialEntities)
			if err != nil {
				t.Fatalf("failed to create initial entities: %v", err)
			}

			// Attempt duplicate creation
			duplicateEntities := []Entity{
				{
					Name:         "Dave",
					EntityType:   "Person",
					Observations: []string{"Sings well"},
				},
				{
					Name:         "Eve",
					EntityType:   "Person",
					Observations: []string{"Plays piano"},
				},
			}

			newEntities, err := kb.createEntities(duplicateEntities)
			if err != nil {
				t.Fatalf("failed when adding duplicate entities: %v", err)
			}

			// Verify only new entities created
			if len(newEntities) != 1 || newEntities[0].Name != "Eve" {
				t.Errorf("expected only 'Eve' to be created, got %+v", newEntities)
			}

			// Setup initial relation
			initialRelation := []Relation{
				{
					From:         "Dave",
					To:           "Eve",
					RelationType: "friend",
				},
			}

			_, err = kb.createRelations(initialRelation)
			if err != nil {
				t.Fatalf("failed to create initial relation: %v", err)
			}

			// Test relation deduplication
			duplicateRelations := []Relation{
				{
					From:         "Dave",
					To:           "Eve",
					RelationType: "friend",
				},
				{
					From:         "Eve",
					To:           "Dave",
					RelationType: "friend",
				},
			}

			newRelations, err := kb.createRelations(duplicateRelations)
			if err != nil {
				t.Fatalf("failed when adding duplicate relations: %v", err)
			}

			// Verify only new relations created
			if len(newRelations) != 1 || newRelations[0].From != "Eve" || newRelations[0].To != "Dave" {
				t.Errorf("expected only 'Eve->Dave' relation to be created, got %+v", newRelations)
			}
		})
	}
}

// TestErrorHandling verifies proper error responses for invalid operations.
func TestErrorHandling(t *testing.T) {
	t.Run("FileStoreWriteError", func(t *testing.T) {
		// Test file write to invalid path
		kb := knowledgeBase{
			s: &fileStore{path: filepath.Join("nonexistent", "directory", "file.json")},
		}

		testEntities := []Entity{
			{Name: "TestEntity"},
		}

		_, err := kb.createEntities(testEntities)
		if err == nil {
			t.Errorf("expected error when writing to non-existent directory, got nil")
		}
	})

	for name, newStore := range stores() {
		t.Run(fmt.Sprintf("AddObservationToNonExistentEntity_%s", name), func(t *testing.T) {
			s := newStore(t)
			kb := knowledgeBase{s: s}

			// Setup valid entity for comparison
			_, err := kb.createEntities([]Entity{{Name: "RealEntity"}})
			if err != nil {
				t.Fatalf("failed to create test entity: %v", err)
			}

			// Test invalid entity reference
			nonExistentObs := []Observation{
				{
					EntityName: "NonExistentEntity",
					Contents:   []string{"This shouldn't work"},
				},
			}

			_, err = kb.addObservations(nonExistentObs)
			if err == nil {
				t.Errorf("expected error when adding observations to non-existent entity, got nil")
			}
		})
	}
}

// TestFileFormatting verifies the JSON storage format structure.
func TestFileFormatting(t *testing.T) {
	for name, newStore := range stores() {
		t.Run(name, func(t *testing.T) {
			s := newStore(t)
			kb := knowledgeBase{s: s}

			// Setup test entity
			testEntities := []Entity{
				{
					Name:         "FileTest",
					EntityType:   "TestEntity",
					Observations: []string{"Test observation"},
				},
			}

			_, err := kb.createEntities(testEntities)
			if err != nil {
				t.Fatalf("failed to create test entity: %v", err)
			}

			// Extract raw storage data
			data, err := s.Read()
			if err != nil {
				t.Fatalf("failed to read from store: %v", err)
			}

			// Validate JSON format
			var items []kbItem
			err = json.Unmarshal(data, &items)
			if err != nil {
				t.Fatalf("failed to parse store data JSON: %v", err)
			}

			// Check data structure
			if len(items) != 1 {
				t.Fatalf("expected 1 item in memory file, got %d", len(items))
			}

			item := items[0]
			if item.Type != "entity" ||
				item.Name != "FileTest" ||
				item.EntityType != "TestEntity" ||
				len(item.Observations) != 1 ||
				item.Observations[0] != "Test observation" {
				t.Errorf("store item format incorrect: %+v", item)
			}
		})
	}
}

// TestMCPServerIntegration tests the knowledge base through MCP server layer.
func TestMCPServerIntegration(t *testing.T) {
	for name, newStore := range stores() {
		t.Run(name, func(t *testing.T) {
			s := newStore(t)
			kb := knowledgeBase{s: s}

			// Create mock server session
			ctx := context.Background()

			createResult, out, err := kb.CreateEntities(ctx, nil, CreateEntitiesArgs{
				Entities: []Entity{
					{
						Name:         "TestPerson",
						EntityType:   "Person",
						Observations: []string{"Likes testing"},
					},
				},
			})
			if err != nil {
				t.Fatalf("MCP CreateEntities failed: %v", err)
			}
			if createResult.IsError {
				t.Fatalf("MCP CreateEntities returned error: %v", createResult.Content)
			}
			if len(out.Entities) != 1 {
				t.Errorf("expected 1 entity created, got %d", len(out.Entities))
			}

			// Test ReadGraph through MCP
			readResult, outg, err := kb.ReadGraph(ctx, nil, nil)
			if err != nil {
				t.Fatalf("MCP ReadGraph failed: %v", err)
			}
			if readResult.IsError {
				t.Fatalf("MCP ReadGraph returned error: %v", readResult.Content)
			}
			if len(outg.Entities) != 1 {
				t.Errorf("expected 1 entity in graph, got %d", len(outg.Entities))
			}

			relationsResult, outr, err := kb.CreateRelations(ctx, nil, CreateRelationsArgs{
				Relations: []Relation{
					{
						From:         "TestPerson",
						To:           "Testing",
						RelationType: "likes",
					},
				},
			})
			if err != nil {
				t.Fatalf("MCP CreateRelations failed: %v", err)
			}
			if relationsResult.IsError {
				t.Fatalf("MCP CreateRelations returned error: %v", relationsResult.Content)
			}
			if len(outr.Relations) != 1 {
				t.Errorf("expected 1 relation created, got %d", len(outr.Relations))
			}

			obsResult, outo, err := kb.AddObservations(ctx, nil, AddObservationsArgs{
				Observations: []Observation{
					{
						EntityName: "TestPerson",
						Contents:   []string{"Works remotely", "Drinks coffee"},
					},
				},
			})
			if err != nil {
				t.Fatalf("MCP AddObservations failed: %v", err)
			}
			if obsResult.IsError {
				t.Fatalf("MCP AddObservations returned error: %v", obsResult.Content)
			}
			if len(outo.Observations) != 1 {
				t.Errorf("expected 1 observation result, got %d", len(outo.Observations))
			}

			searchResult, outg, err := kb.SearchNodes(ctx, nil, SearchNodesArgs{
				Query: "coffee",
			})
			if err != nil {
				t.Fatalf("MCP SearchNodes failed: %v", err)
			}
			if searchResult.IsError {
				t.Fatalf("MCP SearchNodes returned error: %v", searchResult.Content)
			}
			if len(outg.Entities) != 1 {
				t.Errorf("expected 1 entity from search, got %d", len(outg.Entities))
			}

			openResult, outg, err := kb.OpenNodes(ctx, nil, OpenNodesArgs{
				Names: []string{"TestPerson"},
			})
			if err != nil {
				t.Fatalf("MCP OpenNodes failed: %v", err)
			}
			if openResult.IsError {
				t.Fatalf("MCP OpenNodes returned error: %v", openResult.Content)
			}
			if len(outg.Entities) != 1 {
				t.Errorf("expected 1 entity from open, got %d", len(outg.Entities))
			}

			deleteObsResult, _, err := kb.DeleteObservations(ctx, nil, DeleteObservationsArgs{
				Deletions: []Observation{
					{
						EntityName:   "TestPerson",
						Observations: []string{"Works remotely"},
					},
				},
			})
			if err != nil {
				t.Fatalf("MCP DeleteObservations failed: %v", err)
			}
			if deleteObsResult.IsError {
				t.Fatalf("MCP DeleteObservations returned error: %v", deleteObsResult.Content)
			}

			deleteRelResult, _, err := kb.DeleteRelations(ctx, nil, DeleteRelationsArgs{
				Relations: []Relation{
					{
						From:         "TestPerson",
						To:           "Testing",
						RelationType: "likes",
					},
				},
			})
			if err != nil {
				t.Fatalf("MCP DeleteRelations failed: %v", err)
			}
			if deleteRelResult.IsError {
				t.Fatalf("MCP DeleteRelations returned error: %v", deleteRelResult.Content)
			}

			deleteEntResult, _, err := kb.DeleteEntities(ctx, nil, DeleteEntitiesArgs{
				EntityNames: []string{"TestPerson"},
			})
			if err != nil {
				t.Fatalf("MCP DeleteEntities failed: %v", err)
			}
			if deleteEntResult.IsError {
				t.Fatalf("MCP DeleteEntities returned error: %v", deleteEntResult.Content)
			}

			// Verify final state
			_, outg, err = kb.ReadGraph(ctx, nil, nil)
			if err != nil {
				t.Fatalf("Final MCP ReadGraph failed: %v", err)
			}
			if len(outg.Entities) != 0 {
				t.Errorf("expected empty graph after deletion, got %d entities", len(outg.Entities))
			}
		})
	}
}

// TestMCPErrorHandling tests error scenarios through MCP layer.
func TestMCPErrorHandling(t *testing.T) {
	for name, newStore := range stores() {
		t.Run(name, func(t *testing.T) {
			s := newStore(t)
			kb := knowledgeBase{s: s}

			ctx := context.Background()

			_, _, err := kb.AddObservations(ctx, nil, AddObservationsArgs{
				Observations: []Observation{
					{
						EntityName: "NonExistentEntity",
						Contents:   []string{"This should fail"},
					},
				},
			})
			if err == nil {
				t.Errorf("expected MCP AddObservations to return error for non-existent entity")
			} else {
				// Verify the error message contains expected text
				want := "entity with name NonExistentEntity not found"
				if !strings.Contains(err.Error(), want) {
					t.Errorf("expected error message to contain '%s', got: %v", want, err)
				}
			}
		})
	}
}

// TestMCPResponseFormat verifies MCP response format consistency.
func TestMCPResponseFormat(t *testing.T) {
	s := &memoryStore{}
	kb := knowledgeBase{s: s}

	ctx := context.Background()

	result, out, err := kb.CreateEntities(ctx, nil, CreateEntitiesArgs{
		Entities: []Entity{
			{Name: "FormatTest", EntityType: "Test"},
		},
	})
	if err != nil {
		t.Fatalf("CreateEntities failed: %v", err)
	}

	// Verify response has both Content and StructuredContent
	if len(result.Content) == 0 {
		t.Errorf("expected Content field to be populated")
	}
	if len(out.Entities) == 0 {
		t.Errorf("expected StructuredContent.Entities to be populated")
	}

	// Verify Content contains simple success message
	if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
		expectedMessage := "Entities created successfully"
		if textContent.Text != expectedMessage {
			t.Errorf("expected Content field to contain '%s', got '%s'", expectedMessage, textContent.Text)
		}
	} else {
		t.Errorf("expected Content[0] to be TextContent")
	}
}
</content>
</file>
<file path="examples/server/memory/main.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	httpAddr       = flag.String("http", "", "if set, use streamable HTTP at this address, instead of stdin/stdout")
	memoryFilePath = flag.String("memory", "", "if set, persist the knowledge base to this file; otherwise, it will be stored in memory and lost on exit")
)

// HiArgs defines arguments for the greeting tool.
type HiArgs struct {
	Name string `json:"name"`
}

// CreateEntitiesArgs defines the create entities tool parameters.
type CreateEntitiesArgs struct {
	Entities []Entity `json:"entities" mcp:"entities to create"`
}

// CreateEntitiesResult returns newly created entities.
type CreateEntitiesResult struct {
	Entities []Entity `json:"entities"`
}

// CreateRelationsArgs defines the create relations tool parameters.
type CreateRelationsArgs struct {
	Relations []Relation `json:"relations" mcp:"relations to create"`
}

// CreateRelationsResult returns newly created relations.
type CreateRelationsResult struct {
	Relations []Relation `json:"relations"`
}

// AddObservationsArgs defines the add observations tool parameters.
type AddObservationsArgs struct {
	Observations []Observation `json:"observations" mcp:"observations to add"`
}

// AddObservationsResult returns newly added observations.
type AddObservationsResult struct {
	Observations []Observation `json:"observations"`
}

// DeleteEntitiesArgs defines the delete entities tool parameters.
type DeleteEntitiesArgs struct {
	EntityNames []string `json:"entityNames" mcp:"entities to delete"`
}

// DeleteObservationsArgs defines the delete observations tool parameters.
type DeleteObservationsArgs struct {
	Deletions []Observation `json:"deletions" mcp:"obeservations to delete"`
}

// DeleteRelationsArgs defines the delete relations tool parameters.
type DeleteRelationsArgs struct {
	Relations []Relation `json:"relations" mcp:"relations to delete"`
}

// SearchNodesArgs defines the search nodes tool parameters.
type SearchNodesArgs struct {
	Query string `json:"query" mcp:"query string"`
}

// OpenNodesArgs defines the open nodes tool parameters.
type OpenNodesArgs struct {
	Names []string `json:"names" mcp:"names of nodes to open"`
}

func main() {
	flag.Parse()

	// Initialize storage backend
	var kbStore store
	kbStore = &memoryStore{}
	if *memoryFilePath != "" {
		kbStore = &fileStore{path: *memoryFilePath}
	}
	kb := knowledgeBase{s: kbStore}

	// Setup MCP server with knowledge base tools
	server := mcp.NewServer(&mcp.Implementation{Name: "memory"}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_entities",
		Description: "Create multiple new entities in the knowledge graph",
	}, kb.CreateEntities)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_relations",
		Description: "Create multiple new relations between entities",
	}, kb.CreateRelations)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_observations",
		Description: "Add new observations to existing entities",
	}, kb.AddObservations)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_entities",
		Description: "Remove entities and their relations",
	}, kb.DeleteEntities)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_observations",
		Description: "Remove specific observations from entities",
	}, kb.DeleteObservations)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_relations",
		Description: "Remove specific relations from the graph",
	}, kb.DeleteRelations)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "read_graph",
		Description: "Read the entire knowledge graph",
	}, kb.ReadGraph)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_nodes",
		Description: "Search for nodes based on query",
	}, kb.SearchNodes)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "open_nodes",
		Description: "Retrieve specific nodes by name",
	}, kb.OpenNodes)

	// Start server with appropriate transport
	if *httpAddr != "" {
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return server
		}, nil)
		log.Printf("MCP handler listening at %s", *httpAddr)
		http.ListenAndServe(*httpAddr, handler)
	} else {
		t := &mcp.LoggingTransport{Transport: &mcp.StdioTransport{}, Writer: os.Stderr}
		if err := server.Run(context.Background(), t); err != nil {
			log.Printf("Server failed: %v", err)
		}
	}
}
</content>
</file>
<file path="examples/server/middleware/main.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// This example demonstrates server side logging using the mcp.Middleware system.
func main() {
	// Create a logger for demonstration purposes.
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			// Simplify timestamp format for consistent output.
			if a.Key == slog.TimeKey {
				return slog.String("time", "2025-01-01T00:00:00Z")
			}
			return a
		},
	}))

	loggingMiddleware := func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(
			ctx context.Context,
			method string,
			req mcp.Request,
		) (mcp.Result, error) {
			logger.Info("MCP method started",
				"method", method,
				"session_id", req.GetSession().ID(),
				"has_params", req.GetParams() != nil,
			)
			// Log more for tool calls.
			if ctr, ok := req.(*mcp.CallToolRequest); ok {
				logger.Info("Calling tool",
					"name", ctr.Params.Name,
					"args", ctr.Params.Arguments)
			}

			start := time.Now()
			result, err := next(ctx, method, req)
			duration := time.Since(start)
			if err != nil {
				logger.Error("MCP method failed",
					"method", method,
					"session_id", req.GetSession().ID(),
					"duration_ms", duration.Milliseconds(),
					"err", err,
				)
			} else {
				logger.Info("MCP method completed",
					"method", method,
					"session_id", req.GetSession().ID(),
					"duration_ms", duration.Milliseconds(),
					"has_result", result != nil,
				)
				// Log more for tool results.
				if ctr, ok := result.(*mcp.CallToolResult); ok {
					logger.Info("tool result",
						"isError", ctr.IsError,
						"structuredContent", ctr.StructuredContent)
				}
			}
			return result, err
		}
	}

	// Create server with middleware
	server := mcp.NewServer(&mcp.Implementation{Name: "logging-example"}, nil)
	server.AddReceivingMiddleware(loggingMiddleware)

	// Add a simple tool
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "greet",
			Description: "Greet someone with logging.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "Name to greet",
					},
				},
				Required: []string{"name"},
			},
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest, args map[string]any,
		) (*mcp.CallToolResult, any, error) {
			name, ok := args["name"].(string)
			if !ok {
				return nil, nil, fmt.Errorf("name parameter is required and must be a string")
			}

			message := fmt.Sprintf("Hello, %s!", name)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: message},
				},
			}, message, nil
		},
	)

	// Create client-server connection for demonstration
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	ctx := context.Background()

	// Connect server and client
	serverSession, _ := server.Connect(ctx, serverTransport, nil)
	defer serverSession.Close()

	clientSession, _ := client.Connect(ctx, clientTransport, nil)
	defer clientSession.Close()

	// Call the tool to demonstrate logging
	result, _ := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "greet",
		Arguments: map[string]any{
			"name": "World",
		},
	})

	fmt.Printf("Tool result: %s\n", result.Content[0].(*mcp.TextContent).Text)

	// Output:
	// time=2025-01-01T00:00:00Z level=INFO msg="MCP method started" method=initialize session_id="" has_params=true
	// time=2025-01-01T00:00:00Z level=INFO msg="MCP method completed" method=initialize session_id="" duration_ms=0 has_result=true
	// time=2025-01-01T00:00:00Z level=INFO msg="MCP method started" method=notifications/initialized session_id="" has_params=true
	// time=2025-01-01T00:00:00Z level=INFO msg="MCP method completed" method=notifications/initialized session_id="" duration_ms=0 has_result=false
	// time=2025-01-01T00:00:00Z level=INFO msg="MCP method started" method=tools/call session_id="" has_params=true
	// time=2025-01-01T00:00:00Z level=INFO msg="Calling tool" name=greet args="{\"name\":\"World\"}"
	// time=2025-01-01T00:00:00Z level=INFO msg="MCP method completed" method=tools/call session_id="" duration_ms=0 has_result=true
	// Tool result: Hello, World!
}
</content>
</file>
<file path="examples/server/rate-limiting/main.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/time/rate"
)

// GlobalRateLimiterMiddleware creates a middleware that applies a global rate limit.
// Every request attempting to pass through will try to acquire a token.
// If a token cannot be acquired immediately, the request will be rejected.
func GlobalRateLimiterMiddleware(limiter *rate.Limiter) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if !limiter.Allow() {
				return nil, errors.New("JSON RPC overloaded")
			}
			return next(ctx, method, req)
		}
	}
}

// PerMethodRateLimiterMiddleware creates a middleware that applies rate limiting
// on a per-method basis.
// Methods not specified in limiters will not be rate limited by this middleware.
func PerMethodRateLimiterMiddleware(limiters map[string]*rate.Limiter) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if limiter, ok := limiters[method]; ok {
				if !limiter.Allow() {
					return nil, errors.New("JSON RPC overloaded")
				}
			}
			return next(ctx, method, req)
		}
	}
}

// PerSessionRateLimiterMiddleware creates a middleware that applies rate limiting
// on a per-session basis for receiving requests.
func PerSessionRateLimiterMiddleware(limit rate.Limit, burst int) mcp.Middleware {
	// A map to store limiters, keyed by the session ID.
	var (
		sessionLimiters = make(map[string]*rate.Limiter)
		mu              sync.Mutex
	)

	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			// It's possible that session.ID() may be empty at this point in time
			// for some transports (e.g., stdio) or until the MCP initialize handshake
			// has completed.
			sessionID := req.GetSession().ID()
			if sessionID == "" {
				// In this situation, you could apply a single global identifier
				// if session ID is empty or bypass the rate limiter.
				// In this example, we bypass the rate limiter.
				log.Printf("Warning: Session ID is empty for method %q. Skipping per-session rate limiting.", method)
				return next(ctx, method, req) // Skip limiting if ID is unavailable
			}
			mu.Lock()
			limiter, ok := sessionLimiters[sessionID]
			if !ok {
				limiter = rate.NewLimiter(limit, burst)
				sessionLimiters[sessionID] = limiter
			}
			mu.Unlock()
			if !limiter.Allow() {
				return nil, errors.New("JSON RPC overloaded")
			}
			return next(ctx, method, req)
		}
	}
}

func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: "greeter1", Version: "v0.0.1"}, nil)
	server.AddReceivingMiddleware(GlobalRateLimiterMiddleware(rate.NewLimiter(rate.Every(time.Second/5), 10)))
	server.AddReceivingMiddleware(PerMethodRateLimiterMiddleware(map[string]*rate.Limiter{
		"callTool":  rate.NewLimiter(rate.Every(time.Second), 5),  // once a second with a burst up to 5
		"listTools": rate.NewLimiter(rate.Every(time.Minute), 20), // once a minute with a burst up to 20
	}))
	server.AddReceivingMiddleware(PerSessionRateLimiterMiddleware(rate.Every(time.Second/5), 10))
	// Run Server logic.
	log.Println("MCP Server instance created with Middleware (but not running).")
	log.Println("This example demonstrates configuration, not live interaction.")
}
</content>
</file>
<file path="examples/server/sequentialthinking/main.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"maps"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var httpAddr = flag.String("http", "", "if set, use streamable HTTP at this address, instead of stdin/stdout")

// A Thought is a single step in the thinking process.
type Thought struct {
	// Index of the thought within the session (1-based).
	Index int `json:"index"`
	// Content of the thought.
	Content string `json:"content"`
	// Time the thought was created.
	Created time.Time `json:"created"`
	// Whether the thought has been revised.
	Revised bool `json:"revised"`
	// Index of parent thought, or nil if this is a root for branching.
	ParentIndex *int `json:"parentIndex,omitempty"`
}

// A ThinkingSession is an active thinking session.
type ThinkingSession struct {
	// Globally unique ID of the session.
	ID string `json:"id"`
	// Problem to solve.
	Problem string `json:"problem"`
	// Thoughts in the session.
	Thoughts []*Thought `json:"thoughts"`
	// Current thought index.
	CurrentThought int `json:"currentThought"`
	// Estimated total number of thoughts.
	EstimatedTotal int `json:"estimatedTotal"`
	// Status of the session.
	Status string `json:"status"` // "active", "completed", "paused"
	// Time the session was created.
	Created time.Time `json:"created"`
	// Time the session was last active.
	LastActivity time.Time `json:"lastActivity"`
	// Branches in the session. Alternative thought paths.
	Branches []string `json:"branches,omitempty"`
	// Version for optimistic concurrency control.
	Version int `json:"version"`
}

// clone returns a deep copy of the ThinkingSession.
func (s *ThinkingSession) clone() *ThinkingSession {
	sessionCopy := *s
	sessionCopy.Thoughts = deepCopyThoughts(s.Thoughts)
	sessionCopy.Branches = slices.Clone(s.Branches)
	return &sessionCopy
}

// A SessionStore is a global session store (in a real implementation, this might be a database).
//
// Locking Strategy:
// The SessionStore uses a RWMutex to protect the sessions map from concurrent access.
// All ThinkingSession modifications happen on deep copies, never on shared instances.
// This means:
// - Read locks protect map access.
// - Write locks protect map modifications (adding/removing/replacing sessions)
// - Session field modifications always happen on local copies via CompareAndSwap
// - No shared ThinkingSession state is ever modified directly
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*ThinkingSession // key is session ID
}

// NewSessionStore creates a new session store for managing thinking sessions.
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*ThinkingSession),
	}
}

// Session retrieves a thinking session by ID, returning the session and whether it exists.
func (s *SessionStore) Session(id string) (*ThinkingSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, exists := s.sessions[id]
	return session, exists
}

// SetSession stores or updates a thinking session in the store.
func (s *SessionStore) SetSession(session *ThinkingSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
}

// CompareAndSwap atomically updates a session if the version matches.
// Returns true if the update succeeded, false if there was a version mismatch.
//
// This method implements optimistic concurrency control:
// 1. Read lock to safely access the map and copy the session
// 2. Deep copy the session (all modifications happen on this copy)
// 3. Release read lock and apply updates to the copy
// 4. Write lock to check version and atomically update if unchanged
//
// The read lock in step 1 is necessary to prevent map access races,
// not to protect ThinkingSession fields (which are never modified in-place).
func (s *SessionStore) CompareAndSwap(sessionID string, updateFunc func(*ThinkingSession) (*ThinkingSession, error)) error {
	for {
		// Get current session
		s.mu.RLock()
		current, exists := s.sessions[sessionID]
		if !exists {
			s.mu.RUnlock()
			return fmt.Errorf("session %s not found", sessionID)
		}
		// Create a deep copy
		sessionCopy := current.clone()
		oldVersion := current.Version
		s.mu.RUnlock()

		// Apply the update
		updated, err := updateFunc(sessionCopy)
		if err != nil {
			return err
		}

		// Try to save
		s.mu.Lock()
		current, exists = s.sessions[sessionID]
		if !exists {
			s.mu.Unlock()
			return fmt.Errorf("session %s not found", sessionID)
		}
		if current.Version != oldVersion {
			// Version mismatch, retry
			s.mu.Unlock()
			continue
		}
		updated.Version = oldVersion + 1
		s.sessions[sessionID] = updated
		s.mu.Unlock()
		return nil
	}
}

// Sessions returns all thinking sessions in the store.
func (s *SessionStore) Sessions() []*ThinkingSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return slices.Collect(maps.Values(s.sessions))
}

// SessionsSnapshot returns a deep copy of all sessions for safe concurrent access.
func (s *SessionStore) SessionsSnapshot() []*ThinkingSession {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sessions []*ThinkingSession
	for _, session := range s.sessions {
		sessions = append(sessions, session.clone())
	}
	return sessions
}

// SessionSnapshot returns a deep copy of a session for safe concurrent access.
// The second return value reports whether a session with the given id exists.
func (s *SessionStore) SessionSnapshot(id string) (*ThinkingSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[id]
	if !exists {
		return nil, false
	}

	return session.clone(), true
}

var store = NewSessionStore()

// StartThinkingArgs are the arguments for starting a new thinking session.
type StartThinkingArgs struct {
	Problem        string `json:"problem"`
	SessionID      string `json:"sessionId,omitempty"`
	EstimatedSteps int    `json:"estimatedSteps,omitempty"`
}

// ContinueThinkingArgs are the arguments for continuing a thinking session.
type ContinueThinkingArgs struct {
	SessionID      string `json:"sessionId"`
	Thought        string `json:"thought"`
	NextNeeded     *bool  `json:"nextNeeded,omitempty"`
	ReviseStep     *int   `json:"reviseStep,omitempty"`
	CreateBranch   bool   `json:"createBranch,omitempty"`
	EstimatedTotal int    `json:"estimatedTotal,omitempty"`
}

// ReviewThinkingArgs are the arguments for reviewing a thinking session.
type ReviewThinkingArgs struct {
	SessionID string `json:"sessionId"`
}

// ThinkingHistoryArgs are the arguments for retrieving thinking history.
type ThinkingHistoryArgs struct {
	SessionID string `json:"sessionId"`
}

// deepCopyThoughts creates a deep copy of a slice of thoughts.
func deepCopyThoughts(thoughts []*Thought) []*Thought {
	thoughtsCopy := make([]*Thought, len(thoughts))
	for i, t := range thoughts {
		t2 := *t
		thoughtsCopy[i] = &t2
	}
	return thoughtsCopy
}

// StartThinking begins a new sequential thinking session for a complex problem.
func StartThinking(ctx context.Context, _ *mcp.CallToolRequest, args StartThinkingArgs) (*mcp.CallToolResult, any, error) {
	sessionID := args.SessionID
	if sessionID == "" {
		sessionID = randText()
	}

	estimatedSteps := args.EstimatedSteps
	if estimatedSteps == 0 {
		estimatedSteps = 5 // Default estimate
	}

	session := &ThinkingSession{
		ID:             sessionID,
		Problem:        args.Problem,
		EstimatedTotal: estimatedSteps,
		Status:         "active",
		Created:        time.Now(),
		LastActivity:   time.Now(),
	}

	store.SetSession(session)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("Started thinking session '%s' for problem: %s\nEstimated steps: %d\nReady for your first thought.",
					sessionID, args.Problem, estimatedSteps),
			},
		},
	}, nil, nil
}

// ContinueThinking adds the next thought step, revises a previous step, or creates a branch in the thinking process.
func ContinueThinking(ctx context.Context, req *mcp.CallToolRequest, args ContinueThinkingArgs) (*mcp.CallToolResult, any, error) {
	// Handle revision of existing thought
	if args.ReviseStep != nil {
		err := store.CompareAndSwap(args.SessionID, func(session *ThinkingSession) (*ThinkingSession, error) {
			stepIndex := *args.ReviseStep - 1
			if stepIndex < 0 || stepIndex >= len(session.Thoughts) {
				return nil, fmt.Errorf("invalid step number: %d", *args.ReviseStep)
			}

			session.Thoughts[stepIndex].Content = args.Thought
			session.Thoughts[stepIndex].Revised = true
			session.LastActivity = time.Now()
			return session, nil
		})
		if err != nil {
			return nil, nil, err
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("Revised step %d in session '%s':\n%s",
						*args.ReviseStep, args.SessionID, args.Thought),
				},
			},
		}, nil, nil
	}

	// Handle branching
	if args.CreateBranch {
		var branchID string
		var branchSession *ThinkingSession

		err := store.CompareAndSwap(args.SessionID, func(session *ThinkingSession) (*ThinkingSession, error) {
			branchID = fmt.Sprintf("%s_branch_%d", args.SessionID, len(session.Branches)+1)
			session.Branches = append(session.Branches, branchID)
			session.LastActivity = time.Now()

			// Create a new session for the branch (deep copy thoughts)
			thoughtsCopy := deepCopyThoughts(session.Thoughts)
			branchSession = &ThinkingSession{
				ID:             branchID,
				Problem:        session.Problem + " (Alternative branch)",
				Thoughts:       thoughtsCopy,
				CurrentThought: len(session.Thoughts),
				EstimatedTotal: session.EstimatedTotal,
				Status:         "active",
				Created:        time.Now(),
				LastActivity:   time.Now(),
			}

			return session, nil
		})
		if err != nil {
			return nil, nil, err
		}

		// Save the branch session
		store.SetSession(branchSession)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("Created branch '%s' from session '%s'. You can now continue thinking in either session.",
						branchID, args.SessionID),
				},
			},
		}, nil, nil
	}

	// Add new thought
	var thoughtID int
	var progress string
	var statusMsg string

	err := store.CompareAndSwap(args.SessionID, func(session *ThinkingSession) (*ThinkingSession, error) {
		thoughtID = len(session.Thoughts) + 1
		thought := &Thought{
			Index:   thoughtID,
			Content: args.Thought,
			Created: time.Now(),
			Revised: false,
		}

		session.Thoughts = append(session.Thoughts, thought)
		session.CurrentThought = thoughtID
		session.LastActivity = time.Now()

		// Update estimated total if provided
		if args.EstimatedTotal > 0 {
			session.EstimatedTotal = args.EstimatedTotal
		}

		// Check if thinking is complete
		if args.NextNeeded != nil && !*args.NextNeeded {
			session.Status = "completed"
		}

		// Prepare response strings
		progress = fmt.Sprintf("Step %d", thoughtID)
		if session.EstimatedTotal > 0 {
			progress += fmt.Sprintf(" of ~%d", session.EstimatedTotal)
		}

		if session.Status == "completed" {
			statusMsg = "\n✓ Thinking process completed!"
		} else {
			statusMsg = "\nReady for next thought..."
		}

		return session, nil
	})
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("Session '%s' - %s:\n%s%s",
					args.SessionID, progress, args.Thought, statusMsg),
			},
		},
	}, nil, nil
}

// ReviewThinking provides a complete review of the thinking process for a session.
func ReviewThinking(ctx context.Context, req *mcp.CallToolRequest, args ReviewThinkingArgs) (*mcp.CallToolResult, any, error) {
	// Get a snapshot of the session to avoid race conditions
	sessionSnapshot, exists := store.SessionSnapshot(args.SessionID)
	if !exists {
		return nil, nil, fmt.Errorf("session %s not found", args.SessionID)
	}

	var review strings.Builder
	fmt.Fprintf(&review, "=== Thinking Review: %s ===\n", sessionSnapshot.ID)
	fmt.Fprintf(&review, "Problem: %s\n", sessionSnapshot.Problem)
	fmt.Fprintf(&review, "Status: %s\n", sessionSnapshot.Status)
	fmt.Fprintf(&review, "Steps: %d of ~%d\n", len(sessionSnapshot.Thoughts), sessionSnapshot.EstimatedTotal)

	if len(sessionSnapshot.Branches) > 0 {
		fmt.Fprintf(&review, "Branches: %s\n", strings.Join(sessionSnapshot.Branches, ", "))
	}

	fmt.Fprintf(&review, "\n--- Thought Sequence ---\n")

	for i, thought := range sessionSnapshot.Thoughts {
		status := ""
		if thought.Revised {
			status = " (revised)"
		}
		fmt.Fprintf(&review, "%d. %s%s\n", i+1, thought.Content, status)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: review.String(),
			},
		},
	}, nil, nil
}

// ThinkingHistory handles resource requests for thinking session data and history.
func ThinkingHistory(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	// Extract session ID from URI (e.g., "thinking://session_123")
	u, err := url.Parse(req.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid thinking resource URI: %s", req.Params.URI)
	}
	if u.Scheme != "thinking" {
		return nil, fmt.Errorf("invalid thinking resource URI scheme: %s", u.Scheme)
	}

	sessionID := u.Host
	if sessionID == "sessions" {
		// List all sessions - use snapshot for thread safety
		sessions := store.SessionsSnapshot()
		data, err := json.MarshalIndent(sessions, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal sessions: %w", err)
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      req.Params.URI,
					MIMEType: "application/json",
					Text:     string(data),
				},
			},
		}, nil
	}

	// Get specific session - use snapshot for thread safety
	session, exists := store.SessionSnapshot(sessionID)
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(data),
			},
		},
	}, nil
}

// Copied from crypto/rand.
// TODO: once 1.24 is assured, just use crypto/rand.
const base32alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"

func randText() string {
	// ⌈log₃₂ 2¹²⁸⌉ = 26 chars
	src := make([]byte, 26)
	rand.Read(src)
	for i := range src {
		src[i] = base32alphabet[src[i]%32]
	}
	return string(src)
}

func main() {
	flag.Parse()

	server := mcp.NewServer(&mcp.Implementation{Name: "sequential-thinking"}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "start_thinking",
		Description: "Begin a new sequential thinking session for a complex problem",
	}, StartThinking)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "continue_thinking",
		Description: "Add the next thought step, revise a previous step, or create a branch",
	}, ContinueThinking)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "review_thinking",
		Description: "Review the complete thinking process for a session",
	}, ReviewThinking)

	server.AddResource(&mcp.Resource{
		Name:        "thinking_sessions",
		Description: "Access thinking session data and history",
		URI:         "thinking://sessions",
		MIMEType:    "application/json",
	}, ThinkingHistory)

	if *httpAddr != "" {
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return server
		}, nil)
		log.Printf("Sequential Thinking MCP server listening at %s", *httpAddr)
		if err := http.ListenAndServe(*httpAddr, handler); err != nil {
			log.Fatal(err)
		}
	} else {
		t := &mcp.LoggingTransport{Transport: &mcp.StdioTransport{}, Writer: os.Stderr}
		if err := server.Run(context.Background(), t); err != nil {
			log.Printf("Server failed: %v", err)
		}
	}
}
</content>
</file>
<file path="examples/server/sequentialthinking/main_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestStartThinking(t *testing.T) {
	// Reset store for clean test
	store = NewSessionStore()

	ctx := context.Background()

	args := StartThinkingArgs{
		Problem:        "How to implement a binary search algorithm",
		SessionID:      "test_session",
		EstimatedSteps: 5,
	}

	result, _, err := StartThinking(ctx, nil, args)
	if err != nil {
		t.Fatalf("StartThinking() error = %v", err)
	}

	if len(result.Content) == 0 {
		t.Fatal("No content in result")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	if !strings.Contains(textContent.Text, "test_session") {
		t.Error("Result should contain session ID")
	}

	if !strings.Contains(textContent.Text, "How to implement a binary search algorithm") {
		t.Error("Result should contain the problem statement")
	}

	// Verify session was stored
	session, exists := store.Session("test_session")
	if !exists {
		t.Fatal("Session was not stored")
	}

	if session.Problem != args.Problem {
		t.Errorf("Expected problem %s, got %s", args.Problem, session.Problem)
	}

	if session.EstimatedTotal != 5 {
		t.Errorf("Expected estimated total 5, got %d", session.EstimatedTotal)
	}

	if session.Status != "active" {
		t.Errorf("Expected status 'active', got %s", session.Status)
	}
}

func TestContinueThinking(t *testing.T) {
	// Reset store and create initial session
	store = NewSessionStore()

	// First start a thinking session
	ctx := context.Background()
	startArgs := StartThinkingArgs{
		Problem:        "Test problem",
		SessionID:      "test_continue",
		EstimatedSteps: 3,
	}

	_, _, err := StartThinking(ctx, nil, startArgs)
	if err != nil {
		t.Fatalf("StartThinking() error = %v", err)
	}

	// Now continue thinking
	continueArgs := ContinueThinkingArgs{
		SessionID: "test_continue",
		Thought:   "First thought: I need to understand the problem",
	}

	result, _, err := ContinueThinking(ctx, nil, continueArgs)
	if err != nil {
		t.Fatalf("ContinueThinking() error = %v", err)
	}

	// Verify result
	if len(result.Content) == 0 {
		t.Fatal("No content in result")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	if !strings.Contains(textContent.Text, "Step 1") {
		t.Error("Result should contain step number")
	}

	// Verify session was updated
	session, exists := store.Session("test_continue")
	if !exists {
		t.Fatal("Session not found")
	}

	if len(session.Thoughts) != 1 {
		t.Errorf("Expected 1 thought, got %d", len(session.Thoughts))
	}

	if session.Thoughts[0].Content != continueArgs.Thought {
		t.Errorf("Expected thought content %s, got %s", continueArgs.Thought, session.Thoughts[0].Content)
	}

	if session.CurrentThought != 1 {
		t.Errorf("Expected current thought 1, got %d", session.CurrentThought)
	}
}

func TestContinueThinkingWithCompletion(t *testing.T) {
	// Reset store and create initial session
	store = NewSessionStore()

	ctx := context.Background()
	startArgs := StartThinkingArgs{
		Problem:   "Simple test",
		SessionID: "test_completion",
	}

	_, _, err := StartThinking(ctx, nil, startArgs)
	if err != nil {
		t.Fatalf("StartThinking() error = %v", err)
	}

	// Continue with completion flag
	nextNeeded := false
	continueArgs := ContinueThinkingArgs{
		SessionID:  "test_completion",
		Thought:    "Final thought",
		NextNeeded: &nextNeeded,
	}

	result, _, err := ContinueThinking(ctx, nil, continueArgs)
	if err != nil {
		t.Fatalf("ContinueThinking() error = %v", err)
	}

	// Check completion message
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	if !strings.Contains(textContent.Text, "completed") {
		t.Error("Result should indicate completion")
	}

	// Verify session status
	session, exists := store.Session("test_completion")
	if !exists {
		t.Fatal("Session not found")
	}

	if session.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", session.Status)
	}
}

func TestContinueThinkingRevision(t *testing.T) {
	// Setup session with existing thoughts
	store = NewSessionStore()
	session := &ThinkingSession{
		ID:      "test_revision",
		Problem: "Test problem",
		Thoughts: []*Thought{
			{Index: 1, Content: "Original thought", Created: time.Now()},
			{Index: 2, Content: "Second thought", Created: time.Now()},
		},
		CurrentThought: 2,
		EstimatedTotal: 3,
		Status:         "active",
		Created:        time.Now(),
		LastActivity:   time.Now(),
	}
	store.SetSession(session)

	ctx := context.Background()
	reviseStep := 1
	continueArgs := ContinueThinkingArgs{
		SessionID:  "test_revision",
		Thought:    "Revised first thought",
		ReviseStep: &reviseStep,
	}

	result, _, err := ContinueThinking(ctx, nil, continueArgs)
	if err != nil {
		t.Fatalf("ContinueThinking() error = %v", err)
	}

	// Verify revision message
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	if !strings.Contains(textContent.Text, "Revised step 1") {
		t.Error("Result should indicate revision")
	}

	// Verify thought was revised
	updatedSession, _ := store.Session("test_revision")
	if updatedSession.Thoughts[0].Content != "Revised first thought" {
		t.Error("First thought should be revised")
	}

	if !updatedSession.Thoughts[0].Revised {
		t.Error("First thought should be marked as revised")
	}
}

// func TestContinueThinkingBranching(t *testing.T) {
// 	// Setup session with existing thoughts
// 	store = NewSessionStore()
// 	session := &ThinkingSession{
// 		ID:      "test_branch",
// 		Problem: "Test problem",
// 		Thoughts: []*Thought{
// 			{Index: 1, Content: "First thought", Created: time.Now()},
// 		},
// 		CurrentThought: 1,
// 		EstimatedTotal: 3,
// 		Status:         "active",
// 		Created:        time.Now(),
// 		LastActivity:   time.Now(),
// 		Branches:       []string{},
// 	}
// 	store.SetSession(session)

// 	ctx := context.Background()
// 	continueArgs := ContinueThinkingArgs{
// 		SessionID:    "test_branch",
// 		Thought:      "Alternative approach",
// 		CreateBranch: true,
// 	}

// 	continueParams := &mcp.CallToolParamsFor[ContinueThinkingArgs]{
// 		Name:      "continue_thinking",
// 		Arguments: continueArgs,
// 	}

// 	// Verify branch creation message
// 	textContent, ok := result.Content[0].(*mcp.TextContent)
// 	if !ok {
// 		t.Fatal("Expected TextContent")
// 	}

// 	if !strings.Contains(textContent.Text, "Created branch") {
// 		t.Error("Result should indicate branch creation")
// 	}

// 	// Verify branch was created
// 	updatedSession, _ := store.Session("test_branch")
// 	if len(updatedSession.Branches) != 1 {
// 		t.Errorf("Expected 1 branch, got %d", len(updatedSession.Branches))
// 	}

// 	branchID := updatedSession.Branches[0]
// 	if !strings.Contains(branchID, "test_branch_branch_") {
// 		t.Error("Branch ID should contain parent session ID")
// 	}

// 	// Verify branch session exists
// 	branchSession, exists := store.Session(branchID)
// 	if !exists {
// 		t.Fatal("Branch session should exist")
// 	}

// 	if len(branchSession.Thoughts) != 1 {
// 		t.Error("Branch should inherit parent thoughts")
// 	}
// }

func TestReviewThinking(t *testing.T) {
	// Setup session with thoughts
	store = NewSessionStore()
	session := &ThinkingSession{
		ID:      "test_review",
		Problem: "Complex problem",
		Thoughts: []*Thought{
			{Index: 1, Content: "First thought", Created: time.Now(), Revised: false},
			{Index: 2, Content: "Second thought", Created: time.Now(), Revised: true},
			{Index: 3, Content: "Final thought", Created: time.Now(), Revised: false},
		},
		CurrentThought: 3,
		EstimatedTotal: 3,
		Status:         "completed",
		Created:        time.Now(),
		LastActivity:   time.Now(),
		Branches:       []string{"test_review_branch_1"},
	}
	store.SetSession(session)

	ctx := context.Background()
	reviewArgs := ReviewThinkingArgs{
		SessionID: "test_review",
	}

	result, _, err := ReviewThinking(ctx, nil, reviewArgs)
	if err != nil {
		t.Fatalf("ReviewThinking() error = %v", err)
	}

	// Verify review content
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	reviewText := textContent.Text

	if !strings.Contains(reviewText, "test_review") {
		t.Error("Review should contain session ID")
	}

	if !strings.Contains(reviewText, "Complex problem") {
		t.Error("Review should contain problem")
	}

	if !strings.Contains(reviewText, "completed") {
		t.Error("Review should contain status")
	}

	if !strings.Contains(reviewText, "Steps: 3 of ~3") {
		t.Error("Review should contain step count")
	}

	if !strings.Contains(reviewText, "First thought") {
		t.Error("Review should contain first thought")
	}

	if !strings.Contains(reviewText, "(revised)") {
		t.Error("Review should indicate revised thoughts")
	}

	if !strings.Contains(reviewText, "test_review_branch_1") {
		t.Error("Review should list branches")
	}
}

func TestThinkingHistory(t *testing.T) {
	// Setup test sessions
	store = NewSessionStore()
	session1 := &ThinkingSession{
		ID:             "session1",
		Problem:        "Problem 1",
		Thoughts:       []*Thought{{Index: 1, Content: "Thought 1", Created: time.Now()}},
		CurrentThought: 1,
		EstimatedTotal: 2,
		Status:         "active",
		Created:        time.Now(),
		LastActivity:   time.Now(),
	}
	session2 := &ThinkingSession{
		ID:             "session2",
		Problem:        "Problem 2",
		Thoughts:       []*Thought{{Index: 1, Content: "Thought 1", Created: time.Now()}},
		CurrentThought: 1,
		EstimatedTotal: 3,
		Status:         "completed",
		Created:        time.Now(),
		LastActivity:   time.Now(),
	}
	store.SetSession(session1)
	store.SetSession(session2)

	ctx := context.Background()

	// Test listing all sessions
	result, err := ThinkingHistory(ctx, &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "thinking://sessions",
		},
	})
	if err != nil {
		t.Fatalf("ThinkingHistory() error = %v", err)
	}

	if len(result.Contents) != 1 {
		t.Fatal("Expected 1 content item")
	}

	content := result.Contents[0]
	if content.MIMEType != "application/json" {
		t.Error("Expected JSON MIME type")
	}

	// Parse and verify sessions list
	var sessions []*ThinkingSession
	err = json.Unmarshal([]byte(content.Text), &sessions)
	if err != nil {
		t.Fatalf("Failed to parse sessions JSON: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}

	// Test getting specific session
	result, err = ThinkingHistory(ctx, &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: "thinking://session1"},
	})
	if err != nil {
		t.Fatalf("ThinkingHistory() error = %v", err)
	}

	var retrievedSession ThinkingSession
	err = json.Unmarshal([]byte(result.Contents[0].Text), &retrievedSession)
	if err != nil {
		t.Fatalf("Failed to parse session JSON: %v", err)
	}

	if retrievedSession.ID != "session1" {
		t.Errorf("Expected session ID 'session1', got %s", retrievedSession.ID)
	}

	if retrievedSession.Problem != "Problem 1" {
		t.Errorf("Expected problem 'Problem 1', got %s", retrievedSession.Problem)
	}
}

func TestInvalidOperations(t *testing.T) {
	store = NewSessionStore()
	ctx := context.Background()

	// Test continue thinking with non-existent session
	continueArgs := ContinueThinkingArgs{
		SessionID: "nonexistent",
		Thought:   "Some thought",
	}

	_, _, err := ContinueThinking(ctx, nil, continueArgs)
	if err == nil {
		t.Error("Expected error for non-existent session")
	}

	// Test review with non-existent session
	reviewArgs := ReviewThinkingArgs{
		SessionID: "nonexistent",
	}

	_, _, err = ReviewThinking(ctx, nil, reviewArgs)
	if err == nil {
		t.Error("Expected error for non-existent session in review")
	}

	// Test invalid revision step
	session := &ThinkingSession{
		ID:             "test_invalid",
		Problem:        "Test",
		Thoughts:       []*Thought{{Index: 1, Content: "Thought", Created: time.Now()}},
		CurrentThought: 1,
		EstimatedTotal: 2,
		Status:         "active",
		Created:        time.Now(),
		LastActivity:   time.Now(),
	}
	store.SetSession(session)

	reviseStep := 5 // Invalid step number
	invalidReviseArgs := ContinueThinkingArgs{
		SessionID:  "test_invalid",
		Thought:    "Revised",
		ReviseStep: &reviseStep,
	}

	_, _, err = ContinueThinking(ctx, nil, invalidReviseArgs)
	if err == nil {
		t.Error("Expected error for invalid revision step")
	}
}
</content>
</file>
<file path="examples/server/sse/main.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	host = flag.String("host", "localhost", "host to listen on")
	port = flag.String("port", "8080", "port to listen on")
)

type SayHiParams struct {
	Name string `json:"name"`
}

func SayHi(ctx context.Context, req *mcp.CallToolRequest, args SayHiParams) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Hi " + args.Name},
		},
	}, nil, nil
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "This program runs MCP servers over SSE HTTP.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEndpoints:\n")
		fmt.Fprintf(os.Stderr, "  /greeter1 - Greeter 1 service\n")
		fmt.Fprintf(os.Stderr, "  /greeter2 - Greeter 2 service\n")
		os.Exit(1)
	}
	flag.Parse()

	addr := fmt.Sprintf("%s:%s", *host, *port)

	server1 := mcp.NewServer(&mcp.Implementation{Name: "greeter1"}, nil)
	mcp.AddTool(server1, &mcp.Tool{Name: "greet1", Description: "say hi"}, SayHi)

	server2 := mcp.NewServer(&mcp.Implementation{Name: "greeter2"}, nil)
	mcp.AddTool(server2, &mcp.Tool{Name: "greet2", Description: "say hello"}, SayHi)

	log.Printf("MCP servers serving at %s", addr)
	handler := mcp.NewSSEHandler(func(request *http.Request) *mcp.Server {
		url := request.URL.Path
		log.Printf("Handling request for URL %s\n", url)
		switch url {
		case "/greeter1":
			return server1
		case "/greeter2":
			return server2
		default:
			return nil
		}
	}, nil)
	log.Fatal(http.ListenAndServe(addr, handler))
}
</content>
</file>
<file path="examples/server/toolschemas/main.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// The toolschemas example demonstrates how to create tools using both the
// low-level [ToolHandler] and high level [ToolHandlerFor], as well as how to
// customize schemas in both cases.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Input is the input into all the tools handlers below.
type Input struct {
	Name string `json:"name" jsonschema:"the person to greet"`
}

// Output is the structured output of the tool.
//
// Not every tool needs to have structured output.
type Output struct {
	Greeting string `json:"greeting" jsonschema:"the greeting to send to the user"`
}

// simpleGreeting is an [mcp.ToolHandlerFor] that only cares about input and output.
func simpleGreeting(_ context.Context, _ *mcp.CallToolRequest, input Input) (*mcp.CallToolResult, Output, error) {
	return nil, Output{"Hi " + input.Name}, nil
}

// manualGreeter handles the parsing and validation of input and output manually.
//
// Therfore, it needs to close over its resolved schemas, to use them in
// validation.
type manualGreeter struct {
	inputSchema  *jsonschema.Resolved
	outputSchema *jsonschema.Resolved
}

func (t *manualGreeter) greet(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// errf produces a 'tool error', embedding the error in a CallToolResult.
	errf := func(format string, args ...any) *mcp.CallToolResult {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf(format, args...)}},
			IsError: true,
		}
	}
	// Handle the parsing and validation of input and output.
	//
	// Note that errors here are treated as tool errors, not protocol errors.

	// First, unmarshal to a map[string]any and validate.
	if err := unmarshalAndValidate(req.Params.Arguments, t.inputSchema); err != nil {
		return errf("invalid input: %v", err), nil
	}

	// Now unmarshal again to input.
	var input Input
	if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
		return errf("failed to unmarshal arguments: %v", err), nil
	}
	output := Output{Greeting: "Hi " + input.Name}
	outputJSON, err := json.Marshal(output)
	if err != nil {
		return errf("output failed to marshal: %v", err), nil
	}
	//
	if err := unmarshalAndValidate(outputJSON, t.outputSchema); err != nil {
		return errf("invalid output: %v", err), nil
	}

	return &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: string(outputJSON)}},
		StructuredContent: output,
	}, nil
}

// unmarshalAndValidate unmarshals data to a map[string]any, then validates that against res.
func unmarshalAndValidate(data []byte, res *jsonschema.Resolved) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	return res.Validate(m)
}

var (
	inputSchema = &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"name": {Type: "string", MaxLength: jsonschema.Ptr(10)},
		},
	}
	outputSchema = &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"greeting": {Type: "string"},
		},
	}
)

func newManualGreeter() (*manualGreeter, error) {
	resIn, err := inputSchema.Resolve(nil)
	if err != nil {
		return nil, err
	}
	resOut, err := outputSchema.Resolve(nil)
	if err != nil {
		return nil, err
	}
	return &manualGreeter{
		inputSchema:  resIn,
		outputSchema: resOut,
	}, nil
}

func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: "greeter"}, nil)

	// Add the 'greeting' tool in a few different ways.

	// First, we can just use [mcp.AddTool], and get the out-of-the-box handling
	// it provides for schema inference, validation, parsing, and packing the
	// result.
	mcp.AddTool(server, &mcp.Tool{Name: "simple greeting"}, simpleGreeting)

	// Alternatively, we can create our schemas entirely manually, and add them
	// using [mcp.Server.AddTool]. Since we're using the 'raw' API, we have to do
	// the parsing and validation ourselves
	manual, err := newManualGreeter()
	if err != nil {
		log.Fatal(err)
	}
	server.AddTool(&mcp.Tool{
		Name:         "manual greeting",
		InputSchema:  inputSchema,
		OutputSchema: outputSchema,
	}, manual.greet)

	// We can even use raw schema values. In this case, note that we're not
	// validating the input at all.
	server.AddTool(&mcp.Tool{
		Name:        "unvalidated greeting",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"user":{"type":"string"}}}`),
	}, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Note: no validation!
		var args struct{ User string }
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Hi " + args.User}},
		}, nil
	})

	// Finally, note that we can also use custom schemas with a ToolHandlerFor.
	// We can do this in two ways: by using one of the schema values constructed
	// above, or by using jsonschema.For and adjusting the resulting schema.
	mcp.AddTool(server, &mcp.Tool{
		Name:        "customized greeting 1",
		InputSchema: inputSchema,
		// OutputSchema will still be derived from Output.
	}, simpleGreeting)

	customSchema, err := jsonschema.For[Input](nil)
	if err != nil {
		log.Fatal(err)
	}
	customSchema.Properties["name"].MaxLength = jsonschema.Ptr(10)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "customized greeting 2",
		InputSchema: customSchema,
	}, simpleGreeting)

	// Now run the server.
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Printf("Server failed: %v", err)
	}
}
</content>
</file>
<file path="examples/server/toolschemas/main_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestGreet(t *testing.T) {
	manual, err := newManualGreeter()
	if err != nil {
		t.Fatal(err)
	}
	res, err := manual.greet(context.Background(), &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"name": "Bob"}`),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("tool error: %q", res.Content[0].(*mcp.TextContent).Text)
	}
}
</content>
</file>
<file path="internal/docs/doc.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

//go:generate -command weave go run golang.org/x/example/internal/cmd/weave@latest
//go:generate weave -o ../../docs/README.md ./README.src.md
//go:generate weave -o ../../docs/protocol.md ./protocol.src.md
//go:generate weave -o ../../docs/client.md ./client.src.md
//go:generate weave -o ../../docs/server.md ./server.src.md
//go:generate weave -o ../../docs/troubleshooting.md ./troubleshooting.src.md

// The doc package generates the documentation at /doc, via go:generate.
//
// Tests in this package are used for examples.
package docs
</content>
</file>
<file path="internal/jsonrpc2/conn.go">
<type>go</type>
<content>
// Copyright 2018 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package jsonrpc2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// Binder builds a connection configuration.
// This may be used in servers to generate a new configuration per connection.
// ConnectionOptions itself implements Binder returning itself unmodified, to
// allow for the simple cases where no per connection information is needed.
type Binder interface {
	// Bind returns the ConnectionOptions to use when establishing the passed-in
	// Connection.
	//
	// The connection is not ready to use when Bind is called,
	// but Bind may close it without reading or writing to it.
	Bind(context.Context, *Connection) ConnectionOptions
}

// A BinderFunc implements the Binder interface for a standalone Bind function.
type BinderFunc func(context.Context, *Connection) ConnectionOptions

func (f BinderFunc) Bind(ctx context.Context, c *Connection) ConnectionOptions {
	return f(ctx, c)
}

var _ Binder = BinderFunc(nil)

// ConnectionOptions holds the options for new connections.
type ConnectionOptions struct {
	// Framer allows control over the message framing and encoding.
	// If nil, HeaderFramer will be used.
	Framer Framer
	// Preempter allows registration of a pre-queue message handler.
	// If nil, no messages will be preempted.
	Preempter Preempter
	// Handler is used as the queued message handler for inbound messages.
	// If nil, all responses will be ErrNotHandled.
	Handler Handler
	// OnInternalError, if non-nil, is called with any internal errors that occur
	// while serving the connection, such as protocol errors or invariant
	// violations. (If nil, internal errors result in panics.)
	OnInternalError func(error)
}

// Connection manages the jsonrpc2 protocol, connecting responses back to their
// calls.
// Connection is bidirectional; it does not have a designated server or client
// end.
type Connection struct {
	seq int64 // must only be accessed using atomic operations

	stateMu sync.Mutex
	state   inFlightState // accessed only in updateInFlight
	done    chan struct{} // closed (under stateMu) when state.closed is true and all goroutines have completed

	writer  Writer
	handler Handler

	onInternalError func(error)
	onDone          func()
}

// inFlightState records the state of the incoming and outgoing calls on a
// Connection.
type inFlightState struct {
	connClosing bool  // true when the Connection's Close method has been called
	reading     bool  // true while the readIncoming goroutine is running
	readErr     error // non-nil when the readIncoming goroutine exits (typically io.EOF)
	writeErr    error // non-nil if a call to the Writer has failed with a non-canceled Context

	// closer shuts down and cleans up the Reader and Writer state, ideally
	// interrupting any Read or Write call that is currently blocked. It is closed
	// when the state is idle and one of: connClosing is true, readErr is non-nil,
	// or writeErr is non-nil.
	//
	// After the closer has been invoked, the closer field is set to nil
	// and the closeErr field is simultaneously set to its result.
	closer   io.Closer
	closeErr error // error returned from closer.Close

	outgoingCalls         map[ID]*AsyncCall // calls only
	outgoingNotifications int               // # of notifications awaiting "write"

	// incoming stores the total number of incoming calls and notifications
	// that have not yet written or processed a result.
	incoming int

	incomingByID map[ID]*incomingRequest // calls only

	// handlerQueue stores the backlog of calls and notifications that were not
	// already handled by a preempter.
	// The queue does not include the request currently being handled (if any).
	handlerQueue   []*incomingRequest
	handlerRunning bool
}

// updateInFlight locks the state of the connection's in-flight requests, allows
// f to mutate that state, and closes the connection if it is idle and either
// is closing or has a read or write error.
func (c *Connection) updateInFlight(f func(*inFlightState)) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	s := &c.state

	f(s)

	select {
	case <-c.done:
		// The connection was already completely done at the start of this call to
		// updateInFlight, so it must remain so. (The call to f should have noticed
		// that and avoided making any updates that would cause the state to be
		// non-idle.)
		if !s.idle() {
			panic("jsonrpc2: updateInFlight transitioned to non-idle when already done")
		}
		return
	default:
	}

	if s.idle() && s.shuttingDown(ErrUnknown) != nil {
		if s.closer != nil {
			s.closeErr = s.closer.Close()
			s.closer = nil // prevent duplicate Close calls
		}
		if s.reading {
			// The readIncoming goroutine is still running. Our call to Close should
			// cause it to exit soon, at which point it will make another call to
			// updateInFlight, set s.reading to false, and mark the Connection done.
		} else {
			// The readIncoming goroutine has exited, or never started to begin with.
			// Since everything else is idle, we're completely done.
			if c.onDone != nil {
				c.onDone()
			}
			close(c.done)
		}
	}
}

// idle reports whether the connection is in a state with no pending calls or
// notifications.
//
// If idle returns true, the readIncoming goroutine may still be running,
// but no other goroutines are doing work on behalf of the connection.
func (s *inFlightState) idle() bool {
	return len(s.outgoingCalls) == 0 && s.outgoingNotifications == 0 && s.incoming == 0 && !s.handlerRunning
}

// shuttingDown reports whether the connection is in a state that should
// disallow new (incoming and outgoing) calls. It returns either nil or
// an error that is or wraps the provided errClosing.
func (s *inFlightState) shuttingDown(errClosing error) error {
	if s.connClosing {
		// If Close has been called explicitly, it doesn't matter what state the
		// Reader and Writer are in: we shouldn't be starting new work because the
		// caller told us not to start new work.
		return errClosing
	}
	if s.readErr != nil {
		// If the read side of the connection is broken, we cannot read new call
		// requests, and cannot read responses to our outgoing calls.
		return fmt.Errorf("%w: %v", errClosing, s.readErr)
	}
	if s.writeErr != nil {
		// If the write side of the connection is broken, we cannot write responses
		// for incoming calls, and cannot write requests for outgoing calls.
		return fmt.Errorf("%w: %v", errClosing, s.writeErr)
	}
	return nil
}

// incomingRequest is used to track an incoming request as it is being handled
type incomingRequest struct {
	*Request // the request being processed
	ctx      context.Context
	cancel   context.CancelFunc
}

// Bind returns the options unmodified.
func (o ConnectionOptions) Bind(context.Context, *Connection) ConnectionOptions {
	return o
}

// A ConnectionConfig configures a bidirectional jsonrpc2 connection.
type ConnectionConfig struct {
	Reader          Reader                    // required
	Writer          Writer                    // required
	Closer          io.Closer                 // required
	Preempter       Preempter                 // optional
	Bind            func(*Connection) Handler // required
	OnDone          func()                    // optional
	OnInternalError func(error)               // optional
}

// NewConnection creates a new [Connection] object and starts processing
// incoming messages.
func NewConnection(ctx context.Context, cfg ConnectionConfig) *Connection {
	ctx = notDone{ctx}

	c := &Connection{
		state:           inFlightState{closer: cfg.Closer},
		done:            make(chan struct{}),
		writer:          cfg.Writer,
		onDone:          cfg.OnDone,
		onInternalError: cfg.OnInternalError,
	}
	c.handler = cfg.Bind(c)
	c.start(ctx, cfg.Reader, cfg.Preempter)
	return c
}

// bindConnection creates a new connection and runs it.
//
// This is used by the Dial and Serve functions to build the actual connection.
//
// The connection is closed automatically (and its resources cleaned up) when
// the last request has completed after the underlying ReadWriteCloser breaks,
// but it may be stopped earlier by calling Close (for a clean shutdown).
func bindConnection(bindCtx context.Context, rwc io.ReadWriteCloser, binder Binder, onDone func()) *Connection {
	// TODO: Should we create a new event span here?
	// This will propagate cancellation from ctx; should it?
	ctx := notDone{bindCtx}

	c := &Connection{
		state:  inFlightState{closer: rwc},
		done:   make(chan struct{}),
		onDone: onDone,
	}
	// It's tempting to set a finalizer on c to verify that the state has gone
	// idle when the connection becomes unreachable. Unfortunately, the Binder
	// interface makes that unsafe: it allows the Handler to close over the
	// Connection, which could create a reference cycle that would cause the
	// Connection to become uncollectable.

	options := binder.Bind(bindCtx, c)
	framer := options.Framer
	if framer == nil {
		framer = HeaderFramer()
	}
	c.handler = options.Handler
	if c.handler == nil {
		c.handler = defaultHandler{}
	}
	c.onInternalError = options.OnInternalError

	c.writer = framer.Writer(rwc)
	reader := framer.Reader(rwc)
	c.start(ctx, reader, options.Preempter)
	return c
}

func (c *Connection) start(ctx context.Context, reader Reader, preempter Preempter) {
	c.updateInFlight(func(s *inFlightState) {
		select {
		case <-c.done:
			// Bind already closed the connection; don't start a goroutine to read it.
			return
		default:
		}

		// The goroutine started here will continue until the underlying stream is closed.
		//
		// (If the Binder closed the Connection already, this should error out and
		// return almost immediately.)
		s.reading = true
		go c.readIncoming(ctx, reader, preempter)
	})
}

// Notify invokes the target method but does not wait for a response.
// The params will be marshaled to JSON before sending over the wire, and will
// be handed to the method invoked.
func (c *Connection) Notify(ctx context.Context, method string, params any) (err error) {
	attempted := false

	defer func() {
		if attempted {
			c.updateInFlight(func(s *inFlightState) {
				s.outgoingNotifications--
			})
		}
	}()

	c.updateInFlight(func(s *inFlightState) {
		// If the connection is shutting down, allow outgoing notifications only if
		// there is at least one call still in flight. The number of calls in flight
		// cannot increase once shutdown begins, and allowing outgoing notifications
		// may permit notifications that will cancel in-flight calls.
		if len(s.outgoingCalls) == 0 && len(s.incomingByID) == 0 {
			err = s.shuttingDown(ErrClientClosing)
			if err != nil {
				return
			}
		}
		s.outgoingNotifications++
		attempted = true
	})
	if err != nil {
		return err
	}

	notify, err := NewNotification(method, params)
	if err != nil {
		return fmt.Errorf("marshaling notify parameters: %v", err)
	}

	return c.write(ctx, notify)
}

// Call invokes the target method and returns an object that can be used to await the response.
// The params will be marshaled to JSON before sending over the wire, and will
// be handed to the method invoked.
// You do not have to wait for the response, it can just be ignored if not needed.
// If sending the call failed, the response will be ready and have the error in it.
func (c *Connection) Call(ctx context.Context, method string, params any) *AsyncCall {
	// Generate a new request identifier.
	id := Int64ID(atomic.AddInt64(&c.seq, 1))

	ac := &AsyncCall{
		id:    id,
		ready: make(chan struct{}),
	}
	// When this method returns, either ac is retired, or the request has been
	// written successfully and the call is awaiting a response (to be provided by
	// the readIncoming goroutine).

	call, err := NewCall(ac.id, method, params)
	if err != nil {
		ac.retire(&Response{ID: id, Error: fmt.Errorf("marshaling call parameters: %w", err)})
		return ac
	}

	c.updateInFlight(func(s *inFlightState) {
		err = s.shuttingDown(ErrClientClosing)
		if err != nil {
			return
		}
		if s.outgoingCalls == nil {
			s.outgoingCalls = make(map[ID]*AsyncCall)
		}
		s.outgoingCalls[ac.id] = ac
	})
	if err != nil {
		ac.retire(&Response{ID: id, Error: err})
		return ac
	}

	if err := c.write(ctx, call); err != nil {
		// Sending failed. We will never get a response, so deliver a fake one if it
		// wasn't already retired by the connection breaking.
		c.updateInFlight(func(s *inFlightState) {
			if s.outgoingCalls[ac.id] == ac {
				delete(s.outgoingCalls, ac.id)
				ac.retire(&Response{ID: id, Error: err})
			} else {
				// ac was already retired by the readIncoming goroutine:
				// perhaps our write raced with the Read side of the connection breaking.
			}
		})
	}
	return ac
}

// Async, signals that the current jsonrpc2 request may be handled
// asynchronously to subsequent requests, when ctx is the request context.
//
// Async must be called at most once on each request's context (and its
// descendants).
func Async(ctx context.Context) {
	if r, ok := ctx.Value(asyncKey).(*releaser); ok {
		r.release(false)
	}
}

type asyncKeyType struct{}

var asyncKey = asyncKeyType{}

// A releaser implements concurrency safe 'releasing' of async requests. (A
// request is released when it is allowed to run concurrent with other
// requests, via a call to [Async].)
type releaser struct {
	mu       sync.Mutex
	ch       chan struct{}
	released bool
}

// release closes the associated channel. If soft is set, multiple calls to
// release are allowed.
func (r *releaser) release(soft bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.released {
		if !soft {
			panic("jsonrpc2.Async called multiple times")
		}
	} else {
		close(r.ch)
		r.released = true
	}
}

type AsyncCall struct {
	id       ID
	ready    chan struct{} // closed after response has been set
	response *Response
}

// ID used for this call.
// This can be used to cancel the call if needed.
func (ac *AsyncCall) ID() ID { return ac.id }

// IsReady can be used to check if the result is already prepared.
// This is guaranteed to return true on a result for which Await has already
// returned, or a call that failed to send in the first place.
func (ac *AsyncCall) IsReady() bool {
	select {
	case <-ac.ready:
		return true
	default:
		return false
	}
}

// retire processes the response to the call.
func (ac *AsyncCall) retire(response *Response) {
	select {
	case <-ac.ready:
		panic(fmt.Sprintf("jsonrpc2: retire called twice for ID %v", ac.id))
	default:
	}

	ac.response = response
	close(ac.ready)
}

// Await waits for (and decodes) the results of a Call.
// The response will be unmarshaled from JSON into the result.
func (ac *AsyncCall) Await(ctx context.Context, result any) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ac.ready:
	}
	if ac.response.Error != nil {
		return ac.response.Error
	}
	if result == nil {
		return nil
	}
	return json.Unmarshal(ac.response.Result, result)
}

// Cancel cancels the Context passed to the Handle call for the inbound message
// with the given ID.
//
// Cancel will not complain if the ID is not a currently active message, and it
// will not cause any messages that have not arrived yet with that ID to be
// cancelled.
func (c *Connection) Cancel(id ID) {
	var req *incomingRequest
	c.updateInFlight(func(s *inFlightState) {
		req = s.incomingByID[id]
	})
	if req != nil {
		req.cancel()
	}
}

// Wait blocks until the connection is fully closed, but does not close it.
func (c *Connection) Wait() error {
	return c.wait(true)
}

// wait for the connection to close, and aggregates the most cause of its
// termination, if abnormal.
//
// The fromWait argument allows this logic to be shared with Close, where we
// only want to expose the closeErr.
//
// (Previously, Wait also only returned the closeErr, which was misleading if
// the connection was broken for another reason).
func (c *Connection) wait(fromWait bool) error {
	var err error
	<-c.done
	c.updateInFlight(func(s *inFlightState) {
		if fromWait {
			if !errors.Is(s.readErr, io.EOF) {
				err = s.readErr
			}
			if err == nil && !errors.Is(s.writeErr, io.EOF) {
				err = s.writeErr
			}
		}
		if err == nil {
			err = s.closeErr
		}
	})
	return err
}

// Close stops accepting new requests, waits for in-flight requests and enqueued
// Handle calls to complete, and then closes the underlying stream.
//
// After the start of a Close, notification requests (that lack IDs and do not
// receive responses) will continue to be passed to the Preempter, but calls
// with IDs will receive immediate responses with ErrServerClosing, and no new
// requests (not even notifications!) will be enqueued to the Handler.
func (c *Connection) Close() error {
	// Stop handling new requests, and interrupt the reader (by closing the
	// connection) as soon as the active requests finish.
	c.updateInFlight(func(s *inFlightState) { s.connClosing = true })
	return c.wait(false)
}

// readIncoming collects inbound messages from the reader and delivers them, either responding
// to outgoing calls or feeding requests to the queue.
func (c *Connection) readIncoming(ctx context.Context, reader Reader, preempter Preempter) {
	var err error
	for {
		var msg Message
		msg, err = reader.Read(ctx)
		if err != nil {
			break
		}

		switch msg := msg.(type) {
		case *Request:
			c.acceptRequest(ctx, msg, preempter)

		case *Response:
			c.updateInFlight(func(s *inFlightState) {
				if ac, ok := s.outgoingCalls[msg.ID]; ok {
					delete(s.outgoingCalls, msg.ID)
					ac.retire(msg)
				} else {
					// TODO: How should we report unexpected responses?
				}
			})

		default:
			c.internalErrorf("Read returned an unexpected message of type %T", msg)
		}
	}

	c.updateInFlight(func(s *inFlightState) {
		s.reading = false
		s.readErr = err

		// Retire any outgoing requests that were still in flight: with the Reader no
		// longer being processed, they necessarily cannot receive a response.
		for id, ac := range s.outgoingCalls {
			ac.retire(&Response{ID: id, Error: err})
		}
		s.outgoingCalls = nil
	})
}

// acceptRequest either handles msg synchronously or enqueues it to be handled
// asynchronously.
func (c *Connection) acceptRequest(ctx context.Context, msg *Request, preempter Preempter) {
	// In theory notifications cannot be cancelled, but we build them a cancel
	// context anyway.
	reqCtx, cancel := context.WithCancel(ctx)
	req := &incomingRequest{
		Request: msg,
		ctx:     reqCtx,
		cancel:  cancel,
	}

	// If the request is a call, add it to the incoming map so it can be
	// cancelled (or responded) by ID.
	var err error
	c.updateInFlight(func(s *inFlightState) {
		s.incoming++

		if req.IsCall() {
			if s.incomingByID[req.ID] != nil {
				err = fmt.Errorf("%w: request ID %v already in use", ErrInvalidRequest, req.ID)
				req.ID = ID{} // Don't misattribute this error to the existing request.
				return
			}

			if s.incomingByID == nil {
				s.incomingByID = make(map[ID]*incomingRequest)
			}
			s.incomingByID[req.ID] = req

			// When shutting down, reject all new Call requests, even if they could
			// theoretically be handled by the preempter. The preempter could return
			// ErrAsyncResponse, which would increase the amount of work in flight
			// when we're trying to ensure that it strictly decreases.
			err = s.shuttingDown(ErrServerClosing)
		}
	})
	if err != nil {
		c.processResult("acceptRequest", req, nil, err)
		return
	}

	if preempter != nil {
		result, err := preempter.Preempt(req.ctx, req.Request)

		if !errors.Is(err, ErrNotHandled) {
			c.processResult("Preempt", req, result, err)
			return
		}
	}

	c.updateInFlight(func(s *inFlightState) {
		// If the connection is shutting down, don't enqueue anything to the
		// handler — not even notifications. That ensures that if the handler
		// continues to make progress, it will eventually become idle and
		// close the connection.
		err = s.shuttingDown(ErrServerClosing)
		if err != nil {
			return
		}

		// We enqueue requests that have not been preempted to an unbounded slice.
		// Unfortunately, we cannot in general limit the size of the handler
		// queue: we have to read every response that comes in on the wire
		// (because it may be responding to a request issued by, say, an
		// asynchronous handler), and in order to get to that response we have
		// to read all of the requests that came in ahead of it.
		s.handlerQueue = append(s.handlerQueue, req)
		if !s.handlerRunning {
			// We start the handleAsync goroutine when it has work to do, and let it
			// exit when the queue empties.
			//
			// Otherwise, in order to synchronize the handler we would need some other
			// goroutine (probably readIncoming?) to explicitly wait for handleAsync
			// to finish, and that would complicate error reporting: either the error
			// report from the goroutine would be blocked on the handler emptying its
			// queue (which was tried, and introduced a deadlock detected by
			// TestCloseCallRace), or the error would need to be reported separately
			// from synchronizing completion. Allowing the handler goroutine to exit
			// when idle seems simpler than trying to implement either of those
			// alternatives correctly.
			s.handlerRunning = true
			go c.handleAsync()
		}
	})
	if err != nil {
		c.processResult("acceptRequest", req, nil, err)
	}
}

// handleAsync invokes the handler on the requests in the handler queue
// sequentially until the queue is empty.
func (c *Connection) handleAsync() {
	for {
		var req *incomingRequest
		c.updateInFlight(func(s *inFlightState) {
			if len(s.handlerQueue) > 0 {
				req, s.handlerQueue = s.handlerQueue[0], s.handlerQueue[1:]
			} else {
				s.handlerRunning = false
			}
		})
		if req == nil {
			return
		}

		// Only deliver to the Handler if not already canceled.
		if err := req.ctx.Err(); err != nil {
			c.updateInFlight(func(s *inFlightState) {
				if s.writeErr != nil {
					// Assume that req.ctx was canceled due to s.writeErr.
					// TODO(#51365): use a Context API to plumb this through req.ctx.
					err = fmt.Errorf("%w: %v", ErrServerClosing, s.writeErr)
				}
			})
			c.processResult("handleAsync", req, nil, err)
			continue
		}

		releaser := &releaser{ch: make(chan struct{})}
		ctx := context.WithValue(req.ctx, asyncKey, releaser)
		go func() {
			defer releaser.release(true)
			result, err := c.handler.Handle(ctx, req.Request)
			c.processResult(c.handler, req, result, err)
		}()
		<-releaser.ch
	}
}

// processResult processes the result of a request and, if appropriate, sends a response.
func (c *Connection) processResult(from any, req *incomingRequest, result any, err error) error {
	switch err {
	case ErrNotHandled, ErrMethodNotFound:
		// Add detail describing the unhandled method.
		err = fmt.Errorf("%w: %q", ErrMethodNotFound, req.Method)
	}

	if result != nil && err != nil {
		c.internalErrorf("%#v returned a non-nil result with a non-nil error for %s:\n%v\n%#v", from, req.Method, err, result)
		result = nil // Discard the spurious result and respond with err.
	}

	if req.IsCall() {
		if result == nil && err == nil {
			err = c.internalErrorf("%#v returned a nil result and nil error for a %q Request that requires a Response", from, req.Method)
		}

		response, respErr := NewResponse(req.ID, result, err)

		// The caller could theoretically reuse the request's ID as soon as we've
		// sent the response, so ensure that it is removed from the incoming map
		// before sending.
		c.updateInFlight(func(s *inFlightState) {
			delete(s.incomingByID, req.ID)
		})
		if respErr == nil {
			writeErr := c.write(notDone{req.ctx}, response)
			if err == nil {
				err = writeErr
			}
		} else {
			err = c.internalErrorf("%#v returned a malformed result for %q: %w", from, req.Method, respErr)
		}
	} else { // req is a notification
		if result != nil {
			err = c.internalErrorf("%#v returned a non-nil result for a %q Request without an ID", from, req.Method)
		} else if err != nil {
			err = fmt.Errorf("%w: %q notification failed: %v", ErrInternal, req.Method, err)
		}
	}
	if err != nil {
		// TODO: can/should we do anything with this error beyond writing it to the event log?
		// (Is this the right label to attach to the log?)
	}

	// Cancel the request to free any associated resources.
	req.cancel()
	c.updateInFlight(func(s *inFlightState) {
		if s.incoming == 0 {
			panic("jsonrpc2: processResult called when incoming count is already zero")
		}
		s.incoming--
	})
	return nil
}

// write is used by all things that write outgoing messages, including replies.
// it makes sure that writes are atomic
func (c *Connection) write(ctx context.Context, msg Message) error {
	var err error
	// Fail writes immediately if the connection is shutting down.
	//
	// TODO(rfindley): should we allow cancellation notifications through? It
	// could be the case that writes can still succeed.
	c.updateInFlight(func(s *inFlightState) {
		err = s.shuttingDown(ErrServerClosing)
	})
	if err == nil {
		err = c.writer.Write(ctx, msg)
	}

	// For rejected requests, we don't set the writeErr (which would break the
	// connection). They can just be returned to the caller.
	if errors.Is(err, ErrRejected) {
		return err
	}

	if err != nil && ctx.Err() == nil {
		// The call to Write failed, and since ctx.Err() is nil we can't attribute
		// the failure (even indirectly) to Context cancellation. The writer appears
		// to be broken, and future writes are likely to also fail.
		//
		// If the read side of the connection is also broken, we might not even be
		// able to receive cancellation notifications. Since we can't reliably write
		// the results of incoming calls and can't receive explicit cancellations,
		// cancel the calls now.
		c.updateInFlight(func(s *inFlightState) {
			if s.writeErr == nil {
				s.writeErr = err
				for _, r := range s.incomingByID {
					r.cancel()
				}
			}
		})
	}

	return err
}

// internalErrorf reports an internal error. By default it panics, but if
// c.onInternalError is non-nil it instead calls that and returns an error
// wrapping ErrInternal.
func (c *Connection) internalErrorf(format string, args ...any) error {
	err := fmt.Errorf(format, args...)
	if c.onInternalError == nil {
		panic("jsonrpc2: " + err.Error())
	}
	c.onInternalError(err)

	return fmt.Errorf("%w: %v", ErrInternal, err)
}

// notDone is a context.Context wrapper that returns a nil Done channel.
type notDone struct{ ctx context.Context }

func (ic notDone) Value(key any) any {
	return ic.ctx.Value(key)
}

func (notDone) Done() <-chan struct{}       { return nil }
func (notDone) Err() error                  { return nil }
func (notDone) Deadline() (time.Time, bool) { return time.Time{}, false }
</content>
</file>
<file path="internal/jsonrpc2/frame.go">
<type>go</type>
<content>
// Copyright 2018 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package jsonrpc2

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

// Reader abstracts the transport mechanics from the JSON RPC protocol.
// A Conn reads messages from the reader it was provided on construction,
// and assumes that each call to Read fully transfers a single message,
// or returns an error.
//
// A reader is not safe for concurrent use, it is expected it will be used by
// a single Conn in a safe manner.
type Reader interface {
	// Read gets the next message from the stream.
	Read(context.Context) (Message, error)
}

// Writer abstracts the transport mechanics from the JSON RPC protocol.
// A Conn writes messages using the writer it was provided on construction,
// and assumes that each call to Write fully transfers a single message,
// or returns an error.
//
// A writer must be safe for concurrent use, as writes may occur concurrently
// in practice: libraries may make calls or respond to requests asynchronously.
type Writer interface {
	// Write sends a message to the stream.
	Write(context.Context, Message) error
}

// Framer wraps low level byte readers and writers into jsonrpc2 message
// readers and writers.
// It is responsible for the framing and encoding of messages into wire form.
//
// TODO(rfindley): rethink the framer interface, as with JSONRPC2 batching
// there is a need for Reader and Writer to be correlated, and while the
// implementation of framing here allows that, it is not made explicit by the
// interface.
//
// Perhaps a better interface would be
//
//	Frame(io.ReadWriteCloser) (Reader, Writer).
type Framer interface {
	// Reader wraps a byte reader into a message reader.
	Reader(io.Reader) Reader
	// Writer wraps a byte writer into a message writer.
	Writer(io.Writer) Writer
}

// RawFramer returns a new Framer.
// The messages are sent with no wrapping, and rely on json decode consistency
// to determine message boundaries.
func RawFramer() Framer { return rawFramer{} }

type rawFramer struct{}
type rawReader struct{ in *json.Decoder }
type rawWriter struct {
	mu  sync.Mutex
	out io.Writer
}

func (rawFramer) Reader(rw io.Reader) Reader {
	return &rawReader{in: json.NewDecoder(rw)}
}

func (rawFramer) Writer(rw io.Writer) Writer {
	return &rawWriter{out: rw}
}

func (r *rawReader) Read(ctx context.Context) (Message, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	var raw json.RawMessage
	if err := r.in.Decode(&raw); err != nil {
		return nil, err
	}
	msg, err := DecodeMessage(raw)
	return msg, err
}

func (w *rawWriter) Write(ctx context.Context, msg Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	data, err := EncodeMessage(msg)
	if err != nil {
		return fmt.Errorf("marshaling message: %v", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	_, err = w.out.Write(data)
	return err
}

// HeaderFramer returns a new Framer.
// The messages are sent with HTTP content length and MIME type headers.
// This is the format used by LSP and others.
func HeaderFramer() Framer { return headerFramer{} }

type headerFramer struct{}
type headerReader struct{ in *bufio.Reader }
type headerWriter struct {
	mu  sync.Mutex
	out io.Writer
}

func (headerFramer) Reader(rw io.Reader) Reader {
	return &headerReader{in: bufio.NewReader(rw)}
}

func (headerFramer) Writer(rw io.Writer) Writer {
	return &headerWriter{out: rw}
}

func (r *headerReader) Read(ctx context.Context) (Message, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	firstRead := true // to detect a clean EOF below
	var contentLength int64
	// read the header, stop on the first empty line
	for {
		line, err := r.in.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				if firstRead && line == "" {
					return nil, io.EOF // clean EOF
				}
				err = io.ErrUnexpectedEOF
			}
			return nil, fmt.Errorf("failed reading header line: %w", err)
		}
		firstRead = false

		line = strings.TrimSpace(line)
		// check we have a header line
		if line == "" {
			break
		}
		colon := strings.IndexRune(line, ':')
		if colon < 0 {
			return nil, fmt.Errorf("invalid header line %q", line)
		}
		name, value := line[:colon], strings.TrimSpace(line[colon+1:])
		switch name {
		case "Content-Length":
			if contentLength, err = strconv.ParseInt(value, 10, 32); err != nil {
				return nil, fmt.Errorf("failed parsing Content-Length: %v", value)
			}
			if contentLength <= 0 {
				return nil, fmt.Errorf("invalid Content-Length: %v", contentLength)
			}
		default:
			// ignoring unknown headers
		}
	}
	if contentLength == 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}
	data := make([]byte, contentLength)
	_, err := io.ReadFull(r.in, data)
	if err != nil {
		return nil, err
	}
	msg, err := DecodeMessage(data)
	return msg, err
}

func (w *headerWriter) Write(ctx context.Context, msg Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	data, err := EncodeMessage(msg)
	if err != nil {
		return fmt.Errorf("marshaling message: %v", err)
	}
	_, err = fmt.Fprintf(w.out, "Content-Length: %v\r\n\r\n", len(data))
	if err == nil {
		_, err = w.out.Write(data)
	}
	return err
}
</content>
</file>
<file path="internal/jsonrpc2/jsonrpc2.go">
<type>go</type>
<content>
// Copyright 2018 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// Package jsonrpc2 is a minimal implementation of the JSON RPC 2 spec.
// https://www.jsonrpc.org/specification
// It is intended to be compatible with other implementations at the wire level.
package jsonrpc2

import (
	"context"
	"errors"
)

var (
	// ErrIdleTimeout is returned when serving timed out waiting for new connections.
	ErrIdleTimeout = errors.New("timed out waiting for new connections")

	// ErrNotHandled is returned from a Handler or Preempter to indicate it did
	// not handle the request.
	//
	// If a Handler returns ErrNotHandled, the server replies with
	// ErrMethodNotFound.
	ErrNotHandled = errors.New("JSON RPC not handled")
)

// Preempter handles messages on a connection before they are queued to the main
// handler.
// Primarily this is used for cancel handlers or notifications for which out of
// order processing is not an issue.
type Preempter interface {
	// Preempt is invoked for each incoming request before it is queued for handling.
	//
	// If Preempt returns ErrNotHandled, the request will be queued,
	// and eventually passed to a Handle call.
	//
	// Otherwise, the result and error are processed as if returned by Handle.
	//
	// Preempt must not block. (The Context passed to it is for Values only.)
	Preempt(ctx context.Context, req *Request) (result any, err error)
}

// A PreempterFunc implements the Preempter interface for a standalone Preempt function.
type PreempterFunc func(ctx context.Context, req *Request) (any, error)

func (f PreempterFunc) Preempt(ctx context.Context, req *Request) (any, error) {
	return f(ctx, req)
}

var _ Preempter = PreempterFunc(nil)

// Handler handles messages on a connection.
type Handler interface {
	// Handle is invoked sequentially for each incoming request that has not
	// already been handled by a Preempter.
	//
	// If the Request has a nil ID, Handle must return a nil result,
	// and any error may be logged but will not be reported to the caller.
	//
	// If the Request has a non-nil ID, Handle must return either a
	// non-nil, JSON-marshalable result, or a non-nil error.
	//
	// The Context passed to Handle will be canceled if the
	// connection is broken or the request is canceled or completed.
	// (If Handle returns ErrAsyncResponse, ctx will remain uncanceled
	// until either Cancel or Respond is called for the request's ID.)
	Handle(ctx context.Context, req *Request) (result any, err error)
}

type defaultHandler struct{}

func (defaultHandler) Preempt(context.Context, *Request) (any, error) {
	return nil, ErrNotHandled
}

func (defaultHandler) Handle(context.Context, *Request) (any, error) {
	return nil, ErrNotHandled
}

// A HandlerFunc implements the Handler interface for a standalone Handle function.
type HandlerFunc func(ctx context.Context, req *Request) (any, error)

func (f HandlerFunc) Handle(ctx context.Context, req *Request) (any, error) {
	return f(ctx, req)
}

var _ Handler = HandlerFunc(nil)

// async is a small helper for operations with an asynchronous result that you
// can wait for.
type async struct {
	ready    chan struct{} // closed when done
	firstErr chan error    // 1-buffered; contains either nil or the first non-nil error
}

func newAsync() *async {
	var a async
	a.ready = make(chan struct{})
	a.firstErr = make(chan error, 1)
	a.firstErr <- nil
	return &a
}

func (a *async) done() {
	close(a.ready)
}

func (a *async) wait() error {
	<-a.ready
	err := <-a.firstErr
	a.firstErr <- err
	return err
}

func (a *async) setError(err error) {
	storedErr := <-a.firstErr
	if storedErr == nil {
		storedErr = err
	}
	a.firstErr <- storedErr
}
</content>
</file>
<file path="internal/jsonrpc2/jsonrpc2_test.go">
<type>go</type>
<content>
// Copyright 2018 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package jsonrpc2_test

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"reflect"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/internal/jsonrpc2"
)

var callTests = []invoker{
	call{"no_args", nil, true},
	call{"one_string", "fish", "got:fish"},
	call{"one_number", 10, "got:10"},
	call{"join", []string{"a", "b", "c"}, "a/b/c"},
	sequence{"notify", []invoker{
		notify{"set", 3},
		notify{"add", 5},
		call{"get", nil, 8},
	}},
	sequence{"preempt", []invoker{
		async{"a", "wait", "a"},
		notify{"unblock", "a"},
		collect{"a", true, false},
	}},
	sequence{"basic cancel", []invoker{
		async{"b", "wait", "b"},
		cancel{"b"},
		collect{"b", nil, true},
	}},
	sequence{"queue", []invoker{
		async{"a", "wait", "a"},
		notify{"set", 1},
		notify{"add", 2},
		notify{"add", 3},
		notify{"add", 4},
		call{"peek", nil, 0}, // accumulator will not have any adds yet
		notify{"unblock", "a"},
		collect{"a", true, false},
		call{"get", nil, 10}, // accumulator now has all the adds
	}},
	sequence{"fork", []invoker{
		async{"a", "fork", "a"},
		notify{"set", 1},
		notify{"add", 2},
		notify{"add", 3},
		notify{"add", 4},
		call{"get", nil, 10}, // fork will not have blocked the adds
		notify{"unblock", "a"},
		collect{"a", true, false},
	}},
	sequence{"concurrent", []invoker{
		async{"a", "fork", "a"},
		notify{"unblock", "a"},
		async{"b", "fork", "b"},
		notify{"unblock", "b"},
		collect{"a", true, false},
		collect{"b", true, false},
	}},
}

type binder struct {
	framer  jsonrpc2.Framer
	runTest func(*handler)
}

type handler struct {
	conn        *jsonrpc2.Connection
	accumulator int
	waiters     chan map[string]chan struct{}
	calls       map[string]*jsonrpc2.AsyncCall
}

type invoker interface {
	Name() string
	Invoke(t *testing.T, ctx context.Context, h *handler)
}

type notify struct {
	method string
	params any
}

type call struct {
	method string
	params any
	expect any
}

type async struct {
	name   string
	method string
	params any
}

type collect struct {
	name   string
	expect any
	fails  bool
}

type cancel struct {
	name string
}

type sequence struct {
	name  string
	tests []invoker
}

type echo call

type cancelParams struct{ ID int64 }

func TestConnectionRaw(t *testing.T) {
	testConnection(t, jsonrpc2.RawFramer())
}

func TestConnectionHeader(t *testing.T) {
	testConnection(t, jsonrpc2.HeaderFramer())
}

func testConnection(t *testing.T, framer jsonrpc2.Framer) {
	ctx := context.Background()
	listener, err := jsonrpc2.NetPipeListener(ctx)
	if err != nil {
		t.Fatal(err)
	}
	server := jsonrpc2.NewServer(ctx, listener, binder{framer, nil})
	defer func() {
		listener.Close()
		server.Wait()
	}()

	for _, test := range callTests {
		t.Run(test.Name(), func(t *testing.T) {
			client, err := jsonrpc2.Dial(ctx,
				listener.Dialer(), binder{framer, func(h *handler) {
					// Sleep a little to a void a race with setting conn.writer in jsonrpc2.bindConnection.
					time.Sleep(50 * time.Millisecond)
					defer h.conn.Close()
					test.Invoke(t, ctx, h)
					if call, ok := test.(*call); ok {
						// also run all simple call tests in echo mode
						(*echo)(call).Invoke(t, ctx, h)
					}
				}}, nil)
			if err != nil {
				t.Fatal(err)
			}
			client.Wait()
		})
	}
}

func (test notify) Name() string { return test.method }
func (test notify) Invoke(t *testing.T, ctx context.Context, h *handler) {
	if err := h.conn.Notify(ctx, test.method, test.params); err != nil {
		t.Fatalf("%v:Notify failed: %v", test.method, err)
	}
}

func (test call) Name() string { return test.method }
func (test call) Invoke(t *testing.T, ctx context.Context, h *handler) {
	results := newResults(test.expect)
	if err := h.conn.Call(ctx, test.method, test.params).Await(ctx, results); err != nil {
		t.Fatalf("%v:Call failed: %v", test.method, err)
	}
	verifyResults(t, test.method, results, test.expect)
}

func (test echo) Invoke(t *testing.T, ctx context.Context, h *handler) {
	results := newResults(test.expect)
	if err := h.conn.Call(ctx, "echo", []any{test.method, test.params}).Await(ctx, results); err != nil {
		t.Fatalf("%v:Echo failed: %v", test.method, err)
	}
	verifyResults(t, test.method, results, test.expect)
}

func (test async) Name() string { return test.name }
func (test async) Invoke(t *testing.T, ctx context.Context, h *handler) {
	h.calls[test.name] = h.conn.Call(ctx, test.method, test.params)
}

func (test collect) Name() string { return test.name }
func (test collect) Invoke(t *testing.T, ctx context.Context, h *handler) {
	o := h.calls[test.name]
	results := newResults(test.expect)
	err := o.Await(ctx, results)
	switch {
	case test.fails && err == nil:
		t.Fatalf("%v:Collect was supposed to fail", test.name)
	case !test.fails && err != nil:
		t.Fatalf("%v:Collect failed: %v", test.name, err)
	}
	verifyResults(t, test.name, results, test.expect)
}

func (test cancel) Name() string { return test.name }
func (test cancel) Invoke(t *testing.T, ctx context.Context, h *handler) {
	o := h.calls[test.name]
	if err := h.conn.Notify(ctx, "cancel", &cancelParams{o.ID().Raw().(int64)}); err != nil {
		t.Fatalf("%v:Collect failed: %v", test.name, err)
	}
}

func (test sequence) Name() string { return test.name }
func (test sequence) Invoke(t *testing.T, ctx context.Context, h *handler) {
	for _, child := range test.tests {
		child.Invoke(t, ctx, h)
	}
}

// newResults makes a new empty copy of the expected type to put the results into
func newResults(expect any) any {
	switch e := expect.(type) {
	case []any:
		var r []any
		for _, v := range e {
			r = append(r, reflect.New(reflect.TypeOf(v)).Interface())
		}
		return r
	case nil:
		return nil
	default:
		return reflect.New(reflect.TypeOf(expect)).Interface()
	}
}

// verifyResults compares the results to the expected values
func verifyResults(t *testing.T, method string, results any, expect any) {
	if expect == nil {
		if results != nil {
			t.Errorf("%v:Got results %+v where none expected", method, expect)
		}
		return
	}
	val := reflect.Indirect(reflect.ValueOf(results)).Interface()
	if !reflect.DeepEqual(val, expect) {
		t.Errorf("%v:Results are incorrect, got %+v expect %+v", method, val, expect)
	}
}

func (b binder) Bind(ctx context.Context, conn *jsonrpc2.Connection) jsonrpc2.ConnectionOptions {
	h := &handler{
		conn:    conn,
		waiters: make(chan map[string]chan struct{}, 1),
		calls:   make(map[string]*jsonrpc2.AsyncCall),
	}
	h.waiters <- make(map[string]chan struct{})
	if b.runTest != nil {
		go b.runTest(h)
	}
	return jsonrpc2.ConnectionOptions{
		Framer:    b.framer,
		Preempter: h,
		Handler:   h,
	}
}

func (h *handler) waiter(name string) chan struct{} {
	waiters := <-h.waiters
	defer func() { h.waiters <- waiters }()
	waiter, found := waiters[name]
	if !found {
		waiter = make(chan struct{})
		waiters[name] = waiter
	}
	return waiter
}

func (h *handler) Preempt(ctx context.Context, req *jsonrpc2.Request) (any, error) {
	switch req.Method {
	case "unblock":
		var name string
		if err := json.Unmarshal(req.Params, &name); err != nil {
			return nil, fmt.Errorf("%w: %s", jsonrpc2.ErrParse, err)
		}
		close(h.waiter(name))
		return nil, nil
	case "peek":
		if len(req.Params) > 0 {
			return nil, fmt.Errorf("%w: expected no params", jsonrpc2.ErrInvalidParams)
		}
		return h.accumulator, nil
	case "cancel":
		var params cancelParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, fmt.Errorf("%w: %s", jsonrpc2.ErrParse, err)
		}
		h.conn.Cancel(jsonrpc2.Int64ID(params.ID))
		return nil, nil
	default:
		return nil, jsonrpc2.ErrNotHandled
	}
}

func (h *handler) Handle(ctx context.Context, req *jsonrpc2.Request) (any, error) {
	switch req.Method {
	case "no_args":
		if len(req.Params) > 0 {
			return nil, fmt.Errorf("%w: expected no params", jsonrpc2.ErrInvalidParams)
		}
		return true, nil
	case "one_string":
		var v string
		if err := json.Unmarshal(req.Params, &v); err != nil {
			return nil, fmt.Errorf("%w: %s", jsonrpc2.ErrParse, err)
		}
		return "got:" + v, nil
	case "one_number":
		var v int
		if err := json.Unmarshal(req.Params, &v); err != nil {
			return nil, fmt.Errorf("%w: %s", jsonrpc2.ErrParse, err)
		}
		return fmt.Sprintf("got:%d", v), nil
	case "set":
		var v int
		if err := json.Unmarshal(req.Params, &v); err != nil {
			return nil, fmt.Errorf("%w: %s", jsonrpc2.ErrParse, err)
		}
		h.accumulator = v
		return nil, nil
	case "add":
		var v int
		if err := json.Unmarshal(req.Params, &v); err != nil {
			return nil, fmt.Errorf("%w: %s", jsonrpc2.ErrParse, err)
		}
		h.accumulator += v
		return nil, nil
	case "get":
		if len(req.Params) > 0 {
			return nil, fmt.Errorf("%w: expected no params", jsonrpc2.ErrInvalidParams)
		}
		return h.accumulator, nil
	case "join":
		var v []string
		if err := json.Unmarshal(req.Params, &v); err != nil {
			return nil, fmt.Errorf("%w: %s", jsonrpc2.ErrParse, err)
		}
		return path.Join(v...), nil
	case "echo":
		var v []any
		if err := json.Unmarshal(req.Params, &v); err != nil {
			return nil, fmt.Errorf("%w: %s", jsonrpc2.ErrParse, err)
		}
		var result any
		err := h.conn.Call(ctx, v[0].(string), v[1]).Await(ctx, &result)
		return result, err
	case "wait":
		var name string
		if err := json.Unmarshal(req.Params, &name); err != nil {
			return nil, fmt.Errorf("%w: %s", jsonrpc2.ErrParse, err)
		}
		select {
		case <-h.waiter(name):
			return true, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	case "fork":
		var name string
		if err := json.Unmarshal(req.Params, &name); err != nil {
			return nil, fmt.Errorf("%w: %s", jsonrpc2.ErrParse, err)
		}
		jsonrpc2.Async(ctx)
		waitFor := h.waiter(name)
		select {
		case <-waitFor:
			return true, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	default:
		return nil, jsonrpc2.ErrNotHandled
	}
}
</content>
</file>
<file path="internal/jsonrpc2/messages.go">
<type>go</type>
<content>
// Copyright 2018 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package jsonrpc2

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ID is a Request identifier, which is defined by the spec to be a string, integer, or null.
// https://www.jsonrpc.org/specification#request_object
type ID struct {
	value any
}

// MakeID coerces the given Go value to an ID. The value should be the
// default JSON marshaling of a Request identifier: nil, float64, or string.
//
// Returns an error if the value type was not a valid Request ID type.
//
// TODO: ID can't be a json.Marshaler/Unmarshaler, because we want to omitzero.
// Simplify this package by making ID json serializable once we can rely on
// omitzero.
func MakeID(v any) (ID, error) {
	switch v := v.(type) {
	case nil:
		return ID{}, nil
	case float64:
		return Int64ID(int64(v)), nil
	case string:
		return StringID(v), nil
	}
	return ID{}, fmt.Errorf("%w: invalid ID type %T", ErrParse, v)
}

// Message is the interface to all jsonrpc2 message types.
// They share no common functionality, but are a closed set of concrete types
// that are allowed to implement this interface. The message types are *Request
// and *Response.
type Message interface {
	// marshal builds the wire form from the API form.
	// It is private, which makes the set of Message implementations closed.
	marshal(to *wireCombined)
}

// Request is a Message sent to a peer to request behavior.
// If it has an ID it is a call, otherwise it is a notification.
type Request struct {
	// ID of this request, used to tie the Response back to the request.
	// This will be nil for notifications.
	ID ID
	// Method is a string containing the method name to invoke.
	Method string
	// Params is either a struct or an array with the parameters of the method.
	Params json.RawMessage
	// Extra is additional information that does not appear on the wire. It can be
	// used to pass information from the application to the underlying transport.
	Extra any
}

// Response is a Message used as a reply to a call Request.
// It will have the same ID as the call it is a response to.
type Response struct {
	// result is the content of the response.
	Result json.RawMessage
	// err is set only if the call failed.
	Error error
	// id of the request this is a response to.
	ID ID
	// Extra is additional information that does not appear on the wire. It can be
	// used to pass information from the underlying transport to the application.
	Extra any
}

// StringID creates a new string request identifier.
func StringID(s string) ID { return ID{value: s} }

// Int64ID creates a new integer request identifier.
func Int64ID(i int64) ID { return ID{value: i} }

// IsValid returns true if the ID is a valid identifier.
// The default value for ID will return false.
func (id ID) IsValid() bool { return id.value != nil }

// Raw returns the underlying value of the ID.
func (id ID) Raw() any { return id.value }

// NewNotification constructs a new Notification message for the supplied
// method and parameters.
func NewNotification(method string, params any) (*Request, error) {
	p, merr := marshalToRaw(params)
	return &Request{Method: method, Params: p}, merr
}

// NewCall constructs a new Call message for the supplied ID, method and
// parameters.
func NewCall(id ID, method string, params any) (*Request, error) {
	p, merr := marshalToRaw(params)
	return &Request{ID: id, Method: method, Params: p}, merr
}

func (msg *Request) IsCall() bool { return msg.ID.IsValid() }

func (msg *Request) marshal(to *wireCombined) {
	to.ID = msg.ID.value
	to.Method = msg.Method
	to.Params = msg.Params
}

// NewResponse constructs a new Response message that is a reply to the
// supplied. If err is set result may be ignored.
func NewResponse(id ID, result any, rerr error) (*Response, error) {
	r, merr := marshalToRaw(result)
	return &Response{ID: id, Result: r, Error: rerr}, merr
}

func (msg *Response) marshal(to *wireCombined) {
	to.ID = msg.ID.value
	to.Error = toWireError(msg.Error)
	to.Result = msg.Result
}

func toWireError(err error) *WireError {
	if err == nil {
		// no error, the response is complete
		return nil
	}
	if err, ok := err.(*WireError); ok {
		// already a wire error, just use it
		return err
	}
	result := &WireError{Message: err.Error()}
	var wrapped *WireError
	if errors.As(err, &wrapped) {
		// if we wrapped a wire error, keep the code from the wrapped error
		// but the message from the outer error
		result.Code = wrapped.Code
	}
	return result
}

func EncodeMessage(msg Message) ([]byte, error) {
	wire := wireCombined{VersionTag: wireVersion}
	msg.marshal(&wire)
	data, err := json.Marshal(&wire)
	if err != nil {
		return data, fmt.Errorf("marshaling jsonrpc message: %w", err)
	}
	return data, nil
}

// EncodeIndent is like EncodeMessage, but honors indents.
// TODO(rfindley): refactor so that this concern is handled independently.
// Perhaps we should pass in a json.Encoder?
func EncodeIndent(msg Message, prefix, indent string) ([]byte, error) {
	wire := wireCombined{VersionTag: wireVersion}
	msg.marshal(&wire)
	data, err := json.MarshalIndent(&wire, prefix, indent)
	if err != nil {
		return data, fmt.Errorf("marshaling jsonrpc message: %w", err)
	}
	return data, nil
}

func DecodeMessage(data []byte) (Message, error) {
	msg := wireCombined{}
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("unmarshaling jsonrpc message: %w", err)
	}
	if msg.VersionTag != wireVersion {
		return nil, fmt.Errorf("invalid message version tag %q; expected %q", msg.VersionTag, wireVersion)
	}
	id, err := MakeID(msg.ID)
	if err != nil {
		return nil, err
	}
	if msg.Method != "" {
		// has a method, must be a call
		return &Request{
			Method: msg.Method,
			ID:     id,
			Params: msg.Params,
		}, nil
	}
	// no method, should be a response
	if !id.IsValid() {
		return nil, ErrInvalidRequest
	}
	resp := &Response{
		ID:     id,
		Result: msg.Result,
	}
	// we have to check if msg.Error is nil to avoid a typed error
	if msg.Error != nil {
		resp.Error = msg.Error
	}
	return resp, nil
}

func marshalToRaw(obj any) (json.RawMessage, error) {
	if obj == nil {
		return nil, nil
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}
</content>
</file>
<file path="internal/jsonrpc2/net.go">
<type>go</type>
<content>
// Copyright 2018 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package jsonrpc2

import (
	"context"
	"io"
	"net"
	"os"
)

// This file contains implementations of the transport primitives that use the standard network
// package.

// NetListenOptions is the optional arguments to the NetListen function.
type NetListenOptions struct {
	NetListenConfig net.ListenConfig
	NetDialer       net.Dialer
}

// NetListener returns a new Listener that listens on a socket using the net package.
func NetListener(ctx context.Context, network, address string, options NetListenOptions) (Listener, error) {
	ln, err := options.NetListenConfig.Listen(ctx, network, address)
	if err != nil {
		return nil, err
	}
	return &netListener{net: ln}, nil
}

// netListener is the implementation of Listener for connections made using the net package.
type netListener struct {
	net net.Listener
}

// Accept blocks waiting for an incoming connection to the listener.
func (l *netListener) Accept(context.Context) (io.ReadWriteCloser, error) {
	return l.net.Accept()
}

// Close will cause the listener to stop listening. It will not close any connections that have
// already been accepted.
func (l *netListener) Close() error {
	addr := l.net.Addr()
	err := l.net.Close()
	if addr.Network() == "unix" {
		rerr := os.Remove(addr.String())
		if rerr != nil && err == nil {
			err = rerr
		}
	}
	return err
}

// Dialer returns a dialer that can be used to connect to the listener.
func (l *netListener) Dialer() Dialer {
	return NetDialer(l.net.Addr().Network(), l.net.Addr().String(), net.Dialer{})
}

// NetDialer returns a Dialer using the supplied standard network dialer.
func NetDialer(network, address string, nd net.Dialer) Dialer {
	return &netDialer{
		network: network,
		address: address,
		dialer:  nd,
	}
}

type netDialer struct {
	network string
	address string
	dialer  net.Dialer
}

func (n *netDialer) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	return n.dialer.DialContext(ctx, n.network, n.address)
}

// NetPipeListener returns a new Listener that listens using net.Pipe.
// It is only possibly to connect to it using the Dialer returned by the
// Dialer method, each call to that method will generate a new pipe the other
// side of which will be returned from the Accept call.
func NetPipeListener(ctx context.Context) (Listener, error) {
	return &netPiper{
		done:   make(chan struct{}),
		dialed: make(chan io.ReadWriteCloser),
	}, nil
}

// netPiper is the implementation of Listener build on top of net.Pipes.
type netPiper struct {
	done   chan struct{}
	dialed chan io.ReadWriteCloser
}

// Accept blocks waiting for an incoming connection to the listener.
func (l *netPiper) Accept(context.Context) (io.ReadWriteCloser, error) {
	// Block until the pipe is dialed or the listener is closed,
	// preferring the latter if already closed at the start of Accept.
	select {
	case <-l.done:
		return nil, net.ErrClosed
	default:
	}
	select {
	case rwc := <-l.dialed:
		return rwc, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}

// Close will cause the listener to stop listening. It will not close any connections that have
// already been accepted.
func (l *netPiper) Close() error {
	// unblock any accept calls that are pending
	close(l.done)
	return nil
}

func (l *netPiper) Dialer() Dialer {
	return l
}

func (l *netPiper) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	client, server := net.Pipe()

	select {
	case l.dialed <- server:
		return client, nil

	case <-l.done:
		client.Close()
		server.Close()
		return nil, net.ErrClosed
	}
}
</content>
</file>
<file path="internal/jsonrpc2/serve.go">
<type>go</type>
<content>
// Copyright 2020 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package jsonrpc2

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Listener is implemented by protocols to accept new inbound connections.
type Listener interface {
	// Accept accepts an inbound connection to a server.
	// It blocks until either an inbound connection is made, or the listener is closed.
	Accept(context.Context) (io.ReadWriteCloser, error)

	// Close closes the listener.
	// Any blocked Accept or Dial operations will unblock and return errors.
	Close() error

	// Dialer returns a dialer that can be used to connect to this listener
	// locally.
	// If a listener does not implement this it will return nil.
	Dialer() Dialer
}

// Dialer is used by clients to dial a server.
type Dialer interface {
	// Dial returns a new communication byte stream to a listening server.
	Dial(ctx context.Context) (io.ReadWriteCloser, error)
}

// Server is a running server that is accepting incoming connections.
type Server struct {
	listener Listener
	binder   Binder
	async    *async

	shutdownOnce sync.Once
	closing      int32 // atomic: set to nonzero when Shutdown is called
}

// Dial uses the dialer to make a new connection, wraps the returned
// reader and writer using the framer to make a stream, and then builds
// a connection on top of that stream using the binder.
//
// The returned Connection will operate independently using the Preempter and/or
// Handler provided by the Binder, and will release its own resources when the
// connection is broken, but the caller may Close it earlier to stop accepting
// (or sending) new requests.
//
// If non-nil, the onDone function is called when the connection is closed.
func Dial(ctx context.Context, dialer Dialer, binder Binder, onDone func()) (*Connection, error) {
	// dial a server
	rwc, err := dialer.Dial(ctx)
	if err != nil {
		return nil, err
	}
	return bindConnection(ctx, rwc, binder, onDone), nil
}

// NewServer starts a new server listening for incoming connections and returns
// it.
// This returns a fully running and connected server, it does not block on
// the listener.
// You can call Wait to block on the server, or Shutdown to get the sever to
// terminate gracefully.
// To notice incoming connections, use an intercepting Binder.
func NewServer(ctx context.Context, listener Listener, binder Binder) *Server {
	server := &Server{
		listener: listener,
		binder:   binder,
		async:    newAsync(),
	}
	go server.run(ctx)
	return server
}

// Wait returns only when the server has shut down.
func (s *Server) Wait() error {
	return s.async.wait()
}

// Shutdown informs the server to stop accepting new connections.
func (s *Server) Shutdown() {
	s.shutdownOnce.Do(func() {
		atomic.StoreInt32(&s.closing, 1)
		s.listener.Close()
	})
}

// run accepts incoming connections from the listener,
// If IdleTimeout is non-zero, run exits after there are no clients for this
// duration, otherwise it exits only on error.
func (s *Server) run(ctx context.Context) {
	defer s.async.done()

	var activeConns sync.WaitGroup
	for {
		rwc, err := s.listener.Accept(ctx)
		if err != nil {
			// Only Shutdown closes the listener. If we get an error after Shutdown is
			// called, assume that was the cause and don't report the error;
			// otherwise, report the error in case it is unexpected.
			if atomic.LoadInt32(&s.closing) == 0 {
				s.async.setError(err)
			}
			// We are done generating new connections for good.
			break
		}

		// A new inbound connection.
		activeConns.Add(1)
		_ = bindConnection(ctx, rwc, s.binder, activeConns.Done) // unregisters itself when done
	}
	activeConns.Wait()
}

// NewIdleListener wraps a listener with an idle timeout.
//
// When there are no active connections for at least the timeout duration,
// calls to Accept will fail with ErrIdleTimeout.
//
// A connection is considered inactive as soon as its Close method is called.
func NewIdleListener(timeout time.Duration, wrap Listener) Listener {
	l := &idleListener{
		wrapped:   wrap,
		timeout:   timeout,
		active:    make(chan int, 1),
		timedOut:  make(chan struct{}),
		idleTimer: make(chan *time.Timer, 1),
	}
	l.idleTimer <- time.AfterFunc(l.timeout, l.timerExpired)
	return l
}

type idleListener struct {
	wrapped Listener
	timeout time.Duration

	// Only one of these channels is receivable at any given time.
	active    chan int         // count of active connections; closed when Close is called if not timed out
	timedOut  chan struct{}    // closed when the idle timer expires
	idleTimer chan *time.Timer // holds the timer only when idle
}

// Accept accepts an incoming connection.
//
// If an incoming connection is accepted concurrent to the listener being closed
// due to idleness, the new connection is immediately closed.
func (l *idleListener) Accept(ctx context.Context) (io.ReadWriteCloser, error) {
	rwc, err := l.wrapped.Accept(ctx)

	select {
	case n, ok := <-l.active:
		if err != nil {
			if ok {
				l.active <- n
			}
			return nil, err
		}
		if ok {
			l.active <- n + 1
		} else {
			// l.wrapped.Close Close has been called, but Accept returned a
			// connection. This race can occur with concurrent Accept and Close calls
			// with any net.Listener, and it is benign: since the listener was closed
			// explicitly, it can't have also timed out.
		}
		return l.newConn(rwc), nil

	case <-l.timedOut:
		if err == nil {
			// Keeping the connection open would leave the listener simultaneously
			// active and closed due to idleness, which would be contradictory and
			// confusing. Close the connection and pretend that it never happened.
			rwc.Close()
		} else {
			// In theory the timeout could have raced with an unrelated error return
			// from Accept. However, ErrIdleTimeout is arguably still valid (since we
			// would have closed due to the timeout independent of the error), and the
			// harm from returning a spurious ErrIdleTimeout is negligible anyway.
		}
		return nil, ErrIdleTimeout

	case timer := <-l.idleTimer:
		if err != nil {
			// The idle timer doesn't run until it receives itself from the idleTimer
			// channel, so it can't have called l.wrapped.Close yet and thus err can't
			// be ErrIdleTimeout. Leave the idle timer as it was and return whatever
			// error we got.
			l.idleTimer <- timer
			return nil, err
		}

		if !timer.Stop() {
			// Failed to stop the timer — the timer goroutine is in the process of
			// firing. Send the timer back to the timer goroutine so that it can
			// safely close the timedOut channel, and then wait for the listener to
			// actually be closed before we return ErrIdleTimeout.
			l.idleTimer <- timer
			rwc.Close()
			<-l.timedOut
			return nil, ErrIdleTimeout
		}

		l.active <- 1
		return l.newConn(rwc), nil
	}
}

func (l *idleListener) Close() error {
	select {
	case _, ok := <-l.active:
		if ok {
			close(l.active)
		}

	case <-l.timedOut:
		// Already closed by the timer; take care not to double-close if the caller
		// only explicitly invokes this Close method once, since the io.Closer
		// interface explicitly leaves doubled Close calls undefined.
		return ErrIdleTimeout

	case timer := <-l.idleTimer:
		if !timer.Stop() {
			// Couldn't stop the timer. It shouldn't take long to run, so just wait
			// (so that the Listener is guaranteed to be closed before we return)
			// and pretend that this call happened afterward.
			// That way we won't leak any timers or goroutines when Close returns.
			l.idleTimer <- timer
			<-l.timedOut
			return ErrIdleTimeout
		}
		close(l.active)
	}

	return l.wrapped.Close()
}

func (l *idleListener) Dialer() Dialer {
	return l.wrapped.Dialer()
}

func (l *idleListener) timerExpired() {
	select {
	case n, ok := <-l.active:
		if ok {
			panic(fmt.Sprintf("jsonrpc2: idleListener idle timer fired with %d connections still active", n))
		} else {
			panic("jsonrpc2: Close finished with idle timer still running")
		}

	case <-l.timedOut:
		panic("jsonrpc2: idleListener idle timer fired more than once")

	case <-l.idleTimer:
		// The timer for this very call!
	}

	// Close the Listener with all channels still blocked to ensure that this call
	// to l.wrapped.Close doesn't race with the one in l.Close.
	defer close(l.timedOut)
	l.wrapped.Close()
}

func (l *idleListener) connClosed() {
	select {
	case n, ok := <-l.active:
		if !ok {
			// l is already closed, so it can't close due to idleness,
			// and we don't need to track the number of active connections any more.
			return
		}
		n--
		if n == 0 {
			l.idleTimer <- time.AfterFunc(l.timeout, l.timerExpired)
		} else {
			l.active <- n
		}

	case <-l.timedOut:
		panic("jsonrpc2: idleListener idle timer fired before last active connection was closed")

	case <-l.idleTimer:
		panic("jsonrpc2: idleListener idle timer active before last active connection was closed")
	}
}

type idleListenerConn struct {
	wrapped   io.ReadWriteCloser
	l         *idleListener
	closeOnce sync.Once
}

func (l *idleListener) newConn(rwc io.ReadWriteCloser) *idleListenerConn {
	c := &idleListenerConn{
		wrapped: rwc,
		l:       l,
	}

	// A caller that forgets to call Close may disrupt the idleListener's
	// accounting, even though the file descriptor for the underlying connection
	// may eventually be garbage-collected anyway.
	//
	// Set a (best-effort) finalizer to verify that a Close call always occurs.
	// (We will clear the finalizer explicitly in Close.)
	runtime.SetFinalizer(c, func(c *idleListenerConn) {
		panic("jsonrpc2: IdleListener connection became unreachable without a call to Close")
	})

	return c
}

func (c *idleListenerConn) Read(p []byte) (int, error)  { return c.wrapped.Read(p) }
func (c *idleListenerConn) Write(p []byte) (int, error) { return c.wrapped.Write(p) }

func (c *idleListenerConn) Close() error {
	defer c.closeOnce.Do(func() {
		c.l.connClosed()
		runtime.SetFinalizer(c, nil)
	})
	return c.wrapped.Close()
}
</content>
</file>
<file path="internal/jsonrpc2/serve_test.go">
<type>go</type>
<content>
// Copyright 2020 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package jsonrpc2_test

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"runtime/debug"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/internal/jsonrpc2"
)

// needsLocalhostNet skips t if networking does not work for ports opened
// with "localhost".
// forked from golang.org/x/tools/internal/testenv.
func needsLocalhostNet(t testing.TB) {
	switch runtime.GOOS {
	case "js", "wasip1":
		t.Skipf(`Listening on "localhost" fails on %s; see https://go.dev/issue/59718`, runtime.GOOS)
	}
}

func TestIdleTimeout(t *testing.T) {
	needsLocalhostNet(t)

	// Use a panicking time.AfterFunc instead of context.WithTimeout so that we
	// get a goroutine dump on failure. We expect the test to take on the order of
	// a few tens of milliseconds at most, so 10s should be several orders of
	// magnitude of headroom.
	timer := time.AfterFunc(10*time.Second, func() {
		debug.SetTraceback("all")
		panic("TestIdleTimeout deadlocked")
	})
	defer timer.Stop()

	ctx := context.Background()

	try := func(d time.Duration) (longEnough bool) {
		listener, err := jsonrpc2.NetListener(ctx, "tcp", "localhost:0", jsonrpc2.NetListenOptions{})
		if err != nil {
			t.Fatal(err)
		}

		idleStart := time.Now()
		listener = jsonrpc2.NewIdleListener(d, listener)
		defer listener.Close()

		server := jsonrpc2.NewServer(ctx, listener, jsonrpc2.ConnectionOptions{})

		// Exercise some connection/disconnection patterns, and then assert that when
		// our timer fires, the server exits.
		conn1, err := jsonrpc2.Dial(ctx, listener.Dialer(), jsonrpc2.ConnectionOptions{}, nil)
		if err != nil {
			if since := time.Since(idleStart); since < d {
				t.Fatalf("conn1 failed to connect after %v: %v", since, err)
			}
			t.Log("jsonrpc2.Dial:", err)
			return false // Took to long to dial, so the failure could have been due to the idle timeout.
		}
		// On the server side, Accept can race with the connection timing out.
		// Send a call and wait for the response to ensure that the connection was
		// actually fully accepted.
		ac := conn1.Call(ctx, "ping", nil)
		if err := ac.Await(ctx, nil); !errors.Is(err, jsonrpc2.ErrMethodNotFound) {
			if since := time.Since(idleStart); since < d {
				t.Fatalf("conn1 broken after %v: %v", since, err)
			}
			t.Log(`conn1.Call(ctx, "ping", nil):`, err)
			conn1.Close()
			return false
		}

		// Since conn1 was successfully accepted and remains open, the server is
		// definitely non-idle. Dialing another simultaneous connection should
		// succeed.
		conn2, err := jsonrpc2.Dial(ctx, listener.Dialer(), jsonrpc2.ConnectionOptions{}, nil)
		if err != nil {
			conn1.Close()
			t.Fatalf("conn2 failed to connect while non-idle after %v: %v", time.Since(idleStart), err)
			return false
		}
		// Ensure that conn2 is also accepted on the server side before we close
		// conn1. Otherwise, the connection can appear idle if the server processes
		// the closure of conn1 and the idle timeout before it finally notices conn2
		// in the accept queue.
		// (That failure mode may explain the failure noted in
		// https://go.dev/issue/49387#issuecomment-1303979877.)
		ac = conn2.Call(ctx, "ping", nil)
		if err := ac.Await(ctx, nil); !errors.Is(err, jsonrpc2.ErrMethodNotFound) {
			t.Fatalf("conn2 broken while non-idle after %v: %v", time.Since(idleStart), err)
		}

		if err := conn1.Close(); err != nil {
			t.Fatalf("conn1.Close failed with error: %v", err)
		}
		idleStart = time.Now()
		if err := conn2.Close(); err != nil {
			t.Fatalf("conn2.Close failed with error: %v", err)
		}

		conn3, err := jsonrpc2.Dial(ctx, listener.Dialer(), jsonrpc2.ConnectionOptions{}, nil)
		if err != nil {
			if since := time.Since(idleStart); since < d {
				t.Fatalf("conn3 failed to connect after %v: %v", since, err)
			}
			t.Log("jsonrpc2.Dial:", err)
			return false // Took to long to dial, so the failure could have been due to the idle timeout.
		}

		ac = conn3.Call(ctx, "ping", nil)
		if err := ac.Await(ctx, nil); !errors.Is(err, jsonrpc2.ErrMethodNotFound) {
			if since := time.Since(idleStart); since < d {
				t.Fatalf("conn3 broken after %v: %v", since, err)
			}
			t.Log(`conn3.Call(ctx, "ping", nil):`, err)
			conn3.Close()
			return false
		}

		idleStart = time.Now()
		if err := conn3.Close(); err != nil {
			t.Fatalf("conn3.Close failed with error: %v", err)
		}

		serverError := server.Wait()

		if !errors.Is(serverError, jsonrpc2.ErrIdleTimeout) {
			t.Errorf("run() returned error %v, want %v", serverError, jsonrpc2.ErrIdleTimeout)
		}
		if since := time.Since(idleStart); since < d {
			t.Errorf("server shut down after %v idle; want at least %v", since, d)
		}
		return true
	}

	d := 1 * time.Millisecond
	for {
		t.Logf("testing with idle timeout %v", d)
		if !try(d) {
			d *= 2
			continue
		}
		break
	}
}

type msg struct {
	Msg string
}

type fakeHandler struct{}

func (fakeHandler) Handle(ctx context.Context, req *jsonrpc2.Request) (any, error) {
	switch req.Method {
	case "ping":
		return &msg{"pong"}, nil
	default:
		return nil, jsonrpc2.ErrNotHandled
	}
}

func TestServe(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		factory func(context.Context, testing.TB) (jsonrpc2.Listener, error)
	}{
		{"tcp", func(ctx context.Context, t testing.TB) (jsonrpc2.Listener, error) {
			needsLocalhostNet(t)
			return jsonrpc2.NetListener(ctx, "tcp", "localhost:0", jsonrpc2.NetListenOptions{})
		}},
		{"pipe", func(ctx context.Context, t testing.TB) (jsonrpc2.Listener, error) {
			return jsonrpc2.NetPipeListener(ctx)
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fake, err := test.factory(ctx, t)
			if err != nil {
				t.Fatal(err)
			}
			conn, shutdown, err := newFake(t, ctx, fake)
			if err != nil {
				t.Fatal(err)
			}
			defer shutdown()
			var got msg
			if err := conn.Call(ctx, "ping", &msg{"ting"}).Await(ctx, &got); err != nil {
				t.Fatal(err)
			}
			if want := "pong"; got.Msg != want {
				t.Errorf("conn.Call(...): returned %q, want %q", got, want)
			}
		})
	}
}

func newFake(t *testing.T, ctx context.Context, l jsonrpc2.Listener) (*jsonrpc2.Connection, func(), error) {
	server := jsonrpc2.NewServer(ctx, l, jsonrpc2.ConnectionOptions{
		Handler: fakeHandler{},
	})

	client, err := jsonrpc2.Dial(ctx,
		l.Dialer(),
		jsonrpc2.ConnectionOptions{
			Handler: fakeHandler{},
		}, nil)
	if err != nil {
		return nil, nil, err
	}
	return client, func() {
		if err := l.Close(); err != nil {
			t.Fatal(err)
		}
		if err := client.Close(); err != nil {
			t.Fatal(err)
		}
		server.Wait()
	}, nil
}

// TestIdleListenerAcceptCloseRace checks for the Accept/Close race fixed in CL 388597.
//
// (A bug in the idleListener implementation caused a successful Accept to block
// on sending to a background goroutine that could have already exited.)
func TestIdleListenerAcceptCloseRace(t *testing.T) {
	ctx := context.Background()

	n := 10

	// Each iteration of the loop appears to take around a millisecond, so to
	// avoid spurious failures we'll set the watchdog for three orders of
	// magnitude longer. When the bug was present, this reproduced the deadlock
	// reliably on a Linux workstation when run with -count=100, which should be
	// frequent enough to show up on the Go build dashboard if it regresses.
	watchdog := time.Duration(n) * 1000 * time.Millisecond
	timer := time.AfterFunc(watchdog, func() {
		debug.SetTraceback("all")
		panic(fmt.Sprintf("%s deadlocked after %v", t.Name(), watchdog))
	})
	defer timer.Stop()

	for ; n > 0; n-- {
		listener, err := jsonrpc2.NetPipeListener(ctx)
		if err != nil {
			t.Fatal(err)
		}
		listener = jsonrpc2.NewIdleListener(24*time.Hour, listener)

		done := make(chan struct{})
		go func() {
			conn, err := jsonrpc2.Dial(ctx, listener.Dialer(), jsonrpc2.ConnectionOptions{}, nil)
			listener.Close()
			if err == nil {
				conn.Close()
			}
			close(done)
		}()

		// Accept may return a non-nil error if Close closes the underlying network
		// connection before the wrapped Accept call unblocks. However, it must not
		// deadlock!
		c, err := listener.Accept(ctx)
		if err == nil {
			c.Close()
		}
		<-done
	}
}

// TestCloseCallRace checks for a race resulting in a deadlock when a Call on
// one side of the connection races with a Close (or otherwise broken
// connection) initiated from the other side.
//
// (The Call method was waiting for a result from the Read goroutine to
// determine which error value to return, but the Read goroutine was waiting for
// in-flight calls to complete before reporting that result.)
func TestCloseCallRace(t *testing.T) {
	ctx := context.Background()
	n := 10

	watchdog := time.Duration(n) * 1000 * time.Millisecond
	timer := time.AfterFunc(watchdog, func() {
		debug.SetTraceback("all")
		panic(fmt.Sprintf("%s deadlocked after %v", t.Name(), watchdog))
	})
	defer timer.Stop()

	for ; n > 0; n-- {
		listener, err := jsonrpc2.NetPipeListener(ctx)
		if err != nil {
			t.Fatal(err)
		}

		pokec := make(chan *jsonrpc2.AsyncCall, 1)

		s := jsonrpc2.NewServer(ctx, listener, jsonrpc2.BinderFunc(func(_ context.Context, srvConn *jsonrpc2.Connection) jsonrpc2.ConnectionOptions {
			h := jsonrpc2.HandlerFunc(func(ctx context.Context, _ *jsonrpc2.Request) (any, error) {
				// Start a concurrent call from the server to the client.
				// The point of this test is to ensure this doesn't deadlock
				// if the client shuts down the connection concurrently.
				//
				// The racing Call may or may not receive a response: it should get a
				// response if it is sent before the client closes the connection, and
				// it should fail with some kind of "connection closed" error otherwise.
				go func() {
					pokec <- srvConn.Call(ctx, "poke", nil)
				}()

				return &msg{"pong"}, nil
			})
			return jsonrpc2.ConnectionOptions{Handler: h}
		}))

		dialConn, err := jsonrpc2.Dial(ctx, listener.Dialer(), jsonrpc2.ConnectionOptions{}, nil)
		if err != nil {
			listener.Close()
			s.Wait()
			t.Fatal(err)
		}

		// Calling any method on the server should provoke it to asynchronously call
		// us back. While it is starting that call, we will close the connection.
		if err := dialConn.Call(ctx, "ping", nil).Await(ctx, nil); err != nil {
			t.Error(err)
		}
		if err := dialConn.Close(); err != nil {
			t.Error(err)
		}

		// Ensure that the Call on the server side did not block forever when the
		// connection closed.
		pokeCall := <-pokec
		if err := pokeCall.Await(ctx, nil); err == nil {
			t.Errorf("unexpected nil error from server-initited call")
		} else if errors.Is(err, jsonrpc2.ErrMethodNotFound) {
			// The call completed before the Close reached the handler.
		} else {
			// The error was something else.
			t.Logf("server-initiated call completed with expected error: %v", err)
		}

		listener.Close()
		s.Wait()
	}
}
</content>
</file>
<file path="internal/jsonrpc2/wire.go">
<type>go</type>
<content>
// Copyright 2018 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package jsonrpc2

import (
	"encoding/json"
)

// This file contains the go forms of the wire specification.
// see http://www.jsonrpc.org/specification for details

var (
	// ErrParse is used when invalid JSON was received by the server.
	ErrParse = NewError(-32700, "parse error")
	// ErrInvalidRequest is used when the JSON sent is not a valid Request object.
	ErrInvalidRequest = NewError(-32600, "invalid request")
	// ErrMethodNotFound should be returned by the handler when the method does
	// not exist / is not available.
	ErrMethodNotFound = NewError(-32601, "method not found")
	// ErrInvalidParams should be returned by the handler when method
	// parameter(s) were invalid.
	ErrInvalidParams = NewError(-32602, "invalid params")
	// ErrInternal indicates a failure to process a call correctly
	ErrInternal = NewError(-32603, "internal error")

	// The following errors are not part of the json specification, but
	// compliant extensions specific to this implementation.

	// ErrServerOverloaded is returned when a message was refused due to a
	// server being temporarily unable to accept any new messages.
	ErrServerOverloaded = NewError(-32000, "overloaded")
	// ErrUnknown should be used for all non coded errors.
	ErrUnknown = NewError(-32001, "unknown error")
	// ErrServerClosing is returned for calls that arrive while the server is closing.
	ErrServerClosing = NewError(-32004, "server is closing")
	// ErrClientClosing is a dummy error returned for calls initiated while the client is closing.
	ErrClientClosing = NewError(-32003, "client is closing")

	// The following errors have special semantics for MCP transports

	// ErrRejected may be wrapped to return errors from calls to Writer.Write
	// that signal that the request was rejected by the transport layer as
	// invalid.
	//
	// Such failures do not indicate that the connection is broken, but rather
	// should be returned to the caller to indicate that the specific request is
	// invalid in the current context.
	ErrRejected = NewError(-32004, "rejected by transport")
)

const wireVersion = "2.0"

// wireCombined has all the fields of both Request and Response.
// We can decode this and then work out which it is.
type wireCombined struct {
	VersionTag string          `json:"jsonrpc"`
	ID         any             `json:"id,omitempty"`
	Method     string          `json:"method,omitempty"`
	Params     json.RawMessage `json:"params,omitempty"`
	Result     json.RawMessage `json:"result,omitempty"`
	Error      *WireError      `json:"error,omitempty"`
}

// WireError represents a structured error in a Response.
type WireError struct {
	// Code is an error code indicating the type of failure.
	Code int64 `json:"code"`
	// Message is a short description of the error.
	Message string `json:"message"`
	// Data is optional structured data containing additional information about the error.
	Data json.RawMessage `json:"data,omitempty"`
}

// NewError returns an error that will encode on the wire correctly.
// The standard codes are made available from this package, this function should
// only be used to build errors for application specific codes as allowed by the
// specification.
func NewError(code int64, message string) error {
	return &WireError{
		Code:    code,
		Message: message,
	}
}

func (err *WireError) Error() string {
	return err.Message
}

func (err *WireError) Is(other error) bool {
	w, ok := other.(*WireError)
	if !ok {
		return false
	}
	return err.Code == w.Code
}
</content>
</file>
<file path="internal/jsonrpc2/wire_test.go">
<type>go</type>
<content>
// Copyright 2020 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package jsonrpc2_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/internal/jsonrpc2"
)

func TestWireMessage(t *testing.T) {
	for _, test := range []struct {
		name    string
		msg     jsonrpc2.Message
		encoded []byte
	}{{
		name:    "notification",
		msg:     newNotification("alive", nil),
		encoded: []byte(`{"jsonrpc":"2.0","method":"alive"}`),
	}, {
		name:    "call",
		msg:     newCall("msg1", "ping", nil),
		encoded: []byte(`{"jsonrpc":"2.0","id":"msg1","method":"ping"}`),
	}, {
		name:    "response",
		msg:     newResponse("msg2", "pong", nil),
		encoded: []byte(`{"jsonrpc":"2.0","id":"msg2","result":"pong"}`),
	}, {
		name:    "numerical id",
		msg:     newCall(1, "poke", nil),
		encoded: []byte(`{"jsonrpc":"2.0","id":1,"method":"poke"}`),
	}, {
		// originally reported in #39719, this checks that result is not present if
		// it is an error response
		name: "computing fix edits",
		msg:  newResponse(3, nil, jsonrpc2.NewError(0, "computing fix edits")),
		encoded: []byte(`{
		"jsonrpc":"2.0",
		"id":3,
		"error":{
			"code":0,
			"message":"computing fix edits"
		}
	}`),
	}} {
		b, err := jsonrpc2.EncodeMessage(test.msg)
		if err != nil {
			t.Fatal(err)
		}
		checkJSON(t, b, test.encoded)
		msg, err := jsonrpc2.DecodeMessage(test.encoded)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(msg, test.msg) {
			t.Errorf("decoded message does not match\nGot:\n%+#v\nWant:\n%+#v", msg, test.msg)
		}
	}
}

func newNotification(method string, params any) jsonrpc2.Message {
	msg, err := jsonrpc2.NewNotification(method, params)
	if err != nil {
		panic(err)
	}
	return msg
}

func newID(id any) jsonrpc2.ID {
	switch v := id.(type) {
	case nil:
		return jsonrpc2.ID{}
	case string:
		return jsonrpc2.StringID(v)
	case int:
		return jsonrpc2.Int64ID(int64(v))
	case int64:
		return jsonrpc2.Int64ID(v)
	default:
		panic("invalid ID type")
	}
}

func newCall(id any, method string, params any) jsonrpc2.Message {
	msg, err := jsonrpc2.NewCall(newID(id), method, params)
	if err != nil {
		panic(err)
	}
	return msg
}

func newResponse(id any, result any, rerr error) jsonrpc2.Message {
	msg, err := jsonrpc2.NewResponse(newID(id), result, rerr)
	if err != nil {
		panic(err)
	}
	return msg
}

func checkJSON(t *testing.T, got, want []byte) {
	// compare the compact form, to allow for formatting differences
	g := &bytes.Buffer{}
	if err := json.Compact(g, []byte(got)); err != nil {
		t.Fatal(err)
	}
	w := &bytes.Buffer{}
	if err := json.Compact(w, []byte(want)); err != nil {
		t.Fatal(err)
	}
	if g.String() != w.String() {
		t.Errorf("encoded message does not match\nGot:\n%s\nWant:\n%s", g, w)
	}
}
</content>
</file>
<file path="internal/oauthex/auth_meta.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// This file implements Authorization Server Metadata.
// See https://www.rfc-editor.org/rfc/rfc8414.html.

package oauthex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AuthServerMeta represents the metadata for an OAuth 2.0 authorization server,
// as defined in [RFC 8414].
//
// Not supported:
// - signed metadata
//
// [RFC 8414]: https://tools.ietf.org/html/rfc8414)
type AuthServerMeta struct {
	// GENERATED BY GEMINI 2.5.

	// Issuer is the REQUIRED URL identifying the authorization server.
	Issuer string `json:"issuer"`

	// AuthorizationEndpoint is the REQUIRED URL of the server's OAuth 2.0 authorization endpoint.
	AuthorizationEndpoint string `json:"authorization_endpoint"`

	// TokenEndpoint is the REQUIRED URL of the server's OAuth 2.0 token endpoint.
	TokenEndpoint string `json:"token_endpoint"`

	// JWKSURI is the REQUIRED URL of the server's JSON Web Key Set [JWK] document.
	JWKSURI string `json:"jwks_uri"`

	// RegistrationEndpoint is the RECOMMENDED URL of the server's OAuth 2.0 Dynamic Client Registration endpoint.
	RegistrationEndpoint string `json:"registration_endpoint,omitempty"`

	// ScopesSupported is a RECOMMENDED JSON array of strings containing a list of the OAuth 2.0
	// "scope" values that this server supports.
	ScopesSupported []string `json:"scopes_supported,omitempty"`

	// ResponseTypesSupported is a REQUIRED JSON array of strings containing a list of the OAuth 2.0
	// "response_type" values that this server supports.
	ResponseTypesSupported []string `json:"response_types_supported"`

	// ResponseModesSupported is a RECOMMENDED JSON array of strings containing a list of the OAuth 2.0
	// "response_mode" values that this server supports.
	ResponseModesSupported []string `json:"response_modes_supported,omitempty"`

	// GrantTypesSupported is a RECOMMENDED JSON array of strings containing a list of the OAuth 2.0
	// grant type values that this server supports.
	GrantTypesSupported []string `json:"grant_types_supported,omitempty"`

	// TokenEndpointAuthMethodsSupported is a RECOMMENDED JSON array of strings containing a list of
	// client authentication methods supported by this token endpoint.
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported,omitempty"`

	// TokenEndpointAuthSigningAlgValuesSupported is a RECOMMENDED JSON array of strings containing
	// a list of the JWS signing algorithms ("alg" values) supported by the token endpoint for
	// the signature on the JWT used to authenticate the client.
	TokenEndpointAuthSigningAlgValuesSupported []string `json:"token_endpoint_auth_signing_alg_values_supported,omitempty"`

	// ServiceDocumentation is a RECOMMENDED URL of a page containing human-readable documentation
	// for the service.
	ServiceDocumentation string `json:"service_documentation,omitempty"`

	// UILocalesSupported is a RECOMMENDED JSON array of strings representing supported
	// BCP47 [RFC5646] language tag values for display in the user interface.
	UILocalesSupported []string `json:"ui_locales_supported,omitempty"`

	// OpPolicyURI is a RECOMMENDED URL that the server provides to the person registering
	// the client to read about the server's operator policies.
	OpPolicyURI string `json:"op_policy_uri,omitempty"`

	// OpTOSURI is a RECOMMENDED URL that the server provides to the person registering the
	// client to read about the server's terms of service.
	OpTOSURI string `json:"op_tos_uri,omitempty"`

	// RevocationEndpoint is a RECOMMENDED URL of the server's OAuth 2.0 revocation endpoint.
	RevocationEndpoint string `json:"revocation_endpoint,omitempty"`

	// RevocationEndpointAuthMethodsSupported is a RECOMMENDED JSON array of strings containing
	// a list of client authentication methods supported by this revocation endpoint.
	RevocationEndpointAuthMethodsSupported []string `json:"revocation_endpoint_auth_methods_supported,omitempty"`

	// RevocationEndpointAuthSigningAlgValuesSupported is a RECOMMENDED JSON array of strings
	// containing a list of the JWS signing algorithms ("alg" values) supported by the revocation
	// endpoint for the signature on the JWT used to authenticate the client.
	RevocationEndpointAuthSigningAlgValuesSupported []string `json:"revocation_endpoint_auth_signing_alg_values_supported,omitempty"`

	// IntrospectionEndpoint is a RECOMMENDED URL of the server's OAuth 2.0 introspection endpoint.
	IntrospectionEndpoint string `json:"introspection_endpoint,omitempty"`

	// IntrospectionEndpointAuthMethodsSupported is a RECOMMENDED JSON array of strings containing
	// a list of client authentication methods supported by this introspection endpoint.
	IntrospectionEndpointAuthMethodsSupported []string `json:"introspection_endpoint_auth_methods_supported,omitempty"`

	// IntrospectionEndpointAuthSigningAlgValuesSupported is a RECOMMENDED JSON array of strings
	// containing a list of the JWS signing algorithms ("alg" values) supported by the introspection
	// endpoint for the signature on the JWT used to authenticate the client.
	IntrospectionEndpointAuthSigningAlgValuesSupported []string `json:"introspection_endpoint_auth_signing_alg_values_supported,omitempty"`

	// CodeChallengeMethodsSupported is a RECOMMENDED JSON array of strings containing a list of
	// PKCE code challenge methods supported by this authorization server.
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported,omitempty"`
}

// ClientRegistrationMetadata represents the client metadata fields for the DCR POST request (RFC 7591).
type ClientRegistrationMetadata struct {
	// RedirectURIs is a REQUIRED JSON array of redirection URI strings for use in
	// redirect-based flows (such as the authorization code grant).
	RedirectURIs []string `json:"redirect_uris"`

	// TokenEndpointAuthMethod is an OPTIONAL string indicator of the requested
	// authentication method for the token endpoint.
	// If omitted, the default is "client_secret_basic".
	TokenEndpointAuthMethod string `json:"token_endpoint_auth_method,omitempty"`

	// GrantTypes is an OPTIONAL JSON array of OAuth 2.0 grant type strings
	// that the client will restrict itself to using.
	// If omitted, the default is ["authorization_code"].
	GrantTypes []string `json:"grant_types,omitempty"`

	// ResponseTypes is an OPTIONAL JSON array of OAuth 2.0 response type strings
	// that the client will restrict itself to using.
	// If omitted, the default is ["code"].
	ResponseTypes []string `json:"response_types,omitempty"`

	// ClientName is a RECOMMENDED human-readable name of the client to be presented
	// to the end-user.
	ClientName string `json:"client_name,omitempty"`

	// ClientURI is a RECOMMENDED URL of a web page providing information about the client.
	ClientURI string `json:"client_uri,omitempty"`

	// LogoURI is an OPTIONAL URL of a logo for the client, which may be displayed
	// to the end-user.
	LogoURI string `json:"logo_uri,omitempty"`

	// Scope is an OPTIONAL string containing a space-separated list of scope values
	// that the client will restrict itself to using.
	Scope string `json:"scope,omitempty"`

	// Contacts is an OPTIONAL JSON array of strings representing ways to contact
	// people responsible for this client (e.g., email addresses).
	Contacts []string `json:"contacts,omitempty"`

	// TOSURI is an OPTIONAL URL that the client provides to the end-user
	// to read about the client's terms of service.
	TOSURI string `json:"tos_uri,omitempty"`

	// PolicyURI is an OPTIONAL URL that the client provides to the end-user
	// to read about the client's privacy policy.
	PolicyURI string `json:"policy_uri,omitempty"`

	// JWKSURI is an OPTIONAL URL for the client's JSON Web Key Set [JWK] document.
	// This is preferred over the 'jwks' parameter.
	JWKSURI string `json:"jwks_uri,omitempty"`

	// JWKS is an OPTIONAL client's JSON Web Key Set [JWK] document, passed by value.
	// This is an alternative to providing a JWKSURI.
	JWKS string `json:"jwks,omitempty"`

	// SoftwareID is an OPTIONAL unique identifier string for the client software,
	// constant across all instances and versions.
	SoftwareID string `json:"software_id,omitempty"`

	// SoftwareVersion is an OPTIONAL version identifier string for the client software.
	SoftwareVersion string `json:"software_version,omitempty"`

	// SoftwareStatement is an OPTIONAL JWT that asserts client metadata values.
	// Values in the software statement take precedence over other metadata values.
	SoftwareStatement string `json:"software_statement,omitempty"`
}

// ClientRegistrationResponse represents the fields returned by the Authorization Server
// (RFC 7591, Section 3.2.1 and 3.2.2).
type ClientRegistrationResponse struct {
	// ClientRegistrationMetadata contains all registered client metadata, returned by the
	// server on success, potentially with modified or defaulted values.
	ClientRegistrationMetadata

	// ClientID is the REQUIRED newly issued OAuth 2.0 client identifier.
	ClientID string `json:"client_id"`

	// ClientSecret is an OPTIONAL client secret string.
	ClientSecret string `json:"client_secret,omitempty"`

	// ClientIDIssuedAt is an OPTIONAL Unix timestamp when the ClientID was issued.
	ClientIDIssuedAt time.Time `json:"client_id_issued_at,omitempty"`

	// ClientSecretExpiresAt is the REQUIRED (if client_secret is issued) Unix
	// timestamp when the secret expires, or 0 if it never expires.
	ClientSecretExpiresAt time.Time `json:"client_secret_expires_at,omitempty"`
}

func (r *ClientRegistrationResponse) MarshalJSON() ([]byte, error) {
	type alias ClientRegistrationResponse
	var clientIDIssuedAt int64
	var clientSecretExpiresAt int64

	if !r.ClientIDIssuedAt.IsZero() {
		clientIDIssuedAt = r.ClientIDIssuedAt.Unix()
	}
	if !r.ClientSecretExpiresAt.IsZero() {
		clientSecretExpiresAt = r.ClientSecretExpiresAt.Unix()
	}

	return json.Marshal(&struct {
		ClientIDIssuedAt      int64 `json:"client_id_issued_at,omitempty"`
		ClientSecretExpiresAt int64 `json:"client_secret_expires_at,omitempty"`
		*alias
	}{
		ClientIDIssuedAt:      clientIDIssuedAt,
		ClientSecretExpiresAt: clientSecretExpiresAt,
		alias:                 (*alias)(r),
	})
}

func (r *ClientRegistrationResponse) UnmarshalJSON(data []byte) error {
	type alias ClientRegistrationResponse
	aux := &struct {
		ClientIDIssuedAt      int64 `json:"client_id_issued_at,omitempty"`
		ClientSecretExpiresAt int64 `json:"client_secret_expires_at,omitempty"`
		*alias
	}{
		alias: (*alias)(r),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.ClientIDIssuedAt != 0 {
		r.ClientIDIssuedAt = time.Unix(aux.ClientIDIssuedAt, 0)
	}
	if aux.ClientSecretExpiresAt != 0 {
		r.ClientSecretExpiresAt = time.Unix(aux.ClientSecretExpiresAt, 0)
	}
	return nil
}

// ClientRegistrationError is the error response from the Authorization Server
// for a failed registration attempt (RFC 7591, Section 3.2.2).
type ClientRegistrationError struct {
	// ErrorCode is the REQUIRED error code if registration failed (RFC 7591, 3.2.2).
	ErrorCode string `json:"error"`

	// ErrorDescription is an OPTIONAL human-readable error message.
	ErrorDescription string `json:"error_description,omitempty"`
}

func (e *ClientRegistrationError) Error() string {
	return fmt.Sprintf("registration failed: %s (%s)", e.ErrorCode, e.ErrorDescription)
}

var wellKnownPaths = []string{
	"/.well-known/oauth-authorization-server",
	"/.well-known/openid-configuration",
}

// GetAuthServerMeta issues a GET request to retrieve authorization server metadata
// from an OAuth authorization server with the given issuerURL.
//
// It follows [RFC 8414]:
//   - The well-known paths specified there are inserted into the URL's path, one at time.
//     The first to succeed is used.
//   - The Issuer field is checked against issuerURL.
//
// [RFC 8414]: https://tools.ietf.org/html/rfc8414
func GetAuthServerMeta(ctx context.Context, issuerURL string, c *http.Client) (*AuthServerMeta, error) {
	var errs []error
	for _, p := range wellKnownPaths {
		u, err := prependToPath(issuerURL, p)
		if err != nil {
			// issuerURL is bad; no point in continuing.
			return nil, err
		}
		asm, err := getJSON[AuthServerMeta](ctx, c, u, 1<<20)
		if err == nil {
			if asm.Issuer != issuerURL { // section 3.3
				// Security violation; don't keep trying.
				return nil, fmt.Errorf("metadata issuer %q does not match issuer URL %q", asm.Issuer, issuerURL)
			}
			return asm, nil
		}
		errs = append(errs, err)
	}
	return nil, fmt.Errorf("failed to get auth server metadata from %q: %w", issuerURL, errors.Join(errs...))
}

// RegisterClient performs Dynamic Client Registration according to RFC 7591.
func RegisterClient(ctx context.Context, registrationEndpoint string, clientMeta *ClientRegistrationMetadata, c *http.Client) (*ClientRegistrationResponse, error) {
	if registrationEndpoint == "" {
		return nil, fmt.Errorf("registration_endpoint is required")
	}

	if c == nil {
		c = http.DefaultClient
	}

	payload, err := json.Marshal(clientMeta)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal client metadata: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", registrationEndpoint, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create registration request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("registration request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read registration response body: %w", err)
	}

	if resp.StatusCode == http.StatusCreated {
		var regResponse ClientRegistrationResponse
		if err := json.Unmarshal(body, &regResponse); err != nil {
			return nil, fmt.Errorf("failed to decode successful registration response: %w (%s)", err, string(body))
		}
		if regResponse.ClientID == "" {
			return nil, fmt.Errorf("registration response is missing required 'client_id' field")
		}
		return &regResponse, nil
	}

	if resp.StatusCode == http.StatusBadRequest {
		var regError ClientRegistrationError
		if err := json.Unmarshal(body, &regError); err != nil {
			return nil, fmt.Errorf("failed to decode registration error response: %w (%s)", err, string(body))
		}
		return nil, &regError
	}

	return nil, fmt.Errorf("registration failed with status %s: %s", resp.Status, string(body))
}
</content>
</file>
<file path="internal/oauthex/auth_meta_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package oauthex

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestAuthMetaParse(t *testing.T) {
	// Verify that we parse Google's auth server metadata.
	data, err := os.ReadFile(filepath.FromSlash("testdata/google-auth-meta.json"))
	if err != nil {
		t.Fatal(err)
	}
	var a AuthServerMeta
	if err := json.Unmarshal(data, &a); err != nil {
		t.Fatal(err)
	}
	// Spot check.
	if g, w := a.Issuer, "https://accounts.google.com"; g != w {
		t.Errorf("got %q, want %q", g, w)
	}
}

func TestClientRegistrationMetadataParse(t *testing.T) {
	// Verify that we can parse a typical client metadata JSON.
	data, err := os.ReadFile(filepath.FromSlash("testdata/client-auth-meta.json"))
	if err != nil {
		t.Fatal(err)
	}
	var a ClientRegistrationMetadata
	if err := json.Unmarshal(data, &a); err != nil {
		t.Fatal(err)
	}
	// Spot check
	if g, w := a.ClientName, "My Test App"; g != w {
		t.Errorf("got ClientName %q, want %q", g, w)
	}
	if g, w := len(a.RedirectURIs), 2; g != w {
		t.Errorf("got %d RedirectURIs, want %d", g, w)
	}
}

func TestRegisterClient(t *testing.T) {
	testCases := []struct {
		name         string
		handler      http.HandlerFunc
		clientMeta   *ClientRegistrationMetadata
		wantClientID string
		wantErr      string
	}{
		{
			name: "Success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatal(err)
				}
				var receivedMeta ClientRegistrationMetadata
				if err := json.Unmarshal(body, &receivedMeta); err != nil {
					t.Fatalf("Failed to unmarshal request body: %v", err)
				}
				if receivedMeta.ClientName != "Test App" {
					t.Errorf("Expected ClientName 'Test App', got '%s'", receivedMeta.ClientName)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"client_id":"test-client-id","client_secret":"test-client-secret","client_name":"Test App"}`))
			},
			clientMeta:   &ClientRegistrationMetadata{ClientName: "Test App", RedirectURIs: []string{"http://localhost/cb"}},
			wantClientID: "test-client-id",
		},
		{
			name: "Missing ClientID in Response",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"client_secret":"test-client-secret"}`)) // No client_id
			},
			clientMeta: &ClientRegistrationMetadata{RedirectURIs: []string{"http://localhost/cb"}},
			wantErr:    "registration response is missing required 'client_id' field",
		},
		{
			name: "Standard OAuth Error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"invalid_redirect_uri","error_description":"Redirect URI is not valid."}`))
			},
			clientMeta: &ClientRegistrationMetadata{RedirectURIs: []string{"http://invalid/cb"}},
			wantErr:    "registration failed: invalid_redirect_uri (Redirect URI is not valid.)",
		},
		{
			name: "Non-JSON Server Error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Internal Server Error"))
			},
			clientMeta: &ClientRegistrationMetadata{RedirectURIs: []string{"http://localhost/cb"}},
			wantErr:    "registration failed with status 500 Internal Server Error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			info, err := RegisterClient(context.Background(), server.URL, tc.clientMeta, server.Client())

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("Expected an error containing '%s', but got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("Expected error to contain '%s', got '%v'", tc.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Expected no error, but got: %v", err)
			}
			if info.ClientID != tc.wantClientID {
				t.Errorf("Expected client_id '%s', got '%s'", tc.wantClientID, info.ClientID)
			}
		})
	}

	t.Run("No Endpoint", func(t *testing.T) {
		_, err := RegisterClient(context.Background(), "", &ClientRegistrationMetadata{}, nil)
		if err == nil {
			t.Fatal("Expected an error for missing registration endpoint, got nil")
		}
		expectedErr := "registration_endpoint is required"
		if err.Error() != expectedErr {
			t.Errorf("Expected error '%s', got '%v'", expectedErr, err)
		}
	})
}

func TestClientRegistrationResponseJSON(t *testing.T) {
	testCases := []struct {
		name     string
		in       ClientRegistrationResponse
		wantJSON string
	}{
		{
			name: "full response",
			in: ClientRegistrationResponse{
				ClientID:              "test-client-id",
				ClientSecret:          "test-client-secret",
				ClientIDIssuedAt:      time.Unix(1758840047, 0),
				ClientSecretExpiresAt: time.Unix(1790376047, 0),
			},
			wantJSON: `{"client_id":"test-client-id","client_secret":"test-client-secret","client_id_issued_at":1758840047,"client_secret_expires_at":1790376047, "redirect_uris": null}`,
		},
		{
			name: "minimal response with only required fields",
			in: ClientRegistrationResponse{
				ClientID: "test-client-id-minimal",
			},
			wantJSON: `{"client_id":"test-client-id-minimal", "redirect_uris":null}`,
		},
		{
			name: "response with a secret that does not expire",
			in: ClientRegistrationResponse{
				ClientID:     "test-client-id-no-expiry",
				ClientSecret: "test-secret-no-expiry",
			},
			wantJSON: `{"client_id":"test-client-id-no-expiry","client_secret":"test-secret-no-expiry", "redirect_uris":null}`,
		},
		{
			name:     "unmarshal with zero timestamp",
			in:       ClientRegistrationResponse{ClientID: "client-id-zero"},
			wantJSON: `{"client_id":"client-id-zero", "redirect_uris":null}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test MarshalJSON
			t.Run("marshal", func(t *testing.T) {
				b, err := json.Marshal(&tc.in)
				if err != nil {
					t.Fatalf("Marshal() error = %v", err)
				}

				var gotMap, wantMap map[string]any
				if err := json.Unmarshal(b, &gotMap); err != nil {
					t.Fatalf("failed to unmarshal actual result: %v", err)
				}
				if err := json.Unmarshal([]byte(tc.wantJSON), &wantMap); err != nil {
					t.Fatalf("failed to unmarshal expected result: %v", err)
				}

				if diff := cmp.Diff(wantMap, gotMap); diff != "" {
					t.Errorf("Marshal() mismatch (-want +got):\n%s", diff)
				}
			})

			// Test UnmarshalJSON
			t.Run("unmarshal", func(t *testing.T) {
				var got ClientRegistrationResponse
				if err := json.Unmarshal([]byte(tc.wantJSON), &got); err != nil {
					t.Fatalf("Unmarshal() error = %v", err)
				}

				if diff := cmp.Diff(tc.in, got); diff != "" {
					t.Errorf("Unmarshal() mismatch (-want +got):\n%s", diff)
				}
			})
		})
	}
}
</content>
</file>
<file path="internal/oauthex/oauth2.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// Package oauthex implements extensions to OAuth2.
package oauthex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// prependToPath prepends pre to the path of urlStr.
// When pre is the well-known path, this is the algorithm specified in both RFC 9728
// section 3.1 and RFC 8414 section 3.1.
func prependToPath(urlStr, pre string) (string, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}
	p := "/" + strings.Trim(pre, "/")
	if u.Path != "" {
		p += "/"
	}

	u.Path = p + strings.TrimLeft(u.Path, "/")
	return u.String(), nil
}

// getJSON retrieves JSON and unmarshals JSON from the URL, as specified in both
// RFC 9728 and RFC 8414.
// It will not read more than limit bytes from the body.
func getJSON[T any](ctx context.Context, c *http.Client, url string, limit int64) (*T, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c == nil {
		c = http.DefaultClient
	}
	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// Specs require a 200.
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status %s", res.Status)
	}
	// Specs require application/json.
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		return nil, fmt.Errorf("bad content type %q", ct)
	}

	var t T
	dec := json.NewDecoder(io.LimitReader(res.Body, limit))
	if err := dec.Decode(&t); err != nil {
		return nil, err
	}
	return &t, nil
}

// checkURLScheme ensures that its argument is a valid URL with a scheme
// that prevents XSS attacks.
// See #526.
func checkURLScheme(u string) error {
	if u == "" {
		return nil
	}
	uu, err := url.Parse(u)
	if err != nil {
		return err
	}
	scheme := strings.ToLower(uu.Scheme)
	if scheme == "javascript" || scheme == "data" || scheme == "vbscript" {
		return fmt.Errorf("URL has disallowed scheme %q", scheme)
	}
	return nil
}
</content>
</file>
<file path="internal/oauthex/oauth2_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package oauthex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestSplitChallenges(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single challenge no params",
			input: `Basic`,
			want:  []string{`Basic`},
		},
		{
			name:  "single challenge with params",
			input: `Bearer realm="example.com", error="invalid_token"`,
			want:  []string{`Bearer realm="example.com", error="invalid_token"`},
		},
		{
			name:  "single challenge with comma in quoted string",
			input: `Bearer realm="example, with comma"`,
			want:  []string{`Bearer realm="example, with comma"`},
		},
		{
			name:  "two challenges",
			input: `Basic, Bearer realm="example"`,
			want:  []string{`Basic`, ` Bearer realm="example"`},
		},
		{
			name:  "multiple challenges complex",
			input: `Newauth realm="apps", Basic, Bearer realm="example.com", error="invalid_token"`,
			want:  []string{`Newauth realm="apps"`, ` Basic`, ` Bearer realm="example.com", error="invalid_token"`},
		},
		{
			name:  "challenge with escaped quote",
			input: `Bearer realm="example \"quoted\""`,
			want:  []string{`Bearer realm="example \"quoted\""`},
		},
		{
			name:  "empty input",
			input: "",
			want:  []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := splitChallenges(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitChallenges() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSplitChallengesError(t *testing.T) {
	if _, err := splitChallenges(`"Bearer"`); err == nil {
		t.Fatal("got nil, want error")
	}
}

func TestParseSingleChallenge(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    challenge
		wantErr bool
	}{
		{
			name:  "scheme only",
			input: "Basic",
			want: challenge{
				Scheme: "basic",
			},
			wantErr: false,
		},
		{
			name:  "scheme with one quoted param",
			input: `Bearer realm="example.com"`,
			want: challenge{
				Scheme: "bearer",
				Params: map[string]string{"realm": "example.com"},
			},
			wantErr: false,
		},
		{
			name:  "scheme with one unquoted param",
			input: `Bearer realm=example.com`,
			want: challenge{
				Scheme: "bearer",
				Params: map[string]string{"realm": "example.com"},
			},
			wantErr: false,
		},
		{
			name:  "scheme with multiple params",
			input: `Bearer realm="example", error="invalid_token", error_description="The token expired"`,
			want: challenge{
				Scheme: "bearer",
				Params: map[string]string{
					"realm":             "example",
					"error":             "invalid_token",
					"error_description": "The token expired",
				},
			},
			wantErr: false,
		},
		{
			name:  "scheme with multiple unquoted params",
			input: `Bearer realm=example, error=invalid_token, error_description=The token expired`,
			want: challenge{
				Scheme: "bearer",
				Params: map[string]string{
					"realm":             "example",
					"error":             "invalid_token",
					"error_description": "The token expired",
				},
			},
			wantErr: false,
		},
		{
			name:  "case-insensitive scheme and keys",
			input: `BEARER ReAlM="example"`,
			want: challenge{
				Scheme: "bearer",
				Params: map[string]string{"realm": "example"},
			},
			wantErr: false,
		},
		{
			name:  "param with escaped quote",
			input: `Bearer realm="example \"foo\" bar"`,
			want: challenge{
				Scheme: "bearer",
				Params: map[string]string{"realm": `example "foo" bar`},
			},
			wantErr: false,
		},
		{
			name:  "param without quotes (token)",
			input: "Bearer realm=example.com",
			want: challenge{
				Scheme: "bearer",
				Params: map[string]string{"realm": "example.com"},
			},
			wantErr: false,
		},
		{
			name:    "malformed param - no value",
			input:   "Bearer realm=",
			wantErr: true,
		},
		{
			name:    "malformed param - unterminated quote",
			input:   `Bearer realm="example`,
			wantErr: true,
		},
		{
			name:    "malformed param - missing comma",
			input:   `Bearer realm="a" error="b"`,
			wantErr: true,
		},
		{
			name:    "malformed param - initial equal",
			input:   `Bearer ="a"`,
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSingleChallenge(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSingleChallenge() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseSingleChallenge() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetProtectedResourceMetadata(t *testing.T) {
	ctx := context.Background()
	t.Run("FromHeader", func(t *testing.T) {
		h := &fakeResourceHandler{serveWWWAuthenticate: true}
		server := httptest.NewTLSServer(h)
		h.installHandlers(server.URL)
		client := server.Client()
		res, err := client.Get(server.URL + "/resource")
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatal("want unauth")
		}
		prm, err := GetProtectedResourceMetadataFromHeader(ctx, res.Header, client)
		if err != nil {
			t.Fatal(err)
		}
		if prm == nil {
			t.Fatal("nil prm")
		}
	})
	t.Run("FromID", func(t *testing.T) {
		h := &fakeResourceHandler{serveWWWAuthenticate: false}
		server := httptest.NewTLSServer(h)
		h.installHandlers(server.URL)
		client := server.Client()
		prm, err := GetProtectedResourceMetadataFromID(ctx, server.URL, client)
		if err != nil {
			t.Fatal(err)
		}
		if prm == nil {
			t.Fatal("nil prm")
		}
	})
}

type fakeResourceHandler struct {
	http.ServeMux
	serveWWWAuthenticate bool
}

func (h *fakeResourceHandler) installHandlers(serverURL string) {
	path := "/.well-known/oauth-protected-resource"
	url := serverURL + path
	h.Handle("GET /resource", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.serveWWWAuthenticate {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer resource_metadata="%s"`, url))
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	h.Handle("GET "+path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// If there is a WWW-Authenticate header, the resource field is the value of that header.
		// If not, it's the server URL without the "/.well-known/..." part.
		resource := serverURL
		if h.serveWWWAuthenticate {
			resource = url
		}
		prm := &ProtectedResourceMetadata{Resource: resource}
		if err := json.NewEncoder(w).Encode(prm); err != nil {
			panic(err)
		}
	}))
}
</content>
</file>
<file path="internal/oauthex/resource_meta.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// This file implements Protected Resource Metadata.
// See https://www.rfc-editor.org/rfc/rfc9728.html.

package oauthex

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"unicode"

	"github.com/modelcontextprotocol/go-sdk/internal/util"
)

const defaultProtectedResourceMetadataURI = "/.well-known/oauth-protected-resource"

// ProtectedResourceMetadata is the metadata for an OAuth 2.0 protected resource,
// as defined in section 2 of https://www.rfc-editor.org/rfc/rfc9728.html.
//
// The following features are not supported:
// - additional keys (§2, last sentence)
// - human-readable metadata (§2.1)
// - signed metadata (§2.2)
type ProtectedResourceMetadata struct {
	// GENERATED BY GEMINI 2.5.

	// Resource (resource) is the protected resource's resource identifier.
	// Required.
	Resource string `json:"resource"`

	// AuthorizationServers (authorization_servers) is an optional slice containing a list of
	// OAuth authorization server issuer identifiers (as defined in RFC 8414) that can be
	// used with this protected resource.
	AuthorizationServers []string `json:"authorization_servers,omitempty"`

	// JWKSURI (jwks_uri) is an optional URL of the protected resource's JSON Web Key (JWK) Set
	// document. This contains public keys belonging to the protected resource, such as
	// signing key(s) that the resource server uses to sign resource responses.
	JWKSURI string `json:"jwks_uri,omitempty"`

	// ScopesSupported (scopes_supported) is a recommended slice containing a list of scope
	// values (as defined in RFC 6749) used in authorization requests to request access
	// to this protected resource.
	ScopesSupported []string `json:"scopes_supported,omitempty"`

	// BearerMethodsSupported (bearer_methods_supported) is an optional slice containing
	// a list of the supported methods of sending an OAuth 2.0 bearer token to the
	// protected resource. Defined values are "header", "body", and "query".
	BearerMethodsSupported []string `json:"bearer_methods_supported,omitempty"`

	// ResourceSigningAlgValuesSupported (resource_signing_alg_values_supported) is an optional
	// slice of JWS signing algorithms (alg values) supported by the protected
	// resource for signing resource responses.
	ResourceSigningAlgValuesSupported []string `json:"resource_signing_alg_values_supported,omitempty"`

	// ResourceName (resource_name) is a human-readable name of the protected resource
	// intended for display to the end user. It is RECOMMENDED that this field be included.
	// This value may be internationalized.
	ResourceName string `json:"resource_name,omitempty"`

	// ResourceDocumentation (resource_documentation) is an optional URL of a page containing
	// human-readable information for developers using the protected resource.
	// This value may be internationalized.
	ResourceDocumentation string `json:"resource_documentation,omitempty"`

	// ResourcePolicyURI (resource_policy_uri) is an optional URL of a page containing
	// human-readable policy information on how a client can use the data provided.
	// This value may be internationalized.
	ResourcePolicyURI string `json:"resource_policy_uri,omitempty"`

	// ResourceTOSURI (resource_tos_uri) is an optional URL of a page containing the protected
	// resource's human-readable terms of service. This value may be internationalized.
	ResourceTOSURI string `json:"resource_tos_uri,omitempty"`

	// TLSClientCertificateBoundAccessTokens (tls_client_certificate_bound_access_tokens) is an
	// optional boolean indicating support for mutual-TLS client certificate-bound
	// access tokens (RFC 8705). Defaults to false if omitted.
	TLSClientCertificateBoundAccessTokens bool `json:"tls_client_certificate_bound_access_tokens,omitempty"`

	// AuthorizationDetailsTypesSupported (authorization_details_types_supported) is an optional
	// slice of 'type' values supported by the resource server for the
	// 'authorization_details' parameter (RFC 9396).
	AuthorizationDetailsTypesSupported []string `json:"authorization_details_types_supported,omitempty"`

	// DPOPSigningAlgValuesSupported (dpop_signing_alg_values_supported) is an optional
	// slice of JWS signing algorithms supported by the resource server for validating
	// DPoP proof JWTs (RFC 9449).
	DPOPSigningAlgValuesSupported []string `json:"dpop_signing_alg_values_supported,omitempty"`

	// DPOPBoundAccessTokensRequired (dpop_bound_access_tokens_required) is an optional boolean
	// specifying whether the protected resource always requires the use of DPoP-bound
	// access tokens (RFC 9449). Defaults to false if omitted.
	DPOPBoundAccessTokensRequired bool `json:"dpop_bound_access_tokens_required,omitempty"`

	// SignedMetadata (signed_metadata) is an optional JWT containing metadata parameters
	// about the protected resource as claims. If present, these values take precedence
	// over values conveyed in plain JSON.
	// TODO:implement.
	// Note that §2.2 says it's okay to ignore this.
	// SignedMetadata string `json:"signed_metadata,omitempty"`
}

// GetProtectedResourceMetadataFromID issues a GET request to retrieve protected resource
// metadata from a resource server by its ID.
// The resource ID is an HTTPS URL, typically with a host:port and possibly a path.
// For example:
//
//	https://example.com/server
//
// This function, following the spec (§3), inserts the default well-known path into the
// URL. In our example, the result would be
//
//	https://example.com/.well-known/oauth-protected-resource/server
//
// It then retrieves the metadata at that location using the given client (or the
// default client if nil) and validates its resource field against resourceID.
func GetProtectedResourceMetadataFromID(ctx context.Context, resourceID string, c *http.Client) (_ *ProtectedResourceMetadata, err error) {
	defer util.Wrapf(&err, "GetProtectedResourceMetadataFromID(%q)", resourceID)

	u, err := url.Parse(resourceID)
	if err != nil {
		return nil, err
	}
	// Insert well-known URI into URL.
	u.Path = path.Join(defaultProtectedResourceMetadataURI, u.Path)
	return getPRM(ctx, u.String(), c, resourceID)
}

// GetProtectedResourceMetadataFromHeader retrieves protected resource metadata
// using information in the given header, using the given client (or the default
// client if nil).
// It issues a GET request to a URL discovered by parsing the WWW-Authenticate headers in the given request,
// It then validates the resource field of the resulting metadata against the given URL.
// If there is no URL in the request, it returns nil, nil.
func GetProtectedResourceMetadataFromHeader(ctx context.Context, header http.Header, c *http.Client) (_ *ProtectedResourceMetadata, err error) {
	defer util.Wrapf(&err, "GetProtectedResourceMetadataFromHeader")
	headers := header[http.CanonicalHeaderKey("WWW-Authenticate")]
	if len(headers) == 0 {
		return nil, nil
	}
	cs, err := ParseWWWAuthenticate(headers)
	if err != nil {
		return nil, err
	}
	url := ResourceMetadataURL(cs)
	if url == "" {
		return nil, nil
	}
	return getPRM(ctx, url, c, url)
}

// getPRM makes a GET request to the given URL, and validates the response.
// As part of the validation, it compares the returned resource field to wantResource.
func getPRM(ctx context.Context, purl string, c *http.Client, wantResource string) (*ProtectedResourceMetadata, error) {
	if !strings.HasPrefix(strings.ToUpper(purl), "HTTPS://") {
		return nil, fmt.Errorf("resource URL %q does not use HTTPS", purl)
	}
	prm, err := getJSON[ProtectedResourceMetadata](ctx, c, purl, 1<<20)
	if err != nil {
		return nil, err
	}
	// Validate the Resource field to thwart impersonation attacks (section 3.3).
	if prm.Resource != wantResource {
		return nil, fmt.Errorf("got metadata resource %q, want %q", prm.Resource, wantResource)
	}
	// Validate the authorization server URLs to prevent XSS attacks (see #526).
	for _, u := range prm.AuthorizationServers {
		if err := checkURLScheme(u); err != nil {
			return nil, err
		}
	}
	return prm, nil
}

// challenge represents a single authentication challenge from a WWW-Authenticate header.
// As per RFC 9110, Section 11.6.1, a challenge consists of a scheme and optional parameters.
type challenge struct {
	// GENERATED BY GEMINI 2.5.
	//
	// Scheme is the authentication scheme (e.g., "Bearer", "Basic").
	// It is case-insensitive. A parsed value will always be lower-case.
	Scheme string
	// Params is a map of authentication parameters.
	// Keys are case-insensitive. Parsed keys are always lower-case.
	Params map[string]string
}

// ResourceMetadataURL returns a resource metadata URL from the given challenges,
// or the empty string if there is none.
func ResourceMetadataURL(cs []challenge) string {
	for _, c := range cs {
		if u := c.Params["resource_metadata"]; u != "" {
			return u
		}
	}
	return ""
}

// ParseWWWAuthenticate parses a WWW-Authenticate header string.
// The header format is defined in RFC 9110, Section 11.6.1, and can contain
// one or more challenges, separated by commas.
// It returns a slice of challenges or an error if one of the headers is malformed.
func ParseWWWAuthenticate(headers []string) ([]challenge, error) {
	// GENERATED BY GEMINI 2.5 (human-tweaked)
	var challenges []challenge
	for _, h := range headers {
		challengeStrings, err := splitChallenges(h)
		if err != nil {
			return nil, err
		}
		for _, cs := range challengeStrings {
			if strings.TrimSpace(cs) == "" {
				continue
			}
			challenge, err := parseSingleChallenge(cs)
			if err != nil {
				return nil, fmt.Errorf("failed to parse challenge %q: %w", cs, err)
			}
			challenges = append(challenges, challenge)
		}
	}
	return challenges, nil
}

// splitChallenges splits a header value containing one or more challenges.
// It correctly handles commas within quoted strings and distinguishes between
// commas separating auth-params and commas separating challenges.
func splitChallenges(header string) ([]string, error) {
	// GENERATED BY GEMINI 2.5.
	var challenges []string
	inQuotes := false
	start := 0
	for i, r := range header {
		if r == '"' {
			if i > 0 && header[i-1] != '\\' {
				inQuotes = !inQuotes
			} else if i == 0 {
				// A challenge begins with an auth-scheme, which is a token, which cannot contain
				// a quote.
				return nil, errors.New(`challenge begins with '"'`)
			}
		} else if r == ',' && !inQuotes {
			// This is a potential challenge separator.
			// A new challenge does not start with `key=value`.
			// We check if the part after the comma looks like a parameter.
			lookahead := strings.TrimSpace(header[i+1:])
			eqPos := strings.Index(lookahead, "=")

			isParam := false
			if eqPos > 0 {
				// Check if the part before '=' is a single token (no spaces).
				token := lookahead[:eqPos]
				if strings.IndexFunc(token, unicode.IsSpace) == -1 {
					isParam = true
				}
			}

			if !isParam {
				// The part after the comma does not look like a parameter,
				// so this comma separates challenges.
				challenges = append(challenges, header[start:i])
				start = i + 1
			}
		}
	}
	// Add the last (or only) challenge to the list.
	challenges = append(challenges, header[start:])
	return challenges, nil
}

// parseSingleChallenge parses a string containing exactly one challenge.
// challenge   = auth-scheme [ 1*SP ( token68 / #auth-param ) ]
func parseSingleChallenge(s string) (challenge, error) {
	// GENERATED BY GEMINI 2.5, human-tweaked.
	s = strings.TrimSpace(s)
	if s == "" {
		return challenge{}, errors.New("empty challenge string")
	}

	scheme, paramsStr, found := strings.Cut(s, " ")
	c := challenge{Scheme: strings.ToLower(scheme)}
	if !found {
		return c, nil
	}

	params := make(map[string]string)

	// Parse the key-value parameters.
	for paramsStr != "" {
		// Find the end of the parameter key.
		keyEnd := strings.Index(paramsStr, "=")
		if keyEnd <= 0 {
			return challenge{}, fmt.Errorf("malformed auth parameter: expected key=value, but got %q", paramsStr)
		}
		key := strings.TrimSpace(paramsStr[:keyEnd])

		// Move the string past the key and the '='.
		paramsStr = strings.TrimSpace(paramsStr[keyEnd+1:])

		var value string
		if strings.HasPrefix(paramsStr, "\"") {
			// The value is a quoted string.
			paramsStr = paramsStr[1:] // Consume the opening quote.
			var valBuilder strings.Builder
			i := 0
			for ; i < len(paramsStr); i++ {
				// Handle escaped characters.
				if paramsStr[i] == '\\' && i+1 < len(paramsStr) {
					valBuilder.WriteByte(paramsStr[i+1])
					i++ // We've consumed two characters.
				} else if paramsStr[i] == '"' {
					// End of the quoted string.
					break
				} else {
					valBuilder.WriteByte(paramsStr[i])
				}
			}

			// A quoted string must be terminated.
			if i == len(paramsStr) {
				return challenge{}, fmt.Errorf("unterminated quoted string in auth parameter")
			}

			value = valBuilder.String()
			// Move the string past the value and the closing quote.
			paramsStr = strings.TrimSpace(paramsStr[i+1:])
		} else {
			// The value is a token. It ends at the next comma or the end of the string.
			commaPos := strings.Index(paramsStr, ",")
			if commaPos == -1 {
				value = paramsStr
				paramsStr = ""
			} else {
				value = strings.TrimSpace(paramsStr[:commaPos])
				paramsStr = strings.TrimSpace(paramsStr[commaPos:]) // Keep comma for next check
			}
		}
		if value == "" {
			return challenge{}, fmt.Errorf("no value for auth param %q", key)
		}

		// Per RFC 9110, parameter keys are case-insensitive.
		params[strings.ToLower(key)] = value

		// If there is a comma, consume it and continue to the next parameter.
		if strings.HasPrefix(paramsStr, ",") {
			paramsStr = strings.TrimSpace(paramsStr[1:])
		} else if paramsStr != "" {
			// If there's content but it's not a new parameter, the format is wrong.
			return challenge{}, fmt.Errorf("malformed auth parameter: expected comma after value, but got %q", paramsStr)
		}
	}

	// Per RFC 9110, the scheme is case-insensitive.
	return challenge{Scheme: strings.ToLower(scheme), Params: params}, nil
}
</content>
</file>
<file path="internal/readme/client/client.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// !+
package main

import (
	"context"
	"log"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	ctx := context.Background()

	// Create a new client, with no features.
	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-client", Version: "v1.0.0"}, nil)

	// Connect to a server over stdin/stdout.
	transport := &mcp.CommandTransport{Command: exec.Command("myserver")}
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	// Call a tool on the server.
	params := &mcp.CallToolParams{
		Name:      "greet",
		Arguments: map[string]any{"name": "you"},
	}
	res, err := session.CallTool(ctx, params)
	if err != nil {
		log.Fatalf("CallTool failed: %v", err)
	}
	if res.IsError {
		log.Fatal("tool failed")
	}
	for _, c := range res.Content {
		log.Print(c.(*mcp.TextContent).Text)
	}
}

// !-
</content>
</file>
<file path="internal/readme/doc.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

//go:generate go run golang.org/x/example/internal/cmd/weave@latest -o ../../README.md ./README.src.md

// The readme package is used to generate README.md at the top-level of this
// repo. Regenerate the README with go generate.
package readme
</content>
</file>
<file path="internal/readme/server/server.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// !+
package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Input struct {
	Name string `json:"name" jsonschema:"the name of the person to greet"`
}

type Output struct {
	Greeting string `json:"greeting" jsonschema:"the greeting to tell to the user"`
}

func SayHi(ctx context.Context, req *mcp.CallToolRequest, input Input) (
	*mcp.CallToolResult,
	Output,
	error,
) {
	return nil, Output{Greeting: "Hi " + input.Name}, nil
}

func main() {
	// Create a server with a single tool.
	server := mcp.NewServer(&mcp.Implementation{Name: "greeter", Version: "v1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "greet", Description: "say hi"}, SayHi)
	// Run the server over stdin/stdout, until the client disconnects.
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}

// !-
</content>
</file>
<file path="internal/testing/fake_auth_server.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package testing

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	authServerPort = ":8080"
	issuer         = "http://localhost" + authServerPort
	tokenExpiry    = time.Hour
)

var jwtSigningKey = []byte("fake-secret-key")

type authCodeInfo struct {
	codeChallenge string
	redirectURI   string
}

type state struct {
	authCodes map[string]authCodeInfo
}

// NewFakeAuthMux constructs a ServeMux that implements an OAuth 2.1 authentication
// server. It should be used with [httptest.NewTLSServer].
func NewFakeAuthMux() *http.ServeMux {
	s := &state{authCodes: make(map[string]authCodeInfo)}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-authorization-server", s.handleMetadata)
	mux.HandleFunc("/authorize", s.handleAuthorize)
	mux.HandleFunc("/token", s.handleToken)
	return mux
}

func (s *state) handleMetadata(w http.ResponseWriter, r *http.Request) {
	issuer := "https://localhost:" + r.URL.Port()
	metadata := map[string]any{
		"issuer":                                issuer,
		"authorization_endpoint":                issuer + "/authorize",
		"token_endpoint":                        issuer + "/token",
		"jwks_uri":                              issuer + "/.well-known/jwks.json",
		"scopes_supported":                      []string{"openid", "profile", "email"},
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"code_challenge_methods_supported":      []string{"S256"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metadata)
}

func (s *state) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	responseType := query.Get("response_type")
	redirectURI := query.Get("redirect_uri")
	codeChallenge := query.Get("code_challenge")
	codeChallengeMethod := query.Get("code_challenge_method")

	if responseType != "code" {
		http.Error(w, "unsupported_response_type", http.StatusBadRequest)
		return
	}
	if redirectURI == "" {
		http.Error(w, "invalid_request (no redirect_uri)", http.StatusBadRequest)
		return
	}
	if codeChallenge == "" || codeChallengeMethod != "S256" {
		http.Error(w, "invalid_request (code challenge is not S256)", http.StatusBadRequest)
		return
	}
	if query.Get("client_id") == "" {
		http.Error(w, "invalid_request (missing client_id)", http.StatusBadRequest)
		return
	}

	authCode := "fake-auth-code-" + fmt.Sprintf("%d", time.Now().UnixNano())
	s.authCodes[authCode] = authCodeInfo{
		codeChallenge: codeChallenge,
		redirectURI:   redirectURI,
	}

	redirectURL := fmt.Sprintf("%s?code=%s&state=%s", redirectURI, authCode, query.Get("state"))
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (s *state) handleToken(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	grantType := r.Form.Get("grant_type")
	code := r.Form.Get("code")
	codeVerifier := r.Form.Get("code_verifier")
	// Ignore redirect_uri; it is not required in 2.1.
	// https://www.ietf.org/archive/id/draft-ietf-oauth-v2-1-13.html#redirect-uri-in-token-request

	if grantType != "authorization_code" {
		http.Error(w, "unsupported_grant_type", http.StatusBadRequest)
		return
	}

	authCodeInfo, ok := s.authCodes[code]
	if !ok {
		http.Error(w, "invalid_grant", http.StatusBadRequest)
		return
	}
	delete(s.authCodes, code)

	// PKCE verification
	hasher := sha256.New()
	hasher.Write([]byte(codeVerifier))
	calculatedChallenge := base64.RawURLEncoding.EncodeToString(hasher.Sum(nil))
	if calculatedChallenge != authCodeInfo.codeChallenge {
		http.Error(w, "invalid_grant", http.StatusBadRequest)
		return
	}

	// Issue JWT
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": issuer,
		"sub": "fake-user-id",
		"aud": "fake-client-id",
		"exp": now.Add(tokenExpiry).Unix(),
		"iat": now.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString(jwtSigningKey)
	if err != nil {
		http.Error(w, "server_error", http.StatusInternalServerError)
		return
	}

	tokenResponse := map[string]any{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   int(tokenExpiry.Seconds()),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokenResponse)
}
</content>
</file>
<file path="internal/util/util.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package util

import (
	"cmp"
	"fmt"
	"iter"
	"slices"
)

// Helpers below are copied from gopls' moremaps package.

// Sorted returns an iterator over the entries of m in key order.
func Sorted[M ~map[K]V, K cmp.Ordered, V any](m M) iter.Seq2[K, V] {
	// TODO(adonovan): use maps.Sorted if proposal #68598 is accepted.
	return func(yield func(K, V) bool) {
		keys := KeySlice(m)
		slices.Sort(keys)
		for _, k := range keys {
			if !yield(k, m[k]) {
				break
			}
		}
	}
}

// KeySlice returns the keys of the map M, like slices.Collect(maps.Keys(m)).
func KeySlice[M ~map[K]V, K comparable, V any](m M) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	return r
}

// Wrapf wraps *errp with the given formatted message if *errp is not nil.
func Wrapf(errp *error, format string, args ...any) {
	if *errp != nil {
		*errp = fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), *errp)
	}
}
</content>
</file>
<file path="internal/xcontext/xcontext.go">
<type>go</type>
<content>
// Copyright 2019 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// Package xcontext is a package to offer the extra functionality we need
// from contexts that is not available from the standard context package.
package xcontext

import (
	"context"
	"time"
)

// Detach returns a context that keeps all the values of its parent context
// but detaches from the cancellation and error handling.
func Detach(ctx context.Context) context.Context { return detachedContext{ctx} }

type detachedContext struct{ parent context.Context }

func (v detachedContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (v detachedContext) Done() <-chan struct{}       { return nil }
func (v detachedContext) Err() error                  { return nil }
func (v detachedContext) Value(key any) any           { return v.parent.Value(key) }
</content>
</file>
<file path="jsonrpc/jsonrpc.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// Package jsonrpc exposes part of a JSON-RPC v2 implementation
// for use by mcp transport authors.
package jsonrpc

import "github.com/modelcontextprotocol/go-sdk/internal/jsonrpc2"

type (
	// ID is a JSON-RPC request ID.
	ID = jsonrpc2.ID
	// Message is a JSON-RPC message.
	Message = jsonrpc2.Message
	// Request is a JSON-RPC request.
	Request = jsonrpc2.Request
	// Response is a JSON-RPC response.
	Response = jsonrpc2.Response
)

// MakeID coerces the given Go value to an ID. The value should be the
// default JSON marshaling of a Request identifier: nil, float64, or string.
//
// Returns an error if the value type was not a valid Request ID type.
func MakeID(v any) (ID, error) {
	return jsonrpc2.MakeID(v)
}

// EncodeMessage serializes a JSON-RPC message to its wire format.
func EncodeMessage(msg Message) ([]byte, error) {
	return jsonrpc2.EncodeMessage(msg)
}

// DecodeMessage deserializes JSON-RPC wire format data into a Message.
// It returns either a Request or Response based on the message content.
func DecodeMessage(data []byte) (Message, error) {
	return jsonrpc2.DecodeMessage(data)
}
</content>
</file>
<file path="mcp/client.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"slices"
	"sync"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/internal/jsonrpc2"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

// A Client is an MCP client, which may be connected to an MCP server
// using the [Client.Connect] method.
type Client struct {
	impl                    *Implementation
	opts                    ClientOptions
	mu                      sync.Mutex
	roots                   *featureSet[*Root]
	sessions                []*ClientSession
	sendingMethodHandler_   MethodHandler
	receivingMethodHandler_ MethodHandler
}

// NewClient creates a new [Client].
//
// Use [Client.Connect] to connect it to an MCP server.
//
// The first argument must not be nil.
//
// If non-nil, the provided options configure the Client.
func NewClient(impl *Implementation, opts *ClientOptions) *Client {
	if impl == nil {
		panic("nil Implementation")
	}
	c := &Client{
		impl:                    impl,
		roots:                   newFeatureSet(func(r *Root) string { return r.URI }),
		sendingMethodHandler_:   defaultSendingMethodHandler[*ClientSession],
		receivingMethodHandler_: defaultReceivingMethodHandler[*ClientSession],
	}
	if opts != nil {
		c.opts = *opts
	}
	return c
}

// ClientOptions configures the behavior of the client.
type ClientOptions struct {
	// CreateMessageHandler handles incoming requests for sampling/createMessage.
	//
	// Setting CreateMessageHandler to a non-nil value causes the client to
	// advertise the sampling capability.
	CreateMessageHandler func(context.Context, *CreateMessageRequest) (*CreateMessageResult, error)
	// ElicitationHandler handles incoming requests for elicitation/create.
	//
	// Setting ElicitationHandler to a non-nil value causes the client to
	// advertise the elicitation capability.
	ElicitationHandler func(context.Context, *ElicitRequest) (*ElicitResult, error)
	// Handlers for notifications from the server.
	ToolListChangedHandler      func(context.Context, *ToolListChangedRequest)
	PromptListChangedHandler    func(context.Context, *PromptListChangedRequest)
	ResourceListChangedHandler  func(context.Context, *ResourceListChangedRequest)
	ResourceUpdatedHandler      func(context.Context, *ResourceUpdatedNotificationRequest)
	LoggingMessageHandler       func(context.Context, *LoggingMessageRequest)
	ProgressNotificationHandler func(context.Context, *ProgressNotificationClientRequest)
	// If non-zero, defines an interval for regular "ping" requests.
	// If the peer fails to respond to pings originating from the keepalive check,
	// the session is automatically closed.
	KeepAlive time.Duration
}

// bind implements the binder[*ClientSession] interface, so that Clients can
// be connected using [connect].
func (c *Client) bind(mcpConn Connection, conn *jsonrpc2.Connection, state *clientSessionState, onClose func()) *ClientSession {
	assert(mcpConn != nil && conn != nil, "nil connection")
	cs := &ClientSession{conn: conn, mcpConn: mcpConn, client: c, onClose: onClose}
	if state != nil {
		cs.state = *state
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessions = append(c.sessions, cs)
	return cs
}

// disconnect implements the binder[*Client] interface, so that
// Clients can be connected using [connect].
func (c *Client) disconnect(cs *ClientSession) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessions = slices.DeleteFunc(c.sessions, func(cs2 *ClientSession) bool {
		return cs2 == cs
	})
}

// TODO: Consider exporting this type and its field.
type unsupportedProtocolVersionError struct {
	version string
}

func (e unsupportedProtocolVersionError) Error() string {
	return fmt.Sprintf("unsupported protocol version: %q", e.version)
}

// ClientSessionOptions is reserved for future use.
type ClientSessionOptions struct{}

func (c *Client) capabilities() *ClientCapabilities {
	caps := &ClientCapabilities{}
	caps.Roots.ListChanged = true
	if c.opts.CreateMessageHandler != nil {
		caps.Sampling = &SamplingCapabilities{}
	}
	if c.opts.ElicitationHandler != nil {
		caps.Elicitation = &ElicitationCapabilities{}
	}
	return caps
}

// Connect begins an MCP session by connecting to a server over the given
// transport. The resulting session is initialized, and ready to use.
//
// Typically, it is the responsibility of the client to close the connection
// when it is no longer needed. However, if the connection is closed by the
// server, calls or notifications will return an error wrapping
// [ErrConnectionClosed].
func (c *Client) Connect(ctx context.Context, t Transport, _ *ClientSessionOptions) (cs *ClientSession, err error) {
	cs, err = connect(ctx, t, c, (*clientSessionState)(nil), nil)
	if err != nil {
		return nil, err
	}

	params := &InitializeParams{
		ProtocolVersion: latestProtocolVersion,
		ClientInfo:      c.impl,
		Capabilities:    c.capabilities(),
	}
	req := &InitializeRequest{Session: cs, Params: params}
	res, err := handleSend[*InitializeResult](ctx, methodInitialize, req)
	if err != nil {
		_ = cs.Close()
		return nil, err
	}
	if !slices.Contains(supportedProtocolVersions, res.ProtocolVersion) {
		return nil, unsupportedProtocolVersionError{res.ProtocolVersion}
	}
	cs.state.InitializeResult = res
	if hc, ok := cs.mcpConn.(clientConnection); ok {
		hc.sessionUpdated(cs.state)
	}
	req2 := &initializedClientRequest{Session: cs, Params: &InitializedParams{}}
	if err := handleNotify(ctx, notificationInitialized, req2); err != nil {
		_ = cs.Close()
		return nil, err
	}

	if c.opts.KeepAlive > 0 {
		cs.startKeepalive(c.opts.KeepAlive)
	}

	return cs, nil
}

// A ClientSession is a logical connection with an MCP server. Its
// methods can be used to send requests or notifications to the server. Create
// a session by calling [Client.Connect].
//
// Call [ClientSession.Close] to close the connection, or await server
// termination with [ClientSession.Wait].
type ClientSession struct {
	onClose func()

	conn            *jsonrpc2.Connection
	client          *Client
	keepaliveCancel context.CancelFunc
	mcpConn         Connection

	// No mutex is (currently) required to guard the session state, because it is
	// only set synchronously during Client.Connect.
	state clientSessionState
}

type clientSessionState struct {
	InitializeResult *InitializeResult
}

func (cs *ClientSession) InitializeResult() *InitializeResult { return cs.state.InitializeResult }

func (cs *ClientSession) ID() string {
	if c, ok := cs.mcpConn.(hasSessionID); ok {
		return c.SessionID()
	}
	return ""
}

// Close performs a graceful close of the connection, preventing new requests
// from being handled, and waiting for ongoing requests to return. Close then
// terminates the connection.
func (cs *ClientSession) Close() error {
	// Note: keepaliveCancel access is safe without a mutex because:
	// 1. keepaliveCancel is only written once during startKeepalive (happens-before all Close calls)
	// 2. context.CancelFunc is safe to call multiple times and from multiple goroutines
	// 3. The keepalive goroutine calls Close on ping failure, but this is safe since
	//    Close is idempotent and conn.Close() handles concurrent calls correctly
	if cs.keepaliveCancel != nil {
		cs.keepaliveCancel()
	}
	err := cs.conn.Close()

	if cs.onClose != nil {
		cs.onClose()
	}

	return err
}

// Wait waits for the connection to be closed by the server.
// Generally, clients should be responsible for closing the connection.
func (cs *ClientSession) Wait() error {
	return cs.conn.Wait()
}

// startKeepalive starts the keepalive mechanism for this client session.
func (cs *ClientSession) startKeepalive(interval time.Duration) {
	startKeepalive(cs, interval, &cs.keepaliveCancel)
}

// AddRoots adds the given roots to the client,
// replacing any with the same URIs,
// and notifies any connected servers.
func (c *Client) AddRoots(roots ...*Root) {
	// Only notify if something could change.
	if len(roots) == 0 {
		return
	}
	changeAndNotify(c, notificationRootsListChanged, &RootsListChangedParams{},
		func() bool { c.roots.add(roots...); return true })
}

// RemoveRoots removes the roots with the given URIs,
// and notifies any connected servers if the list has changed.
// It is not an error to remove a nonexistent root.
func (c *Client) RemoveRoots(uris ...string) {
	changeAndNotify(c, notificationRootsListChanged, &RootsListChangedParams{},
		func() bool { return c.roots.remove(uris...) })
}

// changeAndNotify is called when a feature is added or removed.
// It calls change, which should do the work and report whether a change actually occurred.
// If there was a change, it notifies a snapshot of the sessions.
func changeAndNotify[P Params](c *Client, notification string, params P, change func() bool) {
	var sessions []*ClientSession
	// Lock for the change, but not for the notification.
	c.mu.Lock()
	if change() {
		sessions = slices.Clone(c.sessions)
	}
	c.mu.Unlock()
	notifySessions(sessions, notification, params)
}

func (c *Client) listRoots(_ context.Context, req *ListRootsRequest) (*ListRootsResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	roots := slices.Collect(c.roots.all())
	if roots == nil {
		roots = []*Root{} // avoid JSON null
	}
	return &ListRootsResult{
		Roots: roots,
	}, nil
}

func (c *Client) createMessage(ctx context.Context, req *CreateMessageRequest) (*CreateMessageResult, error) {
	if c.opts.CreateMessageHandler == nil {
		// TODO: wrap or annotate this error? Pick a standard code?
		return nil, jsonrpc2.NewError(codeUnsupportedMethod, "client does not support CreateMessage")
	}
	return c.opts.CreateMessageHandler(ctx, req)
}

func (c *Client) elicit(ctx context.Context, req *ElicitRequest) (*ElicitResult, error) {
	if c.opts.ElicitationHandler == nil {
		// TODO: wrap or annotate this error? Pick a standard code?
		return nil, jsonrpc2.NewError(codeUnsupportedMethod, "client does not support elicitation")
	}

	// Validate that the requested schema only contains top-level properties without nesting
	schema, err := validateElicitSchema(req.Params.RequestedSchema)
	if err != nil {
		return nil, jsonrpc2.NewError(codeInvalidParams, err.Error())
	}

	res, err := c.opts.ElicitationHandler(ctx, req)
	if err != nil {
		return nil, err
	}

	// Validate elicitation result content against requested schema
	if schema != nil && res.Content != nil {
		// TODO: is this the correct behavior if validation fails?
		// It isn't the *server's* params that are invalid, so why would we return
		// this code to the server?
		resolved, err := schema.Resolve(nil)
		if err != nil {
			return nil, jsonrpc2.NewError(codeInvalidParams, fmt.Sprintf("failed to resolve requested schema: %v", err))
		}

		if err := resolved.Validate(res.Content); err != nil {
			return nil, jsonrpc2.NewError(codeInvalidParams, fmt.Sprintf("elicitation result content does not match requested schema: %v", err))
		}
	}

	return res, nil
}

// validateElicitSchema validates that the schema conforms to MCP elicitation schema requirements.
// Per the MCP specification, elicitation schemas are limited to flat objects with primitive properties only.
func validateElicitSchema(wireSchema any) (*jsonschema.Schema, error) {
	if wireSchema == nil {
		return nil, nil // nil schema is allowed
	}

	var schema *jsonschema.Schema
	if err := remarshal(wireSchema, &schema); err != nil {
		return nil, err
	}

	// The root schema must be of type "object" if specified
	if schema.Type != "" && schema.Type != "object" {
		return nil, fmt.Errorf("elicit schema must be of type 'object', got %q", schema.Type)
	}

	// Check if the schema has properties
	if schema.Properties != nil {
		for propName, propSchema := range schema.Properties {
			if propSchema == nil {
				continue
			}

			if err := validateElicitProperty(propName, propSchema); err != nil {
				return nil, err
			}
		}
	}

	return schema, nil
}

// validateElicitProperty validates a single property in an elicitation schema.
func validateElicitProperty(propName string, propSchema *jsonschema.Schema) error {
	// Check if this property has nested properties (not allowed)
	if len(propSchema.Properties) > 0 {
		return fmt.Errorf("elicit schema property %q contains nested properties, only primitive properties are allowed", propName)
	}

	// Validate based on the property type - only primitives are supported
	switch propSchema.Type {
	case "string":
		return validateElicitStringProperty(propName, propSchema)
	case "number", "integer":
		return validateElicitNumberProperty(propName, propSchema)
	case "boolean":
		return validateElicitBooleanProperty(propName, propSchema)
	default:
		return fmt.Errorf("elicit schema property %q has unsupported type %q, only string, number, integer, and boolean are allowed", propName, propSchema.Type)
	}
}

// validateElicitStringProperty validates string-type properties, including enums.
func validateElicitStringProperty(propName string, propSchema *jsonschema.Schema) error {
	// Handle enum validation (enums are a special case of strings)
	if len(propSchema.Enum) > 0 {
		// Enums must be string type (or untyped which defaults to string)
		if propSchema.Type != "" && propSchema.Type != "string" {
			return fmt.Errorf("elicit schema property %q has enum values but type is %q, enums are only supported for string type", propName, propSchema.Type)
		}
		// Enum values themselves are validated by the JSON schema library
		// Validate enumNames if present - must match enum length
		if propSchema.Extra != nil {
			if enumNamesRaw, exists := propSchema.Extra["enumNames"]; exists {
				// Type check enumNames - should be a slice
				if enumNamesSlice, ok := enumNamesRaw.([]any); ok {
					if len(enumNamesSlice) != len(propSchema.Enum) {
						return fmt.Errorf("elicit schema property %q has %d enum values but %d enumNames, they must match", propName, len(propSchema.Enum), len(enumNamesSlice))
					}
				} else {
					return fmt.Errorf("elicit schema property %q has invalid enumNames type, must be an array", propName)
				}
			}
		}
		return nil
	}

	// Validate format if specified - only specific formats are allowed
	if propSchema.Format != "" {
		allowedFormats := map[string]bool{
			"email":     true,
			"uri":       true,
			"date":      true,
			"date-time": true,
		}
		if !allowedFormats[propSchema.Format] {
			return fmt.Errorf("elicit schema property %q has unsupported format %q, only email, uri, date, and date-time are allowed", propName, propSchema.Format)
		}
	}

	// Validate minLength constraint if specified
	if propSchema.MinLength != nil {
		if *propSchema.MinLength < 0 {
			return fmt.Errorf("elicit schema property %q has invalid minLength %d, must be non-negative", propName, *propSchema.MinLength)
		}
	}

	// Validate maxLength constraint if specified
	if propSchema.MaxLength != nil {
		if *propSchema.MaxLength < 0 {
			return fmt.Errorf("elicit schema property %q has invalid maxLength %d, must be non-negative", propName, *propSchema.MaxLength)
		}
		// Check that maxLength >= minLength if both are specified
		if propSchema.MinLength != nil && *propSchema.MaxLength < *propSchema.MinLength {
			return fmt.Errorf("elicit schema property %q has maxLength %d less than minLength %d", propName, *propSchema.MaxLength, *propSchema.MinLength)
		}
	}

	return nil
}

// validateElicitNumberProperty validates number and integer-type properties.
func validateElicitNumberProperty(propName string, propSchema *jsonschema.Schema) error {
	if propSchema.Minimum != nil && propSchema.Maximum != nil {
		if *propSchema.Maximum < *propSchema.Minimum {
			return fmt.Errorf("elicit schema property %q has maximum %g less than minimum %g", propName, *propSchema.Maximum, *propSchema.Minimum)
		}
	}

	return nil
}

// validateElicitBooleanProperty validates boolean-type properties.
func validateElicitBooleanProperty(propName string, propSchema *jsonschema.Schema) error {
	// Validate default value if specified - must be a valid boolean
	if propSchema.Default != nil {
		var defaultValue bool
		if err := json.Unmarshal(propSchema.Default, &defaultValue); err != nil {
			return fmt.Errorf("elicit schema property %q has invalid default value, must be a boolean: %v", propName, err)
		}
	}

	return nil
}

// AddSendingMiddleware wraps the current sending method handler using the provided
// middleware. Middleware is applied from right to left, so that the first one is
// executed first.
//
// For example, AddSendingMiddleware(m1, m2, m3) augments the method handler as
// m1(m2(m3(handler))).
//
// Sending middleware is called when a request is sent. It is useful for tasks
// such as tracing, metrics, and adding progress tokens.
func (c *Client) AddSendingMiddleware(middleware ...Middleware) {
	c.mu.Lock()
	defer c.mu.Unlock()
	addMiddleware(&c.sendingMethodHandler_, middleware)
}

// AddReceivingMiddleware wraps the current receiving method handler using
// the provided middleware. Middleware is applied from right to left, so that the
// first one is executed first.
//
// For example, AddReceivingMiddleware(m1, m2, m3) augments the method handler as
// m1(m2(m3(handler))).
//
// Receiving middleware is called when a request is received. It is useful for tasks
// such as authentication, request logging and metrics.
func (c *Client) AddReceivingMiddleware(middleware ...Middleware) {
	c.mu.Lock()
	defer c.mu.Unlock()
	addMiddleware(&c.receivingMethodHandler_, middleware)
}

// clientMethodInfos maps from the RPC method name to serverMethodInfos.
//
// The 'allowMissingParams' values are extracted from the protocol schema.
// TODO(rfindley): actually load and validate the protocol schema, rather than
// curating these method flags.
var clientMethodInfos = map[string]methodInfo{
	methodComplete:                  newClientMethodInfo(clientSessionMethod((*ClientSession).Complete), 0),
	methodPing:                      newClientMethodInfo(clientSessionMethod((*ClientSession).ping), missingParamsOK),
	methodListRoots:                 newClientMethodInfo(clientMethod((*Client).listRoots), missingParamsOK),
	methodCreateMessage:             newClientMethodInfo(clientMethod((*Client).createMessage), 0),
	methodElicit:                    newClientMethodInfo(clientMethod((*Client).elicit), missingParamsOK),
	notificationCancelled:           newClientMethodInfo(clientSessionMethod((*ClientSession).cancel), notification|missingParamsOK),
	notificationToolListChanged:     newClientMethodInfo(clientMethod((*Client).callToolChangedHandler), notification|missingParamsOK),
	notificationPromptListChanged:   newClientMethodInfo(clientMethod((*Client).callPromptChangedHandler), notification|missingParamsOK),
	notificationResourceListChanged: newClientMethodInfo(clientMethod((*Client).callResourceChangedHandler), notification|missingParamsOK),
	notificationResourceUpdated:     newClientMethodInfo(clientMethod((*Client).callResourceUpdatedHandler), notification|missingParamsOK),
	notificationLoggingMessage:      newClientMethodInfo(clientMethod((*Client).callLoggingHandler), notification),
	notificationProgress:            newClientMethodInfo(clientSessionMethod((*ClientSession).callProgressNotificationHandler), notification),
}

func (cs *ClientSession) sendingMethodInfos() map[string]methodInfo {
	return serverMethodInfos
}

func (cs *ClientSession) receivingMethodInfos() map[string]methodInfo {
	return clientMethodInfos
}

func (cs *ClientSession) handle(ctx context.Context, req *jsonrpc.Request) (any, error) {
	if req.IsCall() {
		jsonrpc2.Async(ctx)
	}
	return handleReceive(ctx, cs, req)
}

func (cs *ClientSession) sendingMethodHandler() MethodHandler {
	cs.client.mu.Lock()
	defer cs.client.mu.Unlock()
	return cs.client.sendingMethodHandler_
}

func (cs *ClientSession) receivingMethodHandler() MethodHandler {
	cs.client.mu.Lock()
	defer cs.client.mu.Unlock()
	return cs.client.receivingMethodHandler_
}

// getConn implements [Session.getConn].
func (cs *ClientSession) getConn() *jsonrpc2.Connection { return cs.conn }

func (*ClientSession) ping(context.Context, *PingParams) (*emptyResult, error) {
	return &emptyResult{}, nil
}

// cancel is a placeholder: cancellation is handled the jsonrpc2 package.
//
// It should never be invoked in practice because cancellation is preempted,
// but having its signature here facilitates the construction of methodInfo
// that can be used to validate incoming cancellation notifications.
func (*ClientSession) cancel(context.Context, *CancelledParams) (Result, error) {
	return nil, nil
}

func newClientRequest[P Params](cs *ClientSession, params P) *ClientRequest[P] {
	return &ClientRequest[P]{Session: cs, Params: params}
}

// Ping makes an MCP "ping" request to the server.
func (cs *ClientSession) Ping(ctx context.Context, params *PingParams) error {
	_, err := handleSend[*emptyResult](ctx, methodPing, newClientRequest(cs, orZero[Params](params)))
	return err
}

// ListPrompts lists prompts that are currently available on the server.
func (cs *ClientSession) ListPrompts(ctx context.Context, params *ListPromptsParams) (*ListPromptsResult, error) {
	return handleSend[*ListPromptsResult](ctx, methodListPrompts, newClientRequest(cs, orZero[Params](params)))
}

// GetPrompt gets a prompt from the server.
func (cs *ClientSession) GetPrompt(ctx context.Context, params *GetPromptParams) (*GetPromptResult, error) {
	return handleSend[*GetPromptResult](ctx, methodGetPrompt, newClientRequest(cs, orZero[Params](params)))
}

// ListTools lists tools that are currently available on the server.
func (cs *ClientSession) ListTools(ctx context.Context, params *ListToolsParams) (*ListToolsResult, error) {
	return handleSend[*ListToolsResult](ctx, methodListTools, newClientRequest(cs, orZero[Params](params)))
}

// CallTool calls the tool with the given parameters.
//
// The params.Arguments can be any value that marshals into a JSON object.
func (cs *ClientSession) CallTool(ctx context.Context, params *CallToolParams) (*CallToolResult, error) {
	if params == nil {
		params = new(CallToolParams)
	}
	if params.Arguments == nil {
		// Avoid sending nil over the wire.
		params.Arguments = map[string]any{}
	}
	return handleSend[*CallToolResult](ctx, methodCallTool, newClientRequest(cs, orZero[Params](params)))
}

func (cs *ClientSession) SetLoggingLevel(ctx context.Context, params *SetLoggingLevelParams) error {
	_, err := handleSend[*emptyResult](ctx, methodSetLevel, newClientRequest(cs, orZero[Params](params)))
	return err
}

// ListResources lists the resources that are currently available on the server.
func (cs *ClientSession) ListResources(ctx context.Context, params *ListResourcesParams) (*ListResourcesResult, error) {
	return handleSend[*ListResourcesResult](ctx, methodListResources, newClientRequest(cs, orZero[Params](params)))
}

// ListResourceTemplates lists the resource templates that are currently available on the server.
func (cs *ClientSession) ListResourceTemplates(ctx context.Context, params *ListResourceTemplatesParams) (*ListResourceTemplatesResult, error) {
	return handleSend[*ListResourceTemplatesResult](ctx, methodListResourceTemplates, newClientRequest(cs, orZero[Params](params)))
}

// ReadResource asks the server to read a resource and return its contents.
func (cs *ClientSession) ReadResource(ctx context.Context, params *ReadResourceParams) (*ReadResourceResult, error) {
	return handleSend[*ReadResourceResult](ctx, methodReadResource, newClientRequest(cs, orZero[Params](params)))
}

func (cs *ClientSession) Complete(ctx context.Context, params *CompleteParams) (*CompleteResult, error) {
	return handleSend[*CompleteResult](ctx, methodComplete, newClientRequest(cs, orZero[Params](params)))
}

// Subscribe sends a "resources/subscribe" request to the server, asking for
// notifications when the specified resource changes.
func (cs *ClientSession) Subscribe(ctx context.Context, params *SubscribeParams) error {
	_, err := handleSend[*emptyResult](ctx, methodSubscribe, newClientRequest(cs, orZero[Params](params)))
	return err
}

// Unsubscribe sends a "resources/unsubscribe" request to the server, cancelling
// a previous subscription.
func (cs *ClientSession) Unsubscribe(ctx context.Context, params *UnsubscribeParams) error {
	_, err := handleSend[*emptyResult](ctx, methodUnsubscribe, newClientRequest(cs, orZero[Params](params)))
	return err
}

func (c *Client) callToolChangedHandler(ctx context.Context, req *ToolListChangedRequest) (Result, error) {
	if h := c.opts.ToolListChangedHandler; h != nil {
		h(ctx, req)
	}
	return nil, nil
}

func (c *Client) callPromptChangedHandler(ctx context.Context, req *PromptListChangedRequest) (Result, error) {
	if h := c.opts.PromptListChangedHandler; h != nil {
		h(ctx, req)
	}
	return nil, nil
}

func (c *Client) callResourceChangedHandler(ctx context.Context, req *ResourceListChangedRequest) (Result, error) {
	if h := c.opts.ResourceListChangedHandler; h != nil {
		h(ctx, req)
	}
	return nil, nil
}

func (c *Client) callResourceUpdatedHandler(ctx context.Context, req *ResourceUpdatedNotificationRequest) (Result, error) {
	if h := c.opts.ResourceUpdatedHandler; h != nil {
		h(ctx, req)
	}
	return nil, nil
}

func (c *Client) callLoggingHandler(ctx context.Context, req *LoggingMessageRequest) (Result, error) {
	if h := c.opts.LoggingMessageHandler; h != nil {
		h(ctx, req)
	}
	return nil, nil
}

func (cs *ClientSession) callProgressNotificationHandler(ctx context.Context, params *ProgressNotificationParams) (Result, error) {
	if h := cs.client.opts.ProgressNotificationHandler; h != nil {
		h(ctx, clientRequestFor(cs, params))
	}
	return nil, nil
}

// NotifyProgress sends a progress notification from the client to the server
// associated with this session.
// This can be used if the client is performing a long-running task that was
// initiated by the server.
func (cs *ClientSession) NotifyProgress(ctx context.Context, params *ProgressNotificationParams) error {
	return handleNotify(ctx, notificationProgress, newClientRequest(cs, orZero[Params](params)))
}

// Tools provides an iterator for all tools available on the server,
// automatically fetching pages and managing cursors.
// The params argument can set the initial cursor.
// Iteration stops at the first encountered error, which will be yielded.
func (cs *ClientSession) Tools(ctx context.Context, params *ListToolsParams) iter.Seq2[*Tool, error] {
	if params == nil {
		params = &ListToolsParams{}
	}
	return paginate(ctx, params, cs.ListTools, func(res *ListToolsResult) []*Tool {
		return res.Tools
	})
}

// Resources provides an iterator for all resources available on the server,
// automatically fetching pages and managing cursors.
// The params argument can set the initial cursor.
// Iteration stops at the first encountered error, which will be yielded.
func (cs *ClientSession) Resources(ctx context.Context, params *ListResourcesParams) iter.Seq2[*Resource, error] {
	if params == nil {
		params = &ListResourcesParams{}
	}
	return paginate(ctx, params, cs.ListResources, func(res *ListResourcesResult) []*Resource {
		return res.Resources
	})
}

// ResourceTemplates provides an iterator for all resource templates available on the server,
// automatically fetching pages and managing cursors.
// The params argument can set the initial cursor.
// Iteration stops at the first encountered error, which will be yielded.
func (cs *ClientSession) ResourceTemplates(ctx context.Context, params *ListResourceTemplatesParams) iter.Seq2[*ResourceTemplate, error] {
	if params == nil {
		params = &ListResourceTemplatesParams{}
	}
	return paginate(ctx, params, cs.ListResourceTemplates, func(res *ListResourceTemplatesResult) []*ResourceTemplate {
		return res.ResourceTemplates
	})
}

// Prompts provides an iterator for all prompts available on the server,
// automatically fetching pages and managing cursors.
// The params argument can set the initial cursor.
// Iteration stops at the first encountered error, which will be yielded.
func (cs *ClientSession) Prompts(ctx context.Context, params *ListPromptsParams) iter.Seq2[*Prompt, error] {
	if params == nil {
		params = &ListPromptsParams{}
	}
	return paginate(ctx, params, cs.ListPrompts, func(res *ListPromptsResult) []*Prompt {
		return res.Prompts
	})
}

// paginate is a generic helper function to provide a paginated iterator.
func paginate[P listParams, R listResult[T], T any](ctx context.Context, params P, listFunc func(context.Context, P) (R, error), items func(R) []*T) iter.Seq2[*T, error] {
	return func(yield func(*T, error) bool) {
		for {
			res, err := listFunc(ctx, params)
			if err != nil {
				yield(nil, err)
				return
			}
			for _, r := range items(res) {
				if !yield(r, nil) {
					return
				}
			}
			nextCursorVal := res.nextCursorPtr()
			if nextCursorVal == nil || *nextCursorVal == "" {
				return
			}
			*params.cursorPtr() = *nextCursorVal
		}
	}
}
</content>
</file>
<file path="mcp/client_example_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp_test

import (
	"context"
	"fmt"
	"log"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// !+roots

func Example_roots() {
	ctx := context.Background()

	// Create a client with a single root.
	c := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, nil)
	c.AddRoots(&mcp.Root{URI: "file://a"})

	// Now create a server with a handler to receive notifications about roots.
	rootsChanged := make(chan struct{})
	handleRootsChanged := func(ctx context.Context, req *mcp.RootsListChangedRequest) {
		rootList, err := req.Session.ListRoots(ctx, nil)
		if err != nil {
			log.Fatal(err)
		}
		var roots []string
		for _, root := range rootList.Roots {
			roots = append(roots, root.URI)
		}
		fmt.Println(roots)
		close(rootsChanged)
	}
	s := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.0.1"}, &mcp.ServerOptions{
		RootsListChangedHandler: handleRootsChanged,
	})

	// Connect the server and client...
	t1, t2 := mcp.NewInMemoryTransports()
	if _, err := s.Connect(ctx, t1, nil); err != nil {
		log.Fatal(err)
	}

	clientSession, err := c.Connect(ctx, t2, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer clientSession.Close()

	// ...and add a root. The server is notified about the change.
	c.AddRoots(&mcp.Root{URI: "file://b"})
	<-rootsChanged
	// Output: [file://a file://b]
}

// !-roots

// !+sampling

func Example_sampling() {
	ctx := context.Background()

	// Create a client with a sampling handler.
	c := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, &mcp.ClientOptions{
		CreateMessageHandler: func(_ context.Context, req *mcp.CreateMessageRequest) (*mcp.CreateMessageResult, error) {
			return &mcp.CreateMessageResult{
				Content: &mcp.TextContent{
					Text: "would have created a message",
				},
			}, nil
		},
	})

	// Connect the server and client...
	ct, st := mcp.NewInMemoryTransports()
	s := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.0.1"}, nil)
	session, err := s.Connect(ctx, st, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	if _, err := c.Connect(ctx, ct, nil); err != nil {
		log.Fatal(err)
	}

	msg, err := session.CreateMessage(ctx, &mcp.CreateMessageParams{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(msg.Content.(*mcp.TextContent).Text)
	// Output: would have created a message
}

// !-sampling

// !+elicitation

func Example_elicitation() {
	ctx := context.Background()
	ct, st := mcp.NewInMemoryTransports()

	s := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.0.1"}, nil)
	ss, err := s.Connect(ctx, st, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ss.Close()

	c := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, &mcp.ClientOptions{
		ElicitationHandler: func(context.Context, *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
			return &mcp.ElicitResult{Action: "accept", Content: map[string]any{"test": "value"}}, nil
		},
	})
	if _, err := c.Connect(ctx, ct, nil); err != nil {
		log.Fatal(err)
	}
	res, err := ss.Elicit(ctx, &mcp.ElicitParams{
		Message: "This should fail",
		RequestedSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"test": {Type: "string"},
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res.Content["test"])
	// Output: value
}

// !-elicitation
</content>
</file>
<file path="mcp/client_list_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp_test

import (
	"context"
	"encoding/json"
	"iter"
	"log"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestList(t *testing.T) {
	ctx := context.Background()
	server := mcp.NewServer(testImpl, nil)
	client := mcp.NewClient(testImpl, nil)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer serverSession.Close()
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer clientSession.Close()

	t.Run("tools", func(t *testing.T) {
		var wantTools []*mcp.Tool
		for _, name := range []string{"apple", "banana", "cherry"} {
			tt := &mcp.Tool{Name: name, Description: name + " tool"}
			mcp.AddTool(server, tt, SayHi)
			is, err := jsonschema.For[SayHiParams](nil)
			if err != nil {
				t.Fatal(err)
			}
			data, err := json.Marshal(is)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(data, &tt.InputSchema); err != nil {
				t.Fatal(err)
			}
			wantTools = append(wantTools, tt)
		}
		t.Run("list", func(t *testing.T) {
			res, err := clientSession.ListTools(ctx, nil)
			if err != nil {
				t.Fatal("ListTools() failed:", err)
			}
			if diff := cmp.Diff(wantTools, res.Tools, cmpopts.IgnoreUnexported(jsonschema.Schema{})); diff != "" {
				t.Fatalf("ListTools() mismatch (-want +got):\n%s", diff)
			}
		})
		t.Run("iterator", func(t *testing.T) {
			testIterator(t, clientSession.Tools(ctx, nil), wantTools)
		})
	})

	t.Run("resources", func(t *testing.T) {
		var wantResources []*mcp.Resource
		for _, name := range []string{"apple", "banana", "cherry"} {
			r := &mcp.Resource{URI: "http://" + name}
			wantResources = append(wantResources, r)
			server.AddResource(r, nil)
		}

		t.Run("list", func(t *testing.T) {
			res, err := clientSession.ListResources(ctx, nil)
			if err != nil {
				t.Fatal("ListResources() failed:", err)
			}
			if diff := cmp.Diff(wantResources, res.Resources, cmpopts.IgnoreUnexported(jsonschema.Schema{})); diff != "" {
				t.Fatalf("ListResources() mismatch (-want +got):\n%s", diff)
			}
		})
		t.Run("iterator", func(t *testing.T) {
			testIterator(t, clientSession.Resources(ctx, nil), wantResources)
		})
	})

	t.Run("templates", func(t *testing.T) {
		var wantResourceTemplates []*mcp.ResourceTemplate
		for _, name := range []string{"apple", "banana", "cherry"} {
			rt := &mcp.ResourceTemplate{URITemplate: "http://" + name + "/{x}"}
			wantResourceTemplates = append(wantResourceTemplates, rt)
			server.AddResourceTemplate(rt, nil)
		}
		t.Run("list", func(t *testing.T) {
			res, err := clientSession.ListResourceTemplates(ctx, nil)
			if err != nil {
				t.Fatal("ListResourceTemplates() failed:", err)
			}
			if diff := cmp.Diff(wantResourceTemplates, res.ResourceTemplates, cmpopts.IgnoreUnexported(jsonschema.Schema{})); diff != "" {
				t.Fatalf("ListResourceTemplates() mismatch (-want +got):\n%s", diff)
			}
		})
		t.Run("ResourceTemplatesIterator", func(t *testing.T) {
			testIterator(t, clientSession.ResourceTemplates(ctx, nil), wantResourceTemplates)
		})
	})

	t.Run("prompts", func(t *testing.T) {
		var wantPrompts []*mcp.Prompt
		for _, name := range []string{"apple", "banana", "cherry"} {
			p := &mcp.Prompt{Name: name, Description: name + " prompt"}
			wantPrompts = append(wantPrompts, p)
			server.AddPrompt(p, testPromptHandler)
		}
		t.Run("list", func(t *testing.T) {
			res, err := clientSession.ListPrompts(ctx, nil)
			if err != nil {
				t.Fatal("ListPrompts() failed:", err)
			}
			if diff := cmp.Diff(wantPrompts, res.Prompts, cmpopts.IgnoreUnexported(jsonschema.Schema{})); diff != "" {
				t.Fatalf("ListPrompts() mismatch (-want +got):\n%s", diff)
			}
		})
		t.Run("iterator", func(t *testing.T) {
			testIterator(t, clientSession.Prompts(ctx, nil), wantPrompts)
		})
	})
}

func testIterator[T any](t *testing.T, seq iter.Seq2[*T, error], want []*T) {
	t.Helper()
	var got []*T
	for x, err := range seq {
		if err != nil {
			t.Fatalf("iteration failed: %v", err)
		}
		got = append(got, x)
	}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreUnexported(jsonschema.Schema{})); diff != "" {
		t.Fatalf("mismatch (-want +got):\n%s", diff)
	}
}

func testPromptHandler(context.Context, *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	panic("not implemented")
}
</content>
</file>
<file path="mcp/client_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/jsonschema-go/jsonschema"
)

type Item struct {
	Name  string
	Value string
}

type ListTestParams struct {
	Cursor string
}

func (p *ListTestParams) cursorPtr() *string {
	return &p.Cursor
}

type ListTestResult struct {
	Items      []*Item
	NextCursor string
}

func (r *ListTestResult) nextCursorPtr() *string {
	return &r.NextCursor
}

var allItems = []*Item{
	{"alpha", "val-A"},
	{"bravo", "val-B"},
	{"charlie", "val-C"},
	{"delta", "val-D"},
	{"echo", "val-E"},
	{"foxtrot", "val-F"},
	{"golf", "val-G"},
	{"hotel", "val-H"},
	{"india", "val-I"},
	{"juliet", "val-J"},
	{"kilo", "val-K"},
}

// generatePaginatedResults is a helper to create a sequence of mock responses for pagination.
// It simulates a server returning items in pages based on a given page size.
func generatePaginatedResults(all []*Item, pageSize int) []*ListTestResult {
	if len(all) == 0 {
		return []*ListTestResult{{Items: []*Item{}, NextCursor: ""}}
	}
	if pageSize <= 0 {
		panic("pageSize must be greater than 0")
	}
	numPages := (len(all) + pageSize - 1) / pageSize // Ceiling division
	var results []*ListTestResult
	for i := range numPages {
		startIndex := i * pageSize
		endIndex := min(startIndex+pageSize, len(all)) // Use min to prevent out of bounds
		nextCursor := ""
		if endIndex < len(all) { // If there are more items after this page
			nextCursor = fmt.Sprintf("cursor_%d", endIndex)
		}
		results = append(results, &ListTestResult{Items: all[startIndex:endIndex], NextCursor: nextCursor})
	}
	return results
}

func TestClientPaginateBasic(t *testing.T) {
	ctx := context.Background()
	testCases := []struct {
		name          string
		results       []*ListTestResult
		mockError     error
		initialParams *ListTestParams
		expected      []*Item
		expectError   bool
	}{
		{
			name:     "SinglePageAllItems",
			results:  generatePaginatedResults(allItems, len(allItems)),
			expected: allItems,
		},
		{
			name:     "MultiplePages",
			results:  generatePaginatedResults(allItems, 3),
			expected: allItems,
		},
		{
			name:     "EmptyResults",
			results:  generatePaginatedResults([]*Item{}, 10),
			expected: nil,
		},
		{
			name:        "ListFuncReturnsErrorImmediately",
			results:     []*ListTestResult{{}},
			mockError:   fmt.Errorf("API error on first call"),
			expected:    nil,
			expectError: true,
		},
		{
			name:          "InitialCursorProvided",
			initialParams: &ListTestParams{Cursor: "cursor_2"},
			results:       generatePaginatedResults(allItems[2:], 3),
			expected:      allItems[2:],
		},
		{
			name:          "CursorBeyondAllItems",
			initialParams: &ListTestParams{Cursor: "cursor_999"},
			results:       []*ListTestResult{{Items: []*Item{}, NextCursor: ""}},
			expected:      nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			listFunc := func(ctx context.Context, params *ListTestParams) (*ListTestResult, error) {
				if len(tc.results) == 0 {
					t.Fatalf("listFunc called but no more results defined for test case %q", tc.name)
				}
				res := tc.results[0]
				tc.results = tc.results[1:]
				var err error
				if tc.mockError != nil {
					err = tc.mockError
				}
				return res, err
			}

			params := tc.initialParams
			if tc.initialParams == nil {
				params = &ListTestParams{}
			}

			var gotItems []*Item
			var iterationErr error
			seq := paginate(ctx, params, listFunc, func(r *ListTestResult) []*Item { return r.Items })
			for item, err := range seq {
				if err != nil {
					iterationErr = err
					break
				}
				gotItems = append(gotItems, item)
			}
			if tc.expectError {
				if iterationErr == nil {
					t.Errorf("paginate() expected an error during iteration, but got none")
				}
			} else {
				if iterationErr != nil {
					t.Errorf("paginate() got: %v, want: nil", iterationErr)
				}
			}
			if diff := cmp.Diff(tc.expected, gotItems, cmpopts.IgnoreUnexported(jsonschema.Schema{})); diff != "" {
				t.Fatalf("paginate() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestClientPaginateVariousPageSizes(t *testing.T) {
	ctx := context.Background()
	for i := 1; i < len(allItems)+1; i++ {
		testname := fmt.Sprintf("PageSize=%d", i)
		t.Run(testname, func(t *testing.T) {
			results := generatePaginatedResults(allItems, i)
			listFunc := func(ctx context.Context, params *ListTestParams) (*ListTestResult, error) {
				res := results[0]
				results = results[1:]
				return res, nil
			}
			var gotItems []*Item
			seq := paginate(ctx, &ListTestParams{}, listFunc, func(r *ListTestResult) []*Item { return r.Items })
			for item, err := range seq {
				if err != nil {
					t.Fatalf("paginate() unexpected error during iteration: %v", err)
				}
				gotItems = append(gotItems, item)
			}
			if diff := cmp.Diff(allItems, gotItems, cmpopts.IgnoreUnexported(jsonschema.Schema{})); diff != "" {
				t.Fatalf("paginate() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestClientCapabilities(t *testing.T) {
	testCases := []struct {
		name             string
		configureClient  func(s *Client)
		clientOpts       ClientOptions
		wantCapabilities *ClientCapabilities
	}{
		{
			name:            "With initial capabilities",
			configureClient: func(s *Client) {},
			wantCapabilities: &ClientCapabilities{
				Roots: struct {
					ListChanged bool "json:\"listChanged,omitempty\""
				}{ListChanged: true},
			},
		},
		{
			name:            "With sampling",
			configureClient: func(s *Client) {},
			clientOpts: ClientOptions{
				CreateMessageHandler: func(context.Context, *CreateMessageRequest) (*CreateMessageResult, error) {
					return nil, nil
				},
			},
			wantCapabilities: &ClientCapabilities{
				Roots: struct {
					ListChanged bool "json:\"listChanged,omitempty\""
				}{ListChanged: true},
				Sampling: &SamplingCapabilities{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := NewClient(testImpl, &tc.clientOpts)
			tc.configureClient(client)
			gotCapabilities := client.capabilities()
			if diff := cmp.Diff(tc.wantCapabilities, gotCapabilities); diff != "" {
				t.Errorf("capabilities() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
</content>
</file>
<file path="mcp/cmd.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"syscall"
	"time"
)

var defaultTerminateDuration = 5 * time.Second // mutable for testing

// A CommandTransport is a [Transport] that runs a command and communicates
// with it over stdin/stdout, using newline-delimited JSON.
type CommandTransport struct {
	Command *exec.Cmd
	// TerminateDuration controls how long Close waits after closing stdin
	// for the process to exit before sending SIGTERM.
	// If zero or negative, the default of 5s is used.
	TerminateDuration time.Duration
}

// Connect starts the command, and connects to it over stdin/stdout.
func (t *CommandTransport) Connect(ctx context.Context) (Connection, error) {
	stdout, err := t.Command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stdout = io.NopCloser(stdout) // close the connection by closing stdin, not stdout
	stdin, err := t.Command.StdinPipe()
	if err != nil {
		return nil, err
	}
	if err := t.Command.Start(); err != nil {
		return nil, err
	}
	td := t.TerminateDuration
	if td <= 0 {
		td = defaultTerminateDuration
	}
	return newIOConn(&pipeRWC{t.Command, stdout, stdin, td}), nil
}

// A pipeRWC is an io.ReadWriteCloser that communicates with a subprocess over
// stdin/stdout pipes.
type pipeRWC struct {
	cmd               *exec.Cmd
	stdout            io.ReadCloser
	stdin             io.WriteCloser
	terminateDuration time.Duration
}

func (s *pipeRWC) Read(p []byte) (n int, err error) {
	return s.stdout.Read(p)
}

func (s *pipeRWC) Write(p []byte) (n int, err error) {
	return s.stdin.Write(p)
}

// Close closes the input stream to the child process, and awaits normal
// termination of the command. If the command does not exit, it is signalled to
// terminate, and then eventually killed.
func (s *pipeRWC) Close() error {
	// Spec:
	// "For the stdio transport, the client SHOULD initiate shutdown by:...

	// "...First, closing the input stream to the child process (the server)"
	if err := s.stdin.Close(); err != nil {
		return fmt.Errorf("closing stdin: %v", err)
	}
	resChan := make(chan error, 1)
	go func() {
		resChan <- s.cmd.Wait()
	}()
	// "...Waiting for the server to exit, or sending SIGTERM if the server does not exit within a reasonable time"
	wait := func() (error, bool) {
		select {
		case err := <-resChan:
			return err, true
		case <-time.After(s.terminateDuration):
		}
		return nil, false
	}
	if err, ok := wait(); ok {
		return err
	}
	// Note the condition here: if sending SIGTERM fails, don't wait and just
	// move on to SIGKILL.
	if err := s.cmd.Process.Signal(syscall.SIGTERM); err == nil {
		if err, ok := wait(); ok {
			return err
		}
	}
	// "...Sending SIGKILL if the server does not exit within a reasonable time after SIGTERM"
	if err := s.cmd.Process.Kill(); err != nil {
		return err
	}
	if err, ok := wait(); ok {
		return err
	}
	return fmt.Errorf("unresponsive subprocess")
}
</content>
</file>
<file path="mcp/cmd_export_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import "time"

// This file exports some helpers for mutating internals of the command
// transport for testing.

// SetDefaultTerminateDuration sets the default command terminate duration,
// and returns a function to reset it to the default.
func SetDefaultTerminateDuration(d time.Duration) (reset func()) {
	initial := defaultTerminateDuration
	defaultTerminateDuration = d
	return func() {
		defaultTerminateDuration = initial
	}
}
</content>
</file>
<file path="mcp/cmd_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp_test

import (
	"context"
	"errors"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const runAsServer = "_MCP_RUN_AS_SERVER"

type SayHiParams struct {
	Name string `json:"name"`
}

func SayHi(ctx context.Context, req *mcp.CallToolRequest, args SayHiParams) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Hi " + args.Name},
		},
	}, nil, nil
}

func TestMain(m *testing.M) {
	// If the runAsServer variable is set, execute the relevant serverFunc
	// instead of running tests (aka the fork and exec trick).
	if name := os.Getenv(runAsServer); name != "" {
		run := serverFuncs[name]
		if run == nil {
			log.Fatalf("Unknown server %q", name)
		}
		os.Unsetenv(runAsServer)
		run()
		return
	}
	os.Exit(m.Run())
}

// serverFuncs defines server functions that may be run as subprocesses via
// [TestMain].
var serverFuncs = map[string]func(){
	"default":       runServer,
	"cancelContext": runCancelContextServer,
}

func runServer() {
	ctx := context.Background()

	server := mcp.NewServer(testImpl, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "greet", Description: "say hi"}, SayHi)
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}

func runCancelContextServer() {
	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT)
	defer done()

	server := mcp.NewServer(testImpl, nil)
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}

func TestServerRunContextCancel(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "greeter", Version: "v0.0.1"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "greet", Description: "say hi"}, SayHi)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	// run the server and capture the exit error
	onServerExit := make(chan error)
	go func() {
		onServerExit <- server.Run(ctx, serverTransport)
	}()

	// send a ping to the server to ensure it's running
	client := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Ping(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	// cancel the context to stop the server
	cancel()

	// wait for the server to exit
	// TODO: use synctest when availble
	select {
	case <-time.After(5 * time.Second):
		t.Fatal("server did not exit after context cancellation")
	case err := <-onServerExit:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("server did not exit after context cancellation, got error: %v", err)
		}
	}
}

func TestServerInterrupt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX signals")
	}
	requireExec(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := createServerCommand(t, "default")

	client := mcp.NewClient(testImpl, nil)
	_, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatal(err)
	}

	// get a signal when the server process exits
	onExit := make(chan struct{})
	go func() {
		cmd.Process.Wait()
		close(onExit)
	}()

	// send a signal to the server process to terminate it
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}

	// wait for the server to exit
	// TODO: use synctest when available
	select {
	case <-time.After(5 * time.Second):
		t.Fatal("server did not exit after SIGINT")
	case <-onExit:
	}
}

func TestStdioContextCancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX signals")
	}
	requireExec(t)

	// This test is a variant of TestServerInterrupt reproducing the conditions
	// of #224, where interrupt failed to shut down the server because reads of
	// Stdin were not unblocked.

	cmd := createServerCommand(t, "cancelContext")
	// Creating a stdin pipe causes os.Stdin.Close to not immediately unblock
	// pending reads.
	_, _ = cmd.StdinPipe()

	// Just Start the command, rather than connecting to the server, because we
	// don't want the client connection to indirectly flush stdin through writes.
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting command: %v", err)
	}

	// Sleep to make it more likely that the server is blocked in the read loop.
	//
	// This sleep isn't necessary for the test to pass, but *was* necessary for
	// it to fail, before closing was fixed. Unfortunately, it is too invasive a
	// change to have the jsonrpc2 package signal across packages when it is
	// actually blocked in its read loop.
	time.Sleep(100 * time.Millisecond)

	onExit := make(chan struct{})
	go func() {
		cmd.Process.Wait()
		close(onExit)
	}()

	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}

	select {
	case <-time.After(5 * time.Second):
		t.Fatal("server did not exit after SIGINT")
	case <-onExit:
		t.Logf("done.")
	}
}

func TestCmdTransport(t *testing.T) {
	requireExec(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := createServerCommand(t, "default")

	client := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "greet",
		Arguments: map[string]any{"name": "user"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Hi user"},
		},
	}
	if diff := cmp.Diff(want, got, ctrCmpOpts...); diff != "" {
		t.Errorf("greet returned unexpected content (-want +got):\n%s", diff)
	}
	if err := session.Close(); err != nil {
		t.Fatalf("closing server: %v", err)
	}
}

// createServerCommand creates a command to fork and exec the test binary as an
// MCP server.
//
// serverName must refer to an entry in the [serverFuncs] map.
func createServerCommand(t *testing.T, serverName string) *exec.Cmd {
	t.Helper()

	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe)
	cmd.Env = append(os.Environ(), runAsServer+"="+serverName)

	return cmd
}

func TestCommandTransportTerminateDuration(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX signals")
	}
	requireExec(t)

	// Unfortunately, since it does I/O, this test needs to rely on timing (we
	// can't use synctest). However, we can still decreate the default
	// termination duration to speed up the test.
	const defaultDur = 50 * time.Millisecond
	defer mcp.SetDefaultTerminateDuration(defaultDur)()

	tests := []struct {
		name            string
		duration        time.Duration
		wantMinDuration time.Duration
		wantMaxDuration time.Duration
	}{
		{
			name:            "default duration (zero)",
			duration:        0,
			wantMinDuration: defaultDur,
			wantMaxDuration: 1 * time.Second, // default + buffer
		},
		{
			name:            "below minimum duration",
			duration:        -500 * time.Millisecond,
			wantMinDuration: defaultDur,
			wantMaxDuration: 1 * time.Second, // should use default + buffer
		},
		{
			name:            "custom valid duration",
			duration:        200 * time.Millisecond,
			wantMinDuration: 200 * time.Millisecond,
			wantMaxDuration: 1 * time.Second, // custom + buffer
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Use a command that won't exit when stdin is closed
			cmd := exec.Command("sleep", "20")
			transport := &mcp.CommandTransport{
				Command:           cmd,
				TerminateDuration: tt.duration,
			}

			conn, err := transport.Connect(ctx)
			if err != nil {
				t.Fatal(err)
			}

			start := time.Now()
			err = conn.Close()
			elapsed := time.Since(start)

			if err != nil {
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) {
					t.Fatalf("Close() failed with unexpected error: %v", err)
				}
			}
			if elapsed < tt.wantMinDuration {
				t.Errorf("Close() took %v, expected at least %v", elapsed, tt.wantMinDuration)
			}
			if elapsed > tt.wantMaxDuration {
				t.Errorf("Close() took %v, expected at most %v", elapsed, tt.wantMaxDuration)
			}

			// Ensure the process was actually terminated
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		})
	}
}

func requireExec(t *testing.T) {
	t.Helper()

	// Conservatively, limit to major OS where we know that os.Exec is
	// supported.
	switch runtime.GOOS {
	case "darwin", "linux", "windows":
	default:
		t.Skip("unsupported OS")
	}
}

var testImpl = &mcp.Implementation{Name: "test", Version: "v1.0.0"}
</content>
</file>
<file path="mcp/conformance_go124_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

//go:build go1.24 && goexperiment.synctest

package mcp

import (
	"testing"
	"testing/synctest"
)

func runSyncTest(t *testing.T, f func(t *testing.T)) {
	synctest.Run(func() { f(t) })
}
</content>
</file>
<file path="mcp/conformance_go125_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

//go:build go1.25

package mcp

import (
	"testing"
	"testing/synctest"
)

func runSyncTest(t *testing.T, f func(t *testing.T)) {
	synctest.Test(t, f)
}
</content>
</file>
<file path="mcp/conformance_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

//go:build (go1.24 && goexperiment.synctest) || go1.25

package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/internal/jsonrpc2"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"golang.org/x/tools/txtar"
)

var update = flag.Bool("update", false, "if set, update conformance test data")

// A conformance test checks JSON-level conformance of a test server or client.
// This allows us to confirm that we can handle the input or output of other
// SDKs, even if they behave differently at the JSON level (for example, have
// different behavior with respect to optional fields).
//
// The client and server fields hold an encoded sequence of JSON-RPC messages.
//
// For server tests, the client messages are a sequence of messages to be sent
// from the (synthetic) client and the server messages are the expected
// messages to be received from the real server.
//
// For client tests, it's the other way around: server messages are synthetic,
// and client messages are expected from the real client.
//
// Conformance tests are loaded from txtar-encoded testdata files. Run the test
// with -update to have the test runner update the expected output, which may
// be client or server depending on the perspective of the test.
type conformanceTest struct {
	name                      string            // test name
	path                      string            // path to test file
	archive                   *txtar.Archive    // raw archive, for updating
	tools, prompts, resources []string          // named features to include
	client                    []jsonrpc.Message // client messages
	server                    []jsonrpc.Message // server messages
}

// TODO(rfindley): add client conformance tests.

func TestServerConformance(t *testing.T) {
	var tests []*conformanceTest
	dir := filepath.Join("testdata", "conformance", "server")
	if err := filepath.WalkDir(dir, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".txtar") {
			test, err := loadConformanceTest(dir, path)
			if err != nil {
				return fmt.Errorf("%s: %v", path, err)
			}
			tests = append(tests, test)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// We use synctest here because in general, there is no way to know when the
			// server is done processing any notifications. As long as our server doesn't
			// do background work, synctest provides an easy way for us to detect when the
			// server is done processing.
			//
			// By comparison, gopls has a complicated framework based on progress
			// reporting and careful accounting to detect when all 'expected' work
			// on the server is complete.
			runSyncTest(t, func(t *testing.T) { runServerTest(t, test) })

			// TODO: in 1.25, use the following instead:
			// synctest.Test(t, func(t *testing.T) {
			// 	runServerTest(t, test)
			// })
		})
	}
}

type structuredInput struct {
	In string `jsonschema:"the input"`
}

type structuredOutput struct {
	Out string `jsonschema:"the output"`
}

func structuredTool(ctx context.Context, req *CallToolRequest, args *structuredInput) (*CallToolResult, *structuredOutput, error) {
	return nil, &structuredOutput{"Ack " + args.In}, nil
}

type tomorrowInput struct {
	Now time.Time
}

type tomorrowOutput struct {
	Tomorrow time.Time
}

func tomorrowTool(ctx context.Context, req *CallToolRequest, args tomorrowInput) (*CallToolResult, tomorrowOutput, error) {
	return nil, tomorrowOutput{args.Now.Add(24 * time.Hour)}, nil
}

type incInput struct {
	X int `json:"x,omitempty"`
}

type incOutput struct {
	Y int `json:"y"`
}

func incTool(_ context.Context, _ *CallToolRequest, args incInput) (*CallToolResult, incOutput, error) {
	return nil, incOutput{args.X + 1}, nil
}

// runServerTest runs the server conformance test.
// It must be executed in a synctest bubble.
func runServerTest(t *testing.T, test *conformanceTest) {
	ctx := t.Context()
	// Construct the server based on features listed in the test.
	s := NewServer(&Implementation{Name: "testServer", Version: "v1.0.0"}, nil)
	for _, tn := range test.tools {
		switch tn {
		case "greet":
			AddTool(s, &Tool{
				Name:        "greet",
				Description: "say hi",
			}, sayHi)
		case "structured":
			AddTool(s, &Tool{Name: "structured"}, structuredTool)
		case "tomorrow":
			AddTool(s, &Tool{Name: "tomorrow"}, tomorrowTool)
		case "inc":
			inSchema, err := jsonschema.For[incInput](nil)
			if err != nil {
				t.Fatal(err)
			}
			inSchema.Properties["x"].Default = json.RawMessage(`6`)
			AddTool(s, &Tool{Name: "inc", InputSchema: inSchema}, incTool)
		default:
			t.Fatalf("unknown tool %q", tn)
		}
	}
	for _, pn := range test.prompts {
		switch pn {
		case "code_review":
			s.AddPrompt(codeReviewPrompt, codReviewPromptHandler)
		default:
			t.Fatalf("unknown prompt %q", pn)
		}
	}
	for _, rn := range test.resources {
		switch rn {
		case "info.txt":
			s.AddResource(resource1, readHandler)
		case "info":
			s.AddResource(resource3, handleEmbeddedResource)
		default:
			t.Fatalf("unknown resource %q", rn)
		}
	}

	// Connect the server, and connect the client stream,
	// but don't connect an actual client.
	cTransport, sTransport := NewInMemoryTransports()
	ss, err := s.Connect(ctx, sTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	cStream, err := cTransport.Connect(ctx)
	if err != nil {
		t.Fatal(err)
	}

	writeMsg := func(msg jsonrpc.Message) {
		if err := cStream.Write(ctx, msg); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	var (
		serverMessages []jsonrpc.Message
		outRequests    []*jsonrpc.Request
		outResponses   []*jsonrpc.Response
	)

	// Separate client requests and responses; we use them differently.
	for _, msg := range test.client {
		switch msg := msg.(type) {
		case *jsonrpc.Request:
			outRequests = append(outRequests, msg)
		case *jsonrpc.Response:
			outResponses = append(outResponses, msg)
		default:
			t.Fatalf("bad message type %T", msg)
		}
	}

	// nextResponse handles incoming requests and notifications, and returns the
	// next incoming response.
	nextResponse := func() (*jsonrpc.Response, error, bool) {
		for {
			msg, err := cStream.Read(ctx)
			if err != nil {
				// TODO(rfindley): we don't document (or want to document) that the in
				// memory transports use a net.Pipe. How can users detect this failure?
				// Should we promote it to EOF?
				if errors.Is(err, io.ErrClosedPipe) {
					err = nil
				}
				return nil, err, false
			}
			serverMessages = append(serverMessages, msg)
			if req, ok := msg.(*jsonrpc.Request); ok && req.IsCall() {
				// Pair up the next outgoing response with this request.
				// We assume requests arrive in the same order every time.
				if len(outResponses) == 0 {
					t.Fatalf("no outgoing response for request %v", req)
				}
				outResponses[0].ID = req.ID
				writeMsg(outResponses[0])
				outResponses = outResponses[1:]
				continue
			}
			return msg.(*jsonrpc.Response), nil, true
		}
	}

	// Synthetic peer interacts with real peer.
	for _, req := range outRequests {
		writeMsg(req)
		if req.IsCall() {
			// A call (as opposed to a notification). Wait for the response.
			res, err, ok := nextResponse()
			if err != nil {
				t.Fatalf("reading server messages failed: %v", err)
			}
			if !ok {
				t.Fatalf("missing response for request %v", req)
			}
			if res.ID != req.ID {
				t.Fatalf("out-of-order response %v to request %v", req, res)
			}
		}
	}
	// There might be more notifications or requests, but there shouldn't be more
	// responses.
	// Run this in a goroutine so the current thread can wait for it.
	var extra *jsonrpc.Response
	go func() {
		extra, err, _ = nextResponse()
	}()
	// Before closing the stream, wait for all messages to be processed.
	synctest.Wait()
	if err != nil {
		t.Fatalf("reading server messages failedd: %v", err)
	}
	if extra != nil {
		t.Fatalf("got extra response: %v", extra)
	}
	if err := cStream.Close(); err != nil {
		t.Fatalf("Stream.Close failed: %v", err)
	}
	ss.Wait()

	// Handle server output. If -update is set, write the 'server' file.
	// Otherwise, compare with expected.
	if *update {
		arch := &txtar.Archive{
			Comment: test.archive.Comment,
		}
		var buf bytes.Buffer
		for _, msg := range serverMessages {
			data, err := jsonrpc2.EncodeIndent(msg, "", "\t")
			if err != nil {
				t.Fatalf("jsonrpc2.EncodeIndent failed: %v", err)
			}
			buf.Write(data)
			buf.WriteByte('\n')
		}
		serverFile := txtar.File{Name: "server", Data: buf.Bytes()}
		seenServer := false // replace or append the 'server' file
		for _, f := range test.archive.Files {
			if f.Name == "server" {
				seenServer = true
				arch.Files = append(arch.Files, serverFile)
			} else {
				arch.Files = append(arch.Files, f)
			}
		}
		if !seenServer {
			arch.Files = append(arch.Files, serverFile)
		}
		if err := os.WriteFile(test.path, txtar.Format(arch), 0o666); err != nil {
			t.Fatalf("os.WriteFile(%q) failed: %v", test.path, err)
		}
	} else {
		// jsonrpc.Messages are not comparable, so we instead compare lines of JSON.
		transform := cmpopts.AcyclicTransformer("toJSON", func(msg jsonrpc.Message) []string {
			encoded, err := jsonrpc2.EncodeIndent(msg, "", "\t")
			if err != nil {
				t.Fatal(err)
			}
			return strings.Split(string(encoded), "\n")
		})
		if diff := cmp.Diff(test.server, serverMessages, transform); diff != "" {
			t.Errorf("Mismatching server messages (-want +got):\n%s", diff)
		}
	}
}

// loadConformanceTest loads one conformance test from the given path contained
// in the root dir.
func loadConformanceTest(dir, path string) (*conformanceTest, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	test := &conformanceTest{
		name:    strings.TrimPrefix(path, dir+string(filepath.Separator)),
		path:    path,
		archive: txtar.Parse(content),
	}
	if len(test.archive.Files) == 0 {
		return nil, fmt.Errorf("txtar archive %q has no '-- filename --' sections", path)
	}

	// decodeMessages loads JSON-RPC messages from the archive file.
	decodeMessages := func(data []byte) ([]jsonrpc.Message, error) {
		dec := json.NewDecoder(bytes.NewReader(data))
		var res []jsonrpc.Message
		for dec.More() {
			var raw json.RawMessage
			if err := dec.Decode(&raw); err != nil {
				return nil, err
			}
			m, err := jsonrpc2.DecodeMessage(raw)
			if err != nil {
				return nil, err
			}
			res = append(res, m)
		}
		return res, nil
	}
	// loadFeatures loads lists of named features from the archive file.
	loadFeatures := func(data []byte) []string {
		var feats []string
		for line := range strings.Lines(string(data)) {
			if f := strings.TrimSpace(line); f != "" {
				feats = append(feats, f)
			}
		}
		return feats
	}

	seen := make(map[string]bool) // catch accidentally duplicate files
	for _, f := range test.archive.Files {
		if seen[f.Name] {
			return nil, fmt.Errorf("duplicate file name %q", f.Name)
		}
		seen[f.Name] = true
		switch f.Name {
		case "tools":
			test.tools = loadFeatures(f.Data)
		case "prompts":
			test.prompts = loadFeatures(f.Data)
		case "resources":
			test.resources = loadFeatures(f.Data)
		case "client":
			test.client, err = decodeMessages(f.Data)
			if err != nil {
				return nil, fmt.Errorf("txtar archive %q contains bad -- client -- section: %v", path, err)
			}
		case "server":
			test.server, err = decodeMessages(f.Data)
			if err != nil {
				return nil, fmt.Errorf("txtar archive %q contains bad -- server -- section: %v", path, err)
			}
		default:
			return nil, fmt.Errorf("txtar archive %q contains unexpected file %q", path, f.Name)
		}
	}

	return test, nil
}
</content>
</file>
<file path="mcp/content.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// TODO(findleyr): update JSON marshalling of all content types to preserve required fields.
// (See [TextContent.MarshalJSON], which handles this for text content).

package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
)

// A Content is a [TextContent], [ImageContent], [AudioContent],
// [ResourceLink], or [EmbeddedResource].
type Content interface {
	MarshalJSON() ([]byte, error)
	fromWire(*wireContent)
}

// TextContent is a textual content.
type TextContent struct {
	Text        string
	Meta        Meta
	Annotations *Annotations
}

func (c *TextContent) MarshalJSON() ([]byte, error) {
	// Custom wire format to ensure the required "text" field is always included, even when empty.
	wire := struct {
		Type        string       `json:"type"`
		Text        string       `json:"text"`
		Meta        Meta         `json:"_meta,omitempty"`
		Annotations *Annotations `json:"annotations,omitempty"`
	}{
		Type:        "text",
		Text:        c.Text,
		Meta:        c.Meta,
		Annotations: c.Annotations,
	}
	return json.Marshal(wire)
}

func (c *TextContent) fromWire(wire *wireContent) {
	c.Text = wire.Text
	c.Meta = wire.Meta
	c.Annotations = wire.Annotations
}

// ImageContent contains base64-encoded image data.
type ImageContent struct {
	Meta        Meta
	Annotations *Annotations
	Data        []byte // base64-encoded
	MIMEType    string
}

func (c *ImageContent) MarshalJSON() ([]byte, error) {
	// Custom wire format to ensure required fields are always included, even when empty.
	data := c.Data
	if data == nil {
		data = []byte{}
	}
	wire := imageAudioWire{
		Type:        "image",
		MIMEType:    c.MIMEType,
		Data:        data,
		Meta:        c.Meta,
		Annotations: c.Annotations,
	}
	return json.Marshal(wire)
}

func (c *ImageContent) fromWire(wire *wireContent) {
	c.MIMEType = wire.MIMEType
	c.Data = wire.Data
	c.Meta = wire.Meta
	c.Annotations = wire.Annotations
}

// AudioContent contains base64-encoded audio data.
type AudioContent struct {
	Data        []byte
	MIMEType    string
	Meta        Meta
	Annotations *Annotations
}

func (c AudioContent) MarshalJSON() ([]byte, error) {
	// Custom wire format to ensure required fields are always included, even when empty.
	data := c.Data
	if data == nil {
		data = []byte{}
	}
	wire := imageAudioWire{
		Type:        "audio",
		MIMEType:    c.MIMEType,
		Data:        data,
		Meta:        c.Meta,
		Annotations: c.Annotations,
	}
	return json.Marshal(wire)
}

func (c *AudioContent) fromWire(wire *wireContent) {
	c.MIMEType = wire.MIMEType
	c.Data = wire.Data
	c.Meta = wire.Meta
	c.Annotations = wire.Annotations
}

// Custom wire format to ensure required fields are always included, even when empty.
type imageAudioWire struct {
	Type        string       `json:"type"`
	MIMEType    string       `json:"mimeType"`
	Data        []byte       `json:"data"`
	Meta        Meta         `json:"_meta,omitempty"`
	Annotations *Annotations `json:"annotations,omitempty"`
}

// ResourceLink is a link to a resource
type ResourceLink struct {
	URI         string
	Name        string
	Title       string
	Description string
	MIMEType    string
	Size        *int64
	Meta        Meta
	Annotations *Annotations
}

func (c *ResourceLink) MarshalJSON() ([]byte, error) {
	return json.Marshal(&wireContent{
		Type:        "resource_link",
		URI:         c.URI,
		Name:        c.Name,
		Title:       c.Title,
		Description: c.Description,
		MIMEType:    c.MIMEType,
		Size:        c.Size,
		Meta:        c.Meta,
		Annotations: c.Annotations,
	})
}

func (c *ResourceLink) fromWire(wire *wireContent) {
	c.URI = wire.URI
	c.Name = wire.Name
	c.Title = wire.Title
	c.Description = wire.Description
	c.MIMEType = wire.MIMEType
	c.Size = wire.Size
	c.Meta = wire.Meta
	c.Annotations = wire.Annotations
}

// EmbeddedResource contains embedded resources.
type EmbeddedResource struct {
	Resource    *ResourceContents
	Meta        Meta
	Annotations *Annotations
}

func (c *EmbeddedResource) MarshalJSON() ([]byte, error) {
	return json.Marshal(&wireContent{
		Type:        "resource",
		Resource:    c.Resource,
		Meta:        c.Meta,
		Annotations: c.Annotations,
	})
}

func (c *EmbeddedResource) fromWire(wire *wireContent) {
	c.Resource = wire.Resource
	c.Meta = wire.Meta
	c.Annotations = wire.Annotations
}

// ResourceContents contains the contents of a specific resource or
// sub-resource.
type ResourceContents struct {
	URI      string `json:"uri"`
	MIMEType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     []byte `json:"blob,omitempty"`
	Meta     Meta   `json:"_meta,omitempty"`
}

func (r *ResourceContents) MarshalJSON() ([]byte, error) {
	// If we could assume Go 1.24, we could use omitzero for Blob and avoid this method.
	if r.URI == "" {
		return nil, errors.New("ResourceContents missing URI")
	}
	if r.Blob == nil {
		// Text. Marshal normally.
		type wireResourceContents ResourceContents // (lacks MarshalJSON method)
		return json.Marshal((wireResourceContents)(*r))
	}
	// Blob.
	if r.Text != "" {
		return nil, errors.New("ResourceContents has non-zero Text and Blob fields")
	}
	// r.Blob may be the empty slice, so marshal with an alternative definition.
	br := struct {
		URI      string `json:"uri,omitempty"`
		MIMEType string `json:"mimeType,omitempty"`
		Blob     []byte `json:"blob"`
		Meta     Meta   `json:"_meta,omitempty"`
	}{
		URI:      r.URI,
		MIMEType: r.MIMEType,
		Blob:     r.Blob,
		Meta:     r.Meta,
	}
	return json.Marshal(br)
}

// wireContent is the wire format for content.
// It represents the protocol types TextContent, ImageContent, AudioContent,
// ResourceLink, and EmbeddedResource.
// The Type field distinguishes them. In the protocol, each type has a constant
// value for the field.
// At most one of Text, Data, Resource, and URI is non-zero.
type wireContent struct {
	Type        string            `json:"type"`
	Text        string            `json:"text,omitempty"`
	MIMEType    string            `json:"mimeType,omitempty"`
	Data        []byte            `json:"data,omitempty"`
	Resource    *ResourceContents `json:"resource,omitempty"`
	URI         string            `json:"uri,omitempty"`
	Name        string            `json:"name,omitempty"`
	Title       string            `json:"title,omitempty"`
	Description string            `json:"description,omitempty"`
	Size        *int64            `json:"size,omitempty"`
	Meta        Meta              `json:"_meta,omitempty"`
	Annotations *Annotations      `json:"annotations,omitempty"`
}

func contentsFromWire(wires []*wireContent, allow map[string]bool) ([]Content, error) {
	var blocks []Content
	for _, wire := range wires {
		block, err := contentFromWire(wire, allow)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

func contentFromWire(wire *wireContent, allow map[string]bool) (Content, error) {
	if wire == nil {
		return nil, fmt.Errorf("nil content")
	}
	if allow != nil && !allow[wire.Type] {
		return nil, fmt.Errorf("invalid content type %q", wire.Type)
	}
	switch wire.Type {
	case "text":
		v := new(TextContent)
		v.fromWire(wire)
		return v, nil
	case "image":
		v := new(ImageContent)
		v.fromWire(wire)
		return v, nil
	case "audio":
		v := new(AudioContent)
		v.fromWire(wire)
		return v, nil
	case "resource_link":
		v := new(ResourceLink)
		v.fromWire(wire)
		return v, nil
	case "resource":
		v := new(EmbeddedResource)
		v.fromWire(wire)
		return v, nil
	}
	return nil, fmt.Errorf("internal error: unrecognized content type %s", wire.Type)
}
</content>
</file>
<file path="mcp/content_nil_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// This file contains tests to verify that UnmarshalJSON methods for Content types
// don't panic when unmarshaling onto nil pointers, as requested in GitHub issue #205.
//
// NOTE: The contentFromWire function has been fixed to handle nil wire.Content
// gracefully by returning an error instead of panicking.

package mcp_test

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestContentUnmarshalNil(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		content interface{}
		want    interface{}
	}{
		{
			name:    "CallToolResult nil Content",
			json:    `{"content":[{"type":"text","text":"hello"}]}`,
			content: &mcp.CallToolResult{},
			want:    &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "hello"}}},
		},
		{
			name:    "CreateMessageResult nil Content",
			json:    `{"content":{"type":"text","text":"hello"},"model":"test","role":"user"}`,
			content: &mcp.CreateMessageResult{},
			want:    &mcp.CreateMessageResult{Content: &mcp.TextContent{Text: "hello"}, Model: "test", Role: "user"},
		},
		{
			name:    "PromptMessage nil Content",
			json:    `{"content":{"type":"text","text":"hello"},"role":"user"}`,
			content: &mcp.PromptMessage{},
			want:    &mcp.PromptMessage{Content: &mcp.TextContent{Text: "hello"}, Role: "user"},
		},
		{
			name:    "SamplingMessage nil Content",
			json:    `{"content":{"type":"text","text":"hello"},"role":"user"}`,
			content: &mcp.SamplingMessage{},
			want:    &mcp.SamplingMessage{Content: &mcp.TextContent{Text: "hello"}, Role: "user"},
		},
		{
			name:    "CallToolResultFor nil Content",
			json:    `{"content":[{"type":"text","text":"hello"}]}`,
			content: &mcp.CallToolResult{},
			want:    &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "hello"}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that unmarshaling doesn't panic on nil Content fields
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("UnmarshalJSON panicked: %v", r)
				}
			}()

			err := json.Unmarshal([]byte(tt.json), tt.content)
			if err != nil {
				t.Errorf("UnmarshalJSON failed: %v", err)
			}

			// Verify that the Content field was properly populated
			if cmp.Diff(tt.want, tt.content, ctrCmpOpts...) != "" {
				t.Errorf("Content is not equal: %v", cmp.Diff(tt.content, tt.content))
			}
		})
	}
}

func TestContentUnmarshalNilWithDifferentTypes(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		content     interface{}
		expectError bool
	}{
		{
			name:        "ImageContent",
			json:        `{"content":{"type":"image","mimeType":"image/png","data":"YTFiMmMz"}}`,
			content:     &mcp.CreateMessageResult{},
			expectError: false,
		},
		{
			name:        "AudioContent",
			json:        `{"content":{"type":"audio","mimeType":"audio/wav","data":"YTFiMmMz"}}`,
			content:     &mcp.CreateMessageResult{},
			expectError: false,
		},
		{
			name:        "ResourceLink",
			json:        `{"content":{"type":"resource_link","uri":"file:///test","name":"test"}}`,
			content:     &mcp.CreateMessageResult{},
			expectError: true, // CreateMessageResult only allows text, image, audio
		},
		{
			name:        "EmbeddedResource",
			json:        `{"content":{"type":"resource","resource":{"uri":"file://test","text":"test"}}}`,
			content:     &mcp.CreateMessageResult{},
			expectError: true, // CreateMessageResult only allows text, image, audio
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that unmarshaling doesn't panic on nil Content fields
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("UnmarshalJSON panicked: %v", r)
				}
			}()

			err := json.Unmarshal([]byte(tt.json), tt.content)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Verify that the Content field was properly populated for successful cases
			if !tt.expectError {
				if result, ok := tt.content.(*mcp.CreateMessageResult); ok {
					if result.Content == nil {
						t.Error("CreateMessageResult.Content was not populated")
					}
				}
			}
		})
	}
}

func TestContentUnmarshalNilWithEmptyContent(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		content     interface{}
		expectError bool
	}{
		{
			name:        "Empty Content array",
			json:        `{"content":[]}`,
			content:     &mcp.CallToolResult{},
			expectError: false,
		},
		{
			name:        "Missing Content field",
			json:        `{"model":"test","role":"user"}`,
			content:     &mcp.CreateMessageResult{},
			expectError: true, // Content field is required for CreateMessageResult
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that unmarshaling doesn't panic on nil Content fields
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("UnmarshalJSON panicked: %v", r)
				}
			}()

			err := json.Unmarshal([]byte(tt.json), tt.content)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestContentUnmarshalNilWithInvalidContent(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		content     interface{}
		expectError bool
	}{
		{
			name:        "Invalid content type",
			json:        `{"content":{"type":"invalid","text":"hello"}}`,
			content:     &mcp.CreateMessageResult{},
			expectError: true,
		},
		{
			name:        "Missing type field",
			json:        `{"content":{"text":"hello"}}`,
			content:     &mcp.CreateMessageResult{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that unmarshaling doesn't panic on nil Content fields
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("UnmarshalJSON panicked: %v", r)
				}
			}()

			err := json.Unmarshal([]byte(tt.json), tt.content)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

var ctrCmpOpts = []cmp.Option{cmp.AllowUnexported(mcp.CallToolResult{})}
</content>
</file>
<file path="mcp/content_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestContent(t *testing.T) {
	tests := []struct {
		in   mcp.Content
		want string // json serialization
	}{
		{
			&mcp.TextContent{Text: "hello"},
			`{"type":"text","text":"hello"}`,
		},
		{
			&mcp.TextContent{Text: ""},
			`{"type":"text","text":""}`,
		},
		{
			&mcp.TextContent{},
			`{"type":"text","text":""}`,
		},
		{
			&mcp.TextContent{
				Text:        "hello",
				Meta:        mcp.Meta{"key": "value"},
				Annotations: &mcp.Annotations{Priority: 1.0},
			},
			`{"type":"text","text":"hello","_meta":{"key":"value"},"annotations":{"priority":1}}`,
		},
		{
			&mcp.ImageContent{
				Data:     []byte("a1b2c3"),
				MIMEType: "image/png",
			},
			`{"type":"image","mimeType":"image/png","data":"YTFiMmMz"}`,
		},
		{
			&mcp.ImageContent{MIMEType: "image/png", Data: []byte{}},
			`{"type":"image","mimeType":"image/png","data":""}`,
		},
		{
			&mcp.ImageContent{Data: []byte("test")},
			`{"type":"image","mimeType":"","data":"dGVzdA=="}`,
		},
		{
			&mcp.ImageContent{
				Data:        []byte("a1b2c3"),
				MIMEType:    "image/png",
				Meta:        mcp.Meta{"key": "value"},
				Annotations: &mcp.Annotations{Priority: 1.0},
			},
			`{"type":"image","mimeType":"image/png","data":"YTFiMmMz","_meta":{"key":"value"},"annotations":{"priority":1}}`,
		},
		{
			&mcp.AudioContent{
				Data:     []byte("a1b2c3"),
				MIMEType: "audio/wav",
			},
			`{"type":"audio","mimeType":"audio/wav","data":"YTFiMmMz"}`,
		},
		{
			&mcp.AudioContent{MIMEType: "audio/wav", Data: []byte{}},
			`{"type":"audio","mimeType":"audio/wav","data":""}`,
		},
		{
			&mcp.AudioContent{Data: []byte("test")},
			`{"type":"audio","mimeType":"","data":"dGVzdA=="}`,
		},
		{
			&mcp.AudioContent{
				Data:        []byte("a1b2c3"),
				MIMEType:    "audio/wav",
				Meta:        mcp.Meta{"key": "value"},
				Annotations: &mcp.Annotations{Priority: 1.0},
			},
			`{"type":"audio","mimeType":"audio/wav","data":"YTFiMmMz","_meta":{"key":"value"},"annotations":{"priority":1}}`,
		},
		{
			&mcp.EmbeddedResource{
				Resource: &mcp.ResourceContents{URI: "file://foo", MIMEType: "text", Text: "abc"},
			},
			`{"type":"resource","resource":{"uri":"file://foo","mimeType":"text","text":"abc"}}`,
		},
		{
			&mcp.EmbeddedResource{
				Resource: &mcp.ResourceContents{URI: "file://foo", MIMEType: "image/png", Blob: []byte("a1b2c3")},
			},
			`{"type":"resource","resource":{"uri":"file://foo","mimeType":"image/png","blob":"YTFiMmMz"}}`,
		},
		{
			&mcp.EmbeddedResource{
				Resource:    &mcp.ResourceContents{URI: "file://foo", MIMEType: "text", Text: "abc"},
				Meta:        mcp.Meta{"key": "value"},
				Annotations: &mcp.Annotations{Priority: 1.0},
			},
			`{"type":"resource","resource":{"uri":"file://foo","mimeType":"text","text":"abc"},"_meta":{"key":"value"},"annotations":{"priority":1}}`,
		},
		{
			&mcp.ResourceLink{
				URI:  "file:///path/to/file.txt",
				Name: "file.txt",
			},
			`{"type":"resource_link","uri":"file:///path/to/file.txt","name":"file.txt"}`,
		},
		{
			&mcp.ResourceLink{
				URI:         "https://example.com/resource",
				Name:        "Example Resource",
				Title:       "A comprehensive example resource",
				Description: "This resource demonstrates all fields",
				MIMEType:    "text/plain",
				Meta:        mcp.Meta{"custom": "metadata"},
			},
			`{"type":"resource_link","mimeType":"text/plain","uri":"https://example.com/resource","name":"Example Resource","title":"A comprehensive example resource","description":"This resource demonstrates all fields","_meta":{"custom":"metadata"}}`,
		},
	}

	for _, test := range tests {
		got, err := json.Marshal(test.in)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(test.want, string(got)); diff != "" {
			t.Errorf("json.Marshal(%v) mismatch (-want +got):\n%s", test.in, diff)
		}
		result := fmt.Sprintf(`{"content":[%s]}`, string(got))
		var out mcp.CallToolResult
		if err := json.Unmarshal([]byte(result), &out); err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(test.in, out.Content[0]); diff != "" {
			t.Errorf("json.Unmarshal(%q) mismatch (-want +got):\n%s", string(got), diff)
		}
	}
}

func TestEmbeddedResource(t *testing.T) {
	for _, tt := range []struct {
		rc   *mcp.ResourceContents
		want string // marshaled JSON
	}{
		{
			&mcp.ResourceContents{URI: "u", Text: "t"},
			`{"uri":"u","text":"t"}`,
		},
		{
			&mcp.ResourceContents{URI: "u", MIMEType: "m", Text: "t", Meta: mcp.Meta{"key": "value"}},
			`{"uri":"u","mimeType":"m","text":"t","_meta":{"key":"value"}}`,
		},
		{
			&mcp.ResourceContents{URI: "u"},
			`{"uri":"u"}`,
		},
		{
			&mcp.ResourceContents{URI: "u", Blob: []byte{}},
			`{"uri":"u","blob":""}`,
		},
		{
			&mcp.ResourceContents{URI: "u", Blob: []byte{1}},
			`{"uri":"u","blob":"AQ=="}`,
		},
		{
			&mcp.ResourceContents{URI: "u", MIMEType: "m", Blob: []byte{1}, Meta: mcp.Meta{"key": "value"}},
			`{"uri":"u","mimeType":"m","blob":"AQ==","_meta":{"key":"value"}}`,
		},
	} {
		data, err := json.Marshal(tt.rc)
		if err != nil {
			t.Fatal(err)
		}
		if got := string(data); got != tt.want {
			t.Errorf("%#v:\ngot  %s\nwant %s", tt.rc, got, tt.want)
		}
		urc := new(mcp.ResourceContents)
		if err := json.Unmarshal(data, urc); err != nil {
			t.Fatal(err)
		}
		// Since Blob is omitempty, the empty slice changes to nil.
		if diff := cmp.Diff(tt.rc, urc); diff != "" {
			t.Errorf("mismatch (-want, +got):\n%s", diff)
		}
	}
}
</content>
</file>
<file path="mcp/event.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// This file is for SSE events.
// See https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events.

package mcp

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
)

// If true, MemoryEventStore will do frequent validation to check invariants, slowing it down.
// Enable for debugging.
const validateMemoryEventStore = false

// An Event is a server-sent event.
// See https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events#fields.
type Event struct {
	Name string // the "event" field
	ID   string // the "id" field
	Data []byte // the "data" field
}

// Empty reports whether the Event is empty.
func (e Event) Empty() bool {
	return e.Name == "" && e.ID == "" && len(e.Data) == 0
}

// writeEvent writes the event to w, and flushes.
func writeEvent(w io.Writer, evt Event) (int, error) {
	var b bytes.Buffer
	if evt.Name != "" {
		fmt.Fprintf(&b, "event: %s\n", evt.Name)
	}
	if evt.ID != "" {
		fmt.Fprintf(&b, "id: %s\n", evt.ID)
	}
	fmt.Fprintf(&b, "data: %s\n\n", string(evt.Data))
	n, err := w.Write(b.Bytes())
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	return n, err
}

// scanEvents iterates SSE events in the given scanner. The iterated error is
// terminal: if encountered, the stream is corrupt or broken and should no
// longer be used.
//
// TODO(rfindley): consider a different API here that makes failure modes more
// apparent.
func scanEvents(r io.Reader) iter.Seq2[Event, error] {
	scanner := bufio.NewScanner(r)
	const maxTokenSize = 1 * 1024 * 1024 // 1 MiB max line size
	scanner.Buffer(nil, maxTokenSize)

	// TODO: investigate proper behavior when events are out of order, or have
	// non-standard names.
	var (
		eventKey = []byte("event")
		idKey    = []byte("id")
		dataKey  = []byte("data")
	)

	return func(yield func(Event, error) bool) {
		// iterate event from the wire.
		// https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events#examples
		//
		//  - `key: value` line records.
		//  - Consecutive `data: ...` fields are joined with newlines.
		//  - Unrecognized fields are ignored. Since we only care about 'event', 'id', and
		//   'data', these are the only three we consider.
		//  - Lines starting with ":" are ignored.
		//  - Records are terminated with two consecutive newlines.
		var (
			evt     Event
			dataBuf *bytes.Buffer // if non-nil, preceding field was also data
		)
		flushData := func() {
			if dataBuf != nil {
				evt.Data = dataBuf.Bytes()
				dataBuf = nil
			}
		}
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				flushData()
				// \n\n is the record delimiter
				if !evt.Empty() && !yield(evt, nil) {
					return
				}
				evt = Event{}
				continue
			}
			before, after, found := bytes.Cut(line, []byte{':'})
			if !found {
				yield(Event{}, fmt.Errorf("malformed line in SSE stream: %q", string(line)))
				return
			}
			if !bytes.Equal(before, dataKey) {
				flushData()
			}
			switch {
			case bytes.Equal(before, eventKey):
				evt.Name = strings.TrimSpace(string(after))
			case bytes.Equal(before, idKey):
				evt.ID = strings.TrimSpace(string(after))
			case bytes.Equal(before, dataKey):
				data := bytes.TrimSpace(after)
				if dataBuf != nil {
					dataBuf.WriteByte('\n')
					dataBuf.Write(data)
				} else {
					dataBuf = new(bytes.Buffer)
					dataBuf.Write(data)
				}
			}
		}
		if err := scanner.Err(); err != nil {
			if errors.Is(err, bufio.ErrTooLong) {
				err = fmt.Errorf("event exceeded max line length of %d", maxTokenSize)
			}
			if !yield(Event{}, err) {
				return
			}
		}
		flushData()
		if !evt.Empty() {
			yield(evt, nil)
		}
	}
}

// An EventStore tracks data for SSE streams.
// A single EventStore suffices for all sessions, since session IDs are
// globally unique. So one EventStore can be created per process, for
// all Servers in the process.
// Such a store is able to bound resource usage for the entire process.
//
// All of an EventStore's methods must be safe for use by multiple goroutines.
type EventStore interface {
	// Open prepares the event store for a given stream. It ensures that the
	// underlying data structure for the stream is initialized, making it
	// ready to store event streams.
	//
	// streamIDs must be globally unique.
	Open(_ context.Context, sessionID, streamID string) error

	// Append appends data for an outgoing event to given stream, which is part of the
	// given session.
	Append(_ context.Context, sessionID, streamID string, data []byte) error

	// After returns an iterator over the data for the given session and stream, beginning
	// just after the given index.
	// Once the iterator yields a non-nil error, it will stop.
	// After's iterator must return an error immediately if any data after index was
	// dropped; it must not return partial results.
	// The stream must have been opened previously (see [EventStore.Open]).
	After(_ context.Context, sessionID, streamID string, index int) iter.Seq2[[]byte, error]

	// SessionClosed informs the store that the given session is finished, along
	// with all of its streams.
	// A store cannot rely on this method being called for cleanup. It should institute
	// additional mechanisms, such as timeouts, to reclaim storage.
	SessionClosed(_ context.Context, sessionID string) error

	// There is no StreamClosed method. A server doesn't know when a stream is finished, because
	// the client can always send a GET with a Last-Event-ID referring to the stream.
}

// A dataList is a list of []byte.
// The zero dataList is ready to use.
type dataList struct {
	size  int // total size of data bytes
	first int // the stream index of the first element in data
	data  [][]byte
}

func (dl *dataList) appendData(d []byte) {
	// If we allowed empty data, we would consume memory without incrementing the size.
	// We could of course account for that, but we keep it simple and assume there is no
	// empty data.
	if len(d) == 0 {
		panic("empty data item")
	}
	dl.data = append(dl.data, d)
	dl.size += len(d)
}

// removeFirst removes the first data item in dl, returning the size of the item.
// It panics if dl is empty.
func (dl *dataList) removeFirst() int {
	if len(dl.data) == 0 {
		panic("empty dataList")
	}
	r := len(dl.data[0])
	dl.size -= r
	dl.data[0] = nil // help GC
	dl.data = dl.data[1:]
	dl.first++
	return r
}

// A MemoryEventStore is an [EventStore] backed by memory.
type MemoryEventStore struct {
	mu       sync.Mutex
	maxBytes int                             // max total size of all data
	nBytes   int                             // current total size of all data
	store    map[string]map[string]*dataList // session ID -> stream ID -> *dataList
}

// MemoryEventStoreOptions are options for a [MemoryEventStore].
type MemoryEventStoreOptions struct{}

// MaxBytes returns the maximum number of bytes that the store will retain before
// purging data.
func (s *MemoryEventStore) MaxBytes() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.maxBytes
}

// SetMaxBytes sets the maximum number of bytes the store will retain before purging
// data. The argument must not be negative. If it is zero, a suitable default will be used.
// SetMaxBytes can be called at any time. The size of the store will be adjusted
// immediately.
func (s *MemoryEventStore) SetMaxBytes(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case n < 0:
		panic("negative argument")
	case n == 0:
		s.maxBytes = defaultMaxBytes
	default:
		s.maxBytes = n
	}
	s.purge()
}

const defaultMaxBytes = 10 << 20 // 10 MiB

// NewMemoryEventStore creates a [MemoryEventStore] with the default value
// for MaxBytes.
func NewMemoryEventStore(opts *MemoryEventStoreOptions) *MemoryEventStore {
	return &MemoryEventStore{
		maxBytes: defaultMaxBytes,
		store:    make(map[string]map[string]*dataList),
	}
}

// Open implements [EventStore.Open]. It ensures that the underlying data
// structures for the given session are initialized and ready for use.
func (s *MemoryEventStore) Open(_ context.Context, sessionID, streamID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.init(sessionID, streamID)
	return nil
}

// init is an internal helper function that ensures the nested map structure for a
// given sessionID and streamID exists, creating it if necessary. It returns the
// dataList associated with the specified IDs.
// Requires s.mu.
func (s *MemoryEventStore) init(sessionID, streamID string) *dataList {
	streamMap, ok := s.store[sessionID]
	if !ok {
		streamMap = make(map[string]*dataList)
		s.store[sessionID] = streamMap
	}
	dl, ok := streamMap[streamID]
	if !ok {
		dl = &dataList{}
		streamMap[streamID] = dl
	}
	return dl
}

// Append implements [EventStore.Append] by recording data in memory.
func (s *MemoryEventStore) Append(_ context.Context, sessionID, streamID string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	dl := s.init(sessionID, streamID)
	// Purge before adding, so at least the current data item will be present.
	// (That could result in nBytes > maxBytes, but we'll live with that.)
	s.purge()
	dl.appendData(data)
	s.nBytes += len(data)
	return nil
}

// ErrEventsPurged is the error that [EventStore.After] should return if the event just after the
// index is no longer available.
var ErrEventsPurged = errors.New("data purged")

// After implements [EventStore.After].
func (s *MemoryEventStore) After(_ context.Context, sessionID, streamID string, index int) iter.Seq2[[]byte, error] {
	// Return the data items to yield.
	// We must copy, because dataList.removeFirst nils out slice elements.
	copyData := func() ([][]byte, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		streamMap, ok := s.store[sessionID]
		if !ok {
			return nil, fmt.Errorf("MemoryEventStore.After: unknown session ID %q", sessionID)
		}
		dl, ok := streamMap[streamID]
		if !ok {
			return nil, fmt.Errorf("MemoryEventStore.After: unknown stream ID %v in session %q", streamID, sessionID)
		}
		start := index + 1
		if dl.first > start {
			return nil, fmt.Errorf("MemoryEventStore.After: index %d, stream ID %v, session %q: %w",
				index, streamID, sessionID, ErrEventsPurged)
		}
		return slices.Clone(dl.data[start-dl.first:]), nil
	}

	return func(yield func([]byte, error) bool) {
		ds, err := copyData()
		if err != nil {
			yield(nil, err)
			return
		}
		for _, d := range ds {
			if !yield(d, nil) {
				return
			}
		}
	}
}

// SessionClosed implements [EventStore.SessionClosed].
func (s *MemoryEventStore) SessionClosed(_ context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, dl := range s.store[sessionID] {
		s.nBytes -= dl.size
	}
	delete(s.store, sessionID)
	s.validate()
	return nil
}

// purge removes data until no more than s.maxBytes bytes are in use.
// It must be called with s.mu held.
func (s *MemoryEventStore) purge() {
	// Remove the first element of every dataList until below the max.
	for s.nBytes > s.maxBytes {
		changed := false
		for _, sm := range s.store {
			for _, dl := range sm {
				if dl.size > 0 {
					r := dl.removeFirst()
					if r > 0 {
						changed = true
						s.nBytes -= r
					}
				}
			}
		}
		if !changed {
			panic("no progress during purge")
		}
	}
	s.validate()
}

// validate checks that the store's data structures are valid.
// It must be called with s.mu held.
func (s *MemoryEventStore) validate() {
	if !validateMemoryEventStore {
		return
	}
	// Check that we're accounting for the size correctly.
	n := 0
	for _, sm := range s.store {
		for _, dl := range sm {
			for _, d := range dl.data {
				n += len(d)
			}
		}
	}
	if n != s.nBytes {
		panic("sizes don't add up")
	}
}

// debugString returns a string containing the state of s.
// Used in tests.
func (s *MemoryEventStore) debugString() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var b strings.Builder
	for i, sess := range slices.Sorted(maps.Keys(s.store)) {
		if i > 0 {
			fmt.Fprintf(&b, "; ")
		}
		sm := s.store[sess]
		for i, sid := range slices.Sorted(maps.Keys(sm)) {
			if i > 0 {
				fmt.Fprintf(&b, "; ")
			}
			dl := sm[sid]
			fmt.Fprintf(&b, "%s %s first=%d", sess, sid, dl.first)
			for _, d := range dl.data {
				fmt.Fprintf(&b, " %s", d)
			}
		}
	}
	return b.String()
}
</content>
</file>
<file path="mcp/event_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestScanEvents(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []Event
		wantErr string
	}{
		{
			name:  "simple event",
			input: "event: message\nid: 1\ndata: hello\n\n",
			want: []Event{
				{Name: "message", ID: "1", Data: []byte("hello")},
			},
		},
		{
			name:  "multiple data lines",
			input: "data: line 1\ndata: line 2\n\n",
			want: []Event{
				{Data: []byte("line 1\nline 2")},
			},
		},
		{
			name:  "multiple events",
			input: "data: first\n\nevent: second\ndata: second\n\n",
			want: []Event{
				{Data: []byte("first")},
				{Name: "second", Data: []byte("second")},
			},
		},
		{
			name:  "no trailing newline",
			input: "data: hello",
			want: []Event{
				{Data: []byte("hello")},
			},
		},
		{
			name:    "malformed line",
			input:   "invalid line\n\n",
			wantErr: "malformed line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			var got []Event
			var err error
			for e, err2 := range scanEvents(r) {
				if err2 != nil {
					err = err2
					break
				}
				got = append(got, e)
			}

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("scanEvents() got nil error, want error containing %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("scanEvents() error = %q, want containing %q", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("scanEvents() returned unexpected error: %v", err)
			}

			if len(got) != len(tt.want) {
				t.Fatalf("scanEvents() got %d events, want %d", len(got), len(tt.want))
			}

			for i := range got {
				if g, w := got[i].Name, tt.want[i].Name; g != w {
					t.Errorf("event %d: name = %q, want %q", i, g, w)
				}
				if g, w := got[i].ID, tt.want[i].ID; g != w {
					t.Errorf("event %d: id = %q, want %q", i, g, w)
				}
				if g, w := string(got[i].Data), string(tt.want[i].Data); g != w {
					t.Errorf("event %d: data = %q, want %q", i, g, w)
				}
			}
		})
	}
}

func TestMemoryEventStoreState(t *testing.T) {
	ctx := context.Background()

	appendEvent := func(s *MemoryEventStore, sess, stream string, data string) {
		if err := s.Append(ctx, sess, stream, []byte(data)); err != nil {
			t.Fatal(err)
		}
	}

	for _, tt := range []struct {
		name     string
		actions  func(*MemoryEventStore)
		want     string // output of debugString
		wantSize int    // value of nBytes
	}{
		{
			"appends",
			func(s *MemoryEventStore) {
				appendEvent(s, "S1", "1", "d1")
				appendEvent(s, "S1", "2", "d2")
				appendEvent(s, "S1", "1", "d3")
				appendEvent(s, "S2", "8", "d4")
			},
			"S1 1 first=0 d1 d3; S1 2 first=0 d2; S2 8 first=0 d4",
			8,
		},
		{
			"session close",
			func(s *MemoryEventStore) {
				appendEvent(s, "S1", "1", "d1")
				appendEvent(s, "S1", "2", "d2")
				appendEvent(s, "S1", "1", "d3")
				appendEvent(s, "S2", "8", "d4")
				s.SessionClosed(ctx, "S1")
			},
			"S2 8 first=0 d4",
			2,
		},
		{
			"purge",
			func(s *MemoryEventStore) {
				appendEvent(s, "S1", "1", "d1")
				appendEvent(s, "S1", "2", "d2")
				appendEvent(s, "S1", "1", "d3")
				appendEvent(s, "S2", "8", "d4")
				// We are using 8 bytes (d1,d2, d3, d4).
				// To purge 6, we remove the first of each stream, leaving only d3.
				s.SetMaxBytes(2)
			},
			// The other streams remain, because we may add to them.
			"S1 1 first=1 d3; S1 2 first=1; S2 8 first=1",
			2,
		},
		{
			"purge append",
			func(s *MemoryEventStore) {
				appendEvent(s, "S1", "1", "d1")
				appendEvent(s, "S1", "2", "d2")
				appendEvent(s, "S1", "1", "d3")
				appendEvent(s, "S2", "8", "d4")
				s.SetMaxBytes(2)
				// Up to here, identical to the "purge" case.
				// Each of these additions will result in a purge.
				appendEvent(s, "S1", "2", "d5") // remove d3
				appendEvent(s, "S1", "2", "d6") // remove d5
			},
			"S1 1 first=2; S1 2 first=2 d6; S2 8 first=1",
			2,
		},
		{
			"purge resize append",
			func(s *MemoryEventStore) {
				appendEvent(s, "S1", "1", "d1")
				appendEvent(s, "S1", "2", "d2")
				appendEvent(s, "S1", "1", "d3")
				appendEvent(s, "S2", "8", "d4")
				s.SetMaxBytes(2)
				// Up to here, identical to the "purge" case.
				s.SetMaxBytes(6) // make room
				appendEvent(s, "S1", "2", "d5")
				appendEvent(s, "S1", "2", "d6")
			},
			// The other streams remain, because we may add to them.
			"S1 1 first=1 d3; S1 2 first=1 d5 d6; S2 8 first=1",
			6,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := NewMemoryEventStore(nil)
			tt.actions(s)
			got := s.debugString()
			if got != tt.want {
				t.Errorf("\ngot  %s\nwant %s", got, tt.want)
			}
			if g, w := s.nBytes, tt.wantSize; g != w {
				t.Errorf("got size %d, want %d", g, w)
			}
		})
	}
}

func TestMemoryEventStoreAfter(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryEventStore(nil)
	s.SetMaxBytes(4)
	s.Append(ctx, "S1", "1", []byte("d1"))
	s.Append(ctx, "S1", "1", []byte("d2"))
	s.Append(ctx, "S1", "1", []byte("d3"))
	s.Append(ctx, "S1", "2", []byte("d4")) // will purge d1
	want := "S1 1 first=1 d2 d3; S1 2 first=0 d4"
	if got := s.debugString(); got != want {
		t.Fatalf("got state %q, want %q", got, want)
	}

	for _, tt := range []struct {
		sessionID string
		streamID  string
		index     int
		want      []string
		wantErr   string // if non-empty, error should contain this string
	}{
		{"S1", "1", 0, []string{"d2", "d3"}, ""},
		{"S1", "1", 1, []string{"d3"}, ""},
		{"S1", "1", 2, nil, ""},
		{"S1", "2", 0, nil, ""},
		{"S1", "3", 0, nil, "unknown stream ID"},
		{"S2", "0", 0, nil, "unknown session ID"},
	} {
		t.Run(fmt.Sprintf("%s-%s-%d", tt.sessionID, tt.streamID, tt.index), func(t *testing.T) {
			var got []string
			for d, err := range s.After(ctx, tt.sessionID, tt.streamID, tt.index) {
				if err != nil {
					if tt.wantErr == "" {
						t.Fatalf("unexpected error %q", err)
					} else if g := err.Error(); !strings.Contains(g, tt.wantErr) {
						t.Fatalf("got error %q, want it to contain %q", g, tt.wantErr)
					} else {
						return
					}
				}
				got = append(got, string(d))
			}
			if tt.wantErr != "" {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkMemoryEventStore(b *testing.B) {
	// Benchmark with various settings for event store size, number of session,
	// and payload size.
	//
	// Assume a small number of streams per session, which is probably realistic.
	tests := []struct {
		name     string
		limit    int
		sessions int
		datasize int
	}{
		{"1KB", 1024, 1, 16},
		{"1MB", 1024 * 1024, 10, 16},
		{"10MB", 10 * 1024 * 1024, 100, 16},
		{"10MB_big", 10 * 1024 * 1024, 1000, 128},
	}

	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			store := NewMemoryEventStore(nil)
			store.SetMaxBytes(test.limit)
			ctx := context.Background()
			sessionIDs := make([]string, test.sessions)
			streamIDs := make([][3]string, test.sessions)
			for i := range sessionIDs {
				sessionIDs[i] = fmt.Sprint(i)
				for j := range 3 {
					streamIDs[i][j] = randText()
				}
			}
			payload := make([]byte, test.datasize)
			start := time.Now()
			b.ResetTimer()
			for i := range b.N {
				sessionID := sessionIDs[i%len(sessionIDs)]
				streamID := streamIDs[i%len(sessionIDs)][i%3]
				store.Append(ctx, sessionID, streamID, payload)
			}
			b.ReportMetric(float64(test.datasize)*float64(b.N)/time.Since(start).Seconds(), "bytes/s")
		})
	}
}
</content>
</file>
<file path="mcp/features.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"iter"
	"maps"
	"slices"
)

// This file contains implementations that are common to all features.
// A feature is an item provided to a peer. In the 2025-03-26 spec,
// the features are prompt, tool, resource and root.

// A featureSet is a collection of features of type T.
// Every feature has a unique ID, and the spec never mentions
// an ordering for the List calls, so what it calls a "list" is actually a set.
//
// An alternative implementation would use an ordered map, but that's probably
// not necessary as adds and removes are rare, and usually batched.
type featureSet[T any] struct {
	uniqueID   func(T) string
	features   map[string]T
	sortedKeys []string // lazily computed; nil after add or remove
}

// newFeatureSet creates a new featureSet for features of type T.
// The argument function should return the unique ID for a single feature.
func newFeatureSet[T any](uniqueIDFunc func(T) string) *featureSet[T] {
	return &featureSet[T]{
		uniqueID: uniqueIDFunc,
		features: make(map[string]T),
	}
}

// add adds each feature to the set if it is not present,
// or replaces an existing feature.
func (s *featureSet[T]) add(fs ...T) {
	for _, f := range fs {
		s.features[s.uniqueID(f)] = f
	}
	s.sortedKeys = nil
}

// remove removes all features with the given uids from the set if present,
// and returns whether any were removed.
// It is not an error to remove a nonexistent feature.
func (s *featureSet[T]) remove(uids ...string) bool {
	changed := false
	for _, uid := range uids {
		if _, ok := s.features[uid]; ok {
			changed = true
			delete(s.features, uid)
		}
	}
	if changed {
		s.sortedKeys = nil
	}
	return changed
}

// get returns the feature with the given uid.
// If there is none, it returns zero, false.
func (s *featureSet[T]) get(uid string) (T, bool) {
	t, ok := s.features[uid]
	return t, ok
}

// len returns the number of features in the set.
func (s *featureSet[T]) len() int { return len(s.features) }

// all returns an iterator over of all the features in the set
// sorted by unique ID.
func (s *featureSet[T]) all() iter.Seq[T] {
	s.sortKeys()
	return func(yield func(T) bool) {
		s.yieldFrom(0, yield)
	}
}

// above returns an iterator over features in the set whose unique IDs are
// greater than `uid`, in ascending ID order.
func (s *featureSet[T]) above(uid string) iter.Seq[T] {
	s.sortKeys()
	index, found := slices.BinarySearch(s.sortedKeys, uid)
	if found {
		index++
	}
	return func(yield func(T) bool) {
		s.yieldFrom(index, yield)
	}
}

// sortKeys is a helper that maintains a sorted list of feature IDs. It
// computes this list lazily upon its first call after a modification, or
// if it's nil.
func (s *featureSet[T]) sortKeys() {
	if s.sortedKeys != nil {
		return
	}
	s.sortedKeys = slices.Sorted(maps.Keys(s.features))
}

// yieldFrom is a helper that iterates over the features in the set,
// starting at the given index, and calls the yield function for each one.
func (s *featureSet[T]) yieldFrom(index int, yield func(T) bool) {
	for i := index; i < len(s.sortedKeys); i++ {
		if !yield(s.features[s.sortedKeys[i]]) {
			return
		}
	}
}
</content>
</file>
<file path="mcp/features_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/jsonschema-go/jsonschema"
)

type SayHiParams struct {
	Name string `json:"name"`
}

func TestFeatureSetOrder(t *testing.T) {
	toolA := &Tool{Name: "apple", Description: "apple tool"}
	toolB := &Tool{Name: "banana", Description: "banana tool"}
	toolC := &Tool{Name: "cherry", Description: "cherry tool"}

	testCases := []struct {
		tools []*Tool
		want  []*Tool
	}{
		{[]*Tool{toolA, toolB, toolC}, []*Tool{toolA, toolB, toolC}},
		{[]*Tool{toolB, toolC, toolA}, []*Tool{toolA, toolB, toolC}},
		{[]*Tool{toolA, toolC}, []*Tool{toolA, toolC}},
		{[]*Tool{toolA, toolA, toolA}, []*Tool{toolA}},
		{[]*Tool{}, nil},
	}
	for _, tc := range testCases {
		fs := newFeatureSet(func(t *Tool) string { return t.Name })
		fs.add(tc.tools...)
		got := slices.Collect(fs.all())
		if diff := cmp.Diff(got, tc.want, cmpopts.IgnoreUnexported(jsonschema.Schema{})); diff != "" {
			t.Errorf("expected %v, got %v, (-want +got):\n%s", tc.want, got, diff)
		}
	}
}

func TestFeatureSetAbove(t *testing.T) {
	toolA := &Tool{Name: "apple", Description: "apple tool"}
	toolB := &Tool{Name: "banana", Description: "banana tool"}
	toolC := &Tool{Name: "cherry", Description: "cherry tool"}

	testCases := []struct {
		tools []*Tool
		above string
		want  []*Tool
	}{
		{[]*Tool{toolA, toolB, toolC}, "apple", []*Tool{toolB, toolC}},
		{[]*Tool{toolA, toolB, toolC}, "banana", []*Tool{toolC}},
		{[]*Tool{toolA, toolB, toolC}, "cherry", nil},
	}
	for _, tc := range testCases {
		fs := newFeatureSet(func(t *Tool) string { return t.Name })
		fs.add(tc.tools...)
		got := slices.Collect(fs.above(tc.above))
		if diff := cmp.Diff(got, tc.want, cmpopts.IgnoreUnexported(jsonschema.Schema{})); diff != "" {
			t.Errorf("expected %v, got %v, (-want +got):\n%s", tc.want, got, diff)
		}
	}
}
</content>
</file>
<file path="mcp/logging.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"
)

// Logging levels.
const (
	LevelDebug     = slog.LevelDebug
	LevelInfo      = slog.LevelInfo
	LevelNotice    = (slog.LevelInfo + slog.LevelWarn) / 2
	LevelWarning   = slog.LevelWarn
	LevelError     = slog.LevelError
	LevelCritical  = slog.LevelError + 4
	LevelAlert     = slog.LevelError + 8
	LevelEmergency = slog.LevelError + 12
)

var slogToMCP = map[slog.Level]LoggingLevel{
	LevelDebug:     "debug",
	LevelInfo:      "info",
	LevelNotice:    "notice",
	LevelWarning:   "warning",
	LevelError:     "error",
	LevelCritical:  "critical",
	LevelAlert:     "alert",
	LevelEmergency: "emergency",
}

var mcpToSlog = make(map[LoggingLevel]slog.Level)

func init() {
	for sl, ml := range slogToMCP {
		mcpToSlog[ml] = sl
	}
}

func slogLevelToMCP(sl slog.Level) LoggingLevel {
	if ml, ok := slogToMCP[sl]; ok {
		return ml
	}
	return "debug" // for lack of a better idea
}

func mcpLevelToSlog(ll LoggingLevel) slog.Level {
	if sl, ok := mcpToSlog[ll]; ok {
		return sl
	}
	// TODO: is there a better default?
	return LevelDebug
}

// compareLevels behaves like [cmp.Compare] for [LoggingLevel]s.
func compareLevels(l1, l2 LoggingLevel) int {
	return cmp.Compare(mcpLevelToSlog(l1), mcpLevelToSlog(l2))
}

// LoggingHandlerOptions are options for a LoggingHandler.
type LoggingHandlerOptions struct {
	// The value for the "logger" field of logging notifications.
	LoggerName string
	// Limits the rate at which log messages are sent.
	// Excess messages are dropped.
	// If zero, there is no rate limiting.
	MinInterval time.Duration
}

// A LoggingHandler is a [slog.Handler] for MCP.
type LoggingHandler struct {
	opts LoggingHandlerOptions
	ss   *ServerSession
	// Ensures that the buffer reset is atomic with the write (see Handle).
	// A pointer so that clones share the mutex. See
	// https://github.com/golang/example/blob/master/slog-handler-guide/README.md#getting-the-mutex-right.
	mu              *sync.Mutex
	lastMessageSent time.Time // for rate-limiting
	buf             *bytes.Buffer
	handler         slog.Handler
}

// NewLoggingHandler creates a [LoggingHandler] that logs to the given [ServerSession] using a
// [slog.JSONHandler].
func NewLoggingHandler(ss *ServerSession, opts *LoggingHandlerOptions) *LoggingHandler {
	var buf bytes.Buffer
	jsonHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			// Remove level: it appears in LoggingMessageParams.
			if a.Key == slog.LevelKey {
				return slog.Attr{}
			}
			return a
		},
	})
	lh := &LoggingHandler{
		ss:      ss,
		mu:      new(sync.Mutex),
		buf:     &buf,
		handler: jsonHandler,
	}
	if opts != nil {
		lh.opts = *opts
	}
	return lh
}

// Enabled implements [slog.Handler.Enabled] by comparing level to the [ServerSession]'s level.
func (h *LoggingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	// This is also checked in ServerSession.LoggingMessage, so checking it here
	// is just an optimization that skips building the JSON.
	h.ss.mu.Lock()
	mcpLevel := h.ss.state.LogLevel
	h.ss.mu.Unlock()
	return level >= mcpLevelToSlog(mcpLevel)
}

// WithAttrs implements [slog.Handler.WithAttrs].
func (h *LoggingHandler) WithAttrs(as []slog.Attr) slog.Handler {
	h2 := *h
	h2.handler = h.handler.WithAttrs(as)
	return &h2
}

// WithGroup implements [slog.Handler.WithGroup].
func (h *LoggingHandler) WithGroup(name string) slog.Handler {
	h2 := *h
	h2.handler = h.handler.WithGroup(name)
	return &h2
}

// Handle implements [slog.Handler.Handle] by writing the Record to a JSONHandler,
// then calling [ServerSession.LoggingMessage] with the result.
func (h *LoggingHandler) Handle(ctx context.Context, r slog.Record) error {
	err := h.handle(ctx, r)
	// TODO(jba): find a way to surface the error.
	// The return value will probably be ignored.
	return err
}

func (h *LoggingHandler) handle(ctx context.Context, r slog.Record) error {
	// Observe the rate limit.
	// TODO(jba): use golang.org/x/time/rate. (We can't here because it would require adding
	// golang.org/x/time to the go.mod file.)
	h.mu.Lock()
	skip := time.Since(h.lastMessageSent) < h.opts.MinInterval
	h.mu.Unlock()
	if skip {
		return nil
	}

	var err error
	// Make the buffer reset atomic with the record write.
	// We are careful here in the unlikely event that the handler panics.
	// We don't want to hold the lock for the entire function, because Notify is
	// an I/O operation.
	// This can result in out-of-order delivery.
	func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.buf.Reset()
		err = h.handler.Handle(ctx, r)
	}()
	if err != nil {
		return err
	}

	h.mu.Lock()
	h.lastMessageSent = time.Now()
	h.mu.Unlock()

	params := &LoggingMessageParams{
		Logger: h.opts.LoggerName,
		Level:  slogLevelToMCP(r.Level),
		Data:   json.RawMessage(h.buf.Bytes()),
	}
	// We pass the argument context to Notify, even though slog.Handler.Handle's
	// documentation says not to.
	// In this case logging is a service to clients, not a means for debugging the
	// server, so we want to cancel the log message.
	return h.ss.Log(ctx, params)
}
</content>
</file>
<file path="mcp/mcp.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// The mcp package provides an SDK for writing model context protocol clients
// and servers.
//
// To get started, create either a [Client] or [Server], add features to it
// using `AddXXX` functions, and connect it to a peer using a [Transport].
//
// For example, to run a simple server on the [StdioTransport]:
//
//	server := mcp.NewServer(&mcp.Implementation{Name: "greeter"}, nil)
//
//	// Using the generic AddTool automatically populates the the input and output
//	// schema of the tool.
//	type args struct {
//		Name string `json:"name" jsonschema:"the person to greet"`
//	}
//	mcp.AddTool(server, &mcp.Tool{
//		Name:        "greet",
//		Description: "say hi",
//	}, func(ctx context.Context, req *mcp.CallToolRequest, args args) (*mcp.CallToolResult, any, error) {
//		return &mcp.CallToolResult{
//			Content: []mcp.Content{
//				&mcp.TextContent{Text: "Hi " + args.Name},
//			},
//		}, nil, nil
//	})
//
//	// Run the server on the stdio transport.
//	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
//		log.Printf("Server failed: %v", err)
//	}
//
// To connect to this server, use the [CommandTransport]:
//
//	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-client", Version: "v1.0.0"}, nil)
//	transport := &mcp.CommandTransport{Command: exec.Command("myserver")}
//	session, err := client.Connect(ctx, transport, nil)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer session.Close()
//
//	params := &mcp.CallToolParams{
//		Name:      "greet",
//		Arguments: map[string]any{"name": "you"},
//	}
//	res, err := session.CallTool(ctx, params)
//	if err != nil {
//		log.Fatalf("CallTool failed: %v", err)
//	}
//
// # Clients, servers, and sessions
//
// In this SDK, both a [Client] and [Server] may handle many concurrent
// connections. Each time a client or server is connected to a peer using a
// [Transport], it creates a new session (either a [ClientSession] or
// [ServerSession]):
//
//	Client                                                   Server
//	 ⇅                          (jsonrpc2)                     ⇅
//	ClientSession ⇄ Client Transport ⇄ Server Transport ⇄ ServerSession
//
// The session types expose an API to interact with its peer. For example,
// [ClientSession.CallTool] or [ServerSession.ListRoots].
//
// # Adding features
//
// Add MCP servers to your Client or Server using AddXXX methods (for example
// [Client.AddRoot] or [Server.AddPrompt]). If any peers are connected when
// AddXXX is called, they will receive a corresponding change notification
// (for example notifications/roots/list_changed).
//
// Adding tools is special: tools may be bound to ordinary Go functions by
// using the top-level generic [AddTool] function, which allows specifying an
// input and output type. When AddTool is used, the tool's input schema and
// output schema are automatically populated, and inputs are automatically
// validated. As a special case, if the output type is 'any', no output schema
// is generated.
//
//	func double(_ context.Context, _ *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error) {
//		return nil, Out{Answer: 2*in.Number}, nil
//	}
//	...
//	mcp.AddTool(server, &mcp.Tool{Name: "double"}, double)
package mcp
</content>
</file>
<file path="mcp/mcp_example_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp_test

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// !+lifecycle

func Example_lifecycle() {
	ctx := context.Background()

	// Create a client and server.
	// Wait for the client to initialize the session.
	client := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, nil)
	server := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.0.1"}, &mcp.ServerOptions{
		InitializedHandler: func(context.Context, *mcp.InitializedRequest) {
			fmt.Println("initialized!")
		},
	})

	// Connect the server and client using in-memory transports.
	//
	// Connect the server first so that it's ready to receive initialization
	// messages from the client.
	t1, t2 := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, t1, nil)
	if err != nil {
		log.Fatal(err)
	}
	clientSession, err := client.Connect(ctx, t2, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Now shut down the session by closing the client, and waiting for the
	// server session to end.
	if err := clientSession.Close(); err != nil {
		log.Fatal(err)
	}
	if err := serverSession.Wait(); err != nil {
		log.Fatal(err)
	}
	// Output: initialized!
}

// !-lifecycle

// !+progress

func Example_progress() {
	server := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.0.1"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "makeProgress"}, func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
		if token := req.Params.GetProgressToken(); token != nil {
			for i := range 3 {
				params := &mcp.ProgressNotificationParams{
					Message:       "frobbing widgets",
					ProgressToken: token,
					Progress:      float64(i),
					Total:         2,
				}
				req.Session.NotifyProgress(ctx, params) // ignore error
			}
		}
		return &mcp.CallToolResult{}, nil, nil
	})
	client := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, &mcp.ClientOptions{
		ProgressNotificationHandler: func(_ context.Context, req *mcp.ProgressNotificationClientRequest) {
			fmt.Printf("%s %.0f/%.0f\n", req.Params.Message, req.Params.Progress, req.Params.Total)
		},
	})
	ctx := context.Background()
	t1, t2 := mcp.NewInMemoryTransports()
	if _, err := server.Connect(ctx, t1, nil); err != nil {
		log.Fatal(err)
	}

	session, err := client.Connect(ctx, t2, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()
	if _, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "makeProgress",
		Meta: mcp.Meta{"progressToken": "abc123"},
	}); err != nil {
		log.Fatal(err)
	}
	// Output:
	// frobbing widgets 0/2
	// frobbing widgets 1/2
	// frobbing widgets 2/2
}

// !-progress

// !+cancellation

func Example_cancellation() {
	// For this example, we're going to be collecting observations from the
	// server and client.
	var clientResult, serverResult string
	var wg sync.WaitGroup
	wg.Add(2)

	// Create a server with a single slow tool.
	// When the client cancels its request, the server should observe
	// cancellation.
	server := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.0.1"}, nil)
	started := make(chan struct{}, 1) // signals that the server started handling the tool call
	mcp.AddTool(server, &mcp.Tool{Name: "slow"}, func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
		started <- struct{}{}
		defer wg.Done()
		select {
		case <-time.After(5 * time.Second):
			serverResult = "tool done"
		case <-ctx.Done():
			serverResult = "tool canceled"
		}
		return &mcp.CallToolResult{}, nil, nil
	})

	// Connect a client to the server.
	client := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, nil)
	ctx := context.Background()
	t1, t2 := mcp.NewInMemoryTransports()
	if _, err := server.Connect(ctx, t1, nil); err != nil {
		log.Fatal(err)
	}
	session, err := client.Connect(ctx, t2, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	// Make a tool call, asynchronously.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer wg.Done()
		_, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "slow"})
		clientResult = fmt.Sprintf("%v", err)
	}()

	// As soon as the server has started handling the call, cancel it from the
	// client side.
	<-started
	cancel()
	wg.Wait()

	fmt.Println(clientResult)
	fmt.Println(serverResult)
	// Output:
	// context canceled
	// tool canceled
}

// !-cancellation
</content>
</file>
<file path="mcp/mcp_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/internal/jsonrpc2"
)

type hiParams struct {
	Name string
}

// TODO(jba): after schemas are stateless (WIP), this can be a variable.
func greetTool() *Tool { return &Tool{Name: "greet", Description: "say hi"} }

func sayHi(ctx context.Context, req *CallToolRequest, args hiParams) (*CallToolResult, any, error) {
	if err := req.Session.Ping(ctx, nil); err != nil {
		return nil, nil, fmt.Errorf("ping failed: %v", err)
	}
	return &CallToolResult{Content: []Content{&TextContent{Text: "hi " + args.Name}}}, nil, nil
}

var codeReviewPrompt = &Prompt{
	Name:        "code_review",
	Description: "do a code review",
	Arguments:   []*PromptArgument{{Name: "Code", Required: true}},
}

func codReviewPromptHandler(_ context.Context, req *GetPromptRequest) (*GetPromptResult, error) {
	return &GetPromptResult{
		Description: "Code review prompt",
		Messages: []*PromptMessage{
			{Role: "user", Content: &TextContent{Text: "Please review the following code: " + req.Params.Arguments["Code"]}},
		},
	}, nil
}

func TestEndToEnd(t *testing.T) {
	ctx := context.Background()
	var ct, st Transport = NewInMemoryTransports()

	// Channels to check if notification callbacks happened.
	notificationChans := map[string]chan int{}
	for _, name := range []string{"initialized", "roots", "tools", "prompts", "resources", "progress_server", "progress_client", "resource_updated", "subscribe", "unsubscribe"} {
		notificationChans[name] = make(chan int, 1)
	}
	waitForNotification := func(t *testing.T, name string) {
		t.Helper()
		select {
		case <-notificationChans[name]:
		case <-time.After(time.Second):
			t.Fatalf("%s handler never called", name)
		}
	}

	sopts := &ServerOptions{
		InitializedHandler: func(context.Context, *InitializedRequest) {
			notificationChans["initialized"] <- 0
		},
		RootsListChangedHandler: func(context.Context, *RootsListChangedRequest) {
			notificationChans["roots"] <- 0
		},
		ProgressNotificationHandler: func(context.Context, *ProgressNotificationServerRequest) {
			notificationChans["progress_server"] <- 0
		},
		SubscribeHandler: func(context.Context, *SubscribeRequest) error {
			notificationChans["subscribe"] <- 0
			return nil
		},
		UnsubscribeHandler: func(context.Context, *UnsubscribeRequest) error {
			notificationChans["unsubscribe"] <- 0
			return nil
		},
	}
	s := NewServer(testImpl, sopts)
	AddTool(s, &Tool{
		Name:        "greet",
		Description: "say hi",
	}, sayHi)
... (1817 more lines)
</content>
</file>
<file path="mcp/prompt.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"context"
)

// A PromptHandler handles a call to prompts/get.
type PromptHandler func(context.Context, *GetPromptRequest) (*GetPromptResult, error)

type serverPrompt struct {
	prompt  *Prompt
	handler PromptHandler
}
</content>
</file>
<file path="mcp/protocol.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

// Protocol types for version 2025-06-18.
// To see the schema changes from the previous version, run:
//
//   prefix=https://raw.githubusercontent.com/modelcontextprotocol/modelcontextprotocol/refs/heads/main/schema
//   sdiff -l <(curl $prefix/2025-03-26/schema.ts) <(curl $prefix/2025/06-18/schema.ts)

import (
	"encoding/json"
	"fmt"
)

// Optional annotations for the client. The client can use annotations to inform
// how objects are used or displayed.
type Annotations struct {
	// Describes who the intended customer of this object or data is.
	//
	// It can include multiple entries to indicate content useful for multiple
	// audiences (e.g., []Role{"user", "assistant"}).
	Audience []Role `json:"audience,omitempty"`
	// The moment the resource was last modified, as an ISO 8601 formatted string.
	//
	// Should be an ISO 8601 formatted string (e.g., "2025-01-12T15:00:58Z").
	//
	// Examples: last activity timestamp in an open file, timestamp when the
	// resource was attached, etc.
	LastModified string `json:"lastModified,omitempty"`
	// Describes how important this data is for operating the server.
	//
	// A value of 1 means "most important," and indicates that the data is
	// effectively required, while 0 means "least important," and indicates that the
	// data is entirely optional.
	Priority float64 `json:"priority,omitempty"`
}

// CallToolParams is used by clients to call a tool.
type CallToolParams struct {
	// Meta is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// Name is the name of the tool to call.
	Name string `json:"name"`
	// Arguments holds the tool arguments. It can hold any value that can be
	// marshaled to JSON.
	Arguments any `json:"arguments,omitempty"`
}

// CallToolParamsRaw is passed to tool handlers on the server. Its arguments
// are not yet unmarshaled (hence "raw"), so that the handlers can perform
// unmarshaling themselves.
type CallToolParamsRaw struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// Name is the name of the tool being called.
	Name string `json:"name"`
	// Arguments is the raw arguments received over the wire from the client. It
	// is the responsibility of the tool handler to unmarshal and validate the
	// Arguments (see [AddTool]).
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// A CallToolResult is the server's response to a tool call.
//
// The [ToolHandler] and [ToolHandlerFor] handler functions return this result,
// though [ToolHandlerFor] populates much of it automatically as documented at
// each field.
type CallToolResult struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`

	// A list of content objects that represent the unstructured result of the tool
	// call.
	//
	// When using a [ToolHandlerFor] with structured output, if Content is unset
	// it will be populated with JSON text content corresponding to the
	// structured output value.
	Content []Content `json:"content"`

	// StructuredContent is an optional value that represents the structured
	// result of the tool call. It must marshal to a JSON object.
	//
	// When using a [ToolHandlerFor] with structured output, you should not
	// populate this field. It will be automatically populated with the typed Out
	// value.
	StructuredContent any `json:"structuredContent,omitempty"`

	// IsError reports whether the tool call ended in an error.
	//
	// If not set, this is assumed to be false (the call was successful).
	//
	// Any errors that originate from the tool should be reported inside the
	// Content field, with IsError set to true, not as an MCP protocol-level
	// error response. Otherwise, the LLM would not be able to see that an error
... (1065 more lines)
</content>
</file>
<file path="mcp/protocol_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"encoding/json"
	"maps"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParamsMeta(t *testing.T) {
	// Verify some properties of the Meta field of Params structs.
	// We use CallToolParams for the test, but the Meta setup of all params types
	// is identical so they should all behave the same.

	toJSON := func(x any) string {
		data, err := json.Marshal(x)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}

	meta := map[string]any{"m": 1}

	// You can set the embedded Meta field to a literal map.
	p := &CallToolParams{
		Meta: meta,
		Name: "name",
	}

	// The Meta field marshals properly when it's present.
	if g, w := toJSON(p), `{"_meta":{"m":1},"name":"name"}`; g != w {
		t.Errorf("got %s, want %s", g, w)
	}
	// ... and when it's absent.
	p2 := &CallToolParams{Name: "n"}
	if g, w := toJSON(p2), `{"name":"n"}`; g != w {
		t.Errorf("got %s, want %s", g, w)
	}

	// The GetMeta and SetMeta functions work as expected.
	if g := p.GetMeta(); !maps.Equal(g, meta) {
		t.Errorf("got %+v, want %+v", g, meta)
	}

	meta2 := map[string]any{"x": 2}
	p.SetMeta(meta2)
	if g := p.GetMeta(); !maps.Equal(g, meta2) {
		t.Errorf("got %+v, want %+v", g, meta2)
	}

	// The GetProgressToken and SetProgressToken methods work as expected.
	if g := p.GetProgressToken(); g != nil {
		t.Errorf("got %v, want nil", g)
	}

	p.SetProgressToken("t")
	if g := p.GetProgressToken(); g != "t" {
		t.Errorf("got %v, want `t`", g)
	}

	// You can set a progress token to an int, int32 or int64.
	p.SetProgressToken(int(1))
	p.SetProgressToken(int32(1))
	p.SetProgressToken(int64(1))
}

func TestCompleteReference(t *testing.T) {
	marshalTests := []struct {
		name    string
		in      CompleteReference // The Go struct to marshal
		want    string            // The expected JSON string output
		wantErr bool              // True if json.Marshal is expected to return an error
	}{
		{
			name:    "ValidPrompt",
			in:      CompleteReference{Type: "ref/prompt", Name: "my_prompt"},
			want:    `{"type":"ref/prompt","name":"my_prompt"}`,
			wantErr: false,
		},
		{
			name:    "ValidResource",
			in:      CompleteReference{Type: "ref/resource", URI: "file:///path/to/resource.txt"},
			want:    `{"type":"ref/resource","uri":"file:///path/to/resource.txt"}`,
			wantErr: false,
		},
		{
			name:    "ValidPromptEmptyName",
			in:      CompleteReference{Type: "ref/prompt", Name: ""},
			want:    `{"type":"ref/prompt"}`,
			wantErr: false,
		},
		{
			name:    "ValidResourceEmptyURI",
			in:      CompleteReference{Type: "ref/resource", URI: ""},
			want:    `{"type":"ref/resource"}`,
			wantErr: false,
		},
		// Error cases for MarshalJSON
		{
			name:    "InvalidType",
			in:      CompleteReference{Type: "ref/unknown", Name: "something"},
			wantErr: true,
		},
		{
			name:    "PromptWithURI",
			in:      CompleteReference{Type: "ref/prompt", Name: "my_prompt", URI: "unexpected_uri"},
			wantErr: true,
		},
		{
			name:    "ResourceWithName",
			in:      CompleteReference{Type: "ref/resource", URI: "my_uri", Name: "unexpected_name"},
			wantErr: true,
		},
		{
			name:    "MissingTypeField",
			in:      CompleteReference{Name: "missing"}, // Type is ""
			wantErr: true,
		},
	}

	// Define test cases specifically for Unmarshalling
	unmarshalTests := []struct {
		name    string
		in      string            // The JSON string input
		want    CompleteReference // The expected Go struct output
		wantErr bool              // True if json.Unmarshal is expected to return an error
	}{
		{
			name:    "ValidPrompt",
			in:      `{"type":"ref/prompt","name":"my_prompt"}`,
			want:    CompleteReference{Type: "ref/prompt", Name: "my_prompt"},
			wantErr: false,
		},
		{
			name:    "ValidResource",
			in:      `{"type":"ref/resource","uri":"file:///path/to/resource.txt"}`,
			want:    CompleteReference{Type: "ref/resource", URI: "file:///path/to/resource.txt"},
			wantErr: false,
		},
		// Error cases for UnmarshalJSON
		{
			name:    "UnrecognizedType",
			in:      `{"type":"ref/unknown","name":"something"}`,
			want:    CompleteReference{}, // placeholder, as unmarshal will fail
			wantErr: true,
		},
		{
			name:    "PromptWithURI",
			in:      `{"type":"ref/prompt","name":"my_prompt","uri":"unexpected_uri"}`,
			want:    CompleteReference{}, // placeholder
			wantErr: true,
		},
		{
			name:    "ResourceWithName",
			in:      `{"type":"ref/resource","uri":"my_uri","name":"unexpected_name"}`,
			want:    CompleteReference{}, // placeholder
			wantErr: true,
		},
		{
			name:    "MissingType",
			in:      `{"name":"missing"}`,
			want:    CompleteReference{}, // placeholder
			wantErr: true,
		},
		{
			name:    "InvalidJSON",
			in:      `invalid json`,
			want:    CompleteReference{}, // placeholder
			wantErr: true,                // json.Unmarshal will fail natively
		},
	}

	// Run Marshal Tests
	for _, test := range marshalTests {
		t.Run("Marshal/"+test.name, func(t *testing.T) {
			gotBytes, err := json.Marshal(&test.in)
			if (err != nil) != test.wantErr {
				t.Errorf("json.Marshal(%v) got error %v (want error %t)", test.in, err, test.wantErr)
			}
			if !test.wantErr { // Only check JSON output if marshal was expected to succeed
				if diff := cmp.Diff(test.want, string(gotBytes)); diff != "" {
					t.Errorf("json.Marshal(%v) mismatch (-want +got):\n%s", test.in, diff)
				}
			}
		})
	}

	// Run Unmarshal Tests
	for _, test := range unmarshalTests {
		t.Run("Unmarshal/"+test.name, func(t *testing.T) {
			var got CompleteReference
			err := json.Unmarshal([]byte(test.in), &got)

			if (err != nil) != test.wantErr {
				t.Errorf("json.Unmarshal(%q) got error %v (want error %t)", test.in, err, test.wantErr)
			}
			if !test.wantErr { // Only check content if unmarshal was expected to succeed
				if diff := cmp.Diff(test.want, got); diff != "" {
					t.Errorf("json.Unmarshal(%q) mismatch (-want +got):\n%s", test.in, diff)
				}
			}
		})
	}
}

func TestCompleteParams(t *testing.T) {
	// Define test cases specifically for Marshalling
	marshalTests := []struct {
		name string
		in   CompleteParams
		want string // Expected JSON output
	}{
		{
			name: "BasicPromptCompletion",
			in: CompleteParams{
				Ref: &CompleteReference{
					Type: "ref/prompt",
					Name: "my_prompt",
				},
				Argument: CompleteParamsArgument{
					Name:  "language",
					Value: "go",
				},
			},
			want: `{"argument":{"name":"language","value":"go"},"ref":{"type":"ref/prompt","name":"my_prompt"}}`,
		},
		{
			name: "ResourceCompletionRequest",
			in: CompleteParams{
				Ref: &CompleteReference{
					Type: "ref/resource",
					URI:  "file:///src/main.java",
				},
				Argument: CompleteParamsArgument{
					Name:  "class",
					Value: "MyClas",
				},
			},
			want: `{"argument":{"name":"class","value":"MyClas"},"ref":{"type":"ref/resource","uri":"file:///src/main.java"}}`,
		},
		{
			name: "PromptCompletionEmptyArgumentValue",
			in: CompleteParams{
				Ref: &CompleteReference{
					Type: "ref/prompt",
					Name: "another_prompt",
				},
				Argument: CompleteParamsArgument{
					Name:  "query",
					Value: "",
				},
			},
			want: `{"argument":{"name":"query","value":""},"ref":{"type":"ref/prompt","name":"another_prompt"}}`,
		},
		{
			name: "PromptCompletionWithContext",
			in: CompleteParams{
				Ref: &CompleteReference{
					Type: "ref/prompt",
					Name: "my_prompt",
				},
				Argument: CompleteParamsArgument{
					Name:  "language",
					Value: "go",
				},
				Context: &CompleteContext{
					Arguments: map[string]string{
						"framework": "mcp",
						"language":  "python",
					},
				},
			},
			want: `{"argument":{"name":"language","value":"go"},"context":{"arguments":{"framework":"mcp","language":"python"}},"ref":{"type":"ref/prompt","name":"my_prompt"}}`,
		},
		{
			name: "PromptCompletionEmptyContextArguments",
			in: CompleteParams{
				Ref: &CompleteReference{
					Type: "ref/prompt",
					Name: "my_prompt",
				},
				Argument: CompleteParamsArgument{
					Name:  "language",
					Value: "go",
				},
				Context: &CompleteContext{
					Arguments: map[string]string{},
				},
			},
			want: `{"argument":{"name":"language","value":"go"},"context":{},"ref":{"type":"ref/prompt","name":"my_prompt"}}`,
		},
	}

	// Define test cases specifically for Unmarshalling
	unmarshalTests := []struct {
		name string
		in   string         // JSON string input
		want CompleteParams // Expected Go struct output
	}{
		{
			name: "BasicPromptCompletion",
			in:   `{"argument":{"name":"language","value":"go"},"ref":{"type":"ref/prompt","name":"my_prompt"}}`,
			want: CompleteParams{
				Ref:      &CompleteReference{Type: "ref/prompt", Name: "my_prompt"},
				Argument: CompleteParamsArgument{Name: "language", Value: "go"},
			},
		},
		{
			name: "ResourceCompletionRequest",
			in:   `{"argument":{"name":"class","value":"MyClas"},"ref":{"type":"ref/resource","uri":"file:///src/main.java"}}`,
			want: CompleteParams{
				Ref:      &CompleteReference{Type: "ref/resource", URI: "file:///src/main.java"},
				Argument: CompleteParamsArgument{Name: "class", Value: "MyClas"},
			},
		},
		{
			name: "PromptCompletionWithContext",
			in:   `{"argument":{"name":"language","value":"go"},"context":{"arguments":{"framework":"mcp","language":"python"}},"ref":{"type":"ref/prompt","name":"my_prompt"}}`,
			want: CompleteParams{
				Ref:      &CompleteReference{Type: "ref/prompt", Name: "my_prompt"},
				Argument: CompleteParamsArgument{Name: "language", Value: "go"},
				Context: &CompleteContext{Arguments: map[string]string{
					"framework": "mcp",
					"language":  "python",
				}},
			},
		},
		{
			name: "PromptCompletionEmptyContextArguments",
			in:   `{"argument":{"name":"language","value":"go"},"context":{"arguments":{}},"ref":{"type":"ref/prompt","name":"my_prompt"}}`,
			want: CompleteParams{
				Ref:      &CompleteReference{Type: "ref/prompt", Name: "my_prompt"},
				Argument: CompleteParamsArgument{Name: "language", Value: "go"},
				Context:  &CompleteContext{Arguments: map[string]string{}},
			},
		},
		{
			name: "PromptCompletionNilContext", // JSON `null` for context
			in:   `{"argument":{"name":"language","value":"go"},"context":null,"ref":{"type":"ref/prompt","name":"my_prompt"}}`,
			want: CompleteParams{
				Ref:      &CompleteReference{Type: "ref/prompt", Name: "my_prompt"},
				Argument: CompleteParamsArgument{Name: "language", Value: "go"},
				Context:  nil, // Should unmarshal to nil pointer
			},
		},
	}

	// Run Marshal Tests
	for _, test := range marshalTests {
		t.Run("Marshal/"+test.name, func(t *testing.T) {
			got, err := json.Marshal(&test.in) // Marshal takes a pointer
			if err != nil {
				t.Fatalf("json.Marshal(CompleteParams) failed: %v", err)
			}
			if diff := cmp.Diff(test.want, string(got)); diff != "" {
				t.Errorf("CompleteParams marshal mismatch (-want +got):\n%s", diff)
			}
		})
	}

	// Run Unmarshal Tests
	for _, test := range unmarshalTests {
		t.Run("Unmarshal/"+test.name, func(t *testing.T) {
			var got CompleteParams
			if err := json.Unmarshal([]byte(test.in), &got); err != nil {
				t.Fatalf("json.Unmarshal(CompleteParams) failed: %v", err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("CompleteParams unmarshal mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCompleteResult(t *testing.T) {
	// Define test cases specifically for Marshalling
	marshalTests := []struct {
		name string
		in   CompleteResult
		want string // Expected JSON output
	}{
		{
			name: "BasicCompletionResult",
			in: CompleteResult{
				Completion: CompletionResultDetails{
					Values:  []string{"golang", "google", "goroutine"},
					Total:   10,
					HasMore: true,
				},
			},
			want: `{"completion":{"hasMore":true,"total":10,"values":["golang","google","goroutine"]}}`,
		},
		{
			name: "CompletionResultNoTotalNoHasMore",
			in: CompleteResult{
				Completion: CompletionResultDetails{
					Values:  []string{"only"},
					HasMore: false,
					Total:   0,
				},
			},
			want: `{"completion":{"values":["only"]}}`,
		},
		{
			name: "CompletionResultEmptyValues",
			in: CompleteResult{
				Completion: CompletionResultDetails{
					Values:  []string{},
					Total:   0,
					HasMore: false,
				},
			},
			want: `{"completion":{"values":[]}}`,
		},
	}

	// Define test cases specifically for Unmarshalling
	unmarshalTests := []struct {
		name string
		in   string         // JSON string input
		want CompleteResult // Expected Go struct output
	}{
		{
			name: "BasicCompletionResult",
			in:   `{"completion":{"hasMore":true,"total":10,"values":["golang","google","goroutine"]}}`,
			want: CompleteResult{
				Completion: CompletionResultDetails{
					Values:  []string{"golang", "google", "goroutine"},
					Total:   10,
					HasMore: true,
				},
			},
		},
		{
			name: "CompletionResultNoTotalNoHasMore",
			in:   `{"completion":{"values":["only"]}}`,
			want: CompleteResult{
				Completion: CompletionResultDetails{
					Values:  []string{"only"},
					HasMore: false,
					Total:   0,
				},
			},
		},
		{
			name: "CompletionResultEmptyValues",
			in:   `{"completion":{"hasMore":false,"total":0,"values":[]}}`,
			want: CompleteResult{
				Completion: CompletionResultDetails{
					Values:  []string{},
					Total:   0,
					HasMore: false,
				},
			},
		},
	}

	// Run Marshal Tests
	for _, test := range marshalTests {
		t.Run("Marshal/"+test.name, func(t *testing.T) {
			got, err := json.Marshal(&test.in) // Marshal takes a pointer
			if err != nil {
				t.Fatalf("json.Marshal(CompleteResult) failed: %v", err)
			}
			if diff := cmp.Diff(test.want, string(got)); diff != "" {
				t.Errorf("CompleteResult marshal mismatch (-want +got):\n%s", diff)
			}
		})
	}

	// Run Unmarshal Tests
	for _, test := range unmarshalTests {
		t.Run("Unmarshal/"+test.name, func(t *testing.T) {
			var got CompleteResult
			if err := json.Unmarshal([]byte(test.in), &got); err != nil {
				t.Fatalf("json.Unmarshal(CompleteResult) failed: %v", err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("CompleteResult unmarshal mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestContentUnmarshal(t *testing.T) {
	// Verify that types with a Content field round-trip properly.
	roundtrip := func(in, out any) {
		t.Helper()
		data, err := json.Marshal(in)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, out); err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(in, out, ctrCmpOpts...); diff != "" {
			t.Errorf("mismatch (-want, +got):\n%s", diff)
		}
	}

	content := []Content{&TextContent{Text: "t"}}

	ctr := &CallToolResult{
		Meta:              Meta{"m": true},
		Content:           content,
		IsError:           true,
		StructuredContent: map[string]any{"s": "x"},
	}
	var got CallToolResult
	roundtrip(ctr, &got)

	ctrf := &CallToolResult{
		Meta:              Meta{"m": true},
		Content:           content,
		IsError:           true,
		StructuredContent: 3.0,
	}
	var gotf CallToolResult
	roundtrip(ctrf, &gotf)

	pm := &PromptMessage{
		Content: content[0],
		Role:    "",
	}
	var gotpm PromptMessage
	roundtrip(pm, &gotpm)
}
</content>
</file>
<file path="mcp/requests.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// This file holds the request types.

package mcp

type (
	CallToolRequest                   = ServerRequest[*CallToolParamsRaw]
	CompleteRequest                   = ServerRequest[*CompleteParams]
	GetPromptRequest                  = ServerRequest[*GetPromptParams]
	InitializedRequest                = ServerRequest[*InitializedParams]
	ListPromptsRequest                = ServerRequest[*ListPromptsParams]
	ListResourcesRequest              = ServerRequest[*ListResourcesParams]
	ListResourceTemplatesRequest      = ServerRequest[*ListResourceTemplatesParams]
	ListToolsRequest                  = ServerRequest[*ListToolsParams]
	ProgressNotificationServerRequest = ServerRequest[*ProgressNotificationParams]
	ReadResourceRequest               = ServerRequest[*ReadResourceParams]
	RootsListChangedRequest           = ServerRequest[*RootsListChangedParams]
	SubscribeRequest                  = ServerRequest[*SubscribeParams]
	UnsubscribeRequest                = ServerRequest[*UnsubscribeParams]
)

type (
	CreateMessageRequest               = ClientRequest[*CreateMessageParams]
	ElicitRequest                      = ClientRequest[*ElicitParams]
	initializedClientRequest           = ClientRequest[*InitializedParams]
	InitializeRequest                  = ClientRequest[*InitializeParams]
	ListRootsRequest                   = ClientRequest[*ListRootsParams]
	LoggingMessageRequest              = ClientRequest[*LoggingMessageParams]
	ProgressNotificationClientRequest  = ClientRequest[*ProgressNotificationParams]
	PromptListChangedRequest           = ClientRequest[*PromptListChangedParams]
	ResourceListChangedRequest         = ClientRequest[*ResourceListChangedParams]
	ResourceUpdatedNotificationRequest = ClientRequest[*ResourceUpdatedNotificationParams]
	ToolListChangedRequest             = ClientRequest[*ToolListChangedParams]
)
</content>
</file>
<file path="mcp/resource.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/internal/jsonrpc2"
	"github.com/modelcontextprotocol/go-sdk/internal/util"
	"github.com/yosida95/uritemplate/v3"
)

// A serverResource associates a Resource with its handler.
type serverResource struct {
	resource *Resource
	handler  ResourceHandler
}

// A serverResourceTemplate associates a ResourceTemplate with its handler.
type serverResourceTemplate struct {
	resourceTemplate *ResourceTemplate
	handler          ResourceHandler
}

// A ResourceHandler is a function that reads a resource.
// It will be called when the client calls [ClientSession.ReadResource].
// If it cannot find the resource, it should return the result of calling [ResourceNotFoundError].
type ResourceHandler func(context.Context, *ReadResourceRequest) (*ReadResourceResult, error)

// ResourceNotFoundError returns an error indicating that a resource being read could
// not be found.
func ResourceNotFoundError(uri string) error {
	return &jsonrpc2.WireError{
		Code:    codeResourceNotFound,
		Message: "Resource not found",
		Data:    json.RawMessage(fmt.Sprintf(`{"uri":%q}`, uri)),
	}
}

// readFileResource reads from the filesystem at a URI relative to dirFilepath, respecting
// the roots.
// dirFilepath and rootFilepaths are absolute filesystem paths.
func readFileResource(rawURI, dirFilepath string, rootFilepaths []string) ([]byte, error) {
	uriFilepath, err := computeURIFilepath(rawURI, dirFilepath, rootFilepaths)
	if err != nil {
		return nil, err
	}

	var data []byte
	err = withFile(dirFilepath, uriFilepath, func(f *os.File) error {
		var err error
		data, err = io.ReadAll(f)
		return err
	})
	if os.IsNotExist(err) {
		err = ResourceNotFoundError(rawURI)
	}
	return data, err
}

// computeURIFilepath returns a path relative to dirFilepath.
// The dirFilepath and rootFilepaths are absolute file paths.
func computeURIFilepath(rawURI, dirFilepath string, rootFilepaths []string) (string, error) {
	// We use "file path" to mean a filesystem path.
	uri, err := url.Parse(rawURI)
	if err != nil {
		return "", err
	}
	if uri.Scheme != "file" {
		return "", fmt.Errorf("URI is not a file: %s", uri)
	}
	if uri.Path == "" {
		// A more specific error than the one below, to catch the
		// common mistake "file://foo".
		return "", errors.New("empty path")
	}
	// The URI's path is interpreted relative to dirFilepath, and in the local filesystem.
	// It must not try to escape its directory.
	uriFilepathRel, err := filepath.Localize(strings.TrimPrefix(uri.Path, "/"))
	if err != nil {
		return "", fmt.Errorf("%q cannot be localized: %w", uriFilepathRel, err)
	}

	// Check roots, if there are any.
	if len(rootFilepaths) > 0 {
		// To check against the roots, we need an absolute file path, not relative to the directory.
		// uriFilepath is local, so the joined path is under dirFilepath.
		uriFilepathAbs := filepath.Join(dirFilepath, uriFilepathRel)
		rootOK := false
		// Check that the requested file path is under some root.
		// Since both paths are absolute, that's equivalent to filepath.Rel constructing
		// a local path.
		for _, rootFilepathAbs := range rootFilepaths {
			if rel, err := filepath.Rel(rootFilepathAbs, uriFilepathAbs); err == nil && filepath.IsLocal(rel) {
				rootOK = true
				break
			}
		}
		if !rootOK {
			return "", fmt.Errorf("URI path %q is not under any root", uriFilepathAbs)
		}
	}
	return uriFilepathRel, nil
}

// fileRoots transforms the Roots obtained from the client into absolute paths on
// the local filesystem.
// TODO(jba): expose this functionality to user ResourceHandlers,
// so they don't have to repeat it.
func fileRoots(rawRoots []*Root) ([]string, error) {
	var fileRoots []string
	for _, r := range rawRoots {
		fr, err := fileRoot(r)
		if err != nil {
			return nil, err
		}
		fileRoots = append(fileRoots, fr)
	}
	return fileRoots, nil
}

// fileRoot returns the absolute path for Root.
func fileRoot(root *Root) (_ string, err error) {
	defer util.Wrapf(&err, "root %q", root.URI)

	// Convert to absolute file path.
	rurl, err := url.Parse(root.URI)
	if err != nil {
		return "", err
	}
	if rurl.Scheme != "file" {
		return "", errors.New("not a file URI")
	}
	if rurl.Path == "" {
		// A more specific error than the one below, to catch the
		// common mistake "file://foo".
		return "", errors.New("empty path")
	}
	// We don't want Localize here: we want an absolute path, which is not local.
	fileRoot := filepath.Clean(filepath.FromSlash(rurl.Path))
	if !filepath.IsAbs(fileRoot) {
		return "", errors.New("not an absolute path")
	}
	return fileRoot, nil
}

// Matches reports whether the receiver's uri template matches the uri.
func (sr *serverResourceTemplate) Matches(uri string) bool {
	tmpl, err := uritemplate.New(sr.resourceTemplate.URITemplate)
	if err != nil {
		return false
	}
	return tmpl.Regexp().MatchString(uri)
}
</content>
</file>
<file path="mcp/resource_go124.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

//go:build go1.24

package mcp

import (
	"errors"
	"os"
)

// withFile calls f on the file at join(dir, rel),
// protecting against path traversal attacks.
func withFile(dir, rel string, f func(*os.File) error) (err error) {
	r, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer r.Close()
	file, err := r.Open(rel)
	if err != nil {
		return err
	}
	// Record error, in case f writes.
	defer func() { err = errors.Join(err, file.Close()) }()
	return f(file)
}
</content>
</file>
<file path="mcp/resource_pre_go124.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

//go:build !go1.24

package mcp

import (
	"errors"
	"os"
	"path/filepath"
)

// withFile calls f on the file at join(dir, rel).
// It does not protect against path traversal attacks.
func withFile(dir, rel string, f func(*os.File) error) (err error) {
	file, err := os.Open(filepath.Join(dir, rel))
	if err != nil {
		return err
	}
	// Record error, in case f writes.
	defer func() { err = errors.Join(err, file.Close()) }()
	return f(file)
}
</content>
</file>
<file path="mcp/resource_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFileRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("TODO: fix for Windows")
	}

	for _, tt := range []struct {
		uri     string
		want    string
		wantErr string // error must contain this string
	}{
		{uri: "file:///foo", want: "/foo"},
		{uri: "file:///foo/bar", want: "/foo/bar"},
		{uri: "file:///foo/../bar", want: "/bar"},
		{uri: "file:/foo", want: "/foo"},
		{uri: "http:///foo", wantErr: "not a file"},
		{uri: "file://foo", wantErr: "empty path"},
		{uri: ":", wantErr: "missing protocol scheme"},
	} {
		got, err := fileRoot(&Root{URI: tt.uri})
		if err != nil {
			if tt.wantErr == "" {
				t.Errorf("%s: got %v, want success", tt.uri, err)
				continue
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("%s: got %v, does not contain %q", tt.uri, err, tt.wantErr)
				continue
			}
		} else if tt.wantErr != "" {
			t.Errorf("%s: succeeded, but wanted error with %q", tt.uri, tt.wantErr)
		} else if got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.uri, got, tt.want)
		}
	}
}

func TestComputeURIFilepath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("TODO: fix for Windows")
	}
	// TODO(jba): test with Windows \\host paths and C: paths
	dirFilepath := filepath.FromSlash("/files")
	rootFilepaths := []string{
		filepath.FromSlash("/files/public"),
		filepath.FromSlash("/files/shared"),
	}
	for _, tt := range []struct {
		uri     string
		want    string
		wantErr string // error must contain this
	}{
		{"file:///public", "public", ""},
		{"file:///public/file", "public/file", ""},
		{"file:///shared/file", "shared/file", ""},
		{"http:///foo", "", "not a file"},
		{"file://foo", "", "empty"},
		{"file://foo/../bar", "", "localized"},
		{"file:///secret", "", "root"},
		{"file:///secret/file", "", "root"},
		{"file:///private/file", "", "root"},
	} {
		t.Run(tt.uri, func(t *testing.T) {
			tt.want = filepath.FromSlash(tt.want) // handle Windows
			got, gotErr := computeURIFilepath(tt.uri, dirFilepath, rootFilepaths)
			if gotErr != nil {
				if tt.wantErr == "" {
					t.Fatalf("got %v, wanted success", gotErr)
				}
				if !strings.Contains(gotErr.Error(), tt.wantErr) {
					t.Fatalf("got error %v, does not contain %q", gotErr, tt.wantErr)
				}
				return
			}
			if tt.wantErr != "" {
				t.Fatal("succeeded unexpectedly")
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReadFileResource(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("TODO: fix for Windows")
	}
	abs, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	dirFilepath := filepath.Join(abs, "files")
	got, err := readFileResource("file:///info.txt", dirFilepath, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := "Contents\n"
	if g := string(got); g != want {
		t.Errorf("got %q, want %q", g, want)
	}
}

func TestTemplateMatch(t *testing.T) {
	uri := "file:///path/to/file"
	for _, tt := range []struct {
		template string
		want     bool
	}{
		{"file:///{}/{a}/{b}", false}, // invalid: empty variable expression "{}" is not allowed in RFC 6570
		{"file:///{a}/{b}", false},
		{"file:///{+path}", true},
		{"file:///{a}/{+path}", true},
	} {
		resourceTmpl := serverResourceTemplate{resourceTemplate: &ResourceTemplate{URITemplate: tt.template}}
		if matched := resourceTmpl.Matches(uri); matched != tt.want {
			t.Errorf("%s: got %t, want %t", tt.template, matched, tt.want)
		}
	}
}
</content>
</file>
<file path="mcp/server.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"iter"
	"maps"
	"net/url"
	"path/filepath"
	"reflect"
	"slices"
	"sync"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/internal/jsonrpc2"
	"github.com/modelcontextprotocol/go-sdk/internal/util"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/yosida95/uritemplate/v3"
)

// DefaultPageSize is the default for [ServerOptions.PageSize].
const DefaultPageSize = 1000

// A Server is an instance of an MCP server.
//
// Servers expose server-side MCP features, which can serve one or more MCP
// sessions by using [Server.Run].
type Server struct {
	// fixed at creation
	impl *Implementation
	opts ServerOptions

	mu                      sync.Mutex
	prompts                 *featureSet[*serverPrompt]
	tools                   *featureSet[*serverTool]
	resources               *featureSet[*serverResource]
	resourceTemplates       *featureSet[*serverResourceTemplate]
	sessions                []*ServerSession
	sendingMethodHandler_   MethodHandler
	receivingMethodHandler_ MethodHandler
	resourceSubscriptions   map[string]map[*ServerSession]bool // uri -> session -> bool
}

// ServerOptions is used to configure behavior of the server.
type ServerOptions struct {
	// Optional instructions for connected clients.
	Instructions string
	// If non-nil, called when "notifications/initialized" is received.
	InitializedHandler func(context.Context, *InitializedRequest)
	// PageSize is the maximum number of items to return in a single page for
	// list methods (e.g. ListTools).
	//
	// If zero, defaults to [DefaultPageSize].
	PageSize int
	// If non-nil, called when "notifications/roots/list_changed" is received.
	RootsListChangedHandler func(context.Context, *RootsListChangedRequest)
	// If non-nil, called when "notifications/progress" is received.
	ProgressNotificationHandler func(context.Context, *ProgressNotificationServerRequest)
	// If non-nil, called when "completion/complete" is received.
	CompletionHandler func(context.Context, *CompleteRequest) (*CompleteResult, error)
	// If non-zero, defines an interval for regular "ping" requests.
	// If the peer fails to respond to pings originating from the keepalive check,
	// the session is automatically closed.
	KeepAlive time.Duration
	// Function called when a client session subscribes to a resource.
	SubscribeHandler func(context.Context, *SubscribeRequest) error
	// Function called when a client session unsubscribes from a resource.
	UnsubscribeHandler func(context.Context, *UnsubscribeRequest) error
	// If true, advertises the prompts capability during initialization,
	// even if no prompts have been registered.
	HasPrompts bool
	// If true, advertises the resources capability during initialization,
	// even if no resources have been registered.
	HasResources bool
	// If true, advertises the tools capability during initialization,
	// even if no tools have been registered.
	HasTools bool

	// GetSessionID provides the next session ID to use for an incoming request.
	// If nil, a default randomly generated ID will be used.
	//
	// Session IDs should be globally unique across the scope of the server,
	// which may span multiple processes in the case of distributed servers.
	//
	// As a special case, if GetSessionID returns the empty string, the
	// Mcp-Session-Id header will not be set.
	GetSessionID func() string
}

// NewServer creates a new MCP server. The resulting server has no features:
// add features using the various Server.AddXXX methods, and the [AddTool] function.
... (1155 more lines)
</content>
</file>
<file path="mcp/server_example_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp_test

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"sync/atomic"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// !+prompts

func Example_prompts() {
	ctx := context.Background()

	promptHandler := func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{
			Description: "Hi prompt",
			Messages: []*mcp.PromptMessage{
				{
					Role:    "user",
					Content: &mcp.TextContent{Text: "Say hi to " + req.Params.Arguments["name"]},
				},
			},
		}, nil
	}

	// Create a server with a single prompt.
	s := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.0.1"}, nil)
	prompt := &mcp.Prompt{
		Name: "greet",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "name",
				Description: "the name of the person to greet",
				Required:    true,
			},
		},
	}
	s.AddPrompt(prompt, promptHandler)

	// Create a client.
	c := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, nil)

	// Connect the server and client.
	t1, t2 := mcp.NewInMemoryTransports()
	if _, err := s.Connect(ctx, t1, nil); err != nil {
		log.Fatal(err)
	}
	cs, err := c.Connect(ctx, t2, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer cs.Close()

	// List the prompts.
	for p, err := range cs.Prompts(ctx, nil) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(p.Name)
	}

	// Get the prompt.
	res, err := cs.GetPrompt(ctx, &mcp.GetPromptParams{
		Name:      "greet",
		Arguments: map[string]string{"name": "Pat"},
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, msg := range res.Messages {
		fmt.Println(msg.Role, msg.Content.(*mcp.TextContent).Text)
	}
	// Output:
	// greet
	// user Say hi to Pat
}

// !-prompts

// !+logging

func Example_logging() {
	ctx := context.Background()

	// Create a server.
	s := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.0.1"}, nil)

	// Create a client that displays log messages.
	done := make(chan struct{}) // solely for the example
	var nmsgs atomic.Int32
	c := mcp.NewClient(
		&mcp.Implementation{Name: "client", Version: "v0.0.1"},
		&mcp.ClientOptions{
			LoggingMessageHandler: func(_ context.Context, r *mcp.LoggingMessageRequest) {
				m := r.Params.Data.(map[string]any)
				fmt.Println(m["msg"], m["value"])
				if nmsgs.Add(1) == 2 { // number depends on logger calls below
					close(done)
				}
			},
		})

	// Connect the server and client.
	t1, t2 := mcp.NewInMemoryTransports()
	ss, err := s.Connect(ctx, t1, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ss.Close()
	cs, err := c.Connect(ctx, t2, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer cs.Close()

	// Set the minimum log level to "info".
	if err := cs.SetLoggingLevel(ctx, &mcp.SetLoggingLevelParams{Level: "info"}); err != nil {
		log.Fatal(err)
	}

	// Get a slog.Logger for the server session.
	logger := slog.New(mcp.NewLoggingHandler(ss, nil))

	// Log some things.
	logger.Info("info shows up", "value", 1)
	logger.Debug("debug doesn't show up", "value", 2)
	logger.Warn("warn shows up", "value", 3)

	// Wait for them to arrive on the client.
	// In a real application, the log messages would appear asynchronously
	// while other work was happening.
	<-done

	// Output:
	// info shows up 1
	// warn shows up 3
}

// !-logging

// !+resources
func Example_resources() {
	ctx := context.Background()

	resources := map[string]string{
		"file:///a":     "a",
		"file:///dir/x": "x",
		"file:///dir/y": "y",
	}

	handler := func(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		uri := req.Params.URI
		c, ok := resources[uri]
		if !ok {
			return nil, mcp.ResourceNotFoundError(uri)
		}
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{URI: uri, Text: c}},
		}, nil
	}

	// Create a server with a single resource.
	s := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.0.1"}, nil)
	s.AddResource(&mcp.Resource{URI: "file:///a"}, handler)
	s.AddResourceTemplate(&mcp.ResourceTemplate{URITemplate: "file:///dir/{f}"}, handler)

	// Create a client.
	c := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, nil)

	// Connect the server and client.
	t1, t2 := mcp.NewInMemoryTransports()
	if _, err := s.Connect(ctx, t1, nil); err != nil {
		log.Fatal(err)
	}
	cs, err := c.Connect(ctx, t2, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer cs.Close()

	// List resources and resource templates.
	for r, err := range cs.Resources(ctx, nil) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(r.URI)
	}
	for r, err := range cs.ResourceTemplates(ctx, nil) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(r.URITemplate)
	}

	// Read resources.
	for _, path := range []string{"a", "dir/x", "b"} {
		res, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: "file:///" + path})
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(res.Contents[0].Text)
		}
	}
	// Output:
	// file:///a
	// file:///dir/{f}
	// a
	// x
	// calling "resources/read": Resource not found
}

// !-resources
</content>
</file>
<file path="mcp/server_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"context"
	"encoding/json"
	"log"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/jsonschema-go/jsonschema"
)

type testItem struct {
	Name  string
	Value string
}

type testListParams struct {
	Cursor string
}

func (p *testListParams) cursorPtr() *string {
	return &p.Cursor
}

type testListResult struct {
	Items      []*testItem
	NextCursor string
}

func (r *testListResult) nextCursorPtr() *string {
	return &r.NextCursor
}

var allTestItems = []*testItem{
	{"alpha", "val-A"},
	{"bravo", "val-B"},
	{"charlie", "val-C"},
	{"delta", "val-D"},
	{"echo", "val-E"},
	{"foxtrot", "val-F"},
	{"golf", "val-G"},
	{"hotel", "val-H"},
	{"india", "val-I"},
	{"juliet", "val-J"},
	{"kilo", "val-K"},
}

// getCursor encodes a string input into a URL-safe base64 cursor,
// fatally logging any encoding errors.
func getCursor(input string) string {
	cursor, err := encodeCursor(input)
	if err != nil {
		log.Fatalf("encodeCursor(%s) error = %v", input, err)
	}
	return cursor
}

func TestServerPaginateBasic(t *testing.T) {
	testCases := []struct {
		name           string
		initialItems   []*testItem
		inputCursor    string
		inputPageSize  int
		wantFeatures   []*testItem
		wantNextCursor string
		wantErr        bool
	}{
		{
			name:           "FirstPage_DefaultSize_Full",
			initialItems:   allTestItems,
			inputCursor:    "",
			inputPageSize:  5,
			wantFeatures:   allTestItems[0:5],
			wantNextCursor: getCursor("echo"), // Based on last item of first page
			wantErr:        false,
		},
		{
			name:           "SecondPage_DefaultSize_Full",
			initialItems:   allTestItems,
			inputCursor:    getCursor("echo"),
			inputPageSize:  5,
			wantFeatures:   allTestItems[5:10],
			wantNextCursor: getCursor("juliet"), // Based on last item of second page
			wantErr:        false,
		},
		{
			name:           "SecondPage_DefaultSize_Full_OutOfOrder",
			initialItems:   append(allTestItems[5:], allTestItems[0:5]...),
			inputCursor:    getCursor("echo"),
			inputPageSize:  5,
			wantFeatures:   allTestItems[5:10],
			wantNextCursor: getCursor("juliet"), // Based on last item of second page
			wantErr:        false,
		},
		{
			name:           "SecondPage_DefaultSize_Full_Duplicates",
			initialItems:   append(allTestItems, allTestItems[0:5]...),
			inputCursor:    getCursor("echo"),
			inputPageSize:  5,
			wantFeatures:   allTestItems[5:10],
			wantNextCursor: getCursor("juliet"), // Based on last item of second page
			wantErr:        false,
		},
		{
			name:           "LastPage_Remaining",
			initialItems:   allTestItems,
			inputCursor:    getCursor("juliet"),
			inputPageSize:  5,
			wantFeatures:   allTestItems[10:11], // Only 1 item left
			wantNextCursor: "",                  // No more pages
			wantErr:        false,
		},
		{
			name:           "PageSize_1",
			initialItems:   allTestItems,
			inputCursor:    "",
			inputPageSize:  1,
			wantFeatures:   allTestItems[0:1],
			wantNextCursor: getCursor("alpha"),
			wantErr:        false,
		},
		{
			name:           "PageSize_All",
			initialItems:   allTestItems,
			inputCursor:    "",
			inputPageSize:  len(allTestItems), // Page size equals total
			wantFeatures:   allTestItems,
			wantNextCursor: "", // No more pages
			wantErr:        false,
		},
		{
			name:           "PageSize_LargerThanAll",
			initialItems:   allTestItems,
			inputCursor:    "",
			inputPageSize:  len(allTestItems) + 5, // Page size larger than total
			wantFeatures:   allTestItems,
			wantNextCursor: "",
			wantErr:        false,
		},
		{
			name:           "EmptySet",
			initialItems:   nil,
			inputCursor:    "",
			inputPageSize:  5,
			wantFeatures:   nil,
			wantNextCursor: "",
			wantErr:        false,
		},
		{
			name:           "InvalidCursor",
			initialItems:   allTestItems,
			inputCursor:    "not-a-valid-gob-base64-cursor",
			inputPageSize:  5,
			wantFeatures:   nil, // Should be nil for error cases
			wantNextCursor: "",
			wantErr:        true,
		},
		{
			name:           "AboveNonExistentID",
			initialItems:   allTestItems,
			inputCursor:    getCursor("dne"), // A UID that doesn't exist
			inputPageSize:  5,
			wantFeatures:   allTestItems[4:9], // Should return elements above UID.
			wantNextCursor: getCursor("india"),
			wantErr:        false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := newFeatureSet(func(t *testItem) string { return t.Name })
			fs.add(tc.initialItems...)
			params := &testListParams{Cursor: tc.inputCursor}
			gotResult, err := paginateList(fs, tc.inputPageSize, params, &testListResult{}, func(res *testListResult, items []*testItem) {
				res.Items = items
			})
			if (err != nil) != tc.wantErr {
				t.Errorf("paginateList(%s) error, got %v, wantErr %v", tc.name, err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if diff := cmp.Diff(tc.wantFeatures, gotResult.Items); diff != "" {
				t.Errorf("paginateList(%s) mismatch (-want +got):\n%s", tc.name, diff)
			}
			if tc.wantNextCursor != gotResult.NextCursor {
				t.Errorf("paginateList(%s) nextCursor, got %v, want %v", tc.name, gotResult.NextCursor, tc.wantNextCursor)
			}
		})
	}
}

func TestServerPaginateVariousPageSizes(t *testing.T) {
	fs := newFeatureSet(func(t *testItem) string { return t.Name })
	fs.add(allTestItems...)
	// Try all possible page sizes, ensuring we get the correct list of items.
	for pageSize := 1; pageSize < len(allTestItems)+1; pageSize++ {
		var gotItems []*testItem
		var nextCursor string
		wantChunks := slices.Collect(slices.Chunk(allTestItems, pageSize))
		index := 0
		// Iterate through all pages, comparing sub-slices to the paginated list.
		for {
			params := &testListParams{Cursor: nextCursor}
			gotResult, err := paginateList(fs, pageSize, params, &testListResult{}, func(res *testListResult, items []*testItem) {
				res.Items = items
			})
			if err != nil {
				t.Fatalf("paginateList() unexpected error for pageSize %d, cursor %q: %v", pageSize, nextCursor, err)
			}
			if diff := cmp.Diff(wantChunks[index], gotResult.Items); diff != "" {
				t.Errorf("paginateList mismatch (-want +got):\n%s", diff)
			}
			gotItems = append(gotItems, gotResult.Items...)
			nextCursor = gotResult.NextCursor
			if nextCursor == "" {
				break
			}
			index++
		}

		if len(gotItems) != len(allTestItems) {
			t.Fatalf("paginateList() returned %d items, want %d", len(allTestItems), len(gotItems))
		}
	}
}

func TestServerCapabilities(t *testing.T) {
	tool := &Tool{Name: "t", InputSchema: &jsonschema.Schema{Type: "object"}}
	testCases := []struct {
		name             string
		configureServer  func(s *Server)
		serverOpts       ServerOptions
		wantCapabilities *ServerCapabilities
	}{
		{
			name:            "No capabilities",
			configureServer: func(s *Server) {},
			wantCapabilities: &ServerCapabilities{
				Logging: &LoggingCapabilities{},
			},
		},
		{
			name: "With prompts",
			configureServer: func(s *Server) {
				s.AddPrompt(&Prompt{Name: "p"}, nil)
			},
			wantCapabilities: &ServerCapabilities{
				Logging: &LoggingCapabilities{},
				Prompts: &PromptCapabilities{ListChanged: true},
			},
		},
		{
			name: "With resources",
			configureServer: func(s *Server) {
				s.AddResource(&Resource{URI: "file:///r"}, nil)
			},
			wantCapabilities: &ServerCapabilities{
				Logging:   &LoggingCapabilities{},
				Resources: &ResourceCapabilities{ListChanged: true},
			},
		},
		{
			name: "With resource templates",
			configureServer: func(s *Server) {
				s.AddResourceTemplate(&ResourceTemplate{URITemplate: "file:///rt"}, nil)
			},
			wantCapabilities: &ServerCapabilities{
				Logging:   &LoggingCapabilities{},
				Resources: &ResourceCapabilities{ListChanged: true},
			},
		},
		{
			name: "With resource subscriptions",
			configureServer: func(s *Server) {
				s.AddResourceTemplate(&ResourceTemplate{URITemplate: "file:///rt"}, nil)
			},
			serverOpts: ServerOptions{
				SubscribeHandler: func(context.Context, *SubscribeRequest) error {
					return nil
				},
				UnsubscribeHandler: func(context.Context, *UnsubscribeRequest) error {
					return nil
				},
			},
			wantCapabilities: &ServerCapabilities{
				Logging:   &LoggingCapabilities{},
				Resources: &ResourceCapabilities{ListChanged: true, Subscribe: true},
			},
		},
		{
			name: "With tools",
			configureServer: func(s *Server) {
				s.AddTool(tool, nil)
			},
			wantCapabilities: &ServerCapabilities{
				Logging: &LoggingCapabilities{},
				Tools:   &ToolCapabilities{ListChanged: true},
			},
		},
		{
			name:            "With completions",
			configureServer: func(s *Server) {},
			serverOpts: ServerOptions{
				CompletionHandler: func(context.Context, *CompleteRequest) (*CompleteResult, error) {
					return nil, nil
				},
			},
			wantCapabilities: &ServerCapabilities{
				Logging:     &LoggingCapabilities{},
				Completions: &CompletionCapabilities{},
			},
		},
		{
			name: "With all capabilities",
			configureServer: func(s *Server) {
				s.AddPrompt(&Prompt{Name: "p"}, nil)
				s.AddResource(&Resource{URI: "file:///r"}, nil)
				s.AddResourceTemplate(&ResourceTemplate{URITemplate: "file:///rt"}, nil)
				s.AddTool(tool, nil)
			},
			serverOpts: ServerOptions{
				SubscribeHandler: func(context.Context, *SubscribeRequest) error {
					return nil
				},
				UnsubscribeHandler: func(context.Context, *UnsubscribeRequest) error {
					return nil
				},
				CompletionHandler: func(context.Context, *CompleteRequest) (*CompleteResult, error) {
					return nil, nil
				},
			},
			wantCapabilities: &ServerCapabilities{
				Completions: &CompletionCapabilities{},
				Logging:     &LoggingCapabilities{},
				Prompts:     &PromptCapabilities{ListChanged: true},
				Resources:   &ResourceCapabilities{ListChanged: true, Subscribe: true},
				Tools:       &ToolCapabilities{ListChanged: true},
			},
		},
		{
			name:            "With initial capabilities",
			configureServer: func(s *Server) {},
			serverOpts: ServerOptions{
				HasPrompts:   true,
				HasResources: true,
				HasTools:     true,
			},
			wantCapabilities: &ServerCapabilities{
				Logging:   &LoggingCapabilities{},
				Prompts:   &PromptCapabilities{ListChanged: true},
				Resources: &ResourceCapabilities{ListChanged: true},
				Tools:     &ToolCapabilities{ListChanged: true},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := NewServer(testImpl, &tc.serverOpts)
			tc.configureServer(server)
			gotCapabilities := server.capabilities()
			if diff := cmp.Diff(tc.wantCapabilities, gotCapabilities); diff != "" {
				t.Errorf("capabilities() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestServerAddResourceTemplate(t *testing.T) {
	tests := []struct {
		name        string
		template    string
		expectPanic bool
	}{
		{"ValidFileTemplate", "file:///{a}/{b}", false},
		{"ValidCustomScheme", "myproto:///{a}", false},
		{"EmptyVariable", "file:///{}/{b}", true},
		{"UnclosedVariable", "file:///{a", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := ResourceTemplate{URITemplate: tt.template}

			defer func() {
				if r := recover(); r != nil {
					if !tt.expectPanic {
						t.Errorf("%s: unexpected panic: %v", tt.name, r)
					}
				} else {
					if tt.expectPanic {
						t.Errorf("%s: expected panic but did not panic", tt.name)
					}
				}
			}()

			s := NewServer(testImpl, nil)
			s.AddResourceTemplate(&rt, nil)
		})
	}
}

// TestServerSessionkeepaliveCancelOverwritten is to verify that `ServerSession.keepaliveCancel` is assigned exactly once,
// ensuring that only a single goroutine is responsible for the session's keepalive ping mechanism.
func TestServerSessionkeepaliveCancelOverwritten(t *testing.T) {
	// Set KeepAlive to a long duration to ensure the keepalive
	// goroutine stays alive for the duration of the test without actually sending
	// ping requests, since we don't have a real client connection established.
	server := NewServer(testImpl, &ServerOptions{KeepAlive: 5 * time.Second})
	ss := &ServerSession{server: server}

	// 1. Initialize the session.
	_, err := ss.initialize(context.Background(), &InitializeParams{})
	if err != nil {
		t.Fatalf("ServerSession initialize failed: %v", err)
	}

	// 2. Call 'initialized' for the first time. This should start the keepalive mechanism.
	_, err = ss.initialized(context.Background(), &InitializedParams{})
	if err != nil {
		t.Fatalf("First initialized call failed: %v", err)
	}
	if ss.keepaliveCancel == nil {
		t.Fatalf("expected ServerSession.keepaliveCancel to be set after the first call of initialized")
	}

	// Save the cancel function and use defer to ensure resources are cleaned up.
	firstCancel := ss.keepaliveCancel
	defer firstCancel()

	// 3. Manually set the field to nil.
	// Do this to facilitate the test's core assertion. The goal is to verify that
	// 'ss.keepaliveCancel' is not assigned a second time. By setting it to nil,
	// we can easily check after the next call if a new keepalive goroutine was started.
	ss.keepaliveCancel = nil

	// 4. Call 'initialized' for the second time. This should return an error.
	_, err = ss.initialized(context.Background(), &InitializedParams{})
	if err == nil {
		t.Fatalf("Expected 'duplicate initialized received' error on second call, got nil")
	}

	// 5. Re-check the field to ensure it remains nil.
	// Since 'initialized' correctly returned an error and did not call
	// 'startKeepalive', the field should remain unchanged.
	if ss.keepaliveCancel != nil {
		t.Fatal("expected ServerSession.keepaliveCancel to be nil after we manually niled it and re-initialized")
	}
}

// panicks reports whether f() panics.
func panics(f func()) (b bool) {
	defer func() {
		b = recover() != nil
	}()
	f()
	return false
}

func TestAddTool(t *testing.T) {
	// AddTool should panic if In or Out are not JSON objects.
	s := NewServer(testImpl, nil)
	if !panics(func() {
		AddTool(s, &Tool{Name: "T1"}, func(context.Context, *CallToolRequest, string) (*CallToolResult, any, error) { return nil, nil, nil })
	}) {
		t.Error("bad In: expected panic")
	}
	if panics(func() {
		AddTool(s, &Tool{Name: "T2"}, func(context.Context, *CallToolRequest, map[string]any) (*CallToolResult, any, error) {
			return nil, nil, nil
		})
	}) {
		t.Error("good In: expected no panic")
	}
	if !panics(func() {
		AddTool(s, &Tool{Name: "T2"}, func(context.Context, *CallToolRequest, map[string]any) (*CallToolResult, int, error) {
			return nil, 0, nil
		})
	}) {
		t.Error("bad Out: expected panic")
	}
}

type schema = jsonschema.Schema

func testToolForSchema[In, Out any](t *testing.T, tool *Tool, in string, out Out, wantIn, wantOut any, wantErrContaining string) {
	t.Helper()
	th := func(context.Context, *CallToolRequest, In) (*CallToolResult, Out, error) {
		return nil, out, nil
	}
	gott, goth, err := toolForErr(tool, th)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(wantIn, gott.InputSchema); diff != "" {
		t.Errorf("input: mismatch (-want, +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantOut, gott.OutputSchema); diff != "" {
		t.Errorf("output: mismatch (-want, +got):\n%s", diff)
	}
	ctr := &CallToolRequest{
		Params: &CallToolParamsRaw{
			Arguments: json.RawMessage(in),
		},
	}
	result, err := goth(context.Background(), ctr)
	if wantErrContaining != "" {
		if err == nil {
			t.Errorf("got nil error, want error containing %q", wantErrContaining)
		} else {
			if !strings.Contains(err.Error(), wantErrContaining) {
				t.Errorf("got error %q, want containing %q", err, wantErrContaining)
			}
		}
	} else if err != nil {
		t.Errorf("got error %v, want no error", err)
	}

	if gott.OutputSchema != nil && err == nil && !result.IsError {
		// Check that structured content matches exactly.
		unstructured := result.Content[0].(*TextContent).Text
		structured := string(result.StructuredContent.(json.RawMessage))
		if diff := cmp.Diff(unstructured, structured); diff != "" {
			t.Errorf("Unstructured content does not match structured content exactly (-unstructured +structured):\n%s", diff)
		}
	}
}

// TODO: move this to tool_test.go
func TestToolForSchemas(t *testing.T) {
	// Validate that toolForErr handles schemas properly.
	type in struct {
		P int `json:"p,omitempty"`
	}
	type out struct {
		B bool `json:"b,omitempty"`
	}

	var (
		falseSchema = &schema{Not: &schema{}}
		inSchema    = &schema{Type: "object", AdditionalProperties: falseSchema, Properties: map[string]*schema{"p": {Type: "integer"}}}
		inSchema2   = &schema{Type: "object", AdditionalProperties: falseSchema, Properties: map[string]*schema{"p": {Type: "string"}}}
		outSchema   = &schema{Type: "object", AdditionalProperties: falseSchema, Properties: map[string]*schema{"b": {Type: "boolean"}}}
		outSchema2  = &schema{Type: "object", AdditionalProperties: falseSchema, Properties: map[string]*schema{"b": {Type: "integer"}}}
	)

	// Infer both schemas.
	testToolForSchema[in](t, &Tool{}, `{"p":3}`, out{true}, inSchema, outSchema, "")
	// Validate the input schema: expect an error if it's wrong.
	// We can't test that the output schema is validated, because it's typed.
	testToolForSchema[in](t, &Tool{}, `{"p":"x"}`, out{true}, inSchema, outSchema, `want "integer"`)
	// Ignore type any for output.
	testToolForSchema[in, any](t, &Tool{}, `{"p":3}`, 0, inSchema, nil, "")
	// Input is still validated.
	testToolForSchema[in, any](t, &Tool{}, `{"p":"x"}`, 0, inSchema, nil, `want "integer"`)
	// Tool sets input schema: that is what's used.
	testToolForSchema[in, any](t, &Tool{InputSchema: inSchema2}, `{"p":3}`, 0, inSchema2, nil, `want "string"`)
	// Tool sets output schema: that is what's used, and validation happens.
	testToolForSchema[in, any](t, &Tool{OutputSchema: outSchema2}, `{"p":3}`, out{true},
		inSchema, outSchema2, `want "integer"`)

	// Check a slightly more complicated case.
	type weatherOutput struct {
		Summary string
		AsOf    time.Time
		Source  string
	}
	testToolForSchema[any](t, &Tool{}, `{}`, weatherOutput{},
		&schema{Type: "object"},
		&schema{
			Type:                 "object",
			Required:             []string{"Summary", "AsOf", "Source"},
			AdditionalProperties: falseSchema,
			Properties: map[string]*schema{
				"Summary": {Type: "string"},
				"AsOf":    {Type: "string"},
				"Source":  {Type: "string"},
			},
		},
		"")
}
</content>
</file>
<file path="mcp/session.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

// hasSessionID is the interface which, if implemented by connections, informs
// the session about their session ID.
//
// TODO(rfindley): remove SessionID methods from connections, when it doesn't
// make sense. Or remove it from the Sessions entirely: why does it even need
// to be exposed?
type hasSessionID interface {
	SessionID() string
}

// ServerSessionState is the state of a session.
type ServerSessionState struct {
	// InitializeParams are the parameters from 'initialize'.
	InitializeParams *InitializeParams `json:"initializeParams"`

	// InitializedParams are the parameters from 'notifications/initialized'.
	InitializedParams *InitializedParams `json:"initializedParams"`

	// LogLevel is the logging level for the session.
	LogLevel LoggingLevel `json:"logLevel"`

	// TODO: resource subscriptions
}
</content>
</file>
<file path="mcp/shared.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// This file contains code shared between client and server, including
// method handler and middleware definitions.
//
// Much of this is here so that we can factor out commonalities using
// generics. If this becomes unwieldy, it can perhaps be simplified with
// reflection.

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/internal/jsonrpc2"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

const (
	// latestProtocolVersion is the latest protocol version that this version of
	// the SDK supports.
	//
	// It is the version that the client sends in the initialization request, and
	// the default version used by the server.
	latestProtocolVersion   = protocolVersion20250618
	protocolVersion20250618 = "2025-06-18"
	protocolVersion20250326 = "2025-03-26"
	protocolVersion20241105 = "2024-11-05"
)

var supportedProtocolVersions = []string{
	protocolVersion20250618,
	protocolVersion20250326,
	protocolVersion20241105,
}

// negotiatedVersion returns the effective protocol version to use, given a
// client version.
func negotiatedVersion(clientVersion string) string {
	// In general, prefer to use the clientVersion, but if we don't support the
	// client's version, use the latest version.
	//
	// This handles the case where a new spec version is released, and the SDK
	// does not support it yet.
	if !slices.Contains(supportedProtocolVersions, clientVersion) {
		return latestProtocolVersion
	}
	return clientVersion
}

// A MethodHandler handles MCP messages.
// For methods, exactly one of the return values must be nil.
// For notifications, both must be nil.
type MethodHandler func(ctx context.Context, method string, req Request) (result Result, err error)

// A Session is either a [ClientSession] or a [ServerSession].
type Session interface {
	// ID returns the session ID, or the empty string if there is none.
	ID() string

	sendingMethodInfos() map[string]methodInfo
	receivingMethodInfos() map[string]methodInfo
	sendingMethodHandler() MethodHandler
	receivingMethodHandler() MethodHandler
	getConn() *jsonrpc2.Connection
}

// Middleware is a function from [MethodHandler] to [MethodHandler].
type Middleware func(MethodHandler) MethodHandler

// addMiddleware wraps the handler in the middleware functions.
func addMiddleware(handlerp *MethodHandler, middleware []Middleware) {
	for _, m := range slices.Backward(middleware) {
		*handlerp = m(*handlerp)
	}
}

func defaultSendingMethodHandler[S Session](ctx context.Context, method string, req Request) (Result, error) {
	info, ok := req.GetSession().sendingMethodInfos()[method]
	if !ok {
		// This can be called from user code, with an arbitrary value for method.
		return nil, jsonrpc2.ErrNotHandled
	}
	// Notifications don't have results.
	if strings.HasPrefix(method, "notifications/") {
		return nil, req.GetSession().getConn().Notify(ctx, method, req.GetParams())
	}
	// Create the result to unmarshal into.
	// The concrete type of the result is the return type of the receiving function.
	res := info.newResult()
	if err := call(ctx, req.GetSession().getConn(), method, req.GetParams(), res); err != nil {
		return nil, err
	}
	return res, nil
}

// Helper method to avoid typed nil.
func orZero[T any, P *U, U any](p P) T {
	if p == nil {
		var zero T
		return zero
	}
	return any(p).(T)
}

func handleNotify(ctx context.Context, method string, req Request) error {
	mh := req.GetSession().sendingMethodHandler()
	_, err := mh(ctx, method, req)
	return err
}

func handleSend[R Result](ctx context.Context, method string, req Request) (R, error) {
	mh := req.GetSession().sendingMethodHandler()
	// mh might be user code, so ensure that it returns the right values for the jsonrpc2 protocol.
	res, err := mh(ctx, method, req)
	if err != nil {
		var z R
		return z, err
	}
	return res.(R), nil
}

// defaultReceivingMethodHandler is the initial MethodHandler for servers and clients, before being wrapped by middleware.
func defaultReceivingMethodHandler[S Session](ctx context.Context, method string, req Request) (Result, error) {
	info, ok := req.GetSession().receivingMethodInfos()[method]
	if !ok {
		// This can be called from user code, with an arbitrary value for method.
		return nil, jsonrpc2.ErrNotHandled
	}
	return info.handleMethod(ctx, method, req)
}

func handleReceive[S Session](ctx context.Context, session S, jreq *jsonrpc.Request) (Result, error) {
	info, err := checkRequest(jreq, session.receivingMethodInfos())
	if err != nil {
		return nil, err
	}
	params, err := info.unmarshalParams(jreq.Params)
	if err != nil {
		return nil, fmt.Errorf("handling '%s': %w", jreq.Method, err)
	}

	mh := session.receivingMethodHandler()
	re, _ := jreq.Extra.(*RequestExtra)
	req := info.newRequest(session, params, re)
	// mh might be user code, so ensure that it returns the right values for the jsonrpc2 protocol.
	res, err := mh(ctx, jreq.Method, req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// checkRequest checks the given request against the provided method info, to
// ensure it is a valid MCP request.
//
// If valid, the relevant method info is returned. Otherwise, a non-nil error
// is returned describing why the request is invalid.
//
// This is extracted from request handling so that it can be called in the
// transport layer to preemptively reject bad requests.
func checkRequest(req *jsonrpc.Request, infos map[string]methodInfo) (methodInfo, error) {
	info, ok := infos[req.Method]
	if !ok {
		return methodInfo{}, fmt.Errorf("%w: %q unsupported", jsonrpc2.ErrNotHandled, req.Method)
	}
	if info.flags&notification != 0 && req.IsCall() {
		return methodInfo{}, fmt.Errorf("%w: unexpected id for %q", jsonrpc2.ErrInvalidRequest, req.Method)
	}
	if info.flags&notification == 0 && !req.IsCall() {
		return methodInfo{}, fmt.Errorf("%w: missing id for %q", jsonrpc2.ErrInvalidRequest, req.Method)
	}
	// missingParamsOK is checked here to catch the common case where "params" is
	// missing entirely.
	//
	// However, it's checked again after unmarshalling to catch the rare but
	// possible case where "params" is JSON null (see https://go.dev/issue/33835).
	if info.flags&missingParamsOK == 0 && len(req.Params) == 0 {
		return methodInfo{}, fmt.Errorf("%w: missing required \"params\"", jsonrpc2.ErrInvalidRequest)
	}
	return info, nil
}

// methodInfo is information about sending and receiving a method.
type methodInfo struct {
	// flags is a collection of flags controlling how the JSONRPC method is
	// handled. See individual flag values for documentation.
	flags methodFlags
	// Unmarshal params from the wire into a Params struct.
	// Used on the receive side.
	unmarshalParams func(json.RawMessage) (Params, error)
	newRequest      func(Session, Params, *RequestExtra) Request
	// Run the code when a call to the method is received.
	// Used on the receive side.
	handleMethod MethodHandler
	// Create a pointer to a Result struct.
	// Used on the send side.
	newResult func() Result
}

// The following definitions support converting from typed to untyped method handlers.
// Type parameter meanings:
// - S: sessions
// - P: params
// - R: results

// A typedMethodHandler is like a MethodHandler, but with type information.
type (
	typedClientMethodHandler[P Params, R Result] func(context.Context, *ClientRequest[P]) (R, error)
	typedServerMethodHandler[P Params, R Result] func(context.Context, *ServerRequest[P]) (R, error)
)

type paramsPtr[T any] interface {
	*T
	Params
}

type methodFlags int

const (
	notification    methodFlags = 1 << iota // method is a notification, not request
	missingParamsOK                         // params may be missing or null
)

func newClientMethodInfo[P paramsPtr[T], R Result, T any](d typedClientMethodHandler[P, R], flags methodFlags) methodInfo {
	mi := newMethodInfo[P, R](flags)
	mi.newRequest = func(s Session, p Params, _ *RequestExtra) Request {
		r := &ClientRequest[P]{Session: s.(*ClientSession)}
		if p != nil {
			r.Params = p.(P)
		}
		return r
	}
	mi.handleMethod = MethodHandler(func(ctx context.Context, _ string, req Request) (Result, error) {
		return d(ctx, req.(*ClientRequest[P]))
	})
	return mi
}

func newServerMethodInfo[P paramsPtr[T], R Result, T any](d typedServerMethodHandler[P, R], flags methodFlags) methodInfo {
	mi := newMethodInfo[P, R](flags)
	mi.newRequest = func(s Session, p Params, re *RequestExtra) Request {
		r := &ServerRequest[P]{Session: s.(*ServerSession), Extra: re}
		if p != nil {
			r.Params = p.(P)
		}
		return r
	}
	mi.handleMethod = MethodHandler(func(ctx context.Context, _ string, req Request) (Result, error) {
		return d(ctx, req.(*ServerRequest[P]))
	})
	return mi
}

// newMethodInfo creates a methodInfo from a typedMethodHandler.
//
// If isRequest is set, the method is treated as a request rather than a
// notification.
func newMethodInfo[P paramsPtr[T], R Result, T any](flags methodFlags) methodInfo {
	return methodInfo{
		flags: flags,
		unmarshalParams: func(m json.RawMessage) (Params, error) {
			var p P
			if m != nil {
				if err := json.Unmarshal(m, &p); err != nil {
					return nil, fmt.Errorf("unmarshaling %q into a %T: %w", m, p, err)
				}
			}
			// We must check missingParamsOK here, in addition to checkRequest, to
			// catch the edge cases where "params" is set to JSON null.
			// See also https://go.dev/issue/33835.
			//
			// We need to ensure that p is non-null to guard against crashes, as our
			// internal code or externally provided handlers may assume that params
			// is non-null.
			if flags&missingParamsOK == 0 && p == nil {
				return nil, fmt.Errorf("%w: missing required \"params\"", jsonrpc2.ErrInvalidRequest)
			}
			return orZero[Params](p), nil
		},
		// newResult is used on the send side, to construct the value to unmarshal the result into.
		// R is a pointer to a result struct. There is no way to "unpointer" it without reflection.
		// TODO(jba): explore generic approaches to this, perhaps by treating R in
		// the signature as the unpointered type.
		newResult: func() Result { return reflect.New(reflect.TypeFor[R]().Elem()).Interface().(R) },
	}
}

// serverMethod is glue for creating a typedMethodHandler from a method on Server.
func serverMethod[P Params, R Result](
	f func(*Server, context.Context, *ServerRequest[P]) (R, error),
) typedServerMethodHandler[P, R] {
	return func(ctx context.Context, req *ServerRequest[P]) (R, error) {
		return f(req.Session.server, ctx, req)
	}
}

// clientMethod is glue for creating a typedMethodHandler from a method on Client.
func clientMethod[P Params, R Result](
	f func(*Client, context.Context, *ClientRequest[P]) (R, error),
) typedClientMethodHandler[P, R] {
	return func(ctx context.Context, req *ClientRequest[P]) (R, error) {
		return f(req.Session.client, ctx, req)
	}
}

// serverSessionMethod is glue for creating a typedServerMethodHandler from a method on ServerSession.
func serverSessionMethod[P Params, R Result](f func(*ServerSession, context.Context, P) (R, error)) typedServerMethodHandler[P, R] {
	return func(ctx context.Context, req *ServerRequest[P]) (R, error) {
		return f(req.GetSession().(*ServerSession), ctx, req.Params)
	}
}

// clientSessionMethod is glue for creating a typedMethodHandler from a method on ServerSession.
func clientSessionMethod[P Params, R Result](f func(*ClientSession, context.Context, P) (R, error)) typedClientMethodHandler[P, R] {
	return func(ctx context.Context, req *ClientRequest[P]) (R, error) {
		return f(req.GetSession().(*ClientSession), ctx, req.Params)
	}
}

// Error codes
const (
	codeResourceNotFound = -32002
	// The error code if the method exists and was called properly, but the peer does not support it.
	codeUnsupportedMethod = -31001
	// The error code for invalid parameters
	codeInvalidParams = -32602
)

// notifySessions calls Notify on all the sessions.
// Should be called on a copy of the peer sessions.
func notifySessions[S Session, P Params](sessions []S, method string, params P) {
	if sessions == nil {
		return
	}
	// TODO: make this timeout configurable, or call handleNotify asynchronously.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// TODO: there's a potential spec violation here, when the feature list
	// changes before the session (client or server) is initialized.
	for _, s := range sessions {
		req := newRequest(s, params)
		if err := handleNotify(ctx, method, req); err != nil {
			// TODO(jba): surface this error better
			log.Printf("calling %s: %v", method, err)
		}
	}
}

func newRequest[S Session, P Params](s S, p P) Request {
	switch s := any(s).(type) {
	case *ClientSession:
		return &ClientRequest[P]{Session: s, Params: p}
	case *ServerSession:
		return &ServerRequest[P]{Session: s, Params: p}
	default:
		panic("bad session")
	}
}

// Meta is additional metadata for requests, responses and other types.
type Meta map[string]any

// GetMeta returns metadata from a value.
func (m Meta) GetMeta() map[string]any { return m }

// SetMeta sets the metadata on a value.
func (m *Meta) SetMeta(x map[string]any) { *m = x }

const progressTokenKey = "progressToken"

func getProgressToken(p Params) any {
	return p.GetMeta()[progressTokenKey]
}

func setProgressToken(p Params, pt any) {
	switch pt.(type) {
	// Support int32 and int64 for atomic.IntNN.
	case int, int32, int64, string:
	default:
		panic(fmt.Sprintf("progress token %v is of type %[1]T, not int or string", pt))
	}
	m := p.GetMeta()
	if m == nil {
		m = map[string]any{}
	}
	m[progressTokenKey] = pt
}

// A Request is a method request with parameters and additional information, such as the session.
// Request is implemented by [*ClientRequest] and [*ServerRequest].
type Request interface {
	isRequest()
	GetSession() Session
	GetParams() Params
	// GetExtra returns the Extra field for ServerRequests, and nil for ClientRequests.
	GetExtra() *RequestExtra
}

// A ClientRequest is a request to a client.
type ClientRequest[P Params] struct {
	Session *ClientSession
	Params  P
}

// A ServerRequest is a request to a server.
type ServerRequest[P Params] struct {
	Session *ServerSession
	Params  P
	Extra   *RequestExtra
}

// RequestExtra is extra information included in requests, typically from
// the transport layer.
type RequestExtra struct {
	TokenInfo *auth.TokenInfo // bearer token info (e.g. from OAuth) if any
	Header    http.Header     // header from HTTP request, if any
}

func (*ClientRequest[P]) isRequest() {}
func (*ServerRequest[P]) isRequest() {}

func (r *ClientRequest[P]) GetSession() Session { return r.Session }
func (r *ServerRequest[P]) GetSession() Session { return r.Session }

func (r *ClientRequest[P]) GetParams() Params { return r.Params }
func (r *ServerRequest[P]) GetParams() Params { return r.Params }

func (r *ClientRequest[P]) GetExtra() *RequestExtra { return nil }
func (r *ServerRequest[P]) GetExtra() *RequestExtra { return r.Extra }

func serverRequestFor[P Params](s *ServerSession, p P) *ServerRequest[P] {
	return &ServerRequest[P]{Session: s, Params: p}
}

func clientRequestFor[P Params](s *ClientSession, p P) *ClientRequest[P] {
	return &ClientRequest[P]{Session: s, Params: p}
}

// Params is a parameter (input) type for an MCP call or notification.
type Params interface {
	// GetMeta returns metadata from a value.
	GetMeta() map[string]any
	// SetMeta sets the metadata on a value.
	SetMeta(map[string]any)

	// isParams discourages implementation of Params outside of this package.
	isParams()
}

// RequestParams is a parameter (input) type for an MCP request.
type RequestParams interface {
	Params

	// GetProgressToken returns the progress token from the params' Meta field, or nil
	// if there is none.
	GetProgressToken() any

	// SetProgressToken sets the given progress token into the params' Meta field.
	// It panics if its argument is not an int or a string.
	SetProgressToken(any)
}

// Result is a result of an MCP call.
type Result interface {
	// isResult discourages implementation of Result outside of this package.
	isResult()

	// GetMeta returns metadata from a value.
	GetMeta() map[string]any
	// SetMeta sets the metadata on a value.
	SetMeta(map[string]any)
}

// emptyResult is returned by methods that have no result, like ping.
// Those methods cannot return nil, because jsonrpc2 cannot handle nils.
type emptyResult struct{}

func (*emptyResult) isResult()               {}
func (*emptyResult) GetMeta() map[string]any { panic("should never be called") }
func (*emptyResult) SetMeta(map[string]any)  { panic("should never be called") }

type listParams interface {
	// Returns a pointer to the param's Cursor field.
	cursorPtr() *string
}

type listResult[T any] interface {
	// Returns a pointer to the param's NextCursor field.
	nextCursorPtr() *string
}

// keepaliveSession represents a session that supports keepalive functionality.
type keepaliveSession interface {
	Ping(ctx context.Context, params *PingParams) error
	Close() error
}

// startKeepalive starts the keepalive mechanism for a session.
// It assigns the cancel function to the provided cancelPtr and starts a goroutine
// that sends ping messages at the specified interval.
func startKeepalive(session keepaliveSession, interval time.Duration, cancelPtr *context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	// Assign cancel function before starting goroutine to avoid race condition.
	// We cannot return it because the caller may need to cancel during the
	// window between goroutine scheduling and function return.
	*cancelPtr = cancel

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pingCtx, pingCancel := context.WithTimeout(context.Background(), interval/2)
				err := session.Ping(pingCtx, nil)
				pingCancel()
				if err != nil {
					// Ping failed, close the session
					_ = session.Close()
					return
				}
			}
		}
	}()
}
</content>
</file>
<file path="mcp/shared_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

// TODO(v0.3.0): rewrite this test.
// func TestToolValidate(t *testing.T) {
// 	// Check that the tool returned from NewServerTool properly validates its input schema.

// 	type req struct {
// 		I int
// 		B bool
// 		S string `json:",omitempty"`
// 		P *int   `json:",omitempty"`
// 	}

// 	dummyHandler := func(context.Context, *CallToolRequest, req) (*CallToolResultFor[any], error) {
// 		return nil, nil
// 	}

// 	st, err := newServerTool(&Tool{Name: "test", Description: "test"}, dummyHandler)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	for _, tt := range []struct {
// 		desc string
// 		args map[string]any
// 		want string // error should contain this string; empty for success
// 	}{
// 		{
// 			"both required",
// 			map[string]any{"I": 1, "B": true},
// 			"",
// 		},
// 		{
// 			"optional",
// 			map[string]any{"I": 1, "B": true, "S": "foo"},
// 			"",
// 		},
// 		{
// 			"wrong type",
// 			map[string]any{"I": 1.5, "B": true},
// 			"cannot unmarshal",
// 		},
// 		{
// 			"extra property",
// 			map[string]any{"I": 1, "B": true, "C": 2},
// 			"unknown field",
// 		},
// 		{
// 			"value for pointer",
// 			map[string]any{"I": 1, "B": true, "P": 3},
// 			"",
// 		},
// 		{
// 			"null for pointer",
// 			map[string]any{"I": 1, "B": true, "P": nil},
// 			"",
// 		},
// 	} {
// 		t.Run(tt.desc, func(t *testing.T) {
// 			raw, err := json.Marshal(tt.args)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
// 			_, err = st.handler(context.Background(), &ServerRequest[*CallToolParamsFor[json.RawMessage]]{
// 				Params: &CallToolParamsFor[json.RawMessage]{Arguments: json.RawMessage(raw)},
// 			})
// 			if err == nil && tt.want != "" {
// 				t.Error("got success, wanted failure")
// 			}
// 			if err != nil {
// 				if tt.want == "" {
// 					t.Fatalf("failed with:\n%s\nwanted success", err)
// 				}
// 				if !strings.Contains(err.Error(), tt.want) {
// 					t.Fatalf("got:\n%s\nwanted to contain %q", err, tt.want)
// 				}
// 			}
// 		})
// 	}
// }

// TestNilParamsHandling tests that nil parameters don't cause panic in unmarshalParams.
// This addresses a vulnerability where missing or null parameters could crash the server.
// func TestNilParamsHandling(t *testing.T) {
// 	// Define test types for clarity
// 	type TestArgs struct {
// 		Name  string `json:"name"`
// 		Value int    `json:"value"`
// 	}

// 	// Simple test handler
// 	testHandler := func(ctx context.Context, req *ServerRequest[**GetPromptParams]) (*GetPromptResult, error) {
// 		result := "processed: " + req.Params.Arguments.Name
// 		return &CallToolResultFor[string]{StructuredContent: result}, nil
// 	}

// 	methodInfo := newServerMethodInfo(testHandler, missingParamsOK)

// 	// Helper function to test that unmarshalParams doesn't panic and handles nil gracefully
// 	mustNotPanic := func(t *testing.T, rawMsg json.RawMessage, expectNil bool) Params {
// 		t.Helper()

// 		defer func() {
// 			if r := recover(); r != nil {
// 				t.Fatalf("unmarshalParams panicked: %v", r)
// 			}
// 		}()

// 		params, err := methodInfo.unmarshalParams(rawMsg)
// 		if err != nil {
// 			t.Fatalf("unmarshalParams failed: %v", err)
// 		}

// 		if expectNil {
// 			if params != nil {
// 				t.Fatalf("Expected nil params, got %v", params)
// 			}
// 			return params
// 		}

// 		if params == nil {
// 			t.Fatal("unmarshalParams returned unexpected nil")
// 		}

// 		// Verify the result can be used safely
// 		typedParams := params.(TestParams)
// 		_ = typedParams.Name
// 		_ = typedParams.Arguments.Name
// 		_ = typedParams.Arguments.Value

// 		return params
// 	}

// 	// Test different nil parameter scenarios - with missingParamsOK flag, nil/null should return nil
// 	t.Run("missing_params", func(t *testing.T) {
// 		mustNotPanic(t, nil, true) // Expect nil with missingParamsOK flag
// 	})

// 	t.Run("explicit_null", func(t *testing.T) {
// 		mustNotPanic(t, json.RawMessage(`null`), true) // Expect nil with missingParamsOK flag
// 	})

// 	t.Run("empty_object", func(t *testing.T) {
// 		mustNotPanic(t, json.RawMessage(`{}`), false) // Empty object should create valid params
// 	})

// 	t.Run("valid_params", func(t *testing.T) {
// 		rawMsg := json.RawMessage(`{"name":"test","arguments":{"name":"hello","value":42}}`)
// 		params := mustNotPanic(t, rawMsg, false)

// 		// For valid params, also verify the values are parsed correctly
// 		typedParams := params.(TestParams)
// 		if typedParams.Name != "test" {
// 			t.Errorf("Expected name 'test', got %q", typedParams.Name)
// 		}
// 		if typedParams.Arguments.Name != "hello" {
// 			t.Errorf("Expected argument name 'hello', got %q", typedParams.Arguments.Name)
// 		}
// 		if typedParams.Arguments.Value != 42 {
// 			t.Errorf("Expected argument value 42, got %d", typedParams.Arguments.Value)
// 		}
// 	})
// }

// TestNilParamsEdgeCases tests edge cases to ensure we don't over-fix
// func TestNilParamsEdgeCases(t *testing.T) {
// 	type TestArgs struct {
// 		Name  string `json:"name"`
// 		Value int    `json:"value"`
// 	}
// 	type TestParams = *CallToolParamsFor[TestArgs]

// 	testHandler := func(context.Context, *ServerRequest[TestParams]) (*CallToolResultFor[string], error) {
// 		return &CallToolResultFor[string]{StructuredContent: "test"}, nil
// 	}

// 	methodInfo := newServerMethodInfo(testHandler, missingParamsOK)

// 	// These should fail normally, not be treated as nil params
// 	invalidCases := []json.RawMessage{
// 		json.RawMessage(""),       // empty string - should error
// 		json.RawMessage("[]"),     // array - should error
// 		json.RawMessage(`"null"`), // string "null" - should error
// 		json.RawMessage("0"),      // number - should error
// 		json.RawMessage("false"),  // boolean - should error
// 	}

// 	for i, rawMsg := range invalidCases {
// 		t.Run(fmt.Sprintf("invalid_case_%d", i), func(t *testing.T) {
// 			params, err := methodInfo.unmarshalParams(rawMsg)
// 			if err == nil && params == nil {
// 				t.Error("Should not return nil params without error")
// 			}
// 		})
// 	}

// 	// Test that methods without missingParamsOK flag properly reject nil params
// 	t.Run("reject_when_params_required", func(t *testing.T) {
// 		methodInfoStrict := newServerMethodInfo(testHandler, 0) // No missingParamsOK flag

// 		testCases := []struct {
// 			name   string
// 			params json.RawMessage
// 		}{
// 			{"nil_params", nil},
// 			{"null_params", json.RawMessage(`null`)},
// 		}

// 		for _, tc := range testCases {
// 			t.Run(tc.name, func(t *testing.T) {
// 				_, err := methodInfoStrict.unmarshalParams(tc.params)
// 				if err == nil {
// 					t.Error("Expected error for required params, got nil")
// 				}
// 				if !strings.Contains(err.Error(), "missing required \"params\"") {
// 					t.Errorf("Expected 'missing required params' error, got: %v", err)
// 				}
// 			})
// 		}
// 	})
// }
</content>
</file>
<file path="mcp/sse.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/internal/jsonrpc2"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

// This file implements support for SSE (HTTP with server-sent events)
// transport server and client.
// https://modelcontextprotocol.io/specification/2024-11-05/basic/transports
//
// The transport is simple, at least relative to the new streamable transport
// introduced in the 2025-03-26 version of the spec. In short:
//
//  1. Sessions are initiated via a hanging GET request, which streams
//     server->client messages as SSE 'message' events.
//  2. The first event in the SSE stream must be an 'endpoint' event that
//     informs the client of the session endpoint.
//  3. The client POSTs client->server messages to the session endpoint.
//
// Therefore, the each new GET request hands off its responsewriter to an
// [SSEServerTransport] type that abstracts the transport as follows:
//  - Write writes a new event to the responseWriter, or fails if the GET has
//  exited.
//  - Read reads off a message queue that is pushed to via POST requests.
//  - Close causes the hanging GET to exit.

// SSEHandler is an http.Handler that serves SSE-based MCP sessions as defined by
// the [2024-11-05 version] of the MCP spec.
//
// [2024-11-05 version]: https://modelcontextprotocol.io/specification/2024-11-05/basic/transports
type SSEHandler struct {
	getServer    func(request *http.Request) *Server
	opts         SSEOptions
	onConnection func(*ServerSession) // for testing; must not block

	mu       sync.Mutex
	sessions map[string]*SSEServerTransport
}

// SSEOptions specifies options for an [SSEHandler].
// for now, it is empty, but may be extended in future.
// https://github.com/modelcontextprotocol/go-sdk/issues/507
type SSEOptions struct{}

// NewSSEHandler returns a new [SSEHandler] that creates and manages MCP
// sessions created via incoming HTTP requests.
//
// Sessions are created when the client issues a GET request to the server,
// which must accept text/event-stream responses (server-sent events).
// For each such request, a new [SSEServerTransport] is created with a distinct
// messages endpoint, and connected to the server returned by getServer.
// The SSEHandler also handles requests to the message endpoints, by
// delegating them to the relevant server transport.
//
// The getServer function may return a distinct [Server] for each new
// request, or reuse an existing server. If it returns nil, the handler
// will return a 400 Bad Request.
func NewSSEHandler(getServer func(request *http.Request) *Server, opts *SSEOptions) *SSEHandler {
	s := &SSEHandler{
		getServer: getServer,
		sessions:  make(map[string]*SSEServerTransport),
	}

	if opts != nil {
		s.opts = *opts
	}

	return s
}

// A SSEServerTransport is a logical SSE session created through a hanging GET
// request.
//
// Use [SSEServerTransport.Connect] to initiate the flow of messages.
//
// When connected, it returns the following [Connection] implementation:
//   - Writes are SSE 'message' events to the GET response.
//   - Reads are received from POSTs to the session endpoint, via
//     [SSEServerTransport.ServeHTTP].
//   - Close terminates the hanging GET.
//
// The transport is itself an [http.Handler]. It is the caller's responsibility
// to ensure that the resulting transport serves HTTP requests on the given
// session endpoint.
//
// Each SSEServerTransport may be connected (via [Server.Connect]) at most
// once, since [SSEServerTransport.ServeHTTP] serves messages to the connected
// session.
//
// Most callers should instead use an [SSEHandler], which transparently handles
// the delegation to SSEServerTransports.
type SSEServerTransport struct {
	// Endpoint is the endpoint for this session, where the client can POST
	// messages.
	Endpoint string

	// Response is the hanging response body to the incoming GET request.
	Response http.ResponseWriter

	// incoming is the queue of incoming messages.
	// It is never closed, and by convention, incoming is non-nil if and only if
	// the transport is connected.
	incoming chan jsonrpc.Message

	// We must guard both pushes to the incoming queue and writes to the response
	// writer, because incoming POST requests are arbitrarily concurrent and we
	// need to ensure we don't write push to the queue, or write to the
	// ResponseWriter, after the session GET request exits.
	mu     sync.Mutex    // also guards writes to Response
	closed bool          // set when the stream is closed
	done   chan struct{} // closed when the connection is closed
}

// ServeHTTP handles POST requests to the transport endpoint.
func (t *SSEServerTransport) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if t.incoming == nil {
		http.Error(w, "session not connected", http.StatusInternalServerError)
		return
	}

	// Read and parse the message.
	data, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	// Optionally, we could just push the data onto a channel, and let the
	// message fail to parse when it is read. This failure seems a bit more
	// useful
	msg, err := jsonrpc2.DecodeMessage(data)
	if err != nil {
		http.Error(w, "failed to parse body", http.StatusBadRequest)
		return
	}
	if req, ok := msg.(*jsonrpc.Request); ok {
		if _, err := checkRequest(req, serverMethodInfos); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	select {
	case t.incoming <- msg:
		w.WriteHeader(http.StatusAccepted)
	case <-t.done:
		http.Error(w, "session closed", http.StatusBadRequest)
	}
}

// Connect sends the 'endpoint' event to the client.
// See [SSEServerTransport] for more details on the [Connection] implementation.
func (t *SSEServerTransport) Connect(context.Context) (Connection, error) {
	if t.incoming != nil {
		return nil, fmt.Errorf("already connected")
	}
	t.incoming = make(chan jsonrpc.Message, 100)
	t.done = make(chan struct{})
	_, err := writeEvent(t.Response, Event{
		Name: "endpoint",
		Data: []byte(t.Endpoint),
	})
	if err != nil {
		return nil, err
	}
	return &sseServerConn{t: t}, nil
}

func (h *SSEHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	sessionID := req.URL.Query().Get("sessionid")

	// TODO: consider checking Content-Type here. For now, we are lax.

	// For POST requests, the message body is a message to send to a session.
	if req.Method == http.MethodPost {
		// Look up the session.
		if sessionID == "" {
			http.Error(w, "sessionid must be provided", http.StatusBadRequest)
			return
		}
		h.mu.Lock()
		session := h.sessions[sessionID]
		h.mu.Unlock()
		if session == nil {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}

		session.ServeHTTP(w, req)
		return
	}

	if req.Method != http.MethodGet {
		http.Error(w, "invalid method", http.StatusMethodNotAllowed)
		return
	}

	// GET requests create a new session, and serve messages over SSE.

	// TODO: it's not entirely documented whether we should check Accept here.
	// Let's again be lax and assume the client will accept SSE.

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sessionID = randText()
	endpoint, err := req.URL.Parse("?sessionid=" + sessionID)
	if err != nil {
		http.Error(w, "internal error: failed to create endpoint", http.StatusInternalServerError)
		return
	}

	transport := &SSEServerTransport{Endpoint: endpoint.RequestURI(), Response: w}

	// The session is terminated when the request exits.
	h.mu.Lock()
	h.sessions[sessionID] = transport
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.sessions, sessionID)
		h.mu.Unlock()
	}()

	server := h.getServer(req)
	if server == nil {
		// The getServer argument to NewSSEHandler returned nil.
		http.Error(w, "no server available", http.StatusBadRequest)
		return
	}
	ss, err := server.Connect(req.Context(), transport, nil)
	if err != nil {
		http.Error(w, "connection failed", http.StatusInternalServerError)
		return
	}
	if h.onConnection != nil {
		h.onConnection(ss)
	}
	defer ss.Close() // close the transport when the GET exits

	select {
	case <-req.Context().Done():
	case <-transport.done:
	}
}

// sseServerConn implements the [Connection] interface for a single [SSEServerTransport].
// It hides the Connection interface from the SSEServerTransport API.
type sseServerConn struct {
	t *SSEServerTransport
}

// TODO(jba): get the session ID. (Not urgent because SSE transports have been removed from the spec.)
func (s *sseServerConn) SessionID() string { return "" }

// Read implements jsonrpc2.Reader.
func (s *sseServerConn) Read(ctx context.Context) (jsonrpc.Message, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case msg := <-s.t.incoming:
		return msg, nil
	case <-s.t.done:
		return nil, io.EOF
	}
}

// Write implements jsonrpc2.Writer.
func (s *sseServerConn) Write(ctx context.Context, msg jsonrpc.Message) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	data, err := jsonrpc2.EncodeMessage(msg)
	if err != nil {
		return err
	}

	s.t.mu.Lock()
	defer s.t.mu.Unlock()

	// Note that it is invalid to write to a ResponseWriter after ServeHTTP has
	// exited, and so we must lock around this write and check isDone, which is
	// set before the hanging GET exits.
	if s.t.closed {
		return io.EOF
	}

	_, err = writeEvent(s.t.Response, Event{Name: "message", Data: data})
	return err
}

// Close implements io.Closer, and closes the session.
//
// It must be safe to call Close more than once, as the close may
// asynchronously be initiated by either the server closing its connection, or
// by the hanging GET exiting.
func (s *sseServerConn) Close() error {
	s.t.mu.Lock()
	defer s.t.mu.Unlock()
	if !s.t.closed {
		s.t.closed = true
		close(s.t.done)
	}
	return nil
}

// An SSEClientTransport is a [Transport] that can communicate with an MCP
// endpoint serving the SSE transport defined by the 2024-11-05 version of the
// spec.
//
// https://modelcontextprotocol.io/specification/2024-11-05/basic/transports
type SSEClientTransport struct {
	// Endpoint is the SSE endpoint to connect to.
	Endpoint string

	// HTTPClient is the client to use for making HTTP requests. If nil,
	// http.DefaultClient is used.
	HTTPClient *http.Client
}

// Connect connects through the client endpoint.
func (c *SSEClientTransport) Connect(ctx context.Context) (Connection, error) {
	parsedURL, err := url.Parse(c.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint: %v", err)
	}
	req, err := http.NewRequestWithContext(ctx, "GET", c.Endpoint, nil)
	if err != nil {
		return nil, err
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	req.Header.Set("Accept", "text/event-stream")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	msgEndpoint, err := func() (*url.URL, error) {
		var evt Event
		for evt, err = range scanEvents(resp.Body) {
			break
		}
		if err != nil {
			return nil, err
		}
		if evt.Name != "endpoint" {
			return nil, fmt.Errorf("first event is %q, want %q", evt.Name, "endpoint")
		}
		raw := string(evt.Data)
		return parsedURL.Parse(raw)
	}()
	if err != nil {
		resp.Body.Close()
		return nil, fmt.Errorf("missing endpoint: %v", err)
	}

	// From here on, the stream takes ownership of resp.Body.
	s := &sseClientConn{
		client:      httpClient,
		msgEndpoint: msgEndpoint,
		incoming:    make(chan []byte, 100),
		body:        resp.Body,
		done:        make(chan struct{}),
	}

	go func() {
		defer s.Close() // close the transport when the GET exits

		for evt, err := range scanEvents(resp.Body) {
			if err != nil {
				return
			}
			select {
			case s.incoming <- evt.Data:
			case <-s.done:
				return
			}
		}
	}()

	return s, nil
}

// An sseClientConn is a logical jsonrpc2 connection that implements the client
// half of the SSE protocol:
//   - Writes are POSTS to the session endpoint.
//   - Reads are SSE 'message' events, and pushes them onto a buffered channel.
//   - Close terminates the GET request.
type sseClientConn struct {
	client      *http.Client // HTTP client to use for requests
	msgEndpoint *url.URL     // session endpoint for POSTs
	incoming    chan []byte  // queue of incoming messages

	mu     sync.Mutex
	body   io.ReadCloser // body of the hanging GET
	closed bool          // set when the stream is closed
	done   chan struct{} // closed when the stream is closed
}

// TODO(jba): get the session ID. (Not urgent because SSE transports have been removed from the spec.)
func (c *sseClientConn) SessionID() string { return "" }

func (c *sseClientConn) isDone() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

func (c *sseClientConn) Read(ctx context.Context) (jsonrpc.Message, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()

	case <-c.done:
		return nil, io.EOF

	case data := <-c.incoming:
		// TODO(rfindley): do we really need to check this? We receive from c.done above.
		if c.isDone() {
			return nil, io.EOF
		}
		msg, err := jsonrpc2.DecodeMessage(data)
		if err != nil {
			return nil, err
		}
		return msg, nil
	}
}

func (c *sseClientConn) Write(ctx context.Context, msg jsonrpc.Message) error {
	data, err := jsonrpc2.EncodeMessage(msg)
	if err != nil {
		return err
	}
	if c.isDone() {
		return io.EOF
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.msgEndpoint.String(), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("failed to write: %s", resp.Status)
	}
	return nil
}

func (c *sseClientConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		_ = c.body.Close()
		close(c.done)
	}
	return nil
}
</content>
</file>
<file path="mcp/sse_example_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type AddParams struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func Add(ctx context.Context, req *mcp.CallToolRequest, args AddParams) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("%d", args.X+args.Y)},
		},
	}, nil, nil
}

func ExampleSSEHandler() {
	server := mcp.NewServer(&mcp.Implementation{Name: "adder", Version: "v0.0.1"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "add", Description: "add two numbers"}, Add)

	handler := mcp.NewSSEHandler(func(*http.Request) *mcp.Server { return server }, nil)
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	ctx := context.Background()
	transport := &mcp.SSEClientTransport{Endpoint: httpServer.URL}
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v1.0.0"}, nil)
	cs, err := client.Connect(ctx, transport, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer cs.Close()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "add",
		Arguments: map[string]any{"x": 1, "y": 2},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res.Content[0].(*mcp.TextContent).Text)

	// Output: 3
}
</content>
</file>
<file path="mcp/sse_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSSEServer(t *testing.T) {
	for _, closeServerFirst := range []bool{false, true} {
		t.Run(fmt.Sprintf("closeServerFirst=%t", closeServerFirst), func(t *testing.T) {
			ctx := context.Background()
			server := NewServer(testImpl, nil)
			AddTool(server, &Tool{Name: "greet"}, sayHi)

			sseHandler := NewSSEHandler(func(*http.Request) *Server { return server }, nil)

			serverSessions := make(chan *ServerSession, 1)
			sseHandler.onConnection = func(ss *ServerSession) {
				select {
				case serverSessions <- ss:
				default:
				}
			}
			httpServer := httptest.NewServer(sseHandler)
			defer httpServer.Close()

			var customClientUsed int64
			customClient := &http.Client{
				Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
					atomic.AddInt64(&customClientUsed, 1)
					return http.DefaultTransport.RoundTrip(req)
				}),
			}

			clientTransport := &SSEClientTransport{
				Endpoint:   httpServer.URL,
				HTTPClient: customClient,
			}

			c := NewClient(testImpl, nil)
			cs, err := c.Connect(ctx, clientTransport, nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := cs.Ping(ctx, nil); err != nil {
				t.Fatal(err)
			}
			ss := <-serverSessions
			gotHi, err := cs.CallTool(ctx, &CallToolParams{
				Name:      "greet",
				Arguments: map[string]any{"Name": "user"},
			})
			if err != nil {
				t.Fatal(err)
			}
			wantHi := &CallToolResult{
				Content: []Content{
					&TextContent{Text: "hi user"},
				},
			}
			if diff := cmp.Diff(wantHi, gotHi, ctrCmpOpts...); diff != "" {
				t.Errorf("tools/call 'greet' mismatch (-want +got):\n%s", diff)
			}

			// Verify that customClient was used
			if atomic.LoadInt64(&customClientUsed) == 0 {
				t.Error("Expected custom HTTP client to be used, but it wasn't")
			}

			t.Run("badrequests", func(t *testing.T) {
				msgEndpoint := cs.mcpConn.(*sseClientConn).msgEndpoint.String()

				// Test some invalid data, and verify that we get 400s.
				badRequests := []struct {
					name             string
					body             string
					responseContains string
				}{
					{"not a method", `{"jsonrpc":"2.0", "method":"notamethod"}`, "not handled"},
					{"missing ID", `{"jsonrpc":"2.0", "method":"ping"}`, "missing id"},
				}
				for _, r := range badRequests {
					t.Run(r.name, func(t *testing.T) {
						resp, err := http.Post(msgEndpoint, "application/json", bytes.NewReader([]byte(r.body)))
						if err != nil {
							t.Fatal(err)
						}
						defer resp.Body.Close()
						if got, want := resp.StatusCode, http.StatusBadRequest; got != want {
							t.Errorf("Sending bad request %q: got status %d, want %d", r.body, got, want)
						}
						result, err := io.ReadAll(resp.Body)
						if err != nil {
							t.Fatalf("Reading response: %v", err)
						}
						if !bytes.Contains(result, []byte(r.responseContains)) {
							t.Errorf("Response body does not contain %q:\n%s", r.responseContains, string(result))
						}
					})
				}
			})

			// Test that closing either end of the connection terminates the other
			// end.
			if closeServerFirst {
				cs.Close()
				ss.Wait()
			} else {
				ss.Close()
				cs.Wait()
			}
		})
	}
}

// roundTripperFunc is a helper to create a custom RoundTripper
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
</content>
</file>
<file path="mcp/streamable.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/internal/jsonrpc2"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

const (
	protocolVersionHeader = "Mcp-Protocol-Version"
	sessionIDHeader       = "Mcp-Session-Id"
)

// A StreamableHTTPHandler is an http.Handler that serves streamable MCP
// sessions, as defined by the [MCP spec].
//
// [MCP spec]: https://modelcontextprotocol.io/2025/03/26/streamable-http-transport.html
type StreamableHTTPHandler struct {
	getServer func(*http.Request) *Server
	opts      StreamableHTTPOptions

	onTransportDeletion func(sessionID string) // for testing only

	mu sync.Mutex
	// TODO: we should store the ServerSession along with the transport, because
	// we need to cancel keepalive requests when closing the transport.
	transports map[string]*StreamableServerTransport // keyed by IDs (from Mcp-Session-Id header)
}

// StreamableHTTPOptions configures the StreamableHTTPHandler.
type StreamableHTTPOptions struct {
	// Stateless controls whether the session is 'stateless'.
	//
	// A stateless server does not validate the Mcp-Session-Id header, and uses a
	// temporary session with default initialization parameters. Any
	// server->client request is rejected immediately as there's no way for the
	// client to respond. Server->Client notifications may reach the client if
	// they are made in the context of an incoming request, as described in the
	// documentation for [StreamableServerTransport].
	Stateless bool

	// TODO(#148): support session retention (?)

	// JSONResponse causes streamable responses to return application/json rather
	// than text/event-stream ([§2.1.5] of the spec).
	//
	// [§2.1.5]: https://modelcontextprotocol.io/specification/2025-06-18/basic/transports#sending-messages-to-the-server
	JSONResponse bool
}

// NewStreamableHTTPHandler returns a new [StreamableHTTPHandler].
//
// The getServer function is used to create or look up servers for new
// sessions. It is OK for getServer to return the same server multiple times.
// If getServer returns nil, a 400 Bad Request will be served.
func NewStreamableHTTPHandler(getServer func(*http.Request) *Server, opts *StreamableHTTPOptions) *StreamableHTTPHandler {
	h := &StreamableHTTPHandler{
		getServer:  getServer,
		transports: make(map[string]*StreamableServerTransport),
	}
	if opts != nil {
		h.opts = *opts
	}
	return h
}

// closeAll closes all ongoing sessions.
//
// TODO(rfindley): investigate the best API for callers to configure their
// session lifecycle. (?)
//
// Should we allow passing in a session store? That would allow the handler to
// be stateless.
func (h *StreamableHTTPHandler) closeAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, s := range h.transports {
		s.connection.Close()
... (1395 more lines)
</content>
</file>
<file path="mcp/streamable_bench_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func BenchmarkStreamableServing(b *testing.B) {
	// This benchmark measures how fast we can handle a single tool on a
	// streamable server, including tool validation and stream management.
	customSchemas := map[reflect.Type]*jsonschema.Schema{
		reflect.TypeFor[Probability](): {Type: "number", Minimum: jsonschema.Ptr(0.0), Maximum: jsonschema.Ptr(1.0)},
		reflect.TypeFor[WeatherType](): {Type: "string", Enum: []any{Sunny, PartlyCloudy, Cloudy, Rainy, Snowy}},
	}
	opts := &jsonschema.ForOptions{TypeSchemas: customSchemas}
	in, err := jsonschema.For[WeatherInput](opts)
	if err != nil {
		b.Fatal(err)
	}
	out, err := jsonschema.For[WeatherOutput](opts)
	if err != nil {
		b.Fatal(err)
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.0.1"}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name:         "weather",
		InputSchema:  in,
		OutputSchema: out,
	}, WeatherTool)

	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{JSONResponse: true})
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session, err := mcp.NewClient(testImpl, nil).Connect(ctx, &mcp.StreamableClientTransport{Endpoint: httpServer.URL}, nil)
	if err != nil {
		b.Fatal(err)
	}
	defer session.Close()
	b.ResetTimer()
	for range b.N {
		if _, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "weather",
			Arguments: WeatherInput{
				Location: Location{Name: "somewhere"},
				Days:     7,
			},
		}); err != nil {
			b.Errorf("CallTool failed: %v", err)
		}
	}
}
</content>
</file>
<file path="mcp/streamable_client_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/modelcontextprotocol/go-sdk/internal/jsonrpc2"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

type streamableRequestKey struct {
	httpMethod    string // http method
	sessionID     string // session ID header
	jsonrpcMethod string // jsonrpc method, or "" for non-requests
}

type header map[string]string

type streamableResponse struct {
	header              header // response headers
	status              int    // or http.StatusOK
	body                string // or ""
	optional            bool   // if set, request need not be sent
	wantProtocolVersion string // if "", unchecked
	callback            func() // if set, called after the request is handled
}

type fakeResponses map[streamableRequestKey]*streamableResponse

type fakeStreamableServer struct {
	t         *testing.T
	responses fakeResponses

	callMu sync.Mutex
	calls  map[streamableRequestKey]int
}

func (s *fakeStreamableServer) missingRequests() []streamableRequestKey {
	s.callMu.Lock()
	defer s.callMu.Unlock()

	var unused []streamableRequestKey
	for k, resp := range s.responses {
		if s.calls[k] == 0 && !resp.optional {
			unused = append(unused, k)
		}
	}
	return unused
}

func (s *fakeStreamableServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	key := streamableRequestKey{
		httpMethod: req.Method,
		sessionID:  req.Header.Get(sessionIDHeader),
	}
	if req.Method == http.MethodPost {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			s.t.Errorf("failed to read body: %v", err)
			http.Error(w, "failed to read body", http.StatusInternalServerError)
			return
		}
		msg, err := jsonrpc.DecodeMessage(body)
		if err != nil {
			s.t.Errorf("invalid body: %v", err)
			http.Error(w, "invalid body", http.StatusInternalServerError)
			return
		}
		if r, ok := msg.(*jsonrpc.Request); ok {
			key.jsonrpcMethod = r.Method
		}
	}

	s.callMu.Lock()
	if s.calls == nil {
		s.calls = make(map[streamableRequestKey]int)
	}
	s.calls[key]++
	s.callMu.Unlock()

	resp, ok := s.responses[key]
	if !ok {
		s.t.Errorf("missing response for %v", key)
		http.Error(w, "no response", http.StatusInternalServerError)
		return
	}
	if resp.callback != nil {
		defer resp.callback()
	}
	for k, v := range resp.header {
		w.Header().Set(k, v)
	}
	status := resp.status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)

	if v := req.Header.Get(protocolVersionHeader); v != resp.wantProtocolVersion && resp.wantProtocolVersion != "" {
		s.t.Errorf("%v: bad protocol version header: got %q, want %q", key, v, resp.wantProtocolVersion)
	}
	w.Write([]byte(resp.body))
}

var (
	initResult = &InitializeResult{
		Capabilities: &ServerCapabilities{
			Completions: &CompletionCapabilities{},
			Logging:     &LoggingCapabilities{},
			Tools:       &ToolCapabilities{ListChanged: true},
		},
		ProtocolVersion: latestProtocolVersion,
		ServerInfo:      &Implementation{Name: "testServer", Version: "v1.0.0"},
	}
	initResp = resp(1, initResult, nil)
)

func jsonBody(t *testing.T, msg jsonrpc2.Message) string {
	data, err := jsonrpc2.EncodeMessage(msg)
	if err != nil {
		t.Fatalf("encoding failed: %v", err)
	}
	return string(data)
}

func TestStreamableClientTransportLifecycle(t *testing.T) {
	ctx := context.Background()

	// The lifecycle test verifies various behavior of the streamable client
	// initialization:
	//  - check that it can handle application/json responses
	//  - check that it sends the negotiated protocol version
	fake := &fakeStreamableServer{
		t: t,
		responses: fakeResponses{
			{"POST", "", methodInitialize}: {
				header: header{
					"Content-Type":  "application/json",
					sessionIDHeader: "123",
				},
				body: jsonBody(t, initResp),
			},
			{"POST", "123", notificationInitialized}: {
				status:              http.StatusAccepted,
				wantProtocolVersion: latestProtocolVersion,
			},
			{"GET", "123", ""}: {
				header: header{
					"Content-Type": "text/event-stream",
				},
				optional:            true,
				wantProtocolVersion: latestProtocolVersion,
			},
			{"DELETE", "123", ""}: {},
		},
	}

	httpServer := httptest.NewServer(fake)
	defer httpServer.Close()

	transport := &StreamableClientTransport{Endpoint: httpServer.URL}
	client := NewClient(testImpl, nil)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("client.Connect() failed: %v", err)
	}
	if err := session.Close(); err != nil {
		t.Errorf("closing session: %v", err)
	}
	if missing := fake.missingRequests(); len(missing) > 0 {
		t.Errorf("did not receive expected requests: %v", missing)
	}
	if diff := cmp.Diff(initResult, session.state.InitializeResult); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func TestStreamableClientRedundantDelete(t *testing.T) {
	ctx := context.Background()

	// The lifecycle test verifies various behavior of the streamable client
	// initialization:
	//  - check that it can handle application/json responses
	//  - check that it sends the negotiated protocol version
	fake := &fakeStreamableServer{
		t: t,
		responses: fakeResponses{
			{"POST", "", methodInitialize}: {
				header: header{
					"Content-Type":  "application/json",
					sessionIDHeader: "123",
				},
				body: jsonBody(t, initResp),
			},
			{"POST", "123", notificationInitialized}: {
				status:              http.StatusAccepted,
				wantProtocolVersion: latestProtocolVersion,
			},
			{"GET", "123", ""}: {
				status:   http.StatusMethodNotAllowed,
				optional: true,
			},
			{"POST", "123", methodListTools}: {
				status: http.StatusNotFound,
			},
		},
	}

	httpServer := httptest.NewServer(fake)
	defer httpServer.Close()

	transport := &StreamableClientTransport{Endpoint: httpServer.URL}
	client := NewClient(testImpl, nil)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("client.Connect() failed: %v", err)
	}
	_, err = session.ListTools(ctx, nil)
	if err == nil {
		t.Errorf("Listing tools: got nil error, want non-nil")
	}
	_ = session.Wait() // must not hang
	if missing := fake.missingRequests(); len(missing) > 0 {
		t.Errorf("did not receive expected requests: %v", missing)
	}
}

func TestStreamableClientGETHandling(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		status              int
		wantErrorContaining string
	}{
		{http.StatusOK, ""},
		{http.StatusMethodNotAllowed, ""},
		{http.StatusBadRequest, "hanging GET"},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("status=%d", test.status), func(t *testing.T) {
			fake := &fakeStreamableServer{
				t: t,
				responses: fakeResponses{
					{"POST", "", methodInitialize}: {
						header: header{
							"Content-Type":  "application/json; charset=utf-8", // should ignore the charset
							sessionIDHeader: "123",
						},
						body: jsonBody(t, initResp),
					},
					{"POST", "123", notificationInitialized}: {
						status:              http.StatusAccepted,
						wantProtocolVersion: latestProtocolVersion,
					},
					{"GET", "123", ""}: {
						header: header{
							"Content-Type": "text/event-stream",
						},
						status:              test.status,
						wantProtocolVersion: latestProtocolVersion,
					},
					{"POST", "123", methodListTools}: {
						header: header{
							"Content-Type":  "application/json",
							sessionIDHeader: "123",
						},
						body:     jsonBody(t, resp(2, &ListToolsResult{Tools: []*Tool{}}, nil)),
						optional: true,
					},
					{"DELETE", "123", ""}: {optional: true},
				},
			}
			httpServer := httptest.NewServer(fake)
			defer httpServer.Close()

			transport := &StreamableClientTransport{Endpoint: httpServer.URL}
			client := NewClient(testImpl, nil)
			session, err := client.Connect(ctx, transport, nil)
			if err != nil {
				t.Fatalf("client.Connect() failed: %v", err)
			}

			// Since we need the client to observe the result of the hanging GET,
			// wait for all requests to be handled.
			start := time.Now()
			delay := 1 * time.Millisecond
			for range 10 {
				if len(fake.missingRequests()) == 0 {
					break
				}
				time.Sleep(delay)
				delay *= 2
			}
			if missing := fake.missingRequests(); len(missing) > 0 {
				t.Errorf("did not receive expected requests after %s: %v", time.Since(start), missing)
			}

			_, err = session.ListTools(ctx, nil)
			if (err != nil) != (test.wantErrorContaining != "") {
				t.Errorf("After initialization, got error %v, want containing %q", err, test.wantErrorContaining)
			} else if err != nil {
				if !strings.Contains(err.Error(), test.wantErrorContaining) {
					t.Errorf("After initialization, got error %s, want containing %q", err, test.wantErrorContaining)
				}
			}

			if err := session.Close(); err != nil {
				t.Errorf("closing session: %v", err)
			}
		})
	}
}

func TestStreamableClientStrictness(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		label             string
		strict            bool
		initializedStatus int
		getStatus         int
		wantConnectError  bool
		wantListError     bool
	}{
		{"conformant server", true, http.StatusAccepted, http.StatusMethodNotAllowed, false, false},
		{"strict initialized", true, http.StatusOK, http.StatusMethodNotAllowed, true, false},
		{"unstrict initialized", false, http.StatusOK, http.StatusMethodNotAllowed, false, false},
		{"strict GET", true, http.StatusAccepted, http.StatusNotFound, false, true},
		{"unstrict GET", false, http.StatusOK, http.StatusNotFound, false, false},
	}
	for _, test := range tests {
		t.Run(test.label, func(t *testing.T) {
			fake := &fakeStreamableServer{
				t: t,
				responses: fakeResponses{
					{"POST", "", methodInitialize}: {
						header: header{
							"Content-Type":  "application/json",
							sessionIDHeader: "123",
						},
						body: jsonBody(t, initResp),
					},
					{"POST", "123", notificationInitialized}: {
						status:              test.initializedStatus,
						wantProtocolVersion: latestProtocolVersion,
					},
					{"GET", "123", ""}: {
						header: header{
							"Content-Type": "text/event-stream",
						},
						status:              test.getStatus,
						wantProtocolVersion: latestProtocolVersion,
					},
					{"POST", "123", methodListTools}: {
						header: header{
							"Content-Type":  "application/json",
							sessionIDHeader: "123",
						},
						body:     jsonBody(t, resp(2, &ListToolsResult{Tools: []*Tool{}}, nil)),
						optional: true,
					},
					{"DELETE", "123", ""}: {optional: true},
				},
			}
			httpServer := httptest.NewServer(fake)
			defer httpServer.Close()

			transport := &StreamableClientTransport{Endpoint: httpServer.URL, strict: test.strict}
			client := NewClient(testImpl, nil)
			session, err := client.Connect(ctx, transport, nil)
			if (err != nil) != test.wantConnectError {
				t.Errorf("client.Connect() returned error %v; want error: %t", err, test.wantConnectError)
			}
			if err != nil {
				return
			}
			// Since we need the client to observe the result of the hanging GET,
			// wait for all requests to be handled.
			start := time.Now()
			delay := 1 * time.Millisecond
			for range 10 {
				if len(fake.missingRequests()) == 0 {
					break
				}
				time.Sleep(delay)
				delay *= 2
			}
			if missing := fake.missingRequests(); len(missing) > 0 {
				t.Errorf("did not receive expected requests after %s: %v", time.Since(start), missing)
			}
			_, err = session.ListTools(ctx, nil)
			if (err != nil) != test.wantListError {
				t.Errorf("ListTools returned error %v; want error: %t", err, test.wantListError)
			}
			if err := session.Close(); err != nil {
				t.Errorf("closing session: %v", err)
			}
		})
	}
}
</content>
</file>
<file path="mcp/streamable_example_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp_test

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// !+streamablehandler

func ExampleStreamableHTTPHandler() {
	// Create a new streamable handler, using the same MCP server for every request.
	//
	// Here, we configure it to serves application/json responses rather than
	// text/event-stream, just so the output below doesn't use random event ids.
	server := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.1.0"}, nil)
	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{JSONResponse: true})
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	// The SDK is currently permissive of some missing keys in "params".
	resp := mustPostMessage(`{"jsonrpc": "2.0", "id": 1, "method":"initialize", "params": {}}`, httpServer.URL)
	fmt.Println(resp)
	// Output:
	// {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"logging":{}},"protocolVersion":"2025-06-18","serverInfo":{"name":"server","version":"v0.1.0"}}}
}

// !-streamablehandler

// !+httpmiddleware

func ExampleStreamableHTTPHandler_middleware() {
	server := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.1.0"}, nil)
	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, nil)
	loggingHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Example debugging; you could also capture the response.
		body, err := io.ReadAll(req.Body)
		if err != nil {
			log.Fatal(err)
		}
		req.Body.Close() // ignore error
		req.Body = io.NopCloser(bytes.NewBuffer(body))
		fmt.Println(req.Method, string(body))
		handler.ServeHTTP(w, req)
	})
	httpServer := httptest.NewServer(loggingHandler)
	defer httpServer.Close()

	// The SDK is currently permissive of some missing keys in "params".
	mustPostMessage(`{"jsonrpc": "2.0", "id": 1, "method":"initialize", "params": {}}`, httpServer.URL)
	// Output:
	// POST {"jsonrpc": "2.0", "id": 1, "method":"initialize", "params": {}}
}

// !-httpmiddleware

func mustPostMessage(msg, url string) string {
	req := orFatal(http.NewRequest("POST", url, strings.NewReader(msg)))
	req.Header["Content-Type"] = []string{"application/json"}
	req.Header["Accept"] = []string{"application/json", "text/event-stream"}
	resp := orFatal(http.DefaultClient.Do(req))
	defer resp.Body.Close()
	body := orFatal(io.ReadAll(resp.Body))
	return string(body)
}

func orFatal[T any](t T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return t
}
</content>
</file>
<file path="mcp/streamable_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/internal/jsonrpc2"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

func TestStreamableTransports(t *testing.T) {
	// This test checks that the streamable server and client transports can
	// communicate.

	ctx := context.Background()

	for _, useJSON := range []bool{false, true} {
		t.Run(fmt.Sprintf("JSONResponse=%v", useJSON), func(t *testing.T) {
			// Create a server with some simple tools.
			server := NewServer(testImpl, nil)
			AddTool(server, &Tool{Name: "greet", Description: "say hi"}, sayHi)
			// The "hang" tool checks that context cancellation is propagated.
			// It hangs until the context is cancelled.
			var (
				start     = make(chan struct{})
				cancelled = make(chan struct{}, 1) // don't block the request
			)
			hang := func(ctx context.Context, req *CallToolRequest, args any) (*CallToolResult, any, error) {
				start <- struct{}{}
				select {
				case <-ctx.Done():
					cancelled <- struct{}{}
				case <-time.After(5 * time.Second):
					return nil, nil, nil
				}
				return nil, nil, nil
			}
			AddTool(server, &Tool{Name: "hang"}, hang)
			AddTool(server, &Tool{Name: "sample"}, func(ctx context.Context, req *CallToolRequest, args any) (*CallToolResult, any, error) {
				// Test that we can make sampling requests during tool handling.
				//
				// Try this on both the request context and a background context, so
				// that messages may be delivered on either the POST or GET connection.
				for _, ctx := range map[string]context.Context{
					"request context":    ctx,
					"background context": context.Background(),
				} {
					res, err := req.Session.CreateMessage(ctx, &CreateMessageParams{})
					if err != nil {
						return nil, nil, err
					}
					if g, w := res.Model, "aModel"; g != w {
						return nil, nil, fmt.Errorf("got %q, want %q", g, w)
					}
				}
				return &CallToolResult{}, nil, nil
			})

			// Start an httptest.Server with the StreamableHTTPHandler, wrapped in a
			// cookie-checking middleware.
			handler := NewStreamableHTTPHandler(func(req *http.Request) *Server { return server }, &StreamableHTTPOptions{
				JSONResponse: useJSON,
			})

			var (
				headerMu   sync.Mutex
				lastHeader http.Header
			)
			httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				headerMu.Lock()
				lastHeader = r.Header
				headerMu.Unlock()
				cookie, err := r.Cookie("test-cookie")
				if err != nil {
					t.Errorf("missing cookie: %v", err)
... (1388 more lines)
</content>
</file>
<file path="mcp/tool.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
)

// A ToolHandler handles a call to tools/call.
//
// This is a low-level API, for use with [Server.AddTool]. It does not do any
// pre- or post-processing of the request or result: the params contain raw
// arguments, no input validation is performed, and the result is returned to
// the user as-is, without any validation of the output.
//
// Most users will write a [ToolHandlerFor] and install it with the generic
// [AddTool] function.
//
// If ToolHandler returns an error, it is treated as a protocol error. By
// contrast, [ToolHandlerFor] automatically populates [CallToolResult.IsError]
// and [CallToolResult.Content] accordingly.
type ToolHandler func(context.Context, *CallToolRequest) (*CallToolResult, error)

// A ToolHandlerFor handles a call to tools/call with typed arguments and results.
//
// Use [AddTool] to add a ToolHandlerFor to a server.
//
// Unlike [ToolHandler], [ToolHandlerFor] provides significant functionality
// out of the box, and enforces that the tool conforms to the MCP spec:
//   - The In type provides a default input schema for the tool, though it may
//     be overridden in [AddTool].
//   - The input value is automatically unmarshaled from req.Params.Arguments.
//   - The input value is automatically validated against its input schema.
//     Invalid input is rejected before getting to the handler.
//   - If the Out type is not the empty interface [any], it provides the
//     default output schema for the tool (which again may be overridden in
//     [AddTool]).
//   - The Out value is used to populate result.StructuredOutput.
//   - If [CallToolResult.Content] is unset, it is populated with the JSON
//     content of the output.
//   - An error result is treated as a tool error, rather than a protocol
//     error, and is therefore packed into CallToolResult.Content, with
//     [IsError] set.
//
// For these reasons, most users can ignore the [CallToolRequest] argument and
// [CallToolResult] return values entirely. In fact, it is permissible to
// return a nil CallToolResult, if you only care about returning a output value
// or error. The effective result will be populated as described above.
type ToolHandlerFor[In, Out any] func(_ context.Context, request *CallToolRequest, input In) (result *CallToolResult, output Out, _ error)

// A serverTool is a tool definition that is bound to a tool handler.
type serverTool struct {
	tool    *Tool
	handler ToolHandler
}

// applySchema validates whether data is valid JSON according to the provided
// schema, after applying schema defaults.
//
// Returns the JSON value augmented with defaults.
func applySchema(data json.RawMessage, resolved *jsonschema.Resolved) (json.RawMessage, error) {
	// TODO: use reflection to create the struct type to unmarshal into.
	// Separate validation from assignment.

	// Use default JSON marshalling for validation.
	//
	// This avoids inconsistent representation due to custom marshallers, such as
	// time.Time (issue #449).
	//
	// Additionally, unmarshalling into a map ensures that the resulting JSON is
	// at least {}, even if data is empty. For example, arguments is technically
	// an optional property of callToolParams, and we still want to apply the
	// defaults in this case.
	//
	// TODO(rfindley): in which cases can resolved be nil?
	if resolved != nil {
		v := make(map[string]any)
		if len(data) > 0 {
			if err := json.Unmarshal(data, &v); err != nil {
				return nil, fmt.Errorf("unmarshaling arguments: %w", err)
			}
		}
		if err := resolved.ApplyDefaults(&v); err != nil {
			return nil, fmt.Errorf("applying schema defaults:\n%w", err)
		}
		if err := resolved.Validate(&v); err != nil {
			return nil, err
		}
		// We must re-marshal with the default values applied.
		var err error
		data, err = json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("marshalling with defaults: %v", err)
		}
	}
	return data, nil
}
</content>
</file>
<file path="mcp/tool_example_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func ExampleServer_AddTool_rawSchema() {
	// In some scenarios, you may want your server to be a pass-through, with
	// JSON schema coming from another source. Or perhaps you want to implement
	// tool validation using a different JSON schema library.
	//
	// For these cases, you can use [mcp.Server.AddTool], which is the "raw" form
	// of the API. Note that it is the caller's responsibility to validate inputs
	// and outputs.
	server := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.0.1"}, nil)
	server.AddTool(&mcp.Tool{
		Name:        "greet",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"user":{"type":"string"}}}`),
	}, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Note: no validation!
		var args struct{ User string }
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			// TODO: we should use a jsonrpc error here, to be consistent with other
			// SDKs.
			return nil, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Hi " + args.User}},
		}, nil
	})

	ctx := context.Background()
	session, err := connect(ctx, server)
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "greet",
		Arguments: map[string]any{"user": "you"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res.Content[0].(*mcp.TextContent).Text)
	// Output: Hi you
}

func ExampleAddTool_customMarshalling() {
	// Sometimes when you want to customize the input or output schema for a
	// tool, you need to customize the schema of a single helper type that's used
	// in several places.
	//
	// For example, suppose you had a type that marshals/unmarshals like a
	// time.Time, and that type was used multiple times in your tool input.
	type MyDate struct {
		time.Time
	}
	type Input struct {
		Query string `json:"query,omitempty"`
		Start MyDate `json:"start,omitempty"`
		End   MyDate `json:"end,omitempty"`
	}

	// In this case, you can use jsonschema.For along with jsonschema.ForOptions
	// to customize the schema inference for your custom type.
	inputSchema, err := jsonschema.For[Input](&jsonschema.ForOptions{
		TypeSchemas: map[reflect.Type]*jsonschema.Schema{
			reflect.TypeFor[MyDate](): {Type: "string"},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.0.1"}, nil)
	toolHandler := func(context.Context, *mcp.CallToolRequest, Input) (*mcp.CallToolResult, any, error) {
		panic("not implemented")
	}
	mcp.AddTool(server, &mcp.Tool{Name: "my_tool", InputSchema: inputSchema}, toolHandler)

	ctx := context.Background()
	session, err := connect(ctx, server) // create an in-memory connection
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	for t, err := range session.Tools(ctx, nil) {
		if err != nil {
			log.Fatal(err)
		}
		schemaJSON, err := json.MarshalIndent(t.InputSchema, "", "\t")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(t.Name, string(schemaJSON))
	}
	// Output:
	// my_tool {
	// 	"additionalProperties": false,
	// 	"properties": {
	// 		"end": {
	// 			"type": "string"
	// 		},
	// 		"query": {
	// 			"type": "string"
	// 		},
	// 		"start": {
	// 			"type": "string"
	// 		}
	// 	},
	// 	"type": "object"
	// }
}

type Location struct {
	Name      string   `json:"name"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
}

type Forecast struct {
	Forecast string      `json:"forecast" jsonschema:"description of the day's weather"`
	Type     WeatherType `json:"type" jsonschema:"type of weather"`
	Rain     float64     `json:"rain" jsonschema:"probability of rain, between 0 and 1"`
	High     float64     `json:"high" jsonschema:"high temperature"`
	Low      float64     `json:"low" jsonschema:"low temperature"`
}

type WeatherType string

const (
	Sunny        WeatherType = "sun"
	PartlyCloudy WeatherType = "partly_cloudy"
	Cloudy       WeatherType = "clouds"
	Rainy        WeatherType = "rain"
	Snowy        WeatherType = "snow"
)

type Probability float64

// !+weathertool

type WeatherInput struct {
	Location Location `json:"location" jsonschema:"user location"`
	Days     int      `json:"days" jsonschema:"number of days to forecast"`
}

type WeatherOutput struct {
	Summary       string      `json:"summary" jsonschema:"a summary of the weather forecast"`
	Confidence    Probability `json:"confidence" jsonschema:"confidence, between 0 and 1"`
	AsOf          time.Time   `json:"asOf" jsonschema:"the time the weather was computed"`
	DailyForecast []Forecast  `json:"dailyForecast" jsonschema:"the daily forecast"`
	Source        string      `json:"source,omitempty" jsonschema:"the organization providing the weather forecast"`
}

func WeatherTool(ctx context.Context, req *mcp.CallToolRequest, in WeatherInput) (*mcp.CallToolResult, WeatherOutput, error) {
	perfectWeather := WeatherOutput{
		Summary:    "perfect",
		Confidence: 1.0,
		AsOf:       time.Now(),
	}
	for range in.Days {
		perfectWeather.DailyForecast = append(perfectWeather.DailyForecast, Forecast{
			Forecast: "another perfect day",
			Type:     Sunny,
			Rain:     0.0,
			High:     72.0,
			Low:      72.0,
		})
	}
	return nil, perfectWeather, nil
}

// !-weathertool

func ExampleAddTool_complexSchema() {
	// This example demonstrates a tool with a more 'realistic' input and output
	// schema. We use a combination of techniques to tune our input and output
	// schemas.

	// !+customschemas

	// Distinguished Go types allow custom schemas to be reused during inference.
	customSchemas := map[reflect.Type]*jsonschema.Schema{
		reflect.TypeFor[Probability](): {Type: "number", Minimum: jsonschema.Ptr(0.0), Maximum: jsonschema.Ptr(1.0)},
		reflect.TypeFor[WeatherType](): {Type: "string", Enum: []any{Sunny, PartlyCloudy, Cloudy, Rainy, Snowy}},
	}
	opts := &jsonschema.ForOptions{TypeSchemas: customSchemas}
	in, err := jsonschema.For[WeatherInput](opts)
	if err != nil {
		log.Fatal(err)
	}

	// Furthermore, we can tweak the inferred schema, in this case limiting
	// forecasts to 0-10 days.
	daysSchema := in.Properties["days"]
	daysSchema.Minimum = jsonschema.Ptr(0.0)
	daysSchema.Maximum = jsonschema.Ptr(10.0)

	// Output schema inference can reuse our custom schemas from input inference.
	out, err := jsonschema.For[WeatherOutput](opts)
	if err != nil {
		log.Fatal(err)
	}

	// Now add our tool to a server. Since we've customized the schemas, we need
	// to override the default schema inference.
	server := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.0.1"}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name:         "weather",
		InputSchema:  in,
		OutputSchema: out,
	}, WeatherTool)

	// !-customschemas

	ctx := context.Background()
	session, err := connect(ctx, server) // create an in-memory connection
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	// Check that the client observes the correct schemas.
	for t, err := range session.Tools(ctx, nil) {
		if err != nil {
			log.Fatal(err)
		}
		// Formatting the entire schemas would be too much output.
		// Just check that our customizations were effective.
		fmt.Println("max days:", jsonPath(t.InputSchema, "properties", "days", "maximum"))
		fmt.Println("max confidence:", jsonPath(t.OutputSchema, "properties", "confidence", "maximum"))
		fmt.Println("weather types:", jsonPath(t.OutputSchema, "properties", "dailyForecast", "items", "properties", "type", "enum"))
	}
	// Output:
	// max days: 10
	// max confidence: 1
	// weather types: [sun partly_cloudy clouds rain snow]
}

func connect(ctx context.Context, server *mcp.Server) (*mcp.ClientSession, error) {
	t1, t2 := mcp.NewInMemoryTransports()
	if _, err := server.Connect(ctx, t1, nil); err != nil {
		return nil, err
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, nil)
	return client.Connect(ctx, t2, nil)
}

func jsonPath(s any, path ...string) any {
	if len(path) == 0 {
		return s
	}
	return jsonPath(s.(map[string]any)[path[0]], path[1:]...)
}
</content>
</file>
<file path="mcp/tool_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/internal/jsonrpc2"
)

func TestApplySchema(t *testing.T) {
	schema := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"x": {Type: "integer", Default: json.RawMessage("3")},
		},
	}
	resolved, err := schema.Resolve(&jsonschema.ResolveOptions{ValidateDefaults: true})
	if err != nil {
		t.Fatal(err)
	}

	type S struct {
		X int `json:"x"`
	}

	for _, tt := range []struct {
		data string
		v    any
		want any
	}{
		{`{"x": 1}`, new(S), &S{X: 1}},
		{`{}`, new(S), &S{X: 3}}, // default applied
		{`{"x": 0}`, new(S), &S{X: 0}},
		{`{"x": 1}`, new(map[string]any), &map[string]any{"x": 1.0}},
		{`{}`, new(map[string]any), &map[string]any{"x": 3.0}}, // default applied
		{`{"x": 0}`, new(map[string]any), &map[string]any{"x": 0.0}},
	} {
		raw := json.RawMessage(tt.data)
		raw, err = applySchema(raw, resolved)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &tt.v); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(tt.v, tt.want) {
			t.Errorf("got %#v, want %#v", tt.v, tt.want)
		}
	}
}

func TestToolErrorHandling(t *testing.T) {
	// Construct server and add both tools at the top level
	server := NewServer(testImpl, nil)

	// Create a tool that returns a structured error
	structuredErrorHandler := func(ctx context.Context, req *CallToolRequest, args map[string]any) (*CallToolResult, any, error) {
		return nil, nil, &jsonrpc2.WireError{
			Code:    codeInvalidParams,
			Message: "internal server error",
		}
	}

	// Create a tool that returns a regular error
	regularErrorHandler := func(ctx context.Context, req *CallToolRequest, args map[string]any) (*CallToolResult, any, error) {
		return nil, nil, fmt.Errorf("tool execution failed")
	}

	AddTool(server, &Tool{Name: "error_tool", Description: "returns structured error"}, structuredErrorHandler)
	AddTool(server, &Tool{Name: "regular_error_tool", Description: "returns regular error"}, regularErrorHandler)

	// Connect server and client once
	ct, st := NewInMemoryTransports()
	_, err := server.Connect(context.Background(), st, nil)
	if err != nil {
		t.Fatal(err)
	}

	client := NewClient(testImpl, nil)
	cs, err := client.Connect(context.Background(), ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()

	// Test that structured JSON-RPC errors are returned directly
	t.Run("structured_error", func(t *testing.T) {
		// Call the tool
		_, err = cs.CallTool(context.Background(), &CallToolParams{
			Name:      "error_tool",
			Arguments: map[string]any{},
		})

		// Should get the structured error directly
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var wireErr *jsonrpc2.WireError
		if !errors.As(err, &wireErr) {
			t.Fatalf("expected WireError, got %[1]T: %[1]v", err)
		}

		if wireErr.Code != codeInvalidParams {
			t.Errorf("expected error code %d, got %d", codeInvalidParams, wireErr.Code)
		}
	})

	// Test that regular errors are embedded in tool results
	t.Run("regular_error", func(t *testing.T) {
		// Call the tool
		result, err := cs.CallTool(context.Background(), &CallToolParams{
			Name:      "regular_error_tool",
			Arguments: map[string]any{},
		})
		// Should not get an error at the protocol level
		if err != nil {
			t.Fatalf("unexpected protocol error: %v", err)
		}

		// Should get a result with IsError=true
		if !result.IsError {
			t.Error("expected IsError=true, got false")
		}

		// Should have error message in content
		if len(result.Content) == 0 {
			t.Error("expected error content, got empty")
		}

		if textContent, ok := result.Content[0].(*TextContent); !ok {
			t.Error("expected TextContent")
		} else if !strings.Contains(textContent.Text, "tool execution failed") {
			t.Errorf("expected error message in content, got: %s", textContent.Text)
		}
	})
}
</content>
</file>
<file path="mcp/transport.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/internal/jsonrpc2"
	"github.com/modelcontextprotocol/go-sdk/internal/xcontext"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

// ErrConnectionClosed is returned when sending a message to a connection that
// is closed or in the process of closing.
var ErrConnectionClosed = errors.New("connection closed")

// A Transport is used to create a bidirectional connection between MCP client
// and server.
//
// Transports should be used for at most one call to [Server.Connect] or
// [Client.Connect].
type Transport interface {
	// Connect returns the logical JSON-RPC connection..
	//
	// It is called exactly once by [Server.Connect] or [Client.Connect].
	Connect(ctx context.Context) (Connection, error)
}

// A Connection is a logical bidirectional JSON-RPC connection.
type Connection interface {
	// Read reads the next message to process off the connection.
	//
	// Connections must allow Read to be called concurrently with Close. In
	// particular, calling Close should unblock a Read waiting for input.
	Read(context.Context) (jsonrpc.Message, error)

	// Write writes a new message to the connection.
	//
	// Write may be called concurrently, as calls or reponses may occur
	// concurrently in user code.
	Write(context.Context, jsonrpc.Message) error

	// Close closes the connection. It is implicitly called whenever a Read or
	// Write fails.
	//
	// Close may be called multiple times, potentially concurrently.
	Close() error

	// TODO(#148): remove SessionID from this interface.
	SessionID() string
}

// A ClientConnection is a [Connection] that is specific to the MCP client.
//
// If client connections implement this interface, they may receive information
// about changes to the client session.
//
// TODO: should this interface be exported?
type clientConnection interface {
	Connection

	// SessionUpdated is called whenever the client session state changes.
	sessionUpdated(clientSessionState)
}

// A serverConnection is a Connection that is specific to the MCP server.
//
// If server connections implement this interface, they receive information
// about changes to the server session.
//
// TODO: should this interface be exported?
type serverConnection interface {
	Connection
	sessionUpdated(ServerSessionState)
}

// A StdioTransport is a [Transport] that communicates over stdin/stdout using
// newline-delimited JSON.
type StdioTransport struct{}

// Connect implements the [Transport] interface.
func (*StdioTransport) Connect(context.Context) (Connection, error) {
	return newIOConn(rwc{os.Stdin, os.Stdout}), nil
}

// An IOTransport is a [Transport] that communicates over separate
// io.ReadCloser and io.WriteCloser using newline-delimited JSON.
type IOTransport struct {
	Reader io.ReadCloser
	Writer io.WriteCloser
}

// Connect implements the [Transport] interface.
func (t *IOTransport) Connect(context.Context) (Connection, error) {
	return newIOConn(rwc{t.Reader, t.Writer}), nil
}

// An InMemoryTransport is a [Transport] that communicates over an in-memory
// network connection, using newline-delimited JSON.
type InMemoryTransport struct {
	rwc io.ReadWriteCloser
}

// Connect implements the [Transport] interface.
func (t *InMemoryTransport) Connect(context.Context) (Connection, error) {
	return newIOConn(t.rwc), nil
}

// NewInMemoryTransports returns two [InMemoryTransport] objects that connect
// to each other.
//
// The resulting transports are symmetrical: use either to connect to a server,
// and then the other to connect to a client. Servers must be connected before
// clients, as the client initializes the MCP session during connection.
func NewInMemoryTransports() (*InMemoryTransport, *InMemoryTransport) {
	c1, c2 := net.Pipe()
	return &InMemoryTransport{c1}, &InMemoryTransport{c2}
}

type binder[T handler, State any] interface {
	// TODO(rfindley): the bind API has gotten too complicated. Simplify.
	bind(Connection, *jsonrpc2.Connection, State, func()) T
	disconnect(T)
}

type handler interface {
	handle(ctx context.Context, req *jsonrpc.Request) (any, error)
}

func connect[H handler, State any](ctx context.Context, t Transport, b binder[H, State], s State, onClose func()) (H, error) {
	var zero H
	mcpConn, err := t.Connect(ctx)
	if err != nil {
		return zero, err
	}
	// If logging is configured, write message logs.
	reader, writer := jsonrpc2.Reader(mcpConn), jsonrpc2.Writer(mcpConn)
	var (
		h         H
		preempter canceller
	)
	bind := func(conn *jsonrpc2.Connection) jsonrpc2.Handler {
		h = b.bind(mcpConn, conn, s, onClose)
		preempter.conn = conn
		return jsonrpc2.HandlerFunc(h.handle)
	}
	_ = jsonrpc2.NewConnection(ctx, jsonrpc2.ConnectionConfig{
		Reader:    reader,
		Writer:    writer,
		Closer:    mcpConn,
		Bind:      bind,
		Preempter: &preempter,
		OnDone: func() {
			b.disconnect(h)
		},
		OnInternalError: func(err error) { log.Printf("jsonrpc2 error: %v", err) },
	})
	assert(preempter.conn != nil, "unbound preempter")
	return h, nil
}

// A canceller is a jsonrpc2.Preempter that cancels in-flight requests on MCP
// cancelled notifications.
type canceller struct {
	conn *jsonrpc2.Connection
}

// Preempt implements [jsonrpc2.Preempter].
func (c *canceller) Preempt(ctx context.Context, req *jsonrpc.Request) (result any, err error) {
	if req.Method == notificationCancelled {
		var params CancelledParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, err
		}
		id, err := jsonrpc2.MakeID(params.RequestID)
		if err != nil {
			return nil, err
		}
		go c.conn.Cancel(id)
	}
	return nil, jsonrpc2.ErrNotHandled
}

// call executes and awaits a jsonrpc2 call on the given connection,
// translating errors into the mcp domain.
func call(ctx context.Context, conn *jsonrpc2.Connection, method string, params Params, result Result) error {
	// TODO: the "%w"s in this function effectively make jsonrpc2.WireError part of the API.
	// Consider alternatives.
	call := conn.Call(ctx, method, params)
	err := call.Await(ctx, result)
	switch {
	case errors.Is(err, jsonrpc2.ErrClientClosing), errors.Is(err, jsonrpc2.ErrServerClosing):
		return fmt.Errorf("%w: calling %q: %v", ErrConnectionClosed, method, err)
	case ctx.Err() != nil:
		// Notify the peer of cancellation.
		err := conn.Notify(xcontext.Detach(ctx), notificationCancelled, &CancelledParams{
			Reason:    ctx.Err().Error(),
			RequestID: call.ID().Raw(),
		})
		return errors.Join(ctx.Err(), err)
	case err != nil:
		return fmt.Errorf("calling %q: %w", method, err)
	}
	return nil
}

// A LoggingTransport is a [Transport] that delegates to another transport,
// writing RPC logs to an io.Writer.
type LoggingTransport struct {
	Transport Transport
	Writer    io.Writer
}

// Connect connects the underlying transport, returning a [Connection] that writes
// logs to the configured destination.
func (t *LoggingTransport) Connect(ctx context.Context) (Connection, error) {
	delegate, err := t.Transport.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &loggingConn{delegate: delegate, w: t.Writer}, nil
}

type loggingConn struct {
	delegate Connection

	mu sync.Mutex
	w  io.Writer
}

func (c *loggingConn) SessionID() string { return c.delegate.SessionID() }

// Read is a stream middleware that logs incoming messages.
func (s *loggingConn) Read(ctx context.Context) (jsonrpc.Message, error) {
	msg, err := s.delegate.Read(ctx)

	if err != nil {
		s.mu.Lock()
		fmt.Fprintf(s.w, "read error: %v\n", err)
		s.mu.Unlock()
	} else {
		data, err := jsonrpc2.EncodeMessage(msg)
		s.mu.Lock()
		if err != nil {
			fmt.Fprintf(s.w, "LoggingTransport: failed to marshal: %v", err)
		}
		fmt.Fprintf(s.w, "read: %s\n", string(data))
		s.mu.Unlock()
	}

	return msg, err
}

// Write is a stream middleware that logs outgoing messages.
func (s *loggingConn) Write(ctx context.Context, msg jsonrpc.Message) error {
	err := s.delegate.Write(ctx, msg)
	if err != nil {
		s.mu.Lock()
		fmt.Fprintf(s.w, "write error: %v\n", err)
		s.mu.Unlock()
	} else {
		data, err := jsonrpc2.EncodeMessage(msg)
		s.mu.Lock()
		if err != nil {
			fmt.Fprintf(s.w, "LoggingTransport: failed to marshal: %v", err)
		}
		fmt.Fprintf(s.w, "write: %s\n", string(data))
		s.mu.Unlock()
	}
	return err
}

func (s *loggingConn) Close() error {
	return s.delegate.Close()
}

// A rwc binds an io.ReadCloser and io.WriteCloser together to create an
// io.ReadWriteCloser.
type rwc struct {
	rc io.ReadCloser
	wc io.WriteCloser
}

func (r rwc) Read(p []byte) (n int, err error) {
	return r.rc.Read(p)
}

func (r rwc) Write(p []byte) (n int, err error) {
	return r.wc.Write(p)
}

func (r rwc) Close() error {
	return errors.Join(r.rc.Close(), r.wc.Close())
}

// An ioConn is a transport that delimits messages with newlines across
// a bidirectional stream, and supports jsonrpc.2 message batching.
//
// See https://github.com/ndjson/ndjson-spec for discussion of newline
// delimited JSON.
//
// See [msgBatch] for more discussion of message batching.
type ioConn struct {
	protocolVersion string // negotiated version, set during session initialization.

	writeMu sync.Mutex         // guards Write, which must be concurrency safe.
	rwc     io.ReadWriteCloser // the underlying stream

	// incoming receives messages from the read loop started in [newIOConn].
	incoming <-chan msgOrErr

	// If outgoiBatch has a positive capacity, it will be used to batch requests
	// and notifications before sending.
	outgoingBatch []jsonrpc.Message

	// Unread messages in the last batch. Since reads are serialized, there is no
	// need to guard here.
	queue []jsonrpc.Message

	// batches correlate incoming requests to the batch in which they arrived.
	// Since writes may be concurrent to reads, we need to guard this with a mutex.
	batchMu sync.Mutex
	batches map[jsonrpc2.ID]*msgBatch // lazily allocated

	closeOnce sync.Once
	closed    chan struct{}
	closeErr  error
}

type msgOrErr struct {
	msg json.RawMessage
	err error
}

func newIOConn(rwc io.ReadWriteCloser) *ioConn {
	var (
		incoming = make(chan msgOrErr)
		closed   = make(chan struct{})
	)
	// Start a goroutine for reads, so that we can select on the incoming channel
	// in [ioConn.Read] and unblock the read as soon as Close is called (see #224).
	//
	// This leaks a goroutine if rwc.Read does not unblock after it is closed,
	// but that is unavoidable since AFAIK there is no (easy and portable) way to
	// guarantee that reads of stdin are unblocked when closed.
	go func() {
		dec := json.NewDecoder(rwc)
		for {
			var raw json.RawMessage
			err := dec.Decode(&raw)
			// If decoding was successful, check for trailing data at the end of the stream.
			if err == nil {
				// Read the next byte to check if there is trailing data.
				var tr [1]byte
				if n, readErr := dec.Buffered().Read(tr[:]); n > 0 {
					// If read byte is not a newline, it is an error.
					if tr[0] != '\n' {
						err = fmt.Errorf("invalid trailing data at the end of stream")
					}
				} else if readErr != nil && readErr != io.EOF {
					err = readErr
				}
			}
			select {
			case incoming <- msgOrErr{msg: raw, err: err}:
			case <-closed:
				return
			}
			if err != nil {
				return
			}
		}
	}()
	return &ioConn{
		rwc:      rwc,
		incoming: incoming,
		closed:   closed,
	}
}

func (c *ioConn) SessionID() string { return "" }

func (c *ioConn) sessionUpdated(state ServerSessionState) {
	protocolVersion := ""
	if state.InitializeParams != nil {
		protocolVersion = state.InitializeParams.ProtocolVersion
	}
	if protocolVersion == "" {
		protocolVersion = protocolVersion20250326
	}
	c.protocolVersion = negotiatedVersion(protocolVersion)
}

// addBatch records a msgBatch for an incoming batch payload.
// It returns an error if batch is malformed, containing previously seen IDs.
//
// See [msgBatch] for more.
func (t *ioConn) addBatch(batch *msgBatch) error {
	t.batchMu.Lock()
	defer t.batchMu.Unlock()
	for id := range batch.unresolved {
		if _, ok := t.batches[id]; ok {
			return fmt.Errorf("%w: batch contains previously seen request %v", jsonrpc2.ErrInvalidRequest, id.Raw())
		}
	}
	for id := range batch.unresolved {
		if t.batches == nil {
			t.batches = make(map[jsonrpc2.ID]*msgBatch)
		}
		t.batches[id] = batch
	}
	return nil
}

// updateBatch records a response in the message batch tracking the
// corresponding incoming call, if any.
//
// The second result reports whether resp was part of a batch. If this is true,
// the first result is nil if the batch is still incomplete, or the full set of
// batch responses if resp completed the batch.
func (t *ioConn) updateBatch(resp *jsonrpc.Response) ([]*jsonrpc.Response, bool) {
	t.batchMu.Lock()
	defer t.batchMu.Unlock()

	if batch, ok := t.batches[resp.ID]; ok {
		idx, ok := batch.unresolved[resp.ID]
		if !ok {
			panic("internal error: inconsistent batches")
		}
		batch.responses[idx] = resp
		delete(batch.unresolved, resp.ID)
		delete(t.batches, resp.ID)
		if len(batch.unresolved) == 0 {
			return batch.responses, true
		}
		return nil, true
	}
	return nil, false
}

// A msgBatch records information about an incoming batch of jsonrpc.2 calls.
//
// The jsonrpc.2 spec (https://www.jsonrpc.org/specification#batch) says:
//
// "The Server should respond with an Array containing the corresponding
// Response objects, after all of the batch Request objects have been
// processed. A Response object SHOULD exist for each Request object, except
// that there SHOULD NOT be any Response objects for notifications. The Server
// MAY process a batch rpc call as a set of concurrent tasks, processing them
// in any order and with any width of parallelism."
//
// Therefore, a msgBatch keeps track of outstanding calls and their responses.
// When there are no unresolved calls, the response payload is sent.
type msgBatch struct {
	unresolved map[jsonrpc2.ID]int
	responses  []*jsonrpc.Response
}

func (t *ioConn) Read(ctx context.Context) (jsonrpc.Message, error) {
	// As a matter of principle, enforce that reads on a closed context return an
	// error.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	if len(t.queue) > 0 {
		next := t.queue[0]
		t.queue = t.queue[1:]
		return next, nil
	}

	var raw json.RawMessage
	select {
	case <-ctx.Done():
		return nil, ctx.Err()

	case v := <-t.incoming:
		if v.err != nil {
			return nil, v.err
		}
		raw = v.msg

	case <-t.closed:
		return nil, io.EOF
	}

	msgs, batch, err := readBatch(raw)
	if err != nil {
		return nil, err
	}
	if batch && t.protocolVersion >= protocolVersion20250618 {
		return nil, fmt.Errorf("JSON-RPC batching is not supported in %s and later (request version: %s)", protocolVersion20250618, t.protocolVersion)
	}

	t.queue = msgs[1:]

	if batch {
		var respBatch *msgBatch // track incoming requests in the batch
		for _, msg := range msgs {
			if req, ok := msg.(*jsonrpc.Request); ok {
				if respBatch == nil {
					respBatch = &msgBatch{
						unresolved: make(map[jsonrpc2.ID]int),
					}
				}
				if _, ok := respBatch.unresolved[req.ID]; ok {
					return nil, fmt.Errorf("duplicate message ID %q", req.ID)
				}
				respBatch.unresolved[req.ID] = len(respBatch.responses)
				respBatch.responses = append(respBatch.responses, nil)
			}
		}
		if respBatch != nil {
			// The batch contains one or more incoming requests to track.
			if err := t.addBatch(respBatch); err != nil {
				return nil, err
			}
		}
	}
	return msgs[0], err
}

// readBatch reads batch data, which may be either a single JSON-RPC message,
// or an array of JSON-RPC messages.
func readBatch(data []byte) (msgs []jsonrpc.Message, isBatch bool, _ error) {
	// Try to read an array of messages first.
	var rawBatch []json.RawMessage
	if err := json.Unmarshal(data, &rawBatch); err == nil {
		if len(rawBatch) == 0 {
			return nil, true, fmt.Errorf("empty batch")
		}
		for _, raw := range rawBatch {
			msg, err := jsonrpc2.DecodeMessage(raw)
			if err != nil {
				return nil, true, err
			}
			msgs = append(msgs, msg)
		}
		return msgs, true, nil
	}
	// Try again with a single message.
	msg, err := jsonrpc2.DecodeMessage(data)
	return []jsonrpc.Message{msg}, false, err
}

func (t *ioConn) Write(ctx context.Context, msg jsonrpc.Message) error {
	// As in [ioConn.Read], enforce that Writes on a closed context are an error.
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	// Batching support: if msg is a Response, it may have completed a batch, so
	// check that first. Otherwise, it is a request or notification, and we may
	// want to collect it into a batch before sending, if we're configured to use
	// outgoing batches.
	if resp, ok := msg.(*jsonrpc.Response); ok {
		if batch, ok := t.updateBatch(resp); ok {
			if len(batch) > 0 {
				data, err := marshalMessages(batch)
				if err != nil {
					return err
				}
				data = append(data, '\n')
				_, err = t.rwc.Write(data)
				return err
			}
			return nil
		}
	} else if len(t.outgoingBatch) < cap(t.outgoingBatch) {
		t.outgoingBatch = append(t.outgoingBatch, msg)
		if len(t.outgoingBatch) == cap(t.outgoingBatch) {
			data, err := marshalMessages(t.outgoingBatch)
			t.outgoingBatch = t.outgoingBatch[:0]
			if err != nil {
				return err
			}
			data = append(data, '\n')
			_, err = t.rwc.Write(data)
			return err
		}
		return nil
	}
	data, err := jsonrpc2.EncodeMessage(msg)
	if err != nil {
		return fmt.Errorf("marshaling message: %v", err)
	}
	data = append(data, '\n') // newline delimited
	_, err = t.rwc.Write(data)
	return err
}

func (t *ioConn) Close() error {
	t.closeOnce.Do(func() {
		t.closeErr = t.rwc.Close()
		close(t.closed)
	})
	return t.closeErr
}

func marshalMessages[T jsonrpc.Message](msgs []T) ([]byte, error) {
	var rawMsgs []json.RawMessage
	for _, msg := range msgs {
		raw, err := jsonrpc2.EncodeMessage(msg)
		if err != nil {
			return nil, fmt.Errorf("encoding batch message: %w", err)
		}
		rawMsgs = append(rawMsgs, raw)
	}
	return json.Marshal(rawMsgs)
}
</content>
</file>
<file path="mcp/transport_example_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// Uses strings.SplitSeq.
//go:build go1.24

package mcp_test

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// !+loggingtransport

func ExampleLoggingTransport() {
	ctx := context.Background()
	t1, t2 := mcp.NewInMemoryTransports()
	server := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.0.1"}, nil)
	serverSession, err := server.Connect(ctx, t1, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer serverSession.Wait()

	client := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, nil)
	var b bytes.Buffer
	logTransport := &mcp.LoggingTransport{Transport: t2, Writer: &b}
	clientSession, err := client.Connect(ctx, logTransport, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer clientSession.Close()

	// Sort for stability: reads are concurrent to writes.
	for _, line := range slices.Sorted(strings.SplitSeq(b.String(), "\n")) {
		fmt.Println(line)
	}

	// Output:
	// read: {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"logging":{}},"protocolVersion":"2025-06-18","serverInfo":{"name":"server","version":"v0.0.1"}}}
	// write: {"jsonrpc":"2.0","id":1,"method":"initialize","params":{"capabilities":{"roots":{"listChanged":true}},"clientInfo":{"name":"client","version":"v0.0.1"},"protocolVersion":"2025-06-18"}}
	// write: {"jsonrpc":"2.0","method":"notifications/initialized","params":{}}
}

// !-loggingtransport
</content>
</file>
<file path="mcp/transport_test.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/internal/jsonrpc2"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

func TestBatchFraming(t *testing.T) {
	// This test checks that the ndjsonFramer can read and write JSON batches.
	//
	// The framer is configured to write a batch size of 2, and we confirm that
	// nothing is sent over the wire until the second write, at which point both
	// messages become available.
	ctx := context.Background()

	r, w := io.Pipe()
	tport := newIOConn(rwc{r, w})
	tport.outgoingBatch = make([]jsonrpc.Message, 0, 2)
	defer tport.Close()

	// Read the two messages into a channel, for easy testing later.
	read := make(chan jsonrpc.Message)
	go func() {
		for range 2 {
			msg, _ := tport.Read(ctx)
			read <- msg
		}
	}()

	// The first write should not yet be observed by the reader.
	tport.Write(ctx, &jsonrpc.Request{ID: jsonrpc2.Int64ID(1), Method: "test"})
	select {
	case got := <-read:
		t.Fatalf("after one write, got message %v", got)
	default:
	}

	// ...but the second write causes both messages to be observed.
	tport.Write(ctx, &jsonrpc.Request{ID: jsonrpc2.Int64ID(2), Method: "test"})
	for _, want := range []int64{1, 2} {
		got := <-read
		if got := got.(*jsonrpc.Request).ID.Raw(); got != want {
			t.Errorf("got message #%d, want #%d", got, want)
		}
	}
}

func TestIOConnRead(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		want            string
		protocolVersion string
	}{
		{
			name:  "valid json input",
			input: `{"jsonrpc":"2.0","id":1,"method":"test","params":{}}`,
			want:  "",
		},
		{
			name: "newline at the end of first valid json input",
			input: `{"jsonrpc":"2.0","id":1,"method":"test","params":{}}
			`,
			want: "",
		},
		{
			name:  "bad data at the end of first valid json input",
			input: `{"jsonrpc":"2.0","id":1,"method":"test","params":{}},`,
			want:  "invalid trailing data at the end of stream",
		},
		{
			name:            "batching unknown protocol",
			input:           `[{"jsonrpc":"2.0","id":1,"method":"test1"},{"jsonrpc":"2.0","id":2,"method":"test2"}]`,
			want:            "",
			protocolVersion: "",
		},
		{
			name:            "batching old protocol",
			input:           `[{"jsonrpc":"2.0","id":1,"method":"test1"},{"jsonrpc":"2.0","id":2,"method":"test2"}]`,
			want:            "",
			protocolVersion: protocolVersion20241105,
		},
		{
			name:            "batching new protocol",
			input:           `[{"jsonrpc":"2.0","id":1,"method":"test1"},{"jsonrpc":"2.0","id":2,"method":"test2"}]`,
			want:            "JSON-RPC batching is not supported in 2025-06-18 and later (request version: 2025-06-18)",
			protocolVersion: protocolVersion20250618,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := newIOConn(rwc{
				rc: io.NopCloser(strings.NewReader(tt.input)),
			})
			if tt.protocolVersion != "" {
				tr.sessionUpdated(ServerSessionState{
					InitializeParams: &InitializeParams{
						ProtocolVersion: tt.protocolVersion,
					},
				})
			}
			_, err := tr.Read(context.Background())
			if err == nil && tt.want != "" {
				t.Errorf("ioConn.Read() got nil error but wanted %v", tt.want)
			}
			if err != nil && err.Error() != tt.want {
				t.Errorf("ioConn.Read() = %v, want %v", err.Error(), tt.want)
			}
		})
	}
}
</content>
</file>
<file path="mcp/util.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"crypto/rand"
	"encoding/json"
)

func assert(cond bool, msg string) {
	if !cond {
		panic(msg)
	}
}

// Copied from crypto/rand.
// TODO: once 1.24 is assured, just use crypto/rand.
const base32alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"

func randText() string {
	// ⌈log₃₂ 2¹²⁸⌉ = 26 chars
	src := make([]byte, 26)
	rand.Read(src)
	for i := range src {
		src[i] = base32alphabet[src[i]%32]
	}
	return string(src)
}

// remarshal marshals from to JSON, and then unmarshals into to, which must be
// a pointer type.
func remarshal(from, to any) error {
	data, err := json.Marshal(from)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, to); err != nil {
		return err
	}
	return nil
}
</content>
</file>
<file path="oauthex/oauthex.go">
<type>go</type>
<content>
// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// Package oauthex implements extensions to OAuth2.
package oauthex

import (
	"github.com/modelcontextprotocol/go-sdk/internal/oauthex"
)

// ProtectedResourceMetadata is the metadata for an OAuth 2.0 protected resource,
// as defined in section 2 of https://www.rfc-editor.org/rfc/rfc9728.html.
type ProtectedResourceMetadata = oauthex.ProtectedResourceMetadata
</content>
</file>
</files>
