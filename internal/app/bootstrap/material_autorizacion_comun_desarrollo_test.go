package bootstrap

import "testing"

func TestCatalogoMaterialComunEsNominalYRechazaDuplicados(t *testing.T) {
	d := descriptorMaterialConsumidorV3Desarrollo{Audiencia: "audiencia.cronos", Dominio: "vec.cronos.material", Prefijo: "clave:cronos:", ProveedorNominal: "proveedor-cronos"}
	for _, casos := range [][]descriptorMaterialConsumidorV3Desarrollo{{d, d}, {d, {Audiencia: "otra", Dominio: d.Dominio, Prefijo: "clave:otra:", ProveedorNominal: "p"}}, {d, {Audiencia: "otra", Dominio: "otro", Prefijo: d.Prefijo, ProveedorNominal: "p"}}} {
		if _, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(casos); err == nil {
			t.Fatal("duplicado de material admitido")
		}
	}
	c, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo([]descriptorMaterialConsumidorV3Desarrollo{d})
	if err != nil {
		t.Fatal(err)
	}
	d.Prefijo = "alterado"
	r, ok := c.descriptorPara("audiencia.cronos")
	if !ok || r.Prefijo != "clave:cronos:" {
		t.Fatal("catálogo no copió descriptor")
	}
	if _, ok := c.descriptorPara("audiencia.desconocida"); ok {
		t.Fatal("audiencia desconocida admitida")
	}
}
