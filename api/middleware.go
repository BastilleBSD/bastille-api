package api

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

var logger *slog.Logger

// sensitiveHeaders are stripped from any logged request headers. Values here are
// in Go's canonical MIME header form (http.Header.Del canonicalizes its
// argument, but we keep these canonical for clarity).
var sensitiveHeaders = []string{
	"Authorization",
	"Authorization-Id",
	"X-Api-Key",
	"X-Api-Key-Id",
}

// redactHeaders returns a copy of h with all sensitive headers removed, so that
// API secrets are never written to logs (even in debug mode).
func redactHeaders(h http.Header) http.Header {
	clone := h.Clone()
	if clone == nil {
		return http.Header{}
	}
	for _, name := range sensitiveHeaders {
		clone.Del(name)
	}
	return clone
}

func InitLogger(debug bool) {

	var level slog.Level

	if debug {
		level = slog.LevelDebug
		gin.SetMode(gin.DebugMode)
	} else {
		level = slog.LevelInfo
		gin.SetMode(gin.ReleaseMode)
	}

	logger = slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		}),
	)

	slog.SetDefault(logger)
}

func logRequest(level string, msg string, c *gin.Context, cmdArgs any, err any) {

	switch level {
	case "info":
		if c != nil {
			// Method + path only. The raw query string is intentionally NOT
			// logged: request parameters can contain sensitive values (targets,
			// paths, cmd payloads) and would otherwise land in always-on logs.
			logger.Info(msg,
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
			)
		} else {
			// Message only
			logger.Info(msg)
		}

	case "debug":
		var attrs []any
		if c != nil {
			headers := redactHeaders(c.Request.Header)
			attrs = append(attrs,
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"query", c.Request.URL.RawQuery,
				"remote", c.ClientIP(),
				"user_agent", c.Request.UserAgent(),
				"headers", headers,
			)
		}
		if cmdArgs != nil {
			attrs = append(attrs, "rawArgs", cmdArgs)
		}
		logger.Debug(msg, attrs...)

	case "error":
		var attrs []any
		if c != nil {
			attrs = append(attrs,
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"query", c.Request.URL.RawQuery,
				"remote", c.ClientIP(),
			)
		}
		if err != nil {
			attrs = append(attrs, "error", err)
		}
		logger.Error(msg, attrs...)
	}
}

func CORSMiddleware() gin.HandlerFunc {

	logRequest("debug", "CORSMiddleware", nil, nil, nil)

	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, Authorization-ID, X-TTYD-Url, X-API-Key, X-API-Key-ID")
		c.Header("Access-Control-Expose-Headers", "X-TTYD-Url")

		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusOK)
			c.Abort()
			return
		}

		c.Next()
	}
}

func apiKeyMiddleware(scope string, action string) gin.HandlerFunc {

	logRequest("debug", "apiKeyMiddleware", nil, nil, nil)

	return func(c *gin.Context) {

		key := c.GetHeader("Authorization")
		keyID := c.GetHeader("Authorization-ID")
		const prefix = "Bearer "

		if key == "" || !strings.HasPrefix(key, prefix) {
			logRequest("error", "missing Authorization header", c, nil, nil)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
			})
			return
		}
		if keyID == "" {
			logRequest("error", "missing Authorization-ID header", c, nil, nil)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
			})
			return
		}

		providedKey := key[len(prefix):]

		keyData, exists := cfg.APIKeys[keyID]
		if !exists {
			logRequest("error", "invalid API keyID", c, nil, nil)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		trialHash := generateHash(providedKey, keyData.Salt)

		if !compareHash(trialHash, keyData.Hash) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
			})
			logRequest("error", "key hash mismatch", c, nil, nil)
			return
		}

		var allowed []string
		switch scope {
		case "bastille":
			allowed = keyData.Permissions.Bastille
		case "admin":
			allowed = keyData.Permissions.Admin
		}

		hasPermission := false
		for _, a := range allowed {
			if a == "*" || a == action {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			logRequest("error", "forbidden action", c, action, nil)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Forbidden",
				"details": "Requires " + scope + " permission: " + action,
			})
			return
		}

		c.Next()
	}
}

// Log all API requests
func loggingMiddleware() gin.HandlerFunc {

	logRequest("debug", "loggingMiddleware", nil, nil, nil)

	return func(c *gin.Context) {
		logRequest("info", "request", c, nil, nil)
		c.Next()
	}
}