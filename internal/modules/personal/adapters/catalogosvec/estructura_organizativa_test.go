package catalogosvec

import (
	"context"
	"testing"
	"time"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type fuenteEstructuraSintetica struct {
	catalogo vecdomain.CatalogoConfigurable
}

func (f fuenteEstructuraSintetica) ObtenerCatalogo(context.Context, string, int) (vecdomain.CatalogoConfigurable, error) {
	return f.catalogo, nil
}
func (f fuenteEstructuraSintetica) ListarVersionesCatalogo(context.Context, string) ([]vecdomain.CatalogoConfigurable, error) {
	return []vecdomain.CatalogoConfigurable{f.catalogo}, nil
}
func TestConsultaEstructuraRechazaDependenciaNulaYVersion(t *testing.T) {
	if _, err := NuevaConsultaEstructuraOrganizativa(nil, "org", 1); err == nil {
		t.Fatal("nil aceptado")
	}
	if _, err := NuevaConsultaEstructuraOrganizativa(fuenteEstructuraSintetica{}, "org", 0); err == nil {
		t.Fatal("version aceptada")
	}
}
func TestConsultaEstructuraRespetaCancelacionAntesDeConsultar(t *testing.T) {
	c, err := NuevaConsultaEstructuraOrganizativa(fuenteEstructuraSintetica{}, "org", 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err = c.Obtener(ctx); err != context.Canceled {
		t.Fatalf("error=%v", err)
	}
}
func TestConsultaEstructuraRechazaCatalogoNoValido(t *testing.T) {
	c, err := NuevaConsultaEstructuraOrganizativa(fuenteEstructuraSintetica{}, "org", 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Obtener(context.Background()); err == nil {
		t.Fatal("catalogo invalido aceptado")
	}
}
func TestConsultaEstructuraNilReceiver(t *testing.T) {
	var c *ConsultaEstructuraOrganizativa
	if _, err := c.Obtener(context.Background()); err != ErrConsultaEstructuraOrganizativaInvalida {
		t.Fatalf("error=%v", err)
	}
}

func TestConsultaEstructuraProyectaRaizYVersionNueva(t *testing.T) {
	fecha := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	base := catalogoEstructura("estructura", 1, fecha, []vecdomain.EntradaCatalogoConfigurable{
		entradaEstructura("subdireccion", "Subdirección", "puesto_responsabilidad", "", "datosreserva"),
	})
	nueva := catalogoEstructura("estructura", 2, fecha, []vecdomain.EntradaCatalogoConfigurable{
		entradaEstructura("subdireccion", "Subdirección", "puesto_responsabilidad", "director", "datosreserva"),
		entradaEstructura("director", "Director", "puesto_responsabilidad", "", "datosreserva"),
	})
	nueva.VersionAnteriorRef = "estructura:1"
	nueva.Revision = 2
	nueva.UltimaModificacionPor = "tecnico-estructura-2"
	nueva.UltimaModificacionEn = fecha.Add(time.Hour)
	nueva.MotivoModificacion = "actualizacion"
	for _, tc := range []struct {
		cat  vecdomain.CatalogoConfigurable
		want int
	}{{base, 1}, {nueva, 2}} {
		if err := tc.cat.Validar(); err != nil {
			t.Fatalf("fixture v%d: %v", tc.cat.Version, err)
		}
		c, err := NuevaConsultaEstructuraOrganizativa(fuenteEstructuraSintetica{tc.cat}, "estructura", tc.cat.Version)
		if err != nil {
			t.Fatal(err)
		}
		got, err := c.Obtener(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Unidades) != tc.want || got.Descripcion != "Estructura pública" {
			t.Fatalf("resultado=%#v", got)
		}
	}
	if base.Entradas[0].Atributos["adscripcion_clave"] != "" || len(base.Entradas) != 1 {
		t.Fatal("la nueva versión modifica la estructura anterior")
	}
	huellaBase, err := base.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huellaNueva, err := nueva.HuellaSHA256()
	if err != nil || huellaBase == huellaNueva {
		t.Fatal("las dos versiones deben conservar huellas distintas")
	}
}

func TestConsultaEstructuraRechazaCicloOPadreAusente(t *testing.T) {
	fecha := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, entradas := range [][]vecdomain.EntradaCatalogoConfigurable{
		{entradaEstructura("a", "A", "centro", "b", ""), entradaEstructura("b", "B", "centro", "a", "")},
		{entradaEstructura("a", "A", "centro", "no-existe", "")},
	} {
		c, _ := NuevaConsultaEstructuraOrganizativa(fuenteEstructuraSintetica{catalogoEstructura("estructura", 1, fecha, entradas)}, "estructura", 1)
		if _, err := c.Obtener(context.Background()); err == nil {
			t.Fatal("estructura invalida aceptada")
		}
	}
}

func entradaEstructura(clave, etiqueta, tipo, padre, codigo string) vecdomain.EntradaCatalogoConfigurable {
	a := map[string]string{"tipo": tipo}
	if padre != "" {
		a["adscripcion_clave"] = padre
	}
	if codigo != "" {
		a["codigo_fuente"] = codigo
	}
	return vecdomain.EntradaCatalogoConfigurable{Clave: clave, Etiqueta: etiqueta, Orden: 1, VigenteDesde: time.Unix(0, 0).UTC(), Atributos: a}
}
func catalogoEstructura(id string, version int, fecha time.Time, entradas []vecdomain.EntradaCatalogoConfigurable) vecdomain.CatalogoConfigurable {
	for i := range entradas {
		entradas[i].VigenteDesde = fecha
	}
	return vecdomain.CatalogoConfigurable{ID: id, Version: version, Revision: 1, ModuloID: "personal", Nombre: "Estructura", Descripcion: "Estructura pública", FuenteRef: "datosreserva", MotivoCreacion: "fixture", Entradas: entradas, Estado: vecdomain.EstadoCatalogoBorrador, CreadoPor: "tecnico-estructura-1", CreadoEn: fecha}
}
