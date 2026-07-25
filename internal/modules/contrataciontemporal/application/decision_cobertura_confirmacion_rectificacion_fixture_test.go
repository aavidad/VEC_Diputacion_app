package application

import (
	"context"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

type escenarioRectificacionConfirmacionCobertura struct {
	*escenarioConfirmacionCobertura
	solicitudRectificar SolicitudRectificarCobertura
	predecesora         domain.PublicacionDecisionCoberturaGobernada
}

func nuevoEscenarioRectificacionConfirmacionCobertura(
	t *testing.T,
	actorDistinto bool,
) *escenarioRectificacionConfirmacionCobertura {
	t.Helper()
	base := nuevoEscenarioPresentacionCobertura(
		t,
		viasPresentacionCoberturaPrueba(2),
	)
	decidido := registrarDecisionInicialParaRectificacion(t, base)
	predecesora := decidido.DecisionesCobertura[len(decidido.DecisionesCobertura)-1]
	if actorDistinto {
		base.contextos.contexto = contextoAutorizacionAltaV3PruebaConMarcas(
			t,
			base.global.entorno.reloj.Ahora(),
			"b",
			"b",
		)
	}
	vinculo, err := base.contextos.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	base.expediente = decidido
	base.analisis.expediente = decidido
	base.solicitud = SolicitudProponerCobertura{
		AutenticacionRef: vinculo.AutenticacionRef,
		SesionRef:        vinculo.SesionRef,
		PerfilRef:        vinculo.PerfilActivoRef,
		OrganizacionRef:  decidido.OrganizacionRef,
		ExpedienteRef:    decidido.Referencia,
		VersionEsperada:  decidido.Version,
	}
	presentacion, err := base.servicio.Proponer(
		context.Background(),
		base.solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitudO3, err := cobertura.NuevaSolicitudInstantaneaAnalisisDurableO3(
		decidido.OrganizacionRef,
		decidido.Referencia,
		decidido.Version,
	)
	if err != nil {
		t.Fatal(err)
	}
	instantanea, err := cobertura.ObtenerInstantaneaAnalisisDurableO3(
		context.Background(),
		base.analisis,
		solicitudO3,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, analisisRef, analisisHuella, err :=
		instantanea.DesplegarPara(solicitudO3)
	if err != nil {
		t.Fatal(err)
	}
	motivos := nuevoResolutorMotivoConfirmacionPrueba(
		t,
		"motivos_rectificacion_confirmacion",
		base.global.entorno.reloj.Ahora(),
	)
	escenario := &escenarioConfirmacionCobertura{
		base:     base,
		motivos:  &resolutorMotivoConfirmacionPrueba{},
		sellador: &selladorConfirmacionPrueba{},
		idempotencia: &idempotenciaConfirmacionPrueba{
			modo:           idempotenciaPropietaria,
			expediente:     decidido,
			analisisRef:    analisisRef,
			analisisHuella: analisisHuella,
			reloj:          base.global.entorno.reloj,
		},
		vec: &preparadorVECConfirmacionPrueba{
			reloj: base.global.entorno.reloj,
		},
	}
	escenario.transaccion = &transaccionConfirmacionPrueba{
		idempotencia: escenario.idempotencia,
		vec:          escenario.vec,
		reloj:        base.global.entorno.reloj,
	}
	escenario.reconciliador = &reconciliadorConfirmacionPrueba{
		reloj:        base.global.entorno.reloj,
		idempotencia: escenario.idempotencia,
		vec:          escenario.vec,
	}
	escenario.servicio, err = NuevoServicioConfirmacionDecisionCobertura(
		base.contextos,
		motivos,
		escenario.sellador,
		escenario.idempotencia,
		base.analisis,
		base.reloj,
		base.gobierno,
		base.global.preparador,
		escenario.vec,
		escenario.transaccion,
		escenario.reconciliador,
	)
	if err != nil {
		t.Fatal(err)
	}
	return &escenarioRectificacionConfirmacionCobertura{
		escenarioConfirmacionCobertura: escenario,
		predecesora:                    predecesora,
		solicitudRectificar: SolicitudRectificarCobertura{
			AutenticacionRef:   vinculo.AutenticacionRef,
			SesionRef:          vinculo.SesionRef,
			PerfilRef:          vinculo.PerfilActivoRef,
			OrganizacionRef:    decidido.OrganizacionRef,
			ExpedienteRef:      decidido.Referencia,
			VersionEsperada:    decidido.Version,
			ClaveIdempotencia:  "12345678-1234-4234-8234-123456789abd",
			IdentidadSemantica: presentacion.IdentidadSemantica,
			ViaElegida:         "via_global_02",
			MotivoClave:        claveMotivoConfirmacionPrueba,
			PredecesoraRef:     predecesora.Referencia,
			PredecesoraHuella:  predecesora.HuellaSHA256,
		},
	}
}

func registrarDecisionInicialParaRectificacion(
	t *testing.T,
	base *escenarioPresentacionCobertura,
) domain.Expediente {
	t.Helper()
	solicitudO3, err := cobertura.NuevaSolicitudInstantaneaAnalisisDurableO3(
		base.expediente.OrganizacionRef,
		base.expediente.Referencia,
		base.expediente.Version,
	)
	if err != nil {
		t.Fatal(err)
	}
	instantanea, err := cobertura.ObtenerInstantaneaAnalisisDurableO3(
		context.Background(),
		base.analisis,
		solicitudO3,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, analisisRef, analisisHuella, err :=
		instantanea.DesplegarPara(solicitudO3)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := nuevosDatosPreparacionGlobalCobertura(
		analisisRef,
		analisisHuella,
		base.global.catalogo,
		base.global.politica,
		base.expediente.OrganizacionRef,
		base.expediente.Referencia,
		base.expediente.Version,
		base.expediente.Analisis.CategoriaRef,
		base.expediente.Analisis.Periodo,
	)
	if err != nil {
		t.Fatal(err)
	}
	preparacion, err := base.global.preparador.Preparar(
		context.Background(),
		datos,
	)
	if err != nil {
		t.Fatal(err)
	}
	instante := base.global.entorno.reloj.Ahora()
	datosPropuesta, err := preparacion.DatosCrearPropuestaEn(instante)
	if err != nil {
		t.Fatal(err)
	}
	propuesta, err := domain.CrearPropuestaDecisionCobertura(datosPropuesta)
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := base.contextos.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	decidido, err := base.expediente.RegistrarDecisionCoberturaGobernada(
		base.expediente.Version,
		domain.DatosAdoptarDecisionCobertura{
			PerfilRef:  vinculo.PerfilActivoRef,
			ViaElegida: propuesta.ViaPropuesta(),
		},
		propuesta,
		domain.DatosActuacion{
			AccionClave:   domain.AccionDecidirCoberturaGobernada,
			ActorRef:      vinculo.PrincipalID,
			UnidadRef:     "unidad_rrhh_presentacion_cobertura_01",
			ReciboRef:     "recibo_decision_inicial_rectificacion_01",
			RealizadaEn:   instante,
			FaseDestino:   "decision_cobertura",
			EstadoDestino: domain.EstadoEnCurso,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return decidido
}
