package domain

import "time"

// PlanBootstrapAdministracionV3 conserva el reparto aprobado y las fuentes
// exactas de sus ámbitos. Su preparación no publica roles ni concede acceso.
// La aprobación por huella y el cotejo de fuentes corresponden al proveedor.
type PlanBootstrapAdministracionV3 struct {
	Version                            uint64                              `json:"version"`
	PreparadoEn                        time.Time                           `json:"preparado_en"`
	CaducaEn                           time.Time                           `json:"caduca_en"`
	ControlContinuidadRevisionEsperada uint64                              `json:"control_continuidad_revision_esperada"`
	BootstrapEstadoEsperado            string                              `json:"bootstrap_estado_esperado"`
	Rol                                RolBootstrapAdministracion          `json:"rol"`
	FuenteIdentidad                    EvidenciaBootstrapAdministracion    `json:"fuente_identidad"`
	FuenteCA                           EvidenciaBootstrapAdministracion    `json:"fuente_ca_admin"`
	Personas                           [2]PersonaBootstrapAdministracionV3 `json:"personas"`
	Gobierno                           GobiernoBootstrapAdministracionV3   `json:"gobierno"`
	FuenteRepartoAprobado              EvidenciaBootstrapAdministracion    `json:"fuente_reparto_aprobado"`
}

type PersonaBootstrapAdministracionV3 struct {
	CuentaRef             string                                        `json:"cuenta_ref"`
	CuentaVersion         uint64                                        `json:"cuenta_version"`
	PersonaRef            string                                        `json:"persona_ref"`
	PersonaVersion        uint64                                        `json:"persona_version"`
	PerfilRef             string                                        `json:"perfil_ref"`
	VinculoRef            string                                        `json:"vinculo_ref"`
	PreimagenHuellaSHA256 string                                        `json:"preimagen_huella_sha256"`
	Procedencia           EvidenciaBootstrapAdministracion              `json:"procedencia"`
	VigenteHasta          time.Time                                     `json:"vigente_hasta"`
	Certificado           CertificadoBootstrapAdministracion            `json:"certificado_admin"`
	Sistemas              []AsignacionSistemasBootstrapAdministracionV3 `json:"sistemas"`
	Ambitos               []AmbitoBootstrapAdministracionV3             `json:"ambitos"`
}

type AsignacionSistemasBootstrapAdministracionV3 struct {
	Rol          RolBootstrapAdministracion        `json:"rol"`
	PerfilRef    string                            `json:"perfil_ref"`
	VinculoRef   string                            `json:"vinculo_ref"`
	VigenteHasta time.Time                         `json:"vigente_hasta"`
	Ambitos      []AmbitoBootstrapAdministracionV3 `json:"ambitos"`
}

type AmbitoBootstrapAdministracionV3 struct {
	Dimension string                           `json:"dimension"`
	Valores   []string                         `json:"valores"`
	Fuente    EvidenciaBootstrapAdministracion `json:"fuente"`
}

type GobiernoBootstrapAdministracionV3 struct {
	AudienciaSelectorADMIN          string                                   `json:"audiencia_selector_admin"`
	AudienciaAdministrativa         string                                   `json:"audiencia_administrativa"`
	PoliticaCertificadoRef          string                                   `json:"politica_certificado_ref"`
	PoliticaCertificadoHuellaSHA256 string                                   `json:"politica_certificado_huella_sha256"`
	Roles                           []RolGobernadoBootstrapAdministracionV3  `json:"roles"`
	Motivos                         []MotivoGobernadoBootstrapAdministracion `json:"motivos"`
}

type RolGobernadoBootstrapAdministracionV3 struct {
	VersionRef                string                              `json:"version_ref"`
	HuellaSHA256              string                              `json:"huella_sha256"`
	Clase                     ClaseControlAdministracionPerfiles  `json:"clase"`
	CategoriaAdmin            string                              `json:"categoria_admin"`
	UnidadRequerida           bool                                `json:"unidad_requerida"`
	AmbitosFijos              []AmbitoFijoBootstrapAdministracion `json:"ambitos_fijos"`
	VigenteDesde              time.Time                           `json:"vigente_desde"`
	VigenteHasta              time.Time                           `json:"vigente_hasta"`
	DuracionPropuestaSegundos uint64                              `json:"duracion_propuesta_segundos"`
	FuenteCategoria           EvidenciaBootstrapAdministracion    `json:"fuente_categoria"`
	DimensionesAmbito         []string                            `json:"dimensiones_ambito"`
	RolID                     string                              `json:"rol_id"`
}
