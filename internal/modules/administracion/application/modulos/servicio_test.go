package modulos

import (
	"context"
	"errors"
	"testing"
	"time"

	admin "vec-diputacion-granada/internal/modules/administracion"
	domain "vec-diputacion-granada/internal/modules/administracion/domain/modulos"
	vec "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type fuente struct {
	versiones []vec.CatalogoConfigurable
	err       error
	truncado  bool
	lecturas  int
}

func (f *fuente) ObtenerCatalogoAcotado(context.Context, string, int, vp.LimitesConsultaCatalogosAcotada) (vp.ResultadoConsultaCatalogoAcotado, error) {
	return vp.ResultadoConsultaCatalogoAcotado{}, errors.New("no se usa")
}
func (f *fuente) ListarVersionesCatalogoAcotado(context.Context, string, vp.LimitesConsultaCatalogosAcotada) (vp.ResultadoConsultaCatalogosAcotada, error) {
	f.lecturas++
	return vp.ResultadoConsultaCatalogosAcotada{Catalogos: f.versiones, Truncado: f.truncado}, f.err
}

type registro []vec.ModuleManifest

func (r registro) ListModules(context.Context) ([]vec.ModuleManifest, error) { return r, nil }

type reloj time.Time

func (r reloj) Ahora() time.Time { return time.Time(r) }

func fixture(t *testing.T) (*Servicio, *fuente, time.Time) {
	t.Helper()
	ahora := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	c := vec.CatalogoConfigurable{ID: "administracion.modulos", Version: 1, Revision: 1, ModuloID: admin.ModuleID, Nombre: "Módulos", FuenteRef: "fuente:operacion", MotivoCreacion: "Preparación", Estado: vec.EstadoCatalogoBorrador, CreadoPor: "actor:administracion", CreadoEn: ahora.Add(-time.Hour), Entradas: []vec.EntradaCatalogoConfigurable{{Clave: "vec.module.cronos", Etiqueta: "Control horario", VigenteDesde: ahora.Add(-time.Hour), Atributos: map[string]string{"habilitado": "true"}}}}
	publicado, err := c.Publicar("actor:revision", "acto:operacion", "Publicación", ahora.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	f := &fuente{versiones: []vec.CatalogoConfigurable{publicado}}
	r := registro{{ID: "vec.module.cronos", NameKey: "ui.vec.module.cronos.name", Permissions: []vec.Permission{{Key: "cronos.fichajes.leer", LabelKey: "ui.permission.cronos.read"}}}, admin.Manifest()}
	s, err := Nuevo(domain.Configuracion{CatalogoID: c.ID, Gobernados: []string{"vec.module.cronos"}}, f, r, reloj(ahora))
	if err != nil {
		t.Fatal(err)
	}
	return s, f, ahora
}

func TestControlReleeSinReinicioYNoReviveVersionRetirada(t *testing.T) {
	s, f, ahora := fixture(t)
	if err := s.ExigirHabilitado(context.Background(), "vec.module.cronos"); err != nil {
		t.Fatal(err)
	}
	siguiente, err := f.versiones[0].NuevaVersion(2, "actor:administracion", "fuente:operacion", "Mantenimiento", ahora.Add(-time.Second))
	if err != nil {
		t.Fatal(err)
	}
	siguiente.Entradas[0].Atributos["habilitado"] = "false"
	// Un borrador no es decisión activa; conserva la publicación vigente.
	f.versiones = append(f.versiones, siguiente)
	if err := s.ExigirHabilitado(context.Background(), "vec.module.cronos"); err != nil {
		t.Fatal(err)
	}
	publicado, err := siguiente.Publicar("actor:revision", "acto:operacion", "Mantenimiento", ahora)
	if err != nil {
		t.Fatal(err)
	}
	f.versiones[1] = publicado
	if err := s.ExigirHabilitado(context.Background(), "vec.module.cronos"); !errors.Is(err, domain.ErrDesactivado) {
		t.Fatal(err)
	}
	retirado, err := publicado.Retirar("actor:administracion", "acto:retirada", "Retirada", ahora)
	if err != nil {
		t.Fatal(err)
	}
	f.versiones[1] = retirado
	if err := s.ExigirHabilitado(context.Background(), "vec.module.cronos"); !errors.Is(err, domain.ErrCatalogo) {
		t.Fatal(err)
	}
	if f.lecturas != 4 {
		t.Fatal(f.lecturas)
	}
}

func TestFalloFuenteYCancelacionCierran(t *testing.T) {
	s, f, _ := fixture(t)
	f.err = errors.New("fuente caida")
	if err := s.ExigirHabilitado(context.Background(), "vec.module.cronos"); !errors.Is(err, domain.ErrCatalogo) {
		t.Fatal(err)
	}
	f.err = nil
	f.truncado = true
	if err := s.ExigirHabilitado(context.Background(), "vec.module.cronos"); !errors.Is(err, domain.ErrCatalogo) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.ExigirHabilitado(ctx, "vec.module.cronos"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := s.ExigirHabilitado(context.Background(), admin.ModuleID); !errors.Is(err, domain.ErrConfiguracion) {
		t.Fatal(err)
	}
	var vacia *fuente
	if _, err := Nuevo(s.cfg, vacia, s.registro, s.reloj); !errors.Is(err, domain.ErrConfiguracion) {
		t.Fatal(err)
	}
}
