package domain

import (
	"errors"
	"time"
)

var (
	ErrCatalogoAccionesAdministracionInvalido  = errors.New("admin_catalogo_acciones_invalido")
	ErrPropuestaPerfilAdministracionInvalida   = errors.New("admin_propuesta_perfil_invalida")
	ErrPerfilAdministracionFijo                = errors.New("admin_perfil_fijo")
	ErrCatalogoAccionesAdministracionNoVigente = errors.New("admin_catalogo_acciones_no_vigente")
	ErrOrigenPerfilAdministracionNoCoincide    = errors.New("admin_origen_perfil_no_coincide")
	ErrPermisoPerfilAdministracionNoCoincide   = errors.New("admin_permiso_perfil_no_coincide")
)

// EntradaAccionAdministracionV1 es una capacidad publicada por el módulo.
// Concesion conserva también garantía y obligaciones; las dimensiones se
// aplicarán en la asignación, nunca se infieren desde VersionRol.
type EntradaAccionAdministracionV1 struct {
	Referencia         string       `json:"referencia"`
	Version            int          `json:"version"`
	FuenteRef          string       `json:"fuente_ref"`
	FuenteVersion      int          `json:"fuente_version"`
	FuenteHuellaSHA256 string       `json:"fuente_huella_sha256"`
	Concesion          ConcesionRol `json:"concesion"`
	DimensionesAmbito  []string     `json:"dimensiones_ambito"`
	ClaseControl       string       `json:"clase_control"`
	VigenteDesde       time.Time    `json:"vigente_desde"`
	VigenteHasta       time.Time    `json:"vigente_hasta,omitempty"`
}

func (e EntradaAccionAdministracionV1) Validar() error {
	if !textoAutorizacionSinComodinSeguro(e.Referencia, 512, false) || e.Version < 1 ||
		!textoAutorizacionSinComodinSeguro(e.FuenteRef, 512, false) || e.FuenteVersion < 1 ||
		!huellaSHA256AutorizacionV3NoNula(e.FuenteHuellaSHA256) || e.Concesion.Validar() != nil ||
		!listaAutorizacionValida(e.DimensionesAmbito, false, true) ||
		!textoAutorizacionSinComodinSeguro(e.ClaseControl, 128, false) ||
		!intervaloAccionesAdministracionValido(e.VigenteDesde, e.VigenteHasta) {
		return ErrCatalogoAccionesAdministracionInvalido
	}
	for _, dimension := range e.DimensionesAmbito {
		// La entrada debe poder trasladarse a AmbitoPerfil sin recortar ni
		// normalizar el nombre de la dimensión central.
		if !textoAutorizacionSinComodinSeguro(dimension, 128, false) || dimension == "global" {
			return ErrCatalogoAccionesAdministracionInvalido
		}
	}
	return nil
}

func (e EntradaAccionAdministracionV1) HuellaSHA256() (string, error) {
	if err := e.Validar(); err != nil {
		return "", err
	}
	return huellaAutorizacion(e)
}

type TipoPerfilAdministracionV1 string

const (
	TipoPerfilAdministracionFijoSistemaV1   TipoPerfilAdministracionV1 = "fijo_sistema"
	TipoPerfilAdministracionAdministrableV1 TipoPerfilAdministracionV1 = "administrable"
)

// PerfilPublicadoAdministracionV1 procede de la fuente administrativa, no de
// la propuesta. El tipo fijo protege la semilla incluso ante otra versión.
type PerfilPublicadoAdministracionV1 struct {
	Rol             VersionRol                 `json:"rol"`
	ControlVigencia ControlVigenciaVersionRol  `json:"control_vigencia"`
	TipoPerfil      TipoPerfilAdministracionV1 `json:"tipo_perfil"`
}

func (p PerfilPublicadoAdministracionV1) Validar() error {
	if (p.TipoPerfil != TipoPerfilAdministracionFijoSistemaV1 && p.TipoPerfil != TipoPerfilAdministracionAdministrableV1) ||
		p.Rol.Validar() != nil || p.ControlVigencia.Validar() != nil ||
		p.ControlVigencia.VersionRolRef != p.Rol.Referencia() ||
		p.ControlVigencia.ActualizadoEn.Before(p.Rol.PublicadaEn) {
		return ErrCatalogoAccionesAdministracionInvalido
	}
	return nil
}

// CatalogoAccionesAdministracionV1 es una instantánea de datos de solo lectura.
// Una huella comprueba integridad, no prueba publicación ni concede acceso.
// La aplicación deberá resolver esta versión en una fuente confiable.
type CatalogoAccionesAdministracionV1 struct {
	Referencia         string                            `json:"referencia"`
	Version            int                               `json:"version"`
	FuenteRef          string                            `json:"fuente_ref"`
	FuenteVersion      int                               `json:"fuente_version"`
	FuenteHuellaSHA256 string                            `json:"fuente_huella_sha256"`
	VigenteDesde       time.Time                         `json:"vigente_desde"`
	VigenteHasta       time.Time                         `json:"vigente_hasta,omitempty"`
	Entradas           []EntradaAccionAdministracionV1   `json:"entradas"`
	Perfiles           []PerfilPublicadoAdministracionV1 `json:"perfiles"`
}

func (c CatalogoAccionesAdministracionV1) Validar() error {
	if !textoAutorizacionSinComodinSeguro(c.Referencia, 512, false) || c.Version < 1 ||
		!textoAutorizacionSinComodinSeguro(c.FuenteRef, 512, false) || c.FuenteVersion < 1 ||
		!huellaSHA256AutorizacionV3NoNula(c.FuenteHuellaSHA256) ||
		!intervaloAccionesAdministracionValido(c.VigenteDesde, c.VigenteHasta) ||
		len(c.Entradas) == 0 || len(c.Entradas) > maximoElementosAutorizacion ||
		len(c.Perfiles) > maximoElementosAutorizacion {
		return ErrCatalogoAccionesAdministracionInvalido
	}
	refs := make(map[string]bool, len(c.Entradas))
	acciones := make(map[string]bool, len(c.Entradas))
	for _, e := range c.Entradas {
		clave := e.Concesion.ModuloID + "\x00" + e.Concesion.Accion + "\x00" + e.Concesion.TipoRecurso
		if e.Validar() != nil || refs[e.Referencia] || acciones[clave] ||
			e.VigenteDesde.Before(c.VigenteDesde) || (!c.VigenteHasta.IsZero() &&
			(e.VigenteHasta.IsZero() || e.VigenteHasta.After(c.VigenteHasta))) {
			return ErrCatalogoAccionesAdministracionInvalido
		}
		refs[e.Referencia], acciones[clave] = true, true
	}
	roles := make(map[string]bool, len(c.Perfiles))
	for _, p := range c.Perfiles {
		if p.Validar() != nil || roles[p.Rol.RolID] {
			return ErrCatalogoAccionesAdministracionInvalido
		}
		roles[p.Rol.RolID] = true
	}
	return nil
}

func (c CatalogoAccionesAdministracionV1) HuellaSHA256() (string, error) {
	if err := c.Validar(); err != nil {
		return "", err
	}
	return huellaAutorizacion(c)
}

func intervaloAccionesAdministracionValido(desde, hasta time.Time) bool {
	return instanteAutorizacionCanonico(desde) && (hasta.IsZero() ||
		(instanteAutorizacionCanonico(hasta) && hasta.After(desde)))
}

func vigenteAccionesAdministracionEn(desde, hasta, instante time.Time) bool {
	return !instante.Before(desde) && (hasta.IsZero() || instante.Before(hasta))
}
