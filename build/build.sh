#!/bin/bash

# Create the local zip for download
mkdir local/bin
mkdir bin/
cp $(which hdnprxy) local/bin/
cp $(which hdnprxy) bin/

# Create the zip
zip -r local.zip local

# Copy the zip to the downloads dir
mv local.zip remote/http/dist/downloads

# Build the docker image
sudo docker build -t hdnprxy .
sudo docker tag hdnprxy:latest 299m/core:hdnprxy-1.0.0
sudo docker push 299m/core:hdnprxy-1.0.0

