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

	"vec-diputacion-granada/internal/modules/seleccion/application"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/shared/i18n"
)

var patronIdiomaCatalogo = regexp.MustCompile(`^[a-z]{2,3}(-[a-z0-9]{2,8})*$`)

const MotivoDiagnosticoNoDisponible = 2

func cargarCatalogo(dir, idioma string) (*i18n.Catalog, error) {
	if !patronIdiomaCatalogo.MatchString(idioma) {
		return nil, errEntradaJSON
	}
	raiz, err := os.OpenRoot(dir)
	if err != nil {
		return nil, errEntradaJSON
	}
	defer raiz.Close()
	f, err := raiz.OpenFile(filepath.Join(idioma, "selectivos-acta-preparacion.json"), os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, errEntradaJSON
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 64*1024 {
		return nil, errEntradaJSON
	}
	raw, err := io.ReadAll(io.LimitReader(f, 64*1024+1))
	var mensajes map[string]string
	if err != nil || len(raw) > 64*1024 || leerJSON(bytes.NewReader(raw), &mensajes) != nil {
		return nil, errEntradaJSON
	}
	for _, clave := range []string{"titulo", "limite", "error_entrada", "error_salida", "antecedente_no_cotejado",
		"pertenencia_no_verificada", "material_ausente", "circuito_pendiente", "campo_antecedente_tribunal",
		"campo_fase_propuesta", "campo_designacion", "campo_habilitacion", "campo_sesion_celebrada", "campo_asistencia",
		"campo_deliberaciones", "campo_acuerdos_adoptados", "campo_aprobacion", "campo_firma", "campo_sesion_ref",
		"campo_fecha_propuesta", "campo_orden_dia_propuesto", "campo_acuerdos_propuestos", "campo_textos_orden_dia", "campo_textos_acuerdos"} {
		if mensajes[clave] == "" {
			return nil, errEntradaJSON
		}
	}
	return i18n.New(idioma, map[string]map[string]string{idioma: mensajes})
}

type pendienteVisible struct {
	Campo   string `json:"campo"`
	Codigo  string `json:"codigo"`
	Mensaje string `json:"mensaje"`
}

func ejecutar(ctx context.Context, args []string, entrada io.Reader, salida, errores io.Writer) int {
	opciones := flag.NewFlagSet("vec-selectivos-preparar-acta", flag.ContinueOnError)
	opciones.SetOutput(io.Discard)
	dir := opciones.String("catalogos-dir", "web/static/textos", "")
	idioma := opciones.String("idioma", i18n.DefaultLocale, "")
	if opciones.Parse(args) != nil || opciones.NArg() != 0 || salida == nil || errores == nil {
		return informarError(errores, nil, *idioma, errEntradaJSON.Error())
	}
	catalogo, err := cargarCatalogo(*dir, *idioma)
	var material domain.MaterialActaPropuesto
	if err == nil {
		err = leerJSON(entrada, &material)
	}
	var preparacion domain.PreparacionActa
	if err == nil {
		preparacion, err = application.PrepararMaterialActa(ctx, material)
	}
	if err != nil {
		clave := errEntradaJSON.Error()
		if err == domain.ErrMaterialActaInvalido || err == application.ErrPreparacionActaNoDisponible {
			clave = err.Error()
		}
		return informarError(errores, catalogo, *idioma, clave)
	}
	mensajes := make([]pendienteVisible, 0, len(preparacion.Pendientes))
	for _, pendiente := range preparacion.Pendientes {
		campo, okCampo := catalogo.Message(*idioma, "campo_"+pendiente.Campo)
		motivo, okMotivo := catalogo.Message(*idioma, pendiente.Codigo)
		if !okCampo || !okMotivo {
			return informarError(errores, catalogo, *idioma, errEntradaJSON.Error())
		}
		mensajes = append(mensajes, pendienteVisible{pendiente.Campo, pendiente.Codigo, campo + ": " + motivo})
	}
	titulo, _ := catalogo.Message(*idioma, "titulo")
	limite, _ := catalogo.Message(*idioma, "limite")
	enc := json.NewEncoder(salida)
	enc.SetIndent("", "  ")
	if enc.Encode(struct {
		Titulo      string                 `json:"titulo"`
		Preparacion domain.PreparacionActa `json:"preparacion"`
		Limite      string                 `json:"limite"`
		Mensajes    []pendienteVisible     `json:"mensajes"`
	}{titulo, preparacion, limite, mensajes}) != nil {
		return informarError(errores, catalogo, *idioma, "seleccion.acta_preparacion.salida_no_disponible")
	}
	return 0
}

func informarError(w io.Writer, c *i18n.Catalog, idioma, clave string) int {
	if w != nil {
		claveMensaje := "error_entrada"
		if clave == "seleccion.acta_preparacion.salida_no_disponible" {
			claveMensaje = "error_salida"
		}
		mensaje, _ := c.Message(idioma, claveMensaje)
		if err := json.NewEncoder(w).Encode(struct {
			Clave   string `json:"error_clave"`
			Mensaje string `json:"mensaje,omitempty"`
		}{clave, mensaje}); err != nil {
			return MotivoDiagnosticoNoDisponible
		}
	}
	return 1
}

func main() {
	ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelar()
	os.Exit(ejecutar(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
