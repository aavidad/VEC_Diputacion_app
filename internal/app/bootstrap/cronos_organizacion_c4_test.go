package bootstrap

import (
	"context"
	"errors"
	"testing"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

func TestOrganizacionC4ExigeProyeccionP52AntesDeAutorizar(t *testing.T) {
	l := &lectorOrganizacionC4{}
	_, err := l.ConsultarC4(context.Background(), personaldomain.SolicitudConsultaOrganizacionHistorica{}, personalports.ResultadoUnidadTrabajoPropiaCronos{})
	if !errors.Is(err, personaldomain.ErrOrganizacionHistoricaNoDisponible) {
		t.Fatalf("unidad P52 ausente: %v", err)
	}
	var proveedor proveedorOrganizacionC4
	_, err = proveedor.AutorizarConsultaOrganizacionHistorica(context.Background(), personaldomain.MaterialConsultaOrganizacionHistorica{})
	if !errors.Is(err, personaldomain.ErrOrganizacionHistoricaNoDisponible) {
		t.Fatalf("contexto P52 ausente: %v", err)
	}
}
