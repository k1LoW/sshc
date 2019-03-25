FROM alpine
RUN apk add --update openssh && rm -rf /tmp/* /var/cache/apk/*
ADD tmp/test_rsa.pub /root/.ssh/authorized_keys
ADD docker-entrypoint.sh /

EXPOSE 22
ENTRYPOINT ["/docker-entrypoint.sh"]
