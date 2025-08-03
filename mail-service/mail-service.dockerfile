FROM alpine:latest

RUN mkdir /app

COPY mailerApp /app/mailerApp
COPY templates /app/templates

WORKDIR /app

CMD [ "./mailerApp" ]
