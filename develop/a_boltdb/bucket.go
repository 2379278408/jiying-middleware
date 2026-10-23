package a_boltdb

import (
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"
	"sync"

	bolt "go.etcd.io/bbolt"
	"jiying/middleware/utils"
)

var (
	dbOnce  sync.Once
	db      *bolt.DB
	dbErrl  error
	dbPath_ string
)

type Store struct {
	name string
}

func NewBucket(name string, count int32) (*Store, error) {
	dbOnce.Do(func() {
		dir := path.Join(utils.GetDataHome(), "data", "buckets")
		dbPath_ = path.Join(dir, "middleware.db")
		utils.MkFileDir(dbPath_)
		db, dbErrl = bolt.Open(dbPath_, 0o600, &bolt.Options{Timeout: 3 * 1000 * 1000 * 1000})
		if dbErrl == nil {
			dbErrl = db.Update(func(tx *bolt.Tx) error {
				_, err := tx.CreateBucketIfNotExists([]byte("buckets"))
				return err
			})
		}
	})
	if dbErrl != nil {
		return nil, dbErrl
	}
	if err := db.Update(func(tx *bolt.Tx) error {
		bk, err := tx.CreateBucketIfNotExists([]byte(name))
		if err != nil {
			return err
		}
		_ = bk
		return nil
	}); err != nil {
		return nil, err
	}
	return &Store{name: name}, nil
}

func keyExpand(base string, v []interface{}) string {
	if len(v) == 0 {
		return anyKey(base)
	}
	parts := make([]string, 0, len(v)+1)
	parts = append(parts, base)
	for _, s := range v {
		parts = append(parts, fmt.Sprint(s))
	}
	return anyKey(strings.Join(parts, ""))
}

func anyKey(k string) string { return k }

func (s *Store) put(key string, data []byte) error {
	return db.Update(func(tx *bolt.Tx) error {
		bk := tx.Bucket([]byte(s.name))
		if bk == nil {
			var err error
			bk, err = tx.CreateBucketIfNotExists([]byte(s.name))
			if err != nil {
				return err
			}
		}
		return bk.Put([]byte(key), data)
	})
}

func (s *Store) get(key string) []byte {
	var data []byte
	db.View(func(tx *bolt.Tx) error {
		bk := tx.Bucket([]byte(s.name))
		if bk != nil {
			data = bk.Get([]byte(key))
		}
		return nil
	})
	return data
}

func (s *Store) Set(key string, v ...interface{}) bool {
	if len(v) != 1 {
		return false
	}
	data, err := json.Marshal(v[0])
	if err != nil {
		data = []byte(fmt.Sprint(v[0]))
	}
	return s.put(anyKey(key), data) == nil
}

func (s *Store) GetBytes(key string, v ...interface{}) []byte {
	return s.get(keyExpand(key, v))
}

func (s *Store) GetString(key string, v ...interface{}) string {
	data := s.get(keyExpand(key, v))
	var out string
	if err := json.Unmarshal(data, &out); err == nil {
		return out
	}
	return string(data)
}

func (s *Store) GetInt(key string, v ...interface{}) int {
	n := 0
	data := s.get(keyExpand(key, v))
	if err := json.Unmarshal(data, &n); err == nil {
		return n
	}
	if i, err := strconv.Atoi(string(data)); err == nil {
		return i
	}
	return 0
}

func (s *Store) GetInt64(key string, v ...interface{}) int64 {
	data := s.get(keyExpand(key, v))
	var n int64
	if err := json.Unmarshal(data, &n); err == nil {
		return n
	}
	if i, err := strconv.ParseInt(string(data), 10, 64); err == nil {
		return i
	}
	return 0
}

func (s *Store) GetUint64(key string, v ...interface{}) uint64 {
	data := s.get(keyExpand(key, v))
	var n uint64
	if err := json.Unmarshal(data, &n); err == nil {
		return n
	}
	if i, err := strconv.ParseUint(string(data), 10, 64); err == nil {
		return i
	}
	return 0
}

func (s *Store) GetFloat64(key string, v ...interface{}) float64 {
	data := s.get(keyExpand(key, v))
	var f float64
	if err := json.Unmarshal(data, &f); err == nil {
		return f
	}
	if f64, err := strconv.ParseFloat(string(data), 64); err == nil {
		return f64
	}
	return 0
}

func (s *Store) GetBool(key string, v ...interface{}) bool {
	data := s.get(keyExpand(key, v))
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		return b
	}
	return string(data) == "true"
}

func (s *Store) GetArray(key string, v ...interface{}) []string {
	data := s.get(keyExpand(key, v))
	arr := []string{}
	if json.Unmarshal(data, &arr) == nil {
		return arr
	}
	if s0 := string(data); s0 != "" {
		if strings.HasPrefix(s0, "[") && strings.HasSuffix(s0, "]") {
			rest := s0[1 : len(s0)-1]
			for _, it := range strings.Split(rest, ",") {
				arr = append(arr, strings.TrimSpace(strings.Trim(it, `"`)))
			}
			return arr
		}
		return []string{s0}
	}
	return arr
}

func (s *Store) Push2Array(key string, v ...interface{}) bool {
	arr := s.GetArray(anyKey(key))
	parts := make([]interface{}, 0, len(v))
	parts = append(parts, v...)
	if len(parts) > 0 {
		switch val := parts[0].(type) {
		case string:
			arr = append(arr, val)
		default:
			j, _ := json.Marshal(parts[0])
			arr = append(arr, string(j))
		}
	}
	data, _ := json.Marshal(arr)
	return s.put(anyKey(key), data) == nil
}

func (s *Store) Delete(key string, v ...interface{}) {
	k := keyExpand(key, v)
	db.Update(func(tx *bolt.Tx) error {
		bk := tx.Bucket([]byte(s.name))
		if bk != nil {
			bk.Delete([]byte(k))
		}
		return nil
	})
}

func (s *Store) Keys() []string {
	var keys []string
	db.View(func(tx *bolt.Tx) error {
		bk := tx.Bucket([]byte(s.name))
		if bk == nil {
			return nil
		}
		c := bk.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			keys = append(keys, string(k))
		}
		return nil
	})
	return keys
}

func (s *Store) MigrateTo(dst *Store) int {
	if dst == nil || s.name == dst.name {
		return 0
	}
	moved := 0
	_ = db.Update(func(tx *bolt.Tx) error {
		src := tx.Bucket([]byte(s.name))
		if src == nil {
			return nil
		}
		target, err := tx.CreateBucketIfNotExists([]byte(dst.name))
		if err != nil {
			return err
		}
		c := src.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if target.Get(k) == nil {
				_ = target.Put(k, v)
				moved++
			}
		}
		return nil
	})
	return moved
}

func (s *Store) String() string {
	return s.name
}
