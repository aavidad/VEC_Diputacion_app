package composicion

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	"vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// El catálogo procede de datos i18n aprobados; no hay textos de documento
// dentro del ejecutable. Solo contiene etiquetas, nunca permisos de campos.
type CatalogoInformeC12 struct {
	Idioma, Titulo, Desconocido string
	Etiquetas                   map[string]string
}
type RenderizadorInformeC12 struct {
	catalogo CatalogoInformeC12
	pdf      pdf.Renderizador
}

func NuevoRenderizadorInformeC12(c CatalogoInformeC12) (*RenderizadorInformeC12, error) {
	if !textoInformeC12(c.Titulo, 160) || !textoInformeC12(c.Desconocido, 160) || c.Idioma == "" || len(c.Etiquetas) == 0 {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &RenderizadorInformeC12{catalogo: c, pdf: pdf.Renderizador{Idioma: c.Idioma, Membrete: true}}, nil
}
func (r *RenderizadorInformeC12) RenderizarInforme(ctx context.Context, formato domain.FormatoInformeCronos, p ports.PreparacionInformeC12) ([]byte, error) {
	if r == nil || ctx == nil || ctx.Err() != nil || len(p.Campos) == 0 || len(p.Campos) > 20 || len(p.Datos) > 10000 {
		return nil, ports.ErrDependenciaNoDisponible
	}
	etiquetas := make([]string, len(p.Campos))
	for i, c := range p.Campos {
		v := r.catalogo.Etiquetas[c]
		if !textoInformeC12(v, 160) {
			return nil, ports.ErrDependenciaNoDisponible
		}
		etiquetas[i] = v
	}
	switch formato {
	case domain.InformeCSV:
		var b bytes.Buffer
		w := csv.NewWriter(&b)
		cabeceras := make([]string, len(etiquetas))
		for i, etiqueta := range etiquetas {
			cabeceras[i] = seguroCSVInformeC12(etiqueta)
		}
		if w.Write(cabeceras) != nil {
			return nil, ports.ErrDependenciaNoDisponible
		}
		for _, fila := range p.Datos {
			if len(fila) != len(etiquetas) {
				return nil, ports.ErrDependenciaNoDisponible
			}
			valores := make([]string, len(fila))
			for i, v := range fila {
				valor, err := valorInformeC12(v, r.catalogo.Desconocido)
				if err != nil {
					return nil, ports.ErrDependenciaNoDisponible
				}
				valores[i] = seguroCSVInformeC12(valor)
			}
			if w.Write(valores) != nil {
				return nil, ports.ErrDependenciaNoDisponible
			}
		}
		w.Flush()
		if w.Error() != nil || b.Len() > 2*1024*1024 {
			return nil, ports.ErrDependenciaNoDisponible
		}
		return bytes.Clone(b.Bytes()), nil
	case domain.InformePDF:
		parrafos := make([]string, 0, len(p.Datos))
		for _, fila := range p.Datos {
			if len(fila) != len(etiquetas) {
				return nil, ports.ErrDependenciaNoDisponible
			}
			partes := make([]string, len(fila))
			for i, v := range fila {
				valor, err := valorInformeC12(v, r.catalogo.Desconocido)
				if err != nil {
					return nil, ports.ErrDependenciaNoDisponible
				}
				partes[i] = etiquetas[i] + ": " + valor
			}
			parrafos = append(parrafos, strings.Join(partes, " · "))
		}
		bytes, err := r.pdf.Renderizar(ctx, vecdomain.ContenidoDocumento{Titulo: r.catalogo.Titulo, Parrafos: parrafos})
		if err != nil || len(bytes) > 2*1024*1024 {
			return nil, ports.ErrDependenciaNoDisponible
		}
		if r.pdf.ValidarSalida(ctx, bytes) != nil {
			return nil, ports.ErrDependenciaNoDisponible
		}
		return bytes, nil
	default:
		return nil, ports.ErrDependenciaNoDisponible
	}
}
func valorInformeC12(raw json.RawMessage, desconocido string) (string, error) {
	if len(raw) == 0 || len(raw) > 2048 {
		return "", ports.ErrDependenciaNoDisponible
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var valor any
	if dec.Decode(&valor) != nil || dec.Decode(new(any)) != io.EOF {
		return "", ports.ErrDependenciaNoDisponible
	}
	switch v := valor.(type) {
	case nil:
		return desconocido, nil
	case string:
		if textoInformeC12(v, 1024) {
			return v, nil
		}
	case json.Number:
		if _, err := strconv.ParseInt(v.String(), 10, 64); err == nil {
			return v.String(), nil
		}
	}
	return "", ports.ErrDependenciaNoDisponible
}
func textoInformeC12(s string, max int) bool {
	if s == "" || len(s) > max || !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
func seguroCSVInformeC12(s string) string {
	if s == "" {
		return s
	}
	t := strings.TrimSpace(s)
	if t != "" && strings.ContainsRune("=+-@", rune(t[0])) {
		return "'" + s
	}
	return s
}

var _ ports.RenderizadorInformeC12 = (*RenderizadorInformeC12)(nil)
