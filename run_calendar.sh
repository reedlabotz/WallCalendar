#!/bin/bash

# /home/reedlabotz/12.48inch_e-Paper_Module_Code/c/epd
cd /home/reedlabotz
./wallcalendar --battery="$(echo "get battery" | nc -q 0 127.0.0.1 8423)"

echo "rtc_web" | nc -q 0 127.0.0.1 8423

current_hour=$(date +%H)

if [ "$current_hour" -eq "06" ]; then
  /usr/sbin/shutdown -P +1
else
  echo "It's not 6 AM, you must be debugging"
fi