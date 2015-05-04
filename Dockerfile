FROM alpine
MAINTAINER support@tutum.co

ENV VERSION 0.10.0

RUN ["apk", "add", "--update","go", "git" ,"ethtool", "conntrack-tools", "curl", "iptables", "iproute2", "util-linux", "python", "py-pip"]
RUN curl -sSLo weave https://github.com/weaveworks/weave/releases/download/v$VERSION/weave && \
    chmod +x weave

RUN mkdir -p /go/src /go/bin && chmod -R 777 /go
ENV GOPATH /go
ENV PATH /go/bin:$PATH
RUN go get github.com/tutumcloud/go-tutum/tutum && go get github.com/fsouza/go-dockerclient

WORKDIR /go
ADD . /go/src/github.com/tutumcloud/weave-daemon

ENV WEAVE_LAUNCH **None**

ENTRYPOINT ["/go/src/github.com/tutumcloud/weave-daemon/run.sh"]
