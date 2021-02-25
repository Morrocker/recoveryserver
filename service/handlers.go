package service

import (
	"github.com/Morrocker/logger"
	"github.com/gin-gonic/gin"
)

func (s *Service) testFunc(c *gin.Context) {
	logger.Info("Function got called!")
}
