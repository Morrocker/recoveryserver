package service

import "github.com/gin-gonic/gin"

// handleCORS returns a middleware that adds CORS headers allowing everything
func (s *Service) handleCORS(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "*")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*") // TODO: Changing this would improve security?
	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, PUT, PATCH, DELETE, OPTIONS")
	if c.Request.Method == "OPTIONS" {
		c.Writer.Header().Set("Access-Control-Max-Age", "600")
		c.AbortWithStatus(204)
		return
	}
	c.Next()
}
