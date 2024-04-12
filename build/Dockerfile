FROM ubuntu:23.10
LABEL authors="299m"

### Copy the config config into the docker image
COPY ./remote /remote
COPY ./bin /bin


ENTRYPOINT ["./bin/hdnprxy", "--config", "./remote/config"]
