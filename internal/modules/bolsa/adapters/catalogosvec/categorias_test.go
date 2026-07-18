package catalogosvec_test

import (
	"context"
	"errors"
	"testing"
	"time"

	catalogosvec "vec-diputacion-granada/internal/modules/bolsa/adapters/catalogosvec"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	ficherovec "vec-diputacion-granada/internal/vec/adapters/fichero"
)

const rutaCatalogoDemo = "../../../../../data/catalogos/categorias-profesionales/v1.demo.json"

func fuenteCatalogoDemo(t *testing.T) *ficherovec.ConsultaCatalogos {
	t.Helper()
	fuente, err := ficherovec.NuevaConsultaCatalogos(rutaCatalogoDemo)
	if err != nil {
		t.Fatal(err)
	}
	return fuente
}

func TestConsultaCategoriasProyectaVersionExactaPublicadaYMinimizada(t *testing.T) {
	consulta, err := catalogosvec.NuevaConsultaCategorias(fuenteCatalogoDemo(t), "categorias-profesionales", 1)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := consulta.ObtenerPublicadas(context.Background(), time.Date(2026, 7, 16, 8, 0, 0, 0, time.UTC))
	if err != nil || resultado.ID != "categorias-profesionales" || resultado.Version != 1 ||
		len(resultado.HuellaSHA256) != 64 || len(resultado.Categorias) != 68 || !resultado.Fuente.Demostracion {
		t.Fatalf("resultado=%#v error=%v", resultado, err)
	}
	primera := resultado.Categorias[0]
	if primera.Clave != "administrativo" || primera.Orden != 1 || primera.Area != "administracion_general" ||
		primera.AreaEtiqueta != "Administración general" || !primera.Suscribible {
		t.Fatalf("primera categoria=%#v", primera)
	}
}

func TestConsultaCategoriasFallaCerradoAntesDePublicacionYAnteReferenciaDistinta(t *testing.T) {
	for _, prueba := range []struct {
		nombre  string
		id      string
		version int
		fecha   time.Time
	}{
		{nombre: "antes de publicacion", id: "categorias-profesionales", version: 1, fecha: time.Date(2026, 7, 15, 23, 59, 0, 0, time.UTC)},
		{nombre: "id distinto", id: "categorias-profesionales-provincia", version: 1, fecha: time.Date(2026, 7, 16, 8, 0, 0, 0, time.UTC)},
		{nombre: "version distinta", id: "categorias-profesionales", version: 2, fecha: time.Date(2026, 7, 16, 8, 0, 0, 0, time.UTC)},
	} {
		t.Run(prueba.nombre, func(t *testing.T) {
			consulta, err := catalogosvec.NuevaConsultaCategorias(fuenteCatalogoDemo(t), prueba.id, prueba.version)
			if err != nil {
				t.Fatal(err)
			}
			_, err = consulta.ObtenerPublicadas(context.Background(), prueba.fecha)
			if !errors.Is(err, puertosbolsa.ErrCatalogoCategoriasNoDisponible) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestConsultaCategoriasRespetaCancelacion(t *testing.T) {
	consulta, err := catalogosvec.NuevaConsultaCategorias(fuenteCatalogoDemo(t), "categorias-profesionales", 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	_, err = consulta.ObtenerPublicadas(ctx, time.Now().UTC())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
}
