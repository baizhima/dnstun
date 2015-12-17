#!/bin/bash


echo "client config.."
sudo openvpn --mktun --dev tun66
sudo ip link set tun66 up
sudo ip addr add 192.168.3.2/24 dev tun66
ip route show
