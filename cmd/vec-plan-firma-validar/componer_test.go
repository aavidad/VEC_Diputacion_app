package main

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/plannominal"
	vd "vec-diputacion-granada/internal/vec/domain"
)

func composicionPrueba(t *testing.T) composicionCopia {
	t.Helper()
	b, _ := materialPrueba(t)
	var m material
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	catalogo, err := base64.StdEncoding.DecodeString(m.CatalogoBase64)
	if err != nil {
		t.Fatal(err)
	}
	var c vd.CatalogoConfigurable
	if err := json.Unmarshal(catalogo, &c); err != nil {
		t.Fatal(err)
	}
	c.Estado, c.PublicadoPor, c.AprobacionRef, c.MotivoPublicacion = vd.EstadoCatalogoBorrador, "", "", ""
	c.PublicadoEn = time.Time{}
	c.CreadoEn = time.Date(2026, 10, 20, 8, 0, 0, 0, time.UTC)
	c.Entradas[0].Clave = "informe_definitivo.p1"
	for _, k := range []string{"circuito_ref", "circuito_version", "circuito_sha256", "paso_ref"} {
		delete(c.Entradas[0].Atributos, k)
	}
	ruta, err := filepath.Abs("../../data/catalogos/contratacion-temporal/circuito_firma.v2.json")
	if err != nil {
		t.Fatal(err)
	}
	circuito, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	return composicionCopia{Catalogo: c, CircuitoArchivo: ruta, CircuitoArchivoSHA256: huella(circuito),
		Publicador: "actor:publicador:001", PerfilCreador: "perfil:admin:001", PerfilPublicador: "perfil:admin:002",
		AutorizacionCrear: "propuesta:crear:001", AutorizacionPublicar: "propuesta:publicar:001",
		CorrelacionCrear: "correlacion:crear:001", CorrelacionPublicar: "correlacion:publicar:001",
		ClaveCrear: "operacion-crear-0001", ClavePublicar: "operacion-publicar-0001",
		AprobacionRef: "aprobacion:001", MotivoPublicacion: "Publicación del plan",
		PublicadoEn: c.CreadoEn.Add(time.Hour)}
}

func TestComponerProduceMaterialesQueAceptaElValidadorExistente(t *testing.T) {
	cfg := composicionPrueba(t)
	antes := cfg.Catalogo.Entradas[0].Atributos["circuito_sha256"]
	crear, publicar, err := componerCopia(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Catalogo.Entradas[0].Atributos["circuito_sha256"] != antes {
		t.Fatal("se modificó la entrada")
	}
	var creado material
	for i, b := range []materialCompuesto{crear, publicar} {
		var m material
		if err := json.Unmarshal(b.Material, &m); err != nil {
			t.Fatal(err)
		}
		if _, err := validar(b.Material, b.Resumen.SHA256); err != nil {
			t.Fatal(err)
		}
		_, recurso, err := plannominal.RecursoGobiernoPlanFirma(b.Material,
			plannominal.AmbitoGobiernoPlanFirma{OrganizacionRef: "org_0123456789abcdef", UnidadRef: "unidad:rrhh:01"})
		if err != nil || recurso.Referencia != cfg.Catalogo.Referencia() {
			t.Fatal("consumidor administrativo rechazó material")
		}
		if i == 0 {
			creado = m
		} else if m.HuellaEsperada == nil || *m.HuellaEsperada != creado.CatalogoSHA {
			t.Fatal("publicación desligada del borrador")
		}
	}
}

func TestComponerRechazaActorUnicoCircuitoAjenoYEntradasIncompletas(t *testing.T) {
	for nombre, mutar := range map[string]func(*composicionCopia){
		"mismo_actor":      func(c *composicionCopia) { c.Publicador = c.Catalogo.CreadoPor },
		"misma_clave":      func(c *composicionCopia) { c.ClavePublicar = c.ClaveCrear },
		"sin_atributos":    func(c *composicionCopia) { c.Catalogo.Entradas[0].Atributos = nil },
		"circuito_ajeno":   func(c *composicionCopia) { c.Catalogo.Entradas[0].Atributos["circuito_version"] = "1" },
		"fichero_cambiado": func(c *composicionCopia) { c.CircuitoArchivoSHA256 = strings.Repeat("0", 64) },
	} {
		t.Run(nombre, func(t *testing.T) {
			cfg := composicionPrueba(t)
			mutar(&cfg)
			if _, _, err := componerCopia(cfg); err == nil {
				t.Fatal("entrada inválida admitida")
			}
		})
	}
}
