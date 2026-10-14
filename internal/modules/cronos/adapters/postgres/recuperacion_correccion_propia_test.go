package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type proveedorSoloSolicitudPrueba struct{}

func (proveedorSoloSolicitudPrueba) ProveerMaterialCorreccion(context.Context, domain.MaterialAutorizacionCorreccion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	panic("no se debe pedir permiso de solicitud al recuperar")
}

type proveedorRecuperacionPrueba struct{ proveedorSoloSolicitudPrueba }

func (proveedorRecuperacionPrueba) ProveerMaterialRecuperacionCorreccionPropia(context.Context, domain.MaterialRecuperacionCorreccionPropia) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	panic("no se debe emitir sin vínculo Personal")
}

func TestRecuperacionCorreccionNoAbreTransaccionSinAutoridadNiVinculo(t *testing.T) {
	actor := actorPrueba(t, "emp_0123456789abcdefghijkl")
	clave := ports.ClaveRecuperacionCorreccion{
		SolicitudRef: "correccion:cronos:olvido_0001", ClaveOperacion: "olvido_0001", Paso: domain.PasoSolicitudCorreccion,
	}
	repositorio := &RepositorioCorreccionesPropias{db: iniciadorNulo{}, zona: domain.ZonaSaldoPeninsula}
	for _, proveedor := range []ports.ProveedorMaterialCorreccion{proveedorSoloSolicitudPrueba{}, proveedorRecuperacionPrueba{}} {
		orden, err := ports.NuevaOrdenConsumoCorreccion(actor, proveedor)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := repositorio.RecuperarRecibo(context.Background(), clave, orden); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
			t.Fatal("recuperación sin autoridad o vínculo abrió transacción", err)
		}
	}
}

func TestErrorRecuperacionCorreccionSoloRevelaAusenciaNominal(t *testing.T) {
	for _, caso := range []struct {
		codigo   string
		esperado error
	}{
		{"PC004", ports.ErrCorreccionNoAutorizada},
		{"42501", ports.ErrDependenciaNoDisponible},
		{"23505", ports.ErrDependenciaNoDisponible},
	} {
		if err := errorRecuperacionCorreccion(context.Background(), &pgconn.PgError{Code: caso.codigo}); !errors.Is(err, caso.esperado) {
			t.Fatalf("%s: %v", caso.codigo, err)
		}
	}
}
