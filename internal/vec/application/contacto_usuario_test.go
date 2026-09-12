package application

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	cose "github.com/veraison/go-cose"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// Gobierno y persistencia son dobles sintéticos. Se reutiliza el entorno V3
// legítimo; PDP, atestador, COSE, confianza y emisor de material son reales.
type entornoContacto struct {
	*entornoAutorizacionSolicitudV3Prueba
	servicio  *ServicioContactoUsuario
	registro  *registroContactoPrueba
	solicitud ports.SolicitudRegistroContactoUsuario
	emisor    *emisorContactoObservado
}
type emisorContactoObservado struct {
	real         *confianza.EmisorMaterialAutorizacionAtestadaV3
	solicitud    domain.SolicitudAutorizacionLigadaV3
	decision     domain.DecisionAutorizacionLigadaV3
	confirmacion ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	material     ports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	mutar        func(*domain.DatosSolicitudAutorizacionLigadaV3)
}

func (e *emisorContactoObservado) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context, s domain.SolicitudAutorizacionLigadaV3, r domain.ResultadoContextoActorRegistradoV2) (domain.DecisionAutorizacionLigadaV3, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, ports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	if e.mutar != nil {
		d, _ := s.Datos()
		e.mutar(&d)
		var err error
		s, err = domain.NuevaSolicitudAutorizacionLigadaV3(d)
		if err != nil {
			return domain.DecisionAutorizacionLigadaV3{}, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, err
		}
	}
	d, c, x, err := e.real.EmitirMaterialAutorizacionAtestadaV3(ctx, s, r)
	e.solicitud = s
	e.decision = d
	e.confirmacion = c
	if err == nil {
		e.material, err = x.ExportarMaterialParaConsumidor()
	}
	return d, c, x, err
}

type firmanteContacto struct {
	privada ed25519.PrivateKey
	ahora   time.Time
}

func (f firmanteContacto) FirmarAtestacionAutorizacionV3(_ context.Context, s ports.SolicitudFirmaAtestacionAutorizacionV3) (ports.ResultadoFirmaAtestacionAutorizacionV3, error) {
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

type auditorContactoPrueba struct {
	ahora                  time.Time
	finalidad, correlacion string
}

func (a auditorContactoPrueba) PrepararAuditoriaContactoUsuario(_ context.Context, actor domain.ContextoActor, accion, modulo, sujeto string, v uint64) (domain.AuditEntry, error) {
	h := hmac.New(sha256.New, bytes.Repeat([]byte{0x35}, 32))
	h.Write([]byte(actor.Principal.ID))
	return domain.AuditEntry{ActorID: "hmac-sha256:prueba:" + hex.EncodeToString(h.Sum(nil)), ActorProfile: actor.PerfilActivoRef, ActorRoles: []string{"rol_contacto_prueba"}, AuthMethod: actor.Principal.AuthMethod, AuthAssurance: actor.Principal.AuthAssurance, AuthorizationRef: "", Purpose: a.finalidad, Action: accion, ModuleID: modulo, SubjectRef: sujeto, ObjectVersion: int(v), Result: "accepted", CorrelationRef: a.correlacion, OccurredAt: a.ahora}, nil
}

type protectorContactoPrueba struct{}

func (protectorContactoPrueba) CifrarContactoUsuario(_ context.Context, sujeto string, v uint64, b []byte) (ports.SobreContactoUsuario, error) {
	bloque, _ := aes.NewCipher(bytes.Repeat([]byte{0x26}, 32))
	gcm, _ := cipher.NewGCM(bloque)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return ports.SobreContactoUsuario{}, err
	}
	return ports.SobreContactoUsuario{Version: v, ClaveRef: "kms:prueba:contacto", Nonce: nonce, Cifrado: gcm.Seal(nil, nonce, b, []byte(sujeto))}, nil
}
func (protectorContactoPrueba) ConContactoUsuarioDescifrado(_ context.Context, sujeto string, s ports.SobreContactoUsuario, fn func([]byte) error) error {
	bloque, _ := aes.NewCipher(bytes.Repeat([]byte{0x26}, 32))
	gcm, _ := cipher.NewGCM(bloque)
	b, err := gcm.Open(nil, s.Nonce, s.Cifrado, []byte(sujeto))
	if err != nil {
		return err
	}
	defer clear(b)
	return fn(b)
}

type registroContactoPrueba struct {
	orden    ports.OrdenRegistroContactoUsuario
	llamadas int
}

func (r *registroContactoPrueba) GuardarContactoUsuario(_ context.Context, o ports.OrdenRegistroContactoUsuario) (ports.ReciboContactoUsuario, error) {
	r.llamadas++
	r.orden = o
	a := clonarAuditoria(o.Preparacion.Auditoria)
	a.AuthorizationRef = o.Material.ResumenCapacidad().DecisionRef()
	a.ID = "aud_prueba"
	a.Seq = 1
	a.Signature = "firma:prueba"
	a.IntegrityAlgorithm = "ed25519"
	return ports.ReciboContactoUsuario{SujetoRef: o.Preparacion.SujetoRef, Version: o.Preparacion.VersionNueva, Auditoria: a}, nil
}
func contactoExigir(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func nuevoEntornoContacto(t *testing.T) *entornoContacto {
	t.Helper()
	e := &entornoContacto{entornoAutorizacionSolicitudV3Prueba: nuevoEntornoAutorizacionSolicitudV3Prueba(t), registro: &registroContactoPrueba{}}
	e.entornoAutorizacionSolicitudV3Prueba.servicio.generador = generadorContactoAleatorio{}
	base, err := e.entornoAutorizacionSolicitudV3Prueba.solicitud.Datos()
	contactoExigir(t, err)
	base.Accion = AccionAltaContactoUsuario
	// Se conserva el módulo de la fixture existente. Esto no registra contacto
	// en un módulo de producto ni aporta composición/gobierno para VEC.
	for i := range e.fuente.instantanea.VersionRol.Concesiones {
		e.fuente.instantanea.VersionRol.Concesiones[i].Accion = base.Accion
	}
	lectura := e.fuente.instantanea.VersionRol.Concesiones[0]
	lectura.Accion = AccionConsultarContactoUsuario
	e.fuente.instantanea.VersionRol.Concesiones = append(e.fuente.instantanea.VersionRol.Concesiones, lectura)
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	contactoExigir(t, err)
	t.Cleanup(func() { clear(priv) })
	reloj := &relojAutorizacionServicioPrueba{ahora: e.ahora}
	at, err := NuevoServicioAtestacionesAutorizacionV3(domain.CabeceraAtestacionAutorizacionV3{FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV3, Suite: confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, ClaveID: "clave:prueba:contacto", Audiencia: "vec/prueba/contacto"}, firmanteContacto{priv, e.ahora})
	contactoExigir(t, err)
	raiz, err := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA("clave:prueba:contacto", 1, pub, "vec/prueba/contacto", confianza.EstadoClaveAtestacionAutorizacionV3Activa, e.ahora.Add(-time.Hour), e.ahora.Add(time.Hour), time.Time{})
	contactoExigir(t, err)
	cfg, err := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3("confianza:prueba:contacto", 1, e.ahora.Add(-time.Minute), e.ahora.Add(time.Hour), raiz)
	contactoExigir(t, err)
	ver, err := confianza.NuevoServicioConfianzaAtestacionAutorizacionV3(cfg, reloj)
	contactoExigir(t, err)
	clave, err := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3("clave:capacidad:contacto", 1, bytes.Repeat([]byte{0x71}, 32), "emisor:prueba:contacto", "vec.contacto_usuario.v1", confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, e.ahora.Add(-time.Hour), e.ahora.Add(time.Hour), time.Time{}, 1, strings.Repeat("7", 64))
	contactoExigir(t, err)
	em, err := confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(clave, reloj)
	contactoExigir(t, err)
	real, err := confianza.NuevoEmisorMaterialAutorizacionAtestadaV3(e.entornoAutorizacionSolicitudV3Prueba.servicio, at, ver, em)
	contactoExigir(t, err)
	e.emisor = &emisorContactoObservado{real: real}
	correlacion, err := base.Correlacion.ValorCanonico()
	contactoExigir(t, err)
	e.servicio, err = NuevoServicioContactoUsuario(auditorContactoPrueba{e.ahora, base.Finalidad, correlacion}, protectorContactoPrueba{}, e.emisor, e.registro)
	contactoExigir(t, err)
	e.servicio.ahora = func() time.Time { return e.ahora }
	c, err := domain.NuevoContactoUsuario(e.resultado.Contexto.PersonaRef, "persona@prueba.local", 1)
	contactoExigir(t, err)
	e.solicitud = ports.SolicitudRegistroContactoUsuario{ContextoActor: e.resultado.Contexto, Contacto: c, FinalidadRef: base.Finalidad, Recurso: base.Recurso, Audiencia: "vec.contacto_usuario.v1", SolicitudBase: base, ResultadoContexto: e.resultado}
	return e
}
func TestContactoGuardarConEmisorV3RealYCorreoCifrado(t *testing.T) {
	e := nuevoEntornoContacto(t)
	recibo, err := e.servicio.Guardar(context.Background(), e.solicitud)
	contactoExigir(t, err)
	if e.registro.orden.Preparacion.Auditoria.AuthorizationRef != "" || recibo.Auditoria.AuthorizationRef != e.emisor.material.ResumenCapacidad().DecisionRef() || recibo.Auditoria.AuthorizationRef == "" {
		t.Fatal("la auditoría anticipó o perdió la decisión real")
	}
	p := e.registro.orden.Preparacion
	if e.registro.llamadas != 1 || bytes.Contains(p.PayloadNegocio, []byte("persona@prueba.local")) || bytes.Contains(p.PayloadNegocio, []byte("material_sha256")) {
		t.Fatal("frontera durable o preimagen incorrecta")
	}
	if !recursoCompromete(p.Recurso, p.PayloadNegocio, p.SujetoRef, p.FinalidadRef, 0, 1) {
		t.Fatal("recurso desligado")
	}
	contactoExigir(t, protectorContactoPrueba{}.ConContactoUsuarioDescifrado(context.Background(), p.SujetoRef, p.Sobre, func(b []byte) error {
		if string(b) != "persona@prueba.local" {
			t.Fatal("correo distinto")
		}
		return nil
	}))
}
func TestContactoGuardarRechazaRecursoFirmadoDistinto(t *testing.T) {
	for _, campo := range []string{"material_sha256", "contacto_sujeto_ref", "contacto_finalidad_ref", "contacto_version", "modulo", "finalidad", "accion"} {
		t.Run(campo, func(t *testing.T) {
			e := nuevoEntornoContacto(t)
			e.emisor.mutar = func(d *domain.DatosSolicitudAutorizacionLigadaV3) {
				switch campo {
				case "modulo":
					d.Recurso.ModuloID = "otro"
				case "finalidad":
					d.Finalidad = "otra"
				case "accion":
					d.Accion = AccionConsultarContactoUsuario
				default:
					d.Recurso.Atributos[campo] = "otro"
				}
			}
			if _, err := e.servicio.Guardar(context.Background(), e.solicitud); err == nil || e.registro.llamadas != 0 {
				t.Fatal("material ajeno alcanzó efecto")
			}
		})
	}
}

type lectorContactoPrueba struct {
	contacto    domain.ContactoUsuario
	antes       func()
	veces       int
	omitirError bool
}

func (l lectorContactoPrueba) ConContactoUsuario(ctx context.Context, _ ports.SolicitudAccesoContactoUsuario, f func(domain.ContactoUsuario) error) error {
	var sobre ports.SobreContactoUsuario
	err := l.contacto.ConDireccion(func(direccion string) error {
		var e error
		sobre, e = (protectorContactoPrueba{}).CifrarContactoUsuario(ctx, l.contacto.SujetoRef(), l.contacto.Version(), []byte(direccion))
		return e
	})
	if err != nil {
		return err
	}
	return (protectorContactoPrueba{}).ConContactoUsuarioDescifrado(ctx, l.contacto.SujetoRef(), sobre, func(b []byte) error {
		contacto, err := domain.NuevoContactoUsuario(l.contacto.SujetoRef(), string(b), sobre.Version)
		if err != nil {
			return err
		}
		if l.antes != nil {
			l.antes()
		}
		for i := 0; i < l.veces; i++ {
			if err := f(contacto); err != nil && !l.omitirError {
				return err
			}
		}
		return nil
	})
}

func accesoContacto(t *testing.T, e *entornoContacto) ports.SolicitudAccesoContactoUsuario {
	a, err := e.servicio.auditoria.PrepararAuditoriaContactoUsuario(context.Background(), e.resultado.Contexto, AccionConsultarContactoUsuario, e.solicitud.Recurso.ModuloID, e.solicitud.Contacto.SujetoRef(), 1)
	contactoExigir(t, err)
	r, b, err := RecursoConsultaContactoUsuario(e.solicitud.Recurso, e.solicitud.Contacto.SujetoRef(), e.solicitud.FinalidadRef, 1, e.solicitud.Audiencia, a)
	contactoExigir(t, err)
	base := e.solicitud.SolicitudBase
	base.Recurso = r
	base.Accion = AccionConsultarContactoUsuario
	s, err := domain.NuevaSolicitudAutorizacionLigadaV3(base)
	contactoExigir(t, err)
	d, c, x, err := e.emisor.EmitirMaterialAutorizacionAtestadaV3(context.Background(), s, e.resultado)
	contactoExigir(t, err)
	m, err := x.ExportarMaterialParaConsumidor()
	contactoExigir(t, err)
	return ports.SolicitudAccesoContactoUsuario{Auditoria: a, SujetoRef: e.solicitud.Contacto.SujetoRef(), ContextoActor: e.resultado.Contexto, FinalidadRef: base.Finalidad, Recurso: r, Audiencia: e.solicitud.Audiencia, PayloadNegocio: b, Material: m, Version: 1, Solicitud: s, Decision: d, Confirmacion: c, ResultadoContexto: e.resultado}
}
func TestContactoLecturaRevalidaVersionTiempoYCallback(t *testing.T) {
	for _, caso := range []string{"correcta", "version_descifrada", "caduca_descifrando", "doble", "error_callback", "sujeto", "finalidad", "cuerpo", "version_selector", "auditoria"} {
		t.Run(caso, func(t *testing.T) {
			e := nuevoEntornoContacto(t)
			a := accesoContacto(t, e)
			l := lectorContactoPrueba{contacto: e.solicitud.Contacto, veces: 1}
			llamadas := 0
			sentinela := errors.New("callback:prueba")
			switch caso {
			case "version_descifrada":
				l.contacto, _ = domain.NuevoContactoUsuario(a.SujetoRef, "persona@prueba.local", 2)
			case "caduca_descifrando":
				l.antes = func() { e.ahora = e.ahora.Add(time.Hour) }
			case "doble":
				l.veces = 2
				l.omitirError = true
			case "error_callback":
				l.omitirError = true
			case "sujeto":
				a.SujetoRef = "per_0123456789abcdefghijkl"
			case "finalidad":
				a.FinalidadRef = "otra"
			case "cuerpo":
				a.PayloadNegocio = append(a.PayloadNegocio, ' ')
			case "version_selector":
				a.Version = 2
			case "auditoria":
				a.Auditoria.ActorID = "hmac-sha256:otro:" + strings.Repeat("a", 64)
			}
			e.servicio.lector = l
			err := e.servicio.ConContactoUsuario(context.Background(), a, func(domain.ContactoUsuario) error {
				llamadas++
				if caso == "error_callback" {
					return sentinela
				}
				return nil
			})
			if caso == "correcta" {
				contactoExigir(t, err)
				if llamadas != 1 {
					t.Fatal("callback ausente")
				}
				return
			}
			if err == nil {
				t.Fatal("lectura no denegada")
			}
			if caso == "error_callback" && !errors.Is(err, sentinela) {
				t.Fatal("error callback perdido")
			}
			esperado := 0
			if caso == "doble" || caso == "error_callback" {
				esperado = 1
			}
			if llamadas != esperado {
				t.Fatalf("callback %d, esperado %d", llamadas, esperado)
			}
		})
	}
}

func TestContactoCuerpoModificadoNoPuedeReutilizarRecurso(t *testing.T) {
	e := nuevoEntornoContacto(t)
	_, err := e.servicio.Guardar(context.Background(), e.solicitud)
	contactoExigir(t, err)
	for _, campo := range []string{"cifrado", "sujeto", "finalidad", "auditoria", "version"} {
		t.Run(campo, func(t *testing.T) {
			p := clonarPreparacion(e.registro.orden.Preparacion)
			switch campo {
			case "cifrado":
				p.Sobre.Cifrado[0] ^= 1
			case "sujeto":
				p.SujetoRef = "per_0123456789abcdefghijkl"
			case "finalidad":
				p.FinalidadRef = "otra"
			case "auditoria":
				p.Auditoria.CorrelationRef = "otra"
			case "version":
				p.VersionNueva = 2
				p.VersionEsperada = 1
				p.Sobre.Version = 2
			}
			if _, err := PayloadNegocioContactoUsuario(p); err == nil {
				t.Fatal("cuerpo distinto conservó recurso")
			}
		})
	}
}

// El preparador de auditoría nunca conoce esta referencia de decisión futura.
type generadorContactoAleatorio struct{}

func (generadorContactoAleatorio) NuevaReferenciaDecisionAutorizacion() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	return "dec_" + hex.EncodeToString(b), err
}
