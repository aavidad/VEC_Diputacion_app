package bootstrap

import (
	"context"
	"crypto/hmac"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgrescontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	seguridadcontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/seguridad"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	responsableAsignacionContratacionTemporalDesarrollo = "persona:responsable-sintetica-001"
	finalidadAsignacionContratacionTemporalDesarrollo   = "gestionar_contratacion_temporal"
	definicionDestinoAsignacionDesarrollo               = "directorio:ct:desarrollo:asignacion:v1"
	evidenciaDestinoAsignacionDesarrollo                = "evidencia:ct:desarrollo:asignacion:v1"
	definicionPoliticaAsignacionDesarrollo              = "politica:ct:desarrollo:asignacion:v1"
)

var errAsignacionContratacionTemporalDesarrolloNoDisponible = errors.New(
	"contratacion temporal: asignacion de desarrollo no disponible",
)

type resolutorDestinoAsignacionContratacionTemporalDesarrollo struct{}

func (resolutorDestinoAsignacionContratacionTemporalDesarrollo) ResolverDestinoAsignacion(
	ctx context.Context,
	solicitud ports.SolicitudResolverDestinoAsignacion,
) (ports.DestinoAsignacionResuelto, error) {
	if contextoInterfazNulo(ctx) || solicitud.Validar() != nil ||
		solicitud.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		solicitud.UnidadRef != unidadCoberturaContratacionTemporalDesarrollo ||
		solicitud.ResponsableRef != responsableAsignacionContratacionTemporalDesarrollo ||
		solicitud.ActorRef == solicitud.ResponsableRef {
		return ports.DestinoAsignacionResuelto{}, ports.ErrDestinoAsignacionNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.DestinoAsignacionResuelto{}, errors.Join(
			ports.ErrDestinoAsignacionNoDisponible,
			err,
		)
	}
	destino := ports.DestinoAsignacionResuelto{
		OrganizacionRef:        solicitud.OrganizacionRef,
		ExpedienteRef:          solicitud.ExpedienteRef,
		VersionExpediente:      solicitud.VersionExpediente,
		ActorRef:               solicitud.ActorRef,
		UnidadRef:              solicitud.UnidadRef,
		ResponsableRef:         solicitud.ResponsableRef,
		DefinicionRef:          definicionDestinoAsignacionDesarrollo,
		DefinicionVersion:      1,
		DefinicionHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("directorio-asignacion"),
		EvidenciaRef:           evidenciaDestinoAsignacionDesarrollo,
		EvidenciaHuellaSHA256:  huellaAltaContratacionTemporalDesarrollo("evidencia-destino-asignacion"),
		EvaluadoEn:             solicitud.Instante,
		ValidoHasta:            solicitud.Instante.Add(5 * time.Minute),
	}
	if destino.ValidarPara(solicitud, solicitud.Instante) != nil {
		return ports.DestinoAsignacionResuelto{}, ports.ErrDestinoAsignacionNoDisponible
	}
	return destino, nil
}

type resolutorPoliticaAsignacionContratacionTemporalDesarrollo struct{}

func (resolutorPoliticaAsignacionContratacionTemporalDesarrollo) ResolverPoliticaAsignacion(
	ctx context.Context,
	solicitud ports.SolicitudResolverPoliticaAsignacion,
) (ports.PoliticaAsignacion, error) {
	if contextoInterfazNulo(ctx) || solicitud.Validar() != nil ||
		solicitud.Operacion != ports.OperacionRegistrarAsignacion ||
		solicitud.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		solicitud.Flujo.DefinicionRef != "flujo:ct:desarrollo" ||
		solicitud.Flujo.Version != 1 ||
		!hmac.Equal(
			[]byte(solicitud.Flujo.HuellaSHA256),
			[]byte(huellaAltaContratacionTemporalDesarrollo("flujo")),
		) ||
		solicitud.FasePrevia != domain.ClaveFase("asignacion_unidad") ||
		solicitud.EstadoPrevio != domain.EstadoEnCurso ||
		solicitud.UnidadAnteriorRef != "" ||
		solicitud.ResponsableAnteriorRef != "" ||
		solicitud.MotivoReasignacionClave != "" ||
		solicitud.Destino.UnidadRef != unidadCoberturaContratacionTemporalDesarrollo ||
		solicitud.Destino.ResponsableRef != responsableAsignacionContratacionTemporalDesarrollo ||
		solicitud.ActorRef == solicitud.Destino.ResponsableRef {
		return ports.PoliticaAsignacion{}, ports.ErrPoliticaAsignacionNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.PoliticaAsignacion{}, errors.Join(
			ports.ErrPoliticaAsignacionNoDisponible,
			err,
		)
	}
	politica := ports.PoliticaAsignacion{
		Operacion:                     solicitud.Operacion,
		OrganizacionRef:               solicitud.OrganizacionRef,
		ExpedienteRef:                 solicitud.ExpedienteRef,
		VersionExpediente:             solicitud.VersionExpediente,
		ActorRef:                      solicitud.ActorRef,
		PerfilRef:                     solicitud.PerfilRef,
		DestinoEvidenciaRef:           solicitud.Destino.EvidenciaRef,
		DestinoEvidenciaHuellaSHA256:  solicitud.Destino.EvidenciaHuellaSHA256,
		DefinicionRef:                 definicionPoliticaAsignacionDesarrollo,
		DefinicionVersion:             1,
		DefinicionHuellaSHA256:        huellaAltaContratacionTemporalDesarrollo("politica-asignacion"),
		Accion:                        domain.ClaveCatalogo(ports.AccionRegistrarAsignacion),
		Finalidad:                     domain.ClaveCatalogo(finalidadAsignacionContratacionTemporalDesarrollo),
		UnidadEjecutoraRef:            unidadCoberturaContratacionTemporalDesarrollo,
		MotivoAutorizacion:            referenciaMotivoAutorizacionAsignacionDesarrollo(),
		ExigeActorDistintoResponsable: true,
		EvaluadaEn:                    solicitud.Instante,
		ValidaHasta:                   solicitud.Instante.Add(5 * time.Minute),
	}
	if politica.ValidarPara(solicitud, solicitud.Instante) != nil {
		return ports.PoliticaAsignacion{}, ports.ErrPoliticaAsignacionNoDisponible
	}
	return politica, nil
}

func nuevasDependenciasAsignacionContratacionTemporalDesarrollo(
	derivador *derivadorIdentidadOperacionDesarrollo,
	alta *dependenciasAltaContratacionTemporalDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (*application.ServicioAsignacion, error) {
	if derivador == nil || !derivador.valido() || alta == nil ||
		alta.soporte == nil || alta.autorizador == nil ||
		alta.postgresql.ejecucion == nil {
		return nil, errAsignacionContratacionTemporalDesarrolloNoDisponible
	}
	ambitoActivo, ambitosRetenidos, err :=
		configuracionesHMACAltaContratacionTemporalDesarrollo(
			derivador,
			ports.DominioAmbitoIdempotenciaAsignacion,
			true,
		)
	if err != nil {
		return nil, errAsignacionContratacionTemporalDesarrolloNoDisponible
	}
	huellaActiva, huellasRetenidas, err :=
		configuracionesHMACAltaContratacionTemporalDesarrollo(
			derivador,
			ports.DominioHuellaPeticionAsignacion,
			false,
		)
	if err != nil {
		return nil, errAsignacionContratacionTemporalDesarrolloNoDisponible
	}
	sellos, err := seguridadcontratacion.NuevaAutoridadSellosAsignacionHMAC(
		ambitoActivo,
		ambitosRetenidos,
		huellaActiva,
		huellasRetenidas,
	)
	if err != nil {
		return nil, errAsignacionContratacionTemporalDesarrolloNoDisponible
	}
	referencias := seguridadcontratacion.NuevoGeneradorReferenciasAltaCriptografico()
	consultas, err := postgrescontratacion.NuevoConsultorAsignacionPostgreSQL(
		alta.postgresql.ejecucion,
	)
	if err != nil {
		return nil, errAsignacionContratacionTemporalDesarrolloNoDisponible
	}
	preparaciones, err := postgrescontratacion.NuevoPreparadorAsignacionPostgreSQL(
		alta.postgresql.ejecucion,
		referencias,
	)
	if err != nil {
		return nil, errAsignacionContratacionTemporalDesarrolloNoDisponible
	}
	transaccion, err := postgrescontratacion.NuevaTransaccionAsignacionesPostgreSQL(
		alta.postgresql.ejecucion,
		alta.postgresql.proveedorMaterial,
	)
	if err != nil {
		return nil, errAsignacionContratacionTemporalDesarrolloNoDisponible
	}
	servicio, err := application.NuevoServicioAsignacion(
		alta.soporte,
		sellos,
		sellos,
		consultas,
		preparaciones,
		resolutorDestinoAsignacionContratacionTemporalDesarrollo{},
		resolutorPoliticaAsignacionContratacionTemporalDesarrollo{},
		seguridadvec.GeneradorReferenciasCriptograficas{},
		alta.autorizador,
		reloj,
		transaccion,
	)
	if err != nil {
		return nil, errAsignacionContratacionTemporalDesarrolloNoDisponible
	}
	return servicio, nil
}

func nuevaInstantaneaAutorizacionAsignacionContratacionTemporalDesarrollo(
	principalID string,
	perfilRef string,
	ahora time.Time,
) (dominiovec.InstantaneaAutorizacion, error) {
	return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(
		principalID,
		perfilRef,
		ahora,
		"tecnico_rrhh_asignacion_desarrollo",
		"Tecnico RRHH de asignacion de desarrollo",
		"asignacion-rrhh-unidad-desarrollo-no-autoritativa",
		[]dominiovec.ConcesionRol{{
			Accion:         ports.AccionRegistrarAsignacion,
			ModuloID:       ports.ModuloContratacion,
			TipoRecurso:    ports.TipoRecursoAsignacion,
			Finalidades:    []string{finalidadAsignacionContratacionTemporalDesarrollo},
			GarantiaMinima: dominiovec.AuthAssuranceHigh,
		}},
		[]dominiovec.AmbitoPerfil{
			{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
			{Clave: "expediente_ref", Valores: []string{expedienteContratacionTemporalDesarrolloRef}},
			{Clave: "fase_previa", Valores: []string{"asignacion_unidad"}},
			{Clave: "estado_previo", Valores: []string{string(domain.EstadoEnCurso)}},
			{Clave: "unidad_destino_ref", Valores: []string{unidadCoberturaContratacionTemporalDesarrollo}},
		},
	)
}

func referenciaMotivoAutorizacionAsignacionDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_autorizacion_asignacion",
		CatalogoVersion:      1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-asignacion"),
		EntradaClave: referenciaAltaContratacionTemporalDesarrollo(
			"motivo_",
			"asignar-unidad",
		),
	}
}

func solicitudAutorizacionAsignacionContratacionTemporalDesarrolloValida(
	ruta string,
	datos dominiovec.DatosSolicitudAutorizacionLigadaV3,
) bool {
	atributos := datos.Recurso.Atributos
	ambitos := datos.Recurso.Ambitos
	return ruta == httpinterno.RutaAsignaciones &&
		datos.Accion == ports.AccionRegistrarAsignacion &&
		datos.ReferenciaMotivo == referenciaMotivoAutorizacionAsignacionDesarrollo() &&
		datos.Recurso.ModuloID == ports.ModuloContratacion &&
		datos.Recurso.Tipo == ports.TipoRecursoAsignacion &&
		datos.Finalidad == finalidadAsignacionContratacionTemporalDesarrollo &&
		len(ambitos) == 5 && len(atributos) == 12 &&
		ambitos["organizacion_ref"] == organizacionAltaContratacionTemporalDesarrollo &&
		ambitos["expediente_ref"] == datos.Recurso.Referencia &&
		ambitos["fase_previa"] == "asignacion_unidad" &&
		ambitos["estado_previo"] == string(domain.EstadoEnCurso) &&
		ambitos["unidad_destino_ref"] == unidadCoberturaContratacionTemporalDesarrollo &&
		atributos[ports.AtributoOperacionAsignacion] == string(ports.OperacionRegistrarAsignacion) &&
		atributos[ports.AtributoVersionAsignacion] == "3" &&
		atributos[ports.AtributoPoliticaAsignacionRef] == definicionPoliticaAsignacionDesarrollo &&
		atributos[ports.AtributoPoliticaAsignacionVersion] == strconv.FormatUint(1, 10) &&
		hmac.Equal(
			[]byte(atributos[ports.AtributoPoliticaAsignacionHuella]),
			[]byte(huellaAltaContratacionTemporalDesarrollo("politica-asignacion")),
		) &&
		atributos[ports.AtributoEvidenciaDestinoRef] == evidenciaDestinoAsignacionDesarrollo &&
		hmac.Equal(
			[]byte(atributos[ports.AtributoEvidenciaDestinoHuella]),
			[]byte(huellaAltaContratacionTemporalDesarrollo("evidencia-destino-asignacion")),
		) &&
		atributos[ports.AtributoUnidadDestino] == unidadCoberturaContratacionTemporalDesarrollo &&
		atributos[ports.AtributoResponsableDestino] == responsableAsignacionContratacionTemporalDesarrollo &&
		selloHMACAsignacionContratacionTemporalDesarrolloValido(
			atributos[ports.AtributoAmbitoIdempotenciaActivo],
			ports.DominioAmbitoIdempotenciaAsignacion,
		) &&
		selloHMACAsignacionContratacionTemporalDesarrolloValido(
			atributos[ports.AtributoHuellaPeticionAsignacion],
			ports.DominioHuellaPeticionAsignacion,
		) &&
		atributos[ports.AtributoSegregacionAsignacion] == "true"
}

func selloHMACAsignacionContratacionTemporalDesarrolloValido(
	valor string,
	dominio string,
) bool {
	prefijo := "hmac-sha256:" + dominio + "/v"
	if !strings.HasPrefix(valor, prefijo) {
		return false
	}
	resto := strings.TrimPrefix(valor, prefijo)
	separador := strings.IndexByte(resto, ':')
	if separador <= 0 {
		return false
	}
	generacion, err := strconv.ParseUint(resto[:separador], 10, 64)
	huella := resto[separador+1:]
	bytesHuella, errHuella := hex.DecodeString(huella)
	return err == nil && generacion > 0 && errHuella == nil &&
		len(bytesHuella) == 32 && hex.EncodeToString(bytesHuella) == huella
}
