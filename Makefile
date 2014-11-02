SUBDIRS = \
	public
TARGETS=\
	iq

all: $(SUBDIRS) $(TARGETS)
.PHONY: all

public:
	$(MAKE) -C $@
.PHONY: public

iq: *.go ircconnection/*.go ircsession/*.go notify/*.go
	go build -o $@

install: all
	go install
.PHONY: install

clean:
	for dir in $(SUBDIRS); do \
	  $(MAKE) -C $$dir $@; \
	done
	rm -f $(TARGETS)
.PHONY: clean
