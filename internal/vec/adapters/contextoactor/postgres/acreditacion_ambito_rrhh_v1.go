package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/vec/ports"
)

const consultaResolverComprobanteAmbitoRRHHV1 = `
SELECT vec_contexto_actor_v1.resolver_comprobante_ambito_rrhh_v1(
 $1::text,$2::numeric,$3::text,$4::numeric,
 $5::text,$6::numeric,$7::text,$8::numeric)`

type lectorFilaAmbitoRRHHV1 interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// ResolutorComprobanteAmbitoCorporativoRRHHV1 usa exclusivamente una conexion
// nominal con el rol selector ContextoActor. SQL revalida en CT al consumir.
type ResolutorComprobanteAmbitoCorporativoRRHHV1 struct {
	lector lectorFilaAmbitoRRHHV1
}

var patronLoginSelectorAmbitoRRHHV1 = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// NuevoResolutorComprobanteAmbitoCorporativoRRHHV1 exige el pool TLS del
// selector corporativo y acredita la membresia nominal exclusiva antes de
// entregar el puerto. El grupo NOLOGIN no puede usarse como usuario de DSN.
func NuevoResolutorComprobanteAmbitoCorporativoRRHHV1(ctx context.Context, pool *pgxpool.Pool, loginNominal string) (*ResolutorComprobanteAmbitoCorporativoRRHHV1, error) {
	if ctx == nil || pool == nil || !patronLoginSelectorAmbitoRRHHV1.MatchString(loginNominal) ||
		loginNominal == "vec_contexto_actor_corporativo_rrhh_selector" {
		return nil, ports.ErrComprobanteAmbitoCorporativoRRHHV1Invalido
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cfg := pool.Config()
	if cfg == nil || cfg.ConnConfig == nil || cfg.ConnConfig.User != loginNominal ||
		cfg.ConnConfig.TLSConfig == nil || cfg.ConnConfig.TLSConfig.InsecureSkipVerify ||
		cfg.ConnConfig.TLSConfig.ServerName == "" {
		return nil, ports.ErrComprobanteAmbitoCorporativoRRHHV1Invalido
	}
	if err := acreditarSelectorAmbitoRRHHV1(ctx, pool, loginNominal); err != nil {
		return nil, err
	}
	return nuevoResolutorComprobanteAmbitoCorporativoRRHHV1(pool)
}

const consultaSelectorAmbitoRRHHV1 = `SELECT session_user::text=$1::text
 AND EXISTS(SELECT 1 FROM pg_roles u WHERE u.rolname=session_user AND u.rolcanlogin
   AND u.rolinherit AND NOT u.rolsuper AND NOT u.rolcreatedb AND NOT u.rolcreaterole
   AND NOT u.rolreplication AND NOT u.rolbypassrls)
 AND EXISTS(SELECT 1 FROM pg_roles g WHERE g.rolname='vec_contexto_actor_corporativo_rrhh_selector'
   AND NOT g.rolcanlogin AND NOT g.rolinherit AND NOT g.rolsuper
   AND NOT g.rolcreatedb AND NOT g.rolcreaterole AND NOT g.rolreplication AND NOT g.rolbypassrls)
 AND (SELECT count(*) FROM pg_auth_members m JOIN pg_roles u ON u.oid=m.member
   WHERE u.rolname=session_user)=1
 AND EXISTS(SELECT 1 FROM pg_auth_members m JOIN pg_roles u ON u.oid=m.member
   JOIN pg_roles g ON g.oid=m.roleid WHERE u.rolname=session_user
   AND g.rolname='vec_contexto_actor_corporativo_rrhh_selector'
   AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
 AND NOT EXISTS(SELECT 1 FROM pg_auth_members m JOIN pg_roles g ON g.oid=m.member
   WHERE g.rolname='vec_contexto_actor_corporativo_rrhh_selector')`

func acreditarSelectorAmbitoRRHHV1(ctx context.Context, lector lectorFilaAmbitoRRHHV1, login string) error {
	if ctx == nil || lector == nil {
		return ports.ErrComprobanteAmbitoCorporativoRRHHV1Invalido
	}
	var correcto bool
	if err := lector.QueryRow(ctx, consultaSelectorAmbitoRRHHV1, login).Scan(&correcto); err != nil || !correcto {
		return ports.ErrComprobanteAmbitoCorporativoRRHHV1Invalido
	}
	return nil
}

func nuevoResolutorComprobanteAmbitoCorporativoRRHHV1(lector lectorFilaAmbitoRRHHV1) (*ResolutorComprobanteAmbitoCorporativoRRHHV1, error) {
	if lector == nil {
		return nil, ports.ErrComprobanteAmbitoCorporativoRRHHV1Invalido
	}
	return &ResolutorComprobanteAmbitoCorporativoRRHHV1{lector: lector}, nil
}

func (r *ResolutorComprobanteAmbitoCorporativoRRHHV1) ResolverComprobanteAmbitoCorporativoRRHHV1(
	ctx context.Context,
	cuentaRef, personaRef, perfilRef, contextoRef string,
	cuentaVersion, personaVersion, perfilVersion, contextoVersion uint64,
) (ports.ComprobanteAmbitoCorporativoRRHHV1, error) {
	if ctx == nil || r == nil || r.lector == nil {
		return ports.ComprobanteAmbitoCorporativoRRHHV1{}, ports.ErrComprobanteAmbitoCorporativoRRHHV1Invalido
	}
	if err := ctx.Err(); err != nil {
		return ports.ComprobanteAmbitoCorporativoRRHHV1{}, err
	}
	var crudo []byte
	err := r.lector.QueryRow(ctx, consultaResolverComprobanteAmbitoRRHHV1,
		cuentaRef, strconv.FormatUint(cuentaVersion, 10),
		personaRef, strconv.FormatUint(personaVersion, 10),
		perfilRef, strconv.FormatUint(perfilVersion, 10),
		contextoRef, strconv.FormatUint(contextoVersion, 10),
	).Scan(&crudo)
	defer clear(crudo)
	if err != nil || len(crudo) == 0 || len(crudo) > 8192 {
		return ports.ComprobanteAmbitoCorporativoRRHHV1{}, errors.Join(ports.ErrComprobanteAmbitoCorporativoRRHHV1Invalido, err)
	}
	var dto comprobanteAmbitoRRHHV1JSON
	dec := json.NewDecoder(bytes.NewReader(crudo))
	dec.DisallowUnknownFields()
	if dec.Decode(&dto) != nil || dec.Decode(new(any)) != io.EOF {
		return ports.ComprobanteAmbitoCorporativoRRHHV1{}, ports.ErrComprobanteAmbitoCorporativoRRHHV1Invalido
	}
	datos := dto.datos()
	if datos.CuentaRef != cuentaRef || datos.PersonaRef != personaRef || datos.PerfilRef != perfilRef ||
		datos.ContextoRef != contextoRef || datos.CuentaVersion != cuentaVersion ||
		datos.PersonaVersion != personaVersion || datos.PerfilVersion != perfilVersion ||
		datos.ContextoVersion != contextoVersion {
		return ports.ComprobanteAmbitoCorporativoRRHHV1{}, ports.ErrComprobanteAmbitoCorporativoRRHHV1Invalido
	}
	return ports.NuevoComprobanteAmbitoCorporativoRRHHV1(datos)
}

type comprobanteAmbitoRRHHV1JSON struct {
	CuentaRef                           string    `json:"cuenta_ref"`
	CuentaVersion                       uint64    `json:"cuenta_version"`
	PersonaRef                          string    `json:"persona_ref"`
	PersonaVersion                      uint64    `json:"persona_version"`
	PerfilRef                           string    `json:"perfil_ref"`
	PerfilVersion                       uint64    `json:"perfil_version"`
	ContextoRef                         string    `json:"contexto_ref"`
	ContextoVersion                     uint64    `json:"contexto_version"`
	VinculoCorporativoRef               string    `json:"vinculo_corporativo_ref"`
	VinculoCorporativoVersion           uint64    `json:"vinculo_corporativo_version"`
	OrganizacionRef                     string    `json:"organizacion_ref"`
	OrganizacionVersion                 uint64    `json:"organizacion_version"`
	OrganizacionProcedenciaRef          string    `json:"organizacion_procedencia_ref"`
	OrganizacionProcedenciaVersion      uint64    `json:"organizacion_procedencia_version"`
	OrganizacionProcedenciaHuellaSHA256 string    `json:"organizacion_procedencia_huella_sha256"`
	OrganizacionProcedenciaAutoridad    string    `json:"organizacion_procedencia_autoridad"`
	VinculoProcedenciaRef               string    `json:"vinculo_procedencia_ref"`
	VinculoProcedenciaVersion           uint64    `json:"vinculo_procedencia_version"`
	VinculoProcedenciaHuellaSHA256      string    `json:"vinculo_procedencia_huella_sha256"`
	VinculoProcedenciaAutoridad         string    `json:"vinculo_procedencia_autoridad"`
	Superficie                          string    `json:"superficie"`
	Uso                                 string    `json:"uso"`
	VigenteDesde                        time.Time `json:"vigente_desde"`
	VigenteHasta                        time.Time `json:"vigente_hasta"`
}

func (d comprobanteAmbitoRRHHV1JSON) datos() ports.DatosComprobanteAmbitoCorporativoRRHHV1 {
	return ports.DatosComprobanteAmbitoCorporativoRRHHV1{
		CuentaRef: d.CuentaRef, CuentaVersion: d.CuentaVersion,
		PersonaRef: d.PersonaRef, PersonaVersion: d.PersonaVersion,
		PerfilRef: d.PerfilRef, PerfilVersion: d.PerfilVersion,
		ContextoRef: d.ContextoRef, ContextoVersion: d.ContextoVersion,
		VinculoCorporativoRef:     d.VinculoCorporativoRef,
		VinculoCorporativoVersion: d.VinculoCorporativoVersion,
		OrganizacionRef:           d.OrganizacionRef, OrganizacionVersion: d.OrganizacionVersion,
		OrganizacionProcedenciaRef:          d.OrganizacionProcedenciaRef,
		OrganizacionProcedenciaVersion:      d.OrganizacionProcedenciaVersion,
		OrganizacionProcedenciaHuellaSHA256: d.OrganizacionProcedenciaHuellaSHA256,
		OrganizacionProcedenciaAutoridad:    d.OrganizacionProcedenciaAutoridad,
		VinculoProcedenciaRef:               d.VinculoProcedenciaRef,
		VinculoProcedenciaVersion:           d.VinculoProcedenciaVersion,
		VinculoProcedenciaHuellaSHA256:      d.VinculoProcedenciaHuellaSHA256,
		VinculoProcedenciaAutoridad:         d.VinculoProcedenciaAutoridad,
		Superficie:                          d.Superficie, Uso: d.Uso,
		VigenteDesde: d.VigenteDesde.UTC(), VigenteHasta: d.VigenteHasta.UTC(),
	}
}
