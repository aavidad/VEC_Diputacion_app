package domain

import (
	"strings"
	"testing"
	"time"
)

var instantePoliticaCoberturaPrueba = time.Date(
	2026, 7, 25, 8, 0, 0, 0, time.UTC,
)

func TestPoliticaDecisionCoberturaGobiernaViasFuturasSinListaCompilada(
	t *testing.T,
) {
	catalogo := catalogoDecisionCoberturaPrueba(t)
	borrador := borradorPoliticaDecisionCoberturaPrueba(catalogo)
	politica, err := PublicarPoliticaDecisionCobertura(borrador, catalogo)
	if err != nil {
		t.Fatalf("publicar política: %v", err)
	}
	if politica.ValidarPara(
		catalogo,
		"organizacion_diputacion_granada",
		"gestionar_cobertura_temporal",
		"finalidad:contratacion-temporal:cobertura",
		instantePoliticaCoberturaPrueba.Add(time.Minute),
	) != nil ||
		politica.Identidad().Validar() != nil ||
		len(politica.Vias()) != 2 ||
		politica.Vias()[0].ViaClave != "via_futura_configurable" {
		t.Fatalf("política publicada inesperada: %#v", politica.Publicacion())
	}
	restaurada, err := RestaurarPoliticaDecisionCobertura(
		politica.Publicacion(),
		catalogo,
	)
	if err != nil ||
		!restaurada.Identidad().coincide(politica.Identidad()) {
		t.Fatalf("restaurar política: %#v, %v", restaurada, err)
	}
}

func TestPoliticaDecisionCoberturaRechazaReglasIncompletasODesconocidas(
	t *testing.T,
) {
	casos := []struct {
		nombre  string
		alterar func(*BorradorPoliticaDecisionCobertura)
	}{
		{
			nombre: "catalogo_mutado",
			alterar: func(b *BorradorPoliticaDecisionCobertura) {
				b.Catalogo.HuellaSHA256 = strings.Repeat("f", 64)
			},
		},
		{
			nombre: "via_ausente",
			alterar: func(b *BorradorPoliticaDecisionCobertura) {
				b.Vias = b.Vias[:1]
			},
		},
		{
			nombre: "via_desconocida",
			alterar: func(b *BorradorPoliticaDecisionCobertura) {
				b.Vias[0].ViaClave = "via_que_no_esta_publicada"
			},
		},
		{
			nombre: "prioridad_duplicada",
			alterar: func(b *BorradorPoliticaDecisionCobertura) {
				b.Vias[1].Prioridad = b.Vias[0].Prioridad
			},
		},
		{
			nombre: "comprobacion_ausente",
			alterar: func(b *BorradorPoliticaDecisionCobertura) {
				b.Vias[0].Comprobaciones =
					b.Vias[0].Comprobaciones[:1]
			},
		},
		{
			nombre: "comprobacion_desconocida",
			alterar: func(b *BorradorPoliticaDecisionCobertura) {
				b.Vias[0].Comprobaciones[0].Clave =
					"comprobacion_no_publicada"
			},
		},
		{
			nombre: "resultado_desconocido",
			alterar: func(b *BorradorPoliticaDecisionCobertura) {
				b.Vias[0].Comprobaciones[0].
					ResultadosHabilitantes[0] = "resultado_desconocido"
			},
		},
		{
			nombre: "resultado_duplicado",
			alterar: func(b *BorradorPoliticaDecisionCobertura) {
				b.Vias[0].Comprobaciones[0].
					ResultadosHabilitantes =
					[]ResultadoComprobacion{
						ComprobacionAfirmativa,
						ComprobacionAfirmativa,
					}
			},
		},
		{
			nombre: "ausencia_ambigua",
			alterar: func(b *BorradorPoliticaDecisionCobertura) {
				b.Vias[0].Comprobaciones[0].
					TratamientoAusencia = "interpretar_como_exito"
			},
		},
		{
			nombre: "politica_anterior_al_catalogo",
			alterar: func(b *BorradorPoliticaDecisionCobertura) {
				b.PublicadaEn = instantePoliticaCoberturaPrueba.Add(-2 * time.Hour)
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			catalogo := catalogoDecisionCoberturaPrueba(t)
			borrador := borradorPoliticaDecisionCoberturaPrueba(catalogo)
			caso.alterar(&borrador)
			if _, err := PublicarPoliticaDecisionCobertura(
				borrador,
				catalogo,
			); err == nil {
				t.Fatal("se publicó una política no gobernada")
			}
		})
	}
}

func TestPoliticaDecisionCoberturaEsDeterministaEInmutable(
	t *testing.T,
) {
	catalogo := catalogoDecisionCoberturaPrueba(t)
	borrador := borradorPoliticaDecisionCoberturaPrueba(catalogo)
	invertido := borrador
	invertido.Vias = []ReglaViaDecisionCobertura{
		borrador.Vias[1].clonar(),
		borrador.Vias[0].clonar(),
	}
	for indice := range invertido.Vias {
		reglas := invertido.Vias[indice].Comprobaciones
		for izquierda, derecha := 0, len(reglas)-1; izquierda < derecha; izquierda, derecha = izquierda+1, derecha-1 {
			reglas[izquierda], reglas[derecha] =
				reglas[derecha], reglas[izquierda]
		}
	}
	primera, errPrimera := PublicarPoliticaDecisionCobertura(
		borrador,
		catalogo,
	)
	segunda, errSegunda := PublicarPoliticaDecisionCobertura(
		invertido,
		catalogo,
	)
	if errPrimera != nil || errSegunda != nil ||
		!primera.Identidad().coincide(segunda.Identidad()) {
		t.Fatalf(
			"normalización no determinista: %v, %v",
			errPrimera,
			errSegunda,
		)
	}
	publicacion := primera.Publicacion()
	publicacion.Vias[0].Comprobaciones[0].
		ResultadosHabilitantes[0] = ComprobacionNegativa
	if primera.ValidarPara(
		catalogo,
		"organizacion_diputacion_granada",
		"gestionar_cobertura_temporal",
		"finalidad:contratacion-temporal:cobertura",
		instantePoliticaCoberturaPrueba.Add(time.Minute),
	) != nil ||
		primera.Vias()[0].Comprobaciones[0].
			ResultadosHabilitantes[0] != ComprobacionAfirmativa {
		t.Fatal("una copia de salida alteró la política")
	}
	if _, err := RestaurarPoliticaDecisionCobertura(
		publicacion,
		catalogo,
	); err == nil {
		t.Fatal("se restauró contenido alterado con la huella anterior")
	}
}

func TestPoliticaDecisionCoberturaLigaAmbitoFinalidadYVigenciaExclusiva(
	t *testing.T,
) {
	catalogo := catalogoDecisionCoberturaPrueba(t)
	politica, err := PublicarPoliticaDecisionCobertura(
		borradorPoliticaDecisionCoberturaPrueba(catalogo),
		catalogo,
	)
	if err != nil {
		t.Fatal(err)
	}
	vigencia := politica.Vigencia()
	casos := []struct {
		nombre       string
		organizacion string
		finalidad    ClaveCatalogo
		finalidadRef string
		instante     time.Time
	}{
		{
			nombre:       "otra_organizacion",
			organizacion: "organizacion_distinta_01",
			finalidad:    "gestionar_cobertura_temporal",
			finalidadRef: "finalidad:contratacion-temporal:cobertura",
			instante:     vigencia.Desde,
		},
		{
			nombre:       "otra_finalidad",
			organizacion: "organizacion_diputacion_granada",
			finalidad:    "finalidad_distinta",
			finalidadRef: "finalidad:contratacion-temporal:cobertura",
			instante:     vigencia.Desde,
		},
		{
			nombre:       "antes_de_vigencia",
			organizacion: "organizacion_diputacion_granada",
			finalidad:    "gestionar_cobertura_temporal",
			finalidadRef: "finalidad:contratacion-temporal:cobertura",
			instante:     vigencia.Desde.Add(-time.Microsecond),
		},
		{
			nombre:       "retirada_o_caducada_en_limite_exclusivo",
			organizacion: "organizacion_diputacion_granada",
			finalidad:    "gestionar_cobertura_temporal",
			finalidadRef: "finalidad:contratacion-temporal:cobertura",
			instante:     vigencia.Hasta,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if politica.ValidarPara(
				catalogo,
				caso.organizacion,
				caso.finalidad,
				caso.finalidadRef,
				caso.instante,
			) == nil {
				t.Fatal("la política se aplicó fuera de su ámbito")
			}
		})
	}
}

func TestPoliticaDecisionCoberturaNoAdmiteNoConstaComoHabilitante(
	t *testing.T,
) {
	catalogo := catalogoDecisionCoberturaPrueba(t)
	borrador := borradorPoliticaDecisionCoberturaPrueba(catalogo)
	borrador.Vias[0].Comprobaciones[0].ResultadosHabilitantes =
		[]ResultadoComprobacion{ComprobacionNoConsta}
	if _, err := PublicarPoliticaDecisionCobertura(
		borrador,
		catalogo,
	); err == nil {
		t.Fatal("no_consta habilitó una vía")
	}
}

func catalogoDecisionCoberturaPrueba(
	t *testing.T,
) CatalogoViasCobertura {
	t.Helper()
	compartida := ComprobacionExigibleCobertura{
		Clave: "hecho_compartido", Orden: 1, Obligatoria: true,
		Procedencia: ProcedenciaComprobacionCobertura{
			Clave:               "fuente_futura",
			DefinicionFuenteRef: "definicion_fuente_futura_01",
		},
	}
	catalogo, err := PublicarCatalogoViasCobertura(
		BorradorCatalogoViasCobertura{
			Referencia:  "catalogo_vias_configurables_01",
			Version:     17,
			PublicadoEn: instantePoliticaCoberturaPrueba.Add(-time.Hour),
			Vigencia: VigenciaCatalogoCobertura{
				Desde: instantePoliticaCoberturaPrueba.Add(-time.Hour),
				Hasta: instantePoliticaCoberturaPrueba.Add(24 * time.Hour),
			},
			ProcedenciaRef: "expediente_gobierno_catalogo_01",
			Vias: []DefinicionViaCobertura{
				{
					Clave: "via_alternativa_configurable",
					Orden: 10,
					Comprobaciones: []ComprobacionExigibleCobertura{
						compartida,
						comprobacionCatalogoDecisionPrueba(
							"hecho_alternativo", 2,
						),
					},
				},
				{
					Clave: "via_futura_configurable",
					Orden: 20,
					Comprobaciones: []ComprobacionExigibleCobertura{
						compartida,
						comprobacionCatalogoDecisionPrueba(
							"hecho_futuro", 2,
						),
					},
				},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return catalogo
}

func comprobacionCatalogoDecisionPrueba(
	clave ClaveCatalogo,
	orden uint16,
) ComprobacionExigibleCobertura {
	return ComprobacionExigibleCobertura{
		Clave:       clave,
		Orden:       orden,
		Obligatoria: true,
		Procedencia: ProcedenciaComprobacionCobertura{
			Clave:               clave,
			DefinicionFuenteRef: "definicion:" + string(clave),
		},
	}
}

func borradorPoliticaDecisionCoberturaPrueba(
	catalogo CatalogoViasCobertura,
) BorradorPoliticaDecisionCobertura {
	regla := func(
		clave ClaveCatalogo,
		ausencia TratamientoAusenciaCobertura,
	) ReglaComprobacionDecisionCobertura {
		return ReglaComprobacionDecisionCobertura{
			Clave: clave,
			ResultadosHabilitantes: []ResultadoComprobacion{
				ComprobacionAfirmativa,
			},
			TratamientoAusencia: ausencia,
		}
	}
	return BorradorPoliticaDecisionCobertura{
		Referencia:      "politica_decision_configurable_01",
		Version:         4,
		Catalogo:        catalogo.Identidad(),
		OrganizacionRef: "organizacion_diputacion_granada",
		FinalidadClave:  "gestionar_cobertura_temporal",
		FinalidadRef:    "finalidad:contratacion-temporal:cobertura",
		PublicadaEn:     instantePoliticaCoberturaPrueba,
		Vigencia: VigenciaCatalogoCobertura{
			Desde: instantePoliticaCoberturaPrueba,
			Hasta: instantePoliticaCoberturaPrueba.Add(12 * time.Hour),
		},
		ProcedenciaRef: "resolucion_gobierno_politica_01",
		Vias: []ReglaViaDecisionCobertura{
			{
				ViaClave:  "via_alternativa_configurable",
				Prioridad: 20,
				Comprobaciones: []ReglaComprobacionDecisionCobertura{
					regla("hecho_compartido", AusenciaCoberturaBloquea),
					regla("hecho_alternativo", AusenciaCoberturaBloquea),
				},
			},
			{
				ViaClave:  "via_futura_configurable",
				Prioridad: 10,
				Comprobaciones: []ReglaComprobacionDecisionCobertura{
					regla("hecho_compartido", AusenciaCoberturaBloquea),
					regla("hecho_futuro", AusenciaCoberturaAdmitida),
				},
			},
		},
	}
}
