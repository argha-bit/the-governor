package db

import (
	"fmt"
	"log"
	"os"
	"sync"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
)

type DBConfig struct {
	*gorm.DB
}

var mysqlConnect *DBConfig
var once sync.Once

func GetMySQL() (*DBConfig, error) {
	var err error
	once.Do(func() {
		connection, err := gorm.Open("mysql", getConnectionString())
		if err != nil {
			log.Println("Error connecting to MySQL", err.Error())
			return
		}
		mysqlConnect = &DBConfig{connection}
	})
	log.Println("MySQL is Connected")
	return mysqlConnect, err
}

func getConnectionString() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True", os.Getenv("DB_USER"), os.Getenv("DB_PASS"), os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"))
}
func Close() {
	log.Println("Closing MySQL Connection")
	if mysqlConnect != nil {
		_ = mysqlConnect.Close()
	}
	mysqlConnect = nil
}
