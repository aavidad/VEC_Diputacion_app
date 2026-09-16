package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const AudienciaRegistroContactoRecuperable = "vec.contacto_usuario.registro.v2"

var ErrIntencionContactoDivergente = errors.New("vec: intención de contacto divergente")
var ErrVersionContactoDivergente = errors.New("vec: versión de contacto divergente")
var patronIntencionContacto = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func IntencionContactoValida(ref string) bool { return patronIntencionContacto.MatchString(ref) }

type ServicioRegistroContactoRecuperable struct {
	base     *ServicioContactoUsuario
	sellador ports.SelladorHuellaPeticionContactoUsuario
	registro ports.RegistroContactoRecuperable
}

func NuevoServicioRegistroContactoRecuperable(base *ServicioContactoUsuario, sellador ports.SelladorHuellaPeticionContactoUsuario, registro ports.RegistroContactoRecuperable) (*ServicioRegistroContactoRecuperable, error) {
	if base == nil || nulo(base.auditoria) || nulo(base.protector) || nulo(base.autorizador) || base.ahora == nil || nulo(sellador) || nulo(registro) {
		return nil, ErrContactoUsuarioNoDisponible
	}
	return &ServicioRegistroContactoRecuperable{base, sellador, registro}, nil
}

func (s *ServicioRegistroContactoRecuperable) Guardar(ctx context.Context, peticion ports.SolicitudRegistroContactoRecuperable) (ports.ResultadoRegistroContactoRecuperable, error) {
	vacio := ports.ResultadoRegistroContactoRecuperable{}
	q := peticion.Solicitud
	if s == nil || s.base == nil || ctx == nil || ctx.Err() != nil || !IntencionContactoValida(peticion.IntentRef) || q.ContextoActor.Validar() != nil || q.Contacto.Validar() != nil || q.Contacto.SujetoRef() != q.ContextoActor.PersonaRef || q.VersionEsperada >= 1<<53-1 || q.Contacto.Version() != q.VersionEsperada+1 || q.FinalidadRef != "gestion_contacto_propio" || q.Audiencia != AudienciaRegistroContactoRecuperable || q.Recurso.Validar() != nil || q.Recurso.Referencia != q.ContextoActor.PersonaRef {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	accion := accionRegistroContactoRecuperable(q.VersionEsperada)
	auditoria, err := s.base.auditoria.PrepararAuditoriaContactoUsuario(ctx, q.ContextoActor, accion, q.Recurso.ModuloID, q.Contacto.SujetoRef(), q.Contacto.Version())
	if err != nil || !auditoriaPreparadaValida(auditoria, q.ContextoActor, accion, q.Contacto, q.FinalidadRef, q.Recurso.ModuloID) {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	var huella string
	err = q.Contacto.ConDireccion(func(direccion string) error {
		claro := []byte(direccion)
		defer borrar(claro)
		var e error
		huella, e = s.sellador.HuellaPeticionContactoUsuario(ctx, q.Contacto.SujetoRef(), q.VersionEsperada, claro)
		return e
	})
	if err != nil || !hexContacto(huella, 64) || huella == strings.Repeat("0", 64) {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	// Reutilizar el protector y la preparación existentes, no su contrato V1.
	p, err := s.base.preparar(ctx, q, auditoria)
	if err != nil || ctx.Err() != nil {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	preparacion := ports.PreparacionRegistroContactoRecuperable{Registro: p, IntentRef: peticion.IntentRef, HuellaPeticion: huella}
	cuerpo, err := cuerpoRegistroContactoRecuperable(preparacion)
	if err != nil {
		return vacio, err
	}
	recurso, err := recursoContactoLigado(q.Recurso, cuerpo, p.SujetoRef, p.FinalidadRef, p.VersionEsperada, p.VersionNueva)
	if err != nil {
		return vacio, err
	}
	for _, clave := range []string{"contacto_intent_ref", "contacto_huella_peticion"} {
		if _, existe := recurso.Atributos[clave]; existe {
			return vacio, ErrContactoUsuarioNoDisponible
		}
	}
	recurso.Atributos["contacto_intent_ref"] = peticion.IntentRef
	recurso.Atributos["contacto_huella_peticion"] = huella
	preparacion.Registro.Recurso = recurso
	preparacion.Registro.PayloadNegocio = cuerpo
	base := q.SolicitudBase
	correlacion, err := base.Correlacion.ValorCanonico()
	if err != nil || correlacion != auditoria.CorrelationRef || base.Accion != accion || base.Finalidad != q.FinalidadRef || !reflect.DeepEqual(base.Recurso, q.Recurso) {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	base.Recurso = recurso
	nominal, err := domain.NuevaSolicitudAutorizacionLigadaV3(base)
	if err != nil || !contextoContactoValido(nominal, q.ResultadoContexto, q.ContextoActor, s.base.ahora()) {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	decision, confirmacion, exportador, err := s.base.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, nominal, q.ResultadoContexto)
	if err != nil || nulo(exportador) || ctx.Err() != nil {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	orden := ports.OrdenRegistroContactoRecuperable{Preparacion: preparacion, Autorizacion: ports.SolicitudAccesoContactoUsuario{
		Auditoria: auditoria, SujetoRef: p.SujetoRef, ContextoActor: q.ContextoActor, FinalidadRef: p.FinalidadRef,
		Recurso: recurso, Audiencia: p.Audiencia, PayloadNegocio: cuerpo, Material: material, Version: p.VersionNueva,
		Solicitud: nominal, Decision: decision, Confirmacion: confirmacion, ResultadoContexto: q.ResultadoContexto,
	}}
	if ValidarOrdenRegistroContactoRecuperable(orden, s.base.ahora()) != nil || ctx.Err() != nil {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	resultado, err := s.registro.GuardarContactoRecuperable(ctx, clonarOrdenRegistroContactoRecuperable(orden))
	if err != nil {
		return vacio, err
	}
	if ValidarResultadoRegistroContactoRecuperable(orden, resultado) != nil {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	// El nuevo efecto confirmado conserva su recibo incluso ante cancelación.
	// Recuperar evidencia histórica sí requiere vigencia al exponerla.
	if resultado.Recuperado && (ctx.Err() != nil || ValidarOrdenRegistroContactoRecuperable(orden, s.base.ahora()) != nil) {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	resultado.ReciboOriginal.EvidenciaCentral.JSONOriginal = bytes.Clone(resultado.ReciboOriginal.EvidenciaCentral.JSONOriginal)
	resultado.AuditoriaIntento.JSONOriginal = bytes.Clone(resultado.AuditoriaIntento.JSONOriginal)
	return resultado, nil
}

func accionRegistroContactoRecuperable(anterior uint64) string {
	if anterior == 0 {
		return AccionAltaContactoUsuario
	}
	return AccionActualizarContactoUsuario
}

func cuerpoRegistroContactoRecuperable(p ports.PreparacionRegistroContactoRecuperable) ([]byte, error) {
	r := p.Registro
	if !preparacionValida(r) || r.Audiencia != AudienciaRegistroContactoRecuperable || r.FinalidadRef != "gestion_contacto_propio" || r.VersionNueva > 1<<53-1 || !IntencionContactoValida(p.IntentRef) || !hexContacto(p.HuellaPeticion, 64) || p.HuellaPeticion == strings.Repeat("0", 64) {
		return nil, ErrContactoUsuarioNoDisponible
	}
	return json.Marshal(struct {
		Esquema, SujetoRef, Audiencia, FinalidadRef string
		VersionEsperada, VersionNueva               uint64
		IntentRef, HuellaPeticion                   string
		Sobre                                       ports.SobreContactoUsuario
		Auditoria                                   domain.AuditEntry
	}{"vec.contacto_usuario.registro-cuerpo.v2", r.SujetoRef, r.Audiencia, r.FinalidadRef, r.VersionEsperada, r.VersionNueva, p.IntentRef, p.HuellaPeticion, r.Sobre, r.Auditoria})
}

func PayloadRegistroContactoRecuperable(p ports.PreparacionRegistroContactoRecuperable) ([]byte, error) {
	b, err := cuerpoRegistroContactoRecuperable(p)
	r := p.Registro
	if err != nil || !recursoCompromete(r.Recurso, b, r.SujetoRef, r.FinalidadRef, r.VersionEsperada, r.VersionNueva) || r.Recurso.Atributos["contacto_intent_ref"] != p.IntentRef || r.Recurso.Atributos["contacto_huella_peticion"] != p.HuellaPeticion {
		return nil, ErrContactoUsuarioNoDisponible
	}
	return b, nil
}

func ValidarOrdenRegistroContactoRecuperable(o ports.OrdenRegistroContactoRecuperable, ahora time.Time) error {
	p, a := o.Preparacion.Registro, o.Autorizacion
	b, err := PayloadRegistroContactoRecuperable(o.Preparacion)
	accion := accionRegistroContactoRecuperable(p.VersionEsperada)
	if err != nil || !bytes.Equal(b, p.PayloadNegocio) || !bytes.Equal(b, a.PayloadNegocio) || p.SujetoRef != a.ContextoActor.PersonaRef || p.SujetoRef != a.SujetoRef || p.Recurso.Referencia != p.SujetoRef || p.VersionNueva != a.Version || p.Audiencia != a.Audiencia || p.FinalidadRef != a.FinalidadRef || !reflect.DeepEqual(p.Recurso, a.Recurso) || !reflect.DeepEqual(p.Auditoria, a.Auditoria) || !auditoriaPreparadaValidaPara(p.Auditoria, a.ContextoActor, accion, p.SujetoRef, p.VersionNueva, p.FinalidadRef, p.Recurso.ModuloID) || !concesionContactoValida(a.Material, a.Solicitud, a.Decision, a.Confirmacion, a.ResultadoContexto, a.ContextoActor, accion, p.Recurso, p.Audiencia, p.FinalidadRef, ahora) {
		return ErrContactoUsuarioNoDisponible
	}
	return nil
}

func DecodificarResultadoRegistroContactoRecuperable(o ports.OrdenRegistroContactoRecuperable, recuperado bool, sujeto string, version uint64, original []byte, consumoOriginalRef, consumoOriginalHuella string, intento []byte, consumoIntentoRef, consumoIntentoHuella string) (ports.ResultadoRegistroContactoRecuperable, error) {
	vacio := ports.ResultadoRegistroContactoRecuperable{}
	e, err := envolverEvidenciaCentralContacto(original)
	if err != nil {
		return vacio, err
	}
	i, err := envolverEvidenciaCentralContacto(intento)
	if err != nil {
		return vacio, err
	}
	r := ports.ResultadoRegistroContactoRecuperable{Recuperado: recuperado, ReciboOriginal: ports.ReciboContactoUsuario{SujetoRef: sujeto, Version: version, ConsumoRef: consumoOriginalRef, ConsumoHuellaSHA256: consumoOriginalHuella, EvidenciaCentral: e}, AuditoriaIntento: i, ConsumoIntentoRef: consumoIntentoRef, ConsumoIntentoHuellaSHA256: consumoIntentoHuella}
	if err = ValidarResultadoRegistroContactoRecuperable(o, r); err != nil {
		return vacio, err
	}
	return r, nil
}

func ValidarResultadoRegistroContactoRecuperable(o ports.OrdenRegistroContactoRecuperable, r ports.ResultadoRegistroContactoRecuperable) error {
	p, a := o.Preparacion.Registro, o.Autorizacion
	if r.ReciboOriginal.SujetoRef != p.SujetoRef || r.ReciboOriginal.Version != p.VersionNueva {
		return ErrContactoUsuarioNoDisponible
	}
	actual := ports.ReciboContactoUsuario{SujetoRef: p.SujetoRef, Version: p.VersionNueva, ConsumoRef: r.ConsumoIntentoRef, ConsumoHuellaSHA256: r.ConsumoIntentoHuellaSHA256, EvidenciaCentral: r.AuditoriaIntento}
	var extras map[string]string
	if r.Recuperado {
		if !reciboOriginalContactoValido(r.ReciboOriginal, p.SujetoRef, p.VersionNueva, p.Recurso.ModuloID, p.Auditoria.CorrelationRef) || r.ReciboOriginal.ConsumoRef == r.ConsumoIntentoRef || r.ReciboOriginal.EvidenciaCentral.Referencia == r.AuditoriaIntento.Referencia {
			return ErrContactoUsuarioNoDisponible
		}
		extras = map[string]string{"intencion_ref": o.Preparacion.IntentRef, "recibo_original_ref": r.ReciboOriginal.EvidenciaCentral.Referencia, "recibo_original_sha256": r.ReciboOriginal.EvidenciaCentral.HuellaJSONSHA256}
	} else if !reflect.DeepEqual(actual, r.ReciboOriginal) {
		return ErrContactoUsuarioNoDisponible
	}
	if !evidenciaCentralEsperadaContactoConMetadata(actual, p.SujetoRef, p.VersionNueva, p.Auditoria, p.PayloadNegocio, p.Recurso, a.Material.ResumenCapacidad().DecisionRef(), extras) {
		return ErrContactoUsuarioNoDisponible
	}
	return nil
}

func clonarOrdenRegistroContactoRecuperable(o ports.OrdenRegistroContactoRecuperable) ports.OrdenRegistroContactoRecuperable {
	o.Preparacion.Registro = clonarPreparacion(o.Preparacion.Registro)
	o.Autorizacion = clonarAccesoContacto(o.Autorizacion)
	o.Autorizacion.ContextoActor, _ = o.Autorizacion.ContextoActor.Clonar()
	return o
}
