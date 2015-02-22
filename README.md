Lighthouse
==============

[Lighthouse](https://lighthouse.github.io) is a Docker controller. It aggregates information about Docker instances across mulitple cloud providers and allows for easy control over the containers deployed on that system.

It bridges the gap between providers hosting docker services. Are you running containers accross hundreds of vms across the world wide web? No problem, Lighthouse gives you the power to manage and maintain those vms with a few simple clicks. Everything was built with the goal of monotizing AWS, GCE, Azure, ect. into an easy to manage platform.

### Dependencies

* [golang](https://golang.org/)
* a Docker instance if running locally. [boot2docker](http://boot2docker.io/) works well for Windows or OS X
* see [lighthouse-client](https://github.com/lighthouse/lighthouse-client) for set up if opting to use web client
* (optional) [PostgresSQL](http://www.postgresql.org/)

### Build & Run

* `go get github.com/lighthouse/lighthouse`
* (optional) build a static client into the root directory named `static`, see [lighthouse-client](https://github.com/lighthouse/lighthouse-client) for more information
* `cd $GOPATH/src/github.com/lighthouse/lighthouse`
* Run postgres locally or inside boot2docker
  * `docker run -p 5432:5432 -d postgres:latest`
  * See more about running PostgresSQL locally [here](http://www.postgresql.org/docs/9.1/static/tutorial-start.html) if you don't want to use docker
* `$GOPATH/bin/lighthouse`

### Build & Run W/ Docker

* `go get github.com/lighthouse/lighthouse`
* (optional) build a static client into the root directory named `static`, see [lighthouse-client](https://github.com/lighthouse/lighthouse-client) for more information
* `cd $GOPATH/src/github.com/lighthouse/lighthouse`
* `docker build -t lighthouse .`
* `docker run --name postgres-image -d postgres:latest`
* `docker run -t -i --rm -p 5000:5000 --link postgres-image:postgres lighthouse`

### API

Write your own client! See the API [documentation](https://github.com/lighthouse/lighthouse/wiki/API-v0.2)

### Team

We're a group of engineering students completing our senior project at Iowa State University. Developed and tested with the help of [Workiva](https://github.com/workiva)

* [Caleb Brose](https://github.com/cmbrose)
* [Chris Fogerty](https://github.com/chfogerty)
* [Zach Taylor](https://github.com/zach-taylor)
* [Rob Sheehy](https://github.com/theMagicalKarp)
* [Nick Miller](https://github.com/ngmiller)

### License

[Apache v2.0](http://www.apache.org/licenses/LICENSE-2.0)
