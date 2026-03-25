package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

type Budget struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Bill struct {
	ID            int     `json:"id"`
	BudgetID      int     `json:"budget_id"`
	Name          string  `json:"name"`
	Account       string  `json:"account"`
	Amount        float64 `json:"amount"`
	Frequency     string  `json:"frequency"`
	DueDate       int     `json:"dueDate"`
	PayDate       int     `json:"payDate"`
	TermNumber    int     `json:"termNumber"`
	TermUnit      string  `json:"termUnit"`
	TermEnd       string  `json:"termEnd"`
	Comments      string  `json:"comments"`
	InterestRate  float64 `json:"interestRate"`
	BillType      string  `json:"billType"`
	Website       string  `json:"website"`
	AutoPay       bool    `json:"autoPay"`
	PaymentMethod string  `json:"paymentMethod"`
	IsPlanned     bool    `json:"isPlanned"`
	StartDate     string  `json:"startDate"`
}

type Income struct {
	ID        int     `json:"id"`
	BudgetID  int     `json:"budget_id"`
	Name      string  `json:"name"`
	Account   string  `json:"account"`
	Amount    float64 `json:"amount"`
	Frequency string  `json:"frequency"`
	Day       int     `json:"day"`
	Comments  string  `json:"comments"`
}

func initDB() {
	var err error
	
	// Create data directory if it doesn't exist (for Docker volumes)
	dbPath := "./budget.db"
	if _, err := os.Stat("./data"); err == nil {
		dbPath = "./data/budget.db"
	}
	
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}

	// Create tables
	schema := `
	CREATE TABLE IF NOT EXISTS budgets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL
	);

	CREATE TABLE IF NOT EXISTS bills (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		budget_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		account TEXT,
		amount REAL NOT NULL,
		frequency TEXT NOT NULL,
		due_date INTEGER,
		pay_date INTEGER,
		term_number INTEGER,
		term_unit TEXT,
		term_end TEXT,
		comments TEXT,
		interest_rate REAL DEFAULT 0,
		bill_type TEXT DEFAULT 'other',
		website TEXT,
		auto_pay INTEGER DEFAULT 0,
		payment_method TEXT DEFAULT 'other',
		is_planned INTEGER DEFAULT 0,
		start_date TEXT,
		FOREIGN KEY (budget_id) REFERENCES budgets(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS incomes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		budget_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		account TEXT,
		amount REAL NOT NULL,
		frequency TEXT NOT NULL,
		day INTEGER,
		comments TEXT,
		FOREIGN KEY (budget_id) REFERENCES budgets(id) ON DELETE CASCADE
	);
	`

	_, err = db.Exec(schema)
	if err != nil {
		log.Fatal(err)
	}

	// Migrate existing tables - add new columns if they don't exist
	migrations := []string{
		"ALTER TABLE bills ADD COLUMN interest_rate REAL DEFAULT 0",
		"ALTER TABLE bills ADD COLUMN bill_type TEXT DEFAULT 'other'",
		"ALTER TABLE bills ADD COLUMN website TEXT",
		"ALTER TABLE bills ADD COLUMN auto_pay INTEGER DEFAULT 0",
		"ALTER TABLE bills ADD COLUMN payment_method TEXT DEFAULT 'other'",
		"ALTER TABLE bills ADD COLUMN is_planned INTEGER DEFAULT 0",
		"ALTER TABLE bills ADD COLUMN start_date TEXT",
	}

	for _, migration := range migrations {
		db.Exec(migration) // Ignore errors - column may already exist
	}
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func listBudgets(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	rows, err := db.Query("SELECT id, name FROM budgets ORDER BY name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	budgets := []Budget{}
	for rows.Next() {
		var b Budget
		rows.Scan(&b.ID, &b.Name)
		budgets = append(budgets, b)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"budgets": budgets})
}

func createBudget(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	var data map[string]string
	json.NewDecoder(r.Body).Decode(&data)

	result, err := db.Exec("INSERT INTO budgets (name) VALUES (?)", data["name"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "id": id})
}

func getBudget(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	budgetID := r.URL.Query().Get("id")

	// Get bills
	rows, _ := db.Query(`
		SELECT id, budget_id, name, account, amount, frequency, due_date, pay_date, 
		       term_number, term_unit, term_end, comments, interest_rate, bill_type, 
		       website, auto_pay, payment_method, is_planned, start_date 
		FROM bills WHERE budget_id = ?`, budgetID)
	defer rows.Close()

	bills := []Bill{}
	for rows.Next() {
		var b Bill
		rows.Scan(&b.ID, &b.BudgetID, &b.Name, &b.Account, &b.Amount, &b.Frequency, 
			&b.DueDate, &b.PayDate, &b.TermNumber, &b.TermUnit, &b.TermEnd, &b.Comments,
			&b.InterestRate, &b.BillType, &b.Website, &b.AutoPay, &b.PaymentMethod,
			&b.IsPlanned, &b.StartDate)
		bills = append(bills, b)
	}

	// Get incomes
	rows2, _ := db.Query("SELECT id, budget_id, name, account, amount, frequency, day, comments FROM incomes WHERE budget_id = ?", budgetID)
	defer rows2.Close()

	incomes := []Income{}
	for rows2.Next() {
		var i Income
		rows2.Scan(&i.ID, &i.BudgetID, &i.Name, &i.Account, &i.Amount, &i.Frequency, &i.Day, &i.Comments)
		incomes = append(incomes, i)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"bills":   bills,
		"incomes": incomes,
	})
}

func saveBill(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	var data struct {
		BudgetID int  `json:"budget_id"`
		Bill     Bill `json:"bill"`
	}
	json.NewDecoder(r.Body).Decode(&data)

	if data.Bill.ID > 0 {
		// Update
		_, err := db.Exec(`UPDATE bills SET name=?, account=?, amount=?, frequency=?, due_date=?, 
			pay_date=?, term_number=?, term_unit=?, term_end=?, comments=?, interest_rate=?, 
			bill_type=?, website=?, auto_pay=?, payment_method=?, is_planned=?, start_date=? 
			WHERE id=? AND budget_id=?`,
			data.Bill.Name, data.Bill.Account, data.Bill.Amount, data.Bill.Frequency,
			data.Bill.DueDate, data.Bill.PayDate, data.Bill.TermNumber, data.Bill.TermUnit,
			data.Bill.TermEnd, data.Bill.Comments, data.Bill.InterestRate, data.Bill.BillType,
			data.Bill.Website, data.Bill.AutoPay, data.Bill.PaymentMethod, data.Bill.IsPlanned,
			data.Bill.StartDate, data.Bill.ID, data.BudgetID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// Insert
		_, err := db.Exec(`INSERT INTO bills (budget_id, name, account, amount, frequency, due_date, 
			pay_date, term_number, term_unit, term_end, comments, interest_rate, bill_type, 
			website, auto_pay, payment_method, is_planned, start_date) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			data.BudgetID, data.Bill.Name, data.Bill.Account, data.Bill.Amount, data.Bill.Frequency,
			data.Bill.DueDate, data.Bill.PayDate, data.Bill.TermNumber, data.Bill.TermUnit,
			data.Bill.TermEnd, data.Bill.Comments, data.Bill.InterestRate, data.Bill.BillType,
			data.Bill.Website, data.Bill.AutoPay, data.Bill.PaymentMethod, data.Bill.IsPlanned,
			data.Bill.StartDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func deleteBill(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	var data map[string]int
	json.NewDecoder(r.Body).Decode(&data)

	_, err := db.Exec("DELETE FROM bills WHERE id=? AND budget_id=?", data["id"], data["budget_id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func saveIncome(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	var data struct {
		BudgetID int    `json:"budget_id"`
		Income   Income `json:"income"`
	}
	json.NewDecoder(r.Body).Decode(&data)

	if data.Income.ID > 0 {
		// Update
		_, err := db.Exec(`UPDATE incomes SET name=?, account=?, amount=?, frequency=?, day=?, comments=? WHERE id=? AND budget_id=?`,
			data.Income.Name, data.Income.Account, data.Income.Amount, data.Income.Frequency,
			data.Income.Day, data.Income.Comments, data.Income.ID, data.BudgetID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// Insert
		_, err := db.Exec(`INSERT INTO incomes (budget_id, name, account, amount, frequency, day, comments) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			data.BudgetID, data.Income.Name, data.Income.Account, data.Income.Amount,
			data.Income.Frequency, data.Income.Day, data.Income.Comments)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func deleteIncome(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	var data map[string]int
	json.NewDecoder(r.Body).Decode(&data)

	_, err := db.Exec("DELETE FROM incomes WHERE id=? AND budget_id=?", data["id"], data["budget_id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func deleteBudget(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	var data map[string]int
	json.NewDecoder(r.Body).Decode(&data)

	_, err := db.Exec("DELETE FROM budgets WHERE id=?", data["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func escapeCSV(s string) string {
	// If string contains comma, quote, or newline, wrap in quotes and escape quotes
	if len(s) == 0 {
		return s
	}
	needsQuotes := false
	for _, c := range s {
		if c == ',' || c == '"' || c == '\n' || c == '\r' {
			needsQuotes = true
			break
		}
	}
	if needsQuotes {
		// Replace " with ""
		escaped := ""
		for _, c := range s {
			if c == '"' {
				escaped += "\"\""
			} else {
				escaped += string(c)
			}
		}
		return "\"" + escaped + "\""
	}
	return s
}

func exportData(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	budgetID := r.URL.Query().Get("id")
	format := r.URL.Query().Get("format") // csv, json

	// Get bills
	billRows, _ := db.Query(`
		SELECT id, budget_id, name, account, amount, frequency, due_date, pay_date, 
		       term_number, term_unit, term_end, comments, interest_rate, bill_type, 
		       website, auto_pay, payment_method, is_planned, start_date 
		FROM bills WHERE budget_id = ?`, budgetID)
	defer billRows.Close()

	bills := []Bill{}
	for billRows.Next() {
		var b Bill
		billRows.Scan(&b.ID, &b.BudgetID, &b.Name, &b.Account, &b.Amount, &b.Frequency,
			&b.DueDate, &b.PayDate, &b.TermNumber, &b.TermUnit, &b.TermEnd, &b.Comments,
			&b.InterestRate, &b.BillType, &b.Website, &b.AutoPay, &b.PaymentMethod,
			&b.IsPlanned, &b.StartDate)
		bills = append(bills, b)
	}

	// Get incomes
	incomeRows, _ := db.Query("SELECT id, budget_id, name, account, amount, frequency, day, comments FROM incomes WHERE budget_id = ?", budgetID)
	defer incomeRows.Close()

	incomes := []Income{}
	for incomeRows.Next() {
		var i Income
		incomeRows.Scan(&i.ID, &i.BudgetID, &i.Name, &i.Account, &i.Amount, &i.Frequency, &i.Day, &i.Comments)
		incomes = append(incomes, i)
	}

	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=budget-export.csv")
		
		// Write bills
		w.Write([]byte("TYPE,NAME,ACCOUNT,AMOUNT,FREQUENCY,DUE_DATE,PAY_DATE,TERM_NUMBER,TERM_UNIT,TERM_END,INTEREST_RATE,BILL_TYPE,WEBSITE,AUTO_PAY,PAYMENT_METHOD,COMMENTS\n"))
		for _, b := range bills {
			autoPay := "false"
			if b.AutoPay {
				autoPay = "true"
			}
			// Escape commas in text fields
			name := escapeCSV(b.Name)
			account := escapeCSV(b.Account)
			website := escapeCSV(b.Website)
			comments := escapeCSV(b.Comments)
			
			line := fmt.Sprintf("BILL,%s,%s,%.2f,%s,%d,%d,%d,%s,%s,%.4f,%s,%s,%s,%s,%s\n",
				name, account, b.Amount, b.Frequency, b.DueDate, b.PayDate,
				b.TermNumber, b.TermUnit, b.TermEnd, b.InterestRate, b.BillType,
				website, autoPay, b.PaymentMethod, comments)
			w.Write([]byte(line))
		}
		
		// Write incomes
		w.Write([]byte("\nTYPE,NAME,ACCOUNT,AMOUNT,FREQUENCY,DAY,COMMENTS\n"))
		for _, i := range incomes {
			name := escapeCSV(i.Name)
			account := escapeCSV(i.Account)
			comments := escapeCSV(i.Comments)
			
			line := fmt.Sprintf("INCOME,%s,%s,%.2f,%s,%d,%s\n",
				name, account, i.Amount, i.Frequency, i.Day, comments)
			w.Write([]byte(line))
		}
	} else {
		// JSON format
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=budget-export.json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"bills":   bills,
			"incomes": incomes,
		})
	}
}

func main() {
	initDB()
	defer db.Close()

	// Serve static files
	fs := http.FileServer(http.Dir("."))
	http.Handle("/", fs)

	// API routes
	http.HandleFunc("/api/budgets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" || r.Method == "OPTIONS" {
			listBudgets(w, r)
		} else if r.Method == "POST" {
			createBudget(w, r)
		}
	})
	http.HandleFunc("/api/budget", getBudget)
	http.HandleFunc("/api/bill", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			saveBill(w, r)
		} else if r.Method == "DELETE" {
			deleteBill(w, r)
		}
	})
	http.HandleFunc("/api/income", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			saveIncome(w, r)
		} else if r.Method == "DELETE" {
			deleteIncome(w, r)
		}
	})
	http.HandleFunc("/api/budget/delete", deleteBudget)
	http.HandleFunc("/api/export", exportData)

	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
