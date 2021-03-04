package service

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/Morrocker/logger"
	"github.com/davecgh/go-spew/spew"
	"github.com/gin-gonic/gin"
	"github.com/recoveryserver/recovery"
)

func (s *Service) testFunc(c *gin.Context) {
	bodyBytes, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error("%v", err)
		return
	}

	var recoveryData recovery.Data
	json.Unmarshal(bodyBytes, &recoveryData)

	spew.Dump(recoveryData)
}

func (s *Service) addRecovery(c *gin.Context) {
	bodyBytes, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error("%v", err)
		return
	}

	var recoveryData recovery.Data
	json.Unmarshal(bodyBytes, &recoveryData)

	hash := s.Director.AddRecovery(recoveryData)

	c.Data(http.StatusOK, "text", []byte("Recovery added with Id:"+hash))

}

func (s *Service) startRecovery(c *gin.Context) {

}

func (s *Service) startRecoveryGroup(c *gin.Context) {

}

func (s *Service) pauseRecovery(c *gin.Context) {

}

func (s *Service) pauseRecoveryGroup(c *gin.Context) {

}

func (s *Service) deleteRecovery(c *gin.Context) {

}
