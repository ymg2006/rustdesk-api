FROM alpine

ARG BUILDARCH
WORKDIR /app
RUN apk add --no-cache tzdata
COPY ./${BUILDARCH}/release /app/
RUN chmod 0755 /app/apimain && \
    mkdir -p /app/runtime && \
    chmod 0755 /app/runtime

VOLUME /app/data

EXPOSE 21114
CMD ["./apimain"]
