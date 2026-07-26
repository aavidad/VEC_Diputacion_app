package ports

import (
	"context"
)

// SolicitudesFuentesAnalisisO3 agrupa peticiones ya selladas por la
// infraestructura interna. No forma parte del DTO de usuario.
type SolicitudesFuentesAnalisisO3 struct {
	ValidacionRC SolicitudValidarRC
	CalculoCoste *SolicitudCalcularCoste
}

func (s SolicitudesFuentesAnalisisO3) ValidarPara(
	solicitud SolicitudPrepararArtefactoAnalisis,
) error {
	datosRC, errRC := s.ValidacionRC.Datos()
	if solicitud.Validar() != nil || errRC != nil ||
		datosRC.OrganizacionRef != solicitud.OrganizacionRef ||
		datosRC.ExpedienteRef != solicitud.ExpedienteRef ||
		datosRC.VersionExpediente != solicitud.VersionExpediente ||
		datosRC.Entrada.Referencia !=
			solicitud.DatosFuncionales.EntradaRC.Referencia ||
		datosRC.Entrada.HuellaSHA256 !=
			solicitud.DatosFuncionales.EntradaRC.HuellaSHA256 ||
		datosRC.SolicitadaEn.Before(solicitud.SolicitadaEn) {
		return ErrSolicitudArtefactoAnalisisInvalida
	}
	if s.CalculoCoste == nil {
		return nil
	}
	datosCoste, errCoste := s.CalculoCoste.Datos()
	if errCoste != nil ||
		datosCoste.OrganizacionRef != solicitud.OrganizacionRef ||
		datosCoste.ExpedienteRef != solicitud.ExpedienteRef ||
		datosCoste.VersionExpediente != solicitud.VersionExpediente ||
		datosCoste.CategoriaRef !=
			solicitud.DatosFuncionales.CategoriaRef ||
		datosCoste.GrupoSubgrupo !=
			solicitud.DatosFuncionales.GrupoSubgrupo ||
		datosCoste.ModalidadClave !=
			solicitud.DatosFuncionales.ModalidadClave ||
		datosCoste.CausaClave != solicitud.DatosFuncionales.CausaClave ||
		!datosCoste.Periodo.Inicio.Equal(
			solicitud.DatosFuncionales.Periodo.Inicio,
		) ||
		!datosCoste.Periodo.Fin.Equal(
			solicitud.DatosFuncionales.Periodo.Fin,
		) ||
		datosCoste.Jornada !=
			solicitud.DatosFuncionales.PorcentajeJornada ||
		datosCoste.SolicitadaEn.Before(solicitud.SolicitadaEn) {
		return ErrSolicitudArtefactoAnalisisInvalida
	}
	return nil
}

type PreparadorSolicitudesFuentesAnalisisO3 interface {
	PrepararSolicitudesFuentesAnalisisO3(
		context.Context,
		SolicitudPrepararArtefactoAnalisis,
	) (SolicitudesFuentesAnalisisO3, error)
}
