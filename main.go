package main

import (
	"flag"
	"github.com/cihub/seelog"
	_ "github.com/yincongcyincong/go12306/action"
	"github.com/yincongcyincong/go12306/utils"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
)

var (
	runType   = flag.String("run_type", "command", "web：网页模式")
	wxrobot   = flag.String("wxrobot", "", "企业微信机器人通知")
	mustDevice = flag.String("must_device", "0", "强制生成设备信息")
	configPath =flag.String("c", "./conf/conf.ini", "配置文件路径")
)

func main() {
	flag.Parse()
	Init()
	if *runType == "command" {
		go CommandStart()
	}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT)
	select {
	case <-sigs:
		seelog.Info("用户登出")
		utils.SaveConf()
		// action.LoginOut()
	}
}
