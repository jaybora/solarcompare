function FrontpagePlantsCtrl($scope, Plants, $timeout, $filter, $routeParams) {
	$scope.plants = [];
	console.log("FrontpagePlantsCtrl to show the following plants: " + 
		$routeParams.plants)
	Plants.then(function(plants) {
		console.log("FrontpagePlantsCtrl plants update")
		for (var i = 0; i < plants.length; i++) {
			var plant = plants[i];
			
			
			// // For map
			// plant.map = {};
			// plant.map.center = {
			// 	latitude: plant.Latitude,
			// 	longitude: plant.Longitude
			// };
			// plant.map.zoom = 7;

			// console.log("Setting map for " + plant.PlantKey);
			// plant.map.markers = [	
			// 	   {latitude: plant.Latitude,
			// 		longitude: plant.Longitude}];
			

			// For gauge chart
			console.log("Setting up gauge for " + plant.PlantKey);
			plant.PowerAcGauge = {
				"type": "Gauge",
				"data": [['Watt'], [0]],
				"options": {
					width: 130, height: 130,
					minorTicks: 10,
					majorTicks: ['0', '1', '2', '3', '4', '5', '6'],
					max: 6000,
					animation: {duration: 1600, easing: 'inAndOut'}
				}
			};

			// For area powerac daily chart
			console.log("Setting up area chart powerac daily for " + plant.PlantKey);
			plant.PowerAcAreaChart = {
				"type": "AreaChart",
				"data": [['Kl', 'Watt'], 
				         ['0:00', 0]
				         ],
				"options": {
					width: 130, height: 60,
					axisTitlesPosition: 'none',
					legend: {position: 'none'},
					vAxis: {maxValue: 6000, minValue: 0, viewWindowMode: 'maximized'},
					animation: {duration: 1600, easing: 'inAndOut'}
				}
			};

			// For area energytoday daily chart
			console.log("Setting up area chart energytoday daily for " + plant.PlantKey);
			plant.EnergyTodayAreaChart = {
				"type": "AreaChart",
				"data": [['Kl', 'Watt'], 
				         ['0:00', 0]
				         ],
				"options": {
					width: 130, height: 60,
					axisTitlesPosition: 'none',
					legend: {position: 'none'},
					vAxis: {maxValue: 60000, minValue: 0, viewWindowMode: 'maximized'},
					animation: {duration: 1600, easing: 'inAndOut'}
				}
			};
				
			$scope.plants.push(plant);

		};

	    $timeout(updateFn, 100);
	    $timeout(updateAreaChartFn, 100);
	});

	// Fast update function for powerac
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
				plant.PowerAcGauge.data = [['Watt'], [pv.PowerAc]];

			});
		}
		$timeout(updateFn, 10000);
	};


	// Slow update function for area chart
	var updateAreaChartFn = function() {
		for (var i = 0; i < $scope.plants.length; i++) {
			var plant = $scope.plants[i];
			// plant.pvdata = plant.one('pvdata').get();
			plant.one('logpvdata').get().then(function(logpvdata) {
				var plantkey = logpvdata.parentResource.PlantKey;
				var plant = _.find($scope.plants, function(plant) {
					return plant.PlantKey === plantkey;
				});

				console.log("Updating logpvdata of " + plant.PlantKey)
				plant.logpvdata = logpvdata;

				// For area powerac chart
				plant.PowerAcAreaChart.data = 
				        [['Kl', 'Watt']];
				for (var i = 0; i < logpvdata.length; i++) {
					plant.PowerAcAreaChart.data.push(
						[$filter('timeFilter')(logpvdata[i].PvData.LatestUpdate),
						logpvdata[i].PvData.PowerAc]);
					
				};

				// For area energytoday chart
				plant.EnergyTodayAreaChart.data = 
				        [['Kl', 'Watt']];
				for (var i = 0; i < logpvdata.length; i++) {
					plant.EnergyTodayAreaChart.data.push(
						[$filter('timeFilter')(logpvdata[i].PvData.LatestUpdate),
						logpvdata[i].PvData.EnergyToday]);
					
				};

			});
		}
		$timeout(updateAreaChartFn, 300000);
	};



}

function MyPlantsCtrl($scope, MyPlants) {
	$scope.plants = MyPlants.getAll();

	// $scope.gridOptions = {
	// 	data: 'plants',
	// 	multiSelect: false,
	// 	keepLastSelected:false,
	// 	columnDefs: [{field:'PlantKey', displayName:'Anlægs id'},
	// 	{field:'Name', displayName:'Navn'}],
	//     beforeSelectionChange:function(row){
	// 		window.location.assign("#/myplants/" + row.entity.PlantKey);
	// 		return false;
	// 	}
	// };

	$scope.add = function() {
		window.location.assign("#/myplants/add");
	}
}

function MyPlantDetailCtrl($scope, $routeParams, MyPlants, $window, $timeout, DataProviders) {
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

	$scope.updatedDataprovider = function() {
		console.log("Changed dataprovider");
		DataProviders.then(function(providers) {
			$scope.selectedDataProvider = _.find(providers, function(provider) {
					return provider.DataProvider === $scope.plant.DataProvider;
				});
		});
	}

	$scope.addMode = $routeParams.PlantKey == 'add';
	console.log("addMode is " + $scope.addMode);

	
	$scope.DataProviders = DataProviders;
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
	
	if ($scope.addMode) {
	} else {
		MyPlants.getAll().then(function(plants) {
			$scope.plant = _.find(plants, function(plant) {
				return plant.PlantKey === $routeParams.PlantKey;
			});
			console.log("Setting map of plant to scope");
			$scope.map.latitude = $scope.plant.Latitude;
			$scope.map.longitude = $scope.plant.Longitude;
			$scope.map.markers = [
				   {latitude: $scope.plant.Latitude,
					longitude: $scope.plant.Longitude}];

			$scope.installationdate = new Date($scope.plant.InstallationData.StartDate)
			if ($scope.installationdate.getFullYear() < 1000) {
				$scope.installationdate = new Date();
			}
			$scope.updatedDataprovider();

			
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
	}

	$scope.createPlant = function() {
		console.log("Create plant")
		$scope.plant = MyPlants.add($scope.PlantKey);
		$scope.plant.PlantKey = $scope.PlantKey;
		$scope.plant.InstallationData = {};
		$scope.plant.InitiateData = {};
		$scope.findMe();
		$scope.installationdate = new Date();
		$scope.addMode = false;
	}



	$scope.update = function() {
		if ($scope.addMode) {
			// Post to be ignored
			return;
		}
		// Update map data from map object and make it a string
		$scope.plant.Longitude = $scope.map.longitude + "";
		$scope.plant.Latitude = $scope.map.latitude + "";
		$scope.plant.InstallationData.StartDate = $scope.installationdate;
		$scope.plant.InitiateData.PlantKey = $scope.plant.PlantKey;
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

	$scope.delete = function() {
		var sure = confirm("Vil du sikker på du ønsker at slette anlægget?");
		if (sure) {
			$scope.plant.remove().then(function() {
				console.log("Deleted ok");
				$window.location.assign('#/myplants');
				
			}, function() {
				alert("Der opstod en fejl ved sletning");

			});
		}
	}

	$scope.cancel = function() {
		if ($scope.addMove) {
			$scope.plant.get().then(function(plant) {
				$scope.plant = plant;
				$window.location.assign('#/myplants');
			});				
		} else {
			$window.location.assign('#/myplants');
		}
	}

	$scope.showUsername = function() {
		if ($scope.selectedDataProvider == undefined) {
			return false;
		}
		return _.contains($scope.selectedDataProvider.RequiredFields, 'UserName');
	}

	$scope.showPassword = function() {
		if ($scope.selectedDataProvider == undefined) {
			return false;
		}
		return _.contains($scope.selectedDataProvider.RequiredFields, 'Password');
	}

	$scope.showPlantno = function() {
		if ($scope.selectedDataProvider == undefined) {
			return false;
		}
		return _.contains($scope.selectedDataProvider.RequiredFields, 'PlantNo');
	}

	$scope.showAddress = function() {
		if ($scope.selectedDataProvider == undefined) {
			return false;
		}
		return _.contains($scope.selectedDataProvider.RequiredFields, 'Address');
	}




}

function UserCtrl($scope, My) {
	$scope.user = My
}

function AboutCtrl() {
	
}

function NewsCtrl() {

}
