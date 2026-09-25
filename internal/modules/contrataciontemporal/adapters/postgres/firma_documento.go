package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// RegistroFirmasDocumentoPostgreSQL escribe y lee el registro CT118 con el
// LOGIN ejecutor de CT. La escritura es una transacción SERIALIZABLE en la
// que CT118 consume la autorización V3 (AD3-85) y guarda firma, auditoría y
// outbox; si algo falla no queda nada.
type RegistroFirmasDocumentoPostgreSQL struct {
	pool iniciadorRegistroIncorporacionV2
}

var _ ports.RegistroFirmasDocumento = (*RegistroFirmasDocumentoPostgreSQL)(nil)

func NuevoRegistroFirmasDocumentoPostgreSQL(pool *pgxpool.Pool) (*RegistroFirmasDocumentoPostgreSQL, error) {
	if nuloRegistroTX(pool) {
		return nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	return &RegistroFirmasDocumentoPostgreSQL{pool: pool}, nil
}

const (
	registrarFirmaSQL118     = `SELECT vec_contratacion_temporal.registrar_firma_documento_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)::text`
	consultarFirmasSQL118    = `SELECT vec_contratacion_temporal.consultar_firmas_documento_v1($1,$2)::text`
	maximoRespuestaFirmas118 = 4 << 20
)

type reciboFirmaSQL118 struct {
	FirmaRef          string    `json:"FirmaRef"`
	ReciboRef         string    `json:"ReciboRef"`
	Secuencia         int       `json:"Secuencia"`
	Resultado         string    `json:"Resultado"`
	ExpedienteVersion uint64    `json:"ExpedienteVersion"`
	ActorRef          string    `json:"ActorRef"`
	PerfilRef         string    `json:"PerfilRef"`
	RegistradaEn      time.Time `json:"RegistradaEn"`
	SolicitudHuella   string    `json:"SolicitudHuella"`
	YaRegistrada      bool      `json:"YaRegistrada"`
}

type firmaSQL118 struct {
	FirmaRef          string    `json:"FirmaRef"`
	ReciboRef         string    `json:"ReciboRef"`
	Documento         string    `json:"Documento"`
	Secuencia         int       `json:"Secuencia"`
	ExpedienteVersion uint64    `json:"ExpedienteVersion"`
	CatalogoRef       string    `json:"CatalogoRef"`
	CatalogoHuella    string    `json:"CatalogoHuella"`
	PasoRef           string    `json:"PasoRef"`
	PasoOrden         int       `json:"PasoOrden"`
	Resultado         string    `json:"Resultado"`
	MotivoDevolucion  *string   `json:"MotivoDevolucion"`
	OriginalHuella    *string   `json:"OriginalHuella"`
	FirmadoHuella     *string   `json:"FirmadoHuella"`
	SelloTiempoEstado *string   `json:"SelloTiempoEstado"`
	RegistradaEn      time.Time `json:"RegistradaEn"`
}

func textoFirma118(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func errorFirma118(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var p *pgconn.PgError
	if errors.As(err, &p) {
		switch p.Code {
		case "P1181":
			return ports.ErrClaveFirmaDocumentoUsada
		case "P1182", "P1183", "40001":
			return ports.ErrFirmaDocumentoEnConflicto
		case "P1184":
			return ports.ErrCadenaFirmaDocumentoRota
		case "42501", "P1102":
			return ports.ErrFirmaDocumentoDenegada
		case "22023":
			return ports.ErrSolicitudFirmaDocumentoInvalida
		}
	}
	return ports.ErrRegistroFirmaDocumentoNoDisponible
}

func decodificarFirma118(b []byte, v any) error {
	if len(b) == 0 || len(b) > maximoRespuestaFirmas118 {
		return ports.ErrResultadoFirmaDocumentoInvalido
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil || d.Decode(&struct{}{}) != io.EOF {
		return ports.ErrResultadoFirmaDocumentoInvalido
	}
	return nil
}

// RegistrarFirma consume la capacidad y escribe en una sola transacción.
func (r *RegistroFirmasDocumentoPostgreSQL) RegistrarFirma(ctx context.Context, m ports.MaterialFirmaDocumento, c ports.CapacidadFirmaDocumento) (ports.ReciboFirmaDocumento, error) {
	var cero ports.ReciboFirmaDocumento
	if ctx == nil || r == nil || nuloRegistroTX(r.pool) {
		return cero, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	canonico, err := m.Canonico()
	if err != nil {
		return cero, ports.ErrSolicitudFirmaDocumentoInvalida
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
			c, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancelar()
			_ = tx.Rollback(c)
		}
	}()
	if _, err = tx.Exec(ctx, ajustesRegistroIncorporacionTXV2); err != nil {
		return cero, errorFirma118(ctx, err)
	}
	var contenido []byte
	args := append([]any{string(canonico)}, parametros...)
	if err = tx.QueryRow(ctx, registrarFirmaSQL118, args...).Scan(&contenido); err != nil {
		return cero, errorFirma118(ctx, err)
	}
	var w reciboFirmaSQL118
	if decodificarFirma118(contenido, &w) != nil {
		return cero, ports.ErrResultadoFirmaDocumentoInvalido
	}
	h, _ := m.HuellaSHA256()
	recibo := ports.ReciboFirmaDocumento{FirmaRef: w.FirmaRef, ReciboRef: w.ReciboRef, Secuencia: w.Secuencia,
		Resultado: domain.ResultadoFirmaDocumento(w.Resultado), ExpedienteVersion: w.ExpedienteVersion, ActorRef: w.ActorRef,
		PerfilRef: w.PerfilRef, RegistradaEn: w.RegistradaEn.UTC(), SolicitudHuella: w.SolicitudHuella, YaRegistrada: w.YaRegistrada}
	if !domain.ReferenciaOpacaValida(recibo.FirmaRef) || !domain.ReferenciaOpacaValida(recibo.ReciboRef) || recibo.SolicitudHuella != h ||
		recibo.Resultado != m.Resultado || recibo.ActorRef == "" || recibo.PerfilRef == "" || recibo.RegistradaEn.IsZero() ||
		(!recibo.YaRegistrada && (recibo.Secuencia != m.Secuencia || recibo.ExpedienteVersion != m.VersionExpediente)) {
		return cero, ports.ErrResultadoFirmaDocumentoInvalido
	}
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	if err = tx.Commit(ctx); err != nil {
		return cero, errorFirma118(ctx, err)
	}
	confirmado = true
	return recibo, nil
}

// ConsultarFirmas lee la historia del expediente en orden de documento y
// secuencia. CT118 no devuelve quién firmó (firmante, certificado, actor ni
// perfil): el estado del circuito no lo necesita.
func (r *RegistroFirmasDocumentoPostgreSQL) ConsultarFirmas(ctx context.Context, organizacionRef, expedienteRef string) ([]ports.FirmaRegistrada, error) {
	if ctx == nil || r == nil || nuloRegistroTX(r.pool) {
		return nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	if !domain.ReferenciaOpacaValida(organizacionRef) || !domain.ReferenciaOpacaValida(expedienteRef) {
		return nil, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil || nuloRegistroTX(tx) {
		return nil, errorFirma118(ctx, err)
	}
	defer func() {
		c, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelar()
		_ = tx.Rollback(c)
	}()
	var contenido []byte
	if err = tx.QueryRow(ctx, consultarFirmasSQL118, organizacionRef, expedienteRef).Scan(&contenido); err != nil {
		return nil, errorFirma118(ctx, err)
	}
	var filas []firmaSQL118
	if decodificarFirma118(contenido, &filas) != nil {
		return nil, ports.ErrResultadoFirmaDocumentoInvalido
	}
	firmas := make([]ports.FirmaRegistrada, 0, len(filas))
	for _, f := range filas {
		firmas = append(firmas, ports.FirmaRegistrada{FirmaRef: f.FirmaRef, ReciboRef: f.ReciboRef, Documento: f.Documento,
			Secuencia: f.Secuencia, ExpedienteVersion: f.ExpedienteVersion, CatalogoRef: f.CatalogoRef, CatalogoHuella: f.CatalogoHuella,
			PasoRef: f.PasoRef, PasoOrden: f.PasoOrden, Resultado: domain.ResultadoFirmaDocumento(f.Resultado),
			MotivoDevolucion: textoFirma118(f.MotivoDevolucion), OriginalHuella: textoFirma118(f.OriginalHuella), FirmadoHuella: textoFirma118(f.FirmadoHuella),
			SelloTiempoEstado: textoFirma118(f.SelloTiempoEstado), RegistradaEn: f.RegistradaEn.UTC()})
	}
	return firmas, nil
}
