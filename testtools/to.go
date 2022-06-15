package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"
	"time"
)

var addr = flag.String("addr", ":8081", "listen addr:port")

func main() {
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s_code := r.FormValue("code")
		s_sleep := r.FormValue("sleep")

		code, _ := strconv.Atoi(s_code)
		sleep, _ := strconv.Atoi(s_sleep)

		if sleep > 0 {
			time.Sleep(time.Duration(sleep) * time.Millisecond)
		}

		w.WriteHeader(code)
		w.Write([]byte(http.StatusText(code)))
	})

	log.Fatal(http.ListenAndServe(*addr, nil))
}
