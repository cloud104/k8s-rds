#!/usr/bin/env sh

docker build -t "cloud104/kube-db:$TRAVIS_BUILD_NUMBER" .
docker login -u "$DOCKER_USERNAME" -p "$DOCKER_PASSWORD";
docker tag "cloud104/kube-db:$TRAVIS_BUILD_NUMBER" cloud104/kube-db:latest
docker push cloud104/kube-db:latest
docker push "cloud104/kube-db:$TRAVIS_BUILD_NUMBER"
