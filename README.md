<h1 align="center">
  <br>
  Fleex
</h1>


Fleex allows you to create multiple VPS on cloud providers and use them to distribute your workload. Run tools like masscan, puredns, ffuf, httpx or anything you need and get results quickly!

<p align="center">
<a href="https://github.com/sw33tLie/fleex/issues"><img src="https://img.shields.io/badge/contributions-welcome-blue.svg?style=flat"></a>
<img alt="AUR license badge" src="https://img.shields.io/badge/license-Apache-blue">
<br>
<a href="https://twitter.com/sw33tLie"><img src="https://img.shields.io/twitter/follow/sw33tLie.svg?logo=twitter"></a>
<a href="https://twitter.com/xm1k3_"><img src="https://img.shields.io/twitter/follow/xm1k3_.svg?logo=twitter"></a>
</p>

# Install 
```
GO111MODULE=on go get -v github.com/sw33tLie/fleex
```

# Supported providers
- Linode
- Digitalocean

# Available commands
```
./fleex -h

Available Commands:
  build       Build image
  config      fleex config setup
  delete      Delete a fleet or a single box
  help        Help about any command
  images      List available images
  ls          List running boxes
  run         Run a command
  scan        Distributed scanning
  scp         SCP client
  spawn       Spawn a fleet
  ssh         Start SSH

```

# Config file (~/fleex/config.yaml)

```
provider: digitalocean or linode
public-ssh-file: id_rsa.pub
private-ssh-file: id_rsa
linode:
  token: YOUR_LINODE_TOKEN
  region: eu-central
  size: g6-nanode-1
  image: private/12345678 : put your image id here (./fleex images to get it)
  port: 2266
  username: op
  password: USER_PASSWORD
digitalocean:
  token: YOUR_DIGITALOCEAN_TOKEN
  region: fra1
  size: s-1vcpu-1gb
  image: 12345678 # put your image id here
  port: 2266
  username: op
  password: USER_PASSWORD
  tags:
    - vps
    - fleex

```

# Examples
## Masscan example command: 
```
fleex scan -n pwn -i ./input-ips.txt -o scan-results.txt -c "sudo masscan -iL {{INPUT}} -p80,443,8080,8443,8000 --rate 10000 --output-format json --output-filename {{OUTPUT}}"
```

# Documentation

See the documentation [here]()

## Massdns example command:
```
fleex scan -n pwn -i /tmp/testdns -o scan-results.txt -c "sudo /usr/bin/massdns -r /home/op/lists/resolvers.txt -t A -o S {{INPUT}} -w {{OUTPUT}}"
```

# Main contributors
<a href="https://github.com/sw33tLie"><img width="75px;" src="https://avatars.githubusercontent.com/u/47645560?v=4"></a>
<a href="https://github.com/xm1k3"><img  width="75px;" src="https://avatars.githubusercontent.com/u/73166077?v=4?s=100"></a>

# License
Fleex is distributed under Apache-2.0 License