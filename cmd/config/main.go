package main

import (
	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
	"nullprogram.com/x/optparse"

	"fmt"
	"os"
	"strings"

	service_config "github.com/anvox/golem-service/pkg/config"
)

const VERSION = "v0.0.1"

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	if os.Getenv("DEBUG") == "1" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

func main() {
	options := []optparse.Option{
		{"help", 'h', optparse.KindNone},
		{"version", 'v', optparse.KindNone},
		{"value-only", 'o', optparse.KindNone},
		{"environment", 'e', optparse.KindRequired},
		{"stack", 't', optparse.KindRequired},
		{"service", 's', optparse.KindRequired},
	}

	results, rest, err := optparse.Parse(options, os.Args)

	if err != nil {
		fmt.Println("Cannot parse arguments!")
		printHelp()
		os.Exit(1)
	}

	cmd := "list"
	id := service_config.Identifier{}
	valueOnly := false
	for _, result := range results {
		switch result.Long {
		case "help":
			printHelp()
			os.Exit(0)
		case "version":
			printVersion()
			os.Exit(0)
		case "value-only":
			valueOnly = true
		case "environment":
			id.Environment = result.Optarg
		case "stack":
			id.Stack = result.Optarg
		case "service":
			id.Service = result.Optarg
		}
	}

	if id.Environment == "" || id.Stack == "" {
		fmt.Println("Environment and Stack are required. Please run with -h/--help for more info.")
		os.Exit(2)
	}

	if len(rest) >= 1 {
		cmd = rest[0]
		rest = rest[1:len(rest)]

		if cmd != "list" && cmd != "get" && cmd != "set" {
			fmt.Println("Subcommand must be list|set|get. Please run with -h/--help for more info.")
			os.Exit(3)
		}
	}

	if cmd == "list" && len(rest) > 0 {
		fmt.Println("Subcommand list requires no argument. Please run with -h/--help for more info.")
		os.Exit(4)
	} else if cmd == "get" && len(rest) == 0 {
		fmt.Println("Subcommand get requires at least 1 argument. Please run with -h/--help for more info.")
		os.Exit(5)
	} else if cmd == "set" && len(rest) == 0 {
		fmt.Println("Subcommand set requires at least 1 argument. Please run with -h/--help for more info.")
		os.Exit(6)
	}

	if valueOnly {
		// valueOnly with non-get command or get more than 1 key
		if cmd != "get" || len(rest) != 1 {
			fmt.Println("Subcommand get with --value-only require exactly 1 argument. Please run with -h/--help for more info.")
			os.Exit(10)
		}
	}

	log.Debugf("Identifier: %+v", id)
	log.Debugf("Command: %s", cmd)
	for _, configArg := range rest {
		log.Debugf("Arg: %+v", configArg)
	}
	switch cmd {
	case "list":
		configs, err := service_config.List(id)
		if err != nil {
			fmt.Printf("Failed to fetch configuration: %+v\n", err)
			os.Exit(7)
		}
		printConfigurations(configs)
	case "get":
		configs, err := service_config.GetEnv(id, rest)
		if err != nil {
			fmt.Printf("Failed to fetch configuration: %+v\n", err)
			os.Exit(8)
		}
		if valueOnly {
			if len(configs) != 1 {
				os.Exit(11)
			} else {
				fmt.Println(configs[0].Value)
			}
		} else {
			printConfigurations(configs)
		}
	case "set":
		newEnvs := []service_config.Configuration{}
		for _, envString := range rest {
			env, err := service_config.ParseEnvString(envString)
			if err == nil {
				newEnvs = append(newEnvs, env)
			} else {
				// Panic!!!
				fmt.Printf("Invalid argument \"%s\". "+
					"Environment variable argument must in pattern \"NAME=value\"\n", envString)
				os.Exit(9)
			}
		}
		service_config.SetEnv(id, newEnvs)
	default:
		return
	}
}

func printConfigurations(configurations []service_config.Configuration) {
	colSize := 0
	configurations = sortConfig(configurations)
	for _, config := range configurations {
		if len(config.Name) > colSize {
			colSize = len(config.Name)
		}
	}
	for _, config := range configurations {
		cyan := color.New(color.FgCyan).SprintFunc()
		padding := strings.Repeat(" ", colSize-len(config.Name))
		fmt.Printf("%s:%s %s\n", cyan(config.Name), padding, config.Value)
	}
}
