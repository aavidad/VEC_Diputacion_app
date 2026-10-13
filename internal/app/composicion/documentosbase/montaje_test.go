package documentosbase

import (
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/almacen/ficheros"
)

type relojMontaje struct{}

func (relojMontaje) Ahora() time.Time { return time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC) }

func TestAbrirAlmacenConservaCausaSinDevolverRecursoParcial(t *testing.T) {
	var dias int64
	almacen, cerrar, err := AbrirAlmacen(t.Context(), ConfiguracionAlmacen{
		Tipo: "ficheros", Directorio: "relativo", TamanoMaximo: 1024, RetencionMinimaDias: &dias,
	}, relojMontaje{})
	if !errors.Is(err, ErrNoDisponible) || !errors.Is(err, ficheros.ErrDirectorioInseguro) || almacen != nil || cerrar != nil {
		t.Fatalf("causa perdida o recurso parcial: almacen=%v, cierre=%v, error=%v", almacen != nil, cerrar != nil, err)
	}
}
