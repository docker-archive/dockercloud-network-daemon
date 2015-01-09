FROM tutum/curl:trusty
MAINTAINER Feng Honglin <hfeng@tutum.co>

RUN apt-get update && \
    apt-get install -y --no-install-recommends iptables python-pip && \
    pip install docker-py==0.5.3 && \ 
    # Change to "https://github.com/zettio/weave/releases/download/latest_release/weave" once "https://github.com/zettio/weave/commit/cce5d2417a75c096546fb3bfbfd975d0a3a24723 is released"
    curl -o weave https://raw.githubusercontent.com/zettio/weave/master/weave && \
    chmod +x weave && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

ADD run.sh /run.sh
ADD monitor.py /monitor.py
RUN chmod +x /run.sh

ENV WEAVE_LAUNCH **None**

CMD ["/run.sh"]
