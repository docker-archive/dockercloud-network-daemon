FROM alpine

ENV VERSION 0.10.0

RUN ["apk", "add", "--update", "ethtool", "conntrack-tools", "curl", "iptables", "iproute2", "util-linux"]
ADD weave-daemon /
ADD weave /
ADD run.sh /
ENTRYPOINT ["/run.sh"]
