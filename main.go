package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func dbConn(dbName string) (db *sql.DB) {
	dbDriver := "mysql"
	dbUser := os.Getenv("MYSQL_USERNAME")
	dbPass := os.Getenv("MYSQL_PASSWORD")
	MYSQL_HOSTNAME := os.Getenv("MYSQL_HOSTNAME")
	db, err := sql.Open(dbDriver, dbUser+":"+dbPass+"@tcp"+"("+MYSQL_HOSTNAME+")"+"/"+dbName)
	if err != nil {
		panic(err.Error())
	}
	return db
}

func sendSeDates(pageordate int) {
	db := dbConn("sediff")
	if pageordate < 8 {
		var rows1 = pageordate * 20
		var rows2 = 0
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

		var arr []string
		for rows.Next() {
			var date int
			var amount int
			rows.Scan(&date, &amount)
			mapD := map[string]int{"date": date, "amount": amount}
			mapB, _ := json.Marshal(mapD)
			//fmt.Println(string(mapB))
			arr = append(arr, string(mapB))
		}
		fmt.Print(arr)
	} else {

	}
}

func main() {

	sendSeDates(1)

}
