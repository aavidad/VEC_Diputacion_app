package reglasvec

import (
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestMotivosSaltoCatalogoSinMotivos(t *testing.T) {
	catalogo := NuevoCatalogoMotivosSalto(catalogoPrueba(t, nil).resolutor)
	motivos, err := catalogo.MotivosSaltoOrden(t.Context())
	if err != nil || motivos == nil || len(motivos) != 0 {
		t.Fatalf("sin entradas b01.salto.* no hay motivos: %v %v", motivos, err)
	}
	for _, clave := range []string{"b01.salto.urgencia", "b01.orden_prelacion", "b24.sancion.apercibimiento", "b01.salto.Mayus"} {
		if _, err := catalogo.ResolverMotivoSaltoOrden(t.Context(), clave); !errors.Is(err, ports.ErrMotivoSaltoNoAdmitido) {
			t.Fatalf("%s no debe admitirse: %v", clave, err)
		}
	}
	if NuevoCatalogoMotivosSalto(nil) != nil {
		t.Fatal("sin resolutor no hay catálogo")
	}
}

func TestMotivosSaltoCatalogoConMotivo(t *testing.T) {
	sanciones := catalogoModificado(t, func(documento map[string]any) {
		base := entradaCatalogo(documento, "b01.orden_prelacion")
		copia := map[string]any{}
		for k, v := range base {
			copia[k] = v
		}
		copia["clave"] = "b01.salto.motivo_de_prueba"
		copia["etiqueta"] = "Motivo de prueba"
		copia["orden"] = 999
		catalogo := documento["catalogo"].(map[string]any)
		catalogo["entradas"] = append(catalogo["entradas"].([]any), copia)
	})
	catalogo := NuevoCatalogoMotivosSalto(sanciones.resolutor)
	motivos, err := catalogo.MotivosSaltoOrden(t.Context())
	if err != nil || len(motivos) != 1 || motivos[0] != (ports.MotivoSaltoOrden{Clave: "b01.salto.motivo_de_prueba", Etiqueta: "Motivo de prueba"}) {
		t.Fatalf("motivos: %v %v", motivos, err)
	}
	salto, err := catalogo.ResolverMotivoSaltoOrden(t.Context(), "b01.salto.motivo_de_prueba")
	if err != nil || salto.Clave != "b01.salto.motivo_de_prueba" || len(salto.HuellaCatalogo) != 64 ||
		salto.ReglaRef == "" || salto.ReglaRef[len(salto.ReglaRef)-len(":b01.salto.motivo_de_prueba"):] != ":b01.salto.motivo_de_prueba" {
		t.Fatalf("salto: %+v %v", salto, err)
	}
}
