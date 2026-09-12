package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var ErrContactoUsuarioNoDisponible = errors.New("vec: contacto de usuario no disponible")

const (
	AccionAltaContactoUsuario       = "vec.contacto_usuario.alta"
	AccionActualizarContactoUsuario = "vec.contacto_usuario.actualizar"
	AccionConsultarContactoUsuario  = "vec.contacto_usuario.consultar"
)

type ServicioContactoUsuario struct {
	auditoria   ports.PreparadorAuditoriaContactoUsuario
	protector   ports.ProtectorContactoUsuario
	autorizador ports.AutorizadorContactoUsuario
	registro    ports.RegistroContactoUsuario
	lector      ports.ResolutorContactoUsuarioAutorizado
	ahora       func() time.Time
}

func NuevoServicioContactoUsuario(auditoria ports.PreparadorAuditoriaContactoUsuario, protector ports.ProtectorContactoUsuario, autorizador ports.AutorizadorContactoUsuario, registro ports.RegistroContactoUsuario) (*ServicioContactoUsuario, error) {
	if nulo(auditoria) || nulo(protector) || nulo(autorizador) || nulo(registro) {
		return nil, ErrContactoUsuarioNoDisponible
	}
	return &ServicioContactoUsuario{auditoria: auditoria, protector: protector, autorizador: autorizador, registro: registro, ahora: func() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }}, nil
}

func NuevoServicioContactoUsuarioConLectura(auditoria ports.PreparadorAuditoriaContactoUsuario, protector ports.ProtectorContactoUsuario, autorizador ports.AutorizadorContactoUsuario, registro ports.RegistroContactoUsuario, lector ports.ResolutorContactoUsuarioAutorizado) (*ServicioContactoUsuario, error) {
	s, err := NuevoServicioContactoUsuario(auditoria, protector, autorizador, registro)
	if err != nil || nulo(lector) {
		return nil, ErrContactoUsuarioNoDisponible
	}
	s.lector = lector
	return s, nil
}

func (s *ServicioContactoUsuario) Guardar(ctx context.Context, solicitud ports.SolicitudRegistroContactoUsuario) (ports.ReciboContactoUsuario, error) {
	if s == nil || ctx == nil || ctx.Err() != nil || nulo(s.auditoria) || nulo(s.protector) || nulo(s.autorizador) || nulo(s.registro) || solicitud.ContextoActor.Validar() != nil || solicitud.Contacto.Validar() != nil || solicitud.Contacto.SujetoRef() != solicitud.ContextoActor.PersonaRef || !textoSeguro(solicitud.FinalidadRef) || solicitud.Recurso.Validar() != nil || !textoSeguro(solicitud.Audiencia) {
		return ports.ReciboContactoUsuario{}, ErrContactoUsuarioNoDisponible
	}
	accion := AccionAltaContactoUsuario
	if solicitud.VersionEsperada == 0 {
		if solicitud.Contacto.Version() != 1 {
			return ports.ReciboContactoUsuario{}, ErrContactoUsuarioNoDisponible
		}
	} else {
		accion = AccionActualizarContactoUsuario
		if solicitud.VersionEsperada == ^uint64(0) || solicitud.Contacto.Version() != solicitud.VersionEsperada+1 {
			return ports.ReciboContactoUsuario{}, ErrContactoUsuarioNoDisponible
		}
	}
	auditoria, err := s.auditoria.PrepararAuditoriaContactoUsuario(ctx, solicitud.ContextoActor, accion, solicitud.Recurso.ModuloID, solicitud.Contacto.SujetoRef(), solicitud.Contacto.Version())
	if err != nil || !auditoriaPreparadaValida(auditoria, solicitud.ContextoActor, accion, solicitud.Contacto, solicitud.FinalidadRef, solicitud.Recurso.ModuloID) {
		return ports.ReciboContactoUsuario{}, ErrContactoUsuarioNoDisponible
	}
	preparacion, err := s.preparar(ctx, solicitud, auditoria)
	if err != nil || ctx.Err() != nil {
		return ports.ReciboContactoUsuario{}, ErrContactoUsuarioNoDisponible
	}
	base := solicitud.SolicitudBase
	correlacion, err := base.Correlacion.ValorCanonico()
	if err != nil || correlacion != auditoria.CorrelationRef || !reflect.DeepEqual(base.Recurso, solicitud.Recurso) {
		return ports.ReciboContactoUsuario{}, ErrContactoUsuarioNoDisponible
	}
	base.Recurso = preparacion.Recurso
	if base.Accion != accion || base.Finalidad != solicitud.FinalidadRef {
		return ports.ReciboContactoUsuario{}, ErrContactoUsuarioNoDisponible
	}
	nominal, err := domain.NuevaSolicitudAutorizacionLigadaV3(base)
	if err != nil || !contextoContactoValido(nominal, solicitud.ResultadoContexto, solicitud.ContextoActor, s.ahora()) {
		return ports.ReciboContactoUsuario{}, ErrContactoUsuarioNoDisponible
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, nominal, solicitud.ResultadoContexto)
	if err != nil || nulo(exportador) || ctx.Err() != nil {
		return ports.ReciboContactoUsuario{}, ErrContactoUsuarioNoDisponible
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || ctx.Err() != nil || !concesionContactoValida(material, nominal, decision, confirmacion, solicitud.ResultadoContexto, solicitud.ContextoActor, accion, preparacion.Recurso, preparacion.Audiencia, preparacion.FinalidadRef, s.ahora()) {
		return ports.ReciboContactoUsuario{}, ErrContactoUsuarioNoDisponible
	}
	recibo, err := s.registro.GuardarContactoUsuario(ctx, ports.OrdenRegistroContactoUsuario{Preparacion: clonarPreparacion(preparacion), Material: material})
	if err != nil || !reciboValido(recibo, preparacion, material.ResumenCapacidad().DecisionRef()) {
		return ports.ReciboContactoUsuario{}, ErrContactoUsuarioNoDisponible
	}
	recibo.EvidenciaCentral.JSONOriginal = bytes.Clone(recibo.EvidenciaCentral.JSONOriginal)
	return recibo, nil
}

func (s *ServicioContactoUsuario) preparar(ctx context.Context, solicitud ports.SolicitudRegistroContactoUsuario, auditoria domain.AuditEntry) (ports.PreparacionRegistroContactoUsuario, error) {
	var sobre ports.SobreContactoUsuario
	err := solicitud.Contacto.ConDireccion(func(direccion string) error {
		claro := []byte(direccion)
		defer borrar(claro)
		var e error
		sobre, e = s.protector.CifrarContactoUsuario(ctx, solicitud.Contacto.SujetoRef(), solicitud.Contacto.Version(), claro)
		return e
	})
	if err != nil || !sobreValido(sobre, solicitud.Contacto.Version()) {
		return ports.PreparacionRegistroContactoUsuario{}, ErrContactoUsuarioNoDisponible
	}
	p := ports.PreparacionRegistroContactoUsuario{SujetoRef: solicitud.Contacto.SujetoRef(), VersionEsperada: solicitud.VersionEsperada, VersionNueva: solicitud.Contacto.Version(), Sobre: clonarSobre(sobre), Auditoria: clonarAuditoria(auditoria), Audiencia: solicitud.Audiencia, FinalidadRef: solicitud.FinalidadRef}
	cuerpo, err := cuerpoRegistroContactoUsuario(p)
	if err != nil {
		return ports.PreparacionRegistroContactoUsuario{}, ErrContactoUsuarioNoDisponible
	}
	p.Recurso, err = recursoContactoLigado(solicitud.Recurso, cuerpo, p.SujetoRef, p.FinalidadRef, p.VersionEsperada, p.VersionNueva)
	if err != nil {
		return ports.PreparacionRegistroContactoUsuario{}, ErrContactoUsuarioNoDisponible
	}
	payload, err := PayloadNegocioContactoUsuario(p)
	if err != nil {
		borrar(p.Sobre.Nonce)
		borrar(p.Sobre.Cifrado)
		return ports.PreparacionRegistroContactoUsuario{}, ErrContactoUsuarioNoDisponible
	}
	p.PayloadNegocio = payload
	return p, nil
}

func cuerpoRegistroContactoUsuario(p ports.PreparacionRegistroContactoUsuario) ([]byte, error) {
	doc := struct {
		Esquema, SujetoRef, Audiencia, FinalidadRef string
		VersionEsperada, VersionNueva               uint64
		Sobre                                       ports.SobreContactoUsuario
		Auditoria                                   domain.AuditEntry
	}{"vec.contacto_usuario.registro-cuerpo.v1", p.SujetoRef, p.Audiencia, p.FinalidadRef, p.VersionEsperada, p.VersionNueva, clonarSobre(p.Sobre), clonarAuditoria(p.Auditoria)}
	b, err := json.Marshal(doc)
	if err != nil || len(b) == 0 || len(b) > 65536 {
		return nil, ErrContactoUsuarioNoDisponible
	}
	return b, nil
}

func recursoContactoLigado(base domain.RecursoAutorizable, cuerpo []byte, sujeto, finalidad string, anterior, nueva uint64) (domain.RecursoAutorizable, error) {
	if base.Validar() != nil || len(cuerpo) == 0 || len(cuerpo) > 65536 {
		return domain.RecursoAutorizable{}, ErrContactoUsuarioNoDisponible
	}
	r := clonarRecurso(base)
	if r.Atributos == nil {
		r.Atributos = map[string]string{}
	}
	for _, k := range []string{"material_sha256", "contacto_sujeto_ref", "contacto_finalidad_ref", "contacto_version_esperada", "contacto_version", "auditoria_sha256"} {
		if _, existe := r.Atributos[k]; existe {
			return domain.RecursoAutorizable{}, ErrContactoUsuarioNoDisponible
		}
	}
	r.Atributos["contacto_sujeto_ref"] = sujeto
	r.Atributos["contacto_finalidad_ref"] = finalidad
	r.Atributos["contacto_version_esperada"] = strconv.FormatUint(anterior, 10)
	r.Atributos["contacto_version"] = strconv.FormatUint(nueva, 10)
	h := sha256.Sum256(cuerpo)
	r.Atributos["material_sha256"] = hex.EncodeToString(h[:])
	if r.Validar() != nil {
		return domain.RecursoAutorizable{}, ErrContactoUsuarioNoDisponible
	}
	return r, nil
}

// PayloadNegocioContactoUsuario es la preimagen V1 exacta de Preparar→Autorizar→Guardar.
func PayloadNegocioContactoUsuario(p ports.PreparacionRegistroContactoUsuario) ([]byte, error) {
	if !preparacionValida(p) {
		return nil, ErrContactoUsuarioNoDisponible
	}
	b, err := cuerpoRegistroContactoUsuario(p)
	if err != nil || !recursoCompromete(p.Recurso, b, p.SujetoRef, p.FinalidadRef, p.VersionEsperada, p.VersionNueva) {
		return nil, ErrContactoUsuarioNoDisponible
	}
	return b, nil
}

// La consulta compromete una versión exacta: nunca autoriza "la última".
func PayloadConsultaContactoUsuario(sujetoRef, finalidadRef string, version uint64, audiencia string) ([]byte, error) {
	if !domain.ReferenciaSujetoContactoUsuarioValida(sujetoRef) || !textoSeguro(finalidadRef) || version == 0 || !textoSeguro(audiencia) {
		return nil, ErrContactoUsuarioNoDisponible
	}
	return json.Marshal(struct {
		Esquema, SujetoRef, FinalidadRef, Audiencia string
		Version                                     uint64
	}{"vec.contacto_usuario.consulta.v1", sujetoRef, finalidadRef, audiencia, version})
}

// RecursoConsultaContactoUsuario completa el selector recibido de la autoridad
// gobernada. No escoge ni registra un módulo de identidad/contacto.
func RecursoConsultaContactoUsuario(base domain.RecursoAutorizable, sujeto, finalidad string, version uint64, audiencia string, auditoria domain.AuditEntry) (domain.RecursoAutorizable, []byte, error) {
	b, err := PayloadConsultaContactoUsuario(sujeto, finalidad, version, audiencia)
	if err != nil {
		return domain.RecursoAutorizable{}, nil, err
	}
	r, err := recursoContactoLigado(base, b, sujeto, finalidad, version, version)
	if err != nil {
		return domain.RecursoAutorizable{}, nil, err
	}
	h, err := HuellaAuditoriaPreparadaContactoUsuario(auditoria)
	if err != nil {
		return domain.RecursoAutorizable{}, nil, err
	}
	r.Atributos["auditoria_sha256"] = h
	if r.Validar() != nil {
		return domain.RecursoAutorizable{}, nil, ErrContactoUsuarioNoDisponible
	}
	return r, b, nil
}
func payloadConsultaValido(s ports.SolicitudAccesoContactoUsuario) bool {
	b, err := PayloadConsultaContactoUsuario(s.SujetoRef, s.FinalidadRef, s.Version, s.Audiencia)
	h, errA := HuellaAuditoriaPreparadaContactoUsuario(s.Auditoria)
	datos, errS := s.Solicitud.Datos()
	correlacion, errC := datos.Correlacion.ValorCanonico()
	return errA == nil && errS == nil && errC == nil && s.Recurso.Atributos["auditoria_sha256"] == h && correlacion == s.Auditoria.CorrelationRef && auditoriaPreparadaValidaPara(s.Auditoria, s.ContextoActor, AccionConsultarContactoUsuario, s.SujetoRef, s.Version, s.FinalidadRef, s.Recurso.ModuloID) && err == nil && bytes.Equal(b, s.PayloadNegocio) && recursoCompromete(s.Recurso, b, s.SujetoRef, s.FinalidadRef, s.Version, s.Version)
}
func recursoCompromete(r domain.RecursoAutorizable, b []byte, sujeto, finalidad string, anterior, nueva uint64) bool {
	h := sha256.Sum256(b)
	return r.Validar() == nil && r.Atributos["material_sha256"] == hex.EncodeToString(h[:]) && r.Atributos["contacto_sujeto_ref"] == sujeto && r.Atributos["contacto_finalidad_ref"] == finalidad && r.Atributos["contacto_version_esperada"] == strconv.FormatUint(anterior, 10) && r.Atributos["contacto_version"] == strconv.FormatUint(nueva, 10)
}

func (s *ServicioContactoUsuario) ConContactoUsuario(ctx context.Context, solicitud ports.SolicitudAccesoContactoUsuario, ejecutar func(domain.ContactoUsuario) error) error {
	if s == nil || ctx == nil || ctx.Err() != nil || nulo(s.lector) || ejecutar == nil || solicitud.ContextoActor.Validar() != nil || !domain.ReferenciaSujetoContactoUsuarioValida(solicitud.SujetoRef) || !textoSeguro(solicitud.FinalidadRef) || solicitud.Recurso.Validar() != nil || !textoSeguro(solicitud.Audiencia) || !payloadConsultaValido(solicitud) || !materialAccesoLigado(solicitud.Material, solicitud, s.ahora()) {
		return ErrContactoUsuarioNoDisponible
	}
	solicitud = clonarAccesoContacto(solicitud)
	var candado sync.Mutex
	abierto := true
	activo := false
	rechazado := false
	var errorCallback error
	err := s.lector.ConContactoUsuario(ctx, clonarAccesoContacto(solicitud), func(c domain.ContactoUsuario) error {
		candado.Lock()
		defer candado.Unlock()
		if !abierto || activo {
			rechazado = true
			return ErrContactoUsuarioNoDisponible
		}
		activo = true
		if ctx.Err() != nil || !payloadConsultaValido(solicitud) || !materialAccesoLigado(solicitud.Material, solicitud, s.ahora()) || c.Validar() != nil || c.SujetoRef() != solicitud.SujetoRef || c.Version() != solicitud.Version {
			errorCallback = ErrContactoUsuarioNoDisponible
			return errorCallback
		}
		errorCallback = ejecutar(c)
		return errorCallback
	})
	candado.Lock()
	abierto = false
	seEjecuto := activo
	huboRechazo := rechazado
	errorEjecutar := errorCallback
	candado.Unlock()
	if errorEjecutar != nil {
		return errorEjecutar
	}
	if err != nil || huboRechazo || !seEjecuto {
		return ErrContactoUsuarioNoDisponible
	}
	return nil
}

func preparacionValida(p ports.PreparacionRegistroContactoUsuario) bool {
	return domain.ReferenciaSujetoContactoUsuarioValida(p.SujetoRef) && p.VersionNueva > 0 && (p.VersionEsperada == 0 && p.VersionNueva == 1 || p.VersionEsperada > 0 && p.VersionEsperada != ^uint64(0) && p.VersionNueva == p.VersionEsperada+1) && sobreValido(p.Sobre, p.VersionNueva) && textoSeguro(p.Audiencia) && p.Recurso.Validar() == nil && textoSeguro(p.FinalidadRef) && p.Auditoria.Purpose == p.FinalidadRef && p.Auditoria.ModuleID == p.Recurso.ModuloID && p.Auditoria.ID == "" && p.Auditoria.Seq == 0 && p.Auditoria.Signature == ""
}
func sobreValido(s ports.SobreContactoUsuario, version uint64) bool {
	return s.Version == version && textoSeguro(s.ClaveRef) && len(s.Nonce) >= 12 && len(s.Nonce) <= 64 && len(s.Cifrado) >= 16 && len(s.Cifrado) <= 32768
}
func textoSeguro(s string) bool {
	return strings.TrimSpace(s) == s && len(s) > 0 && len(s) <= 512 && !strings.ContainsAny(s, "\r\n")
}
func borrar(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
func clonarSobre(s ports.SobreContactoUsuario) ports.SobreContactoUsuario {
	s.Nonce = append([]byte(nil), s.Nonce...)
	s.Cifrado = append([]byte(nil), s.Cifrado...)
	return s
}
func clonarPreparacion(p ports.PreparacionRegistroContactoUsuario) ports.PreparacionRegistroContactoUsuario {
	p.Sobre = clonarSobre(p.Sobre)
	p.Auditoria = clonarAuditoria(p.Auditoria)
	p.PayloadNegocio = append([]byte(nil), p.PayloadNegocio...)
	p.Recurso = clonarRecurso(p.Recurso)
	return p
}
func clonarRecurso(r domain.RecursoAutorizable) domain.RecursoAutorizable {
	r.Ambitos = clonarMapa(r.Ambitos)
	r.Atributos = clonarMapa(r.Atributos)
	return r
}
func clonarMapa(origen map[string]string) map[string]string {
	if origen == nil {
		return nil
	}
	destino := make(map[string]string, len(origen))
	for k, v := range origen {
		destino[k] = v
	}
	return destino
}
func clonarAuditoria(a domain.AuditEntry) domain.AuditEntry {
	a.ActorRoles = append([]string(nil), a.ActorRoles...)
	if a.Metadata != nil {
		m := make(map[string]string, len(a.Metadata))
		for k, v := range a.Metadata {
			m[k] = v
		}
		a.Metadata = m
	}
	return a
}
func nulo(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	switch x.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return x.IsNil()
	}
	return false
}

func auditoriaPreparadaValida(a domain.AuditEntry, actor domain.ContextoActor, accion string, c domain.ContactoUsuario, finalidad, modulo string) bool {
	return auditoriaPreparadaValidaPara(a, actor, accion, c.SujetoRef(), c.Version(), finalidad, modulo)
}
func auditoriaPreparadaValidaPara(a domain.AuditEntry, actor domain.ContextoActor, accion, sujeto string, version uint64, finalidad, modulo string) bool {
	return version > 0 && version <= 1<<53-1 && len(a.Metadata) == 0 && a.RepresentedSubjectID == "" && a.ExpedienteRef == "" && a.DocumentRef == "" && a.RuleRef == "" && a.Reason == "" && a.BeforeHash == "" && a.AfterHash == "" && a.ID == "" && a.Seq == 0 && a.Signature == "" && a.IntegrityAlgorithm == "" && a.PrevSignature == "" && actorHMACValido(a.ActorID) && a.ActorProfile == actor.PerfilActivoRef && len(a.ActorRoles) > 0 && a.AuthMethod == actor.Principal.AuthMethod && a.AuthAssurance == actor.Principal.AuthAssurance && a.AuthorizationRef == "" && a.Purpose == finalidad && a.Action == accion && a.ModuleID == modulo && a.SubjectRef == sujeto && a.ObjectVersion == int(version) && a.Result == "accepted" && a.CorrelationRef != "" && !a.OccurredAt.IsZero() && a.OccurredAt.Location() == time.UTC
}
func actorHMACValido(v string) bool {
	p := strings.Split(v, ":")
	if len(p) != 3 || p[0] != "hmac-sha256" || p[1] == "" || len(p[2]) != 64 {
		return false
	}
	_, e := hex.DecodeString(p[2])
	return e == nil
}
func contextoContactoValido(s domain.SolicitudAutorizacionLigadaV3, resultado domain.ResultadoContextoActorRegistradoV2, actor domain.ContextoActor, ahora time.Time) bool {
	datos, err := s.Datos()
	h, errActor := actor.HuellaSHA256VinculadaV2()
	return err == nil && errActor == nil && resultado.Validar() == nil && resultado.HuellaSHA256 == h && datos.VinculoAutenticacionActor.ValidarPara(resultado) == nil && datos.VinculoAutenticacionActor.VigenteEn(ahora, resultado)
}
func concesionContactoValida(m ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, s domain.SolicitudAutorizacionLigadaV3, d domain.DecisionAutorizacionLigadaV3, c ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, resultado domain.ResultadoContextoActorRegistradoV2, actor domain.ContextoActor, accion string, recurso domain.RecursoAutorizable, audiencia, finalidad string, ahora time.Time) bool {
	if m.ValidarEstructura() != nil || !contextoContactoValido(s, resultado, actor, ahora) || d.ValidarPara(s) != nil {
		return false
	}
	datos, err := s.Datos()
	if err != nil || datos.Accion != accion || datos.Finalidad != finalidad || !reflect.DeepEqual(datos.Recurso, recurso) {
		return false
	}
	orden, err := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(s, d, datos.ReferenciaMotivo, resultado)
	if err != nil || c.ValidarPara(orden) != nil {
		return false
	}
	cd, err := c.Datos()
	if err != nil || !c.DentroDeVentanaEn(cd.RegistradaEn) {
		return false
	}
	dc, errD := domain.RepresentacionCanonicaDecisionAutorizacionV3(d)
	mc, errM := domain.RepresentacionCanonicaMotivoAutorizacionV2(datos.ReferenciaMotivo)
	h, errR := recurso.HuellaContextoAutorizacionSHA256()
	hd := sha256.Sum256(dc)
	hm := sha256.Sum256(mc)
	r := m.ResumenCapacidad()
	proyeccion, err := domain.ParsearMensajeAtestacionAutorizacionV3NoAutoritativo(m.PayloadVECAD3())
	if err != nil {
		return false
	}
	cabecera, err := proyeccion.Cabecera()
	if err != nil {
		return false
	}
	mensaje, err := domain.SerializarMensajeAtestacionAutorizacionV3(cabecera, d, datos.ReferenciaMotivo, resultado)
	if err != nil || !bytes.Equal(mensaje, m.PayloadVECAD3()) {
		return false
	}
	return r.DecisionRef() == cd.DecisionRef && r.DecisionHuellaSHA256() == cd.DecisionHuellaSHA256 && errD == nil && errM == nil && errR == nil && bytes.Equal(dc, m.DecisionCanonica()) && bytes.Equal(mc, m.MotivoCanonico()) && bytes.Equal(resultado.RepresentacionCanonica, m.ContextoActorCanonico()) && r.DecisionHuellaSHA256() == hex.EncodeToString(hd[:]) && r.MotivoHuellaSHA256() == hex.EncodeToString(hm[:]) && r.Operacion() == accion && r.EfectoRef() == recurso.Referencia && r.EfectoHuellaSHA256() == h && r.AudienciaConsumo() == audiencia && r.ContextoRef() == resultado.RegistroContextoRef && r.ContextoHuellaSHA256() == resultado.HuellaSHA256 && m.PersonaVersion() == actor.Instantanea.PersonaVersion && m.PerfilVersion() == actor.Instantanea.PerfilVersion && !ahora.Before(r.EmitidaEn()) && ahora.Before(r.ExpiraEn())
}
func materialAccesoLigado(m ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, s ports.SolicitudAccesoContactoUsuario, ahora time.Time) bool {
	return concesionContactoValida(m, s.Solicitud, s.Decision, s.Confirmacion, s.ResultadoContexto, s.ContextoActor, AccionConsultarContactoUsuario, s.Recurso, s.Audiencia, s.FinalidadRef, ahora)
}

// proyeccionCentralContacto sólo se usa para cotejar campos; nunca se
// reserializa ni se convierte en una AuditEntry con firma trasladada.
type proyeccionCentralContacto struct {
	ID                   string            `json:"id"`
	Seq                  uint64            `json:"seq"`
	IntegrityAlgorithm   string            `json:"integrity_algorithm"`
	PrevSignature        string            `json:"prev_signature"`
	Signature            string            `json:"signature"`
	ActorID              string            `json:"actor_id"`
	ActorProfile         string            `json:"actor_profile"`
	ActorRoles           []string          `json:"actor_roles"`
	RepresentedSubjectID string            `json:"represented_subject_id"`
	AuthMethod           string            `json:"auth_method"`
	AuthAssurance        string            `json:"auth_assurance"`
	AuthorizationRef     string            `json:"authorization_ref"`
	Purpose              string            `json:"purpose"`
	Action               string            `json:"action"`
	ModuleID             string            `json:"module_id"`
	SubjectRef           string            `json:"subject_ref"`
	ObjectVersion        uint64            `json:"object_version"`
	ExpedienteRef        string            `json:"expediente_ref"`
	DocumentRef          string            `json:"document_ref"`
	RuleRef              string            `json:"rule_ref"`
	Reason               string            `json:"reason"`
	Result               string            `json:"result"`
	BeforeHash           string            `json:"before_hash"`
	AfterHash            string            `json:"after_hash"`
	CorrelationRef       string            `json:"correlation_ref"`
	Metadata             map[string]string `json:"metadata"`
	OccurredAt           string            `json:"occurred_at"`
}

func reciboValido(r ports.ReciboContactoUsuario, p ports.PreparacionRegistroContactoUsuario, decisionRef string) bool {
	negocio, err := PayloadNegocioContactoUsuario(p)
	if err != nil || !bytes.Equal(negocio, p.PayloadNegocio) {
		return false
	}
	return evidenciaCentralEsperadaContacto(r, p.SujetoRef, p.VersionNueva, p.Auditoria, p.PayloadNegocio, p.Recurso, decisionRef)
}

// ValidarEvidenciaCentralRegistroContactoUsuario coteja el recibo original
// ANTES del COMMIT del adaptador; no acredita por sí mismo I/O ni cadena.
func ValidarEvidenciaCentralRegistroContactoUsuario(raw []byte, p ports.PreparacionRegistroContactoUsuario, decisionRef, consumoRef, consumoHuella string) (ports.EvidenciaAuditoriaCentralContactoUsuario, error) {
	e, err := envolverEvidenciaCentralContacto(raw)
	if err != nil {
		return ports.EvidenciaAuditoriaCentralContactoUsuario{}, err
	}
	r := ports.ReciboContactoUsuario{SujetoRef: p.SujetoRef, Version: p.VersionNueva, ConsumoRef: consumoRef, ConsumoHuellaSHA256: consumoHuella, EvidenciaCentral: e}
	if !reciboValido(r, p, decisionRef) {
		return ports.EvidenciaAuditoriaCentralContactoUsuario{}, ErrContactoUsuarioNoDisponible
	}
	return e, nil
}

// ValidarEvidenciaCentralConsultaContactoUsuario exige el resultado auditado
// del selector exacto antes de confirmar la lectura y abrir el sobre. La
// revocación fresca posterior al descifrado sigue perteneciendo al adaptador.
func ValidarEvidenciaCentralConsultaContactoUsuario(raw []byte, s ports.SolicitudAccesoContactoUsuario, consumoRef, consumoHuella string) (ports.EvidenciaAuditoriaCentralContactoUsuario, error) {
	if s.Material.ValidarEstructura() != nil || !payloadConsultaValido(s) {
		return ports.EvidenciaAuditoriaCentralContactoUsuario{}, ErrContactoUsuarioNoDisponible
	}
	e, err := envolverEvidenciaCentralContacto(raw)
	if err != nil {
		return ports.EvidenciaAuditoriaCentralContactoUsuario{}, err
	}
	r := ports.ReciboContactoUsuario{SujetoRef: s.SujetoRef, Version: s.Version, ConsumoRef: consumoRef, ConsumoHuellaSHA256: consumoHuella, EvidenciaCentral: e}
	if !evidenciaCentralEsperadaContacto(r, s.SujetoRef, s.Version, s.Auditoria, s.PayloadNegocio, s.Recurso, s.Material.ResumenCapacidad().DecisionRef()) {
		return ports.EvidenciaAuditoriaCentralContactoUsuario{}, ErrContactoUsuarioNoDisponible
	}
	return e, nil
}
func envolverEvidenciaCentralContacto(raw []byte) (ports.EvidenciaAuditoriaCentralContactoUsuario, error) {
	if len(raw) < 2 || len(raw) > 16384 {
		return ports.EvidenciaAuditoriaCentralContactoUsuario{}, ErrContactoUsuarioNoDisponible
	}
	campos, err := objetoCentralContacto(raw)
	var ref string
	if err != nil || json.Unmarshal(campos["id"], &ref) != nil {
		return ports.EvidenciaAuditoriaCentralContactoUsuario{}, ErrContactoUsuarioNoDisponible
	}
	h := sha256.Sum256(raw)
	return ports.EvidenciaAuditoriaCentralContactoUsuario{JSONOriginal: bytes.Clone(raw), Referencia: ref, HuellaJSONSHA256: hex.EncodeToString(h[:])}, nil
}
func evidenciaCentralEsperadaContacto(r ports.ReciboContactoUsuario, sujeto string, version uint64, a domain.AuditEntry, negocio []byte, recursoEsperado domain.RecursoAutorizable, decisionRef string) bool {
	if r.SujetoRef != sujeto || r.Version != version || !textoSeguro(decisionRef) || !hexContacto(r.ConsumoHuellaSHA256, 64) || r.ConsumoHuellaSHA256 == strings.Repeat("0", 64) || r.ConsumoRef != "aud_v3_"+r.ConsumoHuellaSHA256[:32] {
		return false
	}
	original := r.EvidenciaCentral.JSONOriginal
	if len(original) < 2 || len(original) > 16384 || !hexContacto(r.EvidenciaCentral.HuellaJSONSHA256, 64) {
		return false
	}
	h := sha256.Sum256(original)
	if hex.EncodeToString(h[:]) != r.EvidenciaCentral.HuellaJSONSHA256 {
		return false
	}
	campos, err := objetoCentralContacto(original)
	tipo := reflect.TypeOf(proyeccionCentralContacto{})
	if err != nil || len(campos) != tipo.NumField() {
		return false
	}
	for i := 0; i < tipo.NumField(); i++ {
		if _, ok := campos[tipo.Field(i).Tag.Get("json")]; !ok {
			return false
		}
	}
	var c proyeccionCentralContacto
	decoder := json.NewDecoder(bytes.NewReader(original))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&c) != nil {
		return false
	}
	metadata, err := objetoCentralContacto(campos["metadata"])
	if err != nil || len(metadata) != 4 {
		return false
	}
	for _, v := range campos {
		if bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return false
		}
	}
	for _, v := range metadata {
		if bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return false
		}
	}
	if !strings.HasPrefix(c.ID, "acc_") || !hexContacto(strings.TrimPrefix(c.ID, "acc_"), 40) || c.ID != r.EvidenciaCentral.Referencia || c.Seq == 0 || c.Seq > 1<<53-1 || c.IntegrityAlgorithm != "sha256-chain-v1" || !hexContacto(c.Signature, 64) || !hexContacto(c.PrevSignature, 64) {
		return false
	}
	material := sha256.Sum256(negocio)
	recurso, err := recursoEsperado.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return false
	}
	// Se coteja la semántica de la preparación con las transformaciones exactas
	// que T13/6 realiza antes de encadenar. No se retoca la evidencia recibida.
	return a.AuthorizationRef == "" && a.Result == "accepted" &&
		c.ActorID == a.ActorID && c.ActorProfile == a.ActorProfile && reflect.DeepEqual(c.ActorRoles, a.ActorRoles) &&
		c.RepresentedSubjectID == "" && c.AuthMethod == string(a.AuthMethod) && c.AuthAssurance == string(a.AuthAssurance) &&
		c.AuthorizationRef == decisionRef && c.Purpose == a.Purpose && c.Action == a.Action && c.ModuleID == a.ModuleID &&
		c.SubjectRef == sujeto && c.ObjectVersion == version && c.ExpedienteRef == "" && c.DocumentRef == "" && c.RuleRef == "" && c.Reason == "" &&
		c.Result == "permitido" && c.BeforeHash == "" && c.AfterHash == hex.EncodeToString(material[:]) && c.CorrelationRef == a.CorrelationRef &&
		c.OccurredAt == a.OccurredAt.UTC().Format("2006-01-02T15:04:05.000000Z") &&
		c.Metadata["consumo_ref"] == r.ConsumoRef && c.Metadata["consumo_huella_sha256"] == r.ConsumoHuellaSHA256 &&
		c.Metadata["material_sha256"] == c.AfterHash && c.Metadata["contexto_recurso_sha256"] == recurso
}

// objetoCentralContacto rechaza claves duplicadas, objetos incompletos,
// valores exteriores y entradas sobrantes sin normalizar los bytes originales.
func objetoCentralContacto(b []byte) (map[string]json.RawMessage, error) {
	d := json.NewDecoder(bytes.NewReader(b))
	t, err := d.Token()
	if err != nil || t != json.Delim('{') {
		return nil, ErrContactoUsuarioNoDisponible
	}
	campos := make(map[string]json.RawMessage)
	for d.More() {
		t, err := d.Token()
		if err != nil {
			return nil, ErrContactoUsuarioNoDisponible
		}
		k, ok := t.(string)
		if !ok {
			return nil, ErrContactoUsuarioNoDisponible
		}
		if _, existe := campos[k]; existe {
			return nil, ErrContactoUsuarioNoDisponible
		}
		var v json.RawMessage
		if d.Decode(&v) != nil {
			return nil, ErrContactoUsuarioNoDisponible
		}
		campos[k] = v
	}
	t, err = d.Token()
	if err != nil || t != json.Delim('}') {
		return nil, ErrContactoUsuarioNoDisponible
	}
	if _, err = d.Token(); err != io.EOF {
		return nil, ErrContactoUsuarioNoDisponible
	}
	return campos, nil
}
func hexContacto(s string, n int) bool {
	if len(s) != n {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func clonarAccesoContacto(s ports.SolicitudAccesoContactoUsuario) ports.SolicitudAccesoContactoUsuario {
	s.Recurso = clonarRecurso(s.Recurso)
	s.Auditoria = clonarAuditoria(s.Auditoria)
	s.PayloadNegocio = bytes.Clone(s.PayloadNegocio)
	s.ResultadoContexto, _ = s.ResultadoContexto.Clonar()
	return s
}

// HuellaAuditoriaPreparadaContactoUsuario liga la auditoría de lectura sin
// introducirla en el selector de negocio ni anticipar DecisionRef. La frontera
// de lectura valida además actor, acción, sujeto, finalidad y correlación.
func HuellaAuditoriaPreparadaContactoUsuario(a domain.AuditEntry) (string, error) {
	if a.AuthorizationRef != "" || a.ID != "" || a.Seq != 0 || a.Signature != "" {
		return "", ErrContactoUsuarioNoDisponible
	}
	b, err := json.Marshal(a)
	if err != nil || len(b) > 65536 {
		return "", ErrContactoUsuarioNoDisponible
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
