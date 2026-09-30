package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strconv"

	"github.com/jackc/pgx/v5"
	core "vec-diputacion-granada/internal/vec/domain"
)

// SolicitudProvisionUsuariosExterno es material administrativo, nunca una
// configuración del servidor. ContextoActor debe provisionarse previamente
// por su herramienta propia; AUT17 acredita esa misma cuenta/persona/perfil.
type SolicitudProvisionUsuariosExterno struct {
	CuentaRef                 string                       `json:"cuenta_ref"`
	AprobacionRef             string                       `json:"aprobacion_ref"`
	Instantanea               core.InstantaneaAutorizacion `json:"instantanea"`
	RevisionControlEsperada   uint64                       `json:"revision_control_esperada"`
	HuellaControlEsperada     string                       `json:"huella_control_esperada"`
	VersionAsignacionEsperada int                          `json:"version_asignacion_esperada"`
	HuellaAsignacionEsperada  string                       `json:"huella_asignacion_esperada"`
}

// LeerSolicitudProvisionUsuariosExterno limita el material de operación y
// rechaza propiedades desconocidas y documentos concatenados.
func LeerSolicitudProvisionUsuariosExterno(r io.Reader) (SolicitudProvisionUsuariosExterno, error) {
	var s SolicitudProvisionUsuariosExterno
	if r == nil {
		return s, errPerfilUsuariosExterno
	}
	b, err := io.ReadAll(io.LimitReader(r, 262145))
	if err != nil || len(b) > 262144 {
		return s, errPerfilUsuariosExterno
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if validarClavesJSONUnicas(b) != nil || d.Decode(&s) != nil || d.Decode(new(any)) != io.EOF {
		return SolicitudProvisionUsuariosExterno{}, errPerfilUsuariosExterno
	}
	if _, err := s.HuellaSHA256(); err != nil {
		return SolicitudProvisionUsuariosExterno{}, err
	}
	return s, nil
}

func (s SolicitudProvisionUsuariosExterno) preparar() (publicacionPerfilUsuariosExterno, error) {
	return prepararPublicacionPerfilUsuariosExterno(s.Instantanea, s.CuentaRef, s.AprobacionRef,
		s.RevisionControlEsperada, s.HuellaControlEsperada, s.VersionAsignacionEsperada, s.HuellaAsignacionEsperada)
}

// HuellaSHA256 fija el material completo, incluida la aprobación y ambas
// preimágenes. Preparar o mostrar la huella no abre ninguna conexión.
func (s SolicitudProvisionUsuariosExterno) HuellaSHA256() (string, error) {
	if _, err := s.preparar(); err != nil {
		return "", err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return "", errPerfilUsuariosExterno
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

type ReciboProvisionUsuariosExterno struct {
	HuellaPlan        string `json:"plan_huella_sha256"`
	AprobacionRef     string `json:"aprobacion_ref"`
	RolRef            string `json:"rol_ref"`
	RevisionControl   uint64 `json:"revision_control"`
	HuellaRol         string `json:"rol_huella_sha256"`
	HuellaControl     string `json:"control_huella_sha256"`
	AsignacionRef     string `json:"asignacion_ref"`
	VersionAsignacion int    `json:"version_asignacion"`
	HuellaAsignacion  string `json:"asignacion_huella_sha256"`
}

// PublicarPerfilUsuariosExterno solo se invoca desde una herramienta de
// operación explícita. Las fachadas AUT17 rechazan LOGIN web, privilegios
// amplios y preimágenes divergentes. Un reintento no sobreescribe el CAS.
func PublicarPerfilUsuariosExterno(ctx context.Context, con *pgx.Conn, s SolicitudProvisionUsuariosExterno, huellaAprobada string) (r ReciboProvisionUsuariosExterno, e error) {
	if ctx == nil || ctx.Err() != nil || con == nil {
		return ReciboProvisionUsuariosExterno{}, errPerfilUsuariosExterno
	}
	h, err := s.HuellaSHA256()
	if err != nil || !huellaProvisionUsuariosValida(huellaAprobada) || h != huellaAprobada {
		return ReciboProvisionUsuariosExterno{}, errPerfilUsuariosExterno
	}
	tx, err := con.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return ReciboProvisionUsuariosExterno{}, errPerfilUsuariosExterno
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			r = ReciboProvisionUsuariosExterno{}
			e = errPerfilUsuariosExterno
		}
	}()
	return publicarPerfilUsuariosEnTransaccion(ctx, tx, s, h)
}

type transaccionProvisionUsuarios interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Commit(context.Context) error
}

const sqlPublicarRolUsuarios = `SELECT version_rol_ref,version,revision::text,huella_rol,huella_control
 FROM vec_autorizacion.publicar_rol_usuarios_externo_v1($1,$2,$3,$4,$5::numeric,$6,$7,$8)`
const sqlPublicarAsignacionUsuarios = `SELECT asignacion_ref,version,huella_sha256
 FROM vec_autorizacion.publicar_asignacion_usuarios_externo_v1($1,$2,$3::bigint,$4,$5,$6,$7)`

func publicarPerfilUsuariosEnTransaccion(ctx context.Context, tx transaccionProvisionUsuarios, s SolicitudProvisionUsuariosExterno, huella string) (ReciboProvisionUsuariosExterno, error) {
	p, err := s.preparar()
	if err != nil {
		return ReciboProvisionUsuariosExterno{}, err
	}
	r := ReciboProvisionUsuariosExterno{HuellaPlan: huella, AprobacionRef: s.AprobacionRef}
	var versionRol int
	var revision string
	err = tx.QueryRow(ctx, sqlPublicarRolUsuarios, p.Rol, p.HuellaRol, p.Control, p.HuellaControl,
		strconv.FormatUint(p.RevisionEsperada, 10), huellaPreviaUsuarios(p.HuellaControlEsperada),
		s.Instantanea.ControlVigenciaVersionRol.ActualizadoPor, p.AprobacionRef).Scan(&r.RolRef, &versionRol, &revision, &r.HuellaRol, &r.HuellaControl)
	var errRevision error
	r.RevisionControl, errRevision = strconv.ParseUint(revision, 10, 64)
	if err != nil || errRevision != nil || r.RolRef != s.Instantanea.VersionRol.Referencia() || versionRol != s.Instantanea.VersionRol.Version ||
		r.RevisionControl != s.Instantanea.ControlVigenciaVersionRol.Revision || r.HuellaRol != p.HuellaRol || r.HuellaControl != p.HuellaControl {
		return ReciboProvisionUsuariosExterno{}, errPerfilUsuariosExterno
	}
	err = tx.QueryRow(ctx, sqlPublicarAsignacionUsuarios, p.Asignacion, p.HuellaAsignacion, int64(p.VersionEsperada),
		huellaPreviaUsuarios(p.HuellaAsignacionEsperada), s.Instantanea.AsignacionPerfil.EmitidaPor, p.AprobacionRef, p.CuentaRef).
		Scan(&r.AsignacionRef, &r.VersionAsignacion, &r.HuellaAsignacion)
	if err != nil || r.AsignacionRef != s.Instantanea.AsignacionPerfil.Referencia() ||
		r.VersionAsignacion != s.Instantanea.AsignacionPerfil.Version || r.HuellaAsignacion != p.HuellaAsignacion {
		return ReciboProvisionUsuariosExterno{}, errPerfilUsuariosExterno
	}
	if tx.Commit(ctx) != nil {
		return ReciboProvisionUsuariosExterno{}, errPerfilUsuariosExterno
	}
	return r, nil
}

func huellaPreviaUsuarios(h string) any {
	if h == "" {
		return nil
	}
	return h
}
