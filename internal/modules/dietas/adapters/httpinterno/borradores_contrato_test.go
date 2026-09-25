package httpinterno

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

// Prueba de contrato Go → web: el JSON real que produce resultadoAJSON para
// documentos v2 en varios estados se vuelca a un fixture que valida
// cliente-borradores-http.js con el cliente real
// (contrato-documento-go.test.mjs). Si cambia la salida, regenerarlo con
// DIETAS_ACTUALIZAR_CONTRATO=1 go test -run TestContratoDocumentoPropioV2 y
// revisar el diff antes de confirmarlo.
const fixtureContratoDocumento = "testdata/documento_propio_v2.json"

func comisionDocumentoV2Contrato(t *testing.T) domain.ComisionBorrador {
	t.Helper()
	version := "provisional:rd462:20260923"
	regla := domain.ReglaDevengoProvisional{ReglaRef: "provisional:regla:nacional-ordinaria:20260923", VersionTarifaRef: version, PaisISO2: "ES",
		Variante: "nacional_ordinaria", HuellaSHA256: strings.Repeat("a", 64), Configuracion: domain.ConfiguracionDevengoProvisional{
			Regla: "nacional_ordinaria_provisional_v1", Zona: "Europe/Madrid", DuracionMinimaMismoDiaHoras: 5, HoraSalida100AntesDe: 14,
			HoraSalida50AntesDe: 22, HoraRegreso50DespuesDe: 14, HoraRegresoMismoDiaDespuesDe: 16, DiasMaximos: 31,
			Alojamiento: "tope_pendiente_justificante", PorcentajeMismoDia: 50, PorcentajeSalidaTemprana: 100, PorcentajeSalidaMedia: 50,
			PorcentajeRegreso: 50, PorcentajeIntermedio: 100, PorcentajeAlojamientoTope: 100}}
	codigos := []string{"ruta:a", "ruta:b"}
	rutas := []domain.RutaDeclaradaComision{{CodigosRuta: codigos, AjusteKilometros: "0.0000"}}
	calculo := domain.CalculoComision{Procedencia: "osrm_interno", VersionGrafo: "grafo:prueba", Motor: "OSRM", VersionTarifa: version,
		Rotulo: domain.RotuloTarifaProvisional, ReglaRef: regla.ReglaRef, ReglaHuellaSHA256: regla.HuellaSHA256, HoraInicio: "08:00", HoraFin: "18:00",
		Kilometros: "1.0000", EURPorKM: "0.2600", ImporteKilometrajeCentimos: 26, TramosRuta: []domain.TramoRutaComision{}, VehiculoPropio: true,
		Rutas: []domain.RutaCalculadaComision{{CodigosRuta: codigos, VersionGrafo: "grafo:prueba", TramosRuta: []domain.TramoRutaComision{{OrigenCodigo: "ruta:a", DestinoCodigo: "ruta:b", Kilometros: "1.0000"}},
			KilometrosBase: "1.0000", AjusteKilometros: "0.0000", KilometrosFinales: "1.0000", ImporteCentimos: 26}}}
	zona, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	inicio, err := domain.ResolverInstanteCivil("2026-09-21", "08:00", zona)
	if err != nil {
		t.Fatal(err)
	}
	fin, err := domain.ResolverInstanteCivil("2026-09-22", "18:00", zona)
	if err != nil {
		t.Fatal(err)
	}
	for grupo := 1; grupo <= 3; grupo++ {
		tramos, err := domain.CalcularTramosNacionalesProvisionales(inicio, fin, zona, domain.TarifaNacionalProvisional{VersionRef: version,
			Rotulo: domain.RotuloTarifaProvisional, PaisISO2: "ES", Grupo: grupo, VigenteDesde: "2026-09-01",
			ManutencionCentimos: int64(1000 * grupo), AlojamientoTopeCentimos: int64(2000 * grupo)}, regla)
		if err != nil {
			t.Fatal(err)
		}
		calculo.OpcionesDieta = append(calculo.OpcionesDieta, domain.OpcionDietaComision{Grupo: grupo, Calculo: tramos})
	}
	aceptados := make([]int, len(calculo.OpcionesDieta[1].Calculo.Tramos))
	for i := range aceptados {
		aceptados[i] = i
	}
	otros := []domain.OtroGastoDeclarado{{Tipo: domain.ClaseOtroMedio, TipoGasto: "taxi", CatalogoVersion: domain.VersionCatalogoOtrosGastos,
		Fecha: "2026-09-21", Concepto: "Taxi estación a sede", ImporteCentimos: 1250, JustificanteRef: "ticket:taxi-0921", JustificanteSHA256: strings.Repeat("b", 64)}}
	documento, err := domain.ConstruirDocumentoComision(calculo, codigos, rutas, true, 2, aceptados, version, otros)
	if err != nil {
		t.Fatal(err)
	}
	vehiculo := true
	c := domain.ComisionBorrador{Referencia: "dco_" + strings.Repeat("D", 22), NumeroDocumento: "VEC-D-2026-000001",
		FechaApertura: "2026-09-20T08:00:00.000000Z", Version: 4, Estado: domain.EstadoDevuelta, FechaInicio: "2026-09-21", FechaFin: "2026-09-22",
		Motivo: "Visita técnica", CodigosRuta: codigos, RelacionRef: "rel_" + strings.Repeat("R", 22), CentroRef: "centro:uno", UnidadRef: "unidad:uno",
		Calculo: &calculo, Documento: &documento, VehiculoPropio: &vehiculo, Rutas: &rutas,
		Devolucion: &domain.DevolucionComision{Etapa: domain.EtapaAutorizacion, Motivo: "Falta el justificante del taxi", Version: 4, DevueltaEn: "2026-09-23T09:00:00.123456Z"}}
	if err := c.Validar(); err != nil {
		t.Fatalf("documento v2 de contrato inválido: %v", err)
	}
	return c
}

func TestContratoDocumentoPropioV2(t *testing.T) {
	devuelta := comisionDocumentoV2Contrato(t)
	corregida := devuelta
	corregida.Estado, corregida.Version = "borrador", 5
	reenviada := devuelta
	reenviada.Estado, reenviada.Version, reenviada.Devolucion = domain.EstadoEnviadoPendienteRevision, 6, nil
	nueva := domain.ComisionBorrador{Referencia: "dco_" + strings.Repeat("B", 22), Estado: "borrador", FechaInicio: "2026-09-24", FechaFin: "2026-09-24",
		Motivo: "Borrador nuevo", RelacionRef: "rel_" + strings.Repeat("R", 22)}
	// Recibo tal como lo entregan los adaptadores PostgreSQL de lectura y
	// mutación; el recibo nunca lleva la regla (va en comision.calculo).
	recibo := func(version uint64, repeticion bool) dietasports.ReciboBorradorComision {
		return dietasports.ReciboBorradorComision{Referencia: "rcd_" + strings.Repeat("0", 22), Version: version,
			RegistradoEn: time.Date(2026, 9, 23, 9, 0, 0, 123456000, time.UTC), Repeticion: repeticion}
	}
	casos := []struct {
		Nombre    string                `json:"nombre"`
		Resultado resultadoBorradorJSON `json:"resultado"`
	}{
		{"devuelta", resultadoAJSON(dietasports.ResultadoBorradorComision{Comision: devuelta, Recibo: recibo(4, true)})},
		{"en_correccion", resultadoAJSON(dietasports.ResultadoBorradorComision{Comision: corregida, Recibo: recibo(5, false)})},
		{"reenviada", resultadoAJSON(dietasports.ResultadoBorradorComision{Comision: reenviada, Recibo: recibo(6, false)})},
		{"borrador_nuevo", resultadoAJSON(dietasports.ResultadoBorradorComision{Comision: nueva, Recibo: recibo(1, false)})},
	}
	for _, caso := range casos {
		if err := caso.Resultado.Comision.Validar(); err != nil {
			t.Fatalf("%s: la comisión del contrato no valida en Go: %v", caso.Nombre, err)
		}
	}
	salida, err := json.MarshalIndent(casos, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	salida = append(salida, '\n')
	if os.Getenv("DIETAS_ACTUALIZAR_CONTRATO") == "1" {
		if err := os.WriteFile(fixtureContratoDocumento, salida, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	esperado, err := os.ReadFile(fixtureContratoDocumento)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(salida, esperado) {
		t.Fatalf("cambió el JSON real de resultadoAJSON; regenere %s y revise el cliente web:\n%s", fixtureContratoDocumento, salida)
	}
	for _, requerido := range []string{`"devolucion"`, `"documento"`, `"lineas"`, `"centro_ref": "centro:uno"`, `"estado": "devuelta"`, `"codigos_ruta": []`} {
		if !strings.Contains(string(salida), requerido) {
			t.Fatalf("falta %s en el contrato", requerido)
		}
	}
	// El recibo del fixture tiene exactamente las claves que admite el
	// cliente; contrato-documento-go.test.mjs lo comprueba con el cliente real.
	var leidos []struct {
		Resultado struct {
			Recibo map[string]json.RawMessage `json:"recibo"`
		} `json:"resultado"`
	}
	if err := json.Unmarshal(salida, &leidos); err != nil {
		t.Fatal(err)
	}
	for i, leido := range leidos {
		if claves := clavesOrdenadas(leido.Resultado.Recibo); strings.Join(claves, ",") != strings.Join(clavesReciboCliente, ",") {
			t.Fatalf("%s: el recibo HTTP lleva %v; el cliente web solo admite %v", casos[i].Nombre, claves, clavesReciboCliente)
		}
	}
}

// clavesReciboCliente son las cuatro claves del recibo que acepta
// validarRecibo en cliente-borradores-http.js, ordenadas.
var clavesReciboCliente = []string{"referencia", "registrado_en", "repeticion", "version"}

func clavesOrdenadas(m map[string]json.RawMessage) []string {
	claves := make([]string, 0, len(m))
	for clave := range m {
		claves = append(claves, clave)
	}
	sort.Strings(claves)
	return claves
}

// El recibo HTTP no puede ganar claves sin cambiar también el cliente web:
// las etiquetas JSON del tipo son exactamente las del cliente y ninguna es
// omitempty (siempre se emiten las cuatro).
func TestReciboHTTPSoloClavesDelCliente(t *testing.T) {
	tipo := reflect.TypeOf(reciboBorradorJSON{})
	etiquetas := make([]string, 0, tipo.NumField())
	for i := 0; i < tipo.NumField(); i++ {
		etiqueta := tipo.Field(i).Tag.Get("json")
		if strings.Contains(etiqueta, ",") {
			t.Fatalf("la clave %q del recibo HTTP no puede ser opcional", etiqueta)
		}
		etiquetas = append(etiquetas, etiqueta)
	}
	sort.Strings(etiquetas)
	if strings.Join(etiquetas, ",") != strings.Join(clavesReciboCliente, ",") {
		t.Fatalf("recibo HTTP %v; el cliente web solo admite %v", etiquetas, clavesReciboCliente)
	}
	bruto, err := json.Marshal(resultadoAJSON(dietasports.ResultadoBorradorComision{
		Comision: domain.ComisionBorrador{Referencia: "dco_" + strings.Repeat("B", 22)},
		Recibo:   dietasports.ReciboBorradorComision{Referencia: "rcd_" + strings.Repeat("0", 22), Version: 1, RegistradoEn: time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC)}}))
	if err != nil {
		t.Fatal(err)
	}
	var leido struct {
		Recibo map[string]json.RawMessage `json:"recibo"`
	}
	if err := json.Unmarshal(bruto, &leido); err != nil {
		t.Fatal(err)
	}
	if claves := clavesOrdenadas(leido.Recibo); strings.Join(claves, ",") != strings.Join(clavesReciboCliente, ",") {
		t.Fatalf("resultadoAJSON emite el recibo %v", claves)
	}
}
