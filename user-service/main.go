package main

// @title User Service API
// @version 1.0
// @description API Server for Concert Ticketing User Service
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:8081
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer <your_token>" to authenticate.

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
			db.SeedUsers(database)
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
