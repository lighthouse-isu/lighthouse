/*
 * dockerService
 * Requests docker host and container information
 */
function dockerService($http) {
    /*
     * getInfo
     * GET /info
     */
    function getInfo() {
        return $http.get('/info',  {
            'responseType': 'text'
        });
    }

    /*
     * getContainers
     * GET /containers
     */
    function getContainers() {
        return $http.get('/containers');
    }

    /*
     * getImages
     * GET /images
     */
    function getImages() {
        return $http.get('/images');
    }

    return {
        'getInfo': getInfo,
        'getContainers': getContainers,
        'getImages': getImages
    };
}

dockerService.$inject = ['$http'];
module.exports = dockerService;
