package main

import (
	"fmt"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/app/administracion"
)

func TestEtapaComposicionADMINConservaSoloCatalogo(t *testing.T) {
	privado := "nombre_personal_secreto"
	for _, caso := range []struct {
		etapa  string
		quiere string
	}{
		{etapa: "lector_usuarios", quiere: "lector_usuarios"},
		{etapa: "pool_11_grupo", quiere: "pool_11_grupo"},
		{etapa: "gobierno_roles_fuente", quiere: "gobierno_roles_fuente"},
		{etapa: "pool_16_dsn", quiere: "pool_16_dsn"},
		{etapa: "cargos_confianza_material", quiere: "cargos_confianza_material"},
		{etapa: privado, quiere: "composicion"},
		{etapa: "pool_17_dsn", quiere: "composicion"},
	} {
		err := fmt.Errorf("%w: etapa=%s", administracion.ErrConfiguracion, caso.etapa)
		got := etapaComposicionADMIN(err)
		if got != caso.quiere || strings.Contains(got, privado) {
			t.Fatalf("etapa %q: obtenido %q; esperado %q", caso.etapa, got, caso.quiere)
		}
	}
	if got := etapaComposicionADMIN(fmt.Errorf("texto privado: etapa=%s", privado)); got != "composicion" {
		t.Fatalf("error no catalogado expuesto: %q", got)
	}
}
