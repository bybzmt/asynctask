package main

import (
	"asynctask/server"
	"context"
	"flag"
	"os"
	"os/signal"
	"path"
	"syscall"

	"github.com/sirupsen/logrus"
)

var Server server.Server

func init() {
	dbfile := os.Getenv("DBFILE")

	if dbfile == "" {
		file, _ := os.Executable()
		base := path.Base(file)

		if base != "" {
			dbfile = base + ".bolt"
		} else {
			dbfile = "asynctask.bolt"
		}
	}

	logLevel := os.Getenv("LOGLEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	flag.StringVar(&Server.Http.Addr, "http.addr", os.Getenv("HTTPADDR"), "http server addr")
	flag.BoolVar(&Server.HttpEnable, "http.enable", true, "http server enable")

	flag.StringVar(&Server.LogFile, "log.file", os.Getenv("LOGFILE"), "log file")
	flag.StringVar(&Server.LogLevel, "log.level", logLevel, "log level")
	flag.StringVar(&Server.DbFile, "db.file", dbfile, "storage file")

	flag.StringVar(&Server.Redis.Addr, "redis.addr", os.Getenv("REDISADDR"), "redis addr:port")
	flag.StringVar(&Server.Redis.Pwd, "redis.pwd", os.Getenv("REDISPWD"), "redis password")
	flag.StringVar(&Server.Redis.Db, "redis.db", os.Getenv("REDISDB"), "redis db")
	flag.StringVar(&Server.Redis.Key, "redis.key", os.Getenv("REDISKEY"), "redis list key for addTask")
}

func main() {
	flag.Parse()

	err := Server.Init()
	if err != nil {
		logrus.Fatalln(err)
	}

	ctx, canceler := context.WithCancel(context.Background())

    go func() {
        waitSignal()
        canceler()
    }()

	logrus.Info("Scheduler Start")

	Server.Run(ctx)

	logrus.Info("Scheduler Stop")
}

func waitSignal() {
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt, syscall.SIGTERM)
	<-s
}
