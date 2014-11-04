from ubuntu:14.10
maintainer Matt Sparks <ms@quadpoint.org>

run apt-get update
run apt-get install -y build-essential
run apt-get install -y git
run apt-get install -y mercurial
run apt-get install -y protobuf-compiler
run apt-get install -y ca-certificates
run apt-get install -y golang

run mkdir -p /usr/local/go
env GOPATH /usr/local/go
env PATH $GOPATH/bin:$PATH

add . /usr/local/go/src/github.com/msparks/iq

workdir /usr/local/go/src/github.com/msparks/iq
run go get -d ./...

# protoc-gen-gogo isn't imported by anything, so we have to get it separately.
run go get code.google.com/p/gogoprotobuf/protoc-gen-gogo

run make
cmd ["./iq"]
