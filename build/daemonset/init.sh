#!/bin/bash

###
### Install Driver
###

VENDOR=mushroommagnet
DRIVER=rozofs

driver_dir=$VENDOR${VENDOR:+"~"}${DRIVER}
mkdir -p "/flexmnt/$driver_dir"
cp "/jq" "/flexmnt/$driver_dir/.jq"
mv -f "/flexmnt/$driver_dir/.jq" "/flexmnt/$driver_dir/jq"
cp "/$DRIVER" "/flexmnt/$driver_dir/.$DRIVER"
mv -f "/flexmnt/$driver_dir/.$DRIVER" "/flexmnt/$driver_dir/$DRIVER"

###
### Rozo Storage Daemonset
###
start_daemons() {
  # restore fstab
  cp -f /etc/rozofs/fstab /etc/fstab
  # start rozo agent
  /etc/init.d/rozofs-manager-agent start
  # start rozo export
  if [ "$(grep -c '\(volumes\|exports\) *= *( *) *;' /etc/rozofs/export.conf)" != "2" ]; then
    /etc/init.d/rozofs-exportd start
  fi
  # start rozo storage
  if [ "$(grep -c 'storages *= *( *) *;' /etc/rozofs/storage.conf)" != "1" ]; then
    /etc/init.d/rozofs-storaged start
  fi
  # start rozo mounts
  awk '$1 == "rozofsmount" {print $2}' /etc/fstab |
    while read mount; do
      mkdir -p "$mount"
      mount "$mount"
    done
}

stop_daemons() {
  # save fstab
  cp -f /etc/fstab /etc/rozofs/fstab.bak
  mv /etc/rozofs/fstab.bak /etc/rozofs/fstab
  awk '$1 == "rozofsmount" {print $2}' /etc/fstab |
    while read mount; do
      umount "$mount"
    done
  /etc/init.d/rozofs-storaged stop
  /etc/init.d/rozofs-exportd stop
  /etc/init.d/rozofs-manager-agent stop
}

# start logging
/bin/busybox syslogd
tail -F /var/log/messages 2>/dev/null &

trap stop_daemons EXIT
start_daemons

while inotifywait -e modify /etc/fstab; do
  cp -f /etc/fstab /etc/rozofs/fstab.bak
  mv /etc/rozofs/fstab.bak /etc/rozofs/fstab
done &

# Fix failed mounts
while : ; do
	if [[ "$HOSTNAME" == "$ROZO_EXPORT_HOSTNAME" || "$POD_IP" == "$ROZO_EXPORT_HOSTNAME" ]];then

		mountErrors=$(comm -3 <(rozo export get | grep 'EXPORT ' | wc -l) <(df 2>/dev/null | grep rozofs | wc -l))
		if [[ ! -z "$mountErrors" ]];then
			/etc/init.d/rozofs-exportd restart
		fi
	fi
	sleep 20
done
