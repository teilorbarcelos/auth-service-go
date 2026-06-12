package main

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

func main() {
	redisUrl := os.Getenv("REDIS_URL")
	if redisUrl == "" {
		redisUrl = "redis://localhost:6379/0"
	}

	opts, _ := redis.ParseURL(redisUrl)
	client := redis.NewClient(opts)

	ctx := context.Background()
	keys, _, _ := client.Scan(ctx, 0, "session:*", 1000).Result()

	fmt.Println("Keys in Redis:")
	for _, k := range keys {
		fmt.Println(k)
	}
}
