FROM golang:1.19.0

RUN mkdir /app

COPY snippetApp /app

CMD ["/app/snippetApp"]