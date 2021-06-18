# fleex
Distributed computing

# Install 
```
GO111MODULE=on go get -v github.com/sw33tLie/fleex
```

# Supported providers
- Linode
- Digitalocean

# Config file (~/fleex/config.yaml)

```
provider: "digitalocean"
#provider: "linode"
public-ssh-file: "id_rsa.pub"
private-ssh-file: "id_rsa"
linode:
  token: "YOUR_LINODE_TOKEN"
  region: "eu-central"
  size: "g6-nanode-1" 
  image: "private/11147382" # put your image id here (./fleex images to get it)
  port: 2266
  username: "op"
  password: "USER_PASSWORD"
digitalocean:
  token: "YOUR_DIGITALOCEAN_TOKEN"
  region: fra1
  size: s-1vcpu-1gb
  image: 85963266 # put your image id here
  port: 2266
  username: "op"
  password: "USER_PASSWORD" # if using an axiom image you can find this in axiom.json
  tags:
    - vps
    - fleex

```

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
  spawn       Spawn a fleet
  ssh         Start SSH

```

# Masscan example command: 
```
go run main.go scan -n pwn -i ./input-ips.txt -c "sudo masscan -iL {{INPUT}} -p80,443,8080,8443,8000 --rate 10000 --output-format json --output-filename {{OUTPUT}}"
```
Results will be saved in a folder in ~/fleex