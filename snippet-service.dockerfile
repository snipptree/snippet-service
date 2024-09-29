FROM alpine:latest
WORKDIR /app
COPY snippetApp /app/snippetApp
EXPOSE 80
CMD ["./snippetApp"]
