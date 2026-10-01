// Package informepermisos presenta exclusivamente el resumen anual mínimo.
// No conoce identidades, referencias de tipos ni políticas de autorización.
package informepermisos

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type Catalogo struct {
	Referencia      string            `json:"referencia"`
	Version         string            `json:"version"`
	Idioma          string            `json:"idioma"`
	FormatoFecha    string            `json:"formato_fecha"`
	ZonaHoraria     string            `json:"zona_horaria"`
	Titulo          string            `json:"titulo"`
	Ejercicio       string            `json:"ejercicio"`
	Corte           string            `json:"corte"`
	Alcance         string            `json:"alcance"`
	Fila            string            `json:"fila"`
	Dias            string            `json:"dias"`
	Duracion        string            `json:"duracion"`
	DuracionCorta   string            `json:"duracion_corta"`
	Desconocido     string            `json:"desconocido"`
	Vacio           string            `json:"vacio"`
	Limite          string            `json:"limite"`
	Sintetico       string            `json:"sintetico"`
	NombreSintetico string            `json:"nombre_sintetico"`
	Unidades        map[string]string `json:"unidades"`
	Computos        map[string]string `json:"computos"`
	Estados         map[string]string `json:"estados"`
}
type Preparador struct {
	renderer vecports.RenderizadorDocumento
	catalogo Catalogo
	version  int64
	huella   string
	zona     *time.Location
	impresor *message.Printer
}

func Nuevo(renderer vecports.RenderizadorDocumento, datos io.Reader) (*Preparador, error) {
	if nulo(renderer) || nulo(datos) || renderer.Formato() != vecdomain.FormatoDocumentoPDF {
		return nil, ports.ErrExportacionPermisosNoDisponible
	}
	raw, err := io.ReadAll(io.LimitReader(datos, 65537))
	if err != nil || len(raw) > 65536 || !jsonSinDuplicados(raw) {
		return nil, ports.ErrExportacionPermisosInvalida
	}
	var c Catalogo
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(&c) != nil || dec.Decode(new(any)) != io.EOF {
		return nil, ports.ErrExportacionPermisosInvalida
	}
	version, err := strconv.ParseInt(c.Version, 10, 64)
	if err != nil || version < 1 || strconv.FormatInt(version, 10) != c.Version {
		return nil, ports.ErrExportacionPermisosInvalida
	}
	idioma, err := language.Parse(c.Idioma)
	if err != nil || idioma.String() != c.Idioma || !texto(c.Referencia, 512) || !texto(c.FormatoFecha, 64) {
		return nil, ports.ErrExportacionPermisosInvalida
	}
	zona, err := time.LoadLocation(c.ZonaHoraria)
	if err != nil || !texto(c.ZonaHoraria, 64) {
		return nil, ports.ErrExportacionPermisosInvalida
	}
	for _, v := range []string{c.Titulo, c.Ejercicio, c.Corte, c.Alcance, c.Fila, c.Dias, c.Duracion, c.DuracionCorta, c.Desconocido, c.Vacio, c.Limite, c.Sintetico, c.NombreSintetico} {
		if !texto(v, 2048) {
			return nil, ports.ErrExportacionPermisosInvalida
		}
	}
	if !mapaValido(c.Unidades, []string{string(domain.LeaveUnitDay), string(domain.LeaveUnitHour)}) || !mapaValido(c.Computos, []string{string(domain.ComputoLaborables), string(domain.ComputoNaturales)}) || !mapaValido(c.Estados, []string{ports.ConciliacionPermisosConfirmada, ports.ConciliacionPermisosPendiente}) {
		return nil, ports.ErrExportacionPermisosInvalida
	}
	for _, p := range []struct {
		valor  string
		claves []string
	}{
		{c.Ejercicio, []string{"ejercicio"}}, {c.Corte, []string{"corte"}}, {c.Fila, []string{"etiqueta", "unidad", "computo", "pendiente", "concedido", "restante", "estado"}}, {c.Dias, []string{"cantidad"}}, {c.Duracion, []string{"horas", "minutos"}}, {c.DuracionCorta, []string{"minutos"}}, {c.NombreSintetico, []string{"nombre"}},
	} {
		restante := p.valor
		for _, clave := range p.claves {
			marcador := "{{" + clave + "}}"
			if strings.Count(restante, marcador) != 1 {
				return nil, ports.ErrExportacionPermisosInvalida
			}
			restante = strings.ReplaceAll(restante, marcador, "")
		}
		if strings.Contains(restante, "{{") || strings.Contains(restante, "}}") {
			return nil, ports.ErrExportacionPermisosInvalida
		}
	}
	suma := sha256.Sum256(raw)
	return &Preparador{renderer: renderer, catalogo: c, version: version, huella: hex.EncodeToString(suma[:]), zona: zona, impresor: message.NewPrinter(idioma)}, nil
}
func (p *Preparador) PrepararInformePermisos(ctx context.Context, r ports.ResumenPermisosInforme) (ports.DocumentoPermisosPreparado, error) {
	return p.preparar(ctx, r, "")
}

// Ejemplo aislado, marcado en el PDF. No es autoridad conectable a la exportación.
func (p *Preparador) PrepararEjemploSintetico(ctx context.Context, r ports.ResumenPermisosInforme, nombre string) (ports.DocumentoPermisosPreparado, error) {
	if !texto(nombre, 128) {
		return ports.DocumentoPermisosPreparado{}, ports.ErrExportacionPermisosInvalida
	}
	return p.preparar(ctx, r, nombre)
}
func (p *Preparador) preparar(ctx context.Context, r ports.ResumenPermisosInforme, nombre string) (ports.DocumentoPermisosPreparado, error) {
	cero := ports.DocumentoPermisosPreparado{}
	if p == nil || ctx == nil || nulo(p.renderer) || p.zona == nil || p.impresor == nil {
		return cero, ports.ErrExportacionPermisosNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if r.Ejercicio < 1 || r.Ejercicio > 9999 || r.CorteUTC.IsZero() || r.CorteUTC.Location() != time.UTC || r.CorteUTC.Nanosecond()%1000 != 0 || len(r.Filas) > 256 {
		return cero, ports.ErrExportacionPermisosInvalida
	}
	c := p.catalogo
	parrafos := []string{}
	if nombre != "" {
		parrafos = append(parrafos, c.Sintetico, sustituir(c.NombreSintetico, "nombre", nombre))
	}
	parrafos = append(parrafos, sustituir(c.Ejercicio, "ejercicio", strconv.Itoa(r.Ejercicio)), sustituir(c.Corte, "corte", r.CorteUTC.In(p.zona).Format(c.FormatoFecha)), c.Alcance)
	for _, f := range r.Filas {
		if !texto(f.Etiqueta, 256) || c.Unidades[string(f.Unidad)] == "" || c.Computos[string(f.Computo)] == "" || c.Estados[f.Conciliacion] == "" || (f.Conciliacion == ports.ConciliacionPermisosPendiente && f.Restante != nil) {
			return cero, ports.ErrExportacionPermisosInvalida
		}
		for _, v := range []*int64{f.PendienteResolver, f.Concedido, f.Restante} {
			if v != nil && *v < 0 {
				return cero, ports.ErrExportacionPermisosInvalida
			}
		}
		// Replacer sustituye simultáneamente: una etiqueta nunca se trata como plantilla.
		fila := strings.NewReplacer("{{etiqueta}}", f.Etiqueta, "{{unidad}}", c.Unidades[string(f.Unidad)], "{{computo}}", c.Computos[string(f.Computo)], "{{pendiente}}", p.cantidad(f.PendienteResolver, f.Unidad), "{{concedido}}", p.cantidad(f.Concedido, f.Unidad), "{{restante}}", p.cantidad(f.Restante, f.Unidad), "{{estado}}", c.Estados[f.Conciliacion]).Replace(c.Fila)
		parrafos = append(parrafos, fila)
	}
	if len(r.Filas) == 0 {
		parrafos = append(parrafos, c.Vacio)
	}
	parrafos = append(parrafos, c.Limite)
	contenido, err := p.renderer.Renderizar(ctx, vecdomain.ContenidoDocumento{Titulo: c.Titulo, Parrafos: parrafos})
	if err != nil {
		return cero, err
	}
	if len(contenido) == 0 || len(contenido) > 2*1024*1024 {
		return cero, ports.ErrExportacionPermisosInvalida
	}
	contenido = append([]byte(nil), contenido...)
	if err := p.renderer.ValidarSalida(ctx, contenido); err != nil {
		return cero, err
	}
	if !bytes.Contains(contenido, []byte("/Lang ("+c.Idioma+")")) {
		return cero, ports.ErrExportacionPermisosNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	return ports.DocumentoPermisosPreparado{Contenido: contenido, CatalogoRef: c.Referencia, CatalogoVersion: p.version, CatalogoSHA256: p.huella}, nil
}
func (p *Preparador) cantidad(v *int64, u domain.LeaveUnit) string {
	if v == nil {
		return p.catalogo.Desconocido
	}
	if u == domain.LeaveUnitDay {
		return sustituir(p.catalogo.Dias, "cantidad", p.impresor.Sprintf("%d", *v))
	}
	if *v < 60 {
		return sustituir(p.catalogo.DuracionCorta, "minutos", p.impresor.Sprintf("%d", *v))
	}
	return strings.NewReplacer("{{horas}}", p.impresor.Sprintf("%d", *v/60), "{{minutos}}", p.impresor.Sprintf("%d", *v%60)).Replace(p.catalogo.Duracion)
}
func mapaValido(m map[string]string, claves []string) bool {
	if len(m) != len(claves) {
		return false
	}
	for _, k := range claves {
		if !texto(m[k], 256) {
			return false
		}
	}
	return true
}
func sustituir(p, k, v string) string { return strings.ReplaceAll(p, "{{"+k+"}}", v) }
func texto(v string, max int) bool {
	if strings.TrimSpace(v) == "" || len(v) > max || !utf8.ValidString(v) {
		return false
	}
	for _, r := range v {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
func nulo(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		return r.IsNil()
	}
	return false
}

// Rechaza claves repetidas en cualquier objeto del catálogo, además del parseo
// de estructura estricta. La profundidad está acotada al tamaño del catálogo.
func jsonSinDuplicados(raw []byte) bool {
	d := json.NewDecoder(bytes.NewReader(raw))
	var valor func(int) bool
	valor = func(profundidad int) bool {
		if profundidad > 8 {
			return false
		}
		t, err := d.Token()
		if err != nil {
			return false
		}
		delim, ok := t.(json.Delim)
		if !ok {
			_, esTexto := t.(string)
			return esTexto
		}
		if delim != '{' {
			return false
		}
		vistos := map[string]bool{}
		for d.More() {
			clave, err := d.Token()
			if err != nil {
				return false
			}
			k, ok := clave.(string)
			if !ok || vistos[k] {
				return false
			}
			vistos[k] = true
			if !valor(profundidad + 1) {
				return false
			}
		}
		fin, err := d.Token()
		return err == nil && fin == json.Delim('}')
	}
	return valor(0) && func() bool { _, err := d.Token(); return err == io.EOF }()
}

var _ ports.PreparadorInformePermisos = (*Preparador)(nil)
