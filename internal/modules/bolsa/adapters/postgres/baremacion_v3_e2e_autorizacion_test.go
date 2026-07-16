package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	postgresvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func ejecutarE2E(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	sql string,
	argumentos ...any,
) {
	t.Helper()
	if _, err := tx.Exec(ctx, sql, argumentos...); err != nil {
		t.Fatalf("SQL E2E rechazado: %v", err)
	}
}

func JSONE2E(t *testing.T, valor any) []byte {
	t.Helper()
	documento, err := json.Marshal(valor)
	if err != nil {
		t.Fatalf("serializar documento E2E: %v", err)
	}
	return documento
}

func (e entornoAutorizacionBolsaPostgreSQLE2E) autorizar(
	t *testing.T,
	ctx context.Context,
	accion puertosbolsa.AccionOperacionBaremacion,
	recursoRef string,
	decisionRef string,
	registrar bool,
) puertosbolsa.ContextoOperacionBaremacion {
	return e.autorizarEn(t, ctx, accion, recursoRef, decisionRef, registrar, e.ancla)
}

func (e entornoAutorizacionBolsaPostgreSQLE2E) autorizarEn(
	t *testing.T,
	ctx context.Context,
	accion puertosbolsa.AccionOperacionBaremacion,
	recursoRef string,
	decisionRef string,
	registrar bool,
	instanteUso time.Time,
) puertosbolsa.ContextoOperacionBaremacion {
	t.Helper()
	clase, existe := puertosbolsa.ClaseRecursoRequeridaOperacionBaremacion(accion)
	if !existe {
		t.Fatalf("accion E2E desconocida: %s", accion)
	}
	servicio, err := aplicacionvec.NuevoServicioAutorizacion(
		e.fuente,
		registroConcesionesNuloBolsaPostgreSQLE2E{},
		registroDenegacionesBolsaPostgreSQLE2E{},
		relojBolsaPostgreSQLE2E{ahora: e.ancla},
		generadorDecisionBolsaPostgreSQLE2E(decisionRef),
		aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 5 * time.Minute},
	)
	if err != nil {
		t.Fatalf("crear servicio de autorizacion E2E: %v", err)
	}
	decision, err := servicio.Exigir(ctx, dominiovec.SolicitudAutorizacion{
		Principal: e.actor.Principal, PerfilActivoRef: e.actor.PerfilActivoRef,
		ContextoActor: e.actor, VinculoAutenticacionActor: e.vinculo,
		Accion: string(accion),
		Recurso: dominiovec.RecursoAutorizable{
			Referencia: recursoRef, ModuloID: "bolsa", Tipo: string(clase),
			Ambitos:   map[string]string{"sujeto_ref": sujetoRefBolsaPostgreSQLE2E},
			Atributos: map[string]string{},
		},
		Finalidad:      finalidadBolsaPostgreSQLE2E,
		CorrelacionRef: correlacionBolsaPostgreSQLE2E,
		Motivo:         "E2E real de Bolsa V3 sobre PostgreSQL",
	})
	if err != nil {
		t.Fatalf("autorizar %s mediante servicio real: %v", accion, err)
	}
	if registrar {
		sembrarDecisionAutorizacionBolsaPostgreSQLE2E(t, ctx, e.admin, decision)
		insertarAtestacionPDPBolsaPostgreSQLE2E(t, ctx, e.admin, decision, e.ancla)
	}
	contexto, err := puertosbolsa.NuevaAutorizacionOperacionBaremacion(decision, e.vinculoB, instanteUso)
	if err != nil {
		t.Fatalf("crear contexto Bolsa para %s: %v", accion, err)
	}
	return contexto
}

func sembrarDecisionAutorizacionBolsaPostgreSQLE2E(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
	decision dominiovec.DecisionAutorizacion,
) {
	t.Helper()
	registroFixture, err := postgresvec.NuevoAlmacenAutorizacion(admin)
	if err != nil {
		t.Fatalf("crear registro administrativo para fixture PDP: %v", err)
	}
	if err = registroFixture.RegistrarDecisionSiInstantaneaVigente(ctx, decision); err != nil {
		t.Fatalf("sembrar decision PDP concedida para el E2E: %v", err)
	}
}

func insertarAtestacionPDPBolsaPostgreSQLE2E(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
	decision dominiovec.DecisionAutorizacion,
	verificadaEn time.Time,
) {
	t.Helper()
	evidencia, err := puertosvec.NuevaEvidenciaUsoDecisionAutorizacion(decision, verificadaEn)
	if err != nil {
		t.Fatalf("crear evidencia canonica PDP: %v", err)
	}
	datos, err := evidencia.Datos()
	if err != nil {
		t.Fatalf("proyectar evidencia PDP: %v", err)
	}
	decisionCanonica, err := datos.RepresentacionCanonica()
	if err != nil {
		t.Fatalf("representar decision PDP: %v", err)
	}
	defer borrarBytesE2E(decisionCanonica)
	payload := []byte("payload vec-ad-1 real para " + decision.DecisionRef)
	sobre := []byte("COSE_Sign1 E2E verificable prevalidado para " + decision.DecisionRef)
	evidenciaCanonica := []byte("evidencia de verificacion EdDSA controlada para " + decision.DecisionRef)
	defer borrarBytesE2E(payload, sobre, evidenciaCanonica)
	atestacionRef := "atestacion:e2e:postgresql:v3:" + decision.DecisionRef
	actoRef := "acto:atestacion:e2e:postgresql:v3:" + decision.DecisionRef
	_, err = admin.Exec(ctx, `
		INSERT INTO vec_bolsa_baremacion.atestacion_pdp_version
		(atestacion_ref, version, estado, decision_ref, esquema_huella_decision,
		 huella_decision_sha256, decision_canonica, suite, algoritmo_cose,
		 audiencia_cose, clave_id, audiencia_despliegue, estado_confianza,
		 huella_clave_sha256, payload_vec_ad_1, sobre_cose_sign1,
		 evidencia_canonica, huella_payload_sha256, huella_sobre_sha256,
		 huella_evidencia_sha256, verificada_en, raiz_valida_desde,
		 raiz_valida_hasta, revision_confianza, huella_configuracion_sha256,
		 configuracion_publicada_en, configuracion_expira_en, acto_ref, registrada_en)
		VALUES ($1,1,'activa',$2,$3,$4,$5,$6,$7,$8,$9,$10,'activa',$11,$12,$13,$14,
		        $15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,clock_timestamp())`,
		atestacionRef, decision.DecisionRef, datos.EsquemaHuella, datos.HuellaDecisionSHA256,
		decisionCanonica, "VEC-AD-COSE-EDDSA-1", "EdDSA", "atestacion_autorizacion_pdp",
		"clave:e2e:postgresql:v3:eddsa", "despliegue:e2e:postgresql:v3",
		huellaSHA256E2E("clave EdDSA E2E PostgreSQL V3"), payload, sobre, evidenciaCanonica,
		huellaSHA256BytesE2E(payload), huellaSHA256BytesE2E(sobre),
		huellaSHA256BytesE2E(evidenciaCanonica), verificadaEn,
		verificadaEn.Add(-time.Hour), verificadaEn.Add(time.Hour), "revision-confianza-e2e-v1",
		huellaSHA256E2E("configuracion confianza E2E PostgreSQL V3"),
		verificadaEn.Add(-time.Hour), verificadaEn.Add(time.Hour), actoRef)
	if err != nil {
		t.Fatalf("insertar atestacion PDP real para %s: %v", decision.DecisionRef, err)
	}
	_, err = admin.Exec(ctx, `
		INSERT INTO vec_bolsa_baremacion.atestacion_pdp_actual
		(decision_ref, atestacion_ref, version, estado, actualizada_en, acto_ref)
		VALUES ($1,$2,1,'activa',clock_timestamp(),$3)`,
		decision.DecisionRef, atestacionRef, actoRef)
	if err != nil {
		t.Fatalf("insertar puntero de atestacion para %s: %v", decision.DecisionRef, err)
	}
}

func huellaSHA256E2E(valor string) string { return huellaSHA256BytesE2E([]byte(valor)) }

func huellaSHA256BytesE2E(valor []byte) string {
	suma := sha256.Sum256(valor)
	return hex.EncodeToString(suma[:])
}

func nuevaBaremacionBaseBolsaPostgreSQLE2E(
	t *testing.T,
	ancla time.Time,
) dominiobolsa.BaremacionMerito {
	t.Helper()
	criterio := dominiobolsa.ReferenciaCriterio{
		ProcesoRef: procesoRefBolsaPostgreSQLE2E,
		Clave:      "experiencia.entidad_publica.grupo_c1",
		Version:    1,
		HuellaSHA256: huellaSHA256E2E(
			"criterio publicado E2E PostgreSQL V3 experiencia entidad publica",
		),
		PuntosMaximos: 10 * dominiobolsa.UnidadesPorPunto,
		ReglaCalculo: dominiobolsa.ReferenciaReglaCalculo{
			Clave: "experiencia_publica_dias", Version: 1,
			HuellaSHA256: huellaSHA256E2E("regla de calculo E2E PostgreSQL V3 version 1"),
		},
	}
	evidencias := []dominiobolsa.EvidenciaMerito{{
		Referencia: dominiobolsa.ReferenciaEvidencia{
			DocumentoRef: "documento:e2e:postgresql:v3:merito:1", VersionDocumento: 1,
			RepresentacionRef: "representacion:e2e:postgresql:v3:merito:1",
			HuellaSHA256:      huellaSHA256E2E("bytes del documento de merito E2E PostgreSQL V3"),
		},
	}}
	calculo := dominiobolsa.CalculoOficialBaremacion{
		CalculoRef: "calculo:e2e:postgresql:v3:oficial:1",
		ProcesoRef: procesoRefBolsaPostgreSQLE2E, SolicitudRef: solicitudRefBolsaPostgreSQLE2E,
		SujetoRef: sujetoRefBolsaPostgreSQLE2E, BaremacionMeritoRef: baremacionRefBolsaPostgreSQLE2E,
		Criterio: criterio, Regla: criterio.ReglaCalculo, Evidencias: evidencias,
		EntradaRef:            "entrada:e2e:postgresql:v3:calculo:1",
		HuellaEntradaSHA256:   huellaSHA256E2E("entrada canonica del calculo E2E PostgreSQL V3"),
		PuntosCalculados:      4_250_000,
		DesgloseRef:           "desglose:e2e:postgresql:v3:calculo:1",
		HuellaDesgloseSHA256:  huellaSHA256E2E("desglose oficial del calculo E2E PostgreSQL V3"),
		ResultadoRef:          "resultado:e2e:postgresql:v3:calculo:1",
		HuellaResultadoSHA256: huellaSHA256E2E("resultado oficial del calculo E2E PostgreSQL V3"),
		MotorCalculoRef:       "motor:e2e:postgresql:v3:baremo",
		VersionMotorCalculo:   "motor-e2e-v1.0.0",
		EvidenciaEjecucionRef: "ejecucion:e2e:postgresql:v3:calculo:1",
		HuellaEjecucionSHA256: huellaSHA256E2E("traza de ejecucion del motor E2E PostgreSQL V3"),
		CalculadoEn:           ancla.Add(-5 * time.Minute),
	}
	baremacion, err := dominiobolsa.NuevaBaremacionMerito(dominiobolsa.AltaMeritoBaremable{
		ID: baremacionRefBolsaPostgreSQLE2E, ProcesoRef: procesoRefBolsaPostgreSQLE2E,
		SolicitudRef: solicitudRefBolsaPostgreSQLE2E, SujetoRef: sujetoRefBolsaPostgreSQLE2E,
		Criterio: criterio, EvidenciasIniciales: evidencias, PuntosDeclarados: 5_000_000,
		CalculoOficial: calculo, CreadaEn: ancla.Add(-4 * time.Minute),
	})
	if err != nil {
		t.Fatalf("crear agregado base E2E: %v", err)
	}
	return baremacion
}

func firmaBaselineBBolsaPostgreSQLE2E(
	t *testing.T,
	contenido dominiobolsa.ContenidoDecisionTecnica,
	indice int,
	ancla time.Time,
) dominiobolsa.FirmaDecisionTecnica {
	t.Helper()
	huellaContenido, err := contenido.HuellaContenidoSHA256()
	if err != nil {
		t.Fatalf("huella del contenido de decision E2E: %v", err)
	}
	prefijo := fmt.Sprintf("decision:%d", indice)
	representacionProvisional, err := puertosbolsa.NuevaCargaProtegida(
		[]byte("representacion provisional autentica para " + prefijo),
	)
	if err != nil {
		t.Fatalf("crear carga provisional HMAC: %v", err)
	}
	selloProvisional, err := selloHMACBolsaPostgreSQLE2E(
		puertosbolsa.FinalidadSelloManifiestoProbatorioBaremacionV3,
		representacionProvisional,
	)
	if err != nil {
		t.Fatalf("crear HMAC provisional autentico: %v", err)
	}
	firmadaEn := ancla.Add(-50*time.Second + time.Duration(indice)*10*time.Second)
	validadaEn := firmadaEn.Add(time.Second)
	validacionRef := "validacion:e2e:postgresql:v3:" + prefijo + ":inicial-final"
	validacionHuella := huellaSHA256E2E("validacion PAdES baseline B E2E " + prefijo)
	return dominiobolsa.FirmaDecisionTecnica{
		FirmanteRef: contenido.DecisorRef, PerfilFirmanteClave: contenido.PerfilDecisorClave,
		PoliticaFirmaRef: "politica:firma:e2e:postgresql:v3", PoliticaFirmaVersion: 1,
		HuellaPoliticaFirmaSHA256: huellaSHA256E2E("politica de firma PAdES baseline B E2E PostgreSQL V3"),
		PerfilFirmaAlcanzadoClave: puertosbolsa.PerfilFirmaPAdESBaselineB,
		RequiereFirmaInteractiva:  true, RequiereValidacionServidor: true,
		SesionFirmaInteractivaRef:             "sesion:firma:e2e:postgresql:v3:" + prefijo,
		HuellaEvidenciaFirmaInteractivaSHA256: huellaSHA256E2E("evidencia firma interactiva E2E " + prefijo),
		DocumentoFirmableRef:                  "documento:firmable:e2e:postgresql:v3:" + prefijo,
		VersionDocumentoFirmable:              "version-e2e-1",
		HuellaDocumentoFirmableSHA256:         huellaSHA256E2E("documento firmable canonico E2E " + prefijo),
		EvidenciaCustodiaRef:                  "evidencia:custodia:firmable:e2e:postgresql:v3:" + prefijo,
		FirmaRef:                              "firma:e2e:postgresql:v3:" + prefijo,
		HuellaFirmaSHA256:                     huellaSHA256E2E("artefacto PAdES baseline B E2E " + prefijo),
		DocumentoFirmadoRef:                   "documento:firmado:e2e:postgresql:v3:" + prefijo,
		HuellaDocumentoSHA256:                 huellaSHA256E2E("documento PDF firmado E2E " + prefijo),
		DocumentoFirmadoCustodiadoRef:         "objeto:firmado:custodiado:e2e:postgresql:v3:" + prefijo,
		VersionDocumentoFirmadoCustodiado:     "version-firmada-e2e-1",
		EvidenciaRecuperacionFirmadoRef:       "evidencia:recuperacion:firmado:e2e:postgresql:v3:" + prefijo,
		HuellaEvidenciaRecuperacionSHA256:     huellaSHA256E2E("recuperacion documento firmado E2E " + prefijo),
		EvidenciaCustodiaDocumentoFirmadoRef:  "evidencia:custodia:firmado:e2e:postgresql:v3:" + prefijo,
		EvidenciaRetencionDocumentoFirmadoRef: "evidencia:retencion:firmado:e2e:postgresql:v3:" + prefijo,
		PoliticaRetencionDocumentoFirmadoRef:  "politica:retencion:firmado:e2e:postgresql:v3",
		DocumentoFirmadoRetenidoHasta:         ancla.Add(365 * 24 * time.Hour),
		ManifiestoProbatorioRef:               "manifiesto:e2e:postgresql:v3:" + prefijo,
		HuellaManifiestoProbatorioSHA256:      huellaSHA256E2E("manifiesto provisional autentico E2E " + prefijo),
		SelloManifiestoProbatorioHMACSHA256:   selloProvisional,
		HuellaContenidoSHA256:                 huellaContenido,
		ValidacionInicialFirmaRef:             validacionRef,
		HuellaValidacionInicialSHA256:         validacionHuella, ValidadaInicialEn: validadaEn,
		ValidacionFirmaRef: validacionRef, HuellaValidacionSHA256: validacionHuella,
		ValidadaEn: validadaEn, FirmadaEn: firmadaEn,
	}
}

func incorporarDecisionInicialBolsaPostgreSQLE2E(
	t *testing.T,
	base dominiobolsa.BaremacionMerito,
	version puertosbolsa.ReferenciaVersionBaremacion,
	contextoConfirmacion puertosbolsa.ContextoOperacionBaremacion,
	contextoPrevalidacion puertosbolsa.ContextoOperacionBaremacion,
	ancla time.Time,
) (dominiobolsa.BaremacionMerito, puertosbolsa.ManifiestoProbatorioBaremacion) {
	t.Helper()
	proyeccion := contextoConfirmacion.Proyeccion()
	propuesta := dominiobolsa.PropuestaDecisionTecnica{
		ID: "decision:tecnica:e2e:postgresql:v3:1", CalculoOficial: base.CalculoInicial,
		PuntosReconocidos: 4_000_000, Resultado: dominiobolsa.ResultadoAceptado,
		DecisorRef: proyeccion.PrincipalRef, PerfilDecisorClave: proyeccion.PerfilActorClave,
		ValoracionesEvidencia: []dominiobolsa.ValoracionEvidencia{{
			Evidencia: base.EvidenciasIniciales[0], Estado: dominiobolsa.EstadoEvidenciaApta,
			ResultadoSubsanacion: dominiobolsa.ResultadoSubsanacionNoAplica,
			MotivoClave:          "documento_valido", Motivo: "Documento autentico, integro y suficiente.",
		}},
		MotivoClave: "valoracion_inicial", Motivo: "Valoracion E2E conforme al criterio publicado.",
		FuentesNormativasRefs: []string{"norma:e2e:postgresql:v3:baremo:1"},
		AutorizacionRef:       "autorizacion:manifiesto:e2e:postgresql:v3:adopcion:1",
		FinalidadClave:        proyeccion.FinalidadClave, CorrelacionRef: proyeccion.CorrelacionRef,
		DecididaEn: ancla.Add(-60 * time.Second),
	}
	contenido, err := base.PrepararDecisionInicial(propuesta)
	if err != nil {
		t.Fatalf("preparar decision inicial E2E: %v", err)
	}
	firma := firmaBaselineBBolsaPostgreSQLE2E(t, contenido, 1, ancla)
	manifiesto := manifiestoBaselineBBolsaPostgreSQLE2E(
		t, version, contenido, firma,
		contextoPrevalidacion.Proyeccion().AutorizacionRef,
		contextoConfirmacion.Proyeccion().AutorizacionRef,
		1, ancla,
	)
	firma.ManifiestoProbatorioRef = manifiesto.Referencia
	firma.HuellaManifiestoProbatorioSHA256 = manifiesto.HuellaManifiestoSHA256
	firma.SelloManifiestoProbatorioHMACSHA256 = manifiesto.SelloManifiestoHMACSHA256
	decision, err := dominiobolsa.ConstituirDecisionFirmada(contenido, firma)
	if err != nil {
		t.Fatalf("constituir decision inicial E2E: %v", err)
	}
	actualizada, err := base.IncorporarDecision(decision)
	if err != nil {
		t.Fatalf("incorporar decision inicial E2E: %v", err)
	}
	if err = manifiesto.ValidarCoberturaFirmaPara(version, contenido, firma); err != nil {
		t.Fatalf("cobertura 18/16 E2E invalida: %v", err)
	}
	return actualizada, manifiesto
}

func incorporarRectificacionBolsaPostgreSQLE2E(
	t *testing.T,
	base dominiobolsa.BaremacionMerito,
	version puertosbolsa.ReferenciaVersionBaremacion,
	contextoConfirmacion puertosbolsa.ContextoOperacionBaremacion,
	contextoPrevalidacion puertosbolsa.ContextoOperacionBaremacion,
	ancla time.Time,
) (dominiobolsa.BaremacionMerito, puertosbolsa.ManifiestoProbatorioBaremacion) {
	t.Helper()
	ultima, existe := base.UltimaDecision()
	if !existe {
		t.Fatal("la rectificacion E2E requiere una decision previa")
	}
	proyeccion := contextoConfirmacion.Proyeccion()
	propuesta := dominiobolsa.PropuestaDecisionTecnica{
		ID: "decision:tecnica:e2e:postgresql:v3:2", CalculoOficial: base.CalculoInicial,
		PuntosReconocidos: 4_100_000, Resultado: dominiobolsa.ResultadoAceptado,
		DecisorRef: proyeccion.PrincipalRef, PerfilDecisorClave: proyeccion.PerfilActorClave,
		ValoracionesEvidencia: []dominiobolsa.ValoracionEvidencia{{
			Evidencia: base.EvidenciasIniciales[0], Estado: dominiobolsa.EstadoEvidenciaApta,
			ResultadoSubsanacion: dominiobolsa.ResultadoSubsanacionNoAplica,
			MotivoClave:          "documento_valido", Motivo: "Documento revalorado y apto en rectificacion E2E.",
		}},
		MotivoClave:           "rectificacion_puntuacion",
		Motivo:                "Rectificacion E2E motivada de la puntuacion reconocida.",
		FuentesNormativasRefs: []string{"norma:e2e:postgresql:v3:baremo:1"},
		AutorizacionRef:       "autorizacion:manifiesto:e2e:postgresql:v3:adopcion:2",
		FinalidadClave:        proyeccion.FinalidadClave, CorrelacionRef: proyeccion.CorrelacionRef,
		DecididaEn: ancla.Add(-35 * time.Second),
	}
	if propuesta.DecididaEn.Before(ultima.Firma.FirmadaEn) {
		t.Fatal("cronologia de rectificacion E2E invalida")
	}
	contenido, err := base.PrepararRectificacion(propuesta)
	if err != nil {
		t.Fatalf("preparar rectificacion E2E: %v", err)
	}
	firma := firmaBaselineBBolsaPostgreSQLE2E(t, contenido, 2, ancla)
	manifiesto := manifiestoBaselineBBolsaPostgreSQLE2E(
		t, version, contenido, firma,
		contextoPrevalidacion.Proyeccion().AutorizacionRef,
		contextoConfirmacion.Proyeccion().AutorizacionRef,
		2, ancla,
	)
	firma.ManifiestoProbatorioRef = manifiesto.Referencia
	firma.HuellaManifiestoProbatorioSHA256 = manifiesto.HuellaManifiestoSHA256
	firma.SelloManifiestoProbatorioHMACSHA256 = manifiesto.SelloManifiestoHMACSHA256
	decision, err := dominiobolsa.ConstituirDecisionFirmada(contenido, firma)
	if err != nil {
		t.Fatalf("constituir rectificacion E2E: %v", err)
	}
	actualizada, err := base.IncorporarDecision(decision)
	if err != nil {
		t.Fatalf("incorporar rectificacion E2E: %v", err)
	}
	if err = manifiesto.ValidarCoberturaFirmaPara(version, contenido, firma); err != nil {
		t.Fatalf("cobertura 18/16 de rectificacion E2E invalida: %v", err)
	}
	return actualizada, manifiesto
}
