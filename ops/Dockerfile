# build kelp from source
FROM golang:1.27 AS build
# build a static binary so it does not depend on the glibc version of the runtime image
ENV CGO_ENABLED=0
WORKDIR /src/kelp
COPY . .
RUN ./scripts/build.sh

FROM ubuntu:20.04

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /kelp
COPY --from=build /src/kelp/bin/kelp ./kelp
COPY --from=build /src/kelp/examples/configs/trader/ ./

# set ulimit
RUN ulimit -n 10000

# use command line arguments from invocation of docker run against this ENTRYPOINT command - https://stackoverflow.com/a/40312311/1484710
ENTRYPOINT ["./kelp"]
# default arguments are empty
CMD [""]

# sample commands to build and run this container (from the repository root):
#     docker build -f ops/Dockerfile -t kelp .
#     docker run kelp trade -c sample_trader.cfg -s buysell -f sample_buysell.cfg --sim
