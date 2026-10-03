package bootstrap

import (
	"context"
	"net/http"
	"slices"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/consultafirmas"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const fronteraLecturaFirmasIntervencionCT = "ct-firmas-lectura-interna-intervencion"

// El lector conserva el canal de Intervención. Su perfil y sesión propios son
// una dependencia interna escogida por composición, nunca por la petición.
func nuevaSesionConsultaFirmasIntervencionCTDesarrollo(
	ctx context.Context, base *proveedorSesionConsultaRRHHDesarrollo,
	canal *soporteFiscalizacionContratacionTemporalDesarrollo,
	puente *soporteAltaContratacionTemporalDesarrollo, perfil *perfilFijoCTDesarrollo,
) (*proveedorSesionConsultaRRHHDesarrollo, error) {
	if ctx == nil || base == nil || canal == nil || puente == nil || perfil == nil ||
		canal.contexto.Resultado.Validar() != nil || perfil.contexto.Resultado.Validar() != nil ||
		perfil.plantilla.Validar() != nil || len(perfil.plantilla.VersionRol.Concesiones) != 1 || len(perfil.plantilla.AsignacionPerfil.Ambitos) != 1 ||
		puente.sello != canal.sello || puente.principalID != canal.principalID || puente.certificadoSHA256 != canal.certificadoSHA256 {
		return nil, ports.ErrAutorizacionDenegada
	}
	concesion, ambito := perfil.plantilla.VersionRol.Concesiones[0], perfil.plantilla.AsignacionPerfil.Ambitos[0]
	if concesion.Accion != ports.AccionConsultarFirmasDocumento || concesion.ModuloID != ports.ModuloContratacion ||
		concesion.TipoRecurso != ports.TipoRecursoConsultaFirmasDocumento ||
		!slices.Equal(concesion.Finalidades, []string{ports.FinalidadFirmaDocumento}) || concesion.GarantiaMinima != dominiovec.AuthAssuranceHigh ||
		!slices.Equal(concesion.CamposPermitidos, consultafirmas.CamposConsultaFirmasDocumento()) || len(concesion.Obligaciones) != 0 ||
		ambito.Clave != "organizacion_ref" || !slices.Equal(ambito.Valores, []string{organizacionAltaContratacionTemporalDesarrollo}) {
		return nil, ports.ErrAutorizacionDenegada
	}
	lector, origen := perfil.contexto.Resultado.Contexto, canal.contexto.Resultado.Contexto
	if lector.Principal.ID != origen.Principal.ID || lector.PersonaRef != origen.PersonaRef ||
		lector.Instantanea.CuentaRef != origen.Instantanea.CuentaRef || perfil.perfilRef() != lector.PerfilActivoRef ||
		lector.PerfilActivoRef == origen.PerfilActivoRef || perfil.contexto.Vinculo.ValidarPara(perfil.contexto.Resultado) != nil {
		return nil, ports.ErrAutorizacionDenegada
	}
	vLectura, errLectura := perfil.contexto.Vinculo.Datos()
	vOrigen, errOrigen := canal.contexto.Vinculo.Datos()
	if errLectura != nil || errOrigen != nil || vLectura.SesionRef == vOrigen.SesionRef || vLectura.AutenticacionRef == vOrigen.AutenticacionRef {
		return nil, ports.ErrAutorizacionDenegada
	}
	esperado, err := contextoEsperadoRegistradoParaSemillaDesarrollo(ctx, base.resolutor, puente, perfil.contexto.Resultado)
	if err != nil {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	puente.mu.Lock()
	puente.contextoEsperadoRegistrado = esperado
	puente.mu.Unlock()
	frontera := fronteraContratacionTemporalDesarrollo(fronteraLecturaFirmasIntervencionCT,
		ports.AccionConsultarFirmasDocumento, httpinterno.RutaResultadosFiscalizacion, []string{perfil.perfilRef()})
	fronteras, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{frontera})
	if err != nil {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	p, err := nuevoProveedorSesionConsultaRRHHConCatalogoDesarrollo(puente, base.registro, base.revalidador, base.reloj, base.resolutor, fronteras)
	if err != nil {
		return nil, err
	}
	p.canalIntervencion = canal
	return p, nil
}

// El puente seleccionado falla cerrado. No se intenta el canal RRHH si falta
// el permiso, cambia la identidad o se altera la captura sellada del canal.
func (p *proveedorSesionConsultaRRHHDesarrollo) capacidadCanalSesion(ctx context.Context) (capacidadConsultaContratacionTemporalDesarrollo, bool) {
	if p == nil || p.soporte == nil {
		return capacidadConsultaContratacionTemporalDesarrollo{}, false
	}
	if p.canalIntervencion == nil {
		return p.soporte.capacidadValida(ctx)
	}
	canal := p.canalIntervencion
	if ctx == nil || ctx.Err() != nil || !canal.capacidadValida(ctx) {
		return capacidadConsultaContratacionTemporalDesarrollo{}, false
	}
	captura, ok := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	lector, origen := p.base.Contexto, canal.contexto.Resultado.Contexto
	d, declarada := p.fronteras.resolver(http.MethodPost, httpinterno.RutaResultadosFiscalizacion)
	valida := ok && captura.metodo == http.MethodPost && captura.ruta == httpinterno.RutaResultadosFiscalizacion &&
		p.soporte.sello == canal.sello && p.soporte.principalID == canal.principalID && p.soporte.certificadoSHA256 == canal.certificadoSHA256 &&
		lector.Principal.ID == origen.Principal.ID && lector.PersonaRef == origen.PersonaRef &&
		lector.Instantanea.CuentaRef == origen.Instantanea.CuentaRef && lector.PerfilActivoRef != origen.PerfilActivoRef &&
		declarada && d.Clave == fronteraLecturaFirmasIntervencionCT && d.Metodo == http.MethodPost &&
		d.Ruta == httpinterno.RutaResultadosFiscalizacion && d.ClavePolitica == clavePoliticaContratacionTemporalDesarrollo &&
		d.ClaveCapacidad == ports.AccionConsultarFirmasDocumento && d.Superficie == superficieInternaSeguridadComunDesarrollo &&
		len(d.PerfilesActivosRef) == 1 && d.PerfilesActivosRef[0] == lector.PerfilActivoRef
	return captura, valida
}

func (p *proveedorSesionConsultaRRHHDesarrollo) rutaSesionConIndisponibilidad(ruta string, contextos ...context.Context) bool {
	if rutaSesionConIndisponibilidadCTDesarrollo(ruta) || p != nil && p.canalIntervencion != nil && ruta == httpinterno.RutaResultadosFiscalizacion {
		return true
	}
	return len(contextos) == 1 && (p.sesionPreparacionBasesHTTPV3(contextos[0], ruta) ||
		p.sesionGobiernoReglasBaremoHTTPV3(contextos[0], ruta))
}
