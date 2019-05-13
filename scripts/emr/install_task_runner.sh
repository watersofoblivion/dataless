#!/bin/bash

set -e -x

aws s3 cp "s3://datapipeline-us-east-1/us-east-1/software/latest/TaskRunner/TaskRunner-1.0.jar" "${HOME}/TaskRunner-1.0.jar"
nohup java -jar "${HOME}/TaskRunner-1.0.jar" --workerGroup="${1}" --region="${2}" --logUri="${3}" &
