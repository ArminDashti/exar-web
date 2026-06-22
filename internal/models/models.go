package models

type Person struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Shop struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type InvoiceItem struct {
	ID          int     `json:"id,omitempty"`
	InvoiceID   int     `json:"invoice_id,omitempty"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Quantity    float64 `json:"quantity"`
}

type Invoice struct {
	ID         int           `json:"id,omitempty"`
	PersonID   int           `json:"person_id"`
	ShopID     int           `json:"shop_id"`
	Date       string        `json:"date"`
	Total      float64       `json:"total"`
	PersonName string        `json:"person_name,omitempty"`
	ShopName   string        `json:"shop_name,omitempty"`
	Items      []InvoiceItem `json:"items"`
}

type CreateInvoiceRequest struct {
	PersonID int    `json:"person_id" binding:"required"`
	ShopID   int    `json:"shop_id" binding:"required"`
	Date     string `json:"date" binding:"required"`
	Items    []struct {
		Description string  `json:"description" binding:"required"`
		Amount      float64 `json:"amount" binding:"required"`
		Quantity    float64 `json:"quantity"`
	} `json:"items" binding:"required,min=1,dive"`
}

type CreateShopRequest struct {
	Name string `json:"name" binding:"required"`
}
