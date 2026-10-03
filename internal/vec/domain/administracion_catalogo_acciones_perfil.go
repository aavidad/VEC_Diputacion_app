package domain

import "time"

type SeleccionAccionAdministracionV1 struct {
	EntradaRef          string `json:"entrada_ref"`
	EntradaVersion      int    `json:"entrada_version"`
	EntradaHuellaSHA256 string `json:"entrada_huella_sha256"`
}

// PropuestaPerfilAdministracionV1 no representa un rol publicado ni una
// decisión PDP. Conserva la base exacta y todas las entradas seleccionadas.
type PropuestaPerfilAdministracionV1 struct {
	CatalogoRef                string                            `json:"catalogo_ref"`
	CatalogoVersion            int                               `json:"catalogo_version"`
	CatalogoHuellaSHA256       string                            `json:"catalogo_huella_sha256"`
	VersionRolBaseRef          string                            `json:"version_rol_base_ref,omitempty"`
	VersionRolBaseHuellaSHA256 string                            `json:"version_rol_base_huella_sha256,omitempty"`
	RolPropuesto               VersionRol                        `json:"rol_propuesto"`
	Selecciones                []SeleccionAccionAdministracionV1 `json:"selecciones"`
}

// DictamenPerfilAdministracionV1 solo documenta la comprobación. Su huella no
// permite ejecutar ni publicar el rol. El efecto futuro exige autorización,
// doble control y registro duradero de solo adición por los puertos centrales.
type DictamenPerfilAdministracionV1 struct {
	CatalogoRef            string    `json:"catalogo_ref"`
	CatalogoVersion        int       `json:"catalogo_version"`
	CatalogoHuellaSHA256   string    `json:"catalogo_huella_sha256"`
	VersionRolRef          string    `json:"version_rol_ref"`
	VersionRolHuellaSHA256 string    `json:"version_rol_huella_sha256"`
	PropuestaHuellaSHA256  string    `json:"propuesta_huella_sha256"`
	ComprobadoEn           time.Time `json:"comprobado_en"`
}

func ComprobarPropuestaPerfilAdministracionV1(c CatalogoAccionesAdministracionV1, p PropuestaPerfilAdministracionV1, instante time.Time) (DictamenPerfilAdministracionV1, error) {
	if c.Validar() != nil {
		return DictamenPerfilAdministracionV1{}, ErrCatalogoAccionesAdministracionInvalido
	}
	if !instanteAutorizacionCanonico(instante) || !vigenteAccionesAdministracionEn(c.VigenteDesde, c.VigenteHasta, instante) {
		return DictamenPerfilAdministracionV1{}, ErrCatalogoAccionesAdministracionNoVigente
	}
	huella, err := c.HuellaSHA256()
	if err != nil || p.CatalogoRef != c.Referencia || p.CatalogoVersion != c.Version || p.CatalogoHuellaSHA256 != huella {
		return DictamenPerfilAdministracionV1{}, ErrOrigenPerfilAdministracionNoCoincide
	}
	if p.RolPropuesto.Validar() != nil || p.RolPropuesto.Estado != EstadoVersionRolPublicada ||
		p.RolPropuesto.PublicadaEn.Before(c.VigenteDesde) || p.RolPropuesto.PublicadaEn.After(instante) || len(p.Selecciones) == 0 ||
		len(p.Selecciones) != len(p.RolPropuesto.Concesiones) || len(p.Selecciones) > maximoElementosAutorizacion {
		return DictamenPerfilAdministracionV1{}, ErrPropuestaPerfilAdministracionInvalida
	}
	if err := comprobarBasePerfilAdministracionV1(c, p, instante); err != nil {
		return DictamenPerfilAdministracionV1{}, err
	}
	entradas := make(map[string]EntradaAccionAdministracionV1, len(c.Entradas))
	for _, entrada := range c.Entradas {
		entradas[entrada.Referencia] = entrada
	}
	vistas := make(map[string]bool, len(p.Selecciones))
	for i, s := range p.Selecciones {
		e, encontrada := entradas[s.EntradaRef]
		huellaEntrada, err := e.HuellaSHA256()
		if !encontrada || err != nil || vistas[s.EntradaRef] || s.EntradaVersion != e.Version ||
			s.EntradaHuellaSHA256 != huellaEntrada || !vigenteAccionesAdministracionEn(e.VigenteDesde, e.VigenteHasta, instante) ||
			!vigenteAccionesAdministracionEn(e.VigenteDesde, e.VigenteHasta, p.RolPropuesto.PublicadaEn) ||
			!concesionesPerfilAdministracionIguales(e.Concesion, p.RolPropuesto.Concesiones[i]) {
			return DictamenPerfilAdministracionV1{}, ErrPermisoPerfilAdministracionNoCoincide
		}
		vistas[s.EntradaRef] = true
	}
	huellaRol, err := p.RolPropuesto.HuellaSHA256()
	if err != nil {
		return DictamenPerfilAdministracionV1{}, ErrPropuestaPerfilAdministracionInvalida
	}
	huellaPropuesta, err := huellaAutorizacion(p)
	if err != nil {
		return DictamenPerfilAdministracionV1{}, ErrPropuestaPerfilAdministracionInvalida
	}
	return DictamenPerfilAdministracionV1{CatalogoRef: c.Referencia, CatalogoVersion: c.Version, CatalogoHuellaSHA256: huella,
		VersionRolRef: p.RolPropuesto.Referencia(), VersionRolHuellaSHA256: huellaRol, PropuestaHuellaSHA256: huellaPropuesta, ComprobadoEn: instante}, nil
}

func comprobarBasePerfilAdministracionV1(c CatalogoAccionesAdministracionV1, p PropuestaPerfilAdministracionV1, instante time.Time) error {
	for _, publicado := range c.Perfiles {
		if publicado.Rol.RolID != p.RolPropuesto.RolID {
			continue
		}
		if publicado.TipoPerfil == TipoPerfilAdministracionFijoSistemaV1 {
			return ErrPerfilAdministracionFijo
		}
		huella, err := publicado.Rol.HuellaSHA256()
		if err != nil || p.VersionRolBaseRef != publicado.Rol.Referencia() || p.VersionRolBaseHuellaSHA256 != huella ||
			p.RolPropuesto.Version != publicado.Rol.Version+1 || publicado.Rol.Estado != EstadoVersionRolPublicada ||
			publicado.ControlVigencia.Estado != EstadoControlVigenciaVersionRolHabilitada || publicado.ControlVigencia.ActualizadoEn.After(instante) ||
			p.RolPropuesto.PublicadaEn.Before(publicado.ControlVigencia.ActualizadoEn) {
			return ErrOrigenPerfilAdministracionNoCoincide
		}
		return nil
	}
	if p.VersionRolBaseRef != "" || p.VersionRolBaseHuellaSHA256 != "" || p.RolPropuesto.Version != 1 {
		return ErrOrigenPerfilAdministracionNoCoincide
	}
	return nil
}

func concesionesPerfilAdministracionIguales(a, b ConcesionRol) bool {
	return a.Accion == b.Accion && a.ModuloID == b.ModuloID && a.TipoRecurso == b.TipoRecurso && a.GarantiaMinima == b.GarantiaMinima &&
		listasPerfilAdministracionIguales(a.Finalidades, b.Finalidades) && listasPerfilAdministracionIguales(a.CamposPermitidos, b.CamposPermitidos) &&
		listasPerfilAdministracionIguales(a.Obligaciones, b.Obligaciones)
}

func listasPerfilAdministracionIguales(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
