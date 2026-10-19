package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var errRegistroSeleccionOfertaPostgreSQL = errors.New("contratacion temporal: registro de oferta no disponible")

// RegistroSeleccionOfertaPostgreSQL usa las fachadas CT220. Cada consulta
// consume una autorización nueva y toda mutación tiene una sola transacción.
type RegistroSeleccionOfertaPostgreSQL struct{ pool iniciadorTransacciones }

var _ ports.RegistroSeleccionOfertaCT = (*RegistroSeleccionOfertaPostgreSQL)(nil)

func NuevoRegistroSeleccionOfertaPostgreSQL(pool *pgxpool.Pool) (*RegistroSeleccionOfertaPostgreSQL, error) {
	return nuevoRegistroSeleccionOfertaPostgreSQL(pool)
}

func nuevoRegistroSeleccionOfertaPostgreSQL(pool iniciadorTransacciones) (*RegistroSeleccionOfertaPostgreSQL, error) {
	if dependenciaNula(pool) {
		return nil, errRegistroSeleccionOfertaPostgreSQL
	}
	return &RegistroSeleccionOfertaPostgreSQL{pool: pool}, nil
}

type intencionSeleccionOfertaSQL struct {
	OrganizacionRef   string `json:"organizacion_ref"`
	ExpedienteRef     string `json:"expediente_ref"`
	VersionExpediente uint64 `json:"version_expediente"`
	ClaveIdempotencia string `json:"clave_idempotencia"`
	ActorRef          string `json:"actor_ref"`
	PerfilRef         string `json:"perfil_ref"`
	CategoriaRef      string `json:"categoria_ref"`
	CentroRef         string `json:"centro_ref"`
	HuellaIntencion   string `json:"huella_intencion"`
}

func codificarIntencionSeleccionOferta(i ports.IntencionSeleccionOfertaCT) ([]byte, error) {
	if i.Validar() != nil {
		return nil, ports.ErrSeleccionOfertaCTInvalida
	}
	return json.Marshal(intencionSeleccionOfertaSQL{
		OrganizacionRef: i.OrganizacionRef, ExpedienteRef: i.ExpedienteRef,
		VersionExpediente: i.VersionExpediente, ClaveIdempotencia: i.ClaveIdempotencia,
		ActorRef: i.ActorRef, PerfilRef: i.PerfilRef, CategoriaRef: i.CategoriaRef,
		CentroRef: i.CentroRef, HuellaIntencion: i.HuellaIntencion,
	})
}

func validarAutorizacionSeleccionOferta(etapa string, i ports.IntencionSeleccionOfertaCT,
	c *ports.ComandoOfertaBolsaCT, o *ports.OfertaPublicadaSeleccionCT,
	a ports.AutorizacionSeleccionOfertaCT,
) error {
	canon, err := ports.CanonEtapaSeleccionOfertaCT(ports.SolicitudAutorizacionSeleccionOfertaCT{
		Etapa: etapa, Intencion: i, Comando: c, Oferta: o,
	})
	if err != nil || a.ActorRef != i.ActorRef || a.PerfilRef != i.PerfilRef ||
		a.MaterialCanonico != canon || a.Material.ValidarEstructura() != nil {
		return ports.ErrAutorizacionDenegada
	}
	return nil
}

func argumentosV3SeleccionOferta(a ports.AutorizacionSeleccionOfertaCT) []any {
	m := a.Material
	return []any{m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(),
		m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(),
		m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(),
		m.RaizPublicaSPKI()}
}

func (r *RegistroSeleccionOfertaPostgreSQL) ejecutar(ctx context.Context, consulta string, args ...any) ([]byte, error) {
	if r == nil || r.pool == nil || ctx == nil {
		return nil, errRegistroSeleccionOfertaPostgreSQL
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return nil, errorRegistroSeleccionOferta(ctx, err)
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),
		set_config('row_security','on',true),set_config('timezone','UTC',true),
		set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),
		set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		return nil, errorRegistroSeleccionOferta(ctx, err)
	}
	var raw []byte
	if err = tx.QueryRow(ctx, consulta, args...).Scan(&raw); err != nil {
		return nil, errorRegistroSeleccionOferta(ctx, err)
	}
	if len(raw) == 0 || len(raw) > 1<<20 {
		return nil, errRegistroSeleccionOfertaPostgreSQL
	}
	if err = tx.Commit(ctx); err != nil {
		// El resultado del COMMIT incierto se resuelve por Consultar en otro POST.
		return nil, errorRegistroSeleccionOferta(ctx, err)
	}
	return raw, nil
}

type estadoSeleccionOfertaSQL struct {
	Situacion      ports.SituacionSeleccionOfertaCT `json:"situacion"`
	ReservaRef     string                           `json:"reserva_ref"`
	VentanaAbierta bool                             `json:"ventana_abierta"`
	Comando        json.RawMessage                  `json:"comando"`
	Recibo         json.RawMessage                  `json:"recibo"`
	ReciboDescarte json.RawMessage                  `json:"recibo_descarte"`
}

func decodificarEstadoSeleccionOferta(raw []byte, i ports.IntencionSeleccionOfertaCT) (ports.EstadoSeleccionOfertaCT, error) {
	var s estadoSeleccionOfertaSQL
	var estado ports.EstadoSeleccionOfertaCT
	if json.Unmarshal(raw, &s) != nil {
		return estado, errRegistroSeleccionOfertaPostgreSQL
	}
	estado.Situacion, estado.ReservaRef, estado.VentanaAbierta = s.Situacion, s.ReservaRef, s.VentanaAbierta
	switch s.Situacion {
	case ports.SeleccionOfertaAusente, ports.SeleccionOfertaOcupada, ports.SeleccionOfertaColision:
		if s.ReservaRef != "" || s.VentanaAbierta || len(s.Comando) != 0 && string(s.Comando) != "null" ||
			len(s.Recibo) != 0 && string(s.Recibo) != "null" ||
			len(s.ReciboDescarte) != 0 && string(s.ReciboDescarte) != "null" {
			return ports.EstadoSeleccionOfertaCT{}, errRegistroSeleccionOfertaPostgreSQL
		}
	case ports.SeleccionOfertaReservada, ports.SeleccionOfertaIndeterminada,
		ports.SeleccionOfertaConfirmada, ports.SeleccionOfertaDescartada:
		if json.Unmarshal(s.Comando, &estado.Comando) != nil || estado.Comando.ValidarPara(i) != nil {
			return ports.EstadoSeleccionOfertaCT{}, errRegistroSeleccionOfertaPostgreSQL
		}
		if s.Situacion == ports.SeleccionOfertaConfirmada {
			if s.ReservaRef != "" || len(s.ReciboDescarte) != 0 && string(s.ReciboDescarte) != "null" ||
				json.Unmarshal(s.Recibo, &estado.Recibo) != nil ||
				estado.Recibo.ValidarPara(i, estado.Comando) != nil {
				return ports.EstadoSeleccionOfertaCT{}, errRegistroSeleccionOfertaPostgreSQL
			}
		} else if s.Situacion == ports.SeleccionOfertaDescartada {
			if s.ReservaRef != "" || len(s.Recibo) != 0 && string(s.Recibo) != "null" ||
				json.Unmarshal(s.ReciboDescarte, &estado.ReciboDescarte) != nil ||
				estado.ReciboDescarte.ValidarTerminalPara(i, estado.Comando) != nil {
				return ports.EstadoSeleccionOfertaCT{}, errRegistroSeleccionOfertaPostgreSQL
			}
		} else if s.ReservaRef == "" ||
			(s.Situacion == ports.SeleccionOfertaIndeterminada && !s.VentanaAbierta) ||
			len(s.Recibo) != 0 && string(s.Recibo) != "null" ||
			len(s.ReciboDescarte) != 0 && string(s.ReciboDescarte) != "null" {
			return ports.EstadoSeleccionOfertaCT{}, errRegistroSeleccionOfertaPostgreSQL
		}
	default:
		return ports.EstadoSeleccionOfertaCT{}, errRegistroSeleccionOfertaPostgreSQL
	}
	return estado, nil
}

func (r *RegistroSeleccionOfertaPostgreSQL) Consultar(ctx context.Context,
	i ports.IntencionSeleccionOfertaCT, a ports.AutorizacionSeleccionOfertaCT,
) (ports.EstadoSeleccionOfertaCT, error) {
	var vacio ports.EstadoSeleccionOfertaCT
	intencion, err := codificarIntencionSeleccionOferta(i)
	if err != nil || validarAutorizacionSeleccionOferta(ports.EtapaSeleccionOfertaConsultar, i, nil, nil, a) != nil {
		return vacio, ports.ErrAutorizacionDenegada
	}
	args := append([]any{intencion, a.MaterialCanonico}, argumentosV3SeleccionOferta(a)...)
	raw, err := r.ejecutar(ctx, `SELECT vec_contratacion_temporal.consultar_seleccion_oferta_ct220(
		$1::jsonb,$2::text,$3::bytea,$4::bytea,$5::bytea,$6::bytea,$7::numeric,$8::numeric,
		$9::bytea,$10::bytea,$11::bytea,$12::bytea)`, args...)
	if err != nil {
		return vacio, err
	}
	return decodificarEstadoSeleccionOferta(raw, i)
}

func (r *RegistroSeleccionOfertaPostgreSQL) Reservar(ctx context.Context,
	i ports.IntencionSeleccionOfertaCT, c ports.ComandoOfertaBolsaCT,
	a ports.AutorizacionSeleccionOfertaCT,
) (ports.EstadoSeleccionOfertaCT, error) {
	var vacio ports.EstadoSeleccionOfertaCT
	intencion, err := codificarIntencionSeleccionOferta(i)
	if err != nil || c.ValidarPara(i) != nil ||
		validarAutorizacionSeleccionOferta(ports.EtapaSeleccionOfertaReservar, i, &c, nil, a) != nil {
		return vacio, ports.ErrSeleccionOfertaCTInvalida
	}
	comando, err := json.Marshal(c)
	if err != nil {
		return vacio, ports.ErrSeleccionOfertaCTInvalida
	}
	args := append([]any{intencion, comando, a.MaterialCanonico}, argumentosV3SeleccionOferta(a)...)
	raw, err := r.ejecutar(ctx, `SELECT vec_contratacion_temporal.reservar_seleccion_oferta_ct220(
		$1::jsonb,$2::jsonb,$3::text,$4::bytea,$5::bytea,$6::bytea,$7::bytea,
		$8::numeric,$9::numeric,$10::bytea,$11::bytea,$12::bytea,$13::bytea)`, args...)
	if err != nil {
		return vacio, err
	}
	return decodificarEstadoSeleccionOferta(raw, i)
}

func (r *RegistroSeleccionOfertaPostgreSQL) AbrirVentana(ctx context.Context,
	reserva ports.ReservaSeleccionOfertaCT, a ports.AutorizacionSeleccionOfertaCT,
) error {
	i := reserva.Intencion
	intencion, err := codificarIntencionSeleccionOferta(i)
	if err != nil || reserva.ReservaRef == "" || reserva.Comando.ValidarPara(i) != nil ||
		validarAutorizacionSeleccionOferta(ports.EtapaSeleccionOfertaAbrirVentana, i, &reserva.Comando, nil, a) != nil {
		return ports.ErrSeleccionOfertaCTInvalida
	}
	comando, err := json.Marshal(reserva.Comando)
	if err != nil {
		return ports.ErrSeleccionOfertaCTInvalida
	}
	args := append([]any{intencion, reserva.ReservaRef, comando, a.MaterialCanonico}, argumentosV3SeleccionOferta(a)...)
	raw, err := r.ejecutar(ctx, `SELECT to_jsonb(vec_contratacion_temporal.abrir_ventana_seleccion_oferta_ct220(
		$1::jsonb,$2::text,$3::jsonb,$4::text,$5::bytea,$6::bytea,$7::bytea,$8::bytea,
		$9::numeric,$10::numeric,$11::bytea,$12::bytea,$13::bytea,$14::bytea))`, args...)
	if err != nil {
		return err
	}
	if string(raw) != "true" {
		return errRegistroSeleccionOfertaPostgreSQL
	}
	return nil
}

func (r *RegistroSeleccionOfertaPostgreSQL) ConfirmarVinculo(ctx context.Context,
	reserva ports.ReservaSeleccionOfertaCT, oferta ports.OfertaPublicadaSeleccionCT,
	a ports.AutorizacionSeleccionOfertaCT,
) (ports.ReciboSeleccionOfertaCT, error) {
	var vacio ports.ReciboSeleccionOfertaCT
	i := reserva.Intencion
	intencion, err := codificarIntencionSeleccionOferta(i)
	if err != nil || reserva.ReservaRef == "" || reserva.Comando.ValidarPara(i) != nil ||
		oferta.ValidarPara(reserva.Comando) != nil ||
		validarAutorizacionSeleccionOferta(ports.EtapaSeleccionOfertaConfirmar, i, &reserva.Comando, &oferta, a) != nil {
		return vacio, ports.ErrSeleccionOfertaCTInvalida
	}
	comando, err := json.Marshal(reserva.Comando)
	if err != nil {
		return vacio, ports.ErrSeleccionOfertaCTInvalida
	}
	publicada, err := json.Marshal(oferta)
	if err != nil {
		return vacio, ports.ErrSeleccionOfertaCTInvalida
	}
	args := append([]any{intencion, reserva.ReservaRef, comando, publicada, a.MaterialCanonico}, argumentosV3SeleccionOferta(a)...)
	raw, err := r.ejecutar(ctx, `SELECT vec_contratacion_temporal.confirmar_vinculo_seleccion_oferta_ct220(
		$1::jsonb,$2::text,$3::jsonb,$4::jsonb,$5::text,$6::bytea,$7::bytea,$8::bytea,
		$9::bytea,$10::numeric,$11::numeric,$12::bytea,$13::bytea,$14::bytea,$15::bytea)`, args...)
	if err != nil {
		return vacio, err
	}
	return decodificarVinculoOfertaCT(raw, reserva, oferta)
}

func errorRegistroSeleccionOferta(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42501":
			return ports.ErrAutorizacionDenegada
		case "22023":
			return ports.ErrSeleccionOfertaCTInvalida
		case "23505", "40001", "40P01", "55000":
			return ports.ErrVinculoOfertaBolsaCTInvalido
		}
	}
	return errRegistroSeleccionOfertaPostgreSQL
}
