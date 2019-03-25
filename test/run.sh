#!/bin/bash

if [ ! -f "./test/test_rsa" ]; then
  ssh-keygen -P '' -f ./test/test_rsa
fi
docker-compose up -d --build
cd ./test
go run e2e.go
