package server

import (
	"asynctask/scheduler"
	"context"
	"errors"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
	"net"
	"net/http"
	"strings"
	"time"
	_ "time/tzdata"
)

type Server struct {
	Scheduler  scheduler.Scheduler
	Http       http.Server
	Redis      RedisConfig
	LogFile    string
	LogLevel   string
	DbFile     string
	HttpEnable bool
	cronCmd    chan cron_cmd
}

func (s *Server) initLog() error {
	if s.Scheduler.Log == nil {

		var l *logrus.Logger

		if s.LogFile == "" {
			l = logrus.StandardLogger()
		} else {
			l = logrus.New()

			writer, err := rotatelogs.New(
				s.LogFile,
				rotatelogs.WithRotationTime(24*time.Hour),
				rotatelogs.WithMaxAge(45*24*time.Hour),
			)
			if err != nil {
				return err
			}
			l.SetOutput(writer)
		}

		switch strings.ToLower(s.LogLevel) {
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
			return errors.New("Unkown LogLevel")
		}

		l.SetFormatter(&logrus.TextFormatter{
			DisableColors: true,
			FullTimestamp: true,
		})

		s.Scheduler.Log = l
	}

	return nil
}

func (s *Server) initDb() error {
	if s.Scheduler.Db == nil {
		db, err := bolt.Open(s.DbFile, 0666, nil)
		if err != nil {
			return err
		}

		s.Scheduler.Db = db
	}
	return nil
}

func (s *Server) Init() error {

	if err := s.initLog(); err != nil {
		return err
	}

	if err := s.initDb(); err != nil {
		return err
	}

	if err := s.Scheduler.Init(); err != nil {
		return err
	}

	if err := s.initCron(); err != nil {
		return err
	}

	if err := s.initRedis(); err != nil {
		return err
	}

	s.initHttp()

	return nil
}

func (s *Server) Run(ctx context.Context) {

	go func() {
		time.Sleep(time.Millisecond * 10)

		exit := make(chan int, 3)

		go func() {
			if s.HttpEnable {
				s.Http.BaseContext = func(net.Listener) context.Context {
					return ctx
				}

				s.Scheduler.Log.Fatalln(s.Http.ListenAndServe())
			}

			exit <- 1
		}()

		go func() {
			if s.Redis.Key != "" {
				s.RedisRun(ctx)
			}

			exit <- 1
		}()

		go func() {
			s.CronRun(ctx)
			exit <- 1
		}()

		<-ctx.Done()

		<-exit
		<-exit
		<-exit

		s.Scheduler.Close()
	}()

	s.Scheduler.Run()
}
