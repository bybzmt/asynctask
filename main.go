package main

import (
	"asynctask/server"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
)

var s *server.Server
var dbFile string
var config string
var LogFile string
var LogLevel string

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

	flag.StringVar(&LogFile, "log.file", os.Getenv("LOGFILE"), "log file")
	flag.StringVar(&LogLevel, "log.level", logLevel, "log level")
	flag.StringVar(&dbFile, "db.file", dbfile, "storage file")
	flag.StringVar(&config, "config", "config.json", "config file")
}

func main() {
	flag.Parse()

	l, err := initLog()
	if err != nil {
		logrus.Fatalln(err)
	}

	s, err = server.New(config, dbFile, l)
	if err != nil {
		logrus.Fatalln(err)
	}

	go func() {
		waitSignal()

		s.Stop()

		s.CloseWait()

		s.Kill()
	}()

	logrus.Info("Scheduler Start")

	s.Start()

	logrus.Info("Scheduler Stop")
}

func waitSignal() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)

	for {
		n := <-sig

		switch n {
		case syscall.SIGHUP:
			logrus.Warnln("Reload")

			//realod
			err := s.Reload()
			if err != nil {
				logrus.Warnln("reload", err)
			}

		default:
			return
		}
	}
}

func initLog() (*logrus.Logger, error) {
	var l *logrus.Logger

	if LogFile == "" {
		l = logrus.StandardLogger()
	} else {
		l = logrus.New()

		writer, err := rotatelogs.New(
			LogFile,
			rotatelogs.WithRotationTime(24*time.Hour),
			rotatelogs.WithMaxAge(45*24*time.Hour),
		)
		if err != nil {
			return nil, err
		}
		l.SetOutput(writer)
	}

	switch strings.ToLower(LogLevel) {
	case "error":
		l.SetLevel(logrus.ErrorLevel)
	case "warn":
		l.SetLevel(logrus.WarnLevel)
	case "":
		fallthrough
	case "info":
		l.SetLevel(logrus.InfoLevel)
	case "debug":
		l.SetLevel(logrus.DebugLevel)
	default:
		return nil, fmt.Errorf("Unkown LogLevel: %s", LogLevel)
	}

	l.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})

	return l, nil
}
