package db

import (
	"fmt"
	"log"
	"strings"

	"github.com/jjshen2000/simple-ads/config"
	"github.com/jmoiron/sqlx"
)

var db *sqlx.DB

var schema = `
DROP TABLE IF EXISTS condition_country;

DROP TABLE IF EXISTS advertisement_condition;

DROP TABLE IF EXISTS advertisement;

CREATE TABLE advertisement (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    start_at DATETIME NOT NULL,
    end_at DATETIME NOT NULL
)

CREATE TABLE advertisement_condition (
    id INT AUTO_INCREMENT PRIMARY KEY,
    advertisement_id INT NOT NULL,
    age_start TINYINT UNSIGNED, -- 0-100
    age_end TINYINT UNSIGNED,   -- 0-100
    gender CHAR(2),             -- M, F, MF
	unlimited_country BOOL,
    platform TINYINT UNSIGNED,  -- bit-wise 'android', 'ios', 'web'
    FOREIGN KEY (advertisement_id) REFERENCES advertisement(id)
);

CREATE TABLE condition_country (
    condition_id INT,
    country_code CHAR(2), -- ISO-3166 alpha 2 code
    PRIMARY KEY (condition_id, country_code),
    FOREIGN KEY (condition_id) REFERENCES advertisement_condition(id)
);`

func init() {
	// Connect to MySQL database
	config := config.GetConfig()
	dsn := fmt.Sprintf("%s:%s@%s(%s:%d)/%s",
		config.Database.Username,
		config.Database.Password,
		config.Database.Network,
		config.Database.Server,
		config.Database.Port,
		config.Database.Database)

	dbcoon, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		log.Fatalln("Failed to connect to MySQL:", err)
	}
	db = dbcoon

	queries := strings.Split(schema, "\n\n")

	for _, query := range queries {
		db.Exec(query)
	}
}

func GetDB() *sqlx.DB {
	return db
}
