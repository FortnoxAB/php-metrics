FROM scratch
COPY php-metrics /
ENTRYPOINT ["/php-metrics"]
