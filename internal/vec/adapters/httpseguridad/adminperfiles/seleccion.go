package adminperfiles

import (
	"context"
	"strings"
	"time"
)

// PerfilPropio procede del catálogo y de una asignación central vigente. La
// clave i18n nombra un texto del catálogo web; nunca contiene el nombre civil.
type PerfilPropio struct {
	PerfilRef     string
	RolVersionRef string
	ClaveI18N     string
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
		len(p.RolVersionRef) <= 512 && p.ClaveI18N != "" && len(p.ClaveI18N) <= 256
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
		s.AuditoriaRef != "" && len(s.AuditoriaRef) <= 512
}
