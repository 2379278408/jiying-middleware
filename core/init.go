package core

import (
	"fmt"
	"os"

	"github.com/beego/beego/v2/client/httplib"
)

func InitCore() error {
	if err := initBuckets(); err != nil {
		return err
	}
	persistSessionKey()
	initCvt()
	initYamlConf()
	InitReplies()
	initToHandleMessage()
	initDataHome()
	initSys()
	initExecutions()
	InitGoja()
	initPluginsCmd()
	go initWatcher()
	initUpdate()
	httplib.SetDefaultSetting(httplib.BeegoHTTPSettings{})
	initCronCmds()
	initHttpServer()
	InitExternalPlugins()
	initPluginAuth()
	initJiying()
	return nil
}

func initBuckets() error {
	bucket, err := MakeBucket("middleware", 6)
	if err != nil {
		return fmt.Errorf("打开应用存储失败: %w", err)
	}
	BucketApp = bucket
	return nil
}

func initDataHome() {
	if err := os.MkdirAll(DataHome, 0o700); err == nil {
		_ = os.Chmod(DataHome, 0o700)
	}
}
