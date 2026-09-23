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
	if err != nil || ctx.Err() != nil || op.Estado != ports.OperacionContactoPreparada || op.VersionEsperada != p.VersionEsperada || ValidarOperacionContacto(op) != nil || !ReferenciaOperacionContactoValida(op.OperacionRef) {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	e, err := ValidarEvidenciaCentralOperacionContacto(op.AuditoriaOperacion.JSONOriginal, acceso, ref, "preparada", op.ConsumoRef, op.ConsumoHuellaSHA256)
	if err != nil || e.Referencia != op.AuditoriaOperacion.Referencia || e.HuellaJSONSHA256 != op.AuditoriaOperacion.HuellaJSONSHA256 {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	return op, nil
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
