#!/bin/bash
set -euxo pipefail

# Install plandex.
# curl -sL https://plandex.ai/install.sh | bash

# Install Air - https://github.com/air-verse/air
go install github.com/air-verse/air@latest

# Setup aliases.
echo alias ll=\\'ls -alF\\' >> ~/.bash_aliases
echo alias pdx=\\'plandex\\' >> ~/.bash_aliases
