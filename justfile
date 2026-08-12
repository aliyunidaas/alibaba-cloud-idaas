_:
  @just --list

alias b:=build
alias c:=clean

# clean out/*
clean:
  rm out/*; echo 'out/* cleaned'

# run tests
test:
  go test ./...

# build alibaba-cloud-idaas cli
build:
  go build

# build alibaba-cloud-idaas cli without features PKCS#11 & PIV
build-without-features:
  go build -tags disable_pkcs11,disable_yubikey_piv

# try build alibaba-cloud-idaas cli on all platforms
try-build-all:
  CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -tags disable_pkcs11,disable_yubikey_piv -ldflags "-X 'github.com/aliyunidaas/alibaba-cloud-idaas/constants.Version=v.test'" -o out/alibaba-cloud-idaas-darwin-amd64 main.go
  CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -tags disable_pkcs11,disable_yubikey_piv -ldflags "-X 'github.com/aliyunidaas/alibaba-cloud-idaas/constants.Version=v.test'" -o out/alibaba-cloud-idaas-darwin-arm64 main.go
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags disable_pkcs11,disable_yubikey_piv -ldflags "-X 'github.com/aliyunidaas/alibaba-cloud-idaas/constants.Version=v.test'" -o out/alibaba-cloud-idaas-linux-amd64 main.go
  CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags disable_pkcs11,disable_yubikey_piv -ldflags "-X 'github.com/aliyunidaas/alibaba-cloud-idaas/constants.Version=v.test'" -o out/alibaba-cloud-idaas-linux-arm64 main.go
  CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags disable_pkcs11,disable_yubikey_piv -ldflags "-X 'github.com/aliyunidaas/alibaba-cloud-idaas/constants.Version=v.test'" -o out/alibaba-cloud-idaas-windows-amd64.exe main.go
  go build
