package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/morrocker/errors"
	"github.com/morrocker/log"
	"github.com/morrocker/recoveryserver/pdf"
	"github.com/morrocker/recoveryserver/recovery"
)

func (s *Service) addRecovery(c *gin.Context) {
	log.TaskV("Adding new recovery")
	op := "service.addRecovery()"
	bodyBytes, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		badRequest(c, op, err)
		return
	}

	var recoveryData *recovery.Data
	if err := json.Unmarshal(bodyBytes, &recoveryData); err != nil {
		badRequest(c, op, err)
		return
	}

	if err := s.Director.AddRecovery(recoveryData); err != nil {
		badRequest(c, op, err)
		return
	}
	c.Data(http.StatusOK, "text", []byte("ok"))
}

func (s *Service) startRecovery(c *gin.Context) {
	op := "service.startRecovery()"
	id, err := getQueryInt(c, "id")
	if err != nil {
		badRequest(c, op, err)
		return
	}
	if err := s.Director.StartRecovery(id); err != nil {
		badRequest(c, op, err)
		return
	}
	c.Data(http.StatusOK, "text", []byte("ok"))
}

func (s *Service) pauseRecovery(c *gin.Context) {
	op := "service.pauseRecovery()"
	id, err := getQueryInt(c, "id")
	if err != nil {
		badRequest(c, op, err)
		return
	}
	s.Director.PauseRecovery(id)
	c.Data(http.StatusOK, "text", []byte("ok"))
}

func (s *Service) cancelRecovery(c *gin.Context) {
	op := "service.cancelRecovery()"
	id, err := getQueryInt(c, "id")
	if err != nil {
		badRequest(c, op, err)
		return
	}
	s.Director.CancelRecovery(id)
	c.Data(http.StatusOK, "text", []byte("ok"))
}

// func (s *Service) queueRecovery(c *gin.Context) {
// 	// op := "service.queueRecovery()"
// 	// id, err := getQueryInt(c, "id")
// 	// if err != nil {
// 	// 	badRequest(c, op, err)
// 	// 	return
// 	// }
// 	// if err := s.Director.QueueRecovery(id); err != nil {
// 	// 	badRequest(c, op, err)
// 	// 	return
// 	// }
// 	c.Data(http.StatusOK, "text", []byte("ok"))
// }

func (s *Service) setDestination(c *gin.Context) {
	op := "service.setDestination"
	id, err := getQueryInt(c, "id")
	if err != nil {
		badRequest(c, op, err)
		return
	}
	destination, err := getQuery(c, "Destination")
	if err != nil {
		badRequest(c, op, err)
		return
	}
	if err := s.Director.SetDestination(id, destination); err != nil {
		badRequest(c, op, err)
		return
	}

}

func (s *Service) changePriority(c *gin.Context) {
	op := "service.changePriority()"
	id, err := getQueryInt(c, "id")
	if err != nil {
		badRequest(c, op, err)
		return
	}

	n, err := getQueryInt(c, "priority")
	if err != nil {
		badRequest(c, op, err)
		return
	}
	if err := s.Director.ChangePriority(id, n); err != nil {
		badRequest(c, op, err)
		return
	}
	c.Data(http.StatusOK, "text", []byte("ok"))
}

func (s *Service) writeDelivery(c *gin.Context) {
	op := "service.generateDelivery()"
	bodyBytes, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		badRequest(c, op, err)
		return
	}
	var deliveryData *pdf.Delivery
	if err := json.Unmarshal(bodyBytes, &deliveryData); err != nil {
		badRequest(c, op, err)
		return
	}

	out, err := s.Director.WriteDelivery(deliveryData)
	if err != nil {
		badRequest(c, op, err)
		return
	}

	c.Data(http.StatusOK, "text", []byte(fmt.Sprintf("Delivery pdf wrote to %s", out)))
}

func (s *Service) shutdown(c *gin.Context) {
	log.Info("Shutting down server")
	s.Close()
}

func (s *Service) getDevices(c *gin.Context) {
	op := "service.getDevices()"
	devs, err := s.Director.Devices()
	if err != nil {
		badRequest(c, op, err)
		return
	}

	bytes, err := json.Marshal(devs)
	if err != nil {
		badRequest(c, op, err)
		return
	}
	c.Data(http.StatusOK, "json", bytes)
}

func (s *Service) mountDevice(c *gin.Context) {
	op := "service.mountDevice()"
	serial, err := getQuery(c, "serial")
	if err != nil {
		badRequest(c, op, err)
		return
	}
	if err := s.Director.MountDisk(serial); err != nil {
		badRequest(c, op, err)
		return
	}
	c.Data(http.StatusOK, "text", []byte("ok"))
}

func (s *Service) unmountDevice(c *gin.Context) {
	op := "service.unmountDevice()"
	serial, err := getQuery(c, "serial")
	if err != nil {
		badRequest(c, op, err)
		return
	}
	if err := s.Director.UnmountDisk(serial); err != nil {
		badRequest(c, op, err)
		return
	}
	c.Data(http.StatusOK, "text", []byte("ok"))
}

func badRequest(c *gin.Context, op string, err error) {
	err = errors.Extend(op, err)
	c.Data(http.StatusInternalServerError, "text", []byte(err.Error()))
	log.Errorln(err)
}

func getQuery(c *gin.Context, key string) (string, error) {
	value, ok := c.GetQuery(key)
	if !ok {
		return "", errors.New("service.getQueryInt()", "key value '"+key+"' missing from query")
	}
	return value, nil
}

func getQueryInt(c *gin.Context, key string) (int, error) {
	value, ok := c.GetQuery(key)
	if !ok {
		return 0, errors.New("service.getQueryInt()", "key value '"+key+"' missing from query")
	}
	v, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.New("service.getQueryInt()", err)
	}
	return v, nil
}

// func (s *Service) test(c *gin.Context) {
// 	op := "test.test"
// 	id, err := getQueryInt(c, "Id")
// 	if err != nil {
// 		badRequest(c, op, err)
// 		return
// 	}

// 	serial, err := getQuery(c, "Serial")
// 	if err != nil {
// 		badRequest(c, op, err)
// 		return
// 	}
// 	// d.Director.SetDiskID(serial, id)
// }
