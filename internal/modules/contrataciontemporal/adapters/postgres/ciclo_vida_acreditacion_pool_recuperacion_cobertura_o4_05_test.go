package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

func TestAcreditacionPoolO405FallaCerradoYLiberando(
	t *testing.T,
) {
	var origenNulo *origenAcreditacionPoolO405Prueba
	if err := acreditarPoolRecuperacionCoberturaO405(
		context.Background(),
		origenNulo,
		modoTLSAcreditacionPoolO405Produccion,
	); !errors.Is(
		err,
		cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible,
	) {
		t.Fatalf("origen nulo tipado aceptado: %v", err)
	}

	var poolNulo *PoolRecuperacionCoberturaO405PostgreSQL
	if ejecutor, err :=
		NuevoEjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQL(
			context.Background(),
			poolNulo,
		); ejecutor != nil || !errors.Is(
		err,
		cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible,
	) {
		t.Fatalf("pool nulo tipado aceptado: ejecutor=%v err=%v", ejecutor, err)
	}

	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	origen := &origenAcreditacionPoolO405Prueba{
		configuracion: configuracionPoolAcreditacionO405Prueba(),
	}
	if err := acreditarPoolRecuperacionCoberturaO405(
		ctxCancelado,
		origen,
		modoTLSAcreditacionPoolO405Produccion,
	); !errors.Is(err, context.Canceled) || origen.adquisiciones != 0 {
		t.Fatalf(
			"contexto cancelado consultó: adquirir=%d err=%v",
			origen.adquisiciones,
			err,
		)
	}

	conexion := &conexionAcreditacionPoolO405Prueba{
		fila: filaAcreditacionPoolO405Prueba{
			err: errors.New("detalle privado PostgreSQL"),
		},
	}
	origen = &origenAcreditacionPoolO405Prueba{
		configuracion: configuracionPoolAcreditacionO405Prueba(),
		conexion:      conexion,
	}
	err := acreditarPoolRecuperacionCoberturaO405(
		context.Background(),
		origen,
		modoTLSAcreditacionPoolO405Produccion,
	)
	if !errors.Is(
		err,
		cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible,
	) || strings.Contains(err.Error(), "detalle") ||
		conexion.liberaciones != 1 {
		t.Fatalf(
			"fallo de fila no saneado/liberado: liberar=%d err=%v",
			conexion.liberaciones,
			err,
		)
	}

	var conexionNula *conexionAcreditacionPoolO405Prueba
	origen = &origenAcreditacionPoolO405Prueba{
		configuracion: configuracionPoolAcreditacionO405Prueba(),
		conexion:      conexionNula,
	}
	if err := acreditarPoolRecuperacionCoberturaO405(
		context.Background(),
		origen,
		modoTLSAcreditacionPoolO405Produccion,
	); !errors.Is(
		err,
		cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible,
	) {
		t.Fatalf("conexión nula tipada aceptada: %v", err)
	}

	conexion = &conexionAcreditacionPoolO405Prueba{
		fila: filaAcreditacionPoolO405Prueba{
			valores: valoresAcreditacionPoolO405Prueba(true),
		},
	}
	origen = &origenAcreditacionPoolO405Prueba{
		configuracion: configuracionPoolAcreditacionO405Prueba(),
		conexion:      conexion,
		err:           errors.New("detalle privado de adquisición"),
	}
	if err := acreditarPoolRecuperacionCoberturaO405(
		context.Background(),
		origen,
		modoTLSAcreditacionPoolO405Produccion,
	); !errors.Is(
		err,
		cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible,
	) || strings.Contains(err.Error(), "detalle") ||
		conexion.liberaciones != 1 || conexion.consulta != "" {
		t.Fatalf(
			"error de adquisición expuesto/fugado: liberar=%d consulta=%q err=%v",
			conexion.liberaciones,
			conexion.consulta,
			err,
		)
	}

	conexion = &conexionAcreditacionPoolO405Prueba{panico: true}
	origen = &origenAcreditacionPoolO405Prueba{
		configuracion: configuracionPoolAcreditacionO405Prueba(),
		conexion:      conexion,
	}
	if err := acreditarPoolRecuperacionCoberturaO405(
		context.Background(),
		origen,
		modoTLSAcreditacionPoolO405Produccion,
	); !errors.Is(
		err,
		cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible,
	) || conexion.liberaciones != 1 {
		t.Fatalf(
			"pánico no saneado/liberado: liberar=%d err=%v",
			conexion.liberaciones,
			err,
		)
	}
}
