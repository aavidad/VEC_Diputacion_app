package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type fuenteExportacionPrueba struct {
	captura ports.CapturaParaExportacionAuditoria
	err     error
	veces   int
}

func (f *fuenteExportacionPrueba) CapturarAuditoriaParaExportacion(context.Context) (ports.CapturaParaExportacionAuditoria, error) {
	f.veces++
	return f.captura, f.err
}

type firmadorExportacionPrueba struct{ veces int }

func (*firmadorExportacionPrueba) PinExportacionAuditoria() string { return strings.Repeat("a", 64) }
func (f *firmadorExportacionPrueba) FirmarExportacionAuditoria(_ context.Context, r domain.ReciboExportacionAuditoriaDesarrollo) (domain.ReciboExportacionAuditoriaDesarrollo, error) {
	f.veces++
	r.FirmaBase64 = base64.StdEncoding.EncodeToString(make([]byte, 64))
	return r, nil
}

type selladorExportacionPrueba struct{ veces int }

func (s *selladorExportacionPrueba) SellarExportacionAuditoria(_ context.Context, m domain.ManifiestoExportacionAuditoriaDesarrollo) (domain.ReciboTSACheckpoint, error) {
	s.veces++
	b, err := m.Canonico()
	if err != nil {
		return domain.ReciboTSACheckpoint{}, err
	}
	return domain.ReciboTSACheckpoint{
		Referencia: "tsa-desarrollo:hmac-sha256:" + strings.Repeat("b", 64), HuellaPreimagenSHA256: strings.Repeat("c", 64),
		HuellaCheckpointSHA256: domain.HuellaCheckpoint(b), Autoridad: "no_autoritativo", Esquema: "vec.tsa.desarrollo.v1",
	}, nil
}

type verificadorExportacionPrueba struct {
	veces int
	err   error
}

func (v *verificadorExportacionPrueba) VerificarExportacionAuditoria(context.Context, domain.ReciboExportacionAuditoriaDesarrollo) error {
	v.veces++
	return v.err
}

func datosExportacionPrueba(t *testing.T) (ports.CapturaParaExportacionAuditoria, domain.PoliticaCheckpoint) {
	t.Helper()
	b, err := os.ReadFile("../../../cmd/vec-auditoria-verificar/testdata/bootstrap_intentos_ad179.json")
	if err != nil {
		t.Fatal(err)
	}
	var cabecera struct {
		Manifiesto domain.CoberturaCheckpoint `json:"manifiesto"`
	}
	if err := json.Unmarshal(b, &cabecera); err != nil {
		t.Fatal(err)
	}
	return ports.CapturaParaExportacionAuditoria{
			Captura: domain.CapturaExportacionAuditoria{
				Referencia: "captura:prueba", AuditoriaRef: "auditoria:prueba", AuditoriaSHA256: strings.Repeat("d", 64),
				CapturadaEn: "2026-10-04T00:00:00.000000Z",
			},
			Cobertura: cabecera.Manifiesto, Documento: b,
		}, domain.PoliticaCheckpoint{
			Version: 1, PoliticaRef: "politica:prueba", PoliticaVersion: 1, ClaveRef: "clave:prueba", ClaveVersion: 1,
			ProveedorKMS: "kms:prueba", ProveedorKMSVersion: 1, ProveedorTSA: "tsa:prueba", ProveedorTSAVersion: 1,
			OperacionTSA: "operacion:prueba", Modo: "DESARROLLO",
		}
}

func TestExportacionAuditoriaEmiteYVerificaSinAcreditarOrigen(t *testing.T) {
	captura, politica := datosExportacionPrueba(t)
	fuente := &fuenteExportacionPrueba{captura: captura}
	firma, sello := &firmadorExportacionPrueba{}, &selladorExportacionPrueba{}
	recibo, documento, err := EmitirExportacionAuditoriaDesarrollo(context.Background(), fuente, politica, firma, sello, 1<<20, 100)
	if err != nil || fuente.veces != 1 || firma.veces != 1 || sello.veces != 1 || string(documento) != string(captura.Documento) ||
		recibo.Manifiesto.Documento.SHA256 != domain.HuellaCheckpoint(documento) {
		t.Fatalf("emisión: err=%v fuente=%d firma=%d sello=%d", err, fuente.veces, firma.veces, sello.veces)
	}
	v := &verificadorExportacionPrueba{}
	informe := VerificarExportacionAuditoriaDesarrollo(context.Background(), recibo, documento, v, 1<<20, 100)
	if informe.Estado != "verificada" || informe.Firma != "verificada_con_pin_externo" || informe.IntegridadArchivo != "verificada" ||
		informe.IntegridadCadena != "verificada" || informe.OrigenExtraccion != "no_acreditado" || informe.TSA != "no_verificada_offline" ||
		informe.TiempoIndependiente || informe.FirmaLegal || v.veces != 1 {
		t.Fatalf("informe: %+v; verificador=%d", informe, v.veces)
	}
	alterado := append([]byte(nil), documento...)
	alterado[len(alterado)-2] ^= 1
	informe = VerificarExportacionAuditoriaDesarrollo(context.Background(), recibo, alterado, v, 1<<20, 100)
	if informe.Estado != "rechazada" || informe.IntegridadArchivo != "rechazada" || v.veces != 2 {
		t.Fatalf("archivo manipulado: %+v", informe)
	}
	recibo.Manifiesto.HistoricosSinFechaLigada = !recibo.Manifiesto.HistoricosSinFechaLigada
	manifiesto, err := recibo.Manifiesto.Canonico()
	if err != nil {
		t.Fatal(err)
	}
	recibo.TSA.HuellaCheckpointSHA256 = domain.HuellaCheckpoint(manifiesto)
	informe = VerificarExportacionAuditoriaDesarrollo(context.Background(), recibo, documento, v, 1<<20, 100)
	if informe.Estado != "rechazada" || informe.IntegridadCadena != "rechazada" {
		t.Fatalf("aviso manipulado: %+v", informe)
	}
}

func TestExportacionAuditoriaRechazaFuenteInvalidaSinProveedores(t *testing.T) {
	captura, politica := datosExportacionPrueba(t)
	for _, caso := range []struct {
		nombre string
		fuente *fuenteExportacionPrueba
		ctx    context.Context
	}{
		{"nil", nil, context.Background()},
		{"error", &fuenteExportacionPrueba{captura: captura, err: errors.New("material privado")}, context.Background()},
		{"captura", &fuenteExportacionPrueba{captura: ports.CapturaParaExportacionAuditoria{Documento: captura.Documento, Cobertura: captura.Cobertura}}, context.Background()},
		{"documento", &fuenteExportacionPrueba{captura: ports.CapturaParaExportacionAuditoria{Captura: captura.Captura, Cobertura: captura.Cobertura, Documento: []byte(`{}`)}}, context.Background()},
		{"cancelado", &fuenteExportacionPrueba{captura: captura}, contextoExportacionCancelado()},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			firma, sello := &firmadorExportacionPrueba{}, &selladorExportacionPrueba{}
			recibo, documento, err := EmitirExportacionAuditoriaDesarrollo(caso.ctx, caso.fuente, politica, firma, sello, 1<<20, 100)
			if !errors.Is(err, domain.ErrCheckpointInvalido) || documento != nil || recibo.FirmaBase64 != "" || firma.veces != 0 || sello.veces != 0 {
				t.Fatalf("proveedor invocado o datos filtrados: err=%v firma=%d sello=%d", err, firma.veces, sello.veces)
			}
		})
	}
}

func contextoExportacionCancelado() context.Context {
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	return ctx
}

func TestExportacionAuditoriaFirmaAntesDeDocumento(t *testing.T) {
	captura, politica := datosExportacionPrueba(t)
	recibo, documento, err := EmitirExportacionAuditoriaDesarrollo(context.Background(), &fuenteExportacionPrueba{captura: captura},
		politica, &firmadorExportacionPrueba{}, &selladorExportacionPrueba{}, 1<<20, 100)
	if err != nil {
		t.Fatal(err)
	}
	v := &verificadorExportacionPrueba{err: errors.New("firma no válida")}
	informe := VerificarExportacionAuditoriaDesarrollo(context.Background(), recibo, []byte("contenido arbitrario"), v, 1<<20, 100)
	if informe.Estado != "rechazada" || informe.Firma != "rechazada" || informe.IntegridadArchivo != "no_evaluada" || v.veces != 1 {
		t.Fatalf("orden de verificación: %+v", informe)
	}
	informe = VerificarExportacionAuditoriaDesarrollo(context.Background(), recibo, documento, nil, 1<<20, 100)
	if informe.Estado != "rechazada" || v.veces != 1 {
		t.Fatalf("verificador ausente: %+v", informe)
	}
}

func TestExportacionAuditoriaErrorConservaCausaSinExponerMensaje(t *testing.T) {
	captura, politica := datosExportacionPrueba(t)
	causa := errors.New("detalle_privado_sintetico")
	fuente := &fuenteExportacionPrueba{captura: captura, err: causa}
	r, b, err := EmitirExportacionAuditoriaDesarrollo(context.Background(), fuente, politica, &firmadorExportacionPrueba{}, &selladorExportacionPrueba{}, 1<<20, 100)
	if !errors.Is(err, causa) || !errors.Is(err, domain.ErrCheckpointInvalido) || err.Error() != domain.ErrCheckpointInvalido.Error() || b != nil || r.FirmaBase64 != "" {
		t.Fatal("causa perdida o mensaje expuesto")
	}
}
