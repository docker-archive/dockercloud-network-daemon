FROM tutum/curl:trusty
MAINTAINER Feng Honglin <hfeng@tutum.co>

RUN apt-get update && \
    apt-get install -y --no-install-recommends iptables python-pip && \
    curl -Lo weave https://github.com/zettio/weave/releases/download/v0.9.0/weave && \
    chmod +x weave && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

ADD requirements.txt /app/requirements.txt
RUN pip install -r /app/requirements.txt
ADD . /app
RUN chmod +x /app/run.sh

ENV WEAVE_LAUNCH **None**
ENV VERSION git-b76e97ac2426

ENTRYPOINT ["/app/run.sh"]
