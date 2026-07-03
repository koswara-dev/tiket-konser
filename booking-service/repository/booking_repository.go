package repository

import (
	"errors"
	"fmt"
	"time"

	"booking-service/model"

	"gorm.io/gorm"
)

type OrderItem struct {
	TicketCategoryID uint
	Quantity         int
}

type BookingRepository interface {
	GetConcerts() ([]model.Concert, error)
	CreateConcert(concert *model.Concert) error
	CreateBookingWithStockCheck(userID uint, items []OrderItem) (*model.Booking, error)
	ConfirmPayment(bookingID uint) (*model.Booking, error)
}

type bookingRepository struct {
	db *gorm.DB
}

func NewBookingRepository(db *gorm.DB) BookingRepository {
	return &bookingRepository{db: db}
}

func (r *bookingRepository) GetConcerts() ([]model.Concert, error) {
	if r.db == nil {
		return nil, errors.New("database connection is unavailable")
	}
	var concerts []model.Concert
	err := r.db.Preload("TicketCategories").Find(&concerts).Error
	return concerts, err
}

func (r *bookingRepository) CreateConcert(concert *model.Concert) error {
	if r.db == nil {
		return errors.New("database connection is unavailable")
	}
	return r.db.Create(concert).Error
}

func (r *bookingRepository) CreateBookingWithStockCheck(userID uint, items []OrderItem) (*model.Booking, error) {
	if r.db == nil {
		return nil, errors.New("database connection is unavailable")
	}

	var booking model.Booking

	err := r.db.Transaction(func(tx *gorm.DB) error {
		var totalAmount float64
		var bookingItems []model.BookingItem

		for _, item := range items {
			var category model.TicketCategory
			if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&category, item.TicketCategoryID).Error; err != nil {
				return fmt.Errorf("ticket category %d not found", item.TicketCategoryID)
			}

			if category.AvailableSeats < item.Quantity {
				return fmt.Errorf("insufficient seats for ticket category: %s (requested: %d, available: %d)",
					category.Name, item.Quantity, category.AvailableSeats)
			}

			category.AvailableSeats -= item.Quantity
			if err := tx.Save(&category).Error; err != nil {
				return err
			}

			subTotal := category.Price * float64(item.Quantity)
			totalAmount += subTotal

			bookingItems = append(bookingItems, model.BookingItem{
				TicketCategoryID: item.TicketCategoryID,
				Quantity:         item.Quantity,
				SubTotal:         subTotal,
			})
		}

		booking = model.Booking{
			UserID:       userID,
			BookingDate:  time.Now(),
			TotalAmount:  totalAmount,
			Status:       "pending",
			BookingItems: bookingItems,
		}

		if err := tx.Create(&booking).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &booking, nil
}

func (r *bookingRepository) ConfirmPayment(bookingID uint) (*model.Booking, error) {
	if r.db == nil {
		return nil, errors.New("database connection is unavailable")
	}

	var booking model.Booking
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Preload("BookingItems.TicketCategory").First(&booking, bookingID).Error; err != nil {
			return errors.New("booking order not found")
		}

		if booking.Status != "pending" {
			return fmt.Errorf("booking is already in %s status", booking.Status)
		}

		booking.Status = "confirmed"
		return tx.Save(&booking).Error
	})

	if err != nil {
		return nil, err
	}

	return &booking, nil
}
