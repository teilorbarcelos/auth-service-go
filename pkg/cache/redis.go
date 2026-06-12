package cache

import (
	"context"
	"log"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"backend-go/pkg/config"
)

var RedisClient *redis.Client

var (
	logFatalf    = log.Fatalf
	miniredisRun = miniredis.Run
)

func ConnectRedis() {
	if config.AppConfig.Environment == "test" {
		mr, err := miniredisRun()
		if err != nil {
			logFatalf("Falha ao iniciar miniredis: %v", err)
		}
		RedisClient = redis.NewClient(&redis.Options{
			Addr: mr.Addr(),
		})
	} else {
		opts, err := redis.ParseURL(config.AppConfig.RedisUrl)
		if err != nil {
			logFatalf("Falha ao parsear a URL do Redis: %v", err)
		}
		RedisClient = redis.NewClient(opts)
	}

	if err := RedisClient.Ping(context.Background()).Err(); err != nil {
		logFatalf("Falha ao conectar no Redis: %v", err)
	}

	log.Println("Conexão com Redis estabelecida com sucesso.")
}
