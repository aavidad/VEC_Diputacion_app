package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func TestErrorSolicitudTraduceSoloRechazosNominales(t *testing.T) {
	ctx := context.Background()
	for codigo, esperado := range map[string]error{
		"PC001": ports.ErrSolicitudCronosInvalida,
		"PC002": ports.ErrCorreccionEnConflicto,
		"PC003": ports.ErrDependenciaNoDisponible,
		"PC007": ports.ErrPermisoFueraDeLimites,
		"PC008": ports.ErrCalendarioNoPublicado,
		"PC009": ports.ErrPermisoNoSolicitable,
		"PC010": ports.ErrPermisoSolapado,
		"42501": ports.ErrDependenciaNoDisponible,
		"23505": ports.ErrDependenciaNoDisponible,
	} {
		if err := errorSolicitud(ctx, &pgconn.PgError{Code: codigo}, ports.ErrCorreccionEnConflicto); !errors.Is(err, esperado) {
			t.Fatalf("%s: %v", codigo, err)
		}
	}
	if err := errorSolicitud(ctx, errors.New("red"), ports.ErrClaveOperacionEnConflicto); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("un fallo ajeno no es dependencia", err)
	}
	cancelado, cancelar := context.WithCancel(ctx)
	cancelar()
	if err := errorSolicitud(cancelado, &pgconn.PgError{Code: "PC007"}, nil); !errors.Is(err, context.Canceled) {
		t.Fatal("la cancelación no se conserva", err)
	}
}

func TestRepositoriosSolicitudesFallanCerradosSinOrden(t *testing.T) {
	ctx := context.Background()
	if _, err := NuevoRepositorioCorreccionesPropias(nil, domain.ZonaSaldoPeninsula); err == nil {
		t.Fatal("repositorio sin pool")
	}
	mov := &RepositorioConsultaMovimientos{db: iniciadorNulo{}}
	if _, err := mov.ConsultarMovimientos(ctx, ports.OrdenConsultaMovimientos{}, "emp_x", "2026-01-01", "2026-01-01", domain.ZonaSaldoPeninsula); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal(err)
	}
	per := &RepositorioPermisosPropios{db: iniciadorNulo{}}
	if _, err := per.ConsultarPermisosPropios(ctx, ports.OrdenPermisosPropios{}, "emp_x", 2026, domain.ZonaSaldoPeninsula); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal(err)
	}
	if _, err := per.SolicitarPermisoPropio(ctx, ports.OrdenPermisosPropios{}, domain.MaterialSolicitudPermisoPropio{}); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal(err)
	}
	cor := &RepositorioCorreccionesPropias{db: iniciadorNulo{}, zona: domain.ZonaSaldoPeninsula}
	if _, err := cor.SolicitarOlvido(ctx, domain.SolicitudCorreccion{}, ports.OrdenConsumoCorreccion{}); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal(err)
	}
	if _, err := cor.RegistrarActuacion(ctx, domain.ActuacionCorreccion{}, ports.OrdenConsumoCorreccion{}); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("las decisiones de jefatura y RRHH no pertenecen a este corte", err)
	}
}

// iniciadorNulo falla la prueba si un repositorio sin orden llegara a abrir
// una transacción.
type iniciadorNulo struct{}

func (iniciadorNulo) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	panic("transacción abierta sin orden autorizada")
}
