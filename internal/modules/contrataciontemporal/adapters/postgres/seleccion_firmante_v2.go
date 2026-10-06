package postgres

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// AUT56: la fachada exige SERIALIZABLE de lectura y escritura, como la
// transacción de la firma; sólo lee y bloquea con FOR SHARE.
const seleccionarFirmanteSQL56 = `SELECT vec_autorizacion.seleccionar_firmante_plan_ct_v1($1,$2,$3,$4,$5,$6,$7,$8)::text`

var (
	huellaSeleccionFirmante  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	esquemaSeleccionFirmante = "vec.autorizacion.seleccion-firmante-plan.ct.v1"
)

type iniciadorSeleccionFirmante interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// SeleccionFirmantePostgreSQL lee con el LOGIN CT la selección central del
// firmante (AUT56). La transacción se deshace siempre: no hay efectos.
type SeleccionFirmantePostgreSQL struct {
	pool iniciadorSeleccionFirmante
}

var _ ports.FuenteSeleccionFirmanteV2 = (*SeleccionFirmantePostgreSQL)(nil)

func NuevaSeleccionFirmantePostgreSQL(pool iniciadorSeleccionFirmante) (*SeleccionFirmantePostgreSQL, error) {
	if nuloRegistroTX(pool) {
		return nil, ports.ErrCompetenciaFirmanteNoDisponible
	}
	return &SeleccionFirmantePostgreSQL{pool: pool}, nil
}

type respuestaSeleccionFirmante56 struct {
	Esquema            string                               `json:"esquema"`
	PersonaRef         string                               `json:"persona_ref"`
	CuentaRef          string                               `json:"cuenta_ref"`
	PerfilActivoRef    string                               `json:"perfil_activo_ref"`
	RolID              string                               `json:"rol_id"`
	CargoRef           string                               `json:"cargo_ref"`
	EnlaceEjercicioRef string                               `json:"enlace_ejercicio_ref"`
	VinculoCertificado ports.ReferenciaVersionadaFirmanteV2 `json:"vinculo_certificado"`
	Asignacion         struct {
		ports.ReferenciaVersionadaFirmanteV2
		VigenteDesde string `json:"vigente_desde"`
		VigenteHasta string `json:"vigente_hasta"`
	} `json:"asignacion"`
	Rol struct {
		Referencia   string `json:"referencia"`
		HuellaSHA256 string `json:"huella_sha256"`
	} `json:"rol"`
	ControlRol struct {
		Referencia   string `json:"referencia"`
		Revision     uint64 `json:"revision"`
		HuellaSHA256 string `json:"huella_sha256"`
	} `json:"control_rol"`
}

func (s *SeleccionFirmantePostgreSQL) SeleccionarFirmanteV2(ctx context.Context, q ports.SolicitudSeleccionFirmanteV2) (ports.SeleccionFirmanteV2, error) {
	var cero ports.SeleccionFirmanteV2
	if s == nil || nuloRegistroTX(s.pool) || ctx == nil {
		return cero, ports.ErrCompetenciaFirmanteNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if !huellaSeleccionFirmante.MatchString(q.CertificadoHuella) || q.CargoRef == "" || q.RolID == "" || q.Accion == "" ||
		q.TipoRecurso == "" || q.Finalidad == "" || q.OrganizacionRef == "" || q.UnidadRef == "" {
		return cero, ports.ErrCompetenciaFirmanteNoAcreditada
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || nuloRegistroTX(tx) {
		return cero, errorSeleccionFirmante(ctx, err)
	}
	defer func() {
		c, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelar()
		_ = tx.Rollback(c)
	}()
	var bruto []byte
	if err := tx.QueryRow(ctx, seleccionarFirmanteSQL56, q.CertificadoHuella, q.CargoRef, q.RolID, q.Accion, q.TipoRecurso,
		q.Finalidad, q.OrganizacionRef, q.UnidadRef).Scan(&bruto); err != nil {
		return cero, errorSeleccionFirmante(ctx, err)
	}
	var r respuestaSeleccionFirmante56
	if decodificarFirma118(bruto, &r) != nil || r.Esquema != esquemaSeleccionFirmante || r.RolID != q.RolID || r.CargoRef != q.CargoRef ||
		r.PersonaRef == "" || r.CuentaRef == "" || r.PerfilActivoRef == "" || r.EnlaceEjercicioRef == "" ||
		!referenciaVersionadaValida(r.VinculoCertificado) || !referenciaVersionadaValida(r.Asignacion.ReferenciaVersionadaFirmanteV2) ||
		r.Rol.Referencia == "" || !huellaSeleccionFirmante.MatchString(r.Rol.HuellaSHA256) ||
		r.ControlRol.Referencia != r.Rol.Referencia || r.ControlRol.Revision == 0 || !huellaSeleccionFirmante.MatchString(r.ControlRol.HuellaSHA256) {
		return cero, ports.ErrCompetenciaFirmanteNoDisponible
	}
	return ports.SeleccionFirmanteV2{PersonaRef: r.PersonaRef, CuentaRef: r.CuentaRef, PerfilActivoRef: r.PerfilActivoRef, RolID: r.RolID,
		CargoRef: r.CargoRef, EnlaceEjercicioRef: r.EnlaceEjercicioRef, VinculoCertificado: r.VinculoCertificado,
		Asignacion: r.Asignacion.ReferenciaVersionadaFirmanteV2, AsignacionVigenteDesde: r.Asignacion.VigenteDesde,
		AsignacionVigenteHasta: r.Asignacion.VigenteHasta, RolRef: r.Rol.Referencia, RolHuellaSHA256: r.Rol.HuellaSHA256,
		ControlRol: ports.ReferenciaVersionadaFirmanteV2{Referencia: r.ControlRol.Referencia, Version: r.ControlRol.Revision,
			HuellaSHA256: r.ControlRol.HuellaSHA256}}, nil
}

func referenciaVersionadaValida(r ports.ReferenciaVersionadaFirmanteV2) bool {
	return r.Referencia != "" && r.Version > 0 && huellaSeleccionFirmante.MatchString(r.HuellaSHA256)
}

// errorSeleccionFirmante: 42501 es que no hay selección única (o no está
// permitida); el resto, indisponibilidad. Nunca conserva el mensaje SQL.
func errorSeleccionFirmante(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "42501" {
		return ports.ErrCompetenciaFirmanteNoAcreditada
	}
	return ports.ErrCompetenciaFirmanteNoDisponible
}
