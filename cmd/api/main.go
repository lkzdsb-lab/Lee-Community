package main

import (
	"Lee_Community/internal/model"
	"Lee_Community/internal/pkg"
	"Lee_Community/internal/repository/mysql"
	"Lee_Community/internal/repository/redis"
	"Lee_Community/internal/router"
	"Lee_Community/internal/service"
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	dsn := "user:password@tcp(127.0.0.1:3306)/community?charset=utf8mb4&parseTime=True"
	if err := mysql.InitDB(dsn); err != nil {
		panic(err)
	}

	// 连接redis
	if err := redis.Init("127.0.0.1:6379", "203423", 0); err != nil {
		panic(err)
	}

	// 自动建表（开发阶段 OK）
	mysql.DB.AutoMigrate(
		&model.User{},
		&model.Community{},
		&model.CommunityMember{},
		&model.Post{},
		&model.Follow{},
		&model.SocialOutbox{},
	)

	// Gin
	r := router.InitRouter()

	// 启动 OutboxRelayer and FollowCountReconciler
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// 初始化 Kafka Producer（从环境或配置加载）
	brokers := os.Getenv("KAFKA_BROKERS") // 例: "127.0.0.1:9092,127.0.0.1:9093"
	topic := os.Getenv("KAFKA_SOCIAL_TOPIC")
	if topic == "" {
		topic = "social.events"
	}

	prod, err := pkg.NewKafkaProducer(pkg.KafkaConfig{
		Brokers: strings.Split(brokers, ","),
		Topic:   topic,
	})
	if err != nil || brokers == "" {
		log.Printf("Kafka disabled or init failed (brokers='%s', err=%v), fallback to log sender", brokers, err)
	}

	relayer := service.NewOutboxRelayer(func(c context.Context, ob *model.SocialOutbox) error { return nil })
	// 更正：实际应将 model.SocialOutbox 传入 sender，这里直接绑定
	if prod != nil && brokers != "" && err == nil {
		relayer = service.NewOutboxRelayer(service.KafkaSender(prod))
	} else {
		relayer = service.NewOutboxRelayer(service.LogSender)
	}
	go relayer.Run(ctx)

	// 启动对账
	reconciler := service.NewFollowCountReconciler()
	go reconciler.ReconcilerRun(ctx)

	err = r.Run(":8080")
	if err != nil {
		return
	}
}
