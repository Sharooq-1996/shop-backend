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
	Cell          string    `json:"cell"`
	Warranty      string    `json:"warranty"`
	Quantity      int       `json:"quantity"`
	Price         float64   `json:"price"`
	PaymentMethod string    `json:"paymentMethod"`
	CreatedDate   string    `json:"createdDate"`
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

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	createTable()

	http.HandleFunc("/sales", getSales)
	http.HandleFunc("/sales/create", createSale)
	http.HandleFunc("/sales/delete/", deleteSale)

	http.Handle("/", http.FileServer(http.Dir("./static")))

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}

	log.Println("Server running on", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func createTable() {

	query := `
	CREATE TABLE IF NOT EXISTS sales (
		sale_id SERIAL PRIMARY KEY,
		customer_name TEXT,
		product_name TEXT,
		cell TEXT,
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
		SELECT sale_id, customer_name, product_name, cell,
		       warranty, quantity, price,
		       payment_method, created_date
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
		var created time.Time

		err := rows.Scan(
			&s.SaleID,
			&s.CustomerName,
			&s.ProductName,
			&s.Cell,
			&s.Warranty,
			&s.Quantity,
			&s.Price,
			&s.PaymentMethod,
			&created,
		)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		s.CreatedDate = created.Format(time.RFC3339)
		sales = append(sales, s)
	}

	json.NewEncoder(w).Encode(sales)
}

func createSale(w http.ResponseWriter, r *http.Request) {

	var sale Sale
	json.NewDecoder(r.Body).Decode(&sale)

	query := `
	INSERT INTO sales (
		customer_name,
		product_name,
		cell,
		warranty,
		quantity,
		price,
		payment_method,
		created_date
	)
	VALUES ($1,$2,$3,$4,$5,$6,$7,
		CASE 
			WHEN $8 = '' THEN CURRENT_TIMESTAMP
			ELSE $8::timestamp
		END
	)
	`

	_, err := db.Exec(
		query,
		sale.CustomerName,
		sale.ProductName,
		sale.Cell,
		sale.Warranty,
		sale.Quantity,
		sale.Price,
		sale.PaymentMethod,
		sale.CreatedDate,
	)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Write([]byte(`{"status":"ok"}`))
}

func deleteSale(w http.ResponseWriter, r *http.Request) {

	id := r.URL.Path[len("/sales/delete/"):]

	_, err := db.Exec("DELETE FROM sales WHERE sale_id=$1", id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Write([]byte(`{"deleted":"ok"}`))
}
