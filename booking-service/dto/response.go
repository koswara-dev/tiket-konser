package dto

type WebResponse[T any] struct {
	Code    int    `json:"code"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    T      `json:"data,omitempty"`
	Errors  any    `json:"errors,omitempty"`
}

type PagingResponse struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalRows  int64 `json:"total_rows"`
	TotalPages int   `json:"total_pages"`
}

type WebResponseWithPaging[T any] struct {
	Code    int            `json:"code"`
	Status  string         `json:"status"`
	Message string         `json:"message"`
	Data    T              `json:"data"`
	Paging  PagingResponse `json:"paging"`
}

type ValidationError struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

type TicketCategoryResponse struct {
	ID             uint    `json:"id"`
	Name           string  `json:"name"`
	Price          float64 `json:"price"`
	TotalSeats     int     `json:"total_seats"`
	AvailableSeats int     `json:"available_seats"`
}

type ConcertResponse struct {
	ID               uint                     `json:"id"`
	Title            string                   `json:"title"`
	Artist           string                   `json:"artist"`
	Description      string                   `json:"description"`
	Date             string                   `json:"date"`
	Location         string                   `json:"location"`
	TicketCategories []TicketCategoryResponse `json:"ticket_categories"`
}

type BookingItemResponse struct {
	ID               uint                   `json:"id"`
	TicketCategoryID uint                   `json:"ticket_category_id"`
	TicketCategory   TicketCategoryResponse `json:"ticket_category,omitempty"`
	Quantity         int                    `json:"quantity"`
	SubTotal         float64                `json:"sub_total"`
}

type BookingResponse struct {
	ID           uint                  `json:"id"`
	UserID       uint                  `json:"user_id"`
	BookingDate  string                `json:"booking_date"`
	TotalAmount  float64               `json:"total_amount"`
	Status       string                `json:"status"`
	BookingItems []BookingItemResponse `json:"booking_items"`
}
