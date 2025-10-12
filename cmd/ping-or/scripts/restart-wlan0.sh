#!/usr/bin/env bash

echo "restarting wlan0"

nmcli device down wlan0
sleep 5
nmcli device up wlan0

#ifconfig wlan0 down
#sleep 10
#ifconfig wlan0 up
