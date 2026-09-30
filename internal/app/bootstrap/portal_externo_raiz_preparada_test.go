package bootstrap

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"vec-diputacion-granada/config"
)

func TestRaizPropiaExternaNoSeDerivaYSeReutiliza(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	ahora := time.Now()
	idem, err := cargarMaterialIdempotenciaDesarrollo(cfg.DevelopmentMaterialDir, filepath.Join(cfg.DevelopmentMaterialDir, config.DevelopmentIdempotencyHMACConfigRelativePath))
	if err != nil {
		t.Fatal(err)
	}
	defer idem.borrar()
	d, err := nuevoDerivadorIdentidadOperacionDesarrollo(&idem)
	if err != nil {
		t.Fatal(err)
	}
	defer d.borrar()
	interna, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(d, ahora)
	if err != nil {
		t.Fatal(err)
	}
	defer interna.borrarCopiasEfimeras()
	a, err := prepararBasePropiaPortalExterno(cfg.DevelopmentMaterialDir, "", ahora)
	if err != nil {
		t.Fatal(err)
	}
	defer a.borrarCopiasEfimeras()
	b, err := prepararBasePropiaPortalExterno(cfg.DevelopmentMaterialDir, "", ahora.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	defer b.borrarCopiasEfimeras()
	if bytes.Equal(a.privada, interna.privada) || bytes.Equal(a.spki, interna.spki) {
		t.Fatal("la raíz propia coincide con la derivable")
	}
	if !bytes.Equal(a.privada, b.privada) || a.claveID != b.claveID || !bytes.Equal(a.claveHMAC, b.claveHMAC) {
		t.Fatal("la preparación repetida cambió raíz o capacidades")
	}
	// Copiar solo la idempotencia reproduce las capacidades, pero no la raíz.
	otro := directorioExternoPrueba(t)
	if err := os.Mkdir(filepath.Join(otro, "idempotencia"), 0o700); err != nil {
		t.Fatal(err)
	}
	entradas, err := os.ReadDir(filepath.Join(cfg.DevelopmentMaterialDir, "idempotencia"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entradas {
		c, err := os.ReadFile(filepath.Join(cfg.DevelopmentMaterialDir, "idempotencia", e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(otro, "idempotencia", e.Name()), c, 0o600)
		clear(c)
		if err != nil {
			t.Fatal(err)
		}
	}
	c, err := prepararBasePropiaPortalExterno(otro, "", ahora)
	if err != nil {
		t.Fatal(err)
	}
	defer c.borrarCopiasEfimeras()
	if bytes.Equal(a.spki, c.spki) || !bytes.Equal(a.claveHMAC, c.claveHMAC) {
		t.Fatal("la raíz aleatoria depende de la idempotencia o cambió la capacidad")
	}
}

func TestRaizExternaIncompletaNuncaSeRegenera(t *testing.T) {
	for _, modo := range []string{"antigua", "manipulada", "enlazada"} {
		t.Run(modo, func(t *testing.T) {
			cfg, _ := generarMaterialDesarrolloPrueba(t)
			if modo == "antigua" {
				if err := os.MkdirAll(filepath.Join(cfg.DevelopmentMaterialDir, "externo", "v3"), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(cfg.DevelopmentMaterialDir, "externo", "v3", "raiz.bin"), bytes.Repeat([]byte{5}, 32), 0o600); err != nil {
					t.Fatal(err)
				}
			} else {
				m, err := prepararBasePropiaPortalExterno(cfg.DevelopmentMaterialDir, "", time.Now())
				if err != nil {
					t.Fatal(err)
				}
				m.borrarCopiasEfimeras()
				ruta := filepath.Join(cfg.DevelopmentMaterialDir, filepath.FromSlash(estadoRaizPreparadaPortalExterno))
				if modo == "manipulada" {
					if err := os.WriteFile(ruta, []byte(`{"version":1,"semilla":"AQ=="}`), 0o600); err != nil {
						t.Fatal(err)
					}
				} else {
					if err := os.Rename(ruta, ruta+".original"); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(ruta+".original", ruta); err != nil {
						t.Fatal(err)
					}
				}
			}
			if m, err := prepararBasePropiaPortalExterno(cfg.DevelopmentMaterialDir, "", time.Now()); err == nil {
				m.borrarCopiasEfimeras()
				t.Fatal("se regeneró o adoptó una raíz inválida")
			}
		})
	}
}

func TestRotacionRaizExternaExigePreimagenYReutilizaPropuesta(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	a, err := prepararBasePropiaPortalExterno(cfg.DevelopmentMaterialDir, "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	defer a.borrarCopiasEfimeras()
	cat, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(descriptoresMaterialPortalExternoV3())
	if err != nil {
		t.Fatal(err)
	}
	publicados := map[string][]materialAtestacionContratacionTemporalDesarrollo{}
	for _, audiencia := range audienciasConsumidorPortalExternoV3("usuarios_preferencias") {
		d, _ := cat.descriptorPara(audiencia)
		m, err := derivarMaterialConsumidorV3Desarrollo(a, d)
		if err != nil {
			t.Fatal(err)
		}
		publicados["usuarios_preferencias"] = append(publicados["usuarios_preferencias"], m)
	}
	defer borrarMaterialesV3PortalExterno(publicados["usuarios_preferencias"])
	if err := escribirMaterialV3PortalExterno(cfg.DevelopmentMaterialDir, publicados); err != nil {
		t.Fatal(err)
	}
	if _, err := prepararBasePropiaPortalExterno(cfg.DevelopmentMaterialDir, "incorrecta", time.Now()); err == nil {
		t.Fatal("rotación sin preimagen aceptada")
	}
	b, err := prepararBasePropiaPortalExterno(cfg.DevelopmentMaterialDir, a.spkiHuella, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	defer b.borrarCopiasEfimeras()
	c, err := prepararBasePropiaPortalExterno(cfg.DevelopmentMaterialDir, a.spkiHuella, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	defer c.borrarCopiasEfimeras()
	if bytes.Equal(a.spki, b.spki) || !bytes.Equal(b.spki, c.spki) {
		t.Fatal("rotación no propia o repetición no estable")
	}
	if _, err := prepararBasePropiaPortalExterno(cfg.DevelopmentMaterialDir, "", time.Now()); err == nil {
		t.Fatal("se activó implícitamente una rotación sin publicar")
	}
}

func TestPreparacionPublicaPinCAInternaSinModificarAlmacenInterno(t *testing.T) {
	interna, _ := generarMaterialDesarrolloPrueba(t)
	externa, _ := generarMaterialDesarrolloPrueba(t)
	original, err := os.ReadFile(filepath.Join(interna.DevelopmentMaterialDir, "manifiesto.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := prepararManifiestoCAPropiaPortalExterno(interna.DevelopmentMaterialDir, externa.DevelopmentMaterialDir); err != nil {
		t.Fatal(err)
	}
	contenido, err := os.ReadFile(filepath.Join(externa.DevelopmentMaterialDir, "manifiesto.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Version int    `json:"version"`
		Propia  string `json:"huella_ca_sha256"`
		Interna string `json:"huella_ca_interna_sha256"`
	}
	if json.Unmarshal(contenido, &m) != nil || m.Version != 2 || m.Propia == m.Interna {
		t.Fatal("pin inválido")
	}
	b, err := os.ReadFile(filepath.Join(interna.DevelopmentMaterialDir, "ca", "ca.crt"))
	if err != nil {
		t.Fatal(err)
	}
	ca, err := decodificarCertificadoUnico(b)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(ca.Raw)
	if m.Interna != hex.EncodeToString(h[:]) {
		t.Fatal("pin distinto de la huella DER interna")
	}
	posterior, err := os.ReadFile(filepath.Join(interna.DevelopmentMaterialDir, "manifiesto.json"))
	if err != nil || !bytes.Equal(original, posterior) {
		t.Fatal("la preparación modificó el manifiesto interno")
	}
	if err := prepararManifiestoCAPropiaPortalExterno(interna.DevelopmentMaterialDir, interna.DevelopmentMaterialDir); err == nil {
		t.Fatal("se aceptó la CA interna como CA externa")
	}
}
