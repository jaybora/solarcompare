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
	factory('Plants', function(Restangular) {
		r = Restangular.withConfig(function(RestangularConfigurer) {
			//Set the id fieldname for restangular
			RestangularConfigurer.setRestangularFields({
				id: "PlantKey"
			});	
			

		});
		return r.all('plant').getList();
	}).	
	factory('PvData', function(Restangular) {
		r = Restangular.withConfig(function(RestangularConfigurer) {
			//Set the id fieldname for restangular
			RestangularConfigurer.setRestangularFields({
				id: "PlantKey"
			});	
			

		});
		return r.one('plant').one('pvdata');
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

