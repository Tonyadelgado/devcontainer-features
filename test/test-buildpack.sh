#!/bin/bash
set -e
cd "$(dirname "${BASH_SOURCE[0]}")"/..

./stack/build-stack.sh
./builders/build-bulders.sh
