package internactproveedores

import (
	"context"
	"testing"

	pgvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var _ vecports.FuenteAutorizacion = (*pgvec.AlmacenAutorizacion)(nil)

func TestFuenteAutorizacionV3ConservaInstanciaYSeCierraConProveedores(t *testing.T) {
	var ausentes *Proveedores
	if ausentes.FuenteAutorizacionV3() != nil || (&Proveedores{}).FuenteAutorizacionV3() != nil {
		t.Fatal("un proveedor ausente entregó autoridad V3")
	}
	fallido, err := Construir(context.Background(), Configuracion{})
	if err == nil || fallido.FuenteAutorizacionV3() != nil {
		t.Fatalf("el montaje rechazado entregó una fuente de autorización: %v", err)
	}

	// El getter expone exactamente el adaptador ya construido; no fabrica una
	// segunda autoridad. Cerrar retira el acceso antes de reutilizar el valor.
	misma := &pgvec.AlmacenAutorizacion{}
	proveedores := &Proveedores{fuenteAutorizacion: misma}
	if obtenida := proveedores.FuenteAutorizacionV3(); obtenida != misma {
		t.Fatal("el getter sustituyó la fuente V3 original")
	}
	proveedores.Cerrar()
	if proveedores.FuenteAutorizacionV3() != nil {
		t.Fatal("el proveedor cerrado conservó la fuente V3")
	}
}
