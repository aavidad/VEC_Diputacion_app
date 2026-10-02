package fichero

import (
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/reglas"
)

type relojReglasRRHH struct{ instante time.Time }

func (r relojReglasRRHH) Ahora() time.Time { return r.instante }

func TestCatalogoRRHHVersionaDosDiasDesdeNotificacionSinReescribirV1(t *testing.T) {
	nueva, err := NuevaConsultaCatalogos("../../../../data/demo/reglas/bolsa_reglas.rrhh-20261002.v2.json")
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Date(2026, 10, 2, 11, 0, 0, 0, time.UTC)
	resolutor, err := reglas.NuevoResolutor(reglas.Configuracion{CatalogoID: "vec.bolsa.reglas", ModuloID: "bolsa", Consulta: nueva, Metadatos: nueva, Reloj: relojReglasRRHH{ahora}})
	if err != nil {
		t.Fatal(err)
	}
	r, err := resolutor.Regla(t.Context(), reglas.BolsaPlazoPublicacion)
	if err != nil || r.Cantidad != 2 || r.Inicio != "notificacion" || r.ReferenciaEntrada.CatalogoVersion != 2 || r.Atributos["calendario_ratificado"] != "false" {
		t.Fatalf("regla=%+v err=%v", r, err)
	}
	anterior, err := NuevaConsultaCatalogos("../../../../data/demo/reglas/bolsa_reglas.ejemplo.demo.json")
	if err != nil {
		t.Fatal(err)
	}
	c, err := anterior.ObtenerCatalogo(t.Context(), "vec.bolsa.reglas", 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range c.Entradas {
		if e.Clave == reglas.BolsaPlazoPublicacion && e.Atributos["inicio"] != "publicacion" {
			t.Fatal("se reescribió la regla histórica")
		}
	}
}
