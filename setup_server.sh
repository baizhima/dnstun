#!/bin/bash

echo "server network interface config"

sudo openvpn --mktun --dev tun66
sudo ip link set tun66 up
sudo ip addr add 192.168.3.1/24 dev tun66
ip route show

