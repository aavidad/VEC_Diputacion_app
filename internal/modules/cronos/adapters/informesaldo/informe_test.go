package informesaldo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

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
func saldoPrueba() ports.SaldoExportable {
	previsto, saldo := int64(450), int64(-30)
	return ports.SaldoExportable{Periodo: ports.PeriodoConsultaSaldo{Tipo: ports.PeriodoSaldoRango, Desde: "2026-09-21", Hasta: "2026-09-21"}, Resumen: ports.ResumenConsultaSaldo{PrevistosMinutos: &previsto, TrabajadosMinutos: 420, SaldoMinutos: &saldo, Estado: ports.EstadoSaldoDisponible}}
}
func catalogoPrueba(t *testing.T, idioma string) []byte {
	t.Helper()
	datos, err := os.ReadFile("../../../../../web/static/textos/" + idioma + "/cronos-informe-saldo.json")
	if err != nil {
		t.Fatal(err)
	}
	return datos
}

func TestInformeSaldoMinimoConservaDesconocido(t *testing.T) {
	r := &rendererPrueba{}
	p, err := Nuevo(r, bytes.NewReader(catalogoPrueba(t, "es")))
	if err != nil {
		t.Fatal(err)
	}
	s := saldoPrueba()
	s.Resumen.PrevistosMinutos = nil
	s.Resumen.SaldoMinutos = nil
	s.Resumen.Estado = ports.EstadoSaldoNoDisponible
	s.EmpleadoRef = "emp_no_imprimir"
	s.FuenteRef = "fuente_no_imprimir"
	documento, err := p.PrepararInformeSaldo(context.Background(), s)
	if err != nil || len(documento.Contenido) == 0 || r.validaciones != 1 {
		t.Fatal(err, r.validaciones)
	}
	contenido, _ := json.Marshal(r.contenido)
	if !bytes.Contains(contenido, []byte("Tiempo previsto: No disponible")) || !bytes.Contains(contenido, []byte("Saldo: No disponible")) || bytes.Contains(contenido, []byte("emp_no_imprimir")) || bytes.Contains(contenido, []byte("fuente_no_imprimir")) {
		t.Fatalf("proyeccion inesperada: %s", contenido)
	}
	s = saldoPrueba()
	_, err = p.PrepararInformeSaldo(context.Background(), s)
	contenido, _ = json.Marshal(r.contenido)
	if err != nil || !bytes.Contains(contenido, []byte("Saldo: -30 min")) {
		t.Fatal(err, string(contenido))
	}
}

func TestInformeSaldoValidaSalidaEIdiomaAntesDeBytes(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		t.Run(idioma, func(t *testing.T) {
			r := &rendererPrueba{fallo: idioma == "es"}
			p, err := Nuevo(r, bytes.NewReader(catalogoPrueba(t, idioma)))
			if err != nil {
				t.Fatal(err)
			}
			d, err := p.PrepararInformeSaldo(context.Background(), saldoPrueba())
			if err == nil || len(d.Contenido) != 0 || r.validaciones != 1 {
				t.Fatalf("fallo abierto: bytes=%d validaciones=%d err=%v", len(d.Contenido), r.validaciones, err)
			}
		})
	}
}

func TestInformeSaldoRealYCancelacion(t *testing.T) {
	p, err := Nuevo(pdf.Renderizador{}, bytes.NewReader(catalogoPrueba(t, "es")))
	if err != nil {
		t.Fatal(err)
	}
	d, err := p.PrepararEjemploSintetico(context.Background(), saldoPrueba(), "Carmen Molina Ortega")
	if err != nil || !bytes.HasPrefix(d.Contenido, []byte("%PDF-")) || d.CatalogoVersion != 1 || len(d.CatalogoSHA256) != 64 {
		t.Fatal(err, len(d.Contenido))
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d, err = p.PrepararInformeSaldo(ctx, saldoPrueba())
	if !errors.Is(err, context.Canceled) || len(d.Contenido) != 0 {
		t.Fatal(err)
	}
}

func TestInformeSaldoRechazaSaldoFalsoYCatalogoIncomplete(t *testing.T) {
	p, err := Nuevo(&rendererPrueba{}, bytes.NewReader(catalogoPrueba(t, "es")))
	if err != nil {
		t.Fatal(err)
	}
	s := saldoPrueba()
	s.Resumen.Estado = ports.EstadoSaldoIncompleto
	if d, err := p.PrepararInformeSaldo(context.Background(), s); err == nil || len(d.Contenido) != 0 {
		t.Fatal("saldo incompleto presentado como conocido")
	}
	c := p.catalogo
	c.Saldo = "Saldo"
	datos, _ := json.Marshal(c)
	if _, err := Nuevo(&rendererPrueba{}, bytes.NewReader(datos)); err == nil {
		t.Fatal("catalogo pierde dato sin marcador")
	}
}

func TestInformeSaldoVersionDecimalCanonica(t *testing.T) {
	var catalogo Catalogo
	if err := json.Unmarshal(catalogoPrueba(t, "es"), &catalogo); err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"", "0", "-1", "01", "+1", "1.0", "1e0", " 1", "9223372036854775808"} {
		t.Run(version, func(t *testing.T) {
			c := catalogo
			c.Version = version
			datos, err := json.Marshal(c)
			if err != nil {
				t.Fatal(err)
			}
			if p, err := Nuevo(&rendererPrueba{}, bytes.NewReader(datos)); p != nil || !errors.Is(err, ports.ErrExportacionSaldoInvalida) {
				t.Fatalf("version=%q preparador=%v err=%v", version, p, err)
			}
		})
	}
}
