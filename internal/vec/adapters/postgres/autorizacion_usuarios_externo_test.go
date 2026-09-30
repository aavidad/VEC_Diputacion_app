package postgres

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/ports"
)

type iniciadorFuenteUsuariosExternoPrueba struct {
	errores  []error
	opciones []pgx.TxOptions
}

// Activar solo sobre un clon sintético con CTX-14, AUT-17 y provisión explícita.
// Ejercita la fuente real: la prueba unitaria anterior comprueba la carrera.
func TestFuenteUsuariosExternoPG18(t *testing.T) {
	dsn := os.Getenv("VEC_AD3_118_DSN_FUENTE")
	persona := os.Getenv("VEC_AD3_118_PERSONA_REF")
	perfil := os.Getenv("VEC_AD3_118_PERFIL_REF")
	if dsn == "" && persona == "" && perfil == "" {
		t.Skip("requiere clon sintético provisionado")
	}
	if dsn == "" || persona == "" || perfil == "" {
		t.Fatal("fixture PG18 incompleta")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal("pool del clon no disponible")
	}
	t.Cleanup(pool.Close)
	almacen, err := NuevoAlmacenAutorizacionUsuariosExterno(pool)
	if err != nil {
		t.Fatal(err)
	}
	instantanea, err := almacen.ObtenerInstantaneaAutorizacion(ctx, persona, perfil)
	if err != nil || instantanea.Validar() != nil ||
		instantanea.AsignacionPerfil.PrincipalID != persona || instantanea.AsignacionPerfil.PerfilActivoRef != perfil {
		t.Fatalf("instantánea externa no acreditada: %v", err)
	}
}

func (i *iniciadorFuenteUsuariosExternoPrueba) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	i.opciones = append(i.opciones, opciones)
	indice := len(i.opciones) - 1
	if indice >= len(i.errores) {
		return nil, pgx.ErrNoRows
	}
	return nil, i.errores[indice]
}

func TestFuenteUsuariosExternoExigeSerializableYReintentaCarrera(t *testing.T) {
	for _, codigo := range []string{"40001", "40P01"} {
		t.Run(codigo, func(t *testing.T) {
			iniciador := &iniciadorFuenteUsuariosExternoPrueba{errores: []error{&pgconn.PgError{Code: codigo}}}
			almacen, err := nuevoAlmacenAutorizacionUsuariosExterno(iniciador)
			if err != nil {
				t.Fatal(err)
			}
			_, err = almacen.ObtenerInstantaneaAutorizacion(context.Background(), "principal:sintetico", "perfil:sintetico")
			if !errors.Is(err, ports.ErrAsignacionPerfilNoEncontrada) || len(iniciador.opciones) != 2 {
				t.Fatalf("carrera no recuperada: intentos=%d error=%v", len(iniciador.opciones), err)
			}
			for _, opciones := range iniciador.opciones {
				if opciones.IsoLevel != pgx.Serializable || opciones.AccessMode != pgx.ReadWrite {
					t.Fatalf("transacción no admitida por CTX-14: %+v", opciones)
				}
			}
		})
	}
	t.Run("denegacion no se reintenta ni filtra", func(t *testing.T) {
		iniciador := &iniciadorFuenteUsuariosExternoPrueba{errores: []error{&pgconn.PgError{Code: "42501", Message: "dato_privado"}}}
		almacen, _ := nuevoAlmacenAutorizacionUsuariosExterno(iniciador)
		_, err := almacen.ObtenerInstantaneaAutorizacion(context.Background(), "principal:sintetico", "perfil:sintetico")
		if err != ports.ErrFuenteAutorizacionNoDisponible || len(iniciador.opciones) != 1 {
			t.Fatalf("denegación mal clasificada: intentos=%d error=%v", len(iniciador.opciones), err)
		}
	})
}
