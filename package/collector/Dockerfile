FROM --platform=$BUILDPLATFORM registry.suse.com/bci/golang:1.23 AS builder
WORKDIR /usr/src/app
COPY go.mod go.sum  ./
RUN go mod download
COPY . .
RUN go build -o ./bin/collector  ./cmd/collector/main.go

FROM registry.suse.com/bci/bci-micro:15.6
RUN echo 'pprofcollector:x:1000:1000::/home/pprofcollector:/bin/bash' >> /etc/passwd && \
    echo 'pprofcollector:x:1000:' >> /etc/group && \
    mkdir /home/pprofcollector && \
    chown -R pprofcollector:pprofcollector /home/pprofcollector
COPY --from=builder /usr/src/app/bin/collector /usr/bin
USER pprofcollector
ENTRYPOINT ["collector"]