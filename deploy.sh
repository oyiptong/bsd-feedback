#!/bin/bash
set -e
set -x

gcloud app deploy $1
