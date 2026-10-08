package postgres

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	consultaPreflightFronteraIdentidadTecnica = `SELECT vec_autorizacion_atestada_v3.acreditar_frontera_identidad_tecnica_v1()`
	consultaRegistroFronteraIdentidadTecnica  = `SELECT auditoria_ref, secuencia::text, material_sha256, correlacion_ref, registrada_en FROM vec_autorizacion_atestada_v3.registrar_frontera_identidad_tecnica_v1($1::jsonb)`
	canalFronteraIdentidadTecnica             = "identidad_http_interno_preacreditacion"
	tipoFronteraIdentidadTecnica              = "pre_identidad_tecnica_v1"
	faseFronteraIdentidadTecnica              = "preacreditacion"
	accionFronteraIdentidadTecnica            = "registrar_pre_identidad_tecnica_v1"
	finalidadFronteraIdentidadTecnica         = "preacreditacion_identidad"
	dominioMaterialFronteraIdentidadTecnica   = "vec.auditoria.pre-identidad-tecnica.v1"
)

type iniciadorFronteraIdentidadTecnica interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

type rutaFronteraIdentidadPreflight struct {
	MetodoEsperado string `json:"metodo_esperado"`
	Ruta           string `json:"ruta"`
}

type codigoFronteraIdentidadPreflight struct {
	MotivoRef string `json:"motivo_ref"`
	Resultado string `json:"resultado"`
}

type configuracionFronteraIdentidadPreflight struct {
	OperadorLogin string                             `json:"operador_login"`
	Proceso       string                             `json:"proceso"`
	Canal         string                             `json:"canal"`
	Superficie    string                             `json:"superficie"`
	Rutas         []rutaFronteraIdentidadPreflight   `json:"rutas"`
	Codigos       []codigoFronteraIdentidadPreflight `json:"codigos"`
}

// RegistradorFronteraIdentidadTecnicaPostgreSQL fija al arrancar el LOGIN,
// proceso, canal, superficie, rutas y motivos acreditados por AD222. Ningún
// campo de identidad humana ni de certificado cruza este puerto.
type RegistradorFronteraIdentidadTecnicaPostgreSQL struct {
	pool   iniciadorFronteraIdentidadTecnica
	config configuracionFronteraIdentidadPreflight
}

var _ ports.RegistradorFronteraIdentidadTecnica = (*RegistradorFronteraIdentidadTecnicaPostgreSQL)(nil)

func NuevoRegistradorFronteraIdentidadTecnicaPostgreSQL(
	ctx context.Context, pool *pgxpool.Pool,
) (*RegistradorFronteraIdentidadTecnicaPostgreSQL, error) {
	return nuevoRegistradorFronteraIdentidadTecnicaPostgreSQL(ctx, pool)
}

func nuevoRegistradorFronteraIdentidadTecnicaPostgreSQL(
	ctx context.Context, pool iniciadorFronteraIdentidadTecnica,
) (*RegistradorFronteraIdentidadTecnicaPostgreSQL, error) {
	if ctx == nil || valorNuloPostgreSQL(pool) {
		return nil, ports.ErrFronteraIdentidadTecnicaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, errorFronteraIdentidadTecnica(ctx)
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || valorNuloPostgreSQL(tx) {
		return nil, errorFronteraIdentidadTecnica(ctx)
	}
	defer revertirTransaccionPostgreSQL(tx)
	if configurarTransaccionAutorizacion(ctx, tx) != nil {
		return nil, errorFronteraIdentidadTecnica(ctx)
	}
	var raw []byte
	if tx.QueryRow(ctx, consultaPreflightFronteraIdentidadTecnica).Scan(&raw) != nil {
		return nil, errorFronteraIdentidadTecnica(ctx)
	}
	config, err := leerPreflightFronteraIdentidadTecnica(raw)
	if err != nil || ctx.Err() != nil {
		return nil, errorFronteraIdentidadTecnica(ctx)
	}
	if tx.Commit(ctx) != nil {
		return nil, errorFronteraIdentidadTecnica(ctx)
	}
	return &RegistradorFronteraIdentidadTecnicaPostgreSQL{pool: pool, config: config}, nil
}

func leerPreflightFronteraIdentidadTecnica(raw []byte) (configuracionFronteraIdentidadPreflight, error) {
	var vacia configuracionFronteraIdentidadPreflight
	var claves map[string]json.RawMessage
	if json.Unmarshal(raw, &claves) != nil || len(claves) != 6 {
		return vacia, ports.ErrFronteraIdentidadTecnicaNoDisponible
	}
	for _, clave := range []string{"operador_login", "proceso", "canal", "superficie", "rutas", "codigos"} {
		if _, ok := claves[clave]; !ok {
			return vacia, ports.ErrFronteraIdentidadTecnicaNoDisponible
		}
	}
	var c configuracionFronteraIdentidadPreflight
	decodificador := json.NewDecoder(bytes.NewReader(raw))
	decodificador.DisallowUnknownFields()
	if decodificador.Decode(&c) != nil || !preflightFronteraIdentidadValido(c) {
		return vacia, ports.ErrFronteraIdentidadTecnicaNoDisponible
	}
	return c, nil
}

func preflightFronteraIdentidadValido(c configuracionFronteraIdentidadPreflight) bool {
	if !identificadorPostgreSQLSeguro(c.OperadorLogin, 63) ||
		!nombreTecnicoFronteraIdentidad(c.Proceso, 80) ||
		c.Canal != canalFronteraIdentidadTecnica ||
		c.Superficie != domain.SuperficieFronteraIdentidadTecnica ||
		len(c.Rutas) != 2 || len(c.Codigos) != 8 ||
		c.Rutas[0] != (rutaFronteraIdentidadPreflight{MetodoEsperado: domain.MetodoInicioSesionGET, Ruta: domain.RutaSesionActual}) ||
		c.Rutas[1] != (rutaFronteraIdentidadPreflight{MetodoEsperado: domain.MetodoInicioSesionPOST, Ruta: domain.RutaInicioSesion}) {
		return false
	}
	esperados := []codigoFronteraIdentidadPreflight{
		{"certificado_requerido", "denegado"}, {"autenticacion_requerida", "denegado"},
		{"acceso_denegado", "denegado"}, {"metodo_no_permitido", "denegado"},
		{"recurso_no_encontrado", "denegado"}, {"solicitud_invalida", "denegado"},
		{"servicio_no_disponible", "error"}, {"respuesta_incompatible", "error"},
	}
	for i, esperado := range esperados {
		if c.Codigos[i] != esperado {
			return false
		}
	}
	return true
}

func nombreTecnicoFronteraIdentidad(s string, maximo int) bool {
	if len(s) < 2 || len(s) > maximo || s[0] < 'a' || s[0] > 'z' {
		return false
	}
	for _, b := range s[1:] {
		if !(b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || b == '_' || b == '-' || b == '.') {
			return false
		}
	}
	return true
}

type eventoFronteraIdentidadTecnica struct {
	TipoRegistro   string `json:"tipo_registro"`
	EventoRef      string `json:"evento_ref"`
	OperadorLogin  string `json:"operador_login"`
	Fase           string `json:"fase"`
	MetodoEsperado string `json:"metodo_esperado"`
	Ruta           string `json:"ruta"`
	Accion         string `json:"accion"`
	RecursoRef     string `json:"recurso_ref"`
	Resultado      string `json:"resultado"`
	MotivoRef      string `json:"motivo_ref"`
	Proceso        string `json:"proceso"`
	Canal          string `json:"canal"`
	Superficie     string `json:"superficie"`
	FinalidadRef   string `json:"finalidad_ref"`
	CorrelacionRef string `json:"correlacion_ref"`
}

func (r *RegistradorFronteraIdentidadTecnicaPostgreSQL) RegistrarRechazoInicioSesion(
	ctx context.Context, orden ports.OrdenFronteraIdentidadTecnica,
) (ports.AcuseFronteraIdentidadTecnica, error) {
	var vacio ports.AcuseFronteraIdentidadTecnica
	if r == nil || ctx == nil || valorNuloPostgreSQL(r.pool) || !preflightFronteraIdentidadValido(r.config) {
		return vacio, ports.ErrFronteraIdentidadTecnicaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, errorFronteraIdentidadTecnica(ctx)
	}
	datos, err := orden.Datos()
	if err != nil {
		return vacio, ports.ErrOrdenFronteraIdentidadTecnicaInvalida
	}
	recurso, err := domain.RecursoFronteraIdentidadTecnica(datos.CorrelacionRef)
	if err != nil {
		return vacio, ports.ErrOrdenFronteraIdentidadTecnicaInvalida
	}
	var aleatorio [16]byte
	if _, err := rand.Read(aleatorio[:]); err != nil {
		return vacio, ports.ErrFronteraIdentidadTecnicaNoDisponible
	}
	e := eventoFronteraIdentidadTecnica{
		TipoRegistro: tipoFronteraIdentidadTecnica, EventoRef: "evento_" + hex.EncodeToString(aleatorio[:]),
		OperadorLogin: r.config.OperadorLogin, Fase: faseFronteraIdentidadTecnica,
		MetodoEsperado: datos.MetodoEsperado, Ruta: datos.Ruta,
		Accion: accionFronteraIdentidadTecnica, RecursoRef: recurso,
		Resultado: datos.Resultado, MotivoRef: string(datos.Motivo),
		Proceso: r.config.Proceso, Canal: r.config.Canal, Superficie: r.config.Superficie,
		FinalidadRef: finalidadFronteraIdentidadTecnica, CorrelacionRef: datos.CorrelacionRef,
	}
	material := huellaMaterialFronteraIdentidadTecnica(e)
	payload, err := json.Marshal(e)
	if err != nil {
		return vacio, ports.ErrFronteraIdentidadTecnicaNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || valorNuloPostgreSQL(tx) {
		return vacio, errorFronteraIdentidadTecnica(ctx)
	}
	defer revertirTransaccionPostgreSQL(tx)
	if configurarTransaccionAutorizacion(ctx, tx) != nil {
		return vacio, errorFronteraIdentidadTecnica(ctx)
	}
	var secuenciaSQL string
	var acuse ports.AcuseFronteraIdentidadTecnica
	if tx.QueryRow(ctx, consultaRegistroFronteraIdentidadTecnica, payload).Scan(
		&acuse.AuditoriaRef, &secuenciaSQL, &acuse.MaterialSHA256,
		&acuse.CorrelacionRef, &acuse.RegistradaEn,
	) != nil {
		return vacio, errorFronteraIdentidadTecnica(ctx)
	}
	acuse.Secuencia, err = strconv.ParseUint(secuenciaSQL, 10, 64)
	if err != nil {
		return vacio, errors.Join(ports.ErrFronteraIdentidadTecnicaNoDisponible,
			ports.ErrAcuseFronteraIdentidadTecnicaInvalido)
	}
	acuse.RegistradaEn = acuse.RegistradaEn.UTC()
	if acuse.ValidarPara(orden, e.EventoRef, material, time.Now().UTC()) != nil {
		return vacio, errors.Join(ports.ErrFronteraIdentidadTecnicaNoDisponible,
			ports.ErrAcuseFronteraIdentidadTecnicaInvalido)
	}
	if ctx.Err() != nil {
		return vacio, errorFronteraIdentidadTecnica(ctx)
	}
	if tx.Commit(ctx) != nil {
		return vacio, errors.Join(ports.ErrFronteraIdentidadTecnicaNoDisponible,
			ports.ErrFronteraIdentidadTecnicaCommitIncierto, ctx.Err())
	}
	return acuse, nil
}

func huellaMaterialFronteraIdentidadTecnica(e eventoFronteraIdentidadTecnica) string {
	material := make([]byte, 0, 512)
	for _, valor := range []string{
		dominioMaterialFronteraIdentidadTecnica, e.TipoRegistro, e.EventoRef,
		e.OperadorLogin, e.Fase, e.MetodoEsperado, e.Ruta, e.Accion,
		e.RecursoRef, e.Resultado, e.MotivoRef, e.Proceso, e.Canal,
		e.Superficie, e.FinalidadRef, e.CorrelacionRef,
	} {
		material = strconv.AppendInt(material, int64(len(valor)), 10)
		material = append(material, ':')
		material = append(material, valor...)
		material = append(material, '\n')
	}
	huella := sha256.Sum256(material)
	clear(material)
	return hex.EncodeToString(huella[:])
}

func errorFronteraIdentidadTecnica(ctx context.Context) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return errors.Join(ports.ErrFronteraIdentidadTecnicaNoDisponible, err)
		}
	}
	return ports.ErrFronteraIdentidadTecnicaNoDisponible
}
