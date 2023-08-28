package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"

	"log"
	"net/http"
	"path"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

// AUTH_USERNAME=## AUTH_PASSWORD=## ./budget

type BudgetItem struct {
	ID    int       `json:"id"`
	Price float64   `json:"price"`
	Name  string    `json:"name"`
	Date  time.Time `json:"-"`
}
type BudgetItemWeb struct {
	BudgetItem
	Date string `json:"date"`
}

var (
	db       *gorm.DB
	user     string
	password string
)

func removeItemHandler(w http.ResponseWriter, r *http.Request) {
	item_id := r.URL.Query().Get("id")
	var budgetItem BudgetItem
	db.Where("id = ?", item_id).Find(&budgetItem)
	db.Delete(budgetItem)

	resp := make(map[string]string)
	resp["status"] = "ok"
	json.NewEncoder(w).Encode(resp)
}
func addItemHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	price := r.URL.Query().Get("price")

	priceFloat, _ := strconv.ParseFloat(strings.TrimSpace(price), 64)

	resp := make(map[string]string)

	budgetItem := &BudgetItem{
		Name:  name,
		Price: priceFloat,
		Date:  time.Now(),
	}

	db.Save(&budgetItem)

	resp["status"] = "ok"

	json.NewEncoder(w).Encode(resp)

}
func getItemsHander(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var budgetItems []BudgetItem
	var budgetWebItems []BudgetItemWeb
	db.Order("id DESC").Find(&budgetItems)

	for _, item := range budgetItems {
		budgetWebItem := BudgetItemWeb{
			BudgetItem: item,
			Date:       item.Date.Format("2006-01-02 15:04"),
		}
		budgetWebItems = append(budgetWebItems, budgetWebItem)
	}
	json.NewEncoder(w).Encode(budgetWebItems)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	p := path.Dir("./index.html")
	w.Header().Set("Content-type", "text/html")
	http.ServeFile(w, r, p)
}

func basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			usernameHash := sha256.Sum256([]byte(username))
			passwordHash := sha256.Sum256([]byte(password))
			expectedUsernameHash := sha256.Sum256([]byte(user))
			expectedPasswordHash := sha256.Sum256([]byte(password))

			usernameMatch := (subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1)
			passwordMatch := (subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1)

			if usernameMatch && passwordMatch {
				next.ServeHTTP(w, r)
				return
			}
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

func main() {
	user = os.Getenv("AUTH_USERNAME")
	if user == "" {
		user = "demo"
	}
	password = os.Getenv("AUTH_PASSWORD")
	if password == "" {
		password = "demo"
	}
	if user == "" {
		log.Fatal("basic auth username must be provided")
	}

	if password == "" {
		log.Fatal("basic auth password must be provided")
	}
	var err error
	db, err = gorm.Open("sqlite3", "budget.db")
	if err != nil {
		log.Println(err.Error())
		panic("Failed to connect to database")
	}
	defer db.Close()

	db.AutoMigrate(&BudgetItem{})

	router := mux.NewRouter()

	router.HandleFunc("/", basicAuth(indexHandler))
	router.HandleFunc("/add-item", basicAuth(addItemHandler))
	router.HandleFunc("/remove-item", basicAuth(removeItemHandler))
	router.HandleFunc("/get-items", basicAuth(getItemsHander))

	http.Handle("/", router)
	http.ListenAndServe(":9111", nil)

}
