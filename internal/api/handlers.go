package api

import (
	"database/sql"
	"encoding/json"
	"go-axfr-backend/internal/models"
	"go-axfr-backend/pkg/health"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
)

func dbConn(dbName string, dbUser string, dbPass string) (db *sql.DB, err error) {
	dbDriver := "mysql"
	MYSQL_HOSTNAME := os.Getenv("MYSQL_HOSTNAME")
	db, err = sql.Open(dbDriver, dbUser+":"+dbPass+"@tcp"+"("+MYSQL_HOSTNAME+")"+"/"+dbName)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return db, nil
}

func sendRows(diffdb string, dbUser string, dbPass string, date int, page int) []byte {
	db, _ := dbConn(diffdb, dbUser, dbPass)
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

func sendDates(diffdb string, dbUser string, dbPass string, pageordate int) []byte {
	db, _ := dbConn(diffdb, dbUser, dbPass)

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

func searchDomain(dumpdb string, dbUser string, dbPass string, query string) []byte {
	db, _ := dbConn(dumpdb, dbUser, dbPass)
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

func domainAmounts(dumpdb, dbUser, dbPass string) []byte {
	type DateAmount struct {
		Date   string `json:"date"`
		Amount int    `json:"amount"`
	}

	db, _ := dbConn(dumpdb, dbUser, dbPass)
	defer db.Close()

	rows, err := db.Query("SELECT date, amount FROM dates")
	if err != nil {
		log.Panic(err.Error())
	}
	defer rows.Close()

	var results []DateAmount

	for rows.Next() {
		var da DateAmount
		err := rows.Scan(&da.Date, &da.Amount)
		if err != nil {
			log.Panic(err.Error())
		}

		parsedDate, err := time.Parse("20060102", da.Date)
		if err != nil {
			log.Panic(err.Error())
		}
		da.Date = parsedDate.Format("2006-01-02")
		results = append(results, da)
	}

	if err = rows.Err(); err != nil {
		log.Panic(err.Error())
	}

	jsonData, err := json.Marshal(results)
	if err != nil {
		log.Panic(err.Error())
	}

	return jsonData
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

func domainStats(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	tld := ps.ByName("tld")
	switch tld {
	case "se":
		result := domainAmounts(os.Getenv("MYSQL_SEDUMP_DATABASE"), os.Getenv("MYSQL_SEDUMP_USERNAME"), os.Getenv("MYSQL_SEDUMP_PASSWORD"))
		w.Write(result)
	case "nu":
		result := domainAmounts(os.Getenv("MYSQL_NUDUMP_DATABASE"), os.Getenv("MYSQL_NUDUMP_USERNAME"), os.Getenv("MYSQL_NUDUMP_PASSWORD"))
		w.Write(result)
	case "ch":
		result := domainAmounts(os.Getenv("MYSQL_CHDUMP_DATABASE"), os.Getenv("MYSQL_CHDUMP_USERNAME"), os.Getenv("MYSQL_CHDUMP_PASSWORD"))
		w.Write(result)
	case "li":
		result := domainAmounts(os.Getenv("MYSQL_LIDUMP_DATABASE"), os.Getenv("MYSQL_LIDUMP_USERNAME"), os.Getenv("MYSQL_LIDUMP_PASSWORD"))
		w.Write(result)
	case "ee":
		result := domainAmounts(os.Getenv("MYSQL_EEDUMP_DATABASE"), os.Getenv("MYSQL_EEDUMP_USERNAME"), os.Getenv("MYSQL_EEDUMP_PASSWORD"))
		w.Write(result)
	case "sk":
		result := domainAmounts(os.Getenv("MYSQL_SKDUMP_DATABASE"), os.Getenv("MYSQL_SKDUMP_USERNAME"), os.Getenv("MYSQL_SKDUMP_PASSWORD"))
		w.Write(result)
	}
}

func readyness(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	dbs := []models.DbConfig{
		{Database: os.Getenv("MYSQL_NU_DATABASE"), Username: os.Getenv("MYSQL_NU_USERNAME"), Password: os.Getenv("MYSQL_NU_PASSWORD"), Name: "NU"},
		{Database: os.Getenv("MYSQL_SE_DATABASE"), Username: os.Getenv("MYSQL_SE_USERNAME"), Password: os.Getenv("MYSQL_SE_PASSWORD"), Name: "SE"},
		{Database: os.Getenv("MYSQL_SEDUMP_DATABASE"), Username: os.Getenv("MYSQL_SEDUMP_USERNAME"), Password: os.Getenv("MYSQL_SEDUMP_PASSWORD"), Name: "SE dump"},
		{Database: os.Getenv("MYSQL_NUDUMP_DATABASE"), Username: os.Getenv("MYSQL_NUDUMP_USERNAME"), Password: os.Getenv("MYSQL_NUDUMP_PASSWORD"), Name: "NU dump"},
		{Database: os.Getenv("MYSQL_CHDUMP_DATABASE"), Username: os.Getenv("MYSQL_CHDUMP_USERNAME"), Password: os.Getenv("MYSQL_CHDUMP_PASSWORD"), Name: "CH dump"},
		{Database: os.Getenv("MYSQL_LIDUMP_DATABASE"), Username: os.Getenv("MYSQL_LIDUMP_USERNAME"), Password: os.Getenv("MYSQL_LIDUMP_PASSWORD"), Name: "LI dump"},
		{Database: os.Getenv("MYSQL_EEDUMP_DATABASE"), Username: os.Getenv("MYSQL_EEDUMP_USERNAME"), Password: os.Getenv("MYSQL_EEDUMP_PASSWORD"), Name: "EE dump"},
		{Database: os.Getenv("MYSQL_SKDUMP_DATABASE"), Username: os.Getenv("MYSQL_SKDUMP_USERNAME"), Password: os.Getenv("MYSQL_SKDUMP_PASSWORD"), Name: "SK dump"},
	}

	if err := health.CheckDatabases(dbs); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("All database connections successful"))
}

func liveness(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	MYSQL_HOSTNAME := os.Getenv("MYSQL_HOSTNAME")
	timeout := 5 * time.Second

	conn, err := net.DialTimeout("tcp", MYSQL_HOSTNAME+":3306", timeout)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("MySQL server is not reachable"))
		return
	}
	defer conn.Close()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("MySQL server is reachable"))
}
