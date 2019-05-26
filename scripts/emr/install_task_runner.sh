#!/bin/bash

# Make everything verbose and exit on command failure
set -e -x

# Download the TaskRunner
aws s3 cp "s3://datapipeline-us-east-1/us-east-1/software/latest/TaskRunner/TaskRunner-1.0.jar" "${HOME}/TaskRunner-1.0.jar"

# Create a configuration file with the credentials
echo "{\"access-id\": \"${1}\", \"private-key\": \"${2}\"}" | jq -cr '.' > "${HOME}/credentials.json"

# Start the Task runner in the background
nohup java -jar "${HOME}/TaskRunner-1.0.jar" --config="${HOME}/credentials.json" --workerGroup="${3}" --region="${4}" --logUri="${5}" &
