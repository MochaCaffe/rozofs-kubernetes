#!/bin/bash

INIT_NODE="master"
STORAGE=("master" "worker01" "worker02" "worker03")

echo master=${STORAGE[*]}

#config volume
if [ $(rozo volume list -E ${INIT_NODE} | grep VOLUME | wc -l) -gt  1 ];then
	exit 1
fi
for i in {1..2};
do

	rozo volume expand ${STORAGE[*]} -E ${INIT_NODE}
	rozo export create ${i} -E ${INIT_NODE}
	rozo mount create -i ${i} -E ${INIT_NODE}
done
exit 0
