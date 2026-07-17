package ports

import "time"

const (
	maximoConvocatoriasPanel = 40
	maximoActuacionesPanel   = 80
	maximoContadorPanel      = 1_000_000_000
)

// OrigenPanelInterno permite rechazar de forma positiva adaptadores de
// demostracion. El servicio productivo solo acepta Demostracion=false.
type OrigenPanelInterno struct {
	Revision      string    `json:"revision"`
	ActualizadaEn time.Time `json:"actualizada_en"`
	Demostracion  bool      `json:"demostracion"`
}

// PruebaLecturaPanelInterno acredita que la consulta y su auditoria quedaron
// confirmadas. Solo contiene referencias tecnicas opacas.
type PruebaLecturaPanelInterno struct {
	LecturaRef           string    `json:"lectura_ref"`
	AuditoriaRef         string    `json:"auditoria_ref"`
	AuditoriaSecuencia   uint64    `json:"auditoria_secuencia"`
	DecisionRef          string    `json:"decision_ref"`
	HuellaDecisionSHA256 string    `json:"huella_decision_sha256"`
	CorrelacionRef       string    `json:"correlacion_ref"`
	ConfirmadaEn         time.Time `json:"confirmada_en"`
}

// IndicadoresPanelInterno contiene exclusivamente magnitudes agregadas. No
// transporta identidades ni permite reconstruir un listado de personas.
type IndicadoresPanelInterno struct {
	ConvocatoriasBorrador        int `json:"convocatorias_borrador"`
	ConvocatoriasRevision        int `json:"convocatorias_revision"`
	ConvocatoriasPendientesFirma int `json:"convocatorias_pendientes_firma"`
	ConvocatoriasPublicadas      int `json:"convocatorias_publicadas"`
	BolsasActivas                int `json:"bolsas_activas"`
	BolsasSuspendidas            int `json:"bolsas_suspendidas"`
	BolsasAgotadas               int `json:"bolsas_agotadas"`
	LlamamientosPendientes       int `json:"llamamientos_pendientes"`
	LlamamientosEnCurso          int `json:"llamamientos_en_curso"`
	LlamamientosVencenHoy        int `json:"llamamientos_vencen_hoy"`
	DocumentosPendientesFirma    int `json:"documentos_pendientes_firma"`
	IncidenciasAbiertas          int `json:"incidencias_abiertas"`
}

// ResumenConvocatoriaPanelInterno usa claves de catalogo y referencias
// opacas. Las etiquetas se resuelven aparte desde catalogos gobernados.
type ResumenConvocatoriaPanelInterno struct {
	ConvocatoriaRef   string    `json:"convocatoria_ref"`
	CategoriaClave    string    `json:"categoria_clave"`
	EstadoClave       string    `json:"estado_clave"`
	PlazoCierraEn     time.Time `json:"plazo_cierra_en,omitempty"`
	NumeroSolicitudes int       `json:"numero_solicitudes"`
	NumeroPendientes  int       `json:"numero_pendientes"`
}

// ActuacionPendientePanelInterno describe trabajo administrativo sin actor ni
// interesado. RecursoRef apunta al expediente o agregado autorizado.
type ActuacionPendientePanelInterno struct {
	ActuacionRef    string    `json:"actuacion_ref"`
	RecursoRef      string    `json:"recurso_ref"`
	TipoClave       string    `json:"tipo_clave"`
	EstadoClave     string    `json:"estado_clave"`
	PrioridadClave  string    `json:"prioridad_clave"`
	FechaLimite     time.Time `json:"fecha_limite,omitempty"`
	NumeroElementos int       `json:"numero_elementos"`
}

// InstantaneaPanelInterno es el contrato minimo del cuadro operativo. No hay
// campos de nombre, documento identificativo, correo, telefono, direccion ni
// colecciones globales de candidatos.
type InstantaneaPanelInterno struct {
	Esquema               string                            `json:"esquema"`
	Selector              SelectorPanelInterno              `json:"selector"`
	Origen                OrigenPanelInterno                `json:"origen"`
	PruebaLectura         PruebaLecturaPanelInterno         `json:"prueba_lectura"`
	Indicadores           IndicadoresPanelInterno           `json:"indicadores"`
	Convocatorias         []ResumenConvocatoriaPanelInterno `json:"convocatorias"`
	ActuacionesPendientes []ActuacionPendientePanelInterno  `json:"actuaciones_pendientes"`
}

// ClonarValidadaPara aplica copia defensiva y coteja el selector exacto. Un
// origen de demostracion o una prueba de lectura incoherente fallan cerrado.
func (i InstantaneaPanelInterno) ClonarValidadaPara(
	solicitud SolicitudConsultaPanelInterno,
) (InstantaneaPanelInterno, error) {
	if validarInstantaneaPanelInterno(i, solicitud) != nil {
		return InstantaneaPanelInterno{}, ErrResultadoPanelInternoInvalido
	}
	i.Convocatorias = append([]ResumenConvocatoriaPanelInterno(nil), i.Convocatorias...)
	i.ActuacionesPendientes = append([]ActuacionPendientePanelInterno(nil), i.ActuacionesPendientes...)
	return i, nil
}
