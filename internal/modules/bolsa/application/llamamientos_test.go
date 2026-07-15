package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var instanteAplicacionLlamamientoPrueba = time.Date(2026, time.July, 15, 10, 30, 0, 123_456_000, time.UTC)

type resolutorLlamamientoFunc func(context.Context, string) ([]dominiovec.RecursoAutorizable, error)

func (f resolutorLlamamientoFunc) ResolverRecursosNecesidad(ctx context.Context, ref string) ([]dominiovec.RecursoAutorizable, error) {
	return f(ctx, ref)
}

type autorizadorLlamamientoFunc func(context.Context, dominiovec.SolicitudAutorizacion) (dominiovec.DecisionAutorizacion, error)

func (f autorizadorLlamamientoFunc) Exigir(ctx context.Context, s dominiovec.SolicitudAutorizacion) (dominiovec.DecisionAutorizacion, error) {
	return f(ctx, s)
}

type vinculadorLlamamientoFunc func(context.Context, dominiovec.SolicitudRevalidacionAutenticacionActorV1, dominiovec.ContextoActor) (dominiovec.VinculoAutenticacionActorV1, error)

func (f vinculadorLlamamientoFunc) Crear(
	ctx context.Context,
	s dominiovec.SolicitudRevalidacionAutenticacionActorV1,
	a dominiovec.ContextoActor,
) (dominiovec.VinculoAutenticacionActorV1, error) {
	return f(ctx, s, a)
}

type revalidadorActorAplicacionFunc func(context.Context, dominiovec.SolicitudRevalidacionAutenticacionActorV1) (dominiovec.AutenticacionRevalidadaV1, error)

func (f revalidadorActorAplicacionFunc) RevalidarAutenticacionActorV1(
	ctx context.Context,
	s dominiovec.SolicitudRevalidacionAutenticacionActorV1,
) (dominiovec.AutenticacionRevalidadaV1, error) {
	return f(ctx, s)
}

type fuenteLlamamientoFunc func(context.Context, string) ([]puertosbolsa.DatosAutoritativosLlamamiento, error)

func (f fuenteLlamamientoFunc) CargarDatosAutoritativosLlamamiento(ctx context.Context, ref string) ([]puertosbolsa.DatosAutoritativosLlamamiento, error) {
	return f(ctx, ref)
}

type motorLlamamientoFunc func(context.Context, puertosbolsa.SolicitudEvaluarParticipacionLlamamiento) (dominiobolsa.EvaluacionParticipacionLlamamiento, error)

func (f motorLlamamientoFunc) EvaluarParticipacion(ctx context.Context, s puertosbolsa.SolicitudEvaluarParticipacionLlamamiento) (dominiobolsa.EvaluacionParticipacionLlamamiento, error) {
	return f(ctx, s)
}

type transaccionLlamamientoFunc func(context.Context, dominiobolsa.PropuestaLlamamiento, puertosvec.EvidenciaUsoDecisionAutorizacion) error

func (f transaccionLlamamientoFunc) GuardarPropuestaLlamamiento(ctx context.Context, p dominiobolsa.PropuestaLlamamiento, e puertosvec.EvidenciaUsoDecisionAutorizacion) error {
	return f(ctx, p, e)
}

type relojFijoLlamamiento struct{ instante time.Time }

func (r *relojFijoLlamamiento) Ahora() time.Time { return r.instante }

type generadorSecuencialLlamamiento struct {
	mu            sync.Mutex
	instantaneas  int
	propuestas    int
	errorInstant  error
	errorProponer error
}

func (g *generadorSecuencialLlamamiento) NuevaReferenciaInstantaneaOrdenBolsa() (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.errorInstant != nil {
		return "", g.errorInstant
	}
	g.instantaneas++
	return fmt.Sprintf("instantanea:aplicacion:%06d", g.instantaneas), nil
}

func (g *generadorSecuencialLlamamiento) NuevaReferenciaPropuestaLlamamiento() (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.errorProponer != nil {
		return "", g.errorProponer
	}
	g.propuestas++
	return fmt.Sprintf("propuesta:aplicacion:%06d", g.propuestas), nil
}

type escenarioAplicacionLlamamiento struct {
	solicitud                puertosbolsa.SolicitudProponerLlamamiento
	datos                    puertosbolsa.DatosAutoritativosLlamamiento
	recurso                  dominiovec.RecursoAutorizable
	reloj                    *relojFijoLlamamiento
	generador                *generadorSecuencialLlamamiento
	secuencia                []string
	evaluadas                int
	persistencias            int
	mutarDecision            func(*dominiovec.DecisionAutorizacion)
	errorAutorizar           error
	decisionCero             bool
	resultados               []dominiobolsa.ResultadoElegibilidadLlamamiento
	recursos                 []dominiovec.RecursoAutorizable
	datosFuente              []puertosbolsa.DatosAutoritativosLlamamiento
	errorMotor               error
	cancelarMotor            context.CancelFunc
	mutarEvaluacion          func(*dominiobolsa.EvaluacionParticipacionLlamamiento)
	autenticacionEsperadaRef string
	sesionEsperadaRef        string
	personaEsperadaRef       string
}

func nuevoEscenarioAplicacionLlamamiento(t *testing.T) *escenarioAplicacionLlamamiento {
	t.Helper()
	datos := datosAutoritativosAplicacionPrueba(t, 3)
	actor := actorCanonicoAplicacionPrueba(t)
	recurso := dominiovec.RecursoAutorizable{
		Referencia: datos.Necesidad.NecesidadRef,
		ModuloID:   puertosbolsa.ModuloLlamamientos,
		Tipo:       puertosbolsa.TipoRecursoNecesidad,
		Ambitos: map[string]string{
			"categoria_ref": datos.Necesidad.CategoriaRef,
			"unidad_ref":    datos.Necesidad.UnidadRef,
		},
	}
	autenticacionRef := "aut_" + strings.Repeat("a", 22)
	sesionRef := "ses_" + strings.Repeat("s", 22)
	return &escenarioAplicacionLlamamiento{
		solicitud: puertosbolsa.SolicitudProponerLlamamiento{
			Actor: actor, PerfilActivoRef: actor.PerfilActivoRef,
			AutenticacionRef: autenticacionRef, SesionRef: sesionRef,
			NecesidadRef: datos.Necesidad.NecesidadRef, CorrelacionRef: "correlacion:llamamiento:0001",
		},
		datos: datos, recurso: recurso, reloj: &relojFijoLlamamiento{instante: instanteAplicacionLlamamientoPrueba},
		generador: &generadorSecuencialLlamamiento{},
		resultados: []dominiobolsa.ResultadoElegibilidadLlamamiento{
			dominiobolsa.ResultadoNoElegible, dominiobolsa.ResultadoElegible, dominiobolsa.ResultadoElegible,
		},
		recursos:                 []dominiovec.RecursoAutorizable{recurso},
		datosFuente:              []puertosbolsa.DatosAutoritativosLlamamiento{datos},
		autenticacionEsperadaRef: autenticacionRef, sesionEsperadaRef: sesionRef,
		personaEsperadaRef: actor.PersonaRef,
	}
}

func (e *escenarioAplicacionLlamamiento) servicio(t *testing.T) *ServicioLlamamientos {
	t.Helper()
	resolutor := resolutorLlamamientoFunc(func(_ context.Context, ref string) ([]dominiovec.RecursoAutorizable, error) {
		e.secuencia = append(e.secuencia, "recurso")
		if ref != e.solicitud.NecesidadRef {
			t.Fatalf("referencia resuelta no exacta: %q", ref)
		}
		return append([]dominiovec.RecursoAutorizable(nil), e.recursos...), nil
	})
	vinculador := vinculadorLlamamientoFunc(func(
		_ context.Context,
		solicitud dominiovec.SolicitudRevalidacionAutenticacionActorV1,
		actor dominiovec.ContextoActor,
	) (dominiovec.VinculoAutenticacionActorV1, error) {
		e.secuencia = append(e.secuencia, "vinculo")
		if solicitud.AutenticacionRef != e.autenticacionEsperadaRef || solicitud.SesionRef != e.sesionEsperadaRef ||
			actor.PersonaRef != e.personaEsperadaRef {
			return dominiovec.VinculoAutenticacionActorV1{}, dominiovec.ErrVinculoAutenticacionActorInvalido
		}
		return vinculoActorAplicacionPrueba(t, actor, solicitud)
	})
	autorizador := autorizadorLlamamientoFunc(func(_ context.Context, solicitud dominiovec.SolicitudAutorizacion) (dominiovec.DecisionAutorizacion, error) {
		e.secuencia = append(e.secuencia, "autorizar")
		if solicitud.Accion != puertosbolsa.AccionProponerLlamamiento ||
			solicitud.Finalidad != puertosbolsa.FinalidadProponerLlamamiento ||
			solicitud.Recurso.Referencia != e.solicitud.NecesidadRef ||
			solicitud.PerfilActivoRef != e.solicitud.PerfilActivoRef ||
			len(solicitud.Recurso.Atributos) != 0 || len(solicitud.Recurso.Ambitos) != 2 {
			t.Fatalf("solicitud al PDP no minima o no exacta: %+v", solicitud)
		}
		if e.errorAutorizar != nil {
			return dominiovec.DecisionAutorizacion{}, e.errorAutorizar
		}
		if e.decisionCero {
			return dominiovec.DecisionAutorizacion{}, nil
		}
		decision := decisionAplicacionLlamamientoPrueba(t, solicitud, e.reloj.instante)
		if e.mutarDecision != nil {
			e.mutarDecision(&decision)
		}
		return decision, nil
	})
	fuente := fuenteLlamamientoFunc(func(_ context.Context, ref string) ([]puertosbolsa.DatosAutoritativosLlamamiento, error) {
		e.secuencia = append(e.secuencia, "datos")
		if ref != e.solicitud.NecesidadRef {
			t.Fatalf("carga no exacta: %q", ref)
		}
		return append([]puertosbolsa.DatosAutoritativosLlamamiento(nil), e.datosFuente...), nil
	})
	motor := motorLlamamientoFunc(func(_ context.Context, solicitud puertosbolsa.SolicitudEvaluarParticipacionLlamamiento) (dominiobolsa.EvaluacionParticipacionLlamamiento, error) {
		e.secuencia = append(e.secuencia, fmt.Sprintf("evaluar:%d", solicitud.Entrada.Orden))
		e.evaluadas++
		if e.cancelarMotor != nil {
			e.cancelarMotor()
		}
		if e.errorMotor != nil {
			return dominiobolsa.EvaluacionParticipacionLlamamiento{}, e.errorMotor
		}
		evaluacion := evaluacionDesdeSolicitudAplicacionPrueba(t, solicitud, e.resultados[e.evaluadas-1])
		if e.mutarEvaluacion != nil {
			e.mutarEvaluacion(&evaluacion)
		}
		return evaluacion, nil
	})
	transaccion := transaccionLlamamientoFunc(func(_ context.Context, propuesta dominiobolsa.PropuestaLlamamiento, evidencia puertosvec.EvidenciaUsoDecisionAutorizacion) error {
		e.secuencia = append(e.secuencia, "persistir")
		e.persistencias++
		if propuesta.Validar() != nil || evidencia.ValidarEn(propuesta.GeneradaEn) != nil {
			t.Fatal("la transaccion recibio efecto o evidencia invalidos")
		}
		datos, err := evidencia.Datos()
		if err != nil || datos.Decision.RecursoRef != propuesta.NecesidadRef ||
			!datos.VerificadaEn.Equal(propuesta.GeneradaEn) {
			t.Fatalf("evidencia no ligada al efecto: %+v / %v", datos, err)
		}
		return nil
	})
	servicio, err := NuevoServicioLlamamientos(resolutor, vinculador, autorizador, fuente, motor, e.reloj, e.generador, transaccion)
	if err != nil {
		t.Fatalf("construir servicio: %v", err)
	}
	return servicio
}

func TestServicioLlamamientosResuelveAutorizaEvaluaPrefijoYPersisteAtomicamente(t *testing.T) {
	escenario := nuevoEscenarioAplicacionLlamamiento(t)
	propuesta, err := escenario.servicio(t).ProponerPrimerLlamamiento(context.Background(), escenario.solicitud)
	if err != nil {
		t.Fatalf("proponer: %v", err)
	}
	if propuesta.Validar() != nil || propuesta.OrdenSeleccionado != 2 || len(propuesta.Evaluaciones) != 2 ||
		escenario.evaluadas != 2 || escenario.persistencias != 1 {
		t.Fatalf("resultado inesperado: propuesta=%+v evaluadas=%d persistencias=%d", propuesta, escenario.evaluadas, escenario.persistencias)
	}
	esperada := []string{"recurso", "vinculo", "autorizar", "datos", "evaluar:1", "evaluar:2", "persistir"}
	if !reflect.DeepEqual(escenario.secuencia, esperada) {
		t.Fatalf("orden hexagonal incorrecto: %v", escenario.secuencia)
	}
	// La respuesta tambien es defensiva: mutar la fuente tras el caso de uso no
	// modifica la propuesta ya construida.
	escenario.datos.Entradas[0].Participacion.Situaciones[0].EstadoClave = "mutado"
	if propuesta.Evaluaciones[0].EstadoClave == "mutado" {
		t.Fatal("la propuesta comparte estado con la fuente")
	}
}

func TestServicioLlamamientosDeniegaRecursoAusenteOAmbiguoAntesDelPDP(t *testing.T) {
	for nombre, recursos := range map[string][]dominiovec.RecursoAutorizable{
		"ausente": nil,
		"ambiguo": {{Referencia: "necesidad:una"}, {Referencia: "necesidad:dos"}},
	} {
		t.Run(nombre, func(t *testing.T) {
			escenario := nuevoEscenarioAplicacionLlamamiento(t)
			escenario.recursos = recursos
			_, err := escenario.servicio(t).ProponerPrimerLlamamiento(context.Background(), escenario.solicitud)
			if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || escenario.persistencias != 0 ||
				len(escenario.secuencia) != 1 || escenario.secuencia[0] != "recurso" {
				t.Fatalf("no fallo cerrado antes del PDP: secuencia=%v error=%v", escenario.secuencia, err)
			}
		})
	}
}

func TestServicioLlamamientosDeniegaCamposYObligacionesNoSoportadosAntesDeLeerDatos(t *testing.T) {
	casos := map[string]func(*dominiovec.DecisionAutorizacion){
		"campo":      func(d *dominiovec.DecisionAutorizacion) { d.CamposPermitidos = []string{"sujeto_ref"} },
		"obligacion": func(d *dominiovec.DecisionAutorizacion) { d.Obligaciones = []string{"doble_revision"} },
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			escenario := nuevoEscenarioAplicacionLlamamiento(t)
			escenario.mutarDecision = mutar
			_, err := escenario.servicio(t).ProponerPrimerLlamamiento(context.Background(), escenario.solicitud)
			if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || escenario.persistencias != 0 ||
				!reflect.DeepEqual(escenario.secuencia, []string{"recurso", "vinculo", "autorizar"}) {
				t.Fatalf("capacidad parcial no denegada antes de datos: %v / %v", escenario.secuencia, err)
			}
		})
	}
}

func TestServicioLlamamientosDeniegaSinConcesionPositivaYValida(t *testing.T) {
	t.Run("pdp_deniega", func(t *testing.T) {
		escenario := nuevoEscenarioAplicacionLlamamiento(t)
		escenario.errorAutorizar = dominiovec.ErrAutorizacionDenegada
		_, err := escenario.servicio(t).ProponerPrimerLlamamiento(context.Background(), escenario.solicitud)
		if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || escenario.persistencias != 0 ||
			!reflect.DeepEqual(escenario.secuencia, []string{"recurso", "vinculo", "autorizar"}) {
			t.Fatalf("denegacion del PDP no prevalecio: %v / %v", escenario.secuencia, err)
		}
	})
	t.Run("decision_cero", func(t *testing.T) {
		escenario := nuevoEscenarioAplicacionLlamamiento(t)
		escenario.decisionCero = true
		_, err := escenario.servicio(t).ProponerPrimerLlamamiento(context.Background(), escenario.solicitud)
		if !errors.Is(err, dominiovec.ErrDecisionAutorizacionInvalida) || escenario.persistencias != 0 ||
			!reflect.DeepEqual(escenario.secuencia, []string{"recurso", "vinculo", "autorizar"}) {
			t.Fatalf("decision cero habilito datos: %v / %v", escenario.secuencia, err)
		}
	})
}

func TestServicioLlamamientosDeniegaFuenteAusenteOAmbiguaSinEvaluar(t *testing.T) {
	for nombre, datos := range map[string][]puertosbolsa.DatosAutoritativosLlamamiento{
		"ausente": nil,
		"ambigua": {{}, {}},
	} {
		t.Run(nombre, func(t *testing.T) {
			escenario := nuevoEscenarioAplicacionLlamamiento(t)
			escenario.datosFuente = datos
			_, err := escenario.servicio(t).ProponerPrimerLlamamiento(context.Background(), escenario.solicitud)
			if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || escenario.evaluadas != 0 || escenario.persistencias != 0 ||
				!reflect.DeepEqual(escenario.secuencia, []string{"recurso", "vinculo", "autorizar", "datos"}) {
				t.Fatalf("fuente no fallo cerrada: %v / %v", escenario.secuencia, err)
			}
		})
	}
}

func TestServicioLlamamientosSinElegiblesConservaDenegacionYSinEfecto(t *testing.T) {
	escenario := nuevoEscenarioAplicacionLlamamiento(t)
	escenario.resultados = []dominiobolsa.ResultadoElegibilidadLlamamiento{
		dominiobolsa.ResultadoNoElegible, dominiobolsa.ResultadoNoElegible, dominiobolsa.ResultadoNoElegible,
	}
	_, err := escenario.servicio(t).ProponerPrimerLlamamiento(context.Background(), escenario.solicitud)
	if !errors.Is(err, dominiobolsa.ErrSinParticipacionElegible) || escenario.evaluadas != 3 || escenario.persistencias != 0 {
		t.Fatalf("cero elegibles genero efecto: evaluadas=%d persistencias=%d error=%v", escenario.evaluadas, escenario.persistencias, err)
	}
}

func TestServicioLlamamientosCancelacionYErrorDeMotorNoPersisten(t *testing.T) {
	t.Run("cancelacion", func(t *testing.T) {
		escenario := nuevoEscenarioAplicacionLlamamiento(t)
		ctx, cancelar := context.WithCancel(context.Background())
		escenario.cancelarMotor = cancelar
		_, err := escenario.servicio(t).ProponerPrimerLlamamiento(ctx, escenario.solicitud)
		if !errors.Is(err, context.Canceled) || !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || escenario.persistencias != 0 {
			t.Fatalf("cancelacion no cerro el caso: %v", err)
		}
	})
	t.Run("error", func(t *testing.T) {
		escenario := nuevoEscenarioAplicacionLlamamiento(t)
		escenario.errorMotor = errors.New("motor caido")
		_, err := escenario.servicio(t).ProponerPrimerLlamamiento(context.Background(), escenario.solicitud)
		if !errors.Is(err, puertosbolsa.ErrMotorElegibilidadNoDisponible) || escenario.persistencias != 0 {
			t.Fatalf("error de motor genero efecto: %v", err)
		}
	})
}

func TestServicioLlamamientosDeniegaEvaluacionNoLigadaYFallosDeRelojOReferencias(t *testing.T) {
	t.Run("evaluacion_cruzada", func(t *testing.T) {
		escenario := nuevoEscenarioAplicacionLlamamiento(t)
		escenario.mutarEvaluacion = func(e *dominiobolsa.EvaluacionParticipacionLlamamiento) {
			e.SujetoRef = "sujeto:opaco:ajeno"
		}
		_, err := escenario.servicio(t).ProponerPrimerLlamamiento(context.Background(), escenario.solicitud)
		if !errors.Is(err, puertosbolsa.ErrEvaluacionMotorNoConfiable) || escenario.persistencias != 0 {
			t.Fatalf("evaluacion cruzada genero efecto: %v", err)
		}
	})
	t.Run("reloj_cero", func(t *testing.T) {
		escenario := nuevoEscenarioAplicacionLlamamiento(t)
		escenario.reloj.instante = time.Time{}
		_, err := escenario.servicio(t).ProponerPrimerLlamamiento(context.Background(), escenario.solicitud)
		if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || escenario.persistencias != 0 ||
			!reflect.DeepEqual(escenario.secuencia, []string{"recurso", "vinculo"}) {
			t.Fatalf("reloj cero alcanzo efecto: %v / %v", escenario.secuencia, err)
		}
	})
	t.Run("referencia_instantanea", func(t *testing.T) {
		escenario := nuevoEscenarioAplicacionLlamamiento(t)
		escenario.generador.errorInstant = errors.New("sin entropia")
		_, err := escenario.servicio(t).ProponerPrimerLlamamiento(context.Background(), escenario.solicitud)
		if !errors.Is(err, puertosbolsa.ErrGeneracionReferenciaLlamamiento) || escenario.evaluadas != 0 || escenario.persistencias != 0 {
			t.Fatalf("fallo de referencia genero efecto: %v", err)
		}
	})
	t.Run("referencia_propuesta", func(t *testing.T) {
		escenario := nuevoEscenarioAplicacionLlamamiento(t)
		escenario.generador.errorProponer = errors.New("sin entropia")
		_, err := escenario.servicio(t).ProponerPrimerLlamamiento(context.Background(), escenario.solicitud)
		if !errors.Is(err, puertosbolsa.ErrGeneracionReferenciaLlamamiento) || escenario.evaluadas != 2 || escenario.persistencias != 0 {
			t.Fatalf("fallo de referencia final genero efecto: %v", err)
		}
	})
}

func TestServicioLlamamientosDeniegaPerfilImplicitoPIIYContextoCanceladoAntesDeEfectos(t *testing.T) {
	casos := map[string]func(*puertosbolsa.SolicitudProponerLlamamiento){
		"perfil_vacio":    func(s *puertosbolsa.SolicitudProponerLlamamiento) { s.PerfilActivoRef = "" },
		"perfil_distinto": func(s *puertosbolsa.SolicitudProponerLlamamiento) { s.PerfilActivoRef = "prf_XXXXXXXXXXXXXXXXXXXXXX" },
		"dni_sintetico":   func(s *puertosbolsa.SolicitudProponerLlamamiento) { s.NecesidadRef = "necesidad:00000000A" },
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			escenario := nuevoEscenarioAplicacionLlamamiento(t)
			mutar(&escenario.solicitud)
			_, err := escenario.servicio(t).ProponerPrimerLlamamiento(context.Background(), escenario.solicitud)
			if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || len(escenario.secuencia) != 0 || escenario.persistencias != 0 {
				t.Fatalf("entrada no explicita produjo actividad: %v / %v", escenario.secuencia, err)
			}
		})
	}
	escenario := nuevoEscenarioAplicacionLlamamiento(t)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	_, err := escenario.servicio(t).ProponerPrimerLlamamiento(ctx, escenario.solicitud)
	if !errors.Is(err, context.Canceled) || len(escenario.secuencia) != 0 || escenario.persistencias != 0 {
		t.Fatalf("contexto cancelado produjo actividad: %v / %v", escenario.secuencia, err)
	}
	escenario = nuevoEscenarioAplicacionLlamamiento(t)
	_, err = escenario.servicio(t).ProponerPrimerLlamamiento(nil, escenario.solicitud)
	if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || len(escenario.secuencia) != 0 || escenario.persistencias != 0 {
		t.Fatalf("contexto nulo produjo actividad: %v / %v", escenario.secuencia, err)
	}
}

func TestServicioLlamamientosNoPermiteFabricarSesionNiTitularidad(t *testing.T) {
	t.Run("sesion_ajena", func(t *testing.T) {
		escenario := nuevoEscenarioAplicacionLlamamiento(t)
		escenario.solicitud.SesionRef = "ses_" + strings.Repeat("z", 22)
		_, err := escenario.servicio(t).ProponerPrimerLlamamiento(context.Background(), escenario.solicitud)
		if !errors.Is(err, dominiovec.ErrVinculoAutenticacionActorInvalido) || escenario.persistencias != 0 ||
			!reflect.DeepEqual(escenario.secuencia, []string{"recurso", "vinculo"}) {
			t.Fatalf("sesion declarada por llamante alcanzo el PDP: %v / %v", escenario.secuencia, err)
		}
	})
	t.Run("actor_ajeno", func(t *testing.T) {
		escenario := nuevoEscenarioAplicacionLlamamiento(t)
		ajeno := actorCanonicoAplicacionAlternativoPrueba(t)
		escenario.solicitud.Actor = ajeno
		escenario.solicitud.PerfilActivoRef = ajeno.PerfilActivoRef
		_, err := escenario.servicio(t).ProponerPrimerLlamamiento(context.Background(), escenario.solicitud)
		if !errors.Is(err, dominiovec.ErrVinculoAutenticacionActorInvalido) || escenario.persistencias != 0 ||
			!reflect.DeepEqual(escenario.secuencia, []string{"recurso", "vinculo"}) {
			t.Fatalf("titularidad declarada por llamante alcanzo el PDP: %v / %v", escenario.secuencia, err)
		}
	})
}

func TestNuevoServicioLlamamientosRechazaDependenciasNulasYTipadasNulas(t *testing.T) {
	resolutor := resolutorLlamamientoFunc(func(context.Context, string) ([]dominiovec.RecursoAutorizable, error) { return nil, nil })
	vinculador := vinculadorLlamamientoFunc(func(context.Context, dominiovec.SolicitudRevalidacionAutenticacionActorV1, dominiovec.ContextoActor) (dominiovec.VinculoAutenticacionActorV1, error) {
		return dominiovec.VinculoAutenticacionActorV1{}, nil
	})
	autorizador := autorizadorLlamamientoFunc(func(context.Context, dominiovec.SolicitudAutorizacion) (dominiovec.DecisionAutorizacion, error) {
		return dominiovec.DecisionAutorizacion{}, nil
	})
	fuente := fuenteLlamamientoFunc(func(context.Context, string) ([]puertosbolsa.DatosAutoritativosLlamamiento, error) { return nil, nil })
	motor := motorLlamamientoFunc(func(context.Context, puertosbolsa.SolicitudEvaluarParticipacionLlamamiento) (dominiobolsa.EvaluacionParticipacionLlamamiento, error) {
		return dominiobolsa.EvaluacionParticipacionLlamamiento{}, nil
	})
	reloj := &relojFijoLlamamiento{instante: instanteAplicacionLlamamientoPrueba}
	generador := &generadorSecuencialLlamamiento{}
	transaccion := transaccionLlamamientoFunc(func(context.Context, dominiobolsa.PropuestaLlamamiento, puertosvec.EvidenciaUsoDecisionAutorizacion) error {
		return nil
	})
	if servicio, err := NuevoServicioLlamamientos(resolutor, vinculador, autorizador, fuente, motor, reloj, generador, transaccion); err != nil || servicio == nil {
		t.Fatalf("precondicion de dependencias validas: %v", err)
	}

	var resolutorNulo *resolutorPunteroNulo
	var vinculadorNulo vinculadorLlamamientoFunc
	var autorizadorNulo autorizadorLlamamientoFunc
	var fuenteNula fuenteLlamamientoFunc
	var motorNulo motorLlamamientoFunc
	var relojNulo *relojFijoLlamamiento
	var generadorNulo *generadorSecuencialLlamamiento
	var transaccionNula transaccionLlamamientoFunc
	type dependencias struct {
		resolutor   puertosbolsa.ResolutorRecursoNecesidad
		vinculador  puertosbolsa.CreadorVinculoAutenticacionActor
		autorizador puertosvec.Autorizador
		fuente      puertosbolsa.FuenteDatosLlamamiento
		motor       puertosbolsa.MotorElegibilidadLlamamiento
		reloj       puertosbolsa.RelojLlamamientos
		generador   puertosbolsa.GeneradorReferenciasLlamamiento
		transaccion puertosbolsa.TransaccionPropuestasLlamamiento
	}
	base := dependencias{resolutor, vinculador, autorizador, fuente, motor, reloj, generador, transaccion}
	casos := map[string]func(*dependencias){
		"resolutor":   func(d *dependencias) { d.resolutor = resolutorNulo },
		"vinculador":  func(d *dependencias) { d.vinculador = vinculadorNulo },
		"autorizador": func(d *dependencias) { d.autorizador = autorizadorNulo },
		"fuente":      func(d *dependencias) { d.fuente = fuenteNula },
		"motor":       func(d *dependencias) { d.motor = motorNulo },
		"reloj":       func(d *dependencias) { d.reloj = relojNulo },
		"generador":   func(d *dependencias) { d.generador = generadorNulo },
		"transaccion": func(d *dependencias) { d.transaccion = transaccionNula },
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			actual := base
			mutar(&actual)
			_, err := NuevoServicioLlamamientos(
				actual.resolutor, actual.vinculador, actual.autorizador, actual.fuente, actual.motor,
				actual.reloj, actual.generador, actual.transaccion,
			)
			if !errors.Is(err, puertosbolsa.ErrSolicitudPropuestaLlamamientoInvalida) {
				t.Fatalf("dependencia tipada nula admitida: %v", err)
			}
		})
	}
}

type resolutorPunteroNulo struct{}

func (*resolutorPunteroNulo) ResolverRecursosNecesidad(context.Context, string) ([]dominiovec.RecursoAutorizable, error) {
	panic("no debe invocarse")
}

func actorCanonicoAplicacionPrueba(t *testing.T) dominiovec.ContextoActor {
	t.Helper()
	token := func(caracter string) string { return strings.Repeat(caracter, 22) }
	cuenta := dominiovec.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_" + token("c"), Metodo: dominiovec.AuthMethodCertificate, Garantia: dominiovec.AuthAssuranceHigh,
	}
	instantanea := dominiovec.InstantaneaContextoActor{
		VinculoRef: "vca_" + token("v"), VinculoVersion: 1, CuentaRef: cuenta.CuentaRef,
		PersonaRef: "per_" + token("p"), PersonaVersion: 1, PerfilActivoRef: "prf_" + token("r"), PerfilVersion: 1,
		Estado:       dominiovec.EstadoVinculoContextoActorActivo,
		VigenteDesde: instanteAplicacionLlamamientoPrueba.Add(-24 * time.Hour),
		VigenteHasta: instanteAplicacionLlamamientoPrueba.Add(24 * time.Hour),
	}
	actor, err := dominiovec.NuevoContextoActor(cuenta, instantanea, instanteAplicacionLlamamientoPrueba.Add(-time.Minute))
	if err != nil {
		t.Fatalf("actor canonico: %v", err)
	}
	return actor
}

func actorCanonicoAplicacionAlternativoPrueba(t *testing.T) dominiovec.ContextoActor {
	t.Helper()
	token := func(caracter string) string { return strings.Repeat(caracter, 22) }
	cuenta := dominiovec.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_" + token("x"), Metodo: dominiovec.AuthMethodCertificate, Garantia: dominiovec.AuthAssuranceHigh,
	}
	instantanea := dominiovec.InstantaneaContextoActor{
		VinculoRef: "vca_" + token("y"), VinculoVersion: 1, CuentaRef: cuenta.CuentaRef,
		PersonaRef: "per_" + token("z"), PersonaVersion: 1, PerfilActivoRef: "prf_" + token("q"), PerfilVersion: 1,
		Estado:       dominiovec.EstadoVinculoContextoActorActivo,
		VigenteDesde: instanteAplicacionLlamamientoPrueba.Add(-24 * time.Hour),
		VigenteHasta: instanteAplicacionLlamamientoPrueba.Add(24 * time.Hour),
	}
	actor, err := dominiovec.NuevoContextoActor(cuenta, instantanea, instanteAplicacionLlamamientoPrueba.Add(-time.Minute))
	if err != nil {
		t.Fatalf("actor alternativo: %v", err)
	}
	return actor
}

func vinculoActorAplicacionPrueba(
	t *testing.T,
	actor dominiovec.ContextoActor,
	solicitud dominiovec.SolicitudRevalidacionAutenticacionActorV1,
) (dominiovec.VinculoAutenticacionActorV1, error) {
	t.Helper()
	token := func(caracter string) string { return strings.Repeat(caracter, 22) }
	autenticacion := dominiovec.AutenticacionRevalidadaV1{
		AutenticacionRef: solicitud.AutenticacionRef, AutenticacionHuellaSHA256: huellaAplicacionLlamamiento('a'),
		AsercionRef: "ase_" + token("e"), SesionRef: solicitud.SesionRef,
		ControlSesionRef: "cse_" + token("c"), ControlSesionRevision: 1,
		ControlSesionHuellaSHA256: huellaAplicacionLlamamiento('b'),
		CuentaRef:                 actor.Instantanea.CuentaRef, CuentaOrdinariaRef: actor.Instantanea.CuentaRef,
		Superficie:      dominiovec.SuperficieAutenticacionExternaPersonalV1,
		MetodoObservado: actor.Principal.AuthMethod, GarantiaObservada: actor.Principal.AuthAssurance,
		PoliticaGarantiaRef: "pga_" + token("g"), PoliticaGarantiaHuellaSHA256: huellaAplicacionLlamamiento('c'),
		SesionEmitidaEn:           instanteAplicacionLlamamientoPrueba.Add(-time.Hour),
		AutenticacionVerificadaEn: instanteAplicacionLlamamientoPrueba.Add(-2 * time.Hour),
		SesionRevalidadaEn:        instanteAplicacionLlamamientoPrueba.Add(-90 * time.Second),
		SesionValidaHasta:         instanteAplicacionLlamamientoPrueba.Add(time.Hour),
	}
	return dominiovec.CrearVinculoAutenticacionActorV1(
		context.Background(),
		revalidadorActorAplicacionFunc(func(_ context.Context, recibida dominiovec.SolicitudRevalidacionAutenticacionActorV1) (dominiovec.AutenticacionRevalidadaV1, error) {
			if recibida != solicitud {
				return dominiovec.AutenticacionRevalidadaV1{}, dominiovec.ErrAutenticacionRevalidadaInvalida
			}
			return autenticacion, nil
		}),
		solicitud,
		actor,
		instanteAplicacionLlamamientoPrueba,
	)
}

func datosAutoritativosAplicacionPrueba(t *testing.T, total int) puertosbolsa.DatosAutoritativosLlamamiento {
	t.Helper()
	bolsa, err := dominiobolsa.NuevaBolsaConstituida(dominiobolsa.AltaBolsaConstituida{
		BolsaRef: "bolsa:aplicacion:0001", Version: 2, ProcesoRef: "proceso:aplicacion:0001",
		CategoriaRef: "categoria:aplicacion:0001", ListadoDefinitivoRef: "listado:aplicacion:0001",
		VersionListado: 3, HuellaListadoSHA256: huellaAplicacionLlamamiento('a'),
		ResolucionConstitucionRef: "resolucion:aplicacion:0001", HuellaResolucionSHA256: huellaAplicacionLlamamiento('b'),
		ConstituidaEn: instanteAplicacionLlamamientoPrueba.Add(-48 * time.Hour),
		VigenteDesde:  instanteAplicacionLlamamientoPrueba.Add(-24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("bolsa: %v", err)
	}
	huellaBolsa, _ := bolsa.HuellaCanonicaSHA256()
	necesidad, err := dominiobolsa.NuevaNecesidadCobertura(dominiobolsa.AltaNecesidadCobertura{
		NecesidadRef: "necesidad:aplicacion:0001", Version: 2, BolsaRef: bolsa.BolsaRef,
		VersionBolsa: bolsa.Version, HuellaBolsaSHA256: huellaBolsa, CategoriaRef: bolsa.CategoriaRef,
		PuestoRef: "puesto:aplicacion:0001", UnidadRef: "unidad:aplicacion:0001",
		TipoCoberturaRef: "tipo:aplicacion:0001", NumeroPuestos: 1,
		InicioPrevisto: instanteAplicacionLlamamientoPrueba.Add(24 * time.Hour),
		FinPrevisto:    instanteAplicacionLlamamientoPrueba.Add(30 * 24 * time.Hour),
		CreadaEn:       instanteAplicacionLlamamientoPrueba.Add(-time.Hour),
		Requisitos: []dominiobolsa.RequisitoCobertura{{
			Clave: "requisito_gobernado", ValorRef: "valor:aplicacion:0001", Version: 4, HuellaSHA256: huellaAplicacionLlamamiento('c'),
		}},
	})
	if err != nil {
		t.Fatalf("necesidad: %v", err)
	}
	politica, err := dominiobolsa.NuevaReferenciaPoliticaLlamamiento(dominiobolsa.ReferenciaPoliticaLlamamiento{
		PoliticaRef: "politica:aplicacion:0001", Clave: "politica_llamamiento_publicada", Version: 7,
		HuellaSHA256: huellaAplicacionLlamamiento('d'), PublicadaEn: instanteAplicacionLlamamientoPrueba.Add(-72 * time.Hour),
		VigenteDesde: instanteAplicacionLlamamientoPrueba.Add(-48 * time.Hour),
	})
	if err != nil {
		t.Fatalf("politica: %v", err)
	}
	entradas := make([]dominiobolsa.EntradaOrdenBolsa, total)
	for indice := 0; indice < total; indice++ {
		orden := indice + 1
		participacion, err := dominiobolsa.NuevaParticipacionBolsa(dominiobolsa.AltaParticipacionBolsa{
			ParticipacionRef: fmt.Sprintf("participacion:aplicacion:%04d", orden), BolsaRef: bolsa.BolsaRef,
			SujetoRef: fmt.Sprintf("sujeto:opaco:%04d", orden), Version: 1,
			AltaEn: instanteAplicacionLlamamientoPrueba.Add(-12 * time.Hour),
			Situaciones: []dominiobolsa.SituacionParticipacionBolsa{{
				Secuencia: 1, EstadoClave: fmt.Sprintf("estado_gobernado_%d", orden), EstadoVersion: uint64(orden),
				HuellaEstadoSHA256: huellaAplicacionLlamamiento('e'), CausaClave: "causa_gobernada",
				CausaVersion: 1, HuellaCausaSHA256: huellaAplicacionLlamamiento('8'),
				DecisionRef: fmt.Sprintf("decision:situacion:%04d", orden), HuellaDecisionSHA256: huellaAplicacionLlamamiento('9'),
				Desde: instanteAplicacionLlamamientoPrueba.Add(-12 * time.Hour),
			}},
		})
		if err != nil {
			t.Fatalf("participacion %d: %v", orden, err)
		}
		entradas[indice] = dominiobolsa.EntradaOrdenBolsa{Orden: uint64(orden), Participacion: participacion}
	}
	return puertosbolsa.DatosAutoritativosLlamamiento{Bolsa: bolsa, Necesidad: necesidad, Politica: politica, Entradas: entradas}
}

func evaluacionDesdeSolicitudAplicacionPrueba(
	t *testing.T,
	s puertosbolsa.SolicitudEvaluarParticipacionLlamamiento,
	resultado dominiobolsa.ResultadoElegibilidadLlamamiento,
) dominiobolsa.EvaluacionParticipacionLlamamiento {
	t.Helper()
	situacion, existe := s.Entrada.Participacion.SituacionVigenteEn(s.InstanteReferencia)
	if !existe {
		t.Fatal("situacion no vigente")
	}
	huellaNecesidad, _ := s.Necesidad.HuellaCanonicaSHA256()
	sufijo := fmt.Sprintf("%04d", s.Entrada.Orden)
	return dominiobolsa.EvaluacionParticipacionLlamamiento{
		ParticipacionRef: s.Entrada.Participacion.ParticipacionRef, SujetoRef: s.Entrada.Participacion.SujetoRef,
		Orden: s.Entrada.Orden, SituacionSecuencia: situacion.Secuencia, EstadoClave: situacion.EstadoClave,
		EstadoVersion: situacion.EstadoVersion, HuellaEstadoSHA256: situacion.HuellaEstadoSHA256,
		NecesidadRef: s.Necesidad.NecesidadRef, VersionNecesidad: s.Necesidad.Version, HuellaNecesidadSHA256: huellaNecesidad,
		InstantaneaRef: s.InstantaneaRef, VersionInstantanea: s.VersionInstantanea, HuellaInstantaneaSHA256: s.HuellaInstantaneaSHA256,
		PoliticaRef: s.Politica.PoliticaRef, VersionPolitica: s.Politica.Version, HuellaPoliticaSHA256: s.Politica.HuellaSHA256,
		Resultado: resultado,
		Motivos: []dominiobolsa.MotivoEvaluacionLlamamiento{{
			Clave: "resultado_motor", ReglaRef: "regla:aplicacion:" + sufijo, VersionRegla: 5, HuellaReglaSHA256: huellaAplicacionLlamamiento('f'),
		}},
		EntradaEvaluacionRef: "recibo:entrada:" + sufijo, HuellaEntradaSHA256: huellaAplicacionLlamamiento('1'),
		ResultadoEvaluacionRef: "recibo:resultado:" + sufijo, HuellaResultadoSHA256: huellaAplicacionLlamamiento('2'),
		EvaluadaEn: s.EvaluadaEn,
	}
}

func decisionAplicacionLlamamientoPrueba(
	t *testing.T,
	s dominiovec.SolicitudAutorizacion,
	instante time.Time,
) dominiovec.DecisionAutorizacion {
	t.Helper()
	referenciaPolitica := "politica:seguridad:llamamientos:v1"
	huellas := map[string]string{referenciaPolitica: huellaAplicacionLlamamiento('3')}
	huellaCatalogo, err := dominiovec.HuellaEvidenciasCatalogoPoliticasAutorizacion([]string{referenciaPolitica}, huellas)
	if err != nil {
		t.Fatalf("catalogo: %v", err)
	}
	huellaContexto, err := s.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatalf("contexto recurso: %v", err)
	}
	decision := dominiovec.DecisionAutorizacion{
		DecisionRef: "decision:autorizacion:llamamiento:0001", Concedida: true, Codigo: "concedida",
		PrincipalID: s.Principal.ID, PerfilActivoRef: s.PerfilActivoRef, Accion: s.Accion,
		RecursoRef: s.Recurso.Referencia, ModuloID: s.Recurso.ModuloID, TipoRecurso: s.Recurso.Tipo,
		ContextoRecursoHuellaSHA256: huellaContexto, Finalidad: s.Finalidad, CorrelacionRef: s.CorrelacionRef,
		VinculoAutenticacionActor: s.VinculoAutenticacionActor,
		AsignacionRef:             "asignacion:llamamiento:0001", AsignacionHuellaSHA256: huellaAplicacionLlamamiento('4'),
		VersionRolRef: "rol:tecnico_rrhh:v1", VersionRolHuellaSHA256: huellaAplicacionLlamamiento('5'),
		ControlVigenciaVersionRolRef: "rol:tecnico_rrhh:v1", ControlVigenciaVersionRolRevision: 1,
		ControlVigenciaVersionRolHuellaSHA256: huellaAplicacionLlamamiento('6'), RevisionCatalogoPoliticas: 1,
		CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		PoliticasEvaluadasRefs:        []string{referenciaPolitica}, PoliticasEvaluadasHuellasSHA256: huellas,
		PoliticasRefs: []string{referenciaPolitica}, PoliticasHuellasSHA256: huellas,
		GarantiaMinima: dominiovec.AuthAssuranceHigh,
		EmitidaEn:      instante.Add(-time.Second), ValidaHasta: instante.Add(2 * time.Minute),
	}
	if err := decision.ValidarEvidenciaInstantanea(); err != nil {
		t.Fatalf("decision: %v (validar=%v; accion=%q recurso=%q perfil=%q)", err, decision.Validar(), decision.Accion, decision.RecursoRef, decision.PerfilActivoRef)
	}
	return decision
}

func huellaAplicacionLlamamiento(caracter byte) string {
	return strings.Repeat(string(caracter), 64)
}
