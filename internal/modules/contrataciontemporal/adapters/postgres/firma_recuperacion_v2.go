package postgres

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// CT186: como la consulta, la v3 usa AD210 y CC10.
const recuperarFirmasSQL175 = `SELECT vec_contratacion_temporal.recuperar_firmas_r5_atestadas_v3($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)::text`

type recuperacionFirmaSQL175 struct {
	FirmaRef           string `json:"FirmaRef"`
	MaterialRootSHA256 string `json:"MaterialRootSHA256"`
	CanonNominal       string `json:"CanonNominal"`
	CanonNominalSHA256 string `json:"CanonNominalSHA256"`
	CanonNominalRef    string `json:"CanonNominalRef"`
}

type respuestaRecuperacionSQL175 struct {
	respuestaFirmasR5SQL172
	Recuperaciones []recuperacionFirmaSQL175 `json:"Recuperaciones"`
}

var _ ports.LectorRecuperacionFirmasV2 = (*RegistroFirmasVerificadasPostgreSQL)(nil)

// RecuperarFirmasAutorizadasV2 consume una decisión propia de 48 campos.
// La consulta SQL audita y recupera los bytes originales en la misma TX;
// ningún resultado sale si la proyección o el COMMIT fallan.
func (r *RegistroFirmasVerificadasPostgreSQL) RecuperarFirmasAutorizadasV2(ctx context.Context,
	m ports.MaterialConsultaFirmasR5V2, c ports.CapacidadRecuperacionFirmasV2,
) (ports.LecturaRecuperacionFirmasV2, error) {
	var cero ports.LecturaRecuperacionFirmasV2
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
	if ctapp.ValidarCapacidadRecuperacionFirmasV2(c, m) != nil {
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
			c, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancelar()
			if err := tx.Rollback(c); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
				slog.Warn("contratacion temporal: rollback de recuperación de firmas V2 no confirmado")
			}
		}
	}()
	if _, err = tx.Exec(ctx, ajustesRegistroIncorporacionTXV2); err != nil {
		return cero, errorFirma118(ctx, err)
	}
	var contenido []byte
	if err = tx.QueryRow(ctx, recuperarFirmasSQL175, append([]any{string(canonico)}, parametros...)...).Scan(&contenido); err != nil {
		return cero, errorFirma118(ctx, err)
	}
	defer clear(contenido)
	var w respuestaRecuperacionSQL175
	if decodificarFirma118(contenido, &w) != nil {
		return cero, ports.ErrResultadoFirmaDocumentoInvalido
	}
	base, err := proyectarLecturaFirmasSQL172(m, w.respuestaFirmasR5SQL172)
	if err != nil {
		return cero, err
	}
	recuperaciones, err := proyectarRecuperacionesSQL175(m, base, w.Recuperaciones)
	if err != nil {
		return cero, err
	}
	// La proyección de aplicación coteja certificado y fecha con Firmas; la
	// fila base de CT172 sólo los lleva en RevisionesPDF.
	porRef := make(map[string]ports.FirmaRegistradaRevisionPDFV2, len(base.RevisionesPDF))
	for _, v := range base.RevisionesPDF {
		porRef[v.FirmaRef] = v
	}
	for i := range base.Firmas {
		if v, ok := porRef[base.Firmas[i].FirmaRef]; ok {
			base.Firmas[i].CertificadoHuella = v.CertificadoHuella
			base.Firmas[i].FirmanteRef = v.FirmanteRef
		}
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
	if err = ctx.Err(); err != nil {
		return cero, err
	}
	if !*w.Encontrado {
		return cero, ports.ErrExpedienteConsultaFirmasNoEncontrado
	}
	return ports.LecturaRecuperacionFirmasV2{LecturaFirmasR5V2: base, Recuperaciones: recuperaciones}, nil
}

func proyectarRecuperacionesSQL175(m ports.MaterialConsultaFirmasR5V2, base ports.LecturaFirmasR5V2, filas []recuperacionFirmaSQL175) ([]ports.RecuperacionFirmaV2, error) {
	if filas == nil || len(filas) > 128 || len(filas) != len(base.RevisionesPDF) {
		return nil, ports.ErrResultadoFirmaDocumentoInvalido
	}
	porRef := make(map[string]ports.FirmaRegistradaRevisionPDFV2, len(base.RevisionesPDF))
	for _, v := range base.RevisionesPDF {
		porRef[v.FirmaRef] = v
	}
	vistos := make(map[string]bool, len(filas))
	salida := make([]ports.RecuperacionFirmaV2, 0, len(filas))
	for _, v := range filas {
		revision, ok := porRef[v.FirmaRef]
		if !ok || vistos[v.FirmaRef] || !domain.HuellaSHA256FirmaValida(v.MaterialRootSHA256) ||
			len(v.CanonNominal) < 512 || len(v.CanonNominal) > 32768 ||
			!strings.HasPrefix(v.CanonNominalRef, "evidencia:competencia-firmante-ct:") ||
			!domain.HuellaSHA256FirmaValida(strings.TrimPrefix(v.CanonNominalRef, "evidencia:competencia-firmante-ct:")) {
			return nil, ports.ErrResultadoFirmaDocumentoInvalido
		}
		canon, err := vecdomain.RecuperarCanonCompetenciaFirmanteHistoricaV1([]byte(v.CanonNominal), v.CanonNominalSHA256)
		if err != nil {
			return nil, errors.Join(ports.ErrResultadoFirmaDocumentoInvalido, err)
		}
		if canon.Recurso.ModuloID != ports.ModuloContratacion ||
			canon.Recurso.OrganizacionRef != m.OrganizacionRef || canon.Recurso.ExpedienteRef != m.ExpedienteRef ||
			canon.Recurso.DocumentoRef != revision.OriginalRef ||
			canon.Recurso.Original.Referencia != revision.OriginalRef || canon.Recurso.Original.Version != revision.OriginalVersion ||
			canon.Recurso.Original.HuellaSHA256 != revision.OriginalHuella ||
			canon.Recurso.Firmado.Referencia != revision.DocumentoCustodiaRef || canon.Recurso.Firmado.Version != revision.DocumentoCustodiaVersion ||
			canon.Recurso.Firmado.HuellaSHA256 != revision.FirmadoHuella || canon.Identidad.CertificadoDERSHA256 != revision.CertificadoHuella ||
			canon.PasoRef != revision.PasoRef || !ordenRecuperacionCoincide(canon.PasoOrden, revision.PasoOrden) ||
			!ordenRecuperacionCoincide(canon.Recurso.NumeroFirmas, revision.OrdenFirmaPDF) ||
			canon.Circuito.Referencia != revision.CatalogoRef || canon.Circuito.HuellaSHA256 != revision.CatalogoHuella ||
			!canon.FechaHistorica.Equal(revision.RegistradaEn) ||
			(revision.OrdenFirmaPDF == 1 && canon.Recurso.EntradaRevision != nil) ||
			(revision.OrdenFirmaPDF == 2 && (canon.Recurso.EntradaRevision == nil ||
				canon.Recurso.EntradaRevision.Referencia != revision.EntradaDocumentoRef ||
				canon.Recurso.EntradaRevision.Version != revision.EntradaDocumentoVersion ||
				canon.Recurso.EntradaRevision.HuellaSHA256 != revision.EntradaDocumentoHuella)) {
			return nil, ports.ErrResultadoFirmaDocumentoInvalido
		}
		vistos[v.FirmaRef] = true
		salida = append(salida, ports.RecuperacionFirmaV2{FirmaRef: v.FirmaRef, MaterialRootSHA256: v.MaterialRootSHA256,
			CanonNominal: v.CanonNominal, CanonNominalSHA256: v.CanonNominalSHA256, CanonNominalRef: v.CanonNominalRef})
	}
	return salida, nil
}

func ordenRecuperacionCoincide(canon uint64, orden int) bool {
	return canon == 1 && orden == 1 || canon == 2 && orden == 2
}
