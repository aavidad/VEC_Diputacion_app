package gobiernoconvocatorias

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrServicioBorradoresInvalido     = errors.New("gobierno convocatorias: servicio de borradores invalido")
	ErrReservaBorradorInvalida        = errors.New("gobierno convocatorias: reserva de borrador invalida")
	ErrOperacionBorradorEnCurso       = errors.New("gobierno convocatorias: operacion de borrador en curso")
	ErrOperacionBorradorIndeterminada = errors.New("gobierno convocatorias: resultado de operacion indeterminado")
	ErrConfirmacionBorradorNoAplicada = errors.New("gobierno convocatorias: confirmacion no aplicada")
	ErrResultadoBorradorInseguro      = errors.New("gobierno convocatorias: resultado de borrador no confiable")
)

// ProyeccionHMACDiario es la forma durable deliberada de L o F. No contiene
// la clave cliente ni material de clave HMAC; Dominio mantiene tipos nominales.
type ProyeccionHMACDiario struct {
	bloqueoSerializacionDiario
	VersionEsquema  uint16
	Dominio         string
	ClaveRef        string
	GeneracionClave uint32
	ValorHMACSHA256 string
}

func proyeccionHMACDiario(h hmacNominalIdempotencia) (ProyeccionHMACDiario, error) {
	if !h.valido() {
		return ProyeccionHMACDiario{}, ErrReservaBorradorInvalida
	}
	dominio := ""
	switch h.clave.dominio {
	case dominioClaveHMACLocalizador:
		dominio = "localizador"
	case dominioClaveHMACHuellaSolicitud:
		dominio = "huella_solicitud"
	default:
		return ProyeccionHMACDiario{}, ErrReservaBorradorInvalida
	}
	return ProyeccionHMACDiario{
		VersionEsquema: h.versionEsquema, Dominio: dominio,
		ClaveRef:        string(h.clave.referencia[:h.clave.longitud]),
		GeneracionClave: h.clave.generacionClave,
		ValorHMACSHA256: hex.EncodeToString(h.valor[:]),
	}, nil
}

func (p ProyeccionHMACDiario) valida(dominio string) bool {
	if p.VersionEsquema == 0 || p.Dominio != dominio || p.GeneracionClave == 0 ||
		len(p.ValorHMACSHA256) != 64 {
		return false
	}
	bytes, err := hex.DecodeString(p.ValorHMACSHA256)
	if err != nil || len(bytes) != sha256.Size || hex.EncodeToString(bytes) != p.ValorHMACSHA256 {
		return false
	}
	clave, err := func() (ReferenciaClaveHMAC, error) {
		if dominio == "localizador" {
			return NuevaReferenciaClaveHMACLocalizador(p.ClaveRef, p.GeneracionClave)
		}
		return NuevaReferenciaClaveHMACHuellaSolicitud(p.ClaveRef, p.GeneracionClave)
	}()
	return err == nil && clave.valida()
}

type ProyeccionIdentidadOperacion struct {
	bloqueoSerializacionDiario
	Localizador     ProyeccionHMACDiario
	HuellaSolicitud ProyeccionHMACDiario
}

func nuevaProyeccionIdentidadOperacion(
	localizador LocalizadorOperacion,
	huella HuellaSolicitud,
) (ProyeccionIdentidadOperacion, error) {
	l, errL := proyeccionHMACDiario(localizador.hmac)
	f, errF := proyeccionHMACDiario(huella.hmac)
	resultado := ProyeccionIdentidadOperacion{Localizador: l, HuellaSolicitud: f}
	if errL != nil || errF != nil || !resultado.valida() {
		return ProyeccionIdentidadOperacion{}, ErrReservaBorradorInvalida
	}
	return resultado, nil
}

func (p ProyeccionIdentidadOperacion) valida() bool {
	return p.Localizador.valida("localizador") && p.HuellaSolicitud.valida("huella_solicitud")
}

func (p ProyeccionIdentidadOperacion) Validar() error {
	if !p.valida() {
		return ErrReservaBorradorInvalida
	}
	return nil
}

// ProyeccionDecisionDiario liga la reserva a la decision y a las revisiones
// exactas de rol y catalogo de politicas. No conserva PrincipalID.
type ProyeccionDecisionDiario struct {
	bloqueoSerializacionDiario
	EsquemaHuella                         string
	DecisionRef                           string
	HuellaDecisionSHA256                  string
	Accion                                string
	RecursoRef                            string
	ModuloID                              string
	TipoRecurso                           string
	ContextoRecursoHuellaSHA256           string
	Finalidad                             string
	AsignacionRef                         string
	AsignacionHuellaSHA256                string
	VersionRolRef                         string
	VersionRolHuellaSHA256                string
	ControlVigenciaVersionRolRef          string
	ControlVigenciaVersionRolRevision     uint64
	ControlVigenciaVersionRolHuellaSHA256 string
	RevisionCatalogoPoliticas             uint64
	CatalogoPoliticasHuellaSHA256         string
	EmitidaEn                             time.Time
	VerificadaEn                          time.Time
	ValidaHasta                           time.Time
	AtestacionPDP                         ProyeccionAtestacionPDP
}

// ProyeccionAtestacionPDP identifica el registro durable exacto que el diario
// debe releer y verificar. Es integridad y procedencia a comprobar, no una
// autorizacion autocontenida.
type ProyeccionAtestacionPDP struct {
	bloqueoSerializacionDiario
	DecisionRef            string
	AtestacionRef          string
	VersionAtestacion      uint32
	EstadoAtestacion       string
	HuellaAtestacionSHA256 string
	VerificadorRef         string
	VerificadaEn           time.Time
}

func (p ProyeccionAtestacionPDP) validaPara(decisionRef string, verificadaEn time.Time) bool {
	return referenciaProyeccionValida(p.DecisionRef) && p.DecisionRef == decisionRef &&
		referenciaProyeccionValida(p.AtestacionRef) && p.VersionAtestacion > 0 &&
		p.EstadoAtestacion == "activa" && huellaHexValida(p.HuellaAtestacionSHA256) &&
		referenciaProyeccionValida(p.VerificadorRef) && p.AtestacionRef != p.VerificadorRef &&
		instanteOperacionCanonico(p.VerificadaEn) && !p.VerificadaEn.After(verificadaEn)
}

// ConcesionBorradorDurable agrupa evidencia opaca y atestacion registrada. El
// diario no confia en la copia: relee ambas por sus referencias exactas.
type ConcesionBorradorDurable struct {
	bloqueoSerializacionDiario
	Evidencia  puertosvec.EvidenciaUsoDecisionAutorizacion
	Atestacion ProyeccionAtestacionPDP
}

func nuevaProyeccionDecisionDiario(
	evidencia puertosvec.EvidenciaUsoDecisionAutorizacion,
	material puertosbolsa.MaterialIntencionGobiernoConvocatoria,
	version dominiobolsa.VersionConvocatoriaGobernada,
	actor dominiovec.ContextoActor,
	correlacionRef string,
	instante time.Time,
	atestacion ProyeccionAtestacionPDP,
) (ProyeccionDecisionDiario, error) {
	datos, errDatos := evidencia.Datos()
	_, errActor := actor.Clonar()
	recurso, errRecurso := puertosbolsa.RecursoAutorizableMutacionConvocatoria(material, version)
	huellaContexto, errHuella := recurso.HuellaContextoAutorizacionSHA256()
	decision := datos.Decision
	camposValidos := mismosCamposDecision(decision.CamposPermitidos,
		[]string{"auditoria", "evento_outbox", "version_convocatoria"})
	if errDatos != nil || errRecurso != nil || errHuella != nil ||
		evidencia.ValidarEn(instante) != nil || !instanteOperacionCanonico(instante) ||
		errActor != nil || decision.PrincipalID != actor.PersonaRef ||
		decision.PerfilActivoRef != actor.PerfilActivoRef || decision.CorrelacionRef != correlacionRef ||
		decision.Accion != material.Accion || decision.RecursoRef != recurso.Referencia ||
		decision.ModuloID != puertosbolsa.ModuloGobiernoConvocatorias ||
		decision.TipoRecurso != puertosbolsa.TipoRecursoVersionConvocatoriaGobernada ||
		decision.ContextoRecursoHuellaSHA256 != huellaContexto ||
		decision.Finalidad != puertosbolsa.FinalidadGobiernoConvocatorias || !camposValidos ||
		len(decision.Obligaciones) != 0 || decision.GarantiaMinima != dominiovec.AuthAssuranceHigh ||
		decision.RevisionCatalogoPoliticas == 0 || decision.ControlVigenciaVersionRolRevision == 0 ||
		!atestacion.validaPara(decision.DecisionRef, datos.VerificadaEn) ||
		atestacion.VerificadaEn.Before(decision.EmitidaEn) {
		return ProyeccionDecisionDiario{}, ErrReservaBorradorInvalida
	}
	proyeccion := ProyeccionDecisionDiario{
		EsquemaHuella: datos.EsquemaHuella, DecisionRef: decision.DecisionRef,
		HuellaDecisionSHA256: datos.HuellaDecisionSHA256, Accion: decision.Accion,
		RecursoRef: decision.RecursoRef, ModuloID: decision.ModuloID,
		TipoRecurso:                 decision.TipoRecurso,
		ContextoRecursoHuellaSHA256: decision.ContextoRecursoHuellaSHA256,
		Finalidad:                   decision.Finalidad,
		AsignacionRef:               decision.AsignacionRef,
		AsignacionHuellaSHA256:      decision.AsignacionHuellaSHA256,
		VersionRolRef:               decision.VersionRolRef, VersionRolHuellaSHA256: decision.VersionRolHuellaSHA256,
		ControlVigenciaVersionRolRef:          decision.ControlVigenciaVersionRolRef,
		ControlVigenciaVersionRolRevision:     decision.ControlVigenciaVersionRolRevision,
		ControlVigenciaVersionRolHuellaSHA256: decision.ControlVigenciaVersionRolHuellaSHA256,
		RevisionCatalogoPoliticas:             decision.RevisionCatalogoPoliticas,
		CatalogoPoliticasHuellaSHA256:         decision.CatalogoPoliticasHuellaSHA256,
		EmitidaEn:                             decision.EmitidaEn,
		VerificadaEn:                          datos.VerificadaEn, ValidaHasta: decision.ValidaHasta,
		AtestacionPDP: atestacion,
	}
	if !proyeccion.valida() {
		return ProyeccionDecisionDiario{}, ErrReservaBorradorInvalida
	}
	return proyeccion, nil
}

func mismosCamposDecision(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	vistos := make(map[string]int, len(a))
	for _, valor := range a {
		vistos[valor]++
	}
	for _, valor := range b {
		vistos[valor]--
	}
	for _, cantidad := range vistos {
		if cantidad != 0 {
			return false
		}
	}
	return true
}

// ProyeccionReservaDecision es el unico valor que cruza al diario durable.
// Clave cliente, actor/principal y motivo en claro son imposibles de expresar.
type ProyeccionReservaDecision struct {
	bloqueoSerializacionDiario
	Identidad             ProyeccionIdentidadOperacion
	Accion                string
	Decision              ProyeccionDecisionDiario
	ArrendamientoIniciaEn time.Time
	ArrendamientoVenceEn  time.Time
}

// SolicitudReservaDecisionBorrador es la orden efimera de la barrera durable.
// Permite releer y verificar material, agregado, recurso, decision y atestacion
// antes del INSERT CAS. Solo Proyeccion puede persistirse en el diario.
type SolicitudReservaDecisionBorrador struct {
	bloqueoSerializacionDiario
	Proyeccion          ProyeccionReservaDecision
	IdentidadesConsulta []ProyeccionIdentidadOperacion
	Intencion           IntencionBorradorCanonica
	Plantilla           *PlantillaBorradorResuelta
	Material            puertosbolsa.MaterialIntencionGobiernoConvocatoria
	Version             dominiobolsa.VersionConvocatoriaGobernada
	Recurso             dominiovec.RecursoAutorizable
	Actor               dominiovec.ContextoActor
	CorrelacionRef      string
	Concesion           ConcesionBorradorDurable
	SolicitadaEn        time.Time
}

func (s SolicitudReservaDecisionBorrador) Validar() error {
	return s.validar(true)
}

func (s SolicitudReservaDecisionBorrador) validar(requierePrimaria bool) error {
	estado, errEstado := puertosbolsa.EstadoVersionConvocatoria(s.Version)
	recursoEsperado, errRecurso := puertosbolsa.RecursoAutorizableMutacionConvocatoria(
		s.Material, s.Version,
	)
	decisionEsperada, errDecision := nuevaProyeccionDecisionDiario(
		s.Concesion.Evidencia, s.Material, s.Version, s.Actor, s.CorrelacionRef,
		s.SolicitadaEn, s.Concesion.Atestacion,
	)
	if errEstado != nil || errRecurso != nil || errDecision != nil ||
		s.Material.Validar() != nil || s.Version.Validar() != nil ||
		!s.Intencion.coincideEjecucion(s.Version, s.Material, s.Plantilla) ||
		estado != s.Material.EstadoPrincipalNuevo ||
		!recursosReservaExactos(recursoEsperado, s.Recurso) ||
		!instanteOperacionCanonico(s.SolicitadaEn) ||
		!s.SolicitadaEn.Equal(s.Proyeccion.ArrendamientoIniciaEn) ||
		!s.Proyeccion.valida() || s.Proyeccion.Accion != s.Material.Accion ||
		s.Proyeccion.Decision != decisionEsperada ||
		!identidadesConsultaValidas(s.IdentidadesConsulta) ||
		!identidadIncluidaExactamente(s.Proyeccion.Identidad, s.IdentidadesConsulta) ||
		requierePrimaria && !identidadesProyectadasCoinciden(
			s.Proyeccion.Identidad, s.IdentidadesConsulta[0],
		) {
		return ErrReservaBorradorInvalida
	}
	return nil
}

func materialesReservaExactos(
	a, b puertosbolsa.MaterialIntencionGobiernoConvocatoria,
) bool {
	return a.Esquema == b.Esquema && a.Accion == b.Accion &&
		referenciasEstadoReservaExactas(a.EstadoPrincipalEsperado, b.EstadoPrincipalEsperado) &&
		a.EstadoPrincipalNuevo == b.EstadoPrincipalNuevo &&
		referenciasEstadoReservaExactas(a.EstadoRelacionadoEsperado, b.EstadoRelacionadoEsperado) &&
		referenciasEstadoReservaExactas(a.EstadoRelacionadoNuevo, b.EstadoRelacionadoNuevo) &&
		a.DominioCriptograficoMotivo == b.DominioCriptograficoMotivo &&
		a.GeneracionClaveMotivo == b.GeneracionClaveMotivo &&
		coincideTextoConstante(a.HuellaMotivoHMACSHA256, b.HuellaMotivoHMACSHA256)
}

func referenciasEstadoReservaExactas(
	a, b *puertosbolsa.ReferenciaEstadoVersionConvocatoria,
) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func recursosReservaExactos(a, b dominiovec.RecursoAutorizable) bool {
	if a.Referencia != b.Referencia || a.ModuloID != b.ModuloID || a.Tipo != b.Tipo ||
		len(a.Ambitos) != len(b.Ambitos) || len(a.Atributos) != len(b.Atributos) {
		return false
	}
	for clave, valor := range a.Ambitos {
		if b.Ambitos[clave] != valor {
			return false
		}
	}
	for clave, valor := range a.Atributos {
		if b.Atributos[clave] != valor {
			return false
		}
	}
	return true
}

func nuevaProyeccionReservaDecision(
	identidad ProyeccionIdentidadOperacion,
	accion string,
	decision ProyeccionDecisionDiario,
	arrendamiento ArrendamientoDiario,
) (ProyeccionReservaDecision, error) {
	resultado := ProyeccionReservaDecision{
		Identidad: identidad, Accion: accion,
		Decision: decision, ArrendamientoIniciaEn: arrendamiento.iniciaEn,
		ArrendamientoVenceEn: arrendamiento.venceEn,
	}
	if !resultado.valida() {
		return ProyeccionReservaDecision{}, ErrReservaBorradorInvalida
	}
	return resultado, nil
}

func (p ProyeccionReservaDecision) valida() bool {
	return p.Identidad.valida() &&
		(p.Accion == puertosbolsa.AccionCrearBorradorConvocatoria ||
			p.Accion == puertosbolsa.AccionActualizarBorradorConvocatoria) &&
		p.Decision.valida() && p.Decision.Accion == p.Accion &&
		!p.ArrendamientoIniciaEn.Before(p.Decision.VerificadaEn) &&
		!p.ArrendamientoVenceEn.After(p.Decision.ValidaHasta) &&
		instanteOperacionCanonico(p.ArrendamientoIniciaEn) &&
		instanteOperacionCanonico(p.ArrendamientoVenceEn) &&
		p.ArrendamientoVenceEn.After(p.ArrendamientoIniciaEn) &&
		p.ArrendamientoVenceEn.Sub(p.ArrendamientoIniciaEn) <= DuracionMaximaArrendamientoDiario
}

func (p ProyeccionReservaDecision) Validar() error {
	if !p.valida() {
		return ErrReservaBorradorInvalida
	}
	return nil
}

type EstadoResultadoDiario string

const (
	ResultadoDiarioAusente       EstadoResultadoDiario = "ausente"
	ResultadoDiarioReservado     EstadoResultadoDiario = "reservado"
	ResultadoDiarioEnCurso       EstadoResultadoDiario = "en_curso"
	ResultadoDiarioIndeterminado EstadoResultadoDiario = "indeterminado"
	ResultadoDiarioConfirmado    EstadoResultadoDiario = "confirmado"
	ResultadoDiarioNoAplicado    EstadoResultadoDiario = "no_aplicado"
	ResultadoDiarioConflicto     EstadoResultadoDiario = "conflicto"
)

const esquemaReciboBorradorV2 = "bolsa.convocatoria.borrador.recibo.v2"

// ProyeccionReciboBorrador permite aplicar el mismo veredicto temporal en la
// respuesta inmediata y en replay. Liga L/F, PDP, sellado, lease y fence sin
// principal, perfil, motivo ni clave cliente en claro.
type ProyeccionReciboBorrador struct {
	bloqueoSerializacionDiario
	Esquema                  string
	ReciboRef                string
	TransaccionRef           string
	Accion                   string
	EstadoPrincipal          puertosbolsa.ReferenciaEstadoVersionConvocatoria
	Identidad                ProyeccionIdentidadOperacion
	Decision                 ProyeccionDecisionDiario
	SelladoMotivo            ProyeccionSelladoMotivoBorrador
	RevisionConfirmada       uint64
	CercadoConfirmado        uint64
	ArrendamientoIniciaEn    time.Time
	ArrendamientoVenceEn     time.Time
	AuditoriaRef             string
	HuellaAuditoriaSHA256    string
	EventoOutboxRef          string
	HuellaEventoOutboxSHA256 string
	ConfirmadaEn             time.Time
}

type ResultadoOperacionDiario struct {
	bloqueoSerializacionDiario
	Estado                EstadoResultadoDiario
	Revision              uint64
	Cercado               uint64
	ArrendamientoIniciaEn time.Time
	ArrendamientoVenceEn  time.Time
	Recibo                *ProyeccionReciboBorrador
}

// DiarioOperacionesBorrador es el limite PostgreSQL del diario. Cada metodo
// usa reloj de base de datos. Reserva comprueba todas las generaciones y solo
// inserta la primaria; reconciliacion demuestra COMMIT/ROLLBACK y reclamacion
// aplica CAS. Cerrar como no_aplicado y reclamar elevan revision+cercado; un
// COMMIT eleva revision pero conserva el cercado del epoch propietario.
type DiarioOperacionesBorrador interface {
	ConsultarIdentidades(context.Context, SolicitudConsultaIdentidadesBorrador) (ResultadoConsultaIdentidadesBorrador, error)
	ReservarDecision(context.Context, SolicitudReservaDecisionBorrador) (ResultadoReservaDecisionBorrador, error)
	Reconciliar(context.Context, SolicitudReconciliacionBorrador) (ResultadoReconciliacionBorrador, error)
	ReclamarDecision(context.Context, SolicitudReclamacionDecisionBorrador) (ResultadoOperacionDiario, error)
}

type DerivadorIdentidadOperacion interface {
	Derivar(context.Context, SolicitudDerivacionIdempotencia) (ConjuntoIdentidadesOperacion, error)
}

type PreparadorAltaBorrador interface {
	ResolverPlantillaBorrador(
		context.Context,
		SelectorPlantillaBorrador,
		time.Time,
	) (PlantillaBorradorResuelta, error)
	PrepararAltaBorrador(
		context.Context,
		PlantillaBorradorResuelta,
		string,
		string,
		time.Time,
	) (PreparacionAltaBorrador, error)
}

type EstadoEvaluacionPDPBorrador string

const (
	EvaluacionPDPConcedida EstadoEvaluacionPDPBorrador = "concedida"
	EvaluacionPDPDenegada  EstadoEvaluacionPDPBorrador = "denegada"
)

type ResultadoEvaluacionPDPBorrador struct {
	bloqueoSerializacionDiario
	Estado        EstadoEvaluacionPDPBorrador
	Concesion     ConcesionBorradorDurable
	DenegacionRef string
}

// ResolvedorMotivoBorrador relee el catalogo vigente y devuelve exactamente
// la referencia versionada y publicada. El texto libre no forma parte de la
// orden de este caso de uso.
type ResolvedorMotivoBorrador interface {
	ResolverMotivoBorrador(
		context.Context,
		dominiovec.ReferenciaEntradaCatalogo,
		time.Time,
	) (dominiovec.ReferenciaEntradaCatalogo, error)
}

type LectorBorradorExacto interface {
	ObtenerBorradorExacto(context.Context, puertosbolsa.ReferenciaEstadoVersionConvocatoria) (dominiobolsa.VersionConvocatoriaGobernada, error)
}

// AutorizadorIntencionBorrador debe registrar durablemente la decision exacta
// antes de devolver la evidencia. La evidencia sola no acredita procedencia;
// el diario la relee dentro de ReservarDecision.
type AutorizadorIntencionBorrador interface {
	EvaluarDecisionBorrador(
		context.Context,
		dominiovec.ContextoActor,
		dominiovec.VinculoAutenticacionActorV1,
		dominiovec.RecursoAutorizable,
		string,
		dominiovec.ReferenciaEntradaCatalogo,
		IntencionBorradorCanonica,
		time.Time,
	) (ResultadoEvaluacionPDPBorrador, error)
}

// ProyeccionSelladoMotivoBorrador identifica la atestacion HSM/KMS posterior a
// la concesion. No contiene principal, motivo en claro ni huella SHA semantica.
type ProyeccionSelladoMotivoBorrador struct {
	bloqueoSerializacionDiario
	Accion                 string
	ConvocatoriaRef        string
	HMAC                   puertosbolsa.ProyeccionHMACMotivoGobiernoConvocatoriaDurable
	AtestacionRef          string
	VersionAtestacion      uint32
	EstadoAtestacion       string
	HuellaAtestacionSHA256 string
	TokenConsumoRef        string
	MaterializadorRef      string
	AtestacionEmitidaEn    time.Time
	AtestacionValidaHasta  time.Time
}

func (p ProyeccionSelladoMotivoBorrador) validaPara(
	material puertosbolsa.MaterialIntencionGobiernoConvocatoria,
	instante time.Time,
) bool {
	representacion := "hmac-sha256:" + p.HMAC.ClaveHMACRef + ":" + p.HMAC.ValorHMACSHA256
	return p.validaEstructural() && p.Accion == material.Accion &&
		p.ConvocatoriaRef == material.EstadoPrincipalNuevo.Referencia &&
		p.HMAC.DominioCriptografico == material.DominioCriptograficoMotivo &&
		p.HMAC.GeneracionClave == material.GeneracionClaveMotivo &&
		coincideTextoConstante(representacion, material.HuellaMotivoHMACSHA256) &&
		!instante.Before(p.AtestacionEmitidaEn) && instante.Before(p.AtestacionValidaHasta)
}

func (p ProyeccionSelladoMotivoBorrador) validaEstructural() bool {
	return p.HMAC.Validar() == nil &&
		(p.Accion == puertosbolsa.AccionCrearBorradorConvocatoria ||
			p.Accion == puertosbolsa.AccionActualizarBorradorConvocatoria) &&
		referenciaProyeccionValida(p.ConvocatoriaRef) &&
		referenciaProyeccionValida(p.AtestacionRef) && p.VersionAtestacion > 0 &&
		p.EstadoAtestacion == "verificada" && huellaHexValida(p.HuellaAtestacionSHA256) &&
		referenciaProyeccionValida(p.TokenConsumoRef) && referenciaProyeccionValida(p.MaterializadorRef) &&
		p.AtestacionRef != p.TokenConsumoRef && p.AtestacionRef != p.MaterializadorRef &&
		instanteOperacionCanonico(p.AtestacionEmitidaEn) && instanteOperacionCanonico(p.AtestacionValidaHasta) &&
		p.AtestacionValidaHasta.After(p.AtestacionEmitidaEn) &&
		p.AtestacionValidaHasta.Sub(p.AtestacionEmitidaEn) <=
			puertosbolsa.VigenciaMaximaAtestacionMotivoGobiernoConvocatoria
}

// SolicitudSelladoMotivoBorrador es efimera y solo nace tras la reserva
// post-PDP. El adaptador HSM relee decision, atestacion y generacion de clave.
type SolicitudSelladoMotivoBorrador struct {
	bloqueoSerializacionDiario
	Reserva        ProyeccionReservaDecision
	Control        ResultadoOperacionDiario
	Version        dominiobolsa.VersionConvocatoriaGobernada
	Material       puertosbolsa.MaterialIntencionGobiernoConvocatoria
	Compromiso     puertosbolsa.CompromisoMotivoGobiernoConvocatoria
	Actor          dominiovec.ContextoActor
	CorrelacionRef string
	Concesion      ConcesionBorradorDurable
	SolicitadaEn   time.Time
}

func (s SolicitudSelladoMotivoBorrador) valida() bool {
	datosCompromiso, errCompromiso := s.Compromiso.DatosParaMaterial()
	datosDecision, errDecision := s.Concesion.Evidencia.Datos()
	hmacDurable, errHMAC := datosCompromiso.HMAC.ProyeccionDurable()
	decisionEsperada, errProyeccion := nuevaProyeccionDecisionDiario(
		s.Concesion.Evidencia, s.Material, s.Version, s.Actor, s.CorrelacionRef,
		s.SolicitadaEn, s.Concesion.Atestacion,
	)
	representacion := "hmac-sha256:" + hmacDurable.ClaveHMACRef + ":" + hmacDurable.ValorHMACSHA256
	return errCompromiso == nil && errDecision == nil && errHMAC == nil && errProyeccion == nil &&
		s.Material.Validar() == nil && s.Version.Validar() == nil &&
		s.Reserva.valida() && s.Reserva.Decision == decisionEsperada &&
		s.Reserva.Accion == s.Material.Accion && s.Control.Estado == ResultadoDiarioReservado &&
		s.Control.Revision > 0 && s.Control.Cercado > 0 &&
		s.Control.ArrendamientoIniciaEn.Equal(s.Reserva.ArrendamientoIniciaEn) &&
		s.Control.ArrendamientoVenceEn.Equal(s.Reserva.ArrendamientoVenceEn) &&
		datosCompromiso.Accion == s.Material.Accion &&
		datosCompromiso.ConvocatoriaRef == s.Material.EstadoPrincipalNuevo.Referencia &&
		datosCompromiso.PrincipalRef == datosDecision.Decision.PrincipalID &&
		datosCompromiso.CorrelacionRef == datosDecision.Decision.CorrelacionRef &&
		hmacDurable.DominioCriptografico == s.Material.DominioCriptograficoMotivo &&
		hmacDurable.GeneracionClave == s.Material.GeneracionClaveMotivo &&
		coincideTextoConstante(representacion, s.Material.HuellaMotivoHMACSHA256) &&
		instanteOperacionCanonico(s.SolicitadaEn) &&
		!s.SolicitadaEn.Before(s.Reserva.ArrendamientoIniciaEn) &&
		s.SolicitadaEn.Before(s.Reserva.ArrendamientoVenceEn)
}

func (s SolicitudSelladoMotivoBorrador) Validar() error {
	if !s.valida() {
		return ErrReservaBorradorInvalida
	}
	return nil
}

type SelladorMotivoBorrador interface {
	VerificarYSellarMotivo(
		context.Context,
		SolicitudSelladoMotivoBorrador,
	) (ProyeccionSelladoMotivoBorrador, error)
}

// SolicitudConfirmacionBorrador entrega al unico limite transaccional el
// agregado canonico, material V2, concesion/atestacion, L/F y cercado. El
// diario solo persiste sus proyecciones seudonimizadas; el agregado pertenece
// al almacen de negocio cifrado y conserva su trazabilidad legal.
type SolicitudConfirmacionBorrador struct {
	bloqueoSerializacionDiario
	Reserva        ProyeccionReservaDecision
	Control        ResultadoOperacionDiario
	Version        dominiobolsa.VersionConvocatoriaGobernada
	Material       puertosbolsa.MaterialIntencionGobiernoConvocatoria
	Actor          dominiovec.ContextoActor
	CorrelacionRef string
	Concesion      ConcesionBorradorDurable
	SelladoMotivo  ProyeccionSelladoMotivoBorrador
	SolicitadaEn   time.Time
}

func (s SolicitudConfirmacionBorrador) valida() bool {
	estado, err := puertosbolsa.EstadoVersionConvocatoria(s.Version)
	datosDecision, errDecision := s.Concesion.Evidencia.Datos()
	decisionEsperada, errProyeccion := nuevaProyeccionDecisionDiario(
		s.Concesion.Evidencia, s.Material, s.Version, s.Actor, s.CorrelacionRef,
		s.SolicitadaEn, s.Concesion.Atestacion,
	)
	return err == nil && errDecision == nil && s.Version.Validar() == nil &&
		errProyeccion == nil && s.Reserva.Decision == decisionEsperada &&
		s.Material.Validar() == nil && estado == s.Material.EstadoPrincipalNuevo &&
		s.Reserva.valida() && s.Reserva.Accion == s.Material.Accion &&
		s.Control.Estado == ResultadoDiarioReservado && s.Control.Revision > 0 &&
		s.Control.Cercado > 0 && s.Control.ArrendamientoIniciaEn.Equal(s.Reserva.ArrendamientoIniciaEn) &&
		s.Control.ArrendamientoVenceEn.Equal(s.Reserva.ArrendamientoVenceEn) &&
		s.Reserva.Decision.DecisionRef == datosDecision.Decision.DecisionRef &&
		s.Reserva.Decision.HuellaDecisionSHA256 == datosDecision.HuellaDecisionSHA256 &&
		s.SelladoMotivo.validaPara(s.Material, s.SolicitadaEn) &&
		instanteOperacionCanonico(s.SolicitadaEn) &&
		!s.SolicitadaEn.Before(s.Reserva.ArrendamientoIniciaEn) &&
		s.SolicitadaEn.Before(s.Reserva.ArrendamientoVenceEn)
}

func (s SolicitudConfirmacionBorrador) Validar() error {
	if !s.valida() {
		return ErrResultadoBorradorInseguro
	}
	return nil
}

type ConfirmadorAtomicoBorrador interface {
	// ConfirmarBorrador bloquea y relee diario, decision/atestacion y agregado;
	// aplica CAS sobre revision+cercado y confirma agregado, consumo de sellado,
	// auditoria, outbox, recibo y estado terminal en un unico COMMIT. Un cercado
	// obsoleto nunca produce efecto; los recibos terminales se recuperan por la
	// consulta idempotente, no reutilizando una orden de confirmacion obsoleta.
	ConfirmarBorrador(
		context.Context,
		SolicitudConfirmacionBorrador,
	) (ResultadoConfirmacionAtomica, error)
}

// ResultadoConfirmacionAtomica obliga al adaptador a distinguir COMMIT,
// ROLLBACK demostrado y desenlace desconocido. Un error de transporte sin un
// estado valido se trata siempre como indeterminado.
type ResultadoConfirmacionAtomica struct {
	bloqueoSerializacionDiario
	Estado EstadoResultadoDiario
	Recibo ProyeccionReciboBorrador
}

func (p ProyeccionDecisionDiario) valida() bool {
	return p.EsquemaHuella == puertosvec.EsquemaHuellaDecisionAutorizacionReforzadaV1 &&
		referenciaProyeccionValida(p.DecisionRef) && huellaHexValida(p.HuellaDecisionSHA256) &&
		(p.Accion == puertosbolsa.AccionCrearBorradorConvocatoria ||
			p.Accion == puertosbolsa.AccionActualizarBorradorConvocatoria) &&
		referenciaProyeccionValida(p.RecursoRef) &&
		p.ModuloID == puertosbolsa.ModuloGobiernoConvocatorias &&
		p.TipoRecurso == puertosbolsa.TipoRecursoVersionConvocatoriaGobernada &&
		huellaHexValida(p.ContextoRecursoHuellaSHA256) &&
		p.Finalidad == puertosbolsa.FinalidadGobiernoConvocatorias &&
		referenciaProyeccionValida(p.AsignacionRef) && huellaHexValida(p.AsignacionHuellaSHA256) &&
		referenciaProyeccionValida(p.VersionRolRef) && huellaHexValida(p.VersionRolHuellaSHA256) &&
		referenciaProyeccionValida(p.ControlVigenciaVersionRolRef) &&
		p.ControlVigenciaVersionRolRevision > 0 &&
		huellaHexValida(p.ControlVigenciaVersionRolHuellaSHA256) &&
		p.RevisionCatalogoPoliticas > 0 &&
		huellaHexValida(p.CatalogoPoliticasHuellaSHA256) &&
		instanteOperacionCanonico(p.EmitidaEn) && instanteOperacionCanonico(p.VerificadaEn) &&
		instanteOperacionCanonico(p.ValidaHasta) && !p.VerificadaEn.Before(p.EmitidaEn) &&
		p.ValidaHasta.After(p.VerificadaEn) && p.AtestacionPDP.validaPara(p.DecisionRef, p.VerificadaEn) &&
		p.DecisionRef != p.VersionRolRef && p.DecisionRef != p.AtestacionPDP.AtestacionRef &&
		p.VersionRolRef != p.AtestacionPDP.AtestacionRef &&
		p.AsignacionRef != p.DecisionRef && p.ControlVigenciaVersionRolRef == p.VersionRolRef
}

func (p ProyeccionDecisionDiario) Validar() error {
	if !p.valida() {
		return ErrReservaBorradorInvalida
	}
	return nil
}

func referenciaProyeccionValida(valor string) bool {
	if valor == "" || len(valor) > 512 || valor != strings.TrimSpace(valor) ||
		!utf8.ValidString(valor) || strings.ContainsRune(valor, '*') {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) || unicode.IsSpace(caracter) ||
			unicode.Is(unicode.Bidi_Control, caracter) || caracter == unicode.ReplacementChar {
			return false
		}
	}
	return true
}

func huellaHexValida(valor string) bool {
	if len(valor) != sha256.Size*2 || valor != strings.ToLower(valor) {
		return false
	}
	bytes, err := hex.DecodeString(valor)
	return err == nil && len(bytes) == sha256.Size
}

func coincideTextoConstante(izquierda, derecha string) bool {
	const maximo = 256
	if len(izquierda) > maximo || len(derecha) > maximo {
		return false
	}
	var a, b [maximo]byte
	copy(a[:], izquierda)
	copy(b[:], derecha)
	coincide := subtle.ConstantTimeEq(int32(len(izquierda)), int32(len(derecha)))
	coincide &= subtle.ConstantTimeCompare(a[:], b[:])
	return coincide == 1
}
