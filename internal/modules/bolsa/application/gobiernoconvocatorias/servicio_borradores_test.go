package gobiernoconvocatorias

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var instanteBorradorPrueba = time.Date(2026, time.July, 18, 9, 0, 0, 0, time.UTC)

type relojPrueba struct {
	mu       sync.Mutex
	instante time.Time
	paso     time.Duration
}

func (r *relojPrueba) Ahora() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	actual := r.instante
	r.instante = r.instante.Add(r.paso)
	return actual
}

func (r *relojPrueba) avanzar(duracion time.Duration) {
	r.mu.Lock()
	r.instante = r.instante.Add(duracion)
	r.mu.Unlock()
}

type catalogoPrueba struct {
	mu            sync.Mutex
	plantilla     PlantillaBorradorResuelta
	ambito        dominiobolsa.AmbitoOrganizativoConvocatoria
	preparaciones int
}

func (c *catalogoPrueba) ResolverPlantillaBorrador(
	_ context.Context,
	selector SelectorPlantillaBorrador,
	_ time.Time,
) (PlantillaBorradorResuelta, error) {
	if selector.ID != c.plantilla.Referencia.ID || selector.Version != c.plantilla.Referencia.Version ||
		selector.HuellaContenidoSHA256 != c.plantilla.Referencia.HuellaContenidoSHA256 {
		return PlantillaBorradorResuelta{}, errors.New("plantilla ausente")
	}
	return c.plantilla, nil
}

func (c *catalogoPrueba) PrepararAltaBorrador(
	_ context.Context,
	plantilla PlantillaBorradorResuelta,
	_, _ string,
	_ time.Time,
) (PreparacionAltaBorrador, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.preparaciones++
	return PreparacionAltaBorrador{
		Plantilla: plantilla, ID: fmt.Sprintf("proceso:bolsa:auxiliar-2026-%d", c.preparaciones),
		InstanciaFlujoRef: "instancia:flujo:convocatoria:001", AmbitoOrganizativo: c.ambito,
	}, nil
}

func (c *catalogoPrueba) ResolverMotivoBorrador(
	_ context.Context,
	referencia dominiovec.ReferenciaEntradaCatalogo,
	_ time.Time,
) (dominiovec.ReferenciaEntradaCatalogo, error) {
	if referencia.Validar() != nil {
		return dominiovec.ReferenciaEntradaCatalogo{}, errors.New("motivo no publicado")
	}
	return referencia, nil
}

type lectorPrueba struct {
	version dominiobolsa.VersionConvocatoriaGobernada
}

func (l lectorPrueba) ObtenerBorradorExacto(
	context.Context,
	puertosbolsa.ReferenciaEstadoVersionConvocatoria,
) (dominiobolsa.VersionConvocatoriaGobernada, error) {
	return l.version.ClonarCanonico()
}

type comprometedorPrueba struct{}

func (comprometedorPrueba) ComprometerMotivo(
	_ context.Context,
	s puertosbolsa.SolicitudComprometerMotivoGobiernoConvocatoria,
) (puertosbolsa.HMACMotivoGobiernoConvocatoria, error) {
	return puertosbolsa.HMACMotivoGobiernoConvocatoria{
		DominioCriptografico: s.DominioCriptografico, GeneracionClave: 3,
		ClaveHMACRef: "motivo-gobierno-v3", HuellaEntradaSHA256: s.HuellaSolicitudSHA256,
		ValorHMACSHA256: huellaHexPrueba('a'),
	}, nil
}

type derivadorPrueba struct{ generaciones []uint32 }

func (d derivadorPrueba) Derivar(
	_ context.Context,
	s SolicitudDerivacionIdempotencia,
) (ConjuntoIdentidadesOperacion, error) {
	preimagenL, preimagenF, err := s.MaterialParaConectorConfiable()
	if err != nil {
		return ConjuntoIdentidadesOperacion{}, err
	}
	generaciones := d.generaciones
	if len(generaciones) == 0 {
		generaciones = []uint32{2, 1}
	}
	identidades := make([]IdentidadOperacionDerivada, 0, len(generaciones))
	for _, generacion := range generaciones {
		macL := hmac.New(sha256.New, []byte(fmt.Sprintf("clave-localizador-prueba-%02d-xxxx", generacion)))
		_, _ = macL.Write(preimagenL)
		macF := hmac.New(sha256.New, []byte(fmt.Sprintf("clave-huella-solicitud-%02d-xxxx", generacion)))
		_, _ = macF.Write(preimagenF)
		refL, _ := NuevaReferenciaClaveHMACLocalizador(
			fmt.Sprintf("clave:hmac:convocatorias:localizador:v%d", generacion), generacion,
		)
		refF, _ := NuevaReferenciaClaveHMACHuellaSolicitud(
			fmt.Sprintf("clave:hmac:convocatorias:huella:v%d", generacion), generacion,
		)
		l, errL := NuevoLocalizadorOperacion(2, refL, hex.EncodeToString(macL.Sum(nil)))
		f, errF := NuevaHuellaSolicitud(2, refF, hex.EncodeToString(macF.Sum(nil)))
		identidad, errIdentidad := NuevaIdentidadOperacionDerivada(l, f)
		if err := errors.Join(errL, errF, errIdentidad); err != nil {
			return ConjuntoIdentidadesOperacion{}, err
		}
		identidades = append(identidades, identidad)
	}
	return NuevoConjuntoIdentidadesOperacion(identidades...)
}

type modoPDP string

const (
	pdpConceder modoPDP = "conceder"
	pdpDenegar  modoPDP = "denegar"
	pdpCaido    modoPDP = "caido"
)

type autorizadorPrueba struct {
	mu       sync.Mutex
	llamadas int
	modo     modoPDP
}

func (a *autorizadorPrueba) EvaluarDecisionBorrador(
	_ context.Context,
	actor dominiovec.ContextoActor,
	vinculo dominiovec.VinculoAutenticacionActorV1,
	recurso dominiovec.RecursoAutorizable,
	correlacion string,
	_ dominiovec.ReferenciaEntradaCatalogo,
	intencion IntencionBorradorCanonica,
	instante time.Time,
) (ResultadoEvaluacionPDPBorrador, error) {
	a.mu.Lock()
	a.llamadas++
	modo := a.modo
	a.mu.Unlock()
	if modo == pdpCaido {
		return ResultadoEvaluacionPDPBorrador{}, errors.New("pdp no disponible")
	}
	if modo == pdpDenegar {
		return ResultadoEvaluacionPDPBorrador{
			Estado: EvaluacionPDPDenegada, DenegacionRef: "denegacion:pdp:borrador:001",
		}, nil
	}
	huellaContexto, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || !intencion.valida() {
		return ResultadoEvaluacionPDPBorrador{}, ErrIntencionBorradorInvalida
	}
	huellaCatalogo, _ := dominiovec.HuellaCatalogoPoliticasAutorizacion(nil)
	decision := dominiovec.DecisionAutorizacion{
		DecisionRef: "decision:pdp:borrador:001", Concedida: true, Codigo: "concedida",
		PrincipalID: actor.PersonaRef, PerfilActivoRef: actor.PerfilActivoRef,
		Accion: intencion.accion(), RecursoRef: recurso.Referencia,
		ModuloID:                    puertosbolsa.ModuloGobiernoConvocatorias,
		TipoRecurso:                 puertosbolsa.TipoRecursoVersionConvocatoriaGobernada,
		ContextoRecursoHuellaSHA256: huellaContexto,
		Finalidad:                   puertosbolsa.FinalidadGobiernoConvocatorias, CorrelacionRef: correlacion,
		VinculoAutenticacionActor: vinculo, AsignacionRef: "asignacion:rrhh:v1",
		AsignacionHuellaSHA256: huellaHexPrueba('1'), VersionRolRef: "rol:rrhh:v1",
		VersionRolHuellaSHA256: huellaHexPrueba('2'), ControlVigenciaVersionRolRef: "rol:rrhh:v1",
		ControlVigenciaVersionRolRevision:     1,
		ControlVigenciaVersionRolHuellaSHA256: huellaHexPrueba('3'),
		RevisionCatalogoPoliticas:             1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		PoliticasEvaluadasHuellasSHA256: map[string]string{}, GarantiaMinima: dominiovec.AuthAssuranceHigh,
		CamposPermitidos: []string{"auditoria", "evento_outbox", "version_convocatoria"},
		EmitidaEn:        instante, ValidaHasta: instante.Add(4 * time.Minute),
	}
	evidencia, err := puertosvec.NuevaEvidenciaUsoDecisionAutorizacion(decision, instante)
	if err != nil {
		return ResultadoEvaluacionPDPBorrador{}, err
	}
	return ResultadoEvaluacionPDPBorrador{
		Estado: EvaluacionPDPConcedida,
		Concesion: ConcesionBorradorDurable{
			Evidencia: evidencia,
			Atestacion: ProyeccionAtestacionPDP{
				DecisionRef: decision.DecisionRef, AtestacionRef: "atestacion:pdp:borrador:001",
				VersionAtestacion: 1, EstadoAtestacion: "activa",
				HuellaAtestacionSHA256: huellaHexPrueba('4'), VerificadorRef: "verificador:pdp:v1",
				VerificadaEn: instante,
			},
		},
	}, nil
}

type filaDiarioPrueba struct {
	identidadPrimaria  ProyeccionIdentidadOperacion
	resultado          ResultadoOperacionDiario
	confirmacionOculta *SolicitudConfirmacionBorrador
}

type aliasDiarioPrueba struct {
	identidad     ProyeccionIdentidadOperacion
	clavePrimaria string
}

type diarioPrueba struct {
	mu                 sync.Mutex
	filas              map[string]filaDiarioPrueba
	aliases            map[string]aliasDiarioPrueba
	reservas, reclamos int
	fallarTrasReserva  bool
	forzarMergeG2      bool
	ultima             *SolicitudReservaDecisionBorrador
	ultimaReconciliada *ProyeccionIdentidadOperacion
	ultimaReclamacion  *SolicitudReclamacionDecisionBorrador
}

func nuevoDiarioPrueba() *diarioPrueba {
	return &diarioPrueba{
		filas: make(map[string]filaDiarioPrueba), aliases: make(map[string]aliasDiarioPrueba),
	}
}

func claveL(i ProyeccionIdentidadOperacion) string {
	p := i.Localizador
	return fmt.Sprintf("%d:%s:%s:%d:%s", p.VersionEsquema, p.Dominio, p.ClaveRef, p.GeneracionClave, p.ValorHMACSHA256)
}

func mismaF(a, b ProyeccionIdentidadOperacion) bool {
	return proyeccionesHMACCoinciden(
		a.HuellaSolicitud, b.HuellaSolicitud, dominioClaveHMACHuellaSolicitud,
	)
}

func copiarResultado(r ResultadoOperacionDiario) ResultadoOperacionDiario {
	if r.Recibo != nil {
		copia := *r.Recibo
		r.Recibo = &copia
	}
	return r
}

func resolucionIdentidadPrueba(
	primaria ProyeccionIdentidadOperacion,
	aliases []ProyeccionIdentidadOperacion,
	_ time.Time,
) (ResolucionIdentidadBorrador, error) {
	return ResolucionIdentidadBorrador{
		IdentidadesConsultadas: append([]ProyeccionIdentidadOperacion(nil), aliases...),
		IdentidadPrimaria:      primaria,
	}, nil
}

func (d *diarioPrueba) ConsultarIdentidades(
	_ context.Context,
	s SolicitudConsultaIdentidadesBorrador,
) (ResultadoConsultaIdentidadesBorrador, error) {
	if !identidadesConsultaValidas(s.Identidades) {
		return ResultadoConsultaIdentidadesBorrador{}, ErrRotacionIdempotenciaInvalida
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	resultado := ResultadoConsultaIdentidadesBorrador{}
	porPrimaria := make(map[string]int)
	for _, identidad := range s.Identidades {
		alias, existe := d.aliases[claveL(identidad)]
		if !existe {
			continue
		}
		fila, existe := d.filas[alias.clavePrimaria]
		if !existe {
			return ResultadoConsultaIdentidadesBorrador{}, errors.New("alias huerfano")
		}
		estado := copiarResultado(fila.resultado)
		if !mismaF(alias.identidad, identidad) {
			estado = ResultadoOperacionDiario{Estado: ResultadoDiarioConflicto}
		}
		indice, agrupada := porPrimaria[alias.clavePrimaria]
		if agrupada {
			resultado.Coincidencias[indice].Resolucion.IdentidadesConsultadas = append(
				resultado.Coincidencias[indice].Resolucion.IdentidadesConsultadas, identidad,
			)
			if estado.Estado == ResultadoDiarioConflicto {
				resultado.Coincidencias[indice].Resultado = estado
			}
			continue
		}
		porPrimaria[alias.clavePrimaria] = len(resultado.Coincidencias)
		resultado.Coincidencias = append(resultado.Coincidencias, CoincidenciaIdentidadBorrador{
			Resolucion: ResolucionIdentidadBorrador{
				IdentidadesConsultadas: []ProyeccionIdentidadOperacion{identidad},
				IdentidadPrimaria:      fila.identidadPrimaria,
			},
			Resultado: estado,
		})
	}
	for indice := range resultado.Coincidencias {
		actual := &resultado.Coincidencias[indice]
		resolucion, err := resolucionIdentidadPrueba(
			actual.Resolucion.IdentidadPrimaria,
			actual.Resolucion.IdentidadesConsultadas,
			s.SolicitadaEn,
		)
		if err != nil {
			return ResultadoConsultaIdentidadesBorrador{}, err
		}
		actual.Resolucion = resolucion
	}
	return resultado, nil
}

func (d *diarioPrueba) ReservarDecision(
	_ context.Context,
	s SolicitudReservaDecisionBorrador,
) (ResultadoReservaDecisionBorrador, error) {
	if s.Validar() != nil {
		return ResultadoReservaDecisionBorrador{}, ErrReservaBorradorInvalida
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, identidad := range s.IdentidadesConsulta {
		if alias, existe := d.aliases[claveL(identidad)]; existe {
			fila, existe := d.filas[alias.clavePrimaria]
			if !existe {
				return ResultadoReservaDecisionBorrador{}, errors.New("alias huerfano")
			}
			resultado := copiarResultado(fila.resultado)
			if !mismaF(alias.identidad, identidad) {
				resultado = ResultadoOperacionDiario{Estado: ResultadoDiarioConflicto}
			} else if resultado.Estado == ResultadoDiarioReservado {
				resultado.Estado = ResultadoDiarioEnCurso
			}
			identidadesResueltas := s.IdentidadesConsulta
			if resultado.Estado == ResultadoDiarioConflicto {
				identidadesResueltas = []ProyeccionIdentidadOperacion{identidad}
			}
			resolucion, err := resolucionIdentidadPrueba(
				fila.identidadPrimaria,
				identidadesResueltas,
				s.SolicitadaEn,
			)
			if err != nil {
				return ResultadoReservaDecisionBorrador{}, err
			}
			return ResultadoReservaDecisionBorrador{
				Resolucion: resolucion,
				Resultado:  resultado,
			}, nil
		}
	}
	if d.forzarMergeG2 {
		if len(s.IdentidadesConsulta) < 2 {
			return ResultadoReservaDecisionBorrador{}, ErrReservaBorradorInvalida
		}
		d.forzarMergeG2 = false
		primaria := s.IdentidadesConsulta[1]
		// Modela al ganador real: reservó con otra decisión y, por tanto, con
		// otro instante/lease. La función SQL proyecta esa reserva ajena como
		// en_curso para que el perdedor no pueda confirmarla.
		inicioGanador := s.Proyeccion.ArrendamientoIniciaEn.Add(-time.Second)
		venceGanador := s.Proyeccion.ArrendamientoVenceEn.Add(-time.Second)
		resultadoPersistido := ResultadoOperacionDiario{
			Estado: ResultadoDiarioReservado, Revision: 1, Cercado: 1,
			ArrendamientoIniciaEn: inicioGanador,
			ArrendamientoVenceEn:  venceGanador,
		}
		clavePrimaria := claveL(primaria)
		d.filas[clavePrimaria] = filaDiarioPrueba{
			identidadPrimaria: primaria, resultado: resultadoPersistido,
		}
		for _, identidad := range s.IdentidadesConsulta {
			d.aliases[claveL(identidad)] = aliasDiarioPrueba{
				identidad: identidad, clavePrimaria: clavePrimaria,
			}
		}
		final, err := resolucionIdentidadPrueba(
			primaria, s.IdentidadesConsulta, s.SolicitadaEn,
		)
		if err != nil {
			return ResultadoReservaDecisionBorrador{}, err
		}
		d.reservas++
		copia := s
		d.ultima = &copia
		resultadoRespuesta := resultadoPersistido
		resultadoRespuesta.Estado = ResultadoDiarioEnCurso
		return ResultadoReservaDecisionBorrador{
			Resolucion: final, Resultado: resultadoRespuesta,
		}, nil
	}
	p := s.Proyeccion
	resultado := ResultadoOperacionDiario{
		Estado: ResultadoDiarioReservado, Revision: 1, Cercado: 1,
		ArrendamientoIniciaEn: p.ArrendamientoIniciaEn, ArrendamientoVenceEn: p.ArrendamientoVenceEn,
	}
	clavePrimaria := claveL(p.IdentidadPrimaria)
	d.filas[clavePrimaria] = filaDiarioPrueba{
		identidadPrimaria: p.IdentidadPrimaria, resultado: resultado,
	}
	for _, identidad := range s.IdentidadesConsulta {
		d.aliases[claveL(identidad)] = aliasDiarioPrueba{
			identidad: identidad, clavePrimaria: clavePrimaria,
		}
	}
	d.reservas++
	copia := s
	d.ultima = &copia
	resolucion, err := resolucionIdentidadPrueba(
		p.IdentidadPrimaria, s.IdentidadesConsulta, s.SolicitadaEn,
	)
	if err != nil {
		return ResultadoReservaDecisionBorrador{}, err
	}
	respuesta := ResultadoReservaDecisionBorrador{
		Resolucion: resolucion,
		Resultado:  resultado,
	}
	if d.fallarTrasReserva {
		d.fallarTrasReserva = false
		return ResultadoReservaDecisionBorrador{}, errors.New("caida tras commit de reserva")
	}
	return respuesta, nil
}

func huellaHexPrueba(marca byte) string {
	bytes := make([]byte, 64)
	for indice := range bytes {
		bytes[indice] = marca
	}
	return string(bytes)
}

func datosPublicablesPrueba(t *testing.T) (
	dominiobolsa.ContenidoPublicableConvocatoria,
	dominiobolsa.ConfiguracionFijadaConvocatoria,
	dominiobolsa.AmbitoOrganizativoConvocatoria,
) {
	t.Helper()
	ambito, err := dominiobolsa.NuevoAmbitoOrganizativoConvocatoria("org_diputaciongranada", "uni_seleccionexterna")
	if err != nil {
		t.Fatal(err)
	}
	contenido := dominiobolsa.ContenidoPublicableConvocatoria{
		IdentificadorPublico: "auxiliar-administrativo-2026", Tipo: "bolsa_temporal",
		CatalogoCategorias: dominiobolsa.ReferenciaCatalogoCategorias{
			CatalogoID: "categorias-profesionales", CatalogoVersion: 1,
			CatalogoHuellaSHA256: huellaHexPrueba('a'),
		},
		Categorias: []string{"auxiliar_administrativo"}, Titulo: "Bolsa temporal de auxiliares",
		Resumen: "Convocatoria publica para bolsa temporal.", Descripcion: "Proceso sujeto a bases.",
		Plazos: []dominiobolsa.PlazoConvocatoria{{
			Referencia: "plazo:inscripcion", Tipo: "inscripcion", Titulo: "Inscripcion",
			Descripcion: "Plazo de presentacion.", AbreEn: instanteBorradorPrueba.Add(24 * time.Hour),
			CierraEn: instanteBorradorPrueba.Add(30 * 24 * time.Hour),
		}},
		Requisitos: []dominiobolsa.RequisitoConvocatoria{{
			Referencia: "requisito:edad", Orden: 1, Titulo: "Edad",
			Descripcion: "Cumplir la edad exigida.", Obligatorio: true,
		}},
		Documentos: []dominiobolsa.DocumentoPublicableConvocatoria{{
			Referencia: "documento:bases", Tipo: "bases", Orden: 1, Titulo: "Bases",
			Descripcion: "Bases firmadas.", Formato: "pdf", URL: "/bolsa/documentos/bases.pdf",
		}},
	}
	ref := func(id string, marca byte) dominiobolsa.ReferenciaConfiguracionConvocatoria {
		return dominiobolsa.ReferenciaConfiguracionConvocatoria{ID: id, Version: 1, HuellaContenidoSHA256: huellaHexPrueba(marca)}
	}
	configuracion := dominiobolsa.ConfiguracionFijadaConvocatoria{
		Catalogos: ref("catalogos:bolsa", '1'), Calendario: ref("calendario:bolsa", '2'),
		ReglasBaremacion: ref("baremo:bolsa", '3'), FlujoProceso: ref("convocatoria-bolsa", '4'),
		FlujoSolicitud: ref("solicitud-bolsa", '5'),
		Plantilla:      ref("plantilla:bolsa:general", '8'),
		Documentos: []dominiobolsa.ReferenciaDocumentoOficialConvocatoria{{
			Rol: "bases", PublicacionRef: "documento:bases", DocumentoRef: "documento:logico:bases:001",
			VersionDocumento: 1, RepresentacionRef: "representacion:pdf:bases:001",
			HuellaContenidoSHA256: huellaHexPrueba('b'), FirmaValidadaRef: "firma:validada:bases:001",
			ReciboCustodiaRef: "custodia:bases:001",
		}},
	}
	return contenido, configuracion, ambito
}
