# @Copyright    VSETH - Verband der Studierenden an der ETH
# @Author       Maximilian Falkenstein <maximilian.falkenstein@vseth.ethz.ch>
#
# Build microkubed inside a container

FROM golang:stretch
RUN apt update && apt install -y unzip
RUN mkdir -p /go/src/github.com/uubk
COPY . /go/src/github.com/uubk/microkube/
WORKDIR /go
ADD .glide.sha256sum /go/
RUN wget https://github.com/Masterminds/glide/releases/download/v0.13.1/glide-v0.13.1-linux-amd64.zip && \
  sha256sum -c .glide.sha256sum && \
  unzip glide-v0.13.1-linux-amd64.zip && \
  mv linux-amd64/glide bin/ && chmod +x bin/glide
WORKDIR /go/src/github.com/uubk/microkube
RUN glide i && cd /go && go build github.com/uubk/microkube/cmd/microkubed
CMD cp /go/microkubed /target/microkubed && chmod uga+rwx /target/microkubed
