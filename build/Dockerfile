FROM ubuntu:latest
LABEL authors="299m"

## Export the environment variable for the path to the certs
ENV REMOTE_CERT=./certs/combined-certs.pem
ENV REMOTE_KEY=./certs/combined-key.pem

### Copy the config config into the docker image
COPY ./remote /remote

ENTRYPOINT ["hdnprxy", "--config", "./remote/config"]
