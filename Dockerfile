FROM tutum/curl:trusty
MAINTAINER Feng Honglin <hfeng@tutum.co>

ENV VERSION 0.10.0

RUN apt-get update && \
    apt-get install -y --no-install-recommends iptables python-pip && \
    curl -Lo weave https://github.com/weaveworks/weave/releases/download/v$VERSION/weave && \
    chmod +x weave && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

ADD requirements.txt /app/requirements.txt
RUN pip install -r /app/requirements.txt
ADD . /app
RUN chmod +x /app/run.sh

ENV WEAVE_LAUNCH **None**

ENTRYPOINT ["/app/run.sh"]
