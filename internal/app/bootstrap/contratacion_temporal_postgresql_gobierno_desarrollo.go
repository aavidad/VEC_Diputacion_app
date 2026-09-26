package bootstrap

import (
	"context"
	"crypto/ed25519"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"slices"
	"time"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	postgrescontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	cronosapp "vec-diputacion-granada/internal/modules/cronos/application"
	altapersonal "vec-diputacion-granada/internal/modules/personal/adapters/contrataciontemporal"
	lecturapersonal "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	confianzaatestacion "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

func prepararRotacionGobiernoPostgreSQLContratacionTemporalDesarrollo(
	ctx context.Context,
	tx pgx.Tx,
	material *materialAtestacionContratacionTemporalDesarrollo,
) error {
	if ctx == nil || tx == nil || material == nil {
		return errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	propio, err := gobiernoActualPostgreSQLContratacionTemporalDesarrolloEsPropio(ctx, tx)
	if err != nil {
		return errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	if !propio {
		return errGobiernoPostgreSQLContratacionTemporalDesarrolloAjeno
	}

	claveHMACNueva := false
	var claveHMACVersion, claveHMACRevision int64
	err = tx.QueryRow(ctx, `
		SELECT clave_id,version,revision_gobierno
		  FROM vec_autorizacion_atestada_v3.clave_capacidad_version
		 WHERE huella_secreto_sha256=$1 AND secreto_hmac=$2
		   AND huella_gobierno_sha256=$3 AND emisor_id=$4
		   AND audiencia_consumo=$5 AND valida_desde=$6 AND valida_hasta=$7
		   AND pg_catalog.left(acto_ref,
		       pg_catalog.length('acto:ct:desarrollo:clave-capacidad:'))=
		       'acto:ct:desarrollo:clave-capacidad:'`,
		material.claveHMACSecreto, material.claveHMAC, material.claveHMACHuella,
		material.emisorID, material.audienciaConsumo,
		material.validaDesde, material.validaHasta,
	).Scan(&material.claveHMACID, &claveHMACVersion, &claveHMACRevision)
	if errors.Is(err, pgx.ErrNoRows) {
		claveHMACNueva = true
	} else if err != nil {
		return errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}

	raizNueva := false
	var raizVersion int64
	err = tx.QueryRow(ctx, `
		SELECT clave_id,version
		  FROM vec_autorizacion_atestada_v3.raiz_confianza_version
		 WHERE huella_spki_sha256=$1 AND clave_publica_spki=$2
		   AND valida_desde=$3 AND valida_hasta=$4 AND suite=$5
		   AND audiencia_despliegue=$6
		   AND pg_catalog.left(acto_ref,
		       pg_catalog.length('acto:ct:desarrollo:raiz-atestacion:'))=
		       'acto:ct:desarrollo:raiz-atestacion:'`,
		material.spkiHuella, material.spki, material.validaDesde, material.validaHasta,
		confianzaatestacion.SuiteAtestacionAutorizacionV3COSEEdDSA,
		audienciaAtestacionContratacionTemporalDesarrollo,
	).Scan(&material.claveID, &raizVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		raizNueva = true
	} else if err != nil {
		return errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}

	var siguiente int64
	if claveHMACNueva || raizNueva {
		err = tx.QueryRow(ctx, `
			SELECT GREATEST(
			 COALESCE((SELECT max(revision_gobierno) FROM
			  vec_autorizacion_atestada_v3.clave_capacidad_version),0),
			 COALESCE((SELECT max(version) FROM
			  vec_autorizacion_atestada_v3.clave_capacidad_version),0),
			 COALESCE((SELECT max(orden) FROM
			  vec_autorizacion_atestada_v3.puntero_clave_emision),0),
			 COALESCE((SELECT max(version) FROM
			  vec_autorizacion_atestada_v3.raiz_confianza_version),0)) + 1`,
		).Scan(&siguiente)
		if err != nil || siguiente < 1 ||
			siguiente > maximoVersionGobiernoPostgreSQLContratacionTemporalDesarrollo {
			return errGobiernoPostgreSQLContratacionTemporalDesarrolloAgotado
		}
	}
	if claveHMACNueva {
		claveHMACVersion = siguiente
		claveHMACRevision = siguiente
		material.claveHMACOrden = uint64(siguiente)
	} else {
		var orden int64
		err = tx.QueryRow(ctx, `
			SELECT orden FROM vec_autorizacion_atestada_v3.puntero_clave_emision
			 WHERE clave_id=$1 AND version=$2
			   AND orden=(SELECT max(orden) FROM
			    vec_autorizacion_atestada_v3.puntero_clave_emision p
			    JOIN vec_autorizacion_atestada_v3.clave_capacidad_version k
			      ON (k.clave_id,k.version)=(p.clave_id,p.version)
			    WHERE p.establecida_en <= pg_catalog.statement_timestamp()
			      AND k.audiencia_consumo=$3)`,
			material.claveHMACID, claveHMACVersion, material.audienciaConsumo,
		).Scan(&orden)
		if err != nil {
			return errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
		}
		material.claveHMACOrden = uint64(orden)
	}
	if raizNueva {
		raizVersion = siguiente
	}
	if claveHMACVersion < 1 || claveHMACRevision < 1 || raizVersion < 1 {
		return errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	material.claveHMACVersion = uint64(claveHMACVersion)
	material.claveHMACRevision = uint64(claveHMACRevision)
	material.claveVersion = uint64(raizVersion)
	if err := reconstruirClavesGobiernoPostgreSQLContratacionTemporalDesarrollo(material); err != nil {
		return err
	}
	return prepararConfiguracionGobiernoPostgreSQLContratacionTemporalDesarrollo(ctx, tx, material)
}

func gobiernoActualPostgreSQLContratacionTemporalDesarrolloEsPropio(
	ctx context.Context,
	tx pgx.Tx,
) (bool, error) {
	var punterosClave, punterosConfiguracion int64
	if err := tx.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM vec_autorizacion_atestada_v3.puntero_clave_emision),
		       (SELECT count(*) FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual)`,
	).Scan(&punterosClave, &punterosConfiguracion); err != nil {
		return false, err
	}
	if punterosClave == 0 && punterosConfiguracion == 0 {
		return true, nil
	}
	if punterosClave == 0 || punterosConfiguracion == 0 {
		return false, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	var propio bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS (
		 SELECT 1
		   FROM vec_autorizacion_atestada_v3.puntero_clave_emision p
		   JOIN vec_autorizacion_atestada_v3.clave_capacidad_version c
		     ON (c.clave_id,c.version)=(p.clave_id,p.version)
		  WHERE p.orden=(SELECT max(orden) FROM
		         vec_autorizacion_atestada_v3.puntero_clave_emision
		         WHERE establecida_en <= pg_catalog.statement_timestamp())
		    AND pg_catalog.left(p.acto_ref,
		        pg_catalog.length('acto:ct:desarrollo:puntero-clave:'))=
		        'acto:ct:desarrollo:puntero-clave:'
		    AND pg_catalog.left(c.acto_ref,
		        pg_catalog.length('acto:ct:desarrollo:clave-capacidad:'))=
		        'acto:ct:desarrollo:clave-capacidad:'
		    AND c.audiencia_consumo = ANY($2::text[]))
		AND EXISTS (
		 SELECT 1
		   FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p
		   JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version c
		     ON c.revision=p.configuracion_revision
		   JOIN vec_autorizacion_atestada_v3.configuracion_raiz cr
		     ON cr.configuracion_revision=c.revision
		   JOIN vec_autorizacion_atestada_v3.raiz_confianza_version r
		     ON (r.clave_id,r.version)=(cr.raiz_clave_id,cr.raiz_version)
		  WHERE p.orden=(SELECT max(orden) FROM
		         vec_autorizacion_atestada_v3.puntero_configuracion_actual
		         WHERE establecida_en <= pg_catalog.statement_timestamp())
		    AND pg_catalog.left(p.acto_ref,
		        pg_catalog.length('acto:ct:desarrollo:puntero-configuracion:'))=
		        'acto:ct:desarrollo:puntero-configuracion:'
		    AND pg_catalog.left(c.acto_ref,
		        pg_catalog.length('acto:ct:desarrollo:configuracion:'))=
		        'acto:ct:desarrollo:configuracion:'
		    AND pg_catalog.left(r.acto_ref,
		        pg_catalog.length('acto:ct:desarrollo:raiz-atestacion:'))=
		        'acto:ct:desarrollo:raiz-atestacion:'
		    AND r.audiencia_despliegue=$1)`,
		audienciaAtestacionContratacionTemporalDesarrollo,
		audienciasConsumoGobiernoCTDesarrollo(),
	).Scan(&propio)
	return propio, err
}

// audienciasConsumoGobiernoCTDesarrollo es la lista cerrada y única de
// consumidores cuya clave de capacidad publica vec-server bajo la raíz de CT.
// La consultan el publicador, antes de escribir, y la comprobación de gobierno
// propio, sobre el puntero de emisión vigente. Mantener dos listas hizo que un
// consumidor publicable (Cronos resolución y notificaciones) dejara, en la
// publicación siguiente, el gobierno como «ajeno» y tumbara el arranque. Cada
// audiencia es nominal y la admite su migración AD3; no hay comodines.
func audienciasConsumoGobiernoCTDesarrollo() []string {
	return []string{
		audienciaConsumoAltaContratacionTemporal,
		puertosbolsa.AudienciaIntegracionLlamamientoDesarrollo,
		ports.AudienciaConsumoConsultaCuadroRRHHV3,
		ports.AudienciaConsumoConsultaDetalleRRHHV3,
		altapersonal.AudienciaAltaEjercicio,
		lecturapersonal.AudienciaV2,
		ports.AudienciaConfirmacionIncorporacionV2,
		postgrescontratacion.AudienciaAnotacionAdministrativaV1,
		postgrescontratacion.AudienciaCierreAdministrativoSinCese,
		ports.AudienciaConsumoCeseV1,
		ports.AudienciaConsumoCierreExpedienteV1,
		ports.AudienciaConsumoModificacionNombramientoV1,
		// Cancelación del expediente (AD3-87); solo con VEC_CT_CANCELACION_ENABLED.
		ports.AudienciaConsumoCancelacionV1,
		// Confirmación de GINPIX (AD3-88); solo se publica con
		// VEC_CT_INCORPORACION_ACREDITADA_ENABLED.
		ports.AudienciaConsumoConfirmacionGINPIXV1,
		ctapplication.AudienciaDespachoCorreoLlamamientoV3,
		ctapplication.AudienciaResultadoCorreoLlamamientoV3,
		// Registro de firmas de prueba de los borradores (AD3-85); solo se
		// publica con VEC_CT_FIRMA_REGISTRO_ENABLED.
		ports.AudienciaFirmaDocumentoV3,
		puertosbolsa.AudienciaCrearBorradorLlamamientoInterno,
		puertosbolsa.AudienciaConsultarBorradorLlamamientoInterno,
		puertosbolsa.AudienciaCambiarSituacionParticipacion,
		puertosbolsa.AudienciaRegistrarContactoParticipacion,
		puertosbolsa.AudienciaConsultarContactoParticipacion,
		puertosbolsa.AudienciaRegistrarDatosContactoParticipacion,
		puertosbolsa.AudienciaEmitirLlamamiento,
		puertosbolsa.AudienciaSolicitarPausaPropia,
		puertosbolsa.AudienciaSolicitarReactivacionPropia,
		puertosbolsa.AudienciaResponderLlamamientoPropio,
		puertosbolsa.AudienciaManifestarDisposicionPropia,
		// Confirmación del contacto propio (AD3-86); solo con el portal.
		puertosbolsa.AudienciaConfirmarContactoPropio,
		audienciaConsumoPersonalDietasDesarrollo,
		audienciaConsumoCrearDietasDesarrollo,
		audienciaConsumoConsultarDietasDesarrollo,
		audienciaConsumoEditarDietasDesarrollo,
		audienciaConsumoBorrarDietasDesarrollo,
		audienciaConsumoEnviarDietasDesarrollo,
		audienciaConsumoDocumentoDietasDesarrollo,
		audienciaConsumoConsultarAsignacionDietas,
		audienciaConsumoRegistrarAsignacionDietas,
		audienciaConsumoCorregirAsignacionDietas,
		audienciaConsumoCorregirGrupoDietas,
		audienciaConsumoRevisarDietas,
		audienciaConsumoAutorizarDietas,
		audienciaConsumoLiquidarDietas,
		audienciaConsumoFiscalizarDietas,
		audienciaConsumoBandejaRevisionDietas,
		audienciaConsumoBandejaAutorizacionDietas,
		audienciaConsumoBandejaLiquidacionDietas,
		audienciaConsumoBandejaFiscalizacionDietas,
		audienciaConsumoRevisorDocumentoDietas,
		cronosapp.AudienciaMarcajePropio,
		cronosapp.AudienciaDisponibilidadMarcajeRemoto,
		cronosapp.AudienciaRecuperacionMarcajeRemoto,
		cronosapp.AudienciaConsultaSaldoPropio,
		cronosapp.AudienciaConsultaMovimientosPropios,
		cronosapp.AudienciaSolicitudCorreccionPropia,
		cronosapp.AudienciaConsultaPermisosPropios,
		cronosapp.AudienciaSolicitudPermisoPropio,
		cronosapp.AudienciaBandejaPermisos,
		cronosapp.AudienciaResolucionPermiso,
		cronosapp.AudienciaConsultaAvisosPropios,
		cronosapp.AudienciaArchivoAvisoPropio,
		cronosapp.AudienciaRegistroNotificacion,
		cronosapp.AudienciaConsultaNotificacionesPropias,
		cronosapp.AudienciaBandejaNotificaciones,
		cronosapp.AudienciaAtencionNotificacion,
		personal.AudienciaFichaEmpleadoB2,
		personal.AudienciaVacantesB2,
		personal.AudienciaAltaEmpleadoB2,
		personal.AudienciaHechoEmpleadoB2,
		personal.AudienciaConsultarCatalogoEmpleadoB2,
		personal.AudienciaPublicarCatalogoEmpleadoB2,
		personal.AudienciaRetirarCatalogoEmpleadoB2,
		personal.AudienciaEmpleadosB2,
		// Consumidores del catálogo común sin entrada previa en la lista:
		// Mi bolsa (AD3-43), Documentos (AD3-60) y ficha propia (AD3-74).
		puertosbolsa.AudienciaMiBolsa,
		docports.AudienciaV3,
		personal.AudienciaFichaPropia,
	}
}

// Las audiencias publicables se mantienen nominales: extender el gobierno de
// anotación y cierre no concede una capacidad a otro consumidor.
func audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(
	audiencia string,
) bool {
	return slices.Contains(audienciasConsumoGobiernoCTDesarrollo(), audiencia)
}

func reconstruirClavesGobiernoPostgreSQLContratacionTemporalDesarrollo(
	material *materialAtestacionContratacionTemporalDesarrollo,
) error {
	if material == nil || len(material.privada) != ed25519.PrivateKeySize ||
		len(material.claveHMAC) == 0 {
		return errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	publica := material.privada.Public().(ed25519.PublicKey)
	raiz, err := confianzaatestacion.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(
		material.claveID, material.claveVersion, publica,
		audienciaAtestacionContratacionTemporalDesarrollo,
		confianzaatestacion.EstadoClaveAtestacionAutorizacionV3Activa,
		material.validaDesde, material.validaHasta, time.Time{},
	)
	if err != nil {
		return errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	capacidad, err := confianzaatestacion.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(
		material.claveHMACID, material.claveHMACVersion, material.claveHMAC,
		material.emisorID, material.audienciaConsumo,
		confianzaatestacion.EstadoClaveHMACCapacidadAtestacionV3Emision,
		material.validaDesde, material.validaHasta, time.Time{},
		material.claveHMACRevision, material.claveHMACHuella,
	)
	if err != nil {
		return errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	material.raiz = raiz
	material.capacidad = capacidad
	return nil
}

func prepararConfiguracionGobiernoPostgreSQLContratacionTemporalDesarrollo(
	ctx context.Context,
	tx pgx.Tx,
	material *materialAtestacionContratacionTemporalDesarrollo,
) error {
	var revision, huella string
	var secuencia int64
	err := tx.QueryRow(ctx, `
		SELECT c.revision,c.secuencia,c.huella_configuracion_sha256
		  FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p
		  JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version c
		    ON c.revision=p.configuracion_revision
		  JOIN vec_autorizacion_atestada_v3.configuracion_raiz cr
		    ON cr.configuracion_revision=c.revision
		 WHERE p.orden=(SELECT max(orden) FROM
		        vec_autorizacion_atestada_v3.puntero_configuracion_actual
		        WHERE establecida_en <= pg_catalog.statement_timestamp())
		   AND cr.raiz_clave_id=$1 AND cr.raiz_version=$2
		   AND c.publicada_en=$3 AND c.expira_en=$4
		   AND pg_catalog.left(p.acto_ref,
		       pg_catalog.length('acto:ct:desarrollo:puntero-configuracion:'))=
		       'acto:ct:desarrollo:puntero-configuracion:'
		   AND pg_catalog.left(c.acto_ref,
		       pg_catalog.length('acto:ct:desarrollo:configuracion:'))=
		       'acto:ct:desarrollo:configuracion:'`,
		material.claveID, material.claveVersion,
		material.publicadaEn, material.expiraEn,
	).Scan(&revision, &secuencia, &huella)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			SELECT GREATEST(
			 COALESCE((SELECT max(secuencia) FROM
			  vec_autorizacion_atestada_v3.configuracion_confianza_version),0),
			 COALESCE((SELECT max(orden) FROM
			  vec_autorizacion_atestada_v3.puntero_configuracion_actual),0))`,
		).Scan(&secuencia)
		if err != nil {
			return errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
		}
		base := int64(material.configuracionOrden)
		if secuencia >= base {
			secuencia++
		} else {
			secuencia = base
		}
		if secuencia < 1 || secuencia > maximoVersionGobiernoPostgreSQLContratacionTemporalDesarrollo {
			return errGobiernoPostgreSQLContratacionTemporalDesarrolloAgotado
		}
		revision = "confianza:atestacion:ct:desarrollo:" +
			material.publicadaEn.Format("2006-01-02")
		var ocupada bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS (
			SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_confianza_version
			 WHERE revision=$1)`, revision).Scan(&ocupada); err != nil {
			return errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
		}
		if ocupada {
			revision += ":r" + numeroDecimal64(uint64(secuencia))
		}
	} else if err != nil {
		return errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	configuracion, err := confianzaatestacion.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(
		revision, uint64(secuencia), material.publicadaEn, material.expiraEn, material.raiz,
	)
	if err != nil {
		return errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	huellaCalculada, err := configuracion.HuellaSHA256ParaGobierno()
	if err != nil || (huella != "" && huella != huellaCalculada) {
		return errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	material.configuracion = configuracion
	material.configuracionRef = revision
	material.configuracionOrden = uint64(secuencia)
	material.configuracionHuella = huellaCalculada
	return nil
}

func numeroDecimal64(valor uint64) string {
	const digitos = "0123456789"
	if valor == 0 {
		return "0"
	}
	var buffer [20]byte
	indice := len(buffer)
	for valor > 0 {
		indice--
		buffer[indice] = digitos[valor%10]
		valor /= 10
	}
	return string(buffer[indice:])
}

func publicarGobiernoAtestacionContratacionTemporalDesarrollo(
	ctx context.Context,
	pool *pgxpool.Pool,
	material *materialAtestacionContratacionTemporalDesarrollo,
) error {
	if ctx == nil || pool == nil || material == nil ||
		len(material.claveHMAC) == 0 || len(material.spki) == 0 ||
		!audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(
			material.audienciaConsumo,
		) {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	return ejecutarTransaccionGobiernoCTDesarrollo(ctx, pool, func(tx pgx.Tx) error {
		return publicarGobiernoAtestacionCTEnTxDesarrollo(ctx, tx, material)
	})
}

func publicarGobiernoAtestacionCTEnTxDesarrollo(ctx context.Context, tx pgx.Tx, material *materialAtestacionContratacionTemporalDesarrollo) error {
	if err := prepararRotacionGobiernoPostgreSQLContratacionTemporalDesarrollo(
		ctx, tx, material,
	); err != nil {
		return err
	}
	actoClaveHMAC := "acto:ct:desarrollo:clave-capacidad:r" +
		numeroDecimal64(material.claveHMACRevision)
	actoPunteroClave := "acto:ct:desarrollo:puntero-clave:r" +
		numeroDecimal64(material.claveHMACOrden)
	actoRaiz := "acto:ct:desarrollo:raiz-atestacion:r" +
		numeroDecimal64(material.claveVersion)
	actoConfiguracion := "acto:ct:desarrollo:configuracion:r" +
		numeroDecimal64(material.configuracionOrden)
	actoPunteroConfiguracion := "acto:ct:desarrollo:puntero-configuracion:r" +
		numeroDecimal64(material.configuracionOrden)
	if material.claveHMACRevision == 1 {
		actoClaveHMAC = "acto:ct:desarrollo:clave-capacidad:v1"
		actoPunteroClave = "acto:ct:desarrollo:puntero-clave:v1"
	}
	if material.claveVersion == 1 {
		actoRaiz = "acto:ct:desarrollo:raiz-atestacion:v1"
	}
	if material.configuracionRef == "confianza:atestacion:ct:desarrollo:"+
		material.publicadaEn.Format("2006-01-02") {
		actoConfiguracion = "acto:ct:desarrollo:configuracion:" +
			material.publicadaEn.Format("2006-01-02")
		actoPunteroConfiguracion = "acto:ct:desarrollo:puntero-configuracion:" +
			material.publicadaEn.Format("2006-01-02")
	}
	consultas := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version
		  (clave_id,version,revision_gobierno,huella_gobierno_sha256,secreto_hmac,
		   huella_secreto_sha256,emisor_id,audiencia_consumo,valida_desde,valida_hasta,acto_ref)
		  VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT DO NOTHING`,
			[]any{material.claveHMACID, material.claveHMACVersion, material.claveHMACRevision,
				material.claveHMACHuella, material.claveHMAC, material.claveHMACSecreto,
				material.emisorID, material.audienciaConsumo,
				material.validaDesde, material.validaHasta, actoClaveHMAC}},
		{`INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision
		  (orden,clave_id,version,establecida_en,acto_ref)
		  SELECT $1,$2,$3,$4,$5 WHERE NOT EXISTS (
		   SELECT 1 FROM vec_autorizacion_atestada_v3.puntero_clave_emision
		    WHERE orden=$1)`,
			[]any{material.claveHMACOrden, material.claveHMACID,
				material.claveHMACVersion, material.validaDesde, actoPunteroClave}},
		{`INSERT INTO vec_autorizacion_atestada_v3.raiz_confianza_version
		  (clave_id,version,clave_publica_spki,huella_spki_sha256,valida_desde,
		   valida_hasta,suite,audiencia_despliegue,acto_ref)
		  VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT DO NOTHING`,
			[]any{material.claveID, material.claveVersion, material.spki, material.spkiHuella,
				material.validaDesde, material.validaHasta,
				confianzaatestacion.SuiteAtestacionAutorizacionV3COSEEdDSA,
				audienciaAtestacionContratacionTemporalDesarrollo,
				actoRaiz}},
		{`INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version
		  (revision,secuencia,huella_configuracion_sha256,publicada_en,expira_en,acto_ref)
		  VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`,
			[]any{material.configuracionRef, material.configuracionOrden,
				material.configuracionHuella, material.publicadaEn, material.expiraEn,
				actoConfiguracion}},
		{`INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz
		  (configuracion_revision,raiz_clave_id,raiz_version)
		  VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`,
			[]any{material.configuracionRef, material.claveID, material.claveVersion}},
		{`INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual
		  (orden,configuracion_revision,establecida_en,acto_ref)
		  SELECT $1,$2,$3,$4 WHERE NOT EXISTS (
		   SELECT 1 FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual
		    WHERE orden=$1)`,
			[]any{material.configuracionOrden, material.configuracionRef, material.publicadaEn,
				actoPunteroConfiguracion}},
	}
	for _, consulta := range consultas {
		if _, err := tx.Exec(ctx, consulta.sql, consulta.args...); err != nil {
			return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
		}
	}
	var coincide bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS (
		 SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
		 WHERE clave_id=$1 AND version=$2 AND revision_gobierno=$3
		   AND huella_gobierno_sha256=$4 AND secreto_hmac=$5
		   AND huella_secreto_sha256=$6 AND emisor_id=$7
		   AND audiencia_consumo=$8 AND valida_desde=$9 AND valida_hasta=$10)
		AND EXISTS (
		 SELECT 1 FROM vec_autorizacion_atestada_v3.raiz_confianza_version
		 WHERE clave_id=$11 AND version=$12 AND clave_publica_spki=$13
		   AND huella_spki_sha256=$14 AND valida_desde=$9 AND valida_hasta=$10
		   AND suite=$15 AND audiencia_despliegue=$16)
		AND EXISTS (
		 SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_confianza_version
		 WHERE revision=$17 AND secuencia=$18 AND huella_configuracion_sha256=$19
		   AND publicada_en=$20 AND expira_en=$21)
		AND EXISTS (
		 SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_raiz
		 WHERE configuracion_revision=$17 AND raiz_clave_id=$11 AND raiz_version=$12)
		AND EXISTS (
		 SELECT 1 FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual
		 WHERE orden=$18 AND configuracion_revision=$17)
		AND EXISTS (
		 SELECT 1 FROM vec_autorizacion_atestada_v3.puntero_clave_emision
		 WHERE orden=$22 AND clave_id=$1 AND version=$2)`,
		material.claveHMACID, material.claveHMACVersion, material.claveHMACRevision,
		material.claveHMACHuella, material.claveHMAC, material.claveHMACSecreto,
		material.emisorID, material.audienciaConsumo,
		material.validaDesde, material.validaHasta,
		material.claveID, material.claveVersion, material.spki, material.spkiHuella,
		confianzaatestacion.SuiteAtestacionAutorizacionV3COSEEdDSA,
		audienciaAtestacionContratacionTemporalDesarrollo,
		material.configuracionRef, material.configuracionOrden,
		material.configuracionHuella, material.publicadaEn, material.expiraEn,
		material.claveHMACOrden,
	).Scan(&coincide)
	if err != nil || !coincide {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	return nil
}

// Serializa con el publicador existente antes de tomar el snapshot SERIALIZABLE.
func ejecutarTransaccionGobiernoCTDesarrollo(ctx context.Context, pool *pgxpool.Pool, operar func(pgx.Tx) error) error {
	conexion, err := pool.Acquire(ctx)
	if err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if _, err = conexion.Exec(ctx, `
		SELECT pg_catalog.pg_advisory_lock(
		 pg_catalog.hashtextextended('vec:ct:desarrollo:gobierno-atestacion',0))`); err != nil {
		conexion.Release()
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	defer func() {
		ctxDesbloqueo, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelar()
		var liberado bool
		errDesbloqueo := conexion.QueryRow(ctxDesbloqueo, `
			SELECT pg_catalog.pg_advisory_unlock(
			 pg_catalog.hashtextextended('vec:ct:desarrollo:gobierno-atestacion',0))`,
		).Scan(&liberado)
		if errDesbloqueo != nil || !liberado {
			_ = conexion.Conn().Close(ctxDesbloqueo)
		}
		conexion.Release()
	}()
	tx, err := conexion.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	// Plazo propio también para ROLLBACK: se ejecuta con f.mu tomado por la
	// renovación y no debe esperar indefinidamente a una red cortada.
	defer func() {
		ctxRollback, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelar()
		_ = tx.Rollback(ctxRollback)
	}()
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario`); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if err := operar(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	return nil
}
