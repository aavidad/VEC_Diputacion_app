package application

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	FinalidadConsultaReciboContactoUsuario = "gestion_contacto_propio"
	AudienciaConsultaReciboContactoUsuario = "vec.contacto_usuario.recibo.v1"
)

type ServicioConsultaReciboContactoUsuario struct {
	auditoria   ports.PreparadorAuditoriaContactoUsuario
	autorizador ports.AutorizadorContactoUsuario
	consultor   ports.ConsultorReciboContactoUsuario
	ahora       func() time.Time
}

func NuevoServicioConsultaReciboContactoUsuario(a ports.PreparadorAuditoriaContactoUsuario, v ports.AutorizadorContactoUsuario, c ports.ConsultorReciboContactoUsuario) (*ServicioConsultaReciboContactoUsuario, error) {
	if nulo(a) || nulo(v) || nulo(c) {
		return nil, ErrContactoUsuarioNoDisponible
	}
	return &ServicioConsultaReciboContactoUsuario{a, v, c, func() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }}, nil
}

func (s *ServicioConsultaReciboContactoUsuario) Consultar(ctx context.Context, peticion ports.SolicitudConsultaReciboContactoUsuario) (ports.ResultadoConsultaReciboContactoUsuario, error) {
	vacio := ports.ResultadoConsultaReciboContactoUsuario{}
	if s == nil || ctx == nil || ctx.Err() != nil || nulo(s.auditoria) || nulo(s.autorizador) || nulo(s.consultor) || peticion.ContextoActor.Validar() != nil || peticion.Version == 0 || peticion.Version > 1<<53-1 || peticion.Recurso.Validar() != nil || peticion.Recurso.Referencia != peticion.ContextoActor.PersonaRef {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	sujeto := peticion.ContextoActor.PersonaRef
	a, err := s.auditoria.PrepararAuditoriaContactoUsuario(ctx, peticion.ContextoActor, AccionConsultarContactoUsuario, peticion.Recurso.ModuloID, sujeto, peticion.Version)
	if err != nil || !auditoriaPreparadaValidaPara(a, peticion.ContextoActor, AccionConsultarContactoUsuario, sujeto, peticion.Version, FinalidadConsultaReciboContactoUsuario, peticion.Recurso.ModuloID) {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	negocio, err := PayloadConsultaReciboContactoUsuario(sujeto, peticion.Version)
	if err != nil {
		return vacio, err
	}
	recurso, err := recursoContactoLigado(peticion.Recurso, negocio, sujeto, FinalidadConsultaReciboContactoUsuario, peticion.Version, peticion.Version)
	if err != nil {
		return vacio, err
	}
	huella, err := HuellaAuditoriaPreparadaContactoUsuario(a)
	if err != nil {
		return vacio, err
	}
	recurso.Atributos["auditoria_sha256"] = huella
	base := peticion.SolicitudBase
	correlacion, err := base.Correlacion.ValorCanonico()
	if err != nil || correlacion != a.CorrelationRef || base.Accion != AccionConsultarContactoUsuario || base.Finalidad != FinalidadConsultaReciboContactoUsuario || !reflect.DeepEqual(base.Recurso, peticion.Recurso) {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	base.Recurso = recurso
	nominal, err := domain.NuevaSolicitudAutorizacionLigadaV3(base)
	if err != nil || !contextoContactoValido(nominal, peticion.ResultadoContexto, peticion.ContextoActor, s.ahora()) {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, nominal, peticion.ResultadoContexto)
	if err != nil || nulo(exportador) || ctx.Err() != nil {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	acceso := ports.SolicitudAccesoContactoUsuario{Auditoria: a, SujetoRef: sujeto, ContextoActor: peticion.ContextoActor, FinalidadRef: FinalidadConsultaReciboContactoUsuario, Recurso: recurso, Audiencia: AudienciaConsultaReciboContactoUsuario, PayloadNegocio: negocio, Material: material, Version: peticion.Version, Solicitud: nominal, Decision: decision, Confirmacion: confirmacion, ResultadoContexto: peticion.ResultadoContexto}
	orden := ports.OrdenConsultaReciboContactoUsuario{Acceso: acceso}
	if ValidarOrdenConsultaReciboContactoUsuario(orden, s.ahora()) != nil {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	resultado, err := s.consultor.ConsultarReciboContactoUsuario(ctx, clonarOrdenConsultaRecibo(orden))
	if err != nil || ctx.Err() != nil || ValidarOrdenConsultaReciboContactoUsuario(orden, s.ahora()) != nil || ValidarResultadoConsultaReciboContactoUsuario(orden, resultado) != nil {
		return vacio, ErrContactoUsuarioNoDisponible
	}
	resultado.ReciboOriginal.EvidenciaCentral.JSONOriginal = bytes.Clone(resultado.ReciboOriginal.EvidenciaCentral.JSONOriginal)
	resultado.AuditoriaConsulta.JSONOriginal = bytes.Clone(resultado.AuditoriaConsulta.JSONOriginal)
	return resultado, nil
}

func PayloadConsultaReciboContactoUsuario(sujeto string, version uint64) ([]byte, error) {
	if !domain.ReferenciaSujetoContactoUsuarioValida(sujeto) || version == 0 || version > 1<<53-1 {
		return nil, ErrContactoUsuarioNoDisponible
	}
	return json.Marshal(struct {
		Esquema, SujetoRef, FinalidadRef, Audiencia string
		Version                                     uint64
	}{AudienciaConsultaReciboContactoUsuario, sujeto, FinalidadConsultaReciboContactoUsuario, AudienciaConsultaReciboContactoUsuario, version})
}

// ValidarOrdenConsultaReciboContactoUsuario no convierte esta operación en
// lectura de correo: exige su propio esquema/audiencia y el sujeto del actor.
func ValidarOrdenConsultaReciboContactoUsuario(o ports.OrdenConsultaReciboContactoUsuario, ahora time.Time) error {
	a := o.Acceso
	b, err := PayloadConsultaReciboContactoUsuario(a.SujetoRef, a.Version)
	h, errA := HuellaAuditoriaPreparadaContactoUsuario(a.Auditoria)
	base, errS := a.Solicitud.Datos()
	correlacion, errC := base.Correlacion.ValorCanonico()
	if err != nil || errA != nil || errS != nil || errC != nil || a.SujetoRef != a.ContextoActor.PersonaRef || a.Recurso.Referencia != a.SujetoRef || a.FinalidadRef != FinalidadConsultaReciboContactoUsuario || a.Audiencia != AudienciaConsultaReciboContactoUsuario || !bytes.Equal(b, a.PayloadNegocio) || a.Recurso.Atributos["auditoria_sha256"] != h || correlacion != a.Auditoria.CorrelationRef || !auditoriaPreparadaValidaPara(a.Auditoria, a.ContextoActor, AccionConsultarContactoUsuario, a.SujetoRef, a.Version, a.FinalidadRef, a.Recurso.ModuloID) || !recursoCompromete(a.Recurso, b, a.SujetoRef, a.FinalidadRef, a.Version, a.Version) || !concesionContactoValida(a.Material, a.Solicitud, a.Decision, a.Confirmacion, a.ResultadoContexto, a.ContextoActor, AccionConsultarContactoUsuario, a.Recurso, a.Audiencia, a.FinalidadRef, ahora) {
		return ErrContactoUsuarioNoDisponible
	}
	return nil
}

// DecodificarResultadoConsultaReciboContactoUsuario preserva las dos evidencias
// originales. El adaptador lo llama ANTES del COMMIT, sin reserializar firmas.
func DecodificarResultadoConsultaReciboContactoUsuario(o ports.OrdenConsultaReciboContactoUsuario, encontrado bool, sujeto string, version uint64, original []byte, consumoOriginalRef, consumoOriginalHuella string, consulta []byte, consumoConsultaRef, consumoConsultaHuella string) (ports.ResultadoConsultaReciboContactoUsuario, error) {
	r := ports.ResultadoConsultaReciboContactoUsuario{Encontrado: encontrado, SujetoRef: sujeto, Version: version, ConsumoConsultaRef: consumoConsultaRef, ConsumoConsultaHuellaSHA256: consumoConsultaHuella}
	e, err := envolverEvidenciaCentralContacto(consulta)
	if err != nil {
		return ports.ResultadoConsultaReciboContactoUsuario{}, err
	}
	r.AuditoriaConsulta = e
	if encontrado {
		e, err = envolverEvidenciaCentralContacto(original)
		if err != nil {
			return ports.ResultadoConsultaReciboContactoUsuario{}, err
		}
		r.ReciboOriginal = ports.ReciboContactoUsuario{SujetoRef: sujeto, Version: version, ConsumoRef: consumoOriginalRef, ConsumoHuellaSHA256: consumoOriginalHuella, EvidenciaCentral: e}
	} else if len(original) != 0 || consumoOriginalRef != "" || consumoOriginalHuella != "" {
		return ports.ResultadoConsultaReciboContactoUsuario{}, ErrContactoUsuarioNoDisponible
	}
	if err = ValidarResultadoConsultaReciboContactoUsuario(o, r); err != nil {
		return ports.ResultadoConsultaReciboContactoUsuario{}, err
	}
	return r, nil
}

func ValidarResultadoConsultaReciboContactoUsuario(o ports.OrdenConsultaReciboContactoUsuario, r ports.ResultadoConsultaReciboContactoUsuario) error {
	a := o.Acceso
	if r.SujetoRef != a.SujetoRef || r.Version != a.Version {
		return ErrContactoUsuarioNoDisponible
	}
	ref, huella := "", ""
	if r.Encontrado {
		if !reciboOriginalContactoValido(r.ReciboOriginal, a.SujetoRef, a.Version, a.Recurso.ModuloID, a.Auditoria.CorrelationRef) {
			return ErrContactoUsuarioNoDisponible
		}
		ref, huella = r.ReciboOriginal.EvidenciaCentral.Referencia, r.ReciboOriginal.EvidenciaCentral.HuellaJSONSHA256
	} else if !reflect.DeepEqual(r.ReciboOriginal, ports.ReciboContactoUsuario{}) {
		return ErrContactoUsuarioNoDisponible
	}
	esperado := ports.ReciboContactoUsuario{SujetoRef: r.SujetoRef, Version: r.Version, ConsumoRef: r.ConsumoConsultaRef, ConsumoHuellaSHA256: r.ConsumoConsultaHuellaSHA256, EvidenciaCentral: r.AuditoriaConsulta}
	extras := map[string]string{"recibo_encontrado": strconv.FormatBool(r.Encontrado), "recibo_original_ref": ref, "recibo_original_sha256": huella}
	if !evidenciaCentralEsperadaContactoConMetadata(esperado, a.SujetoRef, a.Version, a.Auditoria, a.PayloadNegocio, a.Recurso, a.Material.ResumenCapacidad().DecisionRef(), extras) {
		return ErrContactoUsuarioNoDisponible
	}
	return nil
}

// El original no se coteja contra la decisión nueva. La autoridad SQL lo
// obtiene de la versión durable, y el acceso actual compromete sus bytes/id.
// Sin registro canónico/ancla central esto NO verifica la cadena criptográfica.
func reciboOriginalContactoValido(r ports.ReciboContactoUsuario, sujeto string, version uint64, modulo, correlacionConsulta string) bool {
	if r.SujetoRef != sujeto || r.Version != version || !hexContacto(r.ConsumoHuellaSHA256, 64) || r.ConsumoHuellaSHA256 == strings.Repeat("0", 64) || r.ConsumoRef != "aud_v3_"+r.ConsumoHuellaSHA256[:32] {
		return false
	}
	e, err := envolverEvidenciaCentralContacto(r.EvidenciaCentral.JSONOriginal)
	if err != nil || e.Referencia != r.EvidenciaCentral.Referencia || e.HuellaJSONSHA256 != r.EvidenciaCentral.HuellaJSONSHA256 {
		return false
	}
	campos, err := objetoCentralContacto(e.JSONOriginal)
	tipo := reflect.TypeOf(proyeccionCentralContacto{})
	if err != nil || len(campos) != tipo.NumField() {
		return false
	}
	for i := 0; i < tipo.NumField(); i++ {
		v, ok := campos[tipo.Field(i).Tag.Get("json")]
		if !ok || bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return false
		}
	}
	var c proyeccionCentralContacto
	if json.Unmarshal(e.JSONOriginal, &c) != nil {
		return false
	}
	metadata, err := objetoCentralContacto(campos["metadata"])
	if err != nil || len(metadata) != 4 {
		return false
	}
	for _, clave := range []string{"consumo_ref", "consumo_huella_sha256", "material_sha256", "contexto_recurso_sha256"} {
		v, ok := metadata[clave]
		if !ok || bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return false
		}
	}
	if len(c.ActorRoles) == 0 {
		return false
	}
	for _, rol := range c.ActorRoles {
		if !textoSeguro(rol) {
			return false
		}
	}
	instante, err := time.Parse("2006-01-02T15:04:05.000000Z", c.OccurredAt)
	accion := AccionActualizarContactoUsuario
	if version == 1 {
		accion = AccionAltaContactoUsuario
	}
	return err == nil && instante.Format("2006-01-02T15:04:05.000000Z") == c.OccurredAt && strings.HasPrefix(c.ID, "acc_") && hexContacto(strings.TrimPrefix(c.ID, "acc_"), 40) && c.Seq > 0 && c.Seq <= 1<<53-1 && c.IntegrityAlgorithm == "sha256-chain-v1" && hexContacto(c.Signature, 64) && hexContacto(c.PrevSignature, 64) && actorHMACValido(c.ActorID) && textoSeguro(c.ActorProfile) && c.RepresentedSubjectID == "" && (c.AuthMethod == string(domain.AuthMethodCertificate) || c.AuthMethod == string(domain.AuthMethodDNIe)) && c.AuthAssurance == string(domain.AuthAssuranceHigh) && textoSeguro(c.AuthorizationRef) && c.Purpose == FinalidadConsultaReciboContactoUsuario && c.Action == accion && c.ModuleID == modulo && c.SubjectRef == sujeto && c.ObjectVersion == version && c.ExpedienteRef == "" && c.DocumentRef == "" && c.RuleRef == "" && c.Reason == "" && c.Result == "permitido" && c.BeforeHash == "" && hexContacto(c.AfterHash, 64) && textoSeguro(c.CorrelationRef) && c.CorrelationRef != correlacionConsulta && c.Metadata["consumo_ref"] == r.ConsumoRef && c.Metadata["consumo_huella_sha256"] == r.ConsumoHuellaSHA256 && c.Metadata["material_sha256"] == c.AfterHash && hexContacto(c.Metadata["contexto_recurso_sha256"], 64)
}

func clonarOrdenConsultaRecibo(o ports.OrdenConsultaReciboContactoUsuario) ports.OrdenConsultaReciboContactoUsuario {
	o.Acceso = clonarAccesoContacto(o.Acceso)
	o.Acceso.ContextoActor, _ = o.Acceso.ContextoActor.Clonar()
	return o
}
