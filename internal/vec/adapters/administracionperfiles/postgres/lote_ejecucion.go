package postgres

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// Un error de COMMIT no permite concluir si el efecto se aplicó. Sólo se
// expone la indisponibilidad y no se reintenta la operación automáticamente.
var errCommitLoteIndeterminado = errors.New("vec.admin.lote.commit_indeterminado")

var _ ports.CatalogoRolesAdministrables = (*AutoridadLoteOrdinario)(nil)

// ResolverRolAdministrable expone al servicio y al HTTP del lote el mismo
// cotejo con el perfil registrado (AUT24, con el LOGIN del lote).
func (a *AutoridadLoteOrdinario) ResolverRolAdministrable(ctx context.Context, ref string) (ports.RolAdministrable, error) {
	return a.resolverRolLote(ctx, ref)
}

func (a *AutoridadLoteOrdinario) resolverRolLote(ctx context.Context, ref string) (ports.RolAdministrable, error) {
	var vacio ports.RolAdministrable
	if ctx == nil || ctx.Err() != nil || a == nil || ausente(a.pool) || !domain.RolVersionAdministracionPerfilesValido(ref) {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	var b []byte
	if err := a.pool.QueryRow(ctx, catalogoSQL, ref).Scan(&b); err != nil {
		return vacio, traducir(ctx, err)
	}
	var x rolJSON
	if decodificar(b, &x) != nil || x.UnidadRequerida == nil {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	r := ports.RolAdministrable{VersionRef: x.VersionRef, Clase: x.Clase, HuellaSHA256: x.HuellaSHA256,
		VigenteDesde: x.VigenteDesde, VigenteHasta: x.VigenteHasta, UnidadRequerida: *x.UnidadRequerida}
	if x.CategoriaAdmin != nil {
		r.CategoriaAdmin = *x.CategoriaAdmin
	}
	if r.VersionRef != ref || r.ValidarEn(a.reloj.Ahora()) != nil {
		return vacio, domain.ErrActoAdministracionPerfilesInvalido
	}
	return r, nil
}

// Ejecuta la única fachada AUT44. El material de fuentes es un argumento
// privado adicional; no modifica el transporte ABI11 de actos singulares.
func (a *AutoridadLoteOrdinario) ejecutarLote(ctx context.Context, actor domain.ContextoActor,
	evidencia domain.EvidenciaSesionAdministracionPerfiles, instantanea domain.InstantaneaAutorizacion,
	recurso domain.RecursoAutorizable, efecto Efecto, fuentes []byte, validar func([]byte) error) error {
	var fuentesArg any
	if len(fuentes) != 0 {
		fuentesArg = string(fuentes)
	}
	return a.ejecutarConsumoLote(ctx, actor, evidencia, instantanea, recurso, efecto, aplicarLoteOrdinarioSQL,
		[]any{string(efecto.Material), fuentesArg}, validar)
}

// ejecutarConsumoLote pide la decisión al emisor y llama, en una transacción
// SERIALIZABLE, a una fachada que consume esa decisión por AD190 (aplicar o
// preparar). iniciales son los argumentos propios de la fachada, que preceden
// a los diez del material V3.
func (a *AutoridadLoteOrdinario) ejecutarConsumoLote(ctx context.Context, actor domain.ContextoActor,
	evidencia domain.EvidenciaSesionAdministracionPerfiles, instantanea domain.InstantaneaAutorizacion,
	recurso domain.RecursoAutorizable, efecto Efecto, sql string, iniciales []any, validar func([]byte) error) error {
	fallo := ports.ErrAutoridadAdministracionPerfilesNoDisponible
	if a == nil || ctx == nil || ctx.Err() != nil || ausente(a.pool) || ausente(a.emisor) ||
		ausente(a.reloj) || validar == nil || sql == "" || evidencia.ValidarEn(actor, a.reloj.Ahora()) != nil ||
		!domain.ReferenciaCorrelacionAutorizacionV2Valida(efecto.CorrelacionAccesoRef) {
		return fallo
	}
	contextoSHA, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || recurso.Referencia != efecto.Referencia || recurso.ModuloID != "administracion" ||
		recurso.Tipo != "persona" {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	// La solicitud y el recurso son nuevos valores. El emisor recibe su propia
	// copia y no puede alterar los bytes que consulta AUT44.
	entrega := efecto
	entrega.Material = append([]byte(nil), efecto.Material...)
	recursoEmisor := domain.RecursoAutorizable{Referencia: recurso.Referencia, ModuloID: recurso.ModuloID,
		Tipo: recurso.Tipo, Ambitos: make(map[string]string, len(recurso.Ambitos)),
		Atributos: make(map[string]string, len(recurso.Atributos))}
	for k, v := range recurso.Ambitos {
		recursoEmisor.Ambitos[k] = v
	}
	for k, v := range recurso.Atributos {
		recursoEmisor.Atributos[k] = v
	}
	material, err := a.emisor.EmitirLoteOrdinario(ctx, actor, evidencia, instantanea, recursoEmisor, entrega)
	clear(entrega.Material)
	if err != nil {
		// La denegación explícita del PDP se conserva: es un 403 auditado como
		// denegado, no una indisponibilidad.
		if ctx.Err() == nil && errors.Is(err, domain.ErrAutorizacionDenegada) {
			return domain.ErrAutorizacionDenegada
		}
		return traducir(ctx, err)
	}
	r := material.ResumenCapacidad()
	ahora := a.reloj.Ahora()
	if material.ValidarEstructura() != nil || r.Operacion() != efecto.Accion ||
		r.AudienciaConsumo() != efecto.Audiencia || r.EfectoRef() != efecto.Referencia ||
		r.EfectoHuellaSHA256() != contextoSHA || r.ContextoRef() != evidencia.ResultadoContexto.RegistroContextoRef ||
		r.ContextoHuellaSHA256() != evidencia.ResultadoContexto.HuellaSHA256 ||
		!bytes.Equal(material.ContextoActorCanonico(), evidencia.ResultadoContexto.RepresentacionCanonica) ||
		material.PersonaVersion() != actor.Instantanea.PersonaVersion ||
		material.PerfilVersion() != actor.Instantanea.PerfilVersion ||
		ahora.Before(r.EmitidaEn()) || !ahora.Before(r.ExpiraEn()) || ctx.Err() != nil {
		return fallo
	}
	args := append(append(make([]any, 0, len(iniciales)+10), iniciales...), material.CapacidadCanonica(), material.DecisionCanonica(),
		material.MotivoCanonico(), material.ContextoActorCanonico(),
		strconv.FormatUint(material.PersonaVersion(), 10), strconv.FormatUint(material.PerfilVersion(), 10),
		material.PayloadVECAD3(), material.SobreCOSESign1(), material.EvidenciaVerificacion(), material.RaizPublicaSPKI())
	defer func() {
		for _, arg := range args {
			if b, ok := arg.([]byte); ok {
				clear(b)
			}
		}
	}()
	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || ausente(tx) {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()
	var bruto []byte
	if err := tx.QueryRow(ctx, sql, args...).Scan(&bruto); err != nil {
		return traducirErrorLoteSQL(ctx, err)
	}
	if err := validar(bruto); err != nil {
		return err
	}
	if ctx.Err() != nil {
		return fallo
	}
	// Un COMMIT indeterminado jamás entrega un recibo provisional. La recuperación
	// se hace con otra autorización sobre el mismo canon de solicitud.
	if err := tx.Commit(ctx); err != nil {
		return errCommitLoteIndeterminado
	}
	return nil
}

// traducirErrorLoteSQL clasifica los SQLSTATE de AUT44/AD190/CA35 en las
// categorías del puerto, sin conservar el mensaje de PostgreSQL:
// 42501 denegado; 40001, 40P01, 55P03, 55000 y P0002 conflicto de estado
// (preimagen, contexto o cerrojo; se puede reintentar con otra preparación) y
// 23505 conflicto (idempotencia o perfil ya asignado): 409, auditado como error
// y no como denegación; 22023, 22P02, 22007 y 23514 solicitud inválida; el
// resto, no disponible.
func traducirErrorLoteSQL(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42501":
			return domain.ErrAutorizacionDenegada
		case "40001", "40P01", "55P03", "55000", "P0002", "23505":
			return api.ErrConflictoEstado
		case "22023", "22P02", "22007", "23514":
			return domain.ErrActoAdministracionPerfilesInvalido
		}
	}
	return ports.ErrAutoridadAdministracionPerfilesNoDisponible
}
