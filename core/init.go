package core

import (
	"os"

	"github.com/beego/beego/v2/client/httplib"
)

func InitCore() {
	initBuckets()
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
}

func initBuckets() {
	BucketApp, _ = MakeBucket("middleware", 6)
}

func initDataHome() {
	if err := os.MkdirAll(DataHome, 0o700); err == nil {
		_ = os.Chmod(DataHome, 0o700)
	}
}
