all:
	make run

build:
	docker build -t mygo .

run:
	docker run -it --rm --name mygocontainer -p 8088:8088 -v ${PWD}:/go/src/app mygo

dockerbuild:
	docker run --rm -v ${PWD}:/usr/src/myapp -w /usr/src/myapp -e GOOS=darwin golang:1.8 go run -v

dockerrun:
	go get -v ./...
	go install -v ./...
	app

clientrun:
	./myapp
