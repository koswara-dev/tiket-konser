package dto

type TicketCategoryInput struct {
	Name  string  `json:"name" binding:"required"`
	Price float64 `json:"price" binding:"required,gt=0"`
	Seats int     `json:"seats" binding:"required,min=1"`
}

type CreateConcertRequest struct {
	Title            string                `json:"title" binding:"required"`
	Artist           string                `json:"artist" binding:"required"`
	Description      string                `json:"description"`
	Location         string                `json:"location" binding:"required"`
	Date             string                `json:"date" binding:"required"` // Should be RFC3339 format
	TicketCategories []TicketCategoryInput `json:"ticket_categories" binding:"required,dive"`
}

type OrderItemInput struct {
	TicketCategoryID uint `json:"ticket_category_id" binding:"required"`
	Quantity         int  `json:"quantity" binding:"required,min=1"`
}

type CreateBookingRequest struct {
	Items []OrderItemInput `json:"items" binding:"required,dive"`
}
