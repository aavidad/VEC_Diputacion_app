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
	"syscall"
	"time"

	"vec-diputacion-granada/internal/modules/seleccion/adapters/bolsa"
	"vec-diputacion-granada/internal/modules/seleccion/application"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
	"vec-diputacion-granada/internal/shared/i18n"
)

var patronIdiomaCatalogo = regexp.MustCompile(`^[a-z]{2,3}(-[a-z0-9]{2,8})*$`)

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
	f, err := raiz.OpenFile(filepath.Join(idioma, "selectivos-preparar-bases.json"), os.O_RDONLY|syscall.O_NONBLOCK, 0)
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

type pendienteVisible struct {
	Campo   string `json:"campo"`
	Codigo  string `json:"codigo"`
	Mensaje string `json:"mensaje"`
}

func ejecutar(ctx context.Context, args []string, entrada io.Reader, salida, errores io.Writer) int {
	opciones := flag.NewFlagSet("vec-selectivos-preparar-bases", flag.ContinueOnError)
	opciones.SetOutput(io.Discard)
	dir := opciones.String("catalogos-dir", "web/static/textos", "")
	idioma := opciones.String("idioma", i18n.DefaultLocale, "")
	formato := opciones.String("salida", "preparacion", "")
	if opciones.Parse(args) != nil || opciones.NArg() != 0 || salida == nil || errores == nil || (*formato != "preparacion" && *formato != "solicitud-s2") {
		return informarError(errores, nil, *idioma, errEntradaJSON.Error())
	}
	catalogo, err := cargarCatalogo(*dir, *idioma)
	if *formato == "solicitud-s2" {
		if err != nil {
			return informarError(errores, catalogo, *idioma, err.Error())
		}
		return ejecutarSolicitudS2(ctx, entrada, salida, errores, catalogo, *idioma)
	}
	var material ports.MaterialBasesPropuesto
	if err == nil {
		err = leerJSON(entrada, &material)
	}
	var preparacion ports.PreparacionBases
	if err == nil {
		preparacion, err = application.PrepararMaterialBases(ctx, material, bolsa.CanonizadorBases{})
	}
	if err != nil {
		return informarError(errores, catalogo, *idioma, err.Error())
	}
	mensajes := make([]pendienteVisible, 0, len(preparacion.Pendientes))
	for _, pendiente := range preparacion.Pendientes {
		campo, okCampo := catalogo.Message(*idioma, "campo_"+pendiente.Campo)
		motivo, okMotivo := catalogo.Message(*idioma, pendiente.Codigo)
		if !okCampo || !okMotivo {
			return informarError(errores, catalogo, *idioma, errEntradaJSON.Error())
		}
		mensajes = append(mensajes, pendienteVisible{Campo: pendiente.Campo, Codigo: pendiente.Codigo, Mensaje: campo + ": " + motivo})
	}
	limite, existe := catalogo.Message(*idioma, "limite")
	if !existe {
		return informarError(errores, catalogo, *idioma, errEntradaJSON.Error())
	}
	enc := json.NewEncoder(salida)
	enc.SetIndent("", "  ")
	if enc.Encode(struct {
		Preparacion ports.PreparacionBases `json:"preparacion"`
		Limite      string                 `json:"limite"`
		Mensajes    []pendienteVisible     `json:"mensajes"`
	}{preparacion, limite, mensajes}) != nil {
		return informarError(errores, catalogo, *idioma, "seleccion.preparacion.salida_no_disponible")
	}
	return 0
}

func informarError(w io.Writer, c *i18n.Catalog, idioma, clave string) int {
	if w == nil {
		return 1
	}
	mensaje, _ := c.Message(idioma, "error_entrada")
	_ = json.NewEncoder(w).Encode(struct {
		Clave   string `json:"error_clave"`
		Mensaje string `json:"mensaje,omitempty"`
	}{clave, mensaje})
	return 1
}

func main() {
	ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelar()
	os.Exit(ejecutar(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
