package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type EvidenciaBootstrapAdministracion struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type CertificadoBootstrapAdministracion struct {
	PersonaRef     string                           `json:"persona_ref"`
	CuentaRef      string                           `json:"cuenta_ref"`
	HuellaSHA256   string                           `json:"huella_sha256"`
	CAHuellaSHA256 string                           `json:"ca_huella_sha256"`
	Acreditacion   EvidenciaBootstrapAdministracion `json:"acreditacion"`
}

type PersonaBootstrapAdministracion struct {
	CuentaRef             string                             `json:"cuenta_ref"`
	CuentaVersion         uint64                             `json:"cuenta_version"`
	PersonaRef            string                             `json:"persona_ref"`
	PersonaVersion        uint64                             `json:"persona_version"`
	PerfilRef             string                             `json:"perfil_ref"`
	VinculoRef            string                             `json:"vinculo_ref"`
	PreimagenHuellaSHA256 string                             `json:"preimagen_huella_sha256"`
	Procedencia           EvidenciaBootstrapAdministracion   `json:"procedencia"`
	VigenteHasta          time.Time                          `json:"vigente_hasta"`
	Certificado           CertificadoBootstrapAdministracion `json:"certificado_admin"`
}

func (p PersonaBootstrapAdministracion) Preimagen() PreimagenAdministracionPerfiles {
	return PreimagenAdministracionPerfiles{CuentaRef: p.CuentaRef, CuentaVersion: p.CuentaVersion, PersonaRef: p.PersonaRef,
		PersonaVersion: p.PersonaVersion, PerfilRef: p.PerfilRef, VinculoRef: p.VinculoRef, HuellaSHA256: p.PreimagenHuellaSHA256,
		ProcedenciaRef: p.Procedencia.Referencia, ProcedenciaVersion: p.Procedencia.Version,
		ProcedenciaHuellaSHA256: p.Procedencia.HuellaSHA256, VigenteHasta: p.VigenteHasta}
}

type RolBootstrapAdministracion struct {
	VersionRef          string `json:"version_ref"`
	HuellaSHA256        string `json:"huella_sha256"`
	ControlRevision     uint64 `json:"control_revision"`
	ControlHuellaSHA256 string `json:"control_huella_sha256"`
}

type AmbitoFijoBootstrapAdministracion struct {
	Clave   string   `json:"clave"`
	Valores []string `json:"valores"`
}

type RolGobernadoBootstrapAdministracion struct {
	VersionRef                string                              `json:"version_ref"`
	HuellaSHA256              string                              `json:"huella_sha256"`
	Clase                     ClaseControlAdministracionPerfiles  `json:"clase"`
	UnidadRequerida           bool                                `json:"unidad_requerida"`
	AmbitosFijos              []AmbitoFijoBootstrapAdministracion `json:"ambitos_fijos"`
	VigenteDesde              time.Time                           `json:"vigente_desde"`
	VigenteHasta              time.Time                           `json:"vigente_hasta"`
	DuracionPropuestaSegundos uint64                              `json:"duracion_propuesta_segundos"`
}

type GobiernoBootstrapAdministracion struct {
	AudienciaAdministrativa         string                                `json:"audiencia_administrativa"`
	PoliticaCertificadoRef          string                                `json:"politica_certificado_ref"`
	PoliticaCertificadoHuellaSHA256 string                                `json:"politica_certificado_huella_sha256"`
	Roles                           []RolGobernadoBootstrapAdministracion `json:"roles"`
}

// PlanBootstrapAdministracionV2 es una configuración finita aprobada por su
// huella. No contiene concesiones, claves ni una facultad universal de ADMIN.
// El proveedor durable coteja las fuentes originales y los ámbitos admitidos.
type PlanBootstrapAdministracionV2 struct {
	Version                            uint64                            `json:"version"`
	PreparadoEn                        time.Time                         `json:"preparado_en"`
	CaducaEn                           time.Time                         `json:"caduca_en"`
	ControlContinuidadRevisionEsperada uint64                            `json:"control_continuidad_revision_esperada"`
	BootstrapEstadoEsperado            string                            `json:"bootstrap_estado_esperado"`
	Rol                                RolBootstrapAdministracion        `json:"rol"`
	FuenteIdentidad                    EvidenciaBootstrapAdministracion  `json:"fuente_identidad"`
	FuenteCA                           EvidenciaBootstrapAdministracion  `json:"fuente_ca_admin"`
	Personas                           [2]PersonaBootstrapAdministracion `json:"personas"`
	Gobierno                           GobiernoBootstrapAdministracion   `json:"gobierno"`
}

func (p PlanBootstrapAdministracionV2) Validar() error {
	if p.Version != 2 || p.ControlContinuidadRevisionEsperada != 1 || p.BootstrapEstadoEsperado != "pendiente" ||
		!instanteBootstrap(p.PreparadoEn) || !instanteBootstrap(p.CaducaEn) || !p.CaducaEn.After(p.PreparadoEn) ||
		!RolVersionAdministracionPerfilesValido(p.Rol.VersionRef) || !HuellaAdministracionPerfilesValida(p.Rol.HuellaSHA256) ||
		p.Rol.ControlRevision == 0 || !HuellaAdministracionPerfilesValida(p.Rol.ControlHuellaSHA256) ||
		!evidenciaBootstrapValida(p.FuenteIdentidad) || !evidenciaBootstrapValida(p.FuenteCA) ||
		p.Personas[0].PersonaRef >= p.Personas[1].PersonaRef || p.Personas[0].CuentaRef == p.Personas[1].CuentaRef ||
		p.Personas[0].PerfilRef == p.Personas[1].PerfilRef || p.Personas[0].VinculoRef == p.Personas[1].VinculoRef ||
		p.Personas[0].Certificado.HuellaSHA256 == p.Personas[1].Certificado.HuellaSHA256 ||
		!textoFijoBootstrap(p.Gobierno.AudienciaAdministrativa) || !textoFijoBootstrap(p.Gobierno.PoliticaCertificadoRef) ||
		!HuellaAdministracionPerfilesValida(p.Gobierno.PoliticaCertificadoHuellaSHA256) || len(p.Gobierno.Roles) == 0 || len(p.Gobierno.Roles) > 64 {
		return ErrControlAdministracionPerfilesInvalido
	}
	for _, persona := range p.Personas {
		if persona.Preimagen().ValidarBootstrap() != nil || !evidenciaBootstrapValida(persona.Procedencia) ||
			!evidenciaBootstrapValida(persona.Certificado.Acreditacion) || persona.Certificado.PersonaRef != persona.PersonaRef ||
			persona.Certificado.CuentaRef != persona.CuentaRef || !HuellaAdministracionPerfilesValida(persona.Certificado.HuellaSHA256) ||
			persona.Certificado.CAHuellaSHA256 != p.FuenteCA.HuellaSHA256 || !instanteBootstrap(persona.VigenteHasta) || persona.VigenteHasta.Before(p.CaducaEn) {
			return ErrControlAdministracionPerfilesInvalido
		}
	}
	adminEncontrado := false
	ultimo := ""
	for _, rol := range p.Gobierno.Roles {
		if !RolVersionAdministracionPerfilesValido(rol.VersionRef) || rol.VersionRef <= ultimo || !HuellaAdministracionPerfilesValida(rol.HuellaSHA256) ||
			!rol.Clase.Valida() || !instanteBootstrap(rol.VigenteDesde) || !instanteBootstrap(rol.VigenteHasta) || rol.VigenteDesde.After(p.PreparadoEn) ||
			rol.VigenteHasta.Before(p.CaducaEn) || !rol.VigenteHasta.After(rol.VigenteDesde) || rol.DuracionPropuestaSegundos == 0 ||
			rol.AmbitosFijos == nil || len(rol.AmbitosFijos) > 64 || rol.UnidadRequerida && len(rol.AmbitosFijos) != 0 || !rol.UnidadRequerida && len(rol.AmbitosFijos) == 0 {
			return ErrControlAdministracionPerfilesInvalido
		}
		segundosDisponibles := int64(rol.VigenteHasta.Sub(p.PreparadoEn) / time.Second)
		if segundosDisponibles <= 0 {
			return ErrControlAdministracionPerfilesInvalido
		}
		if rol.DuracionPropuestaSegundos > uint64(segundosDisponibles) {
			return ErrControlAdministracionPerfilesInvalido
		}
		ultimo = rol.VersionRef
		if rol.VersionRef == p.Rol.VersionRef {
			if rol.Clase != ClaseControlPerfilAdministrador || rol.HuellaSHA256 != p.Rol.HuellaSHA256 || rol.UnidadRequerida {
				return ErrControlAdministracionPerfilesInvalido
			}
			adminEncontrado = true
		}
		clavePrevia := ""
		for _, ambito := range rol.AmbitosFijos {
			if !textoFijoBootstrap(ambito.Clave) || ambito.Clave <= clavePrevia || len(ambito.Valores) == 0 || len(ambito.Valores) > 64 {
				return ErrControlAdministracionPerfilesInvalido
			}
			clavePrevia = ambito.Clave
			valorPrevio := ""
			for _, valor := range ambito.Valores {
				if !textoFijoBootstrap(valor) || valor <= valorPrevio {
					return ErrControlAdministracionPerfilesInvalido
				}
				valorPrevio = valor
			}
		}
	}
	if !adminEncontrado {
		return ErrControlAdministracionPerfilesInvalido
	}
	return nil
}

func (p PlanBootstrapAdministracionV2) CanonicoYHuella() ([]byte, string, error) {
	if p.Validar() != nil {
		return nil, "", ErrControlAdministracionPerfilesInvalido
	}
	b, err := json.Marshal(p)
	if err != nil || len(b) > 64<<10 {
		return nil, "", ErrControlAdministracionPerfilesInvalido
	}
	h := sha256.Sum256(b)
	return b, hex.EncodeToString(h[:]), nil
}

func (p PlanBootstrapAdministracionV2) Preimagen() (PreimagenBootstrapAdministracionPerfiles, error) {
	_, huella, err := p.CanonicoYHuella()
	if err != nil {
		return PreimagenBootstrapAdministracionPerfiles{}, err
	}
	return PreimagenBootstrapAdministracionPerfiles{Primera: p.Personas[0].Preimagen(), Segunda: p.Personas[1].Preimagen(), HuellaPlanSHA256: huella, PlanV2: &p}, nil
}

func evidenciaBootstrapValida(e EvidenciaBootstrapAdministracion) bool {
	return textoFijoBootstrap(e.Referencia) && e.Version > 0 && HuellaAdministracionPerfilesValida(e.HuellaSHA256)
}
func textoFijoBootstrap(s string) bool {
	if len(s) == 0 || len(s) > 256 || !utf8.ValidString(s) || strings.ContainsAny(s, "* \t\r\n") {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
func instanteBootstrap(t time.Time) bool {
	return !t.IsZero() && t.Location() == time.UTC && t.Nanosecond() == 0
}
