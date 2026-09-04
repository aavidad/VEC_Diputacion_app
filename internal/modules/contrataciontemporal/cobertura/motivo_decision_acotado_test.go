package cobertura

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type consultaMotivoDecisionCoberturaAcotadaPrueba struct {
	motivo     domain.MotivoGobernadoDecisionCobertura
	err        error
	consultas  int
	catalogoID string
	moduloID   string
	clave      domain.ClaveCatalogo
	instante   time.Time
}

func (c *consultaMotivoDecisionCoberturaAcotadaPrueba) ConsultarMotivoDecisionCobertura(
	_ context.Context,
	catalogoID string,
	moduloID string,
	clave domain.ClaveCatalogo,
	instante time.Time,
) (domain.MotivoGobernadoDecisionCobertura, error) {
	c.consultas++
	c.catalogoID = catalogoID
	c.moduloID = moduloID
	c.clave = clave
	c.instante = instante
	return c.motivo, c.err
}

func TestResolutorMotivoDecisionCoberturaAcotadoResuelveEntradaExacta(t *testing.T) {
	instante := time.Date(2026, time.September, 4, 0, 0, 0, 0, time.UTC)
	motivo := motivoDecisionCoberturaAcotadoPrueba()
	consulta := &consultaMotivoDecisionCoberturaAcotadaPrueba{motivo: motivo}
	resolutor, err := NuevoResolutorMotivoDecisionCoberturaAcotado(
		consulta,
		"motivos_cobertura",
		"contratacion_temporal",
	)
	if err != nil {
		t.Fatal(err)
	}
	resolucion, err := resolutor.ResolverClave(
		context.Background(),
		domain.ClaveCatalogo("rectificacion"),
		instante,
	)
	if err != nil {
		t.Fatal(err)
	}
	obtenido, err := resolucion.Motivo()
	if err != nil || obtenido != motivo {
		t.Fatalf("motivo inesperado: %+v %v", obtenido, err)
	}
	resueltaEn, err := resolucion.ResueltaEn()
	if err != nil || !resueltaEn.Equal(instante) {
		t.Fatalf("instante inesperado: %v %v", resueltaEn, err)
	}
	if consulta.consultas != 1 ||
		consulta.catalogoID != "motivos_cobertura" ||
		consulta.moduloID != "contratacion_temporal" ||
		consulta.clave != "rectificacion" ||
		!consulta.instante.Equal(instante) {
		t.Fatalf("consulta no exacta: %+v", consulta)
	}
	if _, err := resolutor.Resolver(context.Background(), motivo, instante); err != nil {
		t.Fatalf("revalidacion exacta fallida: %v", err)
	}
}

func TestResolutorMotivoDecisionCoberturaAcotadoFallaCerrado(t *testing.T) {
	instante := time.Date(2026, time.September, 4, 0, 0, 0, 0, time.UTC)
	motivo := motivoDecisionCoberturaAcotadoPrueba()
	if _, err := NuevoResolutorMotivoDecisionCoberturaAcotado(
		nil,
		"motivos_cobertura",
		"contratacion_temporal",
	); !errors.Is(err, ErrConfiguracionResolutorMotivoDecisionCobertura) {
		t.Fatalf("consulta nula aceptada: %v", err)
	}
	consulta := &consultaMotivoDecisionCoberturaAcotadaPrueba{motivo: motivo}
	resolutor, err := NuevoResolutorMotivoDecisionCoberturaAcotado(
		consulta,
		"motivos_cobertura",
		"contratacion_temporal",
	)
	if err != nil {
		t.Fatal(err)
	}
	consulta.motivo.ReferenciaCatalogo.EntradaClave = "otra"
	if _, err := resolutor.ResolverClave(
		context.Background(),
		"rectificacion",
		instante,
	); !errors.Is(err, ErrMotivoDecisionCoberturaNoConfiable) {
		t.Fatalf("entrada cruzada aceptada: %v", err)
	}
	consulta.motivo = motivo
	consulta.err = errors.New("postgres no disponible")
	if _, err := resolutor.ResolverClave(
		context.Background(),
		"rectificacion",
		instante,
	); !errors.Is(err, ErrMotivoDecisionCoberturaNoConfiable) {
		t.Fatalf("fallo de fuente aceptado: %v", err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := resolutor.ResolverClave(
		ctx,
		"rectificacion",
		instante,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion no propagada: %v", err)
	}
}

func motivoDecisionCoberturaAcotadoPrueba() domain.MotivoGobernadoDecisionCobertura {
	return domain.MotivoGobernadoDecisionCobertura{
		ReferenciaCatalogo: dominiovec.ReferenciaEntradaCatalogo{
			CatalogoID:           "motivos_cobertura",
			CatalogoVersion:      1,
			CatalogoHuellaSHA256: strings.Repeat("a", 64),
			EntradaClave:         "rectificacion",
		},
		ClaveI18n: "contratacion_temporal.cobertura.motivo.rectificacion",
	}
}
