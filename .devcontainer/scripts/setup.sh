#!/bin/bash
set -euxo pipefail

# Install plandex.
curl -sL https://plandex.ai/install.sh | bash

# Setup aliases.
echo alias ll=\\'ls -alF\\' >> ~/.bash_aliases
echo alias pdx=\\'plandex\\' >> ~/.bash_aliases
