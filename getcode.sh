#!/bin/bash

# Pull backend updates
cd /data/lighthouse
git fetch origin
git checkout back-branch
git pull origin back-branch

# Pull frontend updates and build
cd /data/lighthouse-client
git fetch origin
git checkout front-branch
git pull origin back-branch

npm install
gulp build

# Run server 
cd /data/lighthouse/backend/static
# just testing
python -m SimpleHTTPServer 5000
