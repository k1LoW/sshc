#!/usr/bin/env bash

set -e

cp /keys/id_rsa.pub /root/.ssh/authorized_keys
cp /keys/id_rsa.pub /etc/authorized_keys/k1low
chmod 600 /root/.ssh/authorized_keys
chmod 600 /etc/authorized_keys/k1low
chown root: /root/.ssh/authorized_keys
chown k1low: /etc/authorized_keys/k1low
ls -la /root/.ssh
ls -la /etc/authorized_keys
