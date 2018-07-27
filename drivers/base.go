package drivers

import "github.com/victorgama/ponos/models"

var drivers = map[string]func() BaseDriver{}

func init() {
	drivers["sidekiq-activejob"] = func() BaseDriver { return &SidekiqActiveJobDriver{} }
}

func Get(name string) BaseDriver {
	if item, ok := drivers[name]; ok {
		return item()
	}
	return nil
}

func Available() []string {
	names := []string{}
	for k := range drivers {
		names = append(names, k)
	}
	return names
}

type BaseDriver interface {
	Configure(config *models.Config) error
	Execute(task *models.ScheduleItem) error
	DryRun(task *models.ScheduleItem) error
}
