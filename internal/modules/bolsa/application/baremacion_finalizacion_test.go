package application

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type selladorCentinelaPorFinalidadBaremacionPrueba struct {
	finalidad []byte
}

func (s selladorCentinelaPorFinalidadBaremacionPrueba) SellarSolicitudBaremacion(
	ctx context.Context,
	carga puertosbolsa.CargaProtegida,
) (string, error) {
	material := carga.Revelar()
	defer func() {
		for posicion := range material {
			material[posicion] = 0
		}
	}()
	if bytes.Contains(material, s.finalidad) {
		return hmacBaremacionPendiente, nil
	}
	return selladorSolicitudBaremacionPrueba{}.SellarSolicitudBaremacion(ctx, carga)
}

func (s selladorCentinelaPorFinalidadBaremacionPrueba) SellarSelloBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudSellarSelloBaremacion,
) (string, error) {
	if bytes.Equal([]byte(solicitud.Finalidad), s.finalidad) {
		return hmacBaremacionPendiente, nil
	}
	return selladorSolicitudBaremacionPrueba{}.SellarSelloBaremacion(ctx, solicitud)
}

func TestServicioBaremacionNoPersisteHMACCentinelaDevueltoPorSellador(t *testing.T) {
	casos := []struct {
		nombre                  string
		finalidad               puertosbolsa.FinalidadSelloBaremacion
		reservasEsperadas       int
		confirmacionesEsperadas int
		abandonosEsperados      int
	}{
		{"reserva", puertosbolsa.FinalidadSelloReservaBaremacion, 0, 0, 0},
		{"manifiesto", puertosbolsa.FinalidadSelloManifiestoProbatorioBaremacionV3, 1, 0, 1},
		{"confirmacion", puertosbolsa.FinalidadSelloConfirmacionBaremacionV2, 1, 0, 1},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			entorno := nuevoEntornoBaremacionPrueba(t)
			preparada := prepararFirmaBaremacionPrueba(t, entorno)
			entorno.servicio.selladorSolicitud = selladorCentinelaPorFinalidadBaremacionPrueba{
				finalidad: []byte(caso.finalidad),
			}

			_, err := entorno.servicio.FinalizarFirma(
				context.Background(), ordenFinalizarBaremacionPrueba(preparada, "centinela-"+caso.nombre),
			)
			if !errors.Is(err, ErrResultadoBaremacionNoConfiable) {
				t.Fatalf("FinalizarFirma() error = %v", err)
			}
			if entorno.repositorio.reservas != caso.reservasEsperadas ||
				entorno.repositorio.confirmaciones != caso.confirmacionesEsperadas ||
				entorno.repositorio.abandonos != caso.abandonosEsperados {
				t.Fatalf("efectos del centinela: reservas=%d confirmaciones=%d abandonos=%d",
					entorno.repositorio.reservas, entorno.repositorio.confirmaciones,
					entorno.repositorio.abandonos)
			}
		})
	}
}

func TestServicioBaremacionProduceManifiestoVerificableYSeparaFinalidades(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	preparada := prepararFirmaBaremacionPrueba(t, entorno)
	resultado, err := entorno.servicio.FinalizarFirma(
		context.Background(), ordenFinalizarBaremacionPrueba(preparada, "vertical-hmac"),
	)
	if err != nil {
		t.Fatalf("finalizar con productor y verificador compartidos: %v", err)
	}
	if entorno.repositorio.confirmacion == nil || entorno.repositorio.confirmacion.Manifiesto == nil {
		t.Fatal("el repositorio no recibio el manifiesto producido por el servicio")
	}
	manifiesto := entorno.repositorio.confirmacion.Manifiesto.Clonar()
	representacion, err := puertosbolsa.RepresentacionCanonicaManifiestoProbatorioBaremacion(manifiesto)
	if err != nil {
		t.Fatal(err)
	}
	verificador := selladorSolicitudBaremacionPrueba{}
	peticion := puertosbolsa.SolicitudVerificarSelloBaremacion{
		Finalidad:              puertosbolsa.FinalidadSelloManifiestoProbatorioBaremacionV3,
		RepresentacionCanonica: representacion,
		SelloHMAC:              manifiesto.SelloManifiestoHMACSHA256,
	}
	if err := verificador.VerificarSelloBaremacion(context.Background(), peticion); err != nil {
		t.Fatalf("el verificador rechazo el sello producido por ServicioBaremacion: %v", err)
	}
	peticion.Finalidad = puertosbolsa.FinalidadSelloConfirmacionBaremacionV2
	if err := verificador.VerificarSelloBaremacion(context.Background(), peticion); !errors.Is(
		err, puertosbolsa.ErrSelloBaremacionNoAutentico,
	) {
		t.Fatalf("el sello de manifiesto se reutilizo en confirmacion: %v", err)
	}
	if resultado.Decision.Firma.SelloManifiestoProbatorioHMACSHA256 != manifiesto.SelloManifiestoHMACSHA256 {
		t.Fatal("el resultado no conserva el sello que verifico el repositorio")
	}
}

func TestServicioBaremacionIntegraServicioAutorizacionV1Real(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	contextoActor, vinculo := contextoYVinculoAutenticacionAplicacionPrueba(instanteBaremacionPrueba)
	campos, existe := puertosbolsa.CamposRequeridosOperacionBaremacion(
		puertosbolsa.AccionConsultarBaremacionVigente,
	)
	if !existe {
		t.Fatal("accion de prueba no catalogada")
	}
	versionRol := dominiovec.VersionRol{
		RolID: "tecnico_baremacion", Version: 1, Nombre: "Tecnico de baremacion",
		Estado: dominiovec.EstadoVersionRolPublicada,
		Concesiones: []dominiovec.ConcesionRol{{
			Accion: string(puertosbolsa.AccionConsultarBaremacionVigente), ModuloID: "bolsa",
			TipoRecurso: string(puertosbolsa.ClaseRecursoBaremacion),
			Finalidades: []string{"baremacion_proceso_selectivo"}, GarantiaMinima: dominiovec.AuthAssuranceHigh,
			CamposPermitidos: campos,
		}},
		PublicadaPor: "responsable-seguridad", PublicadaEn: instanteBaremacionPrueba.Add(-24 * time.Hour),
	}
	asignacion := dominiovec.AsignacionPerfil{
		AsignacionID: "asignacion_baremacion", Version: 1, PerfilActivoRef: contextoActor.PerfilActivoRef,
		PrincipalID: contextoActor.Principal.ID, VersionRolRef: versionRol.Referencia(),
		Estado:       dominiovec.EstadoAsignacionPerfilActiva,
		Ambitos:      []dominiovec.AmbitoPerfil{{Clave: "sujeto_ref", Valores: []string{"sujeto-001"}}},
		VigenteDesde: instanteBaremacionPrueba.Add(-time.Hour), VigenteHasta: instanteBaremacionPrueba.Add(time.Hour),
		EmitidaPor: "administrador-identidades", EmitidaEn: instanteBaremacionPrueba.Add(-2 * time.Hour),
	}
	huellaCatalogo, err := dominiovec.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	instantanea := dominiovec.InstantaneaAutorizacion{
		AsignacionPerfil: asignacion, VersionRol: versionRol,
		ControlVigenciaVersionRol: dominiovec.ControlVigenciaVersionRol{
			VersionRolRef: versionRol.Referencia(), Revision: 1,
			Estado:         dominiovec.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: "responsable-seguridad", ActualizadoEn: instanteBaremacionPrueba.Add(-24 * time.Hour),
		},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
	}
	fuente := &fuenteAutorizacionServicioPrueba{instantanea: instantanea}
	registro := &registroAutorizacionServicioPrueba{}
	autorizador := nuevoServicioAutorizacionServicioPrueba(
		t, fuente, registro,
		&generadorAutorizacionServicioPrueba{referencia: "decision:baremacion:v1:real"},
		instanteBaremacionPrueba,
	)
	entorno.servicio.autorizador = autorizador

	resultado, err := entorno.servicio.IniciarRevision(context.Background(), ordenIniciarBaremacionPrueba())
	if err != nil || validarRevisionIniciada(resultado) != nil {
		t.Fatalf("integracion PDP V1 real: resultado=%v error=%v", resultado.version.Referencia, err)
	}
	if registro.concesiones != 1 || fuente.invocaciones != 1 || entorno.repositorio.consultas != 1 {
		t.Fatalf("recorrido incompleto: concesiones=%d fuente=%d consultas=%d",
			registro.concesiones, fuente.invocaciones, entorno.repositorio.consultas)
	}
	if registro.decision.PrincipalID != contextoActor.Principal.ID ||
		registro.decision.PerfilActivoRef != contextoActor.PerfilActivoRef ||
		!registro.decision.VinculoAutenticacionActor.CoincideExactamenteCon(vinculo) {
		t.Fatal("la decision real no quedo ligada al actor y sesion exactos")
	}
}

func TestServicioBaremacionDeniegaObligacionDesconocidaAntesDeReservar(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	preparada := prepararFirmaBaremacionPrueba(t, entorno)
	entorno.autorizador.obligacionEn = puertosbolsa.AccionReservarDecisionBaremacion

	_, err := entorno.servicio.FinalizarFirma(context.Background(), ordenFinalizarBaremacionPrueba(preparada, "obligacion"))
	if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("FinalizarFirma() error = %v", err)
	}
	if entorno.repositorio.consultas != 1 || entorno.repositorio.reservas != 0 {
		t.Fatalf("la obligacion desconocida no fallo antes de reservar: consultas=%d reservas=%d",
			entorno.repositorio.consultas, entorno.repositorio.reservas)
	}
}

func TestServicioBaremacionDeniegaReferenciaAutorizacionReutilizadaAntesDeReservar(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	preparada := prepararFirmaBaremacionPrueba(t, entorno)
	entorno.autorizador.reutilizarReferencia = true

	_, err := entorno.servicio.FinalizarFirma(context.Background(), ordenFinalizarBaremacionPrueba(preparada, "reutilizada"))
	if !errors.Is(err, ErrResultadoBaremacionNoConfiable) {
		t.Fatalf("FinalizarFirma() error = %v", err)
	}
	if entorno.repositorio.consultas != 1 || entorno.repositorio.reservas != 0 {
		t.Fatalf("la referencia reutilizada no fallo antes de reservar: consultas=%d reservas=%d",
			entorno.repositorio.consultas, entorno.repositorio.reservas)
	}
}

func TestCustodiarBinarioFirmadoDeniegaCruceDeSesionAntesDeEfectosDeAlmacen(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	preparada := prepararFirmaBaremacionPrueba(t, entorno)
	contenido := preparada.decision.decision.revision.contenido
	artefacto := artefactoFirmadoBaremacionPrueba(t, preparada)
	orden := ordenFinalizarBaremacionPrueba(preparada, "cruce-sesion-custodia")

	contextoRecuperacion := autorizarContextoFirmaBaremacionPrueba(
		t, entorno, contenido, artefacto.DocumentoFirmadoRef,
		puertosbolsa.AccionRecuperarBinarioFirmadoBaremacion, "recuperacion",
	)
	entorno.sesiones.sesiones = []SesionAutenticadaBaremacion{
		sesionBaremacionAutenticacionAlternativaPrueba(t),
	}
	seudonimizador := &seudonimizadorContadorBaremacionPrueba{}
	entorno.servicio.seudonimizador = seudonimizador
	escriturasPrevias, retencionesPrevias := entorno.almacen.escrituras, entorno.almacen.retenciones

	_, err := entorno.servicio.custodiarBinarioFirmado(
		context.Background(), orden, contenido, artefacto,
		autorizadorFijoCustodiaBinarioBaremacionPrueba(
			contextoRecuperacion, puertosbolsa.ContextoOperacionFirma{}, puertosbolsa.ContextoOperacionFirma{},
		),
		autorizadorAlmacenServicioBaremacionPrueba(entorno, contenido),
		nil,
	)
	if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) ||
		!errors.Is(err, ErrResultadoBaremacionNoConfiable) {
		t.Fatalf("el cruce de sesion no fue denegado: %v", err)
	}
	if entorno.recuperador.llamadas != 0 || seudonimizador.llamadas != 0 ||
		entorno.almacen.escrituras != escriturasPrevias ||
		entorno.almacen.retenciones != retencionesPrevias {
		t.Fatalf(
			"el cruce de sesion produjo efectos: recuperaciones=%d seudonimizaciones=%d escrituras=%d/%d retenciones=%d/%d",
			entorno.recuperador.llamadas, seudonimizador.llamadas, entorno.almacen.escrituras, escriturasPrevias,
			entorno.almacen.retenciones, retencionesPrevias,
		)
	}
}

func TestCustodiarBinarioFirmadoDeniegaDecisionRefReutilizadaAntesDeEfectosDeAlmacen(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	preparada := prepararFirmaBaremacionPrueba(t, entorno)
	contenido := preparada.decision.decision.revision.contenido
	artefacto := artefactoFirmadoBaremacionPrueba(t, preparada)
	orden := ordenFinalizarBaremacionPrueba(preparada, "decision-reutilizada-custodia")
	entorno.autorizador.reutilizarReferencia = true

	contextoRecuperacion := autorizarContextoFirmaBaremacionPrueba(
		t, entorno, contenido, artefacto.DocumentoFirmadoRef,
		puertosbolsa.AccionRecuperarBinarioFirmadoBaremacion, "recuperacion",
	)
	seudonimizador := &seudonimizadorContadorBaremacionPrueba{}
	entorno.servicio.seudonimizador = seudonimizador
	escriturasPrevias, retencionesPrevias := entorno.almacen.escrituras, entorno.almacen.retenciones

	_, err := entorno.servicio.custodiarBinarioFirmado(
		context.Background(), orden, contenido, artefacto,
		autorizadorFijoCustodiaBinarioBaremacionPrueba(
			contextoRecuperacion, puertosbolsa.ContextoOperacionFirma{}, puertosbolsa.ContextoOperacionFirma{},
		),
		autorizadorAlmacenServicioBaremacionPrueba(entorno, contenido),
		nil,
	)
	if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) ||
		!errors.Is(err, ErrResultadoBaremacionNoConfiable) {
		t.Fatalf("la DecisionRef reutilizada no fue denegada: %v", err)
	}
	if entorno.recuperador.llamadas != 0 || seudonimizador.llamadas != 0 ||
		entorno.almacen.escrituras != escriturasPrevias ||
		entorno.almacen.retenciones != retencionesPrevias {
		t.Fatalf(
			"la DecisionRef reutilizada produjo efectos: recuperaciones=%d seudonimizaciones=%d escrituras=%d/%d retenciones=%d/%d",
			entorno.recuperador.llamadas, seudonimizador.llamadas, entorno.almacen.escrituras, escriturasPrevias,
			entorno.almacen.retenciones, retencionesPrevias,
		)
	}
}

func TestServicioBaremacionDeniegaCamposNoExactosAntesDeReservar(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	preparada := prepararFirmaBaremacionPrueba(t, entorno)
	entorno.autorizador.camposInvalidosEn = puertosbolsa.AccionReservarDecisionBaremacion

	_, err := entorno.servicio.FinalizarFirma(context.Background(), ordenFinalizarBaremacionPrueba(preparada, "campos"))
	if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("FinalizarFirma() error = %v", err)
	}
	if entorno.repositorio.consultas != 1 || entorno.repositorio.reservas != 0 {
		t.Fatalf("los campos incompletos no fallaron antes de reservar: consultas=%d reservas=%d",
			entorno.repositorio.consultas, entorno.repositorio.reservas)
	}
}

func TestServicioBaremacionNoConfirmaFirmaPendiente(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	preparada := prepararFirmaBaremacionPrueba(t, entorno)
	entorno.firmador.pendiente = true

	_, err := entorno.servicio.FinalizarFirma(context.Background(), OrdenFinalizarFirmaBaremacion{
		Actor: actorBaremacionPrueba(), Firma: preparada, OperacionRef: "operacion:finalizar:pendiente",
		OperacionCustodiaRef: "operacion:custodia:firmado:pendiente", ClaveIdempotenciaCustodia: "idempotencia:custodia:firmado:pendiente",
		CargaDocumentoFirmadoRef: "carga:documento:firmado:pendiente",
		ClaveIdempotenciaReserva: "idempotencia:reserva:confirmacion:pendiente",
		MotivoClaveConfirmacion:  "decision_tecnica_firmada",
		MotivoConfirmacion:       "Incorporacion de la decision tecnica validada y firmada.",
	})
	if !errors.Is(err, ErrFirmaBaremacionNoCompletada) {
		t.Fatalf("FinalizarFirma() error = %v", err)
	}
	if entorno.validador.llamadas != 0 || entorno.repositorio.confirmaciones != 0 {
		t.Fatalf("una firma pendiente produjo efectos: validaciones=%d confirmaciones=%d",
			entorno.validador.llamadas, entorno.repositorio.confirmaciones)
	}
}

func TestServicioBaremacionNoMantieneReservaDuranteLaRevisionHumana(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	_ = prepararFirmaBaremacionPrueba(t, entorno)
	if entorno.repositorio.reservas != 0 || entorno.repositorio.reserva != nil {
		t.Fatalf("la revision humana mantuvo una reserva OCC: reservas=%d", entorno.repositorio.reservas)
	}
}

func TestServicioBaremacionConservaDocumentoHuerfanoAnteConflictoOCC(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	preparada := prepararFirmaBaremacionPrueba(t, entorno)
	// Simula que otra transaccion sustituyo la version base durante la firma.
	entorno.repositorio.version.Referencia.Numero++

	_, err := entorno.servicio.FinalizarFirma(
		context.Background(), ordenFinalizarBaremacionPrueba(preparada, "conflicto-occ"),
	)
	var huerfano *ErrorDocumentoFirmadoHuerfano
	if !errors.As(err, &huerfano) || !errors.Is(err, puertosbolsa.ErrVersionBaremacionConflicto) {
		t.Fatalf("conflicto OCC no devolvio evidencia huerfana tipada: %v", err)
	}
	if huerfano.DecisionRef == "" || huerfano.Documento.Objeto.Validar() != nil ||
		huerfano.Documento.EvidenciaEscritura.Referencia == "" ||
		huerfano.Documento.EvidenciaRetencion.Referencia == "" ||
		entorno.almacen.escrituras != 2 || entorno.almacen.retenciones != 1 ||
		entorno.repositorio.reservas != 1 || entorno.repositorio.confirmaciones != 0 {
		t.Fatalf("documento huerfano no gestionable: %+v", huerfano)
	}
}

func TestFalloDeRetencionConservaReferenciaDelObjetoParaReconciliar(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	preparada := prepararFirmaBaremacionPrueba(t, entorno)
	entorno.almacen.errorRetencion = errors.New("fallo de retencion simulado")

	_, err := entorno.servicio.FinalizarFirma(
		context.Background(),
		ordenFinalizarBaremacionPrueba(preparada, "retencion-fallida"),
	)
	var incompleta *ErrorCustodiaBaremacionIncompleta
	if !errors.As(err, &incompleta) {
		t.Fatalf("fallo de retencion sin recibo recuperable: %v", err)
	}
	if incompleta.DecisionRef == "" || incompleta.DocumentoRef == "" ||
		incompleta.Escritura.Validar() != nil || entorno.repositorio.reservas != 0 {
		t.Fatalf("recibo de reconciliacion incompleto: %+v", incompleta)
	}
	if obtenido := fmt.Sprintf("%+v %#v", incompleta, incompleta); obtenido != "bolsa: objeto custodiado pendiente de reconciliacion bolsa: objeto custodiado pendiente de reconciliacion" {
		t.Fatalf("el error filtro metadatos del objeto: %s", obtenido)
	}
}

func TestServicioBaremacionAplicaSelloTiempoYLongevidadSoloCuandoLaPoliticaLoExige(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	politica := entorno.politicas.politica
	politica.RequiereSelloTiempo = true
	politica.PoliticaSelloTiempoRef = "politica-sello-tiempo"
	politica.PoliticaSelloTiempoVersion = 2
	politica.HuellaPoliticaSelloTiempoSHA256 = huellaBaremacionPrueba("3")
	politica.RequiereAumentoLongevidad = true
	politica.PerfilFirmaClave = puertosbolsa.PerfilFirmaPAdESBaselineLTA
	politica.NivelAumentoClave = "pades_lta"
	politica.PoliticaLongevidadRef = "politica-longevidad"
	politica.PoliticaLongevidadVersion = 3
	politica.HuellaPoliticaLongevidadSHA256 = huellaBaremacionPrueba("4")
	entorno.politicas.politica = politica
	sellador := &selladorTiempoActivoBaremacionPrueba{ahora: instanteBaremacionPrueba}
	aumentador := &aumentadorActivoBaremacionPrueba{ahora: instanteBaremacionPrueba}
	entorno.servicio.selladorTiempo = sellador
	entorno.servicio.aumentadorFirma = aumentador
	preparada := prepararFirmaBaremacionPrueba(t, entorno)

	resultado, err := entorno.servicio.FinalizarFirma(context.Background(), OrdenFinalizarFirmaBaremacion{
		Actor: actorBaremacionPrueba(), Firma: preparada, OperacionRef: "operacion:finalizar:lta",
		OperacionCustodiaRef: "operacion:custodia:firmado:lta", ClaveIdempotenciaCustodia: "idempotencia:custodia:firmado:lta",
		CargaDocumentoFirmadoRef: "carga:documento:firmado:lta",
		ClaveIdempotenciaReserva: "idempotencia:reserva:confirmacion:lta",
		ClaveIdempotenciaSello:   "idempotencia:sello:1", ClaveIdempotenciaAumento: "idempotencia:aumento:1",
		MotivoClaveConfirmacion: "decision_tecnica_firmada",
		MotivoConfirmacion:      "Incorporacion con sello de tiempo y preservacion longeva.",
	})
	if err != nil {
		t.Fatalf("FinalizarFirma() error = %v", err)
	}
	if resultado.SelloTiempo == nil || resultado.Aumento == nil || sellador.llamadas != 1 || aumentador.llamadas != 1 ||
		resultado.ValidacionTrasSello == nil || entorno.validador.llamadas != 3 || resultado.Decision.Validar() != nil {
		t.Fatalf("capas de firma incompletas: sello=%v aumento=%v validaciones=%d",
			resultado.SelloTiempo != nil, resultado.Aumento != nil, entorno.validador.llamadas)
	}
	acciones := entorno.autorizador.acciones()
	esperadoFinal := []puertosbolsa.AccionOperacionBaremacion{
		puertosbolsa.AccionConsultarFirmaDecisionBaremacion,
		puertosbolsa.AccionValidarFirmaDecisionBaremacion,
		puertosbolsa.AccionSellarTiempoDecisionBaremacion,
		puertosbolsa.AccionValidarFirmaDecisionBaremacion,
		puertosbolsa.AccionAumentarFirmaDecisionBaremacion,
		puertosbolsa.AccionValidarFirmaDecisionBaremacion,
		puertosbolsa.AccionRecuperarBinarioFirmadoBaremacion,
		puertosbolsa.AccionCustodiarDocumentoFirmadoBaremacion,
		puertosbolsa.AccionRetenerDocumentoFirmadoBaremacion,
		puertosbolsa.AccionReservarDecisionBaremacion,
		puertosbolsa.AccionPrevalidarArchivoProbatorioBaremacion,
		puertosbolsa.AccionConfirmarDecisionBaremacion,
	}
	if len(acciones) < len(esperadoFinal) ||
		!accionesBaremacionIguales(acciones[len(acciones)-len(esperadoFinal):], esperadoFinal) {
		t.Fatalf("autorizaciones finales = %v", acciones)
	}
}
