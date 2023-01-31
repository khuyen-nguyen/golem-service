package main

import (
	log "github.com/sirupsen/logrus"

	"fmt"
	"io/ioutil"
	"strings"
)

var (
	BUILTIN_DEVELOPMENT_PROC_TYPES = []string{
		"wait",
		"bash",
	}

	BUILTIN_PROC_TYPES = []string{
		"ssh",
	}
)

func procCommand(config Configuration) (string, error) {
	procTypes, err := parseProcfile("./Procfile")
	if err != nil {
		return "", err
	}

	for _, procType := range BUILTIN_PROC_TYPES {
		procTypes[procType] = procType
	}
	if config.Platform == "local" {
		devProcTypes, err := parseProcfile("./Procfile.development")
		if err != nil {
			log.Warnf("Unable to load Procfile.development: %s", err)
		}

		for procType, command := range devProcTypes {
			procTypes[procType] = command
		}
		for _, procType := range BUILTIN_DEVELOPMENT_PROC_TYPES {
			procTypes[procType] = procType
		}
	}

	for procType, command := range procTypes {
		if procType == config.ProcType {
			return command, nil
		}
	}

	return "", fmt.Errorf("Unable to find executable for procType=%s", config.ProcType)
}

func parseProcfile(filePath string) (map[string]string, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	procs := make(map[string]string)
	for _, line := range strings.Split(string(content), "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			procs[parts[0]] = parts[1]
		} else {
			if line != "" {
				log.Warnf("Unable to parse line [%s] from %s", line, filePath)
			}
		}
	}

	return procs, nil
}
