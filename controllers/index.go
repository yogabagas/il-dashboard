package controllers

import (
	"sync"
	"time"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/logger"
)

type IndexController struct {
}

func (o *IndexController) Init() {
	gocom.GET("instant-link/health-check", o.healthCheck)
}

func (o *IndexController) healthCheck(ctx gocom.Context) error {
	ctx.SetHeader("Access-Control-Allow-Private-Network", "true")

	status := map[string]interface{}{
		"status": "ok",
		"time":   time.Now().String(),
	}

	logger.Infof("[IndexController healthCheck] start.")
	logger.Infof("[IndexController healthCheck] status is %v", status)
	return ctx.SendResult(status)
}

////////////////////////////////////////

var indexController *IndexController
var indexControllerOnce sync.Once

func GetIndexController() *IndexController {
	if indexController == nil {
		indexControllerOnce.Do(func() {
			indexController = &IndexController{}
		})
	}

	return indexController
}
