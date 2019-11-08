dev:
	gin 

deploy:
	cp -r web pkg/handler && \
	now && \
	rm -rf pkg/handler/web

yolo:
	cp -r web pkg/handler && \
	now && now alias smscp.minieggs40.now.sh smscp.xyz && \
	rm -rf pkg/handler/web

test: 
	cat .env | xargs -I {} printf "%s " {} | xargs -I {} echo "env {} go test ./..." | bash

lint: 
	golangci-lint run --no-config --issues-exit-code=0 \
	--disable-all --enable=deadcode  --enable=gocyclo --enable=golint --enable=varcheck \
	--enable=structcheck --enable=maligned --enable=errcheck --enable=dupl --enable=ineffassign \
	--enable=interfacer --enable=unconvert --enable=goconst --enable=gosec --enable=megacheck

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
