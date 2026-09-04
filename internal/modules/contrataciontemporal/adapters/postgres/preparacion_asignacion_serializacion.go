package postgres

import (
	"bytes"
	"crypto/hmac"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	esquemaConsultarAsignacion        = "vec.contratacion-temporal.consultar-asignacion.v1"
	esquemaConfirmarAsignacion        = "vec.contratacion-temporal.confirmar-asignacion.v1"
	maximoCargaConfirmacionAsignacion = 2 * 1024 * 1024
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

type operacionConfirmarAsignacionV1 struct {
	Esquema                string                            `json:"esquema"`
	ReservaRef             string                            `json:"reserva_ref"`
	Referencias            ports.ReferenciasEfectoAsignacion `json:"referencias"`
	AmbitoIdempotenciaHMAC string                            `json:"ambito_idempotencia_hmac"`
	HuellaPeticionHMAC     string                            `json:"huella_peticion_hmac"`
	Operacion              ports.TipoOperacionAsignacion     `json:"operacion"`
	OrganizacionRef        string                            `json:"organizacion_ref"`
	ExpedienteRef          string                            `json:"expediente_ref"`
	VersionAnterior        uint64                            `json:"version_anterior"`
	ActorRef               string                            `json:"actor_ref"`
	PerfilRef              string                            `json:"perfil_ref"`
	UnidadRef              string                            `json:"unidad_ref"`
	ResponsableRef         string                            `json:"responsable_ref"`
	ExpedienteAnterior     domain.Expediente                 `json:"expediente_anterior"`
	ExpedienteSiguiente    domain.Expediente                 `json:"expediente_siguiente"`
	Actuacion              domain.Actuacion                  `json:"actuacion"`
	Destino                destinoConfirmarAsignacionV1      `json:"destino"`
	Politica               politicaConfirmarAsignacionV1     `json:"politica"`
	Autorizacion           autorizacionConfirmarAsignacionV1 `json:"autorizacion"`
	InstanteEfecto         time.Time                         `json:"instante_efecto"`
}

type destinoConfirmarAsignacionV1 struct {
	EvidenciaRef          string `json:"evidencia_ref"`
	EvidenciaHuellaSHA256 string `json:"evidencia_huella_sha256"`
}

type politicaConfirmarAsignacionV1 struct {
	DefinicionRef          string               `json:"definicion_ref"`
	DefinicionVersion      uint64               `json:"definicion_version"`
	DefinicionHuellaSHA256 string               `json:"definicion_huella_sha256"`
	Accion                 domain.ClaveCatalogo `json:"accion"`
	Finalidad              domain.ClaveCatalogo `json:"finalidad"`
	UnidadEjecutoraRef     string               `json:"unidad_ejecutora_ref"`
}

type autorizacionConfirmarAsignacionV1 struct {
	DecisionCanonicaHex         string `json:"decision_canonica_hex"`
	MotivoCanonicoHex           string `json:"motivo_canonico_hex"`
	PersonaVersion              uint64 `json:"persona_version"`
	PerfilVersion               uint64 `json:"perfil_version"`
	DecisionRef                 string `json:"decision_ref"`
	DecisionHuellaSHA256        string `json:"decision_huella_sha256"`
	PrincipalID                 string `json:"principal_id"`
	PerfilActivoRef             string `json:"perfil_activo_ref"`
	Accion                      string `json:"accion"`
	RecursoRef                  string `json:"recurso_ref"`
	ContextoRecursoHuellaSHA256 string `json:"contexto_recurso_huella_sha256"`
	Finalidad                   string `json:"finalidad"`
}

func codificarOperacionConfirmarAsignacion(
	orden ports.OrdenConfirmarAsignacion,
) ([]byte, error) {
	evidencia, err := orden.Datos()
	if err != nil || len(evidencia.ExpedienteSiguiente.Actuaciones) == 0 {
		return nil, ports.ErrOrdenAsignacionInvalida
	}
	autorizacion, err := nuevaAutorizacionConfirmarAsignacion(evidencia)
	if err != nil {
		return nil, err
	}
	operacion := operacionConfirmarAsignacionV1{
		Esquema:                esquemaConfirmarAsignacion,
		ReservaRef:             evidencia.Preparacion.Referencias.ReservaRef,
		Referencias:            evidencia.Preparacion.Referencias,
		AmbitoIdempotenciaHMAC: evidencia.Preparacion.AmbitoIdempotenciaHMAC,
		HuellaPeticionHMAC:     evidencia.Preparacion.HuellaPeticionHMAC,
		Operacion:              evidencia.Material.Operacion,
		OrganizacionRef:        evidencia.Material.OrganizacionRef,
		ExpedienteRef:          evidencia.Material.ExpedienteRef,
		VersionAnterior:        evidencia.Material.VersionExpediente,
		ActorRef:               evidencia.Material.ActorRef, PerfilRef: evidencia.Material.PerfilRef,
		UnidadRef:           evidencia.Material.UnidadRef,
		ResponsableRef:      evidencia.Material.ResponsableRef,
		ExpedienteAnterior:  evidencia.ExpedienteAnterior,
		ExpedienteSiguiente: evidencia.ExpedienteSiguiente,
		Actuacion:           evidencia.ExpedienteSiguiente.Actuaciones[len(evidencia.ExpedienteSiguiente.Actuaciones)-1],
		Destino: destinoConfirmarAsignacionV1{
			EvidenciaRef:          evidencia.Destino.EvidenciaRef,
			EvidenciaHuellaSHA256: evidencia.Destino.EvidenciaHuellaSHA256,
		},
		Politica: politicaConfirmarAsignacionV1{
			DefinicionRef:          evidencia.Politica.DefinicionRef,
			DefinicionVersion:      evidencia.Politica.DefinicionVersion,
			DefinicionHuellaSHA256: evidencia.Politica.DefinicionHuellaSHA256,
			Accion:                 evidencia.Politica.Accion, Finalidad: evidencia.Politica.Finalidad,
			UnidadEjecutoraRef: evidencia.Politica.UnidadEjecutoraRef,
		},
		Autorizacion: autorizacion, InstanteEfecto: evidencia.InstanteEfecto,
	}
	contenido, err := json.Marshal(operacion)
	if err != nil || len(contenido) == 0 || len(contenido) > maximoCargaConfirmacionAsignacion {
		return nil, fmt.Errorf("%w: proyección JSON rechazada", ports.ErrOrdenAsignacionInvalida)
	}
	return contenido, nil
}

func nuevaAutorizacionConfirmarAsignacion(
	evidencia ports.EvidenciaOrdenConfirmarAsignacion,
) (autorizacionConfirmarAsignacionV1, error) {
	solicitud, errSolicitud := evidencia.SolicitudV3.Datos()
	decisionCanonica, errDecision := dominiovec.RepresentacionCanonicaDecisionAutorizacionV3(
		evidencia.DecisionV3,
	)
	motivoCanonico, errMotivo := dominiovec.RepresentacionCanonicaMotivoAutorizacionV2(
		solicitud.ReferenciaMotivo,
	)
	confirmacion, errConfirmacion := evidencia.ConfirmacionV3.Datos()
	huellaContexto, errContexto := solicitud.Recurso.HuellaContextoAutorizacionSHA256()
	instantanea := evidencia.ContextoAutorizacion.Resultado.Contexto.Instantanea
	if errSolicitud != nil || errDecision != nil || errMotivo != nil ||
		errConfirmacion != nil || errContexto != nil {
		return autorizacionConfirmarAsignacionV1{}, ports.ErrOrdenAsignacionInvalida
	}
	return autorizacionConfirmarAsignacionV1{
		DecisionCanonicaHex: hex.EncodeToString(decisionCanonica),
		MotivoCanonicoHex:   hex.EncodeToString(motivoCanonico),
		PersonaVersion:      instantanea.PersonaVersion, PerfilVersion: instantanea.PerfilVersion,
		DecisionRef:          confirmacion.DecisionRef,
		DecisionHuellaSHA256: confirmacion.DecisionHuellaSHA256,
		PrincipalID:          instantanea.PersonaRef,
		PerfilActivoRef:      instantanea.PerfilActivoRef,
		Accion:               solicitud.Accion, RecursoRef: solicitud.Recurso.Referencia,
		ContextoRecursoHuellaSHA256: huellaContexto,
		Finalidad:                   solicitud.Finalidad,
	}, nil
}
