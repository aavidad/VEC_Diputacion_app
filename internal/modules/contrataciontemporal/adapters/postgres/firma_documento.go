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
	ConMotivo         bool      `json:"ConMotivoDevolucion"`
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

// Restricciones únicas de CT118 por las que se distingue la carrera perdida
// entre dos registros simultáneos (SERIALIZABLE no la impide con el cerrojo).
const (
	restriccionClaveFirma118     = "firma_documento_v1_clave_unica"
	restriccionSecuenciaFirma118 = "firma_documento_v1_secuencia_unica"
	intentosRegistroFirma118     = 3
)

// reintentableFirma118 indica que la transacción perdió una carrera y debe
// repetirse entera: un fallo de serialización o un interbloqueo (40001,
// 40P01) o la misma clave de idempotencia ya confirmada por otra
// transacción (23505 de la clave), que el reintento recupera como recibo
// existente o rechaza como clave reutilizada con otro material.
func reintentableFirma118(err error) bool {
	var p *pgconn.PgError
	if !errors.As(err, &p) {
		return false
	}
	return p.Code == "40001" || p.Code == "40P01" ||
		(p.Code == "23505" && p.ConstraintName == restriccionClaveFirma118)
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
		case "P1182", "P1183":
			return ports.ErrFirmaDocumentoEnConflicto
		case "23505":
			// Otra transacción ocupó la misma secuencia del documento: es el
			// mismo conflicto de historia que P1183.
			if p.ConstraintName == restriccionSecuenciaFirma118 {
				return ports.ErrFirmaDocumentoEnConflicto
			}
		case "P1184":
			return ports.ErrCadenaFirmaDocumentoRota
		case "42501", "P1102":
			return ports.ErrFirmaDocumentoDenegada
		case "22023":
			return ports.ErrSolicitudFirmaDocumentoInvalida
		}
	}
	// 40001/40P01 agotados, 23505 de la clave sin recuperar y cualquier otro
	// fallo: indisponibilidad transitoria, nunca éxito ni conflicto.
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

// RegistrarFirma consume la capacidad y escribe en una sola transacción. Si
// la transacción pierde una carrera con otro registro simultáneo (fallo de
// serialización, interbloqueo o la misma clave ya confirmada), se repite
// entera con el mismo material: el consumo de la transacción perdida se
// deshizo con ella, y el reintento recupera el recibo existente o informa
// del conflicto.
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
	var recibo ports.ReciboFirmaDocumento
	h, _ := m.HuellaSHA256()
	// El recibo se valida antes de confirmar: uno incoherente no se guarda.
	validar := func(w reciboFirmaSQL118) error {
		recibo = ports.ReciboFirmaDocumento{FirmaRef: w.FirmaRef, ReciboRef: w.ReciboRef, Secuencia: w.Secuencia,
			Resultado: domain.ResultadoFirmaDocumento(w.Resultado), ExpedienteVersion: w.ExpedienteVersion, ActorRef: w.ActorRef,
			PerfilRef: w.PerfilRef, RegistradaEn: w.RegistradaEn.UTC(), SolicitudHuella: w.SolicitudHuella, YaRegistrada: w.YaRegistrada}
		if !domain.ReferenciaOpacaValida(recibo.FirmaRef) || !domain.ReferenciaOpacaValida(recibo.ReciboRef) || recibo.SolicitudHuella != h ||
			recibo.Resultado != m.Resultado || recibo.ActorRef == "" || recibo.PerfilRef == "" || recibo.RegistradaEn.IsZero() ||
			(!recibo.YaRegistrada && (recibo.Secuencia != m.Secuencia || recibo.ExpedienteVersion != m.VersionExpediente)) {
			return ports.ErrResultadoFirmaDocumentoInvalido
		}
		return nil
	}
	if err := r.registrarConReintentos(ctx, append([]any{string(canonico)}, parametros...), validar); err != nil {
		return cero, err
	}
	return recibo, nil
}

// registrarConReintentos repite el intento completo mientras pierda una
// carrera reintentable, hasta intentosRegistroFirma118 veces, y traduce el
// error final al vocabulario del puerto.
func (r *RegistroFirmasDocumentoPostgreSQL) registrarConReintentos(ctx context.Context, args []any, validar func(reciboFirmaSQL118) error) error {
	var err error
	for intento := 1; ; intento++ {
		err = r.registrarUnaVez(ctx, args, validar)
		if err == nil || !reintentableFirma118(err) || intento == intentosRegistroFirma118 || ctx.Err() != nil {
			break
		}
	}
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ports.ErrResultadoFirmaDocumentoInvalido):
		return err
	default:
		return errorFirma118(ctx, err)
	}
}

// registrarUnaVez ejecuta un intento completo: BEGIN SERIALIZABLE, CT118,
// validación del recibo y COMMIT. Devuelve el error de PostgreSQL sin
// traducir para que el llamador decida si repetir.
func (r *RegistroFirmasDocumentoPostgreSQL) registrarUnaVez(ctx context.Context, args []any, validar func(reciboFirmaSQL118) error) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return err
	}
	if nuloRegistroTX(tx) {
		return ports.ErrRegistroFirmaDocumentoNoDisponible
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
		return err
	}
	var contenido []byte
	if err = tx.QueryRow(ctx, registrarFirmaSQL118, args...).Scan(&contenido); err != nil {
		return err
	}
	var w reciboFirmaSQL118
	if decodificarFirma118(contenido, &w) != nil {
		return ports.ErrResultadoFirmaDocumentoInvalido
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
			ConMotivoDevolucion: f.ConMotivo, OriginalHuella: textoFirma118(f.OriginalHuella), FirmadoHuella: textoFirma118(f.FirmadoHuella),
			SelloTiempoEstado: textoFirma118(f.SelloTiempoEstado), RegistradaEn: f.RegistradaEn.UTC()})
	}
	return firmas, nil
}
