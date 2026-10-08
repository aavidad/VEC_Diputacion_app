package informepermisos

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	"vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type rendererPrueba struct {
	contenido    vecdomain.ContenidoDocumento
	validaciones int
	fallo        bool
}

func (*rendererPrueba) Formato() vecdomain.FormatoDocumento { return vecdomain.FormatoDocumentoPDF }
func (r *rendererPrueba) Renderizar(_ context.Context, c vecdomain.ContenidoDocumento) ([]byte, error) {
	r.contenido = c
	return []byte("%PDF-1.7\n/Lang (es-ES)\n%%EOF"), nil
}
func (r *rendererPrueba) ValidarSalida(context.Context, []byte) error {
	r.validaciones++
	if r.fallo {
		return pdf.ErrSalidaPDFInvalida
	}
	return nil
}
func catalogoPrueba(t *testing.T, idioma string) []byte {
	t.Helper()
	b, err := os.ReadFile("../../../../../web/static/textos/" + idioma + "/cronos-informe-permisos.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func resumenPrueba() ports.ResumenPermisosInforme {
	concedido := int64(150)
	return ports.ResumenPermisosInforme{Ejercicio: 2026, CorteUTC: time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC), CamposPermitidos: []string{"etiqueta", "unidad", "computo", "pendiente_resolver", "concedido", "restante", "conciliacion"}, Filas: []ports.FilaInformePermisos{{Etiqueta: "Permiso propio de ejemplo", Unidad: domain.LeaveUnitHour, Computo: domain.ComputoLaborables, Concedido: &concedido, Conciliacion: ports.ConciliacionPermisosPendiente}}}
}
func TestInformePermisosSoloResumenSinHuecosNiSubtotales(t *testing.T) {
	r := &rendererPrueba{}
	p, err := Nuevo(r, bytes.NewReader(catalogoPrueba(t, "es")))
	if err != nil {
		t.Fatal(err)
	}
	d, err := p.PrepararInformePermisos(context.Background(), resumenPrueba())
	if err != nil || len(d.Contenido) == 0 || r.validaciones != 1 {
		t.Fatal(err)
	}
	texto := strings.Join(r.contenido.Parrafos, "\n")
	if !strings.Contains(texto, "Restante: No disponible") || !strings.Contains(texto, "Concedido: 2 h 30 min") || !strings.Contains(texto, "resumen parcial") || !strings.Contains(texto, "Datos al 01/10/2026 12:00 CEST") {
		t.Fatal(texto)
	}
	if len(r.contenido.Parrafos) != 5 {
		t.Fatal("huecos/recuentos inesperados", r.contenido.Parrafos)
	}
	for _, prohibido := range []string{"tipo_ref", "excluido", "Total:", "Subtotal:", "solicitud", "motivo", "documento", "referencia"} {
		if strings.Contains(texto, prohibido) {
			t.Fatal("dato fuera del resumen", prohibido)
		}
	}
	// Una etiqueta que coincide con un marcador se conserva como dato literal.
	s := resumenPrueba()
	s.Filas[0].Etiqueta = "{{concedido}}"
	_, err = p.PrepararInformePermisos(context.Background(), s)
	if err != nil || !strings.Contains(strings.Join(r.contenido.Parrafos, "\n"), "{{concedido}};") {
		t.Fatal("etiqueta reinterpretada", err)
	}
}
func TestInformePermisosValidaPDFIdiomaYResumen(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		t.Run(idioma, func(t *testing.T) {
			r := &rendererPrueba{fallo: idioma == "es"}
			p, err := Nuevo(r, bytes.NewReader(catalogoPrueba(t, idioma)))
			if err != nil {
				t.Fatal(err)
			}
			d, err := p.PrepararInformePermisos(context.Background(), resumenPrueba())
			if err == nil || len(d.Contenido) != 0 || r.validaciones != 1 {
				t.Fatal("salida abierta", err)
			}
		})
	}
	p, err := Nuevo(&rendererPrueba{}, bytes.NewReader(catalogoPrueba(t, "es")))
	if err != nil {
		t.Fatal(err)
	}
	casos := []func(*ports.ResumenPermisosInforme){func(r *ports.ResumenPermisosInforme) { v := int64(1); r.Filas[0].Restante = &v }, func(r *ports.ResumenPermisosInforme) { v := int64(-1); r.Filas[0].Concedido = &v }, func(r *ports.ResumenPermisosInforme) { r.Filas[0].Unidad = "inventada" }, func(r *ports.ResumenPermisosInforme) { r.Filas[0].Etiqueta = "" }}
	for _, alterar := range casos {
		s := resumenPrueba()
		alterar(&s)
		d, err := p.PrepararInformePermisos(context.Background(), s)
		if err == nil || len(d.Contenido) != 0 {
			t.Fatal("resumen falso admitido")
		}
	}
}
func TestInformePermisosCatalogoEstricto(t *testing.T) {
	base := catalogoPrueba(t, "es")
	for _, version := range []string{"", "0", "01", "+1", "1.0", "1e0", " 1", "9223372036854775808"} {
		var c Catalogo
		if err := json.Unmarshal(base, &c); err != nil {
			t.Fatal(err)
		}
		c.Version = version
		b, err := json.Marshal(c)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Nuevo(&rendererPrueba{}, bytes.NewReader(b)); err == nil {
			t.Fatal("versión no canónica", version)
		}
	}
	malos := [][]byte{bytes.Replace(base, []byte(`"version": "2"`), []byte(`"version": 2`), 1), bytes.Replace(base, []byte(`"version": "2"`), []byte(`"version": "2", "version": "3"`), 1), bytes.Replace(base, []byte(`"version": "2"`), []byte(`"version": "2", "dato_ajeno": "x"`), 1), bytes.Replace(base, []byte(`"Restante: {{valor}}"`), []byte(`"Restante"`), 1), append(append([]byte(nil), base...), []byte(` {}`)...)}
	for _, b := range malos {
		if _, err := Nuevo(&rendererPrueba{}, bytes.NewReader(b)); err == nil {
			t.Fatal("catálogo incompleto/ambiguo admitido")
		}
	}
}
func TestInformePermisosPDFRealYCancelacion(t *testing.T) {
	p, err := Nuevo(pdf.Renderizador{}, bytes.NewReader(catalogoPrueba(t, "es")))
	if err != nil {
		t.Fatal(err)
	}
	d, err := p.PrepararEjemploSintetico(context.Background(), resumenPrueba(), "Carmen Molina Ortega")
	if err != nil || !bytes.HasPrefix(d.Contenido, []byte("%PDF-")) || d.CatalogoVersion != 2 {
		t.Fatal(err, len(d.Contenido))
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d, err = p.PrepararInformePermisos(ctx, resumenPrueba())
	if !errors.Is(err, context.Canceled) || len(d.Contenido) != 0 {
		t.Fatal(err)
	}
}

func TestInformePermisosSubconjuntoSinRotulosExcluidos(t *testing.T) {
	r := &rendererPrueba{}
	p, err := Nuevo(r, bytes.NewReader(catalogoPrueba(t, "es")))
	if err != nil {
		t.Fatal(err)
	}
	s := resumenPrueba()
	s.CamposPermitidos = []string{"etiqueta", "unidad", "concedido"}
	if _, err := p.PrepararInformePermisos(context.Background(), s); err != nil {
		t.Fatal(err)
	}
	texto := strings.Join(r.contenido.Parrafos, "\n")
	if !strings.Contains(texto, "Concedido: 2 h 30 min") || !strings.Contains(texto, "Unidad: Horas") {
		t.Fatal(texto)
	}
	for _, excluido := range []string{"Pendiente de resolver:", "Restante:", "Conciliación:", "Cómputo:", "No disponible", "Los valores no disponibles"} {
		if strings.Contains(texto, excluido) {
			t.Fatal("se filtró el campo", excluido, texto)
		}
	}
	for _, campos := range [][]string{nil, {"etiqueta", "etiqueta"}, {"etiqueta", "motivo"}, {"concedido"}} {
		s.CamposPermitidos = campos
		d, err := p.PrepararInformePermisos(context.Background(), s)
		if err == nil || len(d.Contenido) != 0 {
			t.Fatal("selección inválida admitida", campos)
		}
	}
}

func TestInformePermisosAceptaFilaMinimizada(t *testing.T) {
	r := &rendererPrueba{}
	p, err := Nuevo(r, bytes.NewReader(catalogoPrueba(t, "es")))
	if err != nil {
		t.Fatal(err)
	}
	s := resumenPrueba()
	s.CamposPermitidos = []string{"computo"}
	s.Filas[0] = ports.FilaInformePermisos{Computo: domain.ComputoLaborables}
	if _, err := p.PrepararInformePermisos(context.Background(), s); err != nil {
		t.Fatal(err)
	}
	texto := strings.Join(r.contenido.Parrafos, "\n")
	if !strings.Contains(texto, "Cómputo: Días laborables") || strings.Contains(texto, "Permiso propio") || strings.Contains(texto, "Unidad:") {
		t.Fatal(texto)
	}
	s.CamposPermitidos = []string{"unidad", "restante"}
	s.Filas[0] = ports.FilaInformePermisos{Unidad: domain.LeaveUnitDay, Restante: cantidadInformePermisosPrueba(4)}
	if _, err := p.PrepararInformePermisos(context.Background(), s); err != nil {
		t.Fatal(err)
	}
	texto = strings.Join(r.contenido.Parrafos, "\n")
	if !strings.Contains(texto, "Restante: 4") || strings.Contains(texto, "Conciliación:") {
		t.Fatal(texto)
	}
}

func cantidadInformePermisosPrueba(v int64) *int64 { return &v }

func TestInformePermisosParserPropagaErrorCerrado(t *testing.T) {
	for _, raw := range []string{
		``, `{"campo":`, `{"campo":"dato"`, `{"campo":"dato",}`, `{"mapa":{"campo":"a","campo":"b"}}`,
		`{"campo":1}`, `{"campo":null}`, `[]`, `{"campo":"a"} {}`, `{"campo":"a"} !`,
		strings.Repeat(`{"mapa":`, 10) + `"dato"` + strings.Repeat(`}`, 10),
	} {
		if err := validarJSONSinDuplicados([]byte(raw)); !errors.Is(err, ports.ErrExportacionPermisosInvalida) {
			t.Fatalf("parser sin error cerrado: %v", err)
		}
	}
	if err := validarJSONSinDuplicados([]byte(`{"mapa":{"campo":"dato"}}`)); err != nil {
		t.Fatal(err)
	}
}
