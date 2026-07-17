package calculoexperienciaoficial

import (
	"context"
	"errors"
	"strings"
	"time"

	oficial "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperienciaoficial"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type relojPrueba struct{ ahora time.Time }

func (r relojPrueba) Ahora() time.Time { return r.ahora }

type fuentePrueba struct {
	resultado puertosbolsa.FuenteExactaCalculoReglasBaremo
	error     error
	llamadas  int
	cancelar  context.CancelFunc
}

func (f *fuentePrueba) ObtenerFuenteExacta(
	ctx context.Context,
	_ puertosbolsa.SolicitudFuenteExactaCalculoReglasBaremo,
) (puertosbolsa.FuenteExactaCalculoReglasBaremo, error) {
	f.llamadas++
	if err := ctx.Err(); err != nil {
		return puertosbolsa.FuenteExactaCalculoReglasBaremo{}, err
	}
	if f.cancelar != nil {
		f.cancelar()
	}
	return f.resultado, f.error
}

type llamadaAutorizacionPrueba struct {
	recurso     dominiovec.RecursoAutorizable
	correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2
}

type exigidorPrueba struct {
	ahora            time.Time
	fallarEn         int
	antiguaEn        int
	cancelarEn       int
	cancelar         context.CancelFunc
	garantiaDecision dominiovec.AuthAssurance
	llamadas         []llamadaAutorizacionPrueba
}

func (e *exigidorPrueba) ExigirEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(
	ctx context.Context,
	actor dominiovec.ContextoActor,
	vinculo dominiovec.VinculoAutenticacionActorV1,
	recurso dominiovec.RecursoAutorizable,
	correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2,
	motivo dominiovec.ReferenciaEntradaCatalogo,
	_ aplicacionvec.PoliticaUsoDecisionAutorizacion,
) (puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2, error) {
	e.llamadas = append(e.llamadas, llamadaAutorizacionPrueba{recurso, correlacion})
	if err := ctx.Err(); err != nil {
		return puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}, err
	}
	if e.fallarEn == len(e.llamadas) {
		return puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}, errors.New("denegada")
	}
	accion, finalidad := accionLeerFuenteCalculo, finalidadLeerFuente
	campos := []string{"fuente_reglas", "instantanea_experiencia", "prueba_procedencia"}
	if recurso.Tipo == tipoRecursoCalculoOficial || recurso.Tipo == tipoRecursoRectificacion {
		accion, finalidad = accionConfirmarCalculo, finalidadConfirmarCalculo
		campos = []string{"auditoria", "resultado_canonico", "salida_eventos"}
		if recurso.Tipo == tipoRecursoRectificacion {
			accion, finalidad = accionRectificarCalculo, finalidadRectificarCalculo
		}
	}
	correlacionRef, err := correlacion.ValorCanonico()
	if err != nil {
		return puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}, err
	}
	solicitud := debeSinPrueba(dominiovec.NuevaSolicitudAutorizacionLigadaV2(
		dominiovec.DatosSolicitudAutorizacionLigadaV2{
			ContextoActor: actor, VinculoAutenticacionActor: vinculo,
			ReferenciaMotivo: motivo, Accion: accion, Recurso: recurso,
			Finalidad: finalidad, Correlacion: correlacion,
		},
	))
	huellaSolicitud := debeSinPrueba(dominiovec.HuellaSHA256SolicitudAutorizacionV2(solicitud))
	huellaMotivo := debeSinPrueba(dominiovec.HuellaSHA256MotivoAutorizacionV2(motivo))
	huellaContexto := debeSinPrueba(recurso.HuellaContextoAutorizacionSHA256())
	huellaCatalogo := debeSinPrueba(dominiovec.HuellaEvidenciasCatalogoPoliticasAutorizacion(nil, nil))
	garantia := dominiovec.AuthAssuranceSubstantial
	datosVinculo, _ := vinculo.Datos()
	if datosVinculo.Superficie != dominiovec.SuperficieAutenticacionExternaPersonalV1 {
		garantia = dominiovec.AuthAssuranceHigh
	}
	indice := len(e.llamadas)
	if e.garantiaDecision.Valida() {
		garantia = e.garantiaDecision
	}
	verificadaEn := e.ahora
	if e.antiguaEn == indice {
		verificadaEn = e.ahora.Add(-30 * time.Second)
	}
	decision := dominiovec.DecisionAutorizacion{
		DecisionRef: "decision:calculo:" + string(rune('0'+indice)), Concedida: true, Codigo: "concedida",
		PrincipalID: actor.Principal.ID, PerfilActivoRef: actor.PerfilActivoRef,
		Accion: accion, RecursoRef: recurso.Referencia, ModuloID: recurso.ModuloID,
		TipoRecurso: recurso.Tipo, ContextoRecursoHuellaSHA256: huellaContexto,
		Finalidad: finalidad, CorrelacionRef: correlacionRef,
		EsquemaHuellaSolicitud: dominiovec.EsquemaHuellaSolicitudAutorizacionV2,
		SolicitudHuellaSHA256:  huellaSolicitud,
		EsquemaHuellaMotivo:    dominiovec.EsquemaHuellaMotivoAutorizacionV2,
		MotivoHuellaSHA256:     huellaMotivo, VinculoAutenticacionActor: vinculo,
		AsignacionRef: "asignacion:calculo:v1", AsignacionHuellaSHA256: strings.Repeat("a", 64),
		VersionRolRef: "rol:calculo:v1", VersionRolHuellaSHA256: strings.Repeat("b", 64),
		ControlVigenciaVersionRolRef: "rol:calculo:v1", ControlVigenciaVersionRolRevision: 1,
		ControlVigenciaVersionRolHuellaSHA256: strings.Repeat("c", 64),
		RevisionCatalogoPoliticas:             1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		GarantiaMinima: garantia, CamposPermitidos: campos,
		EmitidaEn: verificadaEn.Add(-time.Second), ValidaHasta: e.ahora.Add(time.Minute),
	}
	evidencia, err := puertosvec.NuevaEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(
		decision, verificadaEn,
	)
	if err == nil && e.cancelar != nil && e.cancelarEn == indice {
		e.cancelar()
	}
	return evidencia, err
}

type confirmadorPrueba struct {
	llamadas            int
	reconciliaciones    int
	error               error
	errorReconciliacion error
	cometerYFallar      bool
	adulterar           bool
	adulterarIntento    bool
	desenlace           DesenlaceConfirmacionDuradera
	ultimaSolicitud     SolicitudConfirmacionDuradera
	ultimoResultado     ResultadoConfirmacionDuradera
}

func (c *confirmadorPrueba) Confirmar(
	ctx context.Context,
	solicitud SolicitudConfirmacionDuradera,
) (ResultadoConfirmacionDuradera, error) {
	c.llamadas++
	if err := ctx.Err(); err != nil {
		return ResultadoConfirmacionDuradera{}, err
	}
	datos, err := solicitud.Datos()
	if err != nil {
		return ResultadoConfirmacionDuradera{}, err
	}
	c.ultimaSolicitud = solicitud
	if c.error != nil && !c.cometerYFallar {
		return ResultadoConfirmacionDuradera{}, c.error
	}
	intencion := datos.Intencion
	if c.adulterar {
		intencion = debeSinPrueba(oficial.NuevaIntencionResultadoV1(
			datos.Clave, strings.Repeat("f", 64), intencion.Estado(), intencion.Fase(),
		))
	}
	indice := strings.Repeat("9", 64)
	recibo := debeSinPrueba(oficial.NuevoReciboV1(
		"recibo:calculo:oficial:1", 1, indice, intencion,
	))
	referenciaIntento := datos.ReferenciaIntento
	if c.adulterarIntento {
		referenciaIntento += ":otro"
	}
	desenlace := c.desenlace
	if !desenlace.valido() {
		desenlace = ConfirmacionCreada
	}
	resultado, err := NuevoResultadoConfirmacionDuradera(
		referenciaIntento, recibo, indice, intencion.HuellaResultadoSHA256(), desenlace,
	)
	if err == nil {
		c.ultimoResultado = resultado
	}
	if c.error != nil {
		return ResultadoConfirmacionDuradera{}, c.error
	}
	return resultado, err
}

func (c *confirmadorPrueba) Reconciliar(
	ctx context.Context,
	_ SolicitudReconciliacionDuradera,
) (ResultadoConfirmacionDuradera, error) {
	c.reconciliaciones++
	if err := ctx.Err(); err != nil {
		return ResultadoConfirmacionDuradera{}, err
	}
	if c.errorReconciliacion != nil {
		return ResultadoConfirmacionDuradera{}, c.errorReconciliacion
	}
	if c.ultimoResultado.validarEstructura() != nil {
		return ResultadoConfirmacionDuradera{}, errors.New("resultado no encontrado")
	}
	return c.ultimoResultado, nil
}

func debeSinPrueba[T any](valor T, err error) T {
	if err != nil {
		panic(err)
	}
	return valor
}
