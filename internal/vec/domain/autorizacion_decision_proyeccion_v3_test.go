package domain

import (
	"testing"
	"time"
)

func TestProyeccionDecisionLigadaV3ExigeCamposYObligacionesYCopia(t *testing.T) {
	solicitud, instantanea, emitida := escenarioDecisionAutorizacionV3Prueba(t)
	evidencia, err := NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud, instantanea, "dec_proyeccion_0123456789abcdef", emitida, emitida.Add(time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	permitidas := []string{"estado"}
	soportadas := []string{"auditar_acceso", "registrar_revision"}
	if err := decision.ExigirProyeccionPara(solicitud, permitidas, soportadas); err != nil {
		t.Fatalf("proyeccion concedida: %v", err)
	}
	for nombre, caso := range map[string]struct{ campos, obligaciones []string }{
		"campo ausente":    {[]string{"nombre"}, soportadas},
		"campo futuro":     {[]string{"estado", "dato_futuro"}, soportadas},
		"campo vacio":      {nil, soportadas},
		"campo repetido":   {[]string{"estado", "estado"}, soportadas},
		"obligacion nueva": {permitidas, []string{"registrar_revision"}},
	} {
		if err := decision.ExigirProyeccionPara(solicitud, caso.campos, caso.obligaciones); err == nil {
			t.Fatalf("%s aceptado", nombre)
		}
	}
	copia, err := decision.RestriccionesProyeccionPara(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	copia.CamposPermitidos[0] = "nombre"
	copia.Obligaciones[0] = "sin_auditoria"
	if err := decision.ExigirProyeccionPara(solicitud, permitidas, soportadas); err != nil {
		t.Fatalf("la copia altero la decision: %v", err)
	}
	if err := (DecisionAutorizacionLigadaV3{}).ExigirProyeccionPara(solicitud, permitidas, soportadas); err == nil {
		t.Fatal("decision vacia aceptada")
	}
}

func TestProyeccionDecisionLigadaV3DeniegaCampoFuturoEnConcesion(t *testing.T) {
	solicitud, instantanea, emitida := escenarioDecisionAutorizacionV3Prueba(t)
	instantanea.VersionRol.Concesiones[0].CamposPermitidos = []string{"estado", "dato_futuro"}
	instantanea.Politicas[0].CamposPermitidos = []string{"estado", "dato_futuro"}
	instantanea.CatalogoPoliticasHuellaSHA256 = huellaCatalogoDecisionAutorizacionV3Prueba(t, instantanea.Politicas)
	evidencia, err := NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud, instantanea, "dec_futura_0123456789abcdef", emitida, emitida.Add(time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	restricciones, err := decision.RestriccionesProyeccionPara(solicitud)
	if err != nil || len(restricciones.CamposPermitidos) != 2 {
		t.Fatalf("fixture sin campo adicional: %+v, %v", restricciones, err)
	}
	if err := decision.ExigirProyeccionPara(solicitud, []string{"estado"}, []string{"auditar_acceso", "registrar_revision"}); err == nil {
		t.Fatal("decision con campo futuro no fue denegada")
	}
}
