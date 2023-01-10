package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/julienschmidt/httprouter"
)

func dbConn(dbName string, dbUser string, dbPass string) (db *sql.DB) {
	dbDriver := "mysql"
	//dbUser := dbName //os.Getenv("MYSQL_USERNAME")
	//dbPass := dbName //os.Getenv("MYSQL_PASSWORD")
	MYSQL_HOSTNAME := os.Getenv("MYSQL_HOSTNAME")
	db, err := sql.Open(dbDriver, dbUser+":"+dbPass+"@tcp"+"("+MYSQL_HOSTNAME+")"+"/"+dbName)
	if err != nil {
		panic(err.Error())
	}
	return db
}

func sendDates(diffdb string, dbUser string, dbPass string, pageordate int) []byte {
	db := dbConn(diffdb, dbUser, dbPass)
	/* if pageordate < 8 { */

	var rows1 = pageordate * 20
	var rows2 int
	if pageordate == 0 {
		rows2 = 0
	} else {
		rows2 = rows1
	}
	fmt.Println(pageordate)
	fmt.Println(rows2)
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
	//fmt.Print(string(j))
	return j
	/* } */ /* else {
		rows, err := db.Query("SELECT date, amount FROM dates ORDER BY date DESC")
		if err != nil {
			panic(err.Error())
		}
		lol := fmt.Println(rows)
		return lol
	} */
	//return "asd"
}

func httpSendDates(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id, err := strconv.Atoi(ps.ByName("id"))
	if err != nil {
		panic(err.Error())
	}
	result := sendDates(os.Getenv("MYSQL_SE_DATABASE"), os.Getenv("MYSQL_SE_USERNAME"), os.Getenv("MYSQL_SE_PASSWORD"), id)
	w.Write(result)
}

func main() {
	router := httprouter.New()

	router.GET("/se/:id", httpSendDates)

	//fmt.Println(sendDates(os.Getenv("MYSQL_SE_DATABASE"), os.Getenv("MYSQL_SE_USERNAME"), os.Getenv("MYSQL_SE_PASSWORD"), 0))
	//fmt.Println(sendDates(os.Getenv("MYSQL_NU_DATABASE"), os.Getenv("MYSQL_NU_USERNAME"), os.Getenv("MYSQL_NU_PASSWORD"), 1))

	http.ListenAndServe(":8080", router)
}
