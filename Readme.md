# Map

## Raspberry Pi Zero (W) Setup

1. Install full Raspbian on a microSD card.
2. Open the card.
3. Append `dtoverlay=dwc2` to `config.txt`.
4. Create an empty file in the root called `ssh` (no extension) to enable SSH.
5. Append `modules-load=dwc2,g_ether` to `cmdline.txt`. No newlines, single space separation.
6. Boot the Pi. Plug in both USB ports (power and OTG).
7. The device should enumerate as an ethernet adapter.
8. `ssh pi@raspberrypi.local` with the password `raspberry`.
9. `passwd` to change the password.
10. `sudo nano /etc/wpa_supplicant/wpa_supplicant.conf`
11. Add

    network={
        ssid="mynetwork"
        psk="wifipassword"
    }

12. `sudo nano /etc/hostname` to change the hostname from `raspberrypi`
13. `sudo nano /etc/hosts` - change it there too.
14. Reboot! `sudo shutdown -r now` (Linux is so intuitive...)
15. `sudo apt-get install golang`
16. `cd`
17. `mkdir go`
18. `nano .profile`
19. Append:

    export GOPATH="$HOME/go"
    export GOBIN="$GOPATH/bin"
    export PATH="$PATH:$GOBIN"

20. `source .profile`
21. Copy files over via scp.
22. `cd map`
23. `go get`
24. `go get github.com/codegangsta/gin`
25. `sudo iptables -A PREROUTING -t nat -i wlan0 -p tcp --dport 80 -j REDIRECT --to-port 3000`
26. `nohup gin &`