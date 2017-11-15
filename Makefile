PACKAGE  = link-motion.com/lm-toolchain-sdk-tools

GOPATH   = $(CURDIR)/.gopath
BIN      = $(GOPATH)/bin
BASE     = $(GOPATH)/src/$(PACKAGE)

GO       = env GOPATH=$(GOPATH) go


all:	$(BASE) $(BIN)/lmsdk-target $(BIN)/lmsdk-wrapper $(BIN)/lmsdk-download $(BIN)/lxc-lm-download
clean:	
	rm -rf $(GOPATH)/pkg $(GOPATH)/bin

$(BASE): ; $(info setting GOPATH…)
	@mkdir -p $(dir $@)
	@ln -sf $(CURDIR) $@

$(CURDIR)/.gopath/patched:
	cd $(GOPATH) && $(GO) get -d link-motion.com/lm-toolchain-sdk-tools/lmsdk-target
	cd $(GOPATH) && $(GO) get -d link-motion.com/lm-toolchain-sdk-tools/lmsdk-download
	cd $(GOPATH) && $(GO) get -d link-motion.com/lm-toolchain-sdk-tools/lmsdk-wrapper
	cd $(GOPATH)/src && patch -p1 -i $(BASE)/patches/lxc.patch
	touch $(CURDIR)/.gopath/patched

$(BIN)/lmsdk-target: $(CURDIR)/.gopath/patched
	cd $(GOPATH) && $(GO) get -d link-motion.com/lm-toolchain-sdk-tools/lmsdk-target && $(GO) install link-motion.com/lm-toolchain-sdk-tools/lmsdk-target

$(BIN)/lmsdk-wrapper: $(CURDIR)/.gopath/patched
	cd $(GOPATH) && $(GO) get -d link-motion.com/lm-toolchain-sdk-tools/lmsdk-wrapper && $(GO) install link-motion.com/lm-toolchain-sdk-tools/lmsdk-wrapper

$(BIN)/lmsdk-download: $(CURDIR)/.gopath/patched
	 cd $(GOPATH) && $(GO) get -d link-motion.com/lm-toolchain-sdk-tools/lmsdk-download && $(GO) install link-motion.com/lm-toolchain-sdk-tools/lmsdk-download

$(BIN)/lxc-lm-download: $(CURDIR)/.gopath/patched
	 cp $(CURDIR)/share/lxc-lm-download $(BIN)



