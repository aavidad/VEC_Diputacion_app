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
	"encoding/json"
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
	return reciboCentralContactoFuente(o)
}

// Fixture de CONTRATO, no ejecución PostgreSQL: forma completa de
// auditoria_json_v1 (T13/1) y transformación entrada de T13/6. Los identificadores
// de cadena/consumo son sintéticos; no se afirma verificar la cadena con ellos.
// A diferencia del antiguo doble, nunca devuelve ni reetiqueta una AuditEntry.
func reciboCentralContactoFuente(o ports.OrdenRegistroContactoUsuario) (ports.ReciboContactoUsuario, error) {
	p := o.Preparacion
	a := p.Auditoria
	h := sha256.Sum256(p.PayloadNegocio)
	material := hex.EncodeToString(h[:])
	recurso, err := p.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return ports.ReciboContactoUsuario{}, err
	}
	consumo := strings.Repeat("b", 64)
	consumoRef := "aud_v3_" + consumo[:32]
	ref := "acc_" + strings.Repeat("c", 40)
	central := map[string]any{
		"id": ref, "seq": 1, "integrity_algorithm": "sha256-chain-v1", "prev_signature": strings.Repeat("0", 64), "signature": strings.Repeat("9", 64),
		"actor_id": a.ActorID, "actor_profile": a.ActorProfile, "actor_roles": a.ActorRoles, "represented_subject_id": "",
		"auth_method": string(a.AuthMethod), "auth_assurance": string(a.AuthAssurance), "authorization_ref": o.Material.ResumenCapacidad().DecisionRef(),
		"purpose": a.Purpose, "action": a.Action, "module_id": a.ModuleID, "subject_ref": a.SubjectRef, "object_version": a.ObjectVersion,
		"expediente_ref": "", "document_ref": "", "rule_ref": "", "reason": "", "result": "permitido", "before_hash": "", "after_hash": material,
		"correlation_ref": a.CorrelationRef, "metadata": map[string]string{"consumo_ref": consumoRef, "consumo_huella_sha256": consumo, "material_sha256": material, "contexto_recurso_sha256": recurso},
		"occurred_at": a.OccurredAt.UTC().Format("2006-01-02T15:04:05.000000Z"),
	}
	original, err := json.MarshalIndent(central, "", " ")
	if err != nil {
		return ports.ReciboContactoUsuario{}, err
	}
	huella := sha256.Sum256(original)
	return ports.ReciboContactoUsuario{SujetoRef: p.SujetoRef, Version: p.VersionNueva, ConsumoRef: consumoRef, ConsumoHuellaSHA256: consumo, EvidenciaCentral: ports.EvidenciaAuditoriaCentralContactoUsuario{JSONOriginal: original, Referencia: ref, HuellaJSONSHA256: hex.EncodeToString(huella[:])}}, nil
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
	var central proyeccionCentralContacto
	contactoExigir(t, json.Unmarshal(recibo.EvidenciaCentral.JSONOriginal, &central))
	if e.registro.orden.Preparacion.Auditoria.AuthorizationRef != "" || central.AuthorizationRef != e.emisor.material.ResumenCapacidad().DecisionRef() || central.AuthorizationRef == "" || central.Result != "permitido" {
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

func TestContactoReciboCentralConservaOriginalYCotejaTransformacionT13(t *testing.T) {
	e := nuevoEntornoContacto(t)
	recibo, err := e.servicio.Guardar(context.Background(), e.solicitud)
	contactoExigir(t, err)
	original, err := reciboCentralContactoFuente(e.registro.orden)
	contactoExigir(t, err)
	if !bytes.Equal(recibo.EvidenciaCentral.JSONOriginal, original.EvidenciaCentral.JSONOriginal) {
		t.Fatal("evidencia central normalizada o reinterpretada")
	}
	// El transporte conserva incluso espacios y campos vacíos de la respuesta.
	if !bytes.Contains(recibo.EvidenciaCentral.JSONOriginal, []byte("\n ")) || !bytes.Contains(recibo.EvidenciaCentral.JSONOriginal, []byte(`"represented_subject_id": ""`)) {
		t.Fatal("original perdido")
	}
	p := e.registro.orden.Preparacion
	decision := e.emisor.material.ResumenCapacidad().DecisionRef()
	for _, caso := range []string{"resultado_accepted", "actor", "version", "decision", "material", "consumo", "recurso", "ausente", "extra", "duplicada", "duplicada_metadata", "null", "huella", "referencia", "firma_malformada", "mayuscula"} {
		t.Run(caso, func(t *testing.T) {
			r := recibo
			r.EvidenciaCentral.JSONOriginal = bytes.Clone(recibo.EvidenciaCentral.JSONOriginal)
			var campos map[string]any
			contactoExigir(t, json.Unmarshal(r.EvidenciaCentral.JSONOriginal, &campos))
			meta := campos["metadata"].(map[string]any)
			switch caso {
			case "resultado_accepted":
				campos["result"] = "accepted"
			case "actor":
				campos["actor_id"] = "hmac-sha256:otro:" + strings.Repeat("a", 64)
			case "version":
				campos["object_version"] = 2
			case "decision":
				campos["authorization_ref"] = "dec_otra"
			case "material":
				campos["after_hash"] = strings.Repeat("1", 64)
			case "consumo":
				meta["consumo_huella_sha256"] = strings.Repeat("1", 64)
			case "recurso":
				meta["contexto_recurso_sha256"] = strings.Repeat("1", 64)
			case "ausente":
				delete(campos, "reason")
			case "extra":
				campos["sobra"] = ""
			case "null":
				campos["reason"] = nil
			case "referencia":
				r.EvidenciaCentral.Referencia = "acc_" + strings.Repeat("d", 40)
			case "firma_malformada":
				campos["signature"] = "firma:no-central"
			case "mayuscula":
				campos["ID"] = campos["id"]
				delete(campos, "id")
			}
			r.EvidenciaCentral.JSONOriginal, err = json.Marshal(campos)
			contactoExigir(t, err)
			if caso == "duplicada" {
				r.EvidenciaCentral.JSONOriginal = append([]byte(`{"result":"permitido",`), r.EvidenciaCentral.JSONOriginal[1:]...)
			}
			if caso == "duplicada_metadata" {
				r.EvidenciaCentral.JSONOriginal = bytes.Replace(r.EvidenciaCentral.JSONOriginal, []byte(`"metadata":{`), []byte(`"metadata":{"material_sha256":"otra",`), 1)
			}
			h := sha256.Sum256(r.EvidenciaCentral.JSONOriginal)
			r.EvidenciaCentral.HuellaJSONSHA256 = hex.EncodeToString(h[:])
			if caso == "huella" {
				r.EvidenciaCentral.HuellaJSONSHA256 = strings.Repeat("0", 64)
			}
			if reciboValido(r, p, decision) {
				t.Fatal("evidencia central ajena aceptada")
			}
		})
	}
}

func TestContactoValidadoresCentralesAntesDeConfirmarRegistroYLectura(t *testing.T) {
	e := nuevoEntornoContacto(t)
	_, err := e.servicio.Guardar(context.Background(), e.solicitud)
	contactoExigir(t, err)
	r, err := reciboCentralContactoFuente(e.registro.orden)
	contactoExigir(t, err)
	evidencia, err := ValidarEvidenciaCentralRegistroContactoUsuario(r.EvidenciaCentral.JSONOriginal, e.registro.orden.Preparacion, e.emisor.material.ResumenCapacidad().DecisionRef(), r.ConsumoRef, r.ConsumoHuellaSHA256)
	contactoExigir(t, err)
	if !bytes.Equal(evidencia.JSONOriginal, r.EvidenciaCentral.JSONOriginal) {
		t.Fatal("registro perdió original")
	}
	s := accesoContacto(t, e)
	// T13/6 aplica la misma transformación central al audit y negocio de consulta.
	lectura, err := reciboCentralContactoFuente(ports.OrdenRegistroContactoUsuario{Preparacion: ports.PreparacionRegistroContactoUsuario{SujetoRef: s.SujetoRef, VersionNueva: s.Version, Auditoria: s.Auditoria, PayloadNegocio: s.PayloadNegocio, Recurso: s.Recurso}, Material: s.Material})
	contactoExigir(t, err)
	evidencia, err = ValidarEvidenciaCentralConsultaContactoUsuario(lectura.EvidenciaCentral.JSONOriginal, s, lectura.ConsumoRef, lectura.ConsumoHuellaSHA256)
	contactoExigir(t, err)
	if !bytes.Equal(evidencia.JSONOriginal, lectura.EvidenciaCentral.JSONOriginal) {
		t.Fatal("lectura perdió original")
	}
	for _, caso := range []string{"parcial", "otra_operacion", "actor", "sujeto", "version", "finalidad", "correlacion", "consumo", "recurso", "metadata"} {
		t.Run(caso, func(t *testing.T) {
			original := bytes.Clone(lectura.EvidenciaCentral.JSONOriginal)
			var c map[string]any
			contactoExigir(t, json.Unmarshal(original, &c))
			switch caso {
			case "parcial":
				c = map[string]any{"id": lectura.EvidenciaCentral.Referencia, "result": "permitido"}
			case "otra_operacion":
				c["authorization_ref"] = "dec_otra"
			case "actor":
				c["actor_id"] = "otro"
			case "sujeto":
				c["subject_ref"] = "per_otra"
			case "version":
				c["object_version"] = 2
			case "finalidad":
				c["purpose"] = "otra"
			case "correlacion":
				c["correlation_ref"] = "otra"
			case "consumo":
				c["metadata"].(map[string]any)["consumo_ref"] = "aud_v3_otro"
			case "recurso":
				c["metadata"].(map[string]any)["contexto_recurso_sha256"] = strings.Repeat("1", 64)
			case "metadata":
				c["metadata"] = map[string]any{}
			}
			original, err = json.Marshal(c)
			contactoExigir(t, err)
			if _, err := ValidarEvidenciaCentralConsultaContactoUsuario(original, s, lectura.ConsumoRef, lectura.ConsumoHuellaSHA256); err == nil {
				t.Fatal("consulta central desligada aceptada")
			}
		})
	}
}

func TestContactoConstructorConservaRelojNativoCanonicoParaVinculoV2(t *testing.T) {
	e := nuevoEntornoContacto(t)
	// Nueva instancia normal: no se sustituye ni inyecta s.ahora. El emisor V3
	// real del entorno conserva su prueba positiva separada con fecha fija.
	s, err := NuevoServicioContactoUsuario(e.servicio.auditoria, e.servicio.protector, e.servicio.autorizador, e.servicio.registro)
	contactoExigir(t, err)
	ahora := s.ahora()
	if ahora.Location() != time.UTC || ahora.Nanosecond()%1000 != 0 {
		t.Fatal("reloj nativo fuera del contrato UTC/microsegundos")
	}
	resultado := resultadoContextoAutorizacionV3AlternativoPrueba(t, ahora)
	solicitudContexto := solicitudServicioContextoActorPrueba()
	solicitudContexto.PerfilActivoRef = resultado.Contexto.PerfilActivoRef
	autenticacion := domain.AutenticacionRevalidadaV1{
		AutenticacionRef:             referenciaServicioContextoActorPrueba("aut_", "a"),
		AutenticacionHuellaSHA256:    strings.Repeat("1", 64),
		AsercionRef:                  referenciaServicioContextoActorPrueba("ase_", "s"),
		SesionRef:                    referenciaServicioContextoActorPrueba("ses_", "e"),
		ControlSesionRef:             referenciaServicioContextoActorPrueba("cse_", "c"),
		ControlSesionRevision:        2,
		ControlSesionHuellaSHA256:    strings.Repeat("2", 64),
		CuentaRef:                    solicitudContexto.Cuenta.CuentaRef,
		CuentaOrdinariaRef:           solicitudContexto.Cuenta.CuentaRef,
		Superficie:                   domain.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado:              solicitudContexto.Cuenta.Metodo,
		GarantiaObservada:            solicitudContexto.Cuenta.Garantia,
		PoliticaGarantiaRef:          referenciaServicioContextoActorPrueba("pga_", "g"),
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    ahora.Add(-10 * time.Minute),
		SesionEmitidaEn:              ahora.Add(-9 * time.Minute),
		SesionRevalidadaEn:           ahora.Add(-3 * time.Minute),
		SesionValidaHasta:            ahora.Add(20 * time.Minute),
	}
	vinculo, err := domain.CrearVinculoAutenticacionActorV2(context.Background(), &revalidadorVinculoAplicacionAdversarial{resultado: autenticacion}, domain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef}, resolutorContextoAutorizacionV3Prueba{resultado: resultado}, solicitudContexto, &relojAutorizacionServicioPrueba{ahora: ahora})
	contactoExigir(t, err)
	base := e.solicitud.SolicitudBase
	base.VinculoAutenticacionActor = vinculo
	nominal, err := domain.NuevaSolicitudAutorizacionLigadaV3(base)
	contactoExigir(t, err)
	if !contextoContactoValido(nominal, resultado, resultado.Contexto, s.ahora()) {
		t.Fatal("reloj del constructor no alcanza la validación nominal del servicio")
	}
	// La aceptación anterior depende del contrato, no de relajar el dominio.
	if vinculo.VigenteEn(ahora.Add(time.Nanosecond), resultado) || vinculo.VigenteEn(ahora.In(time.FixedZone("local-prueba", 3600)), resultado) {
		t.Fatal("el dominio admitió un instante no canónico")
	}
}
