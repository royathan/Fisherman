# Makefile

ICON = Icon.png

all: clean get package

clean:
	rm -rf Fisherman.app

get:
	go get fyne.io/fyne/v2/cmd/fyne

package:
	fyne package -os darwin -icon $(ICON)
