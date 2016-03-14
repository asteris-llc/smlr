#!/usr/bin/env bash
if [ ! -f glide/glide ]; then
    [ ! -d glide ] && mkdir glide
    curl -L https://github.com/Masterminds/glide/releases/download/0.9.3/glide-0.9.3-linux-amd64.tar.gz | tar -xzv --strip-components 1 -C glide
fi

export PATH=glide:$PATH
