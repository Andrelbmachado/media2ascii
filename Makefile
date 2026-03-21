BINARY   = media2ascii
VERSION  = v1.0.0
DIST_DIR = dist
NPM_BIN  = npm/bin

.PHONY: all darwin-amd64 darwin-arm64 windows-amd64 clean npm-copy

all: darwin-amd64 darwin-arm64 windows-amd64

darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build -o $(DIST_DIR)/$(BINARY)_darwin_amd64 .
	cd $(DIST_DIR) && tar -czf $(BINARY)_darwin_amd64.tar.gz $(BINARY)_darwin_amd64

darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -o $(DIST_DIR)/$(BINARY)_darwin_arm64 .
	cd $(DIST_DIR) && tar -czf $(BINARY)_darwin_arm64.tar.gz $(BINARY)_darwin_arm64

windows-amd64:
	GOOS=windows GOARCH=amd64 go build -o $(DIST_DIR)/$(BINARY)_windows_amd64.exe .
	cd $(DIST_DIR) && zip $(BINARY)_windows_amd64.zip $(BINARY)_windows_amd64.exe

npm-copy: all
	cp $(DIST_DIR)/$(BINARY)_darwin_amd64  $(NPM_BIN)/$(BINARY)_darwin_amd64
	cp $(DIST_DIR)/$(BINARY)_darwin_arm64  $(NPM_BIN)/$(BINARY)_darwin_arm64
	cp $(DIST_DIR)/$(BINARY)_windows_amd64.exe $(NPM_BIN)/$(BINARY)_windows_amd64.exe

clean:
	rm -f $(DIST_DIR)/$(BINARY)_darwin_amd64   $(DIST_DIR)/$(BINARY)_darwin_amd64.tar.gz
	rm -f $(DIST_DIR)/$(BINARY)_darwin_arm64   $(DIST_DIR)/$(BINARY)_darwin_arm64.tar.gz
	rm -f $(DIST_DIR)/$(BINARY)_windows_amd64.exe $(DIST_DIR)/$(BINARY)_windows_amd64.zip
	rm -f $(NPM_BIN)/$(BINARY)_windows_amd64.exe
