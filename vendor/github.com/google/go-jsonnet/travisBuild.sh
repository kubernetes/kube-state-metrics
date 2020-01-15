#!/usr/bin/env bash

run_tests() {
  $GOPATH/bin/goveralls -service=travis-ci
  ./tests.sh --skip-go-test
}

release() {
  env VERSION=$TRAVIS_TAG ./release.sh
}

if [ "$TRAVIS_PULL_REQUEST" != "false" ]; then
  # Pull Requests.
  echo -e "Build Pull Request #$TRAVIS_PULL_REQUEST => Branch [$TRAVIS_BRANCH]"
  run_tests
elif [ "$TRAVIS_TAG" == "" ]; then
  # Pushed branches.
  echo -e "Build Branch $TRAVIS_BRANCH"
  run_tests
else
  # $TRAVIS_PULL_REQUEST == "false" and $TRAVIS_TAG != "" -> Releases.
  echo -e 'Build Branch for Release => Branch ['$TRAVIS_BRANCH']  Tag ['$TRAVIS_TAG']'
  release
fi

