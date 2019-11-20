.SILENT :
.PHONY : await clean fmt

TAG:=`git describe --abbrev=0 --tags`
LDFLAGS:=-s -w -X main.buildVersion=$(TAG)

all: await

deps:
	go get github.com/robfig/glock
	glock sync -n < GLOCKFILE

await:
	echo "Building await"
	go install -ldflags "$(LDFLAGS)"

dist-clean:
	rm -rf dist
	rm -f await-*.tar.gz

dist: deps dist-clean
	mkdir -p dist/alpine-linux/amd64 && GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -a -tags netgo -installsuffix netgo -o dist/alpine-linux/amd64/await
	mkdir -p dist/linux/amd64 && GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/linux/amd64/await
	mkdir -p dist/linux/386 && GOOS=linux GOARCH=386 go build -ldflags "$(LDFLAGS)" -o dist/linux/386/await
	mkdir -p dist/linux/armel && GOOS=linux GOARCH=arm GOARM=5 go build -ldflags "$(LDFLAGS)" -o dist/linux/armel/await
	mkdir -p dist/linux/armhf && GOOS=linux GOARCH=arm GOARM=6 go build -ldflags "$(LDFLAGS)" -o dist/linux/armhf/await
	mkdir -p dist/linux/arm64 && GOOS=linux GOARCH=arm64 GOARM=7 go build -ldflags "$(LDFLAGS)" -o dist/linux/arm64/await
	mkdir -p dist/darwin/amd64 && GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/darwin/amd64/await

release: dist
	find dist -type f | xargs upx --brute
	tar -cvzf await-alpine-linux-amd64-$(TAG).tar.gz -C dist/alpine-linux/amd64 await
	tar -cvzf await-linux-amd64-$(TAG).tar.gz -C dist/linux/amd64 await
	tar -cvzf await-linux-386-$(TAG).tar.gz -C dist/linux/386 await
	tar -cvzf await-linux-armel-$(TAG).tar.gz -C dist/linux/armel await
	tar -cvzf await-linux-armhf-$(TAG).tar.gz -C dist/linux/armhf await
	tar -cvzf await-linux-arm64-$(TAG).tar.gz -C dist/linux/arm64 await
	tar -cvzf await-darwin-amd64-$(TAG).tar.gz -C dist/darwin/amd64 await
