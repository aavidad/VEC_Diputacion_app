package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	ficherobolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/fichero"
	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type relojPublicoFijo struct{ instante time.Time }

func (r relojPublicoFijo) Ahora() time.Time { return r.instante }

func servicioPublicoPrueba(t *testing.T, instante time.Time) *aplicacionbolsa.ServicioConsultaPublica {
	t.Helper()
	adaptador, err := ficherobolsa.NuevaConsultaConvocatorias("../../../../data/demo/convocatorias_publicas.demo.json")
	if err != nil {
		t.Fatal(err)
	}
	servicio, err := aplicacionbolsa.NuevoServicioConsultaPublica(adaptador, relojPublicoFijo{instante: instante})
	if err != nil {
		t.Fatal(err)
	}
	return servicio
}

func TestListadoPublicoMinimizaDatosYResuelveCatalogos(t *testing.T) {
	servicio := servicioPublicoPrueba(t, time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC))
	resultado, err := servicio.Listar(context.Background(), aplicacionbolsa.SolicitudListadoPublico{Tamano: 12})
	if err != nil || resultado.Paginacion.Total != 2 || len(resultado.Facetas.Tipos) != 2 {
		t.Fatalf("resultado = %#v, error = %v", resultado, err)
	}
	contenido, err := json.Marshal(resultado)
	if err != nil {
		t.Fatal(err)
	}
	texto := string(contenido)
	for _, prohibido := range []string{`"proceso_ref"`, "Titulación indicada en las bases", `"documentos":`, `"requisitos":`, `"dni":`, `"correo":`} {
		if strings.Contains(texto, prohibido) {
			t.Fatalf("la proyección de listado contiene %q", prohibido)
		}
	}
}

func TestConsultaPublicaAcotaFiltrosYPaginacion(t *testing.T) {
	servicio := servicioPublicoPrueba(t, time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC))
	pruebas := []struct {
		nombre    string
		solicitud aplicacionbolsa.SolicitudListadoPublico
	}{
		{nombre: "tamaño excesivo", solicitud: aplicacionbolsa.SolicitudListadoPublico{Tamano: 25}},
		{nombre: "página excesiva", solicitud: aplicacionbolsa.SolicitudListadoPublico{Pagina: 501}},
		{nombre: "texto excesivo", solicitud: aplicacionbolsa.SolicitudListadoPublico{Texto: strings.Repeat("a", 101)}},
		{nombre: "tipo no canónico", solicitud: aplicacionbolsa.SolicitudListadoPublico{Tipo: " tipo"}},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			_, err := servicio.Listar(context.Background(), prueba.solicitud)
			if !errors.Is(err, aplicacionbolsa.ErrFiltroPublicoInvalido) {
				t.Fatalf("error = %v para %#v", err, prueba.solicitud)
			}
		})
	}
}

func TestDetallePublicoCierreEsInclusivoAlMicrosegundo(t *testing.T) {
	cierre := time.Date(2026, 8, 15, 21, 59, 59, 0, time.UTC)
	for _, prueba := range []struct {
		instante time.Time
		quiere   string
	}{
		{instante: cierre, quiere: "abierto"},
		{instante: cierre.Add(time.Microsecond), quiere: "cerrado"},
	} {
		servicio := servicioPublicoPrueba(t, prueba.instante)
		detalle, err := servicio.Obtener(context.Background(), "bolsa-auxiliar-administrativo-demo-2026")
		if err != nil || detalle.Plazos[0].Situacion != prueba.quiere {
			t.Fatalf("situacion = %q, error = %v", detalle.Plazos[0].Situacion, err)
		}
	}
}

func TestObtenerRespetaContextoCanceladoAntesDeFuente(t *testing.T) {
	servicio := servicioPublicoPrueba(t, time.Now().UTC())
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := servicio.Obtener(ctx, "bolsa-auxiliar-administrativo-demo-2026"); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
}

type fuenteDocumentoNoPublicable struct {
	base puertosbolsa.ConsultaConvocatoriasPublicas
}

func (f fuenteDocumentoNoPublicable) BuscarPublicadas(ctx context.Context, filtro puertosbolsa.FiltroConvocatoriasPublicas) (puertosbolsa.PaginaConvocatorias, error) {
	return f.base.BuscarPublicadas(ctx, filtro)
}

func (f fuenteDocumentoNoPublicable) ObtenerPublicada(ctx context.Context, id string) (puertosbolsa.DetalleConvocatoria, error) {
	detalle, err := f.base.ObtenerPublicada(ctx, id)
	if err != nil {
		return detalle, err
	}
	for i := range detalle.Catalogos {
		if detalle.Catalogos[i].Referencia != puertosbolsa.CatalogoTiposDocumento {
			continue
		}
		for j := range detalle.Catalogos[i].Entradas {
			detalle.Catalogos[i].Entradas[j].Publicable = false
		}
	}
	return detalle, nil
}

func TestDetalleFallaCerradoSiDocumentoPierdeCatalogoPublicable(t *testing.T) {
	base, err := ficherobolsa.NuevaConsultaConvocatorias("../../../../data/demo/convocatorias_publicas.demo.json")
	if err != nil {
		t.Fatal(err)
	}
	servicio, err := aplicacionbolsa.NuevoServicioConsultaPublica(
		fuenteDocumentoNoPublicable{base: base},
		relojPublicoFijo{instante: time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = servicio.Obtener(context.Background(), "bolsa-auxiliar-administrativo-demo-2026")
	if !errors.Is(err, aplicacionbolsa.ErrDatosPublicosNoConfiables) {
		t.Fatalf("error = %v", err)
	}
}
