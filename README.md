<h1 align="center">
  <br>
  Fleex
</h1>


Fleex allows you to create multiple VPS on cloud providers and use them to distribute your workload. Run tools like masscan, puredns, ffuf, httpx or anything you need and get results quickly!

<p align="center">
<a href="https://github.com/sw33tLie/fleex/issues"><img src="https://img.shields.io/badge/contributions-welcome-blue.svg?style=flat"></a>
<img alt="AUR license badge" src="https://img.shields.io/badge/license-Apache-blue">
<a href="https://github.com/sw33tLie/fleex/releases"><img src="https://img.shields.io/github/release/sw33tLie/fleex"></a>
<br>
<a href="https://twitter.com/sw33tLie"><img src="https://img.shields.io/twitter/follow/sw33tLie.svg?logo=twitter"></a>
<a href="https://twitter.com/xm1k3_"><img src="https://img.shields.io/twitter/follow/xm1k3_.svg?logo=twitter"></a>
</p>

# Install 
```
GO111MODULE=on go get -v github.com/sw33tLie/fleex
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
  scp         SCP client
  spawn       Spawn a fleet
  ssh         Start SSH
```

# Documentation

See the documentation [here](https://sw33tlie.github.io/fleex-docs/)

# Main contributors
<a href="https://github.com/sw33tLie"><img width="75px;" src="https://avatars.githubusercontent.com/u/47645560?v=4"></a>
<a href="https://github.com/xm1k3"><img  width="75px;" src="https://avatars.githubusercontent.com/u/73166077?v=4?s=100"></a>

# License
Fleex is distributed under Apache-2.0 License
