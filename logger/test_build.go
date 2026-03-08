// +build ignore

package main

import (
	"fmt"
	"gitlab.com/bot3342545/il-dashboard/logger"
)

func main() {
	if err := logger.Setup("logs/test"); err != nil {
		panic(err)
	}
	
	logger.Info("Test message")
	logger.Infof("Test formatted: %s", "hello")
	
	fmt.Println("Logger test successful!")
}
