package domain

import "testing"

func TestCapacidadActualV3ConsultaPublicarSinEmitirDecision(t *testing.T) {
	vinculo, resultado, instantanea, consulta, ahora := escenarioCapacidadesInformativasV3Prueba(t)
	accion := "bolsa.politica_transiciones.publicar"
	permitida, err := PuedeEjecutarAccionActualV3(vinculo, resultado, instantanea,
		accion, consulta.Recurso, consulta.Finalidad, ahora)
	if err != nil || permitida {
		t.Fatalf("una lectura creó permiso de publicación: %t %v", permitida, err)
	}
	instantanea.VersionRol.Concesiones = append(instantanea.VersionRol.Concesiones,
		ConcesionRol{Accion: accion, ModuloID: consulta.Recurso.ModuloID, TipoRecurso: consulta.Recurso.Tipo,
			Finalidades: []string{consulta.Finalidad}, GarantiaMinima: AuthAssuranceHigh})
	huella, err := HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	instantanea.Politicas = nil
	instantanea.CatalogoPoliticasHuellaSHA256 = huella
	permitida, err = PuedeEjecutarAccionActualV3(vinculo, resultado, instantanea,
		accion, consulta.Recurso, consulta.Finalidad, ahora)
	if err != nil || !permitida {
		t.Fatalf("la concesión positiva propia no se leyó: %t %v", permitida, err)
	}
}
