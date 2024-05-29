package db

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)

// DBConfig holds the database configuration
type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// SetupDB initializes the database and creates the necessary tables
func SetupDB(config DBConfig) (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Проверяем соединение
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	// Создаем таблицу, если она не существует
	query := `
	CREATE TABLE IF NOT EXISTS air_quality_data (
		id SERIAL PRIMARY KEY,
		timestamp TIMESTAMP NOT NULL,
		coords TEXT,
		wind_direction INT,
		wind_speed INT,
		pm1 INT,
		pm25 INT,
		pm10 INT
		
	);
	`
	_, err = db.Exec(query)
	if err != nil {
		return nil, err
	}

	return db, nil
}
