package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// RepositorioAvisosExternosPostgreSQL usa un pool exclusivo del consumidor.
// Su función SQL devuelve únicamente evento neutral y huella, sin material Bolsa.
type RepositorioAvisosExternosPostgreSQL struct{ pool *pgxpool.Pool }

func NuevoRepositorioAvisosExternosPostgreSQL(pool *pgxpool.Pool) (*RepositorioAvisosExternosPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrEmisionLlamamientoNoDisponible
	}
	cfg := pool.Config().ConnConfig
	if cfg.User != "vec_externo_avisos_bolsa" || cfg.TLSConfig == nil || cfg.TLSConfig.InsecureSkipVerify {
		return nil, ports.ErrEmisionLlamamientoNoDisponible
	}
	for _, f := range cfg.Fallbacks {
		if f.TLSConfig == nil || f.TLSConfig.InsecureSkipVerify {
			return nil, ports.ErrEmisionLlamamientoNoDisponible
		}
	}
	return &RepositorioAvisosExternosPostgreSQL{pool: pool}, nil
}

func (r *RepositorioAvisosExternosPostgreSQL) Extraer(ctx context.Context, limite int) (out []ports.AvisoExternoPendiente, errRet error) {
	if r == nil || r.pool == nil || ctx == nil {
		return nil, ports.ErrEmisionLlamamientoNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, errorEmision(err)
	}
	defer cerrarTransaccionAvisos(ctx, tx, &errRet)
	rows, err := tx.Query(ctx, `SELECT evento,huella,auditoria_ref,error_codigo FROM vec_bolsa_llamamientos.tirar_avisos_externos_v1($1)`, limite)
	if err != nil {
		return nil, errorEmision(err)
	}
	defer rows.Close()
	capacidad := limite
	if capacidad < 1 || capacidad > 100 {
		capacidad = 1
	}
	out = make([]ports.AvisoExternoPendiente, 0, capacidad)
	var errNominal error
	filas := 0
	for rows.Next() {
		filas++
		if filas > capacidad {
			return nil, ports.ErrEmisionLlamamientoNoDisponible
		}
		var raw []byte
		var hash *string
		var auditRef string
		var codigo *string
		if rows.Scan(&raw, &hash, &auditRef, &codigo) != nil {
			return nil, ports.ErrEmisionLlamamientoNoDisponible
		}
		if !patronAuditoriaAvisoExterno.MatchString(auditRef) {
			return nil, ports.ErrEmisionLlamamientoNoDisponible
		}
		if codigo != nil {
			if filas != 1 || len(raw) != 0 || hash != nil {
				return nil, ports.ErrEmisionLlamamientoNoDisponible
			}
			errNominal = errorCodigoAvisoExterno(*codigo)
			continue
		}
		if len(raw) == 0 && hash == nil {
			if filas != 1 {
				return nil, ports.ErrEmisionLlamamientoNoDisponible
			}
			continue
		}
		if hash == nil {
			return nil, ports.ErrEmisionLlamamientoNoDisponible
		}
		evento, err := decodificarEventoAvisoExterno(raw, *hash)
		if err != nil {
			return nil, err
		}
		out = append(out, ports.AvisoExternoPendiente{Evento: evento, Huella: *hash})
	}
	if err = rows.Err(); err != nil {
		return nil, errorEmision(err)
	}
	rows.Close()
	if filas == 0 {
		return nil, ports.ErrEmisionLlamamientoNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, errorEmision(err)
	}
	if errNominal != nil {
		return nil, errNominal
	}
	return out, nil
}

func (r *RepositorioAvisosExternosPostgreSQL) ConfirmarAceptacion(ctx context.Context, productor, eventoref, huella, recibo string) error {
	return r.operacionAuditada(ctx, `SELECT aceptada,auditoria_ref,error_codigo FROM vec_bolsa_llamamientos.confirmar_aceptacion_aviso_externo_v1($1,$2,$3,$4)`, productor, eventoref, huella, recibo)
}

func (r *RepositorioAvisosExternosPostgreSQL) RegistrarResultadoDespacho(ctx context.Context, productor, eventoref, huella, recibo, estado string) error {
	return r.operacionAuditada(ctx, `SELECT registrada,auditoria_ref,error_codigo FROM vec_bolsa_llamamientos.registrar_resultado_aviso_externo_v1($1,$2,$3,$4,$5)`, productor, eventoref, huella, recibo, estado)
}

func (r *RepositorioAvisosExternosPostgreSQL) operacionAuditada(ctx context.Context, consulta string, args ...any) (errRet error) {
	if r == nil || r.pool == nil || ctx == nil {
		return ports.ErrEmisionLlamamientoNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return errorEmision(err)
	}
	defer cerrarTransaccionAvisos(ctx, tx, &errRet)
	var aceptada bool
	var ref string
	var codigo *string
	if err = tx.QueryRow(ctx, consulta, args...).Scan(&aceptada, &ref, &codigo); err != nil {
		return errorEmision(err)
	}
	if !patronAuditoriaAvisoExterno.MatchString(ref) || (aceptada && codigo != nil) || (!aceptada && codigo == nil) {
		return ports.ErrEmisionLlamamientoNoDisponible
	}
	// El rechazo nominal conserva su auditoría; no hubo efecto funcional.
	if err = tx.Commit(ctx); err != nil {
		return errorEmision(err)
	}
	if codigo != nil {
		return errorCodigoAvisoExterno(*codigo)
	}
	return nil
}

func cerrarTransaccionAvisos(ctx context.Context, tx pgx.Tx, errRet *error) {
	// ErrTxClosed es la consecuencia esperada de Commit; los demás fallos de
	// limpieza se traducen a indisponibilidad sin exponer el error del proveedor.
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		*errRet = ports.ErrEmisionLlamamientoNoDisponible
	}
}

func errorCodigoAvisoExterno(codigo string) error {
	switch codigo {
	case "22023":
		return ports.ErrEmisionLlamamientoInvalida
	case "VBE01":
		return ports.ErrEmisionLlamamientoConflicto
	default:
		return ports.ErrEmisionLlamamientoNoDisponible
	}
}

var (
	patronAuditoriaAvisoExterno    = regexp.MustCompile(`^auditoria_tecnica_interna:[0-9a-f]{32}$`)
	patronReciboOutboxAvisoExterno = regexp.MustCompile(`^recibo_outbox:[0-9a-f]{64}$`)
	patronRefAvisoExterno          = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$`)
	patronEventoAvisoExterno       = regexp.MustCompile(`^evento_aviso:[0-9a-f]{64}$`)
	patronHuellaAvisoExterno       = regexp.MustCompile(`^[0-9a-f]{64}$`)
	patronCandidatoAvisoExterno    = regexp.MustCompile(`^can_[A-Za-z0-9_-]{22,128}$`)
	patronComunicacionAvisoExterno = regexp.MustCompile(`^llamamiento:[0-9a-f]{64}$`)
)

// El JSON de PostgreSQL puede variar su espacio y orden. Se valida el DTO y
// se reconstruye el canon compartido, nunca se firma el jsonb::text.
func decodificarEventoAvisoExterno(raw []byte, hash string) (ports.EventoAvisoExterno, error) {
	var e ports.EventoAvisoExterno
	if len(raw) > 4096 || !patronHuellaAvisoExterno.MatchString(hash) {
		return e, ports.ErrEmisionLlamamientoNoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&e) != nil {
		return e, ports.ErrEmisionLlamamientoNoDisponible
	}
	var resto any
	if d.Decode(&resto) != io.EOF {
		return e, ports.ErrEmisionLlamamientoNoDisponible
	}
	// Exige las diez propiedades también si recurso_publico_ref es vacío.
	var propiedades map[string]json.RawMessage
	if json.Unmarshal(raw, &propiedades) != nil || len(propiedades) != 10 {
		return e, ports.ErrEmisionLlamamientoNoDisponible
	}
	for _, v := range propiedades {
		var s string
		if string(v) == "null" || json.Unmarshal(v, &s) != nil {
			return e, ports.ErrEmisionLlamamientoNoDisponible
		}
	}
	if !patronEventoAvisoExterno.MatchString(e.EventoRef) || !patronRefAvisoExterno.MatchString(e.ProductorRef) || e.TipoVersionado != "vec.bolsa.aviso-llamamiento.v1" || !patronRefAvisoExterno.MatchString(e.CorrelacionRef) || !patronCandidatoAvisoExterno.MatchString(e.DestinatarioExternoRef) || !patronComunicacionAvisoExterno.MatchString(e.ComunicacionRef) || !patronRefAvisoExterno.MatchString(e.PlantillaRef) || !patronRefAvisoExterno.MatchString(e.PlantillaVersion) || (e.RecursoPublicoRef != "" && !patronRefAvisoExterno.MatchString(e.RecursoPublicoRef)) {
		return e, ports.ErrEmisionLlamamientoNoDisponible
	}
	const formato = "2006-01-02T15:04:05.000000Z"
	fecha, err := time.Parse(formato, e.OcurridoEn)
	if err != nil || fecha.UTC().Format(formato) != e.OcurridoEn {
		return e, ports.ErrEmisionLlamamientoNoDisponible
	}
	canon, err := json.Marshal(e)
	h := sha256.Sum256(canon)
	if err != nil || hex.EncodeToString(h[:]) != hash {
		return e, ports.ErrEmisionLlamamientoConflicto
	}
	return e, nil
}

func (r *RepositorioEmisionLlamamientoPostgreSQL) ActivarAvisosExternos(ctx context.Context) error {
	if r == nil || r.pool == nil || ctx == nil || r.fuentes {
		return ports.ErrEmisionLlamamientoNoDisponible
	}
	var ok bool
	if err := r.pool.QueryRow(ctx, `SELECT pg_catalog.to_regclass('vec_bolsa_llamamientos.aviso_externo_outbox') IS NOT NULL
 AND pg_catalog.has_function_privilege(session_user,'vec_bolsa_llamamientos.reservar_llamamiento_avisos_externos_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb)','EXECUTE')
 AND pg_catalog.has_function_privilege(session_user,'vec_bolsa_llamamientos.destinatario_externo_participacion_v1(text,text)','EXECUTE')
 AND pg_catalog.has_function_privilege(session_user,'vec_bolsa_llamamientos.avisos_externos_llamamiento_v1(text,text)','EXECUTE')`).Scan(&ok); err != nil || !ok {
		return ports.ErrEmisionLlamamientoNoDisponible
	}
	r.avisosExternos = true
	return nil
}

func (r *RepositorioEmisionLlamamientoPostgreSQL) DestinatarioExterno(ctx context.Context, bolsa, participacion string) (string, error) {
	if r == nil || r.pool == nil || ctx == nil || !r.avisosExternos || bolsa == "" || participacion == "" || len(participacion) > 512 {
		return "", ports.ErrEmisionLlamamientoNoDisponible
	}
	var candidato *string
	if err := r.pool.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.destinatario_externo_participacion_v1($1,$2)`, bolsa, participacion).Scan(&candidato); err != nil {
		return "", errorEmision(err)
	}
	if candidato == nil {
		return "", nil
	}
	if !patronCandidatoAvisoExterno.MatchString(*candidato) {
		return "", ports.ErrEmisionLlamamientoNoDisponible
	}
	return *candidato, nil
}

func (r *RepositorioEmisionLlamamientoPostgreSQL) conAvisosExternos(ctx context.Context, e ports.EmisionLlamamiento, bolsa, clave string) (ports.EmisionLlamamiento, error) {
	if !r.avisosExternos {
		return e, nil
	}
	var raw []byte
	if err := r.pool.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.avisos_externos_llamamiento_v1($1,$2)`, bolsa, clave).Scan(&raw); err != nil {
		return ports.EmisionLlamamiento{}, errorEmision(err)
	}
	var pendientes []ports.AvisoExternoPendiente
	if json.Unmarshal(raw, &pendientes) != nil || len(pendientes) > 100 {
		return ports.EmisionLlamamiento{}, ports.ErrEmisionLlamamientoNoDisponible
	}
	for _, p := range pendientes {
		rawEvento, marshalErr := json.Marshal(p.Evento)
		if marshalErr != nil {
			return ports.EmisionLlamamiento{}, ports.ErrEmisionLlamamientoNoDisponible
		}
		if _, err := decodificarEventoAvisoExterno(rawEvento, p.Huella); err != nil {
			return ports.EmisionLlamamiento{}, err
		}
	}
	e.AvisosExternos = pendientes
	if len(pendientes) > 0 {
		if len(pendientes) != len(e.Participaciones) {
			return ports.EmisionLlamamiento{}, ports.ErrEmisionLlamamientoNoDisponible
		}
		contactos, err := proyectarContactosAvisosExternos(e, pendientes, bolsa, clave)
		if err != nil {
			return ports.EmisionLlamamiento{}, err
		}
		e.Contactos = contactos
		e.Estado = "emitido_pendiente_respuesta"
	}
	return e, nil
}

// La emisión reúne todos los participantes aunque solo parte del lote tenga
// resultado terminal. Un terminal referencia su contacto B7; el pendiente
// conserva el recibo de cola y no acredita un intento SMTP confirmado.
func proyectarContactosAvisosExternos(e ports.EmisionLlamamiento, pendientes []ports.AvisoExternoPendiente, bolsa, clave string) ([]ports.ResultadoContactoEmision, error) {
	if bolsa == "" || clave == "" || len(pendientes) != len(e.Participaciones) {
		return nil, ports.ErrEmisionLlamamientoNoDisponible
	}
	contactos := make([]ports.ResultadoContactoEmision, 0, len(e.Participaciones))
	eventos := make(map[string]ports.AvisoExternoPendiente, len(pendientes))
	for _, p := range pendientes {
		if !patronReciboOutboxAvisoExterno.MatchString(p.ReciboOutboxRef) {
			return nil, ports.ErrEmisionLlamamientoNoDisponible
		}
		hRecibo := sha256.Sum256([]byte(p.Evento.ProductorRef + "\x1f" + p.Evento.EventoRef + "\x1f" + p.Huella))
		if p.ReciboOutboxRef != "recibo_outbox:"+hex.EncodeToString(hRecibo[:]) {
			return nil, ports.ErrEmisionLlamamientoNoDisponible
		}
		eventos[p.Evento.EventoRef] = p
	}
	for _, participacion := range e.Participaciones {
		eventHash := sha256.Sum256([]byte(e.LlamamientoRef + "\x1f" + participacion))
		pendiente, exists := eventos["evento_aviso:"+hex.EncodeToString(eventHash[:])]
		if !exists {
			return nil, ports.ErrEmisionLlamamientoNoDisponible
		}
		resultado := "aviso_pendiente"
		switch pendiente.EstadoDespacho {
		case "", "reservado_incierto":
		case "aceptado":
			resultado = "enviado"
		case "no_aceptado", "sin_destino":
			resultado = "no_enviado"
		default:
			return nil, ports.ErrEmisionLlamamientoNoDisponible
		}
		recibo := pendiente.ReciboOutboxRef
		if resultado != "aviso_pendiente" {
			hContacto := sha256.Sum256([]byte(bolsa + "\x1f" + clave + "\x1f" + participacion))
			recibo = "recibo:contacto:" + hex.EncodeToString(hContacto[:])
		}
		contactos = append(contactos, ports.ResultadoContactoEmision{ParticipacionRef: participacion, Resultado: resultado, ReciboRef: recibo})
	}
	return contactos, nil
}

var _ ports.RepositorioAvisosExternos = (*RepositorioAvisosExternosPostgreSQL)(nil)
var _ ports.FuenteDestinatarioExternoParticipacion = (*RepositorioEmisionLlamamientoPostgreSQL)(nil)
