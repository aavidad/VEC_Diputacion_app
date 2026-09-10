package postgres

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

const AudienciaAnotacionAdministrativaV1 = "vec_contratacion_temporal.anotacion_administrativa.v1"
const sqlPrepararAnotacion = `SELECT vec_contratacion_temporal.preparar_anotacion_administrativa_incorporacion_v1($1::jsonb)::text`
const sqlConfirmarAnotacion = `SELECT vec_contratacion_temporal.registrar_anotacion_administrativa_incorporacion_v1($1::jsonb,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)::text`

// ProveedorMaterialAnotacionAdministrativa emite una capacidad breve a partir
// de la concesión V3 actual. No debe recuperar una exportación ya consumida.
type ProveedorMaterialAnotacionAdministrativa interface {
	ProveerMaterialAnotacionAdministrativa(context.Context, ct.OrdenConfirmarAnotacionAdministrativa) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type PreparadorAnotacionAdministrativaPostgreSQL struct{ pool iniciadorTransacciones }
type TransaccionAnotacionesAdministrativasPostgreSQL struct {
	pool      iniciadorTransacciones
	proveedor ProveedorMaterialAnotacionAdministrativa
}

var _ ct.PreparadorAnotacionAdministrativaIdempotente = (*PreparadorAnotacionAdministrativaPostgreSQL)(nil)
var _ ct.TransaccionAnotacionesAdministrativas = (*TransaccionAnotacionesAdministrativasPostgreSQL)(nil)

func NuevoPreparadorAnotacionAdministrativaPostgreSQL(p *pgxpool.Pool) (*PreparadorAnotacionAdministrativaPostgreSQL, error) {
	if dependenciaNula(p) {
		return nil, ct.ErrPersistenciaAnotacionAdministrativaNoDisponible
	}
	return &PreparadorAnotacionAdministrativaPostgreSQL{p}, nil
}
func NuevaTransaccionAnotacionesAdministrativasPostgreSQL(p *pgxpool.Pool, v ProveedorMaterialAnotacionAdministrativa) (*TransaccionAnotacionesAdministrativasPostgreSQL, error) {
	if dependenciaNula(p) || dependenciaNula(v) {
		return nil, ct.ErrPersistenciaAnotacionAdministrativaNoDisponible
	}
	return &TransaccionAnotacionesAdministrativasPostgreSQL{p, v}, nil
}

type materialAnotacionWire struct {
	Organizacion  string `json:"organizacion_ref"`
	Expediente    string `json:"expediente_ref"`
	Solicitud     string `json:"solicitud_personal_ref"`
	Version       uint64 `json:"version_esperada"`
	Actor         string `json:"actor_ref"`
	Perfil        string `json:"perfil_ref"`
	Observaciones string `json:"observaciones"`
}

func materialAnotacion(m ct.MaterialAnotacionAdministrativa) materialAnotacionWire {
	return materialAnotacionWire{m.OrganizacionRef, m.ExpedienteRef, m.SolicitudPersonalRef, m.VersionEsperada, m.ActorRef, m.PerfilRef, m.Observaciones}
}

type preparacionAnotacionWire struct {
	Expediente          json.RawMessage                `json:"expediente"`
	Material            materialAnotacionWire          `json:"material"`
	Seguimiento         dom.VinculoSeguimientoOriginal `json:"seguimiento_original"`
	EstadoSHA           string                         `json:"estado_seguimiento_sha256"`
	ReciboIncorporacion string                         `json:"recibo_incorporacion_ref"`
	Referencias         struct {
		Recibo string `json:"recibo_ref"`
		Evento string `json:"evento_ref"`
	} `json:"referencias"`
	Ambito string                                      `json:"ambito_idempotencia_hmac"`
	Huella string                                      `json:"huella_peticion_hmac"`
	Estado ct.EstadoPreparacionAnotacionAdministrativa `json:"estado"`
	Recibo *ct.ReciboAnotacionAdministrativa           `json:"recibo_confirmado"`
}

func entradaPrepararAnotacion(m ct.MaterialAnotacionAdministrativa, s sellosPrepararAltaV2) ([]byte, error) {
	return json.Marshal(struct {
		Material materialAnotacionWire `json:"material"`
		Sellos   sellosPrepararAltaV2  `json:"sellos_hmac"`
	}{materialAnotacion(m), s})
}
func leerPreparacionAnotacion(ctx context.Context, tx pgx.Tx, b []byte) (preparacionAnotacionWire, error) {
	var w preparacionAnotacionWire
	var raw []byte
	defer func() { clear(raw) }()
	if e := tx.QueryRow(ctx, sqlPrepararAnotacion, b).Scan(&raw); e != nil {
		return w, e
	}
	if len(raw) > 4<<20 || decodificarJSONEstricto(raw, &w) != nil {
		return w, ct.ErrResultadoAnotacionAdministrativaNoConfiable
	}
	return w, nil
}
func (w preparacionAnotacionWire) restaurar(m ct.MaterialAnotacionAdministrativa) (ct.PreparacionAnotacionAdministrativa, error) {
	var p ct.PreparacionAnotacionAdministrativa
	if w.Material != materialAnotacion(m) || decodificarJSONEstricto(w.Expediente, &p.Expediente) != nil {
		return p, ct.ErrResultadoAnotacionAdministrativaNoConfiable
	}
	p.Material = m
	p.SeguimientoOriginal = w.Seguimiento
	p.Estado = w.Estado
	p.ReciboConfirmado = w.Recibo
	p.HuellaEstadoSeguimientoSHA256 = w.EstadoSHA
	p.ReciboIncorporacionOriginalRef = w.ReciboIncorporacion
	p.Referencias.ReciboRef = w.Referencias.Recibo
	p.Referencias.EventoRef = w.Referencias.Evento
	p.AmbitoIdempotenciaHMAC = w.Ambito
	p.HuellaPeticionHMAC = w.Huella
	if validarPreparacionAnotacion(p) != nil {
		return ct.PreparacionAnotacionAdministrativa{}, ct.ErrResultadoAnotacionAdministrativaNoConfiable
	}
	return p, nil
}
func validarPreparacionAnotacion(p ct.PreparacionAnotacionAdministrativa) error {
	m := p.Material
	if m.Validar() != nil || p.Expediente.Validar() != nil || p.SeguimientoOriginal.Validar() != nil ||
		p.Expediente.OrganizacionRef != m.OrganizacionRef || p.Expediente.Referencia != m.ExpedienteRef || p.Expediente.Version != m.VersionEsperada ||
		!ct.SelloHMACSHA256Valido(p.AmbitoIdempotenciaHMAC) || !ct.SelloHMACSHA256Valido(p.HuellaPeticionHMAC) ||
		!dom.ReferenciaOpacaValida(p.Referencias.ReciboRef) || !dom.ReferenciaOpacaValida(p.Referencias.EventoRef) ||
		!refSeguimientoRegistroV2(p.ReciboIncorporacionOriginalRef) || !hashAnotacion(p.HuellaEstadoSeguimientoSHA256) {
		return ct.ErrPreparacionAnotacionAdministrativaInvalida
	}
	switch p.Estado {
	case ct.PreparacionAnotacionAdministrativaPreparada:
		if p.ReciboConfirmado != nil {
			return ct.ErrPreparacionAnotacionAdministrativaInvalida
		}
	case ct.PreparacionAnotacionAdministrativaConfirmada:
		if p.ReciboConfirmado == nil || validarReciboAnotacion(*p.ReciboConfirmado, p) != nil {
			return ct.ErrPreparacionAnotacionAdministrativaInvalida
		}
	default:
		return ct.ErrPreparacionAnotacionAdministrativaInvalida
	}
	return nil
}
func hashAnotacion(s string) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == 32 && s != "0000000000000000000000000000000000000000000000000000000000000000" && hex.EncodeToString(b) == s
}
func (a *PreparadorAnotacionAdministrativaPostgreSQL) PrepararAnotacionAdministrativa(ctx context.Context, s ct.SolicitudPrepararAnotacionAdministrativa) (ct.PreparacionAnotacionAdministrativa, error) {
	var cero ct.PreparacionAnotacionAdministrativa
	if ctx == nil || a == nil || dependenciaNula(a.pool) || s.Validar() != nil {
		return cero, ct.ErrPreparacionAnotacionAdministrativaInvalida
	}
	sellos, e := nuevosSellosPrepararAltaV2(s.AmbitosHMAC, s.HuellasPeticionHMAC)
	if e != nil {
		return cero, ct.ErrPreparacionAnotacionAdministrativaInvalida
	}
	b, e := entradaPrepararAnotacion(s.Material, sellos)
	if e != nil {
		return cero, ct.ErrPreparacionAnotacionAdministrativaInvalida
	}
	defer clear(b)
	tx, e := a.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly})
	if e != nil {
		return cero, errorAnotacion(ctx, e)
	}
	defer revertirTransaccion(tx)
	if _, e = tx.Exec(ctx, ajustesRegistroIncorporacionTXV2); e != nil {
		return cero, errorAnotacion(ctx, e)
	}
	w, e := leerPreparacionAnotacion(ctx, tx, b)
	if e != nil {
		return cero, errorAnotacion(ctx, e)
	}
	p, e := w.restaurar(s.Material)
	if e != nil {
		return cero, e
	}
	if !sellos.contienePar(p.AmbitoIdempotenciaHMAC, p.HuellaPeticionHMAC) {
		return cero, ct.ErrResultadoAnotacionAdministrativaNoConfiable
	}
	if e = tx.Commit(ctx); e != nil {
		return cero, errorAnotacion(ctx, e)
	}
	return p, nil
}

func contextoRecursoAnotacion(o ct.OrdenConfirmarAnotacionAdministrativa) (map[string]string, map[string]string) {
	r := ct.RecursoAutorizacionAnotacionAdministrativa(o.Preparacion, o.Politica)
	return r.Ambitos, r.Atributos
}

func validarOrdenAnotacion(o ct.OrdenConfirmarAnotacionAdministrativa, t time.Time) error {
	p := o.Preparacion
	m := p.Material
	fallo := ct.ErrPreparacionAnotacionAdministrativaInvalida
	if validarPreparacionAnotacion(p) != nil || o.Material != m || o.SeguimientoOriginal != p.SeguimientoOriginal || o.Referencias != p.Referencias ||
		!dom.InstanteUTCCanonico(t) || !dom.InstanteUTCCanonico(o.InstanteEfecto) || t.Before(o.InstanteEfecto) ||
		!dom.InstanteUTCCanonico(o.Politica.EvaluadaEn) || !dom.InstanteUTCCanonico(o.Politica.ValidaHasta) ||
		t.Before(o.Politica.EvaluadaEn) || !t.Before(o.Politica.ValidaHasta) || !dom.ReferenciaOpacaValida(o.Politica.DefinicionRef) ||
		o.Politica.DefinicionVersion == 0 || !hashAnotacion(o.Politica.DefinicionHuellaSHA256) || o.Politica.MotivoAutorizacion.Validar() != nil {
		return fallo
	}
	s, e := o.Evidencia.SolicitudV3.Datos()
	v, ev := s.VinculoAutenticacionActor.Datos()
	vc, ec := o.Evidencia.Contexto.Vinculo.Datos()
	c, ce := o.Evidencia.ConfirmacionV3.Datos()
	h, he := vd.HuellaSHA256DecisionAutorizacionV3(o.Evidencia.DecisionV3)
	concedida, _, de := o.Evidencia.DecisionV3.Resultado()
	amb, atr := contextoRecursoAnotacion(o)
	if e != nil || ev != nil || ec != nil || ce != nil || he != nil || de != nil || !concedida ||
		o.Evidencia.Contexto.Resultado.Validar() != nil || o.Evidencia.Contexto.Vinculo.ValidarPara(o.Evidencia.Contexto.Resultado) != nil || !o.Evidencia.Contexto.Vinculo.VigenteEn(t, o.Evidencia.Contexto.Resultado) ||
		o.Evidencia.DecisionV3.ValidarPara(o.Evidencia.SolicitudV3) != nil || !reflect.DeepEqual(v, vc) || v.PrincipalID != m.ActorRef || v.PerfilActivoRef != m.PerfilRef ||
		s.Accion != string(dom.AccionRegistrarAnotacionAdministrativa) || s.Finalidad != ct.FinalidadRegistrarAnotacionAdministrativa || s.ReferenciaMotivo != o.Politica.MotivoAutorizacion ||
		s.Recurso.Referencia != m.ExpedienteRef || s.Recurso.ModuloID != ct.ModuloContratacion || s.Recurso.Tipo != ct.TipoRecursoAnotacionAdministrativa ||
		!reflect.DeepEqual(s.Recurso.Ambitos, amb) || !reflect.DeepEqual(s.Recurso.Atributos, atr) || c.DecisionHuellaSHA256 != h || !o.Evidencia.ConfirmacionV3.DentroDeVentanaEn(t) {
		return fallo
	}
	// Incluso el replay se valida antes de abrir la transacción; SQL consumirá
	// concesión nueva y devolverá exclusivamente el recibo histórico.
	if p.Estado == ct.PreparacionAnotacionAdministrativaConfirmada {
		return nil
	}
	if p.Expediente.Asignacion == nil || o.InstanteEfecto.Before(p.Expediente.ActualizadoEn) {
		return fallo
	}
	esperado, e := p.Expediente.RegistrarAnotacionAdministrativa(m.VersionEsperada, p.SeguimientoOriginal, dom.DatosActuacion{
		AccionClave: dom.AccionRegistrarAnotacionAdministrativa, ActorRef: m.ActorRef, UnidadRef: p.Expediente.Asignacion.UnidadRef, ReciboRef: p.Referencias.ReciboRef, RealizadaEn: o.InstanteEfecto,
		FaseDestino: p.Expediente.FaseActual, EstadoDestino: p.Expediente.EstadoActual, Observaciones: m.Observaciones})
	if e != nil || !reflect.DeepEqual(esperado, o.ExpedienteSiguiente) {
		return fallo
	}
	return nil
}

// Conserva los bytes JSON de toda la historia anterior: encoding/json de
// time.Time normaliza .000000Z a Z. La postimagen se valida en dominio antes
// de transportar solamente los tres campos que cambian.
func postimagenAnotacion(original json.RawMessage, o ct.OrdenConfirmarAnotacionAdministrativa) (json.RawMessage, error) {
	var originalDom dom.Expediente
	if decodificarJSONEstricto(original, &originalDom) != nil || !reflect.DeepEqual(originalDom, o.Preparacion.Expediente) {
		return nil, ct.ErrResultadoAnotacionAdministrativaNoConfiable
	}
	if o.Preparacion.Estado == ct.PreparacionAnotacionAdministrativaConfirmada {
		return original, nil
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal(original, &raw) != nil {
		return nil, ct.ErrResultadoAnotacionAdministrativaNoConfiable
	}
	var actos []json.RawMessage
	if json.Unmarshal(raw["actuaciones"], &actos) != nil {
		return nil, ct.ErrResultadoAnotacionAdministrativaNoConfiable
	}
	if len(o.ExpedienteSiguiente.Actuaciones) != len(actos)+1 {
		return nil, ct.ErrResultadoAnotacionAdministrativaNoConfiable
	}
	a, e := json.Marshal(o.ExpedienteSiguiente.Actuaciones[len(actos)])
	if e != nil {
		return nil, e
	}
	actos = append(actos, a)
	raw["actuaciones"], e = json.Marshal(actos)
	if e != nil {
		return nil, e
	}
	raw["version"], e = json.Marshal(o.ExpedienteSiguiente.Version)
	if e != nil {
		return nil, e
	}
	raw["actualizado_en"], e = json.Marshal(o.InstanteEfecto)
	if e != nil {
		return nil, e
	}
	return json.Marshal(raw)
}
func entradaConfirmarAnotacion(o ct.OrdenConfirmarAnotacionAdministrativa, original json.RawMessage) ([]byte, error) {
	siguiente, e := postimagenAnotacion(original, o)
	if e != nil {
		return nil, e
	}
	return json.Marshal(map[string]any{
		"material": materialAnotacion(o.Material), "expediente_anterior": original, "expediente_siguiente": siguiente, "seguimiento_original": o.SeguimientoOriginal,
		"estado_seguimiento_sha256": o.Preparacion.HuellaEstadoSeguimientoSHA256, "recibo_incorporacion_ref": o.Preparacion.ReciboIncorporacionOriginalRef,
		"ambito_idempotencia_hmac": o.Preparacion.AmbitoIdempotenciaHMAC, "huella_peticion_hmac": o.Preparacion.HuellaPeticionHMAC,
		"referencias": map[string]string{"recibo_ref": o.Referencias.ReciboRef, "evento_ref": o.Referencias.EventoRef},
		"politica":    map[string]any{"definicion_ref": o.Politica.DefinicionRef, "definicion_version": o.Politica.DefinicionVersion, "definicion_huella_sha256": o.Politica.DefinicionHuellaSHA256, "evaluada_en": o.Politica.EvaluadaEn, "valida_hasta": o.Politica.ValidaHasta}, "instante_efecto": o.InstanteEfecto})
}
func (a *TransaccionAnotacionesAdministrativasPostgreSQL) ConfirmarAnotacionAdministrativa(ctx context.Context, o ct.OrdenConfirmarAnotacionAdministrativa) (ct.ReciboAnotacionAdministrativa, error) {
	var cero ct.ReciboAnotacionAdministrativa
	if ctx == nil || a == nil || dependenciaNula(a.pool) || dependenciaNula(a.proveedor) || validarOrdenAnotacion(o, o.InstanteEfecto) != nil {
		return cero, ct.ErrPreparacionAnotacionAdministrativaInvalida
	}
	x, e := a.proveedor.ProveerMaterialAnotacionAdministrativa(ctx, o)
	if e != nil {
		return cero, errorAnotacion(ctx, e)
	}
	if x.ValidarEstructura() != nil {
		return cero, ct.ErrPreparacionAnotacionAdministrativaInvalida
	}
	s, _ := o.Evidencia.SolicitudV3.Datos()
	c, _ := o.Evidencia.ConfirmacionV3.Datos()
	v, _ := s.VinculoAutenticacionActor.Datos()
	h, e := s.Recurso.HuellaContextoAutorizacionSHA256()
	r := x.ResumenCapacidad()
	decision, de := vd.RepresentacionCanonicaDecisionAutorizacionV3(o.Evidencia.DecisionV3)
	defer clear(decision)
	if e != nil || de != nil || r.DecisionRef() != c.DecisionRef || r.DecisionHuellaSHA256() != c.DecisionHuellaSHA256 || r.ContextoRef() != v.RegistroContextoRef || r.ContextoHuellaSHA256() != v.ContextoActorHuellaSHA256 ||
		r.Operacion() != s.Accion || r.EfectoRef() != o.Material.ExpedienteRef || r.EfectoHuellaSHA256() != h || r.AudienciaConsumo() != AudienciaAnotacionAdministrativaV1 || !capacidadBreveContenidaEnConcesion(r, c) ||
		string(x.DecisionCanonica()) != string(decision) {
		return cero, ct.ErrPreparacionAnotacionAdministrativaInvalida
	}
	tx, e := a.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if e != nil {
		return cero, errorAnotacion(ctx, e)
	}
	defer revertirTransaccion(tx)
	if _, e = tx.Exec(ctx, ajustesRegistroIncorporacionTXV2); e != nil {
		return cero, errorAnotacion(ctx, e)
	}
	var ahora time.Time
	if e = tx.QueryRow(ctx, `SELECT date_trunc('microseconds',clock_timestamp())`).Scan(&ahora); e != nil {
		return cero, errorAnotacion(ctx, e)
	}
	if validarOrdenAnotacion(o, normalizarInstantePostgreSQL(ahora)) != nil {
		return cero, ct.ErrPreparacionAnotacionAdministrativaInvalida
	}
	// Relectura dentro de la misma TX: preserva el JSON anterior sin ampliar el puerto.
	prep, e := entradaPrepararAnotacion(o.Material, sellosPrepararAltaV2{Activo: parSellosPrepararAltaV2{Generacion: 1, AmbitoHMAC: o.Preparacion.AmbitoIdempotenciaHMAC, HuellaPeticionHMAC: o.Preparacion.HuellaPeticionHMAC}, Retenidos: []parSellosPrepararAltaV2{}})
	if e != nil {
		return cero, e
	}
	defer clear(prep)
	w, e := leerPreparacionAnotacion(ctx, tx, prep)
	if e != nil {
		return cero, errorAnotacion(ctx, e)
	}
	if w.Material != materialAnotacion(o.Material) || w.Seguimiento != o.SeguimientoOriginal || w.EstadoSHA != o.Preparacion.HuellaEstadoSeguimientoSHA256 || w.ReciboIncorporacion != o.Preparacion.ReciboIncorporacionOriginalRef {
		return cero, ct.ErrResultadoAnotacionAdministrativaNoConfiable
	}
	b, e := entradaConfirmarAnotacion(o, w.Expediente)
	if e != nil {
		return cero, e
	}
	defer clear(b)
	args, e := exportacionParametrosRegistroV2(x, true)
	if e != nil {
		return cero, ct.ErrPreparacionAnotacionAdministrativaInvalida
	}
	defer func() {
		for _, v := range args {
			if b, ok := v.([]byte); ok {
				clear(b)
			}
		}
	}()
	args = append([]any{b}, args...)
	var reciboJSON []byte
	defer func() { clear(reciboJSON) }()
	if e = tx.QueryRow(ctx, sqlConfirmarAnotacion, args...).Scan(&reciboJSON); e != nil {
		return cero, errorAnotacion(ctx, e)
	}
	var recibo ct.ReciboAnotacionAdministrativa
	if len(reciboJSON) > 32768 || decodificarJSONEstricto(reciboJSON, &recibo) != nil || validarReciboAnotacion(recibo, o.Preparacion) != nil {
		return cero, ct.ErrResultadoAnotacionAdministrativaNoConfiable
	}
	if o.Preparacion.Estado == ct.PreparacionAnotacionAdministrativaPreparada && !recibo.RegistradaEn.Equal(o.InstanteEfecto) {
		return cero, ct.ErrResultadoAnotacionAdministrativaNoConfiable
	}
	if e = tx.QueryRow(ctx, `SELECT date_trunc('microseconds',clock_timestamp())`).Scan(&ahora); e != nil {
		return cero, errorAnotacion(ctx, e)
	}
	if validarOrdenAnotacion(o, normalizarInstantePostgreSQL(ahora)) != nil {
		return cero, ct.ErrPreparacionAnotacionAdministrativaInvalida
	}
	if e = tx.Commit(ctx); e != nil {
		return cero, errorAnotacion(ctx, e)
	}
	return recibo, nil
}
func validarReciboAnotacion(r ct.ReciboAnotacionAdministrativa, p ct.PreparacionAnotacionAdministrativa) error {
	if r.Operacion != ct.OperacionRegistrarAnotacionAdministrativa || r.OrganizacionRef != p.Material.OrganizacionRef || r.ExpedienteRef != p.Material.ExpedienteRef || r.VersionAnterior != p.Material.VersionEsperada || r.VersionResultante != r.VersionAnterior+1 ||
		r.FaseResultante != p.Expediente.FaseActual || r.EstadoResultante != p.Expediente.EstadoActual || r.SeguimientoOriginal != p.SeguimientoOriginal || r.ReciboRef != p.Referencias.ReciboRef || r.EventoRef != p.Referencias.EventoRef || r.ActorRef != p.Material.ActorRef || !dom.ReferenciaOpacaValida(r.AuditoriaRef) || !dom.InstanteUTCCanonico(r.RegistradaEn) {
		return ct.ErrResultadoAnotacionAdministrativaNoConfiable
	}
	if p.ReciboConfirmado != nil && !reflect.DeepEqual(*p.ReciboConfirmado, r) {
		return ct.ErrResultadoAnotacionAdministrativaNoConfiable
	}
	return nil
}
func errorAnotacion(ctx context.Context, e error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pe *pgconn.PgError
	if errors.As(e, &pe) {
		switch pe.Code {
		case "P0861":
			return ct.ErrClaveIdempotenciaUsada
		case "P0865":
			return dom.ErrVersionEnConflicto
		case "P0860", "P0862":
			return ct.ErrPreparacionAnotacionAdministrativaInvalida
		}
	}
	if errors.Is(e, ct.ErrResultadoAnotacionAdministrativaNoConfiable) {
		return e
	}
	return ct.ErrPersistenciaAnotacionAdministrativaNoDisponible
}
