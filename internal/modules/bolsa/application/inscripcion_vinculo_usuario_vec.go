package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var ErrInscripcionPropiaUsuarioVECInvalida = errors.New("bolsa: inscripcion propia VEC invalida")

type ServicioInscripcionPropiaUsuarioVEC struct {
	revalidador   dominiovec.RevalidadorAutenticacionActorV1
	contextos     dominiovec.ResolutorContextoActorRegistradoV2
	reloj         dominiovec.RelojVinculoAutenticacionActorV2
	politicas     puertosbolsa.ResolverPoliticaInscripcionPropiaUsuarioVEC
	reservas      puertosbolsa.ReservadorInscripcionPropiaUsuarioVEC
	auditoria     puertosbolsa.PreparadorAuditoriaInscripcionPropiaUsuarioVEC
	autorizador   puertosbolsa.AutorizadorInscripcionUsuarioVEC
	registro      puertosbolsa.RegistradorInscripcionPropiaUsuarioVEC
	correlaciones puertosvec.GeneradorReferenciasAutorizacionV2
}

func NuevoServicioInscripcionPropiaUsuarioVEC(r dominiovec.RevalidadorAutenticacionActorV1, c dominiovec.ResolutorContextoActorRegistradoV2, reloj dominiovec.RelojVinculoAutenticacionActorV2, p puertosbolsa.ResolverPoliticaInscripcionPropiaUsuarioVEC, reservas puertosbolsa.ReservadorInscripcionPropiaUsuarioVEC, a puertosbolsa.PreparadorAuditoriaInscripcionPropiaUsuarioVEC, z puertosbolsa.AutorizadorInscripcionUsuarioVEC, registro puertosbolsa.RegistradorInscripcionPropiaUsuarioVEC, correlaciones puertosvec.GeneradorReferenciasAutorizacionV2) (*ServicioInscripcionPropiaUsuarioVEC, error) {
	if nuloInscripcion(r) || nuloInscripcion(c) || nuloInscripcion(reloj) || nuloInscripcion(p) || nuloInscripcion(reservas) || nuloInscripcion(a) || nuloInscripcion(z) || nuloInscripcion(registro) || nuloInscripcion(correlaciones) {
		return nil, ErrInscripcionPropiaUsuarioVECInvalida
	}
	return &ServicioInscripcionPropiaUsuarioVEC{revalidador: r, contextos: c, reloj: reloj, politicas: p, reservas: reservas, auditoria: a, autorizador: z, registro: registro, correlaciones: correlaciones}, nil
}

func (s *ServicioInscripcionPropiaUsuarioVEC) Registrar(ctx context.Context, entrada puertosbolsa.SolicitudRegistrarInscripcionPropiaUsuarioVEC) (puertosbolsa.ReciboInscripcionPropiaUsuarioVEC, error) {
	vacio := puertosbolsa.ReciboInscripcionPropiaUsuarioVEC{}
	if s == nil || nuloInscripcion(s.revalidador) || nuloInscripcion(s.contextos) || nuloInscripcion(s.reloj) || nuloInscripcion(s.politicas) || nuloInscripcion(s.reservas) || nuloInscripcion(s.auditoria) || nuloInscripcion(s.autorizador) || nuloInscripcion(s.registro) || nuloInscripcion(s.correlaciones) || ctx == nil || ctx.Err() != nil || !solicitudInscripcionValida(entrada) {
		return vacio, ErrInscripcionPropiaUsuarioVECInvalida
	}
	auth := dominiovec.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: entrada.AutenticacionRef, SesionRef: entrada.SesionRef}
	acreditada, err := s.revalidador.RevalidarAutenticacionActorV1(ctx, auth)
	if err != nil || ctx.Err() != nil || acreditada.Validar() != nil {
		return vacio, ErrInscripcionPropiaUsuarioVECInvalida
	}
	contexto := dominiovec.SolicitudContextoActor{Cuenta: dominiovec.CuentaAutenticadaContextoActor{CuentaRef: acreditada.CuentaRef, Metodo: acreditada.MetodoObservado, Garantia: acreditada.GarantiaObservada}, PerfilActivoRef: entrada.PerfilActivoRef}
	vinculo, resultado, err := dominiovec.CrearVinculoAutenticacionActorV2ConResultado(ctx, s.revalidador, auth, s.contextos, contexto, s.reloj)
	ahora := s.reloj.Ahora()
	if err != nil || ctx.Err() != nil || resultado.Validar() != nil || vinculo.ValidarPara(resultado) != nil || !vinculo.VigenteEn(ahora, resultado) {
		return vacio, ErrInscripcionPropiaUsuarioVECInvalida
	}
	candidato, ok := candidatoActivo(resultado, ahora)
	if !ok {
		return vacio, ErrInscripcionPropiaUsuarioVECInvalida
	}
	politica, err := s.politicas.ResolverPoliticaInscripcionPropiaUsuarioVEC(ctx, entrada.ConvocatoriaRef, ahora)
	politica.RecursoBase = clonarRecursoInscripcion(politica.RecursoBase)
	if err != nil || ctx.Err() != nil || !politicaValida(politica, ahora) {
		return vacio, ErrInscripcionPropiaUsuarioVECInvalida
	}
	reserva, err := s.reservas.ReservarInscripcionPropiaUsuarioVEC(ctx, entrada.IntencionRef, resultado.Contexto.PersonaRef, entrada.ConvocatoriaRef)
	if err != nil || ctx.Err() != nil || reserva.IntencionRef != entrada.IntencionRef || reserva.PersonaRef != resultado.Contexto.PersonaRef || reserva.ConvocatoriaRef != entrada.ConvocatoriaRef || !puertosbolsa.ReferenciaOpacaLlamamientoValida(reserva.InscripcionRef) || !puertosbolsa.ReferenciaOpacaLlamamientoValida(reserva.SujetoRef) {
		return vacio, ErrInscripcionPropiaUsuarioVECInvalida
	}
	recurso, payload, err := recursoInscripcion(politica, reserva, resultado, candidato, entrada)
	if err != nil || ctx.Err() != nil {
		return vacio, ErrInscripcionPropiaUsuarioVECInvalida
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, s.correlaciones)
	if err != nil || ctx.Err() != nil {
		return vacio, ErrInscripcionPropiaUsuarioVECInvalida
	}
	audit, err := s.auditoria.PrepararAuditoriaInscripcionPropiaUsuarioVEC(ctx, clonarResultadoInscripcion(resultado), clonarRecursoInscripcion(recurso), correlacion, politica.Finalidad)
	if err != nil || ctx.Err() != nil || !auditoriaInscripcionValida(audit, resultado, recurso, correlacion, politica.Finalidad) {
		return vacio, ErrInscripcionPropiaUsuarioVECInvalida
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: vinculo, ReferenciaMotivo: politica.Motivo, Accion: puertosbolsa.AccionRegistrarInscripcionPropiaUsuarioVEC, Recurso: recurso, Finalidad: politica.Finalidad, Correlacion: correlacion})
	if err != nil {
		return vacio, ErrInscripcionPropiaUsuarioVECInvalida
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, clonarResultadoInscripcion(resultado))
	if err != nil || ctx.Err() != nil || nuloInscripcion(exportador) {
		return vacio, ErrInscripcionPropiaUsuarioVECInvalida
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || ctx.Err() != nil || !concesionInscripcionValida(solicitud, decision, confirmacion, material, resultado, recurso, politica, s.reloj.Ahora()) {
		return vacio, ErrInscripcionPropiaUsuarioVECInvalida
	}
	confirmada, err := confirmacion.Datos()
	if err != nil || confirmada.DecisionRef == "" {
		return vacio, ErrInscripcionPropiaUsuarioVECInvalida
	}
	audit.AuthorizationRef = confirmada.DecisionRef
	ahora = s.reloj.Ahora()
	politicaFresca, err := s.politicas.ResolverPoliticaInscripcionPropiaUsuarioVEC(ctx, entrada.ConvocatoriaRef, ahora)
	politicaFresca.RecursoBase = clonarRecursoInscripcion(politicaFresca.RecursoBase)
	ahora = s.reloj.Ahora()
	candidatoFresco, candidatoVigente := candidatoActivo(resultado, ahora)
	if err != nil || ctx.Err() != nil || !candidatoVigente || candidatoFresco.Referencia != candidato.Referencia || candidatoFresco.VinculoRef != candidato.VinculoRef || candidatoFresco.Version != candidato.Version || !politicaValida(politicaFresca, ahora) || !reflect.DeepEqual(politica, politicaFresca) || !vinculoYResultadoVigentes(solicitud, resultado, ahora) || !concesionInscripcionValida(solicitud, decision, confirmacion, material, resultado, recurso, politicaFresca, ahora) {
		return vacio, ErrInscripcionPropiaUsuarioVECInvalida
	}
	orden := puertosbolsa.OrdenRegistroInscripcionPropiaUsuarioVEC{InscripcionRef: reserva.InscripcionRef, SujetoRef: reserva.SujetoRef, PersonaRef: resultado.Contexto.PersonaRef, CandidatoRef: candidato.Referencia, VinculoCandidatoRef: candidato.VinculoRef, VersionVinculoCandidato: candidato.Version, VersionPersona: resultado.Contexto.Instantanea.PersonaVersion, ConvocatoriaRef: entrada.ConvocatoriaRef, IntencionRef: entrada.IntencionRef, PoliticaRef: politica.Referencia, PoliticaVersion: politica.Version, PoliticaHuella: politica.Huella, Audiencia: politica.Audiencia, Finalidad: politica.Finalidad, PayloadNegocio: append([]byte(nil), payload...), Recurso: clonarRecursoInscripcion(recurso), Solicitud: solicitud, Decision: decision, Confirmacion: confirmacion, Material: material, ResultadoContexto: clonarResultadoInscripcion(resultado), Auditoria: clonarAuditoriaInscripcion(audit)}
	if ctx.Err() != nil {
		return vacio, ErrInscripcionPropiaUsuarioVECInvalida
	}
	recibo, err := s.registro.RegistrarInscripcionPropiaUsuarioVEC(ctx, clonarOrdenInscripcion(orden))
	// Tras un éxito durable se devuelve su recibo validado: una cancelación
	// tardía no puede convertir el efecto confirmado en un falso error.
	if err != nil || recibo.InscripcionRef != reserva.InscripcionRef || recibo.SujetoRef != reserva.SujetoRef || !puertosbolsa.ReferenciaOpacaLlamamientoValida(recibo.AuditoriaRef) || recibo.AuditoriaRef == audit.ID || recibo.Version != 1 || !puertosbolsa.ReferenciaOpacaLlamamientoValida(recibo.ReciboRef) || !puertosbolsa.ReferenciaOpacaLlamamientoValida(recibo.EventoRef) || !instanteInscripcionValido(recibo.RegistradaEn) || recibo.RegistradaEn.Before(politica.VigenteDesde) || !recibo.RegistradaEn.Before(politica.VigenteHasta) {
		return vacio, ErrInscripcionPropiaUsuarioVECInvalida
	}
	return recibo, nil
}

func candidatoActivo(r dominiovec.ResultadoContextoActorRegistradoV2, ahora time.Time) (dominiovec.VinculoReferenciaContextoActor, bool) {
	var elegido dominiovec.VinculoReferenciaContextoActor
	for _, v := range r.Contexto.Instantanea.Vinculos {
		if v.Tipo == dominiovec.TipoReferenciaContextoActorCandidato && v.VigenteEn(ahora) {
			if elegido.VinculoRef != "" {
				return elegido, false
			}
			elegido = v
		}
	}
	return elegido, elegido.VinculoRef != ""
}
func politicaValida(p puertosbolsa.PoliticaInscripcionPropiaUsuarioVEC, ahora time.Time) bool {
	return puertosbolsa.ReferenciaOpacaLlamamientoValida(p.Referencia) && p.Version > 0 && p.Version <= 1<<53-1 && huellaInscripcionValida(p.Huella) && instanteInscripcionValido(p.VigenteDesde) && instanteInscripcionValido(p.VigenteHasta) && instanteInscripcionValido(ahora) && p.VigenteDesde.Before(p.VigenteHasta) && !ahora.Before(p.VigenteDesde) && ahora.Before(p.VigenteHasta) && p.Accion == puertosbolsa.AccionRegistrarInscripcionPropiaUsuarioVEC && puertosbolsa.ReferenciaOpacaLlamamientoValida(p.Finalidad) && puertosbolsa.ReferenciaOpacaLlamamientoValida(p.Audiencia) && dominiovec.ReferenciaMotivoAutorizacionV2Valida(p.Motivo) && p.RecursoBase.Validar() == nil && p.RecursoBase.ModuloID == puertosbolsa.ModuloInscripcionPropiaUsuarioVEC && p.RecursoBase.Tipo == puertosbolsa.TipoRecursoInscripcionPropiaUsuarioVEC
}
func recursoInscripcion(p puertosbolsa.PoliticaInscripcionPropiaUsuarioVEC, r puertosbolsa.ReservaInscripcionPropiaUsuarioVEC, resultado dominiovec.ResultadoContextoActorRegistradoV2, c dominiovec.VinculoReferenciaContextoActor, e puertosbolsa.SolicitudRegistrarInscripcionPropiaUsuarioVEC) (dominiovec.RecursoAutorizable, []byte, error) {
	cuerpo, err := json.Marshal(struct {
		Esquema         string `json:"esquema"`
		InscripcionRef  string `json:"inscripcion_ref"`
		SujetoRef       string `json:"sujeto_ref"`
		PersonaRef      string `json:"persona_ref"`
		CandidatoRef    string `json:"candidato_ref"`
		VinculoRef      string `json:"vinculo_ref"`
		ConvocatoriaRef string `json:"convocatoria_ref"`
		IntencionRef    string `json:"intencion_ref"`
		PoliticaRef     string `json:"politica_ref"`
		PoliticaHuella  string `json:"politica_huella"`
		VersionVinculo  uint64 `json:"version_vinculo"`
		VersionPersona  uint64 `json:"version_persona"`
		VersionPolitica uint64 `json:"version_politica"`
	}{"bolsa.inscripcion-propia.v1", r.InscripcionRef, r.SujetoRef, resultado.Contexto.PersonaRef, c.Referencia, c.VinculoRef, e.ConvocatoriaRef, e.IntencionRef, p.Referencia, p.Huella, c.Version, resultado.Contexto.Instantanea.PersonaVersion, p.Version})
	if err != nil {
		return dominiovec.RecursoAutorizable{}, nil, err
	}
	recurso := p.RecursoBase
	recurso.Referencia = r.InscripcionRef
	recurso.Ambitos = mapaInscripcion(recurso.Ambitos)
	recurso.Atributos = mapaInscripcion(recurso.Atributos)
	for k, v := range map[string]string{"convocatoria_ref": e.ConvocatoriaRef, "candidato_ref": c.Referencia} {
		if previo, ok := recurso.Ambitos[k]; ok && previo != v {
			return dominiovec.RecursoAutorizable{}, nil, ErrInscripcionPropiaUsuarioVECInvalida
		}
		recurso.Ambitos[k] = v
	}
	h := sha256.Sum256(cuerpo)
	recurso.Atributos["payload_negocio_sha256"] = hex.EncodeToString(h[:])
	recurso.Atributos["persona_ref"] = resultado.Contexto.PersonaRef
	recurso.Atributos["persona_version"] = itoa(resultado.Contexto.Instantanea.PersonaVersion)
	recurso.Atributos["vinculo_candidato_ref"] = c.VinculoRef
	recurso.Atributos["vinculo_candidato_version"] = itoa(c.Version)
	recurso.Atributos["politica_ref"] = p.Referencia
	recurso.Atributos["politica_version"] = itoa(p.Version)
	recurso.Atributos["politica_huella_sha256"] = p.Huella
	if recurso.Validar() != nil {
		return dominiovec.RecursoAutorizable{}, nil, ErrInscripcionPropiaUsuarioVECInvalida
	}
	return recurso, cuerpo, nil
}
func concesionInscripcionValida(s dominiovec.SolicitudAutorizacionLigadaV3, d dominiovec.DecisionAutorizacionLigadaV3, c puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, material puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, r dominiovec.ResultadoContextoActorRegistradoV2, recurso dominiovec.RecursoAutorizable, politica puertosbolsa.PoliticaInscripcionPropiaUsuarioVEC, ahora time.Time) bool {
	if material.ValidarEstructura() != nil || !vinculoYResultadoVigentes(s, r, ahora) || d.ValidarPara(s) != nil {
		return false
	}
	orden, e := puertosvec.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(s, d, politica.Motivo, r)
	datos, e2 := s.Datos()
	desde, hasta, e3 := d.VentanaValidez()
	cd, ed := dominiovec.RepresentacionCanonicaDecisionAutorizacionV3(d)
	mc, emc := dominiovec.RepresentacionCanonicaMotivoAutorizacionV2(politica.Motivo)
	hr, ehr := recurso.HuellaContextoAutorizacionSHA256()
	dc := sha256.Sum256(cd)
	hm := sha256.Sum256(mc)
	resumen := material.ResumenCapacidad()
	proyeccion, ep := dominiovec.ParsearMensajeAtestacionAutorizacionV3NoAutoritativo(material.PayloadVECAD3())
	cabecera, ec := proyeccion.Cabecera()
	mensaje, es := dominiovec.SerializarMensajeAtestacionAutorizacionV3(cabecera, d, politica.Motivo, r)
	confirmacion, econf := c.Datos()
	return e == nil && e2 == nil && e3 == nil && ed == nil && emc == nil && ehr == nil && ep == nil && ec == nil && es == nil && econf == nil && restriccionesDecisionInscripcionVacias(cd) && reflect.DeepEqual(datos.Recurso, recurso) && datos.Accion == puertosbolsa.AccionRegistrarInscripcionPropiaUsuarioVEC && datos.Finalidad == politica.Finalidad && d.ValidarPara(s) == nil && !ahora.Before(desde) && ahora.Before(hasta) && c.ValidarPara(orden) == nil && c.DentroDeVentanaEn(ahora) && resumen.DecisionRef() == confirmacion.DecisionRef && resumen.DecisionHuellaSHA256() == confirmacion.DecisionHuellaSHA256 && bytes.Equal(cd, material.DecisionCanonica()) && bytes.Equal(mc, material.MotivoCanonico()) && bytes.Equal(r.RepresentacionCanonica, material.ContextoActorCanonico()) && resumen.DecisionHuellaSHA256() == hex.EncodeToString(dc[:]) && resumen.MotivoHuellaSHA256() == hex.EncodeToString(hm[:]) && resumen.Operacion() == puertosbolsa.AccionRegistrarInscripcionPropiaUsuarioVEC && resumen.EfectoRef() == recurso.Referencia && resumen.EfectoHuellaSHA256() == hr && resumen.AudienciaConsumo() == politica.Audiencia && resumen.ContextoRef() == r.RegistroContextoRef && resumen.ContextoHuellaSHA256() == r.HuellaSHA256 && material.PersonaVersion() == r.Contexto.Instantanea.PersonaVersion && material.PerfilVersion() == r.Contexto.Instantanea.PerfilVersion && !ahora.Before(resumen.EmitidaEn()) && ahora.Before(resumen.ExpiraEn()) && !resumen.EmitidaEn().Before(confirmacion.EmitidaEn) && !resumen.ExpiraEn().After(confirmacion.ValidaHasta) && bytes.Equal(mensaje, material.PayloadVECAD3())
}

// restriccionesDecisionInscripcionVacias interpreta exclusivamente la
// representación canónica ya sellada por V3. Mientras este caso de uso no
// implemente filtrado ni cumplimiento de obligaciones, cualquier valor que no
// sea la lista canónica vacía niega el registro.
func restriccionesDecisionInscripcionVacias(canonica []byte) bool {
	var decision struct {
		Campos       json.RawMessage `json:"campos_permitidos"`
		Obligaciones json.RawMessage `json:"obligaciones"`
	}
	if json.Unmarshal(canonica, &decision) != nil {
		return false
	}
	return bytes.Equal(decision.Campos, []byte("[]")) && bytes.Equal(decision.Obligaciones, []byte("[]"))
}
func vinculoYResultadoVigentes(s dominiovec.SolicitudAutorizacionLigadaV3, r dominiovec.ResultadoContextoActorRegistradoV2, ahora time.Time) bool {
	d, e := s.Datos()
	return e == nil && r.Validar() == nil && d.VinculoAutenticacionActor.ValidarPara(r) == nil && d.VinculoAutenticacionActor.VigenteEn(ahora, r)
}
func auditoriaInscripcionValida(a dominiovec.AuditEntry, r dominiovec.ResultadoContextoActorRegistradoV2, recurso dominiovec.RecursoAutorizable, correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2, finalidad string) bool {
	c, e := correlacion.ValorCanonico()
	return e == nil && a.ID == "" && a.Seq == 0 && a.Signature == "" && a.IntegrityAlgorithm == "" && a.PrevSignature == "" && a.AuthorizationRef == "" && len(a.ActorRoles) == 0 && a.RepresentedSubjectID == "" && a.ExpedienteRef == "" && a.DocumentRef == "" && a.RuleRef == "" && a.Reason == "" && a.BeforeHash == "" && a.AfterHash == "" && len(a.Metadata) == 0 && a.ObjectVersion == 0 && a.Action == puertosbolsa.AccionRegistrarInscripcionPropiaUsuarioVEC && a.ModuleID == recurso.ModuloID && a.Purpose == finalidad && a.SubjectRef == recurso.Referencia && a.Result == "accepted" && a.CorrelationRef == c && a.ActorProfile == r.Contexto.PerfilActivoRef && a.AuthMethod == r.Contexto.Principal.AuthMethod && a.AuthAssurance == r.Contexto.Principal.AuthAssurance && instanteInscripcionValido(a.OccurredAt) && actorHMACInscripcionValido(a.ActorID)
}
func solicitudInscripcionValida(s puertosbolsa.SolicitudRegistrarInscripcionPropiaUsuarioVEC) bool {
	return (dominiovec.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: s.AutenticacionRef, SesionRef: s.SesionRef}).Validar() == nil && perfilActivoInscripcionValido(s.PerfilActivoRef) && puertosbolsa.ReferenciaOpacaLlamamientoValida(s.ConvocatoriaRef) && puertosbolsa.ReferenciaOpacaLlamamientoValida(s.IntencionRef)
}
func mapaInscripcion(m map[string]string) map[string]string {
	n := make(map[string]string, len(m)+8)
	for k, v := range m {
		n[k] = v
	}
	return n
}
func itoa(v uint64) string {
	return strconv.FormatUint(v, 10)
}
func huellaInscripcionValida(v string) bool {
	if len(v) != sha256.Size*2 || v == strings.Repeat("0", sha256.Size*2) {
		return false
	}
	for _, c := range v {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

func instanteInscripcionValido(v time.Time) bool {
	return !v.IsZero() && v.Location() == time.UTC && v.Year() >= 1 && v.Year() <= 9999 && v.Nanosecond()%1_000 == 0
}

func perfilActivoInscripcionValido(v string) bool {
	if v == "" || len(v) > 512 || strings.TrimSpace(v) != v {
		return false
	}
	for _, c := range v {
		if c < 0x20 || c == 0x7f {
			return false
		}
	}
	return true
}

func actorHMACInscripcionValido(v string) bool {
	partes := strings.Split(v, ":")
	if len(partes) != 3 || partes[0] != "hmac-sha256" || partes[1] == "" || len(partes[1]) > 64 || len(partes[2]) != sha256.Size*2 || partes[2] == strings.Repeat("0", sha256.Size*2) {
		return false
	}
	for i, c := range partes[1] {
		if (i == 0 && (c < 'a' || c > 'z')) || (i > 0 && (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '.' && c != '_' && c != '-') {
			return false
		}
	}
	return huellaInscripcionValida(partes[2])
}

func clonarRecursoInscripcion(r dominiovec.RecursoAutorizable) dominiovec.RecursoAutorizable {
	r.Ambitos = mapaInscripcion(r.Ambitos)
	r.Atributos = mapaInscripcion(r.Atributos)
	return r
}

func clonarResultadoInscripcion(r dominiovec.ResultadoContextoActorRegistradoV2) dominiovec.ResultadoContextoActorRegistradoV2 {
	copia, err := r.Clonar()
	if err != nil {
		return dominiovec.ResultadoContextoActorRegistradoV2{}
	}
	return copia
}

func clonarAuditoriaInscripcion(a dominiovec.AuditEntry) dominiovec.AuditEntry {
	a.ActorRoles = append([]string(nil), a.ActorRoles...)
	if a.Metadata != nil {
		a.Metadata = mapaInscripcion(a.Metadata)
	}
	return a
}

func clonarOrdenInscripcion(o puertosbolsa.OrdenRegistroInscripcionPropiaUsuarioVEC) puertosbolsa.OrdenRegistroInscripcionPropiaUsuarioVEC {
	o.PayloadNegocio = append([]byte(nil), o.PayloadNegocio...)
	o.Recurso = clonarRecursoInscripcion(o.Recurso)
	o.ResultadoContexto = clonarResultadoInscripcion(o.ResultadoContexto)
	o.Auditoria = clonarAuditoriaInscripcion(o.Auditoria)
	return o
}

func nuloInscripcion(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	return (x.Kind() == reflect.Ptr || x.Kind() == reflect.Interface || x.Kind() == reflect.Func || x.Kind() == reflect.Map || x.Kind() == reflect.Slice) && x.IsNil()
}
