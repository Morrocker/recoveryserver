package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/morrocker/errors"
	"github.com/morrocker/logger"
	"github.com/morrocker/recoveryserver/recovery"
)

func (s *Service) addRecovery(c *gin.Context) {
	logger.TaskV("Adding new recovery")
	errPath := "service.addRecovery()"
	bodyBytes, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		err = errors.New(errPath, err)
		c.Data(http.StatusInternalServerError, "text", []byte(err.Error()))
		logger.Error("Error while adding recovery: %s", err)
		return
	}

	var recoveryData *recovery.Data
	if err := json.Unmarshal(bodyBytes, &recoveryData); err != nil {
		err = errors.New(errPath, err)
		c.Data(http.StatusInternalServerError, "text", []byte(err.Error()))
		logger.Error("Error while adding recovery: %s", err)
		return
	}

	hash, err := s.Director.AddRecovery(recoveryData)
	if err != nil {
		err := errors.New(errPath, err)
		c.Data(http.StatusInternalServerError, "text", []byte(err.Error()))
		logger.Error("%s", err)
		return
	}
	c.Data(http.StatusOK, "text", []byte(fmt.Sprintf("Recovery %v added with Id:%s", recoveryData, hash)))
}

func (s *Service) addRecoveries(c *gin.Context) {
	logger.TaskV("Adding new set of recovery")
	errPath := "service.addRecoveries()"
	bodyBytes, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		err = errors.New(errPath, err)
		c.Data(http.StatusInternalServerError, "text", []byte(err.Error()))
		logger.Error("%s", err)
		return
	}

	var data recovery.Multiple
	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		err = errors.New(errPath, err)
		c.Data(http.StatusInternalServerError, "text", []byte(err.Error()))
		logger.Error("%s", err)
		return
	}

	var msg string
	for _, recovery := range data.Recoveries {
		hash, err := s.Director.AddRecovery(recovery)
		if err != nil {
			err = errors.Extend(errPath, err)
			c.Data(http.StatusInternalServerError, "text", []byte(err.Error()))
			logger.Error("%s", err)
			return
		}
		msg = fmt.Sprintf("Recovery %v added with id:%s.\n%s", recovery, hash, msg)
	}
	c.Data(http.StatusOK, "text", []byte(msg))
}

func (s *Service) startRecovery(c *gin.Context) {
	errPath := "service.startRecovery()"
	id, ok := c.GetQuery("Id")
	if !ok {
		err := errors.New(errPath, "Error starting recovery")
		c.Data(http.StatusBadRequest, "text", []byte(err.Error()))
		logger.Error("%s", err)
		return
	}
	if err := s.Director.StartRecovery(id); err != nil {
		err := errors.New(errPath, "Error starting recovery")
		c.Data(http.StatusBadRequest, "text", []byte(err.Error()))
		logger.Error("%s", err)
		return
	}
	c.Data(http.StatusOK, "text", []byte(fmt.Sprintf("Starting Recovery with id:%s", id)))
}

func (s *Service) startRecoveryGroup(c *gin.Context) {

}

func (s *Service) pauseRecovery(c *gin.Context) {
	errPath := "service.pauseRecovery()"
	id, ok := c.GetQuery("Id")
	if !ok {
		err := errors.New(errPath, "Query missing recovery Id")
		c.Data(http.StatusBadRequest, "text", []byte(err.Error()))
		return
	}
	s.Director.PauseRecovery(id)
	c.Data(http.StatusOK, "text", []byte(fmt.Sprintf("Pausing Recovery with id:%s", id)))
}

func (s *Service) pauseRecoveryGroup(c *gin.Context) {

}

func (s *Service) deleteRecovery(c *gin.Context) {

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
	errPath := "service.queueRecovery()"
	id, ok := c.GetQuery("Id")
	if !ok {
		err := errors.New(errPath, "Query missing recovery Id")
		c.Data(http.StatusBadRequest, "text", []byte(err.Error()))
		return
	}
	if err := s.Director.QueueRecovery(id); err != nil {
		err := errors.New(errPath, err)
		c.Data(http.StatusBadRequest, "text", []byte(err.Error()))
		return
	}
	c.Data(http.StatusOK, "text", []byte(fmt.Sprintf("Recovery %s queued", id)))
}

func (s *Service) setDestination(c *gin.Context) {
	errPath := "service.setDestination"
	id, ok := c.GetQuery("Id")
	if !ok {
		err := errors.New(errPath, "Query missing recovery Id")
		c.Data(http.StatusBadRequest, "text", []byte(err.Error()))
		return
	}
	destination, ok := c.GetQuery("Destination")
	if !ok {
		err := errors.New(errPath, "Query missing recovery destination")
		c.Data(http.StatusBadRequest, "text", []byte(err.Error()))
		return
	}
	if err := s.Director.SetDestination(id, destination); err != nil {
		err := errors.New(errPath, err)
		c.Data(http.StatusBadRequest, "text", []byte(err.Error()))
		return
	}

}

func (s *Service) changePriority(c *gin.Context) {
	errPath := "service.changePriority()"
	id, ok := c.GetQuery("Id")
	if !ok {
		err := errors.New(errPath, "Query missing recovery Id")
		c.Data(http.StatusBadRequest, "text", []byte(err.Error()))
		logger.Error("%s", err)
		return
	}

	n, ok := c.GetQuery("Priority")
	if !ok {
		err := errors.New(errPath, "Query missing priority")
		c.Data(http.StatusBadRequest, "text", []byte(err.Error()))
		logger.Error("%s", err)
		return
	}
	x, err := strconv.Atoi(n)
	if err != nil {
		err := errors.New(errPath, err)
		c.Data(http.StatusInternalServerError, "text", []byte(err.Error()))
		logger.Error("%s", err)
		return
	}
	if err := s.Director.ChangePriority(id, x); err != nil {
		err := errors.New(errPath, err)
		c.Data(http.StatusBadRequest, "text", []byte(err.Error()))
		logger.Error("%s", err)
		return
	}
	c.Data(http.StatusOK, "text", []byte(fmt.Sprintf("Recovery %s set priority to %d:", id, x)))
}
