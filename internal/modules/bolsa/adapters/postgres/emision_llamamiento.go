package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type RepositorioEmisionLlamamientoPostgreSQL struct {
	pool *pgxpool.Pool
	// fuentes indica que B59 está instalado y activo: la respuesta y la
	// recuperación incluyen la fuente del correo de cada aviso.
	fuentes        bool
	avisosExternos bool
}

func NuevoRepositorioEmisionLlamamientoPostgreSQL(pool *pgxpool.Pool) (*RepositorioEmisionLlamamientoPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrEmisionLlamamientoNoDisponible
	}
	return &RepositorioEmisionLlamamientoPostgreSQL{pool: pool}, nil
}

func (r *RepositorioEmisionLlamamientoPostgreSQL) Reservar(ctx context.Context, c ports.ComandoEmitirLlamamiento) (ports.EmisionLlamamiento, error) {
	if r == nil || r.pool == nil || ctx == nil || c.Material.ValidarEstructura() != nil {
		return ports.EmisionLlamamiento{}, ports.ErrEmisionLlamamientoNoDisponible
	}
	participaciones, _ := json.Marshal(c.Participaciones)
	configuracion, _ := json.Marshal(c.Configuracion)
	m := c.Material
	huellaFinalizacion := sha256.Sum256(c.TokenFinalizacion)
	if len(c.TokenFinalizacion) != 32 {
		return ports.EmisionLlamamiento{}, ports.ErrEmisionLlamamientoNoDisponible
	}
	var salidaJSON []byte
	var reutilizada bool
	consulta := `SELECT emision,reutilizada FROM vec_bolsa_llamamientos.reservar_llamamiento_v1($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb,$8,$9,$10,$11,$12,$13,$14::numeric,$15::numeric,$16,$17,$18,$19)`
	args := []any{c.LlamamientoRef, c.ReciboRef, c.BolsaRef, c.ActorRef, c.ClaveIdempotencia, participaciones, configuracion, c.EmitidoEn, huellaFinalizacion[:], m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()}
	if c.AvisosExternos != nil {
		if !r.avisosExternos {
			return ports.EmisionLlamamiento{}, ports.ErrEmisionLlamamientoNoDisponible
		}
		eventos, err := json.Marshal(c.AvisosExternos)
		if err != nil {
			return ports.EmisionLlamamiento{}, ports.ErrEmisionLlamamientoInvalida
		}
		consulta = `SELECT emision,reutilizada FROM vec_bolsa_llamamientos.reservar_llamamiento_avisos_externos_v1($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb,$8,$9,$10,$11,$12,$13,$14::numeric,$15::numeric,$16,$17,$18,$19,$20::jsonb)`
		args[8] = c.TokenFinalizacion
		args = append(args, eventos)
	} else if r.avisosExternos {
		return ports.EmisionLlamamiento{}, ports.ErrEmisionLlamamientoNoDisponible
	}
	err := r.pool.QueryRow(ctx, consulta, args...).Scan(&salidaJSON, &reutilizada)
	if err != nil {
		return ports.EmisionLlamamiento{}, errorEmision(err)
	}
	var out ports.EmisionLlamamiento
	if json.Unmarshal(salidaJSON, &out) != nil {
		return ports.EmisionLlamamiento{}, ports.ErrEmisionLlamamientoNoDisponible
	}
	out.Reutilizada = reutilizada
	if reutilizada {
		out, err = r.conFuentesCorreo(ctx, out, c.BolsaRef, c.ClaveIdempotencia)
		if err != nil {
			return ports.EmisionLlamamiento{}, err
		}
		return r.conAvisosExternos(ctx, out, c.BolsaRef, c.ClaveIdempotencia)
	}
	return r.conAvisosExternos(ctx, out, c.BolsaRef, c.ClaveIdempotencia)
}

func (r *RepositorioEmisionLlamamientoPostgreSQL) RegistrarContactos(ctx context.Context, bolsa, clave, actor string, token []byte, contactos []ports.ResultadoContactoEmision) (ports.EmisionLlamamiento, error) {
	if r == nil || r.pool == nil || ctx == nil || r.avisosExternos || bolsa == "" || clave == "" || actor == "" || len(token) != 32 || len(contactos) == 0 {
		return ports.EmisionLlamamiento{}, ports.ErrEmisionLlamamientoNoDisponible
	}
	conFuente := 0
	for _, c := range contactos {
		if c.FuenteCorreo != nil {
			if !r.fuentes || !c.FuenteCorreo.Valida() {
				return ports.EmisionLlamamiento{}, ports.ErrEmisionLlamamientoInvalida
			}
			conFuente++
		}
	}
	if conFuente != 0 && conFuente != len(contactos) {
		return ports.EmisionLlamamiento{}, ports.ErrEmisionLlamamientoInvalida
	}
	rawContactos, _ := json.Marshal(contactos)
	sql := `SELECT vec_bolsa_llamamientos.registrar_contactos_llamamiento_v1($1,$2,$3,$4,$5::jsonb)`
	if conFuente != 0 {
		sql = `SELECT vec_bolsa_llamamientos.registrar_contactos_llamamiento_v2($1,$2,$3,$4,$5::jsonb)`
	}
	var salidaJSON []byte
	if err := r.pool.QueryRow(ctx, sql, bolsa, clave, actor, token, rawContactos).Scan(&salidaJSON); err != nil {
		return ports.EmisionLlamamiento{}, errorEmision(err)
	}
	var out emisionConFuentesSQL
	if json.Unmarshal(salidaJSON, &out) != nil {
		return ports.EmisionLlamamiento{}, ports.ErrEmisionLlamamientoNoDisponible
	}
	return r.conAvisosExternos(ctx, aplicarFuentesCorreo(out.EmisionLlamamiento, out.FuentesCorreo), bolsa, clave)
}

func (r *RepositorioEmisionLlamamientoPostgreSQL) Recuperar(ctx context.Context, bolsa, clave string) (ports.EmisionLlamamiento, error) {
	if r == nil || r.pool == nil || ctx == nil {
		return ports.EmisionLlamamiento{}, ports.ErrEmisionLlamamientoNoDisponible
	}
	var raw []byte
	if err := r.pool.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.recuperar_llamamiento_emitido_v1($1,$2)`, bolsa, clave).Scan(&raw); err != nil {
		return ports.EmisionLlamamiento{}, errorEmision(err)
	}
	var out ports.EmisionLlamamiento
	if json.Unmarshal(raw, &out) != nil {
		return out, ports.ErrEmisionLlamamientoNoDisponible
	}
	out.Reutilizada = true
	out, err := r.conFuentesCorreo(ctx, out, bolsa, clave)
	if err != nil {
		return ports.EmisionLlamamiento{}, err
	}
	return r.conAvisosExternos(ctx, out, bolsa, clave)
}

func (r *RepositorioEmisionLlamamientoPostgreSQL) ContarEnCurso(ctx context.Context, bolsa string) (int, error) {
	if r == nil || r.pool == nil || ctx == nil || bolsa == "" {
		return 0, ports.ErrEmisionLlamamientoNoDisponible
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.contar_llamamientos_en_curso_v1($1)`, bolsa).Scan(&total); err != nil {
		return 0, errorEmision(err)
	}
	return total, nil
}

func errorEmision(err error) error {
	var p *pgconn.PgError
	if errors.As(err, &p) {
		switch p.Code {
		case "42501":
			return dominiovec.ErrAutorizacionDenegada
		case "VBE01":
			return ports.ErrEmisionLlamamientoConflicto
		case "22023", "23503":
			return ports.ErrEmisionLlamamientoInvalida
		}
	}
	return ports.ErrEmisionLlamamientoNoDisponible
}

// emisionConFuentesSQL es la salida de registrar_contactos_llamamiento_v2:
// la emisión de siempre más las fuentes por recibo de contacto (B59).
type emisionConFuentesSQL struct {
	ports.EmisionLlamamiento
	FuentesCorreo map[string]ports.FuenteCorreoContacto `json:"fuentes_correo"`
}

// aplicarFuentesCorreo copia en cada contacto su fuente registrada. Una
// fuente que no cumple el contrato se descarta: no se muestra algo dudoso.
func aplicarFuentesCorreo(e ports.EmisionLlamamiento, fuentes map[string]ports.FuenteCorreoContacto) ports.EmisionLlamamiento {
	if len(fuentes) == 0 {
		return e
	}
	contactos := make([]ports.ResultadoContactoEmision, len(e.Contactos))
	for i, c := range e.Contactos {
		if f, ok := fuentes[c.ReciboRef]; ok && f.Valida() {
			copia := f
			c.FuenteCorreo = &copia
		}
		contactos[i] = c
	}
	e.Contactos = contactos
	return e
}

// ActivarFuentesCorreo comprueba que B59 está instalado y accesible al
// ejecutor. Sin esta llamada el repositorio se comporta como antes de B59.
func (r *RepositorioEmisionLlamamientoPostgreSQL) ActivarFuentesCorreo(ctx context.Context) error {
	if r == nil || r.pool == nil || ctx == nil || r.avisosExternos {
		return ports.ErrEmisionLlamamientoNoDisponible
	}
	var ok bool
	if err := r.pool.QueryRow(ctx, `SELECT pg_catalog.to_regclass('vec_bolsa_llamamientos.contacto_fuente_correo') IS NOT NULL
 AND pg_catalog.has_function_privilege(session_user,'vec_bolsa_llamamientos.registrar_contactos_llamamiento_v2(text,text,text,bytea,jsonb)','EXECUTE')
 AND pg_catalog.has_function_privilege(session_user,'vec_bolsa_llamamientos.fuentes_correo_llamamiento_v1(text,text)','EXECUTE')
 AND pg_catalog.has_function_privilege(session_user,'vec_bolsa_llamamientos.candidato_participacion_avisos_v1(text,text,text)','EXECUTE')`).Scan(&ok); err != nil || !ok {
		return ports.ErrEmisionLlamamientoNoDisponible
	}
	r.fuentes = true
	return nil
}

// conFuentesCorreo añade a una emisión recuperada las fuentes registradas.
func (r *RepositorioEmisionLlamamientoPostgreSQL) conFuentesCorreo(ctx context.Context, e ports.EmisionLlamamiento, bolsa, clave string) (ports.EmisionLlamamiento, error) {
	if !r.fuentes || len(e.Contactos) == 0 {
		return e, nil
	}
	var raw []byte
	if err := r.pool.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.fuentes_correo_llamamiento_v1($1,$2)`, bolsa, clave).Scan(&raw); err != nil {
		return ports.EmisionLlamamiento{}, errorEmision(err)
	}
	var fuentes map[string]ports.FuenteCorreoContacto
	if json.Unmarshal(raw, &fuentes) != nil {
		return ports.EmisionLlamamiento{}, ports.ErrEmisionLlamamientoNoDisponible
	}
	return aplicarFuentesCorreo(e, fuentes), nil
}

// CandidatoParticipacionAvisos devuelve la referencia de candidato de una
// participación de un llamamiento reservado de la bolsa, o "" si no hay
// vínculo (B59).
func (r *RepositorioEmisionLlamamientoPostgreSQL) CandidatoParticipacionAvisos(ctx context.Context, bolsa, llamamiento, participacion string) (string, error) {
	if r == nil || r.pool == nil || ctx == nil || !r.fuentes || bolsa == "" || llamamiento == "" || participacion == "" || len(participacion) > 512 {
		return "", ports.ErrEmisionLlamamientoNoDisponible
	}
	var candidato *string
	if err := r.pool.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.candidato_participacion_avisos_v1($1,$2,$3)`, bolsa, llamamiento, participacion).Scan(&candidato); err != nil {
		return "", errorEmision(err)
	}
	if candidato == nil {
		return "", nil
	}
	return *candidato, nil
}
