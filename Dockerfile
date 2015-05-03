FROM alpine
MAINTAINER support@tutum.co

ENV VERSION 0.10.0

RUN ["apk", "add", "--update", "ethtool", "conntrack-tools", "curl", "iptables", "iproute2", "util-linux", "python", "py-pip"]
RUN curl -sSLo weave https://github.com/weaveworks/weave/releases/download/v$VERSION/weave && \
    chmod +x weave

ADD requirements.txt /app/requirements.txt
RUN pip install -r /app/requirements.txt
ADD . /app

ENV WEAVE_LAUNCH **None**

ENTRYPOINT ["/app/run.sh"]
