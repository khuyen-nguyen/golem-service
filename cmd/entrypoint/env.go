package main

import (
	service_config "github.com/anvox/golem-service/pkg/config"
	service_ps "github.com/anvox/golem-service/pkg/process"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"

	"fmt"
	"os"
)

func getEnvVars(config Configuration) (map[string]string, error) {
	envKeys, err := fetchEnvList("./.env.sample")
	if err != nil {
		return nil, err
	}
	var envVars map[string]string
	switch config.Platform {
	case "aws-ecs":
		envVars, err = loadEnvVarFromSsm(config, envKeys)
		if err != nil {
			return make(map[string]string), err
		}
	default: // local
		envVars = loadEnvVerFromLocalFile(envKeys)
	}

	return envVars, nil
}

func fetchConfigFromEnvionment(config Configuration) (Configuration, error) {
	config.Environment = os.Getenv("GOLEM_ENVIRONMENT")
	config.Stack = os.Getenv("GOLEM_STACK")
	config.Service = os.Getenv("GOLEM_SERVICE")

	if platform := os.Getenv("CLOUD_PROVIDER"); platform != "" {
		config.Platform = platform
	}

	if len(os.Args) == 2 {
		config.ProcType = os.Args[1]
	} else if procType := os.Getenv("PROC_TYPE"); procType != "" {
		config.ProcType = procType
	}

	config.Cluster = service_ps.EcsClusterName(service_ps.Identifier{
		Environment: config.Environment,
		Stack:       config.Stack,
		Service:     config.Service,
	})

	if config.Cluster == "" {
		return Configuration{}, fmt.Errorf("Unable to resolve Cluster from Envionment variables")
	}

	return config, nil
}

func fetchEnvList(envSamplePath string) ([]string, error) {
	myEnv, err := godotenv.Read(envSamplePath)
	if err != nil {
		return nil, err
	}

	keys := []string{}
	for key := range myEnv {
		keys = append(keys, key)
	}

	return keys, nil
}

func loadEnvVarFromSsm(config Configuration,
	environmentKeys []string) (map[string]string, error) {
	identifier := service_config.Identifier{
		Environment: config.Environment,
		Stack:       config.Stack,
		Service:     config.Service,
	}
	envConfigs, err := service_config.GetEnv(identifier, environmentKeys)
	if err != nil {
		return nil, err
	}

	envVars := make(map[string]string)
	for _, envConfig := range envConfigs {
		envVars[envConfig.Name] = envConfig.Value
	}

	return envVars, nil
}

func loadEnvVerFromLocalFile(environmentKeys []string) map[string]string {
	composedEnvs := make(map[string]string)
	if envs, err := godotenv.Read(".env.local"); err == nil {
		composedEnvs = mergeEnvs(composedEnvs, envs)
		log.Debugln("Loaded .env.local")
	} else {
		log.Debugf("Failed to load .env.local: %+v\n", err)
	}
	if envs, err := godotenv.Read(".env.development"); err == nil {
		composedEnvs = mergeEnvs(composedEnvs, envs)
		log.Debugln("Loaded .env.development")
	} else {
		log.Debugf("Failed to load .env.development: %+v\n", err)
	}

	return composedEnvs
}

func mergeEnvs(envs1 map[string]string, envs2 map[string]string) map[string]string {
	composedEnvs := make(map[string]string)
	for key, value := range envs1 {
		composedEnvs[key] = value
	}
	for key, value := range envs2 {
		composedEnvs[key] = value
	}

	return composedEnvs
}
