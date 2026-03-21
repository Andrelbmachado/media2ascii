BINARY   = media2ascii
VERSION  = v2.0.0
DIST_DIR = dist
NPM_BIN  = npm/bin

.PHONY: all darwin-amd64 darwin-arm64 linux-amd64 linux-arm64 linux-arm windows-amd64 windows-arm64 freebsd-amd64 clean npm-copy

all: darwin-amd64 darwin-arm64 linux-amd64 linux-arm64 linux-arm windows-amd64 windows-arm64 freebsd-amd64

darwin-amd64:
	GOOS=darwin  GOARCH=amd64 go build -o $(DIST_DIR)/$(BINARY)_darwin_amd64 .
	cd $(DIST_DIR) && tar -czf $(BINARY)_darwin_amd64.tar.gz $(BINARY)_darwin_amd64

darwin-arm64:
	GOOS=darwin  GOARCH=arm64 go build -o $(DIST_DIR)/$(BINARY)_darwin_arm64 .
	cd $(DIST_DIR) && tar -czf $(BINARY)_darwin_arm64.tar.gz $(BINARY)_darwin_arm64

linux-amd64:
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o $(DIST_DIR)/$(BINARY)_linux_amd64 .
	cd $(DIST_DIR) && tar -czf $(BINARY)_linux_amd64.tar.gz $(BINARY)_linux_amd64

linux-arm64:
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -o $(DIST_DIR)/$(BINARY)_linux_arm64 .
	cd $(DIST_DIR) && tar -czf $(BINARY)_linux_arm64.tar.gz $(BINARY)_linux_arm64

linux-arm:
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm GOARM=7 go build -o $(DIST_DIR)/$(BINARY)_linux_arm .
	cd $(DIST_DIR) && tar -czf $(BINARY)_linux_arm.tar.gz $(BINARY)_linux_arm

windows-amd64:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o $(DIST_DIR)/$(BINARY)_windows_amd64.exe .
	cd $(DIST_DIR) && zip $(BINARY)_windows_amd64.zip $(BINARY)_windows_amd64.exe

windows-arm64:
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -o $(DIST_DIR)/$(BINARY)_windows_arm64.exe .
	cd $(DIST_DIR) && zip $(BINARY)_windows_arm64.zip $(BINARY)_windows_arm64.exe

freebsd-amd64:
	CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -o $(DIST_DIR)/$(BINARY)_freebsd_amd64 .
	cd $(DIST_DIR) && tar -czf $(BINARY)_freebsd_amd64.tar.gz $(BINARY)_freebsd_amd64

npm-copy: all
	cp $(DIST_DIR)/$(BINARY)_darwin_amd64        $(NPM_BIN)/$(BINARY)_darwin_amd64
	cp $(DIST_DIR)/$(BINARY)_darwin_arm64        $(NPM_BIN)/$(BINARY)_darwin_arm64
	cp $(DIST_DIR)/$(BINARY)_linux_amd64         $(NPM_BIN)/$(BINARY)_linux_amd64
	cp $(DIST_DIR)/$(BINARY)_linux_arm64         $(NPM_BIN)/$(BINARY)_linux_arm64
	cp $(DIST_DIR)/$(BINARY)_linux_arm           $(NPM_BIN)/$(BINARY)_linux_arm
	cp $(DIST_DIR)/$(BINARY)_freebsd_amd64       $(NPM_BIN)/$(BINARY)_freebsd_amd64
	cp $(DIST_DIR)/$(BINARY)_windows_amd64.exe   $(NPM_BIN)/$(BINARY)_windows_amd64.exe
	cp $(DIST_DIR)/$(BINARY)_windows_arm64.exe   $(NPM_BIN)/$(BINARY)_windows_arm64.exe

clean:
	rm -f $(DIST_DIR)/$(BINARY)_darwin_amd64     $(DIST_DIR)/$(BINARY)_darwin_amd64.tar.gz
	rm -f $(DIST_DIR)/$(BINARY)_darwin_arm64     $(DIST_DIR)/$(BINARY)_darwin_arm64.tar.gz
	rm -f $(DIST_DIR)/$(BINARY)_linux_amd64      $(DIST_DIR)/$(BINARY)_linux_amd64.tar.gz
	rm -f $(DIST_DIR)/$(BINARY)_linux_arm64      $(DIST_DIR)/$(BINARY)_linux_arm64.tar.gz
	rm -f $(DIST_DIR)/$(BINARY)_linux_arm        $(DIST_DIR)/$(BINARY)_linux_arm.tar.gz
	rm -f $(DIST_DIR)/$(BINARY)_freebsd_amd64    $(DIST_DIR)/$(BINARY)_freebsd_amd64.tar.gz
	rm -f $(DIST_DIR)/$(BINARY)_windows_amd64.exe $(DIST_DIR)/$(BINARY)_windows_amd64.zip
	rm -f $(DIST_DIR)/$(BINARY)_windows_arm64.exe $(DIST_DIR)/$(BINARY)_windows_arm64.zip
