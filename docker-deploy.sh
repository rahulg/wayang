#!/bin/bash
BASE_DIR=$(realpath "$(dirname $0)")
docker run -d -p 8001:8000 -v "$BASE_DIR/bin:/opt/deploy:rw" rahulg/wayang
