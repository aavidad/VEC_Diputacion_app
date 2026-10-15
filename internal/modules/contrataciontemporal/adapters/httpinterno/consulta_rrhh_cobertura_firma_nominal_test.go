package httpinterno

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestDetalleRRHHHTTPExponeCoberturaNominalAncladaYMinima(t *testing.T) {
	base := detalleRRHHPrueba()
	base.CoberturaFirmaNominal = ports.CoberturaFirmaNominalRRHH{
		Estado: ports.CoberturaFirmaNominalDisponible, UnidadRef: "unidad:rrhh:prueba",
		Plan:       domain.VersionPlanFirmaV2{Referencia: "plan:nominal:prueba", Version: 2, HuellaSHA256: strings.Repeat("a", 64)},
		Documentos: []string{"contrato", "resolucion"},
	}
	lector := &consultorDetalleRRHHPrueba{detalle: base}
	h, err := NuevoManejadorConsultaDetalleRRHH(lector)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, nuevaPeticionConsultaRRHHPrueba(RutaConsultaDetalleRRHH, cuerpoDetalleRRHHPrueba()))
	if w.Code != http.StatusOK || lector.llamadas != 1 {
		t.Fatalf("HTTP=%d, lecturas=%d, cuerpo=%s", w.Code, lector.llamadas, w.Body.String())
	}
	var salida struct {
		Data struct {
			CapacidadesFicha capacidadesFichaRRHHJSON `json:"capacidades_ficha"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &salida); err != nil {
		t.Fatal(err)
	}
	firma := salida.Data.CapacidadesFicha.FirmaNominal
	if firma == nil || firma.Estado != "disponible" || firma.ExpedienteRef != base.Resumen.ExpedienteRef ||
		firma.VersionObservada != base.Resumen.Version || firma.UnidadRef != base.CoberturaFirmaNominal.UnidadRef ||
		firma.PlanRef != base.CoberturaFirmaNominal.Plan.Referencia || firma.PlanVersion != 2 ||
		firma.PlanSHA256 != base.CoberturaFirmaNominal.Plan.HuellaSHA256 ||
		len(firma.Documentos) != 2 || strings.Contains(w.Body.String(), "cargo:prueba") {
		t.Fatalf("cobertura HTTP incorrecta: %+v", firma)
	}
}

func TestDetalleRRHHHTTPNoDisponibleSinPlanNiDocumentos(t *testing.T) {
	base := detalleRRHHPrueba()
	base.CoberturaFirmaNominal = ports.CoberturaFirmaNominalRRHH{
		Estado: ports.CoberturaFirmaNominalNoDisponible, Documentos: []string{},
	}
	salida := proyectarDetalleRRHH(base)
	if salida.CapacidadesFicha == nil || salida.CapacidadesFicha.FirmaNominal == nil ||
		salida.CapacidadesFicha.FirmaNominal.Estado != "no_disponible" ||
		salida.CapacidadesFicha.FirmaNominal.ExpedienteRef != base.Resumen.ExpedienteRef ||
		salida.CapacidadesFicha.FirmaNominal.VersionObservada != base.Resumen.Version ||
		len(salida.CapacidadesFicha.FirmaNominal.Documentos) != 0 {
		t.Fatalf("estado sin plan incorrecto: %+v", salida.CapacidadesFicha)
	}
	b, err := json.Marshal(salida.CapacidadesFicha.FirmaNominal)
	if err != nil || strings.Contains(string(b), "plan_ref") || strings.Contains(string(b), "plan_sha256") ||
		!strings.Contains(string(b), `"documentos":[]`) {
		t.Fatalf("se publicaron datos del plan sin cobertura: %s, error=%v", b, err)
	}
}
