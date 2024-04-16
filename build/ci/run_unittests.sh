docker build -f build/ci/Dockerfile.unittest -t volume-group-operator-unit-tests .
docker run --rm -t volume-group-operator-unit-tests
