package database

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// local_root_password
func ConnectMySQL() (*gorm.DB, error) {
	dsn := "video_feed:video_feed_password@tcp(127.0.0.1:3307)/video_feed?charset=utf8mb4&parseTime=True&loc=Local"

	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}
