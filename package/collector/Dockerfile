FROM --platform=$BUILDPLATFORM registry.suse.com/bci/golang:1.23 AS builder
RUN zypper install -y graphviz
WORKDIR /usr/src/app
COPY go.mod go.sum  ./
RUN go mod download
COPY . .
RUN go build -o ./bin/collector  ./cmd/collector/main.go

FROM registry.suse.com/bci/bci-base:15.6
RUN echo 'pprofcollector:x:1000:1000::/home/pprofcollector:/bin/bash' >> /etc/passwd && \
    echo 'pprofcollector:x:1000:' >> /etc/group && \
    mkdir /home/pprofcollector && \
    chown -R pprofcollector:pprofcollector /home/pprofcollector
# COPY --from=builder /usr/bin/dot /usr/bin/dot
# COPY --from=builder /usr/lib64/graphviz /usr/lib64/graphviz
# COPY --from=builder /usr/share/graphviz /usr/share/graphviz
COPY --from=builder /usr/src/app/bin/collector /usr/bin
RUN zypper install -y graphviz
# RUN chmod +x /usr/bin/dot
# RUN ls /usr/bin/dot
# RUN which dot
# RUN head
USER pprofcollector
ENTRYPOINT ["collector"]