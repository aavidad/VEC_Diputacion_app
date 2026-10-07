package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// Dobles exclusivos del ensayo unitario: no son fuente o proveedores reales.
type fuentePeriodicaPrueba struct {
	captura                  ports.CapturaCheckpointPeriodico
	capturas, confirmaciones int
	errCaptura, errConfirmar error
	alterar                  func(*ports.AcuseCheckpointPeriodico)
}

func (f *fuentePeriodicaPrueba) CapturarCheckpointPendiente(context.Context) (ports.CapturaCheckpointPeriodico, error) {
	f.capturas++
	return f.captura, f.errCaptura
}
func (f *fuentePeriodicaPrueba) ConfirmarCheckpoint(_ context.Context, ref string, r domain.ReciboCheckpointDesarrollo) (ports.AcuseCheckpointPeriodico, error) {
	f.confirmaciones++
	raw, _ := json.Marshal(r)
	a := f.captura.Acuse
	a.Secuencia++
	a.HuellaSHA256 = strings.Repeat("e", 64)
	a.CapturaRef, a.ReciboHuellaSHA256 = ref, domain.HuellaCheckpoint(raw)
	if f.alterar != nil {
		f.alterar(&a)
	}
	return a, f.errConfirmar
}

type proveedorPeriodicoPrueba struct {
	firmas, sellos int
	err            error
	alterar        bool
}

func (*proveedorPeriodicoPrueba) PinCheckpoint() string { return strings.Repeat("b", 64) }
func (p *proveedorPeriodicoPrueba) FirmarCheckpoint(_ context.Context, r domain.ReciboCheckpointDesarrollo) (domain.ReciboCheckpointDesarrollo, error) {
	p.firmas++
	r.FirmaBase64 = "Zml4dHVyYQ=="
	if p.alterar {
		r.Checkpoint.Politica.ClaveVersion++
	}
	return r, p.err
}
func (p *proveedorPeriodicoPrueba) SellarCheckpoint(_ context.Context, c domain.CheckpointDesarrollo) (domain.ReciboTSACheckpoint, error) {
	p.sellos++
	b, _ := c.Canonico()
	return domain.ReciboTSACheckpoint{Referencia: "tsa-desarrollo:hmac-sha256:" + strings.Repeat("c", 64),
		HuellaPreimagenSHA256: strings.Repeat("a", 64), HuellaCheckpointSHA256: domain.HuellaCheckpoint(b),
		Autoridad: "no_autoritativo", Esquema: "vec.tsa.desarrollo.v1"}, p.err
}

func capturaPeriodicaPrueba() ports.CapturaCheckpointPeriodico {
	return ports.CapturaCheckpointPeriodico{Estado: "pendiente", CapturaRef: "captura:desarrollo:1",
		ConfiguracionVersion: 1, ConfiguracionSHA256: strings.Repeat("a", 64), PinSPKISHA256: strings.Repeat("b", 64),
		Checkpoint: domain.CheckpointDesarrollo{Esquema: domain.EsquemaCheckpointDesarrollo,
			Politica: domain.PoliticaCheckpoint{Version: 1, PoliticaRef: "politica:desarrollo", PoliticaVersion: 1,
				ClaveRef: "clave:desarrollo", ClaveVersion: 1, ProveedorKMS: "kms:desarrollo", ProveedorKMSVersion: 1,
				ProveedorTSA: "tsa:desarrollo", ProveedorTSAVersion: 1, OperacionTSA: "operacion:tsa", Modo: "DESARROLLO"},
			Cobertura: domain.CoberturaCheckpoint{CadenaID: "cadena:desarrollo", PrimeraSecuencia: 1, UltimaSecuencia: 2,
				Registros: 2, AnteriorSHA256: strings.Repeat("0", 64), CabezaSHA256: strings.Repeat("d", 64)}},
		Acuse: ports.AcuseCheckpointPeriodico{AuditoriaRef: "auditoria:desarrollo:1", Secuencia: 2,
			HuellaSHA256: strings.Repeat("d", 64), CorrelacionRef: "correlacion:desarrollo:1",
			RegistradaEn: time.Date(2026, 10, 4, 5, 0, 0, 0, time.UTC)}}
}

func TestCheckpointPeriodicoCapturaUnaVezYConfirmaReciboExacto(t *testing.T) {
	f := &fuentePeriodicaPrueba{captura: capturaPeriodicaPrueba()}
	p := &proveedorPeriodicoPrueba{}
	c, err := CapturarCheckpointPeriodico(context.Background(), f, 10)
	if err != nil {
		t.Fatal(err)
	}
	r, err := SellarConfirmarCheckpointPeriodico(context.Background(), f, c, p, p, 10)
	if err != nil || r.Estado != "confirmado" || r.Captura != c || r.Acuse.CapturaRef != c.CapturaRef ||
		f.capturas != 1 || f.confirmaciones != 1 || p.firmas != 1 || p.sellos != 1 {
		t.Fatalf("resultado o número de operaciones inesperado: %v", err)
	}
	if fmt.Sprintf("%+v %#v", r, r) != "[RESULTADO-CHECKPOINT-PERIODICO] application.ResultadoCheckpointPeriodico{[OPACO]}" {
		t.Fatal("resultado sin redacción")
	}
}

func TestCheckpointPeriodicoNoVencidoRequiereAuditoriaSinFirmar(t *testing.T) {
	c := ports.CapturaCheckpointPeriodico{Estado: "no_vencido", Acuse: capturaPeriodicaPrueba().Acuse}
	f := &fuentePeriodicaPrueba{captura: c}
	c, err := CapturarCheckpointPeriodico(context.Background(), f, 10)
	if err != nil {
		t.Fatal(err)
	}
	r, err := SellarConfirmarCheckpointPeriodico(context.Background(), f, c, nil, nil, 10)
	if err != nil || r.Estado != "no_vencido" || r.Acuse != c.Acuse || r.Recibo != (domain.ReciboCheckpointDesarrollo{}) || f.confirmaciones != 0 {
		t.Fatal("no vencido hizo efectos o perdió auditoría")
	}
	f.captura.Acuse = ports.AcuseCheckpointPeriodico{}
	if _, err := CapturarCheckpointPeriodico(context.Background(), f, 10); err == nil {
		t.Fatal("sin auditoría aceptado")
	}
}

func TestCheckpointPeriodicoRechazaCapturasSinCoberturaAuditada(t *testing.T) {
	for nombre, alterar := range map[string]func(*ports.CapturaCheckpointPeriodico){
		"estado":                func(c *ports.CapturaCheckpointPeriodico) { c.Estado = "confirmado" },
		"version":               func(c *ports.CapturaCheckpointPeriodico) { c.ConfiguracionVersion = 0 },
		"configuracion":         func(c *ports.CapturaCheckpointPeriodico) { c.ConfiguracionSHA256 = strings.Repeat("0", 64) },
		"acuse fuera cobertura": func(c *ports.CapturaCheckpointPeriodico) { c.Acuse.Secuencia++ },
		"huella ajena":          func(c *ports.CapturaCheckpointPeriodico) { c.Acuse.HuellaSHA256 = strings.Repeat("e", 64) },
		"acuse previo":          func(c *ports.CapturaCheckpointPeriodico) { c.Acuse.CapturaRef = c.CapturaRef },
		"instante": func(c *ports.CapturaCheckpointPeriodico) {
			c.Acuse.RegistradaEn = c.Acuse.RegistradaEn.Add(time.Nanosecond)
		},
		"limite":                 func(c *ports.CapturaCheckpointPeriodico) { c.Checkpoint.Cobertura.Registros = 11 },
		"no vencido con captura": func(c *ports.CapturaCheckpointPeriodico) { c.Estado = "no_vencido" },
	} {
		t.Run(nombre, func(t *testing.T) {
			c := capturaPeriodicaPrueba()
			alterar(&c)
			f := &fuentePeriodicaPrueba{captura: c}
			p := &proveedorPeriodicoPrueba{}
			if _, err := CapturarCheckpointPeriodico(context.Background(), f, 10); err == nil {
				t.Fatal("captura inválida aceptada")
			}
			if _, err := SellarConfirmarCheckpointPeriodico(context.Background(), f, c, p, p, 10); err == nil {
				t.Fatal("captura inválida firmada")
			}
			if p.sellos != 0 || p.firmas != 0 || f.confirmaciones != 0 {
				t.Fatal("efecto tras rechazo")
			}
		})
	}
}

func TestCheckpointPeriodicoNoDevuelveConfirmadoAnteFallo(t *testing.T) {
	causa := errors.New("proveedor: secreto sintético")
	for _, etapa := range []string{"captura", "sello", "firma alterada", "confirmacion", "ref ajena", "recibo ajeno", "secuencia antigua"} {
		t.Run(etapa, func(t *testing.T) {
			f := &fuentePeriodicaPrueba{captura: capturaPeriodicaPrueba()}
			p := &proveedorPeriodicoPrueba{}
			switch etapa {
			case "captura":
				f.errCaptura = causa
			case "sello":
				p.err = causa
			case "firma alterada":
				p.alterar = true
			case "confirmacion":
				f.errConfirmar = causa
			case "ref ajena":
				f.alterar = func(a *ports.AcuseCheckpointPeriodico) { a.CapturaRef = "captura:ajena:1" }
			case "recibo ajeno":
				f.alterar = func(a *ports.AcuseCheckpointPeriodico) { a.ReciboHuellaSHA256 = strings.Repeat("f", 64) }
			case "secuencia antigua":
				f.alterar = func(a *ports.AcuseCheckpointPeriodico) { a.Secuencia = f.captura.Acuse.Secuencia }
			}
			c, err := CapturarCheckpointPeriodico(context.Background(), f, 10)
			var r ResultadoCheckpointPeriodico
			if err == nil {
				r, err = SellarConfirmarCheckpointPeriodico(context.Background(), f, c, p, p, 10)
			}
			if err == nil || r != (ResultadoCheckpointPeriodico{}) || f.capturas != 1 || f.confirmaciones > 1 {
				t.Fatal("éxito ficticio o reintento")
			}
			if strings.Contains(err.Error(), "secreto") {
				t.Fatal("causa visible")
			}
			if etapa == "captura" || etapa == "sello" || etapa == "confirmacion" {
				if !errors.Is(err, causa) || !errors.Is(err, ErrCheckpointPeriodicoNoDisponible) {
					t.Fatal("causa nominal perdida")
				}
			}
		})
	}
}

// Desde AD207 la captura cubre la cabeza sellada, por debajo de su asiento.
func TestCheckpointPeriodicoAdmiteCabezaSelladaAD207(t *testing.T) {
	c := capturaPeriodicaPrueba()
	c.Acuse.Secuencia = 5
	c.Acuse.HuellaSHA256 = strings.Repeat("9", 64)
	f := &fuentePeriodicaPrueba{captura: c}
	p := &proveedorPeriodicoPrueba{}
	if _, err := CapturarCheckpointPeriodico(context.Background(), f, 10); err != nil {
		t.Fatalf("cabeza sellada rechazada: %v", err)
	}
	if r, err := SellarConfirmarCheckpointPeriodico(context.Background(), f, c, p, p, 10); err != nil || r.Estado != "confirmado" {
		t.Fatalf("confirmación de cabeza sellada: %v", err)
	}
	c.Checkpoint.Cobertura.UltimaSecuencia, c.Checkpoint.Cobertura.Registros = 6, 6
	if _, err := CapturarCheckpointPeriodico(context.Background(), &fuentePeriodicaPrueba{captura: c}, 10); err == nil {
		t.Fatal("cobertura por encima del asiento de la captura aceptada")
	}
}
