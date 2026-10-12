package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/caarlos0/env/v11"
	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sharasha07/clash-bot/internal/data"
	"github.com/sharasha07/clash-bot/internal/objectstorage"
)

type application struct {
	logger        *slog.Logger
	cfg           Config
	models        data.Models
	s3Client      objectstorage.ObjectStorage
	brokerChannel *amqp.Channel
}

type Config struct {
	Port   int `env:"PORT,required"`
	Server struct {
		ReadTimeout  time.Duration `env:"SERVER_READ_TIMEOUT,required"`
		WriteTimeout time.Duration `env:"SERVER_WRITE_TIMEOUT,required"`
		IdleTimeout  time.Duration `env:"SERVER_IDLE_TIMEOUT,required"`
	}
	CORS struct {
		TrustedOrigins []string `env:"TRUSTED_ORIGINS,required"`
	}
	Limiter struct {
		RPS     int  `env:"LIMITER_RPS,required"`
		Burst   int  `env:"LIMITER_BURST,required"`
		Enabled bool `env:"LIMITER_ENABLED,required"`
	}
	DB struct {
		DSN         string        `env:"DB_DSN,required"`
		MinConns    int32         `env:"DB_MIN_CONNS,required"`
		MaxConns    int32         `env:"DB_MAX_CONNS,required"`
		MaxIdleTime time.Duration `env:"DB_MAX_IDLE_TIME,required"`
	}
	JWT struct {
		Secret     string        `env:"JWT_SECRET,required"`
		AccessTTL  time.Duration `env:"JWT_ACCESS_TTL,required"`
		RefreshTTL time.Duration `env:"JWT_REFRESH_TTL,required"`
	}
	R2 struct {
		AccessKey       string `env:"R2_ACCESS_KEY,required"`
		SecretAccessKey string `env:"R2_SECRET_ACCESS_KEY,required"`
		Bucket          string `env:"R2_BUCKET,required"`
		PublicURL       string `env:"R2_PUBLIC_URL,required"`
		S3ApiEndpoint   string `env:"S3_API_ENDPOINT,required"`
	}
	RabbitMQ struct {
		URL string `env:"RABBITMQ_URL,required"`
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	exitOnError := func(err error, msg string) {
		if err != nil {
			logger.Error(msg, "err", err)
			os.Exit(1)
		}
	}

	var cfg Config
	err := env.Parse(&cfg)
	exitOnError(err, "failed to parse env vars")

	pool, err := connectToDB(cfg)
	exitOnError(err, "failed to connect to db")
	defer pool.Close()
	logger.Info("successfully connected to db")

	httpClient := &http.Client{Timeout: 10 * time.Second}

	s3Client := s3.New(s3.Options{
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.R2.AccessKey, cfg.R2.SecretAccessKey, ""),
		Region:       "auto",
		BaseEndpoint: aws.String(cfg.R2.S3ApiEndpoint),
		UsePathStyle: true,
		HTTPClient:   httpClient,
	})

	conn, err := amqp.Dial(cfg.RabbitMQ.URL)
	exitOnError(err, "failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	exitOnError(err, "failed to open a channel")
	defer ch.Close()

	_, err = ch.QueueDeclare(
		"generate_reply",
		true,
		false,
		false,
		false,
		amqp.Table{
			amqp.QueueTypeArg: amqp.QueueTypeQuorum,
		},
	)
	exitOnError(err, "failed to declare a queue")

	app := &application{
		logger:        logger,
		cfg:           cfg,
		models:        data.NewModels(pool),
		s3Client:      s3Client,
		brokerChannel: ch,
	}

	err = app.serve()
	exitOnError(err, "server failed")
}

func connectToDB(cfg Config) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DB.DSN)
	if err != nil {
		return nil, err
	}

	poolConfig.MinConns = cfg.DB.MinConns
	poolConfig.MaxConns = cfg.DB.MaxConns
	poolConfig.MaxConnIdleTime = cfg.DB.MaxIdleTime

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}
