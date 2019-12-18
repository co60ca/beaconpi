# GNU Makefile for beaconpi

PACKAGE = github.com/co60ca/beaconpi
SERVERFLAGS := $(SERVERFLAGS)
CLIENTFLAGS = 
SERVERENV = #CGO=0
CLIENTENV = GOARCH=arm GOOS=linux #CGO=0

ALLGO = *.go

.PHONY: all
all: reqs build/beaconserv build/metricsserv # build/beaconclient

.PHONY: clean
clean:
	rm -rf build

# Using go get to fetch any prequisite libraries
.PHONY: reqs
reqs:
	@go get .

build/beaconserv: $(ALLGO)
	$(SERVERENV) \
	go build -o $@ $(SERVERFLAGS) $(PACKAGE)/cmd/beaconserver
build/beaconclient: $(ALLGO)
	$(CLIENTENV) \
	go build -o $@ $(CLIENTFLAGS) $(PACKAGE)/cmd/beaconclient
build/metricsserv: $(ALLGO)
	$(SERVERENV) \
	# metrics flags includes metrics only files
	go build -o $@ $(SERVERFLAGS) --tags=metrics $(PACKAGE)/cmd/metricsserver
