package fichero

import (
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/reglas"
)

func TestCatalogoTelematicoVersionaLaAceptacionPrevia(t *testing.T) {
	consulta, err := NuevaConsultaCatalogos("../../../../data/demo/reglas/bolsa_reglas.rrhh-20261002.v3.json")
	if err != nil {
		t.Fatal(err)
	}
	resolutor, err := reglas.NuevoResolutor(reglas.Configuracion{CatalogoID: "vec.bolsa.reglas", ModuloID: "bolsa", Consulta: consulta, Metadatos: consulta,
		Reloj: relojReglasRRHH{time.Date(2026, 10, 2, 11, 0, 0, 0, time.UTC)}})
	if err != nil {
		t.Fatal(err)
	}
	r, err := resolutor.Regla(t.Context(), "b30.confirmacion_adjudicacion")
	if err != nil || r.Valor != "aceptacion_previa" || r.ReferenciaEntrada.CatalogoVersion != 3 || r.Atributos["silencio"] != "sin_consecuencias" || r.Atributos["resto"] != "disponible" {
		t.Fatalf("regla telemática=%+v, error=%v", r, err)
	}
	plazo, err := resolutor.Regla(t.Context(), reglas.BolsaPlazoPublicacion)
	if err != nil || plazo.Cantidad != 2 || plazo.Inicio != "notificacion" {
		t.Fatalf("plazo de la versión anterior perdido: %+v, %v", plazo, err)
	}
}
