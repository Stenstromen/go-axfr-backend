package main

import (
	"database/sql"
	"encoding/json"
	"log"
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

func sendRows(diffdb string, dbUser string, dbPass string, date int, page int) []byte {
	db := dbConn(diffdb, dbUser, dbPass)
	var rows2 = page * 20
	rows, err := db.Query("SELECT domain FROM domains JOIN dates ON domains.dategrp = dates.id WHERE date = ? ORDER BY domain ASC OFFSET ? ROWS FETCH FIRST 20 ROWS ONLY", date, rows2)
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()

	type Rows struct {
		Domain string `json:"domain"`
	}
	var arr []Rows
	for rows.Next() {
		var domain string
		rows.Scan(&domain)
		a := Rows{Domain: domain}
		arr = append(arr, a)
	}
	j, _ := json.Marshal(arr)
	return j
}

func searchDomain(dumpdb string, dbUser string, dbPass string, query string) []byte {
	db := dbConn(dumpdb, dbUser, dbPass)
	rows, err := db.Query("SELECT domain FROM domains WHERE domain LIKE ? ORDER BY CHAR_LENGTH(domain) ASC", "%"+query+"%")
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()

	type Rows struct {
		Domain string `json:"domain"`
	}
	var arr []Rows
	for rows.Next() {
		var domain string
		rows.Scan(&domain)
		a := Rows{Domain: domain}
		arr = append(arr, a)
	}
	j, _ := json.Marshal(arr)
	return j
}

func middleware(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.Header().Set("access-control-allow-headers", "Accept,content-type,Access-Control-Allow-Origin,access-control-allow-headers, access-control-allow-methods, Authorization")
		w.Header().Set("content-type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", os.Getenv("CORS_HEADER"))
		w.Header().Set("access-control-allow-methods", "GET, OPTIONS")
		authHeader := r.Header.Get("Authorization")
		if authHeader != os.Getenv("AUTHHEADER_PASSWORD") {
			resp := make(map[string]string)
			resp["error"] = "Invalid or no credentials"
			jsonResp, err := json.Marshal(resp)
			if err != nil {
				log.Fatalf("Error happened in JSON marshal. Err: %s", err)
			}
			w.WriteHeader(http.StatusForbidden)
			w.Write(jsonResp)
		} else {
			next(w, r, ps)
		}
	}
}

func sendSEDates(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	page, err := strconv.Atoi(ps.ByName("page"))
	if err != nil {
		panic(err.Error())
	}
	result := sendDates(os.Getenv("MYSQL_SE_DATABASE"), os.Getenv("MYSQL_SE_USERNAME"), os.Getenv("MYSQL_SE_PASSWORD"), page)
	w.Write(result)
}

func sendNUDates(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	page, err := strconv.Atoi(ps.ByName("page"))
	if err != nil {
		panic(err.Error())
	}
	result := sendDates(os.Getenv("MYSQL_NU_DATABASE"), os.Getenv("MYSQL_NU_USERNAME"), os.Getenv("MYSQL_NU_PASSWORD"), page)
	w.Write(result)
}

func sendSERows(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	date, err := strconv.Atoi(ps.ByName("date"))
	if err != nil {
		panic(err.Error())
	}
	page, err := strconv.Atoi(ps.ByName("page"))
	if err != nil {
		panic(err.Error())
	}

	result := sendRows(os.Getenv("MYSQL_SE_DATABASE"), os.Getenv("MYSQL_SE_USERNAME"), os.Getenv("MYSQL_SE_PASSWORD"), date, page)
	w.Write(result)
}

func sendNURows(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	date, err := strconv.Atoi(ps.ByName("date"))
	if err != nil {
		panic(err.Error())
	}
	page, err := strconv.Atoi(ps.ByName("page"))
	if err != nil {
		panic(err.Error())
	}

	result := sendRows(os.Getenv("MYSQL_NU_DATABASE"), os.Getenv("MYSQL_NU_USERNAME"), os.Getenv("MYSQL_NU_PASSWORD"), date, page)
	w.Write(result)
}

func domainSearch(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	tld := ps.ByName("tld")
	query := ps.ByName("query")
	switch tld {
	case "se":
		result := searchDomain(os.Getenv("MYSQL_SEDUMP_DATABASE"), os.Getenv("MYSQL_SEDUMP_USERNAME"), os.Getenv("MYSQL_SEDUMP_PASSWORD"), query)
		w.Write(result)
	case "nu":
		result := searchDomain(os.Getenv("MYSQL_NUDUMP_DATABASE"), os.Getenv("MYSQL_NUDUMP_USERNAME"), os.Getenv("MYSQL_NUDUMP_PASSWORD"), query)
		w.Write(result)
	case "ch":
		result := searchDomain(os.Getenv("MYSQL_CHDUMP_DATABASE"), os.Getenv("MYSQL_CHDUMP_USERNAME"), os.Getenv("MYSQL_CHDUMP_PASSWORD"), query)
		w.Write(result)
	case "li":
		result := searchDomain(os.Getenv("MYSQL_LIDUMP_DATABASE"), os.Getenv("MYSQL_LIDUMP_USERNAME"), os.Getenv("MYSQL_LIDUMP_PASSWORD"), query)
		w.Write(result)
	case "ee":
		result := searchDomain(os.Getenv("MYSQL_EEDUMP_DATABASE"), os.Getenv("MYSQL_EEDUMP_USERNAME"), os.Getenv("MYSQL_EEDUMP_PASSWORD"), query)
		w.Write(result)
	case "sk":
		result := searchDomain(os.Getenv("MYSQL_SKDUMP_DATABASE"), os.Getenv("MYSQL_SKDUMP_USERNAME"), os.Getenv("MYSQL_SKDUMP_PASSWORD"), query)
		w.Write(result)
	}
}

func main() {
	router := httprouter.New()

	router.GlobalOPTIONS = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("Access-Control-Allow-Methods", header.Get("Allow"))
		header.Set("Access-Control-Allow-Origin", os.Getenv("CORS_HEADER"))
		header.Set("access-control-allow-headers", "Accept,content-type,Access-Control-Allow-Origin,access-control-allow-headers, access-control-allow-methods, Authorization")
		w.WriteHeader(http.StatusNoContent)
	})

	router.GET("/se/:page", middleware(sendSEDates))
	router.GET("/nu/:page", middleware(sendNUDates))
	router.GET("/sedomains/:date/:page", middleware(sendSERows))
	router.GET("/nudomains/:date/:page", middleware(sendNURows))
	router.GET("/search/:tld/:query", middleware(domainSearch))

	http.ListenAndServe(":8080", router)
}
