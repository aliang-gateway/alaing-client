sudo ifconfig utun98 198.18.1.1 198.18.1.2 up
sudo route -n add -net 172.16.238.2/32 198.18.1.1
