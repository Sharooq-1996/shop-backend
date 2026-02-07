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

// Sale model (matches DB exactly)
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

	// DB connection string (keep as-is for now)
	connString := "server=localhost;user id=sa;password=Sharooq@1996;database=ShopDB"

	db, err = sql.Open("sqlserver", connString)
	if err != nil {
		log.Fatal("DB connection error:", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("DB ping failed:", err)
	}

	log.Println("✅ Connected to SQL Server")

	http.HandleFunc("/sales", getSales)
	http.HandleFunc("/sales/create", createSale)

	// ✅ REQUIRED FOR CLOUD (Render / Railway)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // local default
	}

	log.Println("🚀 Go backend running on port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// 🔹 GET all sales
func getSales(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	rows, err := db.Query(`
		SELECT SaleId, CustomerName, ProductName, Quantity, Price, CreatedDate
		FROM sales
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

	json.NewEncoder(w).Encode(sales)
}

// 🔹 INSERT sale
func createSale(w http.ResponseWriter, r *http.Request) {
	// CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")

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
		INSERT INTO sales (CustomerName, ProductName, Quantity, Price)
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
