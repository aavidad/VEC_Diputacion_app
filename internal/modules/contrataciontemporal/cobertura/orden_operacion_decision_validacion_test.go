package cobertura

import (
	"bytes"
	"encoding"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestVentanaPreparacionOrdenDecisionExigeCausalidadPosteriorAReserva(
	t *testing.T,
) {
	reservadaEn := time.Date(2026, 7, 25, 10, 0, 0, 0, time.UTC)
	preparadaC1En := reservadaEn.Add(time.Microsecond)
	generadaEn := preparadaC1En.Add(time.Microsecond)
	gobiernoEn := reservadaEn.Add(time.Microsecond)
	ahora := generadaEn.Add(time.Microsecond)
	reserva := DatosReservaPropietariaOperacionDecisionCobertura{
		ObservadaEnDB:  reservadaEn,
		PropiedadHasta: reservadaEn.Add(5 * time.Second),
	}
	gobierno := DatosGobiernoOperacionCobertura{
		EvaluadaEn:  gobiernoEn,
		ValidaHasta: reservadaEn.Add(5 * time.Second),
	}
	if !ventanaPreparacionOrdenOperacionDecisionCoberturaValida(
		ahora,
		generadaEn,
		preparadaC1En,
		reserva,
		gobierno,
		reservadaEn.Add(5*time.Second),
		reservadaEn.Add(5*time.Second),
	) {
		t.Fatal("se rechazó una cadena causal posterior a la reserva")
	}
	casos := []struct {
		nombre      string
		generada    time.Time
		preparadaC1 time.Time
		evaluada    time.Time
	}{
		{
			nombre:      "C1 anterior",
			generada:    generadaEn,
			preparadaC1: reservadaEn.Add(-time.Microsecond),
			evaluada:    gobiernoEn,
		},
		{
			nombre:      "propuesta anterior",
			generada:    reservadaEn.Add(-time.Microsecond),
			preparadaC1: preparadaC1En,
			evaluada:    gobiernoEn,
		},
		{
			nombre:      "gobierno anterior",
			generada:    generadaEn,
			preparadaC1: preparadaC1En,
			evaluada:    reservadaEn.Add(-time.Microsecond),
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			otroGobierno := gobierno
			otroGobierno.EvaluadaEn = caso.evaluada
			if ventanaPreparacionOrdenOperacionDecisionCoberturaValida(
				ahora,
				caso.generada,
				caso.preparadaC1,
				reserva,
				otroGobierno,
				reservadaEn.Add(5*time.Second),
				reservadaEn.Add(5*time.Second),
			) {
				t.Fatal("se aceptó autoridad anterior a la reserva")
			}
		})
	}
}

func TestVentanaPreparacionOrdenDecisionExcluyeCadaLimiteSuperior(
	t *testing.T,
) {
	inicio := time.Date(2026, 7, 25, 10, 0, 0, 0, time.UTC)
	ahora := inicio.Add(time.Second)
	baseReserva := DatosReservaPropietariaOperacionDecisionCobertura{
		ObservadaEnDB: inicio, PropiedadHasta: ahora.Add(time.Second),
	}
	baseGobierno := DatosGobiernoOperacionCobertura{
		EvaluadaEn: inicio, ValidaHasta: ahora.Add(time.Second),
	}
	casos := []struct {
		nombre           string
		reservaHasta     time.Time
		gobiernoHasta    time.Time
		propuestaHasta   time.Time
		preparacionHasta time.Time
	}{
		{"propiedad", ahora, ahora.Add(time.Second), ahora.Add(time.Second), ahora.Add(time.Second)},
		{"gobierno", ahora.Add(time.Second), ahora, ahora.Add(time.Second), ahora.Add(time.Second)},
		{"propuesta", ahora.Add(time.Second), ahora.Add(time.Second), ahora, ahora.Add(time.Second)},
		{"C1", ahora.Add(time.Second), ahora.Add(time.Second), ahora.Add(time.Second), ahora},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			reserva := baseReserva
			reserva.PropiedadHasta = caso.reservaHasta
			gobierno := baseGobierno
			gobierno.ValidaHasta = caso.gobiernoHasta
			if ventanaPreparacionOrdenOperacionDecisionCoberturaValida(
				ahora,
				inicio.Add(2*time.Microsecond),
				inicio.Add(time.Microsecond),
				reserva,
				gobierno,
				caso.propuestaHasta,
				caso.preparacionHasta,
			) {
				t.Fatal("se aceptó el extremo superior exclusivo")
			}
		})
	}
}

func TestMotivoFuncionalOrdenDecisionExigeResolucionCausal(t *testing.T) {
	generadaEn := time.Date(2026, 7, 25, 10, 0, 0, 2_000, time.UTC)
	limite := generadaEn.Add(time.Second)
	motivo := motivoOperacionDecisionCoberturaPrueba("8")
	identidad := identidadOperacionDecisionCoberturaPrueba()
	identidad.motivo = motivo
	casos := []struct {
		nombre     string
		resueltaEn time.Time
		valida     bool
	}{
		{"exacta", generadaEn, true},
		{"posterior", generadaEn.Add(time.Microsecond), true},
		{"un microsegundo anterior", generadaEn.Add(-time.Microsecond), false},
		{"posterior al limite", limite.Add(time.Microsecond), false},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			_, err := motivoFuncionalOperacionDecisionCobertura(
				identidad,
				ResolucionMotivoDecisionCobertura{
					motivo: motivo, resueltaEn: caso.resueltaEn,
				},
				generadaEn,
				limite,
			)
			if (err == nil) != caso.valida {
				t.Fatalf("resultado causal inesperado: %v", err)
			}
		})
	}
}

func TestRamaDenegadaOrdenDecisionNoPuedeConservarEfectosC2NiC1(
	t *testing.T,
) {
	tipo := reflect.TypeOf(datosOrdenDenegadaOperacionDecisionCobertura{})
	prohibidos := []reflect.Type{
		reflect.TypeOf(PreparacionConjuntosViasCobertura{}),
		reflect.TypeOf(domain.PropuestaDecisionCobertura{}),
		reflect.TypeOf(domain.Expediente{}),
		reflect.TypeOf(ResolucionMotivoDecisionCobertura{}),
		reflect.TypeOf(DatosReservaPropietariaOperacionDecisionCobertura{}),
		reflect.TypeOf(ports.OrdenConsumoCobertura{}),
	}
	for indice := 0; indice < tipo.NumField(); indice++ {
		campo := tipo.Field(indice)
		for _, prohibido := range prohibidos {
			if campo.Type == prohibido ||
				(campo.Type.Kind() == reflect.Pointer &&
					campo.Type.Elem() == prohibido) {
				t.Fatalf(
					"la denegación conserva %s en %s",
					prohibido,
					campo.Name,
				)
			}
		}
	}
}

func TestReservaMinimaDenegadaRechazaCrucesEnTodaLigadura(t *testing.T) {
	identidad := identidadOperacionDecisionCoberturaPrueba()
	_, solicitud := solicitudReservaOperacionDecisionCoberturaPrueba(
		t,
		identidad,
	)
	propiedad := datosPropiedadOperacionDecisionCoberturaPrueba(t, solicitud)
	base, err := nuevaReservaMinimaOperacionDecisionCobertura(
		solicitud,
		propiedad,
	)
	if err != nil || base.validar() != nil {
		t.Fatalf("reserva mínima base inválida: %v", err)
	}
	casos := []struct {
		nombre string
		mutar  func(*reservaMinimaOperacionDecisionCobertura)
	}{
		{"organización", func(r *reservaMinimaOperacionDecisionCobertura) {
			r.organizacionRef = "organizacion_ajena"
		}},
		{"expediente", func(r *reservaMinimaOperacionDecisionCobertura) {
			r.expedienteRef = "expediente_ajeno"
		}},
		{"versión", func(r *reservaMinimaOperacionDecisionCobertura) {
			r.versionExpediente++
		}},
		{"ámbito HMAC", func(r *reservaMinimaOperacionDecisionCobertura) {
			r.ambitoHMAC += "a"
		}},
		{"semántica HMAC", func(r *reservaMinimaOperacionDecisionCobertura) {
			r.semanticaHMAC += "a"
		}},
		{"token", func(r *reservaMinimaOperacionDecisionCobertura) {
			r.tokenSHA256 = strings.Repeat("7", 64)
		}},
		{"reserva", func(r *reservaMinimaOperacionDecisionCobertura) {
			r.reservaRef = "reserva_ajena"
		}},
		{"recibo", func(r *reservaMinimaOperacionDecisionCobertura) {
			r.reciboRef = "recibo_ajeno"
		}},
		{"actuación", func(r *reservaMinimaOperacionDecisionCobertura) {
			r.actuacionRef = "actuacion_ajena"
		}},
		{"auditoría", func(r *reservaMinimaOperacionDecisionCobertura) {
			r.auditoriaRef = "auditoria_ajena"
		}},
		{"evento", func(r *reservaMinimaOperacionDecisionCobertura) {
			r.eventoRef = "evento_ajeno"
		}},
		{"correlación", func(r *reservaMinimaOperacionDecisionCobertura) {
			r.correlacionVECRef = "correlacion_ajena"
		}},
		{"decisión VEC", func(r *reservaMinimaOperacionDecisionCobertura) {
			r.decisionVECRef = "decision_vec_ajena"
		}},
		{"cercado anterior", func(r *reservaMinimaOperacionDecisionCobertura) {
			r.revisionCercadoAnterior++
		}},
		{"cercado", func(r *reservaMinimaOperacionDecisionCobertura) {
			r.revisionCercado++
		}},
		{"observada", func(r *reservaMinimaOperacionDecisionCobertura) {
			r.observadaEnDB = r.observadaEnDB.Add(time.Microsecond)
		}},
		{"propiedad", func(r *reservaMinimaOperacionDecisionCobertura) {
			r.propiedadHasta = r.propiedadHasta.Add(-time.Microsecond)
		}},
		{"huella", func(r *reservaMinimaOperacionDecisionCobertura) {
			r.huellaSHA256 = strings.Repeat("8", 64)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			alterada := base
			caso.mutar(&alterada)
			if alterada.validar() == nil {
				t.Fatal("se aceptó una reserva mínima cruzada")
			}
		})
	}
}

func TestReservaMinimaDenegadaRechazaCercadoFueraDeEnteroSeguro(t *testing.T) {
	identidad := identidadOperacionDecisionCoberturaPrueba()
	_, solicitud := solicitudReservaOperacionDecisionCoberturaPrueba(
		t,
		identidad,
	)
	propiedad := datosPropiedadOperacionDecisionCoberturaPrueba(t, solicitud)
	base, err := nuevaReservaMinimaOperacionDecisionCobertura(
		solicitud,
		propiedad,
	)
	if err != nil {
		t.Fatal(err)
	}
	base.revisionCercadoAnterior =
		MaximoEnteroSeguroOperacionDecisionCobertura
	base.revisionCercado = MaximoEnteroSeguroOperacionDecisionCobertura + 1
	base.huellaSHA256, _ =
		calcularHuellaReservaMinimaOperacionDecisionCobertura(base)
	if base.validar() == nil {
		t.Fatal("se aceptó un cercado no portable por encima de 2^53-1")
	}
}

func TestPruebaDenegacionSellaAutoridadesRecursoYLimite(t *testing.T) {
	preparacion := preparacionMinimaPruebaDenegacionOperacionDecisionCobertura(
		t,
	)
	base, err := nuevaPruebaDenegacionOperacionDecisionCobertura(preparacion)
	if err != nil || base.validar() != nil {
		t.Fatalf("prueba denegada base inválida: %v", err)
	}
	casos := []struct {
		nombre string
		mutar  func(*pruebaDenegacionOperacionDecisionCobertura)
	}{
		{"reserva", func(p *pruebaDenegacionOperacionDecisionCobertura) {
			p.reserva.reservaRef = "reserva_ajena"
		}},
		{"referencia recurso", func(p *pruebaDenegacionOperacionDecisionCobertura) {
			p.recursoVEC.Referencia = "reserva_ajena"
		}},
		{"módulo recurso", func(p *pruebaDenegacionOperacionDecisionCobertura) {
			p.recursoVEC.ModuloID = "modulo_ajeno"
		}},
		{"tipo recurso", func(p *pruebaDenegacionOperacionDecisionCobertura) {
			p.recursoVEC.Tipo = "recurso_ajeno"
		}},
		{"ámbito recurso", func(p *pruebaDenegacionOperacionDecisionCobertura) {
			p.recursoVEC.Ambitos["unidad_ejecutora_ref"] = "unidad_ajena"
		}},
		{"semántica recurso", func(p *pruebaDenegacionOperacionDecisionCobertura) {
			p.recursoVEC.Atributos["propuesta_semantica_huella_sha256"] =
				strings.Repeat("9", 64)
		}},
		{"propuesta recurso", func(p *pruebaDenegacionOperacionDecisionCobertura) {
			p.recursoVEC.Atributos["propuesta_ref"] = "propuesta_ajena"
		}},
		{"análisis recurso", func(p *pruebaDenegacionOperacionDecisionCobertura) {
			p.recursoVEC.Atributos["analisis_ref"] = "analisis_ajeno"
		}},
		{"catálogo recurso", func(p *pruebaDenegacionOperacionDecisionCobertura) {
			p.recursoVEC.Atributos["catalogo_version"] = "2"
		}},
		{"política recurso", func(p *pruebaDenegacionOperacionDecisionCobertura) {
			p.recursoVEC.Atributos["politica_ref"] = "politica_ajena"
		}},
		{"vía recurso", func(p *pruebaDenegacionOperacionDecisionCobertura) {
			p.recursoVEC.Atributos["via_elegida"] = "oferta_ajena"
		}},
		{"actuación recurso", func(p *pruebaDenegacionOperacionDecisionCobertura) {
			p.recursoVEC.Atributos["politica_actuacion_ref"] =
				"politica_actuacion_ajena"
		}},
		{"actor", func(p *pruebaDenegacionOperacionDecisionCobertura) {
			p.actorRef = "actor_ajeno"
		}},
		{"perfil", func(p *pruebaDenegacionOperacionDecisionCobertura) {
			p.perfilRef = "perfil_ajeno"
		}},
		{"acción", func(p *pruebaDenegacionOperacionDecisionCobertura) {
			p.accionVEC = domain.AccionRectificarCoberturaGobernada
		}},
		{"finalidad", func(p *pruebaDenegacionOperacionDecisionCobertura) {
			p.finalidadVEC = "finalidad_ajena"
		}},
		{"motivo", func(p *pruebaDenegacionOperacionDecisionCobertura) {
			p.motivoVEC.EntradaClave = "motivo_ajeno"
		}},
		{"límite", func(p *pruebaDenegacionOperacionDecisionCobertura) {
			p.limitePreparacion = p.limitePreparacion.Add(time.Microsecond)
		}},
		{"huella", func(p *pruebaDenegacionOperacionDecisionCobertura) {
			p.huellaSHA256 = strings.Repeat("8", 64)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			alterada := clonarPruebaDenegacionOperacionDecisionCobertura(base)
			caso.mutar(&alterada)
			if alterada.validar() == nil {
				t.Fatal("se aceptó una prueba denegada adulterada")
			}
		})
	}
}

func TestPruebaDenegacionNoComparteSolicitudIdentidadNiRecurso(t *testing.T) {
	preparacion := preparacionMinimaPruebaDenegacionOperacionDecisionCobertura(
		t,
	)
	prueba, err := nuevaPruebaDenegacionOperacionDecisionCobertura(preparacion)
	if err != nil {
		t.Fatal(err)
	}
	preparacion.solicitudReserva.consulta.identidad.actorRef = "actor_ajeno"
	preparacion.recursoVEC.Ambitos["unidad_ejecutora_ref"] = "unidad_ajena"
	preparacion.recursoVEC.Atributos["propuesta_ref"] = "propuesta_ajena"
	if prueba.validar() != nil ||
		prueba.actorRef != "actor_rrhh_opaco_01" ||
		prueba.recursoVEC.Ambitos["unidad_ejecutora_ref"] != "unidad_rrhh_01" ||
		prueba.recursoVEC.Atributos["propuesta_ref"] !=
			"propuesta_cobertura_01" {
		t.Fatal("la prueba denegada retuvo alias de una entrada mutable")
	}
}

func TestClonOrdenDecisionAislaReservaRecursoYColeccionesC1(t *testing.T) {
	origen := &datosPreparacionOrdenOperacionDecisionCobertura{
		reserva: DatosReservaPropietariaOperacionDecisionCobertura{
			AgregadoAnterior: &domain.Expediente{
				Referencia: "expediente_original",
			},
		},
		recursoVEC: dominiovec.RecursoAutorizable{
			Ambitos:   map[string]string{"unidad": "rrhh"},
			Atributos: map[string]string{"fase": "decision"},
		},
		preparacionC1: PreparacionConjuntosViasCobertura{
			conjuntos: []ConjuntoEvidenciasCobertura{{
				evidencias: []EvidenciaConsultaCobertura{{}},
			}},
		},
	}
	clon := clonarDatosPreparacionOrdenOperacionDecisionCobertura(origen)
	if clon == origen ||
		clon.reserva.AgregadoAnterior == origen.reserva.AgregadoAnterior {
		t.Fatal("la orden comparte preparación o agregado reservado")
	}
	clon.reserva.AgregadoAnterior.Referencia = "expediente_clonado"
	clon.recursoVEC.Ambitos["unidad"] = "ajena"
	clon.recursoVEC.Atributos["fase"] = "ajena"
	clon.preparacionC1.conjuntos[0].evidencias[0].comprobadaEn =
		time.Date(2026, 7, 25, 10, 0, 0, 0, time.UTC)
	if origen.reserva.AgregadoAnterior.Referencia != "expediente_original" ||
		origen.recursoVEC.Ambitos["unidad"] != "rrhh" ||
		origen.recursoVEC.Atributos["fase"] != "decision" ||
		!origen.preparacionC1.conjuntos[0].evidencias[0].comprobadaEn.IsZero() {
		t.Fatal("una mutación del clon alcanzó la preparación original")
	}
}

func TestTiposOrdenDecisionSonOpacosYNoSerializables(t *testing.T) {
	valores := []any{
		PreparacionOrdenOperacionDecisionCobertura{},
		OrdenOperacionDecisionCobertura{},
	}
	for _, valor := range valores {
		if _, err := json.Marshal(valor); !errors.Is(
			err,
			ErrSerializacionOperacionDecisionCoberturaProhibida,
		) {
			t.Fatalf("JSON no bloqueado para %T: %v", valor, err)
		}
		if _, err := xml.Marshal(valor); !errors.Is(
			err,
			ErrSerializacionOperacionDecisionCoberturaProhibida,
		) {
			t.Fatalf("XML no bloqueado para %T: %v", valor, err)
		}
		var gobBuffer bytes.Buffer
		if err := gob.NewEncoder(&gobBuffer).Encode(valor); !errors.Is(
			err,
			ErrSerializacionOperacionDecisionCoberturaProhibida,
		) {
			t.Fatalf("gob no bloqueado para %T: %v", valor, err)
		}
		if _, err := valor.(encoding.TextMarshaler).MarshalText(); !errors.Is(
			err,
			ErrSerializacionOperacionDecisionCoberturaProhibida,
		) {
			t.Fatalf("texto no bloqueado para %T: %v", valor, err)
		}
		if _, err := valor.(encoding.BinaryMarshaler).MarshalBinary(); !errors.Is(
			err,
			ErrSerializacionOperacionDecisionCoberturaProhibida,
		) {
			t.Fatalf("binario no bloqueado para %T: %v", valor, err)
		}
		if _, err := valor.(interface {
			MarshalYAML() (any, error)
		}).MarshalYAML(); !errors.Is(
			err,
			ErrSerializacionOperacionDecisionCoberturaProhibida,
		) {
			t.Fatalf("YAML no bloqueado para %T: %v", valor, err)
		}
		if _, err := valor.(interface {
			MarshalCBOR() ([]byte, error)
		}).MarshalCBOR(); !errors.Is(
			err,
			ErrSerializacionOperacionDecisionCoberturaProhibida,
		) {
			t.Fatalf("CBOR no bloqueado para %T: %v", valor, err)
		}
		texto := strings.ToUpper(
			reflect.ValueOf(valor).MethodByName("String").Call(nil)[0].String(),
		)
		if !strings.Contains(texto, "OPACA") {
			t.Fatalf("redacción ausente para %T: %q", valor, texto)
		}
		if formato := strings.ToUpper(fmt.Sprintf("%+v", valor)); !strings.Contains(
			formato,
			"OPACA",
		) {
			t.Fatalf("formateo no redactado para %T: %q", valor, formato)
		}
		if registro := strings.ToUpper(slog.AnyValue(valor).String()); !strings.Contains(
			registro,
			"OPACA",
		) {
			t.Fatalf("log no redactado para %T: %q", valor, registro)
		}
	}
}

func TestRecursoOrdenDecisionExigeCoincidenciaExactaYCopiaMapas(t *testing.T) {
	esperado := dominiovec.RecursoAutorizable{
		Referencia: "reserva_decision_cobertura_01",
		ModuloID:   moduloRecursoOperacionDecisionCobertura,
		Tipo:       tipoRecursoOperacionDecisionCobertura,
		Ambitos:    map[string]string{"organizacion_ref": "organizacion_01"},
		Atributos:  map[string]string{"revision_cercado": "1"},
	}
	if !recursosOperacionDecisionCoberturaIguales(esperado, esperado) {
		t.Fatal("un recurso exacto no coincide consigo mismo")
	}
	copia := clonarRecursoOperacionDecisionCobertura(esperado)
	copia.Atributos["revision_cercado"] = "2"
	if esperado.Atributos["revision_cercado"] != "1" ||
		recursosOperacionDecisionCoberturaIguales(copia, esperado) {
		t.Fatal("la copia comparte mapas o una sustitución fue aceptada")
	}
}

func TestMinimoVigenciasOrdenDecision(t *testing.T) {
	inicio := time.Date(2026, 7, 25, 10, 0, 0, 0, time.UTC)
	esperado := inicio.Add(time.Second)
	if obtenido := minimoInstanteOperacionDecisionCobertura(
		inicio.Add(4*time.Second),
		esperado,
		inicio.Add(3*time.Second),
	); !obtenido.Equal(esperado) {
		t.Fatalf("mínimo de vigencias inesperado: %v", obtenido)
	}
}

func preparacionMinimaPruebaDenegacionOperacionDecisionCobertura(
	t *testing.T,
) *datosPreparacionOrdenOperacionDecisionCobertura {
	t.Helper()
	identidad := identidadOperacionDecisionCoberturaPrueba()
	_, solicitud := solicitudReservaOperacionDecisionCoberturaPrueba(
		t,
		identidad,
	)
	reserva := datosPropiedadOperacionDecisionCoberturaPrueba(t, solicitud)
	huella := strings.Repeat("a", 64)
	recurso := dominiovec.RecursoAutorizable{
		Referencia: reserva.ReservaRef,
		ModuloID:   moduloRecursoOperacionDecisionCobertura,
		Tipo:       tipoRecursoOperacionDecisionCobertura,
		Ambitos: map[string]string{
			"organizacion_ref":     identidad.organizacionRef,
			"unidad_ejecutora_ref": "unidad_rrhh_01",
		},
		Atributos: map[string]string{
			"tipo_operacion":              string(identidad.tipo),
			"expediente_ref":              identidad.expedienteRef,
			"version_expediente_esperada": "2",
			"accion":                      string(identidad.accion),
			"via_elegida":                 string(identidad.viaElegida),
			"propuesta_ref":               "propuesta_cobertura_01",
			"propuesta_huella_sha256":     huella,
			"propuesta_semantica_ref":     identidad.identidadSemantica.Referencia,
			"propuesta_semantica_huella_sha256": identidad.
				identidadSemantica.HuellaSHA256,
			"preparacion_evidencias_ref":           "preparacion_evidencias_01",
			"preparacion_evidencias_huella_sha256": huella,
			"analisis_ref":                         reserva.AnalisisRef,
			"analisis_huella_sha256":               reserva.AnalisisHuellaSHA256,
			"catalogo_ref":                         "catalogo_vias_cobertura",
			"catalogo_version":                     "1",
			"catalogo_huella_sha256":               huella,
			"politica_ref":                         "politica_decision_cobertura",
			"politica_version":                     "1",
			"politica_huella_sha256":               huella,
			"politica_actuacion_ref":               "politica_actuacion_cobertura",
			"politica_actuacion_version":           "1",
			"politica_actuacion_huella_sha256":     huella,
			"reserva_ref":                          reserva.ReservaRef,
			"revision_cercado":                     "1",
		},
	}
	return &datosPreparacionOrdenOperacionDecisionCobertura{
		solicitudReserva: solicitud,
		reserva:          reserva,
		datosGobierno: DatosGobiernoOperacionCobertura{
			Accion:       identidad.accion,
			FinalidadVEC: "contratacion_temporal.decision_cobertura",
			MotivoAutorizacion: motivoAutorizacionGobiernoPrueba(
				"motivo_11111111111111111111111111111111",
			),
		},
		recursoVEC:  recurso,
		preparadaEn: reserva.ObservadaEnDB.Add(time.Microsecond),
		validaHasta: reserva.ObservadaEnDB.Add(time.Second),
	}
}

func clonarPruebaDenegacionOperacionDecisionCobertura(
	origen pruebaDenegacionOperacionDecisionCobertura,
) pruebaDenegacionOperacionDecisionCobertura {
	clon := origen
	clon.recursoVEC = clonarRecursoOperacionDecisionCobertura(origen.recursoVEC)
	return clon
}
