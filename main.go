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
	PaymentMethod string    `json:"paymentMethod"`
	CreatedDate   time.Time `json:"createdDate"`
}

var db *sql.DB

/* ---------- MAIN ---------- */

func main() {

	var err error

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("DB open error:", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("DB connection failed:", err)
	}

	db.SetMaxOpenConns(3)
	db.SetMaxIdleConns(0)
	db.SetConnMaxLifetime(2 * time.Minute)

	ensureTables()

	log.Println("✅ Database connected")

	/* ROUTES */
	http.HandleFunc("/health", health)
	http.HandleFunc("/sales", getSales)
	http.HandleFunc("/sales/create", createSale)
	http.HandleFunc("/sales/clear", clearSales)

	/* STATIC */
	http.Handle("/", http.FileServer(http.Dir("./static")))

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}

	log.Println("🚀 Server running on port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

/* ---------- TABLE CREATION ---------- */

func ensureTables() {

	createTable := `
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

	_, err := db.Exec(createTable)
	if err != nil {
		log.Fatal("Table creation failed:", err)
	}

	log.Println("✅ sales table ready")
}

/* ---------- HEALTH ---------- */

func health(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	w.Write([]byte("OK"))
}

/* ---------- GET SALES (with optional date filter) ---------- */

func getSales(w http.ResponseWriter, r *http.Request) {

	enableCORS(w)
	w.Header().Set("Content-Type", "application/json")

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var rows *sql.Rows
	var err error

	if from != "" && to != "" {
		rows, err = db.QueryContext(ctx, `
			SELECT sale_id, customer_name, product_name,
			       quantity, price, payment_method, created_date
			FROM sales
			WHERE DATE(created_date) BETWEEN $1 AND $2
			ORDER BY created_date DESC
		`, from, to)
	} else {
		rows, err = db.QueryContext(ctx, `
			SELECT sale_id, customer_name, product_name,
			       quantity, price, payment_method, created_date
			FROM sales
			ORDER BY created_date DESC
			LIMIT 200
		`)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	err := json.NewDecoder(r.Body).Decode(&sale)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
			payment_method
		)
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

/* ---------- CLEAR ALL SALES ---------- */

func clearSales(w http.ResponseWriter, r *http.Request) {

	enableCORS(w)

	_, err := db.Exec(`TRUNCATE TABLE sales RESTART IDENTITY;`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("All sales deleted & ID reset"))
}

/* ---------- CORS ---------- */

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
}
