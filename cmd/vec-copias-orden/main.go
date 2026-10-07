// vec-copias-orden verifies and accepts authenticated synthetic orders only.
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
	destination "vec-diputacion-granada/internal/modules/administracion/adapters/destinocopias"
	adapter "vec-diputacion-granada/internal/modules/administracion/adapters/ordenescopias"
	"vec-diputacion-granada/internal/modules/administracion/adapters/ordenescopias/sintetico"
	app "vec-diputacion-granada/internal/modules/administracion/application/ordenescopias"
	dominio "vec-diputacion-granada/internal/modules/administracion/domain/ordenescopias"
	puerto "vec-diputacion-granada/internal/modules/administracion/ports/ordenescopias"
	"vec-diputacion-granada/internal/shared/i18n"
)

const maxBytes = 1 << 20

type configuracion struct {
	ClaveFichero string `json:"clave_fichero"`
	ClaveRef     string `json:"clave_ref"`
	ClaveVersion string `json:"clave_version"`
}
type entrada struct {
	Sintetica bool          `json:"sintetica"`
	Orden     dominio.Datos `json:"orden"`
	Instante  time.Time     `json:"instante"`
	Pasos     []string      `json:"pasos"`
}
type paso struct {
	Accion string `json:"accion"`
	Recibo string `json:"recibo,omitempty"`
	Replay bool   `json:"replay"`
}

func main() {
	timer := time.AfterFunc(30*time.Second, func() { os.Exit(2) })
	code := ejecutar(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	timer.Stop()
	os.Exit(code)
}
func ejecutar(args []string, in io.Reader, out, diag io.Writer) int {
	f := flag.NewFlagSet("vec-copias-orden", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	ruta := f.String("textos", "", "")
	configRuta := f.String("config", "", "")
	if f.Parse(args) != nil || f.NArg() != 0 || *ruta == "" || *configRuta == "" {
		return fallo(diag, nil, "argumentos_invalidos")
	}
	catalogo, err := catalogo(*ruta)
	if err != nil {
		return fallo(diag, nil, "catalogo_invalido")
	}
	var e entrada
	if decodificar(in, &e) != nil || !e.Sintetica || len(e.Pasos) == 0 || len(e.Pasos) > 128 {
		return fallo(diag, catalogo, "entrada_invalida")
	}
	o, err := dominio.Nueva(e.Orden)
	if err != nil {
		return fallo(diag, catalogo, "orden_invalida")
	}
	entorno, err := sintetico.Nuevo(o)
	if err != nil {
		return fallo(diag, catalogo, "orden_denegada")
	}
	var cfg configuracion
	cb, err := destination.LeerMaterialLocal(*configRuta, 65536, true)
	if err != nil || decodificar(bytes.NewReader(cb), &cfg) != nil {
		return fallo(diag, catalogo, "configuracion_invalida")
	}
	kb, err := destination.LeerMaterialLocal(cfg.ClaveFichero, 32, true)
	if err != nil || len(kb) != 32 {
		clear(kb)
		return fallo(diag, catalogo, "configuracion_invalida")
	}
	var maestra [32]byte
	copy(maestra[:], kb)
	clear(kb)
	defer clear(maestra[:])
	protector, err := bootstrap.NuevoProtectorOrdenCopiasDesarrollo(maestra, cfg.ClaveRef, cfg.ClaveVersion, 8192)
	if err != nil {
		return fallo(diag, catalogo, "configuracion_invalida")
	}
	autenticador, err := adapter.NuevoProtectorOrden(protector)
	if err != nil {
		return fallo(diag, catalogo, "configuracion_invalida")
	}
	s, err := app.Nuevo(entorno, autenticador, entorno, entorno)
	if err != nil {
		return fallo(diag, catalogo, "orden_denegada")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	sobre, err := s.Publicar(ctx, o, e.Instante)
	if err != nil {
		return fallo(diag, catalogo, "orden_denegada")
	}
	pasos := make([]paso, 0, len(e.Pasos))
	var reciboPrevio string
	for _, accion := range e.Pasos {
		var r puerto.Aceptacion
		switch accion {
		case "verificar":
			err = s.Verificar(ctx, sobre, e.Instante)
		case "aceptar":
			r, err = s.Aceptar(ctx, sobre, e.Instante)
			reciboPrevio = r.Recibo
		case "replay":
			if reciboPrevio == "" {
				return fallo(diag, catalogo, "entrada_invalida")
			}
			r, err = s.Aceptar(ctx, sobre, e.Instante)
			if err == nil && (!r.Replay || r.Recibo != reciboPrevio) {
				return fallo(diag, catalogo, "orden_denegada")
			}
		default:
			return fallo(diag, catalogo, "entrada_invalida")
		}
		if err != nil {
			return fallo(diag, catalogo, "orden_denegada")
		}
		pasos = append(pasos, paso{accion, r.Recibo, r.Replay})
	}
	result := struct {
		Alcance   string `json:"alcance"`
		Aviso     string `json:"aviso"`
		ConsumoV3 bool   `json:"consumo_v3_acreditado"`
		Durable   bool   `json:"aceptacion_durable"`
		Efecto    bool   `json:"efecto_plataforma"`
		Pasos     []paso `json:"pasos"`
	}{Alcance: "ensayo_sintetico_sobre_autenticado", Aviso: catalogo.T(i18n.DefaultLocale, "aviso_sintetico"), Pasos: pasos}
	if json.NewEncoder(out).Encode(result) != nil {
		return fallo(diag, catalogo, "salida_fallida")
	}
	return 0
}
func catalogo(ruta string) (*i18n.Catalog, error) {
	// #nosec G304 -- Operator-selected local catalogue only; no request path or
	// server integration, bounded regular input, no path/error reflected.
	f, err := os.Open(ruta)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Size() > maxBytes {
		return nil, dominio.ErrOrden
	}
	var m map[string]string
	if decodificar(f, &m) != nil {
		return nil, dominio.ErrOrden
	}
	for _, k := range []string{"aviso_sintetico", "argumentos_invalidos", "catalogo_invalido", "entrada_invalida", "orden_invalida", "orden_denegada", "salida_fallida", "configuracion_invalida"} {
		if m[k] == "" {
			return nil, dominio.ErrOrden
		}
	}
	return i18n.New(i18n.DefaultLocale, map[string]map[string]string{i18n.DefaultLocale: m})
}
func decodificar(r io.Reader, target any) error {
	b, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil || len(b) > maxBytes {
		return dominio.ErrOrden
	}
	scanner := json.NewDecoder(bytes.NewReader(b))
	if claves(scanner) != nil {
		return dominio.ErrOrden
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(target) != nil || d.Decode(new(any)) != io.EOF {
		return dominio.ErrOrden
	}
	return nil
}
func claves(d *json.Decoder) error {
	t, err := d.Token()
	if err != nil {
		return dominio.ErrOrden
	}
	switch t {
	case json.Delim('{'):
		seen := map[string]bool{}
		for d.More() {
			t, e := d.Token()
			k, ok := t.(string)
			if e != nil || !ok || k == "" || seen[k] {
				return dominio.ErrOrden
			}
			for _, c := range k {
				if !(c >= 'a' && c <= 'z') && !(c >= '0' && c <= '9') && c != '_' {
					return dominio.ErrOrden
				}
			}
			seen[k] = true
			if claves(d) != nil {
				return dominio.ErrOrden
			}
		}
		t, err = d.Token()
		if err != nil || t != json.Delim('}') {
			return dominio.ErrOrden
		}
	case json.Delim('['):
		for d.More() {
			if claves(d) != nil {
				return dominio.ErrOrden
			}
		}
		t, err = d.Token()
		if err != nil || t != json.Delim(']') {
			return dominio.ErrOrden
		}
	}
	return nil
}
func fallo(out io.Writer, c *i18n.Catalog, key string) int {
	r := map[string]string{"error": key}
	if c != nil {
		r["mensaje"] = c.T(i18n.DefaultLocale, key)
	}
	_ = json.NewEncoder(out).Encode(r)
	return 2
}
