package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"go-axfr-backend/internal/models"
	"go-axfr-backend/pkg/health"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

/* Optimize queries for se-nu appearance
CREATE INDEX domain_idx ON domains(domain);
CREATE INDEX dategrp_idx ON domains(dategrp);
CREATE INDEX date_idx ON dates(date); */

type TLDConfig struct {
	Database string
	Username string
	Password string
}

var tldConfigs = map[string]TLDConfig{
	"se": {
		Database: "MYSQL_SEDUMP_DATABASE",
		Username: "MYSQL_SEDUMP_USERNAME",
		Password: "MYSQL_SEDUMP_PASSWORD",
	},
	"nu": {
		Database: "MYSQL_NUDUMP_DATABASE",
		Username: "MYSQL_NUDUMP_USERNAME",
		Password: "MYSQL_NUDUMP_PASSWORD",
	},
	"ch": {
		Database: "MYSQL_CHDUMP_DATABASE",
		Username: "MYSQL_CHDUMP_USERNAME",
		Password: "MYSQL_CHDUMP_PASSWORD",
	},
	"li": {
		Database: "MYSQL_LIDUMP_DATABASE",
		Username: "MYSQL_LIDUMP_USERNAME",
		Password: "MYSQL_LIDUMP_PASSWORD",
	},
	"ee": {
		Database: "MYSQL_EEDUMP_DATABASE",
		Username: "MYSQL_EEDUMP_USERNAME",
		Password: "MYSQL_EEDUMP_PASSWORD",
	},
	"sk": {
		Database: "MYSQL_SKDUMP_DATABASE",
		Username: "MYSQL_SKDUMP_USERNAME",
		Password: "MYSQL_SKDUMP_PASSWORD",
	},
	"se_diff": {
		Database: "MYSQL_SE_DATABASE",
		Username: "MYSQL_SE_USERNAME",
		Password: "MYSQL_SE_PASSWORD",
	},
	"nu_diff": {
		Database: "MYSQL_NU_DATABASE",
		Username: "MYSQL_NU_USERNAME",
		Password: "MYSQL_NU_PASSWORD",
	},
}

var (
	redisClient *redis.Client
	ctx         = context.Background()
)

func InitRedis() {
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		redisClient = redis.NewClient(&redis.Options{
			Addr: redisURL,
		})

		_, err := redisClient.Ping(ctx).Result()
		if err != nil {
			log.Printf("Failed to connect to Redis: %v", err)
			redisClient = nil
		} else {
			log.Printf("Successfully connected to Redis at %s", redisURL)
		}
	} else {
		log.Printf("No REDIS_URL provided, running without cache")
	}
}

func getOrSetCache(key string, ttl time.Duration, generator func() []byte) ([]byte, bool, error) {
	if redisClient == nil {
		return generator(), false, nil
	}

	val, err := redisClient.Get(ctx, key).Bytes()
	if err == nil {
		log.Printf("Cache HIT for key: %s", key)
		return val, true, nil
	}
	if err != redis.Nil {
		log.Printf("Redis error for key %s: %v", key, err)
		return nil, false, err
	}

	log.Printf("Cache MISS for key: %s", key)
	data := generator()

	err = redisClient.Set(ctx, key, data, ttl).Err()
	if err != nil {
		log.Printf("Failed to set cache for key %s: %v", key, err)
		return data, false, err
	}

	return data, false, nil
}

func dbConn(dbName string, dbUser string, dbPass string) (db *sql.DB, err error) {
	dbDriver := "mysql"
	MYSQL_HOSTNAME := os.Getenv("MYSQL_HOSTNAME")
	db, err = sql.Open(dbDriver, dbUser+":"+dbPass+"@tcp"+"("+MYSQL_HOSTNAME+")"+"/"+dbName)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(50)
	db.SetConnMaxLifetime(time.Hour)

	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return db, nil
}

func sendRows(diffdb string, dbUser string, dbPass string, date int, page int) []byte {
	db, err := dbConn(diffdb, dbUser, dbPass)
	if err != nil {
		log.Printf("Database connection error: %v", err)
		return []byte(`{"error": "database connection failed"}`)
	}
	defer db.Close()

	var rows2 = page * 20
	rows, err := db.Query("SELECT domain FROM domains JOIN dates ON domains.dategrp = dates.id WHERE date = ? ORDER BY domain ASC LIMIT 20 OFFSET ?", date, rows2)
	if err != nil {
		log.Printf("Query error: %v", err)
		return []byte(`{"error": "query failed"}`)
	}
	defer rows.Close()

	type Rows struct {
		Domain string `json:"domain"`
	}
	var arr []Rows
	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err != nil {
			log.Printf("Row scan error: %v", err)
			continue
		}
		arr = append(arr, Rows{Domain: domain})
	}
	if err := rows.Err(); err != nil {
		log.Printf("Rows error: %v", err)
	}

	j, err := json.Marshal(arr)
	if err != nil {
		log.Printf("JSON marshal error: %v", err)
		return []byte(`{"error": "json marshal failed"}`)
	}
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

func searchDomain(dumpdb, dbUser, dbPass, query string) []byte {
	db, err := dbConn(dumpdb, dbUser, dbPass)
	if err != nil {
		log.Printf("Database connection error: %v", err)
		return []byte(`{"error": "database connection failed"}`)
	}
	defer db.Close()

	rows, err := db.Query("SELECT domain FROM domains WHERE domain LIKE ? ORDER BY CHAR_LENGTH(domain) ASC", "%"+query+"%")
	if err != nil {
		log.Printf("Query error: %v", err)
		return []byte(`{"error": "query failed"}`)
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

const (
	ShortTTL  = 5 * time.Minute
	MediumTTL = 1 * time.Hour
	LongTTL   = 6 * time.Hour
	DayTTL    = 24 * time.Hour
)

func sendSEDates(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) != 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	page, err := strconv.Atoi(pathParts[1])
	if err != nil {
		http.Error(w, "Invalid page number", http.StatusBadRequest)
		return
	}

	cacheKey := fmt.Sprintf("sedates:page:%d", page)

	result, cacheHit, err := getOrSetCache(cacheKey, MediumTTL, func() []byte {
		return sendDates(os.Getenv("MYSQL_SE_DATABASE"), os.Getenv("MYSQL_SE_USERNAME"), os.Getenv("MYSQL_SE_PASSWORD"), page)
	})

	if err != nil {
		log.Printf("Cache error: %v", err)
	}

	if cacheHit {
		w.Header().Set("X-Cache", "HIT")
	} else {
		w.Header().Set("X-Cache", "MISS")
	}

	w.Write(result)
}

func sendNUDates(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) != 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	page, err := strconv.Atoi(pathParts[1])
	if err != nil {
		http.Error(w, "Invalid page number", http.StatusBadRequest)
		return
	}

	cacheKey := fmt.Sprintf("nudates:page:%d", page)

	result, cacheHit, err := getOrSetCache(cacheKey, MediumTTL, func() []byte {
		return sendDates(os.Getenv("MYSQL_NU_DATABASE"), os.Getenv("MYSQL_NU_USERNAME"), os.Getenv("MYSQL_NU_PASSWORD"), page)
	})

	if err != nil {
		log.Printf("Cache error: %v", err)
	}

	if cacheHit {
		w.Header().Set("X-Cache", "HIT")
	} else {
		w.Header().Set("X-Cache", "MISS")
	}

	w.Write(result)
}

func sendSERows(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) != 3 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	date, err := strconv.Atoi(pathParts[1])
	if err != nil {
		http.Error(w, "Invalid date number", http.StatusBadRequest)
		return
	}

	page, err := strconv.Atoi(pathParts[2])
	if err != nil {
		http.Error(w, "Invalid page number", http.StatusBadRequest)
		return
	}

	cacheKey := fmt.Sprintf("serows:date:%d:page:%d", date, page)

	result, cacheHit, err := getOrSetCache(cacheKey, MediumTTL, func() []byte {
		return sendRows(os.Getenv("MYSQL_SE_DATABASE"), os.Getenv("MYSQL_SE_USERNAME"), os.Getenv("MYSQL_SE_PASSWORD"), date, page)
	})

	if err != nil {
		log.Printf("Cache error: %v", err)
	}

	if cacheHit {
		w.Header().Set("X-Cache", "HIT")
	} else {
		w.Header().Set("X-Cache", "MISS")
	}

	w.Write(result)
}

func sendNURows(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) != 3 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	date, err := strconv.Atoi(pathParts[1])
	if err != nil {
		http.Error(w, "Invalid date number", http.StatusBadRequest)
		return
	}

	page, err := strconv.Atoi(pathParts[2])
	if err != nil {
		http.Error(w, "Invalid page number", http.StatusBadRequest)
		return
	}

	cacheKey := fmt.Sprintf("nurows:date:%d:page:%d", date, page)

	result, cacheHit, err := getOrSetCache(cacheKey, MediumTTL, func() []byte {
		return sendRows(os.Getenv("MYSQL_NU_DATABASE"), os.Getenv("MYSQL_NU_USERNAME"), os.Getenv("MYSQL_NU_PASSWORD"), date, page)
	})

	if err != nil {
		log.Printf("Cache error: %v", err)
	}

	if cacheHit {
		w.Header().Set("X-Cache", "HIT")
	} else {
		w.Header().Set("X-Cache", "MISS")
	}

	w.Write(result)
}

func domainSearch(w http.ResponseWriter, r *http.Request) {
	parts, err := getPathParams(r.URL.Path, 3)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tld, query := parts[1], parts[2]
	db, user, pass, err := getTLDEnvVars(tld)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cacheKey := fmt.Sprintf("search:%s:%s", tld, query)

	result, cacheHit, err := getOrSetCache(cacheKey, ShortTTL, func() []byte {
		return searchDomain(db, user, pass, query)
	})

	if err != nil {
		log.Printf("Cache error: %v", err)
	}

	if cacheHit {
		w.Header().Set("X-Cache", "HIT")
	} else {
		w.Header().Set("X-Cache", "MISS")
	}

	w.Write(result)
}

func domainStats(w http.ResponseWriter, r *http.Request) {
	parts, err := getPathParams(r.URL.Path, 2)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tld := parts[1]
	db, user, pass, err := getTLDEnvVars(tld)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cacheKey := fmt.Sprintf("stats:%s", tld)

	result, cacheHit, err := getOrSetCache(cacheKey, LongTTL, func() []byte {
		return domainAmounts(db, user, pass)
	})

	if err != nil {
		log.Printf("Cache error: %v", err)
	}

	if cacheHit {
		w.Header().Set("X-Cache", "HIT")
	} else {
		w.Header().Set("X-Cache", "MISS")
	}

	w.Write(result)
}

func readyness(w http.ResponseWriter, r *http.Request) {
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

func liveness(w http.ResponseWriter, r *http.Request) {
	MYSQL_HOSTNAME := os.Getenv("MYSQL_HOSTNAME")
	dbDriver := "mysql"
	dbUser := os.Getenv("MYSQL_SE_USERNAME")
	dbPass := os.Getenv("MYSQL_SE_PASSWORD")
	dbName := os.Getenv("MYSQL_SE_DATABASE")

	if dbUser == "" || dbPass == "" || dbName == "" {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("MySQL configuration not available"))
		return
	}

	db, err := sql.Open(dbDriver, dbUser+":"+dbPass+"@tcp"+"("+MYSQL_HOSTNAME+")"+"/"+dbName)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("MySQL server connection failed"))
		return
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("MySQL server is not responding"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("MySQL server is healthy"))
}

func getTLDEnvVars(tld string) (string, string, string, error) {
	config, ok := tldConfigs[tld]
	if !ok {
		return "", "", "", fmt.Errorf("unsupported TLD: %s", tld)
	}
	return os.Getenv(config.Database), os.Getenv(config.Username), os.Getenv(config.Password), nil
}

func getPathParams(path string, expectedParts int) ([]string, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != expectedParts {
		return nil, fmt.Errorf("invalid path: expected %d parts, got %d", expectedParts, len(parts))
	}
	return parts, nil
}

func getDomainFirstAppearance(dumpdb, dbUser, dbPass, query string) []byte {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		if duration > time.Second {
			log.Printf("Slow query warning: getDomainFirstAppearance took %v for query: %s", duration, query)
		}
	}()

	db, err := dbConn(dumpdb, dbUser, dbPass)
	if err != nil {
		log.Printf("Database connection error: %v", err)
		return []byte(`{"error": "database connection failed"}`)
	}
	defer db.Close()

	var earliestDate sql.NullString
	var queryStmt string
	var queryArgs []interface{}

	if strings.Contains(query, "%") {
		queryStmt = `
			SELECT MIN(dt.date) AS earliest_date 
			FROM domains d FORCE INDEX (domain_idx)
			JOIN dates dt ON d.dategrp = dt.id 
			WHERE d.domain LIKE ?`
		queryArgs = []interface{}{"%" + query + "%"}
	} else {
		queryStmt = `
			SELECT MIN(dt.date) AS earliest_date 
			FROM domains d FORCE INDEX (domain_idx)
			JOIN dates dt ON d.dategrp = dt.id 
			WHERE d.domain = ?`
		queryArgs = []interface{}{query}
	}

	err = db.QueryRow(queryStmt, queryArgs...).Scan(&earliestDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return []byte(`{"earliest_date": null}`)
		}
		log.Printf("Query error: %v", err)
		return []byte(`{"error": "query failed"}`)
	}

	if !earliestDate.Valid {
		return []byte(`{"earliest_date": null}`)
	}

	parsedDate, err := time.Parse("20060102", earliestDate.String)
	if err != nil {
		log.Printf("Date parsing error: %v", err)
		return []byte(`{"error": "date parsing failed"}`)
	}
	formattedDate := parsedDate.Format("2006-01-02")

	result := struct {
		EarliestDate string `json:"earliest_date"`
	}{
		EarliestDate: formattedDate,
	}

	j, err := json.Marshal(result)
	if err != nil {
		log.Printf("JSON marshal error: %v", err)
		return []byte(`{"error": "json marshal failed"}`)
	}
	return j
}

func seDomainFirstAppearance(w http.ResponseWriter, r *http.Request) {
	parts, err := getPathParams(r.URL.Path, 2)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := parts[1]
	db, user, pass, err := getTLDEnvVars("se_diff")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cacheKey := fmt.Sprintf("seappearance:%s", query)

	result, cacheHit, err := getOrSetCache(cacheKey, MediumTTL, func() []byte {
		return getDomainFirstAppearance(db, user, pass, query)
	})

	if err != nil {
		log.Printf("Cache error: %v", err)
	}

	if cacheHit {
		w.Header().Set("X-Cache", "HIT")
	} else {
		w.Header().Set("X-Cache", "MISS")
	}

	w.Write(result)
}

func nuDomainFirstAppearance(w http.ResponseWriter, r *http.Request) {
	parts, err := getPathParams(r.URL.Path, 2)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := parts[1]
	db, user, pass, err := getTLDEnvVars("nu_diff")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cacheKey := fmt.Sprintf("nuappearance:%s", query)

	result, cacheHit, err := getOrSetCache(cacheKey, MediumTTL, func() []byte {
		return getDomainFirstAppearance(db, user, pass, query)
	})

	if err != nil {
		log.Printf("Cache error: %v", err)
	}

	if cacheHit {
		w.Header().Set("X-Cache", "HIT")
	} else {
		w.Header().Set("X-Cache", "MISS")
	}

	w.Write(result)
}
