package main

import (
	"asynctask/scheduler"
	"asynctask/tool"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

var addr = flag.String("addr", "", "listen addr:port")

var redisHost string
var redisPwd string
var redisDb string
var redisKey string

var cfg scheduler.Config
var logFile string
var dbFile string
var WorkerNum uint
var Parallel uint

func init() {
	flag.UintVar(&WorkerNum, "num", 10, "default worker number")
	flag.UintVar(&Parallel, "parallel", 1, "default parallel of task")
	flag.StringVar(&logFile, "log", os.Getenv("log"), "log file e.g: my-[date].log [ENV]")
	flag.StringVar(&dbFile, "dbfile", os.Getenv("dbfile"), "storage file [ENV]")

	flag.StringVar(&redisHost, "redisHost", os.Getenv("redisHost"), "redis addr:port [ENV]")
	flag.StringVar(&redisPwd, "redisPwd", os.Getenv("redisPwd"), "redis addr:port [ENV]")
	flag.StringVar(&redisDb, "redisDb", os.Getenv("redisDb"), "redis addr:port [ENV]")
	flag.StringVar(&redisKey, "redisKey", os.Getenv("redisKey"), "redis addr:port [ENV]")
}

func flagCheck() {
	if WorkerNum < 1 {
		cfg.Log.Fatalln("parameter num must > 0")
	}
	cfg.WorkerNum = uint32(WorkerNum)

	if Parallel < 1 {
		cfg.Log.Fatalln("parameter parallel must >= 1")
	}
	cfg.Parallel = uint32(Parallel)

	if dbFile == "" {
		dbFile = "./asynctask.bolt"
	}

}

func main() {
	flag.Parse()

	logrus.SetLevel(logrus.DebugLevel)
	cfg.Log = logrus.StandardLogger()

	flagCheck()

	db, err := bolt.Open(dbFile, 0666, nil)
	if err != nil {
		cfg.Log.Fatalln("open db error", err)
	}

	cfg.Db = db

	hub, err := scheduler.New(cfg)
	if err != nil {
		cfg.Log.Fatalln(err)
	}

	go func() {
		time.Sleep(time.Millisecond * 10)

		if *addr != "" {
			go tool.HttpRun(hub, *addr)
		}

		if redisHost != "" {
			go tool.RedisRun(hub, redisHost, redisPwd, redisDb, redisKey)
		}

		waitSignal()

		hub.Close()
	}()

	hub.Run()

	cfg.Log.Debugln("main close")
}

func waitSignal() {
	co := make(chan os.Signal, 1)
	signal.Notify(co, os.Interrupt, syscall.SIGTERM)
	<-co
}
