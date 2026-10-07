package application

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// GobernarEvidenciasCircuitoRRHH compone una fuente corporativa positiva.
// La ausencia de fuente deniega el análisis del flujo nuevo.
func (s *ServicioOperacionAnalisis) GobernarEvidenciasCircuitoRRHH(f ports.FuenteEvidenciasCircuitoRRHH) error {
	if s == nil || dependenciaNula(f) || s.evidenciasCircuito != nil {
		return ErrServicioOperacionAnalisisInvalido
	}
	s.evidenciasCircuito = f
	return nil
}

func (s *ServicioOperacionAnalisis) adjuntarHitosAnalisisCircuitoRRHH(
	ctx context.Context,
	anterior, siguiente domain.Expediente,
	actorRef, perfilRef string,
) (domain.Expediente, error) {
	if s == nil || dependenciaNula(s.evidenciasCircuito) ||
		anterior.Circuito == nil || siguiente.Circuito == nil ||
		siguiente.Version != anterior.Version+1 {
		return domain.Expediente{}, ports.ErrEvidenciasCircuitoRRHHNoDisponibles
	}
	solicitud := ports.SolicitudEvidenciasAnalisisCircuitoRRHH{
		OrganizacionRef: anterior.OrganizacionRef,
		ExpedienteRef:   anterior.Referencia,
		VersionEntrada:  anterior.Version,
		Flujo:           anterior.Flujo,
		ActorRef:        actorRef,
		PerfilRef:       perfilRef,
	}
	if solicitud.Validar() != nil {
		return domain.Expediente{}, ports.ErrEvidenciasCircuitoRRHHNoDisponibles
	}
	evidencias, err := s.evidenciasCircuito.AcreditarAnalisisCircuitoRRHH(ctx, solicitud)
	if err != nil || ctx.Err() != nil {
		return domain.Expediente{}, ports.ErrEvidenciasCircuitoRRHHNoDisponibles
	}
	evidencias.FirmasPeticion = append([]domain.FirmaCircuitoRRHH(nil), evidencias.FirmasPeticion...)
	if evidencias.ValidarPara(solicitud) != nil {
		return domain.Expediente{}, ports.ErrEvidenciasCircuitoRRHHNoDisponibles
	}
	primera, ok := buscarTransicionCircuitoRRHH(
		evidencias.Definicion, domain.HitoPeticionFirmada, anterior.Circuito.EstadoActual,
	)
	if !ok || primera.PerfilClave != evidencias.PerfilRegistradorClave {
		return domain.Expediente{}, ports.ErrEvidenciasCircuitoRRHHNoDisponibles
	}
	segunda, ok := buscarTransicionCircuitoRRHH(
		evidencias.Definicion, domain.HitoAutorizacionRRHH, primera.Destino,
	)
	if !ok || segunda.PerfilClave != evidencias.PerfilRegistradorClave {
		return domain.Expediente{}, ports.ErrEvidenciasCircuitoRRHHNoDisponibles
	}
	actuacion := siguiente.Actuaciones[len(siguiente.Actuaciones)-1]
	comun := domain.HitoCircuitoRRHH{
		ActuacionClave: actuacion.AccionClave,
		ActorRef:       actuacion.ActorRef,
		PerfilClave:    evidencias.PerfilRegistradorClave,
		PerfilRef:      perfilRef,
		UnidadRef:      actuacion.UnidadRef,
		ReciboRef:      actuacion.ReciboRef,
		RegistradoEn:   actuacion.RealizadaEn,
	}
	peticion := comun
	peticion.Clave = primera.Clave
	peticion.DocumentoRef = evidencias.DocumentoPeticionRef
	peticion.HuellaDocumentoSHA256 = evidencias.HuellaPeticionSHA256
	peticion.Firmas = evidencias.FirmasPeticion
	autorizacion := comun
	autorizacion.Clave = segunda.Clave
	autorizacion.DocumentoRef = evidencias.DocumentoAutorizacionRef
	autorizacion.HuellaDocumentoSHA256 = evidencias.HuellaAutorizacionSHA256
	autorizacion.ActoAutorizacionRef = evidencias.ActoAutorizacionRef
	autorizacion.AutorizanteRef = evidencias.AutorizanteRef
	autorizacion.CargoAutorizanteClave = evidencias.CargoAutorizanteClave
	return siguiente.AdjuntarHitosCircuito(
		evidencias.Definicion, siguiente.Version,
		[]domain.HitoCircuitoRRHH{peticion, autorizacion},
	)
}

func buscarTransicionCircuitoRRHH(
	definicion domain.DefinicionCircuitoRRHH,
	tipo domain.TipoHitoCircuitoRRHH,
	origen domain.ClaveFase,
) (domain.TransicionCircuitoRRHH, bool) {
	for _, transicion := range definicion.Transiciones {
		if transicion.Tipo == tipo && transicion.Origen == origen {
			return transicion, true
		}
	}
	return domain.TransicionCircuitoRRHH{}, false
}
