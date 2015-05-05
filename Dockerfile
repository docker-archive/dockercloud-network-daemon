FROM progrium/busybox
ADD weave-daemon /
ADD weave /
ADD run.sh /
ENTRYPOINT ["/run.sh"]
