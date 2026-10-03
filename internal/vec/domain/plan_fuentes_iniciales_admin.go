package domain

import "time"

// PlanFuentesInicialesAdminV1 prepara fuentes declaradas sintéticas. No contiene
// material HMAC, referencias de cuenta elegidas por el cliente ni perfiles.
// LOGIN, configuración, preimagen y aprobación se cotejan fuera del plan.
type PlanFuentesInicialesAdminV1 struct {
	Version       uint64                            `json:"version"`
	OperacionRef  string                            `json:"operacion_ref"`
	PreparadoEn   time.Time                         `json:"preparado_en"`
	CaducaEn      time.Time                         `json:"caduca_en"`
	Entorno       string                            `json:"entorno"`
	AlcanceFuente string                            `json:"alcance_fuente"`
	Procedencia   EvidenciaFuentesInicialesAdmin    `json:"procedencia"`
	Organizacion  OrganizacionFuentesInicialesAdmin `json:"organizacion"`
	Personas      [2]PersonaFuentesInicialesAdmin   `json:"personas"`
	FuenteHMAC    EvidenciaFuentesInicialesAdmin    `json:"fuente_hmac"`
	PoliticaADMIN PoliticaFuentesInicialesAdmin     `json:"politica_admin"`
}

type EvidenciaFuentesInicialesAdmin struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type OrganizacionFuentesInicialesAdmin struct {
	OrganizacionRef string    `json:"organizacion_ref"`
	VersionEsperada uint64    `json:"version_esperada"`
	VigenteHasta    time.Time `json:"vigente_hasta"`
}

type PersonaFuentesInicialesAdmin struct {
	PersonaRef                     string                         `json:"persona_ref"`
	VersionEsperada                uint64                         `json:"version_esperada"`
	VigenteHasta                   time.Time                      `json:"vigente_hasta"`
	OperacionCuentaOrdinariaRef    string                         `json:"operacion_cuenta_ordinaria_ref"`
	OperacionCuentaPrivilegiadaRef string                         `json:"operacion_cuenta_privilegiada_ref"`
	FuenteTitularidad              EvidenciaFuentesInicialesAdmin `json:"fuente_titularidad"`
}

type PoliticaFuentesInicialesAdmin struct {
	PoliticaRef                  string    `json:"politica_ref"`
	HostADMIN                    string    `json:"host_admin"`
	CAHuellaSHA256               string    `json:"ca_sha256"`
	HuellaAprobacionSHA256       string    `json:"huella_aprobacion_sha256"`
	MaximaEdadRevocacionSegundos uint64    `json:"maxima_edad_revocacion_segundos"`
	VigenteHasta                 time.Time `json:"vigente_hasta"`
}
