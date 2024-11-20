package database

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"os"
)

func Connect(dbName string, dbUser string, dbPass string) (db *sql.DB, err error) {
	dbDriver := "mysql"
	mysqlHostname := os.Getenv("MYSQL_HOSTNAME")
	db, err = sql.Open(dbDriver, dbUser+":"+dbPass+"@tcp"+"("+mysqlHostname+")"+"/"+dbName)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return db, nil
}
