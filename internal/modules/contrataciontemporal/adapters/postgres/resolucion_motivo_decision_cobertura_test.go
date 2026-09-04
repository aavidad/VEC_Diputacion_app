package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

type filaMotivoDecisionCoberturaPrueba struct {
	cardinalidad int64
	version      int64
	huella       string
	modulo       string
	entrada      string
	claveI18n    string
	err          error
}

func (f filaMotivoDecisionCoberturaPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(destinos) != 6 {
		return errors.New("destinos inesperados")
	}
	*(destinos[0].(*int64)) = f.cardinalidad
	*(destinos[1].(*int64)) = f.version
	*(destinos[2].(*string)) = f.huella
	*(destinos[3].(*string)) = f.modulo
	*(destinos[4].(*string)) = f.entrada
	*(destinos[5].(*string)) = f.claveI18n
	return nil
}

type consultadorMotivoDecisionCoberturaPrueba struct {
	fila       pgx.Row
	consultas  int
	consulta   string
	argumentos []any
}

func (c *consultadorMotivoDecisionCoberturaPrueba) QueryRow(
	_ context.Context,
	consulta string,
	argumentos ...any,
) pgx.Row {
	c.consultas++
	c.consulta = consulta
	c.argumentos = append([]any(nil), argumentos...)
	return c.fila
}

func TestConsultaMotivoDecisionCoberturaPostgreSQLResuelveFilaExacta(t *testing.T) {
	instante := time.Date(2026, time.September, 4, 1, 0, 0, 0, time.UTC)
	consultador := &consultadorMotivoDecisionCoberturaPrueba{
		fila: filaMotivoDecisionCoberturaPrueba{
			cardinalidad: 1,
			version:      7,
			huella:       strings.Repeat("a", 64),
			modulo:       "contratacion_temporal",
			entrada:      "rectificacion",
			claveI18n:    "contratacion_temporal.cobertura.motivo.rectificacion",
		},
	}
	consulta, err := nuevaConsultaMotivoDecisionCoberturaPostgreSQL(
		consultador,
		"motivos_cobertura",
		"contratacion_temporal",
	)
	if err != nil {
		t.Fatal(err)
	}
	motivo, err := consulta.ConsultarMotivoDecisionCobertura(
		context.Background(),
		"motivos_cobertura",
		"contratacion_temporal",
		domain.ClaveCatalogo("rectificacion"),
		instante,
	)
	if err != nil {
		t.Fatal(err)
	}
	if motivo.ReferenciaCatalogo.CatalogoVersion != 7 ||
		motivo.ReferenciaCatalogo.CatalogoHuellaSHA256 != strings.Repeat("a", 64) ||
		motivo.ReferenciaCatalogo.EntradaClave != "rectificacion" ||
		motivo.ClaveI18n != "contratacion_temporal.cobertura.motivo.rectificacion" {
		t.Fatalf("motivo inesperado: %+v", motivo)
	}
	if consultador.consultas != 1 ||
		!strings.Contains(
			consultador.consulta,
			"vec_autorizacion.resolver_motivo_decision_cobertura_v1",
		) ||
		len(consultador.argumentos) != 3 ||
		consultador.argumentos[0] != "motivos_cobertura" ||
		consultador.argumentos[1] != "rectificacion" ||
		consultador.argumentos[2] != instante {
		t.Fatalf("consulta no exacta: %+v", consultador)
	}
}

func TestConsultaMotivoDecisionCoberturaPostgreSQLFallaCerrado(t *testing.T) {
	instante := time.Date(2026, time.September, 4, 1, 0, 0, 0, time.UTC)
	if _, err := nuevaConsultaMotivoDecisionCoberturaPostgreSQL(
		nil,
		"motivos_cobertura",
		"contratacion_temporal",
	); !errors.Is(err, cobertura.ErrConfiguracionResolutorMotivoDecisionCobertura) {
		t.Fatalf("consultador nulo aceptado: %v", err)
	}
	consultador := &consultadorMotivoDecisionCoberturaPrueba{
		fila: filaMotivoDecisionCoberturaPrueba{cardinalidad: 2},
	}
	consulta, err := nuevaConsultaMotivoDecisionCoberturaPostgreSQL(
		consultador,
		"motivos_cobertura",
		"contratacion_temporal",
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := consulta.ConsultarMotivoDecisionCobertura(
		context.Background(),
		"motivos_cobertura",
		"contratacion_temporal",
		"rectificacion",
		instante,
	); !errors.Is(err, cobertura.ErrMotivoDecisionCoberturaNoConfiable) {
		t.Fatalf("cardinalidad ambigua aceptada: %v", err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := consulta.ConsultarMotivoDecisionCobertura(
		ctx,
		"motivos_cobertura",
		"contratacion_temporal",
		"rectificacion",
		instante,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion no propagada: %v", err)
	}
}
