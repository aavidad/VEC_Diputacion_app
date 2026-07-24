package postgres

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type filaAcreditacionPoolAltaPrueba struct {
	valores []bool
	err     error
}

func (f filaAcreditacionPoolAltaPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(destinos) != len(f.valores) {
		return errors.New("columnas inesperadas")
	}
	for indice, destino := range destinos {
		puntero, valido := destino.(*bool)
		if !valido {
			return errors.New("destino inesperado")
		}
		*puntero = f.valores[indice]
	}
	return nil
}

type consultorAcreditacionPoolAltaPrueba struct {
	fila     pgx.Row
	consulta string
}

func (c *consultorAcreditacionPoolAltaPrueba) QueryRow(
	_ context.Context,
	consulta string,
	_ ...any,
) pgx.Row {
	c.consulta = consulta
	return c.fila
}

type filaAcreditacionPoolAltaCancelada struct {
	cancelar context.CancelFunc
}

func (f filaAcreditacionPoolAltaCancelada) Scan(...any) error {
	f.cancelar()
	return errors.New("detalle interno de cancelación")
}

func comprobacionesPoolAltaPrueba(valor bool) []bool {
	resultado := make(
		[]bool,
		totalComprobacionesAcreditacionPoolAlta,
	)
	for indice := range resultado {
		resultado[indice] = valor
	}
	return resultado
}

func TestAcreditarPoolAltasPostgreSQLExigeContratoNominalCompleto(
	t *testing.T,
) {
	consultor := &consultorAcreditacionPoolAltaPrueba{
		fila: filaAcreditacionPoolAltaPrueba{
			valores: comprobacionesPoolAltaPrueba(true),
		},
	}
	if err := acreditarPoolAltasPostgreSQL(
		context.Background(),
		consultor,
	); err != nil {
		t.Fatal(err)
	}
	for _, fragmento := range []string{
		"current_user = session_user",
		"vec_contratacion_temporal_ejecutor",
		"resolver_candidatura_alta_tecnica_v1",
		"confirmar_alta_atestada_v2",
		"NOT has_function_privilege",
		"preparar_alta_v2",
		"reconciliar_agregado_alta_v1",
		"NOT has_table_privilege",
		"identidad_reserva_alta",
		"expediente_alta",
		"candidatura_alta_tecnica",
	} {
		if !strings.Contains(consultor.consulta, fragmento) {
			t.Errorf("acreditación sin %q", fragmento)
		}
	}
}

func TestAcreditarPoolAltasPostgreSQLFallaCerradoPorCadaComprobacion(
	t *testing.T,
) {
	for indice := range totalComprobacionesAcreditacionPoolAlta {
		t.Run(strconv.Itoa(indice), func(t *testing.T) {
			comprobaciones := comprobacionesPoolAltaPrueba(true)
			comprobaciones[indice] = false
			consultor := &consultorAcreditacionPoolAltaPrueba{
				fila: filaAcreditacionPoolAltaPrueba{
					valores: comprobaciones,
				},
			}
			err := acreditarPoolAltasPostgreSQL(
				context.Background(),
				consultor,
			)
			if !errors.Is(err, ports.ErrPersistenciaNoDisponible) {
				t.Fatalf("comprobación %d aceptada: %v", indice, err)
			}
		})
	}
}

func TestAcreditarPoolAltasPostgreSQLSaneaErroresYCancelacion(t *testing.T) {
	consultor := &consultorAcreditacionPoolAltaPrueba{
		fila: filaAcreditacionPoolAltaPrueba{
			err: errors.New("dsn=MARCADOR_SECRETO"),
		},
	}
	err := acreditarPoolAltasPostgreSQL(context.Background(), consultor)
	if !errors.Is(err, ports.ErrPersistenciaNoDisponible) ||
		strings.Contains(err.Error(), "MARCADOR") {
		t.Fatalf("error no saneado: %v", err)
	}

	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	err = acreditarPoolAltasPostgreSQL(ctx, consultor)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelación perdida: %v", err)
	}

	ctxDuranteConsulta, cancelarDuranteConsulta := context.WithCancel(
		context.Background(),
	)
	consultor.fila = filaAcreditacionPoolAltaCancelada{
		cancelar: cancelarDuranteConsulta,
	}
	err = acreditarPoolAltasPostgreSQL(ctxDuranteConsulta, consultor)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelación durante consulta perdida: %v", err)
	}
}
