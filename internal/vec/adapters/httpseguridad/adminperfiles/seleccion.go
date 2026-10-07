package adminperfiles

import (
	"context"
	"strings"
	"time"
)

// PerfilPropio procede del catálogo y de una asignación central vigente. La
// clave i18n nombra un texto del catálogo web; nunca contiene el nombre civil.
type PerfilPropio struct {
	PerfilRef      string
	RolVersionRef  string
	ClaveI18N      string
	CategoriaADMIN string
}

// PerfilesPropios contiene únicamente asignaciones de la cuenta acreditada.
// Revisión cero indica que aún no se ha elegido un perfil para esa cuenta.
type PerfilesPropios struct {
	Revision        uint64
	PerfilActivoRef string
	Perfiles        []PerfilPropio
}

type SeleccionPerfil struct {
	PerfilActivoRef string
	Revision        uint64
	SeleccionadaEn  time.Time
	AuditoriaRef    string
}

// FuenteSeleccionADMIN recibe observaciones del canal, nunca credenciales ni
// concesiones de un cuerpo HTTP. La elección solo nombra una asignación que la
// fuente central comprueba de nuevo bajo CAS.
type FuenteSeleccionADMIN interface {
	ListarPropiosADMIN(context.Context, ObservacionADMIN) (PerfilesPropios, error)
	SeleccionarPerfilADMIN(context.Context, ObservacionADMIN, string, uint64) (SeleccionPerfil, error)
}

func (p PerfilPropio) Valido() bool {
	return referencia(p.PerfilRef, "prf_") && strings.HasPrefix(p.RolVersionRef, "rol:") &&
		textoCatalogo(p.RolVersionRef, 512) && textoCatalogo(p.ClaveI18N, 256) &&
		(p.CategoriaADMIN == "aplicacion" || p.CategoriaADMIN == "sistemas")
}

func (p PerfilesPropios) Validos() bool {
	if len(p.Perfiles) > 16 {
		return false
	}
	vistos := make(map[string]bool, len(p.Perfiles))
	activo := false
	for _, perfil := range p.Perfiles {
		if !perfil.Valido() || vistos[perfil.PerfilRef] {
			return false
		}
		vistos[perfil.PerfilRef] = true
		activo = activo || perfil.PerfilRef == p.PerfilActivoRef
	}
	return p.PerfilActivoRef == "" || (p.Revision > 0 && activo)
}

func (s SeleccionPerfil) Valida() bool {
	return referencia(s.PerfilActivoRef, "prf_") && s.Revision > 0 && instante(s.SeleccionadaEn) &&
		huellaAuditoriaSeleccion(s.AuditoriaRef)
}

func textoCatalogo(valor string, limite int) bool {
	if valor == "" || len(valor) > limite {
		return false
	}
	for _, c := range valor {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' ||
			c == '_' || c == '-' || c == ':' || c == '.') {
			return false
		}
	}
	return true
}

func huellaAuditoriaSeleccion(valor string) bool {
	const comun = "aud_v3_p_"
	if strings.HasPrefix(valor, comun) && len(valor) == len(comun)+32 {
		for _, c := range valor[len(comun):] {
			if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
				return false
			}
		}
		return true
	}
	const prefijo = "auditoria_seleccion_admin:"
	if !strings.HasPrefix(valor, prefijo) || len(valor) != len(prefijo)+64 {
		return false
	}
	for _, c := range valor[len(prefijo):] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
