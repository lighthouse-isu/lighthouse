/*
 * mainController
 * Initial view control
 */
function mainController($scope, dockerService) {
    'use strict';

    // Host information wrapper
    $scope.remote = {};

    // TODO move to some utility service
    $scope.formatTime = function(unixEpoch) {
        var record = new Date();
        var year = record.getFullYear();
        var month = record.getMonth();
        var date = record.getDate();
        var hour = record.getHours();
        var min = record.getMinutes();
        var sec = record.getSeconds();

        return [
            month, '/', date, '/', year, ' ', hour, ':', min, ':', sec
        ].join('');
    };

    // Basic Docker host information
    dockerService.getInfo().then(
        // success
        function (response) {
            $scope.remote.info = JSON.stringify(response.data, null, " ");
        },
        // error
        function (response) {
            // TODO generate an alertService to show toast alerts on API failures
            console.log('Error retreiving host information');
            console.log('-> status: ' + response.status);
        }
    );

    // Containers on Docker host
    dockerService.getContainers().then(
        // success
        function (response) {
            $scope.remote.containers = response.data;
        },
        // error
        function (response) {
            console.log('Error retreiving host containers');
            console.log('-> status: ' + response.status);
        }
    );

    // Images on Docker host
    dockerService.getImages().then(
        // success
        function (response) {
            $scope.remote.images = response.data;
        },
        // error
        function (response) {
            console.log('Error retreiving host images');
            console.log('-> status: ' + response.status);
        }
    );
}

mainController.$inject = ['$scope', 'dockerService'];
module.exports = mainController;