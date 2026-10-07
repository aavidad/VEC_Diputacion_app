package domain

import (
	"errors"
	"math"
	"time"
)

var ErrPlanGobiernoPerfilInvalido = errors.New("admin_plan_gobierno_perfil_invalido")

type OperacionGobiernoPerfil string

const (
	OperacionCrearPerfilGobernado      OperacionGobiernoPerfil = "crear"
	OperacionVersionarPerfilGobernado  OperacionGobiernoPerfil = "versionar"
	OperacionDeshabilitarVersionPerfil OperacionGobiernoPerfil = "deshabilitar"
)

func (o OperacionGobiernoPerfil) Valida() bool {
	return o == OperacionCrearPerfilGobernado || o == OperacionVersionarPerfilGobernado || o == OperacionDeshabilitarVersionPerfil
}

type SeleccionRetiradaVersionPerfil struct {
	CatalogoRef            string `json:"catalogo_ref"`
	CatalogoVersion        int    `json:"catalogo_version"`
	CatalogoHuellaSHA256   string `json:"catalogo_huella_sha256"`
	VersionRolRef          string `json:"version_rol_ref"`
	VersionRolHuellaSHA256 string `json:"version_rol_huella_sha256"`
	ControlRevision        uint64 `json:"control_revision"`
	ControlHuellaSHA256    string `json:"control_huella_sha256"`
}

// SolicitudPlanGobiernoPerfil es intención de datos, nunca evidencia ADMIN.
// Crear/versionar reutilizan la propuesta y selecciones del catálogo cerrado.
// Deshabilitar selecciona UNA versión exacta, no todas las versiones del rol.
type SolicitudPlanGobiernoPerfil struct {
	Operacion       OperacionGobiernoPerfil          `json:"operacion"`
	Publicacion     *PropuestaPerfilAdministracionV1 `json:"publicacion,omitempty"`
	Deshabilitacion *SeleccionRetiradaVersionPerfil  `json:"deshabilitacion,omitempty"`
	Motivo          ReferenciaEntradaCatalogo        `json:"motivo"`
	ReferenciaActo  string                           `json:"referencia_acto,omitempty"`
}

// DefinicionVersionPerfilGobernado excluye estado, autor y fecha de publicación
// suministrados por el archivo. La autoridad central produce esos metadatos
// reales al publicar; esta definición no es por sí sola un VersionRol publicado.
type DefinicionVersionPerfilGobernado struct {
	RolID       string         `json:"rol_id"`
	Version     int            `json:"version"`
	Nombre      string         `json:"nombre"`
	Concesiones []ConcesionRol `json:"concesiones"`
}

func (d DefinicionVersionPerfilGobernado) Referencia() string {
	return (VersionRol{RolID: d.RolID, Version: d.Version}).Referencia()
}

// PlanGobiernoPerfil es el recurso verificable del plan. Sus huellas prueban
// coherencia e integridad, no origen confiable ni publicación. Una ejecución
// nominal lo reconstruye contra la fuente central completa y vuelve a comprobar
// catálogo, base, control y permisos dentro de su transacción.
type PlanGobiernoPerfil struct {
	Operacion             OperacionGobiernoPerfil           `json:"operacion"`
	CatalogoRef           string                            `json:"catalogo_ref"`
	CatalogoVersion       int                               `json:"catalogo_version"`
	CatalogoHuellaSHA256  string                            `json:"catalogo_huella_sha256"`
	VersionRolObjetivoRef string                            `json:"version_rol_objetivo_ref"`
	Base                  *PerfilPublicadoAdministracionV1  `json:"base,omitempty"`
	DefinicionNueva       *DefinicionVersionPerfilGobernado `json:"definicion_nueva,omitempty"`
	Selecciones           []SeleccionAccionAdministracionV1 `json:"selecciones,omitempty"`
	Motivo                ReferenciaEntradaCatalogo         `json:"motivo"`
	ReferenciaActo        string                            `json:"referencia_acto,omitempty"`
}

func (p PlanGobiernoPerfil) HuellaSHA256() (string, error) {
	if !p.Operacion.Valida() || !textoAutorizacionSinComodinSeguro(p.CatalogoRef, 512, false) ||
		p.CatalogoVersion < 1 || !huellaSHA256AutorizacionV3NoNula(p.CatalogoHuellaSHA256) ||
		!RolVersionAdministracionPerfilesValido(p.VersionRolObjetivoRef) ||
		!ReferenciaMotivoAutorizacionV2Valida(p.Motivo) || !ReferenciaActoAdministracionValida(p.ReferenciaActo) {
		return "", ErrPlanGobiernoPerfilInvalido
	}
	if p.Base != nil && (p.Base.Validar() != nil || p.Base.TipoPerfil != TipoPerfilAdministracionAdministrableV1 ||
		p.Base.Rol.Estado != EstadoVersionRolPublicada || p.Base.ControlVigencia.Estado != EstadoControlVigenciaVersionRolHabilitada) {
		return "", ErrPlanGobiernoPerfilInvalido
	}
	switch p.Operacion {
	case OperacionCrearPerfilGobernado:
		if p.Base != nil || p.DefinicionNueva == nil || p.DefinicionNueva.Version != 1 {
			return "", ErrPlanGobiernoPerfilInvalido
		}
	case OperacionVersionarPerfilGobernado:
		if p.Base == nil || p.DefinicionNueva == nil || p.DefinicionNueva.RolID != p.Base.Rol.RolID ||
			p.DefinicionNueva.Version < 2 || p.DefinicionNueva.Version-1 != p.Base.Rol.Version {
			return "", ErrPlanGobiernoPerfilInvalido
		}
	case OperacionDeshabilitarVersionPerfil:
		if p.Base == nil || p.DefinicionNueva != nil || len(p.Selecciones) != 0 ||
			p.Base.Rol.Referencia() != p.VersionRolObjetivoRef || p.Base.ControlVigencia.Revision == math.MaxUint64 {
			return "", ErrPlanGobiernoPerfilInvalido
		}
	}
	if p.DefinicionNueva != nil {
		d := p.DefinicionNueva
		if d.Referencia() != p.VersionRolObjetivoRef || !textoAutorizacionSinComodinSeguro(d.RolID, 128, false) ||
			!textoAutorizacionSeguro(d.Nombre, 512, true) || len(d.Concesiones) == 0 || len(d.Concesiones) > maximoElementosAutorizacion ||
			len(d.Concesiones) != len(p.Selecciones) {
			return "", ErrPlanGobiernoPerfilInvalido
		}
		for _, c := range d.Concesiones {
			if c.Validar() != nil {
				return "", ErrPlanGobiernoPerfilInvalido
			}
		}
	}
	return huellaAutorizacion(p)
}

func PrepararPlanGobiernoPerfil(c CatalogoAccionesAdministracionV1, s SolicitudPlanGobiernoPerfil, instante time.Time) (PlanGobiernoPerfil, error) {
	if c.Validar() != nil {
		return PlanGobiernoPerfil{}, ErrCatalogoAccionesAdministracionInvalido
	}
	if !instanteAutorizacionCanonico(instante) || !vigenteAccionesAdministracionEn(c.VigenteDesde, c.VigenteHasta, instante) {
		return PlanGobiernoPerfil{}, ErrCatalogoAccionesAdministracionNoVigente
	}
	if !s.Operacion.Valida() || !ReferenciaMotivoAutorizacionV2Valida(s.Motivo) || !ReferenciaActoAdministracionValida(s.ReferenciaActo) {
		return PlanGobiernoPerfil{}, ErrPlanGobiernoPerfilInvalido
	}
	hc, err := c.HuellaSHA256()
	if err != nil {
		return PlanGobiernoPerfil{}, err
	}
	p := PlanGobiernoPerfil{Operacion: s.Operacion, CatalogoRef: c.Referencia, CatalogoVersion: c.Version,
		CatalogoHuellaSHA256: hc, Motivo: s.Motivo, ReferenciaActo: s.ReferenciaActo}
	if s.Operacion != OperacionDeshabilitarVersionPerfil {
		if s.Publicacion == nil || s.Deshabilitacion != nil {
			return PlanGobiernoPerfil{}, ErrPlanGobiernoPerfilInvalido
		}
		if _, err := ComprobarPropuestaPerfilAdministracionV1(c, *s.Publicacion, instante); err != nil {
			return PlanGobiernoPerfil{}, err
		}
		r := s.Publicacion.RolPropuesto
		p.VersionRolObjetivoRef = r.Referencia()
		p.DefinicionNueva = &DefinicionVersionPerfilGobernado{RolID: r.RolID, Version: r.Version, Nombre: r.Nombre,
			Concesiones: copiarConcesionesGobiernoPerfil(r.Concesiones)}
		p.Selecciones = append([]SeleccionAccionAdministracionV1(nil), s.Publicacion.Selecciones...)
		for _, perfil := range c.Perfiles {
			if perfil.Rol.RolID == r.RolID {
				base := perfil
				base.Rol.Concesiones = copiarConcesionesGobiernoPerfil(perfil.Rol.Concesiones)
				p.Base = &base
			}
		}
	} else {
		if s.Publicacion != nil || s.Deshabilitacion == nil {
			return PlanGobiernoPerfil{}, ErrPlanGobiernoPerfilInvalido
		}
		r := s.Deshabilitacion
		if r.CatalogoRef != c.Referencia || r.CatalogoVersion != c.Version || r.CatalogoHuellaSHA256 != hc {
			return PlanGobiernoPerfil{}, ErrOrigenPerfilAdministracionNoCoincide
		}
		for _, perfil := range c.Perfiles {
			if perfil.Rol.Referencia() != r.VersionRolRef {
				continue
			}
			if perfil.TipoPerfil == TipoPerfilAdministracionFijoSistemaV1 {
				return PlanGobiernoPerfil{}, ErrPerfilAdministracionFijo
			}
			hr, er := perfil.Rol.HuellaSHA256()
			if er != nil {
				return PlanGobiernoPerfil{}, er
			}
			hcontrol, ec := perfil.ControlVigencia.HuellaSHA256()
			if ec != nil {
				return PlanGobiernoPerfil{}, ec
			}
			if hr != r.VersionRolHuellaSHA256 ||
				perfil.ControlVigencia.Revision != r.ControlRevision || hcontrol != r.ControlHuellaSHA256 ||
				perfil.ControlVigencia.ActualizadoEn.After(instante) {
				return PlanGobiernoPerfil{}, ErrOrigenPerfilAdministracionNoCoincide
			}
			base := perfil
			base.Rol.Concesiones = copiarConcesionesGobiernoPerfil(perfil.Rol.Concesiones)
			p.Base, p.VersionRolObjetivoRef = &base, r.VersionRolRef
		}
	}
	if _, err := p.HuellaSHA256(); err != nil {
		return PlanGobiernoPerfil{}, err
	}
	return p, nil
}

func copiarConcesionesGobiernoPerfil(original []ConcesionRol) []ConcesionRol {
	copia := append([]ConcesionRol(nil), original...)
	for i := range copia {
		copia[i].Finalidades = append([]string(nil), original[i].Finalidades...)
		copia[i].CamposPermitidos = append([]string(nil), original[i].CamposPermitidos...)
		copia[i].Obligaciones = append([]string(nil), original[i].Obligaciones...)
	}
	return copia
}

func (p PlanGobiernoPerfil) Copia() PlanGobiernoPerfil {
	if p.Base != nil {
		base := *p.Base
		base.Rol.Concesiones = copiarConcesionesGobiernoPerfil(base.Rol.Concesiones)
		p.Base = &base
	}
	if p.DefinicionNueva != nil {
		d := *p.DefinicionNueva
		d.Concesiones = copiarConcesionesGobiernoPerfil(d.Concesiones)
		p.DefinicionNueva = &d
	}
	p.Selecciones = append([]SeleccionAccionAdministracionV1(nil), p.Selecciones...)
	return p
}
