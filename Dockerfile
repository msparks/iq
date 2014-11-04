from ubuntu:14.10
maintainer Matt Sparks <ms@quadpoint.org>

run apt-get update
run apt-get install -y curl
run apt-get install -y build-essential
run apt-get install -y git
run apt-get install -y mercurial
run apt-get install -y protobuf-compiler
run apt-get install -y ca-certificates

# Install Go. Ubuntu Utopic has Go 1.2; we need Go 1.3+.
run curl https://storage.googleapis.com/golang/go1.3.3.linux-amd64.tar.gz \
  | tar -C /usr/local -zx
env GOROOT /usr/local/go

run mkdir -p /usr/local/go/workspace
env GOPATH /usr/local/go/workspace
env PATH $GOROOT/bin:$GOPATH/bin:$PATH

add . /usr/local/go/workspace/src/github.com/msparks/iq

workdir /usr/local/go/workspace/src/github.com/msparks/iq
run go get -d ./...

# protoc-gen-gogo isn't imported by anything, so we have to get it separately.
run go get code.google.com/p/gogoprotobuf/protoc-gen-gogo

run make
cmd ["./iq"]
