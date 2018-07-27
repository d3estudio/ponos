all:
	support/compile linux
	docker build . -t victorgama/ponos:latest
	docker push victorgama/ponos:latest
