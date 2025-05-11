#!/bin/bash

set -e

cd utilbins/linker || true
go build -o linker &&
    (sudo rm /usr/local/go/bin/linker || true) &&
    sudo mv linker /usr/local/go/bin/
cd ../.. || true

cd utilbins/setter || true
go build -o setter &&
    (sudo rm /usr/local/go/bin/setter || true) &&
    sudo mv setter /usr/local/go/bin/
cd ../.. || true

cd utilbins/finder || true
go build -o finder &&
    (sudo rm /usr/local/go/bin/finder || true) &&
    sudo mv finder /usr/local/go/bin/
cd ../.. || true

go build -o tasker &&
    (sudo rm /usr/local/go/bin/tasker || true) &&
    sudo mv tasker /usr/local/go/bin/
