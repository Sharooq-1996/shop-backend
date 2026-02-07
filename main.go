package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
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

	// ✅ Read & sanitize DB connection string
	dbURL := strings.TrimSpace(os.Getenv("DB_CONN"))
	if dbURL == "" {
		log.Fatal("DB_CONN environment variable not set")
	}

	// ✅ DO NOT Ping (pooler-safe)
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("DB open error:", err)
	}

	log.Println("✅ Database connection initialized (lazy)")

	// Routes
	http.HandleFunc("/health", healthCheck)
	http.HandleFunc("/sales", getSales)
	http.HandleFunc("/sales/create", createSale)

	// Port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("🚀 Server running on port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// ---------- HEALTH CHECK ----------
func healthCheck(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// ---------- GET SALES ----------
func getSales(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	w.Header().Set("Content-Type", "application/json")

	log.Println("➡️ /sales called")

	// ⏱️ IMPORTANT: timeout for pooler
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, `
		SELECT sale_id, customer_name, product_name, quantity, price, created_date
		FROM public.sales
		ORDER BY created_date DESC
		LIMIT 100
	`)
	if err != nil {
		log.Println("❌ DB QUERY ERROR:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}
	defer rows.Close()

	sales := make([]Sale, 0)

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
			log.Println("❌ ROW SCAN ERROR:", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": err.Error(),
			})
			return
		}
		sales = append(sales, s)
	}

	if err := rows.Err(); err != nil {
		log.Println("❌ ROWS ERROR:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	log.Println("✅ Returning", len(sales), "rows")
	json.NewEncoder(w).Encode(sales)
}



// ---------- CREATE SALE ----------
func createSale(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)

	if r.Method == http.MethodOptions {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var sale Sale
	if err := json.NewDecoder(r.Body).Decode(&sale); err != nil {
		http.Error(w, err.Error(), 400)
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
		http.Error(w, err.Error(), 500)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"message": "Sale added successfully",
	})
}

// ---------- CORS ----------
func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
}
