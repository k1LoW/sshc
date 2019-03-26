#!/bin/bash
docker-compose up -d --build
cd ./test
go run e2e.go
