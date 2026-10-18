package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/DeanThompson/ginpprof"
	"jiying/middleware/core"
	_ "jiying/middleware/develop/gin"
	"jiying/middleware/utils"
)

func monitorGoroutine() {
	tick := time.NewTicker(60 * time.Second)
	for range tick.C {
		gcount := runtime.NumGoroutine()
		if gcount < 1 {
			gcount = 1
		}
		if gcount > 0x40000 {
			core.GetMyLog().Warnf("进程goroutine数量过多，%d个，可能导致重启", gcount)
			time.Sleep(3 * time.Second)
			utils.Daemon(0)
		}
	}
}

func main() {
	resPath := flag.String("c", "", "resource path")
	disableTerminal := flag.Bool("d", false, "disable terminal")
	enableTerminal := flag.Bool("t", false, "enable terminal")
	flag.Parse()
	if *resPath != "" {
		utils.ResPath = *resPath
	}

	core.InitCore()
	// pprof 会暴露堆栈、内存与 CPU 采样，默认关闭；仅在显式设置 JIYING_PPROF=1 时挂载。
	if os.Getenv("JIYING_PPROF") == "1" {
		ginpprof.Wrapper(core.Server)
		core.GetMyLog().Warn("pprof 调试端点已启用（/debug/pprof/），请勿暴露到公网")
	} else {
		core.GetMyLog().Info("pprof 未启用；如需调试请设置 JIYING_PPROF=1")
	}
	go monitorGoroutine()

	port := 8080
	if v := os.Getenv("JIYING_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			port = p
		}
	}
	if bucketPort := core.BucketApp.GetInt("port"); bucketPort != 0 {
		port = bucketPort
	}
	addr := "0.0.0.0:" + strconv.Itoa(port)
	if port == 443 {
		addr = ":443"
	}
	srv := &http.Server{Addr: addr, Handler: core.Server, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		var err error
		if port == 443 {
			certFile := core.BucketApp.GetString("certFile")
			keyFile := core.BucketApp.GetString("keyFile")
			err = srv.ListenAndServeTLS(certFile, keyFile)
		} else {
			core.GetMyLog().Info("Http服务已运行(%d)。", port)
			err = srv.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			core.GetMyLog().Error("Http服务退出: %v", err)
		}
	}()
	// 收到退出信号后停止接收新请求，补写计数并给在途请求留出收尾时间。
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		core.GetMyLog().Info("收到退出信号，正在停止服务")
		core.FlushMsgStats()
		core.FlushAudit()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		os.Exit(0)
	}()

	core.LocalIPv4 = utils.GetInternalIP()
	if len(core.LocalIPv4) >= 7 && core.LocalIPv4[:7] == "172.17." {
		host := core.BucketApp.GetString("local_host")
		if utils.IsPrivateIP(host) {
			core.LocalIPv4 = host
		} else {
			core.GetMyLog().Warn("疑似Docker网络地址：%s，无法获取真实内网地址，请在系统管理->系统设置->系统参数中设置本机地址参数。", core.LocalIPv4)
		}
	}
	core.GetMyLog().Info("内网地址：%s 内部端口：%s", core.LocalIPv4, strconv.Itoa(port))

	if !*disableTerminal && *enableTerminal {
		core.GetMyLog().Info("终端交互已启用。")
		reader := bufio.NewReader(os.Stdin)
		for {
			line, _, err := reader.ReadLine()
			if err != nil {
				return
			}
			sender := &core.Faker{}
			sender.Type = 2
			sender.Message = string(line)
			core.Senders <- sender
		}
	}
	if !*disableTerminal && !*enableTerminal {
		core.GetMyLog().Info("终端交互不可用，运行带-t参数即可启用。")
	}

	select {}
}
