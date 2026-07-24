package postgres

import (
	"bytes"
	"crypto/hmac"
	"encoding/json"
	"io"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type operacionPrepararAsignacionV1 struct {
	Esquema               string                          `json:"esquema"`
	SellosHMAC            sellosPrepararAltaV2            `json:"sellos_hmac"`
	Operacion             ports.TipoOperacionAsignacion   `json:"operacion"`
	OrganizacionRef       string                          `json:"organizacion_ref"`
	ExpedienteRef         string                          `json:"expediente_ref"`
	VersionExpediente     uint64                          `json:"version_expediente"`
	ActorRef              string                          `json:"actor_ref"`
	PerfilRef             string                          `json:"perfil_ref"`
	UnidadRef             string                          `json:"unidad_ref"`
	ResponsableRef        string                          `json:"responsable_ref"`
	ReferenciasCandidatas referenciasPrepararAsignacionV1 `json:"referencias_candidatas"`
}

type referenciasPrepararAsignacionV1 struct {
	ReservaRef      string `json:"reserva_ref"`
	ReciboRef       string `json:"recibo_ref"`
	NotificacionRef string `json:"notificacion_ref"`
	BandejaRef      string `json:"bandeja_ref"`
	AuditoriaRef    string `json:"auditoria_ref"`
	EventoRef       string `json:"evento_ref"`
}

func nuevasReferenciasPrepararAsignacionV1(
	referencias ports.ReferenciasEfectoAsignacion,
) referenciasPrepararAsignacionV1 {
	return referenciasPrepararAsignacionV1{
		ReservaRef:      referencias.ReservaRef,
		ReciboRef:       referencias.ReciboRef,
		NotificacionRef: referencias.NotificacionRef,
		BandejaRef:      referencias.BandejaRef,
		AuditoriaRef:    referencias.AuditoriaRef,
		EventoRef:       referencias.EventoRef,
	}
}

func (r referenciasPrepararAsignacionV1) puertos() ports.ReferenciasEfectoAsignacion {
	return ports.ReferenciasEfectoAsignacion{
		ReservaRef:      r.ReservaRef,
		ReciboRef:       r.ReciboRef,
		NotificacionRef: r.NotificacionRef,
		BandejaRef:      r.BandejaRef,
		AuditoriaRef:    r.AuditoriaRef,
		EventoRef:       r.EventoRef,
	}
}

func nuevaOperacionPrepararAsignacion(
	solicitud ports.SolicitudPrepararAsignacion,
	referencias ports.ReferenciasEfectoAsignacion,
) (operacionPrepararAsignacionV1, error) {
	if solicitud.Validar() != nil || referencias.Validar() != nil {
		return operacionPrepararAsignacionV1{},
			ports.ErrPreparacionAsignacionInvalida
	}
	sellos, err := nuevosSellosPrepararAltaV2(
		solicitud.AmbitosHMAC,
		solicitud.HuellasPeticionHMAC,
	)
	if err != nil {
		return operacionPrepararAsignacionV1{},
			ports.ErrPreparacionAsignacionInvalida
	}
	return operacionPrepararAsignacionV1{
		Esquema:           esquemaPrepararAsignacion,
		SellosHMAC:        sellos,
		Operacion:         solicitud.Operacion,
		OrganizacionRef:   solicitud.OrganizacionRef,
		ExpedienteRef:     solicitud.ExpedienteRef,
		VersionExpediente: solicitud.VersionExpediente,
		ActorRef:          solicitud.ActorRef,
		PerfilRef:         solicitud.PerfilRef,
		UnidadRef:         solicitud.UnidadRef,
		ResponsableRef:    solicitud.ResponsableRef,
		ReferenciasCandidatas: nuevasReferenciasPrepararAsignacionV1(
			referencias,
		),
	}, nil
}

func (f filaPreparacionAsignacion) restaurar(
	solicitud ports.SolicitudPrepararAsignacion,
	operacion operacionPrepararAsignacionV1,
) (ports.PreparacionAsignacion, error) {
	if operacion.Esquema != esquemaPrepararAsignacion ||
		!operacion.SellosHMAC.contienePar(
			f.ambitoHMAC,
			f.huellaPeticionHMAC,
		) {
		return ports.PreparacionAsignacion{},
			ports.ErrPersistenciaAsignacionNoDisponible
	}
	var expediente domain.Expediente
	if decodificarJSONEstricto(
		[]byte(f.expedienteJSON),
		&expediente,
	) != nil || expediente.Validar() != nil {
		return ports.PreparacionAsignacion{},
			ports.ErrPersistenciaAsignacionNoDisponible
	}
	preparacion := ports.PreparacionAsignacion{
		Expediente: expediente,
		Referencias: ports.ReferenciasEfectoAsignacion{
			ReservaRef:      f.reservaRef,
			ReciboRef:       f.reciboRef,
			NotificacionRef: f.notificacionRef,
			BandejaRef:      f.bandejaRef,
			AuditoriaRef:    f.auditoriaRef,
			EventoRef:       f.eventoRef,
		},
		AmbitoIdempotenciaHMAC: f.ambitoHMAC,
		HuellaPeticionHMAC:     f.huellaPeticionHMAC,
		Operacion:              ports.TipoOperacionAsignacion(f.operacion),
		OrganizacionRef:        f.organizacionRef,
		ActorRef:               f.actorRef,
		PerfilRef:              f.perfilRef,
		UnidadRef:              f.unidadRef,
		ResponsableRef:         f.responsableRef,
	}
	switch f.estado {
	case string(ports.PreparacionAsignacionReservada):
		if f.resultado != "reservada" && f.resultado != "reutilizada" {
			return ports.PreparacionAsignacion{},
				ports.ErrPersistenciaAsignacionNoDisponible
		}
		if f.versionResultante.Valid ||
			f.concesionV3DecisionRef.Valid ||
			f.confirmadaEn.Valid {
			return ports.PreparacionAsignacion{},
				ports.ErrPersistenciaAsignacionNoDisponible
		}
		preparacion.Estado = ports.PreparacionAsignacionReservada
		if f.resultado == "reservada" &&
			!respuestaAsignacionCoincideConCandidatos(
				preparacion,
				operacion,
			) {
			return ports.PreparacionAsignacion{},
				ports.ErrPersistenciaAsignacionNoDisponible
		}
	case string(ports.PreparacionAsignacionConfirmada):
		if f.resultado != "confirmada" ||
			!f.versionResultante.Valid ||
			f.versionResultante.Int64 < 2 ||
			!f.concesionV3DecisionRef.Valid ||
			!f.confirmadaEn.Valid {
			return ports.PreparacionAsignacion{},
				ports.ErrPersistenciaAsignacionNoDisponible
		}
		preparacion.Estado = ports.PreparacionAsignacionConfirmada
		recibo := ports.ReciboAsignacion{
			Operacion:              ports.TipoOperacionAsignacion(f.operacion),
			OrganizacionRef:        f.organizacionRef,
			ExpedienteRef:          expediente.Referencia,
			VersionAnterior:        expediente.Version,
			VersionResultante:      uint64(f.versionResultante.Int64),
			UnidadRef:              f.unidadRef,
			ResponsableRef:         f.responsableRef,
			ReciboRef:              f.reciboRef,
			NotificacionRef:        f.notificacionRef,
			BandejaRef:             f.bandejaRef,
			AuditoriaRef:           f.auditoriaRef,
			EventoRef:              f.eventoRef,
			ConcesionV3DecisionRef: f.concesionV3DecisionRef.String,
			AmbitoIdempotenciaHMAC: f.ambitoHMAC,
			HuellaPeticionHMAC:     f.huellaPeticionHMAC,
			ConfirmadaEn:           f.confirmadaEn.Time.UTC(),
		}
		preparacion.ReciboConfirmado = &recibo
	default:
		return ports.PreparacionAsignacion{},
			ports.ErrPersistenciaAsignacionNoDisponible
	}
	if preparacion.ValidarPara(solicitud) != nil {
		return ports.PreparacionAsignacion{},
			ports.ErrPersistenciaAsignacionNoDisponible
	}
	return preparacion, nil
}

func respuestaAsignacionCoincideConCandidatos(
	preparacion ports.PreparacionAsignacion,
	operacion operacionPrepararAsignacionV1,
) bool {
	return preparacion.Referencias ==
		operacion.ReferenciasCandidatas.puertos() &&
		hmac.Equal(
			[]byte(preparacion.AmbitoIdempotenciaHMAC),
			[]byte(operacion.SellosHMAC.Activo.AmbitoHMAC),
		) &&
		hmac.Equal(
			[]byte(preparacion.HuellaPeticionHMAC),
			[]byte(operacion.SellosHMAC.Activo.HuellaPeticionHMAC),
		)
}

func decodificarJSONEstricto(contenido []byte, destino any) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		return err
	}
	if err := decodificador.Decode(&struct{}{}); err != io.EOF {
		return ports.ErrPersistenciaAsignacionNoDisponible
	}
	return nil
}
