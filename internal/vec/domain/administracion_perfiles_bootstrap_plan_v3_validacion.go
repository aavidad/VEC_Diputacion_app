package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// Validar comprueba el material finito y sus vínculos. No acredita publicación,
// titularidad del ámbito ni aprobación: el proveedor coteja sus fuentes reales.
func (p PlanBootstrapAdministracionV3) Validar() error {
	if p.Version != 3 || p.ControlContinuidadRevisionEsperada != 1 || p.BootstrapEstadoEsperado != "pendiente" ||
		!instanteBootstrap(p.PreparadoEn) || !instanteBootstrap(p.CaducaEn) || !p.CaducaEn.After(p.PreparadoEn) ||
		!rolBootstrapV3Valido(p.Rol) ||
		!evidenciaBootstrapValida(p.FuenteIdentidad) || !evidenciaBootstrapValida(p.FuenteCA) ||
		!evidenciaBootstrapValida(p.FuenteRepartoAprobado) ||
		p.Personas[0].PersonaRef >= p.Personas[1].PersonaRef || p.Personas[0].CuentaRef == p.Personas[1].CuentaRef ||
		p.Personas[0].PerfilRef == p.Personas[1].PerfilRef || p.Personas[0].VinculoRef == p.Personas[1].VinculoRef ||
		p.Personas[0].Certificado.HuellaSHA256 == p.Personas[1].Certificado.HuellaSHA256 || !p.gobiernoV3Valido() {
		return ErrControlAdministracionPerfilesInvalido
	}
	perfiles := map[string]bool{p.Personas[0].PerfilRef: true, p.Personas[1].PerfilRef: true}
	vinculos := map[string]bool{p.Personas[0].VinculoRef: true, p.Personas[1].VinculoRef: true}
	sistemas := 0
	for _, persona := range p.Personas {
		if !p.personaV3Valida(persona) {
			return ErrControlAdministracionPerfilesInvalido
		}
		for _, s := range persona.Sistemas {
			sistemas++
			publicado, existe := p.rolGobernadoV3(s.Rol.VersionRef)
			if !referenciaOpacaAdministracionPerfiles(s.PerfilRef, "prf_") || !referenciaOpacaAdministracionPerfiles(s.VinculoRef, "vca_") ||
				perfiles[s.PerfilRef] || vinculos[s.VinculoRef] || !existe || publicado.CategoriaAdmin != "sistemas" || !rolBootstrapV3Valido(s.Rol) ||
				!instanteBootstrap(s.VigenteHasta) || s.VigenteHasta.Before(p.CaducaEn) || s.VigenteHasta.After(persona.VigenteHasta) ||
				s.Rol.HuellaSHA256 != publicado.HuellaSHA256 || !ambitosBootstrapV3Validos(s.Ambitos, publicado) {
				return ErrControlAdministracionPerfilesInvalido
			}
			perfiles[s.PerfilRef], vinculos[s.VinculoRef] = true, true
		}
	}
	if sistemas != 1 {
		return ErrControlAdministracionPerfilesInvalido
	}
	return nil
}

func (p PlanBootstrapAdministracionV3) personaV3Valida(persona PersonaBootstrapAdministracionV3) bool {
	publicado, existe := p.rolGobernadoV3(p.Rol.VersionRef)
	preimagen := PreimagenAdministracionPerfiles{CuentaRef: persona.CuentaRef, CuentaVersion: persona.CuentaVersion,
		PersonaRef: persona.PersonaRef, PersonaVersion: persona.PersonaVersion, PerfilRef: persona.PerfilRef, VinculoRef: persona.VinculoRef,
		HuellaSHA256: persona.PreimagenHuellaSHA256, ProcedenciaRef: persona.Procedencia.Referencia,
		ProcedenciaVersion: persona.Procedencia.Version, ProcedenciaHuellaSHA256: persona.Procedencia.HuellaSHA256, VigenteHasta: persona.VigenteHasta}
	return existe && publicado.CategoriaAdmin == "aplicacion" && preimagen.ValidarBootstrap() == nil && evidenciaBootstrapValida(persona.Procedencia) &&
		evidenciaBootstrapValida(persona.Certificado.Acreditacion) && persona.Certificado.PersonaRef == persona.PersonaRef &&
		persona.Certificado.CuentaRef == persona.CuentaRef && HuellaAdministracionPerfilesValida(persona.Certificado.HuellaSHA256) &&
		persona.Certificado.CAHuellaSHA256 == p.FuenteCA.HuellaSHA256 && instanteBootstrap(persona.VigenteHasta) &&
		!persona.VigenteHasta.Before(p.CaducaEn) && persona.Sistemas != nil && len(persona.Sistemas) <= 1 &&
		ambitosBootstrapV3Validos(persona.Ambitos, publicado)
}

func rolBootstrapV3Valido(rol RolBootstrapAdministracion) bool {
	return RolVersionAdministracionPerfilesValido(rol.VersionRef) && HuellaAdministracionPerfilesValida(rol.HuellaSHA256) &&
		rol.ControlRevision > 0 && HuellaAdministracionPerfilesValida(rol.ControlHuellaSHA256)
}

func (p PlanBootstrapAdministracionV3) gobiernoV3Valido() bool {
	g := p.Gobierno
	referenciados := map[string]string{p.Rol.VersionRef: "aplicacion"}
	for _, persona := range p.Personas {
		for _, sistema := range persona.Sistemas {
			if sistema.Rol.VersionRef == p.Rol.VersionRef {
				return false
			}
			referenciados[sistema.Rol.VersionRef] = "sistemas"
		}
	}
	if !textoFijoBootstrap(g.AudienciaSelectorADMIN) || !textoFijoBootstrap(g.AudienciaAdministrativa) ||
		!textoFijoBootstrap(g.PoliticaCertificadoRef) || !HuellaAdministracionPerfilesValida(g.PoliticaCertificadoHuellaSHA256) ||
		len(g.Roles) == 0 || len(g.Roles) > 64 || len(g.Roles) != len(referenciados) || !motivosBootstrapV3Validos(g.Motivos) {
		return false
	}
	ultimo := ""
	for _, rol := range g.Roles {
		categoria := referenciados[rol.VersionRef]
		rolID := "administracion_perfiles"
		if categoria == "sistemas" {
			rolID = "operador_plataforma"
		}
		if categoria == "" || rol.RolID != rolID || !strings.HasPrefix(rol.VersionRef, "rol:"+rol.RolID+":v") ||
			!RolVersionAdministracionPerfilesValido(rol.VersionRef) || rol.VersionRef <= ultimo ||
			!HuellaAdministracionPerfilesValida(rol.HuellaSHA256) || rol.Clase != ClaseControlPerfilAdministrador || rol.CategoriaAdmin != categoria ||
			!evidenciaBootstrapValida(rol.FuenteCategoria) || rol.VersionRef == p.Rol.VersionRef && rol.HuellaSHA256 != p.Rol.HuellaSHA256 ||
			!instanteBootstrap(rol.VigenteDesde) || !instanteBootstrap(rol.VigenteHasta) || rol.VigenteDesde.After(p.PreparadoEn) ||
			rol.VigenteHasta.Before(p.CaducaEn) || !rol.VigenteHasta.After(rol.VigenteDesde) || rol.DuracionPropuestaSegundos == 0 ||
			rol.DuracionPropuestaSegundos > uint64(rol.VigenteHasta.Sub(p.PreparadoEn).Seconds()) || !limitesAmbitoBootstrapV3Validos(rol) {
			return false
		}
		ultimo = rol.VersionRef
	}
	return true
}

func (p PlanBootstrapAdministracionV3) rolGobernadoV3(ref string) (RolGobernadoBootstrapAdministracionV3, bool) {
	for _, rol := range p.Gobierno.Roles {
		if rol.VersionRef == ref {
			return rol, true
		}
	}
	return RolGobernadoBootstrapAdministracionV3{}, false
}

func motivosBootstrapV3Validos(motivos []MotivoGobernadoBootstrapAdministracion) bool {
	if motivos == nil || len(motivos) > 64 {
		return false
	}
	tipos := map[string]string{"administracion.perfiles.consultar": "perfil", "administracion.perfiles.otorgar": "perfil", "administracion.perfiles.revocar": "perfil",
		"administracion.perfiles.proponer": "perfil", "administracion.perfiles.aprobar": "propuesta_perfil", "administracion.perfiles.rechazar": "propuesta_perfil",
		"administracion.perfiles.historial.consultar": "historial_perfil", "administracion.perfiles.recibo.consultar": "recibo_perfil"}
	ultima := ""
	for _, motivo := range motivos {
		clave := motivo.Accion + "\x00" + motivo.Referencia.Referencia()
		if tipos[motivo.Accion] == "" || motivo.TipoRecurso != tipos[motivo.Accion] || motivo.Finalidad != "gestion_perfiles" ||
			!ReferenciaMotivoAutorizacionV2Valida(motivo.Referencia) || clave <= ultima {
			return false
		}
		ultima = clave
	}
	return true
}

// CanonicoYHuella conserva el orden explícito de los structs y exige colecciones
// ya ordenadas. Su SHA256 cubre también reparto, organización, unidad y fuentes.
func (p PlanBootstrapAdministracionV3) CanonicoYHuella() ([]byte, string, error) {
	if p.Validar() != nil {
		return nil, "", ErrControlAdministracionPerfilesInvalido
	}
	b, err := json.Marshal(p)
	if err != nil || len(b) > 64<<10 {
		return nil, "", ErrControlAdministracionPerfilesInvalido
	}
	huella := sha256.Sum256(b)
	return b, hex.EncodeToString(huella[:]), nil
}
