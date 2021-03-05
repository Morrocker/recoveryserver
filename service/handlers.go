package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/davecgh/go-spew/spew"
	"github.com/gin-gonic/gin"
	"github.com/morrocker/logger"
	"github.com/morrocker/recoveryserver/recovery"
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
	if err := json.Unmarshal(bodyBytes, &recoveryData); err != nil {
		logger.Error("For now just json error")
	}

	hash := s.Director.AddRecovery(recoveryData)

	msg := fmt.Sprintf("Recovery %v added with Id:%s", recoveryData, hash)
	c.Data(http.StatusOK, "text", []byte(msg))

}

func (s *Service) addRecoveries(c *gin.Context) {
	bodyBytes, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error("%v", err)
		return
	}

	var data recovery.Multiple
	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		logger.Error("For now just json error")
	}

	var msg string
	for _, recovery := range data.Recoveries {
		hash := s.Director.AddRecovery(recovery)
		msg = fmt.Sprintf("Recovery %v added with id:%s.\n%s", recovery, hash, msg)
	}
	c.Data(http.StatusOK, "text", []byte(msg))
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

func (s *Service) changePriority(c *gin.Context) {
	id, ok := c.GetQuery("Id")
	if !ok {
		c.JSON(
			http.StatusBadRequest,
			errors.New("Misssing recovery Id in request"),
		)
		return
	}

	n, ok := c.GetQuery("Priority")
	if !ok {
		c.JSON(
			http.StatusBadRequest,
			errors.New("Misssing set priority in request"),
		)
		return
	}
	x, err := strconv.Atoi(n)
	if err != nil {
		logger.Error("Error converting String to Int")
	}
	msg := fmt.Sprintf("Recovery %s set priority to %d:", id, x)
	c.Data(http.StatusOK, "text", []byte(msg))
	s.Director.ChangePriority(id, x)
}

func (s *Service) pauseDirector(c *gin.Context) {
	s.Director.PausePicker()
	c.Data(http.StatusOK, "text", []byte("Director set to Pause"))
}
func (s *Service) runDirector(c *gin.Context) {
	s.Director.RunPicker()
	c.Data(http.StatusOK, "text", []byte("Director set to Run"))
}

func (s *Service) queueRecovery(c *gin.Context) {
	id, ok := c.GetQuery("Id")
	if !ok {
		c.JSON(
			http.StatusBadRequest,
			errors.New("Misssing recovery Id in request"),
		)
		return
	}
	if err := s.Director.QueueRecovery(id); err != nil {
		c.JSON(
			http.StatusBadRequest,
			errors.New("Recovery does not exist"),
		)
		return
	}
	msg := fmt.Sprintf("Recovery %s queued", id)
	c.Data(http.StatusOK, "text", []byte(msg))
}
