angular.module('solarcompare', ['ngResource'])

function PlantController($scope, $resource) {
  $scope.plant = $resource(
    '/plant/:plantKey',{plantKey:'jbr'});

  $scope.fetch = function() {
    $scope.plantRslt = $scope.plant.get({plantKey:$scope.PlantKey})

  }
}

