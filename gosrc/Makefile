# Makefile for bzwikipedia

# Go compiler and linker.
GO_CC = 6g
GO_LD = 6l
GO_SUFFIX = 6

GO_MAIN  = main.go
GO_FILES = confparse.go

PROG    = bzwikipedia
GOFLAGS = -I . -I build
GOLDFLAGS = -L build

all: $(PROG)

include Makefile.depend

OBJ_FILES = $(patsubst %.go, build/%.$(GO_SUFFIX), $(GO_FILES) $(GO_MAIN))

$(PROG): $(OBJ_FILES)
	$(GO_LD) $(GOLDFLAGS) -o $(PROG) $(patsubst %.go, build/%.$(GO_SUFFIX), $(GO_MAIN))

build/%.$(GO_SUFFIX): %.go
	@mkdir -p build
	$(GO_CC) -c $(GOFLAGS) -o $(patsubst %.go,build/%.$(GO_SUFFIX),$<) $<

clean:
	-rm $(OBJ_FILES) $(PROG)
