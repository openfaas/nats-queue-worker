#!/bin/sh
cp -r ../gateway .

docker build --build-arg http_proxy=$http_proxy -t functions/queue-worker:latest-dev .

