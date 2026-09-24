package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestDocumentoComisionSeparaGastoDeclaradoDeTresMaximosProvisionales(t *testing.T) {
	version := "provisional:rd462:20260923"
	regla := reglaTramosPrueba()
	rutas := []RutaDeclaradaComision{{CodigosRuta: []string{"ruta:a", "ruta:b"}, AjusteKilometros: "0.0000"}}
	calculo := CalculoComision{Procedencia: "osrm_interno", VersionGrafo: "grafo:prueba", Motor: "OSRM", VersionTarifa: version, Rotulo: RotuloTarifaProvisional, ReglaRef: regla.ReglaRef, ReglaHuellaSHA256: regla.HuellaSHA256, HoraInicio: "08:00", HoraFin: "18:00", Kilometros: "1.0000", EURPorKM: "0.2600", ImporteKilometrajeCentimos: 26, TramosRuta: []TramoRutaComision{}, VehiculoPropio: true, Rutas: []RutaCalculadaComision{{CodigosRuta: []string{"ruta:a", "ruta:b"}, VersionGrafo: "grafo:prueba", TramosRuta: []TramoRutaComision{{OrigenCodigo: "ruta:a", DestinoCodigo: "ruta:b", Kilometros: "1.0000"}}, KilometrosBase: "1.0000", AjusteKilometros: "0.0000", KilometrosFinales: "1.0000", ImporteCentimos: 26}}}
	zona, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	inicio, err := ResolverInstanteCivil("2026-09-21", "08:00", zona)
	if err != nil {
		t.Fatal(err)
	}
	fin, err := ResolverInstanteCivil("2026-09-22", "18:00", zona)
	if err != nil {
		t.Fatal(err)
	}
	for grupo := 1; grupo <= 3; grupo++ {
		tramos, err := CalcularTramosNacionalesProvisionales(inicio, fin, zona, TarifaNacionalProvisional{VersionRef: version, Rotulo: RotuloTarifaProvisional, PaisISO2: "ES", Grupo: grupo, VigenteDesde: "2026-09-01", ManutencionCentimos: int64(1000 * grupo), AlojamientoTopeCentimos: int64(2000 * grupo)}, reglaTramosPrueba())
		if err != nil {
			t.Fatal(err)
		}
		calculo.OpcionesDieta = append(calculo.OpcionesDieta, OpcionDietaComision{Grupo: grupo, Calculo: tramos})
	}
	otros := []OtroGastoDeclarado{{Tipo: "otro_gasto", Concepto: "Peaje justificado", ImporteCentimos: 100, JustificanteRef: "doc_0123456789abcdefghijklmn", JustificanteSHA256: strings.Repeat("a", 64)}}
	aceptados := make([]int, len(calculo.OpcionesDieta[1].Calculo.Tramos))
	for i := range aceptados {
		aceptados[i] = i
	}
	doc, err := ConstruirDocumentoComision(calculo, []string{"ruta:a", "ruta:b"}, rutas, true, 2, aceptados, version, otros)
	if err != nil {
		t.Fatal(err)
	}
	if doc.KilometrajeCentimos != 26 || doc.OtrosCentimos != 100 || doc.GrupoDieta != 2 || doc.TotalOrientativoCentimos <= 126 || doc.Validar(calculo, []string{"ruta:a", "ruta:b"}) != nil {
		t.Fatalf("documento incoherente: %+v", doc)
	}
	for _, l := range doc.Lineas {
		if l.Tipo == "dieta" && (l.Grupo != 2 || l.IndiceTramo == nil) {
			t.Fatal("grupo acreditado o índice perdido")
		}
	}
	alterado := doc
	alterado.Lineas = append([]LineaDocumentoComision(nil), doc.Lineas...)
	alterado.Lineas[0].ImporteCentimos++
	if alterado.Validar(calculo, []string{"ruta:a", "ruta:b"}) == nil {
		t.Fatal("importe de dieta manipulado aceptado")
	}
	otros[0].JustificanteSHA256 = "sin-huella"
	if _, err := ConstruirDocumentoComision(calculo, []string{"ruta:a", "ruta:b"}, rutas, true, 2, aceptados, version, otros); err == nil {
		t.Fatal("justificante sin huella aceptado")
	}
	otros[0].JustificanteRef = ""
	otros[0].JustificanteSHA256 = ""
	if _, err := ConstruirDocumentoComision(calculo, []string{"ruta:a", "ruta:b"}, rutas, true, 2, aceptados, version, otros); err != nil {
		t.Fatalf("otro gasto sin justificante rechazado: %v", err)
	}
	otros[0].Tipo = "otro_medio"
	if _, err := ConstruirDocumentoComision(calculo, []string{"ruta:a", "ruta:b"}, rutas, true, 2, aceptados, version, otros); err != nil {
		t.Fatalf("otro medio descrito sin fichero rechazado: %v", err)
	}
	otros[0].JustificanteRef = "doc_0123456789abcdefghijklmn"
	if _, err := ConstruirDocumentoComision(calculo, []string{"ruta:a", "ruta:b"}, rutas, true, 2, aceptados, version, otros); err == nil {
		t.Fatal("referencia sin huella aceptada")
	}
	if len(aceptados) > 2 {
		if _, err := ConstruirDocumentoComision(calculo, []string{"ruta:a", "ruta:b"}, rutas, true, 2, []int{0, 1}, version, nil); err == nil {
			t.Fatal("subconjunto distinto de uno o todos aceptado")
		}
	}
	vehiculo := true
	comision := ComisionBorrador{Referencia: "dco_0123456789abcdefghijklmn", NumeroDocumento: "VEC-D-2026-000001", FechaApertura: "2026-09-21T08:00:00.000000Z", Version: 2, Estado: "borrador", FechaInicio: "2026-09-21", FechaFin: "2026-09-22", Motivo: "Visita técnica", CodigosRuta: []string{"ruta:a", "ruta:b"}, RelacionRef: "rel_0123456789abcdefghijklmn", CentroRef: "centro:prueba", UnidadRef: "unidad:prueba", Calculo: &calculo, Documento: &doc, VehiculoPropio: &vehiculo, Rutas: &rutas}
	if err := comision.Validar(); err != nil {
		t.Fatalf("documento v2 inválido: %v", err)
	}
	b, err := json.Marshal(comision)
	if err != nil {
		t.Fatal(err)
	}
	var recuperada ComisionBorrador
	if err = json.Unmarshal(b, &recuperada); err != nil || recuperada.Validar() != nil || recuperada.VehiculoPropio == nil || !*recuperada.VehiculoPropio || recuperada.Rutas == nil || len(*recuperada.Rutas) != 1 {
		t.Fatalf("roundtrip D4: %v %+v", err, recuperada)
	}
	recuperada.NumeroDocumento = "D-2026-000001"
	if recuperada.Validar() == nil {
		t.Fatal("número externo/no canónico aceptado")
	}
}

func TestDocumentoComisionValidaVariasRutasYAjusteFirmado(t *testing.T) {
	version := "provisional:rd462:20260923"
	regla := reglaTramosPrueba()
	c := CalculoComision{Procedencia: "osrm_interno", VersionGrafo: "grafo:prueba", Motor: "OSRM", VersionTarifa: version, Rotulo: RotuloTarifaProvisional, ReglaRef: regla.ReglaRef, ReglaHuellaSHA256: regla.HuellaSHA256, HoraInicio: "08:00", HoraFin: "18:00", Kilometros: "2.6000", EURPorKM: "0.2600", ImporteKilometrajeCentimos: 68, TramosRuta: []TramoRutaComision{}, VehiculoPropio: true}
	for grupo := 1; grupo <= 3; grupo++ {
		c.OpcionesDieta = append(c.OpcionesDieta, OpcionDietaComision{Grupo: grupo, Calculo: CalculoDietasProvisional{Tramos: []TramoDietaProvisional{}, VersionTarifaRef: version, Rotulo: RotuloTarifaProvisional, ReglaRef: regla.ReglaRef, ReglaHuellaSHA256: regla.HuellaSHA256}})
	}
	rutas := []RutaDeclaradaComision{{CodigosRuta: []string{"ruta:a", "ruta:c"}, AjusteKilometros: "0.1000", MotivoAjuste: "Desvío de obra"}, {CodigosRuta: []string{"ruta:c", "ruta:b"}, AjusteKilometros: "-0.5000", MotivoAjuste: "Atajo documentado"}}
	c.Rutas = []RutaCalculadaComision{{CodigosRuta: []string{"ruta:a", "ruta:c"}, VersionGrafo: "grafo:prueba", TramosRuta: []TramoRutaComision{{OrigenCodigo: "ruta:a", DestinoCodigo: "ruta:c", Kilometros: "1.0000"}}, KilometrosBase: "1.0000", AjusteKilometros: "0.1000", MotivoAjuste: "Desvío de obra", KilometrosFinales: "1.1000", ImporteCentimos: 29}, {CodigosRuta: []string{"ruta:c", "ruta:b"}, VersionGrafo: "grafo:prueba", TramosRuta: []TramoRutaComision{{OrigenCodigo: "ruta:c", DestinoCodigo: "ruta:b", Kilometros: "2.0000"}}, KilometrosBase: "2.0000", AjusteKilometros: "-0.5000", MotivoAjuste: "Atajo documentado", KilometrosFinales: "1.5000", ImporteCentimos: 39}}
	if err := c.ValidarDocumento([]string{"ruta:a", "ruta:b"}, rutas, true); err != nil {
		t.Fatalf("cálculo: %v", err)
	}
	d, err := ConstruirDocumentoComision(c, []string{"ruta:a", "ruta:b"}, rutas, true, 1, nil, version, nil)
	if err != nil || d.Validar(c, []string{"ruta:a", "ruta:b"}) != nil || d.TotalOrientativoCentimos != 68 {
		t.Fatalf("rutas válidas rechazadas: %v %+v", err, d)
	}
	km := 0
	for _, l := range d.Lineas {
		if l.Tipo == "kilometraje" {
			km++
			if l.RutaIndice != km {
				t.Fatal("orden de rutas perdido")
			}
		}
	}
	if km != 2 {
		t.Fatalf("rutas=%d", km)
	}
	rutas[1].MotivoAjuste = ""
	if c.ValidarDocumento([]string{"ruta:a", "ruta:b"}, rutas, true) == nil {
		t.Fatal("ajuste sin motivo aceptado")
	}
	c.Rutas = nil
	c.VehiculoPropio = false
	c.Procedencia = "sin_vehiculo_propio"
	c.Motor = "no_aplica"
	c.VersionGrafo = "no_aplica"
	c.Kilometros = "0.0000"
	c.ImporteKilometrajeCentimos = 0
	d, err = ConstruirDocumentoComision(c, []string{"ruta:a", "ruta:b"}, nil, false, 1, nil, version, nil)
	if err != nil || d.TotalOrientativoCentimos != 0 || d.VehiculoPropio {
		t.Fatalf("sin vehículo rechazado: %v %+v", err, d)
	}
}
