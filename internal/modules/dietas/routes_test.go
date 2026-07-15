package dietas

import "testing"

func TestProvinceLocalitiesUsesGranadaINECatalog(t *testing.T) {
	localities := ProvinceLocalities()
	if len(localities) != 174 {
		t.Fatalf("localities = %d, want 174", len(localities))
	}
	assertLocality(t, localities, "18087", "Granada")
	assertLocality(t, localities, "18003", "Albolote")
	assertLocality(t, localities, "18140", "Motril")
	assertLocality(t, localities, "18904", "Alpujarra de la Sierra")
}

func TestProvinceRouteMatrixStatusCountsDirectedPairs(t *testing.T) {
	status := ProvinceRouteMatrixStatus()
	if got := status["directed_municipality_pairs"]; got != 30102 {
		t.Fatalf("directed_municipality_pairs = %v, want 30102", got)
	}
	if got := status["import_required_before_liquidation"]; got != true {
		t.Fatalf("import_required_before_liquidation = %v, want true", got)
	}
}

func TestProvinceRoutePointsIncludesMecinaBombaron(t *testing.T) {
	for _, point := range ProvinceRoutePoints() {
		if point.Name == "Mecina Bombarón" {
			if point.MunicipalityCode != "18904" || point.Kind != "nucleo" {
				t.Fatalf("Mecina Bombarón point = %#v", point)
			}
			if point.Latitude == 0 || point.Longitude == 0 {
				t.Fatalf("Mecina Bombarón coordinates not loaded: %#v", point)
			}
			return
		}
	}
	t.Fatal("Mecina Bombarón route point not found")
}

func TestProvinceRoutePointsAllHaveCoordinates(t *testing.T) {
	points := ProvinceRoutePoints()
	if len(points) != 175 {
		t.Fatalf("route points = %d, want 175", len(points))
	}
	for _, point := range points {
		if point.Latitude == 0 || point.Longitude == 0 {
			t.Fatalf("route point without coordinates: %#v", point)
		}
	}
}

func TestProvinceRouteItineraryExampleSumsLegs(t *testing.T) {
	examples := ProvinceRouteItineraryExamples()
	if len(examples) == 0 {
		t.Fatal("no itinerary examples")
	}
	example := examples[0]
	if len(example.Stops) != 5 || len(example.Legs) != 4 {
		t.Fatalf("example stops=%d legs=%d, want 5 stops and 4 legs", len(example.Stops), len(example.Legs))
	}
	if example.TotalKM <= 0 || example.TotalMinutes <= 0 || example.MileageAmountEUR <= 0 {
		t.Fatalf("example totals not calculated: %#v", example)
	}
}

func assertLocality(t *testing.T, localities []ProvinceLocality, code, name string) {
	t.Helper()
	for _, locality := range localities {
		if locality.INECode == code && locality.Name == name {
			return
		}
	}
	t.Fatalf("locality %s %s not found", code, name)
}
