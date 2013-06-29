angular.module('MyPlantsService', ['restangular']).
	factory('MyPlants', function(Restangular) {
		r = Restangular.withConfig(function(RestangularConfigurer) {
			//Set the id fieldname for restangular
			RestangularConfigurer.setRestangularFields({
				id: "PlantKey"
			});			
		});
		return r.all('plant').customGETLIST('?myplants=true');


/*
		var myPlants = myplantsResource.query({}, function() {
			
			console.log("Number of my plants: " + MyPlants.NumberOfMyPlants);
		});

		//var myPlants = {"plants" : [{"PlantKey":"jjj"}]};
		return myPlants;
		*/
	}).
	factory('My', function(Restangular) {
		return Restangular.one('auth').one('user').get();
	});

