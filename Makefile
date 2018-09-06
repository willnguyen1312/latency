.PHONY: all clean webapp api reall

reall: clean all

all: webapp api

api:
	cd api && go build

webapp:
	cd webapp && npm run build

clean:
	rm -rf webapp/dist api/api
