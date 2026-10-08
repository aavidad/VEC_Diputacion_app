package postgres

import (
	"encoding/json"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestCapturasCuadroRRHHReconstituyeDiccionariosYRechazaAusencias(t *testing.T) {
	t.Parallel()
	desde := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	fila := capturaSQLRRHH{Estado: "legado_base_transicion", Fase: "fiscalizacion", FaseDesde: desde,
		BaseID: "vec.contratacion_temporal.reglas", BaseVersion: 1, BaseHuella: "base-huella",
		AjustesID: "vec.contratacion_temporal.reglas.ajustes", AjustesHuella: "ajustes-huella",
		CapturadaEn: &desde, Numero: 2, Urgente: true}
	datos := capturasSQLRRHH{Filas: []capturaSQLRRHH{fila}, Grupos: []capturaSQLRRHH{fila},
		Bases:   map[string]string{"base-huella": "{\"id\":\"base\"}"},
		Ajustes: map[string]string{"ajustes-huella": "{}"}}
	paginaJSON, _ := json.Marshal(capturasSQLRRHH{Filas: datos.Filas, Bases: datos.Bases, Ajustes: datos.Ajustes})
	gruposJSON, _ := json.Marshal(capturasSQLRRHH{Grupos: datos.Grupos, Bases: datos.Bases, Ajustes: datos.Ajustes})
	salida := salidaCuadroConsultaRRHH{capturasPlazo: paginaJSON, capturasGrupos: gruposJSON}
	capturas, err := salida.capturasPagina([]ports.ResumenExpedienteRRHH{{FaseClave: domain.ClaveFase("fiscalizacion")}}, []time.Time{desde})
	if err != nil || len(capturas) != 1 || string(capturas[0].BaseCanonico) != "{\"id\":\"base\"}" || string(capturas[0].AjustesCanonico) != "{}" {
		t.Fatalf("captura: %+v %v", capturas, err)
	}
	agregado := &ports.AgregadosCuadroRRHH{GruposPlazo: []ports.GrupoPlazoCuadroRRHH{{FaseClave: domain.ClaveFase("fiscalizacion"), Desde: desde, Numero: 2, Urgente: true}}}
	if err := salida.capturasResumen(agregado); err != nil || agregado.GruposPlazo[0].Captura == nil || agregado.GruposPlazo[0].Captura.Estado != "legado_base_transicion" {
		t.Fatalf("grupo: %+v %v", agregado, err)
	}
	delete(datos.Bases, "base-huella")
	salida.capturasPlazo, _ = json.Marshal(capturasSQLRRHH{Filas: datos.Filas, Bases: datos.Bases, Ajustes: datos.Ajustes})
	if _, err := salida.capturasPagina([]ports.ResumenExpedienteRRHH{{FaseClave: domain.ClaveFase("fiscalizacion")}}, []time.Time{desde}); err == nil {
		t.Fatal("base ausente aceptada")
	}
}

func TestCapturasCuadroRRHHComparteCanonicosPorHuellaEnUnPayload(t *testing.T) {
	desde := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	fila := capturaSQLRRHH{Estado: "capturada", Fase: "fiscalizacion", FaseDesde: desde,
		BaseID: "vec.contratacion_temporal.reglas", BaseVersion: 1, BaseHuella: "base-uno",
		AjustesID: "vec.contratacion_temporal.reglas.ajustes", AjustesHuella: "ajustes-uno", CapturadaEn: &desde}
	otra := fila
	otra.BaseHuella, otra.AjustesHuella = "base-dos", "ajustes-dos"
	entrada := capturasSQLRRHH{Grupos: []capturaSQLRRHH{fila, fila, otra},
		Bases:   map[string]string{"base-uno": "{\"id\":\"uno\"}", "base-dos": "{\"id\":\"dos\"}"},
		Ajustes: map[string]string{"ajustes-uno": "{}", "ajustes-dos": "{\"c\":{}}"}}
	raw, err := json.Marshal(entrada)
	if err != nil {
		t.Fatal(err)
	}
	leida, err := leerCapturasSQLRRHH(raw)
	if err != nil {
		t.Fatal(err)
	}
	var capturas [3]ports.CapturaPlazoFaseRRHH
	for i, grupo := range leida.Grupos {
		capturas[i], err = leida.convertir(grupo)
		if err != nil {
			t.Fatalf("grupo %d: %v", i, err)
		}
	}
	if &capturas[0].BaseCanonico[0] != &capturas[1].BaseCanonico[0] ||
		&capturas[0].AjustesCanonico[0] != &capturas[1].AjustesCanonico[0] {
		t.Fatal("el mismo par se copió por grupo")
	}
	if &capturas[0].BaseCanonico[0] == &capturas[2].BaseCanonico[0] ||
		&capturas[0].AjustesCanonico[0] == &capturas[2].AjustesCanonico[0] ||
		string(capturas[2].BaseCanonico) != entrada.Bases[otra.BaseHuella] ||
		string(capturas[2].AjustesCanonico) != entrada.Ajustes[otra.AjustesHuella] {
		t.Fatal("pares distintos mezclados")
	}
	delete(entrada.Ajustes, "ajustes-uno")
	raw, err = json.Marshal(entrada)
	if err != nil {
		t.Fatal(err)
	}
	leida, err = leerCapturasSQLRRHH(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := leida.convertir(fila); err == nil {
		t.Fatal("ajustes ausentes aceptados")
	}
}
