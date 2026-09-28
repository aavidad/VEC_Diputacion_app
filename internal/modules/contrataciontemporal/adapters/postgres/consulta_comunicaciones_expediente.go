package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionConsultaComunicacionesExpediente      = "contratacion_temporal.llamamiento.comunicaciones.consultar"
	TipoRecursoConsultaComunicacionesExpediente = "expediente_contratacion_temporal"
	AudienciaConsultaComunicacionesExpediente   = "vec_contratacion_temporal.comunicaciones_expediente.consultar.v1"
	maximoRespuestaComunicacionesExpediente     = 32768
)

// La organización se obtiene de la identidad y del expediente en la frontera
// confiable. La consulta HTTP no dispone de un campo para imponerla.
type AmbitoConsultaComunicacionesExpediente struct {
	OrganizacionRef string
	Autorizacion    puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type ProveedorConsultaComunicacionesExpediente interface {
	AutorizarConsultaComunicacionesExpediente(context.Context, ports.ConsultaComunicacionesExpediente) (AmbitoConsultaComunicacionesExpediente, error)
}

type materialConsultaComunicacionesExpediente struct {
	OrganizacionRef string `json:"organizacion_ref"`
	ExpedienteRef   string `json:"expediente_ref"`
	Limite          int    `json:"limite"`
	Cursor          string `json:"cursor"`
}

func RecursoConsultaComunicacionesExpediente(c ports.ConsultaComunicacionesExpediente, organizacion string) (dominiovec.RecursoAutorizable, error) {
	if c.Validar() != nil || !domain.ReferenciaOpacaValida(organizacion) {
		return dominiovec.RecursoAutorizable{}, ports.ErrConsultaComunicacionesExpedienteInvalida
	}
	b, err := json.Marshal(materialConsultaComunicacionesExpediente{organizacion, c.ExpedienteRef, c.Limite, c.Cursor})
	if err != nil {
		return dominiovec.RecursoAutorizable{}, ports.ErrConsultaComunicacionesExpedienteInvalida
	}
	h := sha256.Sum256(b)
	return dominiovec.RecursoAutorizable{
		Referencia: c.ExpedienteRef, ModuloID: "contratacion_temporal",
		Tipo:      TipoRecursoConsultaComunicacionesExpediente,
		Ambitos:   map[string]string{"organizacion_ref": organizacion},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])},
	}, nil
}

type LectorComunicacionesExpedientePostgreSQL struct {
	pool      iniciadorTransacciones
	proveedor ProveedorConsultaComunicacionesExpediente
}

var _ ports.LectorComunicacionesExpediente = (*LectorComunicacionesExpedientePostgreSQL)(nil)

func NuevoLectorComunicacionesExpedientePostgreSQL(pool *pgxpool.Pool, proveedor ProveedorConsultaComunicacionesExpediente) (*LectorComunicacionesExpedientePostgreSQL, error) {
	if dependenciaNula(pool) || dependenciaNula(proveedor) {
		return nil, ports.ErrConsultaComunicacionesExpedienteNoDisponible
	}
	return &LectorComunicacionesExpedientePostgreSQL{pool, proveedor}, nil
}

func (l *LectorComunicacionesExpedientePostgreSQL) ConsultarComunicacionesExpediente(ctx context.Context, c ports.ConsultaComunicacionesExpediente) (ports.PaginaComunicacionesExpediente, error) {
	vacia := ports.PaginaComunicacionesExpediente{}
	if l == nil || ctx == nil || dependenciaNula(l.pool) || dependenciaNula(l.proveedor) {
		return vacia, ports.ErrConsultaComunicacionesExpedienteNoDisponible
	}
	if c.Validar() != nil {
		return vacia, ports.ErrConsultaComunicacionesExpedienteInvalida
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	ambito, err := l.proveedor.AutorizarConsultaComunicacionesExpediente(ctx, c)
	if err != nil {
		return vacia, normalizarErrorConsultaComunicacionesExpediente(ctx, err)
	}
	recurso, err := RecursoConsultaComunicacionesExpediente(c, ambito.OrganizacionRef)
	if err != nil {
		return vacia, ports.ErrConsultaComunicacionesExpedienteDenegada
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	a := ambito.Autorizacion
	r := a.ResumenCapacidad()
	if err != nil || a.ValidarEstructura() != nil ||
		r.Operacion() != AccionConsultaComunicacionesExpediente ||
		r.EfectoRef() != c.ExpedienteRef || r.EfectoHuellaSHA256() != huella ||
		r.AudienciaConsumo() != AudienciaConsultaComunicacionesExpediente {
		return vacia, ports.ErrConsultaComunicacionesExpedienteDenegada
	}
	contenido, err := json.Marshal(materialConsultaComunicacionesExpediente{ambito.OrganizacionRef, c.ExpedienteRef, c.Limite, c.Cursor})
	if err != nil {
		return vacia, ports.ErrConsultaComunicacionesExpedienteInvalida
	}
	tx, err := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacia, normalizarErrorConsultaComunicacionesExpediente(ctx, err)
	}
	defer revertirTransaccion(tx)
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),
		set_config('row_security','on',true),set_config('timezone','UTC',true),
		set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),
		set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		return vacia, normalizarErrorConsultaComunicacionesExpediente(ctx, err)
	}
	secretos := [][]byte{a.CapacidadCanonica(), a.DecisionCanonica(), a.MotivoCanonico(), a.ContextoActorCanonico(),
		a.PayloadVECAD3(), a.SobreCOSESign1(), a.EvidenciaVerificacion(), a.RaizPublicaSPKI()}
	defer func() {
		for _, b := range secretos {
			borrarBytes(b)
		}
	}()
	var salida string
	err = tx.QueryRow(ctx, `SELECT vec_contratacion_temporal.consultar_comunicaciones_expediente_rrhh_v1(
		$1::text,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)::text`,
		string(contenido), secretos[0], secretos[1], secretos[2], secretos[3],
		int64(a.PersonaVersion()), int64(a.PerfilVersion()), secretos[4], secretos[5], secretos[6], secretos[7]).Scan(&salida)
	if err != nil {
		return vacia, normalizarErrorConsultaComunicacionesExpediente(ctx, err)
	}
	var respuesta struct {
		Encontrado *bool `json:"encontrado"`
		ports.PaginaComunicacionesExpediente
	}
	if len(salida) == 0 || len(salida) > maximoRespuestaComunicacionesExpediente ||
		decodificarJSONEstricto([]byte(salida), &respuesta) != nil ||
		respuesta.Encontrado == nil {
		return vacia, ports.ErrResultadoComunicacionesExpedienteNoConfiable
	}
	pagina := respuesta.PaginaComunicacionesExpediente
	if !*respuesta.Encontrado {
		if pagina.ExpedienteRef != c.ExpedienteRef ||
			pagina.Comunicaciones == nil || len(pagina.Comunicaciones) != 0 ||
			pagina.SiguienteCursor != "" {
			return vacia, ports.ErrResultadoComunicacionesExpedienteNoConfiable
		}
		if err = ctx.Err(); err != nil {
			return vacia, err
		}
		// La ausencia es un resultado autorizado. Confirmar su consumo y su
		// auditoría antes de traducirla a 404 en la frontera HTTP.
		if err = tx.Commit(ctx); err != nil {
			return vacia, normalizarErrorConsultaComunicacionesExpediente(ctx, err)
		}
		if err = ctx.Err(); err != nil {
			return vacia, err
		}
		return vacia, ports.ErrConsultaComunicacionesExpedienteNoEncontrado
	}
	for i := range pagina.Comunicaciones {
		pagina.Comunicaciones[i].RegistradaEn = pagina.Comunicaciones[i].RegistradaEn.UTC()
	}
	if pagina.ValidarPara(c) != nil {
		return vacia, ports.ErrResultadoComunicacionesExpedienteNoConfiable
	}
	for _, e := range pagina.Comunicaciones {
		if e.OrganizacionRef != ambito.OrganizacionRef {
			return vacia, ports.ErrResultadoComunicacionesExpedienteNoConfiable
		}
	}
	if err = ctx.Err(); err != nil {
		return vacia, err
	}
	if err = tx.Commit(ctx); err != nil {
		return vacia, normalizarErrorConsultaComunicacionesExpediente(ctx, err)
	}
	if err = ctx.Err(); err != nil {
		return vacia, err
	}
	return pagina, nil
}

func normalizarErrorConsultaComunicacionesExpediente(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, ports.ErrConsultaComunicacionesExpedienteDenegada) ||
		errors.Is(err, ports.ErrConsultaComunicacionesExpedienteNoEncontrado) ||
		errors.Is(err, ports.ErrConsultaComunicacionesExpedienteInvalida) {
		return err
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "P1400":
			return ports.ErrConsultaComunicacionesExpedienteInvalida
		case "P1403", "42501":
			return ports.ErrConsultaComunicacionesExpedienteDenegada
		case "P1404":
			return ports.ErrConsultaComunicacionesExpedienteNoEncontrado
		}
	}
	return ports.ErrConsultaComunicacionesExpedienteNoDisponible
}
