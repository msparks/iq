SUBDIRS = \
	public
TARGETS=\
	iq

all: $(SUBDIRS) $(TARGETS)
.PHONY: all

public:
	$(MAKE) -C $@
.PHONY: public

iq: iq.go eventserver.go clients.go cmdserver.go netconn.go translate.go
	go build $^

clean:
	for dir in $(SUBDIRS); do \
	  $(MAKE) -C $$dir $@; \
	done
	rm -f $(TARGETS)
.PHONY: clean
