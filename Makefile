.PHONY: all clean webapp api reall deploy_api

VERSION := $(shell cat api/main.go| grep "\sVersion" | cut -d '"' -f2)

reall: clean all

all: webapp api

api:
	cd api && go build

webapp:
	cd webapp && npm install && npm run build

clean:
	rm -rf webapp/dist api/api

deploy_api:
	rocket -c api/.rocket_eu.toml
	rocket -c api/.rocket_us.toml

release: clean
	git tag v$(VERSION)
	git push origin v$(VERSION)
