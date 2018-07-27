package drivers

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/go-redis/redis"
	log "github.com/sirupsen/logrus"
	"github.com/twinj/uuid"

	"github.com/victorgama/ponos/models"
)

type SidekiqActiveJobDriver struct {
	c *redis.Client
}

func (d *SidekiqActiveJobDriver) Configure(config *models.Config) error {
	redisOpts, err := redis.ParseURL(config.Redis.URL)
	if err != nil {
		return err
	}
	d.c = redis.NewClient(redisOpts)
	if err := d.c.Ping().Err(); err != nil {
		return err
	}
	return nil
}

func (d *SidekiqActiveJobDriver) Execute(task *models.ScheduleItem) error {
	queue, data := d.serialize(task)
	pipe := d.c.Pipeline()
	pipe.SAdd("queues", queue)
	pipe.LPush("queue:"+queue, data)
	_, err := pipe.Exec()
	return err
}

func (d *SidekiqActiveJobDriver) DryRun(task *models.ScheduleItem) error {
	queue, data := d.serialize(task)
	logger := log.WithField("module", "sidekiq-activejob")
	logger.Infof("Running: \"sadd\" \"queues\" \"%s\"", queue)
	logger.Infof("Running: \"lpush\" \"queue:%s\" \"%s\"", queue, data)
	return nil
}

func (d *SidekiqActiveJobDriver) serialize(task *models.ScheduleItem) (queue, data string) {
	queue = task.Queue
	if queue == "" {
		queue = "default"
	}
	type JobRequest struct {
		Class      string        `json:"class"`
		Wrapped    string        `json:"wrapped"`
		Queue      string        `json:"queue"`
		Args       []interface{} `json:"args"`
		Retry      bool          `json:"retry"`
		JobID      string        `json:"jid"`
		CreatedAt  float64       `json:"created_at"`
		EnqueuedAt float64       `json:"enqueued_at"`
	}

	type JobMainParam struct {
		JobClass      string        `json:"job_class"`
		JobID         string        `json:"job_id"`
		ProviderJobID *string       `json:"provider_job_id"`
		QueueName     string        `json:"queue_name"`
		Priority      *string       `json:"priority"`
		Arguments     []interface{} `json:"arguments"`
		Locale        string        `json:"locale"`
	}

	r := JobRequest{
		Class:   "ActiveJob::QueueAdapters::SidekiqAdapter::JobWrapper",
		Wrapped: task.JobName,
		Queue:   queue,
		Args: []interface{}{
			JobMainParam{
				JobClass:      task.JobName,
				JobID:         uuid.NewV4().String(),
				ProviderJobID: nil,
				QueueName:     queue,
				Priority:      nil,
				Arguments:     []interface{}{},
				Locale:        "en",
			},
		},
		Retry:      task.Retry,
		JobID:      strings.Replace(uuid.NewV4().String(), "-", "", -1)[0:23],
		CreatedAt:  d.ts(),
		EnqueuedAt: d.ts(),
	}
	rawData, err := json.Marshal(&r)
	if err != nil {
		return
	}
	data = string(rawData)
	return
}

func (d *SidekiqActiveJobDriver) ts() float64 {
	return float64(time.Now().UnixNano()/100) / 10000000.0
}
