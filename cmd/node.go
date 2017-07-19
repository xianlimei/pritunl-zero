package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/dropbox/godropbox/errors"
	"github.com/gin-gonic/gin"
	"github.com/pritunl/pritunl-zero/config"
	"github.com/pritunl/pritunl-zero/constants"
	"github.com/pritunl/pritunl-zero/errortypes"
	"github.com/pritunl/pritunl-zero/mhandlers"
	"github.com/pritunl/pritunl-zero/node"
	"github.com/pritunl/pritunl-zero/settings"
	"gopkg.in/mgo.v2/bson"
	"net/http"
	"time"
)

func Node() (err error) {
	if constants.Production {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	if !constants.Production {
		router.Use(gin.Logger())
	}

	mhandlers.Register(router)

	nde := node.Node{
		Id:   bson.ObjectIdHex(config.Config.ManagementNodeId),
		Type: node.Management,
	}
	nde.Init()

	server := &http.Server{
		Addr:           "0.0.0.0:8443",
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 4096,
	}

	logrus.WithFields(logrus.Fields{
		"name":       settings.System.Name,
		"production": constants.Production,
	}).Info("cmd.app: Starting node")

	err = server.ListenAndServe()
	if err != nil {
		err = &errortypes.UnknownError{
			errors.Wrap(err, "node: Server listen failed"),
		}
		return
	}

	return
}
