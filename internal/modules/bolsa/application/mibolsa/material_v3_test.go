package mibolsa

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	cose "github.com/veraison/go-cose"
	"strings"
	"testing"
	"time"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	app "vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type firmanteContacto struct {
	privada  ed25519.PrivateKey
	llamadas *int
	ahora    time.Time
}

func (f firmanteContacto) FirmarAtestacionAutorizacionV3(_ context.Context, s ports.SolicitudFirmaAtestacionAutorizacionV3) (ports.ResultadoFirmaAtestacionAutorizacionV3, error) {
	*f.llamadas++
	b, err := s.Mensaje()
	if err != nil {
		return ports.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	aad, err := confianza.AADExternoAtestacionAutorizacionV3("vec/prueba/contacto")
	if err != nil {
		return ports.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	m := cose.NewSign1Message()
	m.Headers.Protected.SetAlgorithm(cose.AlgorithmEdDSA)
	m.Headers.Protected[cose.HeaderLabelKeyID] = []byte("clave:prueba:contacto")
	m.Payload = b
	signer, err := cose.NewSigner(cose.AlgorithmEdDSA, f.privada)
	if err != nil {
		return ports.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	if err = m.Sign(rand.Reader, aad, signer); err != nil {
		return ports.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	m.Payload = nil
	m.Headers.RawProtected = nil
	m.Headers.RawUnprotected = nil
	sobre, err := m.MarshalCBOR()
	if err != nil {
		return ports.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	return ports.NuevoResultadoFirmaAtestacionAutorizacionV3(s, sobre, "evidencia:prueba:contacto", f.ahora)
}

type entornoMiBolsa struct {
	*entornoAutorizacionSolicitudV3Prueba
	servicio    *Servicio
	repositorio *repositorioPrueba
	emisor      *emisorObservado
	autorizador *autorizadorObservado
	orden       Orden
	firmas      int
}
type repositorioPrueba struct {
	llamadas  int
	solicitud bolsa.SolicitudConsultaMiBolsa
	err       error
	mutar     func(*bolsa.InstantaneaMiBolsa)
}

func (r *repositorioPrueba) ConsultarMiBolsa(_ context.Context, s bolsa.SolicitudConsultaMiBolsa) (bolsa.InstantaneaMiBolsa, error) {
	r.llamadas++
	r.solicitud = s
	i := bolsa.InstantaneaMiBolsa{ConsultadaEn: s.ConsultadaEn, Participaciones: []bolsa.ParticipacionMiBolsa{{Bolsa: "bolsa:auxiliar", Categoria: "categoria:auxiliar", Version: 2, OrdenInicial: 3, TotalInstantanea: 20, EstadoBolsa: "vigente", VigenteDesde: s.ConsultadaEn.Add(-time.Hour)}}}
	if r.mutar != nil {
		r.mutar(&i)
	}
	return i, r.err
}

type autorizadorObservado struct {
	ports.AutorizadorSolicitudLigadaV3
	cambiar func(domain.SolicitudAutorizacionLigadaV3) domain.SolicitudAutorizacionLigadaV3
	despues func()
}

func (a *autorizadorObservado) ExigirSolicitudLigadaV3(ctx context.Context, s domain.SolicitudAutorizacionLigadaV3, r domain.ResultadoContextoActorRegistradoV2) (domain.DecisionAutorizacionLigadaV3, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	if a.cambiar != nil {
		s = a.cambiar(s)
	}
	d, c, e := a.AutorizadorSolicitudLigadaV3.ExigirSolicitudLigadaV3(ctx, s, r)
	if a.despues != nil {
		a.despues()
	}
	return d, c, e
}

// Solo prueba: permite reutilizar el emisor nominal sobre la decisión ya
// obtenida sin invocar de nuevo el PDP ni registrar otra concesión.
type decisionYaExigida struct {
	d domain.DecisionAutorizacionLigadaV3
	c ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3
}

func (a decisionYaExigida) ExigirSolicitudLigadaV3(_ context.Context, s domain.SolicitudAutorizacionLigadaV3, _ domain.ResultadoContextoActorRegistradoV2) (domain.DecisionAutorizacionLigadaV3, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	return a.d, a.c, a.d.ValidarPara(s)
}

type emisorObservado struct {
	at                      *app.ServicioAtestacionesAutorizacionV3
	confianza               *confianza.ServicioConfianzaAtestacionAutorizacionV3
	capacidades             *confianza.EmisorCapacidadesAtestacionAutorizacionV3
	llamadas, exportaciones int
	err                     error
	exportador              ports.ExportadorMaterialConsumoAutorizacionAtestadaV3
	mutarMaterial           func(ports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
	despues                 func()
}

func (e *emisorObservado) EmitirMaterialMiBolsa(ctx context.Context, s domain.SolicitudAutorizacionLigadaV3, r domain.ResultadoContextoActorRegistradoV2, d domain.DecisionAutorizacionLigadaV3, c ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3) (ports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	e.llamadas++
	if e.err != nil {
		return nil, e.err
	}
	real, err := confianza.NuevoEmisorMaterialAutorizacionAtestadaV3(decisionYaExigida{d, c}, e.at, e.confianza, e.capacidades)
	if err != nil {
		return nil, err
	}
	_, _, x, err := real.EmitirMaterialAutorizacionAtestadaV3(ctx, s, r)
	if e.despues != nil {
		e.despues()
	}
	if err != nil {
		return nil, err
	}
	if e.exportador != nil {
		x = e.exportador
	}
	return exportadorObservado{x, &e.exportaciones, e.mutarMaterial}, nil
}

type exportadorObservado struct {
	ports.ExportadorMaterialConsumoAutorizacionAtestadaV3
	llamadas *int
	mutar    func(ports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

func (e exportadorObservado) ExportarMaterialParaConsumidor() (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	*e.llamadas++
	m, err := e.ExportadorMaterialConsumoAutorizacionAtestadaV3.ExportarMaterialParaConsumidor()
	if err == nil && e.mutar != nil {
		return e.mutar(m)
	}
	return m, err
}
func exigir(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func nuevoEntorno(t *testing.T, opciones ...opcionContexto) *entornoMiBolsa {
	t.Helper()
	e := &entornoMiBolsa{entornoAutorizacionSolicitudV3Prueba: nuevoEntornoAutorizacionSolicitudV3Prueba(t, opciones...), repositorio: new(repositorioPrueba)}
	base, err := e.solicitud.Datos()
	exigir(t, err)
	e.orden = Orden{ResultadoContexto: e.resultado, Vinculo: base.VinculoAutenticacionActor, Motivo: base.ReferenciaMotivo, Correlacion: base.Correlacion}
	e.fuente.instantanea.AsignacionPerfil.Ambitos = []domain.AmbitoPerfil{{Clave: "candidato_ref", Valores: []string{referenciaServicioContextoActorPrueba("can_", "c")}}}
	c := &e.fuente.instantanea.VersionRol.Concesiones[0]
	c.Accion = bolsa.AccionConsultarMiBolsa
	c.ModuloID = bolsa.ModuloMiBolsa
	c.TipoRecurso = bolsa.TipoRecursoMiBolsa
	c.Finalidades = []string{bolsa.FinalidadMiBolsa}
	c.CamposPermitidos = []string{bolsa.CampoMiBolsa}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	exigir(t, err)
	t.Cleanup(func() { clear(priv) })
	reloj := &relojAutorizacionServicioPrueba{ahora: e.ahora}
	at, err := app.NuevoServicioAtestacionesAutorizacionV3(domain.CabeceraAtestacionAutorizacionV3{FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV3, Suite: confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, ClaveID: "clave:prueba:contacto", Audiencia: "vec/prueba/contacto"}, firmanteContacto{privada: priv, ahora: e.ahora, llamadas: &e.firmas})
	exigir(t, err)
	raiz, err := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA("clave:prueba:contacto", 1, pub, "vec/prueba/contacto", confianza.EstadoClaveAtestacionAutorizacionV3Activa, e.ahora.Add(-time.Hour), e.ahora.Add(time.Hour), time.Time{})
	exigir(t, err)
	cfg, err := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3("confianza:prueba:contacto", 1, e.ahora.Add(-time.Minute), e.ahora.Add(time.Hour), raiz)
	exigir(t, err)
	ver, err := confianza.NuevoServicioConfianzaAtestacionAutorizacionV3(cfg, reloj)
	exigir(t, err)
	clave, err := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3("clave:capacidad:contacto", 1, bytes.Repeat([]byte{0x71}, 32), "emisor:prueba:contacto", bolsa.AudienciaMiBolsa, confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, e.ahora.Add(-time.Hour), e.ahora.Add(time.Hour), time.Time{}, 1, strings.Repeat("7", 64))
	exigir(t, err)
	em, err := confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(clave, reloj)
	exigir(t, err)
	e.emisor = &emisorObservado{at: at, confianza: ver, capacidades: em}
	e.autorizador = &autorizadorObservado{AutorizadorSolicitudLigadaV3: e.entornoAutorizacionSolicitudV3Prueba.servicio}
	e.servicio, err = Nuevo(e.repositorio, e.autorizador, e.emisor, reloj)
	exigir(t, err)
	return e
}
