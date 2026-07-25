package domain

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDecisionCoberturaCanonDeterministaYRestaurable(t *testing.T) {
	base, propuesta := prepararDecisionCoberturaGobernada(t)
	primera := adoptarDecisionGobernada(t, base, propuesta)
	segunda := adoptarDecisionGobernada(t, base, propuesta)
	a := primera.DecisionesCobertura[0]
	b := segunda.DecisionesCobertura[0]
	if a.Referencia != b.Referencia || a.HuellaSHA256 != b.HuellaSHA256 {
		t.Fatalf("canon no determinista: %s != %s", a.Referencia, b.Referencia)
	}
	restaurada, err := RestaurarDecisionCoberturaGobernada(a)
	if err != nil || restaurada.Publicacion() != a {
		t.Fatalf("restauración: %#v, %v", restaurada.Publicacion(), err)
	}
	materialA, err := materialCanonicoDecisionCoberturaV1(a)
	if err != nil {
		t.Fatal(err)
	}
	materialB, err := materialCanonicoDecisionCoberturaV1(b)
	if err != nil || !reflect.DeepEqual(materialA, materialB) {
		t.Fatal("material binario variable")
	}
}

func TestDecisionCoberturaRechazaAdulteracionDeCadaLigadura(t *testing.T) {
	base, propuesta := prepararDecisionCoberturaGobernada(t)
	decidido := adoptarDecisionGobernada(t, base, propuesta)
	casos := []struct {
		nombre    string
		adulterar func(*PublicacionDecisionCoberturaGobernada)
	}{
		{"propuesta", func(p *PublicacionDecisionCoberturaGobernada) {
			p.PropuestaHuellaSHA256 = cadena64("e")
		}},
		{"preparacion", func(p *PublicacionDecisionCoberturaGobernada) {
			p.PreparacionEvidenciasRef = "preparacion_adulterada_01"
		}},
		{"analisis", func(p *PublicacionDecisionCoberturaGobernada) {
			p.AnalisisRef = "analisis_adulterado_01"
		}},
		{"catalogo", func(p *PublicacionDecisionCoberturaGobernada) {
			p.Catalogo.Version++
		}},
		{"politica", func(p *PublicacionDecisionCoberturaGobernada) {
			p.Politica.HuellaSHA256 = cadena64("d")
		}},
		{"via", func(p *PublicacionDecisionCoberturaGobernada) {
			p.ViaElegida = "via_adulterada"
		}},
		{"actor", func(p *PublicacionDecisionCoberturaGobernada) {
			p.ActorRef = "actor_adulterado_01"
		}},
		{"actuacion", func(p *PublicacionDecisionCoberturaGobernada) {
			p.Actuacion.ReciboRef = "recibo_adulterado_01"
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			adulterada := decidido.DecisionesCobertura[0]
			caso.adulterar(&adulterada)
			if _, err := RestaurarDecisionCoberturaGobernada(adulterada); !errors.Is(err, ErrDatoInvalido) {
				t.Fatalf("restauró contenido adulterado: %v", err)
			}
			expediente := decidido.Clonar()
			expediente.DecisionesCobertura[0] = adulterada
			expediente.ViaCobertura.DecisionGobernada = &adulterada
			if !errors.Is(expediente.Validar(), ErrExpedienteInvalido) {
				t.Fatal("rehidrató expediente adulterado")
			}
		})
	}
}

func TestDecisionCoberturaClonaHistoriaYProyeccion(t *testing.T) {
	base, propuesta := prepararDecisionCoberturaGobernada(t)
	original := adoptarDecisionGobernada(t, base, propuesta)
	clon := original.Clonar()
	clon.DecisionesCobertura[0].ActorRef = "actor_clon_alterado_01"
	clon.ViaCobertura.DecisionGobernada.ActorRef = "actor_proyeccion_alterado_01"
	if original.DecisionesCobertura[0].ActorRef ==
		clon.DecisionesCobertura[0].ActorRef ||
		original.ViaCobertura.DecisionGobernada.ActorRef ==
			clon.ViaCobertura.DecisionGobernada.ActorRef {
		t.Fatal("clon comparte historia o proyección mutable")
	}
}

func TestDecisionCoberturaJSONRehidrataSinDetalleLibreNiPII(t *testing.T) {
	base, propuesta := prepararDecisionCoberturaGobernada(t)
	decidido := adoptarDecisionGobernada(t, base, propuesta)
	contenido, err := json.Marshal(decidido.DecisionesCobertura[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, prohibido := range []string{
		"Nombre Apellidos", "detalle", "motivacion",
	} {
		if strings.Contains(string(contenido), prohibido) {
			t.Fatalf("contenido no minimizado: %s", prohibido)
		}
	}
	var publicacion PublicacionDecisionCoberturaGobernada
	if err := json.Unmarshal(contenido, &publicacion); err != nil {
		t.Fatal(err)
	}
	if _, err := RestaurarDecisionCoberturaGobernada(publicacion); err != nil {
		t.Fatalf("rehidratación JSON: %v", err)
	}

	contenidoExpediente, err := json.Marshal(decidido)
	if err != nil {
		t.Fatal(err)
	}
	var restaurado Expediente
	if err := json.Unmarshal(contenidoExpediente, &restaurado); err != nil {
		t.Fatal(err)
	}
	if err := restaurado.Validar(); err != nil {
		t.Fatalf("rehidratación JSON: %v", err)
	}
}

func TestHistoriaDecisionCoberturaAcotadaAntesDeRecorrer(t *testing.T) {
	base, propuesta := prepararDecisionCoberturaGobernada(t)
	decidido := adoptarDecisionGobernada(t, base, propuesta)
	decidido.DecisionesCobertura = make(
		[]PublicacionDecisionCoberturaGobernada,
		maximoDecisionesCoberturaGobernadas+1,
	)
	if !errors.Is(decidido.Validar(), ErrExpedienteInvalido) {
		t.Fatal("aceptó historia por encima del límite técnico")
	}
}

func TestRectificacionDecisionExigeInstantePosterior(t *testing.T) {
	base, propuesta := prepararDecisionCoberturaGobernada(t)
	decidido := adoptarDecisionGobernada(t, base, propuesta)
	nueva := propuestaDecisionParaExpediente(
		t,
		decidido,
		decidido.ActualizadoEn,
	)
	datos := datosRectificacion(decidido)
	datos.ViaElegida = "via_alternativa_configurable"
	acto := actuacionDecision(
		string(AccionRectificarCoberturaGobernada),
		"actor_rrhh_rectificador_02",
		decidido.FaseActual,
		decidido.ActualizadoEn,
	)
	if _, err := decidido.RectificarDecisionCoberturaGobernada(
		decidido.Version,
		datos,
		nueva,
		acto,
	); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("aceptó instante no posterior: %v", err)
	}

	acto.RealizadaEn = acto.RealizadaEn.Add(time.Nanosecond)
	if _, err := decidido.RectificarDecisionCoberturaGobernada(
		decidido.Version,
		datos,
		nueva,
		acto,
	); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("aceptó instante no canónico: %v", err)
	}
}

func TestDecisionCoberturaExigeAccionTecnicaExactaPorTipo(t *testing.T) {
	base, propuesta := prepararDecisionCoberturaGobernada(t)
	acto := actuacionDecision(
		"cobertura.accion_aproximada",
		"actor_rrhh_decisor_01",
		"asignacion_unidad",
		propuesta.Publicacion().GeneradaEn.Add(time.Minute),
	)
	if _, err := base.RegistrarDecisionCoberturaGobernada(
		base.Version,
		DatosAdoptarDecisionCobertura{
			PerfilRef:  "perfil_rrhh_decisor_01",
			ViaElegida: propuesta.ViaPropuesta(),
		},
		propuesta,
		acto,
	); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("acción inicial aproximada: %v", err)
	}
	acto.AccionClave = AccionDecidirCoberturaGobernada
	acto.Observaciones = "Nombre y documento que no deben entrar en decisión."
	if _, err := base.RegistrarDecisionCoberturaGobernada(
		base.Version,
		DatosAdoptarDecisionCobertura{
			PerfilRef:  "perfil_rrhh_decisor_01",
			ViaElegida: propuesta.ViaPropuesta(),
		},
		propuesta,
		acto,
	); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("acción con texto libre: %v", err)
	}

	decidido := adoptarDecisionGobernada(t, base, propuesta)
	nueva := propuestaDecisionParaExpediente(
		t,
		decidido,
		decidido.ActualizadoEn.Add(time.Minute),
	)
	datos := datosRectificacion(decidido)
	datos.ViaElegida = "via_alternativa_configurable"
	acto = actuacionDecision(
		string(AccionDecidirCoberturaGobernada),
		"actor_rrhh_rectificador_02",
		decidido.FaseActual,
		decidido.ActualizadoEn.Add(2*time.Minute),
	)
	if _, err := decidido.RectificarDecisionCoberturaGobernada(
		decidido.Version,
		datos,
		nueva,
		acto,
	); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("acción inicial reutilizada al rectificar: %v", err)
	}
}

func TestDecisionCoberturaRechazaHistoriaReselladaConRamaOSalto(
	t *testing.T,
) {
	tres := expedienteConTresDecisionesCobertura(t)
	casos := []struct {
		nombre    string
		adulterar func(*PublicacionDecisionCoberturaGobernada, Expediente)
	}{
		{
			nombre: "salto de predecesora",
			adulterar: func(
				p *PublicacionDecisionCoberturaGobernada,
				e Expediente,
			) {
				p.PredecesoraRef = e.DecisionesCobertura[0].Referencia
				p.PredecesoraHuellaSHA256 =
					e.DecisionesCobertura[0].HuellaSHA256
			},
		},
		{
			nombre: "mismo actor consecutivo",
			adulterar: func(
				p *PublicacionDecisionCoberturaGobernada,
				e Expediente,
			) {
				p.ActorRef = e.DecisionesCobertura[1].ActorRef
				p.Actuacion.ActorRef = p.ActorRef
			},
		},
		{
			nombre: "instante no posterior",
			adulterar: func(
				p *PublicacionDecisionCoberturaGobernada,
				e Expediente,
			) {
				p.DecididaEn = e.DecisionesCobertura[1].DecididaEn
				p.Actuacion.RealizadaEn = p.DecididaEn
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			adulterado := tres.Clonar()
			publicacion := adulterado.DecisionesCobertura[2]
			caso.adulterar(&publicacion, adulterado)
			resellarDecisionCoberturaPrueba(t, &publicacion)
			adulterado.DecisionesCobertura[2] = publicacion
			adulterado.ViaCobertura.DecisionGobernada = &publicacion
			if !errors.Is(adulterado.Validar(), ErrExpedienteInvalido) {
				t.Fatal("aceptó cadena resellada no lineal")
			}
		})
	}
}

func TestDecisionCoberturaNoMezclaLegadoEHistoriaGobernada(t *testing.T) {
	base, propuesta := prepararDecisionCoberturaGobernada(t)
	decidido := adoptarDecisionGobernada(t, base, propuesta)
	mezclado := decidido.Clonar()
	mezclado.ViaCobertura.DecisionGobernada = nil
	mezclado.ViaCobertura.ProcedimientoRef = "procedimiento_legado_01"
	mezclado.ViaCobertura.Comprobaciones = decisionValida().Comprobaciones
	mezclado.ViaCobertura.Motivacion = "Proyección histórica incompatible."
	if !errors.Is(mezclado.Validar(), ErrExpedienteInvalido) {
		t.Fatal("aceptó historia gobernada con proyección legacy")
	}
}

func expedienteConTresDecisionesCobertura(t *testing.T) Expediente {
	t.Helper()
	base, propuesta := prepararDecisionCoberturaGobernada(t)
	actual := adoptarDecisionGobernada(t, base, propuesta)
	for indice, via := range []ClaveCatalogo{
		"via_alternativa_configurable",
		"via_futura_configurable",
	} {
		nueva := propuestaDecisionParaExpediente(
			t,
			actual,
			actual.ActualizadoEn.Add(time.Minute),
		)
		datos := datosRectificacion(actual)
		datos.ViaElegida = via
		acto := actuacionDecision(
			string(AccionRectificarCoberturaGobernada),
			"actor_rrhh_rectificador_0"+string(rune('2'+indice)),
			actual.FaseActual,
			actual.ActualizadoEn.Add(2*time.Minute),
		)
		var err error
		actual, err = actual.RectificarDecisionCoberturaGobernada(
			actual.Version,
			datos,
			nueva,
			acto,
		)
		if err != nil {
			t.Fatalf("rectificación %d: %v", indice+1, err)
		}
	}
	return actual
}

func resellarDecisionCoberturaPrueba(
	t *testing.T,
	p *PublicacionDecisionCoberturaGobernada,
) {
	t.Helper()
	huella, err := calcularHuellaDecisionCobertura(*p)
	if err != nil {
		t.Fatalf("resellar: %v", err)
	}
	p.HuellaSHA256 = huella
	p.Referencia = referenciaDecisionCobertura(huella)
}
