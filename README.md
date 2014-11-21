Lighthouse
==============

[Lighthouse](https://lighthouse.github.io) is a Docker controller. It aggregates information about Docker instances across mulitple cloud providers and allows for easy control over the containers deployed on that system.

It bridges the gap between providers hosting docker services. Are you running containers accross hundreds of vms across the world wide web? No problem, Lighthouse gives you the power to manage and maintain those vms with a few simple clicks. Everything was built with the goal of monotizing AWS, GCE, Azure, ect. into an easy to manage platform.

### Build and Deploy

* `git clone git@github.com:lighthouse/lighthouse.git`
* (optional) `git clone git@github.com:lighthouse/lighthouse-client.git && cd lighthouse-client`
* (optional) `gulp prod build && cd ..`
* `docker build -t lighthouse .`
* `docker run --name postgres-image -d postgres`
* `docker run -t -i -p 5000:5000 --link postgres-image:postgres lighthouse`

The optional commands above will grab and build the web app frontend if you desire such functionality. Otherwise, see the API documentation for more information.

### Dependencies

* [golang](https://golang.org/)
* a Docker instance if running locally. [boot2docker](http://boot2docker.io/) works well for Windows or OS X
* see [lighthouse-client](https://github.com/lighthouse/lighthouse-client) for set up if opting to use web client

### API

Write your own client! See the API [documentation](https://github.com/lighthouse/lighthouse/wiki/API-Design)

### Team

We're a group of engineering students completing our senior project at Iowa State University. Developed and tested with the help of [Workiva](https://github.com/workiva)

* [Caleb Brose](https://github.com/cmbrose)
* [Chris Fogerty](https://github.com/chfogerty)
* [Zach Taylor](https://github.com/zach-taylor)
* [Rob Sheehy](https://github.com/theMagicalKarp)
* [Nick Miller](https://github.com/ngmiller)

### License

[Apache v2.0](http://www.apache.org/licenses/LICENSE-2.0)
