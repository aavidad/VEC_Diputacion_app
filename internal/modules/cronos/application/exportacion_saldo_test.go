package application

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/adapters/informesaldo"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	"vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type fuenteExportacionPrueba struct {
	saldo    ports.SaldoExportable
	err      error
	llamadas int
	alLeer   func()
}

func (f *fuenteExportacionPrueba) LeerSaldoPropioParaExportar(_ context.Context, _ ports.OrdenExportacionSaldo, _ ports.PeriodoConsultaSaldo) (ports.SaldoExportable, error) {
	f.llamadas++
	if f.alLeer != nil {
		f.alLeer()
	}
	return f.saldo, f.err
}

type registroExportacionPrueba struct {
	reloj       *relojExportacionPrueba
	err         error
	modificar   func(*ports.ConfirmacionExportacionSaldo)
	llamadas    int
	alConfirmar func()
}

func (r *registroExportacionPrueba) ConfirmarExportacionSaldo(_ context.Context, _ ports.OrdenExportacionSaldo, e ports.EvidenciaExportacionSaldo) (ports.ConfirmacionExportacionSaldo, error) {
	r.llamadas++
	if r.alConfirmar != nil {
		r.alConfirmar()
	}
	c := ports.ConfirmacionExportacionSaldo{Evidencia: e, ReciboRef: "recibo_sintetico", AuditoriaRef: "auditoria_sintetica", ConfirmadaUTC: r.reloj.t}
	if r.modificar != nil {
		r.modificar(&c)
	}
	return c, r.err
}

type relojExportacionPrueba struct{ t time.Time }

func (r *relojExportacionPrueba) AhoraUTC() time.Time { return r.t }

type preparadorExportacionPrueba struct {
	err        error
	alPreparar func()
}

func (p preparadorExportacionPrueba) PrepararInformeSaldo(context.Context, ports.SaldoExportable) (ports.DocumentoSaldoPreparado, error) {
	if p.alPreparar != nil {
		p.alPreparar()
	}
	return ports.DocumentoSaldoPreparado{Contenido: []byte("%PDF-prueba"), CatalogoRef: "catalogo_sintetico", CatalogoVersion: 1, CatalogoSHA256: strings.Repeat("a", 64)}, p.err
}

func prepararExportacionPrueba(t *testing.T) (*ServicioExportacionSaldo, ports.OrdenExportacionSaldo, *fuenteExportacionPrueba, *registroExportacionPrueba) {
	t.Helper()
	actor, err := contexto(t).OrdenConsumo.ContextoActor()
	if err != nil {
		t.Fatal(err)
	}
	orden, err := ports.NuevaOrdenExportacionSaldo(actor)
	if err != nil {
		t.Fatal(err)
	}
	refs, err := actor.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	if err != nil {
		t.Fatal(err)
	}
	reloj := &relojExportacionPrueba{time.Now().UTC().Truncate(time.Microsecond)}
	previsto, saldo := int64(450), int64(-30)
	fuente := &fuenteExportacionPrueba{saldo: ports.SaldoExportable{EmpleadoRef: refs[0], Periodo: ports.PeriodoConsultaSaldo{Tipo: ports.PeriodoSaldoRango, Desde: "2026-09-21", Hasta: "2026-09-21"}, Resumen: ports.ResumenConsultaSaldo{PrevistosMinutos: &previsto, TrabajadosMinutos: 420, SaldoMinutos: &saldo, Estado: ports.EstadoSaldoDisponible}, FuenteRef: "fuente_sintetica", FuenteVersion: 1}}
	registro := &registroExportacionPrueba{reloj: reloj}
	s, err := NuevaExportacionSaldo(fuente, preparadorExportacionPrueba{}, registro, reloj, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	return s, orden, fuente, registro
}

func exportarPrueba(ctx context.Context, s *ServicioExportacionSaldo, o ports.OrdenExportacionSaldo) (ports.ResultadoExportacionSaldo, error) {
	return s.ExportarSaldoPropio(ctx, o, ports.PeriodoSaldoRango, "2026-09-21", "2026-09-21")
}

func TestExportacionSaldoExigeFuenteYAuditoriaAntesDeBytes(t *testing.T) {
	s, o, f, r := prepararExportacionPrueba(t)
	catalogo, err := os.ReadFile("../../../../web/static/textos/es/cronos-informe-saldo.json")
	if err != nil {
		t.Fatal(err)
	}
	s.preparador, err = informesaldo.Nuevo(pdf.Renderizador{}, bytes.NewReader(catalogo))
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := exportarPrueba(context.Background(), s, o)
	if err != nil || !bytes.HasPrefix(resultado.Contenido, []byte("%PDF-")) || f.llamadas != 1 || r.llamadas != 1 || resultado.Confirmacion.Evidencia.Tamano != int64(len(resultado.Contenido)) {
		t.Fatalf("flujo nominal: bytes=%d lectura=%d confirmacion=%d err=%v", len(resultado.Contenido), f.llamadas, r.llamadas, err)
	}
	if resultado.Confirmacion.Evidencia.DocumentoSHA256 == "" || resultado.Confirmacion.Evidencia.CatalogoSHA256 == "" {
		t.Fatal("evidencia incompleta")
	}
}

func TestExportacionSaldoFallaCerrada(t *testing.T) {
	fallo := errors.New("fallo_sintetico")
	casos := []struct {
		nombre         string
		alterar        func(*ServicioExportacionSaldo, *fuenteExportacionPrueba, *registroExportacionPrueba)
		confirmaciones int
	}{
		{"fuente_denegada", func(_ *ServicioExportacionSaldo, f *fuenteExportacionPrueba, _ *registroExportacionPrueba) {
			f.err = fallo
		}, 0},
		{"otra_persona", func(_ *ServicioExportacionSaldo, f *fuenteExportacionPrueba, _ *registroExportacionPrueba) {
			f.saldo.EmpleadoRef = "emp_ajeno"
		}, 0},
		{"otro_periodo", func(_ *ServicioExportacionSaldo, f *fuenteExportacionPrueba, _ *registroExportacionPrueba) {
			f.saldo.Periodo.Hasta = "2026-09-22"
		}, 0},
		{"sin_version", func(_ *ServicioExportacionSaldo, f *fuenteExportacionPrueba, _ *registroExportacionPrueba) {
			f.saldo.FuenteVersion = 0
		}, 0},
		{"pdf_falla", func(s *ServicioExportacionSaldo, _ *fuenteExportacionPrueba, _ *registroExportacionPrueba) {
			s.preparador = preparadorExportacionPrueba{err: fallo}
		}, 0},
		{"auditoria_falla", func(_ *ServicioExportacionSaldo, _ *fuenteExportacionPrueba, r *registroExportacionPrueba) {
			r.err = fallo
		}, 1},
		{"recibo_sin_auditoria", func(_ *ServicioExportacionSaldo, _ *fuenteExportacionPrueba, r *registroExportacionPrueba) {
			r.modificar = func(c *ports.ConfirmacionExportacionSaldo) { c.AuditoriaRef = "" }
		}, 1},
		{"recibo_otro_pdf", func(_ *ServicioExportacionSaldo, _ *fuenteExportacionPrueba, r *registroExportacionPrueba) {
			r.modificar = func(c *ports.ConfirmacionExportacionSaldo) { c.Evidencia.DocumentoSHA256 = strings.Repeat("b", 64) }
		}, 1},
		{"recibo_otro_periodo", func(_ *ServicioExportacionSaldo, _ *fuenteExportacionPrueba, r *registroExportacionPrueba) {
			r.modificar = func(c *ports.ConfirmacionExportacionSaldo) { c.Evidencia.Periodo.Hasta = "2026-09-22" }
		}, 1},
		{"recibo_otro_catalogo", func(_ *ServicioExportacionSaldo, _ *fuenteExportacionPrueba, r *registroExportacionPrueba) {
			r.modificar = func(c *ports.ConfirmacionExportacionSaldo) { c.Evidencia.CatalogoVersion++ }
		}, 1},
		{"recibo_anterior", func(_ *ServicioExportacionSaldo, _ *fuenteExportacionPrueba, r *registroExportacionPrueba) {
			r.modificar = func(c *ports.ConfirmacionExportacionSaldo) { c.ConfirmadaUTC = c.ConfirmadaUTC.Add(-time.Second) }
		}, 1},
		{"vigencia_agotada", func(s *ServicioExportacionSaldo, _ *fuenteExportacionPrueba, r *registroExportacionPrueba) {
			s.preparador = preparadorExportacionPrueba{alPreparar: func() { r.reloj.t = r.reloj.t.Add(24 * time.Hour) }}
		}, 0},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			s, o, f, r := prepararExportacionPrueba(t)
			c.alterar(s, f, r)
			resultado, err := exportarPrueba(context.Background(), s, o)
			if err == nil || len(resultado.Contenido) != 0 || r.llamadas != c.confirmaciones {
				t.Fatalf("fallo abierto bytes=%d confirmaciones=%d err=%v", len(resultado.Contenido), r.llamadas, err)
			}
		})
	}
}

func TestExportacionSaldoCanceladaSiempreSinBytes(t *testing.T) {
	for _, fase := range []string{"inicio", "fuente", "render", "confirmacion"} {
		t.Run(fase, func(t *testing.T) {
			s, o, f, r := prepararExportacionPrueba(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch fase {
			case "inicio":
				cancel()
			case "fuente":
				f.alLeer = cancel
			case "render":
				s.preparador = preparadorExportacionPrueba{alPreparar: cancel}
			case "confirmacion":
				r.alConfirmar = cancel
			}
			resultado, err := exportarPrueba(ctx, s, o)
			if !errors.Is(err, context.Canceled) || len(resultado.Contenido) != 0 {
				t.Fatalf("bytes=%d err=%v", len(resultado.Contenido), err)
			}
		})
	}
}

func TestExportacionSaldoSinAutoridadesNiOrdenNoLee(t *testing.T) {
	s, o, f, r := prepararExportacionPrueba(t)
	var fuente *fuenteExportacionPrueba
	if _, err := NuevaExportacionSaldo(fuente, s.preparador, r, s.reloj, time.UTC); !errors.Is(err, ports.ErrExportacionSaldoNoDisponible) {
		t.Fatal(err)
	}
	if _, err := NuevaExportacionSaldo(f, s.preparador, nil, s.reloj, time.UTC); !errors.Is(err, ports.ErrExportacionSaldoNoDisponible) {
		t.Fatal(err)
	}
	resultado, err := exportarPrueba(context.Background(), s, ports.OrdenExportacionSaldo{})
	if err == nil || len(resultado.Contenido) != 0 || f.llamadas != 0 {
		t.Fatalf("orden vacia: bytes=%d lecturas=%d err=%v", len(resultado.Contenido), f.llamadas, err)
	}
	var cero ServicioExportacionSaldo
	resultado, err = exportarPrueba(context.Background(), &cero, o)
	if err == nil || len(resultado.Contenido) != 0 {
		t.Fatal("servicio cero concedio salida")
	}
}
