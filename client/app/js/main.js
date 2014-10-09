
var angular = require('angular'),
    mainController = require('./controllers/mainController'),
    dockerService = require('./services/dockerService');

// initialize angular app
var app = angular.module('dockerThing', []);

// initialize controllers
app.controller('mainController', mainController);

// initialize factories
app.factory('dockerService', dockerService);
