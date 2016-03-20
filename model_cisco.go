package main

import (
	"log"
)

type model struct {
	name string
}

type device struct {
	modelName string
	hostPort  string
}

func registerModelCiscoIOS(models map[string]*model) {
	modelName := "cisco-ios"
	models[modelName] = &model{modelName}

	log.Printf("registerModelCiscoIOS: FIXME WRITEME program chat sequence")
}

func createDevice(jaz *app, modelName, id, hostPort, protocols string) {
	log.Printf("createDevice: %s %s %s %s: FIXME WRITEME", modelName, id, hostPort, protocols)
}
