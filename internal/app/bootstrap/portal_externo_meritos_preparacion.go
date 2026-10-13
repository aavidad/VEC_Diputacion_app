package bootstrap

import (
	"net/http"

	"vec-diputacion-granada/internal/modules/meritos/application"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	vec "vec-diputacion-granada/internal/vec/domain"
)

// preparadorCarteraExterna solo acepta el contexto que la frontera de esta
// misma ruta fijó con la hoja TLS y sesión vigentes. Nunca lee identidad del
// query, cuerpo o cabeceras del navegador.
type preparadorCarteraExterna struct {
	autoridad *autoridadPreferenciasUsuariosDesarrollo
	ruta      string
	motivoAUT vec.ReferenciaEntradaCatalogo
}

func (p preparadorCarteraExterna) contexto(r *http.Request) (contextoPreferenciasUsuarios, vec.ReferenciaCorrelacionAutorizacionV2, error) {
	var vacio contextoPreferenciasUsuarios
	if p.autoridad == nil || r == nil || r.URL == nil || r.Context() == nil ||
		r.URL.Path != p.ruta || p.autoridad.ruta != p.ruta ||
		p.autoridad.superficie != vec.SuperficieAutenticacionExternaPersonalV1 {
		return vacio, vec.ReferenciaCorrelacionAutorizacionV2{}, ports.ErrCarteraDenegada
	}
	c, ok := r.Context().Value(claveContextoPreferenciasUsuarios{}).(contextoPreferenciasUsuarios)
	if !ok || c.autoridad != p.autoridad || c.certificado == nil {
		return vacio, vec.ReferenciaCorrelacionAutorizacionV2{}, ports.ErrCarteraSinSesion
	}
	datos, err := c.vinculo.Datos()
	if c.resultado.Validar() != nil || c.vinculo.ValidarPara(c.resultado) != nil ||
		!c.vinculo.VigenteEn(p.autoridad.reloj.Ahora(), c.resultado) || err != nil ||
		datos.Superficie != vec.SuperficieAutenticacionExternaPersonalV1 ||
		datos.PrincipalID != c.resultado.Contexto.PersonaRef ||
		datos.CuentaPrivilegiada || datos.GarantiaObservada != vec.AuthAssuranceHigh {
		return vacio, vec.ReferenciaCorrelacionAutorizacionV2{}, ports.ErrCarteraSinSesion
	}
	correlacion, err := vec.GenerarReferenciaCorrelacionAutorizacionV2(r.Context(), seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, vec.ReferenciaCorrelacionAutorizacionV2{}, ports.ErrCarteraNoDisponible
	}
	return c, correlacion, nil
}

func (p preparadorCarteraExterna) PrepararSolicitudCartera(r *http.Request) (application.SolicitudCartera, error) {
	c, correlacion, err := p.contexto(r)
	if err != nil {
		return application.SolicitudCartera{}, err
	}
	return application.SolicitudCartera{ResultadoContexto: c.resultado,
		Vinculo: c.vinculo, Correlacion: correlacion}, nil
}

func (p preparadorCarteraExterna) PrepararSolicitudRetiradaCartera(r *http.Request) (application.SolicitudRetiradaCartera, error) {
	if p.motivoAUT.Validar() != nil || p.motivoAUT.CatalogoID != "motivos_autorizacion_meritos_cartera" {
		return application.SolicitudRetiradaCartera{}, ports.ErrCarteraNoDisponible
	}
	c, correlacion, err := p.contexto(r)
	if err != nil {
		return application.SolicitudRetiradaCartera{}, err
	}
	return application.SolicitudRetiradaCartera{ResultadoContexto: c.resultado,
		Vinculo: c.vinculo, Correlacion: correlacion, MotivoAutorizacion: p.motivoAUT}, nil
}
