package plannominal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// AD201: fachada del gobierno del plan con organización y unidad. Sólo la
// ejecuta un LOGIN exclusivo del grupo vec_plan_firma_gobierno_ejecutor (AD200).
const registrarGobiernoPlanFirmaSQL201 = `SELECT vec_autorizacion_atestada_v3.registrar_y_confirmar_gobierno_plan_firma_v2($1::bytea,$2,$3,$4::bytea,$5::bytea,$6::bytea,$7::bytea,$8::numeric,$9::numeric,$10::bytea,$11::bytea,$12::bytea,$13::bytea)::text`

var (
	ErrGobiernoPlanFirmaNoDisponible = errors.New("contratacion temporal: gobierno del plan de firma no disponible")
	ErrGobiernoPlanFirmaConflicto    = errors.New("contratacion temporal: gobierno del plan de firma en conflicto")

	procesoGobiernoPlanFirma = regexp.MustCompile(`^[a-z][a-z0-9._-]{1,63}$`)
)

// EmisionGobiernoPlanFirma es lo que el efecto recibe del emisor: acción y
// recurso derivados del material y de la asignación, y el material V3.
type EmisionGobiernoPlanFirma struct {
	Accion   string
	Recurso  vd.RecursoAutorizable
	Ambito   AmbitoGobiernoPlanFirma
	Material vp.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

// EmisorGobiernoPlanFirma obtiene la decisión V3 de la sesión ADMIN de la
// misma petición. Lo implementa vec-admin (administracion.EmisorGobiernoPlanFirma).
type EmisorGobiernoPlanFirma interface {
	EmitirGobiernoPlanFirma(context.Context, vd.ContextoActor, vd.EvidenciaSesionAdministracionPerfiles,
		vd.InstantaneaAutorizacion, []byte, string) (EmisionGobiernoPlanFirma, error)
}

// SolicitudGobiernoPlanFirma es la sesión ADMIN resuelta por la frontera y los
// bytes exactos del material del kit. Nada de ella concede acceso por sí solo.
type SolicitudGobiernoPlanFirma struct {
	Actor          vd.ContextoActor
	Evidencia      vd.EvidenciaSesionAdministracionPerfiles
	Instantanea    vd.InstantaneaAutorizacion
	Material       []byte
	CorrelacionRef string
}

// ReciboGobiernoPlanFirma reúne el recibo del efecto de Catálogos (CC9; en un
// reintento, el original) y la auditoría del consumo de este acceso (AD201).
type ReciboGobiernoPlanFirma struct {
	Accion              string    `json:"-"`
	CatalogoRef         string    `json:"-"`
	ReciboRef           string    `json:"recibo_ref"`
	Estado              string    `json:"estado"`
	Revision            int64     `json:"revision"`
	HuellaComun         string    `json:"huella_comun"`
	PublicacionSHA256   *string   `json:"publicacion_sha256"`
	ActorRef            string    `json:"actor_ref"`
	ConfirmadoEn        time.Time `json:"confirmado_en"`
	AuditoriaRef        string    `json:"auditoria_ref"`
	OutboxReciboRef     string    `json:"outbox_recibo_ref"`
	ConsumoAuditoriaRef string    `json:"-"`
}

type consumoGobiernoSQL201 struct {
	DecisionRef         string    `json:"decision_ref"`
	EfectoRef           string    `json:"efecto_ref"`
	HuellaEfectoSHA256  string    `json:"huella_efecto_sha256"`
	ConsumoHuellaSHA256 string    `json:"consumo_huella_sha256"`
	AuditoriaRef        string    `json:"auditoria_ref"`
	ConsumidaEn         time.Time `json:"consumida_en"`
	ConsumoNuevo        bool      `json:"consumo_nuevo"`
}

// ConfiguracionAuditoriaGobiernoPlanFirma procede de configuración privada:
// proceso, canal y motivos del catálogo común para los intentos fallidos.
type ConfiguracionAuditoriaGobiernoPlanFirma struct {
	Proceso        string
	MotivoDenegado vd.ReferenciaEntradaCatalogo
	MotivoError    vd.ReferenciaEntradaCatalogo
	Plazo          time.Duration
}

func (c ConfiguracionAuditoriaGobiernoPlanFirma) valida() bool {
	return procesoGobiernoPlanFirma.MatchString(c.Proceso) && c.MotivoDenegado.Validar() == nil &&
		c.MotivoError.Validar() == nil && c.Plazo > 0 && c.Plazo <= 2*time.Second
}

type iniciadorGobiernoPlanFirma interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// AutoridadGobiernoPlanFirmaPostgreSQL ejecuta el gobierno del plan nominal de
// firma con el LOGIN del grupo dedicado: pide la decisión al emisor, la
// consume con AD201 (que confirma con CC9) en una transacción SERIALIZABLE y,
// si falla, deja el intento común después de cerrar la transacción.
type AutoridadGobiernoPlanFirmaPostgreSQL struct {
	pool        iniciadorGobiernoPlanFirma
	emisor      EmisorGobiernoPlanFirma
	registrador vp.RegistradorIntentosAuditoria
	auditoria   ConfiguracionAuditoriaGobiernoPlanFirma
	reloj       vp.Reloj
}

func NuevaAutoridadGobiernoPlanFirmaPostgreSQL(pool iniciadorGobiernoPlanFirma, emisor EmisorGobiernoPlanFirma,
	registrador vp.RegistradorIntentosAuditoria, auditoria ConfiguracionAuditoriaGobiernoPlanFirma, reloj vp.Reloj,
) (*AutoridadGobiernoPlanFirmaPostgreSQL, error) {
	if nulaGobierno(pool) || nulaGobierno(emisor) || nulaGobierno(registrador) || nulaGobierno(reloj) || !auditoria.valida() {
		return nil, ErrGobiernoPlanFirmaNoDisponible
	}
	return &AutoridadGobiernoPlanFirmaPostgreSQL{pool: pool, emisor: emisor, registrador: registrador, auditoria: auditoria, reloj: reloj}, nil
}

func nulaGobierno(v any) bool {
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

// GobernarPlanFirma aplica una operación del kit (crear, actualizar, publicar o
// retirar). El mismo material y la misma clave devuelven el recibo original
// con un consumo nuevo de este acceso.
func (a *AutoridadGobiernoPlanFirmaPostgreSQL) GobernarPlanFirma(ctx context.Context, s SolicitudGobiernoPlanFirma) (ReciboGobiernoPlanFirma, error) {
	if a == nil || ctx == nil || nulaGobierno(a.registrador) || !a.auditoria.valida() {
		return ReciboGobiernoPlanFirma{}, ErrGobiernoPlanFirmaNoDisponible
	}
	recibo, accion, recurso, err := a.gobernar(ctx, s)
	if err == nil {
		return recibo, nil
	}
	if a.registrarFallo(ctx, s, accion, recurso, err) != nil {
		return ReciboGobiernoPlanFirma{}, ErrGobiernoPlanFirmaNoDisponible
	}
	return ReciboGobiernoPlanFirma{}, err
}

func (a *AutoridadGobiernoPlanFirmaPostgreSQL) gobernar(ctx context.Context, s SolicitudGobiernoPlanFirma) (ReciboGobiernoPlanFirma, string, string, error) {
	var vacio ReciboGobiernoPlanFirma
	if nulaGobierno(a.pool) || nulaGobierno(a.emisor) || nulaGobierno(a.reloj) || ctx.Err() != nil {
		return vacio, "", "", ErrGobiernoPlanFirmaNoDisponible
	}
	ambito, err := AmbitoGobiernoPlanFirmaDeAsignacion(s.Instantanea.AsignacionPerfil)
	if err != nil {
		return vacio, "", "", vd.ErrActoAdministracionPerfilesInvalido
	}
	accion, recurso, err := RecursoGobiernoPlanFirma(s.Material, ambito)
	if err != nil {
		return vacio, "", "", vd.ErrActoAdministracionPerfilesInvalido
	}
	material := bytes.Clone(s.Material)
	defer clear(material)
	emision, err := a.emisor.EmitirGobiernoPlanFirma(ctx, s.Actor, s.Evidencia, s.Instantanea, bytes.Clone(material), s.CorrelacionRef)
	if err != nil {
		if ctx.Err() != nil {
			return vacio, accion, recurso.Referencia, ctx.Err()
		}
		return vacio, accion, recurso.Referencia, err
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	m := emision.Material
	r := m.ResumenCapacidad()
	ahora := a.reloj.Ahora()
	resultado := s.Evidencia.ResultadoContexto
	if err != nil || emision.Accion != accion || emision.Ambito != ambito || emision.Recurso.Referencia != recurso.Referencia ||
		m.ValidarEstructura() != nil || r.Operacion() != accion || r.AudienciaConsumo() != AudienciaGobiernoPlanFirma ||
		r.EfectoRef() != recurso.Referencia || r.EfectoHuellaSHA256() != huella ||
		ahora.Before(r.EmitidaEn()) || !ahora.Before(r.ExpiraEn()) ||
		r.ContextoRef() != resultado.RegistroContextoRef || r.ContextoHuellaSHA256() != resultado.HuellaSHA256 ||
		!bytes.Equal(m.ContextoActorCanonico(), resultado.RepresentacionCanonica) ||
		m.PersonaVersion() != s.Actor.Instantanea.PersonaVersion || m.PerfilVersion() != s.Actor.Instantanea.PerfilVersion {
		return vacio, accion, recurso.Referencia, ErrGobiernoPlanFirmaNoDisponible
	}
	args := []any{material, ambito.OrganizacionRef, ambito.UnidadRef, m.CapacidadCanonica(), m.DecisionCanonica(),
		m.MotivoCanonico(), m.ContextoActorCanonico(), strconv.FormatUint(m.PersonaVersion(), 10),
		strconv.FormatUint(m.PerfilVersion(), 10), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()}
	defer func() {
		for _, x := range args[3:] {
			if b, ok := x.([]byte); ok {
				clear(b)
			}
		}
	}()
	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || nulaGobierno(tx) {
		return vacio, accion, recurso.Referencia, errorGobiernoSQL(ctx, err)
	}
	confirmado := false
	defer func() {
		if !confirmado {
			c, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(2*time.Second))
			defer cancelar()
			_ = tx.Rollback(c)
		}
	}()
	var bruto []byte
	if err := tx.QueryRow(ctx, registrarGobiernoPlanFirmaSQL201, args...).Scan(&bruto); err != nil {
		return vacio, accion, recurso.Referencia, errorGobiernoSQL(ctx, err)
	}
	var salida struct {
		Recibo  ReciboGobiernoPlanFirma `json:"recibo"`
		Consumo consumoGobiernoSQL201   `json:"consumo"`
	}
	if decodificarGobierno(bruto, &salida) != nil {
		return vacio, accion, recurso.Referencia, ErrGobiernoPlanFirmaNoDisponible
	}
	c := salida.Consumo
	if !c.ConsumoNuevo || c.DecisionRef != r.DecisionRef() || c.EfectoRef != recurso.Referencia || c.HuellaEfectoSHA256 != huella ||
		c.AuditoriaRef == "" || !ct.ReferenciaOpacaValida(salida.Recibo.ReciboRef) || salida.Recibo.Estado != estadoPorOperacionGobierno[operacionDeAccion(accion)] {
		return vacio, accion, recurso.Referencia, ErrGobiernoPlanFirmaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, accion, recurso.Referencia, err
	}
	if err := tx.Commit(ctx); err != nil {
		// Un COMMIT indeterminado nunca entrega un recibo provisional: el
		// reintento con el mismo material recupera el original.
		return vacio, accion, recurso.Referencia, ErrGobiernoPlanFirmaNoDisponible
	}
	confirmado = true
	recibo := salida.Recibo
	recibo.Accion, recibo.CatalogoRef, recibo.ConsumoAuditoriaRef = accion, recurso.Referencia, c.AuditoriaRef
	return recibo, accion, recurso.Referencia, nil
}

func operacionDeAccion(accion string) string {
	const prefijo = "vec.catalogos."
	if len(accion) > len(prefijo) && accion[:len(prefijo)] == prefijo {
		return accion[len(prefijo):]
	}
	return ""
}

// errorGobiernoSQL clasifica sin conservar el mensaje de PostgreSQL: 42501
// denegado; 40001/40P01/55P03/23505 y el CAS de CC9 conflicto; 22023/22P02
// material inválido; el resto, no disponible.
func errorGobiernoSQL(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42501":
			return vd.ErrAutorizacionDenegada
		case "40001", "40P01", "55P03", "23505":
			return ErrGobiernoPlanFirmaConflicto
		case "22023", "22P02":
			return vd.ErrActoAdministracionPerfilesInvalido
		}
	}
	return ErrGobiernoPlanFirmaNoDisponible
}

func decodificarGobierno(b []byte, destino any) error {
	if len(b) == 0 || len(b) > 64*1024 || bytes.TrimSpace(b)[0] != '{' {
		return ErrGobiernoPlanFirmaNoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(destino) != nil || d.Decode(new(any)) != io.EOF {
		return ErrGobiernoPlanFirmaNoDisponible
	}
	return nil
}

// registrarFallo deja el intento común (denegado o error) con la sesión ADMIN
// original, después de cerrar la transacción del efecto. Sin catálogo legible
// usa una referencia derivada de la correlación, nunca un dato de la petición.
func (a *AutoridadGobiernoPlanFirmaPostgreSQL) registrarFallo(ctx context.Context, s SolicitudGobiernoPlanFirma, accion, recursoRef string, fallo error) error {
	if !vd.ReferenciaCorrelacionAutorizacionV2Valida(s.CorrelacionRef) {
		return ErrGobiernoPlanFirmaNoDisponible
	}
	actor, err := s.Actor.Clonar()
	if err != nil {
		return ErrGobiernoPlanFirmaNoDisponible
	}
	resultado, err := s.Evidencia.ResultadoContexto.Clonar()
	if err != nil {
		return ErrGobiernoPlanFirmaNoDisponible
	}
	evidencia := vd.EvidenciaSesionAdministracionPerfiles{ResultadoContexto: resultado, Vinculo: s.Evidencia.Vinculo}
	if evidencia.ValidarPara(actor) != nil {
		return ErrGobiernoPlanFirmaNoDisponible
	}
	vinculo, err := evidencia.Vinculo.Datos()
	if err != nil || vinculo.Superficie != vd.SuperficieAutenticacionAdministracionPrivilegiadaV1 || !vinculo.CuentaPrivilegiada {
		return ErrGobiernoPlanFirmaNoDisponible
	}
	// Sin material legible no hay acción ni catálogo de la petición: se anota
	// como acción de gobierno genérica sobre una referencia de la correlación.
	if accion == "" {
		accion = "vec.catalogos.gobernar"
	}
	if recursoRef == "" {
		h := sha256.Sum256([]byte("vec.catalogos.plan-firma.gobierno.solicitud.v1\n" + s.CorrelacionRef))
		recursoRef = "solicitud_gobierno_plan:" + hex.EncodeToString(h[:16])
	}
	clase, motivo := vd.ResultadoIntentoAuditoriaError, a.auditoria.MotivoError
	if errors.Is(fallo, vd.ErrAutorizacionDenegada) || errors.Is(fallo, vd.ErrActoAdministracionPerfilesInvalido) {
		clase, motivo = vd.ResultadoIntentoAuditoriaDenegado, a.auditoria.MotivoDenegado
	}
	datos := vd.DatosIntentoAuditoria{Accion: accion, ModuloID: ModuloGobiernoPlanFirma, RecursoRef: recursoRef,
		FinalidadRef: FinalidadGobiernoPlanFirma, Resultado: clase, Motivo: motivo, Proceso: a.auditoria.Proceso,
		Canal: string(vd.SuperficieAutenticacionAdministracionPrivilegiadaV1), CorrelacionRef: s.CorrelacionRef}
	if datos.Validar() != nil {
		return ErrGobiernoPlanFirmaNoDisponible
	}
	intento, err := vp.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return ErrGobiernoPlanFirmaNoDisponible
	}
	orden, err := vp.NuevaOrdenIntentoAuditoria(intento, resultado, evidencia.Vinculo, datos)
	if err != nil {
		return ErrGobiernoPlanFirmaNoDisponible
	}
	registroCtx, cancelar := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(a.auditoria.Plazo))
	defer cancelar()
	acuse, err := a.registrador.AppendIntentoAuditoria(registroCtx, orden)
	if err != nil || acuse.ValidarPara(orden) != nil {
		return ErrGobiernoPlanFirmaNoDisponible
	}
	return nil
}
