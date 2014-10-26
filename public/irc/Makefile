TARGETS=\
	irc.pb.go

all: $(TARGETS)
.PHONY: all

irc.pb.go: irc.proto
	protoc --go_out=. $^
