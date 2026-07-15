package domain

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

var instanteLlamamientoPrueba = time.Date(2026, time.July, 15, 10, 30, 0, 123_456_000, time.UTC)

func TestProponerPrimerLlamamientoSeleccionaPrimeraElegibleYExplicaPrefijo(t *testing.T) {
	bolsa, necesidad, politica, instantanea := escenarioLlamamientoPrueba(t)
	evaluaciones := []EvaluacionParticipacionLlamamiento{
		evaluacionLlamamientoPrueba(t, instantanea, necesidad, politica, 3, ResultadoElegible),
		evaluacionLlamamientoPrueba(t, instantanea, necesidad, politica, 1, ResultadoNoElegible),
		evaluacionLlamamientoPrueba(t, instantanea, necesidad, politica, 2, ResultadoNoElegible),
	}

	propuesta, err := ProponerPrimerLlamamiento(OrdenProponerPrimerLlamamiento{
		PropuestaRef: "propuesta:01K0VS9A7Q", Bolsa: bolsa, Necesidad: necesidad,
		Instantanea: instantanea, Politica: politica, Evaluaciones: evaluaciones,
		GeneradaEn: instanteLlamamientoPrueba.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("proponer primer llamamiento: %v", err)
	}
	if err := propuesta.Validar(); err != nil {
		t.Fatalf("propuesta valida: %v", err)
	}
	if propuesta.OrdenSeleccionado != 3 ||
		propuesta.ParticipacionSeleccionadaRef != instantanea.Entradas[2].Participacion.ParticipacionRef ||
		propuesta.SujetoSeleccionadoRef != instantanea.Entradas[2].Participacion.SujetoRef {
		t.Fatalf("seleccion inesperada: %+v", propuesta)
	}
	if len(propuesta.Evaluaciones) != 3 || propuesta.Evaluaciones[0].Orden != 1 ||
		propuesta.Evaluaciones[1].Orden != 2 || propuesta.Evaluaciones[2].Resultado != ResultadoElegible {
		t.Fatalf("prefijo explicable no canonico: %+v", propuesta.Evaluaciones)
	}
	if len(propuesta.HuellaContenidoSHA256) != 64 || propuesta.HuellaInstantaneaSHA256 != instantanea.HuellaContenidoSHA256 {
		t.Fatalf("huellas incompletas: %+v", propuesta)
	}
	if propuesta.TotalParticipacionesInstantanea != uint64(len(instantanea.Entradas)) {
		t.Fatalf("cardinalidad de instantanea perdida: %+v", propuesta)
	}
}

func TestProponerPrimerLlamamientoDetieneEvaluacionEnLaPrimeraElegible(t *testing.T) {
	bolsa, necesidad, politica, instantanea := escenarioLlamamientoPrueba(t)
	primera := evaluacionLlamamientoPrueba(t, instantanea, necesidad, politica, 1, ResultadoElegible)
	propuesta, err := ProponerPrimerLlamamiento(OrdenProponerPrimerLlamamiento{
		PropuestaRef: "propuesta:01K0VS9A7T", Bolsa: bolsa, Necesidad: necesidad,
		Instantanea: instantanea, Politica: politica,
		Evaluaciones: []EvaluacionParticipacionLlamamiento{primera},
		GeneradaEn:   instanteLlamamientoPrueba.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("seleccionar primera: %v", err)
	}
	if propuesta.OrdenSeleccionado != 1 || len(propuesta.Evaluaciones) != 1 ||
		propuesta.TotalParticipacionesInstantanea != 3 {
		t.Fatalf("se trataron personas posteriores sin necesidad: %+v", propuesta)
	}
}

func TestPropuestaEInstantaneaTienenHuellasDeterministas(t *testing.T) {
	bolsa, necesidad, politica, instantanea := escenarioLlamamientoPrueba(t)
	entradasInvertidas := []EntradaOrdenBolsa{
		instantanea.Entradas[2], instantanea.Entradas[0], instantanea.Entradas[1],
	}
	segundaInstantanea, err := NuevaInstantaneaOrdenBolsa(AltaInstantaneaOrdenBolsa{
		InstantaneaRef: instantanea.InstantaneaRef, Version: instantanea.Version, Bolsa: bolsa,
		ReferidaEn: instantanea.ReferidaEn.In(time.FixedZone("CEST", 2*60*60)),
		GeneradaEn: instantanea.GeneradaEn.In(time.FixedZone("CEST", 2*60*60)), Entradas: entradasInvertidas,
	})
	if err != nil {
		t.Fatalf("segunda instantanea: %v", err)
	}
	if segundaInstantanea.HuellaContenidoSHA256 != instantanea.HuellaContenidoSHA256 ||
		segundaInstantanea.ReferidaEn.Location() != time.UTC {
		t.Fatalf("instantanea no determinista: %q / %q", segundaInstantanea.HuellaContenidoSHA256, instantanea.HuellaContenidoSHA256)
	}

	evaluaciones := []EvaluacionParticipacionLlamamiento{
		evaluacionLlamamientoPrueba(t, instantanea, necesidad, politica, 2, ResultadoNoElegible),
		evaluacionLlamamientoPrueba(t, instantanea, necesidad, politica, 3, ResultadoElegible),
		evaluacionLlamamientoPrueba(t, instantanea, necesidad, politica, 1, ResultadoNoElegible),
	}
	propuestaUno := propuestaLlamamientoPrueba(t, bolsa, necesidad, politica, instantanea, evaluaciones)
	for indice := range evaluaciones {
		invertirMotivos(evaluaciones[indice].Motivos)
	}
	propuestaDos := propuestaLlamamientoPrueba(t, bolsa, necesidad, politica, segundaInstantanea, evaluaciones)
	if propuestaUno.HuellaContenidoSHA256 != propuestaDos.HuellaContenidoSHA256 ||
		!reflect.DeepEqual(propuestaUno, propuestaDos) {
		t.Fatalf("propuesta no reproducible:\n%+v\n%+v", propuestaUno, propuestaDos)
	}
}

func TestParticipacionMantieneSituacionVigenteSemicerrada(t *testing.T) {
	participacion := participacionLlamamientoPrueba(t, 1)
	frontera := instanteLlamamientoPrueba.Add(-6 * time.Hour)
	anterior, existe := participacion.SituacionVigenteEn(frontera.Add(-time.Microsecond))
	if !existe || anterior.Secuencia != 1 || anterior.Hasta == nil {
		t.Fatalf("situacion anterior: %+v / %t", anterior, existe)
	}
	vigente, existe := participacion.SituacionVigenteEn(frontera)
	if !existe || vigente.Secuencia != 2 || vigente.EstadoClave != "estado_operativo" {
		t.Fatalf("situacion en frontera: %+v / %t", vigente, existe)
	}
	*anterior.Hasta = anterior.Hasta.Add(time.Hour)
	recuperada, _ := participacion.SituacionVigenteEn(frontera.Add(-time.Microsecond))
	if recuperada.Hasta.Equal(*anterior.Hasta) {
		t.Fatal("SituacionVigenteEn comparte el puntero temporal interno")
	}
	if _, existe := participacion.SituacionVigenteEn(participacion.AltaEn.Add(-time.Microsecond)); existe {
		t.Fatal("se encontro situacion antes del alta")
	}
}

func TestParticipacionRechazaHuecosSolapesYFinalSinSituacionAbierta(t *testing.T) {
	base := participacionLlamamientoPrueba(t, 1)
	casos := []struct {
		nombre string
		mutar  func(*ParticipacionBolsa)
	}{
		{"hueco", func(p *ParticipacionBolsa) {
			p.Situaciones[0].Hasta = instantePtr(p.Situaciones[1].Desde.Add(-time.Microsecond))
		}},
		{"solape", func(p *ParticipacionBolsa) {
			p.Situaciones[0].Hasta = instantePtr(p.Situaciones[1].Desde.Add(time.Microsecond))
		}},
		{"secuencia", func(p *ParticipacionBolsa) { p.Situaciones[1].Secuencia = 3 }},
		{"decision_reutilizada", func(p *ParticipacionBolsa) {
			p.Situaciones[1].DecisionRef = p.Situaciones[0].DecisionRef
		}},
		{"ultimo_cerrado", func(p *ParticipacionBolsa) {
			p.Situaciones[1].Hasta = instantePtr(instanteLlamamientoPrueba.Add(time.Hour))
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			mutada, err := base.ClonarCanonica()
			if err != nil {
				t.Fatal(err)
			}
			caso.mutar(&mutada)
			if err := mutada.Validar(); !errors.Is(err, ErrParticipacionBolsaInvalida) {
				t.Fatalf("historia no canonica admitida: %v", err)
			}
		})
	}
}

func TestConstructoresCanonicalizanUTCYRechazanSubmicrosegundos(t *testing.T) {
	zona := time.FixedZone("CEST", 2*60*60)
	alta := altaBolsaLlamamientoPrueba()
	alta.ConstituidaEn = alta.ConstituidaEn.In(zona)
	alta.VigenteDesde = alta.VigenteDesde.In(zona)
	bolsa, err := NuevaBolsaConstituida(alta)
	if err != nil {
		t.Fatalf("canonizar bolsa: %v", err)
	}
	if bolsa.ConstituidaEn.Location() != time.UTC || bolsa.VigenteDesde.Location() != time.UTC {
		t.Fatalf("instantes no UTC: %+v", bolsa)
	}
	directa := bolsa
	directa.ConstituidaEn = directa.ConstituidaEn.In(zona)
	if err := directa.Validar(); !errors.Is(err, ErrBolsaConstituidaInvalida) {
		t.Fatalf("estructura directa no canonica admitida: %v", err)
	}

	necesidad := altaNecesidadLlamamientoPrueba(t, bolsa)
	necesidad.InicioPrevisto = necesidad.InicioPrevisto.Add(time.Nanosecond)
	if _, err := NuevaNecesidadCobertura(necesidad); !errors.Is(err, ErrNecesidadCoberturaInvalida) {
		t.Fatalf("precision submicrosegundo admitida: %v", err)
	}
	necesidad = altaNecesidadLlamamientoPrueba(t, bolsa)
	necesidad.CreadaEn = time.Date(0, time.January, 1, 0, 0, 0, 0, time.UTC)
	if _, err := NuevaNecesidadCobertura(necesidad); !errors.Is(err, ErrNecesidadCoberturaInvalida) {
		t.Fatalf("instante fuera del calendario canonico admitido: %v", err)
	}
}

func TestReferenciasDeLlamamientoRechazanDNI_NIE_ComodinesYNoCanonicas(t *testing.T) {
	bolsa, necesidad, politica, instantanea := escenarioLlamamientoPrueba(t)
	bolsa.BolsaRef = "bolsa:00000000A"
	if err := bolsa.Validar(); !errors.Is(err, ErrBolsaConstituidaInvalida) {
		t.Fatalf("DNI admitido en bolsa: %v", err)
	}
	participacion := participacionLlamamientoPrueba(t, 1)
	participacion.SujetoRef = "sujeto:X0000000A"
	if err := participacion.Validar(); !errors.Is(err, ErrParticipacionBolsaInvalida) {
		t.Fatalf("NIE admitido como sujeto: %v", err)
	}
	for _, referencia := range []string{
		"sujeto:12.345.678-Z",
		"sujeto:X-1234567-L",
		"sujeto:nif:opaco",
		"sujeto:pasaporte:opaco",
	} {
		participacion = participacionLlamamientoPrueba(t, 1)
		participacion.SujetoRef = referencia
		if err := participacion.Validar(); !errors.Is(err, ErrParticipacionBolsaInvalida) {
			t.Fatalf("documento personal admitido en %q: %v", referencia, err)
		}
	}
	necesidad.PuestoRef = "puesto:*"
	if err := necesidad.Validar(); !errors.Is(err, ErrNecesidadCoberturaInvalida) {
		t.Fatalf("comodin admitido: %v", err)
	}
	politica.PoliticaRef = "politica:dni:opaca"
	if err := politica.Validar(); !errors.Is(err, ErrPoliticaLlamamientoInvalida) {
		t.Fatalf("etiqueta DNI admitida: %v", err)
	}
	participacion = participacionLlamamientoPrueba(t, 1)
	participacion.Situaciones[1].EstadoClave = "dni_12_345_678_z"
	if err := participacion.Validar(); !errors.Is(err, ErrParticipacionBolsaInvalida) {
		t.Fatalf("documento personal admitido como clave gobernada: %v", err)
	}
	evaluacion := evaluacionLlamamientoPrueba(t, instantanea, altaNecesidadValida(t), politicaLlamamientoPrueba(t), 1, ResultadoElegible)
	evaluacion.ParticipacionRef = " participacion:01K0VSA1 "
	if err := evaluacion.Validar(); !errors.Is(err, ErrEvaluacionLlamamientoInvalida) {
		t.Fatalf("referencia no canonica admitida: %v", err)
	}
}

func TestInstantaneaOrdenaYCopiaDefensivamenteYDetectaManipulacion(t *testing.T) {
	bolsa := bolsaLlamamientoPrueba(t)
	entradas := []EntradaOrdenBolsa{
		{Orden: 2, Participacion: participacionLlamamientoPrueba(t, 2)},
		{Orden: 1, Participacion: participacionLlamamientoPrueba(t, 1)},
	}
	instantanea, err := NuevaInstantaneaOrdenBolsa(AltaInstantaneaOrdenBolsa{
		InstantaneaRef: "instantanea:01K0VS8Q", Version: 1, Bolsa: bolsa,
		ReferidaEn: instanteLlamamientoPrueba, GeneradaEn: instanteLlamamientoPrueba.Add(time.Minute),
		Entradas: entradas,
	})
	if err != nil {
		t.Fatalf("crear instantanea: %v", err)
	}
	entradas[0].Participacion.SujetoRef = "sujeto:mutado"
	entradas[1].Participacion.Situaciones[1].EstadoClave = "mutado"
	if instantanea.Entradas[0].Participacion.SujetoRef == "sujeto:mutado" ||
		instantanea.Entradas[0].Participacion.Situaciones[1].EstadoClave == "mutado" {
		t.Fatal("la instantanea comparte entradas o situaciones con la entrada")
	}
	manipulada, err := instantanea.ClonarCanonica()
	if err != nil {
		t.Fatal(err)
	}
	manipulada.Entradas[0], manipulada.Entradas[1] = manipulada.Entradas[1], manipulada.Entradas[0]
	if err := manipulada.Validar(); !errors.Is(err, ErrInstantaneaOrdenBolsaInvalida) {
		t.Fatalf("reordenacion posterior no detectada: %v", err)
	}
	manipulada, _ = instantanea.ClonarCanonica()
	manipulada.Entradas[0].Participacion.Situaciones[1].EstadoClave = "otro_estado"
	if err := manipulada.Validar(); !errors.Is(err, ErrInstantaneaOrdenBolsaInvalida) {
		t.Fatalf("alteracion cubierta por huella no detectada: %v", err)
	}
}

func TestInstantaneaRechazaOrdenDuplicadosYParticipacionSinSituacionVigente(t *testing.T) {
	bolsa := bolsaLlamamientoPrueba(t)
	p1 := participacionLlamamientoPrueba(t, 1)
	p2 := participacionLlamamientoPrueba(t, 2)
	casos := []struct {
		nombre   string
		entradas []EntradaOrdenBolsa
	}{
		{"orden_no_contiguo", []EntradaOrdenBolsa{{Orden: 2, Participacion: p1}}},
		{"participacion_duplicada", []EntradaOrdenBolsa{{Orden: 1, Participacion: p1}, {Orden: 2, Participacion: p1}}},
		{"sujeto_duplicado", []EntradaOrdenBolsa{{Orden: 1, Participacion: p1}, {Orden: 2, Participacion: conSujeto(p2, p1.SujetoRef)}}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			_, err := NuevaInstantaneaOrdenBolsa(AltaInstantaneaOrdenBolsa{
				InstantaneaRef: "instantanea:01K0VS8R", Version: 1, Bolsa: bolsa,
				ReferidaEn: instanteLlamamientoPrueba, GeneradaEn: instanteLlamamientoPrueba,
				Entradas: caso.entradas,
			})
			if !errors.Is(err, ErrInstantaneaOrdenBolsaInvalida) {
				t.Fatalf("instantanea invalida admitida: %v", err)
			}
		})
	}
	tardia := participacionLlamamientoPrueba(t, 3)
	desplazamiento := instanteLlamamientoPrueba.Add(time.Hour).Sub(tardia.AltaEn)
	tardia.AltaEn = tardia.AltaEn.Add(desplazamiento)
	for indice := range tardia.Situaciones {
		tardia.Situaciones[indice].Desde = tardia.Situaciones[indice].Desde.Add(desplazamiento)
		if tardia.Situaciones[indice].Hasta != nil {
			*tardia.Situaciones[indice].Hasta = tardia.Situaciones[indice].Hasta.Add(desplazamiento)
		}
	}
	if err := tardia.Validar(); err != nil {
		t.Fatalf("participacion futura estructuralmente valida: %v", err)
	}
	_, err := NuevaInstantaneaOrdenBolsa(AltaInstantaneaOrdenBolsa{
		InstantaneaRef: "instantanea:01K0VS8S", Version: 1, Bolsa: bolsa,
		ReferidaEn: instanteLlamamientoPrueba, GeneradaEn: instanteLlamamientoPrueba,
		Entradas: []EntradaOrdenBolsa{{Orden: 1, Participacion: tardia}},
	})
	if !errors.Is(err, ErrInstantaneaOrdenBolsaInvalida) {
		t.Fatalf("participacion sin situacion vigente admitida: %v", err)
	}
}

func TestPropuestaFallaCerradaAnteEvaluacionesIncompletasOIncoherentes(t *testing.T) {
	bolsa, necesidad, politica, instantanea := escenarioLlamamientoPrueba(t)
	base := []EvaluacionParticipacionLlamamiento{
		evaluacionLlamamientoPrueba(t, instantanea, necesidad, politica, 1, ResultadoNoElegible),
		evaluacionLlamamientoPrueba(t, instantanea, necesidad, politica, 2, ResultadoNoElegible),
		evaluacionLlamamientoPrueba(t, instantanea, necesidad, politica, 3, ResultadoElegible),
	}
	ordenBase := func(evaluaciones []EvaluacionParticipacionLlamamiento) OrdenProponerPrimerLlamamiento {
		return OrdenProponerPrimerLlamamiento{
			PropuestaRef: "propuesta:01K0VS9A7R", Bolsa: bolsa, Necesidad: necesidad,
			Instantanea: instantanea, Politica: politica, Evaluaciones: evaluaciones,
			GeneradaEn: instanteLlamamientoPrueba.Add(2 * time.Minute),
		}
	}
	if _, err := ProponerPrimerLlamamiento(ordenBase(nil)); !errors.Is(err, ErrSinParticipacionElegible) {
		t.Fatalf("ausencia de evaluacion no fallo cerrada: %v", err)
	}
	ninguna := clonarEvaluacionesPrueba(base[:2])
	if _, err := ProponerPrimerLlamamiento(ordenBase(ninguna)); !errors.Is(err, ErrSinParticipacionElegible) {
		t.Fatalf("sin elegible no produjo error tipado: %v", err)
	}
	sinPrimera := clonarEvaluacionesPrueba(base[1:])
	if _, err := ProponerPrimerLlamamiento(ordenBase(sinPrimera)); !errors.Is(err, ErrEvaluacionLlamamientoInvalida) {
		t.Fatalf("se omitio la primera posicion: %v", err)
	}
	politicaDistinta := clonarEvaluacionesPrueba(base)
	politicaDistinta[1].HuellaPoliticaSHA256 = huellaLlamamientoPrueba("e")
	if _, err := ProponerPrimerLlamamiento(ordenBase(politicaDistinta)); !errors.Is(err, ErrEvaluacionLlamamientoInvalida) {
		t.Fatalf("se mezclo politica: %v", err)
	}
	estadoDistinto := clonarEvaluacionesPrueba(base)
	estadoDistinto[0].EstadoVersion++
	if _, err := ProponerPrimerLlamamiento(ordenBase(estadoDistinto)); !errors.Is(err, ErrEvaluacionLlamamientoInvalida) {
		t.Fatalf("se mezclo situacion: %v", err)
	}
	primeraElegibleYContinua := clonarEvaluacionesPrueba(base)
	primeraElegibleYContinua[0].Resultado = ResultadoElegible
	if _, err := ProponerPrimerLlamamiento(ordenBase(primeraElegibleYContinua)); !errors.Is(err, ErrEvaluacionLlamamientoInvalida) {
		t.Fatalf("se evaluo despues de la primera elegible: %v", err)
	}
	desconocida := clonarEvaluacionesPrueba(base)
	desconocida[2].Resultado = ResultadoElegibilidadLlamamiento("desconocido")
	if _, err := ProponerPrimerLlamamiento(ordenBase(desconocida)); !errors.Is(err, ErrEvaluacionLlamamientoInvalida) {
		t.Fatalf("resultado desconocido admitido: %v", err)
	}
}

func TestEvaluacionQuedaLigadaALaHuellaExactaDeLaNecesidad(t *testing.T) {
	bolsa, necesidad, politica, instantanea := escenarioLlamamientoPrueba(t)
	evaluaciones := evaluacionesHastaPrimeraElegible(t, instantanea, necesidad, politica)
	orden := OrdenProponerPrimerLlamamiento{
		PropuestaRef: "propuesta:01K0VS9B1A", Bolsa: bolsa, Necesidad: necesidad,
		Instantanea: instantanea, Politica: politica, Evaluaciones: evaluaciones,
		GeneradaEn: instanteLlamamientoPrueba.Add(2 * time.Minute),
	}
	versionDeBolsaDistinta := necesidad
	versionDeBolsaDistinta.VersionBolsa++
	if err := versionDeBolsaDistinta.Validar(); err != nil {
		t.Fatalf("necesidad con enlace estructural alternativo: %v", err)
	}
	orden.Necesidad = versionDeBolsaDistinta
	if _, err := ProponerPrimerLlamamiento(orden); !errors.Is(err, ErrPropuestaLlamamientoInvalida) {
		t.Fatalf("necesidad de otra version de bolsa admitida: %v", err)
	}

	contenidoDistinto := necesidad
	contenidoDistinto.NumeroPuestos++
	if err := contenidoDistinto.Validar(); err != nil {
		t.Fatalf("necesidad alternativa estructuralmente valida: %v", err)
	}
	orden.Necesidad = contenidoDistinto
	if _, err := ProponerPrimerLlamamiento(orden); !errors.Is(err, ErrEvaluacionLlamamientoInvalida) {
		t.Fatalf("evaluacion reutilizada con otro contenido de necesidad: %v", err)
	}

	orden.Necesidad = necesidad
	orden.Evaluaciones = clonarEvaluacionesPrueba(evaluaciones)
	orden.Evaluaciones[1].HuellaNecesidadSHA256 = huellaLlamamientoPrueba("0")
	if _, err := ProponerPrimerLlamamiento(orden); !errors.Is(err, ErrEvaluacionLlamamientoInvalida) {
		t.Fatalf("huella de necesidad cruzada admitida: %v", err)
	}
}

func TestPropuestaExigeCronologiaCausalDeInstantaneaYEvaluaciones(t *testing.T) {
	bolsa, necesidad, politica, instantanea := escenarioLlamamientoPrueba(t)
	evaluaciones := evaluacionesHastaPrimeraElegible(t, instantanea, necesidad, politica)
	generadaEn := instanteLlamamientoPrueba.Add(2 * time.Minute)
	orden := OrdenProponerPrimerLlamamiento{
		PropuestaRef: "propuesta:01K0VS9B1B", Bolsa: bolsa, Necesidad: necesidad,
		Instantanea: instantanea, Politica: politica, Evaluaciones: evaluaciones,
		GeneradaEn: generadaEn,
	}

	antesDeInstantanea := clonarEvaluacionesPrueba(evaluaciones)
	antesDeInstantanea[0].EvaluadaEn = instantanea.GeneradaEn.Add(-time.Microsecond)
	orden.Evaluaciones = antesDeInstantanea
	if _, err := ProponerPrimerLlamamiento(orden); !errors.Is(err, ErrEvaluacionLlamamientoInvalida) {
		t.Fatalf("evaluacion anterior a su instantanea admitida: %v", err)
	}

	despuesDePropuesta := clonarEvaluacionesPrueba(evaluaciones)
	despuesDePropuesta[len(despuesDePropuesta)-1].EvaluadaEn = generadaEn.Add(time.Microsecond)
	orden.Evaluaciones = despuesDePropuesta
	if _, err := ProponerPrimerLlamamiento(orden); !errors.Is(err, ErrEvaluacionLlamamientoInvalida) {
		t.Fatalf("evaluacion posterior a la propuesta admitida: %v", err)
	}

	dentroDeVentana := clonarEvaluacionesPrueba(evaluaciones)
	for indice := range dentroDeVentana {
		dentroDeVentana[indice].EvaluadaEn = instantanea.GeneradaEn.Add(time.Duration(indice) * time.Second)
	}
	orden.Evaluaciones = dentroDeVentana
	propuesta, err := ProponerPrimerLlamamiento(orden)
	if err != nil {
		t.Fatalf("evaluaciones causales dentro de la ventana: %v", err)
	}
	if !propuesta.InstantaneaGeneradaEn.Equal(instantanea.GeneradaEn) {
		t.Fatalf("se perdio el instante causal de la instantanea: %+v", propuesta)
	}

	manipulada := propuesta
	manipulada.InstantaneaGeneradaEn = propuesta.GeneradaEn.Add(time.Microsecond)
	manipulada.HuellaContenidoSHA256, _ = manipulada.calcularHuellaContenidoSHA256()
	if err := manipulada.Validar(); !errors.Is(err, ErrPropuestaLlamamientoInvalida) {
		t.Fatalf("propuesta anterior a su instantanea admitida: %v", err)
	}
}

func TestPropuestaExigePoliticaYBolsaVigentesYNecesidadPreexistente(t *testing.T) {
	bolsa, necesidad, politica, instantanea := escenarioLlamamientoPrueba(t)
	evaluaciones := evaluacionesHastaPrimeraElegible(t, instantanea, necesidad, politica)
	orden := OrdenProponerPrimerLlamamiento{
		PropuestaRef: "propuesta:01K0VS9A7S", Bolsa: bolsa, Necesidad: necesidad,
		Instantanea: instantanea, Politica: politica, Evaluaciones: evaluaciones,
		GeneradaEn: instanteLlamamientoPrueba.Add(2 * time.Minute),
	}
	politicaCaducada := politica
	politicaCaducada.VigenteHasta = instantePtr(instantanea.ReferidaEn)
	orden.Politica = politicaCaducada
	if _, err := ProponerPrimerLlamamiento(orden); !errors.Is(err, ErrPropuestaLlamamientoInvalida) {
		t.Fatalf("politica caducada admitida: %v", err)
	}
	orden.Politica = politica
	politicaCaducadaAlGenerar := politica
	politicaCaducadaAlGenerar.VigenteHasta = instantePtr(orden.GeneradaEn)
	orden.Politica = politicaCaducadaAlGenerar
	if _, err := ProponerPrimerLlamamiento(orden); !errors.Is(err, ErrPropuestaLlamamientoInvalida) {
		t.Fatalf("politica caducada entre referencia y propuesta admitida: %v", err)
	}
	orden.Politica = politica
	orden.Necesidad.CreadaEn = instantanea.ReferidaEn.Add(time.Microsecond)
	if _, err := ProponerPrimerLlamamiento(orden); !errors.Is(err, ErrPropuestaLlamamientoInvalida) {
		t.Fatalf("necesidad futura evaluada: %v", err)
	}
	orden.Necesidad = necesidad
	bolsaCaducada := bolsa
	bolsaCaducada.VigenteHasta = instantePtr(instantanea.ReferidaEn)
	orden.Bolsa = bolsaCaducada
	if _, err := ProponerPrimerLlamamiento(orden); !errors.Is(err, ErrPropuestaLlamamientoInvalida) {
		t.Fatalf("bolsa caducada admitida: %v", err)
	}
	bolsaCaducadaAlGenerar := bolsa
	bolsaCaducadaAlGenerar.VigenteHasta = instantePtr(orden.GeneradaEn)
	instantaneaCaducadaAlGenerar, err := NuevaInstantaneaOrdenBolsa(AltaInstantaneaOrdenBolsa{
		InstantaneaRef: "instantanea:01K0VS8T", Version: instantanea.Version, Bolsa: bolsaCaducadaAlGenerar,
		ReferidaEn: instantanea.ReferidaEn, GeneradaEn: instantanea.GeneradaEn, Entradas: instantanea.Entradas,
	})
	if err != nil {
		t.Fatalf("instantanea historica de bolsa aun vigente: %v", err)
	}
	orden.Bolsa = bolsaCaducadaAlGenerar
	orden.Instantanea = instantaneaCaducadaAlGenerar
	orden.Evaluaciones = evaluacionesHastaPrimeraElegible(t, instantaneaCaducadaAlGenerar, necesidad, politica)
	if _, err := ProponerPrimerLlamamiento(orden); !errors.Is(err, ErrPropuestaLlamamientoInvalida) {
		t.Fatalf("bolsa caducada entre referencia y propuesta admitida: %v", err)
	}
	orden.Bolsa = bolsa
	orden.Instantanea = instantanea
	orden.Evaluaciones = evaluaciones
	orden.GeneradaEn = necesidad.FinPrevisto
	if _, err := ProponerPrimerLlamamiento(orden); !errors.Is(err, ErrPropuestaLlamamientoInvalida) {
		t.Fatalf("propuesta generada tras finalizar la necesidad: %v", err)
	}
}

func TestPropuestaRechazaCardinalidadFueraDelLimiteDeDominio(t *testing.T) {
	bolsa, necesidad, politica, instantanea := escenarioLlamamientoPrueba(t)
	propuesta := propuestaLlamamientoPrueba(
		t, bolsa, necesidad, politica, instantanea,
		evaluacionesHastaPrimeraElegible(t, instantanea, necesidad, politica),
	)
	propuesta.TotalParticipacionesInstantanea = maximoEntradasOrdenBolsa + 1
	propuesta.HuellaContenidoSHA256, _ = propuesta.calcularHuellaContenidoSHA256()
	if err := propuesta.Validar(); !errors.Is(err, ErrPropuestaLlamamientoInvalida) {
		t.Fatalf("cardinalidad no acotada admitida: %v", err)
	}
}

func TestPropuestaCopiaEvaluacionesYDetectaManipulacion(t *testing.T) {
	bolsa, necesidad, politica, instantanea := escenarioLlamamientoPrueba(t)
	evaluaciones := evaluacionesHastaPrimeraElegible(t, instantanea, necesidad, politica)
	propuesta := propuestaLlamamientoPrueba(t, bolsa, necesidad, politica, instantanea, evaluaciones)
	evaluaciones[0].Motivos[0].Clave = "mutado"
	evaluaciones[1].SujetoRef = "sujeto:mutado"
	if propuesta.Evaluaciones[0].Motivos[0].Clave == "mutado" || propuesta.Evaluaciones[1].SujetoRef == "sujeto:mutado" {
		t.Fatal("la propuesta comparte evaluaciones con la orden")
	}
	copia, err := propuesta.EvaluacionesCanonicas()
	if err != nil {
		t.Fatal(err)
	}
	copia[0].Motivos[0].Clave = "otra_mutacion"
	if propuesta.Evaluaciones[0].Motivos[0].Clave == "otra_mutacion" {
		t.Fatal("EvaluacionesCanonicas comparte motivos")
	}
	manipulada := propuesta
	manipulada.ParticipacionSeleccionadaRef = propuesta.Evaluaciones[0].ParticipacionRef
	if err := manipulada.Validar(); !errors.Is(err, ErrPropuestaLlamamientoInvalida) {
		t.Fatalf("seleccion manipulada admitida: %v", err)
	}
	manipulada = propuesta
	manipulada.Evaluaciones[0].Resultado = ResultadoElegible
	if err := manipulada.Validar(); !errors.Is(err, ErrPropuestaLlamamientoInvalida) {
		t.Fatalf("evaluacion manipulada admitida: %v", err)
	}
}

func TestPropuestaRechazaRecibosReutilizadosEntreParticipaciones(t *testing.T) {
	bolsa, necesidad, politica, instantanea := escenarioLlamamientoPrueba(t)
	evaluaciones := evaluacionesHastaPrimeraElegible(t, instantanea, necesidad, politica)
	propuesta := propuestaLlamamientoPrueba(t, bolsa, necesidad, politica, instantanea, evaluaciones)
	manipulada, err := propuesta.ClonarCanonica()
	if err != nil {
		t.Fatal(err)
	}
	manipulada.Evaluaciones[1].EntradaEvaluacionRef = manipulada.Evaluaciones[0].EntradaEvaluacionRef
	manipulada.HuellaContenidoSHA256, err = manipulada.calcularHuellaContenidoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	if err := manipulada.Validar(); !errors.Is(err, ErrPropuestaLlamamientoInvalida) {
		t.Fatalf("recibo de entrada reutilizado admitido: %v", err)
	}
	manipulada, _ = propuesta.ClonarCanonica()
	manipulada.Evaluaciones[1].ResultadoEvaluacionRef = manipulada.Evaluaciones[0].ResultadoEvaluacionRef
	manipulada.HuellaContenidoSHA256, _ = manipulada.calcularHuellaContenidoSHA256()
	if err := manipulada.Validar(); !errors.Is(err, ErrPropuestaLlamamientoInvalida) {
		t.Fatalf("recibo de resultado reutilizado admitido: %v", err)
	}
	manipulada, _ = propuesta.ClonarCanonica()
	manipulada.Evaluaciones[1].ResultadoEvaluacionRef = manipulada.Evaluaciones[0].EntradaEvaluacionRef
	manipulada.HuellaContenidoSHA256, _ = manipulada.calcularHuellaContenidoSHA256()
	if err := manipulada.Validar(); !errors.Is(err, ErrPropuestaLlamamientoInvalida) {
		t.Fatalf("recibo reutilizado con otra funcion admitido: %v", err)
	}
}

func TestNecesidadOrdenaRequisitosYRechazaDuplicadosSinReglasHardcodeadas(t *testing.T) {
	bolsa := bolsaLlamamientoPrueba(t)
	alta := altaNecesidadLlamamientoPrueba(t, bolsa)
	alta.Requisitos = []RequisitoCobertura{
		{Clave: "zona", ValorRef: "zona:granada", Version: 2, HuellaSHA256: huellaLlamamientoPrueba("a")},
		{Clave: "jornada", ValorRef: "jornada:completa", Version: 5, HuellaSHA256: huellaLlamamientoPrueba("b")},
	}
	necesidad, err := NuevaNecesidadCobertura(alta)
	if err != nil {
		t.Fatalf("necesidad configurable: %v", err)
	}
	if necesidad.Requisitos[0].Clave != "jornada" || necesidad.Requisitos[1].Clave != "zona" {
		t.Fatalf("requisitos no canonicos: %+v", necesidad.Requisitos)
	}
	alta.Requisitos[0].Clave = "mutada"
	if necesidad.Requisitos[1].Clave != "zona" {
		t.Fatal("necesidad comparte requisitos con el alta")
	}
	duplicada := altaNecesidadLlamamientoPrueba(t, bolsa)
	duplicada.Requisitos = []RequisitoCobertura{
		{Clave: "zona", ValorRef: "zona:a", Version: 1, HuellaSHA256: huellaLlamamientoPrueba("a")},
		{Clave: "zona", ValorRef: "zona:b", Version: 2, HuellaSHA256: huellaLlamamientoPrueba("b")},
	}
	if _, err := NuevaNecesidadCobertura(duplicada); !errors.Is(err, ErrNecesidadCoberturaInvalida) {
		t.Fatalf("requisito duplicado admitido: %v", err)
	}
	finalizadaAntesDeCrearse := altaNecesidadLlamamientoPrueba(t, bolsa)
	finalizadaAntesDeCrearse.CreadaEn = finalizadaAntesDeCrearse.FinPrevisto
	if _, err := NuevaNecesidadCobertura(finalizadaAntesDeCrearse); !errors.Is(err, ErrNecesidadCoberturaInvalida) {
		t.Fatalf("necesidad creada tras finalizar admitida: %v", err)
	}
}

func TestEvaluacionExigeExplicacionYRecibosExactos(t *testing.T) {
	_, necesidad, politica, instantanea := escenarioLlamamientoPrueba(t)
	base := evaluacionLlamamientoPrueba(t, instantanea, necesidad, politica, 1, ResultadoNoElegible)
	casos := []struct {
		nombre string
		mutar  func(*EvaluacionParticipacionLlamamiento)
	}{
		{"sin_motivos", func(e *EvaluacionParticipacionLlamamiento) { e.Motivos = nil }},
		{"sin_entrada", func(e *EvaluacionParticipacionLlamamiento) { e.EntradaEvaluacionRef = "" }},
		{"sin_huella_entrada", func(e *EvaluacionParticipacionLlamamiento) { e.HuellaEntradaSHA256 = "" }},
		{"sin_resultado", func(e *EvaluacionParticipacionLlamamiento) { e.ResultadoEvaluacionRef = "" }},
		{"mismo_recibo_entrada_resultado", func(e *EvaluacionParticipacionLlamamiento) {
			e.ResultadoEvaluacionRef = e.EntradaEvaluacionRef
		}},
		{"sin_huella_resultado", func(e *EvaluacionParticipacionLlamamiento) { e.HuellaResultadoSHA256 = "" }},
		{"regla_sin_version", func(e *EvaluacionParticipacionLlamamiento) { e.Motivos[0].VersionRegla = 0 }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			evaluacion := base
			evaluacion.Motivos = append([]MotivoEvaluacionLlamamiento(nil), base.Motivos...)
			caso.mutar(&evaluacion)
			if err := evaluacion.Validar(); !errors.Is(err, ErrEvaluacionLlamamientoInvalida) {
				t.Fatalf("evaluacion incompleta admitida: %v", err)
			}
		})
	}
}

func escenarioLlamamientoPrueba(t *testing.T) (BolsaConstituida, NecesidadCobertura, ReferenciaPoliticaLlamamiento, InstantaneaOrdenBolsa) {
	t.Helper()
	bolsa := bolsaLlamamientoPrueba(t)
	necesidad, err := NuevaNecesidadCobertura(altaNecesidadLlamamientoPrueba(t, bolsa))
	if err != nil {
		t.Fatalf("necesidad: %v", err)
	}
	politica := politicaLlamamientoPrueba(t)
	instantanea, err := NuevaInstantaneaOrdenBolsa(AltaInstantaneaOrdenBolsa{
		InstantaneaRef: "instantanea:01K0VS8Q", Version: 4, Bolsa: bolsa,
		ReferidaEn: instanteLlamamientoPrueba, GeneradaEn: instanteLlamamientoPrueba.Add(time.Minute),
		Entradas: []EntradaOrdenBolsa{
			{Orden: 3, Participacion: participacionLlamamientoPrueba(t, 3)},
			{Orden: 1, Participacion: participacionLlamamientoPrueba(t, 1)},
			{Orden: 2, Participacion: participacionLlamamientoPrueba(t, 2)},
		},
	})
	if err != nil {
		t.Fatalf("instantanea: %v", err)
	}
	return bolsa, necesidad, politica, instantanea
}

func altaBolsaLlamamientoPrueba() AltaBolsaConstituida {
	return AltaBolsaConstituida{
		BolsaRef: "bolsa:01K0VRZZ", Version: 3, ProcesoRef: "proceso:01K0VRZY",
		CategoriaRef: "categoria:auxiliar_administrativo", ListadoDefinitivoRef: "listado:01K0VRZX",
		VersionListado: 7, HuellaListadoSHA256: huellaLlamamientoPrueba("a"),
		ResolucionConstitucionRef: "resolucion:01K0VRZW", HuellaResolucionSHA256: huellaLlamamientoPrueba("b"),
		ConstituidaEn: instanteLlamamientoPrueba.Add(-48 * time.Hour),
		VigenteDesde:  instanteLlamamientoPrueba.Add(-24 * time.Hour),
	}
}

func bolsaLlamamientoPrueba(t *testing.T) BolsaConstituida {
	t.Helper()
	bolsa, err := NuevaBolsaConstituida(altaBolsaLlamamientoPrueba())
	if err != nil {
		t.Fatalf("bolsa: %v", err)
	}
	return bolsa
}

func participacionLlamamientoPrueba(t *testing.T, numero int) ParticipacionBolsa {
	t.Helper()
	altaEn := instanteLlamamientoPrueba.Add(-24 * time.Hour)
	cambio := instanteLlamamientoPrueba.Add(-6 * time.Hour)
	participacion, err := NuevaParticipacionBolsa(AltaParticipacionBolsa{
		ParticipacionRef: "participacion:01K0VS" + string(rune('A'+numero)),
		BolsaRef:         "bolsa:01K0VRZZ", SujetoRef: "sujeto:01K0VT" + string(rune('A'+numero)), Version: 2, AltaEn: altaEn,
		Situaciones: []SituacionParticipacionBolsa{
			{
				Secuencia: 2, EstadoClave: "estado_operativo", EstadoVersion: 4, HuellaEstadoSHA256: huellaLlamamientoPrueba("c"),
				CausaClave: "cambio_gobernado", CausaVersion: 2, HuellaCausaSHA256: huellaLlamamientoPrueba("d"),
				DecisionRef: "decision:situacion:02" + string(rune('A'+numero)), HuellaDecisionSHA256: huellaLlamamientoPrueba("e"),
				Desde: cambio,
			},
			{
				Secuencia: 1, EstadoClave: "estado_inicial", EstadoVersion: 1, HuellaEstadoSHA256: huellaLlamamientoPrueba("f"),
				CausaClave: "constitucion_bolsa", CausaVersion: 1, HuellaCausaSHA256: huellaLlamamientoPrueba("1"),
				DecisionRef: "decision:situacion:01" + string(rune('A'+numero)), HuellaDecisionSHA256: huellaLlamamientoPrueba("2"),
				Desde: altaEn, Hasta: instantePtr(cambio),
			},
		},
	})
	if err != nil {
		t.Fatalf("participacion %d: %v", numero, err)
	}
	return participacion
}

func altaNecesidadLlamamientoPrueba(t *testing.T, bolsa BolsaConstituida) AltaNecesidadCobertura {
	t.Helper()
	huellaBolsa, err := bolsa.HuellaCanonicaSHA256()
	if err != nil {
		t.Fatalf("huella de bolsa para necesidad: %v", err)
	}
	return AltaNecesidadCobertura{
		NecesidadRef: "necesidad:01K0VS7P", Version: 2, BolsaRef: bolsa.BolsaRef,
		VersionBolsa: bolsa.Version, HuellaBolsaSHA256: huellaBolsa,
		CategoriaRef: bolsa.CategoriaRef, PuestoRef: "puesto:01K0VT10", UnidadRef: "unidad:01K0VT11",
		TipoCoberturaRef: "tipo_cobertura:01K0VT12", NumeroPuestos: 1,
		InicioPrevisto: instanteLlamamientoPrueba.Add(24 * time.Hour),
		FinPrevisto:    instanteLlamamientoPrueba.Add(60 * 24 * time.Hour),
		CreadaEn:       instanteLlamamientoPrueba.Add(-time.Hour),
		Requisitos: []RequisitoCobertura{
			{Clave: "zona", ValorRef: "zona:granada", Version: 2, HuellaSHA256: huellaLlamamientoPrueba("3")},
			{Clave: "jornada", ValorRef: "jornada:completa", Version: 5, HuellaSHA256: huellaLlamamientoPrueba("4")},
		},
	}
}

func altaNecesidadValida(t *testing.T) NecesidadCobertura {
	t.Helper()
	bolsa := bolsaLlamamientoPrueba(t)
	necesidad, err := NuevaNecesidadCobertura(altaNecesidadLlamamientoPrueba(t, bolsa))
	if err != nil {
		t.Fatal(err)
	}
	return necesidad
}

func politicaLlamamientoPrueba(t *testing.T) ReferenciaPoliticaLlamamiento {
	t.Helper()
	politica, err := NuevaReferenciaPoliticaLlamamiento(ReferenciaPoliticaLlamamiento{
		PoliticaRef: "politica:01K0VS6N", Clave: "llamamiento.reglamento_publicado",
		Version: 9, HuellaSHA256: huellaLlamamientoPrueba("5"),
		PublicadaEn:  instanteLlamamientoPrueba.Add(-72 * time.Hour),
		VigenteDesde: instanteLlamamientoPrueba.Add(-48 * time.Hour),
	})
	if err != nil {
		t.Fatalf("politica: %v", err)
	}
	return politica
}

func evaluacionLlamamientoPrueba(
	t *testing.T,
	instantanea InstantaneaOrdenBolsa,
	necesidad NecesidadCobertura,
	politica ReferenciaPoliticaLlamamiento,
	orden int,
	resultado ResultadoElegibilidadLlamamiento,
) EvaluacionParticipacionLlamamiento {
	t.Helper()
	entrada := instantanea.Entradas[orden-1]
	situacion, vigente := entrada.Participacion.SituacionVigenteEn(instantanea.ReferidaEn)
	if !vigente {
		t.Fatalf("sin situacion vigente para orden %d", orden)
	}
	huellaNecesidad, err := necesidad.HuellaCanonicaSHA256()
	if err != nil {
		t.Fatalf("huella de necesidad para orden %d: %v", orden, err)
	}
	sufijo := string(rune('A' + orden))
	return EvaluacionParticipacionLlamamiento{
		ParticipacionRef: entrada.Participacion.ParticipacionRef, SujetoRef: entrada.Participacion.SujetoRef,
		Orden: uint64(orden), SituacionSecuencia: situacion.Secuencia, EstadoClave: situacion.EstadoClave,
		EstadoVersion: situacion.EstadoVersion, HuellaEstadoSHA256: situacion.HuellaEstadoSHA256,
		NecesidadRef: necesidad.NecesidadRef, VersionNecesidad: necesidad.Version,
		HuellaNecesidadSHA256: huellaNecesidad,
		InstantaneaRef:        instantanea.InstantaneaRef, VersionInstantanea: instantanea.Version,
		HuellaInstantaneaSHA256: instantanea.HuellaContenidoSHA256,
		PoliticaRef:             politica.PoliticaRef, VersionPolitica: politica.Version, HuellaPoliticaSHA256: politica.HuellaSHA256,
		Resultado: resultado,
		Motivos: []MotivoEvaluacionLlamamiento{
			{Clave: "resultado_final", ReglaRef: "regla:seleccion:" + sufijo, VersionRegla: 3, HuellaReglaSHA256: huellaLlamamientoPrueba("6")},
			{Clave: "situacion_evaluada", ReglaRef: "regla:estado:" + sufijo, VersionRegla: 7, HuellaReglaSHA256: huellaLlamamientoPrueba("7")},
		},
		EntradaEvaluacionRef: "entrada:evaluacion:" + sufijo, HuellaEntradaSHA256: huellaLlamamientoPrueba("8"),
		ResultadoEvaluacionRef: "resultado:evaluacion:" + sufijo, HuellaResultadoSHA256: huellaLlamamientoPrueba("9"),
		EvaluadaEn: instantanea.GeneradaEn,
	}
}

func evaluacionesHastaPrimeraElegible(
	t *testing.T,
	instantanea InstantaneaOrdenBolsa,
	necesidad NecesidadCobertura,
	politica ReferenciaPoliticaLlamamiento,
) []EvaluacionParticipacionLlamamiento {
	t.Helper()
	return []EvaluacionParticipacionLlamamiento{
		evaluacionLlamamientoPrueba(t, instantanea, necesidad, politica, 1, ResultadoNoElegible),
		evaluacionLlamamientoPrueba(t, instantanea, necesidad, politica, 2, ResultadoNoElegible),
		evaluacionLlamamientoPrueba(t, instantanea, necesidad, politica, 3, ResultadoElegible),
	}
}

func propuestaLlamamientoPrueba(
	t *testing.T,
	bolsa BolsaConstituida,
	necesidad NecesidadCobertura,
	politica ReferenciaPoliticaLlamamiento,
	instantanea InstantaneaOrdenBolsa,
	evaluaciones []EvaluacionParticipacionLlamamiento,
) PropuestaLlamamiento {
	t.Helper()
	propuesta, err := ProponerPrimerLlamamiento(OrdenProponerPrimerLlamamiento{
		PropuestaRef: "propuesta:01K0VS9A7Q", Bolsa: bolsa, Necesidad: necesidad,
		Instantanea: instantanea, Politica: politica, Evaluaciones: evaluaciones,
		GeneradaEn: instanteLlamamientoPrueba.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("propuesta de prueba: %v", err)
	}
	return propuesta
}

func clonarEvaluacionesPrueba(origen []EvaluacionParticipacionLlamamiento) []EvaluacionParticipacionLlamamiento {
	clon := make([]EvaluacionParticipacionLlamamiento, len(origen))
	for indice := range origen {
		clon[indice] = origen[indice]
		clon[indice].Motivos = append([]MotivoEvaluacionLlamamiento(nil), origen[indice].Motivos...)
	}
	return clon
}

func invertirMotivos(motivos []MotivoEvaluacionLlamamiento) {
	for izquierda, derecha := 0, len(motivos)-1; izquierda < derecha; izquierda, derecha = izquierda+1, derecha-1 {
		motivos[izquierda], motivos[derecha] = motivos[derecha], motivos[izquierda]
	}
}

func conSujeto(participacion ParticipacionBolsa, sujetoRef string) ParticipacionBolsa {
	participacion.SujetoRef = sujetoRef
	return participacion
}

func instantePtr(instante time.Time) *time.Time {
	return &instante
}

func huellaLlamamientoPrueba(caracter string) string {
	return strings.Repeat(caracter, 64)
}
