package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"vec-diputacion-granada/internal/modules/seleccion/application"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
	"vec-diputacion-granada/internal/shared/i18n"
)

var patronIdiomaCatalogo = regexp.MustCompile(`^[a-z]{2,3}(-[a-z0-9]{2,8})*$`)

const MotivoDiagnosticoNoDisponible = 2

func mensajeCatalogo(c *i18n.Catalog, idioma, codigo string) (string, bool) {
	return c.Message(idioma, strings.ReplaceAll(codigo, ".", "_"))
}

func cargarCatalogo(dir, idioma string) (*i18n.Catalog, error) {
	if !patronIdiomaCatalogo.MatchString(idioma) {
		return nil, errEntradaJSON
	}
	raiz, err := os.OpenRoot(dir)
	if err != nil {
		return nil, errEntradaJSON
	}
	defer raiz.Close()
	// Solo el catálogo fijo dentro de la raíz elegida por el operador; sin rutas del material.
	f, err := raiz.OpenFile(filepath.Join(idioma, "selectivos-admision.json"), os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, errEntradaJSON
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 64*1024 {
		return nil, errEntradaJSON
	}
	var mensajes map[string]string
	raw, err := io.ReadAll(io.LimitReader(f, 64*1024+1))
	if err != nil || len(raw) > 64*1024 || leerJSON(bytes.NewReader(raw), &mensajes) != nil {
		return nil, errEntradaJSON
	}
	catalogo, err := i18n.New(idioma, map[string]map[string]string{idioma: mensajes})
	if err != nil {
		return nil, errEntradaJSON
	}
	return catalogo, nil
}

func ejecutar(ctx context.Context, args []string, entrada io.Reader, salida, errores io.Writer) int {
	opciones := flag.NewFlagSet("vec-selectivos-preparar-admision", flag.ContinueOnError)
	opciones.SetOutput(io.Discard)
	dir := opciones.String("catalogos-dir", "web/static/textos", "")
	idioma := opciones.String("idioma", i18n.DefaultLocale, "")
	formato := opciones.String("salida", "preparacion", "")
	dirCatalogoAdmision := opciones.String("catalogo-admision-dir", "data/catalogos/seleccion", "")
	ficheroCatalogoAdmision := opciones.String("catalogo-admision", "admision_ejemplo.json", "")
	errOpciones := opciones.Parse(args)
	catalogo, err := cargarCatalogo(*dir, *idioma)
	if errOpciones == flag.ErrHelp && err == nil && salida != nil {
		ayuda, ok := mensajeCatalogo(catalogo, *idioma, "ayuda")
		if !ok {
			return informarError(errores, catalogo, *idioma, domain.ErrAdmisionPreparacion.Error())
		}
		if _, err := io.WriteString(salida, ayuda+"\n"); err != nil {
			return informarError(errores, catalogo, *idioma, "seleccion.admision.salida_no_disponible")
		}
		return 0
	}
	if err != nil || errOpciones != nil || opciones.NArg() != 0 || salida == nil || errores == nil ||
		(*formato != "preparacion" && *formato != "aportacion" && *formato != "antecedente" && *formato != "lista-provisional" &&
			*formato != "antecedente-lista" && *formato != "lista-definitiva" && *formato != "revision-provisional") {
		return informarError(errores, catalogo, *idioma, domain.ErrAdmisionPreparacion.Error())
	}
	if *formato == "lista-provisional" || *formato == "antecedente-lista" || *formato == "lista-definitiva" ||
		*formato == "revision-provisional" {
		return ejecutarLista(ctx, *formato, *dirCatalogoAdmision, *ficheroCatalogoAdmision, entrada, salida, errores, catalogo, *idioma)
	}
	if *formato != "preparacion" {
		return ejecutarAportacion(ctx, *formato, entrada, salida, errores, catalogo, *idioma)
	}
	var material ports.MaterialAdmisionPreparacion
	if leerJSON(entrada, &material) != nil {
		return informarError(errores, catalogo, *idioma, domain.ErrAdmisionPreparacion.Error())
	}
	preparacion, err := application.PrepararAdmision(ctx, material)
	if err != nil {
		return informarError(errores, catalogo, *idioma, domain.ErrAdmisionPreparacion.Error())
	}
	enc := json.NewEncoder(salida)
	enc.SetIndent("", "  ")
	if enc.Encode(preparacion) != nil {
		return informarError(errores, catalogo, *idioma, "seleccion.admision.salida_no_disponible")
	}
	return 0
}

func informarError(w io.Writer, c *i18n.Catalog, idioma, clave string) int {
	if w == nil {
		return MotivoDiagnosticoNoDisponible
	}
	mensaje, _ := mensajeCatalogo(c, idioma, clave)
	if err := json.NewEncoder(w).Encode(struct {
		Clave   string `json:"error_clave"`
		Mensaje string `json:"mensaje,omitempty"`
	}{clave, mensaje}); err != nil {
		return MotivoDiagnosticoNoDisponible
	}
	return 1
}

func main() {
	ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelar()
	os.Exit(ejecutar(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
