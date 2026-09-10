package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	ctadapter "vec-diputacion-granada/internal/modules/personal/adapters/contrataciontemporal"
)

const (
	funcionRegistrarAltaEjercicio = "vec_personal.registrar_alta_ejercicio_v1"
	maximoBytesMaterialAlta       = 64 << 10
	maximoBytesRespuestaAlta      = 64 << 10
)

var patronHuellaAlta = regexp.MustCompile(`^[a-f0-9]{64}$`)

type iniciadorTransaccionAlta interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

type relojAlta interface{ Ahora() time.Time }

// TransaccionAltaEjercicioPostgreSQL ejecuta exclusivamente la frontera SQL
// planificada de Personal. No implementa la función ni se compone por ahora.
type TransaccionAltaEjercicioPostgreSQL struct {
	pool  iniciadorTransaccionAlta
	reloj relojAlta
}

var _ ctadapter.TransaccionAltaPersonal = (*TransaccionAltaEjercicioPostgreSQL)(nil)

func NuevaTransaccionAltaEjercicioPostgreSQL(pool *pgxpool.Pool, reloj ctports.Reloj) (*TransaccionAltaEjercicioPostgreSQL, error) {
	return nuevaTransaccionAltaEjercicioPostgreSQL(pool, reloj)
}

func nuevaTransaccionAltaEjercicioPostgreSQL(pool iniciadorTransaccionAlta, reloj relojAlta) (*TransaccionAltaEjercicioPostgreSQL, error) {
	if dependenciaNulaAlta(pool) || dependenciaNulaAlta(reloj) {
		return nil, ctadapter.ErrNoDisponible
	}
	return &TransaccionAltaEjercicioPostgreSQL{pool: pool, reloj: reloj}, nil
}

// RegistrarORecuperarAlta no reintenta: un error de commit es incierto y
// devuelve resultado cero. La función SQL debe verificar/consumir V3 y
// escribir relación, ocupación, recibo, auditoría y outbox atómicamente.
func (r *TransaccionAltaEjercicioPostgreSQL) RegistrarORecuperarAlta(ctx context.Context, orden ctadapter.OrdenAlta) (ctadapter.ResultadoTransaccionAlta, error) {
	var cero ctadapter.ResultadoTransaccionAlta
	if r == nil || dependenciaNulaAlta(r.pool) || dependenciaNulaAlta(r.reloj) || ctx == nil {
		return cero, ctadapter.ErrNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if err := orden.ValidarEn(r.reloj.Ahora()); err != nil {
		return cero, ctadapter.ErrDenegado
	}
	material, exportacion, err := materialYExportacionAlta(orden)
	if err != nil {
		return cero, ctadapter.ErrNoDisponible
	}
	defer borrarBytesAlta(material)
	defer borrarPiezasAlta(exportacion.piezas[:])
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return cero, normalizarErrorAlta(ctx, err)
	}
	confirmada := false
	defer func() {
		if !confirmada {
			revertirTransaccionAlta(tx)
		}
	}()
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true), set_config('row_security','on',true), set_config('timezone','UTC',true), set_config('lock_timeout','2s',true), set_config('statement_timeout','15s',true), set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		return cero, normalizarErrorAlta(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if err := orden.ValidarEn(r.reloj.Ahora()); err != nil {
		return cero, ctadapter.ErrDenegado
	}
	var salida []byte
	// Scan puede asignar bytes antes de fallar: limpiarlos también en esa salida.
	defer func() { borrarBytesAlta(salida) }()
	err = tx.QueryRow(ctx, `SELECT vec_personal.registrar_alta_ejercicio_v1($1::jsonb,$2,$3,$4,$5,$6::bigint,$7::bigint,$8,$9,$10,$11)::text`, material, exportacion.piezas[0], exportacion.piezas[1], exportacion.piezas[2], exportacion.piezas[3], exportacion.personaVersion, exportacion.perfilVersion, exportacion.piezas[4], exportacion.piezas[5], exportacion.piezas[6], exportacion.piezas[7]).Scan(&salida)
	if err != nil {
		return cero, normalizarErrorAlta(ctx, err)
	}
	resultado, err := resultadoAltaDesdeJSON(salida, orden)
	if err != nil {
		return cero, ctadapter.ErrReciboNoConfiable
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if err := orden.ValidarEn(r.reloj.Ahora()); err != nil || resultado.ValidarPara(orden, r.reloj.Ahora()) != nil {
		return cero, ctadapter.ErrReciboNoConfiable
	}
	if err := tx.Commit(ctx); err != nil {
		return cero, normalizarErrorAlta(ctx, err)
	}
	confirmada = true
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	return resultado, nil
}

type parametrosV3Alta struct {
	piezas         [8][]byte
	personaVersion int64
	perfilVersion  int64
}

// materialYExportacionAlta devuelve once parámetros: material JSON, las ocho
// piezas V3 y persona/perfil. Todo byte mutable se clona y limpia al salir.
func materialYExportacionAlta(orden ctadapter.OrdenAlta) ([]byte, parametrosV3Alta, error) {
	material, err := json.Marshal(orden.Material())
	if err != nil || len(material) == 0 || len(material) > maximoBytesMaterialAlta {
		return nil, parametrosV3Alta{}, errors.New("material")
	}
	a := orden.Exportacion()
	if a.ValidarEstructura() != nil {
		borrarBytesAlta(material)
		return nil, parametrosV3Alta{}, errors.New("exportacion")
	}
	piezas := [8][]byte{
		append([]byte(nil), a.CapacidadCanonica()...), append([]byte(nil), a.DecisionCanonica()...),
		append([]byte(nil), a.MotivoCanonico()...), append([]byte(nil), a.ContextoActorCanonico()...),
		append([]byte(nil), a.PayloadVECAD3()...), append([]byte(nil), a.SobreCOSESign1()...),
		append([]byte(nil), a.EvidenciaVerificacion()...), append([]byte(nil), a.RaizPublicaSPKI()...),
	}
	return material, parametrosV3Alta{piezas: piezas, personaVersion: int64(a.PersonaVersion()), perfilVersion: int64(a.PerfilVersion())}, nil
}

type respuestaAltaSQL struct {
	Resultado              ctports.ResultadoAltaPersonalRPT `json:"resultado"`
	MaterialSHA256         string                           `json:"material_sha256"`
	RegistradoEn           string                           `json:"registrado_en"`
	DecisionOriginalRef    string                           `json:"decision_original_ref"`
	AuditoriaRef           string                           `json:"auditoria_ref"`
	OutboxRef              string                           `json:"outbox_ref"`
	EjercicioSintetico     bool                             `json:"ejercicio_sintetico"`
	FirmaOficial           bool                             `json:"firma_oficial"`
	EficaciaAdministrativa bool                             `json:"eficacia_administrativa"`
	Replay                 bool                             `json:"replay"`
	DecisionConsumidaRef   string                           `json:"decision_consumida_ref"`
}

func resultadoAltaDesdeJSON(contenido []byte, orden ctadapter.OrdenAlta) (ctadapter.ResultadoTransaccionAlta, error) {
	if len(contenido) == 0 || len(contenido) > maximoBytesRespuestaAlta || validarJSONAltaEstricto(contenido) != nil {
		return ctadapter.ResultadoTransaccionAlta{}, errors.New("respuesta")
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	var respuesta respuestaAltaSQL
	if err := decodificador.Decode(&respuesta); err != nil || exigirFinJSONAlta(decodificador) != nil {
		return ctadapter.ResultadoTransaccionAlta{}, errors.New("respuesta")
	}
	registradoEn, err := time.Parse(time.RFC3339Nano, respuesta.RegistradoEn)
	if err != nil {
		return ctadapter.ResultadoTransaccionAlta{}, err
	}
	huella, err := orden.Material().HuellaSHA256()
	if err != nil || !patronHuellaAlta.MatchString(respuesta.MaterialSHA256) || respuesta.MaterialSHA256 != huella {
		return ctadapter.ResultadoTransaccionAlta{}, errors.New("material")
	}
	return ctadapter.ResultadoTransaccionAlta{Recibo: ctadapter.ReciboAlta{
		Material: orden.Material(), Resultado: respuesta.Resultado, RegistradoEn: registradoEn,
		DecisionOriginalRef: respuesta.DecisionOriginalRef, AuditoriaRef: respuesta.AuditoriaRef,
		OutboxRef: respuesta.OutboxRef, EjercicioSintetico: respuesta.EjercicioSintetico,
		FirmaOficial: respuesta.FirmaOficial, EficaciaAdministrativa: respuesta.EficaciaAdministrativa,
	}, Replay: respuesta.Replay, DecisionConsumidaRef: respuesta.DecisionConsumidaRef}, nil
}

func validarJSONAltaEstricto(contenido []byte) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	if err := validarValorJSONAlta(decodificador); err != nil {
		return err
	}
	if err := exigirFinJSONAlta(decodificador); err != nil {
		return err
	}
	var salida map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &salida); err != nil || !clavesExactasAlta(salida,
		"resultado", "material_sha256", "registrado_en", "decision_original_ref", "auditoria_ref", "outbox_ref",
		"ejercicio_sintetico", "firma_oficial", "eficacia_administrativa", "replay", "decision_consumida_ref") {
		return errors.New("claves salida")
	}
	var resultado map[string]json.RawMessage
	if err := json.Unmarshal(salida["resultado"], &resultado); err != nil || !clavesExactasAlta(resultado,
		"esquema", "contrato_version", "resultado_ref", "recibo_ref", "solicitud_ref", "correlacion_ref",
		"idempotencia_ref", "huella_solicitud_sha256", "estado", "relacion_ref", "ocupacion_ref") {
		return errors.New("claves resultado")
	}
	for _, valor := range salida {
		if bytes.Equal(bytes.TrimSpace(valor), []byte("null")) {
			return errors.New("nulo")
		}
	}
	for _, valor := range resultado {
		if bytes.Equal(bytes.TrimSpace(valor), []byte("null")) {
			return errors.New("nulo")
		}
	}
	return nil
}

func clavesExactasAlta(campos map[string]json.RawMessage, esperadas ...string) bool {
	if len(campos) != len(esperadas) {
		return false
	}
	for _, clave := range esperadas {
		if _, existe := campos[clave]; !existe {
			return false
		}
	}
	return true
}

func validarValorJSONAlta(decodificador *json.Decoder) error {
	token, err := decodificador.Token()
	if err != nil {
		return err
	}
	delimitador, compuesto := token.(json.Delim)
	if !compuesto {
		return nil
	}
	if delimitador == '{' {
		claves := map[string]struct{}{}
		for decodificador.More() {
			clave, err := decodificador.Token()
			texto, ok := clave.(string)
			if err != nil || !ok {
				return errors.New("clave")
			}
			if _, existe := claves[texto]; existe {
				return errors.New("duplicada")
			}
			claves[texto] = struct{}{}
			if err := validarValorJSONAlta(decodificador); err != nil {
				return err
			}
		}
		_, err = decodificador.Token()
		return err
	}
	if delimitador == '[' {
		for decodificador.More() {
			if err := validarValorJSONAlta(decodificador); err != nil {
				return err
			}
		}
		_, err = decodificador.Token()
		return err
	}
	return errors.New("delimitador")
}

func exigirFinJSONAlta(decodificador *json.Decoder) error {
	var extra any
	if err := decodificador.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("adicional")
	}
	return nil
}

func normalizarErrorAlta(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, ctadapter.ErrConflicto) {
		return ctadapter.ErrConflicto
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "P1102" {
		return ctadapter.ErrConflicto
	}
	return ctadapter.ErrNoDisponible
}

func dependenciaNulaAlta(valor any) bool {
	if valor == nil {
		return true
	}
	rv := reflect.ValueOf(valor)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return rv.IsNil()
	}
	return false
}
func borrarBytesAlta(valor []byte) {
	for indice := range valor {
		valor[indice] = 0
	}
}
func borrarPiezasAlta(piezas [][]byte) {
	for _, pieza := range piezas {
		borrarBytesAlta(pieza)
	}
}
func revertirTransaccionAlta(tx pgx.Tx) {
	if tx != nil {
		ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelar()
		_ = tx.Rollback(ctx)
	}
}
