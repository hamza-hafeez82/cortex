package recon

import (
	"strings"

	"github.com/hamza-hafeez82/cortex/internal/parser"
	"github.com/hamza-hafeez82/cortex/internal/walker"
)

// endpointPattern describes a pattern for detecting HTTP routes.
type endpointPattern struct {
	// substr must appear somewhere on the line
	substr string
	// method is extracted from the match or set statically
	method string
	// framework this pattern belongs to
	framework string
}

// routePatterns covers the most common HTTP routing styles.
var routePatterns = []endpointPattern{
	// Express / Node.js
	{substr: "router.get(", method: "GET", framework: "Express"},
	{substr: "router.post(", method: "POST", framework: "Express"},
	{substr: "router.put(", method: "PUT", framework: "Express"},
	{substr: "router.delete(", method: "DELETE", framework: "Express"},
	{substr: "router.patch(", method: "PATCH", framework: "Express"},
	{substr: "app.get(", method: "GET", framework: "Express"},
	{substr: "app.post(", method: "POST", framework: "Express"},
	{substr: "app.put(", method: "PUT", framework: "Express"},
	{substr: "app.delete(", method: "DELETE", framework: "Express"},
	{substr: "app.patch(", method: "PATCH", framework: "Express"},

	// Fastify
	{substr: "fastify.get(", method: "GET", framework: "Fastify"},
	{substr: "fastify.post(", method: "POST", framework: "Fastify"},
	{substr: "fastify.put(", method: "PUT", framework: "Fastify"},
	{substr: "fastify.delete(", method: "DELETE", framework: "Fastify"},

	// FastAPI / Python decorators
	{substr: "@app.get(", method: "GET", framework: "FastAPI"},
	{substr: "@app.post(", method: "POST", framework: "FastAPI"},
	{substr: "@app.put(", method: "PUT", framework: "FastAPI"},
	{substr: "@app.delete(", method: "DELETE", framework: "FastAPI"},
	{substr: "@app.patch(", method: "PATCH", framework: "FastAPI"},
	{substr: "@router.get(", method: "GET", framework: "FastAPI"},
	{substr: "@router.post(", method: "POST", framework: "FastAPI"},
	{substr: "@router.put(", method: "PUT", framework: "FastAPI"},
	{substr: "@router.delete(", method: "DELETE", framework: "FastAPI"},
	{substr: "@router.patch(", method: "PATCH", framework: "FastAPI"},

	// Django urls.py
	{substr: "path(", method: "*", framework: "Django"},
	{substr: "re_path(", method: "*", framework: "Django"},
	{substr: "url(", method: "*", framework: "Django"},

	// Go — Gin
	{substr: ".GET(", method: "GET", framework: "Gin"},
	{substr: ".POST(", method: "POST", framework: "Gin"},
	{substr: ".PUT(", method: "PUT", framework: "Gin"},
	{substr: ".DELETE(", method: "DELETE", framework: "Gin"},
	{substr: ".PATCH(", method: "PATCH", framework: "Gin"},

	// Go — Echo
	{substr: ".GET(", method: "GET", framework: "Echo"},
	{substr: ".POST(", method: "POST", framework: "Echo"},

	// Go — Chi / stdlib
	{substr: `http.HandleFunc(`, method: "*", framework: "stdlib"},
	{substr: `http.Handle(`, method: "*", framework: "stdlib"},
	{substr: `.HandleFunc(`, method: "*", framework: "Chi"},

	// Go — Fiber
	{substr: "app.Get(", method: "GET", framework: "Fiber"},
	{substr: "app.Post(", method: "POST", framework: "Fiber"},
	{substr: "app.Put(", method: "PUT", framework: "Fiber"},
	{substr: "app.Delete(", method: "DELETE", framework: "Fiber"},
}

// DetectEndpoints scans all source files and returns detected HTTP endpoints.
func DetectEndpoints(repo *walker.RepoMap) []Endpoint {
	var endpoints []Endpoint

	for _, f := range repo.Files {
		if !parser.IsSourceLanguage(f.Language) {
			continue
		}
		if len(f.Lines) == 0 {
			continue
		}

		endpoints = append(endpoints, detectFileEndpoints(f)...)
	}

	return endpoints
}

// detectFileEndpoints scans a single file for route registrations.
func detectFileEndpoints(f *walker.FileNode) []Endpoint {
	var endpoints []Endpoint

	for i, line := range f.Lines {
		trimmed := strings.TrimSpace(line)

		// Skip comments
		if parser.IsComment(trimmed, f.Language) {
			continue
		}

		for _, pattern := range routePatterns {
			if !strings.Contains(trimmed, pattern.substr) {
				continue
			}

			path := extractRoutePath(trimmed)
			if path == "" {
				continue
			}

			endpoints = append(endpoints, Endpoint{
				Method:  pattern.method,
				Path:    path,
				File:    f.Path,
				Line:    i + 1,
				Handler: extractHandlerName(trimmed),
			})
			break // one match per line
		}
	}

	return endpoints
}

// extractRoutePath attempts to extract the route path string from a line.
// It looks for the first string literal after the opening parenthesis.
func extractRoutePath(line string) string {
	// Find the opening paren
	start := strings.Index(line, "(")
	if start < 0 {
		return ""
	}

	rest := line[start+1:]

	// Look for single or double quoted string
	for _, quote := range []byte{'"', '\''} {
		q := string(quote)
		if idx := strings.Index(rest, q); idx >= 0 {
			end := strings.Index(rest[idx+1:], q)
			if end >= 0 {
				path := rest[idx+1 : idx+1+end]
				// Only return if it looks like a path
				if strings.Contains(path, "/") || strings.HasPrefix(path, "/") {
					return path
				}
			}
		}
	}

	// Python f-strings or template literals — return placeholder
	if strings.Contains(rest, "`/") || strings.Contains(rest, "f'/") || strings.Contains(rest, `f"/`) {
		return "/<dynamic>"
	}

	return ""
}

// extractHandlerName tries to extract the handler function name from a route line.
// e.g. `router.get('/users', getUsers)` → "getUsers"
func extractHandlerName(line string) string {
	// Find the last identifier before the closing paren
	end := strings.LastIndex(line, ")")
	if end < 0 {
		return ""
	}

	// Walk backwards from end to find a comma, then grab the identifier
	segment := strings.TrimSpace(line[:end])
	parts := strings.Split(segment, ",")
	if len(parts) < 2 {
		return ""
	}

	last := strings.TrimSpace(parts[len(parts)-1])
	// Strip any trailing paren or bracket
	last = strings.TrimRight(last, ")}")

	// Only return if it looks like an identifier
	if last == "" || strings.Contains(last, `"`) || strings.Contains(last, `'`) {
		return ""
	}

	return last
}
