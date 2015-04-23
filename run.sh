#!/bin/sh
export GOPATH=$(pwd)

go install network
go install driver
go install queueManager
go install stateMachine

#xterm -hold -e go run main.go&
go run main.go
