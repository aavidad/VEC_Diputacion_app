package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	transaccionbolsa "vec-diputacion-granada/internal/modules/bolsa/internal/transaccion"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func manifiestoBaselineBBolsaPostgreSQLE2E(
	t *testing.T,
	version puertosbolsa.ReferenciaVersionBaremacion,
	contenido dominiobolsa.ContenidoDecisionTecnica,
	firma dominiobolsa.FirmaDecisionTecnica,
	autorizacionPrevalidacionRef string,
	autorizacionConfirmacionRef string,
	indice int,
	ancla time.Time,
) puertosbolsa.ManifiestoProbatorioBaremacion {
	t.Helper()
	evidenciasMerito, err := contenido.CalculoOficial.EvidenciasCanonicas()
	if err != nil || len(evidenciasMerito) != 1 {
		t.Fatalf("el E2E exige exactamente una evidencia de merito: %v", err)
	}
	huellaContenido, err := contenido.HuellaContenidoSHA256()
	if err != nil {
		t.Fatalf("huella contenido para manifiesto E2E: %v", err)
	}
	prefijo := fmt.Sprintf("decision:%d", indice)
	accionAdopcion, existe := puertosbolsa.AccionAdopcionParaClase(contenido.Clase)
	if !existe {
		t.Fatalf("clase de decision sin accion de adopcion: %s", contenido.Clase)
	}
	acciones := []struct {
		accion     puertosbolsa.AccionOperacionBaremacion
		recurso    string
		referencia string
	}{
		{puertosbolsa.AccionConsultarBaremacionVigente, contenido.BaremacionMeritoRef, ""},
		{puertosbolsa.AccionRecuperarCalculoBaremacion, contenido.CalculoOficial.CalculoRef, ""},
		{puertosbolsa.AccionConsultarCriterioBaremacion, contenido.Criterio.ProcesoRef, ""},
		{puertosbolsa.AccionConsultarEvidenciaBaremacion, evidenciasMerito[0].Referencia.DocumentoRef, ""},
		{puertosbolsa.AccionConsultarRepresentacionBaremacion, evidenciasMerito[0].Referencia.RepresentacionRef, ""},
		{accionAdopcion, contenido.BaremacionMeritoRef, contenido.AutorizacionRef},
		{puertosbolsa.AccionConsultarPoliticaFirmaBaremacion, firma.PoliticaFirmaRef, ""},
		{puertosbolsa.AccionCodificarDecisionBaremacion, contenido.ID, ""},
		{puertosbolsa.AccionCustodiarDecisionBaremacion, contenido.ID, ""},
		{puertosbolsa.AccionPrepararFirmaDecisionBaremacion, contenido.ID, ""},
		{puertosbolsa.AccionConsultarFirmaDecisionBaremacion, firma.SesionFirmaInteractivaRef, ""},
		{puertosbolsa.AccionValidarFirmaDecisionBaremacion, firma.FirmaRef, ""},
		{puertosbolsa.AccionRecuperarBinarioFirmadoBaremacion, firma.DocumentoFirmadoRef, ""},
		{puertosbolsa.AccionCustodiarDocumentoFirmadoBaremacion, firma.DocumentoFirmadoRef, ""},
		{puertosbolsa.AccionRetenerDocumentoFirmadoBaremacion, firma.DocumentoFirmadoRef, ""},
		{puertosbolsa.AccionReservarDecisionBaremacion, contenido.BaremacionMeritoRef, ""},
		{puertosbolsa.AccionPrevalidarArchivoProbatorioBaremacion, contenido.BaremacionMeritoRef, autorizacionPrevalidacionRef},
		{puertosbolsa.AccionConfirmarDecisionBaremacion, contenido.BaremacionMeritoRef, autorizacionConfirmacionRef},
	}
	autorizaciones := make([]puertosbolsa.AutorizacionProbatoriaBaremacion, 0, len(acciones))
	for indiceAccion, paso := range acciones {
		clase, existe := puertosbolsa.ClaseRecursoRequeridaOperacionBaremacion(paso.accion)
		if !existe {
			t.Fatalf("accion sin clase en manifiesto E2E: %s", paso.accion)
		}
		referencia := paso.referencia
		if referencia == "" {
			referencia = fmt.Sprintf(
				"autorizacion:manifiesto:e2e:postgresql:v3:%s:%02d",
				prefijo,
				indiceAccion+1,
			)
		}
		autorizaciones = append(autorizaciones, puertosbolsa.AutorizacionProbatoriaBaremacion{
			Secuencia: uint32(indiceAccion + 1), Accion: paso.accion,
			ClaseRecurso: clase, RecursoRef: paso.recurso, AutorizacionRef: referencia,
		})
	}
	evidencias := []puertosbolsa.EvidenciaProbatoriaBaremacion{
		{Tipo: puertosbolsa.EvidenciaEstadoBaseBaremacion, Referencia: contenido.BaremacionMeritoRef, HuellaEvidenciaSHA256: version.HuellaEstadoSHA256},
		{Tipo: puertosbolsa.EvidenciaCalculoOficialBaremacion, Referencia: "evidencia:gobierno:calculo:e2e:postgresql:v3:" + prefijo, HuellaEvidenciaSHA256: huellaSHA256E2E("evidencia gobierno calculo E2E " + prefijo)},
		{Tipo: puertosbolsa.EvidenciaCriterioPublicadoBaremacion, Referencia: contenido.Criterio.ProcesoRef, HuellaEvidenciaSHA256: contenido.Criterio.HuellaSHA256},
		{Tipo: puertosbolsa.EvidenciaDocumentoMeritoBaremacion, Referencia: evidenciasMerito[0].Referencia.DocumentoRef, HuellaEvidenciaSHA256: evidenciasMerito[0].Referencia.HuellaSHA256},
		{Tipo: puertosbolsa.EvidenciaRepresentacionBaremacion, Referencia: evidenciasMerito[0].Referencia.RepresentacionRef, HuellaEvidenciaSHA256: evidenciasMerito[0].Referencia.HuellaSHA256},
		{Tipo: puertosbolsa.EvidenciaContenidoDecisionBaremacion, Referencia: contenido.ID, HuellaEvidenciaSHA256: huellaContenido},
		{Tipo: puertosbolsa.EvidenciaPoliticaFirmaBaremacion, Referencia: "evidencia:politica:firma:e2e:postgresql:v3:" + prefijo, HuellaEvidenciaSHA256: firma.HuellaPoliticaFirmaSHA256},
		{Tipo: puertosbolsa.EvidenciaDocumentoCanonicoBaremacion, Referencia: contenido.ID, HuellaEvidenciaSHA256: firma.HuellaDocumentoFirmableSHA256},
		{Tipo: puertosbolsa.EvidenciaCustodiaFirmableBaremacion, Referencia: firma.EvidenciaCustodiaRef, HuellaEvidenciaSHA256: firma.HuellaDocumentoFirmableSHA256},
		{Tipo: puertosbolsa.EvidenciaPreparacionFirmaBaremacion, Referencia: "evidencia:preparacion:firma:e2e:postgresql:v3:" + prefijo, HuellaEvidenciaSHA256: huellaSHA256E2E("evidencia preparacion firma E2E " + prefijo)},
		{Tipo: puertosbolsa.EvidenciaConsultaFirmaBaremacion, Referencia: "evidencia:consulta:firma:e2e:postgresql:v3:" + prefijo, HuellaEvidenciaSHA256: huellaSHA256E2E("evidencia consulta firma E2E " + prefijo)},
		{Tipo: puertosbolsa.EvidenciaValidacionInicialBaremacion, Referencia: firma.ValidacionInicialFirmaRef, HuellaEvidenciaSHA256: firma.HuellaValidacionInicialSHA256},
		{Tipo: puertosbolsa.EvidenciaValidacionFinalBaremacion, Referencia: firma.ValidacionFirmaRef, HuellaEvidenciaSHA256: firma.HuellaValidacionSHA256},
		{Tipo: puertosbolsa.EvidenciaRecuperacionFirmadoBaremacion, Referencia: firma.EvidenciaRecuperacionFirmadoRef, HuellaEvidenciaSHA256: firma.HuellaEvidenciaRecuperacionSHA256},
		{Tipo: puertosbolsa.EvidenciaCustodiaFirmadoBaremacion, Referencia: firma.EvidenciaCustodiaDocumentoFirmadoRef, HuellaEvidenciaSHA256: firma.HuellaDocumentoSHA256},
		{Tipo: puertosbolsa.EvidenciaRetencionFirmadoBaremacion, Referencia: firma.EvidenciaRetencionDocumentoFirmadoRef, HuellaEvidenciaSHA256: firma.HuellaDocumentoSHA256},
	}
	for indiceEvidencia := range evidencias {
		evidencias[indiceEvidencia].Secuencia = uint32(indiceEvidencia + 1)
	}
	base := puertosbolsa.ManifiestoProbatorioBaremacion{
		Esquema:        puertosbolsa.EsquemaManifiestoProbatorioBaremacion,
		Finalidad:      puertosbolsa.FinalidadManifiestoProbatorioBaremacion,
		VersionEsquema: puertosbolsa.VersionManifiestoProbatorioBaremacion,
		Referencia:     "manifiesto:e2e:postgresql:v3:" + prefijo,
		ProcesoRef:     contenido.ProcesoRef, SolicitudRef: contenido.SolicitudRef,
		SujetoRef: contenido.SujetoRef, BaremacionMeritoRef: contenido.BaremacionMeritoRef,
		DecisionRef: contenido.ID, VersionBase: version.Numero,
		HuellaVersionBaseSHA256: version.HuellaEstadoSHA256,
		Autorizaciones:          autorizaciones, Evidencias: evidencias, CreadoEn: ancla.UTC(),
	}
	preparado, representacion, err := base.PrepararSellado()
	if err != nil {
		t.Fatalf("preparar manifiesto baseline-B 18/16 E2E: %v", err)
	}
	sello, err := selloHMACBolsaPostgreSQLE2E(
		puertosbolsa.FinalidadSelloManifiestoProbatorioBaremacionV3,
		representacion,
	)
	if err != nil {
		t.Fatalf("sellar manifiesto baseline-B E2E: %v", err)
	}
	manifiesto, err := preparado.IncorporarSello(sello)
	if err != nil {
		t.Fatalf("incorporar sello a manifiesto E2E: %v", err)
	}
	if len(manifiesto.Autorizaciones) != 18 || len(manifiesto.Evidencias) != 16 {
		t.Fatalf(
			"cobertura baseline-B inesperada: autorizaciones=%d evidencias=%d",
			len(manifiesto.Autorizaciones), len(manifiesto.Evidencias),
		)
	}
	return manifiesto
}

func selloProvisionalBolsaPostgreSQLE2E(
	t *testing.T,
	finalidad puertosbolsa.FinalidadSelloBaremacion,
	etiqueta string,
) string {
	t.Helper()
	carga, err := puertosbolsa.NuevaCargaProtegida([]byte(
		"preimagen provisional autentica del E2E: " + etiqueta,
	))
	if err != nil {
		t.Fatalf("crear carga provisional para %s: %v", etiqueta, err)
	}
	sello, err := selloHMACBolsaPostgreSQLE2E(finalidad, carga)
	if err != nil {
		t.Fatalf("sellar carga provisional para %s: %v", etiqueta, err)
	}
	return sello
}

func sellarReservaBolsaPostgreSQLE2E(
	t *testing.T,
	solicitud puertosbolsa.SolicitudReservarCambioBaremacion,
) puertosbolsa.SolicitudReservarCambioBaremacion {
	t.Helper()
	solicitud.HuellaSolicitudHMAC = selloProvisionalBolsaPostgreSQLE2E(
		t, puertosbolsa.FinalidadSelloReservaBaremacion, "reserva",
	)
	representacion, err := puertosbolsa.RepresentacionCanonicaReservaBaremacion(solicitud)
	if err != nil {
		t.Fatalf("representar reserva E2E: %v", err)
	}
	solicitud.HuellaSolicitudHMAC, err = selloHMACBolsaPostgreSQLE2E(
		puertosbolsa.FinalidadSelloReservaBaremacion, representacion,
	)
	if err != nil || solicitud.Validar() != nil {
		t.Fatalf("sellar reserva E2E: %v", err)
	}
	return solicitud
}

func sellarConfirmacionBolsaPostgreSQLE2E(
	t *testing.T,
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
) puertosbolsa.SolicitudConfirmarCambioBaremacion {
	t.Helper()
	solicitud.HuellaSolicitudHMAC = selloProvisionalBolsaPostgreSQLE2E(
		t, puertosbolsa.FinalidadSelloConfirmacionBaremacionV2, "confirmacion",
	)
	representacion, err := puertosbolsa.RepresentacionCanonicaConfirmacionBaremacion(solicitud)
	if err != nil {
		t.Fatalf("representar confirmacion E2E: %v", err)
	}
	solicitud.HuellaSolicitudHMAC, err = selloHMACBolsaPostgreSQLE2E(
		puertosbolsa.FinalidadSelloConfirmacionBaremacionV2, representacion,
	)
	if err != nil || solicitud.Validar() != nil {
		t.Fatalf("sellar confirmacion E2E: %v", err)
	}
	return solicitud
}

func reservarCambioDiagnosticadoBolsaPostgreSQLE2E(
	ctx context.Context,
	repositorio *RepositorioBaremaciones,
	admin *pgxpool.Pool,
	solicitud puertosbolsa.SolicitudReservarCambioBaremacion,
) (puertosbolsa.ReservaCambioBaremacion, error) {
	resultado, err := repositorio.ReservarCambio(ctx, solicitud)
	if !errors.Is(err, puertosbolsa.ErrAutorizacionBaremacionInvalida) {
		return resultado, err
	}
	ahoraAplicacion, errorReloj := repositorio.ahora()
	accion := accionReserva(solicitud.Clase)
	clase, existeClase := puertosbolsa.ClaseRecursoRequeridaOperacionBaremacion(accion)
	campos, existenCampos := puertosbolsa.CamposRequeridosOperacionBaremacion(accion)
	errorLocal := solicitud.Contexto.ValidarVigentePara(
		accion, puertosbolsa.ClaseRecursoBaremacion,
		solicitud.BaremacionMeritoRef, ahoraAplicacion,
	)
	huellaEfecto, errorHuella := transaccionbolsa.HuellaEfectoReserva(solicitud)
	prueba, decisionCanonica, recursoCanonico, errorPrueba := serializarPruebaYRecurso(
		solicitud.Contexto, ahoraAplicacion, huellaEfecto,
	)
	camposJSON, errorCampos := json.Marshal(campos)
	if errorReloj != nil || !existeClase || !existenCampos || errorHuella != nil ||
		errorPrueba != nil || errorCampos != nil {
		return resultado, fmt.Errorf(
			"%w; diagnostico local incompleto reloj=%v contexto=%v clase=%t campos=%t huella=%v prueba=%v json=%v",
			err, errorReloj, errorLocal, existeClase, existenCampos,
			errorHuella, errorPrueba, errorCampos,
		)
	}
	defer borrarBytesPostgreSQL(prueba, decisionCanonica, recursoCanonico, camposJSON)
	var atestacionValida, rbacValido bool
	var ahoraPostgreSQL time.Time
	errorDiagnostico := admin.QueryRow(ctx, `
		WITH reloj AS (SELECT clock_timestamp() AS ahora)
		SELECT
		  EXISTS (
		    SELECT 1
		      FROM reloj,
		           LATERAL vec_bolsa_baremacion.obtener_atestacion_actual_valida(
		             $1::jsonb, $2::bytea, reloj.ahora
		           )
		  ),
		  vec_autorizacion.revalidar_decision_bolsa_baremacion_v1(
		    $1::jsonb, $2::bytea, $3::bytea, $4::text, $5::text,
		    $6::text, $7::jsonb, reloj.ahora
		  ),
		  reloj.ahora
		FROM reloj`,
		prueba, decisionCanonica, recursoCanonico, string(accion), string(clase),
		solicitud.BaremacionMeritoRef, camposJSON,
	).Scan(&atestacionValida, &rbacValido, &ahoraPostgreSQL)
	detalleRBAC := []byte(`{}`)
	detalleCanonico := []byte(`{}`)
	if errorDiagnostico == nil && !rbacValido {
		errorDetalle := admin.QueryRow(ctx, `
			WITH
			  reloj AS (SELECT clock_timestamp() AS ahora),
			  entrada AS (
			    SELECT $1::jsonb AS prueba,
			           convert_from($2::bytea, 'UTF8')::jsonb AS canonica,
			           $3::bytea AS recurso, $4::text AS accion,
			           $5::text AS clase, $6::text AS recurso_ref,
			           $7::jsonb AS campos
			  ),
			  decision AS (
			    SELECT d.*
			      FROM vec_autorizacion.decision_autorizacion AS d, entrada AS e
			     WHERE d.decision_ref=e.prueba ->> 'decision_ref'
			  ),
			  politicas AS (
			    SELECT COALESCE(jsonb_object_agg(
			             p.politica_ref, p.huella_sha256 ORDER BY p.politica_ref
			           ), '{}'::jsonb) AS manifiesto,
			           COALESCE(jsonb_agg(
			             p.politica_ref ORDER BY p.politica_ref
			           ), '[]'::jsonb) AS referencias
			      FROM vec_autorizacion.politica_restrictiva_actual AS a
			      JOIN vec_autorizacion.politica_restrictiva AS p
			        ON p.politica_id=a.politica_id AND p.politica_ref=a.politica_ref
			  )
			SELECT jsonb_build_object(
			  'decision_existe', EXISTS (SELECT 1 FROM decision),
			  'decision_basica', COALESCE((
			    SELECT d.concedida AND d.codigo='concedida'
			       AND d.accion=e.accion AND d.modulo_id='bolsa'
			       AND d.tipo_recurso=e.clase AND d.recurso_ref=e.recurso_ref
			       AND encode(sha256(e.recurso), 'hex')=d.contexto_recurso_huella_sha256
			       AND d.documento -> 'obligaciones'='[]'::jsonb
			       AND d.emitida_en <= (e.prueba ->> 'verificada_en')::timestamptz
			       AND (e.prueba ->> 'verificada_en')::timestamptz < d.valida_hasta
			       AND r.ahora >= (e.prueba ->> 'verificada_en')::timestamptz
			       AND r.ahora-(e.prueba ->> 'verificada_en')::timestamptz <= interval '30 seconds'
			       AND r.ahora >= d.emitida_en AND r.ahora < d.valida_hasta
			      FROM decision AS d, entrada AS e, reloj AS r
			  ), false),
			  'canonica_30', COALESCE((
			    SELECT jsonb_typeof(e.canonica)='object'
			       AND (SELECT count(*) FROM jsonb_object_keys(e.canonica))=30
			      FROM entrada AS e
			  ), false),
			  'campos_exactos', COALESCE((
			    SELECT (SELECT COALESCE(jsonb_agg(v ORDER BY v), '[]'::jsonb)
			              FROM jsonb_array_elements_text(d.documento -> 'campos_permitidos') AS v)
			         = (SELECT COALESCE(jsonb_agg(v ORDER BY v), '[]'::jsonb)
			              FROM jsonb_array_elements_text(e.campos) AS v)
			      FROM decision AS d, entrada AS e
			  ), false),
			  'asignacion', EXISTS (
			    SELECT 1 FROM decision AS d
			      JOIN vec_autorizacion.asignacion_perfil_actual AS a
			        ON a.perfil_activo_ref=d.perfil_activo_ref
			      JOIN vec_autorizacion.asignacion_perfil AS p
			        ON p.perfil_activo_ref=a.perfil_activo_ref
			       AND p.asignacion_ref=a.asignacion_ref
			     WHERE p.asignacion_ref=d.asignacion_ref
			       AND p.principal_id=d.principal_id
			       AND p.version_rol_ref=d.version_rol_ref
			       AND p.huella_sha256=d.asignacion_huella_sha256
			  ),
			  'rol_control', EXISTS (
			    SELECT 1 FROM decision AS d
			      JOIN vec_autorizacion.version_rol AS v
			        ON v.version_rol_ref=d.version_rol_ref
			      JOIN vec_autorizacion.control_vigencia_version_rol_actual AS a
			        ON a.version_rol_ref=v.version_rol_ref
			      JOIN vec_autorizacion.control_vigencia_version_rol AS c
			        ON c.version_rol_ref=a.version_rol_ref AND c.revision=a.revision
			     WHERE v.huella_sha256=d.version_rol_huella_sha256
			       AND c.version_rol_ref=d.control_vigencia_version_rol_ref
			       AND c.revision=d.control_vigencia_version_rol_revision
			       AND c.estado='habilitada'
			       AND c.huella_sha256=d.control_vigencia_version_rol_huella_sha256
			  ),
			  'concesiones', COALESCE((
			    SELECT count(*) FROM decision AS d
			      JOIN vec_autorizacion.version_rol AS v
			        ON v.version_rol_ref=d.version_rol_ref
			      CROSS JOIN LATERAL jsonb_array_elements(v.documento -> 'concesiones') AS c
			      CROSS JOIN entrada AS e
			     WHERE c ->> 'accion'=d.accion AND c ->> 'modulo_id'='bolsa'
			       AND c ->> 'tipo_recurso'=d.tipo_recurso
			       AND COALESCE(c -> 'obligaciones', '[]'::jsonb)='[]'::jsonb
			       AND (SELECT COALESCE(jsonb_agg(x ORDER BY x), '[]'::jsonb)
			              FROM jsonb_array_elements_text(
			                COALESCE(c -> 'campos_permitidos', '[]'::jsonb)
			              ) AS x)
			           = (SELECT COALESCE(jsonb_agg(x ORDER BY x), '[]'::jsonb)
			                FROM jsonb_array_elements_text(e.campos) AS x)
			       AND EXISTS (SELECT 1 FROM jsonb_array_elements_text(c -> 'finalidades') AS f
			                    WHERE f=d.finalidad)
			  ), -1),
			  'catalogo', EXISTS (
			    SELECT 1 FROM decision AS d
			      JOIN vec_autorizacion.control_catalogo_politicas AS c ON c.control_id=true
			     WHERE c.revision=d.revision_catalogo_politicas
			       AND c.huella_sha256=d.catalogo_politicas_huella_sha256
			  ),
			  'politicas', COALESCE((
			    SELECT p.manifiesto=d.politicas_evaluadas_manifesto
			       AND p.referencias=(SELECT COALESCE(jsonb_agg(x ORDER BY x), '[]'::jsonb)
			                              FROM jsonb_array_elements_text(
			                                d.documento -> 'politicas_evaluadas_refs'
			                              ) AS x)
			      FROM decision AS d, politicas AS p
			  ), false),
			  'vinculo', COALESCE((
			    SELECT vec_autorizacion.revalidar_vinculo_autenticacion_actor_actual_v1(
			      d.documento -> 'vinculo_autenticacion_actor', d.principal_id,
			      d.perfil_activo_ref,
			      d.documento -> 'vinculo_autenticacion_actor' ->> 'contexto_actor_huella_sha256',
			      d.emitida_en, d.valida_hasta, r.ahora
			    ) FROM decision AS d, reloj AS r
			  ), false)
			) FROM reloj`,
			prueba, decisionCanonica, recursoCanonico, string(accion), string(clase),
			solicitud.BaremacionMeritoRef, camposJSON,
		).Scan(&detalleRBAC)
		if errorDetalle != nil {
			detalleRBAC = []byte(fmt.Sprintf(`{"error_detalle":%q}`, errorDetalle.Error()))
		}
		errorCanonico := admin.QueryRow(ctx, `
			WITH
			  entrada AS (
			    SELECT $1::jsonb AS prueba,
			           convert_from($2::bytea, 'UTF8')::jsonb AS canonica
			  ),
			  decision AS (
			    SELECT d.* FROM vec_autorizacion.decision_autorizacion AS d, entrada AS e
			     WHERE d.decision_ref=e.prueba ->> 'decision_ref'
			  ),
			  campos AS (
			    SELECT COALESCE(jsonb_agg(v ORDER BY v), '[]'::jsonb) AS valor
			      FROM decision AS d,
			           LATERAL jsonb_array_elements_text(d.documento -> 'campos_permitidos') AS v
			  ),
			  obligaciones AS (
			    SELECT COALESCE(jsonb_agg(v ORDER BY v), '[]'::jsonb) AS valor
			      FROM decision AS d,
			           LATERAL jsonb_array_elements_text(d.documento -> 'obligaciones') AS v
			  ),
			  evaluadas AS (
			    SELECT COALESCE(jsonb_agg(jsonb_build_object(
			             'referencia', r,
			             'huella_sha256', d.documento -> 'politicas_evaluadas_huellas_sha256' ->> r
			           ) ORDER BY r), '[]'::jsonb) AS valor
			      FROM decision AS d,
			           LATERAL jsonb_array_elements_text(d.documento -> 'politicas_evaluadas_refs') AS r
			  ),
			  aplicables AS (
			    SELECT COALESCE(jsonb_agg(jsonb_build_object(
			             'referencia', r,
			             'huella_sha256', d.documento -> 'politicas_huellas_sha256' ->> r
			           ) ORDER BY r), '[]'::jsonb) AS valor
			      FROM decision AS d,
			           LATERAL jsonb_array_elements_text(d.documento -> 'politicas_refs') AS r
			  ),
			  esperado AS (
			    SELECT jsonb_build_object(
			      'esquema', 'vec.autorizacion.decision.reforzada.v1.autenticacion-actor',
			      'decision_ref', d.documento -> 'decision_ref',
			      'concedida', d.documento -> 'concedida',
			      'codigo', d.documento -> 'codigo',
			      'principal_id', d.documento -> 'principal_id',
			      'perfil_activo_ref', d.documento -> 'perfil_activo_ref',
			      'accion', d.documento -> 'accion',
			      'recurso_ref', d.documento -> 'recurso_ref',
			      'modulo_id', d.documento -> 'modulo_id',
			      'tipo_recurso', d.documento -> 'tipo_recurso',
			      'contexto_recurso_huella_sha256', d.documento -> 'contexto_recurso_huella_sha256',
			      'finalidad', d.documento -> 'finalidad',
			      'correlacion_ref', d.documento -> 'correlacion_ref',
			      'vinculo_autenticacion_actor', d.documento -> 'vinculo_autenticacion_actor',
			      'asignacion_ref', d.documento -> 'asignacion_ref',
			      'asignacion_huella_sha256', d.documento -> 'asignacion_huella_sha256',
			      'version_rol_ref', d.documento -> 'version_rol_ref',
			      'version_rol_huella_sha256', d.documento -> 'version_rol_huella_sha256',
			      'control_vigencia_version_rol_ref', d.documento -> 'control_vigencia_version_rol_ref',
			      'control_vigencia_version_rol_revision', d.documento -> 'control_vigencia_version_rol_revision',
			      'control_vigencia_version_rol_huella_sha256', d.documento -> 'control_vigencia_version_rol_huella_sha256',
			      'revision_catalogo_politicas', d.documento -> 'revision_catalogo_politicas',
			      'catalogo_politicas_huella_sha256', d.documento -> 'catalogo_politicas_huella_sha256',
			      'politicas_evaluadas', e.valor,
			      'politicas_aplicables', a.valor,
			      'garantia_minima', d.documento -> 'garantia_minima',
			      'campos_permitidos', c.valor,
			      'obligaciones', o.valor,
			      'emitida_en', d.documento -> 'emitida_en',
			      'valida_hasta', d.documento -> 'valida_hasta'
			    ) AS valor
			      FROM decision AS d, campos AS c, obligaciones AS o,
			           evaluadas AS e, aplicables AS a
			  )
			SELECT jsonb_build_object(
			  'canonica_exacta', i.canonica=esperado.valor,
			  'diferencias', (
			    SELECT COALESCE(jsonb_agg(k ORDER BY k), '[]'::jsonb)
			      FROM jsonb_object_keys(i.canonica || esperado.valor) AS k
			     WHERE i.canonica -> k IS DISTINCT FROM esperado.valor -> k
			  ),
			  'emitida_canonica', i.canonica -> 'emitida_en',
			  'emitida_documento', esperado.valor -> 'emitida_en',
			  'valida_canonica', i.canonica -> 'valida_hasta',
			  'valida_documento', esperado.valor -> 'valida_hasta'
			) FROM entrada AS i, esperado`,
			prueba, decisionCanonica,
		).Scan(&detalleCanonico)
		if errorCanonico != nil {
			detalleCanonico = []byte(fmt.Sprintf(`{"error_canonico":%q}`, errorCanonico.Error()))
		}
	}
	defer borrarBytesPostgreSQL(detalleRBAC, detalleCanonico)
	return resultado, fmt.Errorf(
		"%w; diagnostico contexto_local=%v atestacion_pdp=%t rbac=%t reloj_app=%s reloj_pg=%s consulta=%v detalle_rbac=%s detalle_canonico=%s",
		err, errorLocal, atestacionValida, rbacValido,
		ahoraAplicacion.UTC().Format(time.RFC3339Nano),
		ahoraPostgreSQL.UTC().Format(time.RFC3339Nano), errorDiagnostico,
		detalleRBAC, detalleCanonico,
	)
}

func referenciaDecisionAutorizacionBolsaPostgreSQLE2E(nombre string) string {
	return "decision:autorizacion:e2e:postgresql:v3:" + nombre
}
