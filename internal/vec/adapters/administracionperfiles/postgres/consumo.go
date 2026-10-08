package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// ejecutarConsumoADMIN reutiliza la transacción de E para fachadas nominales
// de lectura y escritura. Cada constructor acredita sus propias ACL; compartir
// este mecanismo no habilita actos ni crea un emisor o autoridad alternativos.
func ejecutarConsumoADMIN(ctx context.Context, pool conexion, emisor Emisor, reloj ports.Reloj, actor domain.ContextoActor,
	evidencia domain.EvidenciaSesionAdministracionPerfiles, instantanea domain.InstantaneaAutorizacion,
	e Efecto, consulta string, validar func([]byte) error) error {
	if ctx == nil || ausente(pool) || ausente(emisor) || ausente(reloj) || validar == nil {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if evidencia.ValidarEn(actor, reloj.Ahora()) != nil || !domain.ReferenciaCorrelacionAutorizacionV2Valida(e.CorrelacionAccesoRef) {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	traducirError := traducir
	if consulta == capacidadesLecturaSQL {
		// Conserva la denegación nominal de esta consulta antes de que el
		// traductor de actos la reduzca a indisponibilidad. Nunca expone SQL.
		traducirError = traducirErrorLectura
	}
	// El emisor recibe copia: no puede alterar el material ya ligado al efecto.
	entrega := e
	entrega.Material = append([]byte(nil), e.Material...)
	m, err := emisor.EmitirAdministracionPerfiles(ctx, actor, evidencia, instantanea, entrega)
	clear(entrega.Material)
	if err != nil {
		return traducirError(ctx, err)
	}
	huella := sha256.Sum256(e.Material)
	r := m.ResumenCapacidad()
	ahora := reloj.Ahora()
	if m.ValidarEstructura() != nil || r.AudienciaConsumo() != e.Audiencia || r.Operacion() != e.Accion ||
		r.EfectoRef() != e.Referencia || r.EfectoHuellaSHA256() != hex.EncodeToString(huella[:]) || ahora.Before(r.EmitidaEn()) || !ahora.Before(r.ExpiraEn()) ||
		m.PersonaVersion() != actor.Instantanea.PersonaVersion || m.PerfilVersion() != actor.Instantanea.PerfilVersion {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	args := []any{string(e.Material), m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(),
		strconv.FormatUint(m.PersonaVersion(), 10), strconv.FormatUint(m.PerfilVersion(), 10), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()}
	defer func() {
		for _, arg := range args {
			if b, ok := arg.([]byte); ok {
				clear(b)
			}
		}
	}()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return traducirError(ctx, err)
	}
	if ausente(tx) {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), plazoarranque.Ampliar(2*time.Second))
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()
	var bruto []byte
	if err := tx.QueryRow(ctx, consulta, args...).Scan(&bruto); err != nil {
		return traducirError(ctx, err)
	}
	if err := validar(bruto); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return traducirError(ctx, tx.Commit(ctx))
}
