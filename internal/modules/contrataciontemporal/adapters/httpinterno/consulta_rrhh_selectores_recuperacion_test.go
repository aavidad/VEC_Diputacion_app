package httpinterno

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestDetalleRRHHExponeSoloSelectorHistoricoCerrado(t *testing.T) {
	base := detalleRRHHPrueba()
	prueba := ports.ReciboSelectoresRecuperacionRRHH{AccesoRef: "lectura:rrhh:001",
		SelloDetalleSHA256: strings.Repeat("a", 64), SelloSelectoresSHA256: strings.Repeat("b", 64)}
	reciboEsperado := map[string]any{
		"acceso_ref": prueba.AccesoRef, "sello_detalle_sha256": prueba.SelloDetalleSHA256,
		"sello_selectores_sha256": prueba.SelloSelectoresSHA256,
	}
	comprobarRecibo := func(salida map[string]any, cuerpo []byte) {
		t.Helper()
		if !reflect.DeepEqual(salida["recibo_selectores_recuperacion_firmas"], reciboEsperado) {
			t.Fatalf("recibo de selectores ausente, incompleto o cruzado: %s", cuerpo)
		}
	}
	legacy, err := json.Marshal(proyectarDetalleRRHH(base))
	if err != nil {
		t.Fatalf("la ficha legacy cambió: %s, %v", legacy, err)
	}
	var legacyJSON map[string]any
	if err := json.Unmarshal(legacy, &legacyJSON); err != nil {
		t.Fatal(err)
	}
	if _, presente := legacyJSON["selectores_recuperacion_firmas"]; presente {
		t.Fatalf("la ficha legacy expone selectores: %s", legacy)
	}
	if _, presente := legacyJSON["recibo_selectores_recuperacion_firmas"]; presente {
		t.Fatalf("la ficha legacy expone recibo de selectores: %s", legacy)
	}
	vacio, err := base.ConSelectoresRecuperacionFirmas(nil, prueba)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(proyectarDetalleRRHH(vacio))
	if err != nil {
		t.Fatalf("falta la lista vacía del contrato extendido: %s, %v", b, err)
	}
	var vacioJSON map[string]any
	if err := json.Unmarshal(b, &vacioJSON); err != nil {
		t.Fatal(err)
	}
	listaVacia, esArray := vacioJSON["selectores_recuperacion_firmas"].([]any)
	if !esArray || len(listaVacia) != 0 {
		t.Fatalf("la lista vacía debe ser un array JSON: %s", b)
	}
	comprobarRecibo(vacioJSON, b)
	fila := ports.FirmaSelectorRecuperacionRRHH{FirmaRef: "firma:historica:001", Selector: ports.SelectorRecuperacionFirmaRRHH{
		ExpedienteRef: base.Resumen.ExpedienteRef, VersionExpediente: base.Resumen.Version,
		Documento: "resolucion", PasoOrden: 3, ClaveIdempotencia: "clave-firma-historica-000001",
		CatalogoHuella: strings.Repeat("a", 64), Via: ports.ViaFirmaCertificadoVEC,
	}}
	ampliado, err := base.ConSelectoresRecuperacionFirmas([]ports.FirmaSelectorRecuperacionRRHH{fila}, prueba)
	if err != nil {
		t.Fatal(err)
	}
	b, err = json.Marshal(proyectarDetalleRRHH(ampliado))
	if err != nil {
		t.Fatal(err)
	}
	var salida map[string]any
	if err := json.Unmarshal(b, &salida); err != nil {
		t.Fatal(err)
	}
	filas, ok := salida["selectores_recuperacion_firmas"].([]any)
	if !ok || len(filas) != 1 {
		t.Fatalf("selector ausente: %s", b)
	}
	comprobarRecibo(salida, b)
	selectorEsperado := map[string]any{
		"expediente_ref":     fila.Selector.ExpedienteRef,
		"version_expediente": float64(fila.Selector.VersionExpediente),
		"documento":          fila.Selector.Documento, "paso_orden": float64(fila.Selector.PasoOrden),
		"clave_idempotencia": fila.Selector.ClaveIdempotencia,
		"catalogo_huella":    fila.Selector.CatalogoHuella, "via": ports.ViaFirmaCertificadoVEC,
	}
	if !reflect.DeepEqual(filas[0], map[string]any{
		"firma_ref": fila.FirmaRef, "selector": selectorEsperado,
	}) {
		t.Fatalf("selector público divergente: %s", b)
	}
	filasPrivadas, _ := ampliado.SelectoresRecuperacionFirmas()
	filasPrivadas[0].Selector.Documento = "alterado"
	comprobacion, _ := ampliado.SelectoresRecuperacionFirmas()
	if comprobacion[0].Selector.Documento != "resolucion" {
		t.Fatal("el selector comparte memoria con el lector")
	}
}
