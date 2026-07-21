package postgres

const (
	esquemaConfirmacionBorradorPostgreSQLV2 = "vec.bolsa.convocatoria.confirmacion-borrador.v2"
	esquemaEvidenciaCifradoPostgreSQLV1     = "vec.bolsa.convocatoria.cifrado-kms-persistencia.v1"
	esquemaAADPostgreSQLV1                  = "bolsa.convocatoria.borrador.aad.v1"
)

type proyeccionLigeraBorradorPostgreSQL struct {
	ConvocatoriaID       string   `json:"convocatoria_id"`
	Secuencia            int      `json:"secuencia"`
	Referencia           string   `json:"referencia"`
	Revision             int      `json:"revision"`
	HuellaEstadoSHA256   string   `json:"huella_estado_sha256"`
	CodigoVersionPublica string   `json:"codigo_version_publica"`
	IdentificadorPublico string   `json:"identificador_publico"`
	Titulo               string   `json:"titulo"`
	Tipo                 string   `json:"tipo"`
	Categorias           []string `json:"categorias"`
	ExpedienteRef        string   `json:"expediente_ref"`
	OrganizacionRef      string   `json:"organizacion_ref"`
	UnidadGestionRef     string   `json:"unidad_gestion_ref"`
	NumeroPlazos         int      `json:"numero_plazos"`
	NumeroRequisitos     int      `json:"numero_requisitos"`
	NumeroDocumentos     int      `json:"numero_documentos"`
	NumeroAyudas         int      `json:"numero_ayudas"`
	CreadaEn             string   `json:"creada_en"`
	ActualizadaEn        string   `json:"actualizada_en"`
}

type confirmacionBorradorPostgreSQL struct {
	Esquema          string                             `json:"esquema"`
	Identidad        identidadDiarioPostgreSQL          `json:"identidad"`
	Revision         uint64                             `json:"revision"`
	Cercado          uint64                             `json:"cercado"`
	SelladoMotivo    selladoMotivoReciboPostgreSQL      `json:"sellado_motivo"`
	EnvolturaCifrado map[string]any                     `json:"envoltura_cifrado"`
	ProyeccionLigera proyeccionLigeraBorradorPostgreSQL `json:"proyeccion_ligera"`
	SolicitadaEn     string                             `json:"solicitada_en"`
}

type politicaCifradoBorradorPostgreSQL struct {
	Esquema                 string                        `json:"esquema"`
	DecisionPoliticaRef     string                        `json:"decision_politica_ref"`
	VersionDecisionPolitica uint32                        `json:"version_decision_politica"`
	Estado                  string                        `json:"estado"`
	CatalogoRef             string                        `json:"catalogo_ref"`
	RevisionCatalogo        uint64                        `json:"revision_catalogo"`
	HuellaCatalogoSHA256    string                        `json:"huella_catalogo_sha256"`
	Accion                  string                        `json:"accion"`
	HuellaMaterialSHA256    string                        `json:"huella_material_sha256"`
	Perfil                  perfilCifradoReciboPostgreSQL `json:"perfil"`
	IdentidadPrimaria       identidadDiarioPostgreSQL     `json:"identidad_primaria"`
	Revision                uint64                        `json:"revision"`
	Cercado                 uint64                        `json:"cercado"`
	ArrendamientoIniciaEn   string                        `json:"arrendamiento_inicia_en"`
	ArrendamientoVenceEn    string                        `json:"arrendamiento_vence_en"`
	SolicitadaEn            string                        `json:"solicitada_en"`
	EmitidaEn               string                        `json:"emitida_en"`
	VerificadaEn            string                        `json:"verificada_en"`
	ValidaHasta             string                        `json:"valida_hasta"`
	AutoridadRef            string                        `json:"autoridad_ref"`
	HuellaDecisionSHA256    string                        `json:"huella_decision_sha256"`
}

type evidenciaPerfilBorradorPostgreSQL struct {
	Esquema                      string                        `json:"esquema"`
	EvidenciaRef                 string                        `json:"evidencia_ref"`
	VersionEvidencia             uint32                        `json:"version_evidencia"`
	Estado                       string                        `json:"estado"`
	CatalogoRef                  string                        `json:"catalogo_ref"`
	RevisionCatalogo             uint64                        `json:"revision_catalogo"`
	HuellaCatalogoSHA256         string                        `json:"huella_catalogo_sha256"`
	DecisionPoliticaRef          string                        `json:"decision_politica_ref"`
	VersionDecisionPolitica      uint32                        `json:"version_decision_politica"`
	HuellaDecisionPoliticaSHA256 string                        `json:"huella_decision_politica_sha256"`
	Accion                       string                        `json:"accion"`
	HuellaMaterialSHA256         string                        `json:"huella_material_sha256"`
	Perfil                       perfilCifradoReciboPostgreSQL `json:"perfil"`
	IdentidadPrimaria            identidadDiarioPostgreSQL     `json:"identidad_primaria"`
	Revision                     uint64                        `json:"revision"`
	Cercado                      uint64                        `json:"cercado"`
	ArrendamientoIniciaEn        string                        `json:"arrendamiento_inicia_en"`
	ArrendamientoVenceEn         string                        `json:"arrendamiento_vence_en"`
	SolicitudResolucionEn        string                        `json:"solicitud_resolucion_en"`
	EmitidaEn                    string                        `json:"emitida_en"`
	VerificadaEn                 string                        `json:"verificada_en"`
	ValidaHasta                  string                        `json:"valida_hasta"`
	VerificadorRef               string                        `json:"verificador_ref"`
	HuellaEvidenciaSHA256        string                        `json:"huella_evidencia_sha256"`
}

type aadBorradorPostgreSQL struct {
	Esquema      string `json:"esquema"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type envolturaClaveBorradorPostgreSQL struct {
	Esquema               string `json:"esquema"`
	ClaveMaestraRef       string `json:"clave_maestra_ref"`
	VersionClave          uint32 `json:"version_clave"`
	HuellaAAD             string `json:"huella_aad"`
	HuellaEnvolturaSHA256 string `json:"huella_envoltura_sha256"`
}

type sobreCifradoBorradorPostgreSQL struct {
	Esquema           string `json:"esquema"`
	HuellaAAD         string `json:"huella_aad"`
	HuellaSobreSHA256 string `json:"huella_sobre_sha256"`
}

type atestacionKMSBorradorPostgreSQL struct {
	Esquema               string                         `json:"esquema"`
	AtestacionRef         string                         `json:"atestacion_ref"`
	VersionAtestacion     uint32                         `json:"version_atestacion"`
	Estado                string                         `json:"estado"`
	Perfil                perfilCifradoReciboPostgreSQL  `json:"perfil"`
	ClaveMaestraRef       string                         `json:"clave_maestra_ref"`
	VersionClave          uint32                         `json:"version_clave"`
	HuellaAAD             string                         `json:"huella_aad"`
	HuellaEnvolturaSHA256 string                         `json:"huella_envoltura_sha256"`
	HuellaSobreSHA256     string                         `json:"huella_sobre_sha256"`
	VerificadorRef        string                         `json:"verificador_ref"`
	Procedencia           procedenciaReciboPostgreSQL    `json:"procedencia"`
	Firma                 firmaEvidenciaReciboPostgreSQL `json:"firma"`
	EmitidaEn             string                         `json:"emitida_en"`
	ValidaHasta           string                         `json:"valida_hasta"`
}

type evidenciaCifradoBorradorPostgreSQL struct {
	Esquema         string                            `json:"esquema"`
	Perfil          perfilCifradoReciboPostgreSQL     `json:"perfil"`
	Politica        politicaCifradoBorradorPostgreSQL `json:"politica"`
	EvidenciaPerfil evidenciaPerfilBorradorPostgreSQL `json:"evidencia_perfil"`
	Procedencia     procedenciaReciboPostgreSQL       `json:"procedencia"`
	AAD             aadBorradorPostgreSQL             `json:"aad"`
	EnvolturaClave  envolturaClaveBorradorPostgreSQL  `json:"envoltura_clave"`
	Sobre           sobreCifradoBorradorPostgreSQL    `json:"sobre"`
	AtestacionKMS   atestacionKMSBorradorPostgreSQL   `json:"atestacion_kms"`
}

type cargaConfirmacionBorradorPostgreSQL struct {
	Confirmacion, Prueba, Evidencia, Decision, Contexto, Material, Version, AAD []byte
	MaterialEnvuelto, Nonce, TextoCifrado                                       []byte
}

func (c cargaConfirmacionBorradorPostgreSQL) borrar() {
	borrarBytesDiarioPostgreSQL(
		c.Confirmacion, c.Prueba, c.Evidencia, c.Decision, c.Contexto,
		c.Material, c.Version, c.AAD, c.MaterialEnvuelto, c.Nonce, c.TextoCifrado,
	)
}
