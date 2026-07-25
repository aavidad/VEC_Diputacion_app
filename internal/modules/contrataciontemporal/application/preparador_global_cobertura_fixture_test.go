package application

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	analisisPreparacionGlobalRef    = "analisis_durable_preparacion_global_01"
	huellaAnalisisPreparacionGlobal = "abababababababababababababababab" +
		"abababababababababababababababab"
)

type generadorReferenciasPreparacionGlobalPrueba struct {
	mu             sync.Mutex
	siguiente      int
	repetirDesde   int
	referenciaFija string
}

func (g *generadorReferenciasPreparacionGlobalPrueba) NuevaReferenciaComprobacionCobertura(
	ctx context.Context,
) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.siguiente++
	if g.repetirDesde > 0 && g.siguiente >= g.repetirDesde {
		return g.referenciaFija, nil
	}
	referencia := fmt.Sprintf(
		"peticion_preparacion_global_%03d_012345",
		g.siguiente,
	)
	if g.referenciaFija == "" {
		g.referenciaFija = referencia
	}
	return referencia, nil
}

func (g *generadorReferenciasPreparacionGlobalPrueba) llamadas() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.siguiente
}

type escenarioPreparacionGlobalPrueba struct {
	entorno    *entornoCoberturaAplicacionPrueba
	catalogo   domain.CatalogoViasCobertura
	politica   domain.PoliticaDecisionCobertura
	generador  *generadorReferenciasPreparacionGlobalPrueba
	preparador *PreparadorGlobalCobertura
	datos      datosPreparacionGlobalCobertura
	antes      func(context.Context, ports.SolicitudConsultarCobertura) error
}

func nuevoEscenarioPreparacionGlobalPrueba(
	t *testing.T,
	vias []domain.DefinicionViaCobertura,
	concurrencia int,
	tiempoMaximo time.Duration,
) *escenarioPreparacionGlobalPrueba {
	t.Helper()
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	catalogo := publicarCatalogoGlobalC1(
		t,
		entorno.inicio,
		"catalogo_preparacion_global_cobertura_01",
		1,
		vias,
	)
	politica := politicaCoberturaPrueba(t, catalogo, entorno.inicio)
	entorno.catalogo = catalogo
	entorno.publicador.publicar = func(
		_ context.Context,
		_ ports.SolicitudConsultarCobertura,
	) (ports.ConfirmacionPublicacionCobertura, error) {
		return ports.NuevaConfirmacionPublicacionCobertura(
			entorno.publicador.identidad.AutoridadRef(),
			catalogo.Publicacion(),
			entorno.reloj.Ahora(),
		)
	}
	escenario := &escenarioPreparacionGlobalPrueba{
		entorno: entorno, catalogo: catalogo, politica: politica,
		generador: &generadorReferenciasPreparacionGlobalPrueba{},
	}
	entorno.fuente.consultar = func(
		ctx context.Context,
		solicitud ports.SolicitudConsultarCobertura,
	) (ports.ResultadoConsultaCobertura, error) {
		if escenario.antes != nil {
			if err := escenario.antes(ctx, solicitud); err != nil {
				return ports.ResultadoConsultaCobertura{}, err
			}
		}
		entorno.reloj.fijar(solicitud.SolicitadaEn.Add(2 * time.Second))
		return resultadoCoberturaAplicacionPrueba(
			t,
			solicitud,
			func(datos *ports.DatosResultadoConsultaCobertura) {
				datos.Comprobacion.ReciboRef =
					"recibo_" + solicitud.PeticionRef
			},
		), nil
	}
	consultas, err := NuevoPreparadorConsultaCobertura(
		entorno.fuente,
		entorno.verificador,
		entorno.publicador,
		entorno.autenticador,
		entorno.reloj,
		500*time.Millisecond,
	)
	if err != nil {
		t.Fatal(err)
	}
	escenario.preparador, err = NuevoPreparadorGlobalCobertura(
		consultas,
		escenario.generador,
		entorno.reloj,
		concurrencia,
		tiempoMaximo,
	)
	if err != nil {
		t.Fatal(err)
	}
	escenario.datos, err = nuevosDatosPreparacionGlobalCobertura(
		analisisPreparacionGlobalRef,
		huellaAnalisisPreparacionGlobal,
		catalogo,
		politica,
		organizacionCoberturaPrueba,
		"expediente_preparacion_global_012345",
		3,
		"categoria_trabajo_social",
		domain.PeriodoPrevisto{
			Inicio: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			Fin:    time.Date(2027, 3, 31, 0, 0, 0, 0, time.UTC),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return escenario
}

func viasPreparacionGlobalPrueba(
	cantidadVias int,
	comprobacionesPorVia int,
) []domain.DefinicionViaCobertura {
	vias := make([]domain.DefinicionViaCobertura, cantidadVias)
	for viaIndice := range vias {
		comprobaciones := make(
			[]domain.ComprobacionExigibleCobertura,
			comprobacionesPorVia,
		)
		for comprobacionIndice := range comprobaciones {
			comprobaciones[comprobacionIndice] =
				domain.ComprobacionExigibleCobertura{
					Clave: domain.ClaveCatalogo(fmt.Sprintf(
						"comprobacion_%02d_%02d",
						viaIndice+1,
						comprobacionIndice+1,
					)),
					Orden:       uint16(comprobacionIndice + 1),
					Obligatoria: true,
					Procedencia: domain.ProcedenciaComprobacionCobertura{
						Clave:               "fuente_bolsa",
						DefinicionFuenteRef: "fuente_definicion_bolsa_v3",
					},
				}
		}
		vias[viaIndice] = domain.DefinicionViaCobertura{
			Clave: domain.ClaveCatalogo(
				fmt.Sprintf("via_global_%02d", viaIndice+1),
			),
			Orden:          uint16(viaIndice + 1),
			Comprobaciones: comprobaciones,
		}
	}
	return vias
}

func exigirCeroConsumoPreparacionGlobal(
	t *testing.T,
	escenario *escenarioPreparacionGlobalPrueba,
) {
	t.Helper()
	escenario.entorno.consumidor.mu.Lock()
	defer escenario.entorno.consumidor.mu.Unlock()
	if len(escenario.entorno.consumidor.ordenes) != 0 ||
		len(escenario.entorno.consumidor.registros) != 0 {
		t.Fatal("el preparador global produjo consumo C1")
	}
}
