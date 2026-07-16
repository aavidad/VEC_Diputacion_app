package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	transaccionbolsa "vec-diputacion-granada/internal/modules/bolsa/internal/transaccion"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func ejecutarFaseInicialBolsaPostgreSQLE2E(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	poolAdmin *pgxpool.Pool,
	rutaEstado string,
) {
	t.Helper()
	ancla := instanteAutoritativoBolsaPostgreSQLE2E(t, ctx, pool)
	entorno := nuevoEntornoAutorizacionBolsaPostgreSQLE2E(t, ctx, pool, poolAdmin, ancla, true)
	protector := protectorSellosBolsaPostgreSQLE2E{}
	repositorio, err := NuevoRepositorioBaremaciones(
		pool, relojBolsaPostgreSQLE2E{ahora: ancla}, protector,
	)
	if err != nil {
		t.Fatalf("crear RepositorioBaremaciones real: %v", err)
	}
	base := nuevaBaremacionBaseBolsaPostgreSQLE2E(t, ancla)

	contextoReservaAlta := entorno.autorizar(
		t, ctx, puertosbolsa.AccionReservarAltaBaremacion, base.ID,
		referenciaDecisionAutorizacionBolsaPostgreSQLE2E("alta:reservar"), true,
	)
	solicitudReservaAlta := sellarReservaBolsaPostgreSQLE2E(t, puertosbolsa.SolicitudReservarCambioBaremacion{
		Contexto: contextoReservaAlta, Clase: puertosbolsa.ClaseCambioAltaBaremacion,
		ClaveIdempotencia:   "idempotencia:e2e:postgresql:v3:alta",
		BaremacionMeritoRef: base.ID, SolicitadaEn: ancla, ExpiraEn: ancla.Add(5 * time.Minute),
	})
	reservaAlta, err := reservarCambioDiagnosticadoBolsaPostgreSQLE2E(
		ctx, repositorio, poolAdmin, solicitudReservaAlta,
	)
	if err != nil {
		t.Fatalf("reservar alta V1 mediante adaptador real: %v", err)
	}
	contextoConfirmarAlta := entorno.autorizar(
		t, ctx, puertosbolsa.AccionConfirmarAltaBaremacion, base.ID,
		referenciaDecisionAutorizacionBolsaPostgreSQLE2E("alta:confirmar"), true,
	)
	solicitudConfirmarAlta := sellarConfirmacionBolsaPostgreSQLE2E(t, puertosbolsa.SolicitudConfirmarCambioBaremacion{
		Contexto: contextoConfirmarAlta, Token: reservaAlta.Token,
		Clase: puertosbolsa.ClaseCambioAltaBaremacion, Agregado: base,
		Trazabilidad: puertosbolsa.TrazabilidadCambioBaremacion{
			MotivoClave: "alta_autobaremacion",
			Motivo:      "Alta E2E de la autobaremacion calculada oficialmente.",
		},
		ConfirmadaEn: ancla,
	})
	resultadoAlta, err := repositorio.ConfirmarCambio(ctx, solicitudConfirmarAlta)
	if err != nil {
		t.Fatalf("confirmar alta V1 mediante adaptador real: %v", err)
	}
	if resultadoAlta.Version.Referencia.Numero != 1 || resultadoAlta.Version.Validar() != nil {
		t.Fatalf("version V1 no fiable: %+v", resultadoAlta.Version.Referencia)
	}
	baseAutoritativa := resultadoAlta.Version.Agregado
	instanteDecision := instanteAutoritativoBolsaPostgreSQLE2E(t, ctx, pool)
	if instanteDecision.Before(resultadoAlta.Version.ConfirmadaEn) {
		instanteDecision = resultadoAlta.Version.ConfirmadaEn.UTC().Truncate(time.Microsecond)
	}
	repositorioDecision, err := NuevoRepositorioBaremaciones(
		pool, relojBolsaPostgreSQLE2E{ahora: instanteDecision}, protector,
	)
	if err != nil {
		t.Fatalf("recrear repositorio para decision V2: %v", err)
	}

	contextoReservaDecision := entorno.autorizarEn(
		t, ctx, puertosbolsa.AccionReservarDecisionBaremacion, base.ID,
		referenciaDecisionAutorizacionBolsaPostgreSQLE2E("decision:1:reservar"), true,
		instanteDecision,
	)
	contextoPrevalidacion := entorno.autorizarEn(
		t, ctx, puertosbolsa.AccionPrevalidarArchivoProbatorioBaremacion, base.ID,
		referenciaDecisionAutorizacionBolsaPostgreSQLE2E("decision:1:prevalidar"), true,
		instanteDecision,
	)
	contextoConfirmacion := entorno.autorizarEn(
		t, ctx, puertosbolsa.AccionConfirmarDecisionBaremacion, base.ID,
		referenciaDecisionAutorizacionBolsaPostgreSQLE2E("decision:1:confirmar"), true,
		instanteDecision,
	)
	agregadoObjetivo, manifiesto := incorporarDecisionInicialBolsaPostgreSQLE2E(
		t, baseAutoritativa, resultadoAlta.Version.Referencia, contextoConfirmacion,
		contextoPrevalidacion, ancla,
	)
	solicitudReservaDecision := sellarReservaBolsaPostgreSQLE2E(t, puertosbolsa.SolicitudReservarCambioBaremacion{
		Contexto: contextoReservaDecision, Clase: puertosbolsa.ClaseCambioIncorporarDecision,
		ClaveIdempotencia:   "idempotencia:e2e:postgresql:v3:decision:1",
		BaremacionMeritoRef: base.ID, VersionEsperada: &resultadoAlta.Version.Referencia,
		SolicitadaEn: instanteDecision, ExpiraEn: instanteDecision.Add(5 * time.Minute),
	})
	reservaDecision, err := reservarCambioDiagnosticadoBolsaPostgreSQLE2E(
		ctx, repositorioDecision, poolAdmin, solicitudReservaDecision,
	)
	if err != nil {
		t.Fatalf("reservar decision V1 a V2 mediante adaptador real: %v", err)
	}
	solicitudConfirmarDecision := sellarConfirmacionBolsaPostgreSQLE2E(t, puertosbolsa.SolicitudConfirmarCambioBaremacion{
		Contexto: contextoConfirmacion, ContextoPrevalidacionArchivo: contextoPrevalidacion,
		Token: reservaDecision.Token, Clase: puertosbolsa.ClaseCambioIncorporarDecision,
		VersionEsperada: &resultadoAlta.Version.Referencia, Agregado: agregadoObjetivo,
		Manifiesto: &manifiesto,
		Trazabilidad: puertosbolsa.TrazabilidadCambioBaremacion{
			MotivoClave: "decision_tecnica_firmada",
			Motivo:      "Incorporacion E2E de la decision tecnica PAdES baseline B.",
		},
		ConfirmadaEn: instanteDecision,
	})
	resultadoDecision, err := repositorioDecision.ConfirmarCambio(ctx, solicitudConfirmarDecision)
	if err != nil {
		t.Fatalf("confirmar decision V2 con manifiesto 18/16 mediante adaptador real: %v", err)
	}
	if resultadoDecision.Version.Referencia.Numero != 2 || resultadoDecision.Version.Validar() != nil ||
		len(manifiesto.Autorizaciones) != 18 || len(manifiesto.Evidencias) != 16 {
		t.Fatalf("resultado V2/manifiesto no fiable: %+v", resultadoDecision.Version.Referencia)
	}

	instanteDecision2 := instanteAutoritativoBolsaPostgreSQLE2E(t, ctx, pool)
	if instanteDecision2.Before(resultadoDecision.Version.ConfirmadaEn) {
		instanteDecision2 = resultadoDecision.Version.ConfirmadaEn.UTC().Truncate(time.Microsecond)
	}
	repositorioRectificacion, err := NuevoRepositorioBaremaciones(
		pool, relojBolsaPostgreSQLE2E{ahora: instanteDecision2}, protector,
	)
	if err != nil {
		t.Fatalf("recrear repositorio para rectificacion V3: %v", err)
	}
	referenciaReserva2 := referenciaDecisionAutorizacionBolsaPostgreSQLE2E("decision:2:reservar")
	referenciaPrevalidacion2 := referenciaDecisionAutorizacionBolsaPostgreSQLE2E("decision:2:prevalidar")
	referenciaConfirmacion2 := referenciaDecisionAutorizacionBolsaPostgreSQLE2E("decision:2:confirmar")
	contextoReservaRectificacion := entorno.autorizarEn(
		t, ctx, puertosbolsa.AccionReservarDecisionBaremacion, base.ID,
		referenciaReserva2, true, instanteDecision2,
	)
	contextoPrevalidacion2 := entorno.autorizarEn(
		t, ctx, puertosbolsa.AccionPrevalidarArchivoProbatorioBaremacion, base.ID,
		referenciaPrevalidacion2, true, instanteDecision2,
	)
	contextoConfirmacion2 := entorno.autorizarEn(
		t, ctx, puertosbolsa.AccionConfirmarDecisionBaremacion, base.ID,
		referenciaConfirmacion2, true, instanteDecision2,
	)
	agregadoRectificado, manifiesto2 := incorporarRectificacionBolsaPostgreSQLE2E(
		t, resultadoDecision.Version.Agregado, resultadoDecision.Version.Referencia,
		contextoConfirmacion2, contextoPrevalidacion2, instanteDecision2,
	)
	if manifiesto.SelloManifiestoHMACSHA256 == manifiesto2.SelloManifiestoHMACSHA256 {
		t.Fatal("los manifiestos V2 y V3 no quedaron ligados a sellos distintos")
	}
	solicitudReservaRectificacion := sellarReservaBolsaPostgreSQLE2E(t, puertosbolsa.SolicitudReservarCambioBaremacion{
		Contexto: contextoReservaRectificacion, Clase: puertosbolsa.ClaseCambioIncorporarDecision,
		ClaveIdempotencia:   "idempotencia:e2e:postgresql:v3:decision:2",
		BaremacionMeritoRef: base.ID, VersionEsperada: &resultadoDecision.Version.Referencia,
		SolicitadaEn: instanteDecision2, ExpiraEn: instanteDecision2.Add(5 * time.Minute),
	})
	reservaRectificacion, err := reservarCambioDiagnosticadoBolsaPostgreSQLE2E(
		ctx, repositorioRectificacion, poolAdmin, solicitudReservaRectificacion,
	)
	if err != nil {
		t.Fatalf("reservar rectificacion V2 a V3 mediante adaptador real: %v", err)
	}
	trazabilidadRectificacion := puertosbolsa.TrazabilidadCambioBaremacion{
		MotivoClave: "rectificacion_tecnica_firmada",
		Motivo:      "Rectificacion E2E firmada de la puntuacion tecnica reconocida.",
	}
	solicitudConfirmarRectificacion := sellarConfirmacionBolsaPostgreSQLE2E(t, puertosbolsa.SolicitudConfirmarCambioBaremacion{
		Contexto: contextoConfirmacion2, ContextoPrevalidacionArchivo: contextoPrevalidacion2,
		Token: reservaRectificacion.Token, Clase: puertosbolsa.ClaseCambioIncorporarDecision,
		VersionEsperada: &resultadoDecision.Version.Referencia, Agregado: agregadoRectificado,
		Manifiesto: &manifiesto2, Trazabilidad: trazabilidadRectificacion,
		ConfirmadaEn: instanteDecision2,
	})
	huellaConfirmacion2, err := transaccionbolsa.HuellaEfectoConfirmacionV2(solicitudConfirmarRectificacion)
	if err != nil {
		t.Fatalf("calcular huella de confirmacion V3 exacta: %v", err)
	}
	huellaPrevalidacion2, err := transaccionbolsa.HuellaEfectoPrevalidacionArchivoProbatorio(
		solicitudConfirmarRectificacion,
	)
	if err != nil {
		t.Fatalf("calcular huella de prevalidacion V3 exacta: %v", err)
	}
	huellaToken2 := transaccionbolsa.HuellaTokenReserva(reservaRectificacion.Token)
	observadorFallo := &observadorSellosBolsaPostgreSQLE2E{}
	repositorioKMSNoDisponible, err := NuevoRepositorioBaremaciones(
		pool,
		relojBolsaPostgreSQLE2E{ahora: instanteDecision2},
		protectorSellosBolsaPostgreSQLE2E{
			fallaSelloHistorico: manifiesto.SelloManifiestoHMACSHA256,
			observador:          observadorFallo,
		},
	)
	if err != nil {
		t.Fatalf("crear repositorio con fallo KMS historico controlado: %v", err)
	}
	_, err = repositorioKMSNoDisponible.ConfirmarCambio(ctx, solicitudConfirmarRectificacion)
	if !errors.Is(err, puertosbolsa.ErrVerificacionSelloBaremacionNoDisponible) {
		t.Fatalf("fallo KMS historico exacto no propagado: %v", err)
	}
	if !observadorFallo.contieneEnOrden(
		manifiesto2.SelloManifiestoHMACSHA256,
		manifiesto.SelloManifiestoHMACSHA256,
	) {
		t.Fatal("el fallo KMS no ocurrio despues de verificar el manifiesto nuevo y al leer el historico")
	}
	exigirPrevalidacionConsumidaSinV3BolsaPostgreSQLE2E(
		t, ctx, poolAdmin, resultadoDecision.Version.Referencia,
		manifiesto, manifiesto2, referenciaReserva2,
		referenciaPrevalidacion2, referenciaConfirmacion2,
		huellaToken2, huellaConfirmacion2, huellaPrevalidacion2,
	)
	// La capacidad permanece exclusivamente en memoria. Tras restablecer el
	// verificador se completa la misma confirmacion antes de terminar el
	// proceso; ningun estado de recuperacion serializa ni permite reconstruir
	// el token temporal.
	observadorRecuperacion := &observadorSellosBolsaPostgreSQLE2E{}
	repositorioRecuperado, err := NuevoRepositorioBaremaciones(
		pool,
		relojBolsaPostgreSQLE2E{ahora: instanteDecision2},
		protectorSellosBolsaPostgreSQLE2E{observador: observadorRecuperacion},
	)
	if err != nil {
		t.Fatalf("restablecer repositorio tras fallo KMS: %v", err)
	}
	resultadoRectificacion, err := repositorioRecuperado.ConfirmarCambio(
		ctx, solicitudConfirmarRectificacion,
	)
	if err != nil {
		t.Fatalf("confirmar V3 al restablecer KMS con la capacidad aun en memoria: %v", err)
	}
	if resultadoRectificacion.ValidarPara(solicitudConfirmarRectificacion) != nil ||
		resultadoRectificacion.Version.Referencia.Numero != 3 ||
		!observadorRecuperacion.contieneTodos(
			manifiesto.SelloManifiestoHMACSHA256,
			manifiesto2.SelloManifiestoHMACSHA256,
		) {
		t.Fatal("la recuperacion KMS en memoria no produjo la V3 exacta ni verifico todo el archivo")
	}
	versionConfirmada := resultadoRectificacion.Version.Referencia
	estado := estadoBolsaPostgreSQLE2E{
		Esquema: "vec.pruebas.bolsa.postgresql-v3-e2e.v3",
		Etapa:   etapaConfirmadaPendienteReplayPostgreSQLE2E,
		Ancla:   ancla, InstanteUso: instanteDecision2,
		HuellaTokenSHA256: huellaToken2,
		VersionBase:       resultadoDecision.Version.Referencia,
		AgregadoObjetivo:  agregadoRectificado, ManifiestoHistorico: manifiesto,
		Manifiesto: manifiesto2, Trazabilidad: trazabilidadRectificacion,
		AutorizacionReservaRef:          referenciaReserva2,
		AutorizacionPrevalidacionRef:    referenciaPrevalidacion2,
		AutorizacionConfirmacionRef:     referenciaConfirmacion2,
		ConfirmadaEn:                    instanteDecision2,
		HuellaSolicitudReservaHMAC:      solicitudReservaRectificacion.HuellaSolicitudHMAC,
		HuellaSolicitudConfirmacionHMAC: solicitudConfirmarRectificacion.HuellaSolicitudHMAC,
		VersionConfirmada:               &versionConfirmada,
		AuditoriaRef:                    resultadoRectificacion.Evidencia.AuditoriaRef,
		HuellaAuditoriaSHA256:           resultadoRectificacion.Evidencia.HuellaAuditoriaSHA256,
		EventoOutboxRef:                 resultadoRectificacion.Evidencia.EventoOutboxRef,
		HuellaEventoOutboxSHA256:        resultadoRectificacion.Evidencia.HuellaEventoOutboxSHA256,
		ConfirmadaFiableEn:              resultadoRectificacion.Evidencia.ConfirmadaEn,
	}
	exigirConfirmacionV3DurableBolsaPostgreSQLE2E(t, ctx, poolAdmin, estado)
	// El runner entrega un mktemp ya creado 0600; se valida y reemplaza de
	// forma atomica sin seguir enlaces ni truncar un inode no inspeccionado.
	escribirEstadoBolsaPostgreSQLE2E(t, rutaEstado, estado, true)
}

func exigirPrevalidacionConsumidaSinV3BolsaPostgreSQLE2E(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
	versionBase puertosbolsa.ReferenciaVersionBaremacion,
	manifiestoHistorico puertosbolsa.ManifiestoProbatorioBaremacion,
	manifiestoObjetivo puertosbolsa.ManifiestoProbatorioBaremacion,
	autorizacionReservaRef string,
	autorizacionPrevalidacionRef string,
	autorizacionConfirmacionRef string,
	huellaToken string,
	huellaConfirmacion string,
	huellaEfectoPrevalidacion string,
) {
	t.Helper()
	var numeroActual, huellaActual, estadoPrevalidacion, numeroPrevalidacion, huellaPrevalidacion string
	var versiones, versionesV3, manifiestos, manifiestoObjetivoPersistido int64
	var manifiestoHistoricoValido int64
	var autorizaciones, evidencias, auditorias, eventos int64
	var totalPrevalidacion, resultadosPrevalidacion, usosPrevalidacion int64
	var usosConfirmacion, atestacionesConfirmacion, reservasActivas int64
	var prevalidacionLigada int64
	err := admin.QueryRow(ctx, `
		SELECT
			COALESCE((SELECT numero::text FROM vec_bolsa_baremacion.baremacion_actual
			          WHERE baremacion_merito_ref=$1),''),
			COALESCE((SELECT huella_estado_sha256 FROM vec_bolsa_baremacion.baremacion_actual
			          WHERE baremacion_merito_ref=$1),''),
			(SELECT count(*) FROM vec_bolsa_baremacion.version_baremacion
			  WHERE baremacion_merito_ref=$1),
			(SELECT count(*) FROM vec_bolsa_baremacion.version_baremacion
			  WHERE baremacion_merito_ref=$1 AND numero=3),
			(SELECT count(*) FROM vec_bolsa_baremacion.manifiesto_probatorio_v3
			  WHERE baremacion_merito_ref=$1),
			(SELECT count(*) FROM vec_bolsa_baremacion.manifiesto_probatorio_v3
			  WHERE referencia=$2),
			(SELECT count(*) FROM vec_bolsa_baremacion.manifiesto_autorizacion_v3 AS a
			  JOIN vec_bolsa_baremacion.manifiesto_probatorio_v3 AS m
			    ON m.referencia=a.manifiesto_ref WHERE m.baremacion_merito_ref=$1),
			(SELECT count(*) FROM vec_bolsa_baremacion.manifiesto_evidencia_v3 AS e
			  JOIN vec_bolsa_baremacion.manifiesto_probatorio_v3 AS m
			    ON m.referencia=e.manifiesto_ref WHERE m.baremacion_merito_ref=$1),
			(SELECT count(*) FROM vec_bolsa_baremacion.auditoria WHERE baremacion_merito_ref=$1),
			(SELECT count(*) FROM vec_bolsa_baremacion.evento_outbox WHERE baremacion_merito_ref=$1),
			COALESCE((SELECT estado_resultado FROM vec_bolsa_baremacion.prevalidacion_archivo_probatorio_v3
			          WHERE autorizacion_prevalidacion_ref=$3),''),
			COALESCE((SELECT numero_version::text FROM vec_bolsa_baremacion.prevalidacion_archivo_probatorio_v3
			          WHERE autorizacion_prevalidacion_ref=$3),''),
			COALESCE((SELECT total_manifiestos::bigint FROM vec_bolsa_baremacion.prevalidacion_archivo_probatorio_v3
			          WHERE autorizacion_prevalidacion_ref=$3),-1::bigint),
			COALESCE((SELECT huella_estado_sha256 FROM vec_bolsa_baremacion.prevalidacion_archivo_probatorio_v3
			          WHERE autorizacion_prevalidacion_ref=$3),''),
			(SELECT count(*) FROM vec_bolsa_baremacion.resultado_prevalidacion_archivo_v3
			  WHERE autorizacion_prevalidacion_ref=$3),
			(SELECT count(*) FROM vec_bolsa_baremacion.uso_decision
			  WHERE decision_ref=$3 AND tipo_efecto='prevalidacion_archivo'),
			(SELECT count(*) FROM vec_bolsa_baremacion.uso_decision
			  WHERE decision_ref=$4 AND tipo_efecto='confirmacion'),
			(SELECT count(*) FROM vec_bolsa_baremacion.atestacion_pdp_actual
			  WHERE decision_ref=$4 AND estado='activa'),
			(SELECT count(*) FROM vec_bolsa_baremacion.reserva_actual AS actual
			  JOIN vec_bolsa_baremacion.reserva_version AS reserva
			    ON reserva.ambito_idempotencia_sha256=actual.ambito_idempotencia_sha256
			   AND reserva.reserva_ref=actual.reserva_ref AND reserva.version=actual.version
			   AND reserva.estado=actual.estado
			  WHERE reserva.decision_reserva_ref=$5 AND actual.estado='activa'
			    AND reserva.huella_confirmacion_sha256 IS NULL
			    AND reserva.numero_version_confirmada IS NULL),
			(SELECT count(*) FROM vec_bolsa_baremacion.manifiesto_probatorio_v3
			  WHERE referencia=$6 AND sello_manifiesto_hmac_sha256=$7
			    AND total_autorizaciones=18 AND total_evidencias=16),
			(SELECT count(*) FROM vec_bolsa_baremacion.prevalidacion_archivo_probatorio_v3 AS pre
			  WHERE pre.autorizacion_prevalidacion_ref=$3
			    AND pre.huella_token_sha256=$8
			    AND pre.huella_confirmacion_sha256=$9
			    AND pre.huella_efecto_prevalidacion_sha256=$10
			    AND EXISTS (
			      SELECT 1 FROM vec_bolsa_baremacion.token_reserva AS token
			      JOIN vec_bolsa_baremacion.reserva_version AS reserva
			        ON reserva.reserva_ref=token.reserva_ref
			       AND reserva.decision_reserva_ref=$5
			      WHERE token.huella_token_sha256=pre.huella_token_sha256
			        AND token.reserva_ref=pre.reserva_ref))
	`, versionBase.BaremacionMeritoRef, manifiestoObjetivo.Referencia,
		autorizacionPrevalidacionRef, autorizacionConfirmacionRef,
		autorizacionReservaRef, manifiestoHistorico.Referencia,
		manifiestoHistorico.SelloManifiestoHMACSHA256, huellaToken,
		huellaConfirmacion, huellaEfectoPrevalidacion,
	).Scan(
		&numeroActual, &huellaActual, &versiones, &versionesV3,
		&manifiestos, &manifiestoObjetivoPersistido, &autorizaciones,
		&evidencias, &auditorias, &eventos, &estadoPrevalidacion,
		&numeroPrevalidacion, &totalPrevalidacion, &huellaPrevalidacion,
		&resultadosPrevalidacion, &usosPrevalidacion, &usosConfirmacion,
		&atestacionesConfirmacion, &reservasActivas, &manifiestoHistoricoValido,
		&prevalidacionLigada,
	)
	if err != nil {
		t.Fatalf("inspeccionar prevalidacion durable tras fallo KMS: %v", err)
	}
	if numeroActual != "2" || huellaActual != versionBase.HuellaEstadoSHA256 ||
		versiones != 2 || versionesV3 != 0 || manifiestos != 1 ||
		manifiestoObjetivoPersistido != 0 || manifiestoHistoricoValido != 1 ||
		autorizaciones != 18 || evidencias != 16 || auditorias != 2 || eventos != 2 ||
		estadoPrevalidacion != "activa" || numeroPrevalidacion != "2" ||
		totalPrevalidacion != 1 || huellaPrevalidacion != versionBase.HuellaEstadoSHA256 ||
		resultadosPrevalidacion != 0 || usosPrevalidacion != 1 || usosConfirmacion != 0 ||
		atestacionesConfirmacion != 1 || reservasActivas != 1 || prevalidacionLigada != 1 {
		t.Fatalf(
			"el fallo KMS dejo efectos V3 o perdio la prevalidacion: actual=%s versiones=%d manifiestos=%d pre=%s/%s/%d resultado=%d uso_confirmacion=%d",
			numeroActual, versiones, manifiestos, estadoPrevalidacion,
			numeroPrevalidacion, totalPrevalidacion, resultadosPrevalidacion, usosConfirmacion,
		)
	}
}

func exigirConfirmacionV3DurableBolsaPostgreSQLE2E(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
	estado estadoBolsaPostgreSQLE2E,
) {
	t.Helper()
	if estado.VersionConfirmada == nil {
		t.Fatal("no se puede inspeccionar V3 durable sin referencia confirmada")
	}
	versionConfirmada := *estado.VersionConfirmada
	var numeroActual, huellaActual, numeroResultado, huellaResultado string
	var versiones, manifiestos, autorizaciones, evidencias, auditorias, eventos int64
	var manifiestoObjetivo, resultadoPrevalidacion, usoConfirmacion, reservaConfirmada int64
	var transaccionExacta int64
	err := admin.QueryRow(ctx, `
		SELECT
			COALESCE((SELECT numero::text FROM vec_bolsa_baremacion.baremacion_actual
			          WHERE baremacion_merito_ref=$1),''),
			COALESCE((SELECT huella_estado_sha256 FROM vec_bolsa_baremacion.baremacion_actual
			          WHERE baremacion_merito_ref=$1),''),
			(SELECT count(*) FROM vec_bolsa_baremacion.version_baremacion
			  WHERE baremacion_merito_ref=$1),
			(SELECT count(*) FROM vec_bolsa_baremacion.manifiesto_probatorio_v3
			  WHERE baremacion_merito_ref=$1),
			(SELECT count(*) FROM vec_bolsa_baremacion.manifiesto_autorizacion_v3 AS a
			  JOIN vec_bolsa_baremacion.manifiesto_probatorio_v3 AS m
			    ON m.referencia=a.manifiesto_ref WHERE m.baremacion_merito_ref=$1),
			(SELECT count(*) FROM vec_bolsa_baremacion.manifiesto_evidencia_v3 AS e
			  JOIN vec_bolsa_baremacion.manifiesto_probatorio_v3 AS m
			    ON m.referencia=e.manifiesto_ref WHERE m.baremacion_merito_ref=$1),
			(SELECT count(*) FROM vec_bolsa_baremacion.auditoria WHERE baremacion_merito_ref=$1),
			(SELECT count(*) FROM vec_bolsa_baremacion.evento_outbox WHERE baremacion_merito_ref=$1),
			(SELECT count(*) FROM vec_bolsa_baremacion.manifiesto_probatorio_v3
			  WHERE referencia=$2 AND numero_version=3 AND total_autorizaciones=18
			    AND total_evidencias=16 AND sello_manifiesto_hmac_sha256=$3
			    AND auditoria_ref=$4 AND evento_outbox_ref=$5),
			(SELECT count(*) FROM vec_bolsa_baremacion.resultado_prevalidacion_archivo_v3
			  WHERE autorizacion_prevalidacion_ref=$6),
			COALESCE((SELECT numero_version::text FROM vec_bolsa_baremacion.resultado_prevalidacion_archivo_v3
			          WHERE autorizacion_prevalidacion_ref=$6),''),
			COALESCE((SELECT huella_estado_sha256 FROM vec_bolsa_baremacion.resultado_prevalidacion_archivo_v3
			          WHERE autorizacion_prevalidacion_ref=$6),''),
			(SELECT count(*) FROM vec_bolsa_baremacion.uso_decision
			  WHERE decision_ref=$7 AND tipo_efecto='confirmacion' AND resultado_ref=$4),
			(SELECT count(*) FROM vec_bolsa_baremacion.reserva_actual AS actual
			  JOIN vec_bolsa_baremacion.reserva_version AS reserva
			    ON reserva.ambito_idempotencia_sha256=actual.ambito_idempotencia_sha256
			   AND reserva.reserva_ref=actual.reserva_ref AND reserva.version=actual.version
			   AND reserva.estado=actual.estado
			  WHERE reserva.decision_reserva_ref=$8 AND actual.estado='confirmada'
			    AND reserva.numero_version_confirmada=3),
			(SELECT count(*) FROM vec_bolsa_baremacion.version_baremacion AS version
			  JOIN vec_bolsa_baremacion.auditoria AS auditoria
			    ON auditoria.referencia=version.auditoria_ref
			  JOIN vec_bolsa_baremacion.evento_outbox AS evento
			    ON evento.referencia=version.evento_outbox_ref
			  WHERE version.baremacion_merito_ref=$1 AND version.numero=3
			    AND version.confirmada_en=$9
			    AND auditoria.huella_registro_sha256=$10
			    AND evento.huella_registro_sha256=$11)
	`, estado.AgregadoObjetivo.ID, estado.Manifiesto.Referencia,
		estado.Manifiesto.SelloManifiestoHMACSHA256, estado.AuditoriaRef,
		estado.EventoOutboxRef, estado.AutorizacionPrevalidacionRef,
		estado.AutorizacionConfirmacionRef, estado.AutorizacionReservaRef,
		estado.ConfirmadaFiableEn, estado.HuellaAuditoriaSHA256,
		estado.HuellaEventoOutboxSHA256,
	).Scan(
		&numeroActual, &huellaActual, &versiones, &manifiestos,
		&autorizaciones, &evidencias, &auditorias, &eventos,
		&manifiestoObjetivo, &resultadoPrevalidacion, &numeroResultado,
		&huellaResultado, &usoConfirmacion, &reservaConfirmada, &transaccionExacta,
	)
	if err != nil {
		t.Fatalf("inspeccionar confirmacion V3 durable: %v", err)
	}
	if numeroActual != "3" || huellaActual != versionConfirmada.HuellaEstadoSHA256 ||
		versiones != 3 || manifiestos != 2 || autorizaciones != 36 || evidencias != 32 ||
		auditorias != 3 || eventos != 3 || manifiestoObjetivo != 1 ||
		resultadoPrevalidacion != 1 || numeroResultado != "3" ||
		huellaResultado != versionConfirmada.HuellaEstadoSHA256 ||
		usoConfirmacion != 1 || reservaConfirmada != 1 || transaccionExacta != 1 {
		t.Fatalf(
			"confirmacion V3 durable incompleta o duplicada: actual=%s versiones=%d manifiestos=%d cobertura=%d/%d resultado_pre=%d uso_confirmacion=%d",
			numeroActual, versiones, manifiestos, autorizaciones, evidencias,
			resultadoPrevalidacion, usoConfirmacion,
		)
	}
}
