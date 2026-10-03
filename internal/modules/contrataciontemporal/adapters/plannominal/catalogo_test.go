package plannominal

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

func catalogoPrueba() (vd.CatalogoConfigurable, time.Time) {
	en := time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)
	c := vd.CatalogoConfigurable{ID: "ct.plan.prueba", Version: 1, Revision: 1, ModuloID: "contratacion_temporal", Nombre: "Prueba", FuenteRef: "fuente:prueba", MotivoCreacion: "Prueba", Estado: vd.EstadoCatalogoPublicado, CreadoPor: "actor:prueba", CreadoEn: en, PublicadoPor: "revisor:prueba", PublicadoEn: en, AprobacionRef: "acto:prueba", MotivoPublicacion: "Prueba"}
	c.Entradas = []vd.EntradaCatalogoConfigurable{{Clave: "resolucion.p1", Etiqueta: "Prueba", Orden: 1, VigenteDesde: en, Atributos: map[string]string{"esquema": ct.EsquemaPlanCompetenciaFirmaV2, "circuito_ref": "ct.circuito.prueba", "circuito_version": "1", "circuito_sha256": strings.Repeat("a", 64), "documento": "resolucion", "paso_ref": "paso:uno", "paso_orden": "1", "perfil_esperado_ref": "perfil:prueba", "rol_id": "rol.central", "cargo_ref": "cargo:prueba", "organizacion_ref": "org:prueba", "unidad_ref": "unidad:prueba", "accion_competencial": "documento.firmar", "finalidad": "formalizar", "tipo_recurso": "documento", "esquema_contexto": "contexto.v1", "mapeo_version": "1", "mapeo_fuente_ref": "fuente:prueba"}}}
	return c, en
}

func TestCatalogoPlanFirmaV2VersionYProcedencia(t *testing.T) {
	c, en := catalogoPrueba()
	p, err := DesdeCatalogo(c, en)
	sha, _ := c.HuellaSHA256()
	if err != nil || p.Version.HuellaSHA256 != sha || p.Version.Version != 1 || p.Pasos[0].MapeoFuenteRef != c.FuenteRef {
		t.Fatalf("plan %v: %v", p, err)
	}
	for _, campo := range []string{"rol_id", "cargo_ref", "mapeo_fuente_ref", "accion_competencial", "finalidad", "tipo_recurso", "esquema_contexto"} {
		c, en := catalogoPrueba()
		delete(c.Entradas[0].Atributos, campo)
		if _, err := DesdeCatalogo(c, en); err == nil {
			t.Fatalf("acepto campo ausente %s", campo)
		}
	}
	for _, campo := range []string{"circuito_version", "paso_orden", "mapeo_version"} {
		c, en := catalogoPrueba()
		c.Entradas[0].Atributos[campo] = "01"
		if _, err := DesdeCatalogo(c, en); err == nil {
			t.Fatalf("acepto numero no canonico %s", campo)
		}
	}
}

func TestCatalogoDemoCircuitoNoEsPlanNominal(t *testing.T) {
	b, err := os.ReadFile("../../../../../data/demo/reglas/ct_circuito_firma.rrhh.v2.json")
	if err != nil {
		t.Fatal(err)
	}
	var sobre struct {
		Catalogo vd.CatalogoConfigurable `json:"catalogo"`
	}
	if err = json.Unmarshal(b, &sobre); err != nil {
		t.Fatal(err)
	}
	if _, err = DesdeCatalogo(sobre.Catalogo, time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)); err == nil {
		t.Fatal("DEMO aceptada como plan nominal")
	}
}

type publicacionNula struct{}

func (*publicacionNula) ComprobarPublicacionPlanFirmaV2(context.Context, vd.CatalogoConfigurable, string, time.Time) error {
	return nil
}

func TestFuentePlanFirmaV2CerradaSinPublicacion(t *testing.T) {
	var proveedor *publicacionNula
	if f, err := NuevaFuente(nil, proveedor, ct.VersionPlanFirmaV2{}); err == nil || f != nil {
		t.Fatal("acepto fuente sin autoridad")
	}
	var f *Fuente
	if _, err := f.Plan(context.Background()); err == nil {
		t.Fatal("fuente nula abierta")
	}
}

type consultaPlanPrueba struct {
	catalogo vd.CatalogoConfigurable
	ahora    time.Time
}

func (c *consultaPlanPrueba) Ahora() time.Time { return c.ahora }
func (c *consultaPlanPrueba) ListarVersionesCatalogoAcotado(context.Context, string, vp.LimitesConsultaCatalogosAcotada) (vp.ResultadoConsultaCatalogosAcotada, error) {
	return vp.ResultadoConsultaCatalogosAcotada{Catalogos: []vd.CatalogoConfigurable{c.catalogo}}, nil
}
func (c *consultaPlanPrueba) ObtenerCatalogoAcotado(context.Context, string, int, vp.LimitesConsultaCatalogosAcotada) (vp.ResultadoConsultaCatalogoAcotado, error) {
	return vp.ResultadoConsultaCatalogoAcotado{Catalogo: c.catalogo}, nil
}

type publicacionPrueba struct {
	err      error
	llamadas int
}

func (p *publicacionPrueba) ComprobarPublicacionPlanFirmaV2(context.Context, vd.CatalogoConfigurable, string, time.Time) error {
	p.llamadas++
	return p.err
}

func TestFuenteRevalidaPublicacionYNoAceptaVersionCambiante(t *testing.T) {
	c, en := catalogoPrueba()
	consulta := &consultaPlanPrueba{c, en}
	r, err := reglas.NuevoResolutor(reglas.Configuracion{Consulta: consulta, CatalogoID: c.ID, ModuloID: c.ModuloID, Reloj: consulta})
	if err != nil {
		t.Fatal(err)
	}
	sha, _ := c.HuellaSHA256()
	version := ct.VersionPlanFirmaV2{Referencia: c.ID, Version: 1, HuellaSHA256: sha}
	pub := &publicacionPrueba{}
	f, err := NuevaFuente(r, pub, version)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.Plan(context.Background()); err != nil {
		t.Fatal(err)
	}
	pub.err = errors.New("publicacion no acreditada")
	if _, err = f.Plan(context.Background()); err == nil || pub.llamadas != 2 {
		t.Fatal("reutilizo autoridad positiva anterior")
	}
	pub.err = nil
	consulta.catalogo.Entradas[0].Atributos["rol_id"] = "otro.rol"
	if _, err = f.Plan(context.Background()); err == nil || pub.llamadas != 2 {
		t.Fatal("acepto otros bytes bajo la misma version")
	}
	consulta.catalogo = c
	if f, err = NuevaFuente(r, (*publicacionPrueba)(nil), version); err == nil || f != nil {
		t.Fatal("acepto proveedor nulo tipado")
	}
}
