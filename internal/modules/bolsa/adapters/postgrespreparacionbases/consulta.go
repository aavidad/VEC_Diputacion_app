package postgrespreparacionbases

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	prep "vec-diputacion-granada/internal/modules/bolsa/domain/preparacionbases"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
)

const ajustes = `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),set_config('idle_in_transaction_session_timeout','20s',true)`

type filaPreparacion struct {
	Estado          string
	Referencia      *string
	Revision        *int
	HuellaMaterial  *string
	Material        []byte
	Recibo          *string
	Historia        *string
	AuditoriaEfecto *string
	Evento          *string
	HuellaIntencion *string
	ConfirmadaEn    *time.Time
	Acceso          ports.EvidenciaAccesoPreparacionBasesV3
}

func (r *Repositorio) ejecutar(ctx context.Context, pool iniciador, consulta string, ps []any, ambito bolsa.AmbitoOrganizativoConvocatoria, validar func(ports.ResultadoPreparacionBasesV3) error) (ports.ResultadoPreparacionBasesV3, error) {
	var cero ports.ResultadoPreparacionBasesV3
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return cero, err
	}
	if nulo(tx) {
		return cero, ports.ErrPreparacionBasesNoDisponible
	}
	defer func() {
		fin, cancelar := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(2*time.Second))
		defer cancelar()
		_ = tx.Rollback(fin)
	}()
	if _, err := tx.Exec(ctx, ajustes); err != nil {
		return cero, err
	}
	var f filaPreparacion
	a := &f.Acceso
	if err := tx.QueryRow(ctx, consulta, ps...).Scan(&f.Estado, &f.Referencia, &f.Revision, &f.HuellaMaterial, &f.Material, &f.Recibo, &f.Historia, &f.AuditoriaEfecto, &f.Evento, &f.HuellaIntencion, &f.ConfirmadaEn, &a.DecisionRef, &a.ConsumoHuellaSHA256, &a.AuditoriaRef, &a.ReciboRef, &a.CorrelacionRef, &a.AccedidaEn); err != nil {
		return cero, err
	}
	defer limpiarParametros([]any{f.Material})
	resultado, err := f.resultado(ambito)
	if err != nil {
		return cero, err
	}
	if validar(resultado) != nil {
		return cero, ports.ErrResultadoPreparacionBasesInvalido
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if err := tx.Commit(ctx); err != nil {
		return cero, err
	}
	return resultado, nil
}

func (f filaPreparacion) resultado(ambito bolsa.AmbitoOrganizativoConvocatoria) (ports.ResultadoPreparacionBasesV3, error) {
	r := ports.ResultadoPreparacionBasesV3{Estado: f.Estado, Acceso: f.Acceso}
	r.Acceso.AccedidaEn = r.Acceso.AccedidaEn.UTC()
	if f.Estado == "version_en_conflicto" || f.Estado == "clave_reutilizada" || f.Estado == "no_encontrada" {
		if f.Referencia != nil || f.Revision != nil || f.HuellaMaterial != nil || f.Material != nil || f.Recibo != nil ||
			f.Historia != nil || f.AuditoriaEfecto != nil || f.Evento != nil || f.HuellaIntencion != nil || f.ConfirmadaEn != nil {
			return ports.ResultadoPreparacionBasesV3{}, ports.ErrResultadoPreparacionBasesInvalido
		}
		return r, nil
	}
	if f.Referencia == nil || f.Revision == nil || f.HuellaMaterial == nil || f.Material == nil || f.Recibo == nil ||
		f.Historia == nil || f.AuditoriaEfecto == nil || f.Evento == nil || f.HuellaIntencion == nil || f.ConfirmadaEn == nil {
		return ports.ResultadoPreparacionBasesV3{}, ports.ErrResultadoPreparacionBasesInvalido
	}
	m, err := prep.DecodificarMaterialCanonico(f.Material)
	if err != nil {
		return ports.ResultadoPreparacionBasesV3{}, ports.ErrResultadoPreparacionBasesInvalido
	}
	r.Version = prep.Version{Ambito: ambito, Estado: prep.Esperada{PreparacionRef: *f.Referencia, Revision: *f.Revision, HuellaMaterialSHA256: *f.HuellaMaterial}, Material: m}
	r.Recibo = ports.ReciboPreparacionBases{ReciboRef: *f.Recibo, HistoriaRef: *f.Historia, AuditoriaRef: *f.AuditoriaEfecto, EventoRef: *f.Evento, HuellaIntencionSHA256: *f.HuellaIntencion, ConfirmadaEn: f.ConfirmadaEn.UTC()}
	return r, nil
}
