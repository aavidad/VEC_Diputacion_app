package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type filaDenegacionPrueba struct {
	valor bool
	err   error
}

func (f filaDenegacionPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	*destinos[0].(*bool) = f.valor
	return nil
}

type consultorDenegacionPrueba struct {
	preflight bool
	guardado  bool
	err       error
	consultas []struct {
		ctx            context.Context
		estadoAlLlamar error
		sql            string
		args           []any
	}
}

func (c *consultorDenegacionPrueba) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	c.consultas = append(c.consultas, struct {
		ctx            context.Context
		estadoAlLlamar error
		sql            string
		args           []any
	}{ctx, ctx.Err(), sql, args})
	if sql == preflightDenegacionSQL {
		return filaDenegacionPrueba{valor: c.preflight, err: c.err}
	}
	return filaDenegacionPrueba{valor: c.guardado, err: c.err}
}

func ordenDenegacionPrueba(motivo vecports.MotivoAuditoriaFronteraRutaExacta) vecports.OrdenAuditoriaFronteraRutaExacta {
	o := vecports.OrdenAuditoriaFronteraRutaExacta{
		CorrelacionRef: "corr_0123456789abcdef0123456789abcdef",
		Motivo:         motivo, Superficie: superficieAuditoriaPreferencias, Ruta: rutaAuditoriaPreferencias,
	}
	if motivo == vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado {
		o.ActorRef = "per_0123456789abcdefghijkl"
	}
	return o
}

func TestDenegacionPreflightExigeRolYACLEfectiva(t *testing.T) {
	c := &consultorDenegacionPrueba{preflight: true}
	r, err := nuevoRegistradorDenegacionPreferenciasPostgreSQL(context.Background(), c)
	if err != nil || r == nil || len(c.consultas) != 1 || c.consultas[0].sql != preflightDenegacionSQL {
		t.Fatalf("preflight: %v", err)
	}
	for _, fragmento := range []string{"session_user=current_user", "vec_usuarios_registrador_frontera", "m.inherit_option", "NOT m.set_option", "NOT m.admin_option", "has_function_privilege", "has_table_privilege", "has_sequence_privilege", "NOT pg_catalog.pg_is_in_recovery()"} {
		if !strings.Contains(c.consultas[0].sql, fragmento) {
			t.Fatalf("falta guarda efectiva: %s", fragmento)
		}
	}
	c = &consultorDenegacionPrueba{}
	if _, err := nuevoRegistradorDenegacionPreferenciasPostgreSQL(context.Background(), c); !errors.Is(err, ErrAuditoriaFronteraPreferenciasNoDisponible) {
		t.Fatal("rol no acreditado aceptado")
	}
}

func TestRegistra401ConActorSQLNullY403SoloConPersona(t *testing.T) {
	c := &consultorDenegacionPrueba{preflight: true, guardado: true}
	r, _ := nuevoRegistradorDenegacionPreferenciasPostgreSQL(context.Background(), c)
	para401 := ordenDenegacionPrueba(vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida)
	if err := r.RegistrarAuditoriaFronteraRutaExacta(context.Background(), para401); err != nil {
		t.Fatal(err)
	}
	if len(c.consultas) != 2 || c.consultas[1].sql != registrarDenegacionSQL || len(c.consultas[1].args) != 5 ||
		c.consultas[1].args[0] != para401.CorrelacionRef || c.consultas[1].args[1] != string(para401.Motivo) ||
		c.consultas[1].args[2] != superficieAuditoriaPreferencias || c.consultas[1].args[3] != rutaAuditoriaPreferencias ||
		c.consultas[1].args[4] != "" || !strings.Contains(registrarDenegacionSQL, "NULLIF($5::text,'')") {
		t.Fatal("401 no transmite actor SQL NULL")
	}
	para403 := ordenDenegacionPrueba(vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado)
	if err := r.RegistrarAuditoriaFronteraRutaExacta(context.Background(), para403); err != nil || c.consultas[2].args[4] != para403.ActorRef {
		t.Fatal("403 no usa persona verificada")
	}
	antes := len(c.consultas)
	para403.ActorRef = "cuenta_de_cabecera"
	if err := r.RegistrarAuditoriaFronteraRutaExacta(context.Background(), para403); !errors.Is(err, vecports.ErrOrdenAuditoriaFronteraRutaExactaInvalida) || len(c.consultas) != antes {
		t.Fatal("actor ajeno enviado a SQL")
	}
	para401.ActorRef = "per_0123456789abcdefghijkl"
	if err := r.RegistrarAuditoriaFronteraRutaExacta(context.Background(), para401); !errors.Is(err, vecports.ErrOrdenAuditoriaFronteraRutaExactaInvalida) || len(c.consultas) != antes {
		t.Fatal("401 con actor aceptado")
	}
	para403 = ordenDenegacionPrueba(vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado)
	para403.ActorRef = ""
	if err := r.RegistrarAuditoriaFronteraRutaExacta(context.Background(), para403); !errors.Is(err, vecports.ErrOrdenAuditoriaFronteraRutaExactaInvalida) || len(c.consultas) != antes {
		t.Fatal("403 sin persona verificada aceptado")
	}
	para403 = ordenDenegacionPrueba(vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado)
	para403.Ruta += "?actor=privado"
	if err := r.RegistrarAuditoriaFronteraRutaExacta(context.Background(), para403); !errors.Is(err, vecports.ErrOrdenAuditoriaFronteraRutaExactaInvalida) || len(c.consultas) != antes {
		t.Fatal("ruta con query enviada a SQL")
	}
}

func TestRegistroConCancelacionDesacopladaYPlazo(t *testing.T) {
	c := &consultorDenegacionPrueba{preflight: true, guardado: true}
	r, _ := nuevoRegistradorDenegacionPreferenciasPostgreSQL(context.Background(), c)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if err := r.RegistrarAuditoriaFronteraRutaExacta(ctx, ordenDenegacionPrueba(vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado)); err != nil {
		t.Fatal(err)
	}
	ctxRegistro := c.consultas[1].ctx
	plazo, tienePlazo := ctxRegistro.Deadline()
	if c.consultas[1].estadoAlLlamar != nil || !tienePlazo || time.Until(plazo) > 3*time.Second || time.Until(plazo) <= 0 {
		t.Fatal("auditoria no desacoplada o sin plazo acotado")
	}
}

func TestErrorSQLNoExponeDetalles(t *testing.T) {
	c := &consultorDenegacionPrueba{preflight: true, guardado: false}
	r, _ := nuevoRegistradorDenegacionPreferenciasPostgreSQL(context.Background(), c)
	c.err = &pgconn.PgError{Code: "42501", Message: "persona privada y DSN"}
	err := r.RegistrarAuditoriaFronteraRutaExacta(context.Background(), ordenDenegacionPrueba(vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado))
	if !errors.Is(err, ErrAuditoriaFronteraPreferenciasNoDisponible) || strings.Contains(err.Error(), "privada") || strings.Contains(err.Error(), "DSN") {
		t.Fatalf("error no redactado: %v", err)
	}
	if strings.Contains(registrarDenegacionSQL, "INSERT") || strings.Contains(registrarDenegacionSQL, "UPDATE") {
		t.Fatal("DML directo")
	}
}
