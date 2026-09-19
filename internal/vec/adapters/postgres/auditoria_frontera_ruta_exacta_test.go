package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/ports"
)

type filaAuditoriaFronteraRutaExactaPrueba struct {
	registrada bool
	err        error
}

func (f filaAuditoriaFronteraRutaExactaPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(destinos) != 1 {
		return errors.New("destino invalido")
	}
	destino, ok := destinos[0].(*bool)
	if !ok {
		return errors.New("tipo de destino invalido")
	}
	*destino = f.registrada
	return nil
}

type consultorAuditoriaFronteraRutaExactaPrueba struct {
	consulta   string
	argumentos []any
	fila       pgx.Row
}

func (c *consultorAuditoriaFronteraRutaExactaPrueba) QueryRow(
	_ context.Context, consulta string, argumentos ...any,
) pgx.Row {
	c.consulta = consulta
	c.argumentos = append([]any(nil), argumentos...)
	return c.fila
}

func ordenAuditoriaFronteraRutaExactaPrueba() ports.OrdenAuditoriaFronteraRutaExacta {
	return ports.OrdenAuditoriaFronteraRutaExacta{
		CorrelacionRef: "corr_0123456789abcdef0123456789abcdef",
		Motivo:         ports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida,
		Superficie:     ports.SuperficieAuditoriaFronteraRutaExactaContratacionTemporal,
		Ruta:           "/api/vec/contratacion-temporal/expedientes",
	}
}

func TestRegistradorAuditoriaFronteraRutaExactaPostgreSQLTransportaSoloContratoMinimizado(t *testing.T) {
	t.Parallel()
	consultor := &consultorAuditoriaFronteraRutaExactaPrueba{
		fila: filaAuditoriaFronteraRutaExactaPrueba{registrada: true},
	}
	registrador, err := nuevoRegistradorAuditoriaFronteraRutaExactaPostgreSQL(consultor)
	if err != nil {
		t.Fatal(err)
	}
	orden := ordenAuditoriaFronteraRutaExactaPrueba()
	if err = registrador.RegistrarAuditoriaFronteraRutaExacta(context.Background(), orden); err != nil {
		t.Fatalf("registrar: %v", err)
	}
	if !strings.Contains(consultor.consulta, "registrar_auditoria_frontera_ruta_exacta_v1") {
		t.Fatalf("funcion SQL distinta: %q", consultor.consulta)
	}
	if len(consultor.argumentos) != 5 {
		t.Fatalf("argumentos: %#v", consultor.argumentos)
	}
	esperados := []any{orden.CorrelacionRef, string(orden.Motivo), orden.Superficie, orden.Ruta, ""}
	for indice := range esperados {
		if consultor.argumentos[indice] != esperados[indice] {
			t.Fatalf("argumento %d: obtenido=%#v esperado=%#v", indice, consultor.argumentos[indice], esperados[indice])
		}
	}
}

func TestRegistradorAuditoriaFronteraRutaExactaPostgreSQLRechazaAntesDeSQL(t *testing.T) {
	t.Parallel()
	consultor := &consultorAuditoriaFronteraRutaExactaPrueba{fila: filaAuditoriaFronteraRutaExactaPrueba{registrada: true}}
	registrador, err := nuevoRegistradorAuditoriaFronteraRutaExactaPostgreSQL(consultor)
	if err != nil {
		t.Fatal(err)
	}
	orden := ordenAuditoriaFronteraRutaExactaPrueba()
	orden.Ruta += "?dato=privado"
	err = registrador.RegistrarAuditoriaFronteraRutaExacta(context.Background(), orden)
	if !errors.Is(err, ports.ErrOrdenAuditoriaFronteraRutaExactaInvalida) || consultor.consulta != "" {
		t.Fatalf("orden invalida llego a SQL: error=%v consulta=%q", err, consultor.consulta)
	}
}

func TestRegistradorAuditoriaFronteraRutaExactaPostgreSQLOcultaFalloInfraestructura(t *testing.T) {
	t.Parallel()
	consultor := &consultorAuditoriaFronteraRutaExactaPrueba{
		fila: filaAuditoriaFronteraRutaExactaPrueba{err: errors.New("postgresql://usuario:secreto@host/auditoria")},
	}
	registrador, err := nuevoRegistradorAuditoriaFronteraRutaExactaPostgreSQL(consultor)
	if err != nil {
		t.Fatal(err)
	}
	err = registrador.RegistrarAuditoriaFronteraRutaExacta(context.Background(), ordenAuditoriaFronteraRutaExactaPrueba())
	if !errors.Is(err, ErrAuditoriaFronteraRutaExactaNoDisponible) || strings.Contains(err.Error(), "secreto") {
		t.Fatalf("error no cerrado: %v", err)
	}
}

func TestNuevoRegistradorAuditoriaFronteraRutaExactaPostgreSQLRechazaPoolNuloTipado(t *testing.T) {
	t.Parallel()
	var pool *pgxpool.Pool
	registrador, err := NuevoRegistradorAuditoriaFronteraRutaExactaPostgreSQL(pool)
	if registrador != nil || !errors.Is(err, ErrAuditoriaFronteraRutaExactaNoDisponible) {
		t.Fatalf("pool nulo tipado aceptado: registrador=%#v error=%v", registrador, err)
	}
}

func TestPreflightAuditoriaFronteraRutaExactaPostgreSQLExigeFirmaYExecuteExactos(t *testing.T) {
	t.Parallel()
	consultor := &consultorAuditoriaFronteraRutaExactaPrueba{
		fila: filaAuditoriaFronteraRutaExactaPrueba{registrada: true},
	}
	registrador, err := nuevoRegistradorAuditoriaFronteraRutaExactaPostgreSQL(consultor)
	if err != nil {
		t.Fatal(err)
	}
	if err = registrador.PreflightAuditoriaFronteraRutaExacta(context.Background()); err != nil {
		t.Fatalf("preflight: %v", err)
	}
	for _, fragmento := range []string{
		"has_function_privilege",
		"registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text)",
		"EXECUTE",
	} {
		if !strings.Contains(consultor.consulta, fragmento) {
			t.Fatalf("preflight sin %q: %q", fragmento, consultor.consulta)
		}
	}
}
