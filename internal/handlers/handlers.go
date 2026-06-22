package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/armin/expenses/backend/internal/database"
	"github.com/armin/expenses/backend/internal/models"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	db *database.DB
}

func New(db *database.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) ListPersons(c *gin.Context) {
	rows, err := h.db.Query(`SELECT id, name FROM persons ORDER BY id`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list persons"})
		return
	}
	defer rows.Close()

	var persons []models.Person
	for rows.Next() {
		var p models.Person
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read persons"})
			return
		}
		persons = append(persons, p)
	}

	c.JSON(http.StatusOK, persons)
}

func (h *Handler) ListShops(c *gin.Context) {
	rows, err := h.db.Query(`SELECT id, name FROM shops ORDER BY name`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list shops"})
		return
	}
	defer rows.Close()

	var shops []models.Shop
	for rows.Next() {
		var s models.Shop
		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read shops"})
			return
		}
		shops = append(shops, s)
	}

	c.JSON(http.StatusOK, shops)
}

func (h *Handler) CreateShop(c *gin.Context) {
	var req models.CreateShopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	result, err := h.db.Exec(`INSERT INTO shops (name) VALUES (?)`, name)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			c.JSON(http.StatusConflict, gin.H{"error": "shop already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create shop"})
		return
	}

	id, _ := result.LastInsertId()
	c.JSON(http.StatusCreated, models.Shop{ID: int(id), Name: name})
}

func (h *Handler) DeleteShop(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid shop id"})
		return
	}

	var inUse int
	if err := h.db.QueryRow(`SELECT COUNT(*) FROM invoices WHERE shop_id = ?`, id).Scan(&inUse); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check shop usage"})
		return
	}
	if inUse > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "shop is used by invoices"})
		return
	}

	result, err := h.db.Exec(`DELETE FROM shops WHERE id = ?`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete shop"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "shop not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) ListInvoices(c *gin.Context) {
	query := `
		SELECT i.id, i.person_id, p.name, i.shop_id, s.name, i.date, i.total
		FROM invoices i
		JOIN persons p ON p.id = i.person_id
		JOIN shops s ON s.id = i.shop_id
		WHERE 1=1
	`
	args := []any{}

	if personID := c.Query("person_id"); personID != "" {
		query += ` AND i.person_id = ?`
		args = append(args, personID)
	}
	if from := c.Query("from_date"); from != "" {
		query += ` AND i.date >= ?`
		args = append(args, from)
	}
	if to := c.Query("to_date"); to != "" {
		query += ` AND i.date <= ?`
		args = append(args, to)
	}

	query += ` ORDER BY i.date DESC, i.id DESC`

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list invoices"})
		return
	}
	defer rows.Close()

	var invoices []models.Invoice
	var invoiceIDs []int
	for rows.Next() {
		var inv models.Invoice
		if err := rows.Scan(&inv.ID, &inv.PersonID, &inv.PersonName, &inv.ShopID, &inv.ShopName, &inv.Date, &inv.Total); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read invoices"})
			return
		}
		inv.Items = []models.InvoiceItem{}
		invoices = append(invoices, inv)
		invoiceIDs = append(invoiceIDs, inv.ID)
	}

	if len(invoiceIDs) > 0 {
		itemsByInvoice, err := h.loadItemsForInvoices(invoiceIDs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load invoice items"})
			return
		}
		for i := range invoices {
			if items, ok := itemsByInvoice[invoices[i].ID]; ok {
				invoices[i].Items = items
			}
		}
	}

	c.JSON(http.StatusOK, invoices)
}

func (h *Handler) loadItemsForInvoices(invoiceIDs []int) (map[int][]models.InvoiceItem, error) {
	placeholders := make([]string, len(invoiceIDs))
	args := make([]any, len(invoiceIDs))
	for i, id := range invoiceIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := `SELECT id, invoice_id, description, amount, quantity
		FROM invoice_items WHERE invoice_id IN (` + strings.Join(placeholders, ",") + `)
		ORDER BY id`

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int][]models.InvoiceItem)
	for rows.Next() {
		var item models.InvoiceItem
		if err := rows.Scan(&item.ID, &item.InvoiceID, &item.Description, &item.Amount, &item.Quantity); err != nil {
			return nil, err
		}
		result[item.InvoiceID] = append(result[item.InvoiceID], item)
	}

	return result, nil
}

func (h *Handler) GetInvoice(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invoice id"})
		return
	}

	inv, err := h.fetchInvoice(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get invoice"})
		return
	}

	c.JSON(http.StatusOK, inv)
}

func (h *Handler) fetchInvoice(id int) (models.Invoice, error) {
	var inv models.Invoice
	err := h.db.QueryRow(`
		SELECT i.id, i.person_id, p.name, i.shop_id, s.name, i.date, i.total
		FROM invoices i
		JOIN persons p ON p.id = i.person_id
		JOIN shops s ON s.id = i.shop_id
		WHERE i.id = ?`, id,
	).Scan(&inv.ID, &inv.PersonID, &inv.PersonName, &inv.ShopID, &inv.ShopName, &inv.Date, &inv.Total)
	if err != nil {
		return inv, err
	}

	itemsByInvoice, err := h.loadItemsForInvoices([]int{id})
	if err != nil {
		return inv, err
	}
	inv.Items = itemsByInvoice[id]
	if inv.Items == nil {
		inv.Items = []models.InvoiceItem{}
	}

	return inv, nil
}

func (h *Handler) CreateInvoice(c *gin.Context) {
	var req models.CreateInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.PersonID != 1 && req.PersonID != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "person_id must be 1 or 2"})
		return
	}

	var shopExists int
	if err := h.db.QueryRow(`SELECT COUNT(*) FROM shops WHERE id = ?`, req.ShopID).Scan(&shopExists); err != nil || shopExists == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid shop_id"})
		return
	}

	var total float64
	for _, item := range req.Items {
		qty := item.Quantity
		if qty <= 0 {
			qty = 1
		}
		total += item.Amount * qty
	}

	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer tx.Rollback()

	result, err := tx.Exec(
		`INSERT INTO invoices (person_id, shop_id, date, total) VALUES (?, ?, ?, ?)`,
		req.PersonID, req.ShopID, req.Date, total,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create invoice"})
		return
	}

	invoiceID, _ := result.LastInsertId()

	for _, item := range req.Items {
		qty := item.Quantity
		if qty <= 0 {
			qty = 1
		}
		_, err := tx.Exec(
			`INSERT INTO invoice_items (invoice_id, description, amount, quantity) VALUES (?, ?, ?, ?)`,
			invoiceID, strings.TrimSpace(item.Description), item.Amount, qty,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create invoice items"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save invoice"})
		return
	}

	inv, err := h.fetchInvoice(int(invoiceID))
	if err != nil {
		c.JSON(http.StatusCreated, gin.H{"id": invoiceID})
		return
	}

	c.JSON(http.StatusCreated, inv)
}

func (h *Handler) DeleteInvoice(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invoice id"})
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM invoice_items WHERE invoice_id = ?`, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete invoice items"})
		return
	}

	result, err := tx.Exec(`DELETE FROM invoices WHERE id = ?`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete invoice"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit delete"})
		return
	}

	c.Status(http.StatusNoContent)
}
