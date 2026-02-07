package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

type Sale struct {
	SaleID       int       `json:"saleId"`
	CustomerName string    `json:"customerName"`
	ProductName  string    `json:"productName"`
	Quantity     int       `json:"quantity"`
	Price        float64   `json:"price"`
	CreatedDate  time.Time `json:"createdDate"`
}

var db *sql.DB

func main() {
	var err error

	// ✅ Read DB connection from environment (Render / local)
	connString := os.Getenv("DB_CONN")
	if connString == "" {
		log.Fatal("DB_CONN environment variable not set")
	}

	db, err = sql.Open("postgres", connString)
	if err != nil {
		log.Fatal("DB open error:", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("DB ping failed:", err)
	}

	log.Println("✅ Connected to Supabase PostgreSQL")

	http.HandleFunc("/sales", getSales)
	http.HandleFunc("/sales/create", createSale)

	// ✅ Render provides PORT automatically
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("🚀 Server running on port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// ---------- GET SALES ----------
func getSales(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)

	rows, err := db.Query(`
		SELECT sale_id, customer_name, product_name, quantity, price, created_date
		FROM sales
		ORDER BY created_date DESC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var sales []Sale

	for rows.Next() {
		var s Sale
		rows.Scan(
			&s.SaleID,
			&s.CustomerName,
			&s.ProductName,
			&s.Quantity,
			&s.Price,
			&s.CreatedDate,
		)
		sales = append(sales, s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sales)
}

// ---------- CREATE SALE ----------
func createSale(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)

	if r.Method == http.MethodOptions {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var sale Sale
	if err := json.NewDecoder(r.Body).Decode(&sale); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := db.Exec(`
		INSERT INTO sales (customer_name, product_name, quantity, price)
		VALUES ($1, $2, $3, $4)
	`,
		sale.CustomerName,
		sale.ProductName,
		sale.Quantity,
		sale.Price,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"message": "Sale added successfully",
	})
}

// ---------- CORS ----------
func enableCORS(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type")
	(*w).Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
}
