package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

type transaccionSQLStateDecisionCoberturaO404EPrueba struct {
	*transaccionPreparacionPrueba
	consultas int
}

func (t *transaccionSQLStateDecisionCoberturaO404EPrueba) QueryRow(
	ctx context.Context,
	consulta string,
	argumentos ...any,
) pgx.Row {
	t.consultas++
	return t.transaccionPreparacionPrueba.QueryRow(
		ctx,
		consulta,
		argumentos...,
	)
}

func TestConfirmacionDecisionCoberturaO404ENoReintentaSQLStateTransaccional(
	t *testing.T,
) {
	t.Parallel()
	for _, codigo := range []string{"40001", "40P01"} {
		codigo := codigo
		t.Run(codigo, func(t *testing.T) {
			t.Parallel()
			errorSQL := &pgconn.PgError{
				Code: codigo, Message: "detalle interno reservado",
			}
			base := &transaccionPreparacionPrueba{
				fila: filaBytesDecisionCoberturaO404EPrueba{err: errorSQL},
			}
			tx := &transaccionSQLStateDecisionCoberturaO404EPrueba{
				transaccionPreparacionPrueba: base,
			}
			iniciador := &iniciadorPreparacionPrueba{
				transacciones: []pgx.Tx{
					tx,
					&transaccionPreparacionPrueba{},
				},
			}
			ejecutor, err :=
				nuevoEjecutorSesionTCBOperacionDecisionCoberturaPostgreSQL(
					iniciador,
				)
			if err != nil {
				t.Fatalf("crear ejecutor: %v", err)
			}
			err = ejecutor.EjecutarSesionTCB(
				context.Background(),
				func(
					puerto cobertura.SesionTCBOperacionDecisionCobertura,
				) error {
					sesion, ok := puerto.(*sesionDecisionCoberturaO404E)
					if !ok {
						return errors.New("tipo de sesión inesperado")
					}
					prepararSesionDenegadaSQLStateDecisionCoberturaO404EPrueba(
						t,
						sesion,
					)
					_, errConfirmacion := sesion.Confirmar(
						context.Background(),
					)
					return errConfirmacion
				},
			)
			var recibido *pgconn.PgError
			if !errors.As(err, &recibido) || recibido.Code != codigo {
				t.Fatalf("SQLSTATE %s no se conservó: %v", codigo, err)
			}
			if iniciador.inicios != 1 || tx.consultas != 1 ||
				tx.confirmaciones != 0 || tx.reversiones != 1 {
				t.Fatalf(
					"SQLSTATE %s provocó reintento: begin=%d consulta=%d "+
						"commit=%d rollback=%d",
					codigo,
					iniciador.inicios,
					tx.consultas,
					tx.confirmaciones,
					tx.reversiones,
				)
			}
		})
	}
}

func prepararSesionDenegadaSQLStateDecisionCoberturaO404EPrueba(
	t *testing.T,
	sesion *sesionDecisionCoberturaO404E,
) {
	t.Helper()
	recurso := recursoDenegacionDecisionCoberturaO404EPrueba()
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatalf("calcular huella del recurso: %v", err)
	}
	sesion.estado = estadoSesionDecisionCoberturaLista
	sesion.rama = cobertura.RamaSesionTCBOperacionDecisionCoberturaDenegada
	sesion.carga = cargaConfirmarDecisionCoberturaO404E{
		Esquema: esquemaCargaDecisionCoberturaO404E,
		Rama:    cobertura.RamaSesionTCBOperacionDecisionCoberturaDenegada,
		DecisionVEC: decisionVECDecisionCoberturaO404E{
			DecisionCanonica:            []byte{1},
			MotivoCanonico:              []byte{2},
			RecursoRef:                  recurso.Referencia,
			RecursoModulo:               recurso.ModuloID,
			RecursoTipo:                 recurso.Tipo,
			Ambitos:                     clonarMapaDecisionCoberturaO404E(recurso.Ambitos),
			Atributos:                   clonarMapaDecisionCoberturaO404E(recurso.Atributos),
			ContextoRecursoHuellaSHA256: huellaRecurso,
		},
		Denegacion: &denegacionDecisionCoberturaO404E{
			RecursoRef:          recurso.Referencia,
			RecursoModulo:       recurso.ModuloID,
			RecursoTipo:         recurso.Tipo,
			Ambitos:             clonarMapaDecisionCoberturaO404E(recurso.Ambitos),
			Atributos:           clonarMapaDecisionCoberturaO404E(recurso.Atributos),
			RecursoHuellaSHA256: huellaRecurso,
		},
		ConsumosC1: []consumoC1DecisionCoberturaO404E{},
	}
}
