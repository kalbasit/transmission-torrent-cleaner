#!/bin/bash
set -e

COMMIT=${TRAVIS_COMMIT::8}
ROCKER_VERSION=1.1.2
curl -L https://github.com/grammarly/rocker/releases/download/${ROCKER_VERSION}/rocker-${ROCKER_VERSION}-linux_amd64.tar.gz | tar xfz - -C $HOME
$HOME/rocker build --var BRANCH=$TRAVIS_BRANCH --var TAG=$TRAVIS_TAG --var COMMIT=$COMMIT --var TRAVIS_BUILD_NUMBER=$TRAVIS_BUILD_NUMBER --push --auth $DOCKER_USER:$DOCKER_PASS
