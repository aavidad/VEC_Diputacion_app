package postgres

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type fixtureGobiernoCoberturaO404B struct {
	instante  time.Time
	catalogo  domain.CatalogoViasCobertura
	politica  domain.PoliticaDecisionCobertura
	actuacion cobertura.PublicacionPoliticaActuacionCobertura
}

func nuevoFixtureGobiernoCoberturaO404B(
	t *testing.T,
	instante time.Time,
) fixtureGobiernoCoberturaO404B {
	t.Helper()
	instante = instante.UTC().Truncate(time.Microsecond)
	vigencia := domain.VigenciaCatalogoCobertura{
		Desde: instante.Add(-time.Hour),
		Hasta: instante.Add(2 * time.Hour),
	}
	catalogo, err := domain.PublicarCatalogoViasCobertura(
		domain.BorradorCatalogoViasCobertura{
			Referencia:     "catalogo_cobertura_o404b_01",
			Version:        1,
			PublicadoEn:    instante.Add(-2 * time.Hour),
			Vigencia:       vigencia,
			ProcedenciaRef: "procedencia_catalogo_o404b_01",
			Vias: []domain.DefinicionViaCobertura{{
				Clave: "via_configurable_o404b",
				Orden: 1,
				Comprobaciones: []domain.ComprobacionExigibleCobertura{{
					Clave:       "comprobacion_o404b",
					Orden:       1,
					Obligatoria: true,
					Procedencia: domain.ProcedenciaComprobacionCobertura{
						Clave:               "fuente_o404b",
						DefinicionFuenteRef: "fuente_cobertura_o404b_01",
					},
				}},
			}},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	politica, err := domain.PublicarPoliticaDecisionCobertura(
		domain.BorradorPoliticaDecisionCobertura{
			Referencia:      "politica_decision_o404b_01",
			Version:         1,
			Catalogo:        catalogo.Identidad(),
			OrganizacionRef: "organizacion:dipgra",
			FinalidadClave:  "gestionar_cobertura_temporal",
			FinalidadRef:    "finalidad_cobertura_o404b_01",
			PublicadaEn:     instante.Add(-90 * time.Minute),
			Vigencia:        vigencia,
			ProcedenciaRef:  "procedencia_politica_o404b_01",
			Vias: []domain.ReglaViaDecisionCobertura{{
				ViaClave:  "via_configurable_o404b",
				Prioridad: 1,
				Comprobaciones: []domain.ReglaComprobacionDecisionCobertura{{
					Clave: "comprobacion_o404b",
					ResultadosHabilitantes: []domain.ResultadoComprobacion{
						domain.ComprobacionAfirmativa,
					},
					TratamientoAusencia: domain.AusenciaCoberturaBloquea,
				}},
			}},
		},
		catalogo,
	)
	if err != nil {
		t.Fatal(err)
	}
	motivo := func(clave string) dominiovec.ReferenciaEntradaCatalogo {
		return dominiovec.ReferenciaEntradaCatalogo{
			CatalogoID:           "motivos_autorizacion_cobertura",
			CatalogoVersion:      1,
			CatalogoHuellaSHA256: strings.Repeat("b", 64),
			EntradaClave:         clave,
		}
	}
	actuacion := cobertura.PublicacionPoliticaActuacionCobertura{
		Referencia:                 "politica_actuacion_o404b_01",
		Version:                    1,
		Canon:                      cobertura.CanonHuellaPoliticaActuacionCoberturaV1(),
		OrganizacionRef:            "organizacion:dipgra",
		Accion:                     domain.AccionDecidirCoberturaGobernada,
		Catalogo:                   catalogo.Identidad(),
		Politica:                   politica.Identidad(),
		FinalidadContratacionClave: "gestionar_cobertura_temporal",
		FinalidadContratacionRef:   "finalidad_cobertura_o404b_01",
		FinalidadAutorizacionVEC:   "autorizar_cobertura_temporal",
		UnidadEjecutoraRef:         "unidad_ejecutora_o404b_01",
		FaseDestino:                "fase_cobertura_o404b",
		EstadoDestino:              domain.EstadoEnCurso,
		MotivoAutorizacionDecidir: motivo(
			"motivo_0123456789abcdef0123456789abcdef",
		),
		MotivoAutorizacionRectificar: motivo(
			"motivo_fedcba9876543210fedcba9876543210",
		),
		PublicadaEn: instante.Add(-80 * time.Minute),
		Vigencia:    vigencia,
	}
	actuacion.HuellaSHA256, err =
		cobertura.CalcularHuellaSHA256PoliticaActuacionCobertura(actuacion)
	if err != nil {
		t.Fatal(err)
	}
	return fixtureGobiernoCoberturaO404B{
		instante: instante, catalogo: catalogo,
		politica: politica, actuacion: actuacion,
	}
}

func (f fixtureGobiernoCoberturaO404B) JSON(t *testing.T) (
	string,
	string,
	string,
) {
	t.Helper()
	catalogo, err := json.Marshal(f.catalogo.Publicacion())
	if err != nil {
		t.Fatal(err)
	}
	politica, err := json.Marshal(f.politica.Publicacion())
	if err != nil {
		t.Fatal(err)
	}
	actuacion, err := json.Marshal(f.actuacion)
	if err != nil {
		t.Fatal(err)
	}
	return string(catalogo), string(politica), string(actuacion)
}

type relojGobiernoCoberturaO404BPrueba struct {
	instante time.Time
}

func (r relojGobiernoCoberturaO404BPrueba) AhoraGobiernoOperacionCobertura(
	_ context.Context,
) (time.Time, error) {
	return r.instante, nil
}
