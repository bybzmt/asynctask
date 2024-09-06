package main

import (
	"asynctask/server"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"log/syslog"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"
)

var s *server.Server
var dbFile string
var config string
var LogFmt string
var LogFile string
var LogLevel string
var LogType string

var syslogFacility string
var syslogTime bool
var syslogOpts server.SyslogOptions

func init() {
	dbfile := os.Getenv("DBFILE")

	if dbfile == "" {
		dbfile = "asynctask.db"

		if file, err := os.Executable(); err != nil {
			if base := path.Base(file); base != "" {
				dbfile = base + ".db"
			}
		}
	}

	logLevel := os.Getenv("LOGLEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	flag.StringVar(&LogFile, "log.file", os.Getenv("LOGFILE"), "log file")
	flag.StringVar(&LogLevel, "log.level", logLevel, "log level")
	flag.StringVar(&LogFmt, "log.fmt", "logfmt", "log farmat: json or logfmt")
	flag.StringVar(&LogType, "log.type", "stderr", "log type: stderr|file|syslog")

	flag.StringVar(&syslogOpts.Network, "syslog.natwork", "", "syslog tcp|udp")
	flag.StringVar(&syslogOpts.Addr, "syslog.addr", "", "syslog ip:port")
	flag.StringVar(&syslogFacility, "syslog.facility", "LOG_LOCAL0", "syslog LOG_LOCAL0 -> LOG_LOCAL7")
	flag.StringVar(&syslogOpts.Tag, "syslog.tag", "", "syslog tag")
	flag.BoolVar(&syslogOpts.WithTime, "syslog.time", false, "syslog time")

	flag.StringVar(&dbFile, "db.file", dbfile, "storage file")
	flag.StringVar(&config, "config", "config.toml", "config file json or toml")
}

func main() {
	flag.Parse()

	l, err := initLog()
	if err != nil {
		slog.Error("initLog", "err", err)
		os.Exit(1)
	}

	s, err = server.New(config, dbFile, l)
	if err != nil {
		slog.Error("New", "err", err)
		os.Exit(1)
	}

	go func() {
		waitSignal()

		slog.Warn("Stop")

		s.Stop()

		s.WaitStop()

		s.Kill()
	}()

	slog.Info("Start")

	s.Start()

	time.Sleep(time.Millisecond * 500)

	slog.Info("Stoped")
}

func waitSignal() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)

	for {
		n := <-sig

		switch n {
		case syscall.SIGHUP:
			slog.Warn("Reload")

			//realod
			err := s.Reload()
			if err != nil {
				slog.Error("Reload", "err", err)
			} else {
				slog.Warn("Reload Success")
			}
		default:
			return
		}
	}
}

func initLog() (*slog.Logger, error) {
	var level slog.Level

	ho := &slog.HandlerOptions{
		Level: level,
	}

	switch strings.ToLower(LogLevel) {
	case "error":
		level = slog.LevelError
	case "warn":
		level = slog.LevelWarn
	case "":
		fallthrough
	case "info":
		level = slog.LevelInfo
	case "debug":
		level = slog.LevelDebug
	default:
		return nil, fmt.Errorf("Unkown LogLevel: %s", LogLevel)
	}

	var w io.Writer

	if LogType == "syslog" {
		switch syslogFacility {
		case "LOG_LOCAL0":
			syslogOpts.Priority = syslog.LOG_LOCAL0
		case "LOG_LOCAL1":
			syslogOpts.Priority = syslog.LOG_LOCAL1
		case "LOG_LOCAL2":
			syslogOpts.Priority = syslog.LOG_LOCAL2
		case "LOG_LOCAL3":
			syslogOpts.Priority = syslog.LOG_LOCAL3
		case "LOG_LOCAL4":
			syslogOpts.Priority = syslog.LOG_LOCAL4
		case "LOG_LOCAL5":
			syslogOpts.Priority = syslog.LOG_LOCAL5
		case "LOG_LOCAL6":
			syslogOpts.Priority = syslog.LOG_LOCAL6
		case "LOG_LOCAL7":
			syslogOpts.Priority = syslog.LOG_LOCAL7
		default:
			return nil, fmt.Errorf("Unkown syslog.facility: %s", syslogFacility)
		}

		syslogOpts.Level = ho.Level

		return server.NewSyslog(&syslogOpts)
	} else if LogType == "stderr" {
		w = os.Stderr
	} else if LogType == "file" {
		if LogFile == "" {
			return nil, fmt.Errorf("LogFile empty")
		}

		f, err := os.OpenFile(LogFile, os.O_APPEND|os.O_CREATE, 0644)

		if err != nil {
			return nil, err
		}

		w = f
	} else {
		return nil, fmt.Errorf("Unkown log type: %s", LogType)
	}

	var h slog.Handler

	if LogFmt == "json" {
		h = slog.NewJSONHandler(w, ho)
	} else {
		h = slog.NewTextHandler(w, ho)
	}

	return slog.New(h), nil
}
