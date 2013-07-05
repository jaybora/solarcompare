'use strict';

google.setOnLoadCallback(function () {
    angular.bootstrap(document.body, ['solarcompare']);
});
google.load('visualization', '1', {packages: ['corechart']});
google.load('visualization', '1', {packages:['gauge']});

angular.module('solarcompare', ['ngGrid', 'MyPlantsService', 'google-maps', 'googlechart.directives']).
  config(['$routeProvider', function($routeProvider) {
  $routeProvider.
      when('/', {templateUrl: '/html/partials/frontpage-plants.html',   
      		controller: FrontpagePlantsCtrl}).
      when('/myplants', {templateUrl: '/html/partials/myplants.html',   
      		controller: MyPlantsCtrl}).
      when('/myplants/:PlantKey', {templateUrl: '/html/partials/myplant-details.html', 
      	controller: MyPlantDetailCtrl}).
      otherwise({redirectTo: '/'});
}]);
