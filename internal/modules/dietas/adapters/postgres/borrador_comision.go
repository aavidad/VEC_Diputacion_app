// Package postgres adapta el repositorio durable de borradores de Dietas.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/dietas/application"
	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

const crearORecuperarBorradorSQL = `SELECT vec_dietas.crear_o_recuperar_borrador_propio_v1($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
const consultarBorradoresSQL = `SELECT vec_dietas.consultar_borradores_propios_v1($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`

var referenciaReciboBorrador = regexp.MustCompile(`^rcd_[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var referenciaCursorComision = regexp.MustCompile(`^dco_[A-Za-z0-9_-]{22,128}$`)

var errResultadoBorradorNoEncontrado = errors.New("resultado borrador no encontrado")

type iniciadorBorradorComision interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// RepositorioBorradorComisionPostgreSQL sólo ejecuta las funciones nominales
// concedidas a vec_dietas_ejecutor. Las funciones contienen la revalidación
// bloqueante de Personal y el consumo AD3 V3 dentro de la misma transacción.
// El ejecutor no tiene ACL sobre la fachada de Personal: no realiza SELECT
// directo contra tablas ni funciones de Personal.
type RepositorioBorradorComisionPostgreSQL struct {
	pool iniciadorBorradorComision
}

var _ dietasports.RepositorioBorradorComision = (*RepositorioBorradorComisionPostgreSQL)(nil)

func NuevoRepositorioBorradorComisionPostgreSQL(pool *pgxpool.Pool) (*RepositorioBorradorComisionPostgreSQL, error) {
	if pool == nil {
		return nil, dietasports.ErrBorradorNoDisponible
	}
	return nuevoRepositorioBorradorComisionPostgreSQL(pool)
}

func nuevoRepositorioBorradorComisionPostgreSQL(pool iniciadorBorradorComision) (*RepositorioBorradorComisionPostgreSQL, error) {
	if pool == nil {
		return nil, dietasports.ErrBorradorNoDisponible
	}
	return &RepositorioBorradorComisionPostgreSQL{pool: pool}, nil
}

func (r *RepositorioBorradorComisionPostgreSQL) CrearORecuperar(ctx context.Context, identidad dietasports.IdentidadEfectivaBorrador, solicitud dietasports.SolicitudCrearBorradorPropio) (resultado dietasports.ResultadoBorradorComision, err error) {
	if err = r.valido(ctx); err != nil {
		return resultado, err
	}
	if err = validarMaterial(identidad); err != nil {
		return resultado, err
	}
	efecto, err := efectoCrearBorrador(identidad, solicitud)
	if err != nil {
		return resultado, err
	}
	if err = r.ejecutar(ctx, identidad, crearORecuperarBorradorSQL, efecto.Material, &resultado); err != nil {
		return resultado, err
	}
	return resultado, nil
}

func (r *RepositorioBorradorComisionPostgreSQL) ObtenerPropio(ctx context.Context, identidad dietasports.IdentidadEfectivaBorrador, referencia string) (dietasports.ResultadoBorradorComision, error) {
	efecto, err := efectoDetalleBorrador(identidad, referencia)
	if err != nil {
		return dietasports.ResultadoBorradorComision{}, err
	}
	var resultado dietasports.ResultadoBorradorComision
	err = r.ejecutar(ctx, identidad, consultarBorradoresSQL, efecto.Material, &resultado)
	return resultado, err
}

func (r *RepositorioBorradorComisionPostgreSQL) ListarPropios(ctx context.Context, identidad dietasports.IdentidadEfectivaBorrador, consulta dietasports.ConsultaBorradoresPropios) (pagina dietasports.PaginaBorradoresPropios, err error) {
	if err = r.valido(ctx); err != nil {
		return pagina, err
	}
	if err = validarMaterial(identidad); err != nil {
		return pagina, err
	}
	efecto, err := efectoListaBorrador(identidad, consulta)
	if err != nil {
		return pagina, err
	}
	var bruto []byte
	if err = r.ejecutarBruto(ctx, identidad, consultarBorradoresSQL, efecto.Material, &bruto); err != nil {
		return pagina, err
	}
	pagina, err = decodificarPagina(bruto)
	if err != nil {
		return dietasports.PaginaBorradoresPropios{}, dietasports.ErrBorradorNoDisponible
	}
	return pagina, nil
}

func (r *RepositorioBorradorComisionPostgreSQL) ejecutar(ctx context.Context, identidad dietasports.IdentidadEfectivaBorrador, funcion string, consulta []byte, destino *dietasports.ResultadoBorradorComision) error {
	if err := r.valido(ctx); err != nil {
		return err
	}
	var bruto []byte
	if err := r.ejecutarBruto(ctx, identidad, funcion, consulta, &bruto); err != nil {
		return err
	}
	resultado, err := decodificarResultado(bruto)
	if err != nil {
		if errors.Is(err, errResultadoBorradorNoEncontrado) && funcion == consultarBorradoresSQL {
			return dietasports.ErrComisionNoEncontrada
		}
		if funcion == crearORecuperarBorradorSQL {
			return dietasports.ErrResultadoBorradorIncierto
		}
		return dietasports.ErrBorradorNoDisponible
	}
	*destino = resultado // sólo se expone el recibo después de Commit exitoso.
	return nil
}

func (r *RepositorioBorradorComisionPostgreSQL) ejecutarBruto(ctx context.Context, identidad dietasports.IdentidadEfectivaBorrador, funcion string, consulta []byte, destino *[]byte) (err error) {
	if err = r.valido(ctx); err != nil {
		return err
	}
	if err = validarMaterial(identidad); err != nil {
		return err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return errorNoDisponible(ctx, err, funcion == crearORecuperarBorradorSQL)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	argumentos, err := argumentosAD3(identidad.Autorizacion.Material)
	if err != nil {
		return dietasports.ErrAccesoBorradorDenegado
	}
	var salida []byte
	err = tx.QueryRow(ctx, funcion, string(consulta), argumentos.capacidad, argumentos.decision, argumentos.motivo, argumentos.contexto, argumentos.personaVersion, argumentos.perfilVersion, argumentos.payload, argumentos.sobre, argumentos.evidencia, argumentos.raiz).Scan(&salida)
	argumentos.limpiar()
	if err != nil {
		return normalizarErrorBorrador(ctx, err, funcion == crearORecuperarBorradorSQL)
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return errorNoDisponible(ctx, err, funcion == crearORecuperarBorradorSQL)
	}
	*destino = append((*destino)[:0], salida...)
	return nil
}

func validarMaterial(identidad dietasports.IdentidadEfectivaBorrador) error {
	if identidad.Autorizacion.Material.ValidarEstructura() != nil {
		return dietasports.ErrAccesoBorradorDenegado
	}
	return nil
}

func efectoCrearBorrador(identidad dietasports.IdentidadEfectivaBorrador, crear dietasports.SolicitudCrearBorradorPropio) (dietasports.EfectoAutorizacionBorrador, error) {
	solicitud, err := application.NuevaSolicitudOperacionCrearBorrador(crear)
	if err != nil {
		return dietasports.EfectoAutorizacionBorrador{}, err
	}
	return construirEfectoBorrador(identidad, solicitud, "dietas.borrador.propio.crear", "dietas:borradores:propios", "crear_borrador_propio")
}
func efectoDetalleBorrador(identidad dietasports.IdentidadEfectivaBorrador, referencia string) (dietasports.EfectoAutorizacionBorrador, error) {
	solicitud, err := application.NuevaSolicitudOperacionObtenerBorrador(referencia, identidad.Relacion.RelacionRef)
	if err != nil {
		return dietasports.EfectoAutorizacionBorrador{}, err
	}
	return construirEfectoBorrador(identidad, solicitud, "dietas.borrador.propio.consultar", referencia, "consultar_borrador_propio")
}
func efectoListaBorrador(identidad dietasports.IdentidadEfectivaBorrador, consulta dietasports.ConsultaBorradoresPropios) (dietasports.EfectoAutorizacionBorrador, error) {
	solicitud, err := application.NuevaSolicitudOperacionListarBorrador(consulta, identidad.Relacion.RelacionRef)
	if err != nil {
		return dietasports.EfectoAutorizacionBorrador{}, err
	}
	return construirEfectoBorrador(identidad, solicitud, "dietas.borrador.propio.consultar", "dietas:borradores:propios", "consultar_borrador_propio")
}
func construirEfectoBorrador(identidad dietasports.IdentidadEfectivaBorrador, solicitud dietasports.SolicitudOperacionBorrador, accion, recurso, finalidad string) (dietasports.EfectoAutorizacionBorrador, error) {
	if err := validarMaterial(identidad); err != nil || identidad.Autorizacion.Accion != accion || identidad.Autorizacion.RecursoRef != recurso || identidad.Autorizacion.Finalidad != finalidad {
		return dietasports.EfectoAutorizacionBorrador{}, dietasports.ErrAccesoBorradorDenegado
	}
	efecto, err := application.ConstruirEfectoAutorizacionBorrador(identidad.ContextoRegistrado, identidad.Relacion, identidad.Autorizacion.Revalidacion, solicitud)
	if err != nil || len(efecto.Material) == 0 || efecto.Recurso.Referencia != recurso {
		return dietasports.EfectoAutorizacionBorrador{}, dietasports.ErrAccesoBorradorDenegado
	}
	return efecto, nil
}

type argumentosConsumo struct {
	capacidad, decision, motivo, contexto, payload, sobre, evidencia, raiz []byte
	personaVersion, perfilVersion                                          int64
}

func argumentosAD3(material interface {
	CapacidadCanonica() []byte
	DecisionCanonica() []byte
	MotivoCanonico() []byte
	ContextoActorCanonico() []byte
	PersonaVersion() uint64
	PerfilVersion() uint64
	PayloadVECAD3() []byte
	SobreCOSESign1() []byte
	EvidenciaVerificacion() []byte
	RaizPublicaSPKI() []byte
}) (argumentosConsumo, error) {
	if material == nil {
		return argumentosConsumo{}, errors.New("material ausente")
	}
	a := argumentosConsumo{material.CapacidadCanonica(), material.DecisionCanonica(), material.MotivoCanonico(), material.ContextoActorCanonico(), material.PayloadVECAD3(), material.SobreCOSESign1(), material.EvidenciaVerificacion(), material.RaizPublicaSPKI(), int64(material.PersonaVersion()), int64(material.PerfilVersion())}
	if a.personaVersion < 1 || a.perfilVersion < 1 {
		a.limpiar()
		return argumentosConsumo{}, errors.New("version fuera de rango")
	}
	return a, nil
}
func (a *argumentosConsumo) limpiar() {
	clear(a.capacidad)
	clear(a.decision)
	clear(a.motivo)
	clear(a.contexto)
	clear(a.payload)
	clear(a.sobre)
	clear(a.evidencia)
	clear(a.raiz)
}

func (r *RepositorioBorradorComisionPostgreSQL) valido(ctx context.Context) error {
	if r == nil || r.pool == nil {
		return dietasports.ErrBorradorNoDisponible
	}
	if ctx == nil {
		return dietasports.ErrBorradorNoDisponible
	}
	return ctx.Err()
}

func errorNoDisponible(ctx context.Context, err error, escritura bool) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	if escritura {
		return dietasports.ErrResultadoBorradorIncierto
	}
	return dietasports.ErrBorradorNoDisponible
}
func normalizarErrorBorrador(ctx context.Context, err error, escritura bool) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if !errors.As(err, &pg) {
		return errorNoDisponible(ctx, err, escritura)
	}
	switch pg.Code {
	case "P7202":
		return dietasports.ErrRelacionAmbigua
	case "P7201":
		return dietasports.ErrRelacionNoDisponible
	case "PD001", "22023":
		return domain.ErrComisionBorradorInvalida
	case "PD002":
		return dietasports.ErrConflictoIdempotencia
	case "PD003", "42501":
		return dietasports.ErrAccesoBorradorDenegado
	case "PD004":
		return dietasports.ErrComisionNoEncontrada
	}
	return errorNoDisponible(ctx, err, escritura)
}

type resultadoJSON struct {
	Resultado string                  `json:"resultado"`
	Comision  domain.ComisionBorrador `json:"comision"`
	Recibo    reciboJSON              `json:"recibo"`
}
type reciboJSON struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	RegistradoEn string `json:"registrado_en"`
	Repeticion   bool   `json:"repeticion"`
}

func decodificarResultado(bruto []byte) (dietasports.ResultadoBorradorComision, error) {
	var x resultadoJSON
	if json.Unmarshal(bruto, &x) != nil {
		return dietasports.ResultadoBorradorComision{}, errors.New("resultado SQL invalido")
	}
	if x.Resultado == "no_encontrado" && x.Comision.Referencia == "" && x.Comision.Estado == "" && x.Recibo.Referencia == "" && x.Recibo.Version == 0 {
		return dietasports.ResultadoBorradorComision{}, errResultadoBorradorNoEncontrado
	}
	if x.Resultado != "" && x.Resultado != "concedido" || x.Comision.Validar() != nil || !referenciaReciboBorrador.MatchString(x.Recibo.Referencia) || x.Recibo.Version != 1 {
		return dietasports.ResultadoBorradorComision{}, errors.New("resultado SQL invalido")
	}
	en, err := time.Parse(time.RFC3339Nano, x.Recibo.RegistradoEn)
	_, desfase := en.Zone()
	if err != nil || desfase != 0 || en.Nanosecond()%1000 != 0 {
		return dietasports.ResultadoBorradorComision{}, errors.New("instante invalido")
	}
	// json.Unmarshal deja nil cuando una respuesta antigua omitió el campo;
	// la frontera siempre responde el array canónico vacío.
	x.Comision.CodigosRuta = append([]string{}, x.Comision.CodigosRuta...)
	return dietasports.ResultadoBorradorComision{Comision: x.Comision, Recibo: dietasports.ReciboBorradorComision{Referencia: x.Recibo.Referencia, Version: x.Recibo.Version, RegistradoEn: en.UTC(), Repeticion: x.Recibo.Repeticion}}, nil
}
func decodificarPagina(bruto []byte) (dietasports.PaginaBorradoresPropios, error) {
	var x struct {
		Items     []json.RawMessage `json:"items"`
		Siguiente string            `json:"siguiente_cursor"`
	}
	if json.Unmarshal(bruto, &x) != nil || (x.Siguiente != "" && !referenciaComision(x.Siguiente)) {
		return dietasports.PaginaBorradoresPropios{}, errors.New("pagina SQL invalida")
	}
	p := dietasports.PaginaBorradoresPropios{Items: make([]dietasports.ResultadoBorradorComision, 0, len(x.Items)), SiguienteCursor: x.Siguiente}
	for _, raw := range x.Items {
		r, e := decodificarResultado(raw)
		if e != nil {
			return dietasports.PaginaBorradoresPropios{}, e
		}
		p.Items = append(p.Items, r)
	}
	return p, nil
}
func referenciaComision(v string) bool {
	return referenciaCursorComision.MatchString(v)
}
