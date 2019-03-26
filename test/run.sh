#!/bin/bash
chmod 600 ./test/id_rsa
docker-compose up -d --build
cd ./test
go run e2e.go
