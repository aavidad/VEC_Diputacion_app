package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/application"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

type consultaCategoriasDoble struct {
	resultado ports.CatalogoCategoriasProfesionalesConsultable
	error     error
	observado time.Time
	alLeer    func()
}

func (c *consultaCategoriasDoble) ObtenerVigentes(_ context.Context, instante time.Time) (ports.CatalogoCategoriasProfesionalesConsultable, error) {
	c.observado = instante
	if c.alLeer != nil {
		c.alLeer()
	}
	return c.resultado, c.error
}

type relojCategoriasDoble struct{ instante time.Time }

func (r *relojCategoriasDoble) Ahora() time.Time { return r.instante }

func catalogoCategoriasAplicacionPrueba() ports.CatalogoCategoriasProfesionalesConsultable {
	return ports.CatalogoCategoriasProfesionalesConsultable{
		Referencia: ports.ReferenciaCatalogoCategoriasProfesionales{
			CatalogoID:           "categorias-profesionales",
			CatalogoVersion:      1,
			CatalogoHuellaSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
		Fuente: ports.FuenteCategoriasProfesionalesConsultable{
			Revision: "fuente-prueba-v1", ActualizadaEn: time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC),
			Demostracion: true, Aviso: "DEMOSTRACIÓN: fuente sintética de prueba.",
		},
		Categorias: []ports.CategoriaProfesionalConsultable{{
			Clave: "administrativo", Etiqueta: "Administrativo", Orden: 1,
			Area: "administracion_general", AreaEtiqueta: "Administración general",
		}},
	}
}

func TestServicioCategoriasProfesionalesUsaRelojServidorYClona(t *testing.T) {
	instanteLocal := time.Date(2026, 7, 16, 10, 30, 0, 123456789, time.FixedZone("CEST", 2*60*60))
	consulta := &consultaCategoriasDoble{resultado: catalogoCategoriasAplicacionPrueba()}
	servicio, err := application.NuevoServicioConsultaCategoriasProfesionales(
		consulta, &relojCategoriasDoble{instante: instanteLocal},
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := servicio.ListarVigentes(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	esperado := instanteLocal.UTC().Truncate(time.Microsecond)
	if !consulta.observado.Equal(esperado) || consulta.observado.Location() != time.UTC || consulta.observado.Nanosecond()%1000 != 0 {
		t.Fatalf("instante observado=%s; esperado=%s", consulta.observado, esperado)
	}
	resultado.Categorias[0].Etiqueta = "alterada"
	if consulta.resultado.Categorias[0].Etiqueta != "Administrativo" {
		t.Fatal("el servicio expuso el slice devuelto por el puerto")
	}
}

func TestServicioCategoriasProfesionalesRechazaNulosTipados(t *testing.T) {
	var consultaNula *consultaCategoriasDoble
	var relojNulo *relojCategoriasDoble
	valida := &consultaCategoriasDoble{resultado: catalogoCategoriasAplicacionPrueba()}
	reloj := &relojCategoriasDoble{instante: time.Now().UTC()}
	for nombre, construir := range map[string]func() error{
		"consulta": func() error {
			_, err := application.NuevoServicioConsultaCategoriasProfesionales(consultaNula, reloj)
			return err
		},
		"reloj": func() error {
			_, err := application.NuevoServicioConsultaCategoriasProfesionales(valida, relojNulo)
			return err
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			if err := construir(); !errors.Is(err, ports.ErrConsultaCategoriasProfesionalesInvalida) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestServicioCategoriasProfesionalesRespetaCancelacionPosteriorAlPuerto(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	consulta := &consultaCategoriasDoble{resultado: catalogoCategoriasAplicacionPrueba(), alLeer: cancelar}
	servicio, err := application.NuevoServicioConsultaCategoriasProfesionales(
		consulta, &relojCategoriasDoble{instante: time.Now().UTC()},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := servicio.ListarVigentes(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion no propagada: %v", err)
	}
}

func TestServicioCategoriasProfesionalesFallaCerradoAnteProyeccionInvalida(t *testing.T) {
	invalido := catalogoCategoriasAplicacionPrueba()
	invalido.Categorias[0].Area = ""
	servicio, err := application.NuevoServicioConsultaCategoriasProfesionales(
		&consultaCategoriasDoble{resultado: invalido},
		&relojCategoriasDoble{instante: time.Now().UTC()},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := servicio.ListarVigentes(context.Background()); !errors.Is(err, ports.ErrCatalogoCategoriasProfesionalesNoDisponible) {
		t.Fatalf("proyeccion invalida aceptada: %v", err)
	}
}
