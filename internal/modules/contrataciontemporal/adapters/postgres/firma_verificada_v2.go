package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
)

// RegistroFirmasVerificadasPostgreSQL consume las fachadas V2 con el LOGIN CT.
// El plan selecciona el descriptor; AUT35 fabrica el canon dentro de CT172.
type RegistroFirmasVerificadasPostgreSQL struct {
	pool iniciadorRegistroIncorporacionV2
}

func NuevoRegistroFirmasVerificadasPostgreSQL(pool *pgxpool.Pool) (*RegistroFirmasVerificadasPostgreSQL, error) {
	if nuloRegistroTX(pool) {
		return nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	return &RegistroFirmasVerificadasPostgreSQL{pool: pool}, nil
}

// CT186: la v3 calcula la huella con los ámbitos de la asignación de quien
// consulta (AD210) y liga UnidadRef al paso del plan publicado (CC10).
const consultarFirmasSQL172 = `SELECT vec_contratacion_temporal.consultar_firmas_r5_atestadas_v3($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)::text`

type firmaRevisionPDFSQL172 struct {
	firmaExternaSQL170
	FirmanteRef                  string               `json:"FirmanteRef"`
	CertificadoHuella            string               `json:"CertificadoHuella"`
	FirmaAnteriorRef             *string              `json:"FirmaAnteriorRef"`
	ReciboAnteriorRef            *string              `json:"ReciboAnteriorRef"`
	EntradaDocumentoRef          string               `json:"EntradaDocumentoRef"`
	EntradaDocumentoVersion      uint64               `json:"EntradaDocumentoVersion"`
	EntradaDocumentoLongitud     uint64               `json:"EntradaDocumentoLongitud"`
	EntradaDocumentoHuella       string               `json:"EntradaDocumentoHuella"`
	OrdenFirmaPDF                int                  `json:"OrdenFirmaPDF"`
	ByteRange                    byteRangeFirmaSQL172 `json:"ByteRange"`
	RevisionHuellaSHA256         string               `json:"RevisionHuellaSHA256"`
	ContenidoFirmadoHuellaSHA256 string               `json:"ContenidoFirmadoHuellaSHA256"`
	RevisionLongitud             uint64               `json:"RevisionLongitud"`
	EvidenciaFirmasCanonica      string               `json:"EvidenciaFirmasCanonica"`
	EvidenciaFirmasHuellaSHA256  string               `json:"EvidenciaFirmasHuellaSHA256"`
}

type respuestaFirmasR5SQL172 struct {
	respuestaFirmasR5SQL170
	RevisionesPDF []firmaRevisionPDFSQL172 `json:"RevisionesPDF"`
}

// ConsultarFirmasAutorizadasV2 consume una capacidad nueva y devuelve la
// historia sólo después del COMMIT de la auditoría común. CT172 conserva los
// bytes de evidencia como texto JSON para que la lectura no los reconstruya.
func (r *RegistroFirmasVerificadasPostgreSQL) ConsultarFirmasAutorizadasV2(ctx context.Context,
	m ports.MaterialConsultaFirmasR5V2, c ports.CapacidadConsultaFirmasR5V2,
) (ports.LecturaFirmasR5V2, error) {
	var cero ports.LecturaFirmasR5V2
	if ctx == nil || r == nil || nuloRegistroTX(r.pool) {
		return cero, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	canonico, err := m.Canonico()
	if err != nil {
		return cero, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	if ctapp.ValidarCapacidadConsultaFirmasR5V2(c, m) != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	parametros, err := exportacionParametrosRegistroV2(c.ExportarMaterialParaConsumidor(), true)
	if err != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	defer limpiarParametrosFirma172(parametros)
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || nuloRegistroTX(tx) {
		return cero, errorFirma118(ctx, err)
	}
	confirmado := false
	defer func() {
		if !confirmado {
			c, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(2*time.Second))
			defer cancelar()
			if err := tx.Rollback(c); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
				slog.Warn("contratacion temporal: rollback de lectura de firmas V2 no confirmado")
			}
		}
	}()
	if _, err = tx.Exec(ctx, ajustesRegistroIncorporacionTXV2); err != nil {
		return cero, errorFirma118(ctx, err)
	}
	var contenido []byte
	if err = tx.QueryRow(ctx, consultarFirmasSQL172, append([]any{string(canonico)}, parametros...)...).Scan(&contenido); err != nil {
		return cero, errorFirma118(ctx, err)
	}
	defer clear(contenido)
	var w respuestaFirmasR5SQL172
	if decodificarFirma118(contenido, &w) != nil {
		return cero, ports.ErrResultadoFirmaDocumentoInvalido
	}
	lectura, err := proyectarLecturaFirmasSQL172(m, w)
	if err != nil {
		return cero, err
	}
	if err = ctx.Err(); err != nil {
		return cero, err
	}
	if err = tx.Commit(ctx); err != nil {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		return cero, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	confirmado = true
	if !*w.Encontrado {
		return cero, ports.ErrExpedienteConsultaFirmasNoEncontrado
	}
	return lectura, nil
}

// proyectarLecturaFirmasSQL172 valida la proyección común a consulta y recuperación.
// La transacción llamadora conserva el resultado oculto hasta confirmar su COMMIT.
func proyectarLecturaFirmasSQL172(m ports.MaterialConsultaFirmasR5V2, w respuestaFirmasR5SQL172) (ports.LecturaFirmasR5V2, error) {
	var cero ports.LecturaFirmasR5V2
	if w.Encontrado == nil || w.ExpedienteRef != m.ExpedienteRef ||
		w.Firmas == nil || w.RevisionesPDF == nil || len(w.Firmas) > 128 || len(w.RevisionesPDF) > 128 || w.HistoriaRevision == nil ||
		*w.HistoriaRevision > 9007199254740991 || w.CoincideFirmanteEnOtroPaso == nil ||
		w.HistoriaSeparacionAcreditada == nil || !domain.HuellaSHA256FirmaValida(w.HistoriaHuella) ||
		(!*w.Encontrado && (len(w.Firmas) != 0 || len(w.RevisionesPDF) != 0)) {
		return cero, ports.ErrResultadoFirmaDocumentoInvalido
	}
	firmas := make([]ports.FirmaRegistrada, 0, len(w.Firmas))
	porRef := make(map[string]ports.FirmaRegistrada, len(w.Firmas))
	for _, f := range w.Firmas {
		p, ok := proyectarFirmaSQL172(f, m)
		if !ok || porRef[p.FirmaRef].FirmaRef != "" {
			return cero, ports.ErrResultadoFirmaDocumentoInvalido
		}
		firmas = append(firmas, p)
		porRef[p.FirmaRef] = p
	}
	revisiones := make([]ports.FirmaRegistradaRevisionPDFV2, 0, len(w.RevisionesPDF))
	vistas := make(map[string]bool, len(w.RevisionesPDF))
	porRevision := make(map[string]firmaRevisionPDFSQL172, len(w.RevisionesPDF))
	for _, v := range w.RevisionesPDF {
		if _, existe := porRevision[v.FirmaRef]; existe {
			return cero, ports.ErrResultadoFirmaDocumentoInvalido
		}
		porRevision[v.FirmaRef] = v
	}
	for _, v := range w.RevisionesPDF {
		base, existe := porRef[v.FirmaRef]
		h := sha256.Sum256([]byte(v.EvidenciaFirmasCanonica))
		var compacto bytes.Buffer
		var evidencia []json.RawMessage
		if !existe || vistas[v.FirmaRef] || v.FirmaRef == "" ||
			v.ReciboRef != base.ReciboRef || v.Documento != base.Documento ||
			v.Secuencia != base.Secuencia || v.ExpedienteVersion != base.ExpedienteVersion ||
			v.PasoOrden != base.PasoOrden || v.CatalogoHuella != base.CatalogoHuella ||
			!domain.HuellaSHA256FirmaValida(v.CertificadoHuella) || v.FirmanteRef != "ref:"+v.CertificadoHuella ||
			v.RevisionHuellaSHA256 != base.FirmadoHuella || v.OrdenFirmaPDF != base.PasoOrden ||
			v.OrdenFirmaPDF < 1 || v.OrdenFirmaPDF > 2 ||
			!domain.ReferenciaOpacaValida(v.EntradaDocumentoRef) ||
			v.EntradaDocumentoVersion < 1 || v.EntradaDocumentoVersion > 9007199254740991 ||
			!domain.HuellaSHA256FirmaValida(v.EntradaDocumentoHuella) ||
			v.EntradaDocumentoLongitud < 1 || v.EntradaDocumentoLongitud >= v.RevisionLongitud ||
			v.RevisionLongitud > ports.MaximoDocumentoFirmaBytes ||
			len(v.ByteRange) != 4 ||
			v.ByteRange[0] != 0 || v.ByteRange[1] < v.EntradaDocumentoLongitud ||
			v.ByteRange[2] <= v.ByteRange[1] || v.ByteRange[2] > v.RevisionLongitud ||
			v.ByteRange[3] == 0 || v.ByteRange[3] != v.RevisionLongitud-v.ByteRange[2] ||
			(v.OrdenFirmaPDF == 1 && (textoFirma118(v.FirmaAnteriorRef) != "" || textoFirma118(v.ReciboAnteriorRef) != "" ||
				v.EntradaDocumentoRef != base.OriginalRef || v.EntradaDocumentoVersion != base.OriginalVersion ||
				v.EntradaDocumentoHuella != base.OriginalHuella)) ||
			(v.OrdenFirmaPDF == 2 && !antecedenteRevisionFirmaSQL172(v, base, porRef, porRevision)) ||
			!domain.HuellaSHA256FirmaValida(v.ContenidoFirmadoHuellaSHA256) ||
			len(v.EvidenciaFirmasCanonica) < 2 || len(v.EvidenciaFirmasCanonica) > 32768 ||
			json.Compact(&compacto, []byte(v.EvidenciaFirmasCanonica)) != nil ||
			!bytes.Equal(compacto.Bytes(), []byte(v.EvidenciaFirmasCanonica)) ||
			json.Unmarshal([]byte(v.EvidenciaFirmasCanonica), &evidencia) != nil || len(evidencia) != v.OrdenFirmaPDF ||
			hex.EncodeToString(h[:]) != v.EvidenciaFirmasHuellaSHA256 {
			return cero, ports.ErrResultadoFirmaDocumentoInvalido
		}
		vistas[v.FirmaRef] = true
		base.FirmanteRef, base.CertificadoHuella = v.FirmanteRef, v.CertificadoHuella
		revisiones = append(revisiones, ports.FirmaRegistradaRevisionPDFV2{FirmaRegistrada: base,
			FirmaAnteriorRef: textoFirma118(v.FirmaAnteriorRef), ReciboAnteriorRef: textoFirma118(v.ReciboAnteriorRef),
			EntradaDocumentoRef: v.EntradaDocumentoRef, EntradaDocumentoVersion: v.EntradaDocumentoVersion,
			EntradaDocumentoLongitud: v.EntradaDocumentoLongitud, EntradaDocumentoHuella: v.EntradaDocumentoHuella,
			OrdenFirmaPDF: v.OrdenFirmaPDF, ByteRange: [4]uint64(v.ByteRange), RevisionHuellaSHA256: v.RevisionHuellaSHA256,
			ContenidoFirmadoHuellaSHA256: v.ContenidoFirmadoHuellaSHA256, RevisionLongitud: v.RevisionLongitud,
			EvidenciaFirmasCanonica:     json.RawMessage(v.EvidenciaFirmasCanonica),
			EvidenciaFirmasHuellaSHA256: v.EvidenciaFirmasHuellaSHA256})
	}
	return ports.LecturaFirmasR5V2{LecturaFirmasR5: ports.LecturaFirmasR5{
		Firmas: firmas, HistoriaRevision: *w.HistoriaRevision, HistoriaHuella: w.HistoriaHuella,
		CoincideFirmanteEnOtroPaso:   *w.CoincideFirmanteEnOtroPaso,
		HistoriaSeparacionAcreditada: *w.HistoriaSeparacionAcreditada}, RevisionesPDF: revisiones}, nil
}

func antecedenteRevisionFirmaSQL172(v firmaRevisionPDFSQL172, base ports.FirmaRegistrada,
	firmas map[string]ports.FirmaRegistrada, revisiones map[string]firmaRevisionPDFSQL172,
) bool {
	ref := textoFirma118(v.FirmaAnteriorRef)
	previa, existe := firmas[ref]
	revision, revisada := revisiones[ref]
	return existe && revisada && domain.ReferenciaOpacaValida(ref) &&
		previa.ReciboRef == textoFirma118(v.ReciboAnteriorRef) && revision.OrdenFirmaPDF == 1 &&
		previa.PasoOrden == 1 && previa.Secuencia+1 == base.Secuencia &&
		previa.Documento == base.Documento && previa.CatalogoRef == base.CatalogoRef && previa.CatalogoHuella == base.CatalogoHuella &&
		previa.OriginalRef == base.OriginalRef && previa.OriginalVersion == base.OriginalVersion && previa.OriginalHuella == base.OriginalHuella &&
		v.EntradaDocumentoRef == previa.DocumentoCustodiaRef && v.EntradaDocumentoVersion == previa.DocumentoCustodiaVersion &&
		v.EntradaDocumentoHuella == previa.FirmadoHuella && v.EntradaDocumentoLongitud == revision.RevisionLongitud
}

func limpiarParametrosFirma172(parametros []any) {
	for _, v := range parametros {
		if b, ok := v.([]byte); ok {
			clear(b)
		}
	}
}

func proyectarFirmaSQL172(f firmaExternaSQL170, m ports.MaterialConsultaFirmasR5V2) (ports.FirmaRegistrada, bool) {
	via := textoFirma118(f.Via)
	if f.Documento != m.Documento || (via != "" && via != ports.ViaFirmaExternaPortafirmas && via != ports.ViaFirmaCertificadoVEC) ||
		!domain.ReferenciaOpacaValida(f.FirmaRef) || !domain.ReferenciaOpacaValida(f.ReciboRef) ||
		f.Secuencia < 1 || f.ExpedienteVersion == 0 || f.ExpedienteVersion > m.VersionExpediente || f.RegistradaEn.IsZero() ||
		(via != "" && (!f.FirmantePrincipalAcreditado || f.HistoriaRevision == nil || f.HistoriaHuella == nil ||
			!domain.HuellaSHA256FirmaValida(textoFirma118(f.HistoriaHuella)))) {
		return ports.FirmaRegistrada{}, false
	}
	return ports.FirmaRegistrada{Via: via, FirmaRef: f.FirmaRef, ReciboRef: f.ReciboRef, Documento: f.Documento,
		FirmantePrincipalAcreditado: f.FirmantePrincipalAcreditado, CoincideFirmanteCandidato: f.CoincideFirmanteCandidato,
		HistoriaRevision: versionFirma118(f.HistoriaRevision), HistoriaHuella: textoFirma118(f.HistoriaHuella),
		Secuencia: f.Secuencia, ExpedienteVersion: f.ExpedienteVersion, CatalogoRef: f.CatalogoRef,
		CatalogoHuella: f.CatalogoHuella, PasoRef: f.PasoRef, PasoOrden: f.PasoOrden,
		Resultado: domain.ResultadoFirmaDocumento(f.Resultado), ConMotivoDevolucion: f.ConMotivo,
		OriginalHuella: textoFirma118(f.OriginalHuella), OriginalRef: textoFirma118(f.OriginalRef),
		OriginalVersion: versionFirma118(f.OriginalVersion), FirmadoHuella: textoFirma118(f.FirmadoHuella),
		SelloTiempoEstado:              textoFirma118(f.SelloTiempoEstado),
		ReferenciaPortafirmasDeclarada: textoFirma118(f.ReferenciaPortafirmasDeclarada),
		FechaPortafirmasDeclarada:      textoFirma118(f.FechaPortafirmasDeclarada),
		RegistradaEn:                   f.RegistradaEn.UTC(), ClaveIdempotencia: f.ClaveIdempotencia,
		DocumentoCustodiaRef:     textoFirma118(f.DocumentoCustodia),
		DocumentoCustodiaVersion: versionFirma118(f.VersionCustodia)}, true
}
