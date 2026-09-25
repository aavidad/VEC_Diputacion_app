package domain

import (
	"strings"
	"testing"
	"time"
)

func TestSolicitudPermisoLigaVersionCantidadYFechas(t *testing.T) {
	c := catalogoPrueba()
	s := SolicitudPermiso{Referencia: "sol:valid", EmpleadoRef: "emp:valid", CatalogoVersionRef: c.VersionRef, PermisoRef: c.PermisoRef, Desde: "2026-10-01", Hasta: "2026-10-02", Cantidad: 2, Unidad: c.Unidad, Estado: EstadoPermisoSolicitado, Version: 1, ClaveOperacion: "operacion123", HuellaMaterial: strings.Repeat("a", 64), ReciboRef: "recibo:valid", CreadaUTC: time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)}
	if err := s.Validar(c); err != nil {
		t.Fatalf("solicitud valida: %v", err)
	}
	for _, cambiar := range []func(*SolicitudPermiso){
		func(x *SolicitudPermiso) { x.CatalogoVersionRef = "catalogo:v2" },
		func(x *SolicitudPermiso) { x.Cantidad = 0 },
		func(x *SolicitudPermiso) { x.Desde = "2026-10-03" },
		func(x *SolicitudPermiso) { x.Hasta = "2026-02-31" },
		func(x *SolicitudPermiso) { x.Estado = "aprobado_por_cliente" },
	} {
		invalida := s
		cambiar(&invalida)
		if err := invalida.Validar(c); err == nil {
			t.Fatalf("solicitud invalida aceptada: %+v", invalida)
		}
	}
}
