package ficheros

import (
	"errors"
	"testing"

	"vec-diputacion-granada/internal/vec/ports"
)

// Un contexto sin proyección no produce una huella degenerada: el fallo se
// propaga con su causa (M2a, ningún fallo sin registro).
func TestComponentesContextoPropagaLaCausa(t *testing.T) {
	var contexto ports.ContextoOperacionAlmacen
	_, causa := contexto.Proyeccion()
	if causa == nil {
		t.Fatal("un contexto vacío no debería proyectarse")
	}
	componentes, err := componentesContexto(contexto)
	if componentes != nil || !errors.Is(err, ports.ErrSolicitudAlmacenInvalida) ||
		!errors.Is(err, ports.ErrAutorizacionAlmacenInvalida) {
		t.Fatalf("componentes=%v err=%v", componentes, err)
	}
	if huella, err := huellaEscritura(ports.SolicitudEscribirObjeto{Contexto: contexto}); huella != "" ||
		!errors.Is(err, ports.ErrSolicitudAlmacenInvalida) {
		t.Fatalf("escritura: huella=%q err=%v", huella, err)
	}
	if huella, err := huellaPromocion(ports.SolicitudPromoverObjeto{Contexto: contexto}, ports.ObjetoAlmacenado{}); huella != "" ||
		!errors.Is(err, ports.ErrSolicitudAlmacenInvalida) {
		t.Fatalf("promoción: huella=%q err=%v", huella, err)
	}
}
