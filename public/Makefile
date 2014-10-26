TARGETS=\
	irc \
	events.pb.go

all: $(TARGETS)
.PHONY: all

irc:
	$(MAKE) -C $@
.PHONY: irc

events.pb.go: events.proto
	protoc --go_out=. $^
