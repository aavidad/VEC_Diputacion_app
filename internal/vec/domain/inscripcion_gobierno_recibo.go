package domain

import (
	"encoding/json"
	"time"
)

type EfectoAsignacionVersionInscripcion struct {
	AnteriorRef        string           `json:"anterior_ref,omitempty"`
	AnteriorSHA256     string           `json:"anterior_sha256,omitempty"`
	PosteriorRef       string           `json:"posterior_ref"`
	PosteriorSHA256    string           `json:"posterior_sha256"`
	PosteriorDocumento AsignacionPerfil `json:"posterior_documento"`
	UnidadAcuse        json.RawMessage  `json:"unidad_acuse,omitempty"`
}

type ReciboVersionInscripcion struct {
	ActoRef             string                               `json:"acto_ref"`
	ReciboRef           string                               `json:"recibo_ref"`
	ActorPersonaRef     string                               `json:"actor_persona_ref"`
	PerfilActivoRef     string                               `json:"perfil_activo_ref"`
	AsignacionPerfilRef string                               `json:"asignacion_perfil_ref"`
	CorrelacionRef      string                               `json:"correlacion_ref"`
	Motivo              ReferenciaEntradaCatalogo            `json:"motivo"`
	AuditoriaRef        string                               `json:"auditoria_ref"`
	VersionRol          VersionRol                           `json:"version_rol"`
	ControlPosterior    ControlVigenciaVersionRol            `json:"control_posterior"`
	Asignaciones        []EfectoAsignacionVersionInscripcion `json:"asignaciones"`
}

type CierreVersionInscripcion struct {
	OperacionRef          string
	Material              MaterialPropuestaVersionInscripcion
	PropuestaHuellaSHA256 string
	Decision              DecisionPropuestaAdministracionPerfiles
	ConfirmadoEn          time.Time
	AuditoriaAccesoRef    string
	Recibo                *ReciboVersionInscripcion
}

func (c CierreVersionInscripcion) ValidarPara(s SolicitudCierreVersionInscripcion) error {
	h, err := c.Material.HuellaSHA256()
	if s.Validar() != nil || err != nil || c.OperacionRef != s.OperacionRef ||
		c.Material.OperacionRef != s.PropuestaRef || c.PropuestaHuellaSHA256 != s.PropuestaHuellaSHA256 ||
		h != s.PropuestaHuellaSHA256 || c.Material.ProponentePersonaRef == s.Aprobador.PersonaRef ||
		c.Decision != DecisionAprobarPropuestaPerfil || !instanteAutorizacionCanonico(c.ConfirmadoEn) ||
		!textoAutorizacionSinComodinSeguro(c.AuditoriaAccesoRef, 256, false) || c.Recibo == nil {
		return ErrPlanVersionInscripcionInvalido
	}
	r := c.Recibo
	p := c.Material.Plan
	if !ReferenciaAdministracionPerfilesValida(r.ActoRef, "acto_admin:") ||
		!ReferenciaAdministracionPerfilesValida(r.ReciboRef, "recibo_admin:") ||
		r.ActorPersonaRef != s.Aprobador.PersonaRef || r.PerfilActivoRef != s.Aprobador.PerfilActivoRef ||
		r.AsignacionPerfilRef != s.InstantaneaAutorizacion.AsignacionPerfil.Referencia() ||
		r.CorrelacionRef != s.CorrelacionRef || r.Motivo != s.Motivo ||
		!textoAutorizacionSinComodinSeguro(r.AuditoriaRef, 256, false) ||
		r.VersionRol.Validar() != nil || r.VersionRol.Referencia() != p.VersionRolObjetivoRef ||
		r.VersionRol.RolID != p.DefinicionNueva.RolID ||
		r.VersionRol.Version != p.DefinicionNueva.Version ||
		r.VersionRol.Nombre != p.DefinicionNueva.Nombre ||
		r.VersionRol.Estado != EstadoVersionRolPublicada ||
		r.VersionRol.PublicadaPor != s.Aprobador.PersonaRef ||
		!r.VersionRol.PublicadaEn.Equal(c.ConfirmadoEn) ||
		r.ControlPosterior.Validar() != nil ||
		r.ControlPosterior.VersionRolRef != p.VersionRolObjetivoRef ||
		r.ControlPosterior.Revision != 1 ||
		r.ControlPosterior.Estado != EstadoControlVigenciaVersionRolHabilitada ||
		r.ControlPosterior.ActualizadoPor != s.Aprobador.PersonaRef ||
		!r.ControlPosterior.ActualizadoEn.Equal(c.ConfirmadoEn) ||
		len(r.VersionRol.Concesiones) != len(p.DefinicionNueva.Concesiones) ||
		len(r.Asignaciones) != len(p.Asignaciones) {
		return ErrPlanVersionInscripcionInvalido
	}
	for i, concesion := range p.DefinicionNueva.Concesiones {
		if !concesionesPerfilAdministracionIguales(concesion, r.VersionRol.Concesiones[i]) {
			return ErrPlanVersionInscripcionInvalido
		}
	}
	efectos := make(map[string]EfectoAsignacionVersionInscripcion, len(r.Asignaciones))
	for _, efecto := range r.Asignaciones {
		doc := efecto.PosteriorDocumento
		hd, e := doc.HuellaSHA256()
		if e != nil || doc.Estado != EstadoAsignacionPerfilActiva ||
			efecto.PosteriorRef != doc.Referencia() || efecto.PosteriorSHA256 != hd ||
			doc.VersionRolRef != p.VersionRolObjetivoRef || doc.EmitidaPor != s.Aprobador.PersonaRef ||
			!doc.EmitidaEn.Equal(c.ConfirmadoEn) {
			return ErrPlanVersionInscripcionInvalido
		}
		if _, duplicado := efectos[doc.AsignacionID]; duplicado {
			return ErrPlanVersionInscripcionInvalido
		}
		efectos[doc.AsignacionID] = efecto
	}
	for _, cambio := range p.Asignaciones {
		efecto, ok := efectos[cambio.AsignacionID]
		if !ok {
			return ErrPlanVersionInscripcionInvalido
		}
		doc := efecto.PosteriorDocumento
		if doc.PrincipalID != cambio.PrincipalID || doc.PerfilActivoRef != cambio.PerfilActivoRef ||
			!ambitosInscripcionIguales(doc.Ambitos, cambio.Ambitos) ||
			!doc.VigenteDesde.Equal(cambio.VigenteDesde) || !doc.VigenteHasta.Equal(cambio.VigenteHasta) {
			return ErrPlanVersionInscripcionInvalido
		}
		if cambio.Modo == "avance" {
			if cambio.Anterior == nil || efecto.AnteriorRef != cambio.Anterior.AsignacionRef ||
				efecto.AnteriorSHA256 != cambio.Anterior.HuellaSHA256 ||
				doc.Version != cambio.Anterior.Documento.Version+1 ||
				!acuseUnidadInscripcionValido(efecto.UnidadAcuse, cambio, s.InstantaneaAutorizacion.AsignacionPerfil) {
				return ErrPlanVersionInscripcionInvalido
			}
		} else if cambio.Modo != "alta" || efecto.AnteriorRef != "" || efecto.AnteriorSHA256 != "" ||
			doc.Version != 1 || len(efecto.UnidadAcuse) != 0 {
			return ErrPlanVersionInscripcionInvalido
		}
	}
	return nil
}

func acuseUnidadInscripcionValido(raw json.RawMessage, cambio CambioAsignacionInscripcion, admin AsignacionPerfil) bool {
	if len(raw) == 0 || len(raw) > 16384 || cambio.FuenteUnidad == nil {
		return false
	}
	var acuse struct {
		Esquema         string                  `json:"esquema"`
		OrganizacionRef string                  `json:"organizacion_ref"`
		UnidadRef       string                  `json:"unidad_ref"`
		Fuente          FuenteUnidadInscripcion `json:"fuente"`
		ValidaHasta     time.Time               `json:"valida_hasta"`
	}
	if json.Unmarshal(raw, &acuse) != nil || acuse.Esquema != "vec.personal.unidad-bootstrap-admin.v1" ||
		acuse.Fuente != *cambio.FuenteUnidad || acuse.ValidaHasta.Before(cambio.VigenteHasta) {
		return false
	}
	unidad := ""
	for _, ambito := range cambio.Ambitos {
		if ambito.Clave == "unidad_ref" && len(ambito.Valores) == 1 {
			unidad = ambito.Valores[0]
		}
	}
	organizacion := ""
	for _, ambito := range admin.Ambitos {
		if ambito.Clave == "organizacion_ref" && len(ambito.Valores) == 1 {
			organizacion = ambito.Valores[0]
		}
	}
	return unidad != "" && organizacion != "" && acuse.UnidadRef == unidad && acuse.OrganizacionRef == organizacion
}
