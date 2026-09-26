package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const esquemaIncorporacionesCentroSQL = "vec.contratacion-temporal.incorporaciones-centro.v1"

// ProveedorAutorizacionIncorporacionCentro aplica la identidad y la política
// vigentes del servidor y entrega una autorización nueva ligada al recurso
// exacto (acción, referencia, ámbitos y huella del material).
type ProveedorAutorizacionIncorporacionCentro interface {
	AutorizarIncorporacionCentro(ctx context.Context, accion string, recurso vecdomain.RecursoAutorizable) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// RepositorioIncorporacionCentroPostgreSQL usa las fachadas de CT124 con el
// LOGIN ejecutor de CT.
type RepositorioIncorporacionCentroPostgreSQL struct {
	pool      iniciadorTransacciones
	proveedor ProveedorAutorizacionIncorporacionCentro
}

var _ ports.RepositorioIncorporacionCentro = (*RepositorioIncorporacionCentroPostgreSQL)(nil)

func NuevoRepositorioIncorporacionCentroPostgreSQL(pool *pgxpool.Pool, proveedor ProveedorAutorizacionIncorporacionCentro) (*RepositorioIncorporacionCentroPostgreSQL, error) {
	if pool == nil || dependenciaNula(proveedor) {
		return nil, ports.ErrIncorporacionCentroNoDisponible
	}
	return &RepositorioIncorporacionCentroPostgreSQL{pool: pool, proveedor: proveedor}, nil
}

func (r *RepositorioIncorporacionCentroPostgreSQL) ListarIncorporacionesCentro(ctx context.Context, c ports.ConsultaIncorporacionesCentro) ([]ports.ExpedienteIncorporacionCentro, error) {
	contenido, err := ports.SerializarConsultaIncorporacionesCentro(c)
	if err != nil {
		return nil, err
	}
	recurso := ports.RecursoIncorporacionCentro(ports.ReferenciaBandejaIncorporacionesCentro(c.Actor.CentroRef), c.OrganizacionRef, c.Actor.CentroRef, contenido)
	var salida struct {
		Esquema     string                                `json:"esquema"`
		Expedientes []ports.ExpedienteIncorporacionCentro `json:"expedientes"`
	}
	err = r.ejecutar(ctx, "consultar_incorporaciones_centro_v1", contenido, ports.AccionConsultarIncorporacionesCentro, recurso, func(b []byte) error {
		if decodificarJSONEstricto(b, &salida) != nil || salida.Esquema != esquemaIncorporacionesCentroSQL || salida.Expedientes == nil ||
			len(salida.Expedientes) > ports.LimiteExpedientesIncorporacionCentro() {
			return ports.ErrIncorporacionCentroNoDisponible
		}
		vistos := make(map[string]bool, len(salida.Expedientes))
		for i := range salida.Expedientes {
			e := &salida.Expedientes[i]
			if !e.Valido() || vistos[e.ExpedienteRef] {
				return ports.ErrIncorporacionCentroNoDisponible
			}
			if e.Confirmacion != nil {
				e.Confirmacion.RegistradaEn = e.Confirmacion.RegistradaEn.UTC()
			}
			vistos[e.ExpedienteRef] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return salida.Expedientes, nil
}

// ExpedienteDelCentro dice si el expediente exacto procede de una petición
// del centro del actor (CT124, expediente_del_centro_v1). No consume
// autorización: la exige la operación que la usa. Sin paginar.
func (r *RepositorioIncorporacionCentroPostgreSQL) ExpedienteDelCentro(ctx context.Context, organizacionRef string, actor domain.ActorPeticionCentro, expedienteRef string) (bool, error) {
	if r == nil || r.pool == nil || ctx == nil || actor.Validar() != nil || !domain.ReferenciaOpacaValida(organizacionRef) ||
		!domain.ReferenciaOpacaValida(expedienteRef) {
		return false, ports.ErrIncorporacionCentroInvalida
	}
	contenido, err := json.Marshal(actor)
	if err != nil {
		return false, ports.ErrIncorporacionCentroInvalida
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return false, errorIncorporacionCentroSQL(ctx, err)
	}
	defer revertirTransaccion(tx)
	var suyo bool
	if err := tx.QueryRow(ctx, "SELECT vec_contratacion_temporal.expediente_del_centro_v1($1::text,$2::text,$3::text)",
		string(contenido), organizacionRef, expedienteRef).Scan(&suyo); err != nil {
		return false, errorIncorporacionCentroSQL(ctx, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, errorIncorporacionCentroSQL(ctx, err)
	}
	return suyo, nil
}

func (r *RepositorioIncorporacionCentroPostgreSQL) ConfirmarIncorporacionCentro(ctx context.Context, m ports.MaterialIncorporacionCentro) (ports.ReciboIncorporacionCentro, error) {
	var recibo ports.ReciboIncorporacionCentro
	contenido, err := ports.SerializarMaterialIncorporacionCentro(m)
	if err != nil {
		return recibo, err
	}
	recurso := ports.RecursoIncorporacionCentro(m.ExpedienteRef, m.OrganizacionRef, m.Actor.CentroRef, contenido)
	err = r.ejecutar(ctx, "confirmar_incorporacion_centro_v1", contenido, ports.AccionConfirmarIncorporacionCentro, recurso, func(b []byte) error {
		if decodificarJSONEstricto(b, &recibo) != nil {
			return ports.ErrIncorporacionCentroNoDisponible
		}
		recibo.RegistradoEn = recibo.RegistradoEn.UTC()
		return recibo.ValidarPara(m)
	})
	if err != nil {
		return ports.ReciboIncorporacionCentro{}, err
	}
	return recibo, nil
}

func (r *RepositorioIncorporacionCentroPostgreSQL) ejecutar(ctx context.Context, funcion string, contenido []byte, accion string, recurso vecdomain.RecursoAutorizable, validar func([]byte) error) error {
	// Nombres cerrados; nunca se interpola SQL, identidad ni función del cliente.
	if funcion != "consultar_incorporaciones_centro_v1" && funcion != "confirmar_incorporacion_centro_v1" {
		return ports.ErrIncorporacionCentroNoDisponible
	}
	if r == nil || r.pool == nil || dependenciaNula(r.proveedor) || ctx == nil {
		return ports.ErrIncorporacionCentroNoDisponible
	}
	a, err := r.proveedor.AutorizarIncorporacionCentro(ctx, accion, recurso)
	if err != nil {
		return err
	}
	h, err := recurso.HuellaContextoAutorizacionSHA256()
	resumen := a.ResumenCapacidad()
	if err != nil || a.ValidarEstructura() != nil || resumen.Operacion() != accion || resumen.EfectoRef() != recurso.Referencia ||
		resumen.EfectoHuellaSHA256() != h || resumen.AudienciaConsumo() != audienciaPeticionCentro {
		return ports.ErrIncorporacionCentroDenegada
	}
	tx, err := iniciarTransaccionAltaCandidata(ctx, r.pool)
	if err != nil {
		return errorIncorporacionCentroSQL(ctx, err)
	}
	defer revertirTransaccion(tx)
	secretos := [][]byte{a.CapacidadCanonica(), a.DecisionCanonica(), a.MotivoCanonico(), a.ContextoActorCanonico(), a.PayloadVECAD3(), a.SobreCOSESign1(), a.EvidenciaVerificacion(), a.RaizPublicaSPKI()}
	defer func() {
		for _, b := range secretos {
			clear(b)
		}
	}()
	var salida []byte
	err = tx.QueryRow(ctx, "SELECT vec_contratacion_temporal."+funcion+"($1::text,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)::text", string(contenido),
		secretos[0], secretos[1], secretos[2], secretos[3], int64(a.PersonaVersion()), int64(a.PerfilVersion()),
		secretos[4], secretos[5], secretos[6], secretos[7]).Scan(&salida)
	if err != nil {
		return errorIncorporacionCentroSQL(ctx, err)
	}
	defer clear(salida)
	if len(salida) == 0 || len(salida) > 2*1024*1024 {
		return ports.ErrIncorporacionCentroNoDisponible
	}
	if err := validar(salida); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return errorIncorporacionCentroSQL(ctx, err)
	}
	return nil
}

func errorIncorporacionCentroSQL(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "22023":
			return ports.ErrIncorporacionCentroInvalida
		case "P0681":
			return ports.ErrClaveIncorporacionCentroUsada
		case "P0682":
			return ports.ErrIncorporacionCentroNoAdmitida
		case "42501":
			return ports.ErrIncorporacionCentroDenegada
		}
	}
	// Concurrencia serializable o COMMIT incierto: se conserva la clave para
	// reintentar; nunca se declara confirmado lo que no se recibió.
	return ports.ErrIncorporacionCentroNoDisponible
}
