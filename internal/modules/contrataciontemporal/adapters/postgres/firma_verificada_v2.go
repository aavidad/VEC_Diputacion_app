package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const registrarFirmaSQL172 = `SELECT vec_contratacion_temporal.registrar_firma_verificada_v2($1::text,$2::timestamptz,$3,$4,$5,$6,$7::numeric,$8::numeric,$9,$10,$11,$12)::text`
const consultarFirmasSQL172 = `SELECT vec_contratacion_temporal.consultar_firmas_r5_atestadas_v2($1::text,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)::text`

type RegistroFirmasVerificadasV2PostgreSQL struct {
	pool iniciadorRegistroIncorporacionV2
}

var _ ports.RegistroFirmasVerificadasV2 = (*RegistroFirmasVerificadasV2PostgreSQL)(nil)

func NuevoRegistroFirmasVerificadasV2PostgreSQL(pool *pgxpool.Pool) (*RegistroFirmasVerificadasV2PostgreSQL, error) {
	if nuloRegistroTX(pool) {
		return nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	return &RegistroFirmasVerificadasV2PostgreSQL{pool}, nil
}

func (r *RegistroFirmasVerificadasV2PostgreSQL) RegistrarFirmaVerificadaV2(ctx context.Context, m ports.MaterialFirmaVerificadaV2, c ports.CapacidadFirmaVerificadaV2) (ports.ReciboFirmaDocumento, error) {
	var zero ports.ReciboFirmaDocumento
	if ctx == nil || r == nil || nuloRegistroTX(r.pool) {
		return zero, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	canonical, err := m.Canonico()
	if err != nil || m.ComprobadaEn.IsZero() {
		return zero, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	hash, err := m.HuellaSHA256()
	if err != nil {
		return zero, err
	}
	params, err := exportacionParametrosRegistroV2(c.ExportarMaterialParaConsumidor(), true)
	if err != nil {
		return zero, ports.ErrFirmaDocumentoDenegada
	}
	defer limpiarParametrosFirma172(params)
	args := append([]any{string(canonical), m.ComprobadaEn.UTC().Truncate(time.Microsecond)}, params...)
	var result ports.ReciboFirmaDocumento
	validate := func(raw []byte) error {
		var w reciboFirmaSQL118
		if decodificarFirma118(raw, &w) != nil {
			return ports.ErrResultadoFirmaDocumentoInvalido
		}
		result = ports.ReciboFirmaDocumento{FirmaRef: w.FirmaRef, ReciboRef: w.ReciboRef, Secuencia: w.Secuencia, Resultado: domain.ResultadoFirmaDocumento(w.Resultado), ExpedienteVersion: w.ExpedienteVersion, ActorRef: w.ActorRef, PerfilRef: w.PerfilRef, RegistradaEn: w.RegistradaEn.UTC(), SolicitudHuella: w.SolicitudHuella, YaRegistrada: w.YaRegistrada, DocumentoCustodiaRef: textoFirma118(w.DocumentoCustodia), DocumentoCustodiaVersion: versionFirma118(w.VersionCustodia)}
		if !domain.ReferenciaOpacaValida(result.FirmaRef) || !domain.ReferenciaOpacaValida(result.ReciboRef) || result.SolicitudHuella != hash || result.Secuencia != m.Secuencia || result.ExpedienteVersion != m.VersionExpediente || result.Resultado != domain.ResultadoFirmaFirmado || result.ActorRef == "" || (result.ActorRef == m.FirmantePrincipalRef) != (m.Via == ports.ViaFirmaCertificadoVEC) || result.PerfilRef == "" || result.RegistradaEn.IsZero() || result.DocumentoCustodiaRef != m.DocumentoCustodiaRef || result.DocumentoCustodiaVersion != m.DocumentoCustodiaVersion {
			return ports.ErrResultadoFirmaDocumentoInvalido
		}
		return nil
	}
	for attempt := 1; attempt <= intentosRegistroFirma118; attempt++ {
		err = r.operarFirma172(ctx, registrarFirmaSQL172, args, validate)
		if err == nil {
			return result, nil
		}
		if errors.Is(err, ports.ErrResultadoFirmaDocumentoInvalido) {
			return zero, err
		}
		if !reintentableFirma118(err) || attempt == intentosRegistroFirma118 || ctx.Err() != nil {
			break
		}
	}
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == "P1701" {
		return zero, ports.ErrFirmaDocumentoEnConflicto
	}
	return zero, errorFirma118(ctx, err)
}

// El recibo y la proyección se validan antes de confirmar la transacción V3.
func (r *RegistroFirmasVerificadasV2PostgreSQL) operarFirma172(ctx context.Context, sql string, args []any, validate func([]byte) error) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return err
	}
	if nuloRegistroTX(tx) {
		return ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	committed := false
	defer func() {
		if !committed {
			c, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = tx.Rollback(c)
		}
	}()
	if _, err = tx.Exec(ctx, ajustesRegistroIncorporacionTXV2); err != nil {
		return err
	}
	var raw []byte
	if err = tx.QueryRow(ctx, sql, args...).Scan(&raw); err != nil {
		return err
	}
	defer clear(raw)
	if err = validate(raw); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	return nil
}
func limpiarParametrosFirma172(params []any) {
	for _, p := range params {
		if b, ok := p.([]byte); ok {
			clear(b)
		}
	}
}

type revisionFirmaSQL172 struct {
	firmaExternaSQL170
	FirmanteRef                                        string
	CertificadoHuella                                  string
	FirmaAnteriorRef, ReciboAnteriorRef                *string
	EntradaDocumentoRef                                string
	EntradaDocumentoVersion, EntradaDocumentoLongitud  *uint64
	EntradaDocumentoHuella                             string
	OrdenFirmaPDF                                      *int
	ByteRange                                          []uint64
	RevisionHuellaSHA256, ContenidoFirmadoHuellaSHA256 string
	RevisionLongitud                                   *uint64
	EvidenciaFirmasCanonica                            string
	EvidenciaFirmasHuellaSHA256                        string
}
type respuestaFirmasSQL172 struct {
	Encontrado                                               *bool
	ExpedienteRef                                            string
	Firmas                                                   []firmaExternaSQL170
	RevisionesPDF                                            []revisionFirmaSQL172
	HistoriaRevision                                         *uint64
	HistoriaHuella                                           string
	CoincideFirmanteEnOtroPaso, HistoriaSeparacionAcreditada *bool
}

func (r *RegistroFirmasVerificadasV2PostgreSQL) ConsultarFirmasAutorizadasV2(ctx context.Context, m ports.MaterialConsultaFirmasR5V2, c ports.CapacidadConsultaFirmasR5V2) (ports.LecturaFirmasR5V2, error) {
	var zero ports.LecturaFirmasR5V2
	if ctx == nil || r == nil || nuloRegistroTX(r.pool) {
		return zero, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	canonical, err := m.Canonico()
	if err != nil {
		return zero, err
	}
	params, err := exportacionParametrosRegistroV2(c.ExportarMaterialParaConsumidor(), true)
	if err != nil {
		return zero, ports.ErrFirmaDocumentoDenegada
	}
	defer limpiarParametrosFirma172(params)
	var result ports.LecturaFirmasR5V2
	err = r.operarFirma172(ctx, consultarFirmasSQL172, append([]any{string(canonical)}, params...), func(raw []byte) error {
		var w respuestaFirmasSQL172
		if decodificarFirma118(raw, &w) != nil {
			return ports.ErrResultadoFirmaDocumentoInvalido
		}
		var e error
		result, e = lecturaFirma172(w, m.MaterialConsultaFirmasR5)
		return e
	})
	if err != nil {
		if errors.Is(err, ports.ErrResultadoFirmaDocumentoInvalido) {
			return zero, err
		}
		return zero, errorFirma118(ctx, err)
	}
	if result.Firmas == nil {
		return zero, ports.ErrExpedienteConsultaFirmasNoEncontrado
	}
	return result, nil
}
func lecturaFirma172(w respuestaFirmasSQL172, m ports.MaterialConsultaFirmasR5) (ports.LecturaFirmasR5V2, error) {
	var zero ports.LecturaFirmasR5V2
	if w.Encontrado == nil || w.ExpedienteRef != m.ExpedienteRef || w.Firmas == nil || w.RevisionesPDF == nil || w.HistoriaRevision == nil || *w.HistoriaRevision > 9007199254740991 || !domain.HuellaSHA256FirmaValida(w.HistoriaHuella) || w.CoincideFirmanteEnOtroPaso == nil || w.HistoriaSeparacionAcreditada == nil || len(w.Firmas) > 1000 || len(w.RevisionesPDF) > len(w.Firmas) {
		return zero, ports.ErrResultadoFirmaDocumentoInvalido
	}
	if !*w.Encontrado {
		if len(w.Firmas) != 0 || len(w.RevisionesPDF) != 0 {
			return zero, ports.ErrResultadoFirmaDocumentoInvalido
		}
		return zero, nil
	}
	result := ports.LecturaFirmasR5V2{LecturaFirmasR5: ports.LecturaFirmasR5{Firmas: make([]ports.FirmaRegistrada, 0, len(w.Firmas)), HistoriaRevision: *w.HistoriaRevision, HistoriaHuella: w.HistoriaHuella, CoincideFirmanteEnOtroPaso: *w.CoincideFirmanteEnOtroPaso, HistoriaSeparacionAcreditada: *w.HistoriaSeparacionAcreditada}, RevisionesPDF: make([]ports.FirmaRegistradaRevisionPDFV2, 0, len(w.RevisionesPDF))}
	seen := make(map[string]ports.FirmaRegistrada, len(w.Firmas))
	for _, f := range w.Firmas {
		v, e := proyectarFirma172(f, m)
		if e != nil {
			return zero, e
		}
		if _, ok := seen[v.FirmaRef]; ok {
			return zero, ports.ErrResultadoFirmaDocumentoInvalido
		}
		seen[v.FirmaRef] = v
		result.Firmas = append(result.Firmas, v)
	}
	revisionsSeen := make(map[string]bool, len(w.RevisionesPDF))
	for _, f := range w.RevisionesPDF {
		base, e := proyectarFirma172(f.firmaExternaSQL170, m)
		if e != nil {
			return zero, e
		}
		listed, ok := seen[base.FirmaRef]
		if !ok || listed != base || revisionsSeen[base.FirmaRef] || f.EntradaDocumentoVersion == nil || *f.EntradaDocumentoVersion == 0 || *f.EntradaDocumentoVersion > 9007199254740991 || f.EntradaDocumentoLongitud == nil || *f.EntradaDocumentoLongitud == 0 || !domain.ReferenciaOpacaValida(f.EntradaDocumentoRef) || !domain.HuellaSHA256FirmaValida(f.EntradaDocumentoHuella) || f.OrdenFirmaPDF == nil || *f.OrdenFirmaPDF != base.PasoOrden || *f.OrdenFirmaPDF < 1 || *f.OrdenFirmaPDF > 2 || len(f.ByteRange) != 4 || f.RevisionLongitud == nil || *f.RevisionLongitud <= *f.EntradaDocumentoLongitud || *f.RevisionLongitud > 33554432 || f.ByteRange[0] != 0 || f.ByteRange[1] < *f.EntradaDocumentoLongitud || f.ByteRange[2] <= f.ByteRange[1] || f.ByteRange[2] > *f.RevisionLongitud || f.ByteRange[3] != *f.RevisionLongitud-f.ByteRange[2] || f.ByteRange[3] == 0 || f.RevisionHuellaSHA256 != base.FirmadoHuella || !domain.HuellaSHA256FirmaValida(f.ContenidoFirmadoHuellaSHA256) || f.FirmanteRef != "ref:"+f.CertificadoHuella || !domain.HuellaSHA256FirmaValida(f.CertificadoHuella) || len(f.EvidenciaFirmasCanonica) > 32768 {
			return zero, ports.ErrResultadoFirmaDocumentoInvalido
		}
		evidence := []byte(f.EvidenciaFirmasCanonica)
		var records []json.RawMessage
		hash := sha256.Sum256(evidence)
		if json.Unmarshal(evidence, &records) != nil || len(records) != *f.OrdenFirmaPDF || hex.EncodeToString(hash[:]) != f.EvidenciaFirmasHuellaSHA256 {
			return zero, ports.ErrResultadoFirmaDocumentoInvalido
		}
		if (*f.OrdenFirmaPDF == 1) != (f.FirmaAnteriorRef == nil && f.ReciboAnteriorRef == nil) {
			return zero, ports.ErrResultadoFirmaDocumentoInvalido
		}
		if *f.OrdenFirmaPDF == 2 && (f.FirmaAnteriorRef == nil || f.ReciboAnteriorRef == nil || !domain.ReferenciaOpacaValida(*f.FirmaAnteriorRef) || !domain.ReferenciaOpacaValida(*f.ReciboAnteriorRef)) {
			return zero, ports.ErrResultadoFirmaDocumentoInvalido
		}
		base.FirmanteRef = f.FirmanteRef
		base.CertificadoHuella = f.CertificadoHuella
		v := ports.FirmaRegistradaRevisionPDFV2{FirmaRegistrada: base, FirmaAnteriorRef: textoFirma118(f.FirmaAnteriorRef), ReciboAnteriorRef: textoFirma118(f.ReciboAnteriorRef), EntradaDocumentoRef: f.EntradaDocumentoRef, EntradaDocumentoVersion: *f.EntradaDocumentoVersion, EntradaDocumentoLongitud: *f.EntradaDocumentoLongitud, EntradaDocumentoHuella: f.EntradaDocumentoHuella, OrdenFirmaPDF: *f.OrdenFirmaPDF, ByteRange: [4]uint64{f.ByteRange[0], f.ByteRange[1], f.ByteRange[2], f.ByteRange[3]}, RevisionHuellaSHA256: f.RevisionHuellaSHA256, ContenidoFirmadoHuellaSHA256: f.ContenidoFirmadoHuellaSHA256, RevisionLongitud: *f.RevisionLongitud, EvidenciaFirmasCanonica: append(json.RawMessage(nil), evidence...), EvidenciaFirmasHuellaSHA256: f.EvidenciaFirmasHuellaSHA256}
		revisionsSeen[base.FirmaRef] = true
		result.RevisionesPDF = append(result.RevisionesPDF, v)
	}
	return result, nil
}
func proyectarFirma172(f firmaExternaSQL170, m ports.MaterialConsultaFirmasR5) (ports.FirmaRegistrada, error) {
	var zero ports.FirmaRegistrada
	via := textoFirma118(f.Via)
	if f.Documento != m.Documento || !domain.ReferenciaOpacaValida(f.FirmaRef) || !domain.ReferenciaOpacaValida(f.ReciboRef) || f.Secuencia < 1 || f.ExpedienteVersion == 0 || f.ExpedienteVersion > 9007199254740991 || f.RegistradaEn.IsZero() || (via != "" && via != ports.ViaFirmaCertificadoVEC && via != ports.ViaFirmaExternaPortafirmas) || (via != "" && (!f.FirmantePrincipalAcreditado || f.HistoriaRevision == nil || f.HistoriaHuella == nil || !domain.HuellaSHA256FirmaValida(*f.HistoriaHuella))) || (via == "" && (f.FirmantePrincipalAcreditado || f.CoincideFirmanteCandidato || f.HistoriaRevision != nil || f.HistoriaHuella != nil)) {
		return zero, ports.ErrResultadoFirmaDocumentoInvalido
	}
	return ports.FirmaRegistrada{Via: via, FirmaRef: f.FirmaRef, ReciboRef: f.ReciboRef, Documento: f.Documento, Secuencia: f.Secuencia, ExpedienteVersion: f.ExpedienteVersion, CatalogoRef: f.CatalogoRef, CatalogoHuella: f.CatalogoHuella, PasoRef: f.PasoRef, PasoOrden: f.PasoOrden, Resultado: domain.ResultadoFirmaDocumento(f.Resultado), ConMotivoDevolucion: f.ConMotivo, OriginalRef: textoFirma118(f.OriginalRef), OriginalVersion: versionFirma118(f.OriginalVersion), OriginalHuella: textoFirma118(f.OriginalHuella), FirmadoHuella: textoFirma118(f.FirmadoHuella), FirmantePrincipalAcreditado: f.FirmantePrincipalAcreditado, CoincideFirmanteCandidato: f.CoincideFirmanteCandidato, HistoriaRevision: versionFirma118(f.HistoriaRevision), HistoriaHuella: textoFirma118(f.HistoriaHuella), SelloTiempoEstado: textoFirma118(f.SelloTiempoEstado), ReferenciaPortafirmasDeclarada: textoFirma118(f.ReferenciaPortafirmasDeclarada), FechaPortafirmasDeclarada: textoFirma118(f.FechaPortafirmasDeclarada), RegistradaEn: f.RegistradaEn.UTC(), ClaveIdempotencia: f.ClaveIdempotencia, DocumentoCustodiaRef: textoFirma118(f.DocumentoCustodia), DocumentoCustodiaVersion: versionFirma118(f.VersionCustodia)}, nil
}
