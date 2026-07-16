package memory

import (
	"strings"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

func contextoMemoriaPrueba(
	accion puertosbolsa.AccionOperacionBaremacion,
	recursoRef string,
	instante time.Time,
) puertosbolsa.ContextoOperacionBaremacion {
	return contextoMemoriaPruebaIdentidad(
		accion, recursoRef, principalBaremacionMemoriaPrueba, "sujeto-001", "autenticacion-1", "correlacion-1", instante,
	)
}

func contextoMemoriaPruebaIdentidad(
	accion puertosbolsa.AccionOperacionBaremacion,
	recursoRef, principalRef, sujetoRef, autenticacionRef, correlacionRef string,
	instante time.Time,
) puertosbolsa.ContextoOperacionBaremacion {
	return contextoMemoriaPruebaAutorizacion(
		accion, recursoRef, principalRef, sujetoRef, autenticacionRef, correlacionRef,
		referenciaAutorizacionMemoria(accion), instante,
	)
}

func contextoMemoriaPruebaAutorizacion(
	accion puertosbolsa.AccionOperacionBaremacion,
	recursoRef, principalRef, sujetoRef, autenticacionRef, correlacionRef, autorizacionRef string,
	instante time.Time,
) puertosbolsa.ContextoOperacionBaremacion {
	return contextoMemoriaPruebaAutorizacionConFinalidad(
		accion, recursoRef, principalRef, sujetoRef, autenticacionRef, correlacionRef,
		autorizacionRef, "baremacion_proceso_selectivo", instante,
	)
}

func contextoMemoriaPruebaAutorizacionConFinalidad(
	accion puertosbolsa.AccionOperacionBaremacion,
	recursoRef, principalRef, sujetoRef, autenticacionRef, correlacionRef, autorizacionRef, finalidad string,
	instante time.Time,
) puertosbolsa.ContextoOperacionBaremacion {
	return contextoMemoriaPruebaAutorizacionCompleta(
		accion, recursoRef, principalRef, sujetoRef, perfilBaremacionMemoriaPrueba,
		autenticacionRef, correlacionRef, autorizacionRef, finalidad, instante,
	)
}

func contextoMemoriaPruebaAutorizacionCompleta(
	accion puertosbolsa.AccionOperacionBaremacion,
	recursoRef, principalRef, sujetoRef, perfilRef, autenticacionRef, correlacionRef, autorizacionRef, finalidad string,
	instante time.Time,
) puertosbolsa.ContextoOperacionBaremacion {
	campos, existe := puertosbolsa.CamposRequeridosOperacionBaremacion(accion)
	if !existe {
		panic("accion de prueba desconocida")
	}
	clase, existe := puertosbolsa.ClaseRecursoRequeridaOperacionBaremacion(accion)
	if !existe {
		panic("clase de recurso de prueba desconocida")
	}
	recurso := dominiovec.RecursoAutorizable{
		Referencia: recursoRef, ModuloID: "bolsa", Tipo: string(clase),
		Ambitos: map[string]string{"sujeto_ref": sujetoRef},
	}
	huellaContexto, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		panic(err)
	}
	huellaCatalogo, err := dominiovec.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		panic(err)
	}
	contextoActor, vinculo, err := pruebasvec.NuevoContextoYVinculo(
		instante, principalRef, perfilRef,
		dominiovec.AuthMethodCertificate, dominiovec.AuthAssuranceHigh,
	)
	if err != nil {
		panic(err)
	}
	datosVinculo, err := vinculo.Datos()
	if err != nil {
		panic(err)
	}
	decision := dominiovec.DecisionAutorizacion{
		DecisionRef: autorizacionRef, Concedida: true, Codigo: "concedida", PrincipalID: contextoActor.Principal.ID,
		PerfilActivoRef: contextoActor.PerfilActivoRef, Accion: string(accion), RecursoRef: recursoRef,
		ModuloID: "bolsa", TipoRecurso: string(clase), ContextoRecursoHuellaSHA256: huellaContexto,
		Finalidad: finalidad, CorrelacionRef: correlacionRef,
		VinculoAutenticacionActor: vinculo,
		AsignacionRef:             "asignacion-tecnico-v1", AsignacionHuellaSHA256: huellaMemoria("1"),
		VersionRolRef: "rol-tecnico-v1", VersionRolHuellaSHA256: huellaMemoria("2"),
		ControlVigenciaVersionRolRef:      "rol-tecnico-v1",
		ControlVigenciaVersionRolRevision: 1, ControlVigenciaVersionRolHuellaSHA256: huellaMemoria("3"),
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		PoliticasEvaluadasHuellasSHA256: map[string]string{},
		GarantiaMinima:                  dominiovec.AuthAssuranceHigh, CamposPermitidos: campos,
		EmitidaEn: instante.Add(-time.Minute), ValidaHasta: instante.Add(4 * time.Minute),
	}
	contexto, err := puertosbolsa.NuevaAutorizacionOperacionBaremacion(
		decision,
		puertosbolsa.VinculoAutenticacionBaremacion{
			SujetoRef: sujetoRef, Metodo: datosVinculo.MetodoObservado,
			Garantia: datosVinculo.GarantiaObservada, AutenticacionRef: datosVinculo.AutenticacionRef,
			SesionRef: datosVinculo.SesionRef, SesionEmitidaEn: datosVinculo.SesionEmitidaEn,
			SesionValidaHasta: datosVinculo.SesionValidaHasta, VinculoAutenticacionActor: vinculo,
		},
		instante,
	)
	if err != nil {
		panic(err)
	}
	return contexto
}

func referenciaAutorizacionMemoria(accion puertosbolsa.AccionOperacionBaremacion) string {
	return "autorizacion-" + strings.ReplaceAll(string(accion), ".", "-")
}
