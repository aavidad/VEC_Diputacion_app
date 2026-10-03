// Command vec-preparar-fuentes-admin prepara y coteja fuentes sintéticas offline.
// No abre conexiones, no genera HMAC y no asigna cuentas ni permisos.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"time"

	"vec-diputacion-granada/internal/shared/i18n"
	"vec-diputacion-granada/internal/vec/domain"
)

const limiteDocumento = 64 << 10

type documento struct {
	Plan             domain.PlanFuentesInicialesAdminV1 `json:"plan"`
	HuellaPlanSHA256 string                             `json:"huella_plan_sha256"`
}
type diagnostico struct {
	Codigo           string `json:"codigo"`
	Mensaje          string `json:"mensaje"`
	Limite           string `json:"limite"`
	Preparado        bool   `json:"preparado"`
	Cotejado         bool   `json:"cotejado"`
	HuellaPlanSHA256 string `json:"huella_plan_sha256,omitempty"`
}

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdout, os.Stderr, time.Now)) }

func ejecutar(args []string, salida, errores io.Writer, reloj func() time.Time) int {
	f := flag.NewFlagSet("vec-preparar-fuentes-admin", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var fuente, destino, rutaTextos string
	var cotejar bool
	f.StringVar(&fuente, "fuente", "", "")
	f.StringVar(&destino, "plan", "", "")
	f.StringVar(&rutaTextos, "textos", "", "")
	f.BoolVar(&cotejar, "cotejar", false, "")
	parseErr := f.Parse(args)
	textos, idioma, err := cargarTextos(rutaTextos)
	if err != nil {
		return informarFalloCatalogo(errores, err)
	}
	emitir := func(w io.Writer, d diagnostico) int {
		d.Mensaje = textos.T(idioma, d.Codigo)
		d.Limite = textos.T(idioma, "limite")
		if json.NewEncoder(w).Encode(d) != nil {
			return 2
		}
		if !d.Preparado {
			return 1
		}
		return 0
	}
	fallo := func(codigo string) int { return emitir(errores, diagnostico{Codigo: codigo}) }
	if parseErr != nil || f.NArg() != 0 || fuente == "" || destino == "" || fuente == destino ||
		rutaTextos == fuente || rutaTextos == destino || reloj == nil {
		return fallo("uso_invalido")
	}
	b, err := leerPrivado(fuente)
	if err != nil {
		return fallo("fuente_insegura")
	}
	defer clear(b)
	var plan domain.PlanFuentesInicialesAdminV1
	if decodificarEstricto(b, &plan) != nil || plan.ValidarEn(reloj()) != nil {
		return fallo("fuente_invalida")
	}
	_, huella, err := plan.CanonicoYHuella()
	if err != nil {
		return fallo("fuente_invalida")
	}
	doc, err := json.Marshal(documento{Plan: plan, HuellaPlanSHA256: huella})
	if err != nil || len(doc)+1 > limiteDocumento {
		return fallo("fuente_invalida")
	}
	doc = append(doc, '\n')
	defer clear(doc)
	if cotejar {
		previo, err := leerPrivado(destino)
		if err != nil {
			return fallo("plan_inseguro")
		}
		defer clear(previo)
		if !bytes.Equal(previo, doc) {
			return fallo("plan_divergente")
		}
	} else if crearOComparar(destino, doc) != nil {
		return fallo("plan_divergente")
	}
	return emitir(salida, diagnostico{Codigo: "plan_preparado", Preparado: true, Cotejado: cotejar, HuellaPlanSHA256: huella})
}

type datosTextos struct {
	Idioma   string            `json:"idioma"`
	Mensajes map[string]string `json:"mensajes"`
}

func cargarTextos(ruta string) (*i18n.Catalog, string, error) {
	// El catálogo es público, pero se rechazan enlaces y archivos no regulares.
	raiz, err := os.OpenRoot(filepath.Dir(ruta))
	if err != nil {
		return nil, "", err
	}
	defer raiz.Close()
	f, err := raiz.OpenFile(filepath.Base(ruta), os.O_RDONLY|noSeguirEnlaces, 0)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > limiteDocumento {
		return nil, "", os.ErrInvalid
	}
	b, err := io.ReadAll(io.LimitReader(f, limiteDocumento+1))
	if err != nil || len(b) > limiteDocumento {
		return nil, "", os.ErrInvalid
	}
	var datos datosTextos
	if decodificarEstricto(b, &datos) != nil {
		return nil, "", os.ErrInvalid
	}
	claves := []string{"limite", "uso_invalido", "fuente_insegura", "fuente_invalida", "plan_inseguro", "plan_divergente", "plan_preparado"}
	if len(datos.Mensajes) != len(claves) {
		return nil, "", os.ErrInvalid
	}
	for _, clave := range claves {
		if datos.Mensajes[clave] == "" {
			return nil, "", os.ErrInvalid
		}
	}
	catalogo, err := i18n.New(datos.Idioma, map[string]map[string]string{datos.Idioma: datos.Mensajes})
	return catalogo, datos.Idioma, err
}
