package bootstrap

import (
	"context"
	"net/http"
	"testing"
)

type resolutorNuloSeguridadComunPrueba struct{}

func (*resolutorNuloSeguridadComunPrueba) ResolverContexto(context.Context) (contextoSeguridadComunDesarrollo, error) {
	return contextoSeguridadComunDesarrollo{}, nil
}

func TestSeguridadComunRechazaResolutorConPunteroNulo(t *testing.T) {
	var nulo *resolutorNuloSeguridadComunPrueba
	if _, err := nuevaSeguridadComunDesarrollo(nulo, relojContratacionTemporalDesarrollo{}); err == nil {
		t.Fatal("interfaz con puntero nulo admitida")
	}
}

func fronteraComunPrueba(clave, metodo, ruta string, detalle bool) descriptorFronteraComunDesarrollo {
	return descriptorFronteraComunDesarrollo{Clave: clave, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: metodo, Ruta: ruta, PerfilesActivosRef: []string{"prf_modulo"}, ClavePolitica: "politica_modulo", ClaveCapacidad: "capacidad_modulo", DetalleColeccion: detalle}
}

func TestCatalogoFronterasComunRechazaAmbiguedadYConservaMetodoExacto(t *testing.T) {
	base := fronteraComunPrueba("detalle", http.MethodGet, "/api/modulo", true)
	for _, caso := range [][]descriptorFronteraComunDesarrollo{
		{base, base},
		{base, fronteraComunPrueba("exacto", http.MethodGet, "/api/modulo/uno", false)},
		{{Clave: "sin-perfil", Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodGet, Ruta: "/api/modulo", ClavePolitica: "politica", ClaveCapacidad: "capacidad"}},
	} {
		if _, err := nuevoCatalogoFronterasComunDesarrollo(caso); err == nil {
			t.Fatal("catálogo ambiguo o inválido admitido")
		}
	}
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{base})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := catalogo.resolver(http.MethodHead, "/api/modulo/uno"); ok {
		t.Fatal("HEAD fue derivado de GET")
	}
	d, ok := catalogo.resolver(http.MethodGet, "/api/modulo/uno")
	if !ok || d.ClaveCapacidad != "capacidad_modulo" {
		t.Fatal("frontera o capacidad no acreditada")
	}
}

func TestCatalogoFronterasComunPermiteColeccionYDetalleMismaBase(t *testing.T) {
	coleccion := fronteraComunPrueba("coleccion", http.MethodGet, "/api/modulo", false)
	detalle := fronteraComunPrueba("detalle", http.MethodGet, "/api/modulo", true)
	if _, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{coleccion, detalle}); err != nil {
		t.Fatalf("colección y detalle son rutas distintas: %v", err)
	}
}

func TestCatalogoFronterasComunCopiaDescriptoresYAceptaModuloSintetico(t *testing.T) {
	d := fronteraComunPrueba("cronos", http.MethodPost, "/api/vec/cronos/partes", false)
	c, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{d})
	if err != nil {
		t.Fatal(err)
	}
	d.PerfilesActivosRef[0] = "prf_alterado"
	resuelto, ok := c.resolver(http.MethodPost, "/api/vec/cronos/partes")
	if !ok || !resuelto.admitePerfil("prf_modulo") || resuelto.admitePerfil("prf_alterado") {
		t.Fatal("el catálogo retuvo una referencia mutable")
	}
}

func TestCatalogoFronterasComunPerfilesSonFinitosYDefensivos(t *testing.T) {
	d := fronteraComunPrueba("perfiles", http.MethodPost, "/api/vec/modulo", false)
	d.PerfilesActivosRef = []string{"prf_uno", "prf_dos"}
	c, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{d})
	if err != nil {
		t.Fatal(err)
	}
	d.PerfilesActivosRef[0] = "prf_alterado"
	resuelto, ok := c.resolver(http.MethodPost, "/api/vec/modulo")
	if !ok || !resuelto.admitePerfil("prf_uno") || !resuelto.admitePerfil("prf_dos") || resuelto.admitePerfil("prf_alterado") {
		t.Fatal("conjunto de perfiles no quedó sellado")
	}
	for _, invalido := range [][]string{nil, {"prf_uno", "prf_uno"}} {
		d.PerfilesActivosRef = invalido
		if _, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{d}); err == nil {
			t.Fatal("conjunto de perfiles inválido admitido")
		}
	}
}

func TestCatalogoFronterasComunResolverNoExponePerfilesMutables(t *testing.T) {
	d := fronteraComunPrueba("sellado", http.MethodGet, "/api/vec/modulo/sellado", false)
	c, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{d})
	if err != nil {
		t.Fatal(err)
	}
	primero, ok := c.resolver(http.MethodGet, "/api/vec/modulo/sellado")
	if !ok {
		t.Fatal("frontera no resuelta")
	}
	primero.PerfilesActivosRef[0] = "prf_inyectado"
	segundo, ok := c.resolver(http.MethodGet, "/api/vec/modulo/sellado")
	if !ok || !segundo.admitePerfil("prf_modulo") || segundo.admitePerfil("prf_inyectado") {
		t.Fatal("resolver expuso perfiles mutables")
	}
}
