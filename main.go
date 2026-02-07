package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
)

/*
ENV VARIABLES REQUIRED (Render / Cloud):
--------------------------------------
PORT     -> auto provided by Render
DB_CONN  -> your database connection string
*/

// ---------- DATA MODEL ----------
type Sale struct {
	SaleID       int       `json:"saleId"`
	CustomerName string    `json:"customerName"`
	ProductName  string    `json:"productName"`
	Quantity     int       `json:"quantity"`
	Price        float64   `json:"price"`
	CreatedDate  time.Time `json:"createdDate"`
}

var db *sql.DB

// ---------- MAIN ----------
func main() {
	var err error

	// 🔹 Read DB connection from ENV (cloud-safe)
	connString := os.Getenv("DB_CONN")
	if connString == "" {
		log.Fatal("❌ DB_CONN environment variable not set")
	}

	db, err = sql.Open("sqlserver", connString)
	if err != nil {
		log.Fatal("DB connection error:", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("DB ping failed:", err)
	}

	log.Println("✅ Database connected")

	// Routes
	http.HandleFunc("/sales", withCORS(getSales))
	http.HandleFunc("/sales/create", withCORS(createSale))

	// 🔹 PORT required by Render
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // local fallback
	}

	log.Println("🚀 Server running on port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// ---------- CORS MIDDLEWARE ----------
func withCORS(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler(w, r)
	}
}

// ---------- GET SALES ----------
func getSales(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	rows, err := db.Query(`
		SELECT SaleId, CustomerName, ProductName, Quantity, Price, CreatedDate
		FROM Sales
		ORDER BY CreatedDate DESC
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

// ---------- CREATE SALE ----------
func createSale(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var sale Sale
	if err := json.NewDecoder(r.Body).Decode(&sale); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	_, err := db.Exec(`
		INSERT INTO Sales (CustomerName, ProductName, Quantity, Price)
		VALUES (@p1, @p2, @p3, @p4)
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
		"message": "Sale inserted successfully",
	})
}
