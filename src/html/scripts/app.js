angular.module('solarcompare', ['ngGrid', 'MyPlantsService', 'google-maps']).
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
