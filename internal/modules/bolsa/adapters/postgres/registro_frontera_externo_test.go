package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type filaRegistroFronteraBolsaPrueba struct {
	valor bool
	err   error
}

func (f filaRegistroFronteraBolsaPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	*(destinos[0].(*bool)) = f.valor
	return nil
}

type consultorRegistroFronteraBolsaPrueba struct {
	preflight  bool
	guardado   bool
	consultas  int
	argumentos []any
}

func (c *consultorRegistroFronteraBolsaPrueba) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	c.consultas++
	if sql == preflightFronteraBolsaExternaSQL {
		return filaRegistroFronteraBolsaPrueba{valor: c.preflight}
	}
	c.argumentos = args
	return filaRegistroFronteraBolsaPrueba{valor: c.guardado}
}

func ordenRegistroFronteraBolsaPrueba() vecports.OrdenAuditoriaFronteraRutaExacta {
	return vecports.OrdenAuditoriaFronteraRutaExacta{
		CorrelacionRef: "corr_" + strings.Repeat("a", 32),
		Motivo:         vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado,
		Superficie:     superficieRegistroBolsaExterna,
		Ruta:           "/api/vec/bolsa/mi-bolsa/contacto",
		ActorRef:       "per_" + strings.Repeat("b", 22),
	}
}

func TestRegistroFronteraBolsaExternaPreflightYOrdenCerrados(t *testing.T) {
	denegado := &consultorRegistroFronteraBolsaPrueba{}
	if _, err := nuevoRegistradorFronteraBolsaExternaPostgreSQL(context.Background(), denegado); !errors.Is(err, ErrRegistroFronteraBolsaExternaNoDisponible) {
		t.Fatalf("preflight negativo: %v", err)
	}
	consultor := &consultorRegistroFronteraBolsaPrueba{preflight: true, guardado: true}
	registrador, err := nuevoRegistradorFronteraBolsaExternaPostgreSQL(context.Background(), consultor)
	if err != nil {
		t.Fatal(err)
	}
	orden := ordenRegistroFronteraBolsaPrueba()
	if err := registrador.RegistrarAuditoriaFronteraRutaExacta(context.Background(), orden); err != nil {
		t.Fatal(err)
	}
	if consultor.consultas != 2 || consultor.argumentos[4] != orden.ActorRef {
		t.Fatalf("registro no encaminado: consultas=%d", consultor.consultas)
	}
	for _, mutar := range []func(*vecports.OrdenAuditoriaFronteraRutaExacta){
		func(o *vecports.OrdenAuditoriaFronteraRutaExacta) {
			o.Superficie = "api.usuarios.preferencias.ruta_exacta"
		},
		func(o *vecports.OrdenAuditoriaFronteraRutaExacta) { o.Ruta += "?dato=privado" },
		func(o *vecports.OrdenAuditoriaFronteraRutaExacta) { o.ActorRef = "per_123" },
		func(o *vecports.OrdenAuditoriaFronteraRutaExacta) { o.CorrelacionRef = "peticion-libre" },
		func(o *vecports.OrdenAuditoriaFronteraRutaExacta) {
			o.Motivo = vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida
		},
	} {
		invalida := orden
		mutar(&invalida)
		if err := registrador.RegistrarAuditoriaFronteraRutaExacta(context.Background(), invalida); !errors.Is(err, vecports.ErrOrdenAuditoriaFronteraRutaExactaInvalida) {
			t.Fatalf("orden abierta: %v", err)
		}
	}
	if consultor.consultas != 2 {
		t.Fatalf("escribió orden inválida: %d", consultor.consultas)
	}
	orden.Motivo = vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida
	orden.ActorRef = ""
	if err := registrador.RegistrarAuditoriaFronteraRutaExacta(context.Background(), orden); err != nil {
		t.Fatal(err)
	}
	consultor.guardado = false
	if err := registrador.RegistrarAuditoriaFronteraRutaExacta(context.Background(), orden); !errors.Is(err, ErrRegistroFronteraBolsaExternaNoDisponible) {
		t.Fatalf("fallo de escritura permitido: %v", err)
	}
}
