package bootstrap

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestMaterialPublicacionExternaV3CompletoYPropio(t *testing.T) {
	materiales := materialesPublicadosPrueba(t, consumidoresPortalExternoV3...)
	defer func() {
		for _, lista := range materiales {
			borrarMaterialesV3PortalExterno(lista)
		}
	}()
	lista := ordenarMaterialExternoV3(materiales)
	if len(lista) != 28 || !materialPublicableExternoV3Exacto(materiales, lista) {
		t.Fatalf("audiencias publicables incompletas o divergentes: %d", len(lista))
	}
	anteriores, meritos, bolsa := 0, 0, 0
	for _, material := range lista {
		switch material.audienciaConsumo {
		case audienciaSubirComprobanteM4, audienciaDescargarComprobanteM4:
			meritos++
		case audienciaJustificanteBolsaSubir, audienciaJustificanteBolsaDescargar,
			audienciaJustificanteBolsaPreparar:
			bolsa++
		default:
			anteriores++
		}
	}
	if anteriores != 23 || meritos != 2 || bolsa != 3 {
		t.Fatal("la publicación perdió una audiencia anterior, de Méritos o de Bolsa")
	}
	coordenadas := make([]coordenadaClaveExternaV3, len(lista))
	for i, m := range lista {
		coordenadas[i] = coordenadaClaveExternaV3{m.audienciaConsumo, uint64(100 + i), uint64(100 + i), uint64(100 + i)}
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	if err := coordinarMaterialExternoV3(lista, "confianza:atestacion:externo:27", 27,
		ahora.Add(-time.Minute), ahora.Add(time.Hour), lista[0].validaDesde, lista[0].validaHasta, coordenadas); err != nil {
		t.Fatal(err)
	}
	propuesta, err := serializarMaterialExternoV3(lista)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(propuesta, []byte("clave:atestacion:ct:")) || bytes.Contains(propuesta, []byte("emisor:ct:")) {
		t.Fatal("la publicación conserva un identificador interno")
	}
	if !strings.HasPrefix(lista[0].claveID, "clave:atestacion:externo:") {
		t.Fatal("la raíz no es externa")
	}
	var contenido map[string]any
	if err := json.Unmarshal(propuesta, &contenido); err != nil {
		t.Fatal(err)
	}
	claves := contenido["claves"].([]any)
	for _, valor := range claves {
		delete(valor.(map[string]any), "secreto_hmac_hex")
	}
	publico, err := json.Marshal(contenido)
	if err != nil {
		t.Fatal(err)
	}
	coincide, err := publicoIgualExternoV3(propuesta, publico)
	if err != nil || !coincide {
		t.Fatal("la publicación pública no coincide")
	}
	claves[0].(map[string]any)["huella_secreto_sha256"] = strings.Repeat("0", 64)
	alterado, err := json.Marshal(contenido)
	if err != nil {
		t.Fatal(err)
	}
	coincide, err = publicoIgualExternoV3(propuesta, alterado)
	if err != nil || coincide {
		t.Fatal("replay divergente aceptado")
	}
	if coincide, err = publicoIgualExternoV3(propuesta, []byte("{")); err == nil || coincide {
		t.Fatal("JSON público malformado no comunica su error")
	}
}

func TestMaterialPublicacionExternaV3RechazaAudienciasIncompletasDuplicadasYAjena(t *testing.T) {
	base := materialesPublicadosPrueba(t, consumidoresPortalExternoV3...)
	defer func() {
		for _, lista := range base {
			borrarMaterialesV3PortalExterno(lista)
		}
	}()
	clonar := func() map[string][]materialAtestacionContratacionTemporalDesarrollo {
		copia := make(map[string][]materialAtestacionContratacionTemporalDesarrollo, len(base))
		for consumidor, lista := range base {
			copia[consumidor] = append([]materialAtestacionContratacionTemporalDesarrollo(nil), lista...)
		}
		return copia
	}
	if !materialPublicableExternoV3Exacto(base, ordenarMaterialExternoV3(base)) {
		t.Fatal("se rechazó el conjunto nominal completo")
	}
	casos := map[string]func(map[string][]materialAtestacionContratacionTemporalDesarrollo){
		"falta descarga documental": func(m map[string][]materialAtestacionContratacionTemporalDesarrollo) {
			m[consumidorMeritosCarteraPortalExternoV3] = m[consumidorMeritosCarteraPortalExternoV3][:3]
		},
		"falta audiencia anterior": func(m map[string][]materialAtestacionContratacionTemporalDesarrollo) {
			m["mi_bolsa"] = m["mi_bolsa"][:1]
		},
		"falta audiencia Bolsa": func(m map[string][]materialAtestacionContratacionTemporalDesarrollo) {
			m[consumidorJustificantesBolsaExterno] = m[consumidorJustificantesBolsaExterno][:2]
		},
		"duplicada": func(m map[string][]materialAtestacionContratacionTemporalDesarrollo) {
			m[consumidorMeritosCarteraPortalExternoV3][3].audienciaConsumo = m[consumidorMeritosCarteraPortalExternoV3][2].audienciaConsumo
		},
		"ajena": func(m map[string][]materialAtestacionContratacionTemporalDesarrollo) {
			m[consumidorMeritosCarteraPortalExternoV3][3].audienciaConsumo = "vec_documentos.audiencia_ajena.v1"
		},
		"Bolsa duplicada": func(m map[string][]materialAtestacionContratacionTemporalDesarrollo) {
			m[consumidorJustificantesBolsaExterno][2].audienciaConsumo = m[consumidorJustificantesBolsaExterno][1].audienciaConsumo
		},
		"consumidor ausente": func(m map[string][]materialAtestacionContratacionTemporalDesarrollo) {
			delete(m, "mi_bolsa")
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			materiales := clonar()
			mutar(materiales)
			if materialPublicableExternoV3Exacto(materiales, ordenarMaterialExternoV3(materiales)) {
				t.Fatal("se aceptó una publicación externa divergente")
			}
		})
	}
}

func TestAmpliacionPublicacionExternaV3ConservaHistoriaYReplay(t *testing.T) {
	materiales := materialesPublicadosPrueba(t, consumidoresPortalExternoV3...)
	defer func() {
		for _, piezas := range materiales {
			borrarMaterialesV3PortalExterno(piezas)
		}
	}()
	lista := ordenarMaterialExternoV3(materiales)
	anteriores := make([]*materialAtestacionContratacionTemporalDesarrollo, 0, len(lista)-3)
	coordenadasAnteriores := make([]coordenadaClaveExternaV3, 0, len(lista)-3)
	porAudiencia := map[string]coordenadaClaveExternaV3{}
	for _, pieza := range lista {
		if pieza.audienciaConsumo == audienciaJustificanteBolsaSubir ||
			pieza.audienciaConsumo == audienciaJustificanteBolsaDescargar ||
			pieza.audienciaConsumo == audienciaJustificanteBolsaPreparar {
			continue
		}
		coordenada := coordenadaClaveExternaV3{pieza.audienciaConsumo, uint64(100 + len(anteriores)),
			uint64(100 + len(anteriores)), uint64(100 + len(anteriores))}
		anteriores = append(anteriores, pieza)
		coordenadasAnteriores = append(coordenadasAnteriores, coordenada)
		porAudiencia[pieza.audienciaConsumo] = coordenada
	}
	if len(anteriores) != 25 {
		t.Fatal("la preimagen no contiene las 25 audiencias anteriores")
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	if err := coordinarMaterialExternoV3(anteriores, "confianza:atestacion:externo:27", 27,
		ahora.Add(-time.Minute), ahora.Add(2*time.Hour), lista[0].validaDesde, lista[0].validaHasta,
		coordenadasAnteriores); err != nil {
		t.Fatal(err)
	}
	publico := func(propuesta []byte) []byte {
		t.Helper()
		var documento map[string]any
		if err := json.Unmarshal(propuesta, &documento); err != nil {
			t.Fatal(err)
		}
		for _, clave := range documento["claves"].([]any) {
			delete(clave.(map[string]any), "secreto_hmac_hex")
		}
		resultado, err := json.Marshal(documento)
		if err != nil {
			t.Fatal(err)
		}
		return resultado
	}
	propuestaAnterior, err := serializarMaterialExternoV3(anteriores)
	if err != nil {
		t.Fatal(err)
	}
	publicadaAnterior := publico(propuestaAnterior)
	var actual materialPublicoExternoV3
	if err := json.Unmarshal(publicadaAnterior, &actual); err != nil {
		t.Fatal(err)
	}
	estado := estadoPublicacionExternaV3{Actual: publicadaAnterior,
		VersionSiguiente: 500, RevisionSiguiente: 500, OrdenSiguiente: 500, SecuenciaSiguiente: 28}
	coordenadas, err := coordenadasAmpliacionExternaV3(lista, actual, estado)
	if err != nil || len(coordenadas) != len(lista) {
		t.Fatalf("no se pudo ampliar la publicación: %v", err)
	}
	nuevas := 0
	for i, pieza := range lista {
		if anterior, existe := porAudiencia[pieza.audienciaConsumo]; existe {
			if coordenadas[i] != anterior {
				t.Fatal("una clave anterior cambió de coordenadas")
			}
			continue
		}
		if coordenadas[i].Version != 500+uint64(nuevas) ||
			coordenadas[i].Revision != 500+uint64(nuevas) ||
			coordenadas[i].Orden != 500+uint64(nuevas) {
			t.Fatal("las coordenadas nuevas no siguen el estado autoritativo")
		}
		nuevas++
	}
	if nuevas != 3 {
		t.Fatal("la ampliación no se limitó a las tres audiencias de Bolsa")
	}
	if err := coordinarMaterialExternoV3(lista, "confianza:atestacion:externo:28", 28,
		ahora, ahora.Add(time.Hour), actual.Raiz.ValidaDesde, actual.Raiz.ValidaHasta, coordenadas); err != nil {
		t.Fatal(err)
	}
	propuestaNueva, err := serializarMaterialExternoV3(lista)
	if err != nil {
		t.Fatal(err)
	}
	publicadaNueva := publico(propuestaNueva)
	var nueva materialPublicoExternoV3
	if err := json.Unmarshal(publicadaNueva, &nueva); err != nil {
		t.Fatal(err)
	}
	clavesPublicas := func(datos []byte) map[string]map[string]any {
		t.Helper()
		var documento map[string]any
		if err := json.Unmarshal(datos, &documento); err != nil {
			t.Fatal(err)
		}
		resultado := map[string]map[string]any{}
		for _, valor := range documento["claves"].([]any) {
			clave := valor.(map[string]any)
			resultado[clave["audiencia_consumo"].(string)] = clave
		}
		return resultado
	}
	antes, despues := clavesPublicas(publicadaAnterior), clavesPublicas(publicadaNueva)
	for audiencia := range antes {
		for _, campo := range []string{"clave_id", "version", "revision_gobierno", "orden", "huella_gobierno_sha256", "huella_secreto_sha256"} {
			if !reflect.DeepEqual(antes[audiencia][campo], despues[audiencia][campo]) {
				t.Fatalf("la clave anterior cambió %s", campo)
			}
		}
	}
	if err := coordinarMaterialExternoV3(lista, nueva.Configuracion.Revision, nueva.Configuracion.Secuencia,
		nueva.Configuracion.PublicadaEn, nueva.Configuracion.ExpiraEn, nueva.Raiz.ValidaDesde,
		nueva.Raiz.ValidaHasta, nueva.Claves); err != nil {
		t.Fatal(err)
	}
	replay, err := serializarMaterialExternoV3(lista)
	if err != nil {
		t.Fatal(err)
	}
	coincide, err := publicoIgualExternoV3(replay, publicadaNueva)
	if err != nil || !coincide {
		t.Fatal("repetir la publicación cambiaría las claves o sus punteros")
	}
	manipulada := append([]byte(nil), publicadaAnterior...)
	var adulterada map[string]any
	if err := json.Unmarshal(manipulada, &adulterada); err != nil {
		t.Fatal(err)
	}
	adulterada["claves"].([]any)[0].(map[string]any)["huella_secreto_sha256"] = strings.Repeat("0", 64)
	manipulada, err = json.Marshal(adulterada)
	if err != nil {
		t.Fatal(err)
	}
	estado.Actual = manipulada
	if _, err := coordenadasAmpliacionExternaV3(lista, actual, estado); err == nil {
		t.Fatal("se ampliaron claves cuya historia pública no coincide")
	}
}

func TestAmpliacionExternaV3Desde23SoloIncorporaMeritosYBolsa(t *testing.T) {
	materiales := materialesPublicadosPrueba(t, consumidoresPortalExternoV3...)
	defer func() {
		for _, piezas := range materiales {
			borrarMaterialesV3PortalExterno(piezas)
		}
	}()
	lista := ordenarMaterialExternoV3(materiales)
	if len(lista) != 28 {
		t.Fatalf("catálogo completo: %d", len(lista))
	}
	nuevas := map[string]bool{
		audienciaSubirComprobanteM4: true, audienciaDescargarComprobanteM4: true,
		audienciaJustificanteBolsaSubir: true, audienciaJustificanteBolsaDescargar: true,
		audienciaJustificanteBolsaPreparar: true,
	}
	anteriores := make([]*materialAtestacionContratacionTemporalDesarrollo, 0, 23)
	coordenadasAnteriores := make([]coordenadaClaveExternaV3, 0, 23)
	porAudiencia := map[string]coordenadaClaveExternaV3{}
	for _, pieza := range lista {
		if nuevas[pieza.audienciaConsumo] {
			continue
		}
		coord := coordenadaClaveExternaV3{pieza.audienciaConsumo, uint64(100 + len(anteriores)),
			uint64(100 + len(anteriores)), uint64(100 + len(anteriores))}
		anteriores = append(anteriores, pieza)
		coordenadasAnteriores = append(coordenadasAnteriores, coord)
		porAudiencia[pieza.audienciaConsumo] = coord
	}
	if len(anteriores) != 23 {
		t.Fatalf("preimagen histórica: %d", len(anteriores))
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	if err := coordinarMaterialExternoV3(anteriores, "confianza:atestacion:externo:27", 27,
		ahora.Add(-time.Minute), ahora.Add(2*time.Hour), lista[0].validaDesde, lista[0].validaHasta,
		coordenadasAnteriores); err != nil {
		t.Fatal(err)
	}
	propuesta, err := serializarMaterialExternoV3(anteriores)
	if err != nil {
		t.Fatal(err)
	}
	var publicado map[string]any
	if err := json.Unmarshal(propuesta, &publicado); err != nil {
		t.Fatal(err)
	}
	for _, clave := range publicado["claves"].([]any) {
		delete(clave.(map[string]any), "secreto_hmac_hex")
	}
	publico, err := json.Marshal(publicado)
	if err != nil {
		t.Fatal(err)
	}
	var actual materialPublicoExternoV3
	if err := json.Unmarshal(publico, &actual); err != nil {
		t.Fatal(err)
	}
	estado := estadoPublicacionExternaV3{Actual: publico, VersionSiguiente: 500,
		RevisionSiguiente: 500, OrdenSiguiente: 500, SecuenciaSiguiente: 28}
	coordenadas, err := coordenadasAmpliacionExternaV3(lista, actual, estado)
	if err != nil || len(coordenadas) != 28 {
		t.Fatalf("ampliación 23→28: %v", err)
	}
	contador := uint64(0)
	for i, pieza := range lista {
		if anterior, existe := porAudiencia[pieza.audienciaConsumo]; existe {
			if coordenadas[i] != anterior {
				t.Fatalf("coordenada histórica cambiada: %s", pieza.audienciaConsumo)
			}
			continue
		}
		if !nuevas[pieza.audienciaConsumo] || coordenadas[i].Version != 500+contador ||
			coordenadas[i].Revision != 500+contador || coordenadas[i].Orden != 500+contador {
			t.Fatalf("coordenada nueva inesperada: %s", pieza.audienciaConsumo)
		}
		contador++
	}
	if contador != 5 {
		t.Fatalf("audiencias nuevas: %d", contador)
	}
	clonar := func() materialPublicoExternoV3 {
		copia := actual
		copia.Claves = append([]coordenadaClaveExternaV3(nil), actual.Claves...)
		return copia
	}
	for nombre, mutar := range map[string]func(*materialPublicoExternoV3){
		"audiencia ajena":             func(a *materialPublicoExternoV3) { a.Claves[0].Audiencia = "vec_documentos.ajena.v1" },
		"audiencia duplicada":         func(a *materialPublicoExternoV3) { a.Claves[0].Audiencia = a.Claves[1].Audiencia },
		"clave nueva antes de tiempo": func(a *materialPublicoExternoV3) { a.Claves[0].Audiencia = audienciaJustificanteBolsaSubir },
		"clave anterior retirada":     func(a *materialPublicoExternoV3) { a.Claves = a.Claves[:22] },
	} {
		t.Run(nombre, func(t *testing.T) {
			adulterada := clonar()
			mutar(&adulterada)
			if _, err := coordenadasAmpliacionExternaV3(lista, adulterada, estado); err == nil {
				t.Fatal("ampliación divergente aceptada")
			}
		})
	}
	publicado["claves"].([]any)[0].(map[string]any)["huella_secreto_sha256"] = strings.Repeat("0", 64)
	adulterado, err := json.Marshal(publicado)
	if err != nil {
		t.Fatal(err)
	}
	estado.Actual = adulterado
	if _, err := coordenadasAmpliacionExternaV3(lista, actual, estado); err == nil {
		t.Fatal("se aceptó secreto histórico divergente")
	}
}
