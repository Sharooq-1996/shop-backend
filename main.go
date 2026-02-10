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

/* ---------- MODEL ---------- */

type Sale struct {
    SaleID        int       `json:"saleId"`
    CustomerName  string    `json:"customerName"`
    ProductName   string    `json:"productName"`
    Quantity      int       `json:"quantity"`
    Price         float64   `json:"price"`
    PaymentMethod string    `json:"paymentMethod"` // CASH or UPI
    CreatedDate   time.Time `json:"createdDate"`
}


var db *sql.DB

/* ---------- MAIN ---------- */

func main() {
	var err error

	// ✅ Read DATABASE_URL from Render
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("❌ DATABASE_URL environment variable not set")
	}

	// ✅ Open DB connection
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("❌ DB open error:", err)
	}

	// ✅ Verify DB connectivity
	if err = db.Ping(); err != nil {
		log.Fatal("❌ DB ping failed:", err)
	}

	// ✅ Safe pool settings for Render
	db.SetMaxOpenConns(3)
	db.SetMaxIdleConns(0)
	db.SetConnMaxLifetime(2 * time.Minute)

	// ✅ Ensure tables exist
	ensureTables()

	log.Println("✅ Database connected & tables ready")

	// ---------- ROUTES ----------
	http.HandleFunc("/health", health)
	http.HandleFunc("/sales", getSales)
	http.HandleFunc("/sales/create", createSale)

	// ---------- STATIC FILES ----------
	fileServer := http.FileServer(http.Dir("./static"))
	http.Handle("/", fileServer)

	// ---------- PORT ----------
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
		payment_method TEXT DEFAULT 'CASH',
		created_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := db.Exec(query); err != nil {
		log.Fatal("❌ Failed to create sales table:", err)
	}

	// ✅ Add column safely if DB already existed
	_, err := db.Exec(`
		ALTER TABLE sales
		ADD COLUMN IF NOT EXISTS payment_method TEXT DEFAULT 'CASH';
	`)
	if err != nil {
		log.Fatal("❌ Failed to add payment_method column:", err)
	}

	log.Println("✅ sales table ready with payment_method")
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

	SELECT sale_id, customer_name, product_name, quantity, price, payment_method, created_date
FROM sales

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
    &s.PaymentMethod,
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
	INSERT INTO sales (customer_name, product_name, quantity, price, payment_method)
	VALUES ($1, $2, $3, $4, $5)
`,
	sale.CustomerName,
	sale.ProductName,
	sale.Quantity,
	sale.Price,
	sale.PaymentMethod,
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
