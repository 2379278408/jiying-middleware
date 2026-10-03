package core

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"jiying/middleware/utils"
)

type Loggers struct {
	logfilelevel  int8
	terminallevel int8
	path          string
	filename      string
	file          *os.File
	time          *time.Time
}

const (
	DebugLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

var (
	myLog   Loggers
	getOnce sync.Once
	logChan chan string
)

func GetMyLog() Loggers {
	getOnce.Do(func() {
		myLog.path = path.Join(utils.GetResHome(), "logs")
		os.MkdirAll(myLog.path, 0o777)
		myLog.newLoggersFile()
		go func() {
			for {
				time.Sleep(2 * time.Hour)
				GetMyLog().Warn("清理日志文件logFiles")
				files, err := os.ReadDir(path.Join(utils.GetResHome(), "logs"))
				if err != nil {
					continue
				}
				threshold := time.Now().Add(-72*time.Hour).Format("20060102") + ".log"
				for _, f := range files {
					name := f.Name()
					if len(name) > 3 && strings.HasSuffix(name, ".log") && name < threshold {
						os.RemoveAll(path.Join(utils.GetResHome(), "logs", name))
					}
				}
			}
		}()
		if v := os.Getenv("autLogfileLevel"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				if n < 6 {
					myLog.logfilelevel = int8(n)
				} else {
					myLog.logfilelevel = 2
				}
			}
		}
		if v := os.Getenv("autTerminalLevel"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				if n < 6 {
					myLog.terminallevel = int8(n)
				} else {
					myLog.terminallevel = 0
				}
			}
		}
	})
	return myLog
}

func (l *Loggers) SetLogfileLevel(level int8) {
	l.logfilelevel = level
}

func (l *Loggers) SetPath(p string) {
	l.path = p
}

func (l *Loggers) SetTerminalLevel(level int8) {
	l.terminallevel = level
}

func (l *Loggers) isOneDay() bool {
	if l.time == nil {
		return false
	}
	now, t := time.Now(), *l.time
	return now.Year() == t.Year() && now.Month() == t.Month() && now.Day() == t.Day()
}

func (l *Loggers) newLoggersFile() {
	t := time.Now()
	fullPath := path.Join(l.path, t.Format("20060102")+".log")
	w, err := os.OpenFile(fullPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o777)
	if err != nil {
		fmt.Println("日志文件打开失败:", fullPath, err)
		return
	}
	if l.file != nil {
		l.file.Close()
	}
	l.file = w
	l.time = &t
	fmt.Println("日志文件:", fullPath)
}

func (l *Loggers) Sync() {
	if l.file != nil {
		l.file.Sync()
	}
}

func (l *Loggers) writeToLoggers(level int, msg string) {
	if !l.isOneDay() {
		l.newLoggersFile()
	}
	var levelChar string
	switch level {
	case 0:
		levelChar = "DEBUG"
	case 1:
		levelChar = "INFO"
	case 2:
		levelChar = "WARN"
	case 3:
		levelChar = "ERROR"
	default:
		levelChar = "LEVEL"
	}
	_, file, line, _ := runtime.Caller(3)
	fn, _ := os.Hostname()
	_ = fn
	logstr := fmt.Sprintf("[[ %s ] [ %s ] [ %s : %d ] %s]", time.Now().Format("2006-01-02 15:04:05"), levelChar, file, line, msg)

	if logChan != nil {
		logChan <- logstr
	}
	if l.logfilelevel <= int8(level) {
		fmt.Fprintln(l.file, logstr)
	}
}

func (l Loggers) Debug(format string, args ...interface{}) {
	if l.logfilelevel <= DebugLevel {
		l.writeToLoggers(DebugLevel, fmt.Sprintf(format, args...))
	}
}

func (l Loggers) Info(format string, args ...interface{}) {
	if l.logfilelevel <= InfoLevel {
		l.writeToLoggers(InfoLevel, fmt.Sprintf(format, args...))
	}
}

func (l Loggers) Warn(format string, args ...interface{}) {
	if l.logfilelevel <= WarnLevel {
		l.writeToLoggers(WarnLevel, fmt.Sprintf(format, args...))
	}
}

func (l Loggers) Error(format string, args ...interface{}) {
	if l.logfilelevel <= ErrorLevel {
		l.writeToLoggers(ErrorLevel, fmt.Sprintf(format, args...))
	}
}

func (l Loggers) Debugf(format string, args ...interface{}) {
	if l.logfilelevel <= DebugLevel {
		l.writeToLoggers(DebugLevel, fmt.Sprintf(format, args...))
	}
}

func (l Loggers) Infof(format string, args ...interface{}) {
	if l.logfilelevel <= InfoLevel {
		l.writeToLoggers(InfoLevel, fmt.Sprintf(format, args...))
	}
}

func (l Loggers) Warnf(format string, args ...interface{}) {
	if l.logfilelevel <= WarnLevel {
		l.writeToLoggers(WarnLevel, fmt.Sprintf(format, args...))
	}
}

func (l Loggers) Errorf(format string, args ...interface{}) {
	if l.logfilelevel <= ErrorLevel {
		l.writeToLoggers(ErrorLevel, fmt.Sprintf(format, args...))
	}
}

func initLogger() {
	Server.Group("/sselogs", func(c *gin.Context) {
		ip := c.ClientIP()
		if ip != "127.0.0.1" && ip != "::1" {
			c.AbortWithStatus(403)
			return
		}
		if !CheckCookie(c) {
			c.Abort()
			return
		}
		c.Next()
	}, func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")
		newChan := make(chan string)
		oldChan := logChan
		logChan = newChan
		defer func() {
			logChan = oldChan
			close(newChan)
		}()
		clientGone := c.Request.Context().Done()
		for {
			select {
			case msg := <-newChan:
				c.SSEvent("message", msg)
				c.Writer.Flush()
			case <-clientGone:
				GetMyLog().Warnf("SSE客户端断开")
				return
			}
		}
	})
}
