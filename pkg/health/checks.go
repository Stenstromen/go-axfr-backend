package health

import (
	"fmt"
	"go-axfr-backend/internal/database"
	"go-axfr-backend/internal/models"
	"log"
	"net"
	"time"
)

func CheckDatabases(dbs []models.DbConfig) error {
	for _, cfg := range dbs {
		conn, err := database.Connect(cfg.Database, cfg.Username, cfg.Password)
		if err != nil {
			log.Printf("Error connecting to %s database: %v", cfg.DbName, err)
			return fmt.Errorf("failed to connect to %s database: %v", cfg.DbName, err)
		}
		defer conn.Close()
	}
	return nil
}

func CheckMySQLConnection(hostname string) error {
	timeout := 5 * time.Second
	_, err := net.DialTimeout("tcp", hostname+":3306", timeout)
	return err
}
