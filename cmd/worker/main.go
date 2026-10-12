package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/sharasha07/clash-bot/internal/clash"
	"github.com/sharasha07/clash-bot/internal/data"
	"github.com/sharasha07/clash-bot/internal/llm"
)

type Config struct {
	DB struct {
		DSN         string        `env:"DB_DSN,required"`
		MinConns    int32         `env:"DB_MIN_CONNS,required"`
		MaxConns    int32         `env:"DB_MAX_CONNS,required"`
		MaxIdleTime time.Duration `env:"DB_MAX_IDLE_TIME,required"`
	}
	Redis struct {
		URL string `env:"REDIS_URL,required"`
	}
	CR struct {
		BaseURL        string        `env:"CLASH_ROYALE_BASE_URL,required"`
		APIToken       string        `env:"CLASH_ROYALE_API_TOKEN,required"`
		Timeout        time.Duration `env:"CLASH_ROYALE_TIMEOUT,required"`
		MaxResultBytes int           `env:"CLASH_ROYALE_MAX_RESULT_BYTES,required"`
	}
	Gemini struct {
		BaseURL string        `env:"GEMINI_BASE_URL,required"`
		ApiKey  string        `env:"GEMINI_API_KEY,required"`
		Model   string        `env:"GEMINI_MODEL,required"`
		Timeout time.Duration `env:"GEMINI_TIMEOUT,required"`
	}
	RabbitMQ struct {
		URL string `env:"RABBITMQ_URL,required"`
	}
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func main() {
	var cfg Config
	err := env.Parse(&cfg)
	failOnError(err, "failed to parse environment variables")

	pool, err := connectToDB(cfg)
	failOnError(err, "failed to connect to db")

	models := data.NewModels(pool)

	opt, err := redis.ParseURL(cfg.Redis.URL)
	failOnError(err, "couldn't parse redis url")
	redisClient := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = redisClient.Ping(ctx).Err()
	failOnError(err, "failed to connect to Redis")

	crClient := clash.NewAPIClient(cfg.CR.BaseURL, cfg.CR.APIToken, cfg.CR.Timeout, redisClient, slog.Default(), cfg.CR.MaxResultBytes)
	geminiClient, err := llm.NewGemini(cfg.Gemini.ApiKey, cfg.Gemini.BaseURL, cfg.Gemini.Model, cfg.Gemini.Timeout, crClient)
	failOnError(err, "failed to create Gemini client")

	conn, err := amqp.Dial(cfg.RabbitMQ.URL)
	failOnError(err, "failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "failed to open a channel")
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
	failOnError(err, "failed to declare a queue")

	msgs, err := ch.Consume(
		"generate_reply",
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "failed to register a consumer")

	go func() {
		for d := range msgs {
			var body struct {
				User    data.User
				Message data.Message
			}

			err := json.Unmarshal(d.Body, &body)
			if err != nil {
				log.Printf("failed to unmarshal body: %v", err)
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			history, err := models.Messages.GetAll(ctx, body.Message.ChatID, data.Filters{
				Page:         1,
				PageSize:     20,
				Sort:         "-id",
				SortSafeList: []string{"id", "-id", "created_at", "-created_at"},
			})
			cancel()

			if err != nil {
				log.Printf("failed to fetch recent messages: %v", err)
				continue
			}

			slices.Reverse(history)

			// build prompt with latest messages
			var b strings.Builder

			for _, m := range history {
				switch m.Role {
				case data.RoleUser:
					fmt.Fprintf(&b, "User: %s\n", m.Content)
				case data.RoleAssistant:
					fmt.Fprintf(&b, "Assistant: %s\n", m.Content)
				}
			}

			prompt := strings.TrimSpace(b.String())

			// generate LLM reply
			var assistantMessage *data.Message

			ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
			reply, err := geminiClient.GenerateReply(ctx, prompt, body.User.GameTag)
			cancel()

			if err != nil {
				log.Printf("failed to generate reply: %v", err)
				continue
			} else {
				assistantMessage = &data.Message{
					ChatID:  body.Message.ChatID,
					Role:    data.RoleAssistant,
					Content: reply,
				}

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				err := models.Messages.Insert(ctx, assistantMessage)
				cancel()

				if err != nil {
					log.Printf("failed to store reply: %v", err)
					continue
				}
			}
		}
	}()

	log.Println("waiting for messages. To exit press CTRL+C")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("shutting down gracefully")
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
