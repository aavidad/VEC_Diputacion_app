// vec-copias-almacenar is an offline synthetic storage exerciser, not ADMIN.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
	"os"
	"time"
	"vec-diputacion-granada/internal/app/bootstrap"
	adapter "vec-diputacion-granada/internal/modules/administracion/adapters/destinocopias"
	app "vec-diputacion-granada/internal/modules/administracion/application/destinocopias"
	ports "vec-diputacion-granada/internal/modules/administracion/ports/destinocopias"
	"vec-diputacion-granada/internal/shared/i18n"
)

type configuracion struct {
	Raiz                 string `json:"raiz"`
	ClaveFichero         string `json:"clave_fichero"`
	ClaveRef             string `json:"clave_ref"`
	ClaveVersion         string `json:"clave_version"`
	MaximoClaroBytes     int64  `json:"maximo_claro_bytes"`
	TiempoMaximoSegundos int64  `json:"tiempo_maximo_segundos"`
}
type resultado struct {
	Codigo     string            `json:"codigo"`
	Mensaje    string            `json:"mensaje,omitempty"`
	Referencia *ports.Referencia `json:"referencia,omitempty"`
}

func main() {
	if run(os.Args[1:], os.Stdout, os.Stderr) != 0 {
		os.Exit(1)
	}
}
func run(args []string, out, fallos io.Writer) int {
	flags := flag.NewFlagSet("vec-copias-almacenar", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	cfgPath := flags.String("config", "", "")
	accion := flags.String("accion", "", "")
	entrada := flags.String("entrada", "", "")
	manifest := flags.String("manifiesto", "", "")
	referencia := flags.String("referencia", "", "")
	conjunto := flags.String("conjunto", "", "")
	componente := flags.String("componente", "", "")
	posicion := flags.Uint64("posicion", 0, "")
	sintetico := flags.Bool("sintetico", false, "")
	locale := flags.String("idioma", "", "")
	textos := flags.String("catalogo", "", "")
	if flags.Parse(args) != nil || flags.NArg() != 0 {
		return fallo(fallos, nil, *locale, "copias_destino.configuracion")
	}
	catalog, err := cargarCatalogo(*textos, *locale)
	if err != nil {
		return fallo(fallos, nil, *locale, "copias_destino.catalogo")
	}
	if !*sintetico || *cfgPath == "" {
		return fallo(fallos, catalog, *locale, "copias_destino.configuracion")
	}
	var cfg configuracion
	b, err := adapter.LeerMaterialLocal(*cfgPath, 65536, true)
	if err != nil || decodificar(b, &cfg) != nil {
		return fallo(fallos, catalog, *locale, "copias_destino.configuracion")
	}
	if cfg.TiempoMaximoSegundos < 1 || cfg.TiempoMaximoSegundos > int64((1<<63-1)/int64(time.Second)) {
		return fallo(fallos, catalog, *locale, "copias_destino.configuracion")
	}
	k, err := adapter.LeerMaterialLocal(cfg.ClaveFichero, 32, true)
	if err != nil || len(k) != 32 {
		clear(k)
		return fallo(fallos, catalog, *locale, "copias_destino.configuracion")
	}
	var maestra [32]byte
	copy(maestra[:], k)
	clear(k)
	defer clear(maestra[:])
	protector, err := bootstrap.NuevoProtectorCopiasDesarrollo(maestra, cfg.ClaveRef, cfg.ClaveVersion, cfg.MaximoClaroBytes)
	if err != nil {
		return fallo(fallos, catalog, *locale, "copias_destino.configuracion")
	}
	fs, err := adapter.NuevoFilesystem(adapter.ConfiguracionFilesystem{Raiz: cfg.Raiz, MaximoClaroBytes: cfg.MaximoClaroBytes}, protector)
	if err != nil {
		return fallo(fallos, catalog, *locale, "copias_destino.configuracion")
	}
	defer fs.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.TiempoMaximoSegundos)*time.Second)
	defer cancel()
	var r ports.Referencia
	switch *accion {
	case "almacenar":
		if *entrada == "" || *manifest == "" || *referencia != "" {
			return fallo(fallos, catalog, *locale, "copias_destino.configuracion")
		}
		contenido, e := adapter.LeerMaterialLocal(*entrada, cfg.MaximoClaroBytes, false)
		if e != nil {
			return fallo(fallos, catalog, *locale, "copias_destino.material")
		}
		defer clear(contenido)
		m, e := adapter.LeerMaterialLocal(*manifest, cfg.MaximoClaroBytes, false)
		if e != nil {
			return fallo(fallos, catalog, *locale, "copias_destino.material")
		}
		defer clear(m)
		r, err = fs.Publicar(ctx, ports.Solicitud{Vinculo: ports.Vinculo{ConjuntoRef: *conjunto, ComponenteRef: *componente, Posicion: *posicion, ManifiestoSHA256: app.Huella(m)}, Manifiesto: m, Contenido: contenido})
	case "comprobar", "borrar":
		if *referencia == "" || *entrada != "" || *manifest != "" {
			return fallo(fallos, catalog, *locale, "copias_destino.configuracion")
		}
		rb, e := adapter.LeerMaterialLocal(*referencia, 65536, false)
		if e != nil || decodificar(rb, &r) != nil {
			return fallo(fallos, catalog, *locale, "copias_destino.material")
		}
		if *accion == "comprobar" {
			err = fs.Comprobar(ctx, r)
		} else {
			err = fs.Borrar(ctx, r)
		}
	default:
		return fallo(fallos, catalog, *locale, "copias_destino.configuracion")
	}
	if err != nil {
		return fallo(fallos, catalog, *locale, "copias_destino.material")
	}
	return emitir(out, resultado{Codigo: "copias_destino." + *accion, Mensaje: mensajeCodigo(catalog, *locale, "copias_destino."+*accion), Referencia: &r})
}
func fallo(out io.Writer, c *i18n.Catalog, locale, key string) int {
	m := ""
	if c != nil {
		m = mensajeCodigo(c, locale, key)
	}
	_ = emitir(out, resultado{Codigo: key, Mensaje: m})
	return 1
}
func emitir(out io.Writer, r resultado) int {
	if json.NewEncoder(out).Encode(r) != nil {
		return 1
	}
	return 0
}
func decodificar(b []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return app.ErrMaterial
	}
	if d.Decode(new(any)) != io.EOF {
		return app.ErrMaterial
	}
	return nil
}
func cargarCatalogo(fichero, idioma string) (*i18n.Catalog, error) {
	if fichero == "" || !app.ReferenciaOpaca(idioma) {
		return nil, app.ErrConfiguracion
	}
	b, e := adapter.LeerMaterialLocal(fichero, 65536, false)
	if e != nil {
		return nil, app.ErrConfiguracion
	}
	var values map[string]string
	if e = decodificar(b, &values); e != nil {
		return nil, e
	}
	return i18n.New(idioma, map[string]map[string]string{idioma: values})
}

// mensajeCodigo conserva los códigos nominales y adapta únicamente su clave
// al formato plano del catálogo compartido por web y CLI.
func mensajeCodigo(c *i18n.Catalog, locale, codigo string) string {
	var codigoTexto string
	switch codigo {
	case "copias_destino.configuracion":
		codigoTexto = "copias_destino_configuracion"
	case "copias_destino.catalogo":
		codigoTexto = "copias_destino_catalogo"
	case "copias_destino.material":
		codigoTexto = "copias_destino_material"
	case "copias_destino.no_disponible":
		codigoTexto = "copias_destino_no_disponible"
	case "copias_destino.existe":
		codigoTexto = "copias_destino_existe"
	case "copias_destino.almacenar":
		codigoTexto = "copias_destino_almacenar"
	case "copias_destino.comprobar":
		codigoTexto = "copias_destino_comprobar"
	case "copias_destino.borrar":
		codigoTexto = "copias_destino_borrar"
	default:
		return ""
	}
	mensaje, _ := c.Message(locale, codigoTexto)
	return mensaje
}
