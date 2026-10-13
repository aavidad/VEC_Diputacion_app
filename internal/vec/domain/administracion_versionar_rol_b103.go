package domain

import (
	"errors"
	"reflect"
	"slices"
	"strings"
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

var accionesInscripcionRRHHB103 = [...]string{
	"bolsa.inscripcion.rrhh.motivos", "bolsa.inscripcion.rrhh.decidir", "bolsa.inscripcion.rrhh.incorporar",
}

// La ampliación C3 usa el mismo acto de dos ADMIN y conserva el plan histórico.
const AccionLecturaOrigenContactoBolsa = "bolsa.datos_contacto_participacion.origen.consultar"

func CamposLecturaOrigenContactoBolsa() []string {
	return []string{"enmascarados", "origen", "participacion_ref", "registrada_en", "sin_contacto", "version"}
}

func accionesDeltaVersionarBolsa(base VersionRol, selecciones int) []string {
	if selecciones == len(accionesInscripcionRRHHB103) {
		if base.RolID != RolIDVersionarBolsaB1 || base.Version < 1 {
			return nil
		}
		for _, c := range base.Concesiones {
			if slices.Contains(accionesInscripcionRRHHB103[:], c.Accion) {
				return nil
			}
		}
		return accionesInscripcionRRHHB103[:]
	}
	if selecciones == len(accionesVersionarRolBolsaB103) {
		for _, c := range base.Concesiones {
			if accionVersionarRolBolsaB103(c.Accion) || c.Accion == AccionLecturaOrigenContactoBolsa {
				return nil
			}
		}
		return accionesVersionarRolBolsaB103[:]
	}
	if selecciones != 1 {
		return nil
	}
	for _, accion := range accionesVersionarRolBolsaB103 {
		coincidencias := 0
		for _, c := range base.Concesiones {
			if c.Accion == accion {
				if !concesionVersionarRolBolsaB103(c, accion) {
					return nil
				}
				coincidencias++
			}
			if c.Accion == AccionLecturaOrigenContactoBolsa {
				return nil
			}
		}
		if coincidencias != 1 {
			return nil
		}
	}
	return []string{AccionLecturaOrigenContactoBolsa}
}

func concesionDeltaRRHHB103(c ConcesionRol, accion string) bool {
	if !slices.Contains(accionesInscripcionRRHHB103[:], accion) {
		return concesionVersionarRolBolsaB103(c, accion)
	}
	if c.Validar() != nil || c.ModuloID != "bolsa" || c.Accion != accion ||
		c.GarantiaMinima != AuthAssuranceHigh || len(c.Obligaciones) != 0 {
		return false
	}
	switch accion {
	case accionesInscripcionRRHHB103[0]:
		return c.TipoRecurso == "motivos_inscripcion" &&
			slices.Equal(c.Finalidades, []string{"consulta_motivos_inscripcion_rrhh"}) &&
			slices.Equal(c.CamposPermitidos, []string{"catalogo_version", "motivos[].codigo", "motivos[].etiqueta", "motivos[].obligatorio"})
	case accionesInscripcionRRHHB103[1]:
		return c.TipoRecurso == "solicitud_inscripcion" && slices.Equal(c.Finalidades, []string{"revisar_inscripcion"}) && len(c.CamposPermitidos) == 0
	case accionesInscripcionRRHHB103[2]:
		return c.TipoRecurso == "solicitud_inscripcion" && slices.Equal(c.Finalidades, []string{"incorporar_inscripcion"}) && len(c.CamposPermitidos) == 0
	}
	return false
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

// PlanVersionarRolBolsaB103 mantiene la forma del plan Bolsa anterior. Las
// asignaciones conservan sus ámbitos;
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
		len(p.Selecciones) == 0 || len(p.Selecciones) > maximoElementosAutorizacion-len(p.Base.Rol.Concesiones) ||
		len(p.DefinicionNueva.Concesiones) != len(p.Base.Rol.Concesiones)+len(p.Selecciones) ||
		len(p.DefinicionNueva.Concesiones) > maximoElementosAutorizacion ||
		len(p.Asignaciones) < 1 || len(p.Asignaciones) > 16 ||
		!ReferenciaMotivoAutorizacionV2Valida(p.Motivo) || !ReferenciaActoAdministracionValida(p.ReferenciaActo) {
		return "", ErrVersionarRolBolsaB103Invalido
	}
	for i, anterior := range p.Base.Rol.Concesiones {
		if !reflect.DeepEqual(anterior, p.DefinicionNueva.Concesiones[i]) {
			return "", ErrVersionarRolBolsaB103Invalido
		}
	}
	vistas := make(map[string]bool, len(p.Selecciones))
	acciones := make(map[string]bool, len(p.Base.Rol.Concesiones)+len(p.Selecciones))
	for _, anterior := range p.Base.Rol.Concesiones {
		acciones[anterior.Accion] = true
	}
	for i := range p.Selecciones {
		s := p.Selecciones[i]
		concesion := p.DefinicionNueva.Concesiones[len(p.Base.Rol.Concesiones)+i]
		if !textoAutorizacionSinComodinSeguro(s.EntradaRef, 512, false) || s.EntradaVersion < 1 ||
			!huellaSHA256AutorizacionV3NoNula(s.EntradaHuellaSHA256) ||
			vistas[s.EntradaRef] || concesion.Validar() != nil || concesion.ModuloID != "bolsa" ||
			!strings.HasPrefix(concesion.Accion, "bolsa.") || accionExternaBolsaB103(concesion.Accion) ||
			acciones[concesion.Accion] {
			return "", ErrVersionarRolBolsaB103Invalido
		}
		vistas[s.EntradaRef] = true
		acciones[concesion.Accion] = true
	}
	vistasAsignacion := make(map[string]bool, len(p.Asignaciones))
	for _, a := range p.Asignaciones {
		if a.ValidarPara(p.Base.Rol.RolID, p.Base.Rol.Version) != nil ||
			!ambitosRRHHBolsaB103(a.Documento.Ambitos) || vistasAsignacion[a.Documento.AsignacionID] {
			return "", ErrVersionarRolBolsaB103Invalido
		}
		vistasAsignacion[a.Documento.AsignacionID] = true
	}
	return huellaAutorizacion(p)
}

func accionVersionarRolBolsaB103(accion string) bool {
	return slices.Contains(accionesVersionarRolBolsaB103[:], accion)
}

func accionPropiaInscripcionBolsa(accion string) bool {
	return accion == "bolsa.inscripcion.propias.listar" || accion == "bolsa.inscripcion.propia.consultar"
}

// La fuente RRHH no puede incorporar al rol interno las capacidades que
// pertenecen al perfil candidato externo, aunque alguien les diera por error
// dimensiones de unidad y ámbito dentro del catálogo administrativo.
func accionExternaBolsaB103(accion string) bool {
	return slices.Contains(accionesBaseVersionarRolBolsaExterna[:], accion) ||
		slices.Contains(accionesVersionarRolBolsaExterna[:], accion) ||
		slices.Contains(accionesInscripcionBolsaExterna[:], accion)
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
	case AccionLecturaOrigenContactoBolsa:
		campos = CamposLecturaOrigenContactoBolsa()
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
	case AccionLecturaOrigenContactoBolsa:
		return (len(c.Obligaciones) == 0 || slices.Equal(c.Obligaciones, []string{"auditar"})) && c.TipoRecurso == "participacion_bolsa" && slices.Equal(c.Finalidades, []string{"consulta_origen_datos_contacto_participacion"})
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
		len(i.Selecciones) == 0 || len(i.Selecciones) > maximoElementosAutorizacion ||
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
	if len(i.Selecciones) > maximoElementosAutorizacion-len(base.Rol.Concesiones) {
		return vacio, ErrVersionarRolBolsaB103Invalido
	}
	hr, er := base.Rol.HuellaSHA256()
	hcontrol, ec := base.ControlVigencia.HuellaSHA256()
	if er != nil || ec != nil || i.BaseRef != base.Rol.Referencia() || i.BaseHuellaSHA256 != hr ||
		i.ControlRevision != base.ControlVigencia.Revision || i.ControlHuellaSHA256 != hcontrol ||
		len(base.Rol.Concesiones)+len(i.Selecciones) > maximoElementosAutorizacion {
		return vacio, ErrVersionarRolBolsaB103Invalido
	}
	concesiones := copiarConcesionesVersionarBolsa(base.Rol.Concesiones)
	acciones := make(map[string]bool, len(base.Rol.Concesiones)+len(i.Selecciones))
	for _, anterior := range base.Rol.Concesiones {
		acciones[anterior.Accion] = true
	}
	vistas := make(map[string]bool, len(i.Selecciones))
	for _, s := range i.Selecciones {
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
			entrada.Concesion.ModuloID != "bolsa" || !strings.HasPrefix(entrada.Concesion.Accion, "bolsa.") ||
			accionExternaBolsaB103(entrada.Concesion.Accion) ||
			vistas[s.EntradaRef] || acciones[entrada.Concesion.Accion] {
			return vacio, ErrVersionarRolBolsaB103Invalido
		}
		vistas[s.EntradaRef] = true
		acciones[entrada.Concesion.Accion] = true
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
