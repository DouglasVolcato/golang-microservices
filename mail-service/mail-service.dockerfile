FROM alpine:latest

RUN mkdir /app

COPY mailerApp /app
COPY mailerApp /templates

CMD [ "/app/mailerApp" ]