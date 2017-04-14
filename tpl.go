package main

import (
	"html/template"
	"net/http"
)

func page_res(w http.ResponseWriter, r *http.Request) {
	data, err := Asset(r.URL.Path[1:])
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Write(data)
}

func page_favicon(w http.ResponseWriter, r *http.Request) {
	data, err := Asset("res/favicon.ico")
	if err != nil {
		panic(err)
	}

	w.Write(data)
}

func load_tpl(name string) *template.Template {
	tpl_layout, err := Asset("res/layout.tpl")
	if err != nil {
		panic(err)
	}

	tmpl, err := template.New("name").Parse(string(tpl_layout))
	if err != nil {
		panic(err)
	}

	tpl_config_show, err := Asset("res/" + name)
	if err != nil {
		panic(err)
	}

	t, err := tmpl.Parse(string(tpl_config_show))
	if err != nil {
		panic(err)
	}

	return t
}

func show_confirm(w http.ResponseWriter, r *http.Request, msg, yes, no string) {
	tmpl := load_tpl("confirm.tpl")

	var data = struct {
		Msg string
		Yes string
		No  string
	}{
		Msg: msg,
		Yes: yes,
		No:  no,
	}

	tmpl.Execute(w, data)
}
