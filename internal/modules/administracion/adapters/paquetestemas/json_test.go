package paquetestemas

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/administracion/domain/temas"
)

func datos(t *testing.T, nombre string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "data", "temas", nombre))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func politica(t *testing.T) PoliticaValidada {
	t.Helper()
	b := datos(t, "politica-v1.json")
	p, err := LeerPolitica(bytes.NewReader(b), huella(b))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func paquete(t *testing.T) temas.Paquete {
	t.Helper()
	var p temas.Paquete
	if err := json.Unmarshal(datos(t, "institucional-v1.json"), &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestEjemplosYCanonico(t *testing.T) {
	policy := politica(t)
	for _, nombre := range []string{"institucional", "granate", "diputacion_granada", "arena", "salvia", "lavanda", "azul_sereno", "noche_suave"} {
		t.Run(nombre, func(t *testing.T) {
			b := datos(t, nombre+"-v1.json")
			r, err := Preparar(bytes.NewReader(b), policy)
			if err != nil {
				t.Fatal(err)
			}
			if r.Estado != "validado_sin_instalar" || r.OriginalSHA256 != huella(b) || r.PoliticaSHA256 != policy.huella {
				t.Fatal("metadata")
			}
			compacto, err := r.Material.Canonico()
			if err != nil {
				t.Fatal(err)
			}
			r2, err := Preparar(bytes.NewReader(compacto), policy)
			if err != nil || r2.CanonicoSHA256 != r.CanonicoSHA256 || r2.OriginalSHA256 == r.OriginalSHA256 {
				t.Fatal("canonical stability", err)
			}
		})
	}
	// A new ID needs no enum change, and hexadecimal casing normalizes.
	p := paquete(t)
	p.TemaID = "tercero-2027"
	p.NombreKey = "ui.temas.tercero-2027.nombre" // gitleaks:allow -- clave i18n pública de ejemplo, no secreto
	for idioma, textos := range p.Textos {
		for _, nombre := range textos {
			p.Textos[idioma] = map[string]string{p.NombreKey: nombre}
		}
	}
	p.Variantes.Clara["--portal-fondo-logo"] = "#FFFFFF"
	b, _ := json.Marshal(p)
	r, err := Preparar(bytes.NewReader(b), policy)
	if err != nil || r.Material.TemaID != p.TemaID || r.Material.Variantes.Clara["--portal-fondo-logo"] != "#ffffff" {
		t.Fatal("open IDs", err)
	}
}

func TestCopiaPolitica(t *testing.T) {
	p := politica(t)
	c := p.Copia()
	c.Tokens[0] = "--portal-otro"
	c.IdiomasRequeridos[0] = "otro"
	c.Contrastes[0].Minimo = 1
	c.ValoresFijos["--portal-fondo-logo"] = "#000000"
	if p.datos.Validar() != nil {
		t.Fatal("copy aliases policy")
	}
	if (PoliticaValidada{}).Copia().Validar() == nil {
		t.Fatal("zero copy")
	}
}

func TestParserAdversarial(t *testing.T) {
	b := string(datos(t, "institucional-v1.json"))
	for nombre, mal := range map[string]string{
		"duplicate":         strings.Replace(b, `"esquema": 1`, `"esquema": 1,"esquema": 1`, 1),
		"escaped_duplicate": strings.Replace(b, `"esquema": 1`, `"esquema": 1,"\u0065squema": 1`, 1),
		"nested_duplicate":  strings.Replace(b, `"--portal-fondo-logo": "#ffffff"`, `"--portal-fondo-logo":"#ffffff","--portal-fondo-logo":"#ffffff"`, 1),
		"unknown":           strings.Replace(b, `"esquema": 1`, `"esquema": 1,"css":"x"`, 1),
		"wrong_case":        strings.Replace(b, `"esquema"`, `"Esquema"`, 1),
		"trailing":          b + ` {}`, "empty": "", "utf8": b + string([]byte{0xff}),
		"depth":     strings.Repeat("[", temas.MaxProfundidad+1) + "0" + strings.Repeat("]", temas.MaxProfundidad+1),
		"too_large": strings.Repeat(" ", temas.MaxBytes+1),
	} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := Preparar(strings.NewReader(mal), politica(t)); err == nil {
				t.Fatal("accepted")
			}
		})
	}
}

func TestNombreNoColisionaConIdentificadorAnidado(t *testing.T) {
	p := paquete(t)
	p.TemaID = "arena"
	p.NombreKey = "ui.temas." + p.TemaID + ".otro.nombre"
	for idioma, textos := range p.Textos {
		for _, nombre := range textos {
			p.Textos[idioma] = map[string]string{p.NombreKey: nombre}
		}
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Preparar(bytes.NewReader(b), politica(t)); err == nil {
		t.Fatal("ancestor accepted the nested identifier name")
	}
	p.TemaID += ".otro"
	b, err = json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Preparar(bytes.NewReader(b), politica(t)); err != nil {
		t.Fatal("exact name rejected for its owner", err)
	}
}

func TestRechazosDeContrato(t *testing.T) {
	for nombre, cambiar := range map[string]func(*temas.Paquete){
		"tokens_missing": func(p *temas.Paquete) { delete(p.Variantes.Clara, "--portal-tinta") },
		"tokens_extra":   func(p *temas.Paquete) { p.Variantes.Clara["--portal-radio"] = "#ffffff" },
		"url":            func(p *temas.Paquete) { p.Variantes.Clara["--portal-tinta"] = "url(https://example.org)" },
		"alias":          func(p *temas.Paquete) { p.Variantes.Clara["--portal-tinta"] = "var(--portal-muted)" },
		"transparent":    func(p *temas.Paquete) { p.Variantes.Clara["--portal-tinta"] = "#000000ff" },
		"logo":           func(p *temas.Paquete) { p.Variantes.Clara["--portal-fondo-logo"] = "#000000" },
		"contrast_light": func(p *temas.Paquete) { p.Variantes.Clara["--portal-tinta"] = "#ffffff" },
		"contrast_dark": func(p *temas.Paquete) {
			p.Variantes.Oscura["--portal-tinta"] = p.Variantes.Oscura["--portal-superficie"]
		},
		"html_name":     func(p *temas.Paquete) { p.Textos["es"][p.NombreKey] = "<script>" },
		"control_name":  func(p *temas.Paquete) { p.Textos["es"][p.NombreKey] = "A\nB" },
		"interpolation": func(p *temas.Paquete) { p.Textos["es"][p.NombreKey] = "{nombre}" },
		"url_name":      func(p *temas.Paquete) { p.Textos["es"][p.NombreKey] = "https://example.org" },
		"extra_text":    func(p *temas.Paquete) { p.Textos["es"]["ui.temas.otro.nombre"] = "Otro" },
		"policy_ref":    func(p *temas.Paquete) { p.PoliticaRef = "otra-v1" },
		"design_ref":    func(p *temas.Paquete) { p.SistemaDisenoRef = "otro-v1" },
		"unsafe_id":     func(p *temas.Paquete) { p.TemaID = "x\"] body" },
	} {
		t.Run(nombre, func(t *testing.T) {
			p := paquete(t)
			cambiar(&p)
			b, _ := json.Marshal(p)
			if _, err := Preparar(bytes.NewReader(b), politica(t)); err == nil {
				t.Fatal("accepted")
			}
		})
	}
}

func TestPoliticaNoDebilitable(t *testing.T) {
	b := datos(t, "politica-v1.json")
	if _, err := LeerPolitica(bytes.NewReader(b), strings.Repeat("0", 64)); err == nil {
		t.Fatal("wrong hash")
	}
	if _, err := Preparar(bytes.NewReader(datos(t, "institucional-v1.json")), PoliticaValidada{}); err == nil {
		t.Fatal("zero policy")
	}
	for nombre, cambiar := range map[string]func(*temas.Politica){
		"bytes":     func(p *temas.Politica) { p.Limites.Bytes = temas.MaxBytes + 1 },
		"tokens":    func(p *temas.Politica) { p.Limites.Tokens = temas.MaxTokens + 1 },
		"threshold": func(p *temas.Politica) { p.Contrastes[0].Minimo = 4.49 },
		"component": func(p *temas.Politica) {
			for i := range p.Contrastes {
				if p.Contrastes[i].Tipo == "componente" {
					p.Contrastes[i].Minimo = 2.99
				}
			}
		},
		"missing_token":   func(p *temas.Politica) { p.Contrastes[0].PrimerPlano = "--portal-ausente" },
		"duplicate_token": func(p *temas.Politica) { p.Tokens = append(p.Tokens, p.Tokens[0]) },
		"duplicate_pair":  func(p *temas.Politica) { p.Contrastes = append(p.Contrastes, p.Contrastes[0]) },
	} {
		t.Run(nombre, func(t *testing.T) {
			var p temas.Politica
			_ = json.Unmarshal(b, &p)
			cambiar(&p)
			alterada, _ := json.Marshal(p)
			if _, err := LeerPolitica(bytes.NewReader(alterada), huella(alterada)); err == nil {
				t.Fatal("accepted")
			}
		})
	}
	p := politica(t)
	p.datos.Limites.Bytes = 1
	if _, err := Preparar(bytes.NewReader(datos(t, "institucional-v1.json")), p); err == nil {
		t.Fatal("policy reduction")
	}
}
