all:
	support/compile linux
	docker build . -t victorgama/ponos:dev
	docker push victorgama/ponos:dev
