// vec-copias-calendario manages a synthetic external policy journal offline.
// It cannot capture, verify, delete or restore backup content, or grant ADMIN.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"os"
	"time"
	a "vec-diputacion-granada/internal/modules/administracion/adapters/politicacopias"
	d "vec-diputacion-granada/internal/modules/administracion/domain/politicacopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/politicacopias"
	"vec-diputacion-granada/internal/shared/i18n"
)

type entrada struct {
	Sintetica        bool        `json:"sintetica"`
	Accion           string      `json:"accion"`
	VersionEsperada  uint64      `json:"version_esperada,omitempty"`
	Politica         *d.Politica `json:"politica,omitempty"`
	ActorDeclarado   string      `json:"actor_declarado,omitempty"`
	RevisorDeclarado string      `json:"revisor_declarado,omitempty"`
	Correlacion      string      `json:"correlacion,omitempty"`
	Instante         time.Time   `json:"instante,omitempty"`
	Desde            time.Time   `json:"desde,omitempty"`
	Cantidad         int         `json:"cantidad,omitempty"`
	Copias           []d.Copia   `json:"copias,omitempty"`
}

func main() {
	timer := time.AfterFunc(30*time.Second, func() { os.Exit(2) })
	code := ejecutar(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	timer.Stop()
	os.Exit(code)
}
func ejecutar(args []string, in io.Reader, out, errout io.Writer) int {
	flags := flag.NewFlagSet("vec-copias-calendario", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	dir := flags.String("registro", "", "")
	textos := flags.String("textos", "", "")
	var roots []string
	flags.Func("raiz-restaurada", "", func(v string) error { roots = append(roots, v); return nil })
	if flags.Parse(args) != nil || flags.NArg() != 0 || *dir == "" || *textos == "" || len(roots) == 0 {
		return fallo(errout, nil, "argumentos_invalidos")
	}
	cat, e := catalogo(*textos)
	if e != nil {
		return fallo(errout, nil, "catalogo_invalido")
	}
	var input entrada
	if a.Decodificar(in, &input) != nil || !input.Sintetica {
		return fallo(errout, cat, "politica_entrada_invalida")
	}
	store, e := a.AbrirExterno(a.ConfigExterna{Directorio: *dir, RaicesRestauradas: roots})
	if e != nil {
		return fallo(errout, cat, "politica_dependencia_pendiente")
	}
	defer func() { _ = store.Cerrar() }()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var result any
	switch input.Accion {
	case "configurar":
		if input.Politica == nil || input.Cantidad != 0 || !input.Desde.IsZero() || len(input.Copias) != 0 {
			return fallo(errout, cat, "politica_entrada_invalida")
		}
		// These declared synthetic identities are provenance for an offline exercise,
		// not an implementation of central identity, permission or double approval.
		result, e = store.Guardar(ctx, input.VersionEsperada, *input.Politica, p.Atribucion{Actor: input.ActorDeclarado, Revisor: input.RevisorDeclarado, Correlacion: input.Correlacion}, input.Instante, func(context.Context) error { return nil })
	case "consultar":
		if !soloConsulta(input) || !input.Desde.IsZero() || input.Cantidad != 0 || len(input.Copias) != 0 {
			return fallo(errout, cat, "politica_entrada_invalida")
		}
		result, e = store.Historia(ctx)
	case "agenda":
		if !soloConsulta(input) || len(input.Copias) != 0 || !input.Instante.IsZero() {
			return fallo(errout, cat, "politica_entrada_invalida")
		}
		var current p.Registro
		current, e = store.Actual(ctx)
		if e == nil {
			result, e = current.Politica.Proximos(input.Desde, input.Cantidad)
		}
	case "retencion":
		if !soloConsulta(input) || !input.Desde.IsZero() || input.Cantidad != 0 {
			return fallo(errout, cat, "politica_entrada_invalida")
		}
		var current p.Registro
		current, e = store.Actual(ctx)
		if e == nil {
			result, e = current.Politica.PlanificarRetencion(input.Copias, input.Instante)
		}
	default:
		return fallo(errout, cat, "politica_entrada_invalida")
	}
	if e != nil {
		return fallo(errout, cat, e.Error())
	}
	type explicacion struct {
		Referencia string `json:"referencia"`
		Mensaje    string `json:"mensaje"`
	}
	var motivos []explicacion
	if plan, ok := result.(d.PlanRetencion); ok {
		for _, x := range plan.Decisiones {
			motivos = append(motivos, explicacion{x.Referencia, cat.T(i18n.DefaultLocale, x.Motivo)})
		}
	}
	response := struct {
		Alcance           string        `json:"alcance"`
		Aviso             string        `json:"aviso"`
		AutorizacionADMIN bool          `json:"autorizacion_admin"`
		HabilitaCopia     bool          `json:"habilita_copia"`
		HabilitaBorrado   bool          `json:"habilita_borrado"`
		Resultado         any           `json:"resultado"`
		Motivos           []explicacion `json:"motivos,omitempty"`
	}{Alcance: "configuracion_sintetica_local", Aviso: cat.T(i18n.DefaultLocale, "aviso_sintetico"), Resultado: result, Motivos: motivos}
	if json.NewEncoder(out).Encode(response) != nil {
		return fallo(errout, cat, "salida_fallida")
	}
	return 0
}
func soloConsulta(e entrada) bool {
	return e.Politica == nil && e.VersionEsperada == 0 && e.ActorDeclarado == "" && e.RevisorDeclarado == "" && e.Correlacion == "" && (e.Accion != "consultar" || e.Instante.IsZero())
}
func catalogo(ruta string) (*i18n.Catalog, error) {
	// #nosec G304 -- Trusted operator selects a local language catalogue; the CLI
	// does not expose filesystem paths or start a server.
	f, e := os.Open(ruta)
	if e != nil {
		return nil, d.ErrEntrada
	}
	defer func() { _ = f.Close() }()
	stat, e := f.Stat()
	if e != nil || !stat.Mode().IsRegular() || stat.Size() > a.MaxBytes {
		return nil, d.ErrEntrada
	}
	var texts map[string]string
	if a.Decodificar(f, &texts) != nil {
		return nil, d.ErrEntrada
	}
	for _, key := range []string{"aviso_sintetico", "argumentos_invalidos", "catalogo_invalido", "politica_entrada_invalida", "politica_version_distinta", "politica_historia_invalida", "politica_denegada", "politica_dependencia_pendiente", "politica_fuera_ventana", "salida_fallida", "previa_activa", "pendiente_conciliacion", "dependencia_necesaria", "protegida", "no_verificada", "minimo_verificadas", "dentro_retencion", "borrado_no_permitido", "candidata_revision", "fuera_destino"} {
		if texts[key] == "" {
			return nil, d.ErrEntrada
		}
	}
	return i18n.New(i18n.DefaultLocale, map[string]map[string]string{i18n.DefaultLocale: texts})
}
func fallo(out io.Writer, c *i18n.Catalog, key string) int {
	// Do not echo arbitrary filesystem or provider errors, which can contain paths.
	switch key {
	case "argumentos_invalidos", "catalogo_invalido", "politica_entrada_invalida", "politica_version_distinta", "politica_historia_invalida", "politica_denegada", "politica_dependencia_pendiente", "politica_fuera_ventana", "salida_fallida":
	default:
		key = "politica_dependencia_pendiente"
	}
	r := map[string]string{"error": key}
	if c != nil {
		r["mensaje"] = c.T(i18n.DefaultLocale, key)
	}
	_ = json.NewEncoder(out).Encode(r)
	return 2
}
