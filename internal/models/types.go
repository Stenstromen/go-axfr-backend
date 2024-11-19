package models

type Amounts struct {
	Date   int `json:"date"`
	Amount int `json:"amount"`
}

type Rows struct {
	Domain string `json:"domain"`
}

type DateAmount struct {
	Date   string `json:"date"`
	Amount int    `json:"amount"`
}

type DbConfig struct {
	Database string
	Username string
	Password string
	DbName   string
}
