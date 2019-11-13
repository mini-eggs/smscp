dev:
	gin 

deploy: cli cpminify
	now && now alias smscp.minieggs40.now.sh beta.smscp.xyz && \
	rm -rf pkg/handler/web

push: 
	now alias smscp.minieggs40.now.sh smscp.xyz

yolo: cli cpminify
	now && now alias smscp.minieggs40.now.sh smscp.xyz && \
	rm -rf pkg/handler/web

cpminify:
	cp -r web pkg/handler && \
	bash -c "find pkg/handler/web/html -type f | grep -e '\.html' -e '\.css' -e '\.js' | xargs -I {} echo 'minify {} > {}.out && mv {}.out {}' | bash"

test: 
	cat .env | xargs -I {} printf "%s " {} | xargs -I {} echo "env {} go test -count 1 ./..." | bash

lint: 
	golangci-lint run --no-config --issues-exit-code=0 \
	--disable-all --enable=deadcode  --enable=gocyclo --enable=golint --enable=varcheck \
	--enable=structcheck --enable=maligned --enable=errcheck --enable=dupl --enable=ineffassign \
	--enable=interfacer --enable=unconvert --enable=goconst --enable=gosec --enable=megacheck

migrate_dev:
	bash -c "[[ "`curl -s http://localhost:3000/ping`" == pong ]] && curl 'http://localhost:3000/migrate' -H 'Content-Type: application/x-www-form-urlencoded' --data 'MigrationKey=`cat .env | grep MIGRATION_KEY | awk -F'[/=]' '{print\$$2}'`' && exit 0 || echo migration failed && exit 1"

migrate_prod:
	bash -c "[[ "`curl -s https://smscp.xyz/ping`" == pong ]] && curl 'https://smscp.xyz/migrate' -H 'Content-Type: application/x-www-form-urlencoded' --data 'MigrationKey=`cat .env | grep MIGRATION_KEY | awk -F'[/=]' '{print\$$2}'`' && exit 0 || echo migration failed && exit 1"

cli: mac win lin

mac: 
	mkdir -p dl/mac && \
	cd cmd/smscp && GOOS=darwin  GOARCH=386 go build -o ../../dl/mac/smscp && cd ../..

win: 
	mkdir -p dl/win && \
	cd cmd/smscp && GOOS=windows GOARCH=386 go build -o ../../dl/win/smscp && cd ../.. && \
	mv dl/win/smscp dl/win/smscp.exe

lin: 
	mkdir -p dl/lin && \
	cd cmd/smscp && GOOS=linux   GOARCH=386 go build -o ../../dl/lin/smscp && cd ../..
