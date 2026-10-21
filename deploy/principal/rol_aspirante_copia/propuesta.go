package main

import (
	"encoding/json"
	"errors"
	"time"

	dominio "vec-diputacion-granada/internal/vec/domain"
)

// Orden de las selecciones de cada acto. No se mezclan en un mismo plan.
var accionesModo = map[string][]string{
	"inscripcion": {
		"bolsa.inscripcion.convocatorias.listar", "bolsa.inscripcion.convocatoria.consultar",
		"bolsa.inscripcion.propias.listar", "bolsa.inscripcion.propia.consultar", "bolsa.inscripcion.presentar",
	},
	"preferencias": {
		"bolsa.preferencias_bolsas.consultar_propias", "bolsa.preferencias_bolsas.ordenar_propias",
	},
}

type asignacionActual struct {
	AsignacionRef string          `json:"asignacion_ref"`
	HuellaSHA256  string          `json:"huella_sha256"`
	Documento     json.RawMessage `json:"documento"`
}

type entradaPropuesta struct {
	Modo            string                            `json:"modo"`
	OperacionRef    string                            `json:"operacion_ref"`
	CatalogoCanon   string                            `json:"catalogo_canon"`
	Rol             json.RawMessage                   `json:"rol"`
	RolHuellaSHA256 string                            `json:"rol_huella_sha256"`
	Control         json.RawMessage                   `json:"control"`
	ControlHuella   string                            `json:"control_huella_sha256"`
	Asignaciones    []asignacionActual                `json:"asignaciones"`
	Motivo          dominio.ReferenciaEntradaCatalogo `json:"motivo"`
	Ahora           time.Time                         `json:"ahora"`
}

// cuerpoPropuesta es el DTO de
// POST /api/admin/perfiles/v1/gobierno-version-bolsa-externa/propuestas.
type cuerpoPropuesta struct {
	OperacionRef         string                                         `json:"operacion_ref"`
	CatalogoRef          string                                         `json:"catalogo_ref"`
	CatalogoVersion      int                                            `json:"catalogo_version"`
	CatalogoHuellaSHA256 string                                         `json:"catalogo_huella_sha256"`
	BaseRef              string                                         `json:"base_ref"`
	BaseHuellaSHA256     string                                         `json:"base_huella_sha256"`
	ControlRevision      uint64                                         `json:"control_revision"`
	ControlHuellaSHA256  string                                         `json:"control_huella_sha256"`
	Selecciones          []dominio.SeleccionAccionAdministracionV1      `json:"selecciones"`
	Asignaciones         []dominio.SeleccionAsignacionVersionarRolBolsa `json:"asignaciones"`
	Motivo               dominio.ReferenciaEntradaCatalogo              `json:"motivo"`
	HuellaPlanEsperada   string                                         `json:"huella_plan_esperada"`
}

type salidaPropuesta struct {
	Cuerpo             cuerpoPropuesta `json:"cuerpo"`
	VersionRolObjetivo string          `json:"version_rol_objetivo_ref"`
	Concesiones        int             `json:"concesiones"`
}

func prepararPropuesta(e entradaPropuesta) (salidaPropuesta, error) {
	var vacia salidaPropuesta
	acciones, ok := accionesModo[e.Modo]
	if !ok || !dominio.ReferenciaAdministracionPerfilesValida(e.OperacionRef, "propuesta_admin:") {
		return vacia, errors.New("modo u operación fuera de contrato")
	}
	var c dominio.CatalogoAccionesAdministracionV1
	if err := decodificarExacto([]byte(e.CatalogoCanon), &c); err != nil {
		return vacia, err
	}
	hc, err := c.HuellaSHA256()
	if err != nil {
		return vacia, errors.New("catálogo publicado inválido")
	}
	base := dominio.PerfilPublicadoAdministracionV1{}
	if json.Unmarshal(e.Rol, &base.Rol) != nil || json.Unmarshal(e.Control, &base.ControlVigencia) != nil ||
		!mismoJSON(base.Rol, e.Rol) || !mismoJSON(base.ControlVigencia, e.Control) {
		return vacia, errors.New("la cabeza del rol no se representa sin pérdida")
	}
	// El tipo lo publica el catálogo para ese rol; nunca se supone.
	for _, p := range c.Perfiles {
		if p.Rol.RolID == base.Rol.RolID {
			base.TipoPerfil = p.TipoPerfil
		}
	}
	hr, er := base.Rol.HuellaSHA256()
	hctl, ec := base.ControlVigencia.HuellaSHA256()
	if er != nil || ec != nil || hr != e.RolHuellaSHA256 || hctl != e.ControlHuella {
		return vacia, errors.New("las huellas de la cabeza del rol no coinciden con la base")
	}
	intencion := dominio.IntencionVersionarRolBolsaExterna{
		CatalogoRef: c.Referencia, CatalogoVersion: c.Version, CatalogoHuellaSHA256: hc,
		BaseRef: base.Rol.Referencia(), BaseHuellaSHA256: hr,
		ControlRevision: base.ControlVigencia.Revision, ControlHuellaSHA256: hctl, Motivo: e.Motivo,
	}
	for _, accion := range acciones {
		var elegida *dominio.EntradaAccionAdministracionV1
		for i := range c.Entradas {
			if c.Entradas[i].Referencia == "accion:"+accion {
				elegida = &c.Entradas[i]
			}
		}
		if elegida == nil {
			return vacia, errors.New("el catálogo publicado no tiene la acción " + accion)
		}
		h, err := elegida.HuellaSHA256()
		if err != nil {
			return vacia, err
		}
		intencion.Selecciones = append(intencion.Selecciones, dominio.SeleccionAccionAdministracionV1{
			EntradaRef: elegida.Referencia, EntradaVersion: elegida.Version, EntradaHuellaSHA256: h})
	}
	for _, a := range e.Asignaciones {
		var doc dominio.AsignacionPerfil
		if json.Unmarshal(a.Documento, &doc) != nil || !mismoJSON(doc, a.Documento) {
			return vacia, errors.New("una asignación no se representa sin pérdida")
		}
		h, err := doc.HuellaSHA256()
		if err != nil || h != a.HuellaSHA256 || doc.Referencia() != a.AsignacionRef {
			return vacia, errors.New("la huella de una asignación no coincide con la base")
		}
		intencion.Asignaciones = append(intencion.Asignaciones, dominio.SeleccionAsignacionVersionarRolBolsa{
			AsignacionRef: a.AsignacionRef, HuellaSHA256: h, Documento: doc})
	}
	plan, err := dominio.PrepararPlanVersionarRolBolsaExterna(c, base, intencion, e.Ahora.UTC().Truncate(time.Microsecond))
	if err != nil {
		return vacia, errors.New("el dominio rechaza el plan: la cabeza del rol no admite este acto")
	}
	huella, err := plan.HuellaSHA256()
	if err != nil {
		return vacia, err
	}
	return salidaPropuesta{
		Cuerpo: cuerpoPropuesta{
			OperacionRef: e.OperacionRef, CatalogoRef: c.Referencia, CatalogoVersion: c.Version,
			CatalogoHuellaSHA256: hc, BaseRef: intencion.BaseRef, BaseHuellaSHA256: hr,
			ControlRevision: intencion.ControlRevision, ControlHuellaSHA256: hctl,
			Selecciones: intencion.Selecciones, Asignaciones: intencion.Asignaciones,
			Motivo: e.Motivo, HuellaPlanEsperada: huella,
		},
		VersionRolObjetivo: plan.VersionRolObjetivoRef,
		Concesiones:        len(plan.DefinicionNueva.Concesiones),
	}, nil
}
