TARGETS=\
	irc.pb.go

all: $(TARGETS)
.PHONY: all

clean:
	rm -f $(TARGETS)
.PHONY: clean

irc.pb.go: irc.proto
	protoc --gogo_out=. $^
