container_runtime := $(shell which podman || which docker)

$(info using ${container_runtime})

up:
	${container_runtime} compose up --build -d

down:
	${container_runtime} compose down

clean:
	${container_runtime} compose down -v

run-tests: 
	${container_runtime} run --rm --network=host tests:latest

load-test:
	make clean
	make up
	${container_runtime} compose up --build load_test
	make clean

test:
	make clean
	make up
	@echo wait cluster to start && sleep 10
	make run-tests
	make clean
	@echo "test finished"

lint:
	make -C service lint
