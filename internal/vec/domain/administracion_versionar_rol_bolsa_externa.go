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
	if p.Operacion != OperacionVersionarPerfilGobernado ||
		!textoAutorizacionSinComodinSeguro(p.CatalogoRef, 512, false) || p.CatalogoVersion < 1 ||
		!huellaSHA256AutorizacionV3NoNula(p.CatalogoHuellaSHA256) ||
		p.Base.Validar() != nil || p.Base.TipoPerfil != TipoPerfilAdministracionAdministrableV1 ||
		!baseVersionarRolBolsaExterna(p.Base.Rol) || p.Base.Rol.Estado != EstadoVersionRolPublicada ||
		p.Base.ControlVigencia.Estado != EstadoControlVigenciaVersionRolHabilitada ||
		p.DefinicionNueva.Version != 2 ||
		p.DefinicionNueva.RolID != p.Base.Rol.RolID || p.DefinicionNueva.Version != p.Base.Rol.Version+1 ||
		p.DefinicionNueva.Nombre != p.Base.Rol.Nombre || p.VersionRolObjetivoRef != p.DefinicionNueva.Referencia() ||
		len(p.DefinicionNueva.Concesiones) != len(p.Base.Rol.Concesiones)+len(accionesVersionarRolBolsaExterna) ||
		len(p.DefinicionNueva.Concesiones) > maximoElementosAutorizacion ||
		len(p.Selecciones) != len(accionesVersionarRolBolsaExterna) ||
		len(p.Asignaciones) < 1 || len(p.Asignaciones) > 16 ||
		!ReferenciaMotivoAutorizacionV2Valida(p.Motivo) || !ReferenciaActoAdministracionValida(p.ReferenciaActo) {
		return "", ErrVersionarRolBolsaExternaInvalido
	}
	for i, anterior := range p.Base.Rol.Concesiones {
		if !reflect.DeepEqual(anterior, p.DefinicionNueva.Concesiones[i]) ||
			accionVersionarRolBolsaExterna(anterior.Accion) {
			return "", ErrVersionarRolBolsaExternaInvalido
		}
	}
	for i, accion := range accionesVersionarRolBolsaExterna {
		s := p.Selecciones[i]
		if !textoAutorizacionSinComodinSeguro(s.EntradaRef, 512, false) || s.EntradaVersion < 1 ||
			!huellaSHA256AutorizacionV3NoNula(s.EntradaHuellaSHA256) ||
			!concesionVersionarRolBolsaExterna(p.DefinicionNueva.Concesiones[len(p.Base.Rol.Concesiones)+i], accion) {
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

func PrepararPlanVersionarRolBolsaExterna(c CatalogoAccionesAdministracionV1,
	base PerfilPublicadoAdministracionV1, i IntencionVersionarRolBolsaExterna, ahora time.Time) (PlanVersionarRolBolsaExterna, error) {
	var vacio PlanVersionarRolBolsaExterna
	hc, err := c.HuellaSHA256()
	if err != nil || !instanteAutorizacionCanonico(ahora) ||
		!vigenteAccionesAdministracionEn(c.VigenteDesde, c.VigenteHasta, ahora) ||
		c.Referencia != i.CatalogoRef || c.Version != i.CatalogoVersion || hc != i.CatalogoHuellaSHA256 ||
		len(i.Selecciones) != len(accionesVersionarRolBolsaExterna) ||
		len(i.Asignaciones) < 1 || len(i.Asignaciones) > 16 {
		return vacio, ErrVersionarRolBolsaExternaInvalido
	}
	if base.Validar() != nil || base.TipoPerfil != TipoPerfilAdministracionAdministrableV1 ||
		base.Rol.Estado != EstadoVersionRolPublicada || base.ControlVigencia.Estado != EstadoControlVigenciaVersionRolHabilitada ||
		base.ControlVigencia.ActualizadoEn.After(ahora) || !baseVersionarRolBolsaExterna(base.Rol) {
		return vacio, ErrVersionarRolBolsaExternaInvalido
	}
	hr, er := base.Rol.HuellaSHA256()
	hcontrol, ec := base.ControlVigencia.HuellaSHA256()
	if er != nil || ec != nil || i.BaseRef != base.Rol.Referencia() || i.BaseHuellaSHA256 != hr ||
		i.ControlRevision != base.ControlVigencia.Revision || i.ControlHuellaSHA256 != hcontrol ||
		len(base.Rol.Concesiones)+len(accionesVersionarRolBolsaExterna) > maximoElementosAutorizacion {
		return vacio, ErrVersionarRolBolsaExternaInvalido
	}
	for _, anterior := range base.Rol.Concesiones {
		if accionVersionarRolBolsaExterna(anterior.Accion) {
			return vacio, ErrVersionarRolBolsaExternaInvalido
		}
	}
	concesiones := copiarConcesionesVersionarBolsa(base.Rol.Concesiones)
	for j, accion := range accionesVersionarRolBolsaExterna {
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
			!dimensionesCandidatoBolsaExterna(entrada.DimensionesAmbito) ||
			!concesionVersionarRolBolsaExterna(entrada.Concesion, accion) {
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
