package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
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
		Motivo:         motivo, Superficie: superficieAuditoriaPreferencias, Ruta: rutaAuditoriaPreferenciasInterna,
	}
	if motivo == vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado {
		o.ActorRef = "per_0123456789abcdefghijkl"
	}
	return o
}

func TestDenegacionPreflightExigeRolYACLEfectiva(t *testing.T) {
	c := &consultorDenegacionPrueba{preflight: true}
	r, err := nuevoRegistradorDenegacionPreferenciasPostgreSQL(context.Background(), c, vecdomain.SuperficieAutenticacionInternaCorporativaV1)
	if err != nil || r == nil || len(c.consultas) != 1 || c.consultas[0].sql != preflightDenegacionSQL ||
		len(c.consultas[0].args) != 1 || c.consultas[0].args[0] != "vec_usuarios_registrador_frontera_interno" {
		t.Fatalf("preflight: %v", err)
	}
	for _, fragmento := range []string{"session_user=current_user", "g.rolname=$1", "m.inherit_option", "NOT m.set_option", "NOT m.admin_option", "has_function_privilege", "has_table_privilege", "has_sequence_privilege", "pg_catalog.aclexplode", "p.proowner IN (l.oid,g.oid)", "NOT pg_catalog.pg_is_in_recovery()"} {
		if !strings.Contains(c.consultas[0].sql, fragmento) {
			t.Fatalf("falta guarda efectiva: %s", fragmento)
		}
	}
	c = &consultorDenegacionPrueba{}
	if _, err := nuevoRegistradorDenegacionPreferenciasPostgreSQL(context.Background(), c, vecdomain.SuperficieAutenticacionInternaCorporativaV1); !errors.Is(err, ErrAuditoriaFronteraPreferenciasNoDisponible) {
		t.Fatal("rol no acreditado aceptado")
	}
}

func TestDenegacionSeparaRutaYRolPorSuperficie(t *testing.T) {
	if _, err := nuevoRegistradorDenegacionPreferenciasPostgreSQL(context.Background(), &consultorDenegacionPrueba{preflight: true}, "indefinida"); !errors.Is(err, ErrAuditoriaFronteraPreferenciasNoDisponible) {
		t.Fatal("superficie desconocida aceptada")
	}
	c := &consultorDenegacionPrueba{preflight: true, guardado: true}
	r, err := nuevoRegistradorDenegacionPreferenciasPostgreSQL(context.Background(), c, vecdomain.SuperficieAutenticacionExternaPersonalV1)
	if err != nil || c.consultas[0].args[0] != "vec_usuarios_registrador_frontera_externo" {
		t.Fatalf("rol exterior: %v", err)
	}
	interna := ordenDenegacionPrueba(vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado)
	if err := r.RegistrarAuditoriaFronteraRutaExacta(context.Background(), interna); !errors.Is(err, vecports.ErrOrdenAuditoriaFronteraRutaExactaInvalida) || len(c.consultas) != 1 {
		t.Fatal("pool exterior aceptó ruta interna")
	}
	externa := interna
	externa.Ruta = rutaAuditoriaPreferenciasExterna
	if err := r.RegistrarAuditoriaFronteraRutaExacta(context.Background(), externa); err != nil || len(c.consultas) != 2 || c.consultas[1].args[3] != rutaAuditoriaPreferenciasExterna {
		t.Fatalf("ruta exterior: %v", err)
	}
	ci := &consultorDenegacionPrueba{preflight: true, guardado: true}
	ri, _ := nuevoRegistradorDenegacionPreferenciasPostgreSQL(context.Background(), ci, vecdomain.SuperficieAutenticacionInternaCorporativaV1)
	if err := ri.RegistrarAuditoriaFronteraRutaExacta(context.Background(), externa); !errors.Is(err, vecports.ErrOrdenAuditoriaFronteraRutaExactaInvalida) || len(ci.consultas) != 1 {
		t.Fatal("pool interno aceptó ruta exterior")
	}
}

func TestRegistra401ConActorSQLNullY403SoloConPersona(t *testing.T) {
	c := &consultorDenegacionPrueba{preflight: true, guardado: true}
	r, _ := nuevoRegistradorDenegacionPreferenciasPostgreSQL(context.Background(), c, vecdomain.SuperficieAutenticacionInternaCorporativaV1)
	para401 := ordenDenegacionPrueba(vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida)
	if err := r.RegistrarAuditoriaFronteraRutaExacta(context.Background(), para401); err != nil {
		t.Fatal(err)
	}
	if len(c.consultas) != 2 || c.consultas[1].sql != registrarDenegacionSQL || len(c.consultas[1].args) != 5 ||
		c.consultas[1].args[0] != para401.CorrelacionRef || c.consultas[1].args[1] != string(para401.Motivo) ||
		c.consultas[1].args[2] != superficieAuditoriaPreferencias || c.consultas[1].args[3] != rutaAuditoriaPreferenciasInterna ||
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
	r, _ := nuevoRegistradorDenegacionPreferenciasPostgreSQL(context.Background(), c, vecdomain.SuperficieAutenticacionInternaCorporativaV1)
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
	r, _ := nuevoRegistradorDenegacionPreferenciasPostgreSQL(context.Background(), c, vecdomain.SuperficieAutenticacionInternaCorporativaV1)
	c.err = &pgconn.PgError{Code: "42501", Message: "persona privada y DSN"}
	err := r.RegistrarAuditoriaFronteraRutaExacta(context.Background(), ordenDenegacionPrueba(vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado))
	if !errors.Is(err, ErrAuditoriaFronteraPreferenciasNoDisponible) || strings.Contains(err.Error(), "privada") || strings.Contains(err.Error(), "DSN") {
		t.Fatalf("error no redactado: %v", err)
	}
	if strings.Contains(registrarDenegacionSQL, "INSERT") || strings.Contains(registrarDenegacionSQL, "UPDATE") {
		t.Fatal("DML directo")
	}
}
