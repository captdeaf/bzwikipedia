# Makefile for bzwikipedia

# Include go defaults
include $(GOROOT)/src/Make.inc

# Go compiler and linker.
GO_CC = $(GC)
GO_LD = $(LD)
GO_SUFFIX = $(O)

GO_MAIN  = main.go
GO_FILES = confparse.go bzreader.go loadfile.go

PROG    = bzwikipedia
GOFLAGS = -I . -I build
GOLDFLAGS = -L build

all: $(PROG)

include Makefile.depend

OBJ_FILES = $(patsubst %.go, build/%.$(GO_SUFFIX), $(GO_FILES) $(GO_MAIN))

$(PROG): $(OBJ_FILES)
	$(GO_LD) $(GOLDFLAGS) -o $(PROG) $(patsubst %.go, build/%.$(GO_SUFFIX), $(GO_MAIN))

build/loadfile.$(GO_SUFFIX): build/loadfile_$(GOOS).$(GO_SUFFIX)
	mv build/loadfile_$(GOOS).$(GO_SUFFIX) build/loadfile.$(GO_SUFFIX)

build/%.$(GO_SUFFIX): %.go
	@mkdir -p build
	$(GO_CC) -c $(GOFLAGS) -o $(patsubst %.go,build/%.$(GO_SUFFIX),$(patsubst %_.*.go,%.go,$<)) $<

clean:
	-rm $(OBJ_FILES) $(PROG)
