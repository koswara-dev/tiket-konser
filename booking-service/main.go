package main

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
