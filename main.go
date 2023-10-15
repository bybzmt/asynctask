package main

import (
	"asynctask/server"
	"context"
	"flag"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

var Server server.Server

func init() {
	dbfile := os.Getenv("dbfile")

	if dbfile == "" {
		dbfile = "./asynctask.bolt"
	}

	logLevel := os.Getenv("logLevel")
	if logLevel == "" {
		logLevel = "info"
	}

	flag.StringVar(&Server.Http.Addr, "httpAddr", os.Getenv("httpAddr"), "http server addr")
	flag.BoolVar(&Server.HttpEnable, "httpEnable", os.Getenv("httpEnable") != "", "http server enable")

	flag.StringVar(&Server.LogFile, "logfile", os.Getenv("logfile"), "log file")
	flag.StringVar(&Server.LogLevel, "logLevel", logLevel, "log level")
	flag.StringVar(&Server.DbFile, "dbfile", dbfile, "storage file")

	flag.StringVar(&Server.Redis.Addr, "redisAddr", os.Getenv("redisAddr"), "redis addr:port")
	flag.StringVar(&Server.Redis.Pwd, "redisPwd", os.Getenv("redisPwd"), "redis password")
	flag.StringVar(&Server.Redis.Db, "redisDb", os.Getenv("redisDb"), "redis db")
	flag.StringVar(&Server.Redis.Key, "redisKey", os.Getenv("redisKey"), "redis list key for addTask")
}

func main() {
	flag.Parse()

	logrus.SetLevel(logrus.DebugLevel)

	err := Server.Init()
	if err != nil {
		logrus.Fatalln(err)
	}

	ctx, canceler := context.WithCancel(context.Background())

	go Server.Run(ctx)

	logrus.Info("Scheduler Start")

	waitSignal()

	canceler()

	logrus.Info("Scheduler Stop")
}

func waitSignal() {
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt, syscall.SIGTERM)
	<-s
}
