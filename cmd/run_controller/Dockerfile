ARG GOLANG_VERSION
FROM golang:${GOLANG_VERSION}-alpine as builder
RUN mkdir /build
ADD . /build/
WORKDIR /build
RUN apk add --no-cache git
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly -a -installsuffix cgo -ldflags '-extldflags "-static"' -o steward-runctl -v ./cmd/run_controller
RUN mkdir -p /result/app/
RUN mkdir -p /result/tmp/
RUN cp /build/steward-runctl /result/app/


FROM scratch
COPY --from=builder /result/ /
WORKDIR /app
CMD ["./steward-runctl"]
