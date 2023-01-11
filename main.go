package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/julienschmidt/httprouter"
)

func dbConn(dbName string, dbUser string, dbPass string) (db *sql.DB) {
	dbDriver := "mysql"
	MYSQL_HOSTNAME := os.Getenv("MYSQL_HOSTNAME")
	db, err := sql.Open(dbDriver, dbUser+":"+dbPass+"@tcp"+"("+MYSQL_HOSTNAME+")"+"/"+dbName)
	if err != nil {
		panic(err.Error())
	}
	return db
}

func sendDates(diffdb string, dbUser string, dbPass string, pageordate int) []byte {
	db := dbConn(diffdb, dbUser, dbPass)

	var rows1 = pageordate * 20
	var rows2 int
	if pageordate == 0 {
		rows2 = 0
	} else {
		rows2 = rows1
	}
	rows, err := db.Query("SELECT date, amount FROM dates ORDER BY date DESC OFFSET ? ROWS FETCH FIRST 20 ROWS ONLY", rows2)
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()

	type Amounts struct {
		Date   int `json:"date"`
		Amount int `json:"amount"`
	}
	var arr []Amounts
	for rows.Next() {
		var date int
		var amount int
		rows.Scan(&date, &amount)
		a := Amounts{Date: date, Amount: amount}
		arr = append(arr, a)
	}
	j, _ := json.Marshal(arr)
	return j
}

func main() {
	router := httprouter.New()

	router.GET("/se/:id", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		id, err := strconv.Atoi(ps.ByName("id"))
		if err != nil {
			panic(err.Error())
		}
		result := sendDates(os.Getenv("MYSQL_SE_DATABASE"), os.Getenv("MYSQL_SE_USERNAME"), os.Getenv("MYSQL_SE_PASSWORD"), id)
		w.Write(result)
	})

	router.GET("/nu/:id", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		id, err := strconv.Atoi(ps.ByName("id"))
		if err != nil {
			panic(err.Error())
		}
		result := sendDates(os.Getenv("MYSQL_NU_DATABASE"), os.Getenv("MYSQL_NU_USERNAME"), os.Getenv("MYSQL_NY_PASSWORD"), id)
		w.Write(result)
	})

	http.ListenAndServe(":8080", router)
}
