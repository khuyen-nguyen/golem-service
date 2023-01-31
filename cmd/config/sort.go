package main

import (
	service_config "github.com/anvox/golem-service/pkg/config"

	"sort"
)

type ListConfig []service_config.Configuration

func (a ListConfig) Len() int           { return len(a) }
func (a ListConfig) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ListConfig) Less(i, j int) bool { return a[i].Name < a[j].Name }

func sortConfig(configurations []service_config.Configuration) []service_config.Configuration {
	sort.Sort(ListConfig(configurations))

	return configurations
}
