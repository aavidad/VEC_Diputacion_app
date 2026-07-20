package catalogosvec_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	catalogospersonal "vec-diputacion-granada/internal/modules/personal/adapters/catalogosvec"
	puertospersonal "vec-diputacion-granada/internal/modules/personal/ports"
	ficherovec "vec-diputacion-granada/internal/vec/adapters/fichero"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	rutaCatalogoProfesionalDemo = "../../../../../data/catalogos/categorias-profesionales/v1.demo.json"
	huellaCatalogoProfesionalV1 = "b800a7e9c306fa8027709cfb4304cc8ccf8065f888673da71bd73a138c519233"
)

func referenciaCatalogoProfesionalV1() puertospersonal.ReferenciaCatalogoCategoriasProfesionales {
	return puertospersonal.ReferenciaCatalogoCategoriasProfesionales{
		CatalogoID:           "categorias-profesionales",
		CatalogoVersion:      1,
		CatalogoHuellaSHA256: huellaCatalogoProfesionalV1,
	}
}

func instanteCatalogoProfesionalPrueba() time.Time {
	return time.Date(2026, 7, 16, 8, 0, 0, 0, time.UTC)
}

func fuenteCatalogoProfesionalDemo(t *testing.T) *ficherovec.ConsultaCatalogos {
	t.Helper()
	fuente, err := ficherovec.NuevaConsultaCatalogos(rutaCatalogoProfesionalDemo)
	if err != nil {
		t.Fatal(err)
	}
	return fuente
}

func TestConsultaPersonalProyectaLas68CategoriasConHuellaExactaYMinimizada(t *testing.T) {
	consulta, err := catalogospersonal.NuevaConsultaCategoriasProfesionales(
		fuenteCatalogoProfesionalDemo(t), referenciaCatalogoProfesionalV1(),
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := consulta.ObtenerVigentes(
		context.Background(), instanteCatalogoProfesionalPrueba(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if resultado.Referencia != referenciaCatalogoProfesionalV1() || len(resultado.Categorias) != 68 ||
		!resultado.Fuente.Demostracion || resultado.Fuente.Revision != "opes-bop-inventario-publico-demo-v3" ||
		!strings.Contains(resultado.Fuente.Aviso, "pendiente de validación") {
		t.Fatalf("referencia=%#v fuente=%#v categorias=%d", resultado.Referencia, resultado.Fuente, len(resultado.Categorias))
	}
	areas := map[string]int{}
	for indice, categoria := range resultado.Categorias {
		if categoria.Orden != indice+1 {
			t.Fatalf("orden[%d]=%d", indice, categoria.Orden)
		}
		areas[categoria.Area]++
	}
	if areas["administracion_general"] != 5 || areas["administracion_especial"] != 60 ||
		areas["organismos_dependientes"] != 3 || len(areas) != 3 {
		t.Fatalf("areas=%#v", areas)
	}
	if resultado.Categorias[3].Etiqueta != "Técnico de Administración General" ||
		resultado.Categorias[38].Etiqueta != "Médico" ||
		resultado.Categorias[46].Etiqueta != "Psicólogo" {
		t.Fatalf("etiquetas con tildes alteradas: %#v", resultado.Categorias)
	}
	contenido, err := json.Marshal(resultado)
	if err != nil {
		t.Fatal(err)
	}
	for _, prohibido := range []string{"source_path", "creado_por", "publicado_por", "aprobacion_ref", "motivo_publicacion", "sistema:publicador"} {
		if strings.Contains(string(contenido), prohibido) {
			t.Fatalf("la proyeccion expone %q: %s", prohibido, contenido)
		}
	}

	resultado.Categorias[0].Etiqueta = "alterada"
	otra, err := consulta.ObtenerVigentes(context.Background(), instanteCatalogoProfesionalPrueba())
	if err != nil || otra.Categorias[0].Etiqueta != "Administrativo" {
		t.Fatalf("la instantanea comparte memoria: %#v, %v", otra.Categorias[0], err)
	}
}

type fuenteCatalogosDoble struct {
	catalogo dominiovec.CatalogoConfigurable
	error    error
	alLeer   func()
}

func (f *fuenteCatalogosDoble) ObtenerCatalogo(context.Context, string, int) (dominiovec.CatalogoConfigurable, error) {
	if f.alLeer != nil {
		f.alLeer()
	}
	return f.catalogo, f.error
}

func (f *fuenteCatalogosDoble) ListarVersionesCatalogo(context.Context, string) ([]dominiovec.CatalogoConfigurable, error) {
	return []dominiovec.CatalogoConfigurable{f.catalogo}, f.error
}

func (f *fuenteCatalogosDoble) ObtenerMetadatosFuenteCatalogos(context.Context) (puertosvec.MetadatosFuenteCatalogos, error) {
	return puertosvec.MetadatosFuenteCatalogos{
		Revision: "fuente-prueba-v1", ActualizadaEn: time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC),
		Demostracion: true, Aviso: "DEMOSTRACIÓN: fuente sintética de prueba.",
	}, f.error
}

func catalogoProfesionalPublicado(t *testing.T) dominiovec.CatalogoConfigurable {
	t.Helper()
	catalogo, err := fuenteCatalogoProfesionalDemo(t).ObtenerCatalogo(context.Background(), "categorias-profesionales", 1)
	if err != nil {
		t.Fatal(err)
	}
	return catalogo
}

func referenciaParaCatalogo(t *testing.T, catalogo dominiovec.CatalogoConfigurable) puertospersonal.ReferenciaCatalogoCategoriasProfesionales {
	t.Helper()
	huella, err := catalogo.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	return puertospersonal.ReferenciaCatalogoCategoriasProfesionales{
		CatalogoID: catalogo.ID, CatalogoVersion: catalogo.Version, CatalogoHuellaSHA256: huella,
	}
}

func TestConsultaPersonalRechazaNuloTipadoYHuellaDistinta(t *testing.T) {
	var fuenteNula *fuenteCatalogosDoble
	if _, err := catalogospersonal.NuevaConsultaCategoriasProfesionales(fuenteNula, referenciaCatalogoProfesionalV1()); !errors.Is(err, puertospersonal.ErrConsultaCategoriasProfesionalesInvalida) {
		t.Fatalf("nulo tipado aceptado: %v", err)
	}
	consultaConFallo, err := catalogospersonal.NuevaConsultaCategoriasProfesionales(
		&fuenteCatalogosDoble{error: errors.New("fuente no disponible")}, referenciaCatalogoProfesionalV1(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := consultaConFallo.ObtenerVigentes(context.Background(), instanteCatalogoProfesionalPrueba()); !errors.Is(err, puertospersonal.ErrCatalogoCategoriasProfesionalesNoDisponible) {
		t.Fatalf("fallo de fuente no se cerro: %v", err)
	}

	referencia := referenciaCatalogoProfesionalV1()
	referencia.CatalogoHuellaSHA256 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	consulta, err := catalogospersonal.NuevaConsultaCategoriasProfesionales(fuenteCatalogoProfesionalDemo(t), referencia)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := consulta.ObtenerVigentes(context.Background(), instanteCatalogoProfesionalPrueba()); !errors.Is(err, puertospersonal.ErrCatalogoCategoriasProfesionalesNoDisponible) {
		t.Fatalf("huella distinta aceptada: %v", err)
	}
}

func TestConsultaPersonalRespetaCancelacionAntesYDespuesDeLeer(t *testing.T) {
	consulta, err := catalogospersonal.NuevaConsultaCategoriasProfesionales(
		fuenteCatalogoProfesionalDemo(t), referenciaCatalogoProfesionalV1(),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := consulta.ObtenerVigentes(ctx, instanteCatalogoProfesionalPrueba()); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion previa no propagada: %v", err)
	}

	ctx, cancelar = context.WithCancel(context.Background())
	catalogo := catalogoProfesionalPublicado(t)
	doble := &fuenteCatalogosDoble{catalogo: catalogo, alLeer: cancelar}
	consulta, err = catalogospersonal.NuevaConsultaCategoriasProfesionales(doble, referenciaParaCatalogo(t, catalogo))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := consulta.ObtenerVigentes(ctx, instanteCatalogoProfesionalPrueba()); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion posterior no propagada: %v", err)
	}
}

func TestConsultaPersonalFallaCerradoAnteCatalogoNoPublicadoOEntradaInvalida(t *testing.T) {
	publicado := catalogoProfesionalPublicado(t)
	borrador := publicado
	borrador.Estado = dominiovec.EstadoCatalogoBorrador
	borrador.PublicadoPor = ""
	borrador.PublicadoEn = time.Time{}
	borrador.AprobacionRef = ""
	borrador.MotivoPublicacion = ""

	atributoInvalido := publicado
	atributoInvalido.Entradas = append([]dominiovec.EntradaCatalogoConfigurable(nil), publicado.Entradas...)
	atributoInvalido.Entradas[0].Atributos = map[string]string{}
	for clave, valor := range publicado.Entradas[0].Atributos {
		atributoInvalido.Entradas[0].Atributos[clave] = valor
	}
	atributoInvalido.Entradas[0].Atributos["area"] = "Administracion general"

	for nombre, catalogo := range map[string]dominiovec.CatalogoConfigurable{
		"borrador":          borrador,
		"atributo invalido": atributoInvalido,
	} {
		t.Run(nombre, func(t *testing.T) {
			consulta, err := catalogospersonal.NuevaConsultaCategoriasProfesionales(
				&fuenteCatalogosDoble{catalogo: catalogo}, referenciaParaCatalogo(t, catalogo),
			)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := consulta.ObtenerVigentes(context.Background(), instanteCatalogoProfesionalPrueba()); !errors.Is(err, puertospersonal.ErrCatalogoCategoriasProfesionalesNoDisponible) {
				t.Fatalf("catalogo invalido aceptado: %v", err)
			}
		})
	}
}
