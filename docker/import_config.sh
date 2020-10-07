#!/bin/bash
# Container runtime configuration script
# Gets secrets config file from SSM parameter store and uses Deco to substitute parameter values
# This script expects SSMPATH env variable with the full SSMPATH path to the encrypted config file

if [ -n "$SSMPATH" ]; then
  echo "Getting config file from SSM Parameter Store (${SSMPATH}) ..."
  deco version
  if [[ $? -ne 0 ]]; then
    echo "ERROR: deco not found!"
    exit 1
  fi
  deco validate -e ssm://${SSMPATH} || exit 1
  deco run -e ssm://${SSMPATH}
else
  echo "ERROR: SSMPATH variable not set!"
  exit 1
fi
