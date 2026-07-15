package confianzadocumental

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type registradorAtestacionPDPCaptura struct {
	solicitud solicitudEjecucionDocumentalAtestadaV4
	llamadas  int
}

func (r *registradorAtestacionPDPCaptura) ejecutarPlanAtestado(
	_ context.Context,
	solicitud solicitudEjecucionDocumentalAtestadaV4,
) (ResultadoEjecucionPlanDocumentalV4, error) {
	if solicitud.validar() != nil {
		return ResultadoEjecucionPlanDocumentalV4{}, errorAutoridadInternaEjecucionDocumentalV4()
	}
	r.solicitud = solicitud
	r.llamadas++
	return ResultadoEjecucionPlanDocumentalV4{
		OrdenRef: solicitud.orden.OrdenRef, Estado: solicitud.orden.Estado,
		AuditoriaRef: solicitud.orden.AuditoriaRef, EventoOutboxRef: solicitud.orden.EventoOutboxRef,
		RegistradaEn: solicitud.orden.SolicitadaEn,
	}, nil
}

func TestServicioRegistraAtestacionPDPExactaSinDTOYTrasReverificarCOSE(t *testing.T) {
	base := nuevoEscenarioAtestacionAutorizacionPDPV4(t)
	autoridad, err := base.servicio.EmitirAutoridadInternaEjecucionDocumentalV4(
		context.Background(), base.vinculo, base.cabecera, base.sobre,
	)
	if err != nil {
		t.Fatalf("emitir autoridad: %v", err)
	}
	instante := base.escenario.emitidaEn.Add(time.Microsecond)
	base.servicio.reloj = &relojContadorAtestacionPDP{instante: instante}
	registrador := &registradorAtestacionPDPCaptura{}
	base.servicio.repositorioEjecucionV4 = registrador
	_, err = base.servicio.EjecutarPlanDocumentalV4(context.Background(), autoridad)
	if err != nil || registrador.llamadas != 1 {
		t.Fatalf("preparar registro exacto: %v", err)
	}
	persistible := registrador.solicitud.prueba.datos
	preimagenBytes, err := persistible.PreimagenRecurso.SerializacionCanonicaParaPersistencia()
	if err != nil {
		t.Fatalf("leer preimagen: %v", err)
	}
	preimagen, err := persistible.PreimagenRecurso.RecursoCanonico()
	if err != nil || !bytes.Equal(persistible.PayloadVECAD1, base.payload) ||
		persistible.ProyeccionAplicacion.Clave.DecisionRef != base.escenario.decision.DecisionRef ||
		persistible.ProyeccionAplicacion.Clave.HuellaPlanSHA256 !=
			base.escenario.expectativa.HuellaPlanSHA256 ||
		persistible.ProyeccionAplicacion.Clave.EfectoRef != base.escenario.expectativa.EfectoRef ||
		!persistible.ProyeccionAplicacion.SolicitadaEn.Equal(instante) ||
		persistible.HuellaPayloadSHA256 != huellaBytesDocumentales(base.payload) ||
		preimagen.Referencia != base.escenario.expectativa.Recurso.Referencia ||
		len(preimagenBytes) == 0 {
		t.Fatalf("registro no conserva la derivacion exacta: %#v, %v", persistible, err)
	}

}

func TestServicioRegistrarAtestacionPDPRechazaSiSuRaizYaNoVerificaElSobre(t *testing.T) {
	base := nuevoEscenarioAtestacionAutorizacionPDPV4(t)
	autoridad, err := base.servicio.EmitirAutoridadInternaEjecucionDocumentalV4(
		context.Background(), base.vinculo, base.cabecera, base.sobre,
	)
	if err != nil {
		t.Fatal(err)
	}
	otraInstancia := nuevoEscenarioAtestacionAutorizacionPDPV4(t)
	indice := string(base.material.claveID)
	raiz := base.servicio.raices[indice]
	raiz.verificador = otraInstancia.servicio.raices[indice].verificador
	base.servicio.raices[indice] = raiz
	instante := base.escenario.emitidaEn.Add(time.Microsecond)
	base.servicio.reloj = &relojContadorAtestacionPDP{instante: instante}
	registrador := &registradorAtestacionPDPCaptura{}
	base.servicio.repositorioEjecucionV4 = registrador
	_, err = base.servicio.EjecutarPlanDocumentalV4(context.Background(), autoridad)
	if !errors.Is(err, ErrAutoridadInternaEjecucionDocumentalV4Invalida) ||
		registrador.llamadas != 0 {
		t.Fatalf("se registro sin reverificar COSE: %v, llamadas=%d", err, registrador.llamadas)
	}
}

func TestExportadorRegistroAtestacionPDPRechazaMutacionDeCadaUnaDeLas25Claves(t *testing.T) {
	base := nuevoEscenarioAtestacionAutorizacionPDPV4(t)
	autoridad, err := base.servicio.EmitirAutoridadInternaEjecucionDocumentalV4(
		context.Background(), base.vinculo, base.cabecera, base.sobre,
	)
	if err != nil {
		t.Fatal(err)
	}
	instante := base.escenario.emitidaEn.Add(time.Microsecond)
	aplicacion, evidencia, err := autoridad.PrepararAplicacionExactaConEvidenciaEn(
		autoridad.clave.DecisionRef,
		autoridad.clave.HuellaPlanSHA256,
		autoridad.clave.EfectoRef,
		instante,
	)
	if err != nil {
		t.Fatal(err)
	}
	baseProyeccion, err := aplicacion.ProyeccionParaTransaccion()
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := evidencia.PayloadVECAD1()
	sobreBytes, _ := evidencia.SobreCOSESign1()
	sobre, err := ports.NuevoSobreCriptograficoDocumentalCrudoV4(sobreBytes)
	if err != nil {
		t.Fatal(err)
	}
	solicitudCOSE, err := NuevaSolicitudVerificacionCOSESign1(
		payload, AudienciaCOSEAtestacionAutorizacionPDP,
	)
	if err != nil {
		t.Fatal(err)
	}
	pruebaReverificada, err := base.servicio.verificarCOSESign1En(
		context.Background(), solicitudCOSE, sobre, instante,
	)
	if err != nil {
		t.Fatal(err)
	}

	mutarHuella := func(valor string) string {
		if len(valor) != 64 {
			t.Fatalf("huella de fixture no canonica: %q", valor)
		}
		prefijo := byte('0')
		if valor[0] == prefijo {
			prefijo = '1'
		}
		return string(prefijo) + valor[1:]
	}
	casos := []struct {
		nombre string
		mutar  func(*ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4)
	}{
		{"01 esquema", func(p *ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) { p.Esquema += ".otro" }},
		{"02 decision", func(p *ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) { p.Clave.DecisionRef += ":otra" }},
		{"03 plan", func(p *ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) {
			p.Clave.HuellaPlanSHA256 = mutarHuella(p.Clave.HuellaPlanSHA256)
		}},
		{"04 efecto", func(p *ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) { p.Clave.EfectoRef += ":otro" }},
		{"05 esquema huella decision", func(p *ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) {
			p.EsquemaHuellaDecision += ".otro"
		}},
		{"06 huella decision", func(p *ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) {
			p.HuellaDecisionSHA256 = mutarHuella(p.HuellaDecisionSHA256)
		}},
		{"07 perfil", func(p *ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) { p.PerfilActivoRef += ":otro" }},
		{"08 contexto actor", func(p *ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) {
			p.ContextoActorHuellaSHA256 = mutarHuella(p.ContextoActorHuellaSHA256)
		}},
		{"09 accion", func(p *ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) { p.Accion += ".otra" }},
		{"10 recurso", func(p *ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) { p.RecursoRef += ":otro" }},
		{"11 modulo", func(p *ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) { p.ModuloID += "_otro" }},
		{"12 tipo", func(p *ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) { p.TipoRecurso += "_otro" }},
		{"13 huella recurso", func(p *ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) {
			p.HuellaRecursoSHA256 = mutarHuella(p.HuellaRecursoSHA256)
		}},
		{"14 huella ambitos", func(p *ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) {
			p.HuellaAmbitosSHA256 = mutarHuella(p.HuellaAmbitosSHA256)
		}},
		{"15 finalidad", func(p *ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) { p.Finalidad += "_otra" }},
		{"16 correlacion", func(p *ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) { p.CorrelacionRef += ":otra" }},
		{"17 huella campos", func(p *ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) {
			p.HuellaCamposPermitidosSHA256 = mutarHuella(p.HuellaCamposPermitidosSHA256)
		}},
		{"18 huella obligaciones", func(p *ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) {
			p.HuellaObligacionesSHA256 = mutarHuella(p.HuellaObligacionesSHA256)
		}},
		{"19 huella cumplimientos", func(p *ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) {
			p.HuellaCumplimientosSHA256 = mutarHuella(p.HuellaCumplimientosSHA256)
		}},
		{"20 verificada", func(p *ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) {
			p.VerificadaEn = p.VerificadaEn.Add(time.Microsecond)
		}},
		{"21 vinculada", func(p *ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) {
			p.VinculadaEn = p.VinculadaEn.Add(time.Microsecond)
		}},
		{"22 solicitada", func(p *ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) {
			p.SolicitadaEn = p.SolicitadaEn.Add(time.Microsecond)
		}},
		{"23 valida hasta", func(p *ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) {
			p.ValidaHasta = p.ValidaHasta.Add(-time.Microsecond)
		}},
		{"24 huella vinculada", func(p *ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) {
			p.HuellaSolicitudVinculadaSHA256 = mutarHuella(p.HuellaSolicitudVinculadaSHA256)
		}},
		{"25 huella aplicacion", func(p *ports.ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) {
			p.HuellaSolicitudAplicacionSHA256 = mutarHuella(p.HuellaSolicitudAplicacionSHA256)
		}},
	}
	if len(casos) != 25 {
		t.Fatalf("se cubrieron %d claves, se esperaban 25", len(casos))
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			manipulada := baseProyeccion
			caso.mutar(&manipulada)
			if manipulada == baseProyeccion {
				t.Fatal("el caso no altero la proyeccion")
			}
			_, err := prepararPruebaRegistroAtestacionPDPDocumentalV4(
				autoridad, aplicacion, manipulada, evidencia, pruebaReverificada,
			)
			if !errors.Is(err, ErrAutoridadInternaEjecucionDocumentalV4Invalida) ||
				!errors.Is(err, domain.ErrAutorizacionDenegada) {
				t.Fatalf("mutacion aceptada o error no cerrado: %v", err)
			}
		})
	}
}

func TestCotejoRegistroAtestacionPDPRechazaCamposFirmadosDistintos(t *testing.T) {
	base := nuevoEscenarioAtestacionAutorizacionPDPV4(t)
	autoridad, err := base.servicio.EmitirAutoridadInternaEjecucionDocumentalV4(
		context.Background(), base.vinculo, base.cabecera, base.sobre,
	)
	if err != nil {
		t.Fatal(err)
	}
	instante := base.escenario.emitidaEn.Add(time.Microsecond)
	aplicacion, evidencia, err := autoridad.PrepararAplicacionExactaConEvidenciaEn(
		autoridad.clave.DecisionRef,
		autoridad.clave.HuellaPlanSHA256,
		autoridad.clave.EfectoRef,
		instante,
	)
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := evidencia.PayloadVECAD1()
	metadatos, _ := evidencia.Metadatos()
	preimagenBytes, _ := evidencia.PreimagenRecursoCanonica()
	preimagen, err := ports.InterpretarPreimagenRecursoAutorizacionEjecucionDocumentalV4(
		preimagenBytes, metadatos.HuellaPreimagenRecursoSHA256,
	)
	proyeccionHistorica, errParseo := domain.ParsearMensajeAtestacionAutorizacionV1NoAutoritativo(payload)
	datos, errDatos := proyeccionHistorica.Datos()
	if err != nil || errParseo != nil || errDatos != nil ||
		aplicacion.CotejarConDecisionHistoricaAtestacionPDPV1(datos, preimagen) != nil {
		t.Fatalf("fixture historico no coteja: %v / %v / %v", err, errParseo, errDatos)
	}

	casos := []struct {
		nombre string
		mutar  func(*domain.DatosDecisionHistoricaAtestacionAutorizacionV1)
	}{
		{"01 decision_ref", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) { d.DecisionRef += ":otra" }},
		{"02 concedida", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) { d.Concedida = !d.Concedida }},
		{"03 codigo", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) { d.Codigo += "_otro" }},
		{"04 principal", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) { d.PrincipalID += "x" }},
		{"05 perfil", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) { d.PerfilActivoRef += "x" }},
		{"06 accion", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) { d.Accion += ".otra" }},
		{"07 recurso", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) { d.RecursoRef += ":otro" }},
		{"08 modulo", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) { d.ModuloID += "_otro" }},
		{"09 tipo", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) { d.TipoRecurso += "_otro" }},
		{"10 contexto_recurso", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.ContextoRecursoHuellaSHA256 = alternarHuellaRegistroPDPPrueba(d.ContextoRecursoHuellaSHA256)
		}},
		{"11 finalidad", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) { d.Finalidad += "_otra" }},
		{"12 correlacion", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) { d.CorrelacionRef += ":otra" }},
		{"13 asignacion_ref", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) { d.AsignacionRef += ":otra" }},
		{"14 asignacion_huella", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.AsignacionHuellaSHA256 = alternarHuellaRegistroPDPPrueba(d.AsignacionHuellaSHA256)
		}},
		{"15 version_rol_ref", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) { d.VersionRolRef += ":otra" }},
		{"16 version_rol_huella", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.VersionRolHuellaSHA256 = alternarHuellaRegistroPDPPrueba(d.VersionRolHuellaSHA256)
		}},
		{"17 control_rol_ref", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.ControlVigenciaVersionRolRef += ":otro"
		}},
		{"18 control_rol_revision", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) { d.ControlVigenciaVersionRolRevision++ }},
		{"19 control_rol_huella", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.ControlVigenciaVersionRolHuellaSHA256 = alternarHuellaRegistroPDPPrueba(d.ControlVigenciaVersionRolHuellaSHA256)
		}},
		{"20 revision_catalogo", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) { d.RevisionCatalogoPoliticas++ }},
		{"21 catalogo_huella", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.CatalogoPoliticasHuellaSHA256 = alternarHuellaRegistroPDPPrueba(d.CatalogoPoliticasHuellaSHA256)
		}},
		{"22 politicas_evaluadas_refs", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) { d.PoliticasEvaluadasRefs[0] += ":otra" }},
		{"23 politicas_evaluadas_huellas", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			mutarPrimerValorMapaRegistroPDPPrueba(d.PoliticasEvaluadasHuellasSHA256)
		}},
		{"24 politicas_refs", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) { d.PoliticasRefs[0] += ":otra" }},
		{"25 politicas_huellas", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			mutarPrimerValorMapaRegistroPDPPrueba(d.PoliticasHuellasSHA256)
		}},
		{"26 garantia_minima", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.GarantiaMinima = domain.AuthAssuranceLow
		}},
		{"27 campos", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) { d.CamposPermitidos[0] += ".otro" }},
		{"28 obligaciones", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.Obligaciones = []string{"otra_obligacion"}
		}},
		{"29 emitida", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.EmitidaEn = d.EmitidaEn.Add(time.Microsecond)
		}},
		{"30 valida_hasta", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.ValidaHasta = d.ValidaHasta.Add(-time.Microsecond)
		}},
		{"31 vinculo_bloque_version", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.VinculoAutenticacionActor.BloqueVersion++
		}},
		{"32 vinculo_autenticacion_ref", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.VinculoAutenticacionActor.AutenticacionRef += "x"
		}},
		{"33 vinculo_autenticacion_huella", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.VinculoAutenticacionActor.AutenticacionHuellaSHA256 = alternarHuellaRegistroPDPPrueba(d.VinculoAutenticacionActor.AutenticacionHuellaSHA256)
		}},
		{"34 vinculo_asercion_ref", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.VinculoAutenticacionActor.AsercionRef += "x"
		}},
		{"35 vinculo_sesion_ref", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.VinculoAutenticacionActor.SesionRef += "x"
		}},
		{"36 vinculo_control_ref", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.VinculoAutenticacionActor.ControlSesionRef += "x"
		}},
		{"37 vinculo_control_revision", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.VinculoAutenticacionActor.ControlSesionRevision++
		}},
		{"38 vinculo_control_huella", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.VinculoAutenticacionActor.ControlSesionHuellaSHA256 = alternarHuellaRegistroPDPPrueba(d.VinculoAutenticacionActor.ControlSesionHuellaSHA256)
		}},
		{"39 vinculo_cuenta_ref", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.VinculoAutenticacionActor.CuentaRef += "x"
		}},
		{"40 vinculo_cuenta_ordinaria_ref", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.VinculoAutenticacionActor.CuentaOrdinariaRef += "x"
		}},
		{"41 vinculo_principal", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.VinculoAutenticacionActor.PrincipalID += "x"
		}},
		{"42 vinculo_perfil", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.VinculoAutenticacionActor.PerfilActivoRef += "x"
		}},
		{"43 vinculo_privilegiada", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.VinculoAutenticacionActor.CuentaPrivilegiada = !d.VinculoAutenticacionActor.CuentaPrivilegiada
		}},
		{"44 vinculo_superficie", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.VinculoAutenticacionActor.Superficie = otraSuperficieRegistroPDPPrueba(d.VinculoAutenticacionActor.Superficie)
		}},
		{"45 vinculo_metodo", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.VinculoAutenticacionActor.MetodoObservado = domain.AuthMethodDemo
		}},
		{"46 vinculo_garantia", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.VinculoAutenticacionActor.GarantiaObservada = domain.AuthAssuranceLow
		}},
		{"47 vinculo_politica_ref", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.VinculoAutenticacionActor.PoliticaGarantiaRef += "x"
		}},
		{"48 vinculo_politica_huella", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.VinculoAutenticacionActor.PoliticaGarantiaHuellaSHA256 = alternarHuellaRegistroPDPPrueba(d.VinculoAutenticacionActor.PoliticaGarantiaHuellaSHA256)
		}},
		{"49 vinculo_autenticacion_verificada", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.VinculoAutenticacionActor.AutenticacionVerificadaEn = d.VinculoAutenticacionActor.AutenticacionVerificadaEn.Add(time.Microsecond)
		}},
		{"50 vinculo_sesion_emitida", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.VinculoAutenticacionActor.SesionEmitidaEn = d.VinculoAutenticacionActor.SesionEmitidaEn.Add(time.Microsecond)
		}},
		{"51 vinculo_sesion_valida", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.VinculoAutenticacionActor.SesionValidaHasta = d.VinculoAutenticacionActor.SesionValidaHasta.Add(-time.Microsecond)
		}},
		{"52 vinculo_sesion_revalidada", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.VinculoAutenticacionActor.SesionRevalidadaEn = d.VinculoAutenticacionActor.SesionRevalidadaEn.Add(time.Microsecond)
		}},
		{"53 vinculo_contexto_ref", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.VinculoAutenticacionActor.ContextoActorRef += "x"
		}},
		{"54 vinculo_contexto_version", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.VinculoAutenticacionActor.ContextoActorVersion++
		}},
		{"55 vinculo_contexto_huella", func(d *domain.DatosDecisionHistoricaAtestacionAutorizacionV1) {
			d.VinculoAutenticacionActor.ContextoActorHuellaSHA256 = alternarHuellaRegistroPDPPrueba(d.VinculoAutenticacionActor.ContextoActorHuellaSHA256)
		}},
	}
	if len(casos) != 55 {
		t.Fatalf("cobertura historica incompleta: %d", len(casos))
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			manipulados := clonarDatosDecisionHistoricaRegistroPDPPrueba(datos)
			caso.mutar(&manipulados)
			if reflect.DeepEqual(manipulados, datos) {
				t.Fatal("el caso no altero su campo historico")
			}
			if aplicacion.CotejarConDecisionHistoricaAtestacionPDPV1(
				manipulados, preimagen,
			) == nil {
				t.Fatal("campo firmado distinto aceptado")
			}
		})
	}
}

func alternarHuellaRegistroPDPPrueba(valor string) string {
	if len(valor) != 64 {
		return valor + "0"
	}
	prefijo := byte('0')
	if valor[0] == prefijo {
		prefijo = '1'
	}
	return string(prefijo) + valor[1:]
}

func mutarPrimerValorMapaRegistroPDPPrueba(mapa map[string]string) {
	for clave, valor := range mapa {
		mapa[clave] = alternarHuellaRegistroPDPPrueba(valor)
		return
	}
	mapa["politica:inyectada"] = huellaInternaPrueba('f')
}

func otraSuperficieRegistroPDPPrueba(
	actual domain.SuperficieAutenticacionActorV1,
) domain.SuperficieAutenticacionActorV1 {
	if actual != domain.SuperficieAutenticacionExternaPersonalV1 {
		return domain.SuperficieAutenticacionExternaPersonalV1
	}
	return domain.SuperficieAutenticacionInternaCorporativaV1
}

func clonarDatosDecisionHistoricaRegistroPDPPrueba(
	datos domain.DatosDecisionHistoricaAtestacionAutorizacionV1,
) domain.DatosDecisionHistoricaAtestacionAutorizacionV1 {
	datos.PoliticasEvaluadasRefs = append([]string(nil), datos.PoliticasEvaluadasRefs...)
	datos.PoliticasRefs = append([]string(nil), datos.PoliticasRefs...)
	datos.CamposPermitidos = append([]string(nil), datos.CamposPermitidos...)
	datos.Obligaciones = append([]string(nil), datos.Obligaciones...)
	datos.PoliticasEvaluadasHuellasSHA256 = clonarMapaInterno(
		datos.PoliticasEvaluadasHuellasSHA256,
	)
	datos.PoliticasHuellasSHA256 = clonarMapaInterno(datos.PoliticasHuellasSHA256)
	return datos
}
