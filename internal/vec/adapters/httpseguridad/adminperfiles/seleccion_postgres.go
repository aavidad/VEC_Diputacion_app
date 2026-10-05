package adminperfiles

import (
	"context"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
)

const listarPropiosSQL = `SELECT persona_ref,cuenta_ref,perfil_ref,vinculo_ref,audiencia,vigente_hasta,
 seleccionado,seleccion_revision::text,rol_version_ref,clave_i18n,categoria_admin
 FROM vec_identidad_sesiones_v1.listar_perfiles_admin_v1($1,$2,$3,$4,$5,$6,$7,$8,$9)`
const seleccionarPerfilSQL = `SELECT perfil_ref,seleccion_revision::text,seleccionada_en,auditoria_ref
 FROM vec_identidad_sesiones_v1.seleccionar_perfil_admin_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`

func (p *PostgreSQL) ListarPropiosADMIN(ctx context.Context, o ObservacionADMIN) (PerfilesPropios, error) {
	if p == nil || ctx == nil || ctx.Err() != nil || nulo(p.pool) || nulo(p.reloj) || !o.Valida(p.reloj.Ahora().UTC()) {
		return PerfilesPropios{}, api.ErrAutenticacionRequerida
	}
	var salida PerfilesPropios
	// Tras un POST con COMMIT incierto, una instantánea nueva después de los
	// bloqueos muestra la revisión confirmada; el lector SQL revalida la
	// identidad, asignaciones y selección dentro de este READ COMMITTED.
	err := p.transaccionConAislamiento(ctx, pgx.ReadCommitted, func(tx pgx.Tx) error {
		filas, err := tx.Query(ctx, listarPropiosSQL, argumentos(o)...)
		if err != nil {
			return err
		}
		defer filas.Close()
		var persona, cuenta string
		for filas.Next() {
			var actualPersona, actualCuenta, vinculo, audiencia, revision string
			var vigente time.Time
			var seleccionado bool
			var perfil PerfilPropio
			if err = filas.Scan(&actualPersona, &actualCuenta, &perfil.PerfilRef, &vinculo, &audiencia,
				&vigente, &seleccionado, &revision, &perfil.RolVersionRef, &perfil.ClaveI18N, &perfil.CategoriaADMIN); err != nil {
				return err
			}
			r, parseErr := strconv.ParseUint(revision, 10, 64)
			if parseErr != nil || !perfil.Valido() || !referencia(actualPersona, "per_") ||
				!referencia(actualCuenta, "cta_") || !referencia(vinculo, "vca_") ||
				!instante(vigente.UTC().Truncate(time.Microsecond)) || !p.reloj.Ahora().Before(vigente) ||
				audiencia == "" || len(audiencia) > 512 {
				return api.ErrConfiguracionIncompleta
			}
			if persona == "" {
				persona, cuenta, salida.Revision = actualPersona, actualCuenta, r
			} else if persona != actualPersona || cuenta != actualCuenta || salida.Revision != r {
				return api.ErrConfiguracionIncompleta
			}
			if seleccionado {
				if salida.PerfilActivoRef != "" {
					return api.ErrConfiguracionIncompleta
				}
				salida.PerfilActivoRef = perfil.PerfilRef
			}
			salida.Perfiles = append(salida.Perfiles, perfil)
			if len(salida.Perfiles) > 16 {
				return api.ErrConfiguracionIncompleta
			}
		}
		if err := filas.Err(); err != nil {
			return err
		}
		if len(salida.Perfiles) == 0 {
			return api.ErrAccesoDenegado
		}
		if !salida.Validos() {
			return api.ErrConfiguracionIncompleta
		}
		return nil
	})
	if err != nil {
		return PerfilesPropios{}, errorAutoridad(err)
	}
	return salida, nil
}

func (p *PostgreSQL) SeleccionarPerfilADMIN(ctx context.Context, o ObservacionADMIN, perfilRef string, revisionEsperada uint64) (SeleccionPerfil, error) {
	if p == nil || ctx == nil || ctx.Err() != nil || nulo(p.pool) || nulo(p.reloj) || !o.Valida(p.reloj.Ahora().UTC()) ||
		!referencia(perfilRef, "prf_") || revisionEsperada == ^uint64(0) {
		return SeleccionPerfil{}, api.ErrAutenticacionRequerida
	}
	var seleccion SeleccionPerfil
	err := p.transaccion(ctx, func(tx pgx.Tx) error {
		var err error
		seleccion, err = p.leerSeleccion(ctx, tx, seleccionarPerfilSQL,
			append(argumentos(o), perfilRef, strconv.FormatUint(revisionEsperada, 10))...)
		if err != nil {
			return err
		}
		if seleccion.PerfilActivoRef != perfilRef || seleccion.Revision < revisionEsperada ||
			seleccion.Revision > revisionEsperada+1 || !seleccion.Valida() {
			return api.ErrConfiguracionIncompleta
		}
		return nil
	})
	if err != nil {
		return SeleccionPerfil{}, errorAutoridad(err)
	}
	return seleccion, nil
}

func (p *PostgreSQL) leerSeleccion(ctx context.Context, tx pgx.Tx, consulta string, argumentos ...any) (SeleccionPerfil, error) {
	var seleccion SeleccionPerfil
	var revision string
	err := tx.QueryRow(ctx, consulta, argumentos...).Scan(&seleccion.PerfilActivoRef, &revision,
		&seleccion.SeleccionadaEn, &seleccion.AuditoriaRef)
	if err != nil {
		return SeleccionPerfil{}, err
	}
	seleccion.Revision, err = strconv.ParseUint(revision, 10, 64)
	seleccion.SeleccionadaEn = seleccion.SeleccionadaEn.UTC().Truncate(time.Microsecond)
	if err != nil || !seleccion.Valida() {
		return SeleccionPerfil{}, api.ErrConfiguracionIncompleta
	}
	return seleccion, nil
}

var _ FuenteSeleccionADMIN = (*PostgreSQL)(nil)
