package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type secuenciaComunicacionesPrueba struct {
	transacciones []*transaccionEjecucionSeleccionO6Prueba
	inicios       int
}

func (s *secuenciaComunicacionesPrueba) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	if opciones.IsoLevel != pgx.Serializable || opciones.AccessMode != pgx.ReadWrite {
		return nil, errors.New("opciones fuera de contrato")
	}
	s.inicios++
	return s.transacciones[min(s.inicios, len(s.transacciones))-1], nil
}

func TestComunicacionesReintentaCarreraConAutorizacionNuevaTrasRollback(t *testing.T) {
	t.Parallel()
	for _, codigo := range []string{"40001", "40P01"} {
		for _, etapa := range []string{"configurar", "consulta", "commit"} {
			t.Run(codigo+"/"+etapa, func(t *testing.T) {
				t.Parallel()
				c := ports.ConsultaComunicacionesExpediente{ExpedienteRef: "expediente:f124", Limite: 10}
				fila := filaEjecucionSeleccionO6Prueba{valores: []any{`{"encontrado":true,"expediente_ref":"expediente:f124","comunicaciones":[],"siguiente_cursor":""}`}}
				primera := &transaccionEjecucionSeleccionO6Prueba{fila: fila}
				segunda := &transaccionEjecucionSeleccionO6Prueba{fila: fila}
				carrera := &pgconn.PgError{Code: codigo}
				switch etapa {
				case "configurar":
					primera.errConfigurar = carrera
				case "consulta":
					primera.fila = filaEjecucionSeleccionO6Prueba{err: carrera}
				case "commit":
					primera.errCommit = carrera
				}
				pool := &secuenciaComunicacionesPrueba{transacciones: []*transaccionEjecucionSeleccionO6Prueba{primera, segunda}}
				autorizaciones := 0
				proveedor := proveedorComunicacionesExpedientePrueba(func(_ context.Context, entrada ports.ConsultaComunicacionesExpediente) (AmbitoConsultaComunicacionesExpediente, error) {
					autorizaciones++
					if entrada != c || (autorizaciones > 1 && primera.reversiones != 1) {
						t.Fatal("nuevo permiso antes del rollback o consulta cambiada")
					}
					return autorizacionComunicacionesExpedientePrueba(t, entrada, "organizacion:f124"), nil
				})
				l := &LectorComunicacionesExpedientePostgreSQL{pool: pool, proveedor: proveedor}
				p, err := l.ConsultarComunicacionesExpediente(context.Background(), c)
				if err != nil || p.ValidarPara(c) != nil || pool.inicios != 2 || autorizaciones != 2 || segunda.confirmaciones != 1 || segunda.reversiones != 1 {
					t.Fatalf("pagina=%+v error=%v intentos=%d permisos=%d", p, err, pool.inicios, autorizaciones)
				}
			})
		}
	}
}

func TestComunicacionesNoReintentaGuardasNiBloqueos(t *testing.T) {
	t.Parallel()
	for _, codigo := range []string{"P1403", "P1405", "55P03", "08006"} {
		t.Run(codigo, func(t *testing.T) {
			t.Parallel()
			c := ports.ConsultaComunicacionesExpediente{ExpedienteRef: "expediente:f124", Limite: 10}
			tx := &transaccionEjecucionSeleccionO6Prueba{fila: filaEjecucionSeleccionO6Prueba{err: &pgconn.PgError{Code: codigo}}}
			pool := &secuenciaComunicacionesPrueba{transacciones: []*transaccionEjecucionSeleccionO6Prueba{tx}}
			proveedor := proveedorComunicacionesExpedientePrueba(func(_ context.Context, entrada ports.ConsultaComunicacionesExpediente) (AmbitoConsultaComunicacionesExpediente, error) {
				return autorizacionComunicacionesExpedientePrueba(t, entrada, "organizacion:f124"), nil
			})
			l := &LectorComunicacionesExpedientePostgreSQL{pool: pool, proveedor: proveedor}
			p, err := l.ConsultarComunicacionesExpediente(context.Background(), c)
			esperado := ports.ErrConsultaComunicacionesExpedienteNoDisponible
			if codigo == "P1403" {
				esperado = ports.ErrConsultaComunicacionesExpedienteDenegada
			}
			if !errors.Is(err, esperado) || p.ExpedienteRef != "" || pool.inicios != 1 || tx.confirmaciones != 0 || tx.reversiones != 1 {
				t.Fatalf("pagina=%+v error=%v intentos=%d", p, err, pool.inicios)
			}
		})
	}
}
