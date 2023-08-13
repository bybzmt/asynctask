
.PHONY: go js-init ui all

all: js-init go

go: ui
	go build

js-init:
	cd ./tool/webUI && npm i

ui:
	cd ./tool/webUI && npm run build
