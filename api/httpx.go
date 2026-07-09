package api

// httpx.go is a minimal, dependency-free HTTP shim built on the standard
// library. It reproduces the small slice of the Gin API this project actually
// used (a request context with a handful of helpers plus a gin-style middleware
// chain), so the command handlers could drop the Gin dependency with almost no
// change to their bodies.
//
// Only what is needed is implemented. If the API grows, extend this file rather
// than reaching for a framework.

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
)

// H is a convenience type for building JSON objects, mirroring gin.H.
type H map[string]any

// HandlerFunc is a request handler in the middleware chain.
type HandlerFunc func(*Ctx)

// abortIndex is larger than any real chain length; setting the cursor to it
// stops further handlers from running (same technique as Gin).
const abortIndex = 1 << 30

// Ctx wraps a single request/response and threads a handler chain through it.
// Field names (Writer, Request) match gin.Context so handler bodies port over
// unchanged.
type Ctx struct {
	Writer   http.ResponseWriter
	Request  *http.Request
	handlers []HandlerFunc
	index    int
	fullPath string
	wrote    bool
}

// Next advances through the handler chain. A handler that does not call Next
// still lets the chain continue; a handler that calls Abort stops it.
func (c *Ctx) Next() {
	c.index++
	for c.index < len(c.handlers) {
		c.handlers[c.index](c)
		c.index++
	}
}

// Abort prevents any pending handlers from running.
func (c *Ctx) Abort() { c.index = abortIndex }

// Query returns a URL query parameter.
func (c *Ctx) Query(key string) string {
	return c.Request.URL.Query().Get(key)
}

// GetHeader returns a request header value.
func (c *Ctx) GetHeader(key string) string {
	return c.Request.Header.Get(key)
}

// Header sets a response header.
func (c *Ctx) Header(key, value string) {
	c.Writer.Header().Set(key, value)
}

// FullPath returns the route pattern that matched, used to distinguish the
// "live" endpoints.
func (c *Ctx) FullPath() string { return c.fullPath }

// ClientIP returns the remote address without its port. It intentionally does
// not trust forwarding headers: behind the recommended reverse proxy the
// operator should decide explicitly which header to honor.
func (c *Ctx) ClientIP() string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(c.Request.RemoteAddr))
	if err != nil {
		return c.Request.RemoteAddr
	}
	return host
}

func (c *Ctx) writeHeader(code int) {
	if c.wrote {
		return
	}
	c.wrote = true
	c.Writer.WriteHeader(code)
}

// Status writes a bare status code.
func (c *Ctx) Status(code int) { c.writeHeader(code) }

// JSON writes obj as a JSON response with the given status code.
func (c *Ctx) JSON(code int, obj any) {
	c.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.writeHeader(code)
	_ = json.NewEncoder(c.Writer).Encode(obj)
}

// String writes a plain-text response.
func (c *Ctx) String(code int, s string) {
	c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.writeHeader(code)
	_, _ = io.WriteString(c.Writer, s)
}

// AbortWithStatusJSON stops the chain and writes a JSON error response.
func (c *Ctx) AbortWithStatusJSON(code int, obj any) {
	c.Abort()
	c.JSON(code, obj)
}

// chain builds an http.HandlerFunc that runs the given handler chain for one
// request under a fresh Ctx.
func chain(fullPath string, handlers ...HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := &Ctx{Writer: w, Request: r, handlers: handlers, index: -1, fullPath: fullPath}
		c.Next()
	}
}
