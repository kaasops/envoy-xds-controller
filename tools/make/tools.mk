tools.bindir = tools/bin
tools.srcdir = tools/src


.PHONY: prepare-kind
prepare-kind: 
	cd ${tools.srcdir}/kind && go build -o ../../bin/kind $$(sed -En 's,^import _ "(.*)".*,\1,p' pin.go) 


.PHONY: cleanup-tools
cleanup-tools: 
	rm -rf ${tools.bindir}