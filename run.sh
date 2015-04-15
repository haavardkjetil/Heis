export GOPATH=$(pwd)

go install network
go install driver
go install queueManager
go install stateMachine

go run main.go
