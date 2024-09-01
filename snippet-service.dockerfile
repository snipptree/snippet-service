FROM alpine:latest

RUN mkdir /app

COPY snippetApp /app

CMD ["/app/snippetApp"]