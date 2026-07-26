package postgres

import (
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	esquemaCargaDecisionCoberturaO404E = "" +
		"vec.contratacion-temporal.confirmar-operacion-" +
		"decision-cobertura.o4-04e.v1"
	esquemaReciboDecisionCoberturaO404E = "" +
		"vec.contratacion-temporal.recibo-operacion-" +
		"decision-cobertura.o4-04e.v1"
	// Ocho MiB de material canónico permiten 512 consumos reales sin aceptar
	// el peor caso teórico de 224 MiB. Su representación hexadecimal ocupa
	// como máximo 16 MiB; se reservan otros ocho para estructura y agregados.
	maximoBytesMaterialCanonicoDecisionCoberturaO404E = 8 * 1024 * 1024
	maximoBytesCargaDecisionCoberturaO404E            = 24 * 1024 * 1024
	maximoBytesReciboDecisionCoberturaO404E           = 256 * 1024
	maximoBytesCanonVECDecisionCoberturaO404E         = 1024 * 1024
	maximoBytesPruebaC1DecisionCoberturaO404E         = 64 * 1024
)

// Los tipos JSON O4-04E son cerrados deliberadamente: ni map[string]any ni
// serialización directa de las capacidades opacas del núcleo.
type cargaConfirmarDecisionCoberturaO404E struct {
	Esquema     string                                            `json:"esquema"`
	Rama        cobertura.RamaSesionTCBOperacionDecisionCobertura `json:"rama"`
	Cabecera    cabeceraDecisionCoberturaO404E                    `json:"cabecera"`
	Gobierno    *gobiernoDecisionCoberturaO404E                   `json:"gobierno"`
	DecisionVEC decisionVECDecisionCoberturaO404E                 `json:"decision_vec"`
	ConsumosC1  []consumoC1DecisionCoberturaO404E                 `json:"consumos_c1"`
	Concesion   *concesionDecisionCoberturaO404E                  `json:"concesion"`
	Denegacion  *denegacionDecisionCoberturaO404E                 `json:"denegacion"`
}

type cabeceraDecisionCoberturaO404E struct {
	Esquema                      string    `json:"esquema_sesion"`
	HuellaOrdenSHA256            string    `json:"huella_orden_sha256"`
	OrganizacionRef              string    `json:"organizacion_ref"`
	ExpedienteRef                string    `json:"expediente_ref"`
	VersionExpediente            uint64    `json:"version_expediente"`
	ReservaRef                   string    `json:"reserva_ref"`
	ReciboRef                    string    `json:"recibo_ref"`
	ActuacionRef                 string    `json:"actuacion_ref"`
	AuditoriaRef                 string    `json:"auditoria_ref"`
	EventoRef                    string    `json:"evento_ref"`
	CorrelacionVECRef            string    `json:"correlacion_vec_ref"`
	DecisionVECRef               string    `json:"decision_vec_ref"`
	AnalisisRef                  string    `json:"analisis_ref"`
	AnalisisHuellaSHA256         string    `json:"analisis_huella_sha256"`
	TokenPropietarioSHA256       string    `json:"token_propietario_sha256"`
	AmbitoIdempotenciaHMAC       string    `json:"ambito_idempotencia_hmac"`
	HuellaSemanticaHMAC          string    `json:"huella_semantica_hmac"`
	RevisionCercadoAnterior      uint64    `json:"revision_cercado_anterior"`
	RevisionCercado              uint64    `json:"revision_cercado"`
	ObservadaEnDB                time.Time `json:"observada_en_db"`
	PropiedadHasta               time.Time `json:"propiedad_hasta"`
	ValidaHastaOrden             time.Time `json:"valida_hasta_orden"`
	PreparacionC1Ref             string    `json:"preparacion_c1_ref"`
	PreparacionC1HuellaSHA256    string    `json:"preparacion_c1_huella_sha256"`
	PreparacionC1PreparadaEn     time.Time `json:"preparacion_c1_preparada_en"`
	PreparacionC1ValidaHasta     time.Time `json:"preparacion_c1_valida_hasta"`
	NumeroConsumosC1             uint64    `json:"numero_consumos_c1"`
	HuellaOrdenesConsumoC1SHA256 string    `json:"huella_ordenes_consumo_c1_sha256"`
}

type gobiernoDecisionCoberturaO404E struct {
	Catalogo           domain.PublicacionCatalogoViasCobertura         `json:"catalogo"`
	Politica           domain.PublicacionPoliticaDecisionCobertura     `json:"politica"`
	PoliticaActuacion  cobertura.PublicacionPoliticaActuacionCobertura `json:"politica_actuacion"`
	Accion             domain.ClaveCatalogo                            `json:"accion"`
	FinalidadCTClave   domain.ClaveCatalogo                            `json:"finalidad_ct_clave"`
	FinalidadCTRef     string                                          `json:"finalidad_ct_ref"`
	FinalidadVEC       domain.ClaveCatalogo                            `json:"finalidad_vec"`
	UnidadEjecutoraRef string                                          `json:"unidad_ejecutora_ref"`
	FaseDestino        domain.ClaveFase                                `json:"fase_destino"`
	EstadoDestino      domain.EstadoOperativo                          `json:"estado_destino"`
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo            `json:"motivo_autorizacion"`
	EvaluadaEn         time.Time                                       `json:"evaluada_en"`
	ValidaHasta        time.Time                                       `json:"valida_hasta"`
}

type decisionVECDecisionCoberturaO404E struct {
	DecisionCanonica            []byte            `json:"-"`
	MotivoCanonico              []byte            `json:"-"`
	PersonaVersion              uint64            `json:"persona_version"`
	PerfilVersion               uint64            `json:"perfil_version"`
	DecisionRef                 string            `json:"decision_ref"`
	DecisionHuellaSHA256        string            `json:"decision_huella_sha256"`
	CodigoProbatorio            string            `json:"codigo_probatorio"`
	Concedida                   bool              `json:"concedida"`
	EmitidaEn                   time.Time         `json:"emitida_en"`
	ValidaHasta                 time.Time         `json:"valida_hasta"`
	PrincipalID                 string            `json:"principal_id"`
	PerfilActivoRef             string            `json:"perfil_activo_ref"`
	Accion                      string            `json:"accion"`
	RecursoRef                  string            `json:"recurso_ref"`
	RecursoModulo               string            `json:"recurso_modulo"`
	RecursoTipo                 string            `json:"recurso_tipo"`
	Ambitos                     map[string]string `json:"ambitos"`
	Atributos                   map[string]string `json:"atributos"`
	ContextoRecursoHuellaSHA256 string            `json:"contexto_recurso_huella_sha256"`
	Finalidad                   string            `json:"finalidad"`
	CorrelacionRef              string            `json:"correlacion_ref"`
}

type consumoC1DecisionCoberturaO404E struct {
	Posicion uint64 `json:"posicion"`
	Total    uint64 `json:"total"`

	PeticionRef            string                                   `json:"peticion_ref"`
	OrganizacionRef        string                                   `json:"organizacion_ref"`
	ExpedienteRef          string                                   `json:"expediente_ref"`
	VersionExpediente      uint64                                   `json:"version_expediente"`
	CatalogoRef            string                                   `json:"catalogo_ref"`
	CatalogoVersion        uint64                                   `json:"catalogo_version"`
	CatalogoHuellaSHA256   string                                   `json:"catalogo_huella_sha256"`
	ViaClave               domain.ClaveCatalogo                     `json:"via_clave"`
	ComprobacionClave      domain.ClaveCatalogo                     `json:"comprobacion_clave"`
	ComprobacionResultado  domain.ResultadoComprobacion             `json:"comprobacion_resultado"`
	ComprobacionFuenteRef  string                                   `json:"comprobacion_fuente_ref"`
	ComprobacionReciboRef  string                                   `json:"comprobacion_recibo_ref"`
	ComprobacionEvaluadaEn time.Time                                `json:"comprobacion_evaluada_en"`
	OrdenComprobacion      uint16                                   `json:"orden_comprobacion"`
	Obligatoria            bool                                     `json:"obligatoria"`
	ProcedenciaClave       domain.ClaveCatalogo                     `json:"procedencia_clave"`
	DefinicionFuenteRef    string                                   `json:"definicion_fuente_ref"`
	CategoriaRef           string                                   `json:"categoria_ref"`
	Periodo                domain.PeriodoPrevisto                   `json:"periodo"`
	SolicitadaEn           time.Time                                `json:"solicitada_en"`
	EmitidaEn              time.Time                                `json:"emitida_en"`
	ValidaHasta            time.Time                                `json:"valida_hasta"`
	HuellaPeticionSHA256   string                                   `json:"huella_peticion_sha256"`
	HuellaResultadoSHA256  string                                   `json:"huella_resultado_sha256"`
	HuellaRespuestaSHA256  string                                   `json:"huella_respuesta_sha256"`
	AutoridadRef           string                                   `json:"autoridad_ref"`
	Generacion             uint32                                   `json:"generacion"`
	ReciboRespuestaRef     string                                   `json:"recibo_respuesta_ref"`
	VerificadorRef         string                                   `json:"verificador_ref"`
	PublicadorCatalogoRef  string                                   `json:"publicador_catalogo_ref"`
	Pruebas                pruebasCanonicasC1DecisionCoberturaO404E `json:"-"`
}

type pruebasCanonicasC1DecisionCoberturaO404E struct {
	Peticion        []byte `json:"-"`
	Resultado       []byte `json:"-"`
	Atestacion      []byte `json:"-"`
	ConfirmacionTCB []byte `json:"-"`
	Catalogo        []byte `json:"-"`
	Verificador     []byte `json:"-"`
	Resumen         []byte `json:"-"`
}

type concesionDecisionCoberturaO404E struct {
	AgregadoAnterior  domain.Expediente                          `json:"agregado_anterior"`
	AgregadoSiguiente domain.Expediente                          `json:"agregado_siguiente"`
	Propuesta         publicacionPropuestaDecisionCoberturaO404E `json:"propuesta"`
	MotivoFuncional   domain.MotivoGobernadoDecisionCobertura    `json:"motivo_funcional"`
	EfectoEn          time.Time                                  `json:"efecto_en"`
	ValidaHasta       time.Time                                  `json:"valida_hasta"`
}

// ViaPropuesta carece a propósito de omitempty: SQL recibe una forma única
// incluso para propuestas incompletas, conflictivas o sin vía.
type publicacionPropuestaDecisionCoberturaO404E struct {
	Referencia                        string                                       `json:"referencia"`
	HuellaSHA256                      string                                       `json:"huella_sha256"`
	Canon                             domain.CanonHuellaPropuestaDecisionCobertura `json:"canon"`
	OrganizacionRef                   string                                       `json:"organizacion_ref"`
	ExpedienteRef                     string                                       `json:"expediente_ref"`
	VersionExpediente                 uint64                                       `json:"version_expediente"`
	AnalisisRef                       string                                       `json:"analisis_ref"`
	AnalisisHuellaSHA256              string                                       `json:"analisis_huella_sha256"`
	PreparacionEvidenciasRef          string                                       `json:"preparacion_evidencias_ref"`
	PreparacionEvidenciasHuellaSHA256 string                                       `json:"preparacion_evidencias_huella_sha256"`
	Catalogo                          domain.IdentidadCatalogoViasCobertura        `json:"catalogo"`
	Politica                          domain.IdentidadPoliticaDecisionCobertura    `json:"politica"`
	FinalidadClave                    domain.ClaveCatalogo                         `json:"finalidad_clave"`
	FinalidadRef                      string                                       `json:"finalidad_ref"`
	CategoriaRef                      string                                       `json:"categoria_ref"`
	Periodo                           domain.PeriodoPrevisto                       `json:"periodo"`
	GeneradaEn                        time.Time                                    `json:"generada_en"`
	ValidaHasta                       time.Time                                    `json:"valida_hasta"`
	Estado                            domain.EstadoPropuestaDecisionCobertura      `json:"estado"`
	ViaPropuesta                      domain.ClaveCatalogo                         `json:"via_propuesta"`
	Resultados                        []domain.ResultadoAgrupadoPropuestaCobertura `json:"resultados"`
	Evaluaciones                      []domain.EvaluacionViaPropuestaCobertura     `json:"evaluaciones"`
}

type denegacionDecisionCoberturaO404E struct {
	OrganizacionRef     string                               `json:"organizacion_ref"`
	ExpedienteRef       string                               `json:"expediente_ref"`
	VersionExpediente   uint64                               `json:"version_expediente"`
	ReservaRef          string                               `json:"reserva_ref"`
	ReciboRef           string                               `json:"recibo_ref"`
	AuditoriaRef        string                               `json:"auditoria_ref"`
	CorrelacionVECRef   string                               `json:"correlacion_vec_ref"`
	DecisionVECRef      string                               `json:"decision_vec_ref"`
	RevisionCercado     uint64                               `json:"revision_cercado"`
	RecursoRef          string                               `json:"recurso_ref"`
	RecursoModulo       string                               `json:"recurso_modulo"`
	RecursoTipo         string                               `json:"recurso_tipo"`
	Ambitos             map[string]string                    `json:"ambitos"`
	Atributos           map[string]string                    `json:"atributos"`
	RecursoHuellaSHA256 string                               `json:"recurso_huella_sha256"`
	ActorRef            string                               `json:"actor_ref"`
	PerfilRef           string                               `json:"perfil_ref"`
	AccionVEC           domain.ClaveCatalogo                 `json:"accion_vec"`
	FinalidadVEC        domain.ClaveCatalogo                 `json:"finalidad_vec"`
	MotivoVEC           dominiovec.ReferenciaEntradaCatalogo `json:"motivo_vec"`
	LimitePreparacion   time.Time                            `json:"limite_preparacion"`
	ValidaHasta         time.Time                            `json:"valida_hasta"`
	PruebaHuellaSHA256  string                               `json:"prueba_huella_sha256"`
}

type reciboDecisionCoberturaO404E struct {
	Esquema                 string    `json:"esquema"`
	ReciboRef               string    `json:"recibo_ref"`
	ReservaRef              string    `json:"reserva_ref"`
	AuditoriaRef            string    `json:"auditoria_ref"`
	CorrelacionVECRef       string    `json:"correlacion_vec_ref"`
	DecisionVECRef          string    `json:"decision_vec_ref"`
	DecisionVECHuellaSHA256 string    `json:"decision_vec_huella_sha256"`
	CodigoProbatorioVEC     string    `json:"codigo_probatorio_vec"`
	ConcedidaVEC            bool      `json:"concedida_vec"`
	RevisionCercado         uint64    `json:"revision_cercado"`
	AmbitoIdempotenciaHMAC  string    `json:"ambito_idempotencia_hmac"`
	HuellaSemanticaHMAC     string    `json:"huella_semantica_hmac"`
	ConfirmadaEn            time.Time `json:"confirmada_en"`
	Aplicada                bool      `json:"aplicada"`
	DenegadaVEC             bool      `json:"denegada_vec"`
	DecisionCoberturaRef    string    `json:"decision_cobertura_ref"`
	DecisionCoberturaHuella string    `json:"decision_cobertura_huella_sha256"`
	VersionResultante       uint64    `json:"version_resultante"`
	EventoRef               string    `json:"evento_ref"`
	ActuacionRef            string    `json:"actuacion_ref"`
}
