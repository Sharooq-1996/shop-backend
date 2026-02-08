package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

/* ---------- MODEL ---------- */

type Sale struct {
	SaleID       int       `json:"saleId"`
	CustomerName string    `json:"customerName"`
	ProductName  string    `json:"productName"`
	Quantity     int       `json:"quantity"`
	Price        float64   `json:"price"`
	CreatedDate  time.Time `json:"createdDate"`
}

var db *sql.DB

/* ---------- MAIN ---------- */

func main() {
	var err error

	// 🔹 Read DB connection from Render environment
	dbURL := strings.TrimSpace(os.Getenv("DB_CONN"))
	if dbURL == "" {
		log.Fatal("❌ DB_CONN environment variable not set")
	}

	// 🔹 Open DB (Render PostgreSQL)
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("❌ DB open error:", err)
	}

	// 🔹 Safe pool settings for Render Free tier
	db.SetMaxOpenConns(3)
	db.SetMaxIdleConns(0)
	db.SetConnMaxLifetime(2 * time.Minute)

	// 🔹 Auto-create required tables
	ensureTables()

	log.Println("✅ Database connected & tables ready")

	// 🔹 API routes
	http.HandleFunc("/health", health)
	http.HandleFunc("/sales", getSales)
	http.HandleFunc("/sales/create", createSale)

	// 🔹 Serve frontend from /static folder
	fileServer := http.FileServer(http.Dir("./static"))
	http.Handle("/", fileServer)

	// 🔹 Render provides PORT
	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}

	log.Println("🚀 Server running on port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

/* ---------- AUTO TABLE CREATION ---------- */

func ensureTables() {
	query := `
	CREATE TABLE IF NOT EXISTS sales (
		sale_id SERIAL PRIMARY KEY,
		customer_name TEXT NOT NULL,
		product_name TEXT NOT NULL,
		quantity INT NOT NULL,
		price NUMERIC(10,2) NOT NULL,
		created_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := db.Exec(query); err != nil {
		log.Fatal("❌ Failed to create sales table:", err)
	}
	log.Println("✅ sales table exists")
}

/* ---------- HEALTH CHECK ---------- */

func health(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

/* ---------- GET SALES ---------- */

func getSales(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, `
		SELECT sale_id, customer_name, product_name, quantity, price, created_date
		FROM sales
		ORDER BY created_date DESC
		LIMIT 100
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	sales := []Sale{}

	for rows.Next() {
		var s Sale
		if err := rows.Scan(
			&s.SaleID,
			&s.CustomerName,
			&s.ProductName,
			&s.Quantity,
			&s.Price,
			&s.CreatedDate,
		); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		sales = append(sales, s)
	}

	json.NewEncoder(w).Encode(sales)
}

/* ---------- CREATE SALE ---------- */

func createSale(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)

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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.ExecContext(ctx, `
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

/* ---------- CORS ---------- */

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
}
