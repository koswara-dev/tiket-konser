package main

import (
	"log"

	"user-service/config"
	"user-service/db"
	"user-service/handler"
	"user-service/model"
	"user-service/redis"
	"user-service/repository"
	"user-service/routes"
	"user-service/service"
)

func main() {
	cfg := config.LoadConfig()

	database := db.InitDB(cfg)
	if database != nil {
		log.Println("Running database migrations...")
		err := database.AutoMigrate(&model.User{}, &model.OTP{})
		if err != nil {
			log.Printf("Failed to run database migrations: %v", err)
		} else {
			log.Println("Database migration completed successfully")
		}
	}

	redis.InitRedis(cfg)

	userRepo := repository.NewUserRepository(database)
	userSvc := service.NewUserService(cfg, userRepo)
	userHandler := handler.NewUserHandler(userSvc, cfg)

	router := routes.SetupRouter(userHandler)
	log.Printf("Starting user-service on port :%s in %s mode", cfg.Port, cfg.AppEnv)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to run user-service server: %v", err)
	}
}
