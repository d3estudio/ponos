package models

type RedisConfig struct {
	URL string `toml:"url"`
}

type ScheduleItem struct {
	Period  string `toml:"period"`
	JobName string `toml:"job"`
	Queue   string `toml:"queue"`
	Retry   bool   `toml:"retry"`
}

type Config struct {
	Driver   string                   `toml:"driver"`
	Redis    *RedisConfig             `toml:"redis"`
	Schedule map[string]*ScheduleItem `toml:"schedule"`
}
