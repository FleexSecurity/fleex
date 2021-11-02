![Fleex](static/Fleex-Banner.png)


Fleex allows you to create multiple VPS on cloud providers and use them to distribute your workload. Run tools like masscan, puredns, ffuf, httpx or anything you need and get results quickly!

<p align="center">
<a href="https://github.com/FleexSecurity/fleex/issues"><img src="https://img.shields.io/badge/contributions-welcome-blue.svg?style=flat"></a>
<img alt="AUR license badge" src="https://img.shields.io/badge/license-Apache-blue">
<a href="https://github.com/FleexSecurity/fleex/releases"><img src="https://img.shields.io/github/release/FleexSecurity/fleex"></a>
<br>
<a href="https://twitter.com/sw33tLie"><img src="https://img.shields.io/twitter/follow/sw33tLie.svg?logo=twitter"></a>
<a href="https://twitter.com/xm1k3_"><img src="https://img.shields.io/twitter/follow/xm1k3_.svg?logo=twitter"></a>
<br>
<br>
<a href="https://www.buymeacoffee.com/xm1k3"><img src="https://www.buymeacoffee.com/assets/img/custom_images/purple_img.png"></a>
<br>
</p>

# Install 
```
GO111MODULE=on go get -v github.com/FleexSecurity/fleex
```

# Supported providers
- [Linode](https://www.linode.com)
- [Digitalocean](https://www.digitalocean.com)
- [Vultr](https://www.vultr.com/)

# Available commands
```
./fleex -h

Available Commands:
  build       Build image
  config      Config setup
  delete      Delete a fleet or a single box
  help        Help about any command
  images      List available images
  init        This command initializes fleex
  ls          List running boxes
  run         Run a command
  scan        Distributed scanning
  scp         SCP client
  spawn       Spawn a fleet
  ssh         Start SSH
```

# Documentation

<a href="https://fleexsecurity.github.io/fleex-docs/"><img src="static/Fleex-docs.png" alt="Fleex-docs"></a>

# Referrals

<a href="https://www.digitalocean.com/?refcode=91982e64054b&utm_campaign=Referral_Invite&utm_medium=Referral_Program&utm_source=badge">
  <img src="static/Referrals/Digitalocean-referral.png" alt="Digitalocean referral link">
</a>
<a href="https://www.linode.com/?r=172cb6708bc78a41c5014cc2da0f2ab0d7abbe7b">
  <img src="static/Referrals/Linode-referral.png" alt="Linode referral link">
</a>

# Main contributors
<table>
  <tr>
    <td align="center">
      <a href="https://github.com/sw33tLie">
      <img
          width="75px;"
          src="https://avatars.githubusercontent.com/u/47645560?v=4"
          alt="sw33tLie"/>
        <br />
        <b>sw33tLie</b>
        </a>
    </td>
    <td align="center">
      <a href="https://github.com/xm1k3"
        ><img
          width="75px;"
          src="https://avatars.githubusercontent.com/u/73166077?v=4?s=100"
          alt="xm1k3"
        />
        <br />
        <b>xm1k3</b>
        </a>
    </td>
  </tr>
</table>

# Sponsors

<table>
  <tr>
    <td align="center">
      <a href="https://github.com/projectdiscovery">
      <img
          width="75px;"
          src="https://avatars.githubusercontent.com/u/50994705?v=4"
          alt="projectdiscovery"/>
        <br />
        <b>ProjectDiscovery</b>
        </a>
    </td>
     <td align="center">
      <a href="https://twitter.com/bsysop">
      <img
          width="75px;"
          src="https://avatars.githubusercontent.com/u/9998303?v=4"
          alt="bsysop"/>
        <br />
        <b>bsysop</b>
        </a>
    </td>
  </tr>
</table>


# License
Fleex is distributed under Apache-2.0 License
