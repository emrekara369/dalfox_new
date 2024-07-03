---
title: Installation
permalink: /docs/installation/
---

## Using Homebrew
Homebrew is the package manager for MacOS(or linux). On devices using homebrew, you can easily install/update using the brew command.

### Install homebrew
```shell
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install.sh)"
```

### Install dalfox
```shell
brew install dalfox

# https://formulae.brew.sh/formula/dalfox
```

## Using Snapcraft
Snapcraft is one of the packaging managers for Linux. Unlike app and yum, it can be used independently of the deployment OS version.

### Install Snapcraft
Please check this documents [https://snapcraft.io/docs/installing-snapd](https://snapcraft.io/docs/installing-snapd)

### Install dalfox
```
sudo snap install dalfox
```

## From source

```bash
go install github.com/emrekara369/dalfox_new/v2@latest

# The actual release might slightly differ. This is because go install references the main branch.
```

## Using Docker
Dalfox provides docker images by version. It can be used lightly with less capacity.
```
docker pull hahwul/dalfox:latest
```

if you installed it, using like this command
```
docker run -it hahwul/dalfox:latest /app/dalfox url https://www.hahwul.com
```

or live in docker

```
docker run -it hahwul/dalfox:latest /bin/bash
./dalfox
```
