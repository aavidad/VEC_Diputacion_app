package application

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

type sesionLecturaPrimariaSQLStateDecisionCoberturaPrueba struct {
	errorSQL  error
	lecturas  atomic.Int32
	consultas atomic.Int32
}

func (s *sesionLecturaPrimariaSQLStateDecisionCoberturaPrueba) LeerTerminalPrimario(
	_ context.Context,
	consulta cobertura.ConsultaPrimariaSesionTCBOperacionDecisionCobertura,
) (cobertura.DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura, error) {
	s.lecturas.Add(1)
	if _, err := consulta.Datos(); err != nil {
		return cobertura.DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura{},
			err
	}
	s.consultas.Add(1)
	return cobertura.DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura{},
		s.errorSQL
}

type ejecutorLecturaPrimariaSQLStateDecisionCoberturaPrueba struct {
	sesion cobertura.SesionLecturaPrimariaTCBOperacionDecisionCobertura
	ciclos atomic.Int32
}

func (e *ejecutorLecturaPrimariaSQLStateDecisionCoberturaPrueba) EjecutarLecturaPrimariaTCB(
	ctx context.Context,
	usar func(cobertura.SesionLecturaPrimariaTCBOperacionDecisionCobertura) error,
) error {
	e.ciclos.Add(1)
	return usar(e.sesion)
}

func TestDecidirCoberturaSQLStatePrimarioNoReintentaNiPublicaExito(
	t *testing.T,
) {
	t.Parallel()
	for _, codigo := range []string{"40001", "40P01"} {
		codigo := codigo
		t.Run(codigo, func(t *testing.T) {
			t.Parallel()
			escenario := nuevoEscenarioConfirmacionCobertura(t, true)
			escenario.transaccion.ambigua = true
			sesion := &sesionLecturaPrimariaSQLStateDecisionCoberturaPrueba{
				errorSQL: &pgconn.PgError{
					Code: codigo, Message: "detalle interno reservado",
				},
			}
			ejecutor := &ejecutorLecturaPrimariaSQLStateDecisionCoberturaPrueba{
				sesion: sesion,
			}
			reconciliador, err :=
				cobertura.NuevoReconciliadorResultadoAmbiguoOperacionDecisionCoberturaTCB(
					ejecutor,
				)
			if err != nil {
				t.Fatalf("crear reconciliador: %v", err)
			}
			escenario.servicio.reconciliador = reconciliador

			recibo, err := escenario.servicio.Decidir(
				context.Background(),
				escenario.solicitud,
			)
			if !errors.Is(
				err,
				ErrConfirmacionDecisionCoberturaNoDisponible,
			) {
				t.Fatalf("SQLSTATE %s publicó un resultado: %v", codigo, err)
			}
			if recibo != (cobertura.ReciboOperacionDecisionCobertura{}) {
				t.Fatalf("SQLSTATE %s publicó un recibo: %+v", codigo, recibo)
			}
			if escenario.transaccion.total() != 1 ||
				ejecutor.ciclos.Load() != 1 ||
				sesion.lecturas.Load() != 1 ||
				sesion.consultas.Load() != 1 {
				t.Fatalf(
					"SQLSTATE %s provocó reintento: transacción=%d "+
						"ciclos=%d lecturas=%d consultas=%d",
					codigo,
					escenario.transaccion.total(),
					ejecutor.ciclos.Load(),
					sesion.lecturas.Load(),
					sesion.consultas.Load(),
				)
			}
		})
	}
}
