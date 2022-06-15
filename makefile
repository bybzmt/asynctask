
.PHONY: go js-init ui all

all: js-init go

go: ui
	go build

js-init:
	cd ./webUI && npm i

ui:
	cd ./webUI && npm run build
