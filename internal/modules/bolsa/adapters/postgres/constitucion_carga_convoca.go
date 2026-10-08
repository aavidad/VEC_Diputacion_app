package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	importacionpg "vec-diputacion-granada/internal/modules/bolsa/adapters/postgresimportacionconvoca"
	importacion "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type RepositorioCargaConvocaPostgreSQL struct {
	pool      *pgxpool.Pool
	protector importacionpg.ProtectorStagingConvoca
}

var _ ports.RepositorioConstitucionCargaConvoca = (*RepositorioCargaConvocaPostgreSQL)(nil)

func NuevoRepositorioCargaConvocaPostgreSQL(pool *pgxpool.Pool, protector importacionpg.ProtectorStagingConvoca) (*RepositorioCargaConvocaPostgreSQL, error) {
	if pool == nil || protector == nil {
		return nil, ports.ErrConstitucionBolsaNoDisponible
	}
	return &RepositorioCargaConvocaPostgreSQL{pool: pool, protector: protector}, nil
}

// auditoriaRefConsumoCargaConvoca es la forma del asiento de consumo del núcleo AD3.
var auditoriaRefConsumoCargaConvoca = regexp.MustCompile(`^aud_v3_[0-9a-f]{32}$`)

type reciboCargaConvocaJSON struct {
	reciboConstitucionJSON
	DecisionRef        string                        `json:"decision_ref"`
	AuditoriaRef       string                        `json:"auditoria_ref"`
	ConsumidaEn        string                        `json:"consumida_en"`
	ActaReutilizada    bool                          `json:"acta_reutilizada"`
	Vinculos           ports.ReciboVinculosCandidato `json:"vinculos"`
	DecisionAccesoRef  string                        `json:"decision_acceso_ref"`
	AuditoriaAccesoRef string                        `json:"auditoria_acceso_ref"`
}

// ConfirmarCargaConvocaAutorizada es el único efecto B1. La función SQL B79
// confirma original, acta, constitución, vínculos y V3 o revierte todo.
func (r *RepositorioCargaConvocaPostgreSQL) ConfirmarCargaConvocaAutorizada(ctx context.Context, lote importacion.LoteValidado, c ports.Constitucion, vinculos []ports.VinculoCandidato, original ports.OriginalProtegidoCargaConvoca, m puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboCargaConvoca, error) {
	if ctx == nil || r == nil || r.pool == nil || r.protector == nil || m.ValidarEstructura() != nil ||
		lote.Validar() != nil || lote.Acta.ActaRef != c.ActaRef || original.Referencia != lote.Acta.FicheroCustodiadoRef {
		return ports.ReciboCargaConvoca{}, ports.ErrConstitucionBolsaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboCargaConvoca{}, err
	}
	argumentos, err := argumentosConstitucion(c)
	if err != nil {
		return ports.ReciboCargaConvoca{}, err
	}
	actaJSON, filasJSON, err := importacionpg.PrepararCargaAutorizada(ctx, r.protector, lote)
	if err != nil {
		return ports.ReciboCargaConvoca{}, ports.ErrConstitucionBolsaNoDisponible
	}
	defer borrarJSONCargaConvoca(actaJSON, filasJSON)
	originalJSON, err := serializarOriginalCargaConvoca(original, lote)
	if err != nil {
		return ports.ReciboCargaConvoca{}, err
	}
	defer borrarJSONCargaConvoca(originalJSON)
	vinculosSQL := make([]vinculoCandidatoJSON, len(vinculos))
	for i, v := range vinculos {
		vinculosSQL[i] = vinculoCandidatoJSON{CandidatoRef: v.CandidatoRef, ParticipacionRef: v.ParticipacionRef}
	}
	vinculosJSON, err := json.Marshal(vinculosSQL)
	if err != nil {
		return ports.ReciboCargaConvoca{}, ports.ErrConstitucionBolsaInvalida
	}
	argumentos = append([]any{json.RawMessage(actaJSON), json.RawMessage(filasJSON), json.RawMessage(originalJSON)}, argumentos...)
	argumentos = append(argumentos, json.RawMessage(vinculosJSON), m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(),
		m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI())
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ports.ReciboCargaConvoca{}, ports.ErrConstitucionBolsaNoDisponible
	}
	defer tx.Rollback(context.Background())
	var contenido []byte
	err = tx.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.confirmar_carga_convoca_v1(
		$1::jsonb, $2::jsonb, $3::jsonb,
		$4::text, $5::text, $6::text, $7::text, $8::bigint, $9::bytea, $10::timestamptz,
		$11::text, $12::bigint, $13::bytea, $14::timestamptz, $15::timestamptz, $16::jsonb, $17::timestamptz,
		$18::jsonb, $19::bytea, $20::bytea, $21::bytea, $22::bytea, $23::numeric, $24::numeric,
		$25::bytea, $26::bytea, $27::bytea, $28::bytea)`,
		argumentos...,
	).Scan(&contenido)
	if err != nil {
		return ports.ReciboCargaConvoca{}, errorConstitucionCargaConvoca(ctx, err)
	}
	var leido reciboCargaConvocaJSON
	if json.Unmarshal(contenido, &leido) != nil || leido.ActaRef != c.ActaRef || leido.DecisionRef == "" ||
		leido.DecisionAccesoRef == "" || !auditoriaRefConsumoCargaConvoca.MatchString(leido.AuditoriaRef) ||
		!auditoriaRefConsumoCargaConvoca.MatchString(leido.AuditoriaAccesoRef) {
		return ports.ReciboCargaConvoca{}, ports.ErrConstitucionBolsaNoDisponible
	}
	consumida, err := time.Parse("2006-01-02T15:04:05.000000Z", leido.ConsumidaEn)
	if err != nil {
		return ports.ReciboCargaConvoca{}, ports.ErrConstitucionBolsaNoDisponible
	}
	recibo, err := leido.traducir()
	if err != nil {
		return ports.ReciboCargaConvoca{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return ports.ReciboCargaConvoca{}, ports.ErrConstitucionBolsaNoDisponible
	}
	recibo.Vinculos = leido.Vinculos
	return ports.ReciboCargaConvoca{ReciboConstitucion: recibo, DecisionRef: leido.DecisionRef, AuditoriaRef: leido.AuditoriaRef,
		ConsumidaEn: consumida.UTC(), ActaReutilizada: leido.ActaReutilizada,
		DecisionAccesoRef: leido.DecisionAccesoRef, AuditoriaAccesoRef: leido.AuditoriaAccesoRef}, nil
}

func serializarOriginalCargaConvoca(o ports.OriginalProtegidoCargaConvoca, lote importacion.LoteValidado) ([]byte, error) {
	if o.Referencia != lote.Acta.FicheroCustodiadoRef || o.BytesOriginales <= 0 || o.BytesOriginales > 1*1024*1024 ||
		(o.Formato != "xls" && o.Formato != "xlsx") || len(o.Nonce) != 12 || len(o.ContenidoCifrado) < 16 ||
		o.ClaveVersion == 0 || o.ClaveRef == "" || o.EsquemaProteccion == "" {
		return nil, ports.ErrConstitucionBolsaInvalida
	}
	h := sha256.Sum256(o.ContenidoCifrado)
	if hex.EncodeToString(h[:]) != o.HuellaContenidoCifradoSHA256 {
		return nil, ports.ErrConstitucionBolsaInvalida
	}
	return json.Marshal(struct {
		Formato                      string `json:"formato"`
		BytesOriginales              int    `json:"bytes_originales"`
		EsquemaProteccion            string `json:"esquema_proteccion"`
		ClaveRef                     string `json:"clave_ref"`
		ClaveVersion                 uint64 `json:"clave_version"`
		NonceHex                     string `json:"nonce_hex"`
		ContenidoCifradoHex          string `json:"contenido_cifrado_hex"`
		HuellaContenidoCifradoSHA256 string `json:"huella_contenido_cifrado_sha256"`
	}{o.Formato, o.BytesOriginales, o.EsquemaProteccion, o.ClaveRef, o.ClaveVersion, hex.EncodeToString(o.Nonce), hex.EncodeToString(o.ContenidoCifrado), o.HuellaContenidoCifradoSHA256})
}

func borrarJSONCargaConvoca(contenidos ...[]byte) {
	for _, c := range contenidos {
		clear(c)
	}
}

// errorConstitucionCargaConvoca separa la denegación (42501 de AD203/B79) del
// resto de fallos de la constitución. VA172 significa origen técnico ausente
// tras AD208: es indisponibilidad, nunca denegación del perfil de RRHH.
func errorConstitucionCargaConvoca(ctx context.Context, err error) error {
	var errorPG *pgconn.PgError
	if (ctx == nil || ctx.Err() == nil) && errors.As(err, &errorPG) && errorPG.Code == "42501" {
		return dominiovec.ErrAutorizacionDenegada
	}
	return errorConstitucion(ctx, err)
}
