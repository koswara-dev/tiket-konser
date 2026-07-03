package main

// @title Booking Service API
// @version 1.0
// @description API Server for Concert Ticketing Booking Service
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:8082
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer <your_token>" to authenticate.

import (
	"log"

	"booking-service/config"
	"booking-service/db"
	"booking-service/handler"
	"booking-service/model"
	"booking-service/redis"
	"booking-service/repository"
	"booking-service/routes"
	"booking-service/service"
)

func main() {
	cfg := config.LoadConfig()

	database := db.InitDB(cfg)
	if database != nil {
		log.Println("Running database migrations...")
		err := database.AutoMigrate(&model.Concert{}, &model.TicketCategory{}, &model.Booking{}, &model.BookingItem{})
		if err != nil {
			log.Printf("Failed to run database migrations: %v", err)
		} else {
			log.Println("Database migration completed successfully")
		}
	}

	redis.InitRedis(cfg)

	bookingRepo := repository.NewBookingRepository(database)
	bookingSvc := service.NewBookingService(bookingRepo)
	bookingHandler := handler.NewBookingHandler(bookingSvc, cfg)

	router := routes.SetupRouter(bookingHandler)
	log.Printf("Starting booking-service on port :%s in %s mode", cfg.Port, cfg.AppEnv)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to run booking-service server: %v", err)
	}
}
