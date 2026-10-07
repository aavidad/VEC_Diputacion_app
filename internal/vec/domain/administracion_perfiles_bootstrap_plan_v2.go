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
	CuentaRef             string                                      `json:"cuenta_ref"`
	CuentaVersion         uint64                                      `json:"cuenta_version"`
	PersonaRef            string                                      `json:"persona_ref"`
	PersonaVersion        uint64                                      `json:"persona_version"`
	PerfilRef             string                                      `json:"perfil_ref"`
	VinculoRef            string                                      `json:"vinculo_ref"`
	PreimagenHuellaSHA256 string                                      `json:"preimagen_huella_sha256"`
	Procedencia           EvidenciaBootstrapAdministracion            `json:"procedencia"`
	VigenteHasta          time.Time                                   `json:"vigente_hasta"`
	Certificado           CertificadoBootstrapAdministracion          `json:"certificado_admin"`
	Sistemas              []AsignacionSistemasBootstrapAdministracion `json:"sistemas"`
}

type AsignacionSistemasBootstrapAdministracion struct {
	Rol          RolBootstrapAdministracion `json:"rol"`
	PerfilRef    string                     `json:"perfil_ref"`
	VinculoRef   string                     `json:"vinculo_ref"`
	VigenteHasta time.Time                  `json:"vigente_hasta"`
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
	CategoriaAdmin            string                              `json:"categoria_admin"`
	UnidadRequerida           bool                                `json:"unidad_requerida"`
	AmbitosFijos              []AmbitoFijoBootstrapAdministracion `json:"ambitos_fijos"`
	VigenteDesde              time.Time                           `json:"vigente_desde"`
	VigenteHasta              time.Time                           `json:"vigente_hasta"`
	DuracionPropuestaSegundos uint64                              `json:"duracion_propuesta_segundos"`
}

type GobiernoBootstrapAdministracion struct {
	AudienciaSelectorADMIN          string                                   `json:"audiencia_selector_admin"`
	AudienciaAdministrativa         string                                   `json:"audiencia_administrativa"`
	PoliticaCertificadoRef          string                                   `json:"politica_certificado_ref"`
	PoliticaCertificadoHuellaSHA256 string                                   `json:"politica_certificado_huella_sha256"`
	Roles                           []RolGobernadoBootstrapAdministracion    `json:"roles"`
	Motivos                         []MotivoGobernadoBootstrapAdministracion `json:"motivos"`
}

type MotivoGobernadoBootstrapAdministracion struct {
	Accion      string                    `json:"accion"`
	TipoRecurso string                    `json:"tipo_recurso"`
	Finalidad   string                    `json:"finalidad"`
	Referencia  ReferenciaEntradaCatalogo `json:"referencia"`
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
		!RolVersionAdministracionPerfilesValido(p.Rol.VersionRef) || !strings.HasPrefix(p.Rol.VersionRef, "rol:administracion_perfiles:v") || !HuellaAdministracionPerfilesValida(p.Rol.HuellaSHA256) ||
		p.Rol.ControlRevision == 0 || !HuellaAdministracionPerfilesValida(p.Rol.ControlHuellaSHA256) ||
		!evidenciaBootstrapValida(p.FuenteIdentidad) || !evidenciaBootstrapValida(p.FuenteCA) ||
		p.Personas[0].PersonaRef >= p.Personas[1].PersonaRef || p.Personas[0].CuentaRef == p.Personas[1].CuentaRef ||
		p.Personas[0].PerfilRef == p.Personas[1].PerfilRef || p.Personas[0].VinculoRef == p.Personas[1].VinculoRef ||
		p.Personas[0].Certificado.HuellaSHA256 == p.Personas[1].Certificado.HuellaSHA256 ||
		!textoFijoBootstrap(p.Gobierno.AudienciaAdministrativa) || !textoFijoBootstrap(p.Gobierno.AudienciaSelectorADMIN) || !textoFijoBootstrap(p.Gobierno.PoliticaCertificadoRef) ||
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
	if err := p.validarKitYMotivos(); err != nil {
		return err
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
		if rol.CategoriaAdmin != "" && rol.CategoriaAdmin != "aplicacion" && rol.CategoriaAdmin != "sistemas" {
			return ErrControlAdministracionPerfilesInvalido
		}
		if rol.Clase != ClaseControlPerfilAdministrador && rol.CategoriaAdmin != "" {
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
			if rol.Clase != ClaseControlPerfilAdministrador || rol.CategoriaAdmin != "aplicacion" || rol.HuellaSHA256 != p.Rol.HuellaSHA256 || rol.UnidadRequerida {
				return ErrControlAdministracionPerfilesInvalido
			}
			if len(rol.AmbitosFijos) != 1 || rol.AmbitosFijos[0].Clave != "administracion" || len(rol.AmbitosFijos[0].Valores) != 1 || rol.AmbitosFijos[0].Valores[0] != "perfiles" {
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

func (p PlanBootstrapAdministracionV2) validarKitYMotivos() error {
	if p.Gobierno.Motivos == nil || len(p.Gobierno.Motivos) > 64 {
		return ErrControlAdministracionPerfilesInvalido
	}
	tipos := map[string]string{"administracion.perfiles.consultar": "perfil", "administracion.perfiles.otorgar": "perfil", "administracion.perfiles.revocar": "perfil",
		"administracion.perfiles.proponer": "perfil", "administracion.perfiles.aprobar": "propuesta_perfil", "administracion.perfiles.rechazar": "propuesta_perfil",
		"administracion.perfiles.historial.consultar": "historial_perfil", "administracion.perfiles.recibo.consultar": "recibo_perfil"}
	ultimo := ""
	for _, m := range p.Gobierno.Motivos {
		clave := m.Accion + "\x00" + m.Referencia.Referencia()
		if tipos[m.Accion] == "" || m.TipoRecurso != tipos[m.Accion] || m.Finalidad != "gestion_perfiles" || !ReferenciaMotivoAutorizacionV2Valida(m.Referencia) || clave <= ultimo {
			return ErrControlAdministracionPerfilesInvalido
		}
		ultimo = clave
	}
	perfiles := map[string]bool{p.Personas[0].PerfilRef: true, p.Personas[1].PerfilRef: true}
	vinculos := map[string]bool{p.Personas[0].VinculoRef: true, p.Personas[1].VinculoRef: true}
	cuenta := 0
	for _, persona := range p.Personas {
		if persona.Sistemas == nil || len(persona.Sistemas) > 1 {
			return ErrControlAdministracionPerfilesInvalido
		}
		for _, s := range persona.Sistemas {
			cuenta++
			if !referenciaOpacaAdministracionPerfiles(s.PerfilRef, "prf_") || !referenciaOpacaAdministracionPerfiles(s.VinculoRef, "vca_") ||
				perfiles[s.PerfilRef] || vinculos[s.VinculoRef] || !RolVersionAdministracionPerfilesValido(s.Rol.VersionRef) || !strings.HasPrefix(s.Rol.VersionRef, "rol:operador_plataforma:v") || s.Rol.VersionRef == p.Rol.VersionRef ||
				!HuellaAdministracionPerfilesValida(s.Rol.HuellaSHA256) || s.Rol.ControlRevision == 0 || !HuellaAdministracionPerfilesValida(s.Rol.ControlHuellaSHA256) ||
				!instanteBootstrap(s.VigenteHasta) || s.VigenteHasta.Before(p.CaducaEn) || s.VigenteHasta.After(persona.VigenteHasta) {
				return ErrControlAdministracionPerfilesInvalido
			}
			publicado := false
			for _, rol := range p.Gobierno.Roles {
				if rol.VersionRef == s.Rol.VersionRef && rol.HuellaSHA256 == s.Rol.HuellaSHA256 && rol.Clase == ClaseControlPerfilAdministrador && rol.CategoriaAdmin == "sistemas" && !rol.UnidadRequerida {
					publicado = true
				}
			}
			if !publicado {
				return ErrControlAdministracionPerfilesInvalido
			}
			perfiles[s.PerfilRef] = true
			vinculos[s.VinculoRef] = true
		}
	}
	if cuenta != 1 {
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
