package modulos

import (
	"errors"
	"testing"
	"time"

	admin "vec-diputacion-granada/internal/modules/administracion"
	vec "vec-diputacion-granada/internal/vec/domain"
)

func datos(t *testing.T) (Configuracion, vec.CatalogoConfigurable, []vec.ModuleManifest, time.Time) {
	t.Helper()
	ahora := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	cfg := Configuracion{CatalogoID: "administracion.modulos", Gobernados: []string{"vec.module.cronos"}}
	c := vec.CatalogoConfigurable{ID: cfg.CatalogoID, Version: 1, Revision: 1, ModuloID: admin.ModuleID, Nombre: "Módulos", FuenteRef: "fuente:operacion", MotivoCreacion: "Preparación", Estado: vec.EstadoCatalogoBorrador, CreadoPor: "actor:administracion", CreadoEn: ahora.Add(-time.Hour), Entradas: []vec.EntradaCatalogoConfigurable{{Clave: "vec.module.cronos", Etiqueta: "Control horario", VigenteDesde: ahora.Add(-time.Hour), Atributos: map[string]string{"habilitado": "true"}}}}
	publicado, err := c.Publicar("actor:revision", "acto:operacion", "Publicación", ahora.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	cronos := vec.ModuleManifest{ID: "vec.module.cronos", NameKey: "ui.vec.module.cronos.name", Permissions: []vec.Permission{{Key: "cronos.fichajes.leer", LabelKey: "ui.permission.cronos.read"}}}
	return cfg, publicado, []vec.ModuleManifest{cronos, admin.Manifest()}, ahora
}

func TestProyeccionRestringeSinApagarADMIN(t *testing.T) {
	cfg, c, registro, ahora := datos(t)
	e, err := Proyectar(cfg, c, registro, ahora)
	if err != nil {
		t.Fatal(err)
	}
	if e.ExigirHabilitado("vec.module.cronos") != nil || e.ExigirHabilitado(admin.ModuleID) != nil || e.EscrituraDisponible {
		t.Fatal(e)
	}
	c.Entradas[0].Atributos["habilitado"] = "false"
	e, err = Proyectar(cfg, c, registro, ahora)
	if err != nil {
		t.Fatal(err)
	}
	if !errors.Is(e.ExigirHabilitado("vec.module.cronos"), ErrDesactivado) || e.ExigirHabilitado(admin.ModuleID) != nil {
		t.Fatal(e)
	}
	c.Entradas = nil // Un catálogo publicado vacío es inválido.
	if _, err = Proyectar(cfg, c, registro, ahora); err == nil {
		t.Fatal("catalogo vacio aceptado")
	}
}

func TestCatalogoNoAmpliaCapacidadCompuesta(t *testing.T) {
	for _, caso := range []string{"ajeno", "administracion", "valor", "propietario", "futuro", "retirado", "vigencia"} {
		t.Run(caso, func(t *testing.T) {
			cfg, c, registro, ahora := datos(t)
			switch caso {
			case "ajeno":
				c.Entradas[0].Clave = "vec.module.inexistente"
			case "administracion":
				cfg.Gobernados = append(cfg.Gobernados, admin.ModuleID)
			case "valor":
				c.Entradas[0].Atributos["habilitado"] = "TRUE"
			case "propietario":
				c.ModuloID = "vec.module.cronos"
			case "futuro":
				c.PublicadoEn = ahora.Add(time.Minute)
			case "retirado":
				var err error
				c, err = c.Retirar("actor:administracion", "acto:retirada", "Retirada", ahora)
				if err != nil {
					t.Fatal(err)
				}
			case "vigencia":
				c.Entradas[0].VigenteHasta = ahora
			}
			e, err := Proyectar(cfg, c, registro, ahora)
			if caso == "vigencia" {
				if err != nil || !errors.Is(e.ExigirHabilitado("vec.module.cronos"), ErrDesactivado) {
					t.Fatalf("%+v %v", e, err)
				}
			} else if err == nil {
				t.Fatal("configuracion no segura aceptada")
			}
		})
	}
}

func TestPrepararExigePreimagenSinModificarPublicacion(t *testing.T) {
	cfg, c, registro, ahora := datos(t)
	huella, err := c.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	cambio := Cambio{ModuloID: "vec.module.cronos", Habilitado: false, VersionEsperada: 1, HuellaEsperada: huella, Motivo: "Mantenimiento programado\nVentana aprobada"}
	p, err := Preparar(cfg, c, registro, ahora, cambio)
	if err != nil {
		t.Fatal(err)
	}
	if p.Publicado || p.VersionSiguiente != 2 || c.Entradas[0].Atributos["habilitado"] != "true" || p.Entradas[0].Atributos["habilitado"] != "false" {
		t.Fatal("preparacion modifica decision original")
	}
	cambio.HuellaEsperada = "ajena"
	if _, err := Preparar(cfg, c, registro, ahora, cambio); !errors.Is(err, ErrConflicto) {
		t.Fatal(err)
	}
	cambio.HuellaEsperada = huella
	cambio.ModuloID = admin.ModuleID
	if _, err := Preparar(cfg, c, registro, ahora, cambio); !errors.Is(err, ErrCambio) {
		t.Fatal(err)
	}
}
