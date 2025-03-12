FROM --platform=$BUILDPLATFORM registry.suse.com/bci/golang:1.23 AS builder
WORKDIR /usr/src/app
COPY go.mod  ./
RUN go mod download
COPY . .
RUN go build -o ./chaos-tool ./internal/cmd/chaos/main.go

FROM registry.suse.com/bci/bci-micro:15.6
RUN echo 'chaos:x:1000:1000::/home/example:/bin/bash' >> /etc/passwd && \
    echo 'chaos:x:1000:' >> /etc/group && \
    mkdir /home/chaos && \
    chown -R chaos:chaos /home/chaos
COPY --from=builder /usr/src/app/chaos-tool /usr/bin
EXPOSE 6060
USER chaos
CMD ["chaos-tool"]