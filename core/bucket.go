package core

import (
	"jiying/middleware/develop/a_boltdb"
)

type Bucket interface {
	GetInt(key string, v ...interface{}) int
	GetInt64(key string, v ...interface{}) int64
	GetUint64(key string, v ...interface{}) uint64
	GetString(key string, v ...interface{}) string
	GetBool(key string, v ...interface{}) bool
	GetBytes(key string, v ...interface{}) []byte
	GetFloat64(key string, v ...interface{}) float64
	GetArray(key string, v ...interface{}) []string
	Set(key string, v ...interface{}) bool
	Push2Array(key string, v ...interface{}) bool
	Delete(key string, v ...interface{})
	Keys() []string
	String() string
}

var (
	BucketApp Bucket
)

func MakeBucket(name string, count int32) (Bucket, error) {
	return a_boltdb.NewBucket(name, count)
}

func MigrateBucket(from, to string) int {
	src, err := a_boltdb.NewBucket(from, 6)
	if err != nil || src == nil {
		return 0
	}
	dst, err := a_boltdb.NewBucket(to, 6)
	if err != nil || dst == nil {
		return 0
	}
	return src.MigrateTo(dst)
}
