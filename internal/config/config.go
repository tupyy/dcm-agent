package config

// Configuration holds all configuration for the agent
type Configuration struct {
	// Log
	LogFormat string `debugmap:"visible"`
	LogLevel  string `debugmap:"visible"`

	// NATS
	NatsURL     string `debugmap:"visible"`
	NatsSubject string `debugmap:"visible"`

	// Scheduler
	Workers int `debugmap:"visible"`
}

// Option is a functional option for Configuration
type Option func(*Configuration)

// NewConfigurationWithOptionsAndDefaults creates a new Configuration with defaults and applies options
func NewConfigurationWithOptionsAndDefaults(opts ...Option) *Configuration {
	cfg := &Configuration{
		LogFormat:   "console",
		LogLevel:    "debug",
		NatsURL:     "nats://localhost:4222",
		NatsSubject: "dcm.work",
		Workers:     4,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

// WithLogFormat sets the log format
func WithLogFormat(format string) Option {
	return func(c *Configuration) {
		c.LogFormat = format
	}
}

// WithLogLevel sets the log level
func WithLogLevel(level string) Option {
	return func(c *Configuration) {
		c.LogLevel = level
	}
}

// WithNatsURL sets the NATS server URL
func WithNatsURL(url string) Option {
	return func(c *Configuration) {
		c.NatsURL = url
	}
}

// WithNatsSubject sets the NATS subject to subscribe to
func WithNatsSubject(subject string) Option {
	return func(c *Configuration) {
		c.NatsSubject = subject
	}
}

// WithWorkers sets the number of scheduler workers
func WithWorkers(workers int) Option {
	return func(c *Configuration) {
		c.Workers = workers
	}
}
