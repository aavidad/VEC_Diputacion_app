package postgres

import (
	"crypto/hmac"
	"encoding/json"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type operacionPrepararAnalisisV1 struct {
	Esquema               string                      `json:"esquema"`
	SellosHMAC            sellosPrepararAltaV2        `json:"sellos_hmac"`
	Operacion             ports.TipoOperacionAnalisis `json:"operacion"`
	OrganizacionRef       string                      `json:"organizacion_ref"`
	ExpedienteRef         string                      `json:"expediente_ref"`
	VersionExpediente     uint64                      `json:"version_expediente"`
	ActorRef              string                      `json:"actor_ref"`
	PerfilRef             string                      `json:"perfil_ref"`
	ArtefactoRef          string                      `json:"artefacto_ref"`
	ArtefactoHuellaSHA256 string                      `json:"artefacto_huella_sha256"`
}

func nuevaOperacionPrepararAnalisis(
	solicitud ports.SolicitudPrepararOperacionAnalisis,
) (operacionPrepararAnalisisV1, error) {
	if solicitud.Validar() != nil {
		return operacionPrepararAnalisisV1{},
			ports.ErrPreparacionOperacionAnalisisInvalida
	}
	sellos, err := nuevosSellosPrepararAltaV2(
		solicitud.Sellos.AmbitosIdempotenciaHMAC,
		solicitud.Sellos.HuellasSemanticasHMAC,
	)
	if err != nil {
		return operacionPrepararAnalisisV1{},
			ports.ErrPreparacionOperacionAnalisisInvalida
	}
	return operacionPrepararAnalisisV1{
		Esquema:               esquemaPrepararAnalisis,
		SellosHMAC:            sellos,
		Operacion:             solicitud.Operacion,
		OrganizacionRef:       solicitud.OrganizacionRef,
		ExpedienteRef:         solicitud.ExpedienteRef,
		VersionExpediente:     solicitud.VersionExpediente,
		ActorRef:              solicitud.ActorRef,
		PerfilRef:             solicitud.PerfilRef,
		ArtefactoRef:          solicitud.ArtefactoRef,
		ArtefactoHuellaSHA256: solicitud.ArtefactoHuellaSHA256,
	}, nil
}

func (f filaPreparacionAnalisis) restaurar(
	solicitud ports.SolicitudPrepararOperacionAnalisis,
	operacion operacionPrepararAnalisisV1,
) (ports.PreparacionOperacionAnalisis, error) {
	if operacion.Esquema != esquemaPrepararAnalisis ||
		!operacion.SellosHMAC.contienePar(
			f.ambitoHMAC,
			f.huellaSemanticaHMAC,
		) {
		return ports.PreparacionOperacionAnalisis{},
			ports.ErrPersistenciaOperacionAnalisisNoDisponible
	}
	datos := ports.DatosPreparacionOperacionAnalisis{
		ReservaRef:             f.reservaRef,
		ReciboRef:              f.reciboRef,
		Operacion:              ports.TipoOperacionAnalisis(f.operacion),
		OrganizacionRef:        f.organizacionRef,
		ExpedienteRef:          f.expedienteRef,
		VersionExpediente:      uint64(f.versionExpediente),
		ActorRef:               f.actorRef,
		PerfilRef:              f.perfilRef,
		ArtefactoRef:           f.artefactoRef,
		ArtefactoHuellaSHA256:  f.artefactoHuellaSHA256,
		AmbitoIdempotenciaHMAC: f.ambitoHMAC,
		HuellaSemanticaHMAC:    f.huellaSemanticaHMAC,
	}
	switch f.estado {
	case string(ports.PreparacionOperacionAnalisisReservada):
		if f.resultado != "reservada" && f.resultado != "reutilizada" ||
			f.reciboJSON != "" {
			return ports.PreparacionOperacionAnalisis{},
				ports.ErrPersistenciaOperacionAnalisisNoDisponible
		}
		var expediente domain.Expediente
		if decodificarJSONEstricto(
			[]byte(f.expedienteJSON),
			&expediente,
		) != nil || expediente.Validar() != nil {
			return ports.PreparacionOperacionAnalisis{},
				ports.ErrPersistenciaOperacionAnalisisNoDisponible
		}
		datos.Estado = ports.PreparacionOperacionAnalisisReservada
		datos.ExpedienteAnterior = &expediente
	case string(ports.PreparacionOperacionAnalisisConfirmada):
		if f.resultado != "confirmada" || f.expedienteJSON != "" {
			return ports.PreparacionOperacionAnalisis{},
				ports.ErrPersistenciaOperacionAnalisisNoDisponible
		}
		var recibo ports.ReciboOperacionAnalisis
		if decodificarJSONEstricto(
			[]byte(f.reciboJSON),
			&recibo,
		) != nil {
			return ports.PreparacionOperacionAnalisis{},
				ports.ErrPersistenciaOperacionAnalisisNoDisponible
		}
		datos.Estado = ports.PreparacionOperacionAnalisisConfirmada
		datos.ReciboConfirmado = &recibo
		datos.AmbitoConsultaHMAC = recibo.AmbitoConsultaHMAC
		datos.HuellaConsultaHMAC = recibo.HuellaConsultaHMAC
	default:
		return ports.PreparacionOperacionAnalisis{},
			ports.ErrPersistenciaOperacionAnalisisNoDisponible
	}
	preparacion, err := ports.NuevaPreparacionOperacionAnalisis(
		solicitud,
		datos,
	)
	if err != nil {
		return ports.PreparacionOperacionAnalisis{},
			ports.ErrPersistenciaOperacionAnalisisNoDisponible
	}
	return preparacion, nil
}

func codificarAmbitosConsultaAnalisis(
	solicitud ports.SolicitudConsultarOperacionAnalisisConfirmada,
) ([]byte, error) {
	ambitos, err := solicitud.AmbitosIdempotencia()
	if err != nil {
		return nil, ports.ErrPreparacionOperacionAnalisisInvalida
	}
	datos, err := ambitos.Datos()
	if err != nil {
		return nil, ports.ErrPreparacionOperacionAnalisisInvalida
	}
	valores := make([]string, 0, len(datos.Retenidos)+1)
	valores = append(valores, datos.Activo.Valor)
	for _, retenido := range datos.Retenidos {
		valores = append(valores, retenido.Valor)
	}
	return json.Marshal(valores)
}

func reciboConsultaAnalisisSeguro(
	solicitud ports.SolicitudConsultarOperacionAnalisisConfirmada,
	contenido string,
) (ports.ReciboOperacionAnalisis, error) {
	var recibo ports.ReciboOperacionAnalisis
	if decodificarJSONEstricto([]byte(contenido), &recibo) != nil ||
		recibo.ValidarParaConsulta(solicitud) != nil {
		return ports.ReciboOperacionAnalisis{},
			ports.ErrPersistenciaOperacionAnalisisNoDisponible
	}
	return recibo, nil
}

func paresOperacionAnalisisCoinciden(
	operacion operacionPrepararAnalisisV1,
	ambito string,
	huella string,
) bool {
	return hmac.Equal(
		[]byte(operacion.SellosHMAC.Activo.AmbitoHMAC),
		[]byte(ambito),
	) && hmac.Equal(
		[]byte(operacion.SellosHMAC.Activo.HuellaPeticionHMAC),
		[]byte(huella),
	)
}
