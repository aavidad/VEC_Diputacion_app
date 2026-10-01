package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"syscall"
	adapter "vec-diputacion-granada/internal/modules/carrera/adapters/cotejopromocionjson"
	"vec-diputacion-granada/internal/modules/carrera/application"
	"vec-diputacion-granada/internal/shared/i18n"
)

var localeValido = regexp.MustCompile(`^[a-z]{2}$`)

func catalogo(dir, locale string) (*i18n.Catalog, error) {
	if !localeValido.MatchString(locale) {
		return nil, adapter.ErrEntrada
	}
	raiz, err := os.OpenRoot(dir)
	if err != nil {
		return nil, adapter.ErrEntrada
	}
	defer raiz.Close()
	m := map[string]map[string]string{}
	for _, lang := range []string{i18n.DefaultLocale, locale} {
		f, err := raiz.OpenFile(filepath.Join(lang, "carrera-cotejo-promocion.json"), os.O_RDONLY|syscall.O_NONBLOCK, 0)
		if err != nil {
			return nil, adapter.ErrEntrada
		}
		info, fallo := f.Stat()
		if fallo != nil || !info.Mode().IsRegular() {
			if f.Close() != nil {
				return nil, adapter.ErrEntrada
			}
			return nil, adapter.ErrEntrada
		}
		textos, err := adapter.LeerCatalogo(f)
		closeErr := f.Close()
		if err != nil || closeErr != nil {
			return nil, adapter.ErrEntrada
		}
		m[lang] = textos
	}
	c, err := i18n.New(i18n.DefaultLocale, m)
	if err != nil {
		return nil, adapter.ErrEntrada
	}
	for _, clave := range []string{"carrera.cotejo.dictamen_ensayo", "carrera.cotejo.huella_local"} {
		if texto, ok := c.Message(locale, clave); !ok || texto == "" {
			return nil, adapter.ErrEntrada
		}
	}
	return c, nil
}

func run(args []string, in io.Reader, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("carrera-cotejo", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	locale := fs.String("idioma", i18n.DefaultLocale, "")
	dir := fs.String("catalogos", "web/static/textos", "")
	soloHuella := fs.Bool("huella-consulta", false, "")
	var err error
	if fs.Parse(args) != nil || fs.NArg() != 0 {
		err = adapter.ErrEntrada
	}
	var c *i18n.Catalog
	if err == nil {
		c, err = catalogo(*dir, *locale)
	}
	var e adapter.Entrada
	if err == nil {
		e, err = adapter.Leer(in)
	}
	var result any
	if err == nil && *soloHuella {
		var h string
		h, err = application.HuellaConsultaPromocion(e.Consulta)
		if err == nil {
			result = struct {
				EstadoGlobal string `json:"estado_global"`
				Etiqueta     string `json:"etiqueta"`
				Huella       string `json:"huella_consulta_sha256"`
			}{"pendiente", c.T(*locale, "carrera.cotejo.huella_local"), h}
		}
	} else if err == nil {
		r, fallo := (application.Servicio{}).CotejarPromocionSintetica(context.Background(), e.Consulta, adapter.Lector{Dictamen: e.Dictamen})
		err = fallo
		if err == nil {
			motivos := []string{}
			for _, comprobacion := range r.Dictamen.Comprobaciones {
				texto, ok := c.Message(*locale, comprobacion.MotivoClave)
				if !ok {
					err = adapter.ErrEntrada
					break
				}
				motivos = append(motivos, texto)
			}
			if err == nil {
				textos := map[string]string{}
				for _, fila := range r.Resultados {
					for _, clave := range []string{"carrera.cotejo.estado." + fila.EstadoMostrado, fila.MotivoGuardaClave} {
						if clave != "" {
							texto, ok := c.Message(*locale, clave)
							if !ok {
								err = adapter.ErrEntrada
								break
							}
							textos[clave] = texto
						}
					}
				}
				for _, clave := range r.Pendientes {
					texto, ok := c.Message(*locale, clave)
					if !ok {
						err = adapter.ErrEntrada
						break
					}
					textos[clave] = texto
				}

				result = struct {
					Etiqueta  string                               `json:"etiqueta"`
					Resultado application.CotejoPromocionSintetico `json:"resultado"`
					Motivos   []string                             `json:"motivos"`
					Textos    map[string]string                    `json:"textos"`
				}{c.T(*locale, r.EtiquetaClave), r, motivos, textos}
			}
		}
	}
	if err == nil {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if enc.Encode(result) != nil {
			err = adapter.ErrEntrada
		}
	}
	if err != nil {
		mensaje := ""
		if c != nil {
			mensaje = c.T(*locale, err.Error())
		}
		if json.NewEncoder(errOut).Encode(struct {
			Clave   string `json:"error_clave"`
			Mensaje string `json:"mensaje"`
		}{err.Error(), mensaje}) != nil {
			return 1
		}
		return 1
	}
	return 0
}
func main() { os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }
