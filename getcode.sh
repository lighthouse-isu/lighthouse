#!/bin/bash

# Pull code
cd /data
git clone git@github.com:ngmiller/lighthouse.git
git clone git@github.com:lighthouse/lighthouse-client.git

# Pull backend
cd /data/lighthouse
git fetch origin
git checkout back-branch

# Pull frontend and build
cd /data/lighthouse-client
git fetch origin
git checkout front-branch
npm install
gulp build

# Run server 
cd /data/lighthouse
## server commands here
