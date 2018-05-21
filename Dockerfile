ARG GOLANG_VERSION
FROM golang:${GOLANG_VERSION}-alpine3.7 AS build

ARG ROCKSDB_VERSION
RUN apk add --no-cache build-base git ca-certificates bash perl curl linux-headers
RUN git clone --depth 1 --single-branch --branch v${ROCKSDB_VERSION} \
    https://github.com/facebook/rocksdb /rocksdb
WORKDIR /rocksdb
RUN DEBUG_LEVEL=0 PORTABLE=1 make libsnappy.a
RUN PORTABLE=1 make static_lib
RUN strip -g librocksdb.a

ENV ROCKSDB_HOME /rocksdb
ENV CGO_CFLAGS -I${ROCKSDB_HOME}/include
ENV CGO_LDFLAGS -L${ROCKSDB_HOME} -lrocksdb
ENV GOPATH /root/go
ENV PATH ${PATH}:/root/go/bin

RUN mkdir -p /root/go/src/github.com/contribsys
ADD . /root/go/src/github.com/lucaslsl/faktory
WORKDIR /root/go/src/github.com/lucaslsl/faktory
RUN make prepare && make test && make build

FROM alpine:3.7
COPY --from=build /root/go/src/github.com/lucaslsl/faktory/faktory \
                  /root/go/src/github.com/lucaslsl/faktory/faktory-cli \
                  /
RUN apk add --no-cache libstdc++ libgcc
RUN mkdir -p /root/.faktory/db
RUN mkdir -p /var/lib/faktory/db
RUN mkdir -p /etc/faktory

EXPOSE 7419 7420
CMD ["/faktory", "-b", "0.0.0.0:7419", "-e", "development"]
