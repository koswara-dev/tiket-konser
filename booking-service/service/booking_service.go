package service

import (
	"errors"
	"fmt"
	"time"

	"booking-service/model"
	"booking-service/redis"
	"booking-service/repository"
)

type BookingService struct {
	bookingRepo repository.BookingRepository
}

func NewBookingService(bookingRepo repository.BookingRepository) *BookingService {
	return &BookingService{bookingRepo: bookingRepo}
}

func (s *BookingService) GetConcerts() ([]model.Concert, error) {
	return s.bookingRepo.GetConcerts()
}

func (s *BookingService) CreateConcert(title, artist, description, location string, date time.Time, categories []model.TicketCategory) (*model.Concert, error) {
	concert := model.Concert{
		Title:            title,
		Artist:           artist,
		Description:      description,
		Date:             date,
		Location:         location,
		TicketCategories: categories,
	}

	err := s.bookingRepo.CreateConcert(&concert)
	if err != nil {
		return nil, err
	}
	return &concert, nil
}

type OrderItemInput struct {
	TicketCategoryID uint `json:"ticket_category_id" binding:"required"`
	Quantity         int  `json:"quantity" binding:"required,min=1"`
}

func (s *BookingService) CreateBooking(userID uint, items []OrderItemInput) (*model.Booking, error) {
	for _, item := range items {
		lockKey := fmt.Sprintf("category:%d", item.TicketCategoryID)
		acquired := redis.AcquireLock(lockKey, 5*time.Second)
		if !acquired {
			return nil, errors.New("system is busy processing other bookings, please try again")
		}
		defer redis.ReleaseLock(lockKey)
	}

	var repoItems []repository.OrderItem
	for _, item := range items {
		repoItems = append(repoItems, repository.OrderItem{
			TicketCategoryID: item.TicketCategoryID,
			Quantity:         item.Quantity,
		})
	}

	return s.bookingRepo.CreateBookingWithStockCheck(userID, repoItems)
}

func (s *BookingService) ConfirmPayment(bookingID uint) (*model.Booking, error) {
	return s.bookingRepo.ConfirmPayment(bookingID)
}
