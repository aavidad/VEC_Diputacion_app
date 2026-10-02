package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	consultaResolverEmpleadoJustificacion = `SELECT vec_cronos_v1.resolver_empleado_justificacion_v1($1)`
	consultaPrepararJustificacion         = `SELECT vec_cronos_v1.consultar_justificacion_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	consultaReciboJustificacion           = `SELECT vec_cronos_v1.recuperar_recibo_justificacion_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	consultaAnexarJustificacion           = `SELECT vec_cronos_v1.anexar_justificacion_v1($1,$2,$3::jsonb,$4,$5,$6,$7,$8::numeric,$9::numeric,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19::numeric,$20::numeric,$21,$22,$23,$24)`
	consultaRevisarJustificacion          = `SELECT vec_cronos_v1.revisar_justificacion_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
)

// La consulta técnica devuelve sólo el empleado de una solicitud. La lectura
// de su justificación requiere después una V3 nueva sobre ese empleado exacto.
type fuenteJustificacionSQL interface {
	iniciadorMarcaje
	QueryRow(context.Context, string, ...any) pgx.Row
}

type FuenteJustificacion struct {
	db       fuenteJustificacionSQL
	lecturas ports.ProveedorLecturaJustificacion
}

func NuevaFuenteJustificacion(pool *pgxpool.Pool, lecturas ports.ProveedorLecturaJustificacion) (*FuenteJustificacion, error) {
	if pool == nil || dependenciaPostgresNula(lecturas) {
		return nil, ports.ErrJustificacionNoDisponible
	}
	return &FuenteJustificacion{db: pool, lecturas: lecturas}, nil
}

func (f *FuenteJustificacion) PrepararJustificacion(ctx context.Context, orden ports.OrdenJustificacion, ref string) (ports.PreparacionJustificacion, error) {
	if f == nil || f.db == nil || dependenciaPostgresNula(f.lecturas) || ctx == nil || ctx.Err() != nil || !domain.SolicitudPermisoRefValida(ref) {
		return ports.PreparacionJustificacion{}, ports.ErrJustificacionNoDisponible
	}
	actor, err := orden.ContextoActor()
	if err != nil {
		return ports.PreparacionJustificacion{}, ports.ErrJustificacionNoDisponible
	}
	var empleado string
	if err := f.db.QueryRow(ctx, consultaResolverEmpleadoJustificacion, ref).Scan(&empleado); err != nil {
		return ports.PreparacionJustificacion{}, errorSeguro(ctx, err)
	}
	m := domain.MaterialConsultaJustificacion{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, SolicitudRef: ref}
	b, err := m.Canonico()
	if err != nil {
		return ports.PreparacionJustificacion{}, ports.ErrJustificacionNoDisponible
	}
	recurso, err := application.RecursoConsultaJustificacion(m)
	if err != nil {
		return ports.PreparacionJustificacion{}, ports.ErrJustificacionNoDisponible
	}
	v3, err := f.lecturas.ProveerMaterialConsultaJustificacion(ctx, m)
	if err != nil {
		return ports.PreparacionJustificacion{}, errorProveedorV3(ctx, err)
	}
	if !resumenV3Ligado(v3, application.AudienciaConsultaJustificacion, application.AccionConsultaJustificacion, recurso) {
		return ports.PreparacionJustificacion{}, ports.ErrJustificacionNoDisponible
	}
	bruto, err := ejecutarFuncionV3(ctx, f.db, consultaPrepararJustificacion, b, v3, errorSeguro)
	if err != nil {
		return ports.PreparacionJustificacion{}, err
	}
	defer clear(bruto)
	var p ports.PreparacionJustificacion
	if decodificarEstricto(bruto, &p) != nil || p.PoliticaSintetica == nil || p.Solicitud.SolicitudRef != ref || p.Solicitud.EmpleadoRef != empleado || p.Solicitud.Validar(p.Politica) != nil ||
		(p.Actual != nil && p.Actual.Validar(p.Solicitud, p.Politica) != nil) {
		return ports.PreparacionJustificacion{}, ports.ErrJustificacionNoDisponible
	}
	return p, nil
}

type RepositorioJustificacion struct {
	db             iniciadorMarcaje
	lecturas       ports.ProveedorLecturaJustificacion
	docConfirmador docports.ProveedorConfirmacionAltaExternaEnlace
}

func NuevoRepositorioJustificacion(pool *pgxpool.Pool, lecturas ports.ProveedorLecturaJustificacion, docConfirmador docports.ProveedorConfirmacionAltaExternaEnlace) (*RepositorioJustificacion, error) {
	if pool == nil || dependenciaPostgresNula(lecturas) || dependenciaPostgresNula(docConfirmador) {
		return nil, ports.ErrJustificacionNoDisponible
	}
	return &RepositorioJustificacion{db: pool, lecturas: lecturas, docConfirmador: docConfirmador}, nil
}

func errorJustificacion(ctx context.Context, err error) error {
	var pg *pgconn.PgError
	if (ctx == nil || ctx.Err() == nil) && errors.As(err, &pg) {
		switch pg.Code {
		case "PC001":
			return domain.ErrJustificacionInvalida
		case "PC002":
			return domain.ErrJustificacionConflicto
		case "PC015":
			return ports.ErrPoliticaJustificacionNoVigente
		}
	}
	return errorSeguro(ctx, err)
}

func errorProveedorConfirmacionDocumental(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, docports.ErrAccesoDenegado) {
		return vecdomain.ErrPermissionDenied
	}
	return errorProveedorV3(ctx, err)
}

func (r *RepositorioJustificacion) RecuperarJustificacion(ctx context.Context, orden ports.OrdenJustificacion, m domain.MaterialJustificacion) (ports.ReciboJustificacion, bool, error) {
	if r == nil || r.db == nil || dependenciaPostgresNula(r.lecturas) || ctx == nil || ctx.Err() != nil {
		return ports.ReciboJustificacion{}, false, ports.ErrJustificacionNoDisponible
	}
	actor, err := orden.ContextoActor()
	h, errHuella := m.Huella()
	if err != nil || errHuella != nil || actor.PersonaRef != m.ActorRef || actor.PerfilActivoRef != m.PerfilRef {
		return ports.ReciboJustificacion{}, false, ports.ErrJustificacionNoDisponible
	}
	lectura := domain.MaterialReciboJustificacion{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef,
		EmpleadoRef: m.Vinculo.EmpleadoRef, SolicitudRef: m.Vinculo.SolicitudRef, ClaveOperacion: m.ClaveOperacion, HuellaMaterial: h}
	b, err := lectura.Canonico()
	if err != nil {
		return ports.ReciboJustificacion{}, false, ports.ErrJustificacionNoDisponible
	}
	recurso, err := application.RecursoReciboJustificacion(lectura)
	if err != nil {
		return ports.ReciboJustificacion{}, false, ports.ErrJustificacionNoDisponible
	}
	v3, err := r.lecturas.ProveerMaterialReciboJustificacion(ctx, lectura)
	if err != nil {
		return ports.ReciboJustificacion{}, false, errorProveedorV3(ctx, err)
	}
	if !resumenV3Ligado(v3, application.AudienciaReciboJustificacion, application.AccionReciboJustificacion, recurso) {
		return ports.ReciboJustificacion{}, false, ports.ErrJustificacionNoDisponible
	}
	bruto, err := ejecutarFuncionV3(ctx, r.db, consultaReciboJustificacion, b, v3, errorJustificacion)
	if err != nil {
		return ports.ReciboJustificacion{}, false, err
	}
	defer clear(bruto)
	var salida struct {
		Encontrado *bool                      `json:"encontrado"`
		Recibo     *ports.ReciboJustificacion `json:"recibo"`
	}
	if decodificarEstricto(bruto, &salida) != nil || salida.Encontrado == nil || (*salida.Encontrado && salida.Recibo == nil) || (!*salida.Encontrado && salida.Recibo != nil) {
		return ports.ReciboJustificacion{}, false, ports.ErrJustificacionNoDisponible
	}
	if !*salida.Encontrado {
		return ports.ReciboJustificacion{}, false, nil
	}
	normalizarFechasJustificacion(salida.Recibo)
	if !reciboJustificacionSQLCoherente(*salida.Recibo, m) {
		return ports.ReciboJustificacion{}, false, ports.ErrJustificacionNoDisponible
	}
	salida.Recibo.Replay = true
	return *salida.Recibo, true, nil
}

func normalizarFechasJustificacion(r *ports.ReciboJustificacion) {
	if r == nil {
		return
	}
	r.FechaUTC = r.FechaUTC.UTC()
	if r.Registro != nil {
		r.Registro.CreadoEnUTC = r.Registro.CreadoEnUTC.UTC()
		r.Registro.ConservacionHastaUTC = r.Registro.ConservacionHastaUTC.UTC()
	}
}

func reciboJustificacionSQLCoherente(r ports.ReciboJustificacion, m domain.MaterialJustificacion) bool {
	h, err := m.Huella()
	return err == nil && r.HuellaMaterial == h && r.Justificacion.Vinculo == m.Vinculo &&
		r.Justificacion.Version == m.VersionEsperada+1 && r.Registro != nil &&
		r.Registro.Documento == m.Vinculo.Documento && r.ReciboRef != "" && !r.FechaUTC.IsZero()
}

func (r *RepositorioJustificacion) ConfirmarJustificacion(ctx context.Context, m domain.MaterialJustificacion, j domain.Justificacion, registro *ports.RegistroDocumentalConfirmado, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboJustificacion, error) {
	if r == nil || r.db == nil || ctx == nil || ctx.Err() != nil || j.Vinculo != m.Vinculo || j.Version != m.VersionEsperada+1 {
		return ports.ReciboJustificacion{}, ports.ErrJustificacionNoDisponible
	}
	b, err := m.Canonico()
	if err != nil {
		return ports.ReciboJustificacion{}, domain.ErrJustificacionInvalida
	}
	recurso, err := application.RecursoJustificacion(m)
	if err != nil || !resumenV3Ligado(v3, application.AudienciaJustificacion, m.Accion, recurso) {
		return ports.ReciboJustificacion{}, ports.ErrJustificacionNoDisponible
	}
	var bruto []byte
	switch m.Accion {
	case domain.AccionAnexarJustificacion:
		if registro == nil || dependenciaPostgresNula(r.docConfirmador) || registro.Documento != m.Vinculo.Documento || registro.ModuloID != "cronos" || registro.ExpedienteRef != m.Vinculo.ExpedienteDocumentalRef {
			return ports.ReciboJustificacion{}, ports.ErrJustificacionNoDisponible
		}
		registrado, e := json.Marshal(registro)
		if e != nil {
			return ports.ReciboJustificacion{}, ports.ErrJustificacionNoDisponible
		}
		huella, e := m.Huella()
		if e != nil {
			return ports.ReciboJustificacion{}, ports.ErrJustificacionNoDisponible
		}
		solicitudDoc := docports.SolicitudConfirmacionAltaExternaEnlace{
			DocumentoID: registro.Documento.ID, ExpedienteRef: registro.ExpedienteRef, TipoRef: registro.TipoRef,
			Version: registro.Documento.Version, ContenidoSHA256: registro.Documento.SHA256,
			CustodioID: registro.Documento.CustodioID, CustodiaRef: registro.Documento.CustodiaRef,
			ClaveAlta: m.ClaveOperacion, ActorAltaRef: m.ActorRef, SolicitudRef: m.Vinculo.SolicitudRef,
			MaterialEnlaceSHA256: huella,
		}
		preimagenDoc, e := solicitudDoc.Preimagen()
		if e != nil {
			return ports.ReciboJustificacion{}, ports.ErrJustificacionNoDisponible
		}
		defer clear(preimagenDoc)
		autorizacionDoc, e := r.docConfirmador.AutorizarConfirmacionAltaExternaEnlace(ctx, solicitudDoc)
		if e != nil {
			return ports.ReciboJustificacion{}, errorProveedorConfirmacionDocumental(ctx, e)
		}
		if autorizacionDoc.PrincipalID != m.ActorRef || autorizacionDoc.Validar(solicitudDoc, time.Now().UTC()) != nil {
			return ports.ReciboJustificacion{}, ports.ErrJustificacionNoDisponible
		}
		authDoc, e := autorizacionDoc.AuthJSON(solicitudDoc)
		if e != nil {
			return ports.ReciboJustificacion{}, ports.ErrJustificacionNoDisponible
		}
		defer clear(authDoc)
		bruto, err = ejecutarAnexoJustificacionV3(ctx, r.db, registrado, preimagenDoc, authDoc, autorizacionDoc.Material, b, v3)
	case domain.AccionRevisarJustificacion:
		if registro != nil {
			return ports.ReciboJustificacion{}, ports.ErrJustificacionNoDisponible
		}
		bruto, err = ejecutarFuncionV3(ctx, r.db, consultaRevisarJustificacion, b, v3, errorJustificacion)
	default:
		return ports.ReciboJustificacion{}, domain.ErrJustificacionInvalida
	}
	if err != nil {
		return ports.ReciboJustificacion{}, err
	}
	defer clear(bruto)
	var recibo ports.ReciboJustificacion
	if decodificarEstricto(bruto, &recibo) != nil {
		return ports.ReciboJustificacion{}, ports.ErrJustificacionNoDisponible
	}
	normalizarFechasJustificacion(&recibo)
	if !reciboJustificacionSQLCoherente(recibo, m) ||
		(registro != nil && !mismoRegistroSQL(*recibo.Registro, *registro)) {
		return ports.ReciboJustificacion{}, ports.ErrJustificacionNoDisponible
	}
	return recibo, nil
}

func mismoRegistroSQL(a, b ports.RegistroDocumentalConfirmado) bool {
	return a.Documento == b.Documento && a.ModuloID == b.ModuloID && a.ExpedienteRef == b.ExpedienteRef && a.TipoRef == b.TipoRef &&
		a.NumeroVEC == b.NumeroVEC && a.CreadoEnUTC.Equal(b.CreadoEnUTC) && a.PoliticaRef == b.PoliticaRef &&
		a.PoliticaVersion == b.PoliticaVersion && a.PoliticaSHA256 == b.PoliticaSHA256 &&
		a.ConservacionHastaUTC.Equal(b.ConservacionHastaUTC) && a.Proteccion == b.Proteccion && a.EstadoPolitica == b.EstadoPolitica
}

// Anexar transporta una V3 Documentos nueva y la V3 Cronos propia. Ambas se
// consumen por fachadas nominales dentro de la misma transacción.
func ejecutarAnexoJustificacionV3(ctx context.Context, db iniciadorMarcaje, registro, preimagenDoc, authDoc []byte, docV3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, material []byte, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) ([]byte, error) {
	if db == nil || ctx == nil || len(registro) == 0 || len(preimagenDoc) == 0 || len(authDoc) == 0 || len(material) == 0 || docV3.ValidarEstructura() != nil || v3.ValidarEstructura() != nil {
		return nil, ports.ErrJustificacionNoDisponible
	}
	secretosDoc := [][]byte{docV3.CapacidadCanonica(), docV3.DecisionCanonica(), docV3.MotivoCanonico(), docV3.ContextoActorCanonico(), docV3.PayloadVECAD3(), docV3.SobreCOSESign1(), docV3.EvidenciaVerificacion(), docV3.RaizPublicaSPKI()}
	secretos := [][]byte{v3.CapacidadCanonica(), v3.DecisionCanonica(), v3.MotivoCanonico(), v3.ContextoActorCanonico(), v3.PayloadVECAD3(), v3.SobreCOSESign1(), v3.EvidenciaVerificacion(), v3.RaizPublicaSPKI()}
	defer func() {
		for _, b := range secretosDoc {
			clear(b)
		}
		for _, b := range secretos {
			clear(b)
		}
	}()
	tx, err := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return nil, errorSeguro(ctx, err)
	}
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()
	if _, err := tx.Exec(ctx, "SELECT set_config('search_path','pg_catalog',true),set_config('timezone','UTC',true),set_config('row_security','on',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),set_config('idle_in_transaction_session_timeout','10s',true)"); err != nil {
		return nil, errorSeguro(ctx, err)
	}
	var bruto []byte
	if err := tx.QueryRow(ctx, consultaAnexarJustificacion,
		string(registro), preimagenDoc, string(authDoc),
		secretosDoc[0], secretosDoc[1], secretosDoc[2], secretosDoc[3], docV3.PersonaVersion(), docV3.PerfilVersion(), secretosDoc[4], secretosDoc[5], secretosDoc[6], secretosDoc[7],
		string(material), secretos[0], secretos[1], secretos[2], secretos[3], v3.PersonaVersion(), v3.PerfilVersion(), secretos[4], secretos[5], secretos[6], secretos[7]).Scan(&bruto); err != nil {
		return nil, errorJustificacion(ctx, err)
	}
	if err := tx.Commit(ctx); err != nil {
		clear(bruto)
		return nil, ports.ErrJustificacionNoDisponible
	}
	return bruto, nil
}

var _ ports.FuenteJustificacion = (*FuenteJustificacion)(nil)
var _ ports.RepositorioJustificacion = (*RepositorioJustificacion)(nil)
