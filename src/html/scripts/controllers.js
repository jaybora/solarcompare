function FrontpagePlantsCtrl($scope, Plants, $timeout) {
	//var plantsResource = $resource('/plant');
	//$scope.plants = plantsResource.query();

	//var myplantsResource = $resource('/plant?myplants=true');
	// MyPlants.plants = myplantsResource.query({}, function() {
	// 	$scope.NumberOfMyPlants = MyPlants.plants.length;
		
	// });

	// var updateFn = function() {
	// 	$scope.myPlants = MyPlants.then(function(plants) {
	// 		$scope.NumberOfMyPlants = plants.length;
	// 	});
	// 	// MyPlants.plants = myplantsResource.query({}, function() {
	// 	// 	$scope.NumberOfMyPlants = MyPlants.plants.length;
	// 		//$timeout(updateFn, 30000);
	// 	//});
	// };

	$scope.plants = [];
	Plants.then(function(plants) {
		for (var i = 0; i < plants.length; i++) {
			var plant = plants[i];
			
			plant.pvdata = plant.one('pvdata').get();
			$scope.plants.push(plant);
			
		};
	    $timeout(updateFn, 1000);
	});
	var updateFn = function() {
		for (var i = 0; i < $scope.plants.length; i++) {
			var plant = $scope.plants[i];
			// plant.pvdata = plant.one('pvdata').get();
			plant.one('pvdata').get().then(function(pv) {
				var plantkey = pv.parentResource.PlantKey;
				var plant = _.find($scope.plants, function(plant) {
					return plant.PlantKey === plantkey;
				});

				console.log("Updating pvdata of " + plant.PlantKey + " to " + pv.PowerAc)
				plant.pvdata = pv;

				// For gauge chart
				plant.pvdata.chart = {
					"type": "AreaChart",
					"data": [['Watt'], [plant.pvdata.PowerAc]],
					"options": {
						width: 130, height: 130,
						minorTicks: 10,
						majorTicks: ['0', '1', '2', '3', '4', '5', '6'],
						max: 6000,
						animation: {duration: 600, easing: 'inAndOut'}
					}
				};
			});
		}
		
		$timeout(updateFn, 10000);
	};



}

function MyPlantsCtrl($scope, MyPlants) {
	//var myplantsResource = $resource('/plant?myplants=true');
	//MyPlants.plants = myplantsResource.query();
	$scope.plants = MyPlants;

	$scope.gridOptions = {
		data: 'plants',
		multiSelect: false,
		keepLastSelected:false,
		columnDefs: [{field:'PlantKey', displayName:'Anlægs id'},
		{field:'Name', displayName:'Navn'}],
	    beforeSelectionChange:function(row){
			window.location.assign("#/myplants/" + row.entity.PlantKey);
			return false;
		}
	};
}

function MyPlantDetailCtrl($scope, $routeParams, MyPlants, $window, $timeout) {
	if ($scope.plant === undefined) {
		$scope.plant = {Latitude: 0,
					Longitude: 0};
		$scope.map = {};
		$scope.map.center = {
					latitude: $scope.plant.Latitude,
					longitude: $scope.plant.Longitude
				};
		$scope.map.zoom = 7;

		// It needs a marker array to store 
		$scope.map.markers = [];
		console.log("Setting maps to lng 0 ltd 0");
		$scope.map.refresh = false;
	}

	$scope.geolocationAvailable = navigator.geolocation ? true : false;
	
	
	MyPlants.then(function(plants) {
		$scope.plant = _.find(plants, function(plant) {
			return plant.PlantKey === $routeParams.PlantKey;
		});
		console.log("Setting map of plant to scope");
		$scope.map.latitude = $scope.plant.Latitude;
		$scope.map.longitude = $scope.plant.Longitude;
		$scope.map.markers = [
			   {latitude: $scope.plant.Latitude,
				longitude: $scope.plant.Longitude}];
		
		// In order to make the map draw correctly after user 
		// reselect same plant, make the center property update 
		// in new event thread
		$timeout(function() {
			$scope.map.center = {
					latitude: $scope.plant.Latitude,
					longitude: $scope.plant.Longitude
			};
			$scope.$apply();
		}, 0);


	})

	$scope.findMe = function () {
		console.log("Find me running...");
			
			if ($scope.geolocationAvailable) {
				
				navigator.geolocation.getCurrentPosition(function (position) {
					
					$scope.map.center = {
						latitude: position.coords.latitude,
						longitude: position.coords.longitude
					};
					$scope.map.latitude = position.coords.latitude;
					$scope.map.longitude = position.coords.longitude;
					$scope.map.markers = [
					   {latitude: position.coords.latitude,
						longitude: position.coords.longitude}];
							
					$scope.$apply();
				}, function () {
					
				});
			}	
		};


	$scope.update = function() {
		// Update map data from map object and make it a string
		$scope.plant.Longitude = $scope.map.longitude + "";
		$scope.plant.Latitude = $scope.map.latitude + "";
		$scope.plant.put().then(function() {
			console.log("Saved OK");
			$window.location.assign('#/myplants');
		}, function() {
			alert("Der opstod en fejl ")
			$scope.plant.get().then(function(plant) {
				$scope.plant = plant;
			});
		});
	}

	$scope.cancel = function() {
		$scope.plant.get().then(function(plant) {
			$scope.plant = plant;
			$window.location.assign('#/myplants');
		});		
	}


}

function UserCtrl($scope, My) {
	$scope.user = My
}

