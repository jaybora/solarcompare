angular.module('MyPlantsService', ['restangular']).
    config(function(RestangularProvider) {
    	RestangularProvider.setBaseUrl('/api/v1');
    }).
	factory('MyPlants', function(Restangular) {
		r = Restangular.withConfig(function(RestangularConfigurer) {
			//Set the id fieldname for restangular
			RestangularConfigurer.setRestangularFields({
				id: "PlantKey"
			});	
			

		});
		return r.all('plant').customGETLIST('?myplants=true');
	}).
	factory('Plants', function(Restangular, $routeParams) {
		r = Restangular.withConfig(function(RestangularConfigurer) {
			//Set the id fieldname for restangular
			RestangularConfigurer.setRestangularFields({
				id: "PlantKey"
			});	
			

		});
		console.log("Getting list of all plants")
		console.log("Plants given is " + $routeParams.plants)
		if ($routeParams.plants == undefined) {
			return r.all('plant').getList();
		} else {
			return r.all('plant').customGETLIST('?plants=' + $routeParams.plants);
		}
	}).	
	factory('PvData', function(Restangular) {
		r = Restangular.withConfig(function(RestangularConfigurer) {
			//Set the id fieldname for restangular
			RestangularConfigurer.setRestangularFields({
				id: "PlantKey"
			});	
			

		});
		return r.one('plant').one('pvdata');
	}).	factory('DataProviders', function(Restangular) {
		r = Restangular.withConfig(function(RestangularConfigurer) {
			//Set the id fieldname for restangular
			RestangularConfigurer.setRestangularFields({
				id: "DataProvider"
			});	
			

		});
		return r.all('dataprovider').getList();
	}).
	factory('My', function(Restangular) {
		return Restangular.one('auth').one('user').get();
	}).
	filter('timeFilter', function() {
		return function(isodate) {
			if (isodate == null) {	
				return "";
			}
			return /\T(.*?)\./.exec(isodate)[1];
		};
	});

