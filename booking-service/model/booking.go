package model

import (
	"time"

	"gorm.io/gorm"
)

type Concert struct {
	gorm.Model
	Title            string           `json:"title" gorm:"type:varchar(150);not null"`
	Artist           string           `json:"artist" gorm:"type:varchar(100);not null"`
	Description      string           `json:"description" gorm:"type:text"`
	Date             time.Time        `json:"date"`
	Location         string           `json:"location" gorm:"type:varchar(150);not null"`
	TicketCategories []TicketCategory `json:"ticket_categories" gorm:"foreignKey:ConcertID"`
}

type TicketCategory struct {
	gorm.Model
	ConcertID      uint    `json:"concert_id"`
	Name           string  `json:"name" gorm:"type:varchar(50);not null"` // VIP, Regular, etc.
	Price          float64 `json:"price" gorm:"type:decimal(12,2);not null"`
	TotalSeats     int     `json:"total_seats" gorm:"not null"`
	AvailableSeats int     `json:"available_seats" gorm:"not null"`
}

type Booking struct {
	gorm.Model
	UserID       uint          `json:"user_id" gorm:"not null;index"`
	BookingDate  time.Time     `json:"booking_date"`
	TotalAmount  float64       `json:"total_amount" gorm:"type:decimal(12,2)"`
	Status       string        `json:"status" gorm:"type:varchar(50);default:'pending'"` // pending, confirmed, cancelled
	BookingItems []BookingItem `json:"booking_items" gorm:"foreignKey:BookingID"`
}

type BookingItem struct {
	gorm.Model
	BookingID        uint           `json:"booking_id"`
	TicketCategoryID uint           `json:"ticket_category_id" gorm:"not null"`
	TicketCategory   TicketCategory `json:"ticket_category" gorm:"foreignKey:TicketCategoryID"`
	Quantity         int            `json:"quantity" gorm:"not null"`
	SubTotal         float64        `json:"sub_total" gorm:"type:decimal(12,2)"`
}
