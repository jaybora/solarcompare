GOPATH=`pwd` CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -o dpunit_linux_386 src/dpunit/dpunit.go
GOPATH=`pwd` go build -o dpunit_darwin src/dpunit/dpunit.go
