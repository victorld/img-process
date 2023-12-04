FROM golang:1.20.11-alpine3.18

COPY ./ /app/img_process/

WORKDIR /app/img_process/

RUN ls

EXPOSE 8081

RUN chmod +x ./docker/entrypoint.sh

#ENTRYPOINT ["go", "run" ,"main/webserver.go"]
ENTRYPOINT ["./docker/entrypoint.sh"]