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
	SaleID        int       `json:"saleId"`
	CustomerName  string    `json:"customerName"`
	ProductName   string    `json:"productName"`
	CellName      string    `json:"cellName"`
	Warranty      string    `json:"warranty"`
	Quantity      int       `json:"quantity"`
	Price         float64   `json:"price"`
	PaymentMethod string    `json:"paymentMethod"`
	CreatedDate   time.Time `json:"createdDate"`
}

var db *sql.DB

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

	http.HandleFunc("/sales", getSales)
	http.HandleFunc("/sales/create", createSale)
	http.HandleFunc("/sales/delete", deleteSale)
	http.HandleFunc("/sales/reset", resetSales)

	http.Handle("/", http.FileServer(http.Dir("./static")))

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}

	log.Println("Server running on port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func ensureTables() {

	query := `
	CREATE TABLE IF NOT EXISTS sales (
		sale_id SERIAL PRIMARY KEY,
		customer_name TEXT,
		product_name TEXT,
		cell_name TEXT,
		warranty TEXT,
		quantity INT,
		price NUMERIC(10,2),
		payment_method TEXT,
		created_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
}

func getSales(w http.ResponseWriter, r *http.Request) {

	rows, err := db.Query(`
		SELECT sale_id, customer_name, product_name,
		       cell_name, warranty, quantity,
		       price, payment_method, created_date
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
		rows.Scan(
			&s.SaleID,
			&s.CustomerName,
			&s.ProductName,
			&s.CellName,
			&s.Warranty,
			&s.Quantity,
			&s.Price,
			&s.PaymentMethod,
			&s.CreatedDate,
		)
		sales = append(sales, s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sales)
}

func createSale(w http.ResponseWriter, r *http.Request) {

	var sale Sale
	json.NewDecoder(r.Body).Decode(&sale)

	_, err := db.Exec(`
		INSERT INTO sales (
			customer_name, product_name, cell_name,
			warranty, quantity, price, payment_method
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`,
		sale.CustomerName,
		sale.ProductName,
		sale.CellName,
		sale.Warranty,
		sale.Quantity,
		sale.Price,
		sale.PaymentMethod,
	)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"message": "Sale Added",
	})
}

func deleteSale(w http.ResponseWriter, r *http.Request) {

	id := r.URL.Query().Get("id")

	_, err := db.Exec("DELETE FROM sales WHERE sale_id=$1", id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"message": "Deleted",
	})
}

func resetSales(w http.ResponseWriter, r *http.Request) {

	_, err := db.Exec("TRUNCATE TABLE sales RESTART IDENTITY;")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"message": "All Sales Reset",
	})
}
