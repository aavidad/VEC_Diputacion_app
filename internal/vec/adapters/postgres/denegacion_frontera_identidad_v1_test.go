package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/vec/ports"
)

type filaDenegacionFronteraIdentidadV1Prueba struct {
	valor bool
	err   error
}

func (f filaDenegacionFronteraIdentidadV1Prueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(destinos) != 1 {
		return errors.New("destino invalido")
	}
	destino, ok := destinos[0].(*bool)
	if !ok {
		return errors.New("tipo invalido")
	}
	*destino = f.valor
	return nil
}

type consultorDenegacionFronteraIdentidadV1Prueba struct {
	consulta   string
	argumentos []any
	fila       pgx.Row
}

func (c *consultorDenegacionFronteraIdentidadV1Prueba) QueryRow(_ context.Context, consulta string, argumentos ...any) pgx.Row {
	c.consulta = consulta
	c.argumentos = append([]any(nil), argumentos...)
	return c.fila
}
func ordenDenegacionFronteraIdentidadV1Prueba() ports.OrdenDenegacionFronteraIdentidadV1 {
	return ports.OrdenDenegacionFronteraIdentidadV1{
		CorrelacionRef: "correlacion_0123456789abcdef0123456789abcdef", Superficie: string(ports.SuperficieDenegacionFronteraIdentidadV1ExternaPersonal), RutaExacta: ports.RutaExactaDenegacionFronteraIdentidadV1ParticipacionesPropias, Accion: string(ports.AccionDenegacionFronteraIdentidadV1ConsultarParticipacionesPropias), Motivo: string(ports.MotivoDenegacionFronteraIdentidadV1AutenticacionRequerida), CanalRef: "tls-exportador:sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
}
func TestRegistradorDenegacionFronteraIdentidadV1TransportaSoloContrato(t *testing.T) {
	t.Parallel()
	c := &consultorDenegacionFronteraIdentidadV1Prueba{fila: filaDenegacionFronteraIdentidadV1Prueba{valor: true}}
	r, err := nuevoRegistradorDenegacionFronteraIdentidadV1PostgreSQL(c)
	if err != nil {
		t.Fatal(err)
	}
	o := ordenDenegacionFronteraIdentidadV1Prueba()
	if err = r.RegistrarDenegacionFronteraIdentidadV1(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(c.consulta, "registrar_denegacion_frontera_identidad_v1") || len(c.argumentos) != 7 {
		t.Fatalf("transporte inesperado: %q %#v", c.consulta, c.argumentos)
	}
	for i, esperado := range []any{o.CorrelacionRef, o.Superficie, o.RutaExacta, o.Accion, o.Motivo, o.CanalRef, ""} {
		if c.argumentos[i] != esperado {
			t.Fatalf("arg %d=%#v", i, c.argumentos[i])
		}
	}
}
func TestRegistradorDenegacionFronteraIdentidadV1RechazaAntesDeSQL(t *testing.T) {
	t.Parallel()
	c := &consultorDenegacionFronteraIdentidadV1Prueba{fila: filaDenegacionFronteraIdentidadV1Prueba{valor: true}}
	r, _ := nuevoRegistradorDenegacionFronteraIdentidadV1PostgreSQL(c)
	o := ordenDenegacionFronteraIdentidadV1Prueba()
	o.ActorRef = "actor_no_permitido"
	err := r.RegistrarDenegacionFronteraIdentidadV1(context.Background(), o)
	if !errors.Is(err, ports.ErrOrdenDenegacionFronteraIdentidadV1Invalida) || c.consulta != "" {
		t.Fatalf("orden llego a SQL: %v %q", err, c.consulta)
	}
}
func TestPreflightDenegacionFronteraIdentidadV1ExigeExecuteExacto(t *testing.T) {
	t.Parallel()
	c := &consultorDenegacionFronteraIdentidadV1Prueba{fila: filaDenegacionFronteraIdentidadV1Prueba{valor: true}}
	r, _ := nuevoRegistradorDenegacionFronteraIdentidadV1PostgreSQL(c)
	if err := r.PreflightDenegacionFronteraIdentidadV1(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"has_function_privilege", "registrar_denegacion_frontera_identidad_v1(text,text,text,text,text,text,text)", "EXECUTE"} {
		if !strings.Contains(c.consulta, s) {
			t.Fatalf("falta %q", s)
		}
	}
}
func TestRegistradorDenegacionFronteraIdentidadV1OcultaInfraestructura(t *testing.T) {
	t.Parallel()
	c := &consultorDenegacionFronteraIdentidadV1Prueba{fila: filaDenegacionFronteraIdentidadV1Prueba{err: errors.New("dato-secreto")}}
	r, _ := nuevoRegistradorDenegacionFronteraIdentidadV1PostgreSQL(c)
	err := r.RegistrarDenegacionFronteraIdentidadV1(context.Background(), ordenDenegacionFronteraIdentidadV1Prueba())
	if !errors.Is(err, ports.ErrRegistradorDenegacionFronteraIdentidadV1NoDisponible) || strings.Contains(err.Error(), "secreto") {
		t.Fatalf("error abierto: %v", err)
	}
}
