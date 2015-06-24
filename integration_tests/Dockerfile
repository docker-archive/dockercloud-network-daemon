FROM jpetazzo/dind:latest
MAINTAINER support@tutum.co

ADD test.sh /
RUN chmod +x /test.sh

ENTRYPOINT ["/test.sh"]
