package application

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var instantePanelInternoPrueba = time.Date(
	2026, time.July, 17, 12, 30, 0, 123_456_000, time.UTC,
)

type relojPanelInternoPrueba struct{ ahora time.Time }

func (r relojPanelInternoPrueba) Ahora() time.Time { return r.ahora }

type generadorCorrelacionPanelPrueba struct{ valor string }

func (g generadorCorrelacionPanelPrueba) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	return g.valor, nil
}

type revalidadorPanelPrueba struct {
	resultado dominiovec.AutenticacionRevalidadaV1
}

func (r revalidadorPanelPrueba) RevalidarAutenticacionActorV1(
	context.Context,
	dominiovec.SolicitudRevalidacionAutenticacionActorV1,
) (dominiovec.AutenticacionRevalidadaV1, error) {
	return r.resultado, nil
}

type exigidorPanelPrueba struct {
	instante time.Time
	error    error
	llamadas int
	observar func(dominiovec.RecursoAutorizable)
}

func (e *exigidorPanelPrueba) ExigirEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(
	ctx context.Context,
	actor dominiovec.ContextoActor,
	vinculo dominiovec.VinculoAutenticacionActorV1,
	recurso dominiovec.RecursoAutorizable,
	correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2,
	motivo dominiovec.ReferenciaEntradaCatalogo,
	_ aplicacionvec.PoliticaUsoDecisionAutorizacion,
) (puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2, error) {
	e.llamadas++
	if e.observar != nil {
		e.observar(recurso)
	}
	if e.error != nil {
		return puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}, e.error
	}
	if err := ctx.Err(); err != nil {
		return puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}, err
	}
	return nuevaEvidenciaPanelPrueba(actor, vinculo, recurso, correlacion, motivo, e.instante)
}

type consultaPanelPrueba struct {
	resultado puertosbolsa.InstantaneaPanelInterno
	error     error
	antes     func(puertosbolsa.SolicitudConsultaPanelInterno)
	llamadas  int
}

func (c *consultaPanelPrueba) ConsultarPanel(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudConsultaPanelInterno,
) (puertosbolsa.InstantaneaPanelInterno, error) {
	c.llamadas++
	if c.antes != nil {
		c.antes(solicitud)
	}
	if c.error != nil {
		return puertosbolsa.InstantaneaPanelInterno{}, c.error
	}
	if err := ctx.Err(); err != nil {
		return puertosbolsa.InstantaneaPanelInterno{}, err
	}
	resultado := c.resultado
	autorizacion, errAutorizacion := solicitud.Autorizacion()
	datosAutorizacion, errDatos := autorizacion.Datos()
	correlacion, errCorrelacion := solicitud.Correlacion()
	correlacionRef, errCorrelacionRef := correlacion.ValorCanonico()
	if errAutorizacion != nil || errDatos != nil || errCorrelacion != nil || errCorrelacionRef != nil {
		return puertosbolsa.InstantaneaPanelInterno{}, puertosbolsa.ErrConsultaPanelInternoInvalida
	}
	resultado.PruebaLectura.DecisionRef = datosAutorizacion.Decision.DecisionRef
	resultado.PruebaLectura.HuellaDecisionSHA256 = datosAutorizacion.HuellaDecisionSHA256
	resultado.PruebaLectura.CorrelacionRef = correlacionRef
	return resultado, nil
}

func nuevaEvidenciaPanelPrueba(
	actor dominiovec.ContextoActor,
	vinculo dominiovec.VinculoAutenticacionActorV1,
	recurso dominiovec.RecursoAutorizable,
	correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2,
	motivo dominiovec.ReferenciaEntradaCatalogo,
	instante time.Time,
) (puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2, error) {
	correlacionRef, err := correlacion.ValorCanonico()
	if err != nil {
		return puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}, err
	}
	huellaContexto, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}, err
	}
	huellaCatalogo, err := dominiovec.HuellaEvidenciasCatalogoPoliticasAutorizacion(nil, nil)
	if err != nil {
		return puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}, err
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV2(
		dominiovec.DatosSolicitudAutorizacionLigadaV2{
			ContextoActor: actor, VinculoAutenticacionActor: vinculo,
			ReferenciaMotivo: motivo, Accion: puertosbolsa.AccionConsultarPanelInterno,
			Recurso: recurso, Finalidad: puertosbolsa.FinalidadPanelInternoBolsa,
			Correlacion: correlacion,
		},
	)
	if err != nil {
		return puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}, err
	}
	huellaSolicitud, err := dominiovec.HuellaSHA256SolicitudAutorizacionV2(solicitud)
	if err != nil {
		return puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}, err
	}
	huellaMotivo, err := dominiovec.HuellaSHA256MotivoAutorizacionV2(motivo)
	if err != nil {
		return puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}, err
	}
	decision := dominiovec.DecisionAutorizacion{
		DecisionRef: "decision:panel:00000001", Concedida: true, Codigo: "concedida",
		PrincipalID: actor.Principal.ID, PerfilActivoRef: actor.PerfilActivoRef,
		Accion: puertosbolsa.AccionConsultarPanelInterno, RecursoRef: recurso.Referencia,
		ModuloID: recurso.ModuloID, TipoRecurso: recurso.Tipo,
		ContextoRecursoHuellaSHA256: huellaContexto,
		Finalidad:                   puertosbolsa.FinalidadPanelInternoBolsa, CorrelacionRef: correlacionRef,
		EsquemaHuellaSolicitud:    dominiovec.EsquemaHuellaSolicitudAutorizacionV2,
		SolicitudHuellaSHA256:     huellaSolicitud,
		EsquemaHuellaMotivo:       dominiovec.EsquemaHuellaMotivoAutorizacionV2,
		MotivoHuellaSHA256:        huellaMotivo,
		VinculoAutenticacionActor: vinculo,
		AsignacionRef:             "asignacion:panel:v1", AsignacionHuellaSHA256: huellaPanelPrueba('a'),
		VersionRolRef: "rol:panel_rrhh:v1", VersionRolHuellaSHA256: huellaPanelPrueba('b'),
		ControlVigenciaVersionRolRef:          "rol:panel_rrhh:v1",
		ControlVigenciaVersionRolRevision:     1,
		ControlVigenciaVersionRolHuellaSHA256: huellaPanelPrueba('c'),
		RevisionCatalogoPoliticas:             1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		GarantiaMinima:   dominiovec.AuthAssuranceHigh,
		CamposPermitidos: []string{puertosbolsa.CampoPanelInternoAgregado},
		EmitidaEn:        instante.Add(-time.Second), ValidaHasta: instante.Add(time.Minute),
	}
	return puertosvec.NuevaEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(decision, instante)
}

func nuevoContextoYVinculoPanelPrueba(
	t *testing.T,
	metodo dominiovec.AuthMethod,
	garantia dominiovec.AuthAssurance,
	superficie dominiovec.SuperficieAutenticacionActorV1,
) (dominiovec.ContextoActor, dominiovec.VinculoAutenticacionActorV1) {
	t.Helper()
	actor := nuevoContextoPanelPrueba(t, metodo, garantia)
	autenticacion := dominiovec.AutenticacionRevalidadaV1{
		AutenticacionRef:          "aut_0123456789abcdefghijkl",
		AutenticacionHuellaSHA256: huellaPanelPrueba('1'),
		AsercionRef:               "ase_0123456789abcdefghijkl", SesionRef: "ses_0123456789abcdefghijkl",
		ControlSesionRef: "cse_0123456789abcdefghijkl", ControlSesionRevision: 7,
		ControlSesionHuellaSHA256: huellaPanelPrueba('2'),
		CuentaRef:                 "cta_0123456789abcdefghijkl", CuentaOrdinariaRef: "cta_0123456789abcdefghijkl",
		Superficie: superficie, MetodoObservado: metodo, GarantiaObservada: garantia,
		PoliticaGarantiaRef:          "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: huellaPanelPrueba('3'),
		AutenticacionVerificadaEn:    instantePanelInternoPrueba.Add(-5 * time.Minute),
		SesionEmitidaEn:              instantePanelInternoPrueba.Add(-4 * time.Minute),
		SesionRevalidadaEn:           instantePanelInternoPrueba.Add(-3 * time.Minute),
		SesionValidaHasta:            instantePanelInternoPrueba.Add(10 * time.Minute),
	}
	vinculo, err := dominiovec.CrearVinculoAutenticacionActorV1(
		context.Background(),
		revalidadorPanelPrueba{resultado: autenticacion},
		dominiovec.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef,
			SesionRef:        autenticacion.SesionRef,
		},
		actor,
		instantePanelInternoPrueba,
	)
	if err != nil {
		t.Fatalf("crear vinculo del panel: %v", err)
	}
	return actor, vinculo
}

func nuevoContextoPanelPrueba(
	t *testing.T,
	metodo dominiovec.AuthMethod,
	garantia dominiovec.AuthAssurance,
) dominiovec.ContextoActor {
	t.Helper()
	cuenta := dominiovec.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl", Metodo: metodo, Garantia: garantia,
	}
	instantanea := dominiovec.InstantaneaContextoActor{
		VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 5,
		CuentaRef: cuenta.CuentaRef, PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 3,
		PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 4,
		Estado:       dominiovec.EstadoVinculoContextoActorActivo,
		VigenteDesde: instantePanelInternoPrueba.Add(-time.Hour),
		VigenteHasta: instantePanelInternoPrueba.Add(30 * time.Minute),
	}
	actor, err := dominiovec.NuevoContextoActor(
		cuenta,
		instantanea,
		instantePanelInternoPrueba.Add(-2*time.Minute),
	)
	if err != nil {
		t.Fatalf("crear actor del panel: %v", err)
	}
	return actor
}

func nuevaCorrelacionPanelPrueba(t *testing.T) dominiovec.ReferenciaCorrelacionAutorizacionV2 {
	t.Helper()
	referencia, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(),
		generadorCorrelacionPanelPrueba{valor: "correlacion_0123456789abcdef0123456789abcdef"},
	)
	if err != nil {
		t.Fatalf("crear correlacion: %v", err)
	}
	return referencia
}

func motivoPanelPrueba() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: "catalogo_motivos_panel_bolsa", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaPanelPrueba('d'),
		EntradaClave:         "motivo_0123456789abcdef0123456789abcdef",
	}
}

func selectorPanelPrueba() puertosbolsa.SelectorPanelInterno {
	return puertosbolsa.SelectorPanelInterno{
		Clase:            puertosbolsa.AmbitoPanelUnidad,
		OrganizacionRef:  "org_0123456789abcdef",
		UnidadGestionRef: "uni_fedcba9876543210",
	}
}

func ordenPanelPrueba(t *testing.T) OrdenConsultaPanelInterno {
	t.Helper()
	actor, vinculo := nuevoContextoYVinculoPanelPrueba(
		t,
		dominiovec.AuthMethodCertificate,
		dominiovec.AuthAssuranceHigh,
		dominiovec.SuperficieAutenticacionInternaCorporativaV1,
	)
	return OrdenConsultaPanelInterno{
		ContextoActor: actor, VinculoAutenticacionActor: vinculo,
		Selector: selectorPanelPrueba(), MotivoCatalogo: motivoPanelPrueba(),
		Correlacion: nuevaCorrelacionPanelPrueba(t),
	}
}

func instantaneaPanelPrueba(selector puertosbolsa.SelectorPanelInterno) puertosbolsa.InstantaneaPanelInterno {
	return puertosbolsa.InstantaneaPanelInterno{
		Esquema:  puertosbolsa.EsquemaPanelInternoBolsaV1,
		Selector: selector,
		Origen: puertosbolsa.OrigenPanelInterno{
			Revision: "rev_0123456789abcdef", ActualizadaEn: instantePanelInternoPrueba.Add(-time.Minute),
		},
		PruebaLectura: puertosbolsa.PruebaLecturaPanelInterno{
			LecturaRef: "lec_0123456789abcdef", AuditoriaRef: "aud_0123456789abcdef",
			AuditoriaSecuencia: 17, ConfirmadaEn: instantePanelInternoPrueba.Add(time.Microsecond),
		},
		Indicadores: puertosbolsa.IndicadoresPanelInterno{
			ConvocatoriasBorrador: 2, ConvocatoriasRevision: 1,
			ConvocatoriasPendientesFirma: 1, ConvocatoriasPublicadas: 4,
			BolsasActivas: 3, LlamamientosPendientes: 5, IncidenciasAbiertas: 1,
		},
		Convocatorias: []puertosbolsa.ResumenConvocatoriaPanelInterno{{
			ConvocatoriaRef: "cnv_0123456789abcdef", CategoriaClave: "auxiliar_administrativo",
			EstadoClave: "revision", PlazoCierraEn: instantePanelInternoPrueba.Add(48 * time.Hour),
			NumeroSolicitudes: 120, NumeroPendientes: 7,
		}},
		ActuacionesPendientes: []puertosbolsa.ActuacionPendientePanelInterno{{
			ActuacionRef: "act_0123456789abcdef", RecursoRef: "cnv_0123456789abcdef",
			TipoClave: "revisar_bases", EstadoClave: "pendiente", PrioridadClave: "alta",
			FechaLimite: instantePanelInternoPrueba.Add(24 * time.Hour), NumeroElementos: 1,
		}},
	}
}

func nuevoServicioPanelPrueba(
	t *testing.T,
) (*ServicioConsultaPanelInterno, *consultaPanelPrueba, *exigidorPanelPrueba) {
	t.Helper()
	selector := selectorPanelPrueba()
	consulta := &consultaPanelPrueba{resultado: instantaneaPanelPrueba(selector)}
	exigidor := &exigidorPanelPrueba{instante: instantePanelInternoPrueba}
	servicio, err := NuevoServicioConsultaPanelInterno(
		consulta,
		exigidor,
		relojPanelInternoPrueba{ahora: instantePanelInternoPrueba},
	)
	if err != nil {
		t.Fatalf("crear servicio de panel: %v", err)
	}
	return servicio, consulta, exigidor
}

func huellaPanelPrueba(caracter byte) string { return strings.Repeat(string(caracter), 64) }

func describirLlamadasPanel(consulta, pdp int) string {
	return fmt.Sprintf("consulta=%d PDP=%d", consulta, pdp)
}
