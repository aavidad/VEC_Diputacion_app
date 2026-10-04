package application

import (
	"context"
	"encoding/base64"
	"reflect"

	"vec-diputacion-granada/internal/vec/auditoria"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const maxBytesExportacionAuditoriaDesarrollo int64 = 64 * 1024 * 1024

// EmitirExportacionAuditoriaDesarrollo sólo consume una fuente ligada por
// composición a la captura autorizada y confirmada. No acredita el origen del
// archivo ni convierte el sello de desarrollo en tiempo independiente.
func EmitirExportacionAuditoriaDesarrollo(ctx context.Context, fuente ports.FuenteCapturaExportacionAuditoria,
	p domain.PoliticaCheckpoint, f ports.FirmadorExportacionAuditoriaDesarrollo,
	t ports.SelladorExportacionAuditoriaDesarrollo, maxBytes int64, maxRegistros uint64,
) (domain.ReciboExportacionAuditoriaDesarrollo, []byte, error) {
	fallo := domain.ErrCheckpointInvalido
	if ctx == nil || ctx.Err() != nil || dependenciaExportacionNula(fuente) || dependenciaExportacionNula(f) ||
		dependenciaExportacionNula(t) || p.Validar() != nil || !limitesExportacionValidos(maxBytes, maxRegistros) {
		return domain.ReciboExportacionAuditoriaDesarrollo{}, nil, fallo
	}
	captura, err := fuente.CapturarAuditoriaParaExportacion(ctx)
	if err != nil {
		return domain.ReciboExportacionAuditoriaDesarrollo{}, nil, errorExportacionAuditoria{causa: err}
	}
	if ctx.Err() != nil || captura.Captura.Validar() != nil || captura.Cobertura.Validar() != nil ||
		len(captura.Documento) == 0 || int64(len(captura.Documento)) > maxBytes || captura.Cobertura.Registros > maxRegistros {
		return domain.ReciboExportacionAuditoriaDesarrollo{}, nil, fallo
	}
	documento := append([]byte(nil), captura.Documento...)
	esquema, informe, err := auditoria.VerificarDocumentoExportacionAuditoria(documento, captura.Cobertura, maxBytes, maxRegistros)
	if err != nil {
		return domain.ReciboExportacionAuditoriaDesarrollo{}, nil, errorExportacionAuditoria{causa: err}
	}
	if informe.Estado != "verificada" || ctx.Err() != nil {
		return domain.ReciboExportacionAuditoriaDesarrollo{}, nil, fallo
	}
	manifiesto := domain.ManifiestoExportacionAuditoriaDesarrollo{
		Esquema: domain.EsquemaExportacionAuditoriaDesarrollo, Politica: p,
		Captura: captura.Captura, Cobertura: captura.Cobertura,
		Documento: domain.DocumentoExportacionAuditoria{Esquema: esquema, Bytes: int64(len(documento)),
			SHA256: domain.HuellaCheckpoint(documento)},
		HistoricosSinFechaLigada: informe.ConsumosHistoricosSinFechaLigada,
	}
	if _, err := manifiesto.Canonico(); err != nil {
		return domain.ReciboExportacionAuditoriaDesarrollo{}, nil, errorExportacionAuditoria{causa: err}
	}
	pin := f.PinExportacionAuditoria()
	if !domain.SHA256CheckpointValido(pin) || ctx.Err() != nil {
		return domain.ReciboExportacionAuditoriaDesarrollo{}, nil, fallo
	}
	sello, err := t.SellarExportacionAuditoria(ctx, manifiesto)
	if err != nil {
		return domain.ReciboExportacionAuditoriaDesarrollo{}, nil, errorExportacionAuditoria{causa: err}
	}
	if ctx.Err() != nil {
		return domain.ReciboExportacionAuditoriaDesarrollo{}, nil, fallo
	}
	recibo := domain.ReciboExportacionAuditoriaDesarrollo{Manifiesto: manifiesto, TSA: sello, PinSPKISHA256: pin}
	preimagen, err := recibo.CanonicoParaFirma()
	if err != nil {
		return domain.ReciboExportacionAuditoriaDesarrollo{}, nil, errorExportacionAuditoria{causa: err}
	}
	if ctx.Err() != nil {
		return domain.ReciboExportacionAuditoriaDesarrollo{}, nil, fallo
	}
	firmado, err := f.FirmarExportacionAuditoria(ctx, recibo)
	if err != nil {
		return domain.ReciboExportacionAuditoriaDesarrollo{}, nil, errorExportacionAuditoria{causa: err}
	}
	if ctx.Err() != nil {
		return domain.ReciboExportacionAuditoriaDesarrollo{}, nil, fallo
	}
	confirmada, err := firmado.CanonicoParaFirma()
	if err != nil {
		return domain.ReciboExportacionAuditoriaDesarrollo{}, nil, errorExportacionAuditoria{causa: err}
	}
	if string(confirmada) != string(preimagen) || !firmaExportacionCanonica(firmado.FirmaBase64) {
		return domain.ReciboExportacionAuditoriaDesarrollo{}, nil, fallo
	}
	return firmado, documento, nil
}

type InformeExportacionAuditoriaDesarrollo struct {
	Esquema                  string `json:"esquema"`
	Modo                     string `json:"modo"`
	Firma                    string `json:"firma"`
	IntegridadArchivo        string `json:"integridad_archivo"`
	IntegridadCadena         string `json:"integridad_cadena"`
	OrigenExtraccion         string `json:"origen_extraccion"`
	TSA                      string `json:"tsa"`
	TiempoIndependiente      bool   `json:"tiempo_independiente"`
	FirmaLegal               bool   `json:"firma_legal"`
	HistoricosSinFechaLigada bool   `json:"historicos_sin_fecha_ligada"`
	Estado                   string `json:"estado"`
}

func informeExportacionRechazada() InformeExportacionAuditoriaDesarrollo {
	return InformeExportacionAuditoriaDesarrollo{
		Esquema: domain.EsquemaExportacionAuditoriaDesarrollo, Modo: "DESARROLLO", Firma: "rechazada",
		IntegridadArchivo: "no_evaluada", IntegridadCadena: "no_evaluada", OrigenExtraccion: "no_acreditado",
		TSA: "no_verificada_offline", Estado: "rechazada",
	}
}

// VerificarExportacionAuditoriaDesarrollo verifica primero la firma con el
// pin externo del proveedor; sólo entonces inspecciona el archivo recibido.
func VerificarExportacionAuditoriaDesarrollo(ctx context.Context, r domain.ReciboExportacionAuditoriaDesarrollo,
	documento []byte, v ports.VerificadorExportacionAuditoriaDesarrollo, maxBytes int64, maxRegistros uint64,
) InformeExportacionAuditoriaDesarrollo {
	o := informeExportacionRechazada()
	if ctx == nil || ctx.Err() != nil || dependenciaExportacionNula(v) || !limitesExportacionValidos(maxBytes, maxRegistros) ||
		r.Manifiesto.Documento.Bytes > maxBytes || r.Manifiesto.Cobertura.Registros > maxRegistros ||
		!firmaExportacionCanonica(r.FirmaBase64) {
		return o
	}
	if _, err := r.CanonicoParaFirma(); err != nil {
		return rechazarExportacionPorError(o, err, "firma")
	}
	if err := v.VerificarExportacionAuditoria(ctx, r); err != nil {
		return rechazarExportacionPorError(o, err, "firma")
	}
	if ctx.Err() != nil {
		return o
	}
	o.Firma = "verificada_con_pin_externo"
	if int64(len(documento)) != r.Manifiesto.Documento.Bytes || int64(len(documento)) > maxBytes ||
		domain.HuellaCheckpoint(documento) != r.Manifiesto.Documento.SHA256 {
		o.IntegridadArchivo = "rechazada"
		return o
	}
	o.IntegridadArchivo = "verificada"
	esquema, informe, err := auditoria.VerificarDocumentoExportacionAuditoria(documento, r.Manifiesto.Cobertura, maxBytes, maxRegistros)
	if err != nil {
		return rechazarExportacionPorError(o, err, "cadena")
	}
	if informe.Estado != "verificada" || esquema != r.Manifiesto.Documento.Esquema ||
		informe.ConsumosHistoricosSinFechaLigada != r.Manifiesto.HistoricosSinFechaLigada || ctx.Err() != nil {
		o.IntegridadCadena = "rechazada"
		return o
	}
	o.IntegridadCadena = "verificada"
	o.HistoricosSinFechaLigada = informe.ConsumosHistoricosSinFechaLigada
	o.Estado = "verificada"
	return o
}

func rechazarExportacionPorError(o InformeExportacionAuditoriaDesarrollo, err error, fase string) InformeExportacionAuditoriaDesarrollo {
	if err != nil && fase == "cadena" {
		o.IntegridadCadena = "rechazada"
	}
	return o
}

func limitesExportacionValidos(maxBytes int64, maxRegistros uint64) bool {
	return maxBytes > 0 && maxBytes <= maxBytesExportacionAuditoriaDesarrollo && maxRegistros > 0
}

func firmaExportacionCanonica(valor string) bool {
	firma, err := base64.StdEncoding.DecodeString(valor)
	return err == nil && len(firma) == 64 && base64.StdEncoding.EncodeToString(firma) == valor
}

func dependenciaExportacionNula(valor any) bool {
	if valor == nil {
		return true
	}
	v := reflect.ValueOf(valor)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

// Conserva la causa para errors.Is/As sin incluir su mensaje en registros o
// transportes que conviertan el error nominal a texto.
type errorExportacionAuditoria struct{ causa error }

func (errorExportacionAuditoria) Error() string { return domain.ErrCheckpointInvalido.Error() }
func (e errorExportacionAuditoria) Unwrap() []error {
	return []error{domain.ErrCheckpointInvalido, e.causa}
}
