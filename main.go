package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

/* ---------------- MODEL ---------------- */

type Sale struct {
	SaleID        int       `json:"saleId"`
	CustomerName  string    `json:"customerName"`
	ProductName   string    `json:"productName"`
	Quantity      int       `json:"quantity"`
	Price         float64   `json:"price"`
	PaymentMethod string    `json:"paymentMethod"`
	CreatedDate   time.Time `json:"createdDate"`
}

var db *sql.DB

/* ---------------- MAIN ---------------- */

func main() {

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	ensureTables()

	log.Println("✅ Database connected")

	http.HandleFunc("/sales", getSales)
	http.HandleFunc("/sales/create", createSale)
	http.HandleFunc("/sales/delete", deleteSale)
	http.HandleFunc("/health", health)

	// Static folder
	http.Handle("/", http.FileServer(http.Dir("./static")))

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}

	log.Println("🚀 Server running on port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

/* ---------------- TABLE CREATION ---------------- */

func ensureTables() {

	query := `
	CREATE TABLE IF NOT EXISTS sales (
		sale_id SERIAL PRIMARY KEY,
		customer_name TEXT NOT NULL,
		product_name TEXT NOT NULL,
		quantity INT NOT NULL,
		price NUMERIC(10,2) NOT NULL,
		payment_method TEXT DEFAULT 'CASH',
		created_date TIMESTAMP DEFAULT (NOW() AT TIME ZONE 'Asia/Kolkata')
	);
	`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Failed creating table:", err)
	}

	log.Println("✅ sales table ready")
}

/* ---------------- GET SALES ---------------- */

func getSales(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, `
		SELECT sale_id,
		       customer_name,
		       product_name,
		       quantity,
		       price,
		       payment_method,
		       created_date
		FROM sales
		ORDER BY created_date DESC
	`)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var sales []Sale

	for rows.Next() {
		var s Sale
		err := rows.Scan(
			&s.SaleID,
			&s.CustomerName,
			&s.ProductName,
			&s.Quantity,
			&s.Price,
			&s.PaymentMethod,
			&s.CreatedDate,
		)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		sales = append(sales, s)
	}

	json.NewEncoder(w).Encode(sales)
}

/* ---------------- CREATE SALE ---------------- */

func createSale(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var sale Sale
	err := json.NewDecoder(r.Body).Decode(&sale)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = db.ExecContext(ctx, `
		INSERT INTO sales (
			customer_name,
			product_name,
			quantity,
			price,
			payment_method,
			created_date
		)
		VALUES ($1, $2, $3, $4, $5, NOW() AT TIME ZONE 'Asia/Kolkata')
	`,
		sale.CustomerName,
		sale.ProductName,
		sale.Quantity,
		sale.Price,
		sale.PaymentMethod,
	)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"message": "Sale added",
	})
}

/* ---------------- DELETE SALE ---------------- */

func deleteSale(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var req struct {
		SaleID int `json:"saleId"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	_, err = db.Exec("DELETE FROM sales WHERE sale_id=$1", req.SaleID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"message": "Sale deleted successfully",
	})
}

/* ---------------- HEALTH ---------------- */

func health(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	w.Write([]byte("OK"))
}

/* ---------------- CORS ---------------- */

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
}
