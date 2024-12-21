package main

import (
	"context"
	"time"

	"github.com/AlekseyPromet/algo/link_shortner/internal"
)

func main() {
	redisCfg, err := internal.LoadRedisConfig("/config_redis.json")
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	redisClient, err := internal.NewRedisConnection(ctx, redisCfg)
	if err != nil {
		panic(err)
	}

	serviceCfg, err := internal.LoadServiceConfig("/config_service.json")
	if err != nil {
		panic(err)
	}

	service := internal.NewServiceShortnessLink(ctx, serviceCfg, redisClient)

	if err := service.Run(); err != nil {
		panic(err)
	}
}
