package application

import (
	"context"
	"sort"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// CapacidadSesion informa a la interfaz; cada operación se autoriza aparte.
type CapacidadSesion struct {
	ModuloID         string                `json:"modulo_id"`
	TipoRecurso      string                `json:"tipo_recurso"`
	Accion           string                `json:"accion"`
	Finalidades      []string              `json:"finalidades"`
	CamposPermitidos []string              `json:"campos_permitidos"`
	Obligaciones     []string              `json:"obligaciones"`
	Ambitos          []domain.AmbitoPerfil `json:"ambitos"`
}

type RevisionesCapacidadesSesion struct {
	Asignacion        int    `json:"asignacion"`
	ControlRol        uint64 `json:"control_rol"`
	CatalogoPoliticas uint64 `json:"catalogo_politicas"`
}

type CapacidadesSesion struct {
	Capacidades     []CapacidadSesion           `json:"capacidades"`
	PerfilActivoRef string                      `json:"perfil_activo_ref"`
	Superficie      string                      `json:"superficie"`
	VersionRolRef   string                      `json:"version_rol_ref"`
	Revisiones      RevisionesCapacidadesSesion `json:"revisiones"`
	VigenteHasta    time.Time                   `json:"vigente_hasta"`
}

type ProyectorCapacidadesSesion struct {
	fuente   ports.FuenteAutorizacion
	selector ports.SelectorSesion
	reloj    ports.Reloj
}

func NuevoProyectorCapacidadesSesion(fuente ports.FuenteAutorizacion, selector ports.SelectorSesion, reloj ports.Reloj) (*ProyectorCapacidadesSesion, error) {
	if dependenciaAutorizacionNula(fuente) || dependenciaAutorizacionNula(selector) || dependenciaAutorizacionNula(reloj) {
		return nil, domain.ErrConfiguracionAccesoInvalida
	}
	return &ProyectorCapacidadesSesion{fuente: fuente, selector: selector, reloj: reloj}, nil
}

func (p *ProyectorCapacidadesSesion) Proyectar(ctx context.Context, principal domain.Principal) (CapacidadesSesion, error) {
	if p == nil || ctx == nil || principal.Validate() != nil {
		return CapacidadesSesion{}, domain.ErrAutorizacionDenegada
	}
	if err := ctx.Err(); err != nil {
		return CapacidadesSesion{}, err
	}
	seleccion, err := p.selector.SeleccionarSesion(ctx, principal)
	if err != nil {
		return CapacidadesSesion{}, err
	}
	ahora := p.reloj.Ahora().UTC()
	if !identificadorSesionSeguro(seleccion.PerfilActivoRef) || !identificadorSesionSeguro(seleccion.Superficie) ||
		!seleccion.VigenteHasta.After(ahora) {
		return CapacidadesSesion{}, domain.ErrAutorizacionDenegada
	}
	instantanea, err := p.fuente.ObtenerInstantaneaAutorizacion(ctx, principal.ID, seleccion.PerfilActivoRef)
	if err != nil {
		return CapacidadesSesion{}, err
	}
	if err := ctx.Err(); err != nil {
		return CapacidadesSesion{}, err
	}
	// La lectura de la autoridad puede cruzar una caducidad. Evaluar siempre
	// la instantanea contra el instante posterior a esa lectura.
	ahora = p.reloj.Ahora().UTC()
	if instantanea.Validar() != nil {
		return CapacidadesSesion{}, ports.ErrFuenteAutorizacionNoDisponible
	}
	asignacion, rol, control := instantanea.AsignacionPerfil, instantanea.VersionRol, instantanea.ControlVigenciaVersionRol
	if asignacion.PrincipalID != principal.ID || asignacion.PerfilActivoRef != seleccion.PerfilActivoRef ||
		!asignacion.VigenteEn(ahora) || rol.Estado != domain.EstadoVersionRolPublicada || rol.PublicadaEn.After(ahora) ||
		control.Estado != domain.EstadoControlVigenciaVersionRolHabilitada || control.ActualizadoEn.After(ahora) {
		return CapacidadesSesion{}, domain.ErrAutorizacionDenegada
	}
	limite := minimoSesion(seleccion.VigenteHasta, asignacion.VigenteHasta)
	for _, politica := range instantanea.Politicas {
		if politica.Estado != domain.EstadoPoliticaRestrictivaPublicada {
			continue
		}
		for _, frontera := range []time.Time{politica.VigenteDesde, politica.VigenteHasta} {
			if frontera.After(ahora) {
				limite = minimoSesion(limite, frontera)
			}
		}
	}
	expuestas := make(map[string]struct{}, len(seleccion.Exposiciones))
	for _, e := range seleccion.Exposiciones {
		if !identificadorSesionSeguro(e.Superficie) || !identificadorSesionSeguro(e.ModuloID) ||
			!identificadorSesionSeguro(e.TipoRecurso) || !identificadorSesionSeguro(e.Accion) {
			return CapacidadesSesion{}, ports.ErrSeleccionSesionNoDisponible
		}
		if e.Superficie == seleccion.Superficie {
			expuestas[claveCapacidadSesion(e.ModuloID, e.TipoRecurso, e.Accion)] = struct{}{}
		}
	}
	resultado := CapacidadesSesion{Capacidades: []CapacidadSesion{}, PerfilActivoRef: seleccion.PerfilActivoRef,
		Superficie: seleccion.Superficie, VersionRolRef: rol.Referencia(),
		Revisiones: RevisionesCapacidadesSesion{Asignacion: asignacion.Version, ControlRol: control.Revision, CatalogoPoliticas: instantanea.RevisionCatalogoPoliticas}}
	for _, c := range rol.Concesiones {
		if _, ok := expuestas[claveCapacidadSesion(c.ModuloID, c.TipoRecurso, c.Accion)]; !ok ||
			!domain.CumpleGarantiaAutenticacion(principal.AuthAssurance, c.GarantiaMinima) {
			continue
		}
		capacidad := CapacidadSesion{ModuloID: c.ModuloID, TipoRecurso: c.TipoRecurso, Accion: c.Accion,
			Finalidades: append([]string{}, c.Finalidades...), CamposPermitidos: append([]string{}, c.CamposPermitidos...),
			Obligaciones: append([]string{}, c.Obligaciones...), Ambitos: copiarAmbitosSesion(asignacion.Ambitos)}
		for _, politica := range instantanea.Politicas {
			if !politicaSesionAplica(politica, c) {
				continue
			}
			if !politica.VigenteEn(ahora) {
				continue
			}
			if politica.Efecto == domain.EfectoPoliticaDenegar || len(politica.Restricciones) > 0 {
				capacidad.Finalidades = nil
				break
			}
			if len(politica.FinalidadesPermitidas) > 0 && !contieneSesion(politica.FinalidadesPermitidas, "*") {
				capacidad.Finalidades = interseccionSesion(capacidad.Finalidades, politica.FinalidadesPermitidas)
			}
			if politica.RestringeCampos && !contieneSesion(politica.CamposPermitidos, "*") {
				capacidad.CamposPermitidos = interseccionSesion(capacidad.CamposPermitidos, politica.CamposPermitidos)
			}
			capacidad.Obligaciones = unionSesion(capacidad.Obligaciones, politica.Obligaciones)
			if politica.GarantiaMinima != "" && !domain.CumpleGarantiaAutenticacion(principal.AuthAssurance, politica.GarantiaMinima) {
				capacidad.Finalidades = nil
				break
			}
		}
		if len(capacidad.Finalidades) == 0 {
			continue
		}
		sort.Strings(capacidad.Finalidades)
		sort.Strings(capacidad.CamposPermitidos)
		sort.Strings(capacidad.Obligaciones)
		resultado.Capacidades = append(resultado.Capacidades, capacidad)
	}
	if err := ctx.Err(); err != nil {
		return CapacidadesSesion{}, err
	}
	if !limite.After(p.reloj.Ahora().UTC()) {
		return CapacidadesSesion{}, domain.ErrAutorizacionDenegada
	}
	sort.Slice(resultado.Capacidades, func(i, j int) bool {
		a, b := resultado.Capacidades[i], resultado.Capacidades[j]
		return claveCapacidadSesion(a.ModuloID, a.TipoRecurso, a.Accion) < claveCapacidadSesion(b.ModuloID, b.TipoRecurso, b.Accion)
	})
	resultado.VigenteHasta = limite.UTC()
	return resultado, nil
}

func minimoSesion(a, b time.Time) time.Time {
	if b.Before(a) {
		return b
	}
	return a
}
func claveCapacidadSesion(modulo, tipo, accion string) string {
	return modulo + "\x00" + tipo + "\x00" + accion
}
func identificadorSesionSeguro(s string) bool {
	return s != "" && len(s) <= 512 && !strings.ContainsAny(s, "*\x00\r\n") && strings.TrimSpace(s) == s
}
func contieneSesion(lista []string, valor string) bool {
	for _, s := range lista {
		if s == valor {
			return true
		}
	}
	return false
}
func politicaSesionAplica(p domain.PoliticaRestrictiva, c domain.ConcesionRol) bool {
	return (contieneSesion(p.Acciones, "*") || contieneSesion(p.Acciones, c.Accion)) &&
		(contieneSesion(p.Modulos, "*") || contieneSesion(p.Modulos, c.ModuloID)) &&
		(contieneSesion(p.TiposRecurso, "*") || contieneSesion(p.TiposRecurso, c.TipoRecurso))
}
func interseccionSesion(base, filtro []string) []string {
	resultado := make([]string, 0, len(base))
	for _, v := range base {
		if contieneSesion(filtro, v) {
			resultado = append(resultado, v)
		}
	}
	return resultado
}
func unionSesion(base, extra []string) []string {
	resultado := append([]string{}, base...)
	for _, v := range extra {
		if !contieneSesion(resultado, v) {
			resultado = append(resultado, v)
		}
	}
	return resultado
}
func copiarAmbitosSesion(ambitos []domain.AmbitoPerfil) []domain.AmbitoPerfil {
	resultado := make([]domain.AmbitoPerfil, len(ambitos))
	for i, a := range ambitos {
		resultado[i] = domain.AmbitoPerfil{Clave: a.Clave, Valores: append([]string(nil), a.Valores...)}
	}
	return resultado
}
