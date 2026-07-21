package postgres

import (
	"context"
	"encoding/hex"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
)

type filaConfirmacionBorradorPostgreSQL struct {
	resultado, estadoDiario string
	revision, cercado       pgtype.Int8
	transaccionRef, accion  pgtype.Text
	estadoPrincipalRef      pgtype.Text
	estadoPrincipalRevision pgtype.Int8
	estadoPrincipalHuella   pgtype.Text
	auditoriaRef            pgtype.Text
	huellaAuditoria         pgtype.Text
	eventoOutboxRef         pgtype.Text
	huellaEventoOutbox      pgtype.Text
	confirmadaEn            pgtype.Timestamptz
	preparadaEn             pgtype.Timestamptz
	recibo                  []byte
	requiereRevalidacion    bool
	preparacionRef          pgtype.Text
	cuerpoRecibo            []byte
	cuerpoCanonico          []byte
	huellaCuerpo            pgtype.Text
}

func (f *filaConfirmacionBorradorPostgreSQL) borrar() {
	borrarBytesDiarioPostgreSQL(f.recibo, f.cuerpoRecibo, f.cuerpoCanonico)
}

func prepararConfirmacionBorradorPostgreSQL(
	ctx context.Context, tx pgx.Tx, carga cargaConfirmacionBorradorPostgreSQL,
) (filaConfirmacionBorradorPostgreSQL, error) {
	var f filaConfirmacionBorradorPostgreSQL
	err := tx.QueryRow(ctx, `
		SELECT resultado, estado_diario, revision_diario, cercado,
		       transaccion_ref, accion, estado_principal_ref,
		       estado_principal_revision, estado_principal_huella_sha256,
		       auditoria_ref, huella_auditoria_sha256, evento_outbox_ref,
		       huella_evento_outbox_sha256, confirmada_en, recibo,
		       requiere_revalidacion_kms, preparacion_ref, recibo_cuerpo,
		       cuerpo_recibo_canonico, huella_cuerpo_recibo_sha256,
		       preparada_en
		FROM `+funcionPrepararConfirmacionBorradorPostgreSQL+`(
			$1::jsonb, $2::jsonb, $3::jsonb, $4::bytea, $5::bytea,
			$6::bytea, $7::bytea, $8::bytea, $9::bytea, $10::bytea, $11::bytea
		)`,
		carga.Confirmacion, carga.Prueba, carga.Evidencia, carga.Decision, carga.Contexto,
		carga.Material, carga.Version, carga.AAD, carga.MaterialEnvuelto, carga.Nonce,
		carga.TextoCifrado,
	).Scan(
		&f.resultado, &f.estadoDiario, &f.revision, &f.cercado,
		&f.transaccionRef, &f.accion, &f.estadoPrincipalRef, &f.estadoPrincipalRevision,
		&f.estadoPrincipalHuella, &f.auditoriaRef, &f.huellaAuditoria,
		&f.eventoOutboxRef, &f.huellaEventoOutbox, &f.confirmadaEn, &f.recibo,
		&f.requiereRevalidacion, &f.preparacionRef, &f.cuerpoRecibo,
		&f.cuerpoCanonico, &f.huellaCuerpo, &f.preparadaEn,
	)
	return f, err
}

func ejecutarFaseBConfirmacionBorradorPostgreSQL(
	ctx context.Context, tx pgx.Tx, preparacionRef string, acreditacion, cuerpoCanonico []byte,
) (filaConfirmacionBorradorPostgreSQL, error) {
	var f filaConfirmacionBorradorPostgreSQL
	err := tx.QueryRow(ctx, `
		SELECT resultado, estado_diario, revision_diario, cercado,
		       transaccion_ref, accion, estado_principal_ref,
		       estado_principal_revision, estado_principal_huella_sha256,
		       auditoria_ref, huella_auditoria_sha256, evento_outbox_ref,
		       huella_evento_outbox_sha256, confirmada_en, recibo
		FROM `+funcionConfirmarBorradorPostgreSQL+`($1::text, $2::jsonb, $3::bytea)`,
		preparacionRef, acreditacion, cuerpoCanonico,
	).Scan(
		&f.resultado, &f.estadoDiario, &f.revision, &f.cercado,
		&f.transaccionRef, &f.accion, &f.estadoPrincipalRef, &f.estadoPrincipalRevision,
		&f.estadoPrincipalHuella, &f.auditoriaRef, &f.huellaAuditoria,
		&f.eventoOutboxRef, &f.huellaEventoOutbox, &f.confirmadaEn, &f.recibo,
	)
	return f, err
}

func (f filaConfirmacionBorradorPostgreSQL) restaurarConfirmada(
	s gobiernoconvocatorias.SolicitudConfirmacionBorrador,
) (gobiernoconvocatorias.ResultadoConfirmacionAtomica, error) {
	if (f.resultado != "confirmada" && f.resultado != "idempotencia_reutilizada") ||
		f.estadoDiario != string(gobiernoconvocatorias.ResultadoDiarioConfirmado) ||
		f.requiereRevalidacion || f.preparacionRef.Valid || len(f.recibo) == 0 ||
		len(f.cuerpoRecibo) != 0 || len(f.cuerpoCanonico) != 0 || f.huellaCuerpo.Valid ||
		f.preparadaEn.Valid {
		return gobiernoconvocatorias.ResultadoConfirmacionAtomica{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	recibo, err := restaurarReciboBorradorPostgreSQL(f.recibo)
	if err != nil || recibo == nil || f.validarMetadatosRecibo(*recibo) != nil {
		return gobiernoconvocatorias.ResultadoConfirmacionAtomica{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	resultado := gobiernoconvocatorias.ResultadoConfirmacionAtomica{
		Estado: gobiernoconvocatorias.ResultadoDiarioConfirmado,
		Recibo: *recibo, AcreditacionKMS: recibo.AcreditacionKMS,
	}
	if resultado.ValidarPara(s) != nil {
		return gobiernoconvocatorias.ResultadoConfirmacionAtomica{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return resultado, nil
}

func (f filaConfirmacionBorradorPostgreSQL) restaurarPreparada(
	s gobiernoconvocatorias.SolicitudConfirmacionBorrador,
) (gobiernoconvocatorias.ProyeccionReciboBorrador, string, time.Time, error) {
	if f.resultado != "preparada" || f.estadoDiario != string(gobiernoconvocatorias.ResultadoDiarioEnCurso) ||
		!f.requiereRevalidacion || !f.preparacionRef.Valid || !preparacionRefPostgreSQLValida(f.preparacionRef.String) ||
		len(f.recibo) != 0 || len(f.cuerpoRecibo) == 0 || len(f.cuerpoCanonico) == 0 || !f.huellaCuerpo.Valid ||
		huellaBytesDiarioPostgreSQL(f.cuerpoCanonico) != f.huellaCuerpo.String {
		return gobiernoconvocatorias.ProyeccionReciboBorrador{}, "", time.Time{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	preparada, errPreparada := restaurarInstanteObligatorioDiarioPostgreSQL(f.preparadaEn)
	confirmada, errConfirmada := restaurarInstanteObligatorioDiarioPostgreSQL(f.confirmadaEn)
	recibo, err := restaurarCuerpoReciboBorradorPostgreSQL(f.cuerpoRecibo)
	huella, errHuella := recibo.HuellaCuerpoParaRevalidacion()
	if err != nil || errHuella != nil || errPreparada != nil || errConfirmada != nil ||
		!ventanaRevalidacionKMSPostgreSQLValida(preparada, confirmada) ||
		huella != f.huellaCuerpo.String ||
		f.validarMetadatosRecibo(recibo) != nil || !reciboPreparadoCoincideSolicitudPostgreSQL(recibo, s) {
		return gobiernoconvocatorias.ProyeccionReciboBorrador{}, "", time.Time{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return recibo, huella, preparada, nil
}

func (f filaConfirmacionBorradorPostgreSQL) restaurarSinPreparacion() (
	gobiernoconvocatorias.ResultadoConfirmacionAtomica, error,
) {
	if f.requiereRevalidacion || f.preparacionRef.Valid || len(f.recibo) != 0 ||
		len(f.cuerpoRecibo) != 0 || len(f.cuerpoCanonico) != 0 || f.huellaCuerpo.Valid ||
		f.preparadaEn.Valid {
		return gobiernoconvocatorias.ResultadoConfirmacionAtomica{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	if f.estadoDiario != string(gobiernoconvocatorias.ResultadoDiarioNoAplicado) ||
		(f.resultado != "idempotencia_reutilizada" && f.resultado != "conflicto_cas") ||
		!f.accion.Valid || f.transaccionRef.Valid || f.estadoPrincipalRef.Valid ||
		f.estadoPrincipalRevision.Valid || f.estadoPrincipalHuella.Valid || f.auditoriaRef.Valid ||
		f.huellaAuditoria.Valid || f.eventoOutboxRef.Valid || f.huellaEventoOutbox.Valid {
		return gobiernoconvocatorias.ResultadoConfirmacionAtomica{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	if _, err := enteroPositivoConfirmacionPostgreSQL(f.revision); err != nil {
		return gobiernoconvocatorias.ResultadoConfirmacionAtomica{}, err
	}
	if _, err := enteroPositivoConfirmacionPostgreSQL(f.cercado); err != nil {
		return gobiernoconvocatorias.ResultadoConfirmacionAtomica{}, err
	}
	if _, err := restaurarInstanteObligatorioDiarioPostgreSQL(f.confirmadaEn); err != nil {
		return gobiernoconvocatorias.ResultadoConfirmacionAtomica{}, err
	}
	return resultadoNoAplicadoBorradorPostgreSQL(), nil
}

func (f filaConfirmacionBorradorPostgreSQL) validarMetadatosRecibo(
	r gobiernoconvocatorias.ProyeccionReciboBorrador,
) error {
	revision, errRevision := enteroPositivoConfirmacionPostgreSQL(f.revision)
	cercado, errCercado := enteroPositivoConfirmacionPostgreSQL(f.cercado)
	confirmada, errConfirmada := restaurarInstanteObligatorioDiarioPostgreSQL(f.confirmadaEn)
	if errRevision != nil || errCercado != nil || errConfirmada != nil ||
		!f.transaccionRef.Valid || !f.accion.Valid || !f.estadoPrincipalRef.Valid ||
		!f.estadoPrincipalRevision.Valid || f.estadoPrincipalRevision.Int64 < 1 ||
		f.estadoPrincipalRevision.Int64 > math.MaxInt || !f.estadoPrincipalHuella.Valid ||
		!f.auditoriaRef.Valid || !f.huellaAuditoria.Valid || !f.eventoOutboxRef.Valid ||
		!f.huellaEventoOutbox.Valid || revision != r.RevisionConfirmada || cercado != r.CercadoConfirmado ||
		f.transaccionRef.String != r.TransaccionRef || f.accion.String != r.Accion ||
		f.estadoPrincipalRef.String != r.EstadoPrincipal.Referencia ||
		int(f.estadoPrincipalRevision.Int64) != r.EstadoPrincipal.Revision ||
		f.estadoPrincipalHuella.String != r.EstadoPrincipal.HuellaEstadoSHA256 ||
		f.auditoriaRef.String != r.AuditoriaRef || f.huellaAuditoria.String != r.HuellaAuditoriaSHA256 ||
		f.eventoOutboxRef.String != r.EventoOutboxRef ||
		f.huellaEventoOutbox.String != r.HuellaEventoOutboxSHA256 || !confirmada.Equal(r.ConfirmadaEn) {
		return gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return nil
}

func reciboPreparadoCoincideSolicitudPostgreSQL(
	r gobiernoconvocatorias.ProyeccionReciboBorrador,
	s gobiernoconvocatorias.SolicitudConfirmacionBorrador,
) bool {
	return r.AcreditacionKMS == (gobiernoconvocatorias.AcreditacionKMSConfirmacionBorrador{}) &&
		r.IdentidadPrimaria == s.Reserva.IdentidadPrimaria && r.Decision == s.Reserva.Decision &&
		r.SelladoMotivo == s.SelladoMotivo && r.Accion == s.Material.Accion &&
		r.EstadoPrincipal == s.Material.EstadoPrincipalNuevo && r.RevisionConfirmada > s.Control.Revision &&
		r.CercadoConfirmado == s.Control.Cercado &&
		r.ArrendamientoIniciaEn.Equal(s.Control.ArrendamientoIniciaEn) &&
		r.ArrendamientoVenceEn.Equal(s.Control.ArrendamientoVenceEn) && r.Procedencia == s.Procedencia
}

func preparacionRefPostgreSQLValida(valor string) bool {
	const prefijo = "preparacion-kms-borrador-"
	if !strings.HasPrefix(valor, prefijo) || len(valor) != len(prefijo)+64 {
		return false
	}
	_, err := hex.DecodeString(valor[len(prefijo):])
	return err == nil && strings.ToLower(valor) == valor
}
