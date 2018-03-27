# updateDNS

Embedded DNS updater which knows how to restart the Pi's network, especially
when subnets or router leases change.

## Install instructions

Create the relevant directories:

```bash
mkdir -p ~/.bin/
mkdir -p ~/.credentials/
```

Put your username and password creds in the path `/home/pi/.credentials/dnscreds`:

```json
{
    "username": "your_username",
    "password": "your_password"
}
```

### Install the code

```bash
go get -u github.com/tydavis/updatedns
cp $GOPATH/bin/updatedns ~/.bin/
```

### Add systemd service

```bash
sudo cp updatedns.service /lib/systemd/system/
sudo systemctl enable updatedns.service
sudo systemctl start updatedns.service
```
