package postgres

import (
	"encoding/json"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/domain"
)

const limiteLecturasADMIN = 50

func referenciaLectura(v, prefijo string) bool {
	if !strings.HasPrefix(v, prefijo) || len(v) < len(prefijo)+22 || len(v) > len(prefijo)+128 {
		return false
	}
	for _, c := range v[len(prefijo):] {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

func textoLectura(v string, maximo int, vacio bool) bool {
	if !utf8.ValidString(v) || len(v) > maximo || !vacio && strings.TrimSpace(v) == "" {
		return false
	}
	for _, c := range v {
		if unicode.IsControl(c) {
			return false
		}
	}
	return true
}

func huellaLectura(v string) bool {
	if len(v) != 64 {
		return false
	}
	for _, c := range v {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func validarMotivosLectura(motivos []api.MotivoLectura) bool {
	if motivos == nil || len(motivos) > limiteLecturasADMIN {
		return false
	}
	vistas := make(map[string]bool, len(motivos))
	for _, m := range motivos {
		r := domain.ReferenciaEntradaCatalogo{CatalogoID: m.CatalogoID, CatalogoVersion: m.CatalogoVersion, CatalogoHuellaSHA256: m.CatalogoHuellaSHA256, EntradaClave: m.EntradaClave}
		if r.Validar() != nil || !textoLectura(m.Etiqueta, 512, false) || vistas[r.Referencia()] {
			return false
		}
		vistas[r.Referencia()] = true
	}
	return true
}

func preimagenLectura(o api.Objetivo) domain.PreimagenAdministracionPerfiles {
	return domain.PreimagenAdministracionPerfiles{CentroRef: o.CentroRef, VigenteDesde: o.VigenteDesde, UnidadRef: o.UnidadRef, CuentaRef: o.CuentaRef, CuentaVersion: o.CuentaVersion, PersonaRef: o.PersonaRef, PersonaVersion: o.PersonaVersion, PerfilRef: o.PerfilRef, PerfilVersion: o.PerfilVersion, VinculoRef: o.VinculoRef, VinculoVersion: o.VinculoVersion, HuellaSHA256: o.HuellaSHA256, RevisionContinuidad: o.RevisionContinuidad, ProcedenciaRef: o.ProcedenciaRef, ProcedenciaVersion: o.ProcedenciaVersion, ProcedenciaHuellaSHA256: o.ProcedenciaHuellaSHA256, VigenteHasta: o.VigenteHasta}
}

func validarFichaLectura(x api.FichaPersona, persona string, ahora time.Time) bool {
	if x.PersonaRef != persona || !textoLectura(x.Nombre, 512, false) || !unidadLectura(x.UnidadRef) || !textoLectura(x.UnidadNombre, 512, false) || x.Perfiles == nil || x.ActosDisponibles == nil || x.Historia == nil || len(x.Perfiles) > limiteLecturasADMIN || len(x.ActosDisponibles) > limiteLecturasADMIN || len(x.Historia) > limiteLecturasADMIN {
		return false
	}
	perfiles := make(map[string]bool, len(x.Perfiles))
	for _, p := range x.Perfiles {
		if !referenciaLectura(p.PerfilRef, "prf_") || !domain.RolVersionAdministracionPerfilesValido(p.RolVersionRef) || !domain.EstadoVinculoContextoActor(p.Estado).Valido() || p.Version == 0 || !instantePersistible(p.VigenteDesde) || !instantePersistible(p.VigenteHasta) || !p.VigenteHasta.After(p.VigenteDesde) || !validarAmbitosLectura(p.Ambitos) || !textoLectura(p.RolEtiqueta, 512, false) || perfiles[p.PerfilRef] {
			return false
		}
		perfiles[p.PerfilRef] = true
	}
	actos := make(map[string]bool, len(x.ActosDisponibles))
	for _, a := range x.ActosDisponibles {
		op := domain.OperacionAdministracionPerfiles(a.Operacion)
		// La lectura no clasifica roles ni concede permisos. SQL conserva la clase
		// y la preimagen; aquí se comprueba exclusivamente la forma del objetivo.
		if a.Objetivo.PersonaRef != persona || !domain.RolVersionAdministracionPerfilesValido(a.RolVersionRef) || preimagenLectura(a.Objetivo).ValidarPara(op, domain.ClaseControlPerfilOrdinario) != nil || !validarMotivosLectura(a.Motivos) || len(a.Motivos) == 0 || op == domain.OperacionOtorgarPerfil && !a.Objetivo.VigenteHasta.After(ahora) {
			return false
		}
		clave := a.Operacion + "\x00" + a.RolVersionRef + "\x00" + a.Objetivo.PerfilRef
		if actos[clave] {
			return false
		}
		actos[clave] = true
	}
	historia := make(map[string]bool, len(x.Historia))
	for _, h := range x.Historia {
		if !domain.ReferenciaAdministracionPerfilesValida(h.ActoRef, "acto_admin:") || !domain.OperacionAdministracionPerfiles(h.Operacion).Valida() || !domain.EstadoVinculoContextoActor(h.Estado).Valido() || !instantePersistible(h.ConfirmadoEn) || h.ObjetivoPersonaRef != persona || !textoLectura(h.ObjetivoNombre, 512, false) || !validarIdentidadHistoria(h.Actor) || !validarAmbitosLectura(h.Ambitos) || !validarMotivosLectura([]api.MotivoLectura{h.Motivo}) || (h.Proponente != nil && !validarIdentidadHistoria(*h.Proponente)) || (h.Aprobador != nil && !validarIdentidadHistoria(*h.Aprobador)) || historia[h.ActoRef] {
			return false
		}
		historia[h.ActoRef] = true
	}
	return true
}

func validarPropuestaLectura(x api.Propuesta, actor domain.ContextoActor, ahora time.Time) bool {
	if !domain.ReferenciaAdministracionPerfilesValida(x.PropuestaRef, "propuesta_admin:") || !referenciaLectura(x.ProponentePersonaRef, "per_") || !referenciaLectura(x.ObjetivoPersonaRef, "per_") || !textoLectura(x.ObjetivoNombre, 512, false) || !domain.RolVersionAdministracionPerfilesValido(x.RolVersionRef) || !domain.OperacionAdministracionPerfiles(x.Operacion).Valida() || !huellaLectura(x.HuellaSHA256) || !instantePersistible(x.CaducaEn) || !validarMotivosLectura(x.MotivosCierre) {
		return false
	}
	if x.PuedeCerrar && (actor.PersonaRef == x.ProponentePersonaRef || actor.PersonaRef == x.ObjetivoPersonaRef || !x.CaducaEn.After(ahora) || len(x.MotivosCierre) == 0 || !textoLectura(x.ProponenteNombre, 512, false) || !validarAmbitosLectura(x.Ambitos) || !instantePersistible(x.VigenteDesde) || !instantePersistible(x.VigenteHasta) || !x.VigenteHasta.After(x.VigenteDesde) || x.Motivo == nil || !validarMotivosLectura([]api.MotivoLectura{*x.Motivo})) {
		return false
	}
	return true
}

func validarAmbitosLectura(ambitos []api.AmbitoPerfil) bool {
	if len(ambitos) == 0 || len(ambitos) > limiteLecturasADMIN {
		return false
	}
	vistos := make(map[string]bool, len(ambitos))
	for _, a := range ambitos {
		if !textoNominalLectura(a.Dimension, 128) || !textoNominalLectura(a.Referencia, 512) || !textoLectura(a.Nombre, 512, false) || vistos[a.Dimension+"\x00"+a.Referencia] {
			return false
		}
		vistos[a.Dimension+"\x00"+a.Referencia] = true
	}
	return true
}

func validarIdentidadHistoria(x api.IdentidadHistoria) bool {
	return referenciaLectura(x.PersonaRef, "per_") && referenciaLectura(x.PerfilActivoRef, "prf_") && textoNominalLectura(x.AsignacionRef, 512) && textoLectura(x.Nombre, 512, false) && textoLectura(x.PerfilActivoNombre, 512, false)
}

func textoNominalLectura(v string, maximo int) bool {
	if !textoLectura(v, maximo, false) || strings.ContainsRune(v, '*') {
		return false
	}
	for _, c := range []byte(v) {
		if c < 0x21 || c > 0x7e {
			return false
		}
	}
	return true
}

// bool sin puntero no distingue un fijo ausente de uno falso. Rechazar ese
// DTO antiguo evita presentar como gestionable un rol sin clasificación.
func fijosLecturaPresentes(b []byte) bool {
	var documento struct {
		Roles []struct {
			Fijo *bool `json:"fijo"`
		} `json:"roles"`
	}
	if json.Unmarshal(b, &documento) != nil || documento.Roles == nil {
		return false
	}
	for _, rol := range documento.Roles {
		if rol.Fijo == nil {
			return false
		}
	}
	return true
}

func accionCapacidadLectura(s string) bool {
	switch s {
	case "consultar", "aplicar_ordinario", "aplicar_lote_ordinario", "proponer", "cerrar_propuesta":
		return true
	default:
		return false
	}
}
