FROM golang:1.17 AS build

WORKDIR /opt/src
COPY . .
WORKDIR /opt/src/go

RUN go get -d -v ./...
RUN go build -o quickstart

FROM gcr.io/distroless/base-debian10

COPY --from=build /opt/src/go/quickstart /
COPY --from=build /opt/src/go/*.pem /
#COPY --from=build /opt/src/go/*.json /

EXPOSE 8000
ENTRYPOINT ["/quickstart"]
