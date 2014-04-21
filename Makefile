# This file exists purely for convenience and will check for a
# installed GTK that is compatible with gotk3, calling either
# go build or go get with the correct build tags.

VALID_MAJOR=	3
VALID_MINORS=	6 8 10 12
GTK_VERSION!=	pkg-config --modversion gtk+-3.0
GTK_MAJOR!=	echo ${GTK_VERSION} | cut -f 1 -d '.'
GTK_MINOR!=	echo ${GTK_VERSION} | cut -f 2 -d '.'
GTK_TAG=	gtk_${GTK_MAJOR}_${GTK_MINOR}

all: install

check-gtk:
.if ${GTK_VERSION} == ""
	@echo "Cannot find GTK."
	@exit 1
.endif
.if ${GTK_MAJOR} != ${VALID_MAJOR}
	@echo "Unsupported GTK major version."
	@exit 1
.endif
.for _x in ${GTK_MINOR}
.if !${VALID_MINORS:M${_x}}
	@echo "Unsupported GTK minor version."
	@exit 1
.endif
.endfor

build: check-gtk
	go build -tags=${GTK_TAG} ./...

install: check-gtk
	go get -tags=${GTK_TAG} ./...

update: check-gtk
	go get -u -tags=${GTK_TAG} ./...
