
.PHONY: go js-init ui all



all: js-init go

go: ui
	CGO_ENABLED=0 go build

js-init:
	cd ./server/webUI && npm i

ui:
	cd ./server/webUI && npm run build
