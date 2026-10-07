package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/consultafirmas"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// RegistroFirmasExternasPostgreSQL registra la vía externa en CT170 y lee la
// historia común CT118/CT170. El pool usa exclusivamente el LOGIN ejecutor CT.
type RegistroFirmasExternasPostgreSQL struct {
	pool iniciadorRegistroIncorporacionV2
}

var _ ports.RegistroFirmasExternas = (*RegistroFirmasExternasPostgreSQL)(nil)
var _ ports.RegistroFirmasVec = (*RegistroFirmasExternasPostgreSQL)(nil)

func NuevoRegistroFirmasExternasPostgreSQL(pool *pgxpool.Pool) (*RegistroFirmasExternasPostgreSQL, error) {
	if nuloRegistroTX(pool) {
		return nil, ports.ErrRegistroFirmaExternaNoDisponible
	}
	return &RegistroFirmasExternasPostgreSQL{pool: pool}, nil
}

const (
	registrarFirmaExternaSQL170 = `SELECT vec_contratacion_temporal.registrar_firma_verificada_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)::text`
	consultarFirmasSQL170       = `SELECT vec_contratacion_temporal.consultar_firmas_r5_atestadas_v1($1::text,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)::text`
)

type materialCanonicoFirma170 interface {
	Canonico() ([]byte, error)
	HuellaSHA256() (string, error)
}

type esperadoFirma170 struct {
	secuencia         int
	version           uint64
	firmantePrincipal string
	custodiaRef       string
	custodiaVersion   uint64
	actorEsFirmante   bool
}

func (r *RegistroFirmasExternasPostgreSQL) RegistrarFirmaExterna(ctx context.Context, m ports.MaterialFirmaExterna, c ports.CapacidadFirmaExterna) (ports.ReciboFirmaDocumento, error) {
	return r.registrarFirmaVerificada(ctx, m, c.ExportarMaterialParaConsumidor(), esperadoFirma170{
		secuencia: m.Secuencia, version: m.VersionExpediente, firmantePrincipal: m.FirmantePrincipalRef,
		custodiaRef: m.DocumentoCustodiaRef, custodiaVersion: m.DocumentoCustodiaVersion,
	}, ports.ErrRegistroFirmaExternaNoDisponible)
}

func (r *RegistroFirmasExternasPostgreSQL) RegistrarFirmaVec(ctx context.Context, m ports.MaterialFirmaVec, c ports.CapacidadFirmaVec) (ports.ReciboFirmaDocumento, error) {
	return r.registrarFirmaVerificada(ctx, m, c.ExportarMaterialParaConsumidor(), esperadoFirma170{
		secuencia: m.Secuencia, version: m.VersionExpediente, firmantePrincipal: m.FirmantePrincipalRef,
		custodiaRef: m.DocumentoCustodiaRef, custodiaVersion: m.DocumentoCustodiaVersion,
		actorEsFirmante: true,
	}, ports.ErrRegistroFirmaVecNoDisponible)
}

// registrarFirmaVerificada usa una única transacción para ambas vías. La
// capacidad no se interpreta en Go; SQL consume el perfil V3 nominal.
func (r *RegistroFirmasExternasPostgreSQL) registrarFirmaVerificada(ctx context.Context,
	m materialCanonicoFirma170, capacidad vp.ExportacionMaterialConsumoAutorizacionAtestadaV3,
	esperado esperadoFirma170, noDisponible error) (ports.ReciboFirmaDocumento, error) {
	var cero ports.ReciboFirmaDocumento
	if ctx == nil || r == nil || nuloRegistroTX(r.pool) {
		return cero, noDisponible
	}
	canonico, err := m.Canonico()
	if err != nil {
		return cero, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	h, err := m.HuellaSHA256()
	if err != nil {
		return cero, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	parametros, err := exportacionParametrosRegistroV2(capacidad, true)
	if err != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	defer func() {
		for _, v := range parametros {
			if b, ok := v.([]byte); ok {
				clear(b)
			}
		}
	}()
	args := append([]any{string(canonico)}, parametros...)
	var recibo ports.ReciboFirmaDocumento
	validar := func(w reciboFirmaSQL118) error {
		recibo = ports.ReciboFirmaDocumento{
			FirmaRef: w.FirmaRef, ReciboRef: w.ReciboRef, Secuencia: w.Secuencia,
			Resultado: domain.ResultadoFirmaDocumento(w.Resultado), ExpedienteVersion: w.ExpedienteVersion,
			ActorRef: w.ActorRef, PerfilRef: w.PerfilRef, RegistradaEn: w.RegistradaEn.UTC(),
			SolicitudHuella: w.SolicitudHuella, YaRegistrada: w.YaRegistrada,
			DocumentoCustodiaRef:     textoFirma118(w.DocumentoCustodia),
			DocumentoCustodiaVersion: versionFirma118(w.VersionCustodia),
		}
		if !domain.ReferenciaOpacaValida(recibo.FirmaRef) || !domain.ReferenciaOpacaValida(recibo.ReciboRef) ||
			recibo.SolicitudHuella != h || recibo.Secuencia != esperado.secuencia ||
			recibo.ExpedienteVersion != esperado.version ||
			recibo.Resultado != domain.ResultadoFirmaFirmado || recibo.ActorRef == "" ||
			(recibo.ActorRef == esperado.firmantePrincipal) != esperado.actorEsFirmante ||
			recibo.PerfilRef == "" || recibo.RegistradaEn.IsZero() ||
			recibo.DocumentoCustodiaRef != esperado.custodiaRef ||
			recibo.DocumentoCustodiaVersion != esperado.custodiaVersion {
			return ports.ErrResultadoFirmaDocumentoInvalido
		}
		return nil
	}
	for intento := 1; intento <= intentosRegistroFirma118; intento++ {
		err = r.registrarFirmaExternaUnaVez(ctx, args, validar)
		if err == nil {
			return recibo, nil
		}
		if err == ports.ErrResultadoFirmaDocumentoInvalido {
			return cero, err
		}
		if !reintentableFirma118(err) || intento == intentosRegistroFirma118 || ctx.Err() != nil {
			break
		}
	}
	if errContexto := ctx.Err(); errContexto != nil {
		return cero, errContexto
	}
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) && pgerr.Code == "P1701" {
		return cero, ports.ErrFirmaDocumentoEnConflicto
	}
	traducido := errorFirma118(ctx, err)
	if errors.Is(traducido, ports.ErrRegistroFirmaDocumentoNoDisponible) {
		return cero, noDisponible
	}
	return cero, traducido
}

// registrarFirmaExternaUnaVez confirma únicamente después de validar la
// estructura del recibo. Todo intento perdido se revierte antes de repetir.
func (r *RegistroFirmasExternasPostgreSQL) registrarFirmaExternaUnaVez(ctx context.Context, args []any, validar func(reciboFirmaSQL118) error) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return err
	}
	if nuloRegistroTX(tx) {
		return ports.ErrRegistroFirmaExternaNoDisponible
	}
	confirmado := false
	defer func() {
		if !confirmado {
			c, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(2*time.Second))
			defer cancelar()
			_ = tx.Rollback(c)
		}
	}()
	if _, err = tx.Exec(ctx, ajustesRegistroIncorporacionTXV2); err != nil {
		return err
	}
	var contenido []byte
	if err = tx.QueryRow(ctx, registrarFirmaExternaSQL170, args...).Scan(&contenido); err != nil {
		return err
	}
	var w reciboFirmaSQL118
	if err = decodificarFirma118(contenido, &w); err != nil {
		return err
	}
	if err = validar(w); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	confirmado = true
	return nil
}

type firmaExternaSQL170 struct {
	firmaSQL118
	Via                            *string `json:"Via"`
	OriginalRef                    *string `json:"OriginalRef"`
	OriginalVersion                *uint64 `json:"OriginalVersion"`
	FirmantePrincipalAcreditado    bool    `json:"FirmantePrincipalAcreditado"`
	CoincideFirmanteCandidato      bool    `json:"CoincideFirmanteCandidato"`
	HistoriaRevision               *uint64 `json:"HistoriaRevision"`
	HistoriaHuella                 *string `json:"HistoriaHuella"`
	ReferenciaPortafirmasDeclarada *string `json:"ReferenciaPortafirmasDeclarada"`
	FechaPortafirmasDeclarada      *string `json:"FechaPortafirmasDeclarada"`
}

type respuestaFirmasR5SQL170 struct {
	Encontrado                   *bool                `json:"Encontrado"`
	ExpedienteRef                string               `json:"ExpedienteRef"`
	Firmas                       []firmaExternaSQL170 `json:"Firmas"`
	HistoriaRevision             *uint64              `json:"HistoriaRevision"`
	HistoriaHuella               string               `json:"HistoriaHuella"`
	CoincideFirmanteEnOtroPaso   *bool                `json:"CoincideFirmanteEnOtroPaso"`
	HistoriaSeparacionAcreditada *bool                `json:"HistoriaSeparacionAcreditada"`
}

// ConsultarFirmasAutorizadas consume AD159 y confirma la auditoría V3 antes de
// devolver la historia del documento. La proyección interna v3 no es ejecutable
// por el LOGIN técnico. NULL histórico en Via nunca acredita un hito R5.
func (r *RegistroFirmasExternasPostgreSQL) ConsultarFirmasAutorizadas(ctx context.Context,
	m ports.MaterialConsultaFirmasR5, c ports.CapacidadConsultaFirmasR5) (ports.LecturaFirmasR5, error) {
	var cero ports.LecturaFirmasR5
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
	if err = consultafirmas.ValidarCapacidadConsultaFirmasR5(c, m); err != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	parametros, err := exportacionParametrosRegistroV2(c.ExportarMaterialParaConsumidor(), true)
	if err != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	defer func() {
		for _, v := range parametros {
			if b, ok := v.([]byte); ok {
				clear(b)
			}
		}
	}()
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || nuloRegistroTX(tx) {
		return cero, errorFirma118(ctx, err)
	}
	confirmado := false
	defer func() {
		if !confirmado {
			c, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(2*time.Second))
			defer cancelar()
			_ = tx.Rollback(c)
		}
	}()
	if _, err = tx.Exec(ctx, ajustesRegistroIncorporacionTXV2); err != nil {
		return cero, errorFirma118(ctx, err)
	}
	var contenido []byte
	if err = tx.QueryRow(ctx, consultarFirmasSQL170, append([]any{string(canonico)}, parametros...)...).Scan(&contenido); err != nil {
		return cero, errorFirma118(ctx, err)
	}
	defer clear(contenido)
	var w respuestaFirmasR5SQL170
	if decodificarFirma118(contenido, &w) != nil || w.Encontrado == nil || w.ExpedienteRef != m.ExpedienteRef || w.Firmas == nil ||
		w.HistoriaRevision == nil || *w.HistoriaRevision > 9007199254740991 ||
		w.CoincideFirmanteEnOtroPaso == nil || w.HistoriaSeparacionAcreditada == nil ||
		!domain.HuellaSHA256FirmaValida(w.HistoriaHuella) ||
		(!*w.Encontrado && len(w.Firmas) != 0) {
		return cero, ports.ErrResultadoFirmaDocumentoInvalido
	}
	replayExacto := len(w.Firmas) == 1 &&
		(textoFirma118(w.Firmas[0].Via) == ports.ViaFirmaExternaPortafirmas ||
			textoFirma118(w.Firmas[0].Via) == ports.ViaFirmaCertificadoVEC) &&
		w.Firmas[0].ClaveIdempotencia == m.ClaveIdempotencia &&
		w.Firmas[0].Documento == m.Documento && w.Firmas[0].ExpedienteVersion == m.VersionExpediente
	if replayExacto && (*w.CoincideFirmanteEnOtroPaso || *w.HistoriaSeparacionAcreditada) {
		return cero, ports.ErrResultadoFirmaDocumentoInvalido
	}
	firmas := make([]ports.FirmaRegistrada, 0, len(w.Firmas))
	for _, f := range w.Firmas {
		via := textoFirma118(f.Via)
		if f.Documento != m.Documento || (via != "" && via != ports.ViaFirmaExternaPortafirmas && via != ports.ViaFirmaCertificadoVEC) ||
			(via == "" && (f.FirmantePrincipalAcreditado || f.CoincideFirmanteCandidato || f.HistoriaRevision != nil || f.HistoriaHuella != nil)) ||
			(via != "" && (!f.FirmantePrincipalAcreditado || f.HistoriaRevision == nil || f.HistoriaHuella == nil ||
				!domain.HuellaSHA256FirmaValida(textoFirma118(f.HistoriaHuella)) ||
				(replayExacto && (*f.HistoriaRevision != *w.HistoriaRevision ||
					textoFirma118(f.HistoriaHuella) != w.HistoriaHuella)) ||
				(!replayExacto && *f.HistoriaRevision >= *w.HistoriaRevision))) ||
			(f.CoincideFirmanteCandidato && !f.FirmantePrincipalAcreditado) ||
			!domain.ReferenciaOpacaValida(f.FirmaRef) || !domain.ReferenciaOpacaValida(f.ReciboRef) ||
			f.Secuencia < 1 || f.ExpedienteVersion == 0 || f.ExpedienteVersion > m.VersionExpediente ||
			f.RegistradaEn.IsZero() {
			return cero, ports.ErrResultadoFirmaDocumentoInvalido
		}
		firmas = append(firmas, ports.FirmaRegistrada{
			Via: via, FirmaRef: f.FirmaRef, ReciboRef: f.ReciboRef, Documento: f.Documento,
			FirmantePrincipalAcreditado: f.FirmantePrincipalAcreditado,
			CoincideFirmanteCandidato:   f.CoincideFirmanteCandidato,
			HistoriaRevision:            versionFirma118(f.HistoriaRevision),
			HistoriaHuella:              textoFirma118(f.HistoriaHuella),
			Secuencia:                   f.Secuencia, ExpedienteVersion: f.ExpedienteVersion,
			CatalogoRef: f.CatalogoRef, CatalogoHuella: f.CatalogoHuella,
			PasoRef: f.PasoRef, PasoOrden: f.PasoOrden, Resultado: domain.ResultadoFirmaDocumento(f.Resultado),
			ConMotivoDevolucion: f.ConMotivo, OriginalHuella: textoFirma118(f.OriginalHuella),
			OriginalRef: textoFirma118(f.OriginalRef), OriginalVersion: versionFirma118(f.OriginalVersion),
			FirmadoHuella: textoFirma118(f.FirmadoHuella), SelloTiempoEstado: textoFirma118(f.SelloTiempoEstado),
			ReferenciaPortafirmasDeclarada: textoFirma118(f.ReferenciaPortafirmasDeclarada),
			FechaPortafirmasDeclarada:      textoFirma118(f.FechaPortafirmasDeclarada),
			RegistradaEn:                   f.RegistradaEn.UTC(), ClaveIdempotencia: f.ClaveIdempotencia,
			DocumentoCustodiaRef:     textoFirma118(f.DocumentoCustodia),
			DocumentoCustodiaVersion: versionFirma118(f.VersionCustodia),
		})
	}
	if err = ctx.Err(); err != nil {
		return cero, err
	}
	if err = tx.Commit(ctx); err != nil {
		return cero, errorFirma118(ctx, err)
	}
	confirmado = true
	if !*w.Encontrado {
		return cero, ports.ErrExpedienteConsultaFirmasNoEncontrado
	}
	return ports.LecturaFirmasR5{Firmas: firmas, HistoriaRevision: *w.HistoriaRevision,
		HistoriaHuella: w.HistoriaHuella, CoincideFirmanteEnOtroPaso: *w.CoincideFirmanteEnOtroPaso,
		HistoriaSeparacionAcreditada: *w.HistoriaSeparacionAcreditada}, nil
}
