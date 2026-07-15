package dietas

type ProvinceLocality struct {
	INECode string
	Name    string
	Kind    string
	Source  string
}

type RoutePoint struct {
	Code             string
	Name             string
	Kind             string
	MunicipalityCode string
	MunicipalityName string
	Latitude         float64
	Longitude        float64
	Source           string
	State            string
}

type RoutePair struct {
	ID            string
	From          string
	To            string
	FromCode      string
	ToCode        string
	DistanceKM    float64
	DurationMin   int
	Allowance     string
	MatrixVersion string
	Source        string
	State         string
}

type RouteLeg struct {
	From        string
	To          string
	DistanceKM  float64
	DurationMin int
	State       string
}

type RouteItinerary struct {
	ID                  string
	Label               string
	Stops               []string
	Legs                []RouteLeg
	TotalKM             float64
	TotalMinutes        int
	MileageRateEURKM    float64
	MileageAmountEUR    float64
	AllowanceSuggestion string
	MatrixVersion       string
	AuditState          string
}

func ProvinceLocalities() []ProvinceLocality {
	return append([]ProvinceLocality(nil), provinceLocalities...)
}

func ProvinceRoutePoints() []RoutePoint {
	localities := ProvinceLocalities()
	points := make([]RoutePoint, 0, len(localities)+1)
	for _, locality := range localities {
		coordinate := routePointCoordinates[locality.INECode]
		points = append(points, RoutePoint{
			Code:             locality.INECode,
			Name:             locality.Name,
			Kind:             locality.Kind,
			MunicipalityCode: locality.INECode,
			MunicipalityName: locality.Name,
			Latitude:         coordinate.Latitude,
			Longitude:        coordinate.Longitude,
			Source:           locality.Source,
			State:            "Vigente",
		})
	}
	coordinate := routePointCoordinates["NGMEP-MECINA-BOMBARON"]
	points = append(points, RoutePoint{
		Code:             "NGMEP-MECINA-BOMBARON",
		Name:             "Mecina Bombarón",
		Kind:             "nucleo",
		MunicipalityCode: "18904",
		MunicipalityName: "Alpujarra de la Sierra",
		Latitude:         coordinate.Latitude,
		Longitude:        coordinate.Longitude,
		Source:           "NGMEP/INE Nomenclator seed; importar codigo oficial en ETL",
		State:            "Pendiente importar NGMEP completo",
	})
	return points
}

func ProvinceRouteMatrixStatus() map[string]any {
	municipalities := len(ProvinceLocalities())
	routePoints := len(ProvinceRoutePoints())
	pairs := ProvinceRoutePairs()
	return map[string]any{
		"id":                                 "GR-DIETAS-MATRIX-2026-DESIGN",
		"state":                              "pendiente_importacion_completa",
		"scope":                              "Municipios INE y nucleos/entidades NGMEP para comisiones de servicio",
		"municipalities_loaded":              municipalities,
		"route_points_loaded":                routePoints,
		"routing_scope":                      "Provincia de Granada con buffer operativo de 15 km alrededor del limite provincial",
		"directed_municipality_pairs":        municipalities * (municipalities - 1),
		"directed_route_point_pairs":         routePoints * (routePoints - 1),
		"seed_pairs_loaded":                  len(pairs),
		"matrix_version":                     "pendiente-osrm-ngmep-2026",
		"locality_source":                    "INE codmun 2025 + CNIG/IGN NGMEP 2026",
		"routing_engine":                     "OSRM Route/Table service u openrouteservice Matrix on-premise",
		"graph_source":                       "NGMEP/CNIG para puntos y grafo viario OSM/CNIG versionado",
		"distance_policy":                    "Se guarda la distancia por carretera del itinerario aprobado; ajustes manuales exigen motivo y auditoria.",
		"refresh_policy":                     "Grafo OSRM actualizado diariamente si hay datos nuevos; maximo semanal. Las dietas cerradas conservan la version usada.",
		"routing_mode":                       "Servidor OSRM interno/on-premise; sin consultas externas para calculo ordinario.",
		"import_required_before_liquidation": true,
	}
}

func ProvinceRoutePairs() []RoutePair {
	return []RoutePair{
		routePair("R-GR-ALBOLOTE", "Granada", "Albolote", "18087", "18003", 13.4, 18, "sin dieta por defecto"),
		routePair("R-GR-MECINA", "Granada", "Mecina Bombarón", "18087", "NGMEP-MECINA-BOMBARON", 98.4, 93, "media dieta si hay manutencion"),
		routePair("R-MECINA-GR", "Mecina Bombarón", "Granada", "NGMEP-MECINA-BOMBARON", "18087", 98.4, 93, "media dieta si hay manutencion"),
		routePair("R-GR-LANJARON", "Granada", "Lanjarón", "18087", "18116", 45.9, 41, "sin dieta por defecto"),
		routePair("R-LANJARON-MECINA", "Lanjarón", "Mecina Bombarón", "18116", "NGMEP-MECINA-BOMBARON", 49.5, 61, "sin dieta por defecto"),
		routePair("R-GR-VELEZ-BENAUDALLA", "Granada", "Vélez de Benaudalla", "18087", "18184", 54.7, 47, "sin dieta por defecto"),
		routePair("R-VELEZ-BENAUDALLA-MECINA", "Vélez de Benaudalla", "Mecina Bombarón", "18184", "NGMEP-MECINA-BOMBARON", 52.0, 59, "sin dieta por defecto"),
		routePair("R-ALBOLOTE-MECINA", "Albolote", "Mecina Bombarón", "18003", "NGMEP-MECINA-BOMBARON", 124.6, 122, "sin dieta por defecto"),
		routePair("R-MECINA-MOTRIL", "Mecina Bombarón", "Motril", "NGMEP-MECINA-BOMBARON", "18140", 83.2, 96, "media dieta si hay manutencion"),
		routePair("R-MOTRIL-GR", "Motril", "Granada", "18140", "18087", 70.4, 55, "media dieta"),
		routePair("R-GR-MOTRIL", "Granada", "Motril", "18087", "18140", 70.4, 55, "media dieta"),
		routePair("R-GR-LOJA", "Granada", "Loja", "18087", "18122", 54.0, 45, "sin dieta por defecto"),
		routePair("R-GR-GUADIX", "Granada", "Guadix", "18087", "18089", 60.5, 50, "media dieta si hay manutencion"),
		routePair("R-GR-BAZA", "Granada", "Baza", "18087", "18023", 107.2, 80, "dieta completa segun horario"),
		routePair("R-GR-ALMUNECAR", "Granada", "Almunecar", "18087", "18017", 80.5, 65, "media dieta"),
		routePair("R-GR-ORGIVA", "Granada", "Orgiva", "18087", "18147", 55.8, 58, "media dieta si hay manutencion"),
	}
}

func ProvinceRouteItineraryExamples() []RouteItinerary {
	legs := []RouteLeg{
		routeLeg("Granada", "Albolote", 13.4, 18),
		routeLeg("Albolote", "Mecina Bombarón", 124.6, 122),
		routeLeg("Mecina Bombarón", "Motril", 83.2, 96),
		routeLeg("Motril", "Granada", 70.4, 55),
	}
	var totalKM float64
	var totalMinutes int
	for _, leg := range legs {
		totalKM += leg.DistanceKM
		totalMinutes += leg.DurationMin
	}
	rate := 0.26
	return []RouteItinerary{{
		ID:                  "IT-GR-ALBOLOTE-MECINA-MOTRIL-GR",
		Label:               "Granada -> Albolote -> Mecina Bombarón -> Motril -> Granada",
		Stops:               []string{"Granada", "Albolote", "Mecina Bombarón", "Motril", "Granada"},
		Legs:                legs,
		TotalKM:             totalKM,
		TotalMinutes:        totalMinutes,
		MileageRateEURKM:    rate,
		MileageAmountEUR:    totalKM * rate,
		AllowanceSuggestion: "Revisar media dieta/dieta completa por horario real y justificantes.",
		MatrixVersion:       "pendiente-osrm-ngmep-2026",
		AuditState:          "Ejemplo de flujo; requiere matriz oficial importada para liquidar.",
	}}
}

func ProvinceLocalityMaps() []map[string]any {
	localities := ProvinceLocalities()
	out := make([]map[string]any, 0, len(localities))
	for _, item := range localities {
		out = append(out, map[string]any{
			"ine_code": item.INECode,
			"name":     item.Name,
			"kind":     item.Kind,
			"source":   item.Source,
		})
	}
	return out
}

func ProvinceRoutePointMaps() []map[string]any {
	points := ProvinceRoutePoints()
	out := make([]map[string]any, 0, len(points))
	for _, item := range points {
		out = append(out, map[string]any{
			"code":              item.Code,
			"name":              item.Name,
			"kind":              item.Kind,
			"municipality_code": item.MunicipalityCode,
			"municipality_name": item.MunicipalityName,
			"lat":               item.Latitude,
			"lon":               item.Longitude,
			"source":            item.Source,
			"state":             item.State,
		})
	}
	return out
}

func ProvinceRoutePairMaps() []map[string]any {
	pairs := ProvinceRoutePairs()
	out := make([]map[string]any, 0, len(pairs))
	for _, item := range pairs {
		out = append(out, map[string]any{
			"id":                item.ID,
			"from":              item.From,
			"to":                item.To,
			"from_code":         item.FromCode,
			"to_code":           item.ToCode,
			"distance_km":       item.DistanceKM,
			"km_one_way":        item.DistanceKM,
			"duration_minutes":  item.DurationMin,
			"estimated_minutes": item.DurationMin,
			"allowance":         item.Allowance,
			"matrix_version":    item.MatrixVersion,
			"source":            item.Source,
			"state":             item.State,
		})
	}
	return out
}

func ProvinceRouteItineraryExampleMaps() []map[string]any {
	examples := ProvinceRouteItineraryExamples()
	out := make([]map[string]any, 0, len(examples))
	for _, item := range examples {
		legs := make([]map[string]any, 0, len(item.Legs))
		for _, leg := range item.Legs {
			legs = append(legs, map[string]any{
				"from":             leg.From,
				"to":               leg.To,
				"distance_km":      leg.DistanceKM,
				"duration_minutes": leg.DurationMin,
				"state":            leg.State,
			})
		}
		out = append(out, map[string]any{
			"id":                   item.ID,
			"label":                item.Label,
			"stops":                item.Stops,
			"legs":                 legs,
			"total_km":             item.TotalKM,
			"total_minutes":        item.TotalMinutes,
			"mileage_rate_eur_km":  item.MileageRateEURKM,
			"mileage_amount_eur":   item.MileageAmountEUR,
			"allowance_suggestion": item.AllowanceSuggestion,
			"matrix_version":       item.MatrixVersion,
			"audit_state":          item.AuditState,
		})
	}
	return out
}

func routePair(id, from, to, fromCode, toCode string, distanceKM float64, durationMin int, allowance string) RoutePair {
	return RoutePair{
		ID:            id,
		From:          from,
		To:            to,
		FromCode:      fromCode,
		ToCode:        toCode,
		DistanceKM:    distanceKM,
		DurationMin:   durationMin,
		Allowance:     allowance,
		MatrixVersion: "pendiente-osrm-ngmep-2026",
		Source:        "Semilla demo para UI; recalcular con matriz oficial antes de liquidar",
		State:         "Pendiente validacion interna",
	}
}

func routeLeg(from, to string, distanceKM float64, durationMin int) RouteLeg {
	return RouteLeg{
		From:        from,
		To:          to,
		DistanceKM:  distanceKM,
		DurationMin: durationMin,
		State:       "Pendiente validacion interna",
	}
}

type routeCoordinate struct {
	Latitude  float64
	Longitude float64
}

// Coordenadas municipales procedentes del NGMEP/CNIG 2026 (ETRS89, compatible WGS84),
// cruzadas por COD_INE con el catalogo de municipios INE cargado en VEC.
var routePointCoordinates = map[string]routeCoordinate{
	"18001":                 {Latitude: 37.03063867, Longitude: -3.83004916},
	"18002":                 {Latitude: 37.58156120, Longitude: -3.24449019},
	"18003":                 {Latitude: 37.23058226, Longitude: -3.65728557},
	"18004":                 {Latitude: 36.82816885, Longitude: -3.21079723},
	"18005":                 {Latitude: 37.22709331, Longitude: -3.13342972},
	"18006":                 {Latitude: 36.79127420, Longitude: -3.20326737},
	"18007":                 {Latitude: 36.92857586, Longitude: -3.62313875},
	"18010":                 {Latitude: 37.16358722, Longitude: -3.07201821},
	"18011":                 {Latitude: 37.23669974, Longitude: -3.57068571},
	"18012":                 {Latitude: 37.32484097, Longitude: -4.15879969},
	"18013":                 {Latitude: 37.00256185, Longitude: -3.98788680},
	"18014":                 {Latitude: 37.10792515, Longitude: -3.64593634},
	"18015":                 {Latitude: 37.60804008, Longitude: -3.13738910},
	"18016":                 {Latitude: 36.90251905, Longitude: -3.29964711},
	"18017":                 {Latitude: 36.73383574, Longitude: -3.69115671},
	"18904":                 {Latitude: 36.98090162, Longitude: -3.15553334},
	"18018":                 {Latitude: 37.17886674, Longitude: -3.11509372},
	"18020":                 {Latitude: 36.95784841, Longitude: -3.89409526},
	"18021":                 {Latitude: 37.14282172, Longitude: -3.62781714},
	"18022":                 {Latitude: 37.22270525, Longitude: -3.68650127},
	"18023":                 {Latitude: 37.49061681, Longitude: -2.77431128},
	"18024":                 {Latitude: 37.21867330, Longitude: -3.48116385},
	"18025":                 {Latitude: 37.27970652, Longitude: -3.20497355},
	"18027":                 {Latitude: 37.35012457, Longitude: -3.16906979},
	"18028":                 {Latitude: 37.43168175, Longitude: -3.68262161},
	"18029":                 {Latitude: 37.60891437, Longitude: -2.69878440},
	"18030":                 {Latitude: 36.97420183, Longitude: -3.19025477},
	"18032":                 {Latitude: 36.94884587, Longitude: -3.35677315},
	"18033":                 {Latitude: 36.93785051, Longitude: -3.29447978},
	"18034":                 {Latitude: 37.05988585, Longitude: -3.91703538},
	"18035":                 {Latitude: 36.94643627, Longitude: -3.18040825},
	"18036":                 {Latitude: 37.13418958, Longitude: -3.57096890},
	"18114":                 {Latitude: 37.18138299, Longitude: -3.06423290},
	"18037":                 {Latitude: 37.27324113, Longitude: -3.61852534},
	"18038":                 {Latitude: 37.48146300, Longitude: -3.61658808},
	"18039":                 {Latitude: 37.43611357, Longitude: -2.72228389},
	"18040":                 {Latitude: 36.92612508, Longitude: -3.42768277},
	"18042":                 {Latitude: 36.96145234, Longitude: -3.35850854},
	"18043":                 {Latitude: 36.92287615, Longitude: -3.40825313},
	"18044":                 {Latitude: 36.93140224, Longitude: -3.25369514},
	"18045":                 {Latitude: 37.71464276, Longitude: -2.64298654},
	"18046":                 {Latitude: 37.79571995, Longitude: -2.77915678},
	"18047":                 {Latitude: 37.15953035, Longitude: -3.53856109},
	"18059":                 {Latitude: 37.20132742, Longitude: -3.77242334},
	"18061":                 {Latitude: 37.13113614, Longitude: -3.82328969},
	"18062":                 {Latitude: 37.14768759, Longitude: -3.64602593},
	"18048":                 {Latitude: 37.19993201, Longitude: -3.81062098},
	"18049":                 {Latitude: 37.22415925, Longitude: -3.16092411},
	"18050":                 {Latitude: 37.27546810, Longitude: -3.57345303},
	"18051":                 {Latitude: 37.36901228, Longitude: -3.71263211},
	"18053":                 {Latitude: 37.65408144, Longitude: -2.76972060},
	"18054":                 {Latitude: 37.30371498, Longitude: -3.21886070},
	"18912":                 {Latitude: 37.60857341, Longitude: -2.93070889},
	"18056":                 {Latitude: 37.58370534, Longitude: -2.57642710},
	"18057":                 {Latitude: 37.15326572, Longitude: -3.67051880},
	"18063":                 {Latitude: 37.34918984, Longitude: -3.29252748},
	"18064":                 {Latitude: 37.58996707, Longitude: -3.10260327},
	"18065":                 {Latitude: 37.47261088, Longitude: -3.55186793},
	"18066":                 {Latitude: 37.32654190, Longitude: -3.59434400},
	"18067":                 {Latitude: 37.31957949, Longitude: -3.33249487},
	"18068":                 {Latitude: 37.07527987, Longitude: -3.60208588},
	"18069":                 {Latitude: 37.17840303, Longitude: -2.99149977},
	"18915":                 {Latitude: 37.49729442, Longitude: -3.50842806},
	"18070":                 {Latitude: 37.18596540, Longitude: -3.48362270},
	"18071":                 {Latitude: 36.98792154, Longitude: -3.56601289},
	"18072":                 {Latitude: 37.06179526, Longitude: -3.76130988},
	"18074":                 {Latitude: 37.17239422, Longitude: -3.03593384},
	"18076":                 {Latitude: 37.41353990, Longitude: -3.17335080},
	"18077":                 {Latitude: 36.95444333, Longitude: -3.85549252},
	"18078":                 {Latitude: 37.52900485, Longitude: -2.90689898},
	"18079":                 {Latitude: 37.21951067, Longitude: -3.78298965},
	"18905":                 {Latitude: 37.13646916, Longitude: -3.66926298},
	"18082":                 {Latitude: 37.74282232, Longitude: -2.55118710},
	"18083":                 {Latitude: 37.47743928, Longitude: -3.31999534},
	"18084":                 {Latitude: 37.10443754, Longitude: -3.60606948},
	"18085":                 {Latitude: 37.36960313, Longitude: -2.96945607},
	"18086":                 {Latitude: 37.47950845, Longitude: -3.04292054},
	"18087":                 {Latitude: 37.17428891, Longitude: -3.59869101},
	"18088":                 {Latitude: 37.55657630, Longitude: -3.40081448},
	"18089":                 {Latitude: 37.30059972, Longitude: -3.13534312},
	"18906":                 {Latitude: 36.84156767, Longitude: -3.58420676},
	"18093":                 {Latitude: 36.74354622, Longitude: -3.38938135},
	"18094":                 {Latitude: 37.15993693, Longitude: -3.43854344},
	"18095":                 {Latitude: 37.25679432, Longitude: -3.59772406},
	"18096":                 {Latitude: 37.42075094, Longitude: -3.26214028},
	"18097":                 {Latitude: 37.17705820, Longitude: -2.94839436},
	"18098":                 {Latitude: 37.80945284, Longitude: -2.53959067},
	"18099":                 {Latitude: 37.21833335, Longitude: -3.51722220},
	"18100":                 {Latitude: 37.19405990, Longitude: -4.04691990},
	"18101":                 {Latitude: 37.14564101, Longitude: -3.56961716},
	"18102":                 {Latitude: 37.28853114, Longitude: -3.87967343},
	"18103":                 {Latitude: 36.79962817, Longitude: -3.63868190},
	"18105":                 {Latitude: 37.39297200, Longitude: -3.52771300},
	"18106":                 {Latitude: 36.93416396, Longitude: -3.90939826},
	"18107":                 {Latitude: 36.94890457, Longitude: -3.82281540},
	"18108":                 {Latitude: 37.18393621, Longitude: -3.15986954},
	"18109":                 {Latitude: 36.79734016, Longitude: -3.66792939},
	"18111":                 {Latitude: 37.22201353, Longitude: -3.59437470},
	"18112":                 {Latitude: 36.94809514, Longitude: -3.22554717},
	"18115":                 {Latitude: 37.19509618, Longitude: -3.83397312},
	"18116":                 {Latitude: 36.91805310, Longitude: -3.48049282},
	"18117":                 {Latitude: 37.16886580, Longitude: -3.13843810},
	"18119":                 {Latitude: 36.94803319, Longitude: -3.55061954},
	"18120":                 {Latitude: 36.83438894, Longitude: -3.67456880},
	"18121":                 {Latitude: 36.92947048, Longitude: -3.21291516},
	"18122":                 {Latitude: 37.16638900, Longitude: -4.14995700},
	"18123":                 {Latitude: 37.22935889, Longitude: -3.24133280},
	"18124":                 {Latitude: 36.78665161, Longitude: -3.40454062},
	"18126":                 {Latitude: 37.10147367, Longitude: -3.72309534},
	"18127":                 {Latitude: 37.20752705, Longitude: -3.63295020},
	"18128":                 {Latitude: 37.29632190, Longitude: -3.20236212},
	"18132":                 {Latitude: 37.34008729, Longitude: -3.78577327},
	"18133":                 {Latitude: 36.78689549, Longitude: -3.60749385},
	"18134":                 {Latitude: 37.13227970, Longitude: -3.53974599},
	"18135":                 {Latitude: 37.32084970, Longitude: -4.01125848},
	"18136":                 {Latitude: 37.57186392, Longitude: -3.50444969},
	"18137":                 {Latitude: 37.49991497, Longitude: -3.67221159},
	"18138":                 {Latitude: 37.16984217, Longitude: -3.96501406},
	"18909":                 {Latitude: 37.43988697, Longitude: -3.33221580},
	"18140":                 {Latitude: 36.74535308, Longitude: -3.52045559},
	"18141":                 {Latitude: 36.88731788, Longitude: -3.10827114},
	"18903":                 {Latitude: 37.00834695, Longitude: -3.01460527},
	"18143":                 {Latitude: 36.97634968, Longitude: -3.54049609},
	"18144":                 {Latitude: 37.25790951, Longitude: -3.57796848},
	"18145":                 {Latitude: 37.12005587, Longitude: -3.60691450},
	"18146":                 {Latitude: 37.72131030, Longitude: -2.47927728},
	"18147":                 {Latitude: 36.90216652, Longitude: -3.42396210},
	"18148":                 {Latitude: 36.81315909, Longitude: -3.67923185},
	"18150":                 {Latitude: 37.02229481, Longitude: -3.62730064},
	"18151":                 {Latitude: 36.94004470, Longitude: -3.36141480},
	"18152":                 {Latitude: 37.50161842, Longitude: -3.23053483},
	"18153":                 {Latitude: 37.23082165, Longitude: -3.62863363},
	"18154":                 {Latitude: 37.27565791, Longitude: -3.28483150},
	"18910":                 {Latitude: 36.89472225, Longitude: -3.52657962},
	"18157":                 {Latitude: 37.16373079, Longitude: -3.50212023},
	"18158":                 {Latitude: 37.25165273, Longitude: -3.74939735},
	"18159":                 {Latitude: 37.44279504, Longitude: -3.44058338},
	"18161":                 {Latitude: 37.25744704, Longitude: -3.23312110},
	"18162":                 {Latitude: 36.79588888, Longitude: -3.29721867},
	"18163":                 {Latitude: 36.94244346, Longitude: -3.31031685},
	"18164":                 {Latitude: 37.95797427, Longitude: -2.43482739},
	"18165":                 {Latitude: 37.22264505, Longitude: -3.60796721},
	"18167":                 {Latitude: 37.31479514, Longitude: -3.18938899},
	"18168":                 {Latitude: 37.19202852, Longitude: -3.46649968},
	"18170":                 {Latitude: 36.80910400, Longitude: -3.34794254},
	"18171":                 {Latitude: 37.15230761, Longitude: -4.06694152},
	"18173":                 {Latitude: 36.74672265, Longitude: -3.58723518},
	"18174":                 {Latitude: 37.06053379, Longitude: -3.97630456},
	"18175":                 {Latitude: 37.18956690, Longitude: -3.71814502},
	"18176":                 {Latitude: 36.92846282, Longitude: -3.40505353},
	"18177":                 {Latitude: 36.79466342, Longitude: -3.26533298},
	"18901":                 {Latitude: 36.93607523, Longitude: -3.32553743},
	"18178":                 {Latitude: 37.50475199, Longitude: -3.35584054},
	"18916":                 {Latitude: 36.70193595, Longitude: -3.48589156},
	"18179":                 {Latitude: 36.87857177, Longitude: -3.29866156},
	"18180":                 {Latitude: 37.00241942, Longitude: -3.26663940},
	"18181":                 {Latitude: 36.86359278, Longitude: -3.05772346},
	"18182":                 {Latitude: 36.96108192, Longitude: -3.05455006},
	"18914":                 {Latitude: 37.23444418, Longitude: -3.82380108},
	"18907":                 {Latitude: 37.26255310, Longitude: -3.10009391},
	"18902":                 {Latitude: 36.92906603, Longitude: -3.58285575},
	"18183":                 {Latitude: 36.99631868, Longitude: -3.08080592},
	"18911":                 {Latitude: 37.17165065, Longitude: -3.66745184},
	"18184":                 {Latitude: 36.83218852, Longitude: -3.51588284},
	"18185":                 {Latitude: 37.06681074, Longitude: -3.82112385},
	"18149":                 {Latitude: 37.09434358, Longitude: -3.63478326},
	"18908":                 {Latitude: 36.99106200, Longitude: -3.58917985},
	"18187":                 {Latitude: 37.55665242, Longitude: -3.08962604},
	"18188":                 {Latitude: 37.21481606, Longitude: -4.01270540},
	"18189":                 {Latitude: 37.23124745, Longitude: -3.55381866},
	"18192":                 {Latitude: 36.97369203, Longitude: -4.14168111},
	"18913":                 {Latitude: 37.25359674, Longitude: -4.16806138},
	"18193":                 {Latitude: 37.12061942, Longitude: -3.58498469},
	"18194":                 {Latitude: 37.54136176, Longitude: -2.84176313},
	"NGMEP-MECINA-BOMBARON": {Latitude: 36.94600000, Longitude: -3.15470000},
}

var provinceLocalities = []ProvinceLocality{
	{INECode: "18001", Name: "Agrón", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18002", Name: "Alamedilla", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18003", Name: "Albolote", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18004", Name: "Albondón", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18005", Name: "Albuñán", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18006", Name: "Albuñol", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18007", Name: "Albuñuelas", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18010", Name: "Aldeire", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18011", Name: "Alfacar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18012", Name: "Algarinejo", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18013", Name: "Alhama de Granada", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18014", Name: "Alhendín", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18015", Name: "Alicún de Ortega", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18016", Name: "Almegíjar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18017", Name: "Almuñécar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18904", Name: "Alpujarra de la Sierra", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18018", Name: "Alquife", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18020", Name: "Arenas del Rey", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18021", Name: "Armilla", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18022", Name: "Atarfe", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18023", Name: "Baza", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18024", Name: "Beas de Granada", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18025", Name: "Beas de Guadix", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18027", Name: "Benalúa", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18028", Name: "Benalúa de las Villas", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18029", Name: "Benamaurel", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18030", Name: "Bérchules", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18032", Name: "Bubión", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18033", Name: "Busquístar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18034", Name: "Cacín", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18035", Name: "Cádiar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18036", Name: "Cájar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18114", Name: "Calahorra, La", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18037", Name: "Calicasas", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18038", Name: "Campotéjar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18039", Name: "Caniles", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18040", Name: "Cáñar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18042", Name: "Capileira", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18043", Name: "Carataunas", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18044", Name: "Cástaras", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18045", Name: "Castilléjar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18046", Name: "Castril", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18047", Name: "Cenes de la Vega", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18059", Name: "Chauchina", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18061", Name: "Chimeneas", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18062", Name: "Churriana de la Vega", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18048", Name: "Cijuela", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18049", Name: "Cogollos de Guadix", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18050", Name: "Cogollos de la Vega", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18051", Name: "Colomera", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18053", Name: "Cortes de Baza", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18054", Name: "Cortes y Graena", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18912", Name: "Cuevas del Campo", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18056", Name: "Cúllar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18057", Name: "Cúllar Vega", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18063", Name: "Darro", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18064", Name: "Dehesas de Guadix", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18065", Name: "Dehesas Viejas", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18066", Name: "Deifontes", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18067", Name: "Diezma", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18068", Name: "Dílar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18069", Name: "Dólar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18915", Name: "Domingo Pérez de Granada", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18070", Name: "Dúdar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18071", Name: "Dúrcal", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18072", Name: "Escúzar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18074", Name: "Ferreira", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18076", Name: "Fonelas", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18077", Name: "Fornes", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18078", Name: "Freila", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18079", Name: "Fuente Vaqueros", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18905", Name: "Gabias, Las", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18082", Name: "Galera", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18083", Name: "Gobernador", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18084", Name: "Gójar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18085", Name: "Gor", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18086", Name: "Gorafe", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18087", Name: "Granada", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18088", Name: "Guadahortuna", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18089", Name: "Guadix", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18906", Name: "Guájares, Los", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18093", Name: "Gualchos", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18094", Name: "Güéjar Sierra", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18095", Name: "Güevéjar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18096", Name: "Huélago", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18097", Name: "Huéneja", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18098", Name: "Huéscar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18099", Name: "Huétor de Santillán", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18100", Name: "Huétor Tájar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18101", Name: "Huétor Vega", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18102", Name: "Íllora", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18103", Name: "Ítrabo", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18105", Name: "Iznalloz", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18106", Name: "Játar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18107", Name: "Jayena", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18108", Name: "Jérez del Marquesado", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18109", Name: "Jete", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18111", Name: "Jun", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18112", Name: "Juviles", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18115", Name: "Láchar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18116", Name: "Lanjarón", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18117", Name: "Lanteira", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18119", Name: "Lecrín", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18120", Name: "Lentegí", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18121", Name: "Lobras", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18122", Name: "Loja", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18123", Name: "Lugros", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18124", Name: "Lújar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18126", Name: "Malahá, La", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18127", Name: "Maracena", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18128", Name: "Marchal", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18132", Name: "Moclín", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18133", Name: "Molvízar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18134", Name: "Monachil", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18135", Name: "Montefrío", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18136", Name: "Montejícar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18137", Name: "Montillana", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18138", Name: "Moraleda de Zafayona", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18909", Name: "Morelábor", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18140", Name: "Motril", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18141", Name: "Murtas", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18903", Name: "Nevada", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18143", Name: "Nigüelas", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18144", Name: "Nívar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18145", Name: "Ogíjares", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18146", Name: "Orce", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18147", Name: "Órgiva", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18148", Name: "Otívar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18150", Name: "Padul", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18151", Name: "Pampaneira", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18152", Name: "Pedro Martínez", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18153", Name: "Peligros", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18154", Name: "Peza, La", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18910", Name: "Pinar, El", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18157", Name: "Pinos Genil", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18158", Name: "Pinos Puente", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18159", Name: "Píñar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18161", Name: "Polícar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18162", Name: "Polopos", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18163", Name: "Pórtugos", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18164", Name: "Puebla de Don Fadrique", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18165", Name: "Pulianas", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18167", Name: "Purullena", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18168", Name: "Quéntar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18170", Name: "Rubite", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18171", Name: "Salar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18173", Name: "Salobreña", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18174", Name: "Santa Cruz del Comercio", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18175", Name: "Santa Fe", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18176", Name: "Soportújar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18177", Name: "Sorvilán", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18901", Name: "Taha, La", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18178", Name: "Torre-Cardela", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18916", Name: "Torrenueva Costa", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18179", Name: "Torvizcón", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18180", Name: "Trevélez", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18181", Name: "Turón", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18182", Name: "Ugíjar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18914", Name: "Valderrubio", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18907", Name: "Valle del Zalabí", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18902", Name: "Valle, El", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18183", Name: "Válor", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18911", Name: "Vegas del Genil", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18184", Name: "Vélez de Benaudalla", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18185", Name: "Ventas de Huelma", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18149", Name: "Villa de Otura", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18908", Name: "Villamena", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18187", Name: "Villanueva de las Torres", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18188", Name: "Villanueva Mesía", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18189", Name: "Víznar", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18192", Name: "Zafarraya", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18913", Name: "Zagra", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18193", Name: "Zubia, La", Kind: "municipio", Source: "INE codmun 2025"},
	{INECode: "18194", Name: "Zújar", Kind: "municipio", Source: "INE codmun 2025"},
}
