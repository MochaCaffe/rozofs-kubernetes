IMAGE?=mochacoffee/rozofs-provision

all: provisioner-file image clean
provisioner-file:
	CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o rozofs .
image: provisioner-file
	docker build -t $(IMAGE) .
clean:
	rm -rf rozofs
