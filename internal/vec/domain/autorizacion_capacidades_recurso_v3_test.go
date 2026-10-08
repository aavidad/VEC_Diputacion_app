package domain

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"reflect"
	"testing"
	"time"
)

func escenarioCapacidadesInformativasV3Prueba(t *testing.T) (
	VinculoAutenticacionActorV2, ResultadoContextoActorRegistradoV2,
	InstantaneaAutorizacion, ConsultaCapacidadInformativaV3, time.Time,
) {
	t.Helper()
	ahora := instanteVinculoAutenticacionActorV2Prueba()
	fuente := resultadoContextoActorRegistradoV2Prueba(t, ahora)
	autenticacion := autenticacionRevalidadaVinculoPrueba(ahora)
	vinculo, resultado, err := CrearVinculoAutenticacionActorV2ConResultado(
		context.Background(), &revalidadorAutenticacionV2Prueba{resultado: autenticacion},
		solicitudRevalidacionVinculoPrueba(autenticacion),
		&resolutorContextoRegistradoV2Prueba{resultado: fuente},
		solicitudContextoVinculoV2Prueba(fuente), &relojVinculoV2Prueba{ahora: ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	datosVinculo, err := vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	solicitud, instantanea, _ := escenarioDecisionAutorizacionV3Prueba(t)
	datosSolicitud, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	instantanea.AsignacionPerfil.PrincipalID = datosVinculo.PrincipalID
	instantanea.AsignacionPerfil.PerfilActivoRef = datosVinculo.PerfilActivoRef
	if err := instantanea.Validar(); err != nil {
		t.Fatalf("instantanea de prueba invalida: %v", err)
	}
	return vinculo, resultado, instantanea, ConsultaCapacidadInformativaV3{
		Accion: datosSolicitud.Accion, Recurso: datosSolicitud.Recurso, Finalidad: datosSolicitud.Finalidad,
	}, ahora
}

func TestCapacidadesInformativasV3ReusanEvaluadorExactoSinMutarInstantanea(t *testing.T) {
	vinculo, resultado, instantanea, consulta, ahora := escenarioCapacidadesInformativasV3Prueba(t)
	antes, err := json.Marshal(instantanea)
	if err != nil {
		t.Fatal(err)
	}
	noAmbito := consulta
	noAmbito.Recurso.Ambitos = maps.Clone(noAmbito.Recurso.Ambitos)
	noAmbito.Recurso.Ambitos["unidad"] = "ajena"
	noFinalidad := consulta
	noFinalidad.Finalidad = "otra_finalidad"
	evaluadas, limite, err := EvaluarCapacidadesInformativasV3(vinculo, resultado, instantanea,
		[]ConsultaCapacidadInformativaV3{consulta, noAmbito, noFinalidad}, ahora)
	if err != nil || len(evaluadas) != 3 || !limite.After(ahora) {
		t.Fatalf("lote invalido: resultados=%+v limite=%v err=%v", evaluadas, limite, err)
	}
	if evaluadas[1].Concedida || evaluadas[2].Concedida {
		t.Fatalf("ambito o finalidad ajenos concedidos: %+v", evaluadas)
	}
	despues, err := json.Marshal(instantanea)
	if err != nil || !reflect.DeepEqual(despues, antes) {
		t.Fatal("la evaluacion modifico la instantanea")
	}

	base, _, _ := escenarioDecisionAutorizacionV3Prueba(t)
	datosBase, err := base.Datos()
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := NuevaSolicitudAutorizacionLigadaV3(DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: vinculo, ReferenciaMotivo: datosBase.ReferenciaMotivo,
		Accion: consulta.Accion, Recurso: consulta.Recurso, Finalidad: consulta.Finalidad,
		Correlacion: datosBase.Correlacion,
	})
	if err != nil {
		t.Fatal(err)
	}
	evidencia, err := NuevaEvidenciaEvaluacionAutorizacionV3(solicitud, instantanea,
		"dec_capacidad_paridad_0123456789", ahora, ahora.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if evaluadas[0].Concedida != evidencia.datos.concedida ||
		!reflect.DeepEqual(evaluadas[0].CamposPermitidos, evidencia.datos.camposPermitidos) ||
		!reflect.DeepEqual(evaluadas[0].Obligaciones, evidencia.datos.obligaciones) {
		t.Fatalf("proyeccion y PDP divergentes: informativa=%+v PDP=%+v", evaluadas[0], evidencia.datos)
	}
	evaluadas[0].CamposPermitidos[0] = "alterado"
	if instantanea.VersionRol.Concesiones[0].CamposPermitidos[0] == "alterado" {
		t.Fatal("el resultado comparte campos con la concesion")
	}
}

func TestCapacidadesInformativasV3CierranPerfilYVigencia(t *testing.T) {
	vinculo, resultado, instantanea, consulta, ahora := escenarioCapacidadesInformativasV3Prueba(t)
	ajena := instantanea
	ajena.AsignacionPerfil.PerfilActivoRef = "perfil_ajeno"
	if _, _, err := EvaluarCapacidadesInformativasV3(vinculo, resultado, ajena,
		[]ConsultaCapacidadInformativaV3{consulta}, ahora); !errors.Is(err, ErrAutorizacionDenegada) {
		t.Fatalf("perfil ajeno aceptado: %v", err)
	}
	if _, _, err := EvaluarCapacidadesInformativasV3(vinculo, resultado, instantanea,
		[]ConsultaCapacidadInformativaV3{consulta}, resultado.Contexto.Instantanea.VigenteHasta); !errors.Is(err, ErrAutorizacionDenegada) {
		t.Fatalf("contexto vencido aceptado: %v", err)
	}
	consulta.Accion = "*"
	if _, _, err := EvaluarCapacidadesInformativasV3(vinculo, resultado, instantanea,
		[]ConsultaCapacidadInformativaV3{consulta}, ahora); !errors.Is(err, ErrSolicitudAutorizacionInvalida) {
		t.Fatalf("accion comodin aceptada: %v", err)
	}
}

func TestCapacidadesInformativasV3VencenConPrimerVinculoReferencia(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
	contexto := InstantaneaContextoActor{VigenteHasta: ahora.Add(30 * time.Minute),
		Vinculos: []VinculoReferenciaContextoActor{
			{VigenteHasta: ahora.Add(8 * time.Minute)},
			{VigenteHasta: ahora.Add(time.Minute)},
		}}
	limite := limiteContextoCapacidadInformativaV3(contexto, ahora.Add(5*time.Minute))
	if !limite.Equal(ahora.Add(time.Minute)) || limite.After(ahora.Add(2*time.Minute)) {
		t.Fatalf("una capacidad seguiria disponible tras vencer su vinculo: %v", limite)
	}
}
