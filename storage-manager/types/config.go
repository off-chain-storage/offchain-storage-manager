package types

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	Log  LogConfig
	GRPC GRPCConfig
	DB   DatabaseConfig
}

type LogConfig struct {
	Format string
	Level  logrus.Level
}

type GRPCConfig struct {
	ListenAddr string
	MaxMsgSize int
	Timeout    time.Duration
}

type RedisConfig struct {
	Host string
	Port string
	Name int
	User string
	Pass string
}

type MongoDBConfig struct {
	Host       string
	Port       string
	ReplicaSet string
	DBName     string
	Collection string
	User       string
	Pass       string
}

type DatabaseConfig struct {
	MongoDB MongoDBConfig
	Redis   RedisConfig
}

func logConfig() (LogConfig, error) {
	logLevel := viper.GetString("log.level")
	if logLevel == "" {
		logLevel = "info"
		logrus.Warn("Log level not set, using default: info")
	}

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return LogConfig{}, fmt.Errorf("failed to parse log level: %w", err)
	}

	logFormat := viper.GetString("log.format")
	if logFormat == "" {
		logFormat = "text"
		logrus.Warn("Log format not set, using default: text")
	}

	logrus.SetLevel(level)

	switch logFormat {
	case "text":
		// formatter := new(prefixed.TextFormatter)
		// formatter.TimestampFormat = time.DateTime
		// formatter.FullTimestamp = true

		// // Color Options is not recommended when logFileName is set
		// formatter.DisableColors = viper.GetString("log.file") != ""
		// logrus.SetFormatter(formatter)
		logrus.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: time.DateTime,
			FullTimestamp:   true,
		})
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.DateTime,
		})
	case "fluentd":
		// TODO: set fluentd formatter
	default:
		return LogConfig{}, fmt.Errorf("invalid log format: %s. Valid choices are 'json' or 'text' or 'fluentd'", logFormat)
	}

	return LogConfig{
		Format: logFormat,
		Level:  level,
	}, nil
}

func grpcConfig() (GRPCConfig, error) {
	grpc := GRPCConfig{
		ListenAddr: viper.GetString("grpc.listen_addr"),
		MaxMsgSize: viper.GetInt("grpc.max_msg_size"),
		Timeout:    viper.GetDuration("grpc.timeout"),
	}

	return grpc, nil
}

func databaseConfig() (DatabaseConfig, error) {
	redis := RedisConfig{
		Host: viper.GetString("db.redis.address"),
		Port: viper.GetString("db.redis.port"), // default fallback; consider parsing if port is part of address
		Name: viper.GetInt("db.redis.dbname"),
		User: "",
		Pass: viper.GetString("db.redis.password"),
	}

	mongoDB := MongoDBConfig{
		Host:       viper.GetString("db.mongodb.host"),
		Port:       viper.GetString("db.mongodb.port"),
		ReplicaSet: viper.GetString("db.mongodb.replica_set"),
		DBName:     viper.GetString("db.mongodb.dbname"),
		Collection: viper.GetString("db.mongodb.collection"),
		User:       viper.GetString("db.mongodb.user"),
		Pass:       viper.GetString("db.mongodb.password"),
	}

	return DatabaseConfig{
		Redis:   redis,
		MongoDB: mongoDB,
	}, nil
}

func LoadManagerConfig() (*Config, error) {
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "text")

	// Initialize log config
	logConfig, err := logConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize log: %w", err)
	}
	logrus.SetLevel(logConfig.Level)

	// Initialize gRPC config
	grpcConfig, err := grpcConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize gRPC config: %w", err)
	}

	// Initialize database config
	databaseConfig, err := databaseConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database config: %w", err)
	}

	return &Config{
		Log:  logConfig,
		GRPC: grpcConfig,
		DB:   databaseConfig,
	}, nil
}
