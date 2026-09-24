package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/dietas/application"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

const preleerCircuitoComisionSQL = `SELECT vec_dietas.preleer_circuito_comision_v1($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`

var _ dietasports.RepositorioPrelecturaCircuito = (*RepositorioBorradorComisionPostgreSQL)(nil)

// Preleer obtiene sólo el contexto mínimo de la comisión mediante una función
// nominal que consume AD3 y comprueba competencia antes de mostrar la unidad o
// la relación. No consulta tablas de Personal ni de Dietas directamente.
func (r *RepositorioBorradorComisionPostgreSQL) Preleer(ctx context.Context, identidad dietasports.IdentidadEfectivaPrelecturaCircuito, solicitud dietasports.SolicitudPrelecturaCircuito) (dietasports.ContextoComisionCircuito, error) {
	var cero dietasports.ContextoComisionCircuito
	if err := r.valido(ctx); err != nil {
		if ctx != nil && ctx.Err() != nil {
			return cero, ctx.Err()
		}
		return cero, dietasports.ErrCircuitoNoDisponible
	}
	if application.ValidarSolicitudPrelecturaCircuito(solicitud) != nil ||
		identidad.UnidadCompetenciaRef != solicitud.UnidadRef ||
		identidad.ContextoRegistrado.Validar() != nil || identidad.Vinculo.ValidarPara(identidad.ContextoRegistrado) != nil ||
		identidad.Autorizacion.Material.ValidarEstructura() != nil ||
		identidad.Autorizacion.Accion != application.AccionPreleerCircuito ||
		identidad.Autorizacion.RecursoRef != solicitud.Referencia ||
		identidad.Autorizacion.Finalidad != application.FinalidadPreleerCircuito {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	efecto, err := application.ConstruirEfectoAutorizacionPrelecturaCircuito(identidad.ContextoRegistrado, identidad.UnidadCompetenciaRef, solicitud)
	if err != nil || efecto.Recurso.Referencia != solicitud.Referencia || len(efecto.Material) == 0 {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	a, err := argumentosAD3(identidad.Autorizacion.Material)
	if err != nil {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		a.limpiar()
		return cero, normalizarErrorCircuito(ctx, err, false)
	}
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()
	var bruto []byte
	err = tx.QueryRow(ctx, preleerCircuitoComisionSQL, string(efecto.Material), a.capacidad, a.decision, a.motivo, a.contexto, a.personaVersion, a.perfilVersion, a.payload, a.sobre, a.evidencia, a.raiz).Scan(&bruto)
	a.limpiar()
	if err != nil {
		return cero, normalizarErrorCircuito(ctx, err, false)
	}
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	if err = tx.Commit(ctx); err != nil {
		return cero, normalizarErrorCircuito(ctx, err, false)
	}
	var x struct {
		Resultado string                               `json:"resultado"`
		Comision  dietasports.ContextoComisionCircuito `json:"comision"`
	}
	if json.Unmarshal(bruto, &x) != nil {
		return cero, dietasports.ErrCircuitoNoDisponible
	}
	if x.Resultado == "no_encontrado" && x.Comision == cero {
		return cero, dietasports.ErrComisionNoEncontrada
	}
	if x.Resultado != "concedido" || x.Comision.Referencia != solicitud.Referencia || x.Comision.UnidadRef != solicitud.UnidadRef ||
		x.Comision.Estado != solicitud.Etapa.EstadoPendiente() || x.Comision.Version == 0 ||
		x.Comision.RelacionRef == "" || x.Comision.AsignacionRef == "" || x.Comision.AsignacionVersion == 0 ||
		x.Comision.GrupoDieta == "" || x.Comision.CentroRef == "" ||
		x.Comision.AdministrativoPersonaRef == "" || x.Comision.ResponsablePersonaRef == "" {
		return cero, dietasports.ErrCircuitoNoDisponible
	}
	return x.Comision, nil
}
