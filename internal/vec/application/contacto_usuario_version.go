package application

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionVersionContactoPropia         = "vec.contacto_usuario.version_propia"
	AccionVersionContactoLlamamiento    = "vec.contacto_usuario.version_para_llamamiento"
	FinalidadVersionContactoPropia      = "gestion_contacto_propio"
	FinalidadVersionContactoLlamamiento = "envio_llamamiento"
	AudienciaVersionContactoPropia      = "vec.contacto_usuario.version_propia.v1"
	AudienciaVersionContactoLlamamiento = "vec.contacto_usuario.version_llamamiento.v1"
)

func datosVersionContacto(clase ports.ClaseVersionContactoUsuario) (accion, finalidad, audiencia string, ok bool) {
	switch clase {
	case ports.VersionContactoPropia:
		return AccionVersionContactoPropia, FinalidadVersionContactoPropia, AudienciaVersionContactoPropia, true
	case ports.VersionContactoLlamamiento:
		return AccionVersionContactoLlamamiento, FinalidadVersionContactoLlamamiento, AudienciaVersionContactoLlamamiento, true
	default:
		return "", "", "", false
	}
}

type ServicioVersionContactoUsuario struct {
	auditoria   ports.PreparadorAuditoriaContactoUsuario
	autorizador ports.AutorizadorContactoUsuario
	consultor   ports.ConsultorVersionContactoUsuario
	ahora       func() time.Time
}

func NuevoServicioVersionContactoUsuario(a ports.PreparadorAuditoriaContactoUsuario, v ports.AutorizadorContactoUsuario, c ports.ConsultorVersionContactoUsuario) (*ServicioVersionContactoUsuario, error) {
	if nulo(a) || nulo(v) || nulo(c) {
		return nil, ErrContactoUsuarioNoDisponible
	}
	return &ServicioVersionContactoUsuario{a, v, c, func() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }}, nil
}

func PayloadVersionContactoUsuario(clase ports.ClaseVersionContactoUsuario, sujeto string) ([]byte, error) {
	_, finalidad, audiencia, ok := datosVersionContacto(clase)
	if !ok || !domain.ReferenciaSujetoContactoUsuarioValida(sujeto) {
		return nil, ErrContactoUsuarioNoDisponible
	}
	return json.Marshal(struct {
		Esquema, SujetoRef, FinalidadRef, Audiencia string
		Version                                     uint64
	}{"vec.contacto_usuario.version.v1", sujeto, finalidad, audiencia, 1})
}

func (s *ServicioVersionContactoUsuario) Consultar(ctx context.Context, p ports.SolicitudVersionContactoUsuario) (ports.ResultadoVersionContactoUsuario, error) {
	vacio := ports.ResultadoVersionContactoUsuario{}
	accion, finalidad, audiencia, ok := datosVersionContacto(p.Clase)
	if s == nil || ctx == nil || ctx.Err() != nil || !ok || nulo(s.auditoria) || nulo(s.autorizador) || nulo(s.consultor) || p.ContextoActor.Validar() != nil || !domain.ReferenciaSujetoContactoUsuarioValida(p.SujetoRef) || p.Recurso.Validar() != nil || p.Recurso.Referencia != p.SujetoRef || p.Recurso.ModuloID != "vec.module.usuarios" || p.Recurso.Tipo != "contacto_usuario" || p.Clase == ports.VersionContactoPropia && p.ContextoActor.PersonaRef != p.SujetoRef {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	a, err := s.auditoria.PrepararAuditoriaContactoUsuario(ctx, p.ContextoActor, accion, p.Recurso.ModuloID, p.SujetoRef, 1)
	if err != nil || !auditoriaPreparadaValidaPara(a, p.ContextoActor, accion, p.SujetoRef, 1, finalidad, p.Recurso.ModuloID) {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	negocio, err := PayloadVersionContactoUsuario(p.Clase, p.SujetoRef)
	if err != nil {
		return vacio, err
	}
	recurso, err := recursoContactoLigado(p.Recurso, negocio, p.SujetoRef, finalidad, 1, 1)
	if err != nil {
		return vacio, err
	}
	huella, err := HuellaAuditoriaPreparadaContactoUsuario(a)
	if err != nil {
		return vacio, err
	}
	recurso.Atributos["auditoria_sha256"] = huella
	base := p.SolicitudBase
	correlacion, err := base.Correlacion.ValorCanonico()
	if err != nil || correlacion != a.CorrelationRef || base.Accion != accion || base.Finalidad != finalidad || !reflect.DeepEqual(base.Recurso, p.Recurso) {
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
	if err != nil {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	acceso := ports.SolicitudAccesoContactoUsuario{Auditoria: a, SujetoRef: p.SujetoRef, ContextoActor: p.ContextoActor, FinalidadRef: finalidad, Recurso: recurso, Audiencia: audiencia, PayloadNegocio: negocio, Material: material, Version: 1, Solicitud: nominal, Decision: decision, Confirmacion: confirmacion, ResultadoContexto: p.ResultadoContexto}
	orden := ports.OrdenVersionContactoUsuario{Clase: p.Clase, Acceso: acceso}
	if ValidarOrdenVersionContactoUsuario(orden, s.ahora()) != nil {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	r, err := s.consultor.ConsultarVersionContactoUsuario(ctx, orden)
	if err != nil || ctx.Err() != nil || ValidarOrdenVersionContactoUsuario(orden, s.ahora()) != nil || ValidarResultadoVersionContactoUsuario(orden, r) != nil {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	r.AuditoriaConsulta.JSONOriginal = bytes.Clone(r.AuditoriaConsulta.JSONOriginal)
	return r, nil
}

func ValidarOrdenVersionContactoUsuario(o ports.OrdenVersionContactoUsuario, ahora time.Time) error {
	a := o.Acceso
	accion, finalidad, audiencia, ok := datosVersionContacto(o.Clase)
	b, err := PayloadVersionContactoUsuario(o.Clase, a.SujetoRef)
	h, errA := HuellaAuditoriaPreparadaContactoUsuario(a.Auditoria)
	base, errS := a.Solicitud.Datos()
	correlacion, errC := base.Correlacion.ValorCanonico()
	if !ok || err != nil || errA != nil || errS != nil || errC != nil || a.Recurso.Referencia != a.SujetoRef || o.Clase == ports.VersionContactoPropia && a.ContextoActor.PersonaRef != a.SujetoRef || a.FinalidadRef != finalidad || a.Audiencia != audiencia || a.Version != 1 || !bytes.Equal(b, a.PayloadNegocio) || a.Recurso.Atributos["auditoria_sha256"] != h || correlacion != a.Auditoria.CorrelationRef || !auditoriaPreparadaValidaPara(a.Auditoria, a.ContextoActor, accion, a.SujetoRef, 1, finalidad, a.Recurso.ModuloID) || !recursoCompromete(a.Recurso, b, a.SujetoRef, finalidad, 1, 1) || !concesionContactoValida(a.Material, a.Solicitud, a.Decision, a.Confirmacion, a.ResultadoContexto, a.ContextoActor, accion, a.Recurso, a.Audiencia, finalidad, ahora) {
		return ErrContactoUsuarioNoDisponible
	}
	return nil
}

func ValidarResultadoVersionContactoUsuario(o ports.OrdenVersionContactoUsuario, r ports.ResultadoVersionContactoUsuario) error {
	if r.SujetoRef != o.Acceso.SujetoRef || r.Encontrado != (r.Version > 0) || r.Version > 1<<53-1 {
		return ErrContactoUsuarioNoDisponible
	}
	extras := map[string]string{"version_encontrada": "false", "version_actual": "0"}
	if r.Encontrado {
		extras["version_encontrada"] = "true"
		extras["version_actual"] = strconv.FormatUint(r.Version, 10)
	}
	esperado := ports.ReciboContactoUsuario{SujetoRef: r.SujetoRef, Version: 1, ConsumoRef: r.ConsumoConsultaRef, ConsumoHuellaSHA256: r.ConsumoConsultaHuellaSHA256, EvidenciaCentral: r.AuditoriaConsulta}
	if !evidenciaCentralEsperadaContactoConMetadata(esperado, r.SujetoRef, 1, o.Acceso.Auditoria, o.Acceso.PayloadNegocio, o.Acceso.Recurso, o.Acceso.Material.ResumenCapacidad().DecisionRef(), extras) {
		return ErrContactoUsuarioNoDisponible
	}
	return nil
}

// DecodificarResultadoVersionContactoUsuario conserva el recibo de auditoría
// producido por T13 y rechaza resultados con otra versión de contrato.
func DecodificarResultadoVersionContactoUsuario(o ports.OrdenVersionContactoUsuario, encontrado bool, sujeto string, version uint64, auditoria []byte, consumoRef, consumoHuella string) (ports.ResultadoVersionContactoUsuario, error) {
	e, err := envolverEvidenciaCentralContacto(auditoria)
	if err != nil {
		return ports.ResultadoVersionContactoUsuario{}, err
	}
	r := ports.ResultadoVersionContactoUsuario{Encontrado: encontrado, SujetoRef: sujeto, Version: version, AuditoriaConsulta: e, ConsumoConsultaRef: consumoRef, ConsumoConsultaHuellaSHA256: consumoHuella}
	if ValidarResultadoVersionContactoUsuario(o, r) != nil {
		return ports.ResultadoVersionContactoUsuario{}, ErrContactoUsuarioNoDisponible
	}
	return r, nil
}
