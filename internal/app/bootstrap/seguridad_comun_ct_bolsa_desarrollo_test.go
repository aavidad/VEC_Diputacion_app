package bootstrap

import (
	"context"
	"net/http"
	"strings"
	"testing"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
)

type resolutorNuloSeguridadComunPrueba struct{}

func (*resolutorNuloSeguridadComunPrueba) ResolverContexto(context.Context) (contextoSeguridadComunDesarrollo, error) {
	return contextoSeguridadComunDesarrollo{}, nil
}

func TestFronteraJustificanteSoloAdmiteAcuseNominalDeLaMismaPeticion(t *testing.T) {
	d, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{
		fronteraJustificanteBolsaRRHHDesarrollo([]string{"prf_cuadro"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	f := fronteraSeguridadComunDesarrollo{metodo: http.MethodGet,
		ruta: bolsahttp.RutaJustificanteRenunciaRRHH, catalogo: d,
		descriptor: d.porClave["bolsa-justificante-renuncia-rrhh"],
		holder:     &contextoOperacionCTDesarrollo{}}
	z := contextoSeguridadComunDesarrollo{}
	z.Resultado.Contexto.PerfilActivoRef = "prf_cuadro"
	if perfilJustificanteAdmitidoPorFrontera(f, z) {
		t.Fatal("perfil del cuadro sin acuse")
	}
	z.seleccionJustificanteRRHH = &seleccionNominalJustificanteRRHH{
		perfil: "prf_otro", auditoria: "aud_v3_b_" + strings.Repeat("a", 32), ruta: f.ruta,
		metodo: f.metodo, catalogo: d.identidad, holder: f.holder,
	}
	if perfilJustificanteAdmitidoPorFrontera(f, z) {
		t.Fatal("perfil ajeno admitido")
	}
	z.seleccionJustificanteRRHH.perfil = "prf_cuadro"
	if !perfilJustificanteAdmitidoPorFrontera(f, z) {
		t.Fatal("acuse nominal propio rechazado")
	}
	f.holder = &contextoOperacionCTDesarrollo{}
	if perfilJustificanteAdmitidoPorFrontera(f, z) {
		t.Fatal("acuse de otra petición admitido")
	}
	f.holder = z.seleccionJustificanteRRHH.holder
	f.metodo = http.MethodPost
	if perfilJustificanteAdmitidoPorFrontera(f, z) {
		t.Fatal("acuse GET admitido para POST")
	}
	f.metodo = http.MethodGet
	f.catalogo.identidad = &identidadCatalogoFronterasComunDesarrollo{}
	if perfilJustificanteAdmitidoPorFrontera(f, z) {
		t.Fatal("acuse de otro catálogo admitido")
	}
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

func TestCatalogoFronterasComunPermitePlantillasDisjuntasMismaBase(t *testing.T) {
	a := fronteraComunPrueba("lista", http.MethodGet, "/api/modulo", false)
	a.PlantillaDetalle = []string{"*", "candidatos"}
	b := fronteraComunPrueba("contactos", http.MethodGet, "/api/modulo", false)
	b.PlantillaDetalle = []string{"*", "candidatos", "*", "contactos"}
	c, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{a, b})
	if err != nil {
		t.Fatal(err)
	}
	if d, ok := c.resolver(http.MethodGet, "/api/modulo/bolsa/candidatos"); !ok || d.Clave != "lista" {
		t.Fatal("lista no resuelta")
	}
	if d, ok := c.resolver(http.MethodGet, "/api/modulo/bolsa/candidatos/persona/contactos"); !ok || d.Clave != "contactos" {
		t.Fatal("contactos no resueltos")
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
