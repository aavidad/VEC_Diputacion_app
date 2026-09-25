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

// El circuito de cada solicitud propia (cronos_v1 000010) se acepta sólo
// coherente con su estado; sin 000010 faltan los dos campos a la vez.
func TestCircuitoSolicitudPropiaCoherenteConElEstado(t *testing.T) {
	texto := func(v string) *string { return &v }
	si, no := true, false
	for _, c := range []struct {
		estado    string
		circuito  *string
		pendiente *bool
		ok        bool
		esperado  domain.CircuitoPermiso
		asignar   bool
	}{
		{"solicitado", nil, nil, true, "", false},
		{"solicitado", texto("J-A"), nil, false, "", false},
		{"solicitado", texto("J-A"), &no, true, domain.CircuitoResponsableAdministracion, false},
		{"solicitado", texto("J-A"), &si, true, domain.CircuitoResponsableAdministracion, true},
		{"solicitado", texto("A"), &no, true, domain.CircuitoAdministracion, false},
		{"solicitado", texto("A"), &si, false, "", false},
		{"solicitado", nil, &no, false, "", false},
		{"solicitado", texto("X"), &no, false, "", false},
		{"pendiente_administracion", texto("J-A"), &no, true, domain.CircuitoResponsableAdministracion, false},
		{"pendiente_administracion", texto("A"), &no, false, "", false},
		{"pendiente_administracion", texto("J-A"), &si, false, "", false},
		{"concedido", nil, &no, true, "", false},
		{"concedido", texto("A"), &no, true, domain.CircuitoAdministracion, false},
		{"denegado", texto("J-A"), &si, false, "", false},
	} {
		circuito, asignar, ok := circuitoSolicitudPropia(solicitudSQL{Estado: c.estado, Circuito: c.circuito, PendienteAsignacion: c.pendiente})
		if ok != c.ok || circuito != c.esperado || asignar != c.asignar {
			t.Fatalf("%+v: %q %v %v", c, circuito, asignar, ok)
		}
	}
}
