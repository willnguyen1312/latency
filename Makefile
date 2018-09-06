.PHONY: all clean webapp api reall deploy

reall: clean all

all: webapp api

api:
	cd api && go build

webapp:
	cd webapp && npm install && npm run build

clean:
	rm -rf webapp/dist api/api

deploy:
	rocket -c api/.rocket_eu.toml
	rocket -c api/.rocket_us.toml
