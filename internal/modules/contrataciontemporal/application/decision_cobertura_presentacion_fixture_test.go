package application

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type resolutorContextoPresentacionPrueba struct {
	mu       sync.Mutex
	contexto ports.ContextoAutorizacionAltaV3
	err      error
	llamadas int
	cancelar context.CancelFunc
}

func (r *resolutorContextoPresentacionPrueba) ResolverContextoAutorizacionAltaV3(
	_ context.Context,
	_ ports.SolicitudResolverContextoAutorizacionAltaV3,
) (ports.ContextoAutorizacionAltaV3, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.llamadas++
	if r.cancelar != nil {
		r.cancelar()
	}
	return r.contexto, r.err
}

func (r *resolutorContextoPresentacionPrueba) total() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.llamadas
}

type autorizadorPresentacionPrueba struct {
	mu       sync.Mutex
	err      error
	errores  map[int]error
	llamadas int
	cancelar context.CancelFunc
	antes    func(int)
}

func (a *autorizadorPresentacionPrueba) AutorizarPresentacionPropuestaCobertura(
	_ context.Context,
	solicitudContexto ports.SolicitudResolverContextoAutorizacionAltaV3,
	contexto ports.ContextoAutorizacionAltaV3,
	solicitudAnalisis cobertura.SolicitudInstantaneaAnalisisDurableO3,
	instante time.Time,
) error {
	a.mu.Lock()
	a.llamadas++
	llamada := a.llamadas
	antes := a.antes
	cancelar := a.cancelar
	errAutorizacion := a.err
	if errLlamada, existe := a.errores[llamada]; existe {
		errAutorizacion = errLlamada
	}
	a.mu.Unlock()
	if antes != nil {
		antes(llamada)
	}
	if solicitudContexto.Validar() != nil ||
		contexto.ValidarPara(solicitudContexto, instante) != nil {
		return errors.New("contexto de prueba no ligado")
	}
	if _, _, _, err := solicitudAnalisis.Coordenadas(); err != nil {
		return errors.New("coordenadas de prueba no ligadas")
	}
	if cancelar != nil {
		cancelar()
	}
	return errAutorizacion
}

func (a *autorizadorPresentacionPrueba) total() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.llamadas
}

type lectorAnalisisPresentacionPrueba struct {
	mu         sync.Mutex
	expediente domain.Expediente
	err        error
	llamadas   int
	cancelar   context.CancelFunc
}

func (l *lectorAnalisisPresentacionPrueba) LeerExpedienteAnalisisDurableO3(
	_ context.Context,
	_ cobertura.SolicitudInstantaneaAnalisisDurableO3,
) (domain.Expediente, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.llamadas++
	if l.cancelar != nil {
		l.cancelar()
	}
	return l.expediente.Clonar(), l.err
}

func (l *lectorAnalisisPresentacionPrueba) total() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.llamadas
}

type relojGobiernoPresentacionPrueba struct {
	mu       sync.Mutex
	reloj    *relojCoberturaAplicacionPrueba
	err      error
	llamadas int
	cancelar context.CancelFunc
}

func (r *relojGobiernoPresentacionPrueba) AhoraGobiernoOperacionCobertura(
	_ context.Context,
) (time.Time, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.llamadas++
	if r.cancelar != nil {
		r.cancelar()
	}
	return r.reloj.Ahora(), r.err
}

type resolutorGobiernoPresentacionPrueba struct {
	mu       sync.Mutex
	catalogo domain.CatalogoViasCobertura
	politica domain.PoliticaDecisionCobertura
	err      error
	llamadas int
	cancelar context.CancelFunc
}

func (r *resolutorGobiernoPresentacionPrueba) ResolverGobiernoOperacionCobertura(
	_ context.Context,
	solicitud cobertura.SolicitudResolucionGobiernoOperacionCobertura,
) (cobertura.PublicacionGobiernoOperacionCobertura, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.llamadas++
	if r.cancelar != nil {
		r.cancelar()
	}
	if r.err != nil {
		return cobertura.PublicacionGobiernoOperacionCobertura{}, r.err
	}
	organizacion, expediente, version, accion, instante, err :=
		solicitud.Coordenadas()
	if err != nil {
		return cobertura.PublicacionGobiernoOperacionCobertura{}, err
	}
	finalidadClave, finalidadRef := r.politica.Finalidad()
	actuacion := cobertura.PublicacionPoliticaActuacionCobertura{
		Referencia:                 "politica_actuacion_presentacion_cobertura_01",
		Version:                    1,
		Canon:                      cobertura.CanonHuellaPoliticaActuacionCoberturaV1(),
		OrganizacionRef:            organizacion,
		Accion:                     accion,
		Catalogo:                   r.catalogo.Identidad(),
		Politica:                   r.politica.Identidad(),
		FinalidadContratacionClave: finalidadClave,
		FinalidadContratacionRef:   finalidadRef,
		FinalidadAutorizacionVEC:   "tramitar_cobertura_temporal",
		UnidadEjecutoraRef:         "unidad_rrhh_presentacion_cobertura_01",
		FaseDestino:                "decision_cobertura",
		EstadoDestino:              domain.EstadoEnCurso,
		MotivoAutorizacionDecidir: referenciaMotivoPresentacionPrueba(
			"motivo_autorizacion_decidir_presentacion",
		),
		MotivoAutorizacionRectificar: referenciaMotivoPresentacionPrueba(
			"motivo_autorizacion_rectificar_presentacion",
		),
		PublicadaEn: instante.Add(-time.Hour),
		Vigencia: domain.VigenciaCatalogoCobertura{
			Desde: instante.Add(-time.Minute),
			Hasta: instante.Add(20 * time.Minute),
		},
	}
	actuacion.HuellaSHA256, err =
		cobertura.CalcularHuellaSHA256PoliticaActuacionCobertura(actuacion)
	if err != nil {
		return cobertura.PublicacionGobiernoOperacionCobertura{}, err
	}
	return cobertura.PublicacionGobiernoOperacionCobertura{
		OrganizacionRef: organizacion, ExpedienteRef: expediente,
		VersionExpediente: version, Catalogo: r.catalogo,
		Politica: r.politica, PoliticaActuacion: actuacion,
	}, nil
}

func (r *resolutorGobiernoPresentacionPrueba) total() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.llamadas
}

type escenarioPresentacionCobertura struct {
	global     *escenarioPreparacionGlobalPrueba
	contextos  *resolutorContextoPresentacionPrueba
	accesos    *autorizadorPresentacionPrueba
	analisis   *lectorAnalisisPresentacionPrueba
	reloj      *relojGobiernoPresentacionPrueba
	gobierno   *resolutorGobiernoPresentacionPrueba
	servicio   *ServicioPresentacionPropuestaCobertura
	solicitud  SolicitudProponerCobertura
	expediente domain.Expediente
}

func nuevoEscenarioPresentacionCobertura(
	t *testing.T,
	vias []domain.DefinicionViaCobertura,
) *escenarioPresentacionCobertura {
	t.Helper()
	global := nuevoEscenarioPreparacionGlobalPrueba(
		t,
		vias,
		4,
		2*time.Second,
	)
	var avanceReloj sync.Mutex
	global.entorno.fuente.consultar = func(
		ctx context.Context,
		solicitud ports.SolicitudConsultarCobertura,
	) (ports.ResultadoConsultaCobertura, error) {
		if err := ctx.Err(); err != nil {
			return ports.ResultadoConsultaCobertura{}, err
		}
		if global.antes != nil {
			if err := global.antes(ctx, solicitud); err != nil {
				return ports.ResultadoConsultaCobertura{}, err
			}
		}
		avanceReloj.Lock()
		defer avanceReloj.Unlock()
		instanteResultado := solicitud.SolicitadaEn.Add(time.Second)
		if instanteResultado.After(global.entorno.reloj.Ahora()) {
			global.entorno.reloj.fijar(instanteResultado)
		}
		return resultadoCoberturaAplicacionPrueba(
			t,
			solicitud,
			func(datos *ports.DatosResultadoConsultaCobertura) {
				datos.Comprobacion.ReciboRef =
					"recibo_" + solicitud.PeticionRef
			},
		), nil
	}
	instante := global.entorno.reloj.Ahora()
	expediente := expedientePresentacionCoberturaPrueba(t, instante)
	contexto := contextoAutorizacionAltaV3Prueba(t, instante)
	vinculo, err := contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	escenario := &escenarioPresentacionCobertura{
		global: global,
		contextos: &resolutorContextoPresentacionPrueba{
			contexto: contexto,
		},
		accesos: &autorizadorPresentacionPrueba{},
		analisis: &lectorAnalisisPresentacionPrueba{
			expediente: expediente,
		},
		reloj: &relojGobiernoPresentacionPrueba{
			reloj: global.entorno.reloj,
		},
		gobierno: &resolutorGobiernoPresentacionPrueba{
			catalogo: global.catalogo,
			politica: global.politica,
		},
		solicitud: SolicitudProponerCobertura{
			AutenticacionRef: vinculo.AutenticacionRef,
			SesionRef:        vinculo.SesionRef,
			PerfilRef:        vinculo.PerfilActivoRef,
			OrganizacionRef:  expediente.OrganizacionRef,
			ExpedienteRef:    expediente.Referencia,
			VersionEsperada:  expediente.Version,
		},
		expediente: expediente,
	}
	escenario.servicio, err =
		NuevoServicioPresentacionPropuestaCobertura(
			escenario.contextos,
			escenario.accesos,
			escenario.analisis,
			escenario.reloj,
			escenario.gobierno,
			global.preparador,
		)
	if err != nil {
		t.Fatal(err)
	}
	return escenario
}

func expedientePresentacionCoberturaPrueba(
	t *testing.T,
	instante time.Time,
) domain.Expediente {
	t.Helper()
	periodo := domain.PeriodoPrevisto{
		Inicio: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		Fin:    time.Date(2027, 3, 31, 0, 0, 0, 0, time.UTC),
	}
	expediente, err := domain.NuevoExpediente(domain.AltaExpediente{
		Referencia:      "expediente_presentacion_cobertura_012345",
		OrganizacionRef: organizacionCoberturaPrueba,
		NumeroVisible:   "2026/PRES-0001",
		Flujo: domain.ReferenciaFlujo{
			DefinicionRef: "flujo_presentacion_cobertura_01",
			Version:       1,
			HuellaSHA256:  strings.Repeat("7", 64),
		},
		FaseInicial: "analisis_rrhh",
		Solicitud: domain.SolicitudCentro{
			CentroRef:     "centro_presentacion_cobertura_01",
			ContactoRef:   "contacto_presentacion_cobertura_01",
			CategoriaRef:  "categoria_trabajo_social",
			GrupoSubgrupo: "A2",
			MotivoClave:   "necesidad_cobertura_temporal",
			Detalle:       "Necesidad sintética para probar la presentación.",
			Periodo:       periodo,
			RC:            domain.DeclaracionRC{Existe: false},
		},
		Actuacion: domain.DatosActuacion{
			AccionClave:   "registrar_solicitud_cobertura",
			ActorRef:      "actor_centro_presentacion_01",
			UnidadRef:     "unidad_centro_presentacion_01",
			ReciboRef:     "recibo_alta_presentacion_cobertura_01",
			RealizadaEn:   instante.Add(-time.Hour),
			FaseDestino:   "analisis_rrhh",
			EstadoDestino: domain.EstadoEnCurso,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	entrada := domain.VinculoEntradaRC{
		Referencia:   "entrada_rc_presentacion_cobertura_01",
		HuellaSHA256: strings.Repeat("6", 64),
	}
	expediente, err = expediente.RegistrarAnalisis(
		expediente.Version,
		domain.AnalisisRRHH{
			ModalidadClave:    "modalidad_interinidad",
			CategoriaRef:      expediente.Solicitud.CategoriaRef,
			GrupoSubgrupo:     expediente.Solicitud.GrupoSubgrupo,
			CausaClave:        "causa_sustitucion",
			Periodo:           periodo,
			PorcentajeJornada: domain.JornadaCompletaDiezmilesimas,
			EntradaRCEsperada: entrada,
			ValidacionRC: domain.ValidacionRC{
				Resultado:           domain.RCNoRequerida,
				EntradaRef:          entrada.Referencia,
				HuellaEntradaSHA256: entrada.HuellaSHA256,
				FuenteRef:           "fuente_rc_presentacion_cobertura_01",
				ReciboRef:           "recibo_rc_presentacion_cobertura_01",
				ValidadaEn:          instante.Add(-30 * time.Minute),
				Motivo:              "No requiere retención de crédito.",
			},
		},
		domain.DatosActuacion{
			AccionClave: domain.ClaveCatalogo(
				ports.AccionRegistrarAnalisis,
			),
			ActorRef:      "actor_rrhh_presentacion_cobertura_01",
			UnidadRef:     "unidad_rrhh_presentacion_cobertura_01",
			ReciboRef:     "recibo_analisis_presentacion_cobertura_01",
			RealizadaEn:   instante.Add(-20 * time.Minute),
			FaseDestino:   expediente.FaseActual,
			EstadoDestino: expediente.EstadoActual,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return expediente
}

func referenciaMotivoPresentacionPrueba(
	clave string,
) dominiovec.ReferenciaEntradaCatalogo {
	huellaClave := "1"
	if strings.Contains(clave, "rectificar") {
		huellaClave = "2"
	}
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_autorizacion_presentacion_cobertura",
		CatalogoVersion:      1,
		CatalogoHuellaSHA256: strings.Repeat("a", 64),
		EntradaClave:         "motivo_" + strings.Repeat(huellaClave, 32),
	}
}

func viasPresentacionCoberturaPrueba(
	cantidad int,
) []domain.DefinicionViaCobertura {
	return viasPreparacionGlobalPrueba(cantidad, 1)
}

func configurarResultadosPresentacionCobertura(
	t *testing.T,
	escenario *escenarioPresentacionCobertura,
	resultado func(ports.SolicitudConsultarCobertura) domain.ResultadoComprobacion,
) {
	t.Helper()
	escenario.global.entorno.fuente.consultar = func(
		ctx context.Context,
		solicitud ports.SolicitudConsultarCobertura,
	) (ports.ResultadoConsultaCobertura, error) {
		if err := ctx.Err(); err != nil {
			return ports.ResultadoConsultaCobertura{}, err
		}
		escenario.global.entorno.reloj.fijar(
			solicitud.SolicitadaEn.Add(2 * time.Second),
		)
		return resultadoCoberturaAplicacionPrueba(
			t,
			solicitud,
			func(datos *ports.DatosResultadoConsultaCobertura) {
				datos.Comprobacion.Resultado = resultado(solicitud)
				datos.Comprobacion.ReciboRef =
					"recibo_" + solicitud.PeticionRef
			},
		), nil
	}
}
