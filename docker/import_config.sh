#!/bin/bash
# Container runtime configuration script
# Gets secrets config file from S3 and uses Deco to substitute parameter values
# This script expects S3URL env variable with the full S3 path to the encrypted config file

if [ -n "$S3URL" ]; then
  echo "Getting config file from S3 (${S3URL}) ..."
  aws --version
  if [[ $? -ne 0 ]]; then
    echo "ERROR: aws-cli not found!"
    exit 1
  fi
  aws --region us-east-1 s3 cp ${S3URL} ./config.encrypted
  aws --region us-east-1 kms decrypt --ciphertext-blob fileb://config.encrypted --output text --query Plaintext | base64 -d > deco-config.json
  deco validate deco-config.json || exit 1
  deco run deco-config.json
  rm -f deco-config.json config.encrypted
else
  echo "ERROR: S3URL variable not set!"
  exit 1
fi
