package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const funcionConsultarAsignacion = "vec_contratacion_temporal.consultar_asignacion_v1"

var _ ports.ConsultorAsignacionIdempotente = (*ConsultorAsignacionPostgreSQL)(nil)

type ConsultorAsignacionPostgreSQL struct {
	pool iniciadorTransacciones
}

func NuevoConsultorAsignacionPostgreSQL(pool *pgxpool.Pool) (*ConsultorAsignacionPostgreSQL, error) {
	return nuevoConsultorAsignacionPostgreSQL(pool)
}

func nuevoConsultorAsignacionPostgreSQL(pool iniciadorTransacciones) (*ConsultorAsignacionPostgreSQL, error) {
	if dependenciaNula(pool) {
		return nil, ports.ErrPersistenciaAsignacionNoDisponible
	}
	return &ConsultorAsignacionPostgreSQL{pool: pool}, nil
}

type consultaAsignacionV1 struct {
	Esquema                      string                        `json:"esquema"`
	AmbitoIdempotenciaHMACActivo string                        `json:"ambito_idempotencia_hmac_activo"`
	HuellaPeticionHMACActiva     string                        `json:"huella_peticion_hmac_activa"`
	Operacion                    ports.TipoOperacionAsignacion `json:"operacion"`
	OrganizacionRef              string                        `json:"organizacion_ref"`
	ExpedienteRef                string                        `json:"expediente_ref"`
	VersionExpediente            uint64                        `json:"version_expediente"`
	ActorRef                     string                        `json:"actor_ref"`
	PerfilRef                    string                        `json:"perfil_ref"`
	UnidadRef                    string                        `json:"unidad_ref"`
	ResponsableRef               string                        `json:"responsable_ref"`
}

type terminalAsignacionV1 struct {
	ExpedienteAnterior           domain.Expediente                 `json:"expediente_anterior"`
	Recibo                       ports.ReciboAsignacion            `json:"recibo"`
	Referencias                  ports.ReferenciasEfectoAsignacion `json:"referencias"`
	AmbitoIdempotenciaHMAC       string                            `json:"ambito_idempotencia_hmac"`
	HuellaPeticionHMAC           string                            `json:"huella_peticion_hmac"`
	Operacion                    ports.TipoOperacionAsignacion     `json:"operacion"`
	OrganizacionRef              string                            `json:"organizacion_ref"`
	ActorRef                     string                            `json:"actor_ref"`
	PerfilRef                    string                            `json:"perfil_ref"`
	UnidadRef                    string                            `json:"unidad_ref"`
	ResponsableRef               string                            `json:"responsable_ref"`
	DestinoEvidenciaRef          string                            `json:"destino_evidencia_ref"`
	DestinoEvidenciaHuellaSHA256 string                            `json:"destino_evidencia_huella_sha256"`
	PoliticaRef                  string                            `json:"politica_ref"`
	PoliticaVersion              uint64                            `json:"politica_version"`
	PoliticaHuellaSHA256         string                            `json:"politica_huella_sha256"`
	Finalidad                    domain.ClaveCatalogo              `json:"finalidad"`
}

func (c *ConsultorAsignacionPostgreSQL) ConsultarAsignacion(
	ctx context.Context,
	solicitud ports.SolicitudConsultarAsignacionIdempotente,
) (ports.EstadoCandidatoAsignacionIdempotente, bool, error) {
	if ctx == nil || c == nil || dependenciaNula(c.pool) || solicitud.Validar() != nil {
		return ports.EstadoCandidatoAsignacionIdempotente{}, false,
			ports.ErrPreparacionAsignacionInvalida
	}
	entrada := consultaAsignacionV1{
		Esquema:                      esquemaConsultarAsignacion,
		AmbitoIdempotenciaHMACActivo: solicitud.AmbitoIdempotenciaHMACActivo,
		HuellaPeticionHMACActiva:     solicitud.HuellaPeticionHMACActiva,
		Operacion:                    solicitud.Operacion, OrganizacionRef: solicitud.OrganizacionRef,
		ExpedienteRef: solicitud.ExpedienteRef, VersionExpediente: solicitud.VersionExpediente,
		ActorRef: solicitud.ActorRef, PerfilRef: solicitud.PerfilRef,
		UnidadRef: solicitud.UnidadRef, ResponsableRef: solicitud.ResponsableRef,
	}
	contenido, err := json.Marshal(entrada)
	if err != nil {
		return ports.EstadoCandidatoAsignacionIdempotente{}, false,
			ports.ErrPersistenciaAsignacionNoDisponible
	}
	defer borrarBytes(contenido)
	tx, err := c.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return ports.EstadoCandidatoAsignacionIdempotente{}, false,
			normalizarErrorConsultaAsignacion(ctx, err)
	}
	defer revertirTransaccion(tx)
	var terminalJSON string
	err = tx.QueryRow(ctx, `SELECT terminal_json::text FROM `+
		funcionConsultarAsignacion+`($1::jsonb)`, contenido).Scan(&terminalJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.EstadoCandidatoAsignacionIdempotente{}, false, nil
	}
	if err != nil {
		return ports.EstadoCandidatoAsignacionIdempotente{}, false,
			normalizarErrorConsultaAsignacion(ctx, err)
	}
	terminal, err := decodificarTerminalAsignacionPostgreSQL(terminalJSON)
	if err != nil {
		return ports.EstadoCandidatoAsignacionIdempotente{}, false,
			ports.ErrResultadoAsignacionNoConfiable
	}
	preparacion := ports.PreparacionAsignacion{
		Expediente: terminal.ExpedienteAnterior, Referencias: terminal.Referencias,
		AmbitoIdempotenciaHMAC: terminal.AmbitoIdempotenciaHMAC,
		HuellaPeticionHMAC:     terminal.HuellaPeticionHMAC,
		Operacion:              terminal.Operacion, OrganizacionRef: terminal.OrganizacionRef,
		ActorRef: terminal.ActorRef, PerfilRef: terminal.PerfilRef,
		UnidadRef: terminal.UnidadRef, ResponsableRef: terminal.ResponsableRef,
		Estado:           ports.PreparacionAsignacionConfirmada,
		ReciboConfirmado: &terminal.Recibo,
	}
	estado, err := ports.NuevoEstadoCandidatoAsignacionIdempotente(
		ports.DatosEstadoCandidatoAsignacionIdempotente{
			Consulta: solicitud, Preparacion: preparacion,
			DestinoEvidenciaRef:          terminal.DestinoEvidenciaRef,
			DestinoEvidenciaHuellaSHA256: terminal.DestinoEvidenciaHuellaSHA256,
			PoliticaRef:                  terminal.PoliticaRef, PoliticaVersion: terminal.PoliticaVersion,
			PoliticaHuellaSHA256: terminal.PoliticaHuellaSHA256,
			Finalidad:            terminal.Finalidad,
		},
	)
	if err != nil {
		return ports.EstadoCandidatoAsignacionIdempotente{}, false,
			ports.ErrResultadoAsignacionNoConfiable
	}
	if err := tx.Commit(ctx); err != nil {
		return ports.EstadoCandidatoAsignacionIdempotente{}, false,
			normalizarErrorConsultaAsignacion(ctx, err)
	}
	return estado, true, nil
}

func decodificarTerminalAsignacionPostgreSQL(
	contenido string,
) (terminalAsignacionV1, error) {
	var terminal terminalAsignacionV1
	if decodificarJSONEstricto([]byte(contenido), &terminal) != nil {
		return terminalAsignacionV1{}, ports.ErrResultadoAsignacionNoConfiable
	}
	terminal.Recibo = normalizarReciboAsignacionPostgreSQL(terminal.Recibo)
	return terminal, nil
}

func normalizarErrorConsultaAsignacion(ctx context.Context, _ error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return ports.ErrPersistenciaAsignacionNoDisponible
}
