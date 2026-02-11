package api

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
)

func consoleProxy(target string) gin.HandlerFunc {

	logRequest("debug", "consoleProxy", nil, nil, nil)

	return func(c *gin.Context) {

		remote, err := url.Parse(target)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid target URL"})
			logRequest("error", "invalid target url", c, nil, err.Error())
			return
		}

		timeout := 500 * time.Millisecond
		conn, err := net.DialTimeout("tcp", remote.Host, timeout)
		
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error":  "TTYD not reachable"})
			logRequest("error", "ttyd not reachable", c, nil, err.Error())
			return
		}
		conn.Close()

		proxy := httputil.NewSingleHostReverseProxy(remote)

		proxy.Director = func(req *http.Request) {
			req.URL.Scheme = remote.Scheme
			req.URL.Host = remote.Host
			req.URL.Path = remote.Path + req.URL.Path
		}

		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			c.JSON(http.StatusBadGateway, gin.H{"error": "TTYD connection lost"})
			logRequest("error", "ttyd connection lost", c, nil, err.Error())
		}

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}