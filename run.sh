export GOPATH=$(pwd)

go install network
go install driver

go run main.go
