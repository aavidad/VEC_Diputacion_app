package postgres

import (
	"crypto/hmac"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type operacionPrepararInformeJuridicoV1 struct {
	Esquema               string                               `json:"esquema"`
	Operacion             string                               `json:"operacion"`
	SellosHMAC            sellosPrepararAltaV2                 `json:"sellos_hmac"`
	OrganizacionRef       string                               `json:"organizacion_ref"`
	ExpedienteRef         string                               `json:"expediente_ref"`
	VersionExpediente     uint64                               `json:"version_expediente"`
	ActorRef              string                               `json:"actor_ref"`
	PerfilRef             string                               `json:"perfil_ref"`
	ReferenciasCandidatas referenciasPrepararInformeJuridicoV1 `json:"referencias_candidatas"`
}

type referenciasPrepararInformeJuridicoV1 struct {
	ReservaRef   string `json:"reserva_ref"`
	InformeRef   string `json:"informe_ref"`
	DocumentoRef string `json:"documento_ref"`
	ReciboRef    string `json:"recibo_ref"`
	AuditoriaRef string `json:"auditoria_ref"`
	EventoRef    string `json:"evento_ref"`
}

func nuevasReferenciasPrepararInformeJuridicoV1(
	referencias ports.ReferenciasEfectoInformeJuridico,
) referenciasPrepararInformeJuridicoV1 {
	return referenciasPrepararInformeJuridicoV1{
		ReservaRef: referencias.ReservaRef, InformeRef: referencias.InformeRef,
		DocumentoRef: referencias.DocumentoRef, ReciboRef: referencias.ReciboRef,
		AuditoriaRef: referencias.AuditoriaRef, EventoRef: referencias.EventoRef,
	}
}

func (r referenciasPrepararInformeJuridicoV1) puertos() ports.ReferenciasEfectoInformeJuridico {
	return ports.ReferenciasEfectoInformeJuridico{
		ReservaRef: r.ReservaRef, InformeRef: r.InformeRef,
		DocumentoRef: r.DocumentoRef, ReciboRef: r.ReciboRef,
		AuditoriaRef: r.AuditoriaRef, EventoRef: r.EventoRef,
	}
}

func nuevaOperacionPrepararInformeJuridico(
	solicitud ports.SolicitudPrepararInformeJuridico,
	referencias ports.ReferenciasEfectoInformeJuridico,
) (operacionPrepararInformeJuridicoV1, error) {
	if solicitud.Validar() != nil || referencias.Validar() != nil {
		return operacionPrepararInformeJuridicoV1{},
			ports.ErrPreparacionInformeJuridicoInvalida
	}
	sellos, err := nuevosSellosPrepararAltaV2(
		solicitud.AmbitosHMAC,
		solicitud.HuellasPeticionHMAC,
	)
	if err != nil {
		return operacionPrepararInformeJuridicoV1{},
			ports.ErrPreparacionInformeJuridicoInvalida
	}
	return operacionPrepararInformeJuridicoV1{
		Esquema: esquemaPrepararInformeJuridico, Operacion: "preparar",
		SellosHMAC: sellos, OrganizacionRef: solicitud.Material.OrganizacionRef,
		ExpedienteRef:     solicitud.Material.ExpedienteRef,
		VersionExpediente: solicitud.Material.VersionExpediente,
		ActorRef:          solicitud.Material.ActorRef, PerfilRef: solicitud.Material.PerfilRef,
		ReferenciasCandidatas: nuevasReferenciasPrepararInformeJuridicoV1(referencias),
	}, nil
}

func (f filaPreparacionInformeJuridico) restaurar(
	solicitud ports.SolicitudPrepararInformeJuridico,
	operacion operacionPrepararInformeJuridicoV1,
) (ports.PreparacionInformeJuridico, error) {
	if operacion.Esquema != esquemaPrepararInformeJuridico ||
		operacion.Operacion != "preparar" ||
		f.versionExpediente < 1 ||
		!operacion.SellosHMAC.contienePar(f.ambitoHMAC, f.huellaPeticionHMAC) {
		return ports.PreparacionInformeJuridico{},
			ports.ErrPersistenciaInformeJuridicoNoDisponible
	}
	var expediente domain.Expediente
	if decodificarJSONEstricto([]byte(f.expedienteJSON), &expediente) != nil ||
		expediente.Validar() != nil {
		return ports.PreparacionInformeJuridico{},
			ports.ErrPersistenciaInformeJuridicoNoDisponible
	}
	preparacion := ports.PreparacionInformeJuridico{
		Expediente: expediente,
		Referencias: ports.ReferenciasEfectoInformeJuridico{
			ReservaRef: f.reservaRef, InformeRef: f.informeRef,
			DocumentoRef: f.documentoRef, ReciboRef: f.reciboRef,
			AuditoriaRef: f.auditoriaRef, EventoRef: f.eventoRef,
		},
		AmbitoIdempotenciaHMAC: f.ambitoHMAC,
		HuellaPeticionHMAC:     f.huellaPeticionHMAC,
		Material: ports.MaterialHuellaInformeJuridico{
			OrganizacionRef: f.organizacionRef, ExpedienteRef: f.expedienteRef,
			VersionExpediente: uint64(f.versionExpediente),
			ActorRef:          f.actorRef, PerfilRef: f.perfilRef,
		},
	}
	switch f.estado {
	case string(ports.PreparacionInformeJuridicoReservada):
		if f.resultado != "reservada" && f.resultado != "reutilizada" ||
			f.reciboJSON.Valid {
			return ports.PreparacionInformeJuridico{},
				ports.ErrPersistenciaInformeJuridicoNoDisponible
		}
		preparacion.Estado = ports.PreparacionInformeJuridicoReservada
		if f.resultado == "reservada" &&
			!respuestaInformeJuridicoCoincideConCandidatos(preparacion, operacion) {
			return ports.PreparacionInformeJuridico{},
				ports.ErrPersistenciaInformeJuridicoNoDisponible
		}
	case string(ports.PreparacionInformeJuridicoConfirmada):
		if f.resultado != "confirmada" || !f.reciboJSON.Valid {
			return ports.PreparacionInformeJuridico{},
				ports.ErrPersistenciaInformeJuridicoNoDisponible
		}
		recibo, err := decodificarReciboInformeJuridico(f.reciboJSON.String)
		if err != nil {
			return ports.PreparacionInformeJuridico{}, err
		}
		preparacion.Estado = ports.PreparacionInformeJuridicoConfirmada
		preparacion.ReciboConfirmado = &recibo
	default:
		return ports.PreparacionInformeJuridico{},
			ports.ErrPersistenciaInformeJuridicoNoDisponible
	}
	if preparacion.ValidarPara(solicitud) != nil {
		return ports.PreparacionInformeJuridico{},
			ports.ErrPersistenciaInformeJuridicoNoDisponible
	}
	return preparacion, nil
}

func respuestaInformeJuridicoCoincideConCandidatos(
	preparacion ports.PreparacionInformeJuridico,
	operacion operacionPrepararInformeJuridicoV1,
) bool {
	return preparacion.Referencias == operacion.ReferenciasCandidatas.puertos() &&
		hmac.Equal(
			[]byte(preparacion.AmbitoIdempotenciaHMAC),
			[]byte(operacion.SellosHMAC.Activo.AmbitoHMAC),
		) &&
		hmac.Equal(
			[]byte(preparacion.HuellaPeticionHMAC),
			[]byte(operacion.SellosHMAC.Activo.HuellaPeticionHMAC),
		)
}
