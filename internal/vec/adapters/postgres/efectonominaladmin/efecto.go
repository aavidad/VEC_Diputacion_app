// Package efectonominaladmin ejecuta, desde vec-admin, un efecto nominal de
// otro módulo que consume una decisión V3 de la sesión ADMIN: pide la decisión
// al emisor, llama a la fachada SQL del módulo con el material exacto y los
// diez argumentos V3 en una transacción SERIALIZABLE y, si falla, deja el
// intento común después de cerrar la transacción. Cada efecto aporta su
// Contrato (audiencia, coordenadas, recurso, sentencia y lectura del recibo);
// nada de él procede de la petición.
package efectonominaladmin

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrNoDisponible = errors.New("vec: efecto nominal de administración no disponible")
	ErrConflicto    = errors.New("vec: efecto nominal de administración en conflicto")

	procesoEfecto   = regexp.MustCompile(`^[a-z][a-z0-9._-]{1,63}$`)
	sentenciaEfecto = regexp.MustCompile(`^SELECT [a-z_][a-z0-9_]*\.[a-z_][a-z0-9_]*\(\$1::bytea,\$2::bytea,\$3::bytea,\$4::bytea,\$5::bytea,\$6::numeric,\$7::numeric,\$8::bytea,\$9::bytea,\$10::bytea,\$11::bytea\)::text$`)
)

const maximoMaterial = 1 << 20

// Contrato fija un efecto nominal: lo que el emisor pide al PDP y lo que el
// ejecutor manda a la base. Recurso deriva acción y recurso de los bytes
// exactos del material y de la asignación del administrador. Sentencia recibe
// $1 el material y $2..$11 los argumentos V3 en el orden de la exportación.
// Recibo valida la respuesta de la fachada y devuelve el cuerpo público y la
// auditoría del acceso actual; sólo se llama tras un consumo correcto.
type Contrato struct {
	Audiencia string
	Modulo    string
	Tipo      string
	Finalidad string
	// Campos es la lista ordenada exacta de la concesión.
	Campos   []string
	Acciones []string
	// AccionIntento y PrefijoIntento nombran el intento cuando el material no
	// se puede leer: acción genérica y prefijo de una referencia derivada de
	// la correlación.
	AccionIntento  string
	PrefijoIntento string
	Recurso        func(material []byte, asignacion vd.AsignacionPerfil) (string, vd.RecursoAutorizable, error)
	Sentencia      string
	Recibo         func(bruto []byte, accion string, recurso vd.RecursoAutorizable) (Recibo, error)
}

// Valido comprueba la forma del contrato; no su coherencia con la base.
func (c Contrato) Valido() bool {
	return c.Audiencia != "" && c.Modulo != "" && c.Tipo != "" && c.Finalidad != "" && len(c.Acciones) > 0 &&
		slices.IsSorted(c.Campos) && c.AccionIntento != "" && procesoEfecto.MatchString(c.PrefijoIntento) &&
		c.Recurso != nil && c.Recibo != nil && sentenciaEfecto.MatchString(c.Sentencia)
}

// Emision es lo que el ejecutor recibe del emisor.
type Emision struct {
	Accion   string
	Recurso  vd.RecursoAutorizable
	Material vp.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

// Emisor obtiene la decisión V3 de la sesión ADMIN de la misma petición. Lo
// implementa vec-admin (administracion.EmisorEfectoNominalADMIN).
type Emisor interface {
	Emitir(context.Context, vd.ContextoActor, vd.EvidenciaSesionAdministracionPerfiles,
		vd.InstantaneaAutorizacion, []byte, string) (Emision, error)
}

// Solicitud es la sesión ADMIN resuelta por la frontera y los bytes exactos
// del material. Nada de ella concede acceso por sí solo.
type Solicitud struct {
	Actor          vd.ContextoActor
	Evidencia      vd.EvidenciaSesionAdministracionPerfiles
	Instantanea    vd.InstantaneaAutorizacion
	Material       []byte
	CorrelacionRef string
}

// Recibo es la respuesta validada de la fachada (JSON sin campos internos) y
// la auditoría del consumo de este acceso.
type Recibo struct {
	Cuerpo              json.RawMessage
	ConsumoAuditoriaRef string
}

// ConfiguracionAuditoria procede de configuración privada: proceso y motivos
// del catálogo común para los intentos fallidos.
type ConfiguracionAuditoria struct {
	Proceso        string
	MotivoDenegado vd.ReferenciaEntradaCatalogo
	MotivoError    vd.ReferenciaEntradaCatalogo
	Plazo          time.Duration
}

func (c ConfiguracionAuditoria) valida() bool {
	return procesoEfecto.MatchString(c.Proceso) && c.MotivoDenegado.Validar() == nil &&
		c.MotivoError.Validar() == nil && c.Plazo > 0 && c.Plazo <= 2*time.Second
}

type iniciador interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// Ejecutor aplica un efecto nominal con el pool del LOGIN propio del efecto.
type Ejecutor struct {
	pool        iniciador
	emisor      Emisor
	registrador vp.RegistradorIntentosAuditoria
	auditoria   ConfiguracionAuditoria
	reloj       vp.Reloj
	contrato    Contrato
}

func NuevoEjecutor(pool iniciador, emisor Emisor, registrador vp.RegistradorIntentosAuditoria,
	auditoria ConfiguracionAuditoria, reloj vp.Reloj, contrato Contrato,
) (*Ejecutor, error) {
	if nula(pool) || nula(emisor) || nula(registrador) || nula(reloj) || !auditoria.valida() || !contrato.Valido() {
		return nil, ErrNoDisponible
	}
	return &Ejecutor{pool: pool, emisor: emisor, registrador: registrador, auditoria: auditoria, reloj: reloj, contrato: contrato}, nil
}

func nula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return r.IsNil()
	}
	return false
}

// Aplicar ejecuta el efecto. El mismo material devuelve el recibo original
// que decida la fachada, con un consumo nuevo de este acceso.
func (e *Ejecutor) Aplicar(ctx context.Context, s Solicitud) (Recibo, error) {
	if e == nil || ctx == nil || nula(e.registrador) || !e.auditoria.valida() || !e.contrato.Valido() {
		return Recibo{}, ErrNoDisponible
	}
	recibo, accion, recurso, err := e.aplicar(ctx, s)
	if err == nil {
		return recibo, nil
	}
	if e.registrarFallo(ctx, s, accion, recurso, err) != nil {
		return Recibo{}, ErrNoDisponible
	}
	return Recibo{}, err
}

func (e *Ejecutor) aplicar(ctx context.Context, s Solicitud) (Recibo, string, string, error) {
	var vacio Recibo
	if nula(e.pool) || nula(e.emisor) || nula(e.reloj) || ctx.Err() != nil {
		return vacio, "", "", ErrNoDisponible
	}
	if len(s.Material) == 0 || len(s.Material) > maximoMaterial {
		return vacio, "", "", vd.ErrActoAdministracionPerfilesInvalido
	}
	material := bytes.Clone(s.Material)
	defer clear(material)
	accion, recurso, err := e.contrato.Recurso(bytes.Clone(material), s.Instantanea.AsignacionPerfil)
	if err != nil || !slices.Contains(e.contrato.Acciones, accion) || recurso.Validar() != nil ||
		recurso.ModuloID != e.contrato.Modulo || recurso.Tipo != e.contrato.Tipo {
		return vacio, "", "", vd.ErrActoAdministracionPerfilesInvalido
	}
	emision, err := e.emisor.Emitir(ctx, s.Actor, s.Evidencia, s.Instantanea, bytes.Clone(material), s.CorrelacionRef)
	if err != nil {
		if ctx.Err() != nil {
			return vacio, accion, recurso.Referencia, ctx.Err()
		}
		return vacio, accion, recurso.Referencia, err
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	m := emision.Material
	r := m.ResumenCapacidad()
	ahora := e.reloj.Ahora()
	resultado := s.Evidencia.ResultadoContexto
	if err != nil || emision.Accion != accion || emision.Recurso.Referencia != recurso.Referencia ||
		m.ValidarEstructura() != nil || r.Operacion() != accion || r.AudienciaConsumo() != e.contrato.Audiencia ||
		r.EfectoRef() != recurso.Referencia || r.EfectoHuellaSHA256() != huella ||
		ahora.Before(r.EmitidaEn()) || !ahora.Before(r.ExpiraEn()) ||
		r.ContextoRef() != resultado.RegistroContextoRef || r.ContextoHuellaSHA256() != resultado.HuellaSHA256 ||
		!bytes.Equal(m.ContextoActorCanonico(), resultado.RepresentacionCanonica) ||
		m.PersonaVersion() != s.Actor.Instantanea.PersonaVersion || m.PerfilVersion() != s.Actor.Instantanea.PerfilVersion {
		return vacio, accion, recurso.Referencia, ErrNoDisponible
	}
	args := []any{material, m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(),
		strconv.FormatUint(m.PersonaVersion(), 10), strconv.FormatUint(m.PerfilVersion(), 10),
		m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()}
	defer func() {
		for _, x := range args[1:] {
			if b, ok := x.([]byte); ok {
				clear(b)
			}
		}
	}()
	tx, err := e.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || nula(tx) {
		return vacio, accion, recurso.Referencia, errorSQL(ctx, err)
	}
	confirmado := false
	defer func() {
		if !confirmado {
			c, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancelar()
			_ = tx.Rollback(c)
		}
	}()
	var bruto []byte
	if err := tx.QueryRow(ctx, e.contrato.Sentencia, args...).Scan(&bruto); err != nil {
		return vacio, accion, recurso.Referencia, errorSQL(ctx, err)
	}
	if len(bruto) == 0 || len(bruto) > 64*1024 || !json.Valid(bruto) {
		return vacio, accion, recurso.Referencia, ErrNoDisponible
	}
	recibo, err := e.contrato.Recibo(bytes.Clone(bruto), accion, recurso)
	if err != nil || len(recibo.Cuerpo) == 0 || !json.Valid(recibo.Cuerpo) || !strings.HasPrefix(recibo.ConsumoAuditoriaRef, "aud_v3_") {
		return vacio, accion, recurso.Referencia, ErrNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, accion, recurso.Referencia, err
	}
	if err := tx.Commit(ctx); err != nil {
		// Un COMMIT indeterminado nunca entrega un recibo provisional: el
		// reintento con el mismo material recupera el original.
		return vacio, accion, recurso.Referencia, ErrNoDisponible
	}
	confirmado = true
	return recibo, accion, recurso.Referencia, nil
}

// errorSQL clasifica sin conservar el mensaje de PostgreSQL: 42501 denegado;
// 40001/40P01/55P03/23505 conflicto; clase 22 material inválido; el resto, no
// disponible.
func errorSQL(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42501":
			return vd.ErrAutorizacionDenegada
		case "40001", "40P01", "55P03", "23505":
			return ErrConflicto
		}
		// Clase 22: datos del material que la base no admite (formato,
		// fechas, números): es una petición inválida, no una caída.
		if strings.HasPrefix(pg.Code, "22") {
			return vd.ErrActoAdministracionPerfilesInvalido
		}
	}
	return ErrNoDisponible
}

// registrarFallo deja el intento común (denegado o error) con la sesión ADMIN
// original, después de cerrar la transacción del efecto. Sin material legible
// usa la acción genérica del contrato y una referencia de la correlación.
func (e *Ejecutor) registrarFallo(ctx context.Context, s Solicitud, accion, recursoRef string, fallo error) error {
	if !vd.ReferenciaCorrelacionAutorizacionV2Valida(s.CorrelacionRef) {
		return ErrNoDisponible
	}
	actor, err := s.Actor.Clonar()
	if err != nil {
		return ErrNoDisponible
	}
	resultado, err := s.Evidencia.ResultadoContexto.Clonar()
	if err != nil {
		return ErrNoDisponible
	}
	evidencia := vd.EvidenciaSesionAdministracionPerfiles{ResultadoContexto: resultado, Vinculo: s.Evidencia.Vinculo}
	if evidencia.ValidarPara(actor) != nil {
		return ErrNoDisponible
	}
	vinculo, err := evidencia.Vinculo.Datos()
	if err != nil || vinculo.Superficie != vd.SuperficieAutenticacionAdministracionPrivilegiadaV1 || !vinculo.CuentaPrivilegiada {
		return ErrNoDisponible
	}
	if accion == "" {
		accion = e.contrato.AccionIntento
	}
	if recursoRef == "" {
		h := sha256.Sum256([]byte("vec.admin.efecto-nominal.solicitud.v1\n" + e.contrato.Audiencia + "\n" + s.CorrelacionRef))
		recursoRef = e.contrato.PrefijoIntento + ":" + hex.EncodeToString(h[:16])
	}
	clase, motivo := vd.ResultadoIntentoAuditoriaError, e.auditoria.MotivoError
	if errors.Is(fallo, vd.ErrAutorizacionDenegada) || errors.Is(fallo, vd.ErrActoAdministracionPerfilesInvalido) {
		clase, motivo = vd.ResultadoIntentoAuditoriaDenegado, e.auditoria.MotivoDenegado
	}
	datos := vd.DatosIntentoAuditoria{Accion: accion, ModuloID: e.contrato.Modulo, RecursoRef: recursoRef,
		FinalidadRef: e.contrato.Finalidad, Resultado: clase, Motivo: motivo, Proceso: e.auditoria.Proceso,
		Canal: string(vd.SuperficieAutenticacionAdministracionPrivilegiadaV1), CorrelacionRef: s.CorrelacionRef}
	if datos.Validar() != nil {
		// Las referencias opacas de algunos módulos llevan mayúsculas que la
		// auditoría de intentos no admite: se anota una derivada estable.
		h := sha256.Sum256([]byte("vec.admin.efecto-nominal.recurso.v1\n" + e.contrato.Audiencia + "\n" + datos.RecursoRef))
		datos.RecursoRef = e.contrato.Tipo + ":" + hex.EncodeToString(h[:16])
		if datos.Validar() != nil {
			return ErrNoDisponible
		}
	}
	intento, err := vp.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return ErrNoDisponible
	}
	orden, err := vp.NuevaOrdenIntentoAuditoria(intento, resultado, evidencia.Vinculo, datos)
	if err != nil {
		return ErrNoDisponible
	}
	registroCtx, cancelar := context.WithTimeout(context.WithoutCancel(ctx), e.auditoria.Plazo)
	defer cancelar()
	acuse, err := e.registrador.AppendIntentoAuditoria(registroCtx, orden)
	if err != nil || acuse.ValidarPara(orden) != nil {
		return ErrNoDisponible
	}
	return nil
}
