package main

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/olekukonko/tablewriter"
	log "github.com/sirupsen/logrus"
	"github.com/victorgama/cron"

	"github.com/victorgama/ponos/drivers"
	"github.com/victorgama/ponos/models"
)

var executions = map[string]int{}
var lock = sync.Mutex{}

const filePath = "./ponos.toml"

func main() {
	logger := log.WithField("module", "boot")

	lock.Lock()
	logger.Infof("Hello! Ponos 2.0 booting up under PID %d...", os.Getpid())
	logger.Info("Reading configuration from ponos.toml...")
	if _, err := os.Stat("./ponos.toml"); os.IsNotExist(err) {
		logger.Fatal("Cannot find ponos.toml in cwd. Stopping.")
	}

	var config models.Config
	if _, err := toml.DecodeFile(filePath, &config); err != nil {
		logger.Fatal("Error parsing ponos.toml:", err)
	}

	logger.Infof("Looking up for driver: %s", config.Driver)
	driver := drivers.Get(config.Driver)
	if driver == nil {
		options := strings.Join(drivers.Available(), ", ")
		logger.Fatalf("Cannot find driver named '%s'.\nAvailable options are: %s", config.Driver, options)
	}

	if err := driver.Configure(&config); err != nil {
		logger.Fatal("Error configuring driver: %s", err)
	}

	logger.Infof("Building schedule...")
	c := cron.New()
	for k, v := range config.Schedule {
		if err := c.AddNamedFunc(v.Period, k, createTask(k, v, driver)); err != nil {
			log.WithField("module", "scheduler").WithError(err).Fatalf("Cannot schedule job")
		}
	}

	c.Start()
	log.Info("Built schedule:")
	fmt.Printf("%s\n", outputSchedule(c, &config.Schedule))

	go func() {
		ch := make(chan os.Signal)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGUSR2)
		for sig := range ch {
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				log.Info("Received termination signal. Stopping...")
				c.Stop()
				lock.Unlock()
			case syscall.SIGUSR1:
				log.Info("Gathering schedule...")
				fmt.Println(outputSchedule(c, &config.Schedule))
				log.Info("Done.")
			case syscall.SIGUSR2:
				log.Info("Dry-running entire schedule...")
				for _, v := range config.Schedule {
					driver.DryRun(v)
				}
				log.Info("Dry-run: Completed")
			}
		}
	}()

	fmt.Println()
	log.Info("Sending signal USR1 displays information about the schedule and next runs")
	log.Info("               USR2 dumps Redis commands used to perform all jobs (dry-run test)")
	log.Info("               SIGTERM gracefully stops Ponos")
	fmt.Println()

	lock.Lock()
}

func createTask(name string, task *models.ScheduleItem, driver drivers.BaseDriver) func() {
	return func() {
		l := log.WithField("task", name)
		if err := driver.Execute(task); err != nil {
			l.WithError(err).Error("Failed")
		} else {
			l.Info("Enqueued")
			executions[name] = executions[name] + 1
		}
	}
}

func outputSchedule(c *cron.Cron, schedule *map[string]*models.ScheduleItem) string {
	data := [][]string{}
	formatTime := func(t *time.Time) string {
		if t.IsZero() {
			return "Never"
		}

		return t.Format("Mon Jan, 2 15:04:05 MST")
	}

	for _, v := range c.Entries() {
		data = append(data, []string{v.Name, formatTime(&v.Prev), formatTime(&v.Next), (*schedule)[v.Name].Period, strconv.Itoa(executions[v.Name])})
	}

	var b bytes.Buffer
	table := tablewriter.NewWriter(&b)
	table.SetHeader([]string{"Name", "Last Execution", "Next Execution", "Schedule", "Executions"})
	table.AppendBulk(data)
	table.Render()
	return b.String()
}
