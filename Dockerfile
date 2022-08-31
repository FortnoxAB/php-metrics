FROM gcr.io/distroless/static-debian11:nonroot
COPY php-metrics /
USER nonroot
ENTRYPOINT ["/php-metrics"]
