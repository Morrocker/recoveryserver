package service

import (
	"encoding/json"
	"io/ioutil"

	"github.com/Morrocker/logger"
	"github.com/davecgh/go-spew/spew"
	"github.com/gin-gonic/gin"
	d "github.com/recoveryserver/recovery"
	"github.com/recoveryserver/utils"
)

func (s *Service) testFunc(c *gin.Context) {
	bodyBytes, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error("%v", err)
		return
	}

	var recoveryData d.Recovery
	json.Unmarshal(bodyBytes, &recoveryData)

	spew.Dump(recoveryData)
}

func (s *Service) AddRecovery(c *gin.Context) {
	bodyBytes, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error("%v", err)
		return
	}

	hash := utils.RandString(8)
	json.Unmarshal(bodyBytes, s.recoveries[hash])

}
