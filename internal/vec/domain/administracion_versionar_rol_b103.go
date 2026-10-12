package domain

import (
	"errors"
	"reflect"
	"slices"
	"time"
)

var ErrVersionarRolBolsaB103Invalido = errors.New("admin_versionar_rol_bolsa_b103_invalido")

// El orden forma parte del material propuesto a los dos ADMIN. Las tres
// primeras capacidades son RRHH interno; las lecturas propias pertenecen al
// perfil externo y no se incorporan a este rol.
var accionesVersionarRolBolsaB103 = [...]string{
	"bolsa.inscripcion.rrhh.listar",
	"bolsa.inscripcion.rrhh.consultar",
	"bolsa.inscripcion.rrhh.convocatorias.listar",
	"bolsa.rrhh.respuestas_llamamiento.consultar",
}

// IntencionVersionarRolBolsaB103 sólo identifica la cabeza, el catálogo y
// preimágenes esperadas. No acredita publicación, identidad ni vigencia SQL.
type IntencionVersionarRolBolsaB103 struct {
	CatalogoRef          string
	CatalogoVersion      int
	CatalogoHuellaSHA256 string
	BaseRef              string
	BaseHuellaSHA256     string
	ControlRevision      uint64
	ControlHuellaSHA256  string
	Selecciones          []SeleccionAccionAdministracionV1
	Asignaciones         []SeleccionAsignacionVersionarRolBolsa
	Motivo               ReferenciaEntradaCatalogo
	ReferenciaActo       string
}

// PlanVersionarRolBolsaB103 mantiene la forma del plan Bolsa anterior salvo
// las cuatro selecciones nominales. Las asignaciones conservan sus ámbitos;
// bolsa_ref es dato de la consulta B103 y de su auditoría, no un ámbito nuevo.
type PlanVersionarRolBolsaB103 struct {
	Operacion             OperacionGobiernoPerfil                `json:"operacion"`
	CatalogoRef           string                                 `json:"catalogo_ref"`
	CatalogoVersion       int                                    `json:"catalogo_version"`
	CatalogoHuellaSHA256  string                                 `json:"catalogo_huella_sha256"`
	VersionRolObjetivoRef string                                 `json:"version_rol_objetivo_ref"`
	Base                  PerfilPublicadoAdministracionV1        `json:"base"`
	DefinicionNueva       DefinicionVersionPerfilGobernado       `json:"definicion_nueva"`
	Selecciones           []SeleccionAccionAdministracionV1      `json:"selecciones"`
	Asignaciones          []SeleccionAsignacionVersionarRolBolsa `json:"asignaciones"`
	Motivo                ReferenciaEntradaCatalogo              `json:"motivo"`
	ReferenciaActo        string                                 `json:"referencia_acto,omitempty"`
}

func (p PlanVersionarRolBolsaB103) HuellaSHA256() (string, error) {
	if p.Operacion != OperacionVersionarPerfilGobernado ||
		!textoAutorizacionSinComodinSeguro(p.CatalogoRef, 512, false) || p.CatalogoVersion < 1 ||
		!huellaSHA256AutorizacionV3NoNula(p.CatalogoHuellaSHA256) ||
		p.Base.Validar() != nil || p.Base.TipoPerfil != TipoPerfilAdministracionAdministrableV1 ||
		p.Base.Rol.RolID != RolIDVersionarBolsaB1 || p.Base.Rol.Estado != EstadoVersionRolPublicada ||
		p.Base.ControlVigencia.Estado != EstadoControlVigenciaVersionRolHabilitada ||
		p.Base.Rol.Version == int(^uint(0)>>1) ||
		p.DefinicionNueva.RolID != p.Base.Rol.RolID || p.DefinicionNueva.Version != p.Base.Rol.Version+1 ||
		p.DefinicionNueva.Nombre != p.Base.Rol.Nombre || p.VersionRolObjetivoRef != p.DefinicionNueva.Referencia() ||
		len(p.DefinicionNueva.Concesiones) != len(p.Base.Rol.Concesiones)+len(accionesVersionarRolBolsaB103) ||
		len(p.DefinicionNueva.Concesiones) > maximoElementosAutorizacion ||
		len(p.Selecciones) != len(accionesVersionarRolBolsaB103) ||
		len(p.Asignaciones) < 1 || len(p.Asignaciones) > 16 ||
		!ReferenciaMotivoAutorizacionV2Valida(p.Motivo) || !ReferenciaActoAdministracionValida(p.ReferenciaActo) {
		return "", ErrVersionarRolBolsaB103Invalido
	}
	for i, anterior := range p.Base.Rol.Concesiones {
		if !reflect.DeepEqual(anterior, p.DefinicionNueva.Concesiones[i]) ||
			accionVersionarRolBolsaB103(anterior.Accion) || accionPropiaInscripcionBolsa(anterior.Accion) {
			return "", ErrVersionarRolBolsaB103Invalido
		}
	}
	for i, accion := range accionesVersionarRolBolsaB103 {
		s := p.Selecciones[i]
		if !textoAutorizacionSinComodinSeguro(s.EntradaRef, 512, false) || s.EntradaVersion < 1 ||
			!huellaSHA256AutorizacionV3NoNula(s.EntradaHuellaSHA256) ||
			!concesionVersionarRolBolsaB103(p.DefinicionNueva.Concesiones[len(p.Base.Rol.Concesiones)+i], accion) {
			return "", ErrVersionarRolBolsaB103Invalido
		}
	}
	vistas := make(map[string]bool, len(p.Asignaciones))
	for _, a := range p.Asignaciones {
		if a.ValidarPara(p.Base.Rol.RolID, p.Base.Rol.Version) != nil ||
			!ambitosRRHHBolsaB103(a.Documento.Ambitos) || vistas[a.Documento.AsignacionID] {
			return "", ErrVersionarRolBolsaB103Invalido
		}
		vistas[a.Documento.AsignacionID] = true
	}
	return huellaAutorizacion(p)
}

func accionVersionarRolBolsaB103(accion string) bool {
	return slices.Contains(accionesVersionarRolBolsaB103[:], accion)
}

func accionPropiaInscripcionBolsa(accion string) bool {
	return accion == "bolsa.inscripcion.propias.listar" || accion == "bolsa.inscripcion.propia.consultar"
}

func ambitosRRHHBolsaB103(ambitos []AmbitoPerfil) bool {
	if len(ambitos) != 2 {
		return false
	}
	vistos := map[string]bool{}
	for _, a := range ambitos {
		if (a.Clave != "unidad_ref" && a.Clave != "ambito_ref") || a.Validar() != nil {
			return false
		}
		vistos[a.Clave] = true
	}
	return vistos["unidad_ref"] && vistos["ambito_ref"]
}

func dimensionesRRHHBolsaB103(dimensiones []string) bool {
	return len(dimensiones) == 2 && slices.Contains(dimensiones, "unidad_ref") &&
		slices.Contains(dimensiones, "ambito_ref")
}

func camposRRHHBolsaB103(accion string) []string {
	requisitos := []string{"codigo", "descripcion", "obligatorio", "estado", "motivo_codigo", "motivo_etiqueta",
		"procedencia_ref", "hito_cumplimiento", "hito_etiqueta", "hito_fecha"}
	solicitud := []string{"solicitud_ref", "recibo_ref", "convocatoria_ref", "convocatoria_titulo", "categoria_ref", "bolsa_ref",
		"categoria", "declaracion_ref", "bases_ref", "catalogo_version", "plazo_inicio", "plazo_fin",
		"decision_ref", "estado", "version", "registrada_en", "decidida_en", "motivo_codigo",
		"motivo_etiqueta", "participacion_ref", "persona_resumen"}
	for _, hoja := range requisitos {
		solicitud = append(solicitud, "requisitos[]."+hoja)
	}
	var campos []string
	switch accion {
	case accionesVersionarRolBolsaB103[0]:
		campos = []string{"total", "cursor_siguiente", "convocatoria_titulo"}
		for _, hoja := range solicitud {
			campos = append(campos, "solicitudes[]."+hoja)
		}
	case accionesVersionarRolBolsaB103[1]:
		campos = solicitud
	case accionesVersionarRolBolsaB103[2]:
		campos = []string{"convocatorias[].convocatoria_ref", "convocatorias[].titulo", "convocatorias[].categorias_resumen",
			"convocatorias[].plazo_fin", "convocatorias[].estado_publicacion", "convocatorias[].pendientes", "total", "cursor_siguiente"}
	case accionesVersionarRolBolsaB103[3]:
		campos = []string{"llamamiento_ref", "respuestas[].participacion_ref", "respuestas[].respuesta"}
	default:
		return nil
	}
	slices.Sort(campos)
	return campos
}

func concesionVersionarRolBolsaB103(c ConcesionRol, accion string) bool {
	if c.Validar() != nil || c.ModuloID != "bolsa" || c.Accion != accion || c.GarantiaMinima != AuthAssuranceHigh ||
		!slices.Equal(c.CamposPermitidos, camposRRHHBolsaB103(accion)) {
		return false
	}
	switch accion {
	case accionesVersionarRolBolsaB103[0], accionesVersionarRolBolsaB103[1]:
		return len(c.Obligaciones) == 0 && c.TipoRecurso == "solicitud_inscripcion" &&
			slices.Equal(c.Finalidades, []string{"consulta_inscripcion_rrhh"})
	case accionesVersionarRolBolsaB103[2]:
		return len(c.Obligaciones) == 0 && c.TipoRecurso == "conjunto_gestion_inscripcion" &&
			slices.Equal(c.Finalidades, []string{"consulta_convocatorias_gestion_rrhh"})
	case accionesVersionarRolBolsaB103[3]:
		return (len(c.Obligaciones) == 0 || slices.Equal(c.Obligaciones, []string{"auditar"})) &&
			c.TipoRecurso == "llamamiento_bolsa" && slices.Equal(c.Finalidades, []string{"seguimiento_contacto_rrhh"})
	default:
		return false
	}
}

func PrepararPlanVersionarRolBolsaB103(c CatalogoAccionesAdministracionV1,
	i IntencionVersionarRolBolsaB103, ahora time.Time) (PlanVersionarRolBolsaB103, error) {
	var vacio PlanVersionarRolBolsaB103
	hc, err := c.HuellaSHA256()
	if err != nil || !instanteAutorizacionCanonico(ahora) ||
		!vigenteAccionesAdministracionEn(c.VigenteDesde, c.VigenteHasta, ahora) ||
		c.Referencia != i.CatalogoRef || c.Version != i.CatalogoVersion || hc != i.CatalogoHuellaSHA256 ||
		len(i.Selecciones) != len(accionesVersionarRolBolsaB103) ||
		len(i.Asignaciones) < 1 || len(i.Asignaciones) > 16 {
		return vacio, ErrVersionarRolBolsaB103Invalido
	}
	var base *PerfilPublicadoAdministracionV1
	for j := range c.Perfiles {
		if c.Perfiles[j].Rol.RolID == RolIDVersionarBolsaB1 {
			base = &c.Perfiles[j]
			break
		}
	}
	if base == nil || base.TipoPerfil != TipoPerfilAdministracionAdministrableV1 ||
		base.Rol.Estado != EstadoVersionRolPublicada || base.ControlVigencia.Estado != EstadoControlVigenciaVersionRolHabilitada ||
		base.ControlVigencia.ActualizadoEn.After(ahora) || base.Rol.Version == int(^uint(0)>>1) {
		return vacio, ErrVersionarRolBolsaB103Invalido
	}
	hr, er := base.Rol.HuellaSHA256()
	hcontrol, ec := base.ControlVigencia.HuellaSHA256()
	if er != nil || ec != nil || i.BaseRef != base.Rol.Referencia() || i.BaseHuellaSHA256 != hr ||
		i.ControlRevision != base.ControlVigencia.Revision || i.ControlHuellaSHA256 != hcontrol ||
		len(base.Rol.Concesiones)+len(accionesVersionarRolBolsaB103) > maximoElementosAutorizacion {
		return vacio, ErrVersionarRolBolsaB103Invalido
	}
	for _, anterior := range base.Rol.Concesiones {
		if accionVersionarRolBolsaB103(anterior.Accion) || accionPropiaInscripcionBolsa(anterior.Accion) {
			return vacio, ErrVersionarRolBolsaB103Invalido
		}
	}
	concesiones := copiarConcesionesVersionarBolsa(base.Rol.Concesiones)
	for j, accion := range accionesVersionarRolBolsaB103 {
		s := i.Selecciones[j]
		var entrada *EntradaAccionAdministracionV1
		for k := range c.Entradas {
			e := &c.Entradas[k]
			h, eErr := e.HuellaSHA256()
			if eErr == nil && e.Referencia == s.EntradaRef && e.Version == s.EntradaVersion && h == s.EntradaHuellaSHA256 {
				entrada = e
				break
			}
		}
		if entrada == nil || entrada.ClaseControl != string(ClaseControlPerfilOrdinario) ||
			!vigenteAccionesAdministracionEn(entrada.VigenteDesde, entrada.VigenteHasta, ahora) ||
			!dimensionesRRHHBolsaB103(entrada.DimensionesAmbito) ||
			!concesionVersionarRolBolsaB103(entrada.Concesion, accion) {
			return vacio, ErrVersionarRolBolsaB103Invalido
		}
		concesiones = append(concesiones, copiarConcesionesVersionarBolsa([]ConcesionRol{entrada.Concesion})[0])
	}
	asignaciones := make([]SeleccionAsignacionVersionarRolBolsa, len(i.Asignaciones))
	for j, a := range i.Asignaciones {
		if a.ValidarPara(base.Rol.RolID, base.Rol.Version) != nil || !ambitosRRHHBolsaB103(a.Documento.Ambitos) {
			return vacio, ErrVersionarRolBolsaB103Invalido
		}
		asignaciones[j] = SeleccionAsignacionVersionarRolBolsa{AsignacionRef: a.AsignacionRef,
			HuellaSHA256: a.HuellaSHA256, Documento: copiarAsignacionVersionarBolsa(a.Documento)}
	}
	baseCopia := *base
	baseCopia.Rol.Concesiones = copiarConcesionesVersionarBolsa(base.Rol.Concesiones)
	plan := PlanVersionarRolBolsaB103{
		Operacion: OperacionVersionarPerfilGobernado, CatalogoRef: c.Referencia, CatalogoVersion: c.Version,
		CatalogoHuellaSHA256:  hc,
		VersionRolObjetivoRef: (VersionRol{RolID: base.Rol.RolID, Version: base.Rol.Version + 1}).Referencia(),
		Base:                  baseCopia, DefinicionNueva: DefinicionVersionPerfilGobernado{RolID: base.Rol.RolID,
			Version: base.Rol.Version + 1, Nombre: base.Rol.Nombre, Concesiones: concesiones},
		Selecciones:  append([]SeleccionAccionAdministracionV1(nil), i.Selecciones...),
		Asignaciones: asignaciones, Motivo: i.Motivo, ReferenciaActo: i.ReferenciaActo,
	}
	if _, err := plan.HuellaSHA256(); err != nil {
		return vacio, err
	}
	return plan, nil
}

func (p PlanVersionarRolBolsaB103) Copia() PlanVersionarRolBolsaB103 {
	p.Base.Rol.Concesiones = copiarConcesionesVersionarBolsa(p.Base.Rol.Concesiones)
	p.DefinicionNueva.Concesiones = copiarConcesionesVersionarBolsa(p.DefinicionNueva.Concesiones)
	p.Selecciones = append([]SeleccionAccionAdministracionV1(nil), p.Selecciones...)
	p.Asignaciones = append([]SeleccionAsignacionVersionarRolBolsa(nil), p.Asignaciones...)
	for j := range p.Asignaciones {
		p.Asignaciones[j].Documento = copiarAsignacionVersionarBolsa(p.Asignaciones[j].Documento)
	}
	return p
}
