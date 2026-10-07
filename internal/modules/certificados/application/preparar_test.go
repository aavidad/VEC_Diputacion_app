package application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"vec-diputacion-granada/internal/modules/certificados/adapters/fichero"
	"vec-diputacion-granada/internal/modules/certificados/application"
	"vec-diputacion-granada/internal/modules/certificados/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type fuente struct{ valor domain.FuenteServicios }

func (f fuente) Obtener(context.Context) (domain.FuenteServicios, error) { return f.valor, nil }

type renderizador struct {
	contenido vecdomain.ContenidoDocumento
	llamadas  int
}

func (r *renderizador) Renderizar(_ context.Context, c vecdomain.ContenidoDocumento) ([]byte, error) {
	r.llamadas++
	r.contenido = c
	return []byte("%PDF-ensayo"), nil
}
func preparar(t *testing.T) (*application.Preparador, domain.FuenteServicios, *renderizador) {
	t.Helper()
	f, e := (fichero.FuenteServicios{Ruta: "../adapters/fichero/testdata/servicios.ensayo.json"}).Obtener(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	r := &renderizador{}
	p := &application.Preparador{Fuente: fuente{f}, Catalogo: fichero.CatalogoPlantillas{
		RutaPlantilla: "../../../../data/certificados/plantillas/servicios.v1.json",
		RutaTextos:    "../../../../web/static/textos/es/certificados.json"}, Renderizador: r}
	return p, f, r
}
func TestPreparacionConservaServiciosSinSumarNiCambiarEstados(t *testing.T) {
	p, f, r := preparar(t)
	// Deliberate overlap: review material preserves both records, not an invented total.
	f.Servicios[1].Inicio = f.Servicios[0].Inicio
	f.Servicios[1].Fin = f.Servicios[0].Fin
	p.Fuente = fuente{f}
	x, e := p.PrepararEnsayo(context.Background(), application.Orden{PlantillaID: "servicios", Version: 1, Idioma: "es"})
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(x.Borrador.Contenido, r.contenido) {
		t.Fatal("PDF/JSON content mismatch")
	}
	if !reflect.DeepEqual(x.Borrador.Fuente.Servicios, f.Servicios) {
		t.Fatal("source changed")
	}
	for i, g := range x.Borrador.Grupos {
		if len(g.Servicios) != 1 || !reflect.DeepEqual(g.Servicios[0], f.Servicios[i]) || g.Estado != f.Servicios[i].Estado {
			t.Fatal("state changed")
		}
	}
	*f.Servicios[0].Dias = 999
	if *x.Borrador.Fuente.Servicios[0].Dias == 999 || *x.Borrador.Grupos[0].Servicios[0].Dias == 999 {
		t.Fatal("mutable alias")
	}
}
func TestEntradaInvalidaNuncaRenderiza(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		alterar func(*domain.FuenteServicios)
	}{
		{"real", func(f *domain.FuenteServicios) { f.Sintetica = false }},
		{"invalid_leap", func(f *domain.FuenteServicios) { f.Servicios[0].Inicio = "2023-02-29" }},
		{"outside_cut", func(f *domain.FuenteServicios) { f.Servicios[0].Fin = "2027-01-01" }},
		{"state", func(f *domain.FuenteServicios) { f.Servicios[0].Estado = "aprobado" }},
		{"control", func(f *domain.FuenteServicios) { f.Nombre = "name\nsecond" }},
		{"too_many", func(f *domain.FuenteServicios) { f.Servicios = make([]domain.Servicio, 201) }},
		{"missing", func(f *domain.FuenteServicios) { f.Servicios = nil }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			p, f, r := preparar(t)
			caso.alterar(&f)
			p.Fuente = fuente{f}
			_, e := p.PrepararEnsayo(context.Background(), application.Orden{PlantillaID: "servicios", Version: 1, Idioma: "es"})
			if !errors.Is(e, domain.ErrEntrada) || r.llamadas != 0 {
				t.Fatalf("err=%v calls=%d", e, r.llamadas)
			}
		})
	}
}
func TestVersionExactaYCancelacion(t *testing.T) {
	p, _, r := preparar(t)
	_, e := p.PrepararEnsayo(context.Background(), application.Orden{PlantillaID: "servicios", Version: 2, Idioma: "es"})
	if !errors.Is(e, domain.ErrCatalogo) || r.llamadas != 0 {
		t.Fatal("version not enforced")
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	_, e = p.PrepararEnsayo(ctx, application.Orden{PlantillaID: "servicios", Version: 1, Idioma: "es"})
	if !errors.Is(e, context.Canceled) || r.llamadas != 0 {
		t.Fatal("cancellation not enforced")
	}
}
