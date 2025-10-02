Taken from https://github.com/modelcontextprotocol/go-sdk
<files>
<file path="auth/auth.go">
<type>go</type>
<content>
package auth
import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"
)
type TokenInfo struct {
	Scopes     []string
	Expiration time.Time
	
	Extra map[string]any
}
var ErrInvalidToken = errors.New("invalid token")
var ErrOAuth = errors.New("oauth error")
type TokenVerifier func(ctx context.Context, token string, req *http.Request) (*TokenInfo, error)
type RequireBearerTokenOptions struct {
	
	
	ResourceMetadataURL string
	
	Scopes []string
}
type tokenInfoKey struct{}
func TokenInfoFromContext(ctx context.Context) *TokenInfo {
	ti := ctx.Value(tokenInfoKey{})
	if ti == nil {
		return nil
	}
	return ti.(*TokenInfo)
}
func RequireBearerToken(verifier TokenVerifier, opts *RequireBearerTokenOptions) func(http.Handler) http.Handler {
	
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
	
	authHeader := req.Header.Get("Authorization")
	fields := strings.Fields(authHeader)
	if len(fields) != 2 || strings.ToLower(fields[0]) != "bearer" {
		return nil, "no bearer token", http.StatusUnauthorized
	}
	
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
	
	if opts != nil {
		
		for _, s := range opts.Scopes {
			if !slices.Contains(tokenInfo.Scopes, s) {
				return nil, "insufficient scope", http.StatusForbidden
			}
		}
	}
	
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
package auth
import (
	"context"
	"errors"
	"net/http"
	"sync"
	"github.com/modelcontextprotocol/go-sdk/internal/oauthex"
	"golang.org/x/oauth2"
)
type OAuthHandler func(context.Context, OAuthHandlerArgs) (oauth2.TokenSource, error)
type OAuthHandlerArgs struct {
	
	
	ResourceMetadataURL string
}
type HTTPTransport struct {
	handler OAuthHandler
	mu      sync.Mutex 
	opts    HTTPTransportOptions
}
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
type HTTPTransportOptions struct {
	
	
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
		
		return resp, nil
	}
	resp.Body.Close()
	
	t.mu.Lock()
	defer t.mu.Unlock()
	
	
	
	
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
func TestHTTPTransport(t *testing.T) {
	const testToken = "test-token-123"
	fakeTokenSource := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: testToken,
		TokenType:   "Bearer",
	})
	
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == fmt.Sprintf("Bearer %s", testToken) {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="http:
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer authServer.Close()
	t.Run("successful auth flow", func(t *testing.T) {
		var handlerCalls int
		handler := func(ctx context.Context, args OAuthHandlerArgs) (oauth2.TokenSource, error) {
			handlerCalls++
			if args.ResourceMetadataURL != "http:
				t.Errorf("handler got metadata URL %q, want %q", args.ResourceMetadataURL, "http:
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
		
		if d.IsDir() && d.Name() != "." &&
			(strings.HasPrefix(d.Name(), ".") ||
				strings.HasPrefix(d.Name(), "_") ||
				filepath.Base(d.Name()) == "testdata") {
			return filepath.SkipDir
		}
		
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		
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
		fmt.Fprintln(os.Stderr, "Usage: listfeatures --http=\"https:
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
		fmt.Fprintf(out, "Example: loadtest -tool=greet -args='{\"name\": \"foo\"}' http:
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
						return 
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
	stop() 
	
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
package main
import (
	"context"
	"sync/atomic"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)
var nextProgressToken atomic.Int64
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
package main
import (
	"log"
	"net/http"
	"time"
)
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
		
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		log.Printf("[REQUEST] %s | %s | %s %s",
			start.Format(time.RFC3339),
			r.RemoteAddr,
			r.Method,
			r.URL.Path)
		
		handler.ServeHTTP(wrapped, r)
		
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
	proto = flag.String("proto", "http", "if set, use as proto:
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
		runClient(fmt.Sprintf("%s:
	default:
		fmt.Fprintf(os.Stderr, "Error: Invalid mode '%s'. Must be 'client' or 'server'\n\n", mode)
		flag.Usage()
	}
}
type GetTimeParams struct {
	City string `json:"city" jsonschema:"City to get time for (nyc, sf, or boston)"`
}
func getTime(ctx context.Context, req *mcp.CallToolRequest, params *GetTimeParams) (*mcp.CallToolResult, any, error) {
	
	locations := map[string]string{
		"nyc":    "America/New_York",
		"sf":     "America/Los_Angeles",
		"boston": "America/New_York",
	}
	city := params.City
	if city == "" {
		city = "nyc" 
	}
	
	tzName, ok := locations[city]
	if !ok {
		return nil, nil, fmt.Errorf("unknown city: %s", city)
	}
	
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load timezone: %w", err)
	}
	
	now := time.Now().In(loc)
	
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
	
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "time-server",
		Version: "1.0.0",
	}, nil)
	
	mcp.AddTool(server, &mcp.Tool{
		Name:        "cityTime",
		Description: "Get the current time in NYC, San Francisco, or Boston",
	}, getTime)
	
	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, nil)
	handlerWithLogging := loggingHandler(handler)
	log.Printf("MCP server listening on %s", url)
	log.Printf("Available tool: cityTime (cities: nyc, sf, boston)")
	
	if err := http.ListenAndServe(url, handlerWithLogging); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
func runClient(url string) {
	ctx := context.Background()
	
	log.Printf("Connecting to MCP server at %s", url)
	
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "time-client",
		Version: "1.0.0",
	}, nil)
	
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: url}, nil)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer session.Close()
	log.Printf("Connected to server (session ID: %s)", session.ID())
	
	log.Println("Listing available tools...")
	toolsResult, err := session.ListTools(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}
	for _, tool := range toolsResult.Tools {
		log.Printf("  - %s: %s\n", tool.Name, tool.Description)
	}
	
	cities := []string{"nyc", "sf", "boston"}
	log.Println("Getting time for each city...")
	for _, city := range cities {
		
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
var httpAddr = flag.String("http", ":8080", "HTTP address to listen on")
type JWTClaims struct {
	UserID string   `json:"user_id"` 
	Scopes []string `json:"scopes"`  
	jwt.RegisteredClaims
}
type APIKey struct {
	Key    string   `json:"key"`     
	UserID string   `json:"user_id"` 
	Scopes []string `json:"scopes"`  
}
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
var jwtSecret = []byte("your-secret-key")
func generateToken(userID string, scopes []string, expiresIn time.Duration) (string, error) {
	
	claims := JWTClaims{
		UserID: userID,
		Scopes: scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)), 
			IssuedAt:  jwt.NewNumericDate(time.Now()),                
			NotBefore: jwt.NewNumericDate(time.Now()),                
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}
func verifyJWT(ctx context.Context, tokenString string, _ *http.Request) (*auth.TokenInfo, error) {
	
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (any, error) {
		
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})
	if err != nil {
		
		return nil, fmt.Errorf("%w: %v", auth.ErrInvalidToken, err)
	}
	
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return &auth.TokenInfo{
			Scopes:     claims.Scopes,         
			Expiration: claims.ExpiresAt.Time, 
		}, nil
	}
	return nil, fmt.Errorf("%w: invalid token claims", auth.ErrInvalidToken)
}
func verifyAPIKey(ctx context.Context, apiKey string, _ *http.Request) (*auth.TokenInfo, error) {
	
	key, exists := apiKeys[apiKey]
	if !exists {
		return nil, fmt.Errorf("%w: API key not found", auth.ErrInvalidToken)
	}
	
	
	return &auth.TokenInfo{
		Scopes:     key.Scopes,                     
		Expiration: time.Now().Add(24 * time.Hour), 
	}, nil
}
type getUserInfoArgs struct {
	UserID string `json:"user_id" jsonschema:"the user ID to get information for"`
}
type createResourceArgs struct {
	Name        string `json:"name" jsonschema:"the name of the resource"`
	Description string `json:"description" jsonschema:"the description of the resource"`
	Content     string `json:"content" jsonschema:"the content of the resource"`
}
func SayHi(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, any, error) {
	
	userInfo := req.Extra.TokenInfo
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Hello! You have scopes: %v", userInfo.Scopes)},
		},
	}, nil, nil
}
func GetUserInfo(ctx context.Context, req *mcp.CallToolRequest, args getUserInfoArgs) (*mcp.CallToolResult, any, error) {
	
	userInfo := req.Extra.TokenInfo
	
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
func CreateResource(ctx context.Context, req *mcp.CallToolRequest, args createResourceArgs) (*mcp.CallToolResult, any, error) {
	
	userInfo := req.Extra.TokenInfo
	
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
func createMCPServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "authenticated-mcp-server"}, nil)
	
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
	
	server := createMCPServer()
	
	jwtAuth := auth.RequireBearerToken(verifyJWT, &auth.RequireBearerTokenOptions{
		Scopes: []string{"read"}, 
	})
	apiKeyAuth := auth.RequireBearerToken(verifyAPIKey, &auth.RequireBearerTokenOptions{
		Scopes: []string{"read"}, 
	})
	
	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, nil)
	
	authenticatedHandler := jwtAuth(handler)
	apiKeyHandler := apiKeyAuth(handler)
	
	http.HandleFunc("/mcp/jwt", authenticatedHandler.ServeHTTP)
	http.HandleFunc("/mcp/apikey", apiKeyHandler.ServeHTTP)
	
	http.HandleFunc("/generate-token", func(w http.ResponseWriter, r *http.Request) {
		
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			userID = "test-user"
		}
		
		scopes := strings.Split(r.URL.Query().Get("scopes"), ",")
		if len(scopes) == 1 && scopes[0] == "" {
			scopes = []string{"read", "write"}
		}
		
		expiresIn := 1 * time.Hour
		if expStr := r.URL.Query().Get("expires_in"); expStr != "" {
			if exp, err := time.ParseDuration(expStr); err == nil {
				expiresIn = exp
			}
		}
		
		token, err := generateToken(userID, scopes, expiresIn)
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"token": token,
			"type":  "Bearer",
		})
	})
	http.HandleFunc("/generate-api-key", func(w http.ResponseWriter, r *http.Request) {
		
		bytes := make([]byte, 16)
		if _, err := rand.Read(bytes); err != nil {
			http.Error(w, "Failed to generate random bytes", http.StatusInternalServerError)
			return
		}
		apiKey := "sk-" + base64.URLEncoding.EncodeToString(bytes)
		
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			userID = "test-user"
		}
		
		scopes := strings.Split(r.URL.Query().Get("scopes"), ",")
		if len(scopes) == 1 && scopes[0] == "" {
			scopes = []string{"read"}
		}
		
		
		apiKeys[apiKey] = &APIKey{
			Key:    apiKey,
			UserID: userID,
			Scopes: scopes,
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"api_key": apiKey,
			"type":    "Bearer",
		})
	})
	
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		})
	})
	
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
	log.Println("  curl 'http:
	log.Println()
	log.Println("  # Use MCP with JWT authentication")
	log.Println("  curl -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' \\")
	log.Println("       -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"say_hi\",\"arguments\":{}}}' \\")
	log.Println("       http:
	log.Println()
	log.Println("  # Generate an API key")
	log.Println("  curl -X POST 'http:
	log.Println()
	log.Println("  # Use MCP with API key authentication")
	log.Println("  curl -H 'Authorization: Bearer <api_key>' -H 'Content-Type: application/json' \\")
	log.Println("       -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"get_user_info\",\"arguments\":{\"user_id\":\"test\"}}}' \\")
	log.Println("       http:
	log.Fatal(http.ListenAndServe(*httpAddr, nil))
}
</content>
</file>
<file path="examples/server/basic/main.go">
<type>go</type>
<content>
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
	
}
</content>
</file>
<file path="examples/server/completion/main.go">
<type>go</type>
<content>
package main
import (
	"context"
	"fmt"
	"log"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)
func main() {
	
	
	myCompletionHandler := func(_ context.Context, req *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
		
		
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
	
	
	_ = mcp.NewServer(&mcp.Implementation{Name: "server"}, &mcp.ServerOptions{
		CompletionHandler: myCompletionHandler,
	})
	
	log.Println("MCP Server instance created with a CompletionHandler assigned (but not running).")
	log.Println("This example demonstrates configuration, not live interaction.")
}
</content>
</file>
<file path="examples/server/custom-transport/main.go">
<type>go</type>
<content>
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
type IOTransport struct {
	r *bufio.Reader
	w io.Writer
}
func NewIOTransport(r io.Reader, w io.Writer) *IOTransport {
	return &IOTransport{
		r: bufio.NewReader(r),
		w: w,
	}
}
type ioConn struct {
	r *bufio.Reader
	w io.Writer
}
func (t *IOTransport) Connect(ctx context.Context) (mcp.Connection, error) {
	return &ioConn{
		r: t.r,
		w: t.w,
	}, nil
}
func (t *ioConn) Read(context.Context) (jsonrpc.Message, error) {
	data, err := t.r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	return jsonrpc.DecodeMessage(data[:len(data)-1])
}
func (t *ioConn) Write(_ context.Context, msg jsonrpc.Message) error {
	data, err := jsonrpc.EncodeMessage(msg)
	if err != nil {
		return err
	}
	_, err1 := t.w.Write(data)
	_, err2 := t.w.Write([]byte{'\n'})
	return errors.Join(err1, err2)
}
func (t *ioConn) Close() error {
	return nil
}
func (t *ioConn) SessionID() string {
	return ""
}
type HiArgs struct {
	Name string `json:"name" mcp:"the name to say hi to"`
}
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
	
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	
	ports := strings.Split(*childPorts, ",")
	var wg sync.WaitGroup
	childURLs := make([]*url.URL, len(ports))
	for i, port := range ports {
		childURL := fmt.Sprintf("http:
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
	stop() 
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	
	wg.Wait()
	log.Println("Server shutdown gracefully.")
}
func child(port string) {
	
	server := mcp.NewServer(&mcp.Implementation{Name: "counter"}, nil)
	var count atomic.Int64
	inc := func(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, struct{ Count int64 }, error) {
		n := count.Add(1)
		if *verbose {
			log.Printf("request %d (session %s)", n, req.Session.ID())
		}
		
		
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
	
	server := mcp.NewServer(&mcp.Implementation{Name: "config-server", Version: "v1.0.0"}, nil)
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		log.Fatal(err)
	}
	
	
	client := mcp.NewClient(&mcp.Implementation{Name: "config-client", Version: "v1.0.0"}, &mcp.ClientOptions{
		ElicitationHandler: func(ctx context.Context, request *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
			fmt.Printf("Server requests: %s\n", request.Params.Message)
			
			
			return &mcp.ElicitResult{
				Action: "accept",
				Content: map[string]any{
					"serverEndpoint": "https:
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
	
	
	
}
func ptr[T any](v T) *T {
	return &v
}
</content>
</file>
<file path="examples/server/everything/main.go">
<type>go</type>
<content>
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
		
		
		
		http.DefaultServeMux.Handle("/gc", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			for range 3 {
				runtime.GC()
			}
			fmt.Fprintln(w, "GC'ed")
		}))
		go func() {
			
			http.ListenAndServe(*pprofAddr, http.DefaultServeMux)
		}()
	}
	opts := &mcp.ServerOptions{
		Instructions:      "Use this server!",
		CompletionHandler: complete, 
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "everything"}, opts)
	
	mcp.AddTool(server, &mcp.Tool{Name: "greet", Description: "say hi"}, contentTool)
	mcp.AddTool(server, &mcp.Tool{Name: "greet (structured)"}, structuredTool) 
	mcp.AddTool(server, &mcp.Tool{Name: "ping"}, pingingTool)                  
	mcp.AddTool(server, &mcp.Tool{Name: "log"}, loggingTool)                   
	mcp.AddTool(server, &mcp.Tool{Name: "sample"}, samplingTool)               
	mcp.AddTool(server, &mcp.Tool{Name: "elicit"}, elicitingTool)              
	mcp.AddTool(server, &mcp.Tool{Name: "roots"}, rootsTool)                   
	
	server.AddPrompt(&mcp.Prompt{Name: "greet"}, prompt)
	
	server.AddResource(&mcp.Resource{
		Name:     "info",
		MIMEType: "text/plain",
		URI:      "embedded:info",
	}, embeddedResource)
	
	if *httpAddr != "" {
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return server
		}, nil)
		log.Printf("MCP handler listening at %s", *httpAddr)
		if *pprofAddr != "" {
			log.Printf("pprof listening at http:
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
package main
import (
	"context"
	"log"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)
func main() {
	
	server := mcp.NewServer(&mcp.Implementation{Name: "greeter"}, nil)
	
	
	
	
	
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
	
	
	
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Printf("Server failed: %v", err)
	}
}
</content>
</file>
<file path="examples/server/memory/kb.go">
<type>go</type>
<content>
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
type Entity struct {
	Name         string   `json:"name"`
	EntityType   string   `json:"entityType"`
	Observations []string `json:"observations"`
}
type Relation struct {
	From         string `json:"from"`
	To           string `json:"to"`
	RelationType string `json:"relationType"`
}
type Observation struct {
	EntityName string   `json:"entityName"`
	Contents   []string `json:"contents"`
	Observations []string `json:"observations,omitempty"` 
}
type KnowledgeGraph struct {
	Entities  []Entity   `json:"entities"`
	Relations []Relation `json:"relations"`
}
type store interface {
	Read() ([]byte, error)
	Write(data []byte) error
}
type memoryStore struct {
	data []byte
}
func (ms *memoryStore) Read() ([]byte, error) {
	return ms.data, nil
}
func (ms *memoryStore) Write(data []byte) error {
	ms.data = data
	return nil
}
type fileStore struct {
	path string
}
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
func (fs *fileStore) Write(data []byte) error {
	if err := os.WriteFile(fs.path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write file %s: %w", fs.path, err)
	}
	return nil
}
type knowledgeBase struct {
	s store
}
type kbItem struct {
	Type string `json:"type"`
	
	Name         string   `json:"name,omitempty"`
	EntityType   string   `json:"entityType,omitempty"`
	Observations []string `json:"observations,omitempty"`
	
	From         string `json:"from,omitempty"`
	To           string `json:"to,omitempty"`
	RelationType string `json:"relationType,omitempty"`
}
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
func (k knowledgeBase) deleteEntities(entityNames []string) error {
	graph, err := k.loadGraph()
	if err != nil {
		return err
	}
	
	entitiesToDelete := make(map[string]bool)
	for _, name := range entityNames {
		entitiesToDelete[name] = true
	}
	
	graph.Entities = slices.DeleteFunc(graph.Entities, func(entity Entity) bool {
		return entitiesToDelete[entity.Name]
	})
	
	graph.Relations = slices.DeleteFunc(graph.Relations, func(relation Relation) bool {
		return entitiesToDelete[relation.From] || entitiesToDelete[relation.To]
	})
	return k.saveGraph(graph)
}
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
		
		observationsToDelete := make(map[string]bool)
		for _, observation := range deletion.Observations {
			observationsToDelete[observation] = true
		}
		
		graph.Entities[entityIndex].Observations = slices.DeleteFunc(graph.Entities[entityIndex].Observations, func(observation string) bool {
			return observationsToDelete[observation]
		})
	}
	return k.saveGraph(graph)
}
func (k knowledgeBase) deleteRelations(relations []Relation) error {
	graph, err := k.loadGraph()
	if err != nil {
		return err
	}
	
	graph.Relations = slices.DeleteFunc(graph.Relations, func(existingRelation Relation) bool {
		return slices.ContainsFunc(relations, func(relationToDelete Relation) bool {
			return existingRelation.From == relationToDelete.From &&
				existingRelation.To == relationToDelete.To &&
				existingRelation.RelationType == relationToDelete.RelationType
		})
	})
	return k.saveGraph(graph)
}
func (k knowledgeBase) searchNodes(query string) (KnowledgeGraph, error) {
	graph, err := k.loadGraph()
	if err != nil {
		return KnowledgeGraph{}, err
	}
	queryLower := strings.ToLower(query)
	var filteredEntities []Entity
	
	for _, entity := range graph.Entities {
		if strings.Contains(strings.ToLower(entity.Name), queryLower) ||
			strings.Contains(strings.ToLower(entity.EntityType), queryLower) {
			filteredEntities = append(filteredEntities, entity)
			continue
		}
		
		for _, observation := range entity.Observations {
			if strings.Contains(strings.ToLower(observation), queryLower) {
				filteredEntities = append(filteredEntities, entity)
				break
			}
		}
	}
	
	filteredEntityNames := make(map[string]bool)
	for _, entity := range filteredEntities {
		filteredEntityNames[entity.Name] = true
	}
	
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
func (k knowledgeBase) openNodes(names []string) (KnowledgeGraph, error) {
	graph, err := k.loadGraph()
	if err != nil {
		return KnowledgeGraph{}, err
	}
	
	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}
	
	var filteredEntities []Entity
	for _, entity := range graph.Entities {
		if nameSet[entity.Name] {
			filteredEntities = append(filteredEntities, entity)
		}
	}
	
	filteredEntityNames := make(map[string]bool)
	for _, entity := range filteredEntities {
		filteredEntityNames[entity.Name] = true
	}
	
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
func TestKnowledgeBaseOperations(t *testing.T) {
	for name, newStore := range stores() {
		t.Run(name, func(t *testing.T) {
			s := newStore(t)
			kb := knowledgeBase{s: s}
			
			graph, err := kb.loadGraph()
			if err != nil {
				t.Fatalf("failed to load empty graph: %v", err)
			}
			if len(graph.Entities) != 0 || len(graph.Relations) != 0 {
				t.Errorf("expected empty graph, got %+v", graph)
			}
			
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
			
			graph, err = kb.loadGraph()
			if err != nil {
				t.Fatalf("failed to read graph: %v", err)
			}
			if len(graph.Entities) != 2 {
				t.Errorf("expected 2 entities, got %d", len(graph.Entities))
			}
			
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
			
			searchResult, err := kb.searchNodes("developer")
			if err != nil {
				t.Fatalf("failed to search nodes: %v", err)
			}
			if len(searchResult.Entities) != 1 || searchResult.Entities[0].Name != "Alice" {
				t.Errorf("expected to find Alice when searching for 'developer', got %+v", searchResult)
			}
			
			openResult, err := kb.openNodes([]string{"Bob"})
			if err != nil {
				t.Fatalf("failed to open nodes: %v", err)
			}
			if len(openResult.Entities) != 1 || openResult.Entities[0].Name != "Bob" {
				t.Errorf("expected to find Bob when opening 'Bob', got %+v", openResult)
			}
			
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
			
			err = kb.deleteRelations(testRelations)
			if err != nil {
				t.Fatalf("failed to delete relations: %v", err)
			}
			
			graph, _ = kb.loadGraph()
			if len(graph.Relations) != 0 {
				t.Errorf("expected 0 relations after deletion, got %d", len(graph.Relations))
			}
			
			err = kb.deleteEntities([]string{"Alice"})
			if err != nil {
				t.Fatalf("failed to delete entities: %v", err)
			}
			
			graph, _ = kb.loadGraph()
			if len(graph.Entities) != 1 || graph.Entities[0].Name != "Bob" {
				t.Errorf("expected only Bob to remain after deleting Alice, got %+v", graph.Entities)
			}
		})
	}
}
func TestSaveAndLoadGraph(t *testing.T) {
	for name, newStore := range stores() {
		t.Run(name, func(t *testing.T) {
			s := newStore(t)
			kb := knowledgeBase{s: s}
			
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
			
			err := kb.saveGraph(testGraph)
			if err != nil {
				t.Fatalf("failed to save graph: %v", err)
			}
			
			loadedGraph, err := kb.loadGraph()
			if err != nil {
				t.Fatalf("failed to load graph: %v", err)
			}
			
			if !reflect.DeepEqual(testGraph, loadedGraph) {
				t.Errorf("loaded graph does not match saved graph.\nExpected: %+v\nGot: %+v", testGraph, loadedGraph)
			}
			
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
func TestDuplicateEntitiesAndRelations(t *testing.T) {
	for name, newStore := range stores() {
		t.Run(name, func(t *testing.T) {
			s := newStore(t)
			kb := knowledgeBase{s: s}
			
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
			
			if len(newEntities) != 1 || newEntities[0].Name != "Eve" {
				t.Errorf("expected only 'Eve' to be created, got %+v", newEntities)
			}
			
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
			
			if len(newRelations) != 1 || newRelations[0].From != "Eve" || newRelations[0].To != "Dave" {
				t.Errorf("expected only 'Eve->Dave' relation to be created, got %+v", newRelations)
			}
		})
	}
}
func TestErrorHandling(t *testing.T) {
	t.Run("FileStoreWriteError", func(t *testing.T) {
		
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
			
			_, err := kb.createEntities([]Entity{{Name: "RealEntity"}})
			if err != nil {
				t.Fatalf("failed to create test entity: %v", err)
			}
			
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
func TestFileFormatting(t *testing.T) {
	for name, newStore := range stores() {
		t.Run(name, func(t *testing.T) {
			s := newStore(t)
			kb := knowledgeBase{s: s}
			
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
			
			data, err := s.Read()
			if err != nil {
				t.Fatalf("failed to read from store: %v", err)
			}
			
			var items []kbItem
			err = json.Unmarshal(data, &items)
			if err != nil {
				t.Fatalf("failed to parse store data JSON: %v", err)
			}
			
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
func TestMCPServerIntegration(t *testing.T) {
	for name, newStore := range stores() {
		t.Run(name, func(t *testing.T) {
			s := newStore(t)
			kb := knowledgeBase{s: s}
			
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
				
				want := "entity with name NonExistentEntity not found"
				if !strings.Contains(err.Error(), want) {
					t.Errorf("expected error message to contain '%s', got: %v", want, err)
				}
			}
		})
	}
}
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
	
	if len(result.Content) == 0 {
		t.Errorf("expected Content field to be populated")
	}
	if len(out.Entities) == 0 {
		t.Errorf("expected StructuredContent.Entities to be populated")
	}
	
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
type HiArgs struct {
	Name string `json:"name"`
}
type CreateEntitiesArgs struct {
	Entities []Entity `json:"entities" mcp:"entities to create"`
}
type CreateEntitiesResult struct {
	Entities []Entity `json:"entities"`
}
type CreateRelationsArgs struct {
	Relations []Relation `json:"relations" mcp:"relations to create"`
}
type CreateRelationsResult struct {
	Relations []Relation `json:"relations"`
}
type AddObservationsArgs struct {
	Observations []Observation `json:"observations" mcp:"observations to add"`
}
type AddObservationsResult struct {
	Observations []Observation `json:"observations"`
}
type DeleteEntitiesArgs struct {
	EntityNames []string `json:"entityNames" mcp:"entities to delete"`
}
type DeleteObservationsArgs struct {
	Deletions []Observation `json:"deletions" mcp:"obeservations to delete"`
}
type DeleteRelationsArgs struct {
	Relations []Relation `json:"relations" mcp:"relations to delete"`
}
type SearchNodesArgs struct {
	Query string `json:"query" mcp:"query string"`
}
type OpenNodesArgs struct {
	Names []string `json:"names" mcp:"names of nodes to open"`
}
func main() {
	flag.Parse()
	
	var kbStore store
	kbStore = &memoryStore{}
	if *memoryFilePath != "" {
		kbStore = &fileStore{path: *memoryFilePath}
	}
	kb := knowledgeBase{s: kbStore}
	
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
func main() {
	
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			
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
				
				if ctr, ok := result.(*mcp.CallToolResult); ok {
					logger.Info("tool result",
						"isError", ctr.IsError,
						"structuredContent", ctr.StructuredContent)
				}
			}
			return result, err
		}
	}
	
	server := mcp.NewServer(&mcp.Implementation{Name: "logging-example"}, nil)
	server.AddReceivingMiddleware(loggingMiddleware)
	
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
	
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	
	serverSession, _ := server.Connect(ctx, serverTransport, nil)
	defer serverSession.Close()
	clientSession, _ := client.Connect(ctx, clientTransport, nil)
	defer clientSession.Close()
	
	result, _ := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "greet",
		Arguments: map[string]any{
			"name": "World",
		},
	})
	fmt.Printf("Tool result: %s\n", result.Content[0].(*mcp.TextContent).Text)
	
	
	
	
	
	
	
	
	
}
</content>
</file>
<file path="examples/server/rate-limiting/main.go">
<type>go</type>
<content>
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
func PerSessionRateLimiterMiddleware(limit rate.Limit, burst int) mcp.Middleware {
	
	var (
		sessionLimiters = make(map[string]*rate.Limiter)
		mu              sync.Mutex
	)
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			
			
			
			sessionID := req.GetSession().ID()
			if sessionID == "" {
				
				
				
				log.Printf("Warning: Session ID is empty for method %q. Skipping per-session rate limiting.", method)
				return next(ctx, method, req) 
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
		"callTool":  rate.NewLimiter(rate.Every(time.Second), 5),  
		"listTools": rate.NewLimiter(rate.Every(time.Minute), 20), 
	}))
	server.AddReceivingMiddleware(PerSessionRateLimiterMiddleware(rate.Every(time.Second/5), 10))
	
	log.Println("MCP Server instance created with Middleware (but not running).")
	log.Println("This example demonstrates configuration, not live interaction.")
}
</content>
</file>
<file path="examples/server/sequentialthinking/main.go">
<type>go</type>
<content>
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
type Thought struct {
	
	Index int `json:"index"`
	
	Content string `json:"content"`
	
	Created time.Time `json:"created"`
	
	Revised bool `json:"revised"`
	
	ParentIndex *int `json:"parentIndex,omitempty"`
}
type ThinkingSession struct {
	
	ID string `json:"id"`
	
	Problem string `json:"problem"`
	
	Thoughts []*Thought `json:"thoughts"`
	
	CurrentThought int `json:"currentThought"`
	
	EstimatedTotal int `json:"estimatedTotal"`
	
	Status string `json:"status"` 
	
	Created time.Time `json:"created"`
	
	LastActivity time.Time `json:"lastActivity"`
	
	Branches []string `json:"branches,omitempty"`
	
	Version int `json:"version"`
}
func (s *ThinkingSession) clone() *ThinkingSession {
	sessionCopy := *s
	sessionCopy.Thoughts = deepCopyThoughts(s.Thoughts)
	sessionCopy.Branches = slices.Clone(s.Branches)
	return &sessionCopy
}
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*ThinkingSession 
}
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*ThinkingSession),
	}
}
func (s *SessionStore) Session(id string) (*ThinkingSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, exists := s.sessions[id]
	return session, exists
}
func (s *SessionStore) SetSession(session *ThinkingSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
}
func (s *SessionStore) CompareAndSwap(sessionID string, updateFunc func(*ThinkingSession) (*ThinkingSession, error)) error {
	for {
		
		s.mu.RLock()
		current, exists := s.sessions[sessionID]
		if !exists {
			s.mu.RUnlock()
			return fmt.Errorf("session %s not found", sessionID)
		}
		
		sessionCopy := current.clone()
		oldVersion := current.Version
		s.mu.RUnlock()
		
		updated, err := updateFunc(sessionCopy)
		if err != nil {
			return err
		}
		
		s.mu.Lock()
		current, exists = s.sessions[sessionID]
		if !exists {
			s.mu.Unlock()
			return fmt.Errorf("session %s not found", sessionID)
		}
		if current.Version != oldVersion {
			
			s.mu.Unlock()
			continue
		}
		updated.Version = oldVersion + 1
		s.sessions[sessionID] = updated
		s.mu.Unlock()
		return nil
	}
}
func (s *SessionStore) Sessions() []*ThinkingSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return slices.Collect(maps.Values(s.sessions))
}
func (s *SessionStore) SessionsSnapshot() []*ThinkingSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var sessions []*ThinkingSession
	for _, session := range s.sessions {
		sessions = append(sessions, session.clone())
	}
	return sessions
}
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
type StartThinkingArgs struct {
	Problem        string `json:"problem"`
	SessionID      string `json:"sessionId,omitempty"`
	EstimatedSteps int    `json:"estimatedSteps,omitempty"`
}
type ContinueThinkingArgs struct {
	SessionID      string `json:"sessionId"`
	Thought        string `json:"thought"`
	NextNeeded     *bool  `json:"nextNeeded,omitempty"`
	ReviseStep     *int   `json:"reviseStep,omitempty"`
	CreateBranch   bool   `json:"createBranch,omitempty"`
	EstimatedTotal int    `json:"estimatedTotal,omitempty"`
}
type ReviewThinkingArgs struct {
	SessionID string `json:"sessionId"`
}
type ThinkingHistoryArgs struct {
	SessionID string `json:"sessionId"`
}
func deepCopyThoughts(thoughts []*Thought) []*Thought {
	thoughtsCopy := make([]*Thought, len(thoughts))
	for i, t := range thoughts {
		t2 := *t
		thoughtsCopy[i] = &t2
	}
	return thoughtsCopy
}
func StartThinking(ctx context.Context, _ *mcp.CallToolRequest, args StartThinkingArgs) (*mcp.CallToolResult, any, error) {
	sessionID := args.SessionID
	if sessionID == "" {
		sessionID = randText()
	}
	estimatedSteps := args.EstimatedSteps
	if estimatedSteps == 0 {
		estimatedSteps = 5 
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
func ContinueThinking(ctx context.Context, req *mcp.CallToolRequest, args ContinueThinkingArgs) (*mcp.CallToolResult, any, error) {
	
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
	
	if args.CreateBranch {
		var branchID string
		var branchSession *ThinkingSession
		err := store.CompareAndSwap(args.SessionID, func(session *ThinkingSession) (*ThinkingSession, error) {
			branchID = fmt.Sprintf("%s_branch_%d", args.SessionID, len(session.Branches)+1)
			session.Branches = append(session.Branches, branchID)
			session.LastActivity = time.Now()
			
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
		
		if args.EstimatedTotal > 0 {
			session.EstimatedTotal = args.EstimatedTotal
		}
		
		if args.NextNeeded != nil && !*args.NextNeeded {
			session.Status = "completed"
		}
		
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
func ReviewThinking(ctx context.Context, req *mcp.CallToolRequest, args ReviewThinkingArgs) (*mcp.CallToolResult, any, error) {
	
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
func ThinkingHistory(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	
	u, err := url.Parse(req.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid thinking resource URI: %s", req.Params.URI)
	}
	if u.Scheme != "thinking" {
		return nil, fmt.Errorf("invalid thinking resource URI scheme: %s", u.Scheme)
	}
	sessionID := u.Host
	if sessionID == "sessions" {
		
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
const base32alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
func randText() string {
	
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
		URI:         "thinking:
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
	
	store = NewSessionStore()
	
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
	
	continueArgs := ContinueThinkingArgs{
		SessionID: "test_continue",
		Thought:   "First thought: I need to understand the problem",
	}
	result, _, err := ContinueThinking(ctx, nil, continueArgs)
	if err != nil {
		t.Fatalf("ContinueThinking() error = %v", err)
	}
	
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
	
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}
	if !strings.Contains(textContent.Text, "completed") {
		t.Error("Result should indicate completion")
	}
	
	session, exists := store.Session("test_completion")
	if !exists {
		t.Fatal("Session not found")
	}
	if session.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", session.Status)
	}
}
func TestContinueThinkingRevision(t *testing.T) {
	
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
	
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}
	if !strings.Contains(textContent.Text, "Revised step 1") {
		t.Error("Result should indicate revision")
	}
	
	updatedSession, _ := store.Session("test_revision")
	if updatedSession.Thoughts[0].Content != "Revised first thought" {
		t.Error("First thought should be revised")
	}
	if !updatedSession.Thoughts[0].Revised {
		t.Error("First thought should be marked as revised")
	}
}
func TestReviewThinking(t *testing.T) {
	
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
	
	result, err := ThinkingHistory(ctx, &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "thinking:
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
	
	var sessions []*ThinkingSession
	err = json.Unmarshal([]byte(content.Text), &sessions)
	if err != nil {
		t.Fatalf("Failed to parse sessions JSON: %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}
	
	result, err = ThinkingHistory(ctx, &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: "thinking:
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
	
	continueArgs := ContinueThinkingArgs{
		SessionID: "nonexistent",
		Thought:   "Some thought",
	}
	_, _, err := ContinueThinking(ctx, nil, continueArgs)
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
	
	reviewArgs := ReviewThinkingArgs{
		SessionID: "nonexistent",
	}
	_, _, err = ReviewThinking(ctx, nil, reviewArgs)
	if err == nil {
		t.Error("Expected error for non-existent session in review")
	}
	
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
	reviseStep := 5 
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
package main
import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)
type Input struct {
	Name string `json:"name" jsonschema:"the person to greet"`
}
type Output struct {
	Greeting string `json:"greeting" jsonschema:"the greeting to send to the user"`
}
func simpleGreeting(_ context.Context, _ *mcp.CallToolRequest, input Input) (*mcp.CallToolResult, Output, error) {
	return nil, Output{"Hi " + input.Name}, nil
}
type manualGreeter struct {
	inputSchema  *jsonschema.Resolved
	outputSchema *jsonschema.Resolved
}
func (t *manualGreeter) greet(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	
	errf := func(format string, args ...any) *mcp.CallToolResult {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf(format, args...)}},
			IsError: true,
		}
	}
	
	
	
	
	if err := unmarshalAndValidate(req.Params.Arguments, t.inputSchema); err != nil {
		return errf("invalid input: %v", err), nil
	}
	
	var input Input
	if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
		return errf("failed to unmarshal arguments: %v", err), nil
	}
	output := Output{Greeting: "Hi " + input.Name}
	outputJSON, err := json.Marshal(output)
	if err != nil {
		return errf("output failed to marshal: %v", err), nil
	}
	
	if err := unmarshalAndValidate(outputJSON, t.outputSchema); err != nil {
		return errf("invalid output: %v", err), nil
	}
	return &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: string(outputJSON)}},
		StructuredContent: output,
	}, nil
}
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
	
	
	
	
	mcp.AddTool(server, &mcp.Tool{Name: "simple greeting"}, simpleGreeting)
	
	
	
	manual, err := newManualGreeter()
	if err != nil {
		log.Fatal(err)
	}
	server.AddTool(&mcp.Tool{
		Name:         "manual greeting",
		InputSchema:  inputSchema,
		OutputSchema: outputSchema,
	}, manual.greet)
	
	
	server.AddTool(&mcp.Tool{
		Name:        "unvalidated greeting",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"user":{"type":"string"}}}`),
	}, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		
		var args struct{ User string }
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Hi " + args.User}},
		}, nil
	})
	
	
	
	mcp.AddTool(server, &mcp.Tool{
		Name:        "customized greeting 1",
		InputSchema: inputSchema,
		
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
	
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Printf("Server failed: %v", err)
	}
}
</content>
</file>
<file path="examples/server/toolschemas/main_test.go">
<type>go</type>
<content>
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
package docs
</content>
</file>
<file path="internal/jsonrpc2/conn.go">
<type>go</type>
<content>
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
type Binder interface {
	
	
	
	
	
	Bind(context.Context, *Connection) ConnectionOptions
}
type BinderFunc func(context.Context, *Connection) ConnectionOptions
func (f BinderFunc) Bind(ctx context.Context, c *Connection) ConnectionOptions {
	return f(ctx, c)
}
var _ Binder = BinderFunc(nil)
type ConnectionOptions struct {
	
	
	Framer Framer
	
	
	Preempter Preempter
	
	
	Handler Handler
	
	
	
	OnInternalError func(error)
}
type Connection struct {
	seq int64 
	stateMu sync.Mutex
	state   inFlightState 
	done    chan struct{} 
	writer  Writer
	handler Handler
	onInternalError func(error)
	onDone          func()
}
type inFlightState struct {
	connClosing bool  
	reading     bool  
	readErr     error 
	writeErr    error 
	
	
	
	
	
	
	
	closer   io.Closer
	closeErr error 
	outgoingCalls         map[ID]*AsyncCall 
	outgoingNotifications int               
	
	
	incoming int
	incomingByID map[ID]*incomingRequest 
	
	
	
	handlerQueue   []*incomingRequest
	handlerRunning bool
}
func (c *Connection) updateInFlight(f func(*inFlightState)) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	s := &c.state
	f(s)
	select {
	case <-c.done:
		
		
		
		
		if !s.idle() {
			panic("jsonrpc2: updateInFlight transitioned to non-idle when already done")
		}
		return
	default:
	}
	if s.idle() && s.shuttingDown(ErrUnknown) != nil {
		if s.closer != nil {
			s.closeErr = s.closer.Close()
			s.closer = nil 
		}
		if s.reading {
			
			
			
		} else {
			
			
			if c.onDone != nil {
				c.onDone()
			}
			close(c.done)
		}
	}
}
func (s *inFlightState) idle() bool {
	return len(s.outgoingCalls) == 0 && s.outgoingNotifications == 0 && s.incoming == 0 && !s.handlerRunning
}
func (s *inFlightState) shuttingDown(errClosing error) error {
	if s.connClosing {
		
		
		
		return errClosing
	}
	if s.readErr != nil {
		
		
		return fmt.Errorf("%w: %v", errClosing, s.readErr)
	}
	if s.writeErr != nil {
		
		
		return fmt.Errorf("%w: %v", errClosing, s.writeErr)
	}
	return nil
}
type incomingRequest struct {
	*Request 
	ctx      context.Context
	cancel   context.CancelFunc
}
func (o ConnectionOptions) Bind(context.Context, *Connection) ConnectionOptions {
	return o
}
type ConnectionConfig struct {
	Reader          Reader                    
	Writer          Writer                    
	Closer          io.Closer                 
	Preempter       Preempter                 
	Bind            func(*Connection) Handler 
	OnDone          func()                    
	OnInternalError func(error)               
}
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
func bindConnection(bindCtx context.Context, rwc io.ReadWriteCloser, binder Binder, onDone func()) *Connection {
	
	
	ctx := notDone{bindCtx}
	c := &Connection{
		state:  inFlightState{closer: rwc},
		done:   make(chan struct{}),
		onDone: onDone,
	}
	
	
	
	
	
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
			
			return
		default:
		}
		
		
		
		
		s.reading = true
		go c.readIncoming(ctx, reader, preempter)
	})
}
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
func (c *Connection) Call(ctx context.Context, method string, params any) *AsyncCall {
	
	id := Int64ID(atomic.AddInt64(&c.seq, 1))
	ac := &AsyncCall{
		id:    id,
		ready: make(chan struct{}),
	}
	
	
	
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
		
		
		c.updateInFlight(func(s *inFlightState) {
			if s.outgoingCalls[ac.id] == ac {
				delete(s.outgoingCalls, ac.id)
				ac.retire(&Response{ID: id, Error: err})
			} else {
				
				
			}
		})
	}
	return ac
}
func Async(ctx context.Context) {
	if r, ok := ctx.Value(asyncKey).(*releaser); ok {
		r.release(false)
	}
}
type asyncKeyType struct{}
var asyncKey = asyncKeyType{}
type releaser struct {
	mu       sync.Mutex
	ch       chan struct{}
	released bool
}
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
	ready    chan struct{} 
	response *Response
}
func (ac *AsyncCall) ID() ID { return ac.id }
func (ac *AsyncCall) IsReady() bool {
	select {
	case <-ac.ready:
		return true
	default:
		return false
	}
}
func (ac *AsyncCall) retire(response *Response) {
	select {
	case <-ac.ready:
		panic(fmt.Sprintf("jsonrpc2: retire called twice for ID %v", ac.id))
	default:
	}
	ac.response = response
	close(ac.ready)
}
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
func (c *Connection) Cancel(id ID) {
	var req *incomingRequest
	c.updateInFlight(func(s *inFlightState) {
		req = s.incomingByID[id]
	})
	if req != nil {
		req.cancel()
	}
}
func (c *Connection) Wait() error {
	return c.wait(true)
}
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
func (c *Connection) Close() error {
	
	
	c.updateInFlight(func(s *inFlightState) { s.connClosing = true })
	return c.wait(false)
}
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
					
				}
			})
		default:
			c.internalErrorf("Read returned an unexpected message of type %T", msg)
		}
	}
	c.updateInFlight(func(s *inFlightState) {
		s.reading = false
		s.readErr = err
		
		
		for id, ac := range s.outgoingCalls {
			ac.retire(&Response{ID: id, Error: err})
		}
		s.outgoingCalls = nil
	})
}
func (c *Connection) acceptRequest(ctx context.Context, msg *Request, preempter Preempter) {
	
	
	reqCtx, cancel := context.WithCancel(ctx)
	req := &incomingRequest{
		Request: msg,
		ctx:     reqCtx,
		cancel:  cancel,
	}
	
	
	var err error
	c.updateInFlight(func(s *inFlightState) {
		s.incoming++
		if req.IsCall() {
			if s.incomingByID[req.ID] != nil {
				err = fmt.Errorf("%w: request ID %v already in use", ErrInvalidRequest, req.ID)
				req.ID = ID{} 
				return
			}
			if s.incomingByID == nil {
				s.incomingByID = make(map[ID]*incomingRequest)
			}
			s.incomingByID[req.ID] = req
			
			
			
			
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
		
		
		
		
		err = s.shuttingDown(ErrServerClosing)
		if err != nil {
			return
		}
		
		
		
		
		
		
		s.handlerQueue = append(s.handlerQueue, req)
		if !s.handlerRunning {
			
			
			
			
			
			
			
			
			
			
			
			
			s.handlerRunning = true
			go c.handleAsync()
		}
	})
	if err != nil {
		c.processResult("acceptRequest", req, nil, err)
	}
}
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
		
		if err := req.ctx.Err(); err != nil {
			c.updateInFlight(func(s *inFlightState) {
				if s.writeErr != nil {
					
					
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
func (c *Connection) processResult(from any, req *incomingRequest, result any, err error) error {
	switch err {
	case ErrNotHandled, ErrMethodNotFound:
		
		err = fmt.Errorf("%w: %q", ErrMethodNotFound, req.Method)
	}
	if result != nil && err != nil {
		c.internalErrorf("%#v returned a non-nil result with a non-nil error for %s:\n%v\n%#v", from, req.Method, err, result)
		result = nil 
	}
	if req.IsCall() {
		if result == nil && err == nil {
			err = c.internalErrorf("%#v returned a nil result and nil error for a %q Request that requires a Response", from, req.Method)
		}
		response, respErr := NewResponse(req.ID, result, err)
		
		
		
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
	} else { 
		if result != nil {
			err = c.internalErrorf("%#v returned a non-nil result for a %q Request without an ID", from, req.Method)
		} else if err != nil {
			err = fmt.Errorf("%w: %q notification failed: %v", ErrInternal, req.Method, err)
		}
	}
	if err != nil {
		
		
	}
	
	req.cancel()
	c.updateInFlight(func(s *inFlightState) {
		if s.incoming == 0 {
			panic("jsonrpc2: processResult called when incoming count is already zero")
		}
		s.incoming--
	})
	return nil
}
func (c *Connection) write(ctx context.Context, msg Message) error {
	var err error
	
	
	
	
	c.updateInFlight(func(s *inFlightState) {
		err = s.shuttingDown(ErrServerClosing)
	})
	if err == nil {
		err = c.writer.Write(ctx, msg)
	}
	
	
	if errors.Is(err, ErrRejected) {
		return err
	}
	if err != nil && ctx.Err() == nil {
		
		
		
		
		
		
		
		
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
func (c *Connection) internalErrorf(format string, args ...any) error {
	err := fmt.Errorf(format, args...)
	if c.onInternalError == nil {
		panic("jsonrpc2: " + err.Error())
	}
	c.onInternalError(err)
	return fmt.Errorf("%w: %v", ErrInternal, err)
}
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
type Reader interface {
	
	Read(context.Context) (Message, error)
}
type Writer interface {
	
	Write(context.Context, Message) error
}
type Framer interface {
	
	Reader(io.Reader) Reader
	
	Writer(io.Writer) Writer
}
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
	firstRead := true 
	var contentLength int64
	
	for {
		line, err := r.in.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				if firstRead && line == "" {
					return nil, io.EOF 
				}
				err = io.ErrUnexpectedEOF
			}
			return nil, fmt.Errorf("failed reading header line: %w", err)
		}
		firstRead = false
		line = strings.TrimSpace(line)
		
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
package jsonrpc2
import (
	"context"
	"errors"
)
var (
	
	ErrIdleTimeout = errors.New("timed out waiting for new connections")
	
	
	
	
	
	ErrNotHandled = errors.New("JSON RPC not handled")
)
type Preempter interface {
	
	
	
	
	
	
	
	
	Preempt(ctx context.Context, req *Request) (result any, err error)
}
type PreempterFunc func(ctx context.Context, req *Request) (any, error)
func (f PreempterFunc) Preempt(ctx context.Context, req *Request) (any, error) {
	return f(ctx, req)
}
var _ Preempter = PreempterFunc(nil)
type Handler interface {
	
	
	
	
	
	
	
	
	
	
	
	
	
	Handle(ctx context.Context, req *Request) (result any, err error)
}
type defaultHandler struct{}
func (defaultHandler) Preempt(context.Context, *Request) (any, error) {
	return nil, ErrNotHandled
}
func (defaultHandler) Handle(context.Context, *Request) (any, error) {
	return nil, ErrNotHandled
}
type HandlerFunc func(ctx context.Context, req *Request) (any, error)
func (f HandlerFunc) Handle(ctx context.Context, req *Request) (any, error) {
	return f(ctx, req)
}
var _ Handler = HandlerFunc(nil)
type async struct {
	ready    chan struct{} 
	firstErr chan error    
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
		call{"peek", nil, 0}, 
		notify{"unblock", "a"},
		collect{"a", true, false},
		call{"get", nil, 10}, 
	}},
	sequence{"fork", []invoker{
		async{"a", "fork", "a"},
		notify{"set", 1},
		notify{"add", 2},
		notify{"add", 3},
		notify{"add", 4},
		call{"get", nil, 10}, 
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
					
					time.Sleep(50 * time.Millisecond)
					defer h.conn.Close()
					test.Invoke(t, ctx, h)
					if call, ok := test.(*call); ok {
						
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
package jsonrpc2
import (
	"encoding/json"
	"errors"
	"fmt"
)
type ID struct {
	value any
}
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
type Message interface {
	
	
	marshal(to *wireCombined)
}
type Request struct {
	
	
	ID ID
	
	Method string
	
	Params json.RawMessage
	
	
	Extra any
}
type Response struct {
	
	Result json.RawMessage
	
	Error error
	
	ID ID
	
	
	Extra any
}
func StringID(s string) ID { return ID{value: s} }
func Int64ID(i int64) ID { return ID{value: i} }
func (id ID) IsValid() bool { return id.value != nil }
func (id ID) Raw() any { return id.value }
func NewNotification(method string, params any) (*Request, error) {
	p, merr := marshalToRaw(params)
	return &Request{Method: method, Params: p}, merr
}
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
		
		return nil
	}
	if err, ok := err.(*WireError); ok {
		
		return err
	}
	result := &WireError{Message: err.Error()}
	var wrapped *WireError
	if errors.As(err, &wrapped) {
		
		
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
		
		return &Request{
			Method: msg.Method,
			ID:     id,
			Params: msg.Params,
		}, nil
	}
	
	if !id.IsValid() {
		return nil, ErrInvalidRequest
	}
	resp := &Response{
		ID:     id,
		Result: msg.Result,
	}
	
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
package jsonrpc2
import (
	"context"
	"io"
	"net"
	"os"
)
type NetListenOptions struct {
	NetListenConfig net.ListenConfig
	NetDialer       net.Dialer
}
func NetListener(ctx context.Context, network, address string, options NetListenOptions) (Listener, error) {
	ln, err := options.NetListenConfig.Listen(ctx, network, address)
	if err != nil {
		return nil, err
	}
	return &netListener{net: ln}, nil
}
type netListener struct {
	net net.Listener
}
func (l *netListener) Accept(context.Context) (io.ReadWriteCloser, error) {
	return l.net.Accept()
}
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
func (l *netListener) Dialer() Dialer {
	return NetDialer(l.net.Addr().Network(), l.net.Addr().String(), net.Dialer{})
}
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
func NetPipeListener(ctx context.Context) (Listener, error) {
	return &netPiper{
		done:   make(chan struct{}),
		dialed: make(chan io.ReadWriteCloser),
	}, nil
}
type netPiper struct {
	done   chan struct{}
	dialed chan io.ReadWriteCloser
}
func (l *netPiper) Accept(context.Context) (io.ReadWriteCloser, error) {
	
	
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
func (l *netPiper) Close() error {
	
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
type Listener interface {
	
	
	Accept(context.Context) (io.ReadWriteCloser, error)
	
	
	Close() error
	
	
	
	Dialer() Dialer
}
type Dialer interface {
	
	Dial(ctx context.Context) (io.ReadWriteCloser, error)
}
type Server struct {
	listener Listener
	binder   Binder
	async    *async
	shutdownOnce sync.Once
	closing      int32 
}
func Dial(ctx context.Context, dialer Dialer, binder Binder, onDone func()) (*Connection, error) {
	
	rwc, err := dialer.Dial(ctx)
	if err != nil {
		return nil, err
	}
	return bindConnection(ctx, rwc, binder, onDone), nil
}
func NewServer(ctx context.Context, listener Listener, binder Binder) *Server {
	server := &Server{
		listener: listener,
		binder:   binder,
		async:    newAsync(),
	}
	go server.run(ctx)
	return server
}
func (s *Server) Wait() error {
	return s.async.wait()
}
func (s *Server) Shutdown() {
	s.shutdownOnce.Do(func() {
		atomic.StoreInt32(&s.closing, 1)
		s.listener.Close()
	})
}
func (s *Server) run(ctx context.Context) {
	defer s.async.done()
	var activeConns sync.WaitGroup
	for {
		rwc, err := s.listener.Accept(ctx)
		if err != nil {
			
			
			
			if atomic.LoadInt32(&s.closing) == 0 {
				s.async.setError(err)
			}
			
			break
		}
		
		activeConns.Add(1)
		_ = bindConnection(ctx, rwc, s.binder, activeConns.Done) 
	}
	activeConns.Wait()
}
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
	
	active    chan int         
	timedOut  chan struct{}    
	idleTimer chan *time.Timer 
}
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
			
			
			
			
		}
		return l.newConn(rwc), nil
	case <-l.timedOut:
		if err == nil {
			
			
			
			rwc.Close()
		} else {
			
			
			
			
		}
		return nil, ErrIdleTimeout
	case timer := <-l.idleTimer:
		if err != nil {
			
			
			
			
			l.idleTimer <- timer
			return nil, err
		}
		if !timer.Stop() {
			
			
			
			
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
		
		
		
		return ErrIdleTimeout
	case timer := <-l.idleTimer:
		if !timer.Stop() {
			
			
			
			
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
		
	}
	
	
	defer close(l.timedOut)
	l.wrapped.Close()
}
func (l *idleListener) connClosed() {
	select {
	case n, ok := <-l.active:
		if !ok {
			
			
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
func needsLocalhostNet(t testing.TB) {
	switch runtime.GOOS {
	case "js", "wasip1":
		t.Skipf(`Listening on "localhost" fails on %s; see https:
	}
}
func TestIdleTimeout(t *testing.T) {
	needsLocalhostNet(t)
	
	
	
	
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
		
		
		conn1, err := jsonrpc2.Dial(ctx, listener.Dialer(), jsonrpc2.ConnectionOptions{}, nil)
		if err != nil {
			if since := time.Since(idleStart); since < d {
				t.Fatalf("conn1 failed to connect after %v: %v", since, err)
			}
			t.Log("jsonrpc2.Dial:", err)
			return false 
		}
		
		
		
		ac := conn1.Call(ctx, "ping", nil)
		if err := ac.Await(ctx, nil); !errors.Is(err, jsonrpc2.ErrMethodNotFound) {
			if since := time.Since(idleStart); since < d {
				t.Fatalf("conn1 broken after %v: %v", since, err)
			}
			t.Log(`conn1.Call(ctx, "ping", nil):`, err)
			conn1.Close()
			return false
		}
		
		
		
		conn2, err := jsonrpc2.Dial(ctx, listener.Dialer(), jsonrpc2.ConnectionOptions{}, nil)
		if err != nil {
			conn1.Close()
			t.Fatalf("conn2 failed to connect while non-idle after %v: %v", time.Since(idleStart), err)
			return false
		}
		
		
		
		
		
		
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
			return false 
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
func TestIdleListenerAcceptCloseRace(t *testing.T) {
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
		
		
		
		c, err := listener.Accept(ctx)
		if err == nil {
			c.Close()
		}
		<-done
	}
}
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
		
		
		if err := dialConn.Call(ctx, "ping", nil).Await(ctx, nil); err != nil {
			t.Error(err)
		}
		if err := dialConn.Close(); err != nil {
			t.Error(err)
		}
		
		
		pokeCall := <-pokec
		if err := pokeCall.Await(ctx, nil); err == nil {
			t.Errorf("unexpected nil error from server-initited call")
		} else if errors.Is(err, jsonrpc2.ErrMethodNotFound) {
			
		} else {
			
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
package jsonrpc2
import (
	"encoding/json"
)
var (
	
	ErrParse = NewError(-32700, "parse error")
	
	ErrInvalidRequest = NewError(-32600, "invalid request")
	
	
	ErrMethodNotFound = NewError(-32601, "method not found")
	
	
	ErrInvalidParams = NewError(-32602, "invalid params")
	
	ErrInternal = NewError(-32603, "internal error")
	
	
	
	
	ErrServerOverloaded = NewError(-32000, "overloaded")
	
	ErrUnknown = NewError(-32001, "unknown error")
	
	ErrServerClosing = NewError(-32004, "server is closing")
	
	ErrClientClosing = NewError(-32003, "client is closing")
	
	
	
	
	
	
	
	
	ErrRejected = NewError(-32004, "rejected by transport")
)
const wireVersion = "2.0"
type wireCombined struct {
	VersionTag string          `json:"jsonrpc"`
	ID         any             `json:"id,omitempty"`
	Method     string          `json:"method,omitempty"`
	Params     json.RawMessage `json:"params,omitempty"`
	Result     json.RawMessage `json:"result,omitempty"`
	Error      *WireError      `json:"error,omitempty"`
}
type WireError struct {
	
	Code int64 `json:"code"`
	
	Message string `json:"message"`
	
	Data json.RawMessage `json:"data,omitempty"`
}
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
type AuthServerMeta struct {
	
	
	Issuer string `json:"issuer"`
	
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	
	TokenEndpoint string `json:"token_endpoint"`
	
	JWKSURI string `json:"jwks_uri"`
	
	RegistrationEndpoint string `json:"registration_endpoint,omitempty"`
	
	
	ScopesSupported []string `json:"scopes_supported,omitempty"`
	
	
	ResponseTypesSupported []string `json:"response_types_supported"`
	
	
	ResponseModesSupported []string `json:"response_modes_supported,omitempty"`
	
	
	GrantTypesSupported []string `json:"grant_types_supported,omitempty"`
	
	
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported,omitempty"`
	
	
	
	TokenEndpointAuthSigningAlgValuesSupported []string `json:"token_endpoint_auth_signing_alg_values_supported,omitempty"`
	
	
	ServiceDocumentation string `json:"service_documentation,omitempty"`
	
	
	UILocalesSupported []string `json:"ui_locales_supported,omitempty"`
	
	
	OpPolicyURI string `json:"op_policy_uri,omitempty"`
	
	
	OpTOSURI string `json:"op_tos_uri,omitempty"`
	
	RevocationEndpoint string `json:"revocation_endpoint,omitempty"`
	
	
	RevocationEndpointAuthMethodsSupported []string `json:"revocation_endpoint_auth_methods_supported,omitempty"`
	
	
	
	RevocationEndpointAuthSigningAlgValuesSupported []string `json:"revocation_endpoint_auth_signing_alg_values_supported,omitempty"`
	
	IntrospectionEndpoint string `json:"introspection_endpoint,omitempty"`
	
	
	IntrospectionEndpointAuthMethodsSupported []string `json:"introspection_endpoint_auth_methods_supported,omitempty"`
	
	
	
	IntrospectionEndpointAuthSigningAlgValuesSupported []string `json:"introspection_endpoint_auth_signing_alg_values_supported,omitempty"`
	
	
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported,omitempty"`
}
type ClientRegistrationMetadata struct {
	
	
	RedirectURIs []string `json:"redirect_uris"`
	
	
	
	TokenEndpointAuthMethod string `json:"token_endpoint_auth_method,omitempty"`
	
	
	
	GrantTypes []string `json:"grant_types,omitempty"`
	
	
	
	ResponseTypes []string `json:"response_types,omitempty"`
	
	
	ClientName string `json:"client_name,omitempty"`
	
	ClientURI string `json:"client_uri,omitempty"`
	
	
	LogoURI string `json:"logo_uri,omitempty"`
	
	
	Scope string `json:"scope,omitempty"`
	
	
	Contacts []string `json:"contacts,omitempty"`
	
	
	TOSURI string `json:"tos_uri,omitempty"`
	
	
	PolicyURI string `json:"policy_uri,omitempty"`
	
	
	JWKSURI string `json:"jwks_uri,omitempty"`
	
	
	JWKS string `json:"jwks,omitempty"`
	
	
	SoftwareID string `json:"software_id,omitempty"`
	
	SoftwareVersion string `json:"software_version,omitempty"`
	
	
	SoftwareStatement string `json:"software_statement,omitempty"`
}
type ClientRegistrationResponse struct {
	
	
	ClientRegistrationMetadata
	
	ClientID string `json:"client_id"`
	
	ClientSecret string `json:"client_secret,omitempty"`
	
	ClientIDIssuedAt time.Time `json:"client_id_issued_at,omitempty"`
	
	
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
type ClientRegistrationError struct {
	
	ErrorCode string `json:"error"`
	
	ErrorDescription string `json:"error_description,omitempty"`
}
func (e *ClientRegistrationError) Error() string {
	return fmt.Sprintf("registration failed: %s (%s)", e.ErrorCode, e.ErrorDescription)
}
var wellKnownPaths = []string{
	"/.well-known/oauth-authorization-server",
	"/.well-known/openid-configuration",
}
func GetAuthServerMeta(ctx context.Context, issuerURL string, c *http.Client) (*AuthServerMeta, error) {
	var errs []error
	for _, p := range wellKnownPaths {
		u, err := prependToPath(issuerURL, p)
		if err != nil {
			
			return nil, err
		}
		asm, err := getJSON[AuthServerMeta](ctx, c, u, 1<<20)
		if err == nil {
			if asm.Issuer != issuerURL { 
				
				return nil, fmt.Errorf("metadata issuer %q does not match issuer URL %q", asm.Issuer, issuerURL)
			}
			return asm, nil
		}
		errs = append(errs, err)
	}
	return nil, fmt.Errorf("failed to get auth server metadata from %q: %w", issuerURL, errors.Join(errs...))
}
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
	
	data, err := os.ReadFile(filepath.FromSlash("testdata/google-auth-meta.json"))
	if err != nil {
		t.Fatal(err)
	}
	var a AuthServerMeta
	if err := json.Unmarshal(data, &a); err != nil {
		t.Fatal(err)
	}
	
	if g, w := a.Issuer, "https:
		t.Errorf("got %q, want %q", g, w)
	}
}
func TestClientRegistrationMetadataParse(t *testing.T) {
	
	data, err := os.ReadFile(filepath.FromSlash("testdata/client-auth-meta.json"))
	if err != nil {
		t.Fatal(err)
	}
	var a ClientRegistrationMetadata
	if err := json.Unmarshal(data, &a); err != nil {
		t.Fatal(err)
	}
	
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
			clientMeta:   &ClientRegistrationMetadata{ClientName: "Test App", RedirectURIs: []string{"http:
			wantClientID: "test-client-id",
		},
		{
			name: "Missing ClientID in Response",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"client_secret":"test-client-secret"}`)) 
			},
			clientMeta: &ClientRegistrationMetadata{RedirectURIs: []string{"http:
			wantErr:    "registration response is missing required 'client_id' field",
		},
		{
			name: "Standard OAuth Error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"invalid_redirect_uri","error_description":"Redirect URI is not valid."}`))
			},
			clientMeta: &ClientRegistrationMetadata{RedirectURIs: []string{"http:
			wantErr:    "registration failed: invalid_redirect_uri (Redirect URI is not valid.)",
		},
		{
			name: "Non-JSON Server Error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Internal Server Error"))
			},
			clientMeta: &ClientRegistrationMetadata{RedirectURIs: []string{"http:
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
	
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status %s", res.Status)
	}
	
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
type ProtectedResourceMetadata struct {
	
	
	
	Resource string `json:"resource"`
	
	
	
	AuthorizationServers []string `json:"authorization_servers,omitempty"`
	
	
	
	JWKSURI string `json:"jwks_uri,omitempty"`
	
	
	
	ScopesSupported []string `json:"scopes_supported,omitempty"`
	
	
	
	BearerMethodsSupported []string `json:"bearer_methods_supported,omitempty"`
	
	
	
	ResourceSigningAlgValuesSupported []string `json:"resource_signing_alg_values_supported,omitempty"`
	
	
	
	ResourceName string `json:"resource_name,omitempty"`
	
	
	
	ResourceDocumentation string `json:"resource_documentation,omitempty"`
	
	
	
	ResourcePolicyURI string `json:"resource_policy_uri,omitempty"`
	
	
	ResourceTOSURI string `json:"resource_tos_uri,omitempty"`
	
	
	
	TLSClientCertificateBoundAccessTokens bool `json:"tls_client_certificate_bound_access_tokens,omitempty"`
	
	
	
	AuthorizationDetailsTypesSupported []string `json:"authorization_details_types_supported,omitempty"`
	
	
	
	DPOPSigningAlgValuesSupported []string `json:"dpop_signing_alg_values_supported,omitempty"`
	
	
	
	DPOPBoundAccessTokensRequired bool `json:"dpop_bound_access_tokens_required,omitempty"`
	
	
	
	
	
	
}
func GetProtectedResourceMetadataFromID(ctx context.Context, resourceID string, c *http.Client) (_ *ProtectedResourceMetadata, err error) {
	defer util.Wrapf(&err, "GetProtectedResourceMetadataFromID(%q)", resourceID)
	u, err := url.Parse(resourceID)
	if err != nil {
		return nil, err
	}
	
	u.Path = path.Join(defaultProtectedResourceMetadataURI, u.Path)
	return getPRM(ctx, u.String(), c, resourceID)
}
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
func getPRM(ctx context.Context, purl string, c *http.Client, wantResource string) (*ProtectedResourceMetadata, error) {
	if !strings.HasPrefix(strings.ToUpper(purl), "HTTPS:
		return nil, fmt.Errorf("resource URL %q does not use HTTPS", purl)
	}
	prm, err := getJSON[ProtectedResourceMetadata](ctx, c, purl, 1<<20)
	if err != nil {
		return nil, err
	}
	
	if prm.Resource != wantResource {
		return nil, fmt.Errorf("got metadata resource %q, want %q", prm.Resource, wantResource)
	}
	
	for _, u := range prm.AuthorizationServers {
		if err := checkURLScheme(u); err != nil {
			return nil, err
		}
	}
	return prm, nil
}
type challenge struct {
	
	
	
	
	Scheme string
	
	
	Params map[string]string
}
func ResourceMetadataURL(cs []challenge) string {
	for _, c := range cs {
		if u := c.Params["resource_metadata"]; u != "" {
			return u
		}
	}
	return ""
}
func ParseWWWAuthenticate(headers []string) ([]challenge, error) {
	
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
func splitChallenges(header string) ([]string, error) {
	
	var challenges []string
	inQuotes := false
	start := 0
	for i, r := range header {
		if r == '"' {
			if i > 0 && header[i-1] != '\\' {
				inQuotes = !inQuotes
			} else if i == 0 {
				
				
				return nil, errors.New(`challenge begins with '"'`)
			}
		} else if r == ',' && !inQuotes {
			
			
			
			lookahead := strings.TrimSpace(header[i+1:])
			eqPos := strings.Index(lookahead, "=")
			isParam := false
			if eqPos > 0 {
				
				token := lookahead[:eqPos]
				if strings.IndexFunc(token, unicode.IsSpace) == -1 {
					isParam = true
				}
			}
			if !isParam {
				
				
				challenges = append(challenges, header[start:i])
				start = i + 1
			}
		}
	}
	
	challenges = append(challenges, header[start:])
	return challenges, nil
}
func parseSingleChallenge(s string) (challenge, error) {
	
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
	
	for paramsStr != "" {
		
		keyEnd := strings.Index(paramsStr, "=")
		if keyEnd <= 0 {
			return challenge{}, fmt.Errorf("malformed auth parameter: expected key=value, but got %q", paramsStr)
		}
		key := strings.TrimSpace(paramsStr[:keyEnd])
		
		paramsStr = strings.TrimSpace(paramsStr[keyEnd+1:])
		var value string
		if strings.HasPrefix(paramsStr, "\"") {
			
			paramsStr = paramsStr[1:] 
			var valBuilder strings.Builder
			i := 0
			for ; i < len(paramsStr); i++ {
				
				if paramsStr[i] == '\\' && i+1 < len(paramsStr) {
					valBuilder.WriteByte(paramsStr[i+1])
					i++ 
				} else if paramsStr[i] == '"' {
					
					break
				} else {
					valBuilder.WriteByte(paramsStr[i])
				}
			}
			
			if i == len(paramsStr) {
				return challenge{}, fmt.Errorf("unterminated quoted string in auth parameter")
			}
			value = valBuilder.String()
			
			paramsStr = strings.TrimSpace(paramsStr[i+1:])
		} else {
			
			commaPos := strings.Index(paramsStr, ",")
			if commaPos == -1 {
				value = paramsStr
				paramsStr = ""
			} else {
				value = strings.TrimSpace(paramsStr[:commaPos])
				paramsStr = strings.TrimSpace(paramsStr[commaPos:]) 
			}
		}
		if value == "" {
			return challenge{}, fmt.Errorf("no value for auth param %q", key)
		}
		
		params[strings.ToLower(key)] = value
		
		if strings.HasPrefix(paramsStr, ",") {
			paramsStr = strings.TrimSpace(paramsStr[1:])
		} else if paramsStr != "" {
			
			return challenge{}, fmt.Errorf("malformed auth parameter: expected comma after value, but got %q", paramsStr)
		}
	}
	
	return challenge{Scheme: strings.ToLower(scheme), Params: params}, nil
}
</content>
</file>
<file path="internal/readme/client/client.go">
<type>go</type>
<content>
package main
import (
	"context"
	"log"
	"os/exec"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)
func main() {
	ctx := context.Background()
	
	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-client", Version: "v1.0.0"}, nil)
	
	transport := &mcp.CommandTransport{Command: exec.Command("myserver")}
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()
	
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
</content>
</file>
<file path="internal/readme/doc.go">
<type>go</type>
<content>
package readme
</content>
</file>
<file path="internal/readme/server/server.go">
<type>go</type>
<content>
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
	
	server := mcp.NewServer(&mcp.Implementation{Name: "greeter", Version: "v1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "greet", Description: "say hi"}, SayHi)
	
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
</content>
</file>
<file path="internal/testing/fake_auth_server.go">
<type>go</type>
<content>
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
	issuer         = "http:
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
func NewFakeAuthMux() *http.ServeMux {
	s := &state{authCodes: make(map[string]authCodeInfo)}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-authorization-server", s.handleMetadata)
	mux.HandleFunc("/authorize", s.handleAuthorize)
	mux.HandleFunc("/token", s.handleToken)
	return mux
}
func (s *state) handleMetadata(w http.ResponseWriter, r *http.Request) {
	issuer := "https:
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
	
	hasher := sha256.New()
	hasher.Write([]byte(codeVerifier))
	calculatedChallenge := base64.RawURLEncoding.EncodeToString(hasher.Sum(nil))
	if calculatedChallenge != authCodeInfo.codeChallenge {
		http.Error(w, "invalid_grant", http.StatusBadRequest)
		return
	}
	
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
package util
import (
	"cmp"
	"fmt"
	"iter"
	"slices"
)
func Sorted[M ~map[K]V, K cmp.Ordered, V any](m M) iter.Seq2[K, V] {
	
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
func KeySlice[M ~map[K]V, K comparable, V any](m M) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	return r
}
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
package xcontext
import (
	"context"
	"time"
)
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
package jsonrpc
import "github.com/modelcontextprotocol/go-sdk/internal/jsonrpc2"
type (
	
	ID = jsonrpc2.ID
	
	Message = jsonrpc2.Message
	
	Request = jsonrpc2.Request
	
	Response = jsonrpc2.Response
)
func MakeID(v any) (ID, error) {
	return jsonrpc2.MakeID(v)
}
func EncodeMessage(msg Message) ([]byte, error) {
	return jsonrpc2.EncodeMessage(msg)
}
func DecodeMessage(data []byte) (Message, error) {
	return jsonrpc2.DecodeMessage(data)
}
</content>
</file>
<file path="mcp/client.go">
<type>go</type>
<content>
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
type Client struct {
	impl                    *Implementation
	opts                    ClientOptions
	mu                      sync.Mutex
	roots                   *featureSet[*Root]
	sessions                []*ClientSession
	sendingMethodHandler_   MethodHandler
	receivingMethodHandler_ MethodHandler
}
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
type ClientOptions struct {
	
	
	
	
	CreateMessageHandler func(context.Context, *CreateMessageRequest) (*CreateMessageResult, error)
	
	
	
	
	ElicitationHandler func(context.Context, *ElicitRequest) (*ElicitResult, error)
	
	ToolListChangedHandler      func(context.Context, *ToolListChangedRequest)
	PromptListChangedHandler    func(context.Context, *PromptListChangedRequest)
	ResourceListChangedHandler  func(context.Context, *ResourceListChangedRequest)
	ResourceUpdatedHandler      func(context.Context, *ResourceUpdatedNotificationRequest)
	LoggingMessageHandler       func(context.Context, *LoggingMessageRequest)
	ProgressNotificationHandler func(context.Context, *ProgressNotificationClientRequest)
	
	
	
	KeepAlive time.Duration
}
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
func (c *Client) disconnect(cs *ClientSession) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessions = slices.DeleteFunc(c.sessions, func(cs2 *ClientSession) bool {
		return cs2 == cs
	})
}
type unsupportedProtocolVersionError struct {
	version string
}
func (e unsupportedProtocolVersionError) Error() string {
	return fmt.Sprintf("unsupported protocol version: %q", e.version)
}
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
type ClientSession struct {
	onClose func()
	conn            *jsonrpc2.Connection
	client          *Client
	keepaliveCancel context.CancelFunc
	mcpConn         Connection
	
	
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
func (cs *ClientSession) Close() error {
	
	
	
	
	
	if cs.keepaliveCancel != nil {
		cs.keepaliveCancel()
	}
	err := cs.conn.Close()
	if cs.onClose != nil {
		cs.onClose()
	}
	return err
}
func (cs *ClientSession) Wait() error {
	return cs.conn.Wait()
}
func (cs *ClientSession) startKeepalive(interval time.Duration) {
	startKeepalive(cs, interval, &cs.keepaliveCancel)
}
func (c *Client) AddRoots(roots ...*Root) {
	
	if len(roots) == 0 {
		return
	}
	changeAndNotify(c, notificationRootsListChanged, &RootsListChangedParams{},
		func() bool { c.roots.add(roots...); return true })
}
func (c *Client) RemoveRoots(uris ...string) {
	changeAndNotify(c, notificationRootsListChanged, &RootsListChangedParams{},
		func() bool { return c.roots.remove(uris...) })
}
func changeAndNotify[P Params](c *Client, notification string, params P, change func() bool) {
	var sessions []*ClientSession
	
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
		roots = []*Root{} 
	}
	return &ListRootsResult{
		Roots: roots,
	}, nil
}
func (c *Client) createMessage(ctx context.Context, req *CreateMessageRequest) (*CreateMessageResult, error) {
	if c.opts.CreateMessageHandler == nil {
		
		return nil, jsonrpc2.NewError(codeUnsupportedMethod, "client does not support CreateMessage")
	}
	return c.opts.CreateMessageHandler(ctx, req)
}
func (c *Client) elicit(ctx context.Context, req *ElicitRequest) (*ElicitResult, error) {
	if c.opts.ElicitationHandler == nil {
		
		return nil, jsonrpc2.NewError(codeUnsupportedMethod, "client does not support elicitation")
	}
	
	schema, err := validateElicitSchema(req.Params.RequestedSchema)
	if err != nil {
		return nil, jsonrpc2.NewError(codeInvalidParams, err.Error())
	}
	res, err := c.opts.ElicitationHandler(ctx, req)
	if err != nil {
		return nil, err
	}
	
	if schema != nil && res.Content != nil {
		
		
		
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
func validateElicitSchema(wireSchema any) (*jsonschema.Schema, error) {
	if wireSchema == nil {
		return nil, nil 
	}
	var schema *jsonschema.Schema
	if err := remarshal(wireSchema, &schema); err != nil {
		return nil, err
	}
	
	if schema.Type != "" && schema.Type != "object" {
		return nil, fmt.Errorf("elicit schema must be of type 'object', got %q", schema.Type)
	}
	
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
func validateElicitProperty(propName string, propSchema *jsonschema.Schema) error {
	
	if len(propSchema.Properties) > 0 {
		return fmt.Errorf("elicit schema property %q contains nested properties, only primitive properties are allowed", propName)
	}
	
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
func validateElicitStringProperty(propName string, propSchema *jsonschema.Schema) error {
	
	if len(propSchema.Enum) > 0 {
		
		if propSchema.Type != "" && propSchema.Type != "string" {
			return fmt.Errorf("elicit schema property %q has enum values but type is %q, enums are only supported for string type", propName, propSchema.Type)
		}
		
		
		if propSchema.Extra != nil {
			if enumNamesRaw, exists := propSchema.Extra["enumNames"]; exists {
				
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
	
	if propSchema.MinLength != nil {
		if *propSchema.MinLength < 0 {
			return fmt.Errorf("elicit schema property %q has invalid minLength %d, must be non-negative", propName, *propSchema.MinLength)
		}
	}
	
	if propSchema.MaxLength != nil {
		if *propSchema.MaxLength < 0 {
			return fmt.Errorf("elicit schema property %q has invalid maxLength %d, must be non-negative", propName, *propSchema.MaxLength)
		}
		
		if propSchema.MinLength != nil && *propSchema.MaxLength < *propSchema.MinLength {
			return fmt.Errorf("elicit schema property %q has maxLength %d less than minLength %d", propName, *propSchema.MaxLength, *propSchema.MinLength)
		}
	}
	return nil
}
func validateElicitNumberProperty(propName string, propSchema *jsonschema.Schema) error {
	if propSchema.Minimum != nil && propSchema.Maximum != nil {
		if *propSchema.Maximum < *propSchema.Minimum {
			return fmt.Errorf("elicit schema property %q has maximum %g less than minimum %g", propName, *propSchema.Maximum, *propSchema.Minimum)
		}
	}
	return nil
}
func validateElicitBooleanProperty(propName string, propSchema *jsonschema.Schema) error {
	
	if propSchema.Default != nil {
		var defaultValue bool
		if err := json.Unmarshal(propSchema.Default, &defaultValue); err != nil {
			return fmt.Errorf("elicit schema property %q has invalid default value, must be a boolean: %v", propName, err)
		}
	}
	return nil
}
func (c *Client) AddSendingMiddleware(middleware ...Middleware) {
	c.mu.Lock()
	defer c.mu.Unlock()
	addMiddleware(&c.sendingMethodHandler_, middleware)
}
func (c *Client) AddReceivingMiddleware(middleware ...Middleware) {
	c.mu.Lock()
	defer c.mu.Unlock()
	addMiddleware(&c.receivingMethodHandler_, middleware)
}
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
func (cs *ClientSession) getConn() *jsonrpc2.Connection { return cs.conn }
func (*ClientSession) ping(context.Context, *PingParams) (*emptyResult, error) {
	return &emptyResult{}, nil
}
func (*ClientSession) cancel(context.Context, *CancelledParams) (Result, error) {
	return nil, nil
}
func newClientRequest[P Params](cs *ClientSession, params P) *ClientRequest[P] {
	return &ClientRequest[P]{Session: cs, Params: params}
}
func (cs *ClientSession) Ping(ctx context.Context, params *PingParams) error {
	_, err := handleSend[*emptyResult](ctx, methodPing, newClientRequest(cs, orZero[Params](params)))
	return err
}
func (cs *ClientSession) ListPrompts(ctx context.Context, params *ListPromptsParams) (*ListPromptsResult, error) {
	return handleSend[*ListPromptsResult](ctx, methodListPrompts, newClientRequest(cs, orZero[Params](params)))
}
func (cs *ClientSession) GetPrompt(ctx context.Context, params *GetPromptParams) (*GetPromptResult, error) {
	return handleSend[*GetPromptResult](ctx, methodGetPrompt, newClientRequest(cs, orZero[Params](params)))
}
func (cs *ClientSession) ListTools(ctx context.Context, params *ListToolsParams) (*ListToolsResult, error) {
	return handleSend[*ListToolsResult](ctx, methodListTools, newClientRequest(cs, orZero[Params](params)))
}
func (cs *ClientSession) CallTool(ctx context.Context, params *CallToolParams) (*CallToolResult, error) {
	if params == nil {
		params = new(CallToolParams)
	}
	if params.Arguments == nil {
		
		params.Arguments = map[string]any{}
	}
	return handleSend[*CallToolResult](ctx, methodCallTool, newClientRequest(cs, orZero[Params](params)))
}
func (cs *ClientSession) SetLoggingLevel(ctx context.Context, params *SetLoggingLevelParams) error {
	_, err := handleSend[*emptyResult](ctx, methodSetLevel, newClientRequest(cs, orZero[Params](params)))
	return err
}
func (cs *ClientSession) ListResources(ctx context.Context, params *ListResourcesParams) (*ListResourcesResult, error) {
	return handleSend[*ListResourcesResult](ctx, methodListResources, newClientRequest(cs, orZero[Params](params)))
}
func (cs *ClientSession) ListResourceTemplates(ctx context.Context, params *ListResourceTemplatesParams) (*ListResourceTemplatesResult, error) {
	return handleSend[*ListResourceTemplatesResult](ctx, methodListResourceTemplates, newClientRequest(cs, orZero[Params](params)))
}
func (cs *ClientSession) ReadResource(ctx context.Context, params *ReadResourceParams) (*ReadResourceResult, error) {
	return handleSend[*ReadResourceResult](ctx, methodReadResource, newClientRequest(cs, orZero[Params](params)))
}
func (cs *ClientSession) Complete(ctx context.Context, params *CompleteParams) (*CompleteResult, error) {
	return handleSend[*CompleteResult](ctx, methodComplete, newClientRequest(cs, orZero[Params](params)))
}
func (cs *ClientSession) Subscribe(ctx context.Context, params *SubscribeParams) error {
	_, err := handleSend[*emptyResult](ctx, methodSubscribe, newClientRequest(cs, orZero[Params](params)))
	return err
}
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
func (cs *ClientSession) NotifyProgress(ctx context.Context, params *ProgressNotificationParams) error {
	return handleNotify(ctx, notificationProgress, newClientRequest(cs, orZero[Params](params)))
}
func (cs *ClientSession) Tools(ctx context.Context, params *ListToolsParams) iter.Seq2[*Tool, error] {
	if params == nil {
		params = &ListToolsParams{}
	}
	return paginate(ctx, params, cs.ListTools, func(res *ListToolsResult) []*Tool {
		return res.Tools
	})
}
func (cs *ClientSession) Resources(ctx context.Context, params *ListResourcesParams) iter.Seq2[*Resource, error] {
	if params == nil {
		params = &ListResourcesParams{}
	}
	return paginate(ctx, params, cs.ListResources, func(res *ListResourcesResult) []*Resource {
		return res.Resources
	})
}
func (cs *ClientSession) ResourceTemplates(ctx context.Context, params *ListResourceTemplatesParams) iter.Seq2[*ResourceTemplate, error] {
	if params == nil {
		params = &ListResourceTemplatesParams{}
	}
	return paginate(ctx, params, cs.ListResourceTemplates, func(res *ListResourceTemplatesResult) []*ResourceTemplate {
		return res.ResourceTemplates
	})
}
func (cs *ClientSession) Prompts(ctx context.Context, params *ListPromptsParams) iter.Seq2[*Prompt, error] {
	if params == nil {
		params = &ListPromptsParams{}
	}
	return paginate(ctx, params, cs.ListPrompts, func(res *ListPromptsResult) []*Prompt {
		return res.Prompts
	})
}
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
package mcp_test
import (
	"context"
	"fmt"
	"log"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)
func Example_roots() {
	ctx := context.Background()
	
	c := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, nil)
	c.AddRoots(&mcp.Root{URI: "file:
	
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
	
	t1, t2 := mcp.NewInMemoryTransports()
	if _, err := s.Connect(ctx, t1, nil); err != nil {
		log.Fatal(err)
	}
	clientSession, err := c.Connect(ctx, t2, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer clientSession.Close()
	
	c.AddRoots(&mcp.Root{URI: "file:
	<-rootsChanged
	
}
func Example_sampling() {
	ctx := context.Background()
	
	c := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, &mcp.ClientOptions{
		CreateMessageHandler: func(_ context.Context, req *mcp.CreateMessageRequest) (*mcp.CreateMessageResult, error) {
			return &mcp.CreateMessageResult{
				Content: &mcp.TextContent{
					Text: "would have created a message",
				},
			}, nil
		},
	})
	
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
	
}
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
	
}
</content>
</file>
<file path="mcp/client_list_test.go">
<type>go</type>
<content>
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
			r := &mcp.Resource{URI: "http:
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
			rt := &mcp.ResourceTemplate{URITemplate: "http:
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
func generatePaginatedResults(all []*Item, pageSize int) []*ListTestResult {
	if len(all) == 0 {
		return []*ListTestResult{{Items: []*Item{}, NextCursor: ""}}
	}
	if pageSize <= 0 {
		panic("pageSize must be greater than 0")
	}
	numPages := (len(all) + pageSize - 1) / pageSize 
	var results []*ListTestResult
	for i := range numPages {
		startIndex := i * pageSize
		endIndex := min(startIndex+pageSize, len(all)) 
		nextCursor := ""
		if endIndex < len(all) { 
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
package mcp
import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"syscall"
	"time"
)
var defaultTerminateDuration = 5 * time.Second 
type CommandTransport struct {
	Command *exec.Cmd
	
	
	
	TerminateDuration time.Duration
}
func (t *CommandTransport) Connect(ctx context.Context) (Connection, error) {
	stdout, err := t.Command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stdout = io.NopCloser(stdout) 
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
func (s *pipeRWC) Close() error {
	
	
	
	if err := s.stdin.Close(); err != nil {
		return fmt.Errorf("closing stdin: %v", err)
	}
	resChan := make(chan error, 1)
	go func() {
		resChan <- s.cmd.Wait()
	}()
	
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
	
	
	if err := s.cmd.Process.Signal(syscall.SIGTERM); err == nil {
		if err, ok := wait(); ok {
			return err
		}
	}
	
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
package mcp
import "time"
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
	
	onServerExit := make(chan error)
	go func() {
		onServerExit <- server.Run(ctx, serverTransport)
	}()
	
	client := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Ping(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	
	cancel()
	
	
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
	}
}
func TestStdioContextCancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX signals")
	}
	requireExec(t)
	
	
	
	cmd := createServerCommand(t, "cancelContext")
	
	
	_, _ = cmd.StdinPipe()
	
	
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting command: %v", err)
	}
	
	
	
	
	
	
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
			wantMaxDuration: 1 * time.Second, 
		},
		{
			name:            "below minimum duration",
			duration:        -500 * time.Millisecond,
			wantMinDuration: defaultDur,
			wantMaxDuration: 1 * time.Second, 
		},
		{
			name:            "custom valid duration",
			duration:        200 * time.Millisecond,
			wantMinDuration: 200 * time.Millisecond,
			wantMaxDuration: 1 * time.Second, 
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			
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
			
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		})
	}
}
func requireExec(t *testing.T) {
	t.Helper()
	
	
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
type conformanceTest struct {
	name                      string            
	path                      string            
	archive                   *txtar.Archive    
	tools, prompts, resources []string          
	client                    []jsonrpc.Message 
	server                    []jsonrpc.Message 
}
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
			
			
			
			
			
			
			
			
			runSyncTest(t, func(t *testing.T) { runServerTest(t, test) })
			
			
			
			
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
func runServerTest(t *testing.T, test *conformanceTest) {
	ctx := t.Context()
	
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
	
	
	nextResponse := func() (*jsonrpc.Response, error, bool) {
		for {
			msg, err := cStream.Read(ctx)
			if err != nil {
				
				
				
				if errors.Is(err, io.ErrClosedPipe) {
					err = nil
				}
				return nil, err, false
			}
			serverMessages = append(serverMessages, msg)
			if req, ok := msg.(*jsonrpc.Request); ok && req.IsCall() {
				
				
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
	
	for _, req := range outRequests {
		writeMsg(req)
		if req.IsCall() {
			
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
	
	
	
	var extra *jsonrpc.Response
	go func() {
		extra, err, _ = nextResponse()
	}()
	
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
		seenServer := false 
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
	
	loadFeatures := func(data []byte) []string {
		var feats []string
		for line := range strings.Lines(string(data)) {
			if f := strings.TrimSpace(line); f != "" {
				feats = append(feats, f)
			}
		}
		return feats
	}
	seen := make(map[string]bool) 
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
package mcp
import (
	"encoding/json"
	"errors"
	"fmt"
)
type Content interface {
	MarshalJSON() ([]byte, error)
	fromWire(*wireContent)
}
type TextContent struct {
	Text        string
	Meta        Meta
	Annotations *Annotations
}
func (c *TextContent) MarshalJSON() ([]byte, error) {
	
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
type ImageContent struct {
	Meta        Meta
	Annotations *Annotations
	Data        []byte 
	MIMEType    string
}
func (c *ImageContent) MarshalJSON() ([]byte, error) {
	
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
type AudioContent struct {
	Data        []byte
	MIMEType    string
	Meta        Meta
	Annotations *Annotations
}
func (c AudioContent) MarshalJSON() ([]byte, error) {
	
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
type imageAudioWire struct {
	Type        string       `json:"type"`
	MIMEType    string       `json:"mimeType"`
	Data        []byte       `json:"data"`
	Meta        Meta         `json:"_meta,omitempty"`
	Annotations *Annotations `json:"annotations,omitempty"`
}
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
type ResourceContents struct {
	URI      string `json:"uri"`
	MIMEType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     []byte `json:"blob,omitempty"`
	Meta     Meta   `json:"_meta,omitempty"`
}
func (r *ResourceContents) MarshalJSON() ([]byte, error) {
	
	if r.URI == "" {
		return nil, errors.New("ResourceContents missing URI")
	}
	if r.Blob == nil {
		
		type wireResourceContents ResourceContents 
		return json.Marshal((wireResourceContents)(*r))
	}
	
	if r.Text != "" {
		return nil, errors.New("ResourceContents has non-zero Text and Blob fields")
	}
	
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
			
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("UnmarshalJSON panicked: %v", r)
				}
			}()
			err := json.Unmarshal([]byte(tt.json), tt.content)
			if err != nil {
				t.Errorf("UnmarshalJSON failed: %v", err)
			}
			
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
			json:        `{"content":{"type":"resource_link","uri":"file:
			content:     &mcp.CreateMessageResult{},
			expectError: true, 
		},
		{
			name:        "EmbeddedResource",
			json:        `{"content":{"type":"resource","resource":{"uri":"file:
			content:     &mcp.CreateMessageResult{},
			expectError: true, 
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			
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
			expectError: true, 
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			
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
		want string 
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
				Resource: &mcp.ResourceContents{URI: "file:
			},
			`{"type":"resource","resource":{"uri":"file:
		},
		{
			&mcp.EmbeddedResource{
				Resource: &mcp.ResourceContents{URI: "file:
			},
			`{"type":"resource","resource":{"uri":"file:
		},
		{
			&mcp.EmbeddedResource{
				Resource:    &mcp.ResourceContents{URI: "file:
				Meta:        mcp.Meta{"key": "value"},
				Annotations: &mcp.Annotations{Priority: 1.0},
			},
			`{"type":"resource","resource":{"uri":"file:
		},
		{
			&mcp.ResourceLink{
				URI:  "file:
				Name: "file.txt",
			},
			`{"type":"resource_link","uri":"file:
		},
		{
			&mcp.ResourceLink{
				URI:         "https:
				Name:        "Example Resource",
				Title:       "A comprehensive example resource",
				Description: "This resource demonstrates all fields",
				MIMEType:    "text/plain",
				Meta:        mcp.Meta{"custom": "metadata"},
			},
			`{"type":"resource_link","mimeType":"text/plain","uri":"https:
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
		want string 
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
const validateMemoryEventStore = false
type Event struct {
	Name string 
	ID   string 
	Data []byte 
}
func (e Event) Empty() bool {
	return e.Name == "" && e.ID == "" && len(e.Data) == 0
}
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
func scanEvents(r io.Reader) iter.Seq2[Event, error] {
	scanner := bufio.NewScanner(r)
	const maxTokenSize = 1 * 1024 * 1024 
	scanner.Buffer(nil, maxTokenSize)
	
	
	var (
		eventKey = []byte("event")
		idKey    = []byte("id")
		dataKey  = []byte("data")
	)
	return func(yield func(Event, error) bool) {
		
		
		
		
		
		
		
		
		
		var (
			evt     Event
			dataBuf *bytes.Buffer 
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
type EventStore interface {
	
	
	
	
	
	Open(_ context.Context, sessionID, streamID string) error
	
	
	Append(_ context.Context, sessionID, streamID string, data []byte) error
	
	
	
	
	
	
	After(_ context.Context, sessionID, streamID string, index int) iter.Seq2[[]byte, error]
	
	
	
	
	SessionClosed(_ context.Context, sessionID string) error
	
	
}
type dataList struct {
	size  int 
	first int 
	data  [][]byte
}
func (dl *dataList) appendData(d []byte) {
	
	
	
	if len(d) == 0 {
		panic("empty data item")
	}
	dl.data = append(dl.data, d)
	dl.size += len(d)
}
func (dl *dataList) removeFirst() int {
	if len(dl.data) == 0 {
		panic("empty dataList")
	}
	r := len(dl.data[0])
	dl.size -= r
	dl.data[0] = nil 
	dl.data = dl.data[1:]
	dl.first++
	return r
}
type MemoryEventStore struct {
	mu       sync.Mutex
	maxBytes int                             
	nBytes   int                             
	store    map[string]map[string]*dataList 
}
type MemoryEventStoreOptions struct{}
func (s *MemoryEventStore) MaxBytes() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.maxBytes
}
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
const defaultMaxBytes = 10 << 20 
func NewMemoryEventStore(opts *MemoryEventStoreOptions) *MemoryEventStore {
	return &MemoryEventStore{
		maxBytes: defaultMaxBytes,
		store:    make(map[string]map[string]*dataList),
	}
}
func (s *MemoryEventStore) Open(_ context.Context, sessionID, streamID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.init(sessionID, streamID)
	return nil
}
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
func (s *MemoryEventStore) Append(_ context.Context, sessionID, streamID string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	dl := s.init(sessionID, streamID)
	
	
	s.purge()
	dl.appendData(data)
	s.nBytes += len(data)
	return nil
}
var ErrEventsPurged = errors.New("data purged")
func (s *MemoryEventStore) After(_ context.Context, sessionID, streamID string, index int) iter.Seq2[[]byte, error] {
	
	
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
func (s *MemoryEventStore) purge() {
	
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
func (s *MemoryEventStore) validate() {
	if !validateMemoryEventStore {
		return
	}
	
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
		want     string 
		wantSize int    
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
				
				
				s.SetMaxBytes(2)
			},
			
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
				
				
				appendEvent(s, "S1", "2", "d5") 
				appendEvent(s, "S1", "2", "d6") 
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
				
				s.SetMaxBytes(6) 
				appendEvent(s, "S1", "2", "d5")
				appendEvent(s, "S1", "2", "d6")
			},
			
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
	s.Append(ctx, "S1", "2", []byte("d4")) 
	want := "S1 1 first=1 d2 d3; S1 2 first=0 d4"
	if got := s.debugString(); got != want {
		t.Fatalf("got state %q, want %q", got, want)
	}
	for _, tt := range []struct {
		sessionID string
		streamID  string
		index     int
		want      []string
		wantErr   string 
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
package mcp
import (
	"iter"
	"maps"
	"slices"
)
type featureSet[T any] struct {
	uniqueID   func(T) string
	features   map[string]T
	sortedKeys []string 
}
func newFeatureSet[T any](uniqueIDFunc func(T) string) *featureSet[T] {
	return &featureSet[T]{
		uniqueID: uniqueIDFunc,
		features: make(map[string]T),
	}
}
func (s *featureSet[T]) add(fs ...T) {
	for _, f := range fs {
		s.features[s.uniqueID(f)] = f
	}
	s.sortedKeys = nil
}
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
func (s *featureSet[T]) get(uid string) (T, bool) {
	t, ok := s.features[uid]
	return t, ok
}
func (s *featureSet[T]) len() int { return len(s.features) }
func (s *featureSet[T]) all() iter.Seq[T] {
	s.sortKeys()
	return func(yield func(T) bool) {
		s.yieldFrom(0, yield)
	}
}
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
func (s *featureSet[T]) sortKeys() {
	if s.sortedKeys != nil {
		return
	}
	s.sortedKeys = slices.Sorted(maps.Keys(s.features))
}
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
	return "debug" 
}
func mcpLevelToSlog(ll LoggingLevel) slog.Level {
	if sl, ok := mcpToSlog[ll]; ok {
		return sl
	}
	
	return LevelDebug
}
func compareLevels(l1, l2 LoggingLevel) int {
	return cmp.Compare(mcpLevelToSlog(l1), mcpLevelToSlog(l2))
}
type LoggingHandlerOptions struct {
	
	LoggerName string
	
	
	
	MinInterval time.Duration
}
type LoggingHandler struct {
	opts LoggingHandlerOptions
	ss   *ServerSession
	
	
	
	mu              *sync.Mutex
	lastMessageSent time.Time 
	buf             *bytes.Buffer
	handler         slog.Handler
}
func NewLoggingHandler(ss *ServerSession, opts *LoggingHandlerOptions) *LoggingHandler {
	var buf bytes.Buffer
	jsonHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			
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
func (h *LoggingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	
	
	h.ss.mu.Lock()
	mcpLevel := h.ss.state.LogLevel
	h.ss.mu.Unlock()
	return level >= mcpLevelToSlog(mcpLevel)
}
func (h *LoggingHandler) WithAttrs(as []slog.Attr) slog.Handler {
	h2 := *h
	h2.handler = h.handler.WithAttrs(as)
	return &h2
}
func (h *LoggingHandler) WithGroup(name string) slog.Handler {
	h2 := *h
	h2.handler = h.handler.WithGroup(name)
	return &h2
}
func (h *LoggingHandler) Handle(ctx context.Context, r slog.Record) error {
	err := h.handle(ctx, r)
	
	
	return err
}
func (h *LoggingHandler) handle(ctx context.Context, r slog.Record) error {
	
	
	
	h.mu.Lock()
	skip := time.Since(h.lastMessageSent) < h.opts.MinInterval
	h.mu.Unlock()
	if skip {
		return nil
	}
	var err error
	
	
	
	
	
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
	
	
	
	
	return h.ss.Log(ctx, params)
}
</content>
</file>
<file path="mcp/mcp.go">
<type>go</type>
<content>
package mcp
</content>
</file>
<file path="mcp/mcp_example_test.go">
<type>go</type>
<content>
package mcp_test
import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)
func Example_lifecycle() {
	ctx := context.Background()
	
	
	client := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, nil)
	server := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.0.1"}, &mcp.ServerOptions{
		InitializedHandler: func(context.Context, *mcp.InitializedRequest) {
			fmt.Println("initialized!")
		},
	})
	
	
	
	
	t1, t2 := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, t1, nil)
	if err != nil {
		log.Fatal(err)
	}
	clientSession, err := client.Connect(ctx, t2, nil)
	if err != nil {
		log.Fatal(err)
	}
	
	
	if err := clientSession.Close(); err != nil {
		log.Fatal(err)
	}
	if err := serverSession.Wait(); err != nil {
		log.Fatal(err)
	}
	
}
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
				req.Session.NotifyProgress(ctx, params) 
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
	
	
	
	
}
func Example_cancellation() {
	
	
	var clientResult, serverResult string
	var wg sync.WaitGroup
	wg.Add(2)
	
	
	
	server := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.0.1"}, nil)
	started := make(chan struct{}, 1) 
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
	
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer wg.Done()
		_, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "slow"})
		clientResult = fmt.Sprintf("%v", err)
	}()
	
	
	<-started
	cancel()
	wg.Wait()
	fmt.Println(clientResult)
	fmt.Println(serverResult)
	
	
	
}
</content>
</file>
<file path="mcp/mcp_test.go">
<type>go</type>
<content>
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
package mcp
import (
	"context"
)
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
package mcp
import (
	"encoding/json"
	"fmt"
)
type Annotations struct {
	
	
	
	
	Audience []Role `json:"audience,omitempty"`
	
	
	
	
	
	
	LastModified string `json:"lastModified,omitempty"`
	
	
	
	
	
	Priority float64 `json:"priority,omitempty"`
}
type CallToolParams struct {
	
	
	Meta `json:"_meta,omitempty"`
	
	Name string `json:"name"`
	
	
	Arguments any `json:"arguments,omitempty"`
}
type CallToolParamsRaw struct {
	
	
	Meta `json:"_meta,omitempty"`
	
	Name string `json:"name"`
	
	
	
	Arguments json.RawMessage `json:"arguments,omitempty"`
}
type CallToolResult struct {
	
	
	Meta `json:"_meta,omitempty"`
	
	
	
	
	
	
	Content []Content `json:"content"`
	
	
	
	
	
	
	StructuredContent any `json:"structuredContent,omitempty"`
	
	
	
	
	
	
	
... (1065 more lines)
</content>
</file>
<file path="mcp/protocol_test.go">
<type>go</type>
<content>
package mcp
import (
	"encoding/json"
	"maps"
	"testing"
	"github.com/google/go-cmp/cmp"
)
func TestParamsMeta(t *testing.T) {
	
	
	
	toJSON := func(x any) string {
		data, err := json.Marshal(x)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	meta := map[string]any{"m": 1}
	
	p := &CallToolParams{
		Meta: meta,
		Name: "name",
	}
	
	if g, w := toJSON(p), `{"_meta":{"m":1},"name":"name"}`; g != w {
		t.Errorf("got %s, want %s", g, w)
	}
	
	p2 := &CallToolParams{Name: "n"}
	if g, w := toJSON(p2), `{"name":"n"}`; g != w {
		t.Errorf("got %s, want %s", g, w)
	}
	
	if g := p.GetMeta(); !maps.Equal(g, meta) {
		t.Errorf("got %+v, want %+v", g, meta)
	}
	meta2 := map[string]any{"x": 2}
	p.SetMeta(meta2)
	if g := p.GetMeta(); !maps.Equal(g, meta2) {
		t.Errorf("got %+v, want %+v", g, meta2)
	}
	
	if g := p.GetProgressToken(); g != nil {
		t.Errorf("got %v, want nil", g)
	}
	p.SetProgressToken("t")
	if g := p.GetProgressToken(); g != "t" {
		t.Errorf("got %v, want `t`", g)
	}
	
	p.SetProgressToken(int(1))
	p.SetProgressToken(int32(1))
	p.SetProgressToken(int64(1))
}
func TestCompleteReference(t *testing.T) {
	marshalTests := []struct {
		name    string
		in      CompleteReference 
		want    string            
		wantErr bool              
	}{
		{
			name:    "ValidPrompt",
			in:      CompleteReference{Type: "ref/prompt", Name: "my_prompt"},
			want:    `{"type":"ref/prompt","name":"my_prompt"}`,
			wantErr: false,
		},
		{
			name:    "ValidResource",
			in:      CompleteReference{Type: "ref/resource", URI: "file:
			want:    `{"type":"ref/resource","uri":"file:
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
			in:      CompleteReference{Name: "missing"}, 
			wantErr: true,
		},
	}
	
	unmarshalTests := []struct {
		name    string
		in      string            
		want    CompleteReference 
		wantErr bool              
	}{
		{
			name:    "ValidPrompt",
			in:      `{"type":"ref/prompt","name":"my_prompt"}`,
			want:    CompleteReference{Type: "ref/prompt", Name: "my_prompt"},
			wantErr: false,
		},
		{
			name:    "ValidResource",
			in:      `{"type":"ref/resource","uri":"file:
			want:    CompleteReference{Type: "ref/resource", URI: "file:
			wantErr: false,
		},
		
		{
			name:    "UnrecognizedType",
			in:      `{"type":"ref/unknown","name":"something"}`,
			want:    CompleteReference{}, 
			wantErr: true,
		},
		{
			name:    "PromptWithURI",
			in:      `{"type":"ref/prompt","name":"my_prompt","uri":"unexpected_uri"}`,
			want:    CompleteReference{}, 
			wantErr: true,
		},
		{
			name:    "ResourceWithName",
			in:      `{"type":"ref/resource","uri":"my_uri","name":"unexpected_name"}`,
			want:    CompleteReference{}, 
			wantErr: true,
		},
		{
			name:    "MissingType",
			in:      `{"name":"missing"}`,
			want:    CompleteReference{}, 
			wantErr: true,
		},
		{
			name:    "InvalidJSON",
			in:      `invalid json`,
			want:    CompleteReference{}, 
			wantErr: true,                
		},
	}
	
	for _, test := range marshalTests {
		t.Run("Marshal/"+test.name, func(t *testing.T) {
			gotBytes, err := json.Marshal(&test.in)
			if (err != nil) != test.wantErr {
				t.Errorf("json.Marshal(%v) got error %v (want error %t)", test.in, err, test.wantErr)
			}
			if !test.wantErr { 
				if diff := cmp.Diff(test.want, string(gotBytes)); diff != "" {
					t.Errorf("json.Marshal(%v) mismatch (-want +got):\n%s", test.in, diff)
				}
			}
		})
	}
	
	for _, test := range unmarshalTests {
		t.Run("Unmarshal/"+test.name, func(t *testing.T) {
			var got CompleteReference
			err := json.Unmarshal([]byte(test.in), &got)
			if (err != nil) != test.wantErr {
				t.Errorf("json.Unmarshal(%q) got error %v (want error %t)", test.in, err, test.wantErr)
			}
			if !test.wantErr { 
				if diff := cmp.Diff(test.want, got); diff != "" {
					t.Errorf("json.Unmarshal(%q) mismatch (-want +got):\n%s", test.in, diff)
				}
			}
		})
	}
}
func TestCompleteParams(t *testing.T) {
	
	marshalTests := []struct {
		name string
		in   CompleteParams
		want string 
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
					URI:  "file:
				},
				Argument: CompleteParamsArgument{
					Name:  "class",
					Value: "MyClas",
				},
			},
			want: `{"argument":{"name":"class","value":"MyClas"},"ref":{"type":"ref/resource","uri":"file:
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
	
	unmarshalTests := []struct {
		name string
		in   string         
		want CompleteParams 
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
			in:   `{"argument":{"name":"class","value":"MyClas"},"ref":{"type":"ref/resource","uri":"file:
			want: CompleteParams{
				Ref:      &CompleteReference{Type: "ref/resource", URI: "file:
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
			name: "PromptCompletionNilContext", 
			in:   `{"argument":{"name":"language","value":"go"},"context":null,"ref":{"type":"ref/prompt","name":"my_prompt"}}`,
			want: CompleteParams{
				Ref:      &CompleteReference{Type: "ref/prompt", Name: "my_prompt"},
				Argument: CompleteParamsArgument{Name: "language", Value: "go"},
				Context:  nil, 
			},
		},
	}
	
	for _, test := range marshalTests {
		t.Run("Marshal/"+test.name, func(t *testing.T) {
			got, err := json.Marshal(&test.in) 
			if err != nil {
				t.Fatalf("json.Marshal(CompleteParams) failed: %v", err)
			}
			if diff := cmp.Diff(test.want, string(got)); diff != "" {
				t.Errorf("CompleteParams marshal mismatch (-want +got):\n%s", diff)
			}
		})
	}
	
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
	
	marshalTests := []struct {
		name string
		in   CompleteResult
		want string 
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
	
	unmarshalTests := []struct {
		name string
		in   string         
		want CompleteResult 
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
	
	for _, test := range marshalTests {
		t.Run("Marshal/"+test.name, func(t *testing.T) {
			got, err := json.Marshal(&test.in) 
			if err != nil {
				t.Fatalf("json.Marshal(CompleteResult) failed: %v", err)
			}
			if diff := cmp.Diff(test.want, string(got)); diff != "" {
				t.Errorf("CompleteResult marshal mismatch (-want +got):\n%s", diff)
			}
		})
	}
	
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
type serverResource struct {
	resource *Resource
	handler  ResourceHandler
}
type serverResourceTemplate struct {
	resourceTemplate *ResourceTemplate
	handler          ResourceHandler
}
type ResourceHandler func(context.Context, *ReadResourceRequest) (*ReadResourceResult, error)
func ResourceNotFoundError(uri string) error {
	return &jsonrpc2.WireError{
		Code:    codeResourceNotFound,
		Message: "Resource not found",
		Data:    json.RawMessage(fmt.Sprintf(`{"uri":%q}`, uri)),
	}
}
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
func computeURIFilepath(rawURI, dirFilepath string, rootFilepaths []string) (string, error) {
	
	uri, err := url.Parse(rawURI)
	if err != nil {
		return "", err
	}
	if uri.Scheme != "file" {
		return "", fmt.Errorf("URI is not a file: %s", uri)
	}
	if uri.Path == "" {
		
		
		return "", errors.New("empty path")
	}
	
	
	uriFilepathRel, err := filepath.Localize(strings.TrimPrefix(uri.Path, "/"))
	if err != nil {
		return "", fmt.Errorf("%q cannot be localized: %w", uriFilepathRel, err)
	}
	
	if len(rootFilepaths) > 0 {
		
		
		uriFilepathAbs := filepath.Join(dirFilepath, uriFilepathRel)
		rootOK := false
		
		
		
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
func fileRoot(root *Root) (_ string, err error) {
	defer util.Wrapf(&err, "root %q", root.URI)
	
	rurl, err := url.Parse(root.URI)
	if err != nil {
		return "", err
	}
	if rurl.Scheme != "file" {
		return "", errors.New("not a file URI")
	}
	if rurl.Path == "" {
		
		
		return "", errors.New("empty path")
	}
	
	fileRoot := filepath.Clean(filepath.FromSlash(rurl.Path))
	if !filepath.IsAbs(fileRoot) {
		return "", errors.New("not an absolute path")
	}
	return fileRoot, nil
}
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
package mcp
import (
	"errors"
	"os"
)
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
	
	defer func() { err = errors.Join(err, file.Close()) }()
	return f(file)
}
</content>
</file>
<file path="mcp/resource_pre_go124.go">
<type>go</type>
<content>
package mcp
import (
	"errors"
	"os"
	"path/filepath"
)
func withFile(dir, rel string, f func(*os.File) error) (err error) {
	file, err := os.Open(filepath.Join(dir, rel))
	if err != nil {
		return err
	}
	
	defer func() { err = errors.Join(err, file.Close()) }()
	return f(file)
}
</content>
</file>
<file path="mcp/resource_test.go">
<type>go</type>
<content>
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
		wantErr string 
	}{
		{uri: "file:
		{uri: "file:
		{uri: "file:
		{uri: "file:/foo", want: "/foo"},
		{uri: "http:
		{uri: "file:
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
	
	dirFilepath := filepath.FromSlash("/files")
	rootFilepaths := []string{
		filepath.FromSlash("/files/public"),
		filepath.FromSlash("/files/shared"),
	}
	for _, tt := range []struct {
		uri     string
		want    string
		wantErr string 
	}{
		{"file:
		{"file:
		{"file:
		{"http:
		{"file:
		{"file:
		{"file:
		{"file:
		{"file:
	} {
		t.Run(tt.uri, func(t *testing.T) {
			tt.want = filepath.FromSlash(tt.want) 
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
	got, err := readFileResource("file:
	if err != nil {
		t.Fatal(err)
	}
	want := "Contents\n"
	if g := string(got); g != want {
		t.Errorf("got %q, want %q", g, want)
	}
}
func TestTemplateMatch(t *testing.T) {
	uri := "file:
	for _, tt := range []struct {
		template string
		want     bool
	}{
		{"file:
		{"file:
		{"file:
		{"file:
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
const DefaultPageSize = 1000
type Server struct {
	
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
	resourceSubscriptions   map[string]map[*ServerSession]bool 
}
type ServerOptions struct {
	
	Instructions string
	
	InitializedHandler func(context.Context, *InitializedRequest)
	
	
	
	
	PageSize int
	
	RootsListChangedHandler func(context.Context, *RootsListChangedRequest)
	
	ProgressNotificationHandler func(context.Context, *ProgressNotificationServerRequest)
	
	CompletionHandler func(context.Context, *CompleteRequest) (*CompleteResult, error)
	
	
	
	KeepAlive time.Duration
	
	SubscribeHandler func(context.Context, *SubscribeRequest) error
	
	UnsubscribeHandler func(context.Context, *UnsubscribeRequest) error
	
	
	HasPrompts bool
	
	
	HasResources bool
	
	
	HasTools bool
	
	
	
	
	
	
	
	
	GetSessionID func() string
}
... (1155 more lines)
</content>
</file>
<file path="mcp/server_example_test.go">
<type>go</type>
<content>
package mcp_test
import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"sync/atomic"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)
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
	
	c := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, nil)
	
	t1, t2 := mcp.NewInMemoryTransports()
	if _, err := s.Connect(ctx, t1, nil); err != nil {
		log.Fatal(err)
	}
	cs, err := c.Connect(ctx, t2, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer cs.Close()
	
	for p, err := range cs.Prompts(ctx, nil) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(p.Name)
	}
	
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
	
	
	
}
func Example_logging() {
	ctx := context.Background()
	
	s := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.0.1"}, nil)
	
	done := make(chan struct{}) 
	var nmsgs atomic.Int32
	c := mcp.NewClient(
		&mcp.Implementation{Name: "client", Version: "v0.0.1"},
		&mcp.ClientOptions{
			LoggingMessageHandler: func(_ context.Context, r *mcp.LoggingMessageRequest) {
				m := r.Params.Data.(map[string]any)
				fmt.Println(m["msg"], m["value"])
				if nmsgs.Add(1) == 2 { 
					close(done)
				}
			},
		})
	
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
	
	if err := cs.SetLoggingLevel(ctx, &mcp.SetLoggingLevelParams{Level: "info"}); err != nil {
		log.Fatal(err)
	}
	
	logger := slog.New(mcp.NewLoggingHandler(ss, nil))
	
	logger.Info("info shows up", "value", 1)
	logger.Debug("debug doesn't show up", "value", 2)
	logger.Warn("warn shows up", "value", 3)
	
	
	
	<-done
	
	
	
}
func Example_resources() {
	ctx := context.Background()
	resources := map[string]string{
		"file:
		"file:
		"file:
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
	
	s := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.0.1"}, nil)
	s.AddResource(&mcp.Resource{URI: "file:
	s.AddResourceTemplate(&mcp.ResourceTemplate{URITemplate: "file:
	
	c := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, nil)
	
	t1, t2 := mcp.NewInMemoryTransports()
	if _, err := s.Connect(ctx, t1, nil); err != nil {
		log.Fatal(err)
	}
	cs, err := c.Connect(ctx, t2, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer cs.Close()
	
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
	
	for _, path := range []string{"a", "dir/x", "b"} {
		res, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: "file:
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(res.Contents[0].Text)
		}
	}
	
	
	
	
	
	
}
</content>
</file>
<file path="mcp/server_test.go">
<type>go</type>
<content>
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
			wantNextCursor: getCursor("echo"), 
			wantErr:        false,
		},
		{
			name:           "SecondPage_DefaultSize_Full",
			initialItems:   allTestItems,
			inputCursor:    getCursor("echo"),
			inputPageSize:  5,
			wantFeatures:   allTestItems[5:10],
			wantNextCursor: getCursor("juliet"), 
			wantErr:        false,
		},
		{
			name:           "SecondPage_DefaultSize_Full_OutOfOrder",
			initialItems:   append(allTestItems[5:], allTestItems[0:5]...),
			inputCursor:    getCursor("echo"),
			inputPageSize:  5,
			wantFeatures:   allTestItems[5:10],
			wantNextCursor: getCursor("juliet"), 
			wantErr:        false,
		},
		{
			name:           "SecondPage_DefaultSize_Full_Duplicates",
			initialItems:   append(allTestItems, allTestItems[0:5]...),
			inputCursor:    getCursor("echo"),
			inputPageSize:  5,
			wantFeatures:   allTestItems[5:10],
			wantNextCursor: getCursor("juliet"), 
			wantErr:        false,
		},
		{
			name:           "LastPage_Remaining",
			initialItems:   allTestItems,
			inputCursor:    getCursor("juliet"),
			inputPageSize:  5,
			wantFeatures:   allTestItems[10:11], 
			wantNextCursor: "",                  
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
			inputPageSize:  len(allTestItems), 
			wantFeatures:   allTestItems,
			wantNextCursor: "", 
			wantErr:        false,
		},
		{
			name:           "PageSize_LargerThanAll",
			initialItems:   allTestItems,
			inputCursor:    "",
			inputPageSize:  len(allTestItems) + 5, 
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
			wantFeatures:   nil, 
			wantNextCursor: "",
			wantErr:        true,
		},
		{
			name:           "AboveNonExistentID",
			initialItems:   allTestItems,
			inputCursor:    getCursor("dne"), 
			inputPageSize:  5,
			wantFeatures:   allTestItems[4:9], 
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
	
	for pageSize := 1; pageSize < len(allTestItems)+1; pageSize++ {
		var gotItems []*testItem
		var nextCursor string
		wantChunks := slices.Collect(slices.Chunk(allTestItems, pageSize))
		index := 0
		
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
				s.AddResource(&Resource{URI: "file:
			},
			wantCapabilities: &ServerCapabilities{
				Logging:   &LoggingCapabilities{},
				Resources: &ResourceCapabilities{ListChanged: true},
			},
		},
		{
			name: "With resource templates",
			configureServer: func(s *Server) {
				s.AddResourceTemplate(&ResourceTemplate{URITemplate: "file:
			},
			wantCapabilities: &ServerCapabilities{
				Logging:   &LoggingCapabilities{},
				Resources: &ResourceCapabilities{ListChanged: true},
			},
		},
		{
			name: "With resource subscriptions",
			configureServer: func(s *Server) {
				s.AddResourceTemplate(&ResourceTemplate{URITemplate: "file:
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
				s.AddResource(&Resource{URI: "file:
				s.AddResourceTemplate(&ResourceTemplate{URITemplate: "file:
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
		{"ValidFileTemplate", "file:
		{"ValidCustomScheme", "myproto:
		{"EmptyVariable", "file:
		{"UnclosedVariable", "file:
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
func TestServerSessionkeepaliveCancelOverwritten(t *testing.T) {
	
	
	
	server := NewServer(testImpl, &ServerOptions{KeepAlive: 5 * time.Second})
	ss := &ServerSession{server: server}
	
	_, err := ss.initialize(context.Background(), &InitializeParams{})
	if err != nil {
		t.Fatalf("ServerSession initialize failed: %v", err)
	}
	
	_, err = ss.initialized(context.Background(), &InitializedParams{})
	if err != nil {
		t.Fatalf("First initialized call failed: %v", err)
	}
	if ss.keepaliveCancel == nil {
		t.Fatalf("expected ServerSession.keepaliveCancel to be set after the first call of initialized")
	}
	
	firstCancel := ss.keepaliveCancel
	defer firstCancel()
	
	
	
	
	ss.keepaliveCancel = nil
	
	_, err = ss.initialized(context.Background(), &InitializedParams{})
	if err == nil {
		t.Fatalf("Expected 'duplicate initialized received' error on second call, got nil")
	}
	
	
	
	if ss.keepaliveCancel != nil {
		t.Fatal("expected ServerSession.keepaliveCancel to be nil after we manually niled it and re-initialized")
	}
}
func panics(f func()) (b bool) {
	defer func() {
		b = recover() != nil
	}()
	f()
	return false
}
func TestAddTool(t *testing.T) {
	
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
		
		unstructured := result.Content[0].(*TextContent).Text
		structured := string(result.StructuredContent.(json.RawMessage))
		if diff := cmp.Diff(unstructured, structured); diff != "" {
			t.Errorf("Unstructured content does not match structured content exactly (-unstructured +structured):\n%s", diff)
		}
	}
}
func TestToolForSchemas(t *testing.T) {
	
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
	
	testToolForSchema[in](t, &Tool{}, `{"p":3}`, out{true}, inSchema, outSchema, "")
	
	
	testToolForSchema[in](t, &Tool{}, `{"p":"x"}`, out{true}, inSchema, outSchema, `want "integer"`)
	
	testToolForSchema[in, any](t, &Tool{}, `{"p":3}`, 0, inSchema, nil, "")
	
	testToolForSchema[in, any](t, &Tool{}, `{"p":"x"}`, 0, inSchema, nil, `want "integer"`)
	
	testToolForSchema[in, any](t, &Tool{InputSchema: inSchema2}, `{"p":3}`, 0, inSchema2, nil, `want "string"`)
	
	testToolForSchema[in, any](t, &Tool{OutputSchema: outSchema2}, `{"p":3}`, out{true},
		inSchema, outSchema2, `want "integer"`)
	
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
package mcp
type hasSessionID interface {
	SessionID() string
}
type ServerSessionState struct {
	
	InitializeParams *InitializeParams `json:"initializeParams"`
	
	InitializedParams *InitializedParams `json:"initializedParams"`
	
	LogLevel LoggingLevel `json:"logLevel"`
	
}
</content>
</file>
<file path="mcp/shared.go">
<type>go</type>
<content>
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
func negotiatedVersion(clientVersion string) string {
	
	
	
	
	
	if !slices.Contains(supportedProtocolVersions, clientVersion) {
		return latestProtocolVersion
	}
	return clientVersion
}
type MethodHandler func(ctx context.Context, method string, req Request) (result Result, err error)
type Session interface {
	
	ID() string
	sendingMethodInfos() map[string]methodInfo
	receivingMethodInfos() map[string]methodInfo
	sendingMethodHandler() MethodHandler
	receivingMethodHandler() MethodHandler
	getConn() *jsonrpc2.Connection
}
type Middleware func(MethodHandler) MethodHandler
func addMiddleware(handlerp *MethodHandler, middleware []Middleware) {
	for _, m := range slices.Backward(middleware) {
		*handlerp = m(*handlerp)
	}
}
func defaultSendingMethodHandler[S Session](ctx context.Context, method string, req Request) (Result, error) {
	info, ok := req.GetSession().sendingMethodInfos()[method]
	if !ok {
		
		return nil, jsonrpc2.ErrNotHandled
	}
	
	if strings.HasPrefix(method, "notifications/") {
		return nil, req.GetSession().getConn().Notify(ctx, method, req.GetParams())
	}
	
	
	res := info.newResult()
	if err := call(ctx, req.GetSession().getConn(), method, req.GetParams(), res); err != nil {
		return nil, err
	}
	return res, nil
}
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
	
	res, err := mh(ctx, method, req)
	if err != nil {
		var z R
		return z, err
	}
	return res.(R), nil
}
func defaultReceivingMethodHandler[S Session](ctx context.Context, method string, req Request) (Result, error) {
	info, ok := req.GetSession().receivingMethodInfos()[method]
	if !ok {
		
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
	
	res, err := mh(ctx, jreq.Method, req)
	if err != nil {
		return nil, err
	}
	return res, nil
}
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
	
	
	
	
	
	if info.flags&missingParamsOK == 0 && len(req.Params) == 0 {
		return methodInfo{}, fmt.Errorf("%w: missing required \"params\"", jsonrpc2.ErrInvalidRequest)
	}
	return info, nil
}
type methodInfo struct {
	
	
	flags methodFlags
	
	
	unmarshalParams func(json.RawMessage) (Params, error)
	newRequest      func(Session, Params, *RequestExtra) Request
	
	
	handleMethod MethodHandler
	
	
	newResult func() Result
}
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
	notification    methodFlags = 1 << iota 
	missingParamsOK                         
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
			
			
			
			
			
			
			
			if flags&missingParamsOK == 0 && p == nil {
				return nil, fmt.Errorf("%w: missing required \"params\"", jsonrpc2.ErrInvalidRequest)
			}
			return orZero[Params](p), nil
		},
		
		
		
		
		newResult: func() Result { return reflect.New(reflect.TypeFor[R]().Elem()).Interface().(R) },
	}
}
func serverMethod[P Params, R Result](
	f func(*Server, context.Context, *ServerRequest[P]) (R, error),
) typedServerMethodHandler[P, R] {
	return func(ctx context.Context, req *ServerRequest[P]) (R, error) {
		return f(req.Session.server, ctx, req)
	}
}
func clientMethod[P Params, R Result](
	f func(*Client, context.Context, *ClientRequest[P]) (R, error),
) typedClientMethodHandler[P, R] {
	return func(ctx context.Context, req *ClientRequest[P]) (R, error) {
		return f(req.Session.client, ctx, req)
	}
}
func serverSessionMethod[P Params, R Result](f func(*ServerSession, context.Context, P) (R, error)) typedServerMethodHandler[P, R] {
	return func(ctx context.Context, req *ServerRequest[P]) (R, error) {
		return f(req.GetSession().(*ServerSession), ctx, req.Params)
	}
}
func clientSessionMethod[P Params, R Result](f func(*ClientSession, context.Context, P) (R, error)) typedClientMethodHandler[P, R] {
	return func(ctx context.Context, req *ClientRequest[P]) (R, error) {
		return f(req.GetSession().(*ClientSession), ctx, req.Params)
	}
}
const (
	codeResourceNotFound = -32002
	
	codeUnsupportedMethod = -31001
	
	codeInvalidParams = -32602
)
func notifySessions[S Session, P Params](sessions []S, method string, params P) {
	if sessions == nil {
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	
	for _, s := range sessions {
		req := newRequest(s, params)
		if err := handleNotify(ctx, method, req); err != nil {
			
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
type Meta map[string]any
func (m Meta) GetMeta() map[string]any { return m }
func (m *Meta) SetMeta(x map[string]any) { *m = x }
const progressTokenKey = "progressToken"
func getProgressToken(p Params) any {
	return p.GetMeta()[progressTokenKey]
}
func setProgressToken(p Params, pt any) {
	switch pt.(type) {
	
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
type Request interface {
	isRequest()
	GetSession() Session
	GetParams() Params
	
	GetExtra() *RequestExtra
}
type ClientRequest[P Params] struct {
	Session *ClientSession
	Params  P
}
type ServerRequest[P Params] struct {
	Session *ServerSession
	Params  P
	Extra   *RequestExtra
}
type RequestExtra struct {
	TokenInfo *auth.TokenInfo 
	Header    http.Header     
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
type Params interface {
	
	GetMeta() map[string]any
	
	SetMeta(map[string]any)
	
	isParams()
}
type RequestParams interface {
	Params
	
	
	GetProgressToken() any
	
	
	SetProgressToken(any)
}
type Result interface {
	
	isResult()
	
	GetMeta() map[string]any
	
	SetMeta(map[string]any)
}
type emptyResult struct{}
func (*emptyResult) isResult()               {}
func (*emptyResult) GetMeta() map[string]any { panic("should never be called") }
func (*emptyResult) SetMeta(map[string]any)  { panic("should never be called") }
type listParams interface {
	
	cursorPtr() *string
}
type listResult[T any] interface {
	
	nextCursorPtr() *string
}
type keepaliveSession interface {
	Ping(ctx context.Context, params *PingParams) error
	Close() error
}
func startKeepalive(session keepaliveSession, interval time.Duration, cancelPtr *context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	
	
	
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
package mcp
</content>
</file>
<file path="mcp/sse.go">
<type>go</type>
<content>
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
type SSEHandler struct {
	getServer    func(request *http.Request) *Server
	opts         SSEOptions
	onConnection func(*ServerSession) 
	mu       sync.Mutex
	sessions map[string]*SSEServerTransport
}
type SSEOptions struct{}
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
type SSEServerTransport struct {
	
	
	Endpoint string
	
	Response http.ResponseWriter
	
	
	
	incoming chan jsonrpc.Message
	
	
	
	
	mu     sync.Mutex    
	closed bool          
	done   chan struct{} 
}
func (t *SSEServerTransport) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if t.incoming == nil {
		http.Error(w, "session not connected", http.StatusInternalServerError)
		return
	}
	
	data, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	
	
	
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
	
	
	if req.Method == http.MethodPost {
		
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
	defer ss.Close() 
	select {
	case <-req.Context().Done():
	case <-transport.done:
	}
}
type sseServerConn struct {
	t *SSEServerTransport
}
func (s *sseServerConn) SessionID() string { return "" }
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
	
	
	
	if s.t.closed {
		return io.EOF
	}
	_, err = writeEvent(s.t.Response, Event{Name: "message", Data: data})
	return err
}
func (s *sseServerConn) Close() error {
	s.t.mu.Lock()
	defer s.t.mu.Unlock()
	if !s.t.closed {
		s.t.closed = true
		close(s.t.done)
	}
	return nil
}
type SSEClientTransport struct {
	
	Endpoint string
	
	
	HTTPClient *http.Client
}
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
	
	s := &sseClientConn{
		client:      httpClient,
		msgEndpoint: msgEndpoint,
		incoming:    make(chan []byte, 100),
		body:        resp.Body,
		done:        make(chan struct{}),
	}
	go func() {
		defer s.Close() 
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
type sseClientConn struct {
	client      *http.Client 
	msgEndpoint *url.URL     
	incoming    chan []byte  
	mu     sync.Mutex
	body   io.ReadCloser 
	closed bool          
	done   chan struct{} 
}
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
	
}
</content>
</file>
<file path="mcp/sse_test.go">
<type>go</type>
<content>
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
			
			if atomic.LoadInt64(&customClientUsed) == 0 {
				t.Error("Expected custom HTTP client to be used, but it wasn't")
			}
			t.Run("badrequests", func(t *testing.T) {
				msgEndpoint := cs.mcpConn.(*sseClientConn).msgEndpoint.String()
				
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
type roundTripperFunc func(*http.Request) (*http.Response, error)
func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
</content>
</file>
<file path="mcp/streamable.go">
<type>go</type>
<content>
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
type StreamableHTTPHandler struct {
	getServer func(*http.Request) *Server
	opts      StreamableHTTPOptions
	onTransportDeletion func(sessionID string) 
	mu sync.Mutex
	
	
	transports map[string]*StreamableServerTransport 
}
type StreamableHTTPOptions struct {
	
	
	
	
	
	
	
	
	Stateless bool
	
	
	
	
	
	JSONResponse bool
}
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
	httpMethod    string 
	sessionID     string 
	jsonrpcMethod string 
}
type header map[string]string
type streamableResponse struct {
	header              header 
	status              int    
	body                string 
	optional            bool   
	wantProtocolVersion string 
	callback            func() 
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
	_ = session.Wait() 
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
							"Content-Type":  "application/json; charset=utf-8", 
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
func ExampleStreamableHTTPHandler() {
	
	
	
	
	server := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.1.0"}, nil)
	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{JSONResponse: true})
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()
	
	resp := mustPostMessage(`{"jsonrpc": "2.0", "id": 1, "method":"initialize", "params": {}}`, httpServer.URL)
	fmt.Println(resp)
	
	
}
func ExampleStreamableHTTPHandler_middleware() {
	server := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.1.0"}, nil)
	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, nil)
	loggingHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		
		body, err := io.ReadAll(req.Body)
		if err != nil {
			log.Fatal(err)
		}
		req.Body.Close() 
		req.Body = io.NopCloser(bytes.NewBuffer(body))
		fmt.Println(req.Method, string(body))
		handler.ServeHTTP(w, req)
	})
	httpServer := httptest.NewServer(loggingHandler)
	defer httpServer.Close()
	
	mustPostMessage(`{"jsonrpc": "2.0", "id": 1, "method":"initialize", "params": {}}`, httpServer.URL)
	
	
}
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
	
	
	ctx := context.Background()
	for _, useJSON := range []bool{false, true} {
		t.Run(fmt.Sprintf("JSONResponse=%v", useJSON), func(t *testing.T) {
			
			server := NewServer(testImpl, nil)
			AddTool(server, &Tool{Name: "greet", Description: "say hi"}, sayHi)
			
			
			var (
				start     = make(chan struct{})
				cancelled = make(chan struct{}, 1) 
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
package mcp
import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/jsonschema-go/jsonschema"
)
type ToolHandler func(context.Context, *CallToolRequest) (*CallToolResult, error)
type ToolHandlerFor[In, Out any] func(_ context.Context, request *CallToolRequest, input In) (result *CallToolResult, output Out, _ error)
type serverTool struct {
	tool    *Tool
	handler ToolHandler
}
func applySchema(data json.RawMessage, resolved *jsonschema.Resolved) (json.RawMessage, error) {
	
	
	
	
	
	
	
	
	
	
	
	
	
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
	
	
	
	
	
	
	
	server := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.0.1"}, nil)
	server.AddTool(&mcp.Tool{
		Name:        "greet",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"user":{"type":"string"}}}`),
	}, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		
		var args struct{ User string }
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			
			
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
	
}
func ExampleAddTool_customMarshalling() {
	
	
	
	
	
	
	type MyDate struct {
		time.Time
	}
	type Input struct {
		Query string `json:"query,omitempty"`
		Start MyDate `json:"start,omitempty"`
		End   MyDate `json:"end,omitempty"`
	}
	
	
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
	session, err := connect(ctx, server) 
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
func ExampleAddTool_complexSchema() {
	
	
	
	
	
	customSchemas := map[reflect.Type]*jsonschema.Schema{
		reflect.TypeFor[Probability](): {Type: "number", Minimum: jsonschema.Ptr(0.0), Maximum: jsonschema.Ptr(1.0)},
		reflect.TypeFor[WeatherType](): {Type: "string", Enum: []any{Sunny, PartlyCloudy, Cloudy, Rainy, Snowy}},
	}
	opts := &jsonschema.ForOptions{TypeSchemas: customSchemas}
	in, err := jsonschema.For[WeatherInput](opts)
	if err != nil {
		log.Fatal(err)
	}
	
	
	daysSchema := in.Properties["days"]
	daysSchema.Minimum = jsonschema.Ptr(0.0)
	daysSchema.Maximum = jsonschema.Ptr(10.0)
	
	out, err := jsonschema.For[WeatherOutput](opts)
	if err != nil {
		log.Fatal(err)
	}
	
	
	server := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.0.1"}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name:         "weather",
		InputSchema:  in,
		OutputSchema: out,
	}, WeatherTool)
	
	ctx := context.Background()
	session, err := connect(ctx, server) 
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()
	
	for t, err := range session.Tools(ctx, nil) {
		if err != nil {
			log.Fatal(err)
		}
		
		
		fmt.Println("max days:", jsonPath(t.InputSchema, "properties", "days", "maximum"))
		fmt.Println("max confidence:", jsonPath(t.OutputSchema, "properties", "confidence", "maximum"))
		fmt.Println("weather types:", jsonPath(t.OutputSchema, "properties", "dailyForecast", "items", "properties", "type", "enum"))
	}
	
	
	
	
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
		{`{}`, new(S), &S{X: 3}}, 
		{`{"x": 0}`, new(S), &S{X: 0}},
		{`{"x": 1}`, new(map[string]any), &map[string]any{"x": 1.0}},
		{`{}`, new(map[string]any), &map[string]any{"x": 3.0}}, 
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
	
	server := NewServer(testImpl, nil)
	
	structuredErrorHandler := func(ctx context.Context, req *CallToolRequest, args map[string]any) (*CallToolResult, any, error) {
		return nil, nil, &jsonrpc2.WireError{
			Code:    codeInvalidParams,
			Message: "internal server error",
		}
	}
	
	regularErrorHandler := func(ctx context.Context, req *CallToolRequest, args map[string]any) (*CallToolResult, any, error) {
		return nil, nil, fmt.Errorf("tool execution failed")
	}
	AddTool(server, &Tool{Name: "error_tool", Description: "returns structured error"}, structuredErrorHandler)
	AddTool(server, &Tool{Name: "regular_error_tool", Description: "returns regular error"}, regularErrorHandler)
	
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
	
	t.Run("structured_error", func(t *testing.T) {
		
		_, err = cs.CallTool(context.Background(), &CallToolParams{
			Name:      "error_tool",
			Arguments: map[string]any{},
		})
		
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
	
	t.Run("regular_error", func(t *testing.T) {
		
		result, err := cs.CallTool(context.Background(), &CallToolParams{
			Name:      "regular_error_tool",
			Arguments: map[string]any{},
		})
		
		if err != nil {
			t.Fatalf("unexpected protocol error: %v", err)
		}
		
		if !result.IsError {
			t.Error("expected IsError=true, got false")
		}
		
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
var ErrConnectionClosed = errors.New("connection closed")
type Transport interface {
	
	
	
	Connect(ctx context.Context) (Connection, error)
}
type Connection interface {
	
	
	
	
	Read(context.Context) (jsonrpc.Message, error)
	
	
	
	
	Write(context.Context, jsonrpc.Message) error
	
	
	
	
	Close() error
	
	SessionID() string
}
type clientConnection interface {
	Connection
	
	sessionUpdated(clientSessionState)
}
type serverConnection interface {
	Connection
	sessionUpdated(ServerSessionState)
}
type StdioTransport struct{}
func (*StdioTransport) Connect(context.Context) (Connection, error) {
	return newIOConn(rwc{os.Stdin, os.Stdout}), nil
}
type IOTransport struct {
	Reader io.ReadCloser
	Writer io.WriteCloser
}
func (t *IOTransport) Connect(context.Context) (Connection, error) {
	return newIOConn(rwc{t.Reader, t.Writer}), nil
}
type InMemoryTransport struct {
	rwc io.ReadWriteCloser
}
func (t *InMemoryTransport) Connect(context.Context) (Connection, error) {
	return newIOConn(t.rwc), nil
}
func NewInMemoryTransports() (*InMemoryTransport, *InMemoryTransport) {
	c1, c2 := net.Pipe()
	return &InMemoryTransport{c1}, &InMemoryTransport{c2}
}
type binder[T handler, State any] interface {
	
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
type canceller struct {
	conn *jsonrpc2.Connection
}
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
func call(ctx context.Context, conn *jsonrpc2.Connection, method string, params Params, result Result) error {
	
	
	call := conn.Call(ctx, method, params)
	err := call.Await(ctx, result)
	switch {
	case errors.Is(err, jsonrpc2.ErrClientClosing), errors.Is(err, jsonrpc2.ErrServerClosing):
		return fmt.Errorf("%w: calling %q: %v", ErrConnectionClosed, method, err)
	case ctx.Err() != nil:
		
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
type LoggingTransport struct {
	Transport Transport
	Writer    io.Writer
}
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
type ioConn struct {
	protocolVersion string 
	writeMu sync.Mutex         
	rwc     io.ReadWriteCloser 
	
	incoming <-chan msgOrErr
	
	
	outgoingBatch []jsonrpc.Message
	
	
	queue []jsonrpc.Message
	
	
	batchMu sync.Mutex
	batches map[jsonrpc2.ID]*msgBatch 
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
	
	
	
	
	
	
	go func() {
		dec := json.NewDecoder(rwc)
		for {
			var raw json.RawMessage
			err := dec.Decode(&raw)
			
			if err == nil {
				
				var tr [1]byte
				if n, readErr := dec.Buffered().Read(tr[:]); n > 0 {
					
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
type msgBatch struct {
	unresolved map[jsonrpc2.ID]int
	responses  []*jsonrpc.Response
}
func (t *ioConn) Read(ctx context.Context) (jsonrpc.Message, error) {
	
	
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
		var respBatch *msgBatch 
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
			
			if err := t.addBatch(respBatch); err != nil {
				return nil, err
			}
		}
	}
	return msgs[0], err
}
func readBatch(data []byte) (msgs []jsonrpc.Message, isBatch bool, _ error) {
	
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
	
	msg, err := jsonrpc2.DecodeMessage(data)
	return []jsonrpc.Message{msg}, false, err
}
func (t *ioConn) Write(ctx context.Context, msg jsonrpc.Message) error {
	
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	t.writeMu.Lock()
	defer t.writeMu.Unlock()
	
	
	
	
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
	data = append(data, '\n') 
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
	
	for _, line := range slices.Sorted(strings.SplitSeq(b.String(), "\n")) {
		fmt.Println(line)
	}
	
	
	
	
}
</content>
</file>
<file path="mcp/transport_test.go">
<type>go</type>
<content>
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
	
	
	
	
	
	ctx := context.Background()
	r, w := io.Pipe()
	tport := newIOConn(rwc{r, w})
	tport.outgoingBatch = make([]jsonrpc.Message, 0, 2)
	defer tport.Close()
	
	read := make(chan jsonrpc.Message)
	go func() {
		for range 2 {
			msg, _ := tport.Read(ctx)
			read <- msg
		}
	}()
	
	tport.Write(ctx, &jsonrpc.Request{ID: jsonrpc2.Int64ID(1), Method: "test"})
	select {
	case got := <-read:
		t.Fatalf("after one write, got message %v", got)
	default:
	}
	
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
const base32alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
func randText() string {
	
	src := make([]byte, 26)
	rand.Read(src)
	for i := range src {
		src[i] = base32alphabet[src[i]%32]
	}
	return string(src)
}
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
package oauthex
import (
	"github.com/modelcontextprotocol/go-sdk/internal/oauthex"
)
type ProtectedResourceMetadata = oauthex.ProtectedResourceMetadata
</content>
</file>
</files>
