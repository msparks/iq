SUBDIRS = \
	irc
TARGETS=\
	events.pb.go

all: $(SUBDIRS) $(TARGETS)
.PHONY: all

irc:
	$(MAKE) -C $@
.PHONY: irc

events.pb.go: events.proto
	protoc --gogo_out=. --proto_path=$(GOPATH)/src --proto_path=. $^

clean:
	for dir in $(SUBDIRS); do \
	  $(MAKE) -C $$dir $@; \
	done
	rm -f $(TARGETS)
.PHONY: clean
