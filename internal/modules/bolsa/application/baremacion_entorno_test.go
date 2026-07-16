package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type entornoBaremacionPrueba struct {
	servicio    *ServicioBaremacion
	repositorio *repositorioBaremacionPrueba
	autorizador *autorizadorBaremacionPrueba
	sesiones    *sesionesBaremacionPrueba
	calculo     *calculadorBaremacionPrueba
	politicas   *politicasBaremacionPrueba
	almacen     *almacenBaremacionPrueba
	firmador    *firmadorBaremacionPrueba
	recuperador *recuperadorBinarioBaremacionPrueba
	validador   *validadorBaremacionPrueba
}

func nuevoEntornoBaremacionPrueba(t *testing.T) *entornoBaremacionPrueba {
	t.Helper()
	reloj := relojBaremacionPrueba{instante: instanteBaremacionPrueba}
	version, resultadoCalculo, fuente := versionInicialBaremacionPrueba(t)
	repositorio := &repositorioBaremacionPrueba{version: version, token: tokenReservaBaremacionPrueba(t)}
	autorizador := &autorizadorBaremacionPrueba{ahora: instanteBaremacionPrueba}
	contextoActor, vinculo := contextoYVinculoAutenticacionAplicacionPrueba(instanteBaremacionPrueba)
	sesion, err := NuevaSesionAutenticadaBaremacion(contextoActor, vinculo)
	if err != nil {
		t.Fatalf("NuevaSesionAutenticadaBaremacion() error = %v", err)
	}
	sesiones := &sesionesBaremacionPrueba{sesiones: []SesionAutenticadaBaremacion{sesion}}
	calculador := &calculadorBaremacionPrueba{resultado: resultadoCalculo}
	politicas := &politicasBaremacionPrueba{politica: politicaBaremacionPrueba()}
	codificador := &codificadorBaremacionPrueba{}
	almacen := &almacenBaremacionPrueba{ahora: instanteBaremacionPrueba}
	firmador := &firmadorBaremacionPrueba{ahora: instanteBaremacionPrueba}
	recuperador := &recuperadorBinarioBaremacionPrueba{ahora: instanteBaremacionPrueba}
	validador := &validadorBaremacionPrueba{ahora: instanteBaremacionPrueba}
	servicio, err := NuevoServicioBaremacion(
		repositorio, fuente, calculador, politicas, codificador, almacen, firmador, recuperador, validador,
		selladorTiempoBaremacionPrueba{}, aumentadorBaremacionPrueba{}, selladorSolicitudBaremacionPrueba{},
		seudonimizadorBaremacionPrueba{}, &generadorBaremacionPrueba{}, autorizador, sesiones, reloj,
		OpcionesServicioBaremacion{
			DuracionReserva: 5 * time.Second, DuracionFirma: 5 * time.Minute,
			ClasificacionDocumental: "datos_personales_alta", ConectorAlmacenPermitido: "almacen-cifrado-prueba",
			PoliticaRetencionRef: "politica:retencion:baremacion:v1", DuracionRetencion: 3650 * 24 * time.Hour,
		},
	)
	if err != nil {
		t.Fatalf("NuevoServicioBaremacion() error = %v", err)
	}
	return &entornoBaremacionPrueba{
		servicio: servicio, repositorio: repositorio, autorizador: autorizador, sesiones: sesiones,
		calculo: calculador, politicas: politicas, almacen: almacen, firmador: firmador, recuperador: recuperador, validador: validador,
	}
}

func prepararFirmaBaremacionPrueba(t *testing.T, entorno *entornoBaremacionPrueba) FirmaBaremacionPreparada {
	t.Helper()
	ctx := context.Background()
	iniciada, err := entorno.servicio.IniciarRevision(ctx, ordenIniciarBaremacionPrueba())
	if err != nil {
		t.Fatal(err)
	}
	adoptada, err := entorno.servicio.AdoptarDecision(ctx, ordenAdoptarBaremacionPrueba(iniciada, entorno.calculo.resultado.Calculo))
	if err != nil {
		t.Fatal(err)
	}
	codificada, err := entorno.servicio.CodificarDecision(ctx, OrdenCodificarDecisionBaremacion{
		Actor: actorBaremacionPrueba(), Revision: adoptada, PoliticaFirmaRef: entorno.politicas.politica.Referencia,
		PoliticaFirmaVersion: entorno.politicas.politica.Version, HuellaPoliticaSHA256: entorno.politicas.politica.HuellaSHA256,
	})
	if err != nil {
		t.Fatal(err)
	}
	custodiada, err := entorno.servicio.CustodiarDecision(ctx, OrdenCustodiarDecisionBaremacion{
		Actor: actorBaremacionPrueba(), Decision: codificada, OperacionRef: "operacion:decision:pendiente",
		ClaveIdempotencia: "idempotencia:custodia:pendiente", CargaRef: "carga:decision:pendiente",
	})
	if err != nil {
		t.Fatal(err)
	}
	preparada, err := entorno.servicio.PrepararFirma(ctx, OrdenPrepararFirmaBaremacion{
		Actor: actorBaremacionPrueba(), Decision: custodiada, OperacionRef: "operacion:firma:pendiente",
		ClaveIdempotencia: "idempotencia:firma:pendiente",
	})
	if err != nil {
		t.Fatal(err)
	}
	return preparada
}

func ordenFinalizarBaremacionPrueba(
	firma FirmaBaremacionPreparada,
	sufijo string,
) OrdenFinalizarFirmaBaremacion {
	return OrdenFinalizarFirmaBaremacion{
		Actor: actorBaremacionPrueba(), Firma: firma, OperacionRef: "operacion:finalizar:" + sufijo,
		OperacionCustodiaRef:      "operacion:custodia:firmado:" + sufijo,
		ClaveIdempotenciaCustodia: "idempotencia:custodia:firmado:" + sufijo,
		CargaDocumentoFirmadoRef:  "carga:documento:firmado:" + sufijo,
		ClaveIdempotenciaReserva:  "idempotencia:reserva:confirmacion:" + sufijo,
		MotivoClaveConfirmacion:   "decision_tecnica_firmada",
		MotivoConfirmacion:        "Incorporacion de la decision tecnica validada y firmada.",
	}
}

func artefactoFirmadoBaremacionPrueba(
	t *testing.T,
	firma FirmaBaremacionPreparada,
) puertosbolsa.ArtefactoFirma {
	t.Helper()
	documento := firma.decision.documento
	contenido := firma.decision.decision.revision.contenido
	politica := firma.decision.decision.politica
	binarioFirmado := contenidoDocumentoFirmadoBaremacionPrueba()
	huellaBinario := sha256.Sum256(binarioFirmado)
	artefacto := puertosbolsa.ArtefactoFirma{
		ProcesoRef: documento.ProcesoRef, SolicitudRef: documento.SolicitudRef, SujetoRef: documento.SujetoRef,
		BaremacionMeritoRef: documento.BaremacionMeritoRef, DecisionRef: documento.DecisionRef,
		VersionBaremacion: documento.VersionBaremacion, SesionFirmaRef: firma.sesion.SesionRef,
		EvidenciaFirmaInteractivaRef:     "evidencia:firma:interactiva:cruce:1",
		HuellaEvidenciaInteractivaSHA256: huellaBaremacionPrueba("9"),
		DocumentoFirmable:                documento.Objeto.Objeto,
		HuellaDocumentoFirmableSHA256:    documento.HuellaDocumentoSHA256,
		EvidenciaCustodiaRef:             documento.EvidenciaCustodia.Referencia,
		FirmaRef:                         "firma:decision:cruce:1",
		HuellaFirmaSHA256:                huellaBaremacionPrueba("a"),
		DocumentoFirmadoRef:              "documento:firmado:cruce:1",
		HuellaDocumentoSHA256:            hex.EncodeToString(huellaBinario[:]),
		HuellaContenidoSHA256:            documento.HuellaContenidoSHA256,
		PoliticaFirmaRef:                 politica.Referencia,
		PoliticaFirmaVersion:             politica.Version,
		HuellaPoliticaFirmaSHA256:        politica.HuellaSHA256,
		FirmanteRef:                      contenido.DecisorRef,
		PerfilFirmanteClave:              contenido.PerfilDecisorClave,
		FirmadaEn:                        instanteBaremacionPrueba,
	}
	if err := artefacto.ValidarPara(firma.solicitud, firma.sesion); err != nil {
		t.Fatalf("artefacto de prueba invalido: %v", err)
	}
	return artefacto
}

func autorizarContextoFirmaBaremacionPrueba(
	t *testing.T,
	entorno *entornoBaremacionPrueba,
	contenido dominiobolsa.ContenidoDecisionTecnica,
	documentoFirmadoRef string,
	accion puertosbolsa.AccionOperacionBaremacion,
	sufijo string,
) puertosbolsa.ContextoOperacionFirma {
	t.Helper()
	capacidad, err := entorno.servicio.autorizar(
		context.Background(), actorBaremacionPrueba(), accion,
		puertosbolsa.ClaseRecursoDocumentoFirmado, documentoFirmadoRef,
		contenido.SujetoRef, contenido.FinalidadClave, contenido.CorrelacionRef,
	)
	if err != nil {
		t.Fatalf("autorizar %s: %v", accion, err)
	}
	contexto := puertosbolsa.ContextoOperacionFirma{
		ContextoOperacionBaremacion: capacidad,
		OperacionRef:                "operacion:contexto:" + sufijo,
	}
	if err := contexto.Validar(); err != nil {
		t.Fatalf("contexto de %s invalido: %v", accion, err)
	}
	return contexto
}

func autorizadorFijoCustodiaBinarioBaremacionPrueba(
	recuperacion puertosbolsa.ContextoOperacionFirma,
	custodia puertosbolsa.ContextoOperacionFirma,
	retencion puertosbolsa.ContextoOperacionFirma,
) func(
	puertosbolsa.AccionOperacionBaremacion,
	puertosbolsa.ClaseRecursoOperacionBaremacion,
	string,
	string,
) (puertosbolsa.ContextoOperacionFirma, error) {
	return func(
		accion puertosbolsa.AccionOperacionBaremacion,
		_ puertosbolsa.ClaseRecursoOperacionBaremacion,
		_ string,
		_ string,
	) (puertosbolsa.ContextoOperacionFirma, error) {
		switch accion {
		case puertosbolsa.AccionRecuperarBinarioFirmadoBaremacion:
			return recuperacion, nil
		case puertosbolsa.AccionCustodiarDocumentoFirmadoBaremacion:
			return custodia, nil
		case puertosbolsa.AccionRetenerDocumentoFirmadoBaremacion:
			return retencion, nil
		default:
			return puertosbolsa.ContextoOperacionFirma{}, dominiovec.ErrAutorizacionDenegada
		}
	}
}

func autorizadorAlmacenServicioBaremacionPrueba(
	entorno *entornoBaremacionPrueba,
	contenido dominiobolsa.ContenidoDecisionTecnica,
) func(
	puertosbolsa.AccionOperacionBaremacion,
	puertosbolsa.ClaseRecursoOperacionBaremacion,
	string,
	string,
	dominiovec.RecursoAutorizable,
) (puertosbolsa.ContextoOperacionFirma, error) {
	return func(
		accion puertosbolsa.AccionOperacionBaremacion,
		clase puertosbolsa.ClaseRecursoOperacionBaremacion,
		recursoRef string,
		sufijo string,
		recurso dominiovec.RecursoAutorizable,
	) (puertosbolsa.ContextoOperacionFirma, error) {
		capacidad, err := entorno.servicio.autorizarAlmacen(
			context.Background(), actorBaremacionPrueba(), accion, clase, recursoRef,
			contenido.SujetoRef, contenido.FinalidadClave, contenido.CorrelacionRef, recurso,
		)
		if err != nil {
			return puertosbolsa.ContextoOperacionFirma{}, err
		}
		return puertosbolsa.ContextoOperacionFirma{
			ContextoOperacionBaremacion: capacidad,
			OperacionRef:                "operacion:contexto:" + sufijo,
		}, nil
	}
}

func sesionBaremacionAutenticacionAlternativaPrueba(t *testing.T) SesionAutenticadaBaremacion {
	t.Helper()
	contextoActor, vinculoBase := contextoYVinculoAutenticacionAplicacionPrueba(instanteBaremacionPrueba)
	datos, err := vinculoBase.Datos()
	if err != nil {
		t.Fatal(err)
	}
	autenticacion := datos.Autenticacion()
	autenticacion.AutenticacionRef = "aut_autenticacion_custodia_alternativa"
	autenticacion.AsercionRef = "ase_asercion_custodia_alternativa"
	autenticacion.SesionRef = "ses_sesion_custodia_alternativa"
	autenticacion.ControlSesionRef = "cse_control_custodia_alternativa"
	vinculo, err := dominiovec.CrearVinculoAutenticacionActorV1(
		context.Background(),
		revalidadorVinculoAutenticacionAplicacionPrueba{resultado: autenticacion},
		dominiovec.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef,
			SesionRef:        autenticacion.SesionRef,
		},
		contextoActor,
		instanteBaremacionPrueba,
	)
	if err != nil {
		t.Fatalf("crear vinculo alternativo de custodia: %v", err)
	}
	sesion, err := NuevaSesionAutenticadaBaremacion(contextoActor, vinculo)
	if err != nil {
		t.Fatalf("crear sesion alternativa de custodia: %v", err)
	}
	return sesion
}

func ordenIniciarBaremacionPrueba() OrdenIniciarRevisionBaremacion {
	return OrdenIniciarRevisionBaremacion{
		Actor: actorBaremacionPrueba(), BaremacionMeritoRef: "baremacion-001", SujetoRef: "sujeto-001",
		Finalidad:      "baremacion_proceso_selectivo",
		CorrelacionRef: "correlacion:baremacion:001",
	}
}

func actorBaremacionPrueba() ActorBaremacion {
	return ActorBaremacion{Motivo: "Revision tecnica de autobaremacion."}
}

func ordenAdoptarBaremacionPrueba(
	revision RevisionBaremacionIniciada,
	calculo dominiobolsa.CalculoOficialBaremacion,
) OrdenAdoptarDecisionBaremacion {
	return OrdenAdoptarDecisionBaremacion{
		Actor: actorBaremacionPrueba(), Revision: revision, Clase: dominiobolsa.ClaseDecisionInicial,
		CalculoRef: calculo.CalculoRef, HuellaResultadoCalculo: calculo.HuellaResultadoSHA256,
		PuntosReconocidos: calculo.PuntosCalculados, Resultado: dominiobolsa.ResultadoAceptado,
		ValoracionesEvidencia: []dominiobolsa.ValoracionEvidencia{{
			Evidencia: calculo.Evidencias[0], Estado: dominiobolsa.EstadoEvidenciaApta,
			ResultadoSubsanacion: dominiobolsa.ResultadoSubsanacionNoAplica,
			MotivoClave:          "documento_valido", Motivo: "Documento autentico, pertinente y suficiente.",
		}},
		MotivoClave: "valoracion_inicial", MotivoDecision: "Valoracion conforme al criterio publicado.",
		FuentesNormativasRefs: []string{"norma-baremo-v7"},
	}
}

func versionInicialBaremacionPrueba(t *testing.T) (
	puertosbolsa.VersionBaremacion,
	puertosbolsa.ResultadoCalculoOficial,
	*fuenteBaremacionPrueba,
) {
	t.Helper()
	criterio := dominiobolsa.ReferenciaCriterio{
		ProcesoRef: "proceso-selectivo-2026-017", Clave: "experiencia.entidad_publica.grupo_c1", Version: 7,
		HuellaSHA256: huellaBaremacionPrueba("a"), PuntosMaximos: 10 * dominiobolsa.UnidadesPorPunto,
		ReglaCalculo: dominiobolsa.ReferenciaReglaCalculo{
			Clave: "experiencia_publica_dias", Version: 3, HuellaSHA256: huellaBaremacionPrueba("9"),
		},
	}
	evidencia := dominiobolsa.EvidenciaMerito{Referencia: dominiobolsa.ReferenciaEvidencia{
		DocumentoRef: "documento-001", VersionDocumento: 1, RepresentacionRef: "representacion-001",
		HuellaSHA256: huellaBaremacionPrueba("b"),
	}}
	calculo := dominiobolsa.CalculoOficialBaremacion{
		CalculoRef: "calculo-oficial-inicial", ProcesoRef: criterio.ProcesoRef, SolicitudRef: "solicitud-001",
		SujetoRef: "sujeto-001", BaremacionMeritoRef: "baremacion-001", Criterio: criterio,
		Regla: criterio.ReglaCalculo, Evidencias: []dominiobolsa.EvidenciaMerito{evidencia},
		EntradaRef: "entrada-calculo-inicial", HuellaEntradaSHA256: huellaBaremacionPrueba("1"),
		PuntosCalculados: 4_250_000, DesgloseRef: "desglose-calculo-inicial",
		HuellaDesgloseSHA256: huellaBaremacionPrueba("2"), ResultadoRef: "resultado-calculo-inicial",
		HuellaResultadoSHA256: huellaBaremacionPrueba("3"), MotorCalculoRef: "motor-baremo-oficial",
		VersionMotorCalculo: "motor-v2.1.0", EvidenciaEjecucionRef: "ejecucion-calculo-inicial",
		HuellaEjecucionSHA256: huellaBaremacionPrueba("4"), CalculadoEn: instanteBaremacionPrueba.Add(-10 * time.Minute),
	}
	agregado, err := dominiobolsa.NuevaBaremacionMerito(dominiobolsa.AltaMeritoBaremable{
		ID: "baremacion-001", ProcesoRef: criterio.ProcesoRef, SolicitudRef: "solicitud-001", SujetoRef: "sujeto-001",
		Criterio: criterio, EvidenciasIniciales: []dominiobolsa.EvidenciaMerito{evidencia},
		PuntosDeclarados: 5_000_000, CalculoOficial: calculo, CreadaEn: instanteBaremacionPrueba.Add(-9 * time.Minute),
	})
	if err != nil {
		t.Fatalf("NuevaBaremacionMerito() error = %v", err)
	}
	huella, err := agregado.HuellaEstadoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	version := puertosbolsa.VersionBaremacion{
		Referencia: puertosbolsa.ReferenciaVersionBaremacion{
			BaremacionMeritoRef: agregado.ID, Numero: 1, HuellaEstadoSHA256: huella,
		},
		Agregado: agregado, ConfirmadaEn: instanteBaremacionPrueba.Add(-8 * time.Minute),
	}
	documento := documentoBaremacionPrueba()
	representacion := representacionBaremacionPrueba(documento, evidencia.Referencia)
	resultado := puertosbolsa.ResultadoCalculoOficial{
		Calculo: calculo, EvidenciaGobiernoRef: "evidencia-gobierno-calculo-1",
		HuellaEvidenciaSHA256: huellaBaremacionPrueba("5"),
	}
	fuente := &fuenteBaremacionPrueba{
		criterio: puertosbolsa.CriterioBaremacionConfiable{
			Referencia: criterio, PublicacionRef: "publicacion-criterio-v7",
			HuellaPublicacionSHA256: huellaBaremacionPrueba("6"), EvidenciaConsultaRef: "consulta-criterio-1",
			HuellaEvidenciaSHA256: huellaBaremacionPrueba("7"), ConsultadoEn: instanteBaremacionPrueba,
		},
		evidencia: puertosbolsa.EvidenciaBaremacionConfiable{
			Evidencia: evidencia, Documento: documento, VerificacionPertenenciaRef: "pertenencia-documento-1",
			HuellaVerificacionSHA256: huellaBaremacionPrueba("8"), VerificadaEn: instanteBaremacionPrueba,
		},
		representacion: puertosbolsa.RepresentacionBaremacionConfiable{
			Representacion: representacion, EvidenciaConsultaRef: "consulta-representacion-1",
			HuellaEvidenciaSHA256: huellaBaremacionPrueba("c"), ConsultadaEn: instanteBaremacionPrueba,
		},
	}
	return version, resultado, fuente
}

func documentoBaremacionPrueba() dominiovec.DocumentoLogico {
	return dominiovec.DocumentoLogico{
		ID: "documento-001", Version: 1, Revision: 1,
		Plantilla: dominiovec.ReferenciaPlantillaDocumento{
			ID: "plantilla_baremo", Version: 7, HuellaSHA256: huellaBaremacionPrueba("d"),
		},
		ModuloID: "bolsa", TipoDocumental: "merito", Clasificacion: "datos_personales_alta",
		Relaciones: []dominiovec.RelacionDocumento{
			{Tipo: dominiovec.TipoRelacionPersona, Referencia: "sujeto-001", Rol: "interesada"},
			{Tipo: dominiovec.TipoRelacionExpediente, Referencia: "solicitud-001", Rol: "principal"},
		},
		Estado:           dominiovec.EstadoDocumentoLogicoBorrador,
		HuellaDatosHMAC:  "hmac-sha256:documentos_1:" + huellaBaremacionPrueba("e"),
		HuellaFuenteHMAC: "hmac-sha256:documentos_1:" + huellaBaremacionPrueba("f"),
		CreadoPor:        "tecnico-rrhh-17", CreadoEn: instanteBaremacionPrueba.Add(-time.Hour),
		CorrelacionRef: "correlacion:baremacion:001", Motivo: "Evidencia aportada para baremacion.",
		ENI: dominiovec.MetadatosENI{
			Identificador: "documento-001", Organo: "DIPUTACION-GRANADA", Origen: "ciudadano",
			EstadoElaboracion: "original", TipoDocumental: "merito", FechaCaptura: instanteBaremacionPrueba.Add(-time.Hour),
		},
	}
}

func representacionBaremacionPrueba(
	documento dominiovec.DocumentoLogico,
	referencia dominiobolsa.ReferenciaEvidencia,
) dominiovec.RepresentacionDocumento {
	return dominiovec.RepresentacionDocumento{
		ID: referencia.RepresentacionRef, Documento: dominiovec.ReferenciaDocumento{ID: documento.ID, Version: documento.Version},
		Tipo: dominiovec.TipoRepresentacionVisualizacion, Formato: dominiovec.FormatoDocumentoPDF,
		MIME: "application/pdf", NombreFichero: "merito.pdf", Tamano: 1024,
		HuellaContenidoSHA256: referencia.HuellaSHA256, HuellaFuenteHMAC: documento.HuellaFuenteHMAC,
		ReferenciaContenido: "objeto:merito:001", EstadoTecnico: dominiovec.EstadoRepresentacionDisponible,
		EstadoAntivirus: dominiovec.EstadoAntivirusLimpio, GeneradaPor: "sistema-documental",
		GeneradaEn: instanteBaremacionPrueba.Add(-50 * time.Minute),
	}
}

func politicaBaremacionPrueba() puertosbolsa.PoliticaFirmaBaremacion {
	return puertosbolsa.PoliticaFirmaBaremacion{
		Referencia: "politica-firma-baremacion", Version: 4, HuellaSHA256: huellaBaremacionPrueba("1"),
		FormatoFirmaClave: puertosbolsa.FormatoFirmaPDFCanonico, PerfilFirmaClave: puertosbolsa.PerfilFirmaPAdESBaselineB,
		AlgoritmoHuellaClave:       puertosbolsa.AlgoritmoHuellaFirmaSHA256,
		ComprobacionesObligatorias: puertosbolsa.ComprobacionesFirmaObligatorias(), RequiereFirmaInteractiva: true,
		RequiereValidacionServidor: true, AprobacionRef: "aprobacion-politica-firma-4",
		HuellaAprobacionSHA256: huellaBaremacionPrueba("2"),
		VigenteDesde:           instanteBaremacionPrueba.Add(-time.Hour), VigenteHasta: instanteBaremacionPrueba.Add(time.Hour),
	}
}
