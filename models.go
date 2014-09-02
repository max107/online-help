package main

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"time"
)

var db gorm.DB

func initDatabase() {
	// db, err := gorm.Open("postgres", "user=gorm dbname=gorm sslmode=disable")
	// db, err := gorm.Open("mysql", "root:12345@/gorm?charset=utf8&parseTime=True")
	db, _ = gorm.Open("sqlite3", "./chat.db")

	// Get database connection handle [*sql.DB](http://golang.org/pkg/database/sql/#DB)
	db.DB()

	// Then you could invoke `*sql.DB`'s functions with it
	db.DB().Ping()
	db.DB().SetMaxIdleConns(10)
	db.DB().SetMaxOpenConns(100)

	// Disable table name's pluralization
	db.SingularTable(true)

	// Drop table if exists
	db.DropTableIfExists(Message{})
	// Create table
	db.CreateTable(Message{})

	newMessage("qwe", "qwe")
}

func getMessages(from string, limit int) []Message {
	var messages []Message
	db.Where(&Message{From: from}).Find(&messages)
	return messages
}

func saveMessage(msg Message) Message {
	log.Printf("Create new message: {from: %s, message: %s}", msg.From, msg.Message)
	// returns true if record hasn’t been saved (primary key `Id` is blank)
	db.NewRecord(msg) // => true
	db.Create(&msg)
	return msg
}

func newMessage(from string, message string) Message {
	log.Printf("Create new message: {from: %s, message: %s}", from, message)
	msg := Message{From: from, Message: message, CreatedAt: time.Now()}
	// returns true if record hasn’t been saved (primary key `Id` is blank)
	db.NewRecord(msg) // => true
	db.Create(&msg)
	return msg
}
