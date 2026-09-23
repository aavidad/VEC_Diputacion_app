package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrOperacionContactoNoEncontrada   = errors.New("vec: operacion de contacto no encontrada")
	ErrOperacionContactoPreparada      = errors.New("vec: otra operacion de contacto preparada")
	ErrOperacionContactoAccesoDenegado = errors.New("vec: acceso a operacion de contacto denegado")
)

var referenciaOperacionContacto = regexp.MustCompile(`^opr_[A-Za-z0-9_-]{22,128}$`)

const (
	AccionPrepararOperacionContacto    = "vec.contacto_usuario.operacion.preparar"
	AccionCancelarOperacionContacto    = "vec.contacto_usuario.operacion.cancelar"
	AccionListarOperacionesContacto    = "vec.contacto_usuario.operacion.listar"
	AccionDetalleOperacionContacto     = "vec.contacto_usuario.operacion.detalle"
	AudienciaPrepararOperacionContacto = "vec.contacto_usuario.operacion.preparar.v1"
	AudienciaCancelarOperacionContacto = "vec.contacto_usuario.operacion.cancelar.v1"
	AudienciaListarOperacionesContacto = "vec.contacto_usuario.operacion.listar.v1"
	AudienciaDetalleOperacionContacto  = "vec.contacto_usuario.operacion.detalle.v1"
)

type ServicioOperacionesContactoUsuario struct {
	auditoria   ports.PreparadorAuditoriaContactoUsuario
	huellas     ports.DerivadorHuellasContactoUsuario
	autorizador ports.AutorizadorContactoUsuario
	repositorio ports.RepositorioOperacionesContactoUsuario
	generador   ports.GeneradorOperacionContactoUsuario
	ahora       func() time.Time
}

func NuevoServicioOperacionesContactoUsuario(a ports.PreparadorAuditoriaContactoUsuario, h ports.DerivadorHuellasContactoUsuario, v ports.AutorizadorContactoUsuario, r ports.RepositorioOperacionesContactoUsuario, g ports.GeneradorOperacionContactoUsuario) (*ServicioOperacionesContactoUsuario, error) {
	if nulo(a) || nulo(h) || nulo(v) || nulo(r) || nulo(g) {
		return nil, ErrContactoUsuarioNoDisponible
	}
	return &ServicioOperacionesContactoUsuario{a, h, v, r, g, func() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }}, nil
}

// Preparar registra sólo un HMAC semántico. Una respuesta perdida se descubre
// mediante la consulta propia de operaciones; el correo no entra en el puerto
// durable ni se recupera del índice. El repositorio decide el replay bajo
// lock y conserva la auditoría de cada consulta.
func (s *ServicioOperacionesContactoUsuario) Preparar(ctx context.Context, p ports.SolicitudPrepararOperacionContacto) (ports.OperacionContactoUsuario, error) {
	vacio := ports.OperacionContactoUsuario{}
	if s == nil || ctx == nil || ctx.Err() != nil || nulo(s.auditoria) || nulo(s.huellas) || nulo(s.autorizador) || nulo(s.repositorio) || nulo(s.generador) || p.ContextoActor.Validar() != nil || p.VersionEsperada >= 1<<53-1 || p.Recurso.Validar() != nil || p.Recurso.Referencia != p.ContextoActor.PersonaRef || p.Recurso.ModuloID != "vec.module.usuarios" || p.Recurso.Tipo != "contacto_usuario" || p.Recurso.Atributos["contacto_operacion_ref"] != "" {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	contacto, err := domain.NuevoContactoUsuario(p.ContextoActor.PersonaRef, p.Correo, p.VersionEsperada+1)
	if err != nil {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	ref, err := s.generador.NuevaOperacionContactoUsuario(ctx)
	if err != nil || !ReferenciaOperacionContactoValida(ref) || ctx.Err() != nil {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	var huellas []ports.HuellaSolicitudContactoUsuario
	err = contacto.ConDireccion(func(direccion string) error {
		var e error
		huellas, e = s.huellas.DerivarHuellasContactoUsuario(ctx, contacto.SujetoRef(), p.VersionEsperada, direccion)
		return e
	})
	if err != nil || !huellasReplayValidas(huellas) || ctx.Err() != nil {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	a, err := s.auditoria.PrepararAuditoriaContactoUsuario(ctx, p.ContextoActor, AccionPrepararOperacionContacto, p.Recurso.ModuloID, p.ContextoActor.PersonaRef, p.VersionEsperada+1)
	if err != nil || !auditoriaPreparadaValidaPara(a, p.ContextoActor, AccionPrepararOperacionContacto, p.ContextoActor.PersonaRef, p.VersionEsperada+1, "gestion_contacto_propio", p.Recurso.ModuloID) {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	negocio, err := json.Marshal(struct {
		Esquema, SujetoRef, OperacionRef string
		VersionEsperada                  uint64
		HuellasReplay                    []ports.HuellaSolicitudContactoUsuario
		Auditoria                        domain.AuditEntry
	}{AudienciaPrepararOperacionContacto, p.ContextoActor.PersonaRef, ref, p.VersionEsperada, huellas, a})
	if err != nil || len(negocio) == 0 || len(negocio) > 65536 {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	recurso := clonarRecurso(p.Recurso)
	if recurso.Atributos == nil {
		recurso.Atributos = make(map[string]string)
	}
	if recurso.Atributos["material_sha256"] != "" || recurso.Atributos["contacto_version_esperada"] != "" || recurso.Atributos["contacto_sujeto_ref"] != "" {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	huella := sha256.Sum256(negocio)
	recurso.Atributos["material_sha256"] = hex.EncodeToString(huella[:])
	recurso.Atributos["contacto_operacion_ref"] = ref
	recurso.Atributos["contacto_sujeto_ref"] = p.ContextoActor.PersonaRef
	recurso.Atributos["contacto_version_esperada"] = strconv.FormatUint(p.VersionEsperada, 10)
	if recurso.Validar() != nil {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	base := p.SolicitudBase
	correlacion, err := base.Correlacion.ValorCanonico()
	if err != nil || correlacion != a.CorrelationRef || base.Accion != AccionPrepararOperacionContacto || base.Finalidad != "gestion_contacto_propio" || !reflect.DeepEqual(base.Recurso, p.Recurso) {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	base.Recurso = recurso
	nominal, err := domain.NuevaSolicitudAutorizacionLigadaV3(base)
	if err != nil || !contextoContactoValido(nominal, p.ResultadoContexto, p.ContextoActor, s.ahora()) {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, nominal, p.ResultadoContexto)
	if err != nil || nulo(exportador) || ctx.Err() != nil {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !concesionContactoValida(material, nominal, decision, confirmacion, p.ResultadoContexto, p.ContextoActor, AccionPrepararOperacionContacto, recurso, AudienciaPrepararOperacionContacto, "gestion_contacto_propio", s.ahora()) {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	acceso := ports.SolicitudAccesoContactoUsuario{Auditoria: a, SujetoRef: p.ContextoActor.PersonaRef, ContextoActor: p.ContextoActor, FinalidadRef: "gestion_contacto_propio", Recurso: recurso, Audiencia: AudienciaPrepararOperacionContacto,
		PayloadNegocio: bytes.Clone(negocio), Material: material, Version: p.VersionEsperada + 1, Solicitud: nominal, Decision: decision, Confirmacion: confirmacion, ResultadoContexto: p.ResultadoContexto}
	op, err := s.repositorio.PrepararOperacionContacto(ctx, ports.OrdenPrepararOperacionContacto{OperacionRef: ref, SujetoRef: p.ContextoActor.PersonaRef, VersionEsperada: p.VersionEsperada, HuellasReplay: append([]ports.HuellaSolicitudContactoUsuario(nil), huellas...), Acceso: acceso})
	if errors.Is(err, ErrContactoUsuarioConflicto) {
		return vacio, err
	}
	if errors.Is(err, ErrOperacionContactoPreparada) && ReferenciaOperacionContactoValida(op.OperacionRef) && op.VersionEsperada == p.VersionEsperada {
		e, eErr := ValidarEvidenciaCentralOperacionContacto(op.AuditoriaOperacion.JSONOriginal, acceso, ref, "conflicto", op.ConsumoRef, op.ConsumoHuellaSHA256)
		if eErr == nil && e.Referencia == op.AuditoriaOperacion.Referencia && e.HuellaJSONSHA256 == op.AuditoriaOperacion.HuellaJSONSHA256 {
			return op, err
		}
	}
	if err != nil || ctx.Err() != nil || (op.Estado != ports.OperacionContactoPreparada && (op.Estado != ports.OperacionContactoConfirmada || !op.ReplayConfirmado)) || op.VersionEsperada != p.VersionEsperada || ValidarOperacionContacto(op) != nil || !ReferenciaOperacionContactoValida(op.OperacionRef) {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	e, err := ValidarEvidenciaCentralOperacionContacto(op.AuditoriaOperacion.JSONOriginal, acceso, ref, string(op.Estado), op.ConsumoRef, op.ConsumoHuellaSHA256)
	if err != nil || e.Referencia != op.AuditoriaOperacion.Referencia || e.HuellaJSONSHA256 != op.AuditoriaOperacion.HuellaJSONSHA256 {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	return op, nil
}

func (s *ServicioOperacionesContactoUsuario) Cancelar(ctx context.Context, p ports.SolicitudGestionOperacionContacto) (ports.OperacionContactoUsuario, error) {
	vacio := ports.OperacionContactoUsuario{}
	if !ReferenciaOperacionContactoValida(p.OperacionRef) {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	a, err := s.autorizarGestion(ctx, p, AccionCancelarOperacionContacto, AudienciaCancelarOperacionContacto)
	if err != nil {
		return vacio, err
	}
	op, err := s.repositorio.CancelarOperacionContacto(ctx, ports.OrdenCancelarOperacionContacto{OperacionRef: p.OperacionRef, SujetoRef: p.ContextoActor.PersonaRef, Acceso: a})
	if errors.Is(err, ErrOperacionContactoNoEncontrada) || errors.Is(err, ErrContactoUsuarioConflicto) || errors.Is(err, ErrContactoUsuarioCommitIncierto) {
		return vacio, err
	}
	if err != nil || ctx.Err() != nil || op.OperacionRef != p.OperacionRef || op.Estado != ports.OperacionContactoCancelada || ValidarOperacionContacto(op) != nil {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	e, err := ValidarEvidenciaCentralOperacionContacto(op.AuditoriaOperacion.JSONOriginal, a, p.OperacionRef, "cancelada", op.ConsumoRef, op.ConsumoHuellaSHA256)
	if err != nil || e.Referencia != op.AuditoriaOperacion.Referencia || e.HuellaJSONSHA256 != op.AuditoriaOperacion.HuellaJSONSHA256 {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	return op, nil
}

func (s *ServicioOperacionesContactoUsuario) Listar(ctx context.Context, p ports.SolicitudGestionOperacionContacto) (ports.ResultadoListaOperacionesContacto, error) {
	vacio := ports.ResultadoListaOperacionesContacto{}
	if p.Limite < 1 || p.Limite > 50 || p.DespuesDe != "" && !ReferenciaOperacionContactoValida(p.DespuesDe) || p.OperacionRef != "" {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	a, err := s.autorizarGestion(ctx, p, AccionListarOperacionesContacto, AudienciaListarOperacionesContacto)
	if err != nil {
		return vacio, err
	}
	r, err := s.repositorio.ListarOperacionesContacto(ctx, ports.OrdenListarOperacionesContacto{SujetoRef: p.ContextoActor.PersonaRef, Limite: p.Limite, DespuesDe: p.DespuesDe, Acceso: a})
	if errors.Is(err, ErrOperacionContactoNoEncontrada) || errors.Is(err, ErrContactoUsuarioCommitIncierto) {
		return vacio, err
	}
	if err != nil || ctx.Err() != nil || len(r.Operaciones) > int(p.Limite) || r.SiguienteDesde != "" && (len(r.Operaciones) == 0 || r.SiguienteDesde != r.Operaciones[len(r.Operaciones)-1].OperacionRef) {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	for _, op := range r.Operaciones {
		if ValidarOperacionContacto(op) != nil {
			return vacio, ErrContactoUsuarioNoDisponible
		}
	}
	e, err := ValidarEvidenciaCentralOperacionContacto(r.Auditoria.JSONOriginal, a, p.DespuesDe, "consulta", r.ConsumoRef, r.ConsumoHuellaSHA256)
	if err != nil || e.Referencia != r.Auditoria.Referencia || e.HuellaJSONSHA256 != r.Auditoria.HuellaJSONSHA256 {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	return r, nil
}

func (s *ServicioOperacionesContactoUsuario) Detalle(ctx context.Context, p ports.SolicitudGestionOperacionContacto) (ports.ResultadoDetalleOperacionContacto, error) {
	vacio := ports.ResultadoDetalleOperacionContacto{}
	if !ReferenciaOperacionContactoValida(p.OperacionRef) || p.Limite != 0 || p.DespuesDe != "" {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	a, err := s.autorizarGestion(ctx, p, AccionDetalleOperacionContacto, AudienciaDetalleOperacionContacto)
	if err != nil {
		return vacio, err
	}
	r, err := s.repositorio.DetalleOperacionContacto(ctx, ports.OrdenDetalleOperacionContacto{OperacionRef: p.OperacionRef, SujetoRef: p.ContextoActor.PersonaRef, Acceso: a})
	if errors.Is(err, ErrContactoUsuarioCommitIncierto) {
		return vacio, err
	}
	if err != nil || ctx.Err() != nil || r.Encontrada && (r.Operacion.OperacionRef != p.OperacionRef || ValidarOperacionContacto(r.Operacion) != nil) || !r.Encontrada && r.Operacion.OperacionRef != "" {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	estado := "ausente"
	if r.Encontrada {
		estado = "encontrada"
	}
	e, err := ValidarEvidenciaCentralOperacionContacto(r.Auditoria.JSONOriginal, a, p.OperacionRef, estado, r.ConsumoRef, r.ConsumoHuellaSHA256)
	if err != nil || e.Referencia != r.Auditoria.Referencia || e.HuellaJSONSHA256 != r.Auditoria.HuellaJSONSHA256 {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	return r, nil
}

func (s *ServicioOperacionesContactoUsuario) autorizarGestion(ctx context.Context, p ports.SolicitudGestionOperacionContacto, accion, audiencia string) (ports.SolicitudAccesoContactoUsuario, error) {
	vacio := ports.SolicitudAccesoContactoUsuario{}
	if s == nil || ctx == nil || ctx.Err() != nil || nulo(s.auditoria) || nulo(s.autorizador) || nulo(s.repositorio) || p.ContextoActor.Validar() != nil || p.Recurso.Validar() != nil || p.Recurso.Referencia != p.ContextoActor.PersonaRef || p.Recurso.ModuloID != "vec.module.usuarios" || p.Recurso.Tipo != "contacto_usuario" || p.Recurso.Atributos != nil {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	switch accion {
	case AccionCancelarOperacionContacto, AccionDetalleOperacionContacto:
		if !ReferenciaOperacionContactoValida(p.OperacionRef) || p.Limite != 0 || p.DespuesDe != "" {
			return vacio, ErrContactoUsuarioNoDisponible
		}
	case AccionListarOperacionesContacto:
		if p.OperacionRef != "" || p.Limite < 1 || p.Limite > 50 || p.DespuesDe != "" && !ReferenciaOperacionContactoValida(p.DespuesDe) {
			return vacio, ErrContactoUsuarioNoDisponible
		}
	default:
		return vacio, ErrContactoUsuarioNoDisponible
	}
	a, err := s.auditoria.PrepararAuditoriaContactoUsuario(ctx, p.ContextoActor, accion, p.Recurso.ModuloID, p.ContextoActor.PersonaRef, 1)
	if err != nil || !auditoriaPreparadaValidaPara(a, p.ContextoActor, accion, p.ContextoActor.PersonaRef, 1, "gestion_contacto_propio", p.Recurso.ModuloID) {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	var negocio []byte
	if accion == AccionListarOperacionesContacto {
		negocio, err = json.Marshal(struct {
			Esquema, SujetoRef string
			Limite             uint32
			DespuesDe          string
			Auditoria          domain.AuditEntry
		}{audiencia, p.ContextoActor.PersonaRef, p.Limite, p.DespuesDe, a})
	} else {
		negocio, err = json.Marshal(struct {
			Esquema, SujetoRef, OperacionRef string
			Auditoria                        domain.AuditEntry
		}{audiencia, p.ContextoActor.PersonaRef, p.OperacionRef, a})
	}
	if err != nil || len(negocio) == 0 || len(negocio) > 65536 {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	recurso := clonarRecurso(p.Recurso)
	recurso.Atributos = map[string]string{"contacto_sujeto_ref": p.ContextoActor.PersonaRef}
	if p.OperacionRef != "" {
		recurso.Atributos["contacto_operacion_ref"] = p.OperacionRef
	}
	huella := sha256.Sum256(negocio)
	recurso.Atributos["material_sha256"] = hex.EncodeToString(huella[:])
	if recurso.Validar() != nil {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	base := p.SolicitudBase
	correlacion, err := base.Correlacion.ValorCanonico()
	if err != nil || correlacion != a.CorrelationRef || base.Accion != accion || base.Finalidad != "gestion_contacto_propio" || !reflect.DeepEqual(base.Recurso, p.Recurso) {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	base.Recurso = recurso
	nominal, err := domain.NuevaSolicitudAutorizacionLigadaV3(base)
	if err != nil || !contextoContactoValido(nominal, p.ResultadoContexto, p.ContextoActor, s.ahora()) {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, nominal, p.ResultadoContexto)
	if err != nil || nulo(exportador) || ctx.Err() != nil {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !concesionContactoValida(material, nominal, decision, confirmacion, p.ResultadoContexto, p.ContextoActor, accion, recurso, audiencia, "gestion_contacto_propio", s.ahora()) {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	return ports.SolicitudAccesoContactoUsuario{Auditoria: a, SujetoRef: p.ContextoActor.PersonaRef, ContextoActor: p.ContextoActor,
		FinalidadRef: "gestion_contacto_propio", Recurso: recurso, Audiencia: audiencia, PayloadNegocio: bytes.Clone(negocio), Material: material,
		Version: 1, Solicitud: nominal, Decision: decision, Confirmacion: confirmacion, ResultadoContexto: p.ResultadoContexto}, nil
}

func ReferenciaOperacionContactoValida(ref string) bool {
	return referenciaOperacionContacto.MatchString(ref)
}

func ValidarOperacionContacto(op ports.OperacionContactoUsuario) error {
	if !ReferenciaOperacionContactoValida(op.OperacionRef) || op.VersionEsperada >= 1<<53-1 {
		return ErrContactoUsuarioNoDisponible
	}
	switch op.Estado {
	case ports.OperacionContactoPreparada, ports.OperacionContactoCancelada:
		if op.Version != 0 || op.ReciboRef != "" {
			return ErrContactoUsuarioNoDisponible
		}
	case ports.OperacionContactoConfirmada:
		if op.Version != op.VersionEsperada+1 || op.ReciboRef == "" {
			return ErrContactoUsuarioNoDisponible
		}
	default:
		return ErrContactoUsuarioNoDisponible
	}
	return nil
}

// La firma de cadena pertenece a T13. Se cotejan sus bytes canónicos,
// referencia y metadatos contra el consumo V3 actual; verificar la cadena
// completa exige además el registro/ancla de la autoridad central.
func ValidarEvidenciaCentralOperacionContacto(raw []byte, acceso ports.SolicitudAccesoContactoUsuario, operacionRef, estado, consumoRef, consumoHuella string) (ports.EvidenciaAuditoriaCentralContactoUsuario, error) {
	e, err := envolverEvidenciaCentralContacto(raw)
	if err != nil || acceso.Auditoria.SubjectRef != acceso.SujetoRef || acceso.Auditoria.Action == "" || acceso.Version == 0 {
		return ports.EvidenciaAuditoriaCentralContactoUsuario{}, ErrContactoUsuarioNoDisponible
	}
	r := ports.ReciboContactoUsuario{SujetoRef: acceso.SujetoRef, Version: acceso.Version, ConsumoRef: consumoRef, ConsumoHuellaSHA256: consumoHuella, EvidenciaCentral: e}
	if !evidenciaCentralEsperadaContactoConMetadata(r, acceso.SujetoRef, acceso.Version, acceso.Auditoria,
		acceso.PayloadNegocio, acceso.Recurso, acceso.Material.ResumenCapacidad().DecisionRef(),
		map[string]string{"operacion_ref": operacionRef, "estado_operacion": estado}) {
		return ports.EvidenciaAuditoriaCentralContactoUsuario{}, ErrContactoUsuarioNoDisponible
	}
	return e, nil
}
