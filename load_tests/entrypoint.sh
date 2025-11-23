#!/bin/sh
go run seeder/main.go
bombardier -c 10 -d 30s --latencies --print r "http://pull_req:8080/users/getReview?user_id=u_1_1"
