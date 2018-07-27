FROM alpine
LABEL maintainer="Victor Gama <hey@vito.io>"

WORKDIR /ponos

COPY ./bin/ponos /ponos/ponos

CMD ["/ponos/ponos"]
