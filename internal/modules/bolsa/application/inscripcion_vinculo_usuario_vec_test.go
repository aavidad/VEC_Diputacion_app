package application

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"strings"
	"testing"
	"time"

	cose "github.com/veraison/go-cose"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	appvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func TestSolicitudInscripcionPropiaNoAceptaSujetoDeclaradoNiReferenciasInvalidas(t *testing.T) {
	if solicitudInscripcionValida(puertosbolsa.SolicitudRegistrarInscripcionPropiaUsuarioVEC{}) {
		t.Fatal("una solicitud vacia no puede alcanzar las autoridades")
	}
	if puertosbolsa.ReferenciaOpacaLlamamientoValida("DNI-12345678A") {
		t.Fatal("una referencia identificativa no puede ocupar una referencia opaca")
	}
}

type relojInscripcionPrueba struct{ ahora time.Time }

func (r relojInscripcionPrueba) Ahora() time.Time { return r.ahora }

type revalidadorInscripcionPrueba struct {
	valor dominiovec.AutenticacionRevalidadaV1
}

func (r revalidadorInscripcionPrueba) RevalidarAutenticacionActorV1(context.Context, dominiovec.SolicitudRevalidacionAutenticacionActorV1) (dominiovec.AutenticacionRevalidadaV1, error) {
	return r.valor, nil
}

type resolutorInscripcionPrueba struct {
	valor dominiovec.ResultadoContextoActorRegistradoV2
}

func (r resolutorInscripcionPrueba) ResolverContextoActorRegistradoV2(context.Context, dominiovec.SolicitudContextoActor) (dominiovec.ResultadoContextoActorRegistradoV2, error) {
	return r.valor, nil
}

type denegadorInscripcionPrueba struct {
	llamadas  int
	solicitud dominiovec.SolicitudAutorizacionLigadaV3
}

type observadorEmisorInscripcionPrueba struct {
	real         *confianza.EmisorMaterialAutorizacionAtestadaV3
	llamadas     int
	solicitud    dominiovec.SolicitudAutorizacionLigadaV3
	resultado    dominiovec.ResultadoContextoActorRegistradoV2
	decision     dominiovec.DecisionAutorizacionLigadaV3
	confirmacion puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	exportador   puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3
	err          error
}

func (o *observadorEmisorInscripcionPrueba) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context, s dominiovec.SolicitudAutorizacionLigadaV3, r dominiovec.ResultadoContextoActorRegistradoV2) (dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	o.llamadas++
	o.solicitud = s
	o.resultado = r
	o.decision, o.confirmacion, o.exportador, o.err = o.real.EmitirMaterialAutorizacionAtestadaV3(ctx, s, r)
	return o.decision, o.confirmacion, o.exportador, o.err
}

func (d *denegadorInscripcionPrueba) EmitirMaterialAutorizacionAtestadaV3(_ context.Context, solicitud dominiovec.SolicitudAutorizacionLigadaV3, _ dominiovec.ResultadoContextoActorRegistradoV2) (dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	d.llamadas++
	d.solicitud = solicitud
	return dominiovec.DecisionAutorizacionLigadaV3{}, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, errors.New("denegada")
}

func TestCandidatoActivoRechazaAmbiguedadYRevocacion(t *testing.T) {
	ahora := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	base := dominiovec.VinculoReferenciaContextoActor{
		VinculoRef: "vin_abcdefghijklmnopqrstuvwx", Version: 1,
		Tipo:       dominiovec.TipoReferenciaContextoActorCandidato,
		Referencia: "can_abcdefghijklmnopqrstuvwx", Estado: dominiovec.EstadoVinculoContextoActorActivo,
		VigenteDesde: ahora.Add(-time.Minute), VigenteHasta: ahora.Add(time.Minute),
	}
	resultado := dominiovec.ResultadoContextoActorRegistradoV2{Contexto: dominiovec.ContextoActor{Instantanea: dominiovec.InstantaneaContextoActor{Vinculos: []dominiovec.VinculoReferenciaContextoActor{base, base}}}}
	if _, ok := candidatoActivo(resultado, ahora); ok {
		t.Fatal("dos candidatos activos no permiten elegir el primero")
	}
	base.Estado = dominiovec.EstadoVinculoContextoActorRevocado
	resultado.Contexto.Instantanea.Vinculos = []dominiovec.VinculoReferenciaContextoActor{base}
	if _, ok := candidatoActivo(resultado, ahora); ok {
		t.Fatal("un candidato revocado no permite inscripción")
	}
}

func TestReferenciaInscripcionRechazaCaracteresNoOpacos(t *testing.T) {
	if puertosbolsa.ReferenciaOpacaLlamamientoValida("ins_12345678901234567890-DNI") {
		t.Fatal("la referencia no debe aceptar material identificativo ni separadores libres")
	}
}

type politicaInscripcionPrueba struct {
	valor puertosbolsa.PoliticaInscripcionPropiaUsuarioVEC
	err   error
}

func (p politicaInscripcionPrueba) ResolverPoliticaInscripcionPropiaUsuarioVEC(context.Context, string, time.Time) (puertosbolsa.PoliticaInscripcionPropiaUsuarioVEC, error) {
	return p.valor, p.err
}

type reservaInscripcionPrueba struct {
	valor puertosbolsa.ReservaInscripcionPropiaUsuarioVEC
	err   error
}

func (r reservaInscripcionPrueba) ReservarInscripcionPropiaUsuarioVEC(context.Context, string, string, string) (puertosbolsa.ReservaInscripcionPropiaUsuarioVEC, error) {
	return r.valor, r.err
}

type auditorInscripcionPrueba struct {
	ahora    time.Time
	cancelar context.CancelFunc
}

func (a auditorInscripcionPrueba) PrepararAuditoriaInscripcionPropiaUsuarioVEC(_ context.Context, resultado dominiovec.ResultadoContextoActorRegistradoV2, recurso dominiovec.RecursoAutorizable, correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2, finalidad string) (dominiovec.AuditEntry, error) {
	if a.cancelar != nil {
		a.cancelar()
	}
	c, err := correlacion.ValorCanonico()
	if err != nil {
		return dominiovec.AuditEntry{}, err
	}
	return dominiovec.AuditEntry{ActorID: "hmac-sha256:prueba:" + strings.Repeat("d", 64), ActorProfile: resultado.Contexto.PerfilActivoRef, AuthMethod: resultado.Contexto.Principal.AuthMethod, AuthAssurance: resultado.Contexto.Principal.AuthAssurance, Purpose: finalidad, Action: puertosbolsa.AccionRegistrarInscripcionPropiaUsuarioVEC, ModuleID: recurso.ModuloID, SubjectRef: recurso.Referencia, Result: "accepted", CorrelationRef: c, OccurredAt: a.ahora}, nil
}

type registroInscripcionPrueba struct{ llamadas int }

func (r *registroInscripcionPrueba) RegistrarInscripcionPropiaUsuarioVEC(context.Context, puertosbolsa.OrdenRegistroInscripcionPropiaUsuarioVEC) (puertosbolsa.ReciboInscripcionPropiaUsuarioVEC, error) {
	r.llamadas++
	return puertosbolsa.ReciboInscripcionPropiaUsuarioVEC{}, errors.New("no debe registrar")
}

type registroAceptaInscripcionPrueba struct {
	llamadas int
	cancelar context.CancelFunc
}

func (r *registroAceptaInscripcionPrueba) RegistrarInscripcionPropiaUsuarioVEC(_ context.Context, orden puertosbolsa.OrdenRegistroInscripcionPropiaUsuarioVEC) (puertosbolsa.ReciboInscripcionPropiaUsuarioVEC, error) {
	r.llamadas++
	if r.cancelar != nil {
		r.cancelar()
	}
	return puertosbolsa.ReciboInscripcionPropiaUsuarioVEC{InscripcionRef: orden.InscripcionRef, SujetoRef: orden.SujetoRef, ReciboRef: "recibo:inscripcion:prueba", EventoRef: "evento:inscripcion:prueba", AuditoriaRef: "auditoria:inscripcion:prueba", Version: 1, RegistradaEn: time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)}, nil
}

type referenciasInscripcionPrueba struct{}

func (referenciasInscripcionPrueba) NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error) {
	return "correlacion_0123456789abcdef0123456789abcdef", nil
}
func (referenciasInscripcionPrueba) NuevaClaveMotivoAutorizacionV2(context.Context) (string, error) {
	return "motivo_0123456789abcdef0123456789abcdef", nil
}

type fuentePDPInscripcionPrueba struct {
	instantanea dominiovec.InstantaneaAutorizacion
}

func (f fuentePDPInscripcionPrueba) ObtenerInstantaneaAutorizacion(context.Context, string, string) (dominiovec.InstantaneaAutorizacion, error) {
	return f.instantanea, nil
}

type motivosPDPInscripcionPrueba struct {
	referencia dominiovec.ReferenciaEntradaCatalogo
}

func (m motivosPDPInscripcionPrueba) ValidarReferenciaMotivoAutorizacionV2(context.Context, dominiovec.ReferenciaEntradaCatalogo, time.Time) error {
	return nil
}

type relojPDPInscripcionPrueba struct{ ahora time.Time }

func (r relojPDPInscripcionPrueba) Ahora() time.Time { return r.ahora }

type decisionesPDPInscripcionPrueba struct{}

func (decisionesPDPInscripcionPrueba) NuevaReferenciaDecisionAutorizacion() (string, error) {
	return "dec_0123456789abcdef0123456789abcdef", nil
}

type concesionesPDPInscripcionPrueba struct{}

func (concesionesPDPInscripcionPrueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(_ context.Context, orden puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error) {
	d, e := orden.Datos()
	if e != nil {
		return time.Time{}, e
	}
	desde, _, e := d.Decision.VentanaValidez()
	return desde, e
}

type denegacionesPDPInscripcionPrueba struct{}

func (denegacionesPDPInscripcionPrueba) RegistrarDenegacionAutorizacionLigadaV3(context.Context, puertosvec.OrdenRegistroDenegacionAutorizacionLigadaV3) error {
	return nil
}

type firmanteInscripcionPrueba struct {
	privada ed25519.PrivateKey
	ahora   time.Time
}

func (f firmanteInscripcionPrueba) FirmarAtestacionAutorizacionV3(_ context.Context, s puertosvec.SolicitudFirmaAtestacionAutorizacionV3) (puertosvec.ResultadoFirmaAtestacionAutorizacionV3, error) {
	b, e := s.Mensaje()
	if e != nil {
		return puertosvec.ResultadoFirmaAtestacionAutorizacionV3{}, e
	}
	aad, e := confianza.AADExternoAtestacionAutorizacionV3("vec/prueba/inscripcion")
	if e != nil {
		return puertosvec.ResultadoFirmaAtestacionAutorizacionV3{}, e
	}
	m := cose.NewSign1Message()
	m.Headers.Protected.SetAlgorithm(cose.AlgorithmEdDSA)
	m.Headers.Protected[cose.HeaderLabelKeyID] = []byte("clave:prueba:inscripcion")
	m.Payload = b
	firmador, e := cose.NewSigner(cose.AlgorithmEdDSA, f.privada)
	if e != nil {
		return puertosvec.ResultadoFirmaAtestacionAutorizacionV3{}, e
	}
	if e = m.Sign(rand.Reader, aad, firmador); e != nil {
		return puertosvec.ResultadoFirmaAtestacionAutorizacionV3{}, e
	}
	m.Payload = nil
	m.Headers.RawProtected = nil
	m.Headers.RawUnprotected = nil
	sobre, e := m.MarshalCBOR()
	if e != nil {
		return puertosvec.ResultadoFirmaAtestacionAutorizacionV3{}, e
	}
	return puertosvec.NuevoResultadoFirmaAtestacionAutorizacionV3(s, sobre, "evidencia:prueba:inscripcion", f.ahora)
}

func emisorInscripcionRealPrueba(t *testing.T, ahora time.Time, resultado dominiovec.ResultadoContextoActorRegistradoV2, politica puertosbolsa.PoliticaInscripcionPropiaUsuarioVEC, restricciones []string, campos []string) *confianza.EmisorMaterialAutorizacionAtestadaV3 {
	t.Helper()
	rol := dominiovec.VersionRol{RolID: "tecnico_inscripcion", Version: 1, Nombre: "Tecnico inscripcion", Estado: dominiovec.EstadoVersionRolPublicada, Concesiones: []dominiovec.ConcesionRol{{Accion: puertosbolsa.AccionRegistrarInscripcionPropiaUsuarioVEC, ModuloID: puertosbolsa.ModuloInscripcionPropiaUsuarioVEC, TipoRecurso: puertosbolsa.TipoRecursoInscripcionPropiaUsuarioVEC, Finalidades: []string{politica.Finalidad}, GarantiaMinima: dominiovec.AuthAssuranceHigh, CamposPermitidos: campos, Obligaciones: restricciones}}, PublicadaPor: "seguridad_prueba", PublicadaEn: ahora.Add(-time.Hour)}
	asignacion := dominiovec.AsignacionPerfil{AsignacionID: "asignacion_inscripcion", Version: 1, PerfilActivoRef: resultado.Contexto.PerfilActivoRef, PrincipalID: resultado.Contexto.PersonaRef, VersionRolRef: rol.Referencia(), Estado: dominiovec.EstadoAsignacionPerfilActiva, Ambitos: []dominiovec.AmbitoPerfil{{Clave: "candidato_ref", Valores: []string{resultado.Contexto.Instantanea.Vinculos[0].Referencia}}, {Clave: "convocatoria_ref", Valores: []string{"con_abcdefghijklmnopqrstuvwx"}}}, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), EmitidaPor: "seguridad_prueba", EmitidaEn: ahora.Add(-time.Hour)}
	huella, e := dominiovec.HuellaCatalogoPoliticasAutorizacion(nil)
	if e != nil {
		t.Fatal(e)
	}
	pdp, e := appvec.NuevoServicioAutorizacionSolicitudLigadaV3(fuentePDPInscripcionPrueba{dominiovec.InstantaneaAutorizacion{AsignacionPerfil: asignacion, VersionRol: rol, ControlVigenciaVersionRol: dominiovec.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1, Estado: dominiovec.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: rol.PublicadaPor, ActualizadoEn: rol.PublicadaEn}, RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huella}}, concesionesPDPInscripcionPrueba{}, denegacionesPDPInscripcionPrueba{}, motivosPDPInscripcionPrueba{politica.Motivo}, relojPDPInscripcionPrueba{ahora}, decisionesPDPInscripcionPrueba{}, appvec.ConfiguracionServicioAutorizacion{VigenciaDecision: time.Minute})
	if e != nil {
		t.Fatal(e)
	}
	pub, priv, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		for i := range priv {
			priv[i] = 0
		}
	})
	at, e := appvec.NuevoServicioAtestacionesAutorizacionV3(dominiovec.CabeceraAtestacionAutorizacionV3{FormatoVersion: dominiovec.VersionFormatoAtestacionAutorizacionV3, Suite: confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, ClaveID: "clave:prueba:inscripcion", Audiencia: "vec/prueba/inscripcion"}, firmanteInscripcionPrueba{priv, ahora})
	if e != nil {
		t.Fatal(e)
	}
	raiz, e := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA("clave:prueba:inscripcion", 1, pub, "vec/prueba/inscripcion", confianza.EstadoClaveAtestacionAutorizacionV3Activa, ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{})
	if e != nil {
		t.Fatal(e)
	}
	cfg, e := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3("confianza:prueba:inscripcion", 1, ahora.Add(-time.Minute), ahora.Add(time.Hour), raiz)
	if e != nil {
		t.Fatal(e)
	}
	confianzaV3, e := confianza.NuevoServicioConfianzaAtestacionAutorizacionV3(cfg, relojPDPInscripcionPrueba{ahora})
	if e != nil {
		t.Fatal(e)
	}
	clave, e := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3("clave:capacidad:inscripcion", 1, []byte(strings.Repeat("q", 32)), "emisor:prueba:inscripcion", "vec.inscripcion.v1", confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{}, 1, strings.Repeat("7", 64))
	if e != nil {
		t.Fatal(e)
	}
	capacidades, e := confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(clave, relojPDPInscripcionPrueba{ahora})
	if e != nil {
		t.Fatal(e)
	}
	emisor, e := confianza.NuevoEmisorMaterialAutorizacionAtestadaV3(pdp, at, confianzaV3, capacidades)
	if e != nil {
		t.Fatal(e)
	}
	return emisor
}

func resultadoInscripcionV2Prueba(t *testing.T, ahora time.Time, candidato string) dominiovec.ResultadoContextoActorRegistradoV2 {
	t.Helper()
	cuenta := dominiovec.CuentaAutenticadaContextoActor{CuentaRef: "cta_abcdefghijklmnopqrstuvwx", Metodo: dominiovec.AuthMethodCertificate, Garantia: dominiovec.AuthAssuranceHigh}
	instantanea := dominiovec.InstantaneaContextoActor{VinculoRef: "vca_abcdefghijklmnopqrstuvwx", VinculoVersion: 5, CuentaRef: cuenta.CuentaRef, CuentaVersion: 7, PersonaRef: "per_abcdefghijklmnopqrstuvwx", PersonaVersion: 3, PerfilActivoRef: "prf_abcdefghijklmnopqrstuvwx", PerfilVersion: 4, Estado: dominiovec.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(30 * time.Minute), Vinculos: []dominiovec.VinculoReferenciaContextoActor{{VinculoRef: "vin_abcdefghijklmnopqrstuvwx", Version: 1, Tipo: dominiovec.TipoReferenciaContextoActorCandidato, Referencia: candidato, Estado: dominiovec.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(30 * time.Minute)}}}
	actor, err := dominiovec.NuevoContextoActor(cuenta, instantanea, ahora.Add(-2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	acreditacion := dominiovec.AcreditacionProcedenciaComponenteContextoActorV1{ProcedenciaRef: "prc_abcdefghijklmnopqrstuvwx", ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("a", 64), ProcedenciaAutoridad: dominiovec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1}
	manifiesto := dominiovec.ManifiestoProcedenciaContextoActorV1{Esquema: dominiovec.EsquemaManifiestoProcedenciaContextoActorV1, AutoridadEfectiva: dominiovec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, Cuenta: dominiovec.ProcedenciaCuentaContextoActorV1{CuentaRef: cuenta.CuentaRef, Version: 7, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion}, Persona: dominiovec.ProcedenciaPersonaContextoActorV1{PersonaRef: actor.PersonaRef, Version: 3, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion}, Perfil: dominiovec.ProcedenciaPerfilContextoActorV1{PerfilRef: actor.PerfilActivoRef, Version: 4, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion}, Contexto: dominiovec.ProcedenciaVinculoContextoActorV1{VinculoRef: instantanea.VinculoRef, Version: 5, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion}, Vinculos: []dominiovec.ProcedenciaVinculoReferenciaContextoActorV1{{VinculoRef: instantanea.Vinculos[0].VinculoRef, Version: 1, Tipo: dominiovec.TipoReferenciaContextoActorCandidato, Referencia: candidato, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion}}}
	canonManifiesto, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	huellaManifiesto, err := dominiovec.HuellaSHA256ManifiestoProcedenciaContextoActorV1(canonManifiesto)
	if err != nil {
		t.Fatal(err)
	}
	resultado := dominiovec.ResultadoContextoActorRegistradoV2{RegistroContextoRef: "rca_abcdefghijklmnopqrstuvwx", Contexto: actor, RepresentacionCanonica: canon, HuellaSHA256: huella, ManifiestoProcedenciaCanonico: canonManifiesto, ManifiestoProcedenciaHuellaSHA256: huellaManifiesto, AutoridadEfectiva: dominiovec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, ResueltoEnAutoritativo: actor.ResueltoEn}
	if err := resultado.Validar(); err != nil {
		t.Fatal(err)
	}
	return resultado
}

func TestInscripcionPropiaLlegaAlEmisorConSolicitudV3YNoRegistraAlDenegar(t *testing.T) {
	ahora := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	resultado := resultadoInscripcionV2Prueba(t, ahora, "can_abcdefghijklmnopqrstuvwx")
	autenticacion := dominiovec.AutenticacionRevalidadaV1{AutenticacionRef: "aut_abcdefghijklmnopqrstuvwx", AutenticacionHuellaSHA256: strings.Repeat("1", 64), AsercionRef: "ase_abcdefghijklmnopqrstuvwx", SesionRef: "ses_abcdefghijklmnopqrstuvwx", ControlSesionRef: "cse_abcdefghijklmnopqrstuvwx", ControlSesionRevision: 7, ControlSesionHuellaSHA256: strings.Repeat("2", 64), CuentaRef: resultado.Contexto.Instantanea.CuentaRef, CuentaOrdinariaRef: resultado.Contexto.Instantanea.CuentaRef, Superficie: dominiovec.SuperficieAutenticacionInternaCorporativaV1, MetodoObservado: dominiovec.AuthMethodCertificate, GarantiaObservada: dominiovec.AuthAssuranceHigh, PoliticaGarantiaRef: "pga_abcdefghijklmnopqrstuvwx", PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64), AutenticacionVerificadaEn: ahora.Add(-5 * time.Minute), SesionEmitidaEn: ahora.Add(-4 * time.Minute), SesionRevalidadaEn: ahora.Add(-3 * time.Minute), SesionValidaHasta: ahora.Add(10 * time.Minute)}
	politica := puertosbolsa.PoliticaInscripcionPropiaUsuarioVEC{Referencia: "pol_abcdefghijklmnopqrstuvwx", Version: 1, Huella: strings.Repeat("b", 64), VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), Accion: puertosbolsa.AccionRegistrarInscripcionPropiaUsuarioVEC, Finalidad: "finalidad_inscripcion", Audiencia: "vec.inscripcion.v1", Motivo: dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivo_inscripcion", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("c", 64), EntradaClave: "motivo_0123456789abcdef0123456789abcdef"}, RecursoBase: dominiovec.RecursoAutorizable{Referencia: "recurso_inscripcion", ModuloID: puertosbolsa.ModuloInscripcionPropiaUsuarioVEC, Tipo: puertosbolsa.TipoRecursoInscripcionPropiaUsuarioVEC, Ambitos: map[string]string{}, Atributos: map[string]string{}}}
	reserva := puertosbolsa.ReservaInscripcionPropiaUsuarioVEC{InscripcionRef: "ins_abcdefghijklmnopqrstuvwx", SujetoRef: "suj_abcdefghijklmnopqrstuvwx", IntencionRef: "int_abcdefghijklmnopqrstuvwx", PersonaRef: resultado.Contexto.PersonaRef, ConvocatoriaRef: "con_abcdefghijklmnopqrstuvwx"}
	emisor := &denegadorInscripcionPrueba{}
	registro := &registroInscripcionPrueba{}
	servicio, err := NuevoServicioInscripcionPropiaUsuarioVEC(revalidadorInscripcionPrueba{autenticacion}, resolutorInscripcionPrueba{resultado}, relojInscripcionPrueba{ahora}, politicaInscripcionPrueba{valor: politica}, reservaInscripcionPrueba{valor: reserva}, auditorInscripcionPrueba{ahora: ahora}, emisor, registro, referenciasInscripcionPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = servicio.Registrar(context.Background(), puertosbolsa.SolicitudRegistrarInscripcionPropiaUsuarioVEC{AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef, PerfilActivoRef: resultado.Contexto.PerfilActivoRef, ConvocatoriaRef: reserva.ConvocatoriaRef, IntencionRef: reserva.IntencionRef})
	if err == nil || emisor.llamadas != 1 || registro.llamadas != 0 {
		t.Fatal("la denegacion debe llegar al emisor una vez y no registrar")
	}
	datos, err := emisor.solicitud.Datos()
	if err != nil || datos.Recurso.Ambitos["candidato_ref"] != resultado.Contexto.Instantanea.Vinculos[0].Referencia || datos.Recurso.Atributos["persona_ref"] != resultado.Contexto.PersonaRef {
		t.Fatal("la solicitud V3 no liga exactamente candidato y persona autoritativos")
	}
	huellaRecurso, err := datos.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || datos.Recurso.Atributos["payload_negocio_sha256"] == huellaRecurso {
		t.Fatal("la huella del payload no puede confundirse con la huella de efecto del recurso")
	}
	payloadInicial := datos.Recurso.Atributos["payload_negocio_sha256"]
	registroReal := &registroAceptaInscripcionPrueba{}
	emisorConstruido := emisorInscripcionRealPrueba(t, ahora, resultado, politica, nil, nil)
	emisorReal := &observadorEmisorInscripcionPrueba{real: emisorConstruido}
	servicioReal, err := NuevoServicioInscripcionPropiaUsuarioVEC(revalidadorInscripcionPrueba{autenticacion}, resolutorInscripcionPrueba{resultado}, relojInscripcionPrueba{ahora}, politicaInscripcionPrueba{valor: politica}, reservaInscripcionPrueba{valor: reserva}, auditorInscripcionPrueba{ahora: ahora}, emisorReal, registroReal, referenciasInscripcionPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	reciboReal, err := servicioReal.Registrar(context.Background(), puertosbolsa.SolicitudRegistrarInscripcionPropiaUsuarioVEC{AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef, PerfilActivoRef: resultado.Contexto.PerfilActivoRef, ConvocatoriaRef: reserva.ConvocatoriaRef, IntencionRef: reserva.IntencionRef})
	if err != nil || registroReal.llamadas != 1 || reciboReal.InscripcionRef != reserva.InscripcionRef {
		confirmacion, errorConfirmacion := emisorReal.confirmacion.Datos()
		t.Fatalf("concesion V3 real no alcanzo registro sintetico: recibo=%#v err=%v llamadas=%d emisor=%v decision=%v confirmacion=%#v errorConfirmacion=%v", reciboReal, err, emisorReal.llamadas, emisorReal.err, emisorReal.decision.Validar(), confirmacion, errorConfirmacion)
	}
	ctxConfirmado, cancelarConfirmado := context.WithCancel(context.Background())
	registroCancelado := &registroAceptaInscripcionPrueba{cancelar: cancelarConfirmado}
	emisorCancelado := &observadorEmisorInscripcionPrueba{real: emisorInscripcionRealPrueba(t, ahora, resultado, politica, nil, nil)}
	servicioCancelado, e := NuevoServicioInscripcionPropiaUsuarioVEC(revalidadorInscripcionPrueba{autenticacion}, resolutorInscripcionPrueba{resultado}, relojInscripcionPrueba{ahora}, politicaInscripcionPrueba{valor: politica}, reservaInscripcionPrueba{valor: reserva}, auditorInscripcionPrueba{ahora: ahora}, emisorCancelado, registroCancelado, referenciasInscripcionPrueba{})
	if e != nil {
		t.Fatal(e)
	}
	reciboCancelado, e := servicioCancelado.Registrar(ctxConfirmado, puertosbolsa.SolicitudRegistrarInscripcionPropiaUsuarioVEC{AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef, PerfilActivoRef: resultado.Contexto.PerfilActivoRef, ConvocatoriaRef: reserva.ConvocatoriaRef, IntencionRef: reserva.IntencionRef})
	if e != nil || registroCancelado.llamadas != 1 || reciboCancelado != reciboReal || !errors.Is(ctxConfirmado.Err(), context.Canceled) {
		t.Fatalf("la cancelacion tras recibo confirmado no puede convertir el exito en error: recibo=%#v err=%v", reciboCancelado, e)
	}
	for nombre, restricciones := range map[string]struct{ obligaciones, campos []string }{
		"obligacion": {obligaciones: []string{"doble_control"}},
		"campos":     {campos: []string{"persona_ref"}},
	} {
		t.Run(nombre, func(t *testing.T) {
			registroRestringido := &registroAceptaInscripcionPrueba{}
			emisorConstruido := emisorInscripcionRealPrueba(t, ahora, resultado, politica, restricciones.obligaciones, restricciones.campos)
			emisorRestringido := &observadorEmisorInscripcionPrueba{real: emisorConstruido}
			servicioRestringido, e := NuevoServicioInscripcionPropiaUsuarioVEC(revalidadorInscripcionPrueba{autenticacion}, resolutorInscripcionPrueba{resultado}, relojInscripcionPrueba{ahora}, politicaInscripcionPrueba{valor: politica}, reservaInscripcionPrueba{valor: reserva}, auditorInscripcionPrueba{ahora: ahora}, emisorRestringido, registroRestringido, referenciasInscripcionPrueba{})
			if e != nil {
				t.Fatal(e)
			}
			_, e = servicioRestringido.Registrar(context.Background(), puertosbolsa.SolicitudRegistrarInscripcionPropiaUsuarioVEC{AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef, PerfilActivoRef: resultado.Contexto.PerfilActivoRef, ConvocatoriaRef: reserva.ConvocatoriaRef, IntencionRef: reserva.IntencionRef})
			concedida, codigo, errorDecision := emisorRestringido.decision.Resultado()
			if e == nil || registroRestringido.llamadas != 0 || emisorRestringido.llamadas != 1 || emisorRestringido.err != nil || errorDecision != nil || !concedida || codigo != "concedida" || emisorRestringido.exportador == nil {
				t.Fatalf("la concesion real restringida debe ser concedida y no alcanzar registro: servicio=%v emisor=%v concedida=%t codigo=%q", e, emisorRestringido.err, concedida, codigo)
			}
		})
	}

	servicio.politicas = politicaInscripcionPrueba{err: errors.New("fuente ausente")}
	if _, err = servicio.Registrar(context.Background(), puertosbolsa.SolicitudRegistrarInscripcionPropiaUsuarioVEC{AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef, PerfilActivoRef: resultado.Contexto.PerfilActivoRef, ConvocatoriaRef: reserva.ConvocatoriaRef, IntencionRef: reserva.IntencionRef}); err == nil || emisor.llamadas != 1 || registro.llamadas != 0 {
		t.Fatal("una fuente de politica ausente no puede alcanzar emisor ni registro")
	}
	politicaRevocada := politica
	politicaRevocada.VigenteHasta = ahora.Add(-time.Microsecond)
	servicio.politicas = politicaInscripcionPrueba{valor: politicaRevocada}
	if _, err = servicio.Registrar(context.Background(), puertosbolsa.SolicitudRegistrarInscripcionPropiaUsuarioVEC{AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef, PerfilActivoRef: resultado.Contexto.PerfilActivoRef, ConvocatoriaRef: reserva.ConvocatoriaRef, IntencionRef: reserva.IntencionRef}); err == nil || emisor.llamadas != 1 || registro.llamadas != 0 {
		t.Fatal("una politica revocada no puede alcanzar emisor ni registro")
	}
	servicio.politicas = politicaInscripcionPrueba{valor: politica}
	reservaAjena := reserva
	reservaAjena.PersonaRef = "per_abcdefghijklmnopqrstuvwy"
	servicio.reservas = reservaInscripcionPrueba{valor: reservaAjena}
	if _, err = servicio.Registrar(context.Background(), puertosbolsa.SolicitudRegistrarInscripcionPropiaUsuarioVEC{AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef, PerfilActivoRef: resultado.Contexto.PerfilActivoRef, ConvocatoriaRef: reserva.ConvocatoriaRef, IntencionRef: reserva.IntencionRef}); err == nil || emisor.llamadas != 1 || registro.llamadas != 0 {
		t.Fatal("una reserva de otra persona no puede alcanzar emisor ni registro")
	}
	servicio.reservas = reservaInscripcionPrueba{valor: reserva}
	ctx, cancelar := context.WithCancel(context.Background())
	servicio.auditoria = auditorInscripcionPrueba{ahora: ahora, cancelar: cancelar}
	if _, err = servicio.Registrar(ctx, puertosbolsa.SolicitudRegistrarInscripcionPropiaUsuarioVEC{AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef, PerfilActivoRef: resultado.Contexto.PerfilActivoRef, ConvocatoriaRef: reserva.ConvocatoriaRef, IntencionRef: reserva.IntencionRef}); err == nil || emisor.llamadas != 1 || registro.llamadas != 0 {
		t.Fatal("la cancelacion tras el puerto de auditoria no puede alcanzar emisor ni registro")
	}
	resultadoCandidatoDistinto := resultadoInscripcionV2Prueba(t, ahora, "can_abcdefghijklmnopqrstuvwy")
	politicaDistinta := politica
	politicaDistinta.Huella = strings.Repeat("e", 64)
	servicio.contextos = resolutorInscripcionPrueba{resultadoCandidatoDistinto}
	servicio.politicas = politicaInscripcionPrueba{valor: politicaDistinta}
	servicio.auditoria = auditorInscripcionPrueba{ahora: ahora}
	if _, err = servicio.Registrar(context.Background(), puertosbolsa.SolicitudRegistrarInscripcionPropiaUsuarioVEC{AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef, PerfilActivoRef: resultado.Contexto.PerfilActivoRef, ConvocatoriaRef: reserva.ConvocatoriaRef, IntencionRef: reserva.IntencionRef}); err == nil || emisor.llamadas != 2 || registro.llamadas != 0 {
		t.Fatal("la variante solo debe llegar al emisor nominal y no registrar")
	}
	variacion, err := emisor.solicitud.Datos()
	if err != nil || variacion.Recurso.Ambitos["candidato_ref"] != resultadoCandidatoDistinto.Contexto.Instantanea.Vinculos[0].Referencia || variacion.Recurso.Atributos["politica_huella_sha256"] != politicaDistinta.Huella || variacion.Recurso.Atributos["payload_negocio_sha256"] == payloadInicial {
		t.Fatal("el payload debe cambiar con candidato y politica autoritativos")
	}
}
