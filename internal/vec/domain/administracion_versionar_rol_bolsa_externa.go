package domain

import (
	"errors"
	"reflect"
	"slices"
	"time"
)

var ErrVersionarRolBolsaExternaInvalido = errors.New("admin_versionar_rol_bolsa_externa_invalido")

// El orden de las dos concesiones pertenece al plan firmado por dos ADMIN.
const RolIDVersionarBolsaExterna = "candidato_bolsa_portal_historial_propio_desarrollo"

var accionesVersionarRolBolsaExterna = [...]string{
	"bolsa.preferencias_bolsas.consultar_propias",
	"bolsa.preferencias_bolsas.ordenar_propias",
}

var accionesInscripcionBolsaExterna = [...]string{
	"bolsa.inscripcion.convocatorias.listar", "bolsa.inscripcion.convocatoria.consultar",
	"bolsa.inscripcion.propias.listar", "bolsa.inscripcion.propia.consultar", "bolsa.inscripcion.presentar",
}

func accionesDeltaBolsaExterna(rol VersionRol, selecciones int) []string {
	if (selecciones != len(accionesVersionarRolBolsaExterna) && selecciones != len(accionesInscripcionBolsaExterna)) ||
		rol.RolID != RolIDVersionarBolsaExterna ||
		rol.Nombre != "areaPersonal.miBolsa.rolPortal" || rol.Version < 1 || rol.Version > 2 {
		return nil
	}
	base := rol
	if len(base.Concesiones) < len(accionesBaseVersionarRolBolsaExterna) {
		return nil
	}
	base.Version = 1
	base.Concesiones = rol.Concesiones[:len(accionesBaseVersionarRolBolsaExterna)]
	if !baseVersionarRolBolsaExterna(base) {
		return nil
	}
	if selecciones == len(accionesVersionarRolBolsaExterna) {
		if rol.Version == 1 && len(rol.Concesiones) == len(accionesBaseVersionarRolBolsaExterna) {
			return accionesVersionarRolBolsaExterna[:]
		}
		if rol.Version != 2 || len(rol.Concesiones) !=
			len(accionesBaseVersionarRolBolsaExterna)+len(accionesInscripcionBolsaExterna) {
			return nil
		}
		for j, accion := range accionesInscripcionBolsaExterna {
			if !concesionDeltaBolsaExterna(rol.Concesiones[len(accionesBaseVersionarRolBolsaExterna)+j], accion) {
				return nil
			}
		}
		return accionesVersionarRolBolsaExterna[:]
	}
	if rol.Version == 1 {
		if len(rol.Concesiones) != len(accionesBaseVersionarRolBolsaExterna) {
			return nil
		}
	} else {
		if len(rol.Concesiones) != len(accionesBaseVersionarRolBolsaExterna)+len(accionesVersionarRolBolsaExterna) {
			return nil
		}
		for j, accion := range accionesVersionarRolBolsaExterna {
			if !concesionVersionarRolBolsaExterna(rol.Concesiones[len(accionesBaseVersionarRolBolsaExterna)+j], accion) {
				return nil
			}
		}
	}
	for _, c := range rol.Concesiones {
		if slices.Contains(accionesInscripcionBolsaExterna[:], c.Accion) {
			return nil
		}
	}
	return accionesInscripcionBolsaExterna[:]
}

var accionesBaseVersionarRolBolsaExterna = [...]string{
	"bolsa.historial_propio.consultar",
	"bolsa.participaciones_propias.consultar",
	"bolsa.participaciones_propias.solicitar_pausa",
	"bolsa.participaciones_propias.solicitar_reactivacion",
	"bolsa.participaciones_propias.responder_llamamiento",
	"bolsa.participaciones_propias.manifestar_disposicion",
	"bolsa.participaciones_propias.confirmar_contacto",
}

func baseVersionarRolBolsaExterna(rol VersionRol) bool {
	if rol.RolID != RolIDVersionarBolsaExterna || rol.Version != 1 ||
		rol.Nombre != "areaPersonal.miBolsa.rolPortal" || len(rol.Concesiones) != len(accionesBaseVersionarRolBolsaExterna) {
		return false
	}
	vistos := make(map[string]bool, len(rol.Concesiones))
	for _, c := range rol.Concesiones {
		if c.Validar() != nil || c.ModuloID != "bolsa" || c.GarantiaMinima != AuthAssuranceHigh ||
			!slices.Contains(accionesBaseVersionarRolBolsaExterna[:], c.Accion) || vistos[c.Accion] ||
			len(c.Obligaciones) != 0 {
			return false
		}
		recurso, finalidad := "participaciones_candidato", "gestion_participaciones_propias"
		var campos []string
		switch c.Accion {
		case "bolsa.historial_propio.consultar":
			finalidad = "consulta_historial_propio"
			campos = []string{"contratos_propios", "llamamientos_propios", "renuncias_propias"}
		case "bolsa.participaciones_propias.consultar":
			finalidad = "consulta_participaciones_propias"
			campos = []string{"participaciones_candidato_minimizadas"}
		case "bolsa.participaciones_propias.manifestar_disposicion":
			recurso = "oferta_bolsa"
		}
		if c.TipoRecurso != recurso || !slices.Equal(c.Finalidades, []string{finalidad}) ||
			!slices.Equal(c.CamposPermitidos, campos) {
			return false
		}
		vistos[c.Accion] = true
	}
	return true
}

// IntencionVersionarRolBolsaExterna sólo identifica la cabeza, el catálogo y
// preimágenes esperadas. No acredita publicación, identidad ni vigencia SQL.
type IntencionVersionarRolBolsaExterna struct {
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

// PlanVersionarRolBolsaExterna conserva la cabeza v1 y las preimágenes CAS
// de asignaciones externas. El SQL coteja la fuente antes de publicar v2.
type PlanVersionarRolBolsaExterna struct {
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

func (p PlanVersionarRolBolsaExterna) HuellaSHA256() (string, error) {
	acciones := accionesDeltaBolsaExterna(p.Base.Rol, len(p.Selecciones))
	if p.Operacion != OperacionVersionarPerfilGobernado ||
		!textoAutorizacionSinComodinSeguro(p.CatalogoRef, 512, false) || p.CatalogoVersion < 1 ||
		!huellaSHA256AutorizacionV3NoNula(p.CatalogoHuellaSHA256) ||
		p.Base.Validar() != nil || p.Base.TipoPerfil != TipoPerfilAdministracionAdministrableV1 ||
		len(acciones) == 0 || p.Base.Rol.Estado != EstadoVersionRolPublicada ||
		p.Base.ControlVigencia.Estado != EstadoControlVigenciaVersionRolHabilitada ||
		p.DefinicionNueva.RolID != p.Base.Rol.RolID || p.DefinicionNueva.Version != p.Base.Rol.Version+1 ||
		p.DefinicionNueva.Nombre != p.Base.Rol.Nombre || p.VersionRolObjetivoRef != p.DefinicionNueva.Referencia() ||
		len(p.DefinicionNueva.Concesiones) != len(p.Base.Rol.Concesiones)+len(acciones) ||
		len(p.DefinicionNueva.Concesiones) > maximoElementosAutorizacion ||
		len(p.Selecciones) != len(acciones) ||
		len(p.Asignaciones) < 1 || len(p.Asignaciones) > 16 ||
		!ReferenciaMotivoAutorizacionV2Valida(p.Motivo) || !ReferenciaActoAdministracionValida(p.ReferenciaActo) {
		return "", ErrVersionarRolBolsaExternaInvalido
	}
	for i, anterior := range p.Base.Rol.Concesiones {
		if !reflect.DeepEqual(anterior, p.DefinicionNueva.Concesiones[i]) ||
			slices.Contains(acciones, anterior.Accion) {
			return "", ErrVersionarRolBolsaExternaInvalido
		}
	}
	for i, accion := range acciones {
		s := p.Selecciones[i]
		if !textoAutorizacionSinComodinSeguro(s.EntradaRef, 512, false) || s.EntradaVersion < 1 ||
			!huellaSHA256AutorizacionV3NoNula(s.EntradaHuellaSHA256) ||
			!concesionDeltaBolsaExterna(p.DefinicionNueva.Concesiones[len(p.Base.Rol.Concesiones)+i], accion) {
			return "", ErrVersionarRolBolsaExternaInvalido
		}
	}
	vistas := make(map[string]bool, len(p.Asignaciones))
	for _, a := range p.Asignaciones {
		if a.ValidarPara(p.Base.Rol.RolID, p.Base.Rol.Version) != nil ||
			!ambitosCandidatoBolsaExterna(a.Documento.Ambitos) || vistas[a.Documento.AsignacionID] {
			return "", ErrVersionarRolBolsaExternaInvalido
		}
		vistas[a.Documento.AsignacionID] = true
	}
	return huellaAutorizacion(p)
}

func accionVersionarRolBolsaExterna(accion string) bool {
	return slices.Contains(accionesVersionarRolBolsaExterna[:], accion)
}

func ambitosCandidatoBolsaExterna(ambitos []AmbitoPerfil) bool {
	return len(ambitos) == 1 && ambitos[0].Clave == "candidato_ref" &&
		len(ambitos[0].Valores) == 1 && ambitos[0].Validar() == nil
}

func dimensionesCandidatoBolsaExterna(dimensiones []string) bool {
	return len(dimensiones) == 1 && dimensiones[0] == "candidato_ref"
}

func camposPreferenciasBolsaExterna(accion string) []string {
	if accion == accionesVersionarRolBolsaExterna[0] {
		return []string{"version", "orden_guardado", "bolsas_actuales", "bolsas_nuevas",
			"bolsas_retiradas", "revision_requerida", "recibo_ref", "registrado_en"}
	}
	return nil
}

func concesionVersionarRolBolsaExterna(c ConcesionRol, accion string) bool {
	if c.Validar() != nil || c.ModuloID != "bolsa" || c.Accion != accion ||
		c.TipoRecurso != "participaciones_candidato" || c.GarantiaMinima != AuthAssuranceHigh ||
		!slices.Equal(c.Finalidades, []string{"gestion_preferencias_bolsas_propias"}) ||
		!slices.Equal(c.CamposPermitidos, camposPreferenciasBolsaExterna(accion)) {
		return false
	}
	switch accion {
	case accionesVersionarRolBolsaExterna[0]:
		return len(c.Obligaciones) == 0 || slices.Equal(c.Obligaciones, []string{"auditar"})
	case accionesVersionarRolBolsaExterna[1]:
		return len(c.Obligaciones) == 0
	default:
		return false
	}
}

func camposInscripcionBolsaExterna(accion string) []string {
	requisito := []string{"codigo", "descripcion", "estado", "hito_cumplimiento", "hito_etiqueta",
		"hito_fecha", "motivo_codigo", "motivo_etiqueta", "obligatorio", "procedencia_ref"}
	convocatoria := []string{"catalogo_version", "categorias[].categoria", "categorias[].categoria_ref",
		"convocatoria_ref", "estado_solicitud_propia", "impedimento_etiqueta", "numero_categorias",
		"plazo_fin", "plazo_inicio", "puede_iniciar", "requisitos_resumen", "solicitud_ref", "titulo"}
	for _, hoja := range requisito {
		convocatoria = append(convocatoria, "requisitos[]."+hoja)
	}
	solicitud := []string{"bases_ref", "bolsa_ref", "catalogo_version", "categoria", "categoria_ref",
		"convocatoria_ref", "convocatoria_titulo", "decidida_en", "decision_ref", "declaracion_ref",
		"estado", "motivo_codigo", "motivo_etiqueta", "participacion_ref", "plazo_fin", "plazo_inicio",
		"recibo_ref", "registrada_en", "solicitud_ref", "version"}
	for _, hoja := range requisito {
		solicitud = append(solicitud, "requisitos[]."+hoja)
	}
	var campos []string
	switch accion {
	case accionesInscripcionBolsaExterna[0]:
		campos = []string{"cursor_siguiente", "total"}
		for _, hoja := range convocatoria {
			campos = append(campos, "convocatorias[]."+hoja)
		}
	case accionesInscripcionBolsaExterna[1]:
		campos = convocatoria
	case accionesInscripcionBolsaExterna[2]:
		campos = []string{"cursor_siguiente", "total"}
		for _, hoja := range solicitud {
			campos = append(campos, "solicitudes[]."+hoja)
		}
	case accionesInscripcionBolsaExterna[3]:
		campos = solicitud
	case accionesInscripcionBolsaExterna[4]:
		return nil
	default:
		return nil
	}
	slices.Sort(campos)
	return campos
}

func concesionDeltaBolsaExterna(c ConcesionRol, accion string) bool {
	if !slices.Contains(accionesInscripcionBolsaExterna[:], accion) {
		return concesionVersionarRolBolsaExterna(c, accion)
	}
	if c.Validar() != nil || c.ModuloID != "bolsa" || c.Accion != accion ||
		c.GarantiaMinima != AuthAssuranceHigh || len(c.Obligaciones) != 0 ||
		!slices.Equal(c.CamposPermitidos, camposInscripcionBolsaExterna(accion)) {
		return false
	}
	switch accion {
	case accionesInscripcionBolsaExterna[0], accionesInscripcionBolsaExterna[1]:
		return c.TipoRecurso == "convocatoria_inscripcion" && slices.Equal(c.Finalidades, []string{"consulta_convocatoria_abierta"})
	case accionesInscripcionBolsaExterna[2], accionesInscripcionBolsaExterna[3]:
		return c.TipoRecurso == "solicitud_inscripcion" && slices.Equal(c.Finalidades, []string{"consulta_inscripcion_propia"})
	case accionesInscripcionBolsaExterna[4]:
		return c.TipoRecurso == "inscripcion_convocatoria" && slices.Equal(c.Finalidades, []string{"presentar_inscripcion"})
	}
	return false
}

func PrepararPlanVersionarRolBolsaExterna(c CatalogoAccionesAdministracionV1,
	base PerfilPublicadoAdministracionV1, i IntencionVersionarRolBolsaExterna, ahora time.Time) (PlanVersionarRolBolsaExterna, error) {
	return prepararPlanVersionarRolBolsaExterna(c, base, i, ahora, true)
}

// ReconstruirPlanVersionarRolBolsaExterna coteja el catálogo histórico exacto
// para recuperar una propuesta. AUT77 exige vigencia al crearla por primera vez
// y vuelve a comprobar la autorización actual incluso en un replay.
func ReconstruirPlanVersionarRolBolsaExterna(c CatalogoAccionesAdministracionV1,
	base PerfilPublicadoAdministracionV1, i IntencionVersionarRolBolsaExterna, ahora time.Time) (PlanVersionarRolBolsaExterna, error) {
	return prepararPlanVersionarRolBolsaExterna(c, base, i, ahora, false)
}

func prepararPlanVersionarRolBolsaExterna(c CatalogoAccionesAdministracionV1,
	base PerfilPublicadoAdministracionV1, i IntencionVersionarRolBolsaExterna, ahora time.Time,
	exigirVigencia bool) (PlanVersionarRolBolsaExterna, error) {
	var vacio PlanVersionarRolBolsaExterna
	hc, err := c.HuellaSHA256()
	if err != nil || !instanteAutorizacionCanonico(ahora) ||
		(exigirVigencia && !vigenteAccionesAdministracionEn(c.VigenteDesde, c.VigenteHasta, ahora)) ||
		c.Referencia != i.CatalogoRef || c.Version != i.CatalogoVersion || hc != i.CatalogoHuellaSHA256 ||
		(len(i.Selecciones) != len(accionesVersionarRolBolsaExterna) &&
			len(i.Selecciones) != len(accionesInscripcionBolsaExterna)) ||
		len(i.Asignaciones) < 1 || len(i.Asignaciones) > 16 {
		return vacio, ErrVersionarRolBolsaExternaInvalido
	}
	if base.Validar() != nil || base.TipoPerfil != TipoPerfilAdministracionAdministrableV1 ||
		base.Rol.Estado != EstadoVersionRolPublicada || base.ControlVigencia.Estado != EstadoControlVigenciaVersionRolHabilitada ||
		base.ControlVigencia.ActualizadoEn.After(ahora) ||
		len(accionesDeltaBolsaExterna(base.Rol, len(i.Selecciones))) == 0 {
		return vacio, ErrVersionarRolBolsaExternaInvalido
	}
	hr, er := base.Rol.HuellaSHA256()
	hcontrol, ec := base.ControlVigencia.HuellaSHA256()
	if er != nil || ec != nil || i.BaseRef != base.Rol.Referencia() || i.BaseHuellaSHA256 != hr ||
		i.ControlRevision != base.ControlVigencia.Revision || i.ControlHuellaSHA256 != hcontrol ||
		len(base.Rol.Concesiones)+len(i.Selecciones) > maximoElementosAutorizacion {
		return vacio, ErrVersionarRolBolsaExternaInvalido
	}
	acciones := accionesDeltaBolsaExterna(base.Rol, len(i.Selecciones))
	for _, anterior := range base.Rol.Concesiones {
		if slices.Contains(acciones, anterior.Accion) {
			return vacio, ErrVersionarRolBolsaExternaInvalido
		}
	}
	concesiones := copiarConcesionesVersionarBolsa(base.Rol.Concesiones)
	for j, accion := range acciones {
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
			(exigirVigencia && !vigenteAccionesAdministracionEn(entrada.VigenteDesde, entrada.VigenteHasta, ahora)) ||
			!dimensionesCandidatoBolsaExterna(entrada.DimensionesAmbito) ||
			!concesionDeltaBolsaExterna(entrada.Concesion, accion) {
			return vacio, ErrVersionarRolBolsaExternaInvalido
		}
		concesiones = append(concesiones, copiarConcesionesVersionarBolsa([]ConcesionRol{entrada.Concesion})[0])
	}
	asignaciones := make([]SeleccionAsignacionVersionarRolBolsa, len(i.Asignaciones))
	for j, a := range i.Asignaciones {
		if a.ValidarPara(base.Rol.RolID, base.Rol.Version) != nil || !ambitosCandidatoBolsaExterna(a.Documento.Ambitos) {
			return vacio, ErrVersionarRolBolsaExternaInvalido
		}
		asignaciones[j] = SeleccionAsignacionVersionarRolBolsa{AsignacionRef: a.AsignacionRef,
			HuellaSHA256: a.HuellaSHA256, Documento: copiarAsignacionVersionarBolsa(a.Documento)}
	}
	baseCopia := base
	baseCopia.Rol.Concesiones = copiarConcesionesVersionarBolsa(base.Rol.Concesiones)
	plan := PlanVersionarRolBolsaExterna{
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

func (p PlanVersionarRolBolsaExterna) Copia() PlanVersionarRolBolsaExterna {
	p.Base.Rol.Concesiones = copiarConcesionesVersionarBolsa(p.Base.Rol.Concesiones)
	p.DefinicionNueva.Concesiones = copiarConcesionesVersionarBolsa(p.DefinicionNueva.Concesiones)
	p.Selecciones = append([]SeleccionAccionAdministracionV1(nil), p.Selecciones...)
	p.Asignaciones = append([]SeleccionAsignacionVersionarRolBolsa(nil), p.Asignaciones...)
	for j := range p.Asignaciones {
		p.Asignaciones[j].Documento = copiarAsignacionVersionarBolsa(p.Asignaciones[j].Documento)
	}
	return p
}
