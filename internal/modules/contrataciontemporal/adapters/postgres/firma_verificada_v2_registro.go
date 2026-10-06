package postgres

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
)

var _ ports.RegistroFirmasVerificadasV2 = (*RegistroFirmasVerificadasPostgreSQL)(nil)

const registrarFirmaSQL172 = `SELECT vec_contratacion_temporal.registrar_firma_verificada_v2($1,$2::timestamptz,$3,$4,$5,$6,$7::numeric,$8::numeric,$9,$10,$11,$12,$13)::text`

// CT172 añade dos referencias nominales al recibo CT118 de doce claves.
// El DTO de CT118 permanece separado para las funciones anteriores.
type reciboFirmaSQLV2 struct {
	reciboFirmaSQL118
	CompetenciaEvidenciaRef          string `json:"CompetenciaEvidenciaRef"`
	CompetenciaEvidenciaHuellaSHA256 string `json:"CompetenciaEvidenciaHuellaSHA256"`
}

func (r *RegistroFirmasVerificadasPostgreSQL) RegistrarFirmaVerificadaV2(ctx context.Context, m ports.MaterialFirmaVerificadaV2, c ports.CapacidadFirmaVerificadaV2) (ports.ReciboFirmaDocumento, error) {
	var cero ports.ReciboFirmaDocumento
	if ctx == nil || r == nil || nuloRegistroTX(r.pool) {
		return cero, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	canon, err := m.Canonico()
	if err != nil || !domain.InstanteUTCCanonico(m.ComprobadaEn) {
		return cero, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	// AUT35 conserva cargo, enlace y delegación nominales. Estos tres campos
	// de la proyección anterior no están representados en su canon central.
	if m.PuestoFirmanteRef != "" || m.AmbitoFirmanteRef != "" || m.ActoCompetenciaRef != "" {
		return cero, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	if ctapp.ValidarCapacidadFirmaVerificadaV2(c, m) != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	datos, _ := c.ExportarDescriptorParaConsumidor()
	defer clear(datos)
	parametros, err := exportacionParametrosRegistroV2(c.ExportarMaterialParaConsumidor(), true)
	if err != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	defer limpiarParametrosFirma172(parametros)
	args := append([]any{string(canon), m.ComprobadaEn}, parametros...)
	args = append(args, datos)
	huella, err := m.HuellaSHA256()
	if err != nil {
		return cero, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	for intento := 1; intento <= intentosRegistroFirma118; intento++ {
		var recibo ports.ReciboFirmaDocumento
		recibo, err = r.registrarFirmaV2UnaVez(ctx, args, m, huella)
		if err == nil {
			return recibo, nil
		}
		if !reintentableFirma118(err) || ctx.Err() != nil {
			break
		}
	}
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) && pgerr.Code == "P1701" {
		return cero, ports.ErrFirmaDocumentoEnConflicto
	}
	if errors.Is(err, ports.ErrResultadoFirmaDocumentoInvalido) {
		return cero, err
	}
	return cero, errorFirma118(ctx, err)
}

func (r *RegistroFirmasVerificadasPostgreSQL) registrarFirmaV2UnaVez(ctx context.Context, args []any, m ports.MaterialFirmaVerificadaV2, huella string) (ports.ReciboFirmaDocumento, error) {
	var cero ports.ReciboFirmaDocumento
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return cero, err
	}
	if nuloRegistroTX(tx) {
		return cero, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	confirmado := false
	defer func() {
		if !confirmado {
			ctxRollback, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(2*time.Second))
			defer cancelar()
			if err := tx.Rollback(ctxRollback); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
				slog.Warn("contratacion temporal: rollback de registro de firma V2 no confirmado")
			}
		}
	}()
	if _, err = tx.Exec(ctx, ajustesRegistroIncorporacionTXV2); err != nil {
		return cero, err
	}
	var contenido []byte
	if err = tx.QueryRow(ctx, registrarFirmaSQL172, args...).Scan(&contenido); err != nil {
		return cero, err
	}
	defer clear(contenido)
	var w reciboFirmaSQLV2
	if err := decodificarFirma118(contenido, &w); err != nil {
		return cero, errors.Join(ports.ErrResultadoFirmaDocumentoInvalido, err)
	}
	recibo, err := proyectarReciboFirmaV2(w, m, huella)
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
		// Un COMMIT incierto no se reintenta ni se presenta como denegación.
		return cero, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	confirmado = true
	return recibo, nil
}

func proyectarReciboFirmaV2(w reciboFirmaSQLV2, m ports.MaterialFirmaVerificadaV2, huella string) (ports.ReciboFirmaDocumento, error) {
	var cero ports.ReciboFirmaDocumento
	if !strings.HasPrefix(w.CompetenciaEvidenciaRef, "evidencia:competencia-firmante-ct:") ||
		!domain.HuellaSHA256FirmaValida(strings.TrimPrefix(w.CompetenciaEvidenciaRef, "evidencia:competencia-firmante-ct:")) ||
		!domain.HuellaSHA256FirmaValida(w.CompetenciaEvidenciaHuellaSHA256) {
		return cero, ports.ErrResultadoFirmaDocumentoInvalido
	}
	recibo := ports.ReciboFirmaDocumento{FirmaRef: w.FirmaRef, ReciboRef: w.ReciboRef, Secuencia: w.Secuencia, Resultado: domain.ResultadoFirmaDocumento(w.Resultado),
		ExpedienteVersion: w.ExpedienteVersion, ActorRef: w.ActorRef, PerfilRef: w.PerfilRef, RegistradaEn: w.RegistradaEn.UTC(), SolicitudHuella: w.SolicitudHuella, YaRegistrada: w.YaRegistrada,
		DocumentoCustodiaRef: textoFirma118(w.DocumentoCustodia), DocumentoCustodiaVersion: versionFirma118(w.VersionCustodia)}
	if !domain.ReferenciaOpacaValida(recibo.FirmaRef) || !domain.ReferenciaOpacaValida(recibo.ReciboRef) || recibo.SolicitudHuella != huella ||
		recibo.Secuencia != m.Secuencia || recibo.ExpedienteVersion != m.VersionExpediente || recibo.Resultado != domain.ResultadoFirmaFirmado ||
		recibo.ActorRef == "" || (recibo.ActorRef == m.FirmantePrincipalRef) != (m.Via == ports.ViaFirmaCertificadoVEC) ||
		recibo.PerfilRef != m.PerfilActivoOperadorRef || recibo.RegistradaEn.IsZero() || recibo.DocumentoCustodiaRef != m.DocumentoCustodiaRef || recibo.DocumentoCustodiaVersion != m.DocumentoCustodiaVersion {
		return cero, ports.ErrResultadoFirmaDocumentoInvalido
	}
	return recibo, nil
}
