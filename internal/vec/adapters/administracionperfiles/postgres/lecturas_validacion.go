package postgres

import (
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
	return domain.PreimagenAdministracionPerfiles{UnidadRef: o.UnidadRef, CuentaRef: o.CuentaRef, CuentaVersion: o.CuentaVersion, PersonaRef: o.PersonaRef, PersonaVersion: o.PersonaVersion, PerfilRef: o.PerfilRef, PerfilVersion: o.PerfilVersion, VinculoRef: o.VinculoRef, VinculoVersion: o.VinculoVersion, HuellaSHA256: o.HuellaSHA256, RevisionContinuidad: o.RevisionContinuidad, ProcedenciaRef: o.ProcedenciaRef, ProcedenciaVersion: o.ProcedenciaVersion, ProcedenciaHuellaSHA256: o.ProcedenciaHuellaSHA256, VigenteHasta: o.VigenteHasta}
}

func validarFichaLectura(x api.FichaPersona, persona string, ahora time.Time) bool {
	if x.PersonaRef != persona || !textoLectura(x.Nombre, 512, false) || !textoLectura(x.UnidadNombre, 512, true) || x.Perfiles == nil || x.ActosDisponibles == nil || x.Historia == nil || len(x.Perfiles) > limiteLecturasADMIN || len(x.ActosDisponibles) > limiteLecturasADMIN || len(x.Historia) > limiteLecturasADMIN {
		return false
	}
	perfiles := make(map[string]bool, len(x.Perfiles))
	for _, p := range x.Perfiles {
		if !referenciaLectura(p.PerfilRef, "prf_") || !domain.RolVersionAdministracionPerfilesValido(p.RolVersionRef) || !domain.EstadoVinculoContextoActor(p.Estado).Valido() || p.Version == 0 || !instantePersistible(p.VigenteHasta) || perfiles[p.PerfilRef] {
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
		if !domain.ReferenciaAdministracionPerfilesValida(h.ActoRef, "acto_admin:") || !domain.OperacionAdministracionPerfiles(h.Operacion).Valida() || !domain.EstadoVinculoContextoActor(h.Estado).Valido() || !instantePersistible(h.ConfirmadoEn) || historia[h.ActoRef] {
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
	if x.PuedeCerrar && (actor.PersonaRef == x.ProponentePersonaRef || actor.PersonaRef == x.ObjetivoPersonaRef || !x.CaducaEn.After(ahora) || len(x.MotivosCierre) == 0) {
		return false
	}
	return true
}
