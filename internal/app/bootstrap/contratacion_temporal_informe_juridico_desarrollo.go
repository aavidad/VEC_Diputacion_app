package bootstrap

import (
	"context"
	"crypto/hmac"
	"errors"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/informejuridico"
	postgrescontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	seguridadcontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/seguridad"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	finalidadInformeJuridicoContratacionTemporalDesarrollo = "gestionar_contratacion_temporal"
	definicionInformeJuridicoDesarrollo                    = "configuracion:ct:desarrollo:informe-juridico:v1"
	plantillaInformeJuridicoDesarrollo                     = "plantilla:ct:desarrollo:informe-juridico:v1"
)

var errInformeJuridicoContratacionTemporalDesarrolloNoDisponible = errors.New(
	"contratacion temporal: informe juridico de desarrollo no disponible",
)

type resolutorConfiguracionInformeJuridicoContratacionTemporalDesarrollo struct{}

func (resolutorConfiguracionInformeJuridicoContratacionTemporalDesarrollo) ResolverConfiguracionInformeJuridico(
	ctx context.Context,
	solicitud ports.SolicitudResolverConfiguracionInformeJuridico,
) (ports.ConfiguracionInformeJuridico, error) {
	if contextoInterfazNulo(ctx) || solicitud.Validar() != nil ||
		solicitud.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		solicitud.VersionExpediente != 4 ||
		solicitud.FaseActual != domain.ClaveFase("asignacion_unidad") ||
		solicitud.EstadoActual != domain.EstadoEnCurso ||
		solicitud.UnidadAsignadaRef != unidadCoberturaContratacionTemporalDesarrollo {
		return ports.ConfiguracionInformeJuridico{},
			errInformeJuridicoContratacionTemporalDesarrolloNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.ConfiguracionInformeJuridico{}, err
	}
	configuracion := ports.ConfiguracionInformeJuridico{
		Plantilla: domain.ReferenciaPlantillaInformeJuridico{
			PlantillaRef: plantillaInformeJuridicoDesarrollo,
			Version:      1,
			HuellaSHA256: huellaAltaContratacionTemporalDesarrollo("plantilla-informe-juridico"),
		},
		ReferenciasNormativas: []domain.ReferenciaNormativaInformeJuridico{
			{
				NormaRef:     "norma:ct:desarrollo:estatuto-basico-empleado-publico",
				Version:      1,
				HuellaSHA256: huellaAltaContratacionTemporalDesarrollo("norma-ebep-informe-juridico"),
			},
			{
				NormaRef:     "norma:ct:desarrollo:presupuesto-dipgra",
				Version:      1,
				HuellaSHA256: huellaAltaContratacionTemporalDesarrollo("norma-presupuesto-informe-juridico"),
			},
		},
		DefinicionRef:      definicionInformeJuridicoDesarrollo,
		DefinicionVersion:  1,
		DefinicionHuella:   huellaAltaContratacionTemporalDesarrollo("configuracion-informe-juridico"),
		Accion:             domain.AccionEmitirInformeJuridico,
		Finalidad:          domain.ClaveCatalogo(finalidadInformeJuridicoContratacionTemporalDesarrollo),
		UnidadEjecutoraRef: unidadCoberturaContratacionTemporalDesarrollo,
		MotivoAutorizacion: referenciaMotivoAutorizacionInformeJuridicoDesarrollo(),
		EvaluadaEn:         solicitud.Instante,
		ValidaHasta:        solicitud.Instante.Add(5 * time.Minute),
	}
	if configuracion.ValidarPara(solicitud, solicitud.Instante) != nil {
		return ports.ConfiguracionInformeJuridico{},
			errInformeJuridicoContratacionTemporalDesarrolloNoDisponible
	}
	return configuracion, nil
}

func nuevasDependenciasInformeJuridicoContratacionTemporalDesarrollo(
	derivador *derivadorIdentidadOperacionDesarrollo,
	alta *dependenciasAltaContratacionTemporalDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (*application.ServicioInformesJuridicos, error) {
	if derivador == nil || !derivador.valido() || alta == nil ||
		alta.soporte == nil || alta.autorizador == nil ||
		alta.postgresql.ejecucion == nil {
		return nil, errInformeJuridicoContratacionTemporalDesarrolloNoDisponible
	}
	ambitoActivo, ambitosRetenidos, err :=
		configuracionesHMACAltaContratacionTemporalDesarrollo(
			derivador,
			ports.DominioAmbitoIdempotenciaInformeJuridico,
			true,
		)
	if err != nil {
		return nil, errInformeJuridicoContratacionTemporalDesarrolloNoDisponible
	}
	huellaActiva, huellasRetenidas, err :=
		configuracionesHMACAltaContratacionTemporalDesarrollo(
			derivador,
			ports.DominioHuellaPeticionInformeJuridico,
			false,
		)
	if err != nil {
		return nil, errInformeJuridicoContratacionTemporalDesarrolloNoDisponible
	}
	sellos, err := seguridadcontratacion.NuevaAutoridadSellosInformeJuridicoHMAC(
		ambitoActivo, ambitosRetenidos, huellaActiva, huellasRetenidas,
	)
	if err != nil {
		return nil, errInformeJuridicoContratacionTemporalDesarrolloNoDisponible
	}
	referencias := seguridadcontratacion.NuevoGeneradorReferenciasAltaCriptografico()
	preparaciones, err := postgrescontratacion.NuevoPreparadorInformeJuridicoPostgreSQL(
		alta.postgresql.ejecucion,
		referencias,
	)
	if err != nil {
		return nil, errInformeJuridicoContratacionTemporalDesarrolloNoDisponible
	}
	transaccion, err := postgrescontratacion.NuevaTransaccionInformesJuridicosPostgreSQL(
		alta.postgresql.ejecucion,
		alta.postgresql.proveedorMaterial,
	)
	if err != nil {
		return nil, errInformeJuridicoContratacionTemporalDesarrolloNoDisponible
	}
	servicio, err := application.NuevoServicioInformesJuridicos(
		alta.soporte,
		sellos,
		sellos,
		preparaciones,
		resolutorConfiguracionInformeJuridicoContratacionTemporalDesarrollo{},
		seguridadvec.GeneradorReferenciasCriptograficas{},
		alta.autorizador,
		informejuridico.GeneradorDesarrollo{},
		reloj,
		transaccion,
	)
	if err != nil {
		return nil, errInformeJuridicoContratacionTemporalDesarrolloNoDisponible
	}
	return servicio, nil
}

func nuevaInstantaneaAutorizacionInformeJuridicoContratacionTemporalDesarrollo(
	principalID string,
	perfilRef string,
	ahora time.Time,
) (dominiovec.InstantaneaAutorizacion, error) {
	return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(
		principalID,
		perfilRef,
		ahora,
		"tecnico_rrhh_informe_juridico_desarrollo",
		"Tecnico RRHH de informe juridico de desarrollo",
		"informe-juridico-desarrollo-no-autoritativo",
		[]dominiovec.ConcesionRol{{
			Accion:         ports.AccionEmitirInformeJuridico,
			ModuloID:       ports.ModuloContratacion,
			TipoRecurso:    ports.TipoRecursoInformeJuridico,
			Finalidades:    []string{finalidadInformeJuridicoContratacionTemporalDesarrollo},
			GarantiaMinima: dominiovec.AuthAssuranceHigh,
		}},
		[]dominiovec.AmbitoPerfil{
			{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
			{Clave: "expediente_ref", Valores: []string{expedienteContratacionTemporalDesarrolloRef}},
			{Clave: "fase_previa", Valores: []string{"asignacion_unidad"}},
			{Clave: "estado_previo", Valores: []string{string(domain.EstadoEnCurso)}},
		},
	)
}

func referenciaMotivoAutorizacionInformeJuridicoDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_autorizacion_informe_juridico",
		CatalogoVersion:      1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-informe-juridico"),
		EntradaClave: referenciaAltaContratacionTemporalDesarrollo(
			"motivo_",
			"generar-informe-juridico",
		),
	}
}

func solicitudAutorizacionInformeJuridicoContratacionTemporalDesarrolloValida(
	ruta string,
	datos dominiovec.DatosSolicitudAutorizacionLigadaV3,
) bool {
	ambitos := datos.Recurso.Ambitos
	atributos := datos.Recurso.Atributos
	return ruta == "/api/vec/contratacion-temporal/informes-juridicos/preparaciones" &&
		datos.Accion == ports.AccionEmitirInformeJuridico &&
		datos.ReferenciaMotivo == referenciaMotivoAutorizacionInformeJuridicoDesarrollo() &&
		datos.Recurso.ModuloID == ports.ModuloContratacion &&
		datos.Recurso.Tipo == ports.TipoRecursoInformeJuridico &&
		datos.Recurso.Referencia == ambitos["expediente_ref"] &&
		datos.Finalidad == finalidadInformeJuridicoContratacionTemporalDesarrollo &&
		len(ambitos) == 4 && len(atributos) == 10 &&
		ambitos["organizacion_ref"] == organizacionAltaContratacionTemporalDesarrollo &&
		ambitos["fase_previa"] == "asignacion_unidad" &&
		ambitos["estado_previo"] == string(domain.EstadoEnCurso) &&
		atributos["version_expediente"] == strconv.FormatUint(4, 10) &&
		atributos["configuracion_ref"] == definicionInformeJuridicoDesarrollo &&
		atributos["configuracion_version"] == strconv.FormatUint(1, 10) &&
		hmac.Equal(
			[]byte(atributos["configuracion_huella_sha256"]),
			[]byte(huellaAltaContratacionTemporalDesarrollo("configuracion-informe-juridico")),
		) &&
		atributos["plantilla_ref"] == plantillaInformeJuridicoDesarrollo &&
		atributos["plantilla_version"] == strconv.FormatUint(1, 10) &&
		hmac.Equal(
			[]byte(atributos["plantilla_huella_sha256"]),
			[]byte(huellaAltaContratacionTemporalDesarrollo("plantilla-informe-juridico")),
		) &&
		selloHMACAsignacionContratacionTemporalDesarrolloValido(
			atributos["ambito_idempotencia_hmac"],
			ports.DominioAmbitoIdempotenciaInformeJuridico,
		) &&
		selloHMACAsignacionContratacionTemporalDesarrolloValido(
			atributos["huella_peticion_hmac"],
			ports.DominioHuellaPeticionInformeJuridico,
		) &&
		len(atributos["borrador_huella_sha256"]) == 64
}
