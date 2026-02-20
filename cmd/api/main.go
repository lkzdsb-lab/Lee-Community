package main

import (
	"Lee_Community/internal/model"
	"Lee_Community/internal/repository/mysql"
	"Lee_Community/internal/repository/redis"
	"Lee_Community/internal/router"
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
	)

	// Gin
	r := router.InitRouter()
	err := r.Run(":8080")
	if err != nil {
		return
	}
}
