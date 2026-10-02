package application

import (
	"context"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestAnalisisCircuitoRRHHSinFuenteDeFirmasNoConfirma(t *testing.T) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(t, ports.OperacionRegistrarAnalisis, "cirrrhh")
	definicion, err := domain.NuevaDefinicionCircuitoRRHH(
		"flujo:ct:rrhh:sintetico", 2, escenario.expediente.FaseActual,
		[]domain.TransicionCircuitoRRHH{
			{Clave: "contratacion_temporal.circuito.peticion_firmada",
				Tipo: domain.HitoPeticionFirmada, Origen: escenario.expediente.FaseActual,
				Destino: "autorizacion_rrhh", RequiereDocumento: true,
				RequiereFirma: true, PerfilClave: "tecnico_rrhh",
				FirmasRequeridas: []domain.ClaveCatalogo{"tecnico_solicitante", "delegacion_solicitante"}},
			{Clave: "contratacion_temporal.circuito.autorizacion_rrhh",
				Tipo: domain.HitoAutorizacionRRHH, Origen: "autorizacion_rrhh",
				Destino: "credito", RequiereDocumento: true,
				PerfilClave: "tecnico_rrhh", AutorizanteCargoClave: "direccion_rrhh"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	circuito, err := domain.NuevoCircuitoAdministrativo(definicion)
	if err != nil {
		t.Fatal(err)
	}
	escenario.expediente.Flujo = definicion.Flujo
	escenario.expediente.Circuito = &circuito
	if escenario.expediente.Validar() != nil {
		t.Fatal("expediente inicial del circuito inválido")
	}
	servicio, dobles := construirServicioOperacionAnalisisSaneado(t, escenario)
	if _, err := servicio.Registrar(context.Background(), escenario.registrar); err == nil {
		t.Fatal("se confirmó análisis sin fuente de firmas y autorización")
	}
	if dobles.transaccion.llamadas != 0 {
		t.Fatal("el análisis nuevo llegó a SQL sin evidencias")
	}
}
