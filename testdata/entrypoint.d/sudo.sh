#!/usr/bin/env bash

set -e

apk add sudo
echo "k1low:k1low" | chpasswd
echo "k1low ALL=(ALL) ALL" >> /etc/sudoers
