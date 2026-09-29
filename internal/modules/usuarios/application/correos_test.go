package application

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/usuarios/canonico"
	"vec-diputacion-granada/internal/modules/usuarios/domain"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type proveedorCorreosPrueba struct {
	denegar          bool
	audienciaForzada string
	huellaForzada    string
	materiales       []ports.MaterialCorreos
	bytes            [][]byte
}

func (p *proveedorCorreosPrueba) ProveerMaterialCorreos(_ context.Context, vinculo vecdomain.VinculoAutenticacionActorV2, m ports.MaterialCorreos) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	datos, err := vinculo.Datos()
	if err != nil || datos.Superficie != m.Superficie {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrCorreosProhibido
	}
	p.materiales = append(p.materiales, m)
	if p.denegar {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrCorreosProhibido
	}
	b, err := canonico.SerializarMaterialCorreos(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	p.bytes = append(p.bytes, b)
	recurso, err := canonico.RecursoCorreos(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	huellaContexto, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	if p.huellaForzada == "material_directo" {
		h := sha256.Sum256(b)
		huellaContexto = hex.EncodeToString(h[:])
	}
	audiencia, _ := canonico.AudienciaCorreos(m.Accion, m.Superficie)
	if p.audienciaForzada != "" {
		audiencia = p.audienciaForzada
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(fmt.Sprintf("dec_correos_%d", len(p.materiales)), strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), m.Accion, m.PersonaRef, huellaContexto, audiencia, ahora, ahora.Add(3*time.Second))
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	return vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
}

const refCorreoPrueba = "correo:0123456789abcdef0123456789abcdef"

type desafioCorreosPrueba struct {
	llamados      int
	desfase       time.Duration
	venceRecibido time.Time
}

type protectorCorreosPrueba struct {
	llamados         int
	descifrados      int
	sinClaveIgualdad bool
	fallarDescifrado bool
}

type selladorCorreosPrueba struct{}

func (selladorCorreosPrueba) SellarHuellaCorreo(_ context.Context, claro []byte) (ports.HuellasSemanticasCorreo, error) {
	mac := hmac.New(sha256.New, []byte(strings.Repeat("k", 32)))
	_, _ = mac.Write(claro)
	return ports.HuellasSemanticasCorreo{Activa: ports.HuellaSemanticaCorreo{ClaveRef: "clave:huella:v1", Valor: hex.EncodeToString(mac.Sum(nil))}}, nil
}

type selladorRotadoCorreosPrueba struct{}

func (selladorRotadoCorreosPrueba) SellarHuellaCorreo(_ context.Context, claro []byte) (ports.HuellasSemanticasCorreo, error) {
	sellar := func(clave string) string {
		mac := hmac.New(sha256.New, []byte(clave))
		_, _ = mac.Write(claro)
		return hex.EncodeToString(mac.Sum(nil))
	}
	return ports.HuellasSemanticasCorreo{
		Activa:    ports.HuellaSemanticaCorreo{ClaveRef: "clave:huella:v2", Valor: sellar(strings.Repeat("n", 32))},
		Retenidas: []ports.HuellaSemanticaCorreo{{ClaveRef: "clave:huella:v1", Valor: sellar(strings.Repeat("k", 32))}},
	}, nil
}

func (p *protectorCorreosPrueba) CifrarDireccionCorreo(_ context.Context, _, ref string, version uint64, claro []byte) (ports.SobreDireccionCorreo, error) {
	p.llamados++
	claveIgualdad := "clave:igualdad:v1"
	if p.sinClaveIgualdad {
		claveIgualdad = ""
	}
	return ports.SobreDireccionCorreo{CorreoRef: ref, Version: version, ClaveRef: "clave:correo", ClaveIgualdadRef: claveIgualdad, Nonce: []byte(strings.Repeat("n", 12)), Cifrado: append([]byte(nil), claro...), HuellaIgualdad: []byte(strings.Repeat("h", 32))}, nil
}

func (p *protectorCorreosPrueba) ConDireccionCorreoDescifrada(_ context.Context, _ string, sobre ports.SobreDireccionCorreo, usar func([]byte) error) error {
	p.descifrados++
	if p.fallarDescifrado {
		return errors.New("kms caído")
	}
	return usar([]byte("destino@example.org"))
}

func (d *desafioCorreosPrueba) PrepararDesafioCorreo(_ context.Context, _, _ string, vence time.Time) (ports.ReservaDesafio, error) {
	d.llamados++
	d.venceRecibido = vence
	return ports.ReservaDesafio{DesafioRef: "desafio:" + strings.Repeat("d", 32), Codigo: "12345678", HuellaCodigo: []byte(strings.Repeat("h", 32)), ClaveRef: "clave:uno", VenceUTC: vence.Add(d.desfase)}, nil
}

type validadorCorreosPrueba struct{ llamados int }

func (v *validadorCorreosPrueba) ComprobarCodigoCorreo(_ context.Context, _ ports.MetadatosDesafioCorreo, codigo string) (bool, error) {
	v.llamados++
	return codigo == "87654321", nil
}

type transporteCorreosPrueba struct {
	mensajes []ports.MensajeCorreoPropio
	rechazar bool
}

func (t *transporteCorreosPrueba) EnviarCorreoPropio(_ context.Context, m ports.MensajeCorreoPropio) bool {
	t.mensajes = append(t.mensajes, m)
	return !t.rechazar
}

type registroCorreosPrueba struct {
	persona                                 string
	replay                                  bool
	aplicarReplay                           bool
	consultas, recuperaciones, aplicaciones int
	peticion                                ports.PeticionCorreo
	material                                ports.MaterialCorreos
	sobre                                   ports.SobreDireccionCorreo
	reserva                                 ports.ReservaDesafio
	comprobacion                            bool
	reciboOriginal                          ports.ReciboCorreos
	confirmados                             []bool
	sinEnvios                               bool
}

func (r *registroCorreosPrueba) ConsultarPropios(_ context.Context, _ ports.OrdenCorreos, m ports.MaterialCorreos, _ vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.VistaCorreos, error) {
	r.consultas++
	r.material = m
	return ports.VistaCorreos{PersonaRef: r.persona, Correos: []domain.CorreoPropio{}}, nil
}
func (r *registroCorreosPrueba) RecuperarOperacion(_ context.Context, _ ports.OrdenCorreos, m ports.MaterialCorreos, _ vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboCorreos, bool, error) {
	r.recuperaciones++
	r.material = m
	if r.replay {
		if r.reciboOriginal.ReciboRef != "" {
			return r.reciboOriginal, true, nil
		}
		ref := m.CorreoRef
		if ref == "" {
			ref = refCorreoPrueba
		}
		return ports.ReciboCorreos{ReciboRef: "recibo:uno", PersonaRef: r.persona, Accion: m.Accion, CorreoRef: ref, Version: m.VersionEsperada + 1, FechaUTC: time.Now().UTC(), Replay: true}, true, nil
	}
	return ports.ReciboCorreos{}, false, nil
}
func (r *registroCorreosPrueba) Aplicar(ctx context.Context, _ ports.OrdenCorreos, p ports.PeticionCorreo, m ports.MaterialCorreos, _ vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, sobre ports.SobreDireccionCorreo, reserva ports.ReservaDesafio, comprobador ports.ComprobadorCodigoCorreo) (ports.ResultadoCorreos, error) {
	r.aplicaciones++
	r.peticion, r.material, r.sobre, r.reserva = p, m, sobre, reserva
	if m.Accion == ports.AccionVerificarCorreo {
		var err error
		r.comprobacion, err = comprobador.Comprobar(ctx, ports.MetadatosDesafioCorreo{PersonaRef: r.persona, CorreoRef: p.CorreoRef})
		if err != nil {
			return ports.ResultadoCorreos{}, ports.ErrCorreosNoDisponible
		}
		if !r.comprobacion {
			return ports.ResultadoCorreos{}, ports.CodigoIncorrecto{IntentosRestantes: 3}
		}
	}
	if r.aplicarReplay {
		return ports.ResultadoCorreos{Recibo: ports.ReciboCorreos{ReciboRef: "recibo:original", PersonaRef: r.persona, Accion: m.Accion, CorreoRef: refCorreoPrueba, Version: p.VersionEsperada + 1, FechaUTC: time.Now().UTC(), Replay: true}}, nil
	}
	r.reciboOriginal = ports.ReciboCorreos{ReciboRef: "recibo:uno", PersonaRef: r.persona, Accion: m.Accion, CorreoRef: p.CorreoRef, Version: p.VersionEsperada + 1, FechaUTC: time.Now().UTC()}
	resultado := ports.ResultadoCorreos{Recibo: r.reciboOriginal}
	if r.sinEnvios {
		return resultado, nil
	}
	switch m.Accion {
	case ports.AccionAnadirCorreo, ports.AccionReenviarCorreo:
		resultado.Envios = []ports.EnvioPendiente{{EnvioRef: "correo_envio:" + strings.Repeat("e", 32), ReservaRef: "reserva:" + strings.Repeat("f", 32), Tipo: ports.TipoEnvioCodigo, CorreoRef: p.CorreoRef, DesafioRef: reserva.DesafioRef, Sobre: sobre}}
	case ports.AccionActivarCorreo:
		resultado.Envios = []ports.EnvioPendiente{{EnvioRef: "correo_envio:" + strings.Repeat("e", 32), ReservaRef: "reserva:" + strings.Repeat("f", 32), Tipo: ports.TipoEnvioAviso, CorreoRef: "correo:" + strings.Repeat("9", 32)}}
	}
	return resultado, nil
}
func (r *registroCorreosPrueba) ConfirmarEnvio(_ context.Context, _ ports.OrdenCorreos, _ ports.EnvioPendiente, aceptado bool) error {
	r.confirmados = append(r.confirmados, aceptado)
	return nil
}

type revalidadorCorreoPrueba struct {
	auth vecdomain.AutenticacionRevalidadaV1
}

func (r revalidadorCorreoPrueba) RevalidarAutenticacionActorV1(context.Context, vecdomain.SolicitudRevalidacionAutenticacionActorV1) (vecdomain.AutenticacionRevalidadaV1, error) {
	return r.auth, nil
}

type resolutorCorreoPrueba struct {
	res vecdomain.ResultadoContextoActorRegistradoV2
}

func (r resolutorCorreoPrueba) ResolverContextoActorRegistradoV2(context.Context, vecdomain.SolicitudContextoActor) (vecdomain.ResultadoContextoActorRegistradoV2, error) {
	return r.res, nil
}

type relojCorreoPrueba struct{ instante time.Time }

func (r relojCorreoPrueba) Ahora() time.Time { return r.instante }

func identidadCorreosPrueba(t *testing.T, superficie vecdomain.SuperficieAutenticacionActorV1, perfilLetra string) (vecdomain.ContextoActor, vecdomain.VinculoAutenticacionActorV2) {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	z := strings.Repeat("a", 24)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	snap := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + strings.Repeat("r", 24), PersonaVersion: 1, PerfilActivoRef: "prf_" + strings.Repeat(perfilLetra, 24), PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}
	actor, err := vecdomain.NuevoContextoActor(cuenta, snap, ahora)
	if err != nil {
		t.Fatal(err)
	}
	canon, _ := actor.RepresentacionCanonicaVinculadaV2()
	huella, _ := actor.HuellaSHA256VinculadaV2()
	ac := vecdomain.AcreditacionProcedenciaComponenteContextoActorV1{ProcedenciaRef: "prc_" + z, ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("a", 64), ProcedenciaAutoridad: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1}
	man := vecdomain.ManifiestoProcedenciaContextoActorV1{Esquema: vecdomain.EsquemaManifiestoProcedenciaContextoActorV1, AutoridadEfectiva: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, Cuenta: vecdomain.ProcedenciaCuentaContextoActorV1{CuentaRef: cuenta.CuentaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Persona: vecdomain.ProcedenciaPersonaContextoActorV1{PersonaRef: actor.PersonaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Perfil: vecdomain.ProcedenciaPerfilContextoActorV1{PerfilRef: actor.PerfilActivoRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Contexto: vecdomain.ProcedenciaVinculoContextoActorV1{VinculoRef: snap.VinculoRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Vinculos: []vecdomain.ProcedenciaVinculoReferenciaContextoActorV1{}}
	bm, _ := man.RepresentacionCanonicaV1()
	hm, _ := vecdomain.HuellaSHA256ManifiestoProcedenciaContextoActorV1(bm)
	res := vecdomain.ResultadoContextoActorRegistradoV2{RegistroContextoRef: "rca_" + z, Contexto: actor, RepresentacionCanonica: canon, HuellaSHA256: huella, ManifiestoProcedenciaCanonico: bm, ManifiestoProcedenciaHuellaSHA256: hm, AutoridadEfectiva: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, ResueltoEnAutoritativo: ahora}
	if err := res.Validar(); err != nil {
		t.Fatal(err)
	}
	auth := vecdomain.AutenticacionRevalidadaV1{AutenticacionRef: "aut_" + z, AutenticacionHuellaSHA256: strings.Repeat("a", 64), AsercionRef: "ase_" + z, SesionRef: "ses_" + z, ControlSesionRef: "cse_" + z, ControlSesionRevision: 1, ControlSesionHuellaSHA256: strings.Repeat("b", 64), CuentaRef: cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef, Superficie: superficie, MetodoObservado: vecdomain.AuthMethodCertificate, GarantiaObservada: vecdomain.AuthAssuranceHigh, PoliticaGarantiaRef: "pga_" + z, PoliticaGarantiaHuellaSHA256: strings.Repeat("c", 64), AutenticacionVerificadaEn: ahora.Add(-time.Minute), SesionEmitidaEn: ahora.Add(-time.Minute), SesionRevalidadaEn: ahora.Add(-time.Second), SesionValidaHasta: ahora.Add(time.Minute)}
	if err := auth.Validar(); err != nil {
		t.Fatal(err)
	}
	v, err := vecdomain.CrearVinculoAutenticacionActorV2(context.Background(), revalidadorCorreoPrueba{auth}, vecdomain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: auth.AutenticacionRef, SesionRef: auth.SesionRef}, resolutorCorreoPrueba{res}, vecdomain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: actor.PerfilActivoRef}, relojCorreoPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	return actor, v
}

type entornoCorreosPrueba struct {
	orden      ports.OrdenCorreos
	proveedor  *proveedorCorreosPrueba
	registro   *registroCorreosPrueba
	desafio    *desafioCorreosPrueba
	validador  *validadorCorreosPrueba
	protector  *protectorCorreosPrueba
	transporte *transporteCorreosPrueba
	servicio   *ServicioCorreos
}

func (e *entornoCorreosPrueba) dependencias() DependenciasCorreos {
	return DependenciasCorreos{Registro: e.registro, Protector: e.protector, Sellador: selladorCorreosPrueba{}, Desafios: e.desafio, Validador: e.validador, Transporte: e.transporte, AhoraUTC: time.Now}
}

func prepararServicioCorreos(t *testing.T) *entornoCorreosPrueba {
	t.Helper()
	actor, vinculo := identidadCorreosPrueba(t, vecdomain.SuperficieAutenticacionInternaCorporativaV1, "p")
	e := &entornoCorreosPrueba{proveedor: &proveedorCorreosPrueba{}, desafio: &desafioCorreosPrueba{}, validador: &validadorCorreosPrueba{}, protector: &protectorCorreosPrueba{}, transporte: &transporteCorreosPrueba{}}
	var err error
	e.orden, err = NuevaOrdenCorreos(actor, vinculo, vecdomain.SuperficieAutenticacionInternaCorporativaV1, e.proveedor)
	if err != nil {
		t.Fatal(err)
	}
	e.registro = &registroCorreosPrueba{persona: actor.PersonaRef}
	e.servicio, err = NuevoServicioCorreos(e.dependencias())
	if err != nil {
		t.Fatal(err)
	}
	return e
}

const clavePrueba = "operacion-1234567890"

func TestAltaEnviaCodigoTrasCommitYReplayNoReenvia(t *testing.T) {
	e := prepararServicioCorreos(t)
	peticion := ports.PeticionCorreo{ClaveOperacion: clavePrueba, Direccion: " persona@example.org "}
	recibo, err := e.servicio.Anadir(context.Background(), e.orden, peticion)
	if err != nil {
		t.Fatal(err)
	}
	if !ReferenciaCorreoValida(recibo.CorreoRef) || e.registro.aplicaciones != 1 || e.desafio.llamados != 1 || e.registro.reserva.VenceUTC.Before(time.Now().Add(59*time.Minute)) || recibo.Envio != ports.EnvioAceptado {
		t.Fatalf("alta sin desafío de una hora o sin envío: %+v", recibo)
	}
	if e.registro.reserva.Codigo != "" || e.registro.peticion.Direccion != "" || string(e.registro.sobre.Cifrado) != "persona@example.org" {
		t.Fatal("el registro recibió código o dirección en claro")
	}
	if len(e.transporte.mensajes) != 1 || e.transporte.mensajes[0].Codigo != "12345678" || e.transporte.mensajes[0].Destino != "destino@example.org" || e.transporte.mensajes[0].Tipo != ports.TipoEnvioCodigo || len(e.registro.confirmados) != 1 || !e.registro.confirmados[0] {
		t.Fatalf("envío del código incorrecto: %+v %v", e.transporte.mensajes, e.registro.confirmados)
	}
	if !canonico.HuellasSemanticasValidas(e.registro.material.HuellasPeticion) || e.proveedor.materiales[0].PersonaRef != e.registro.persona {
		t.Fatal("material semántico/identidad incorrectos")
	}
	e.registro.replay = true
	recibo, err = e.servicio.Anadir(context.Background(), e.orden, peticion)
	if err != nil || !recibo.Replay || recibo.Envio != ports.EnvioSinCorreo || e.desafio.llamados != 1 || e.registro.aplicaciones != 1 || len(e.transporte.mensajes) != 1 {
		t.Fatalf("replay regeneró desafío o reenvió: %v", err)
	}
}

func TestRelayRechazadoQuedaAnotadoSinDeshacerAlta(t *testing.T) {
	e := prepararServicioCorreos(t)
	e.transporte.rechazar = true
	recibo, err := e.servicio.Anadir(context.Background(), e.orden, ports.PeticionCorreo{ClaveOperacion: clavePrueba, Direccion: "persona@example.org"})
	if err != nil || recibo.Envio != ports.EnvioNoAceptado || len(e.registro.confirmados) != 1 || e.registro.confirmados[0] {
		t.Fatalf("rechazo del relay mal anotado: %+v %v %v", recibo, e.registro.confirmados, err)
	}
	e.transporte.rechazar = false
	e.protector.fallarDescifrado = true
	recibo, err = e.servicio.Reenviar(context.Background(), e.orden, ports.PeticionCorreo{ClaveOperacion: "otra-" + clavePrueba, CorreoRef: refCorreoPrueba, VersionEsperada: 1})
	if err != nil || recibo.Envio != ports.EnvioNoAceptado || len(e.transporte.mensajes) != 1 || e.registro.confirmados[1] {
		t.Fatalf("fallo de descifrado mal tratado: %+v %v", recibo, err)
	}
}

func TestActivarAvisaAlCorreoAnteriorSinCodigo(t *testing.T) {
	e := prepararServicioCorreos(t)
	recibo, err := e.servicio.Activar(context.Background(), e.orden, ports.PeticionCorreo{ClaveOperacion: clavePrueba, CorreoRef: refCorreoPrueba, VersionEsperada: 4})
	if err != nil || recibo.Envio != ports.EnvioSinCorreo || len(e.transporte.mensajes) != 1 || e.transporte.mensajes[0].Tipo != ports.TipoEnvioAviso || e.transporte.mensajes[0].Codigo != "" || e.desafio.llamados != 0 {
		t.Fatalf("aviso de cambio incorrecto: %+v %+v %v", recibo, e.transporte.mensajes, err)
	}
	e.registro.sinEnvios = true
	if _, err := e.servicio.Activar(context.Background(), e.orden, ports.PeticionCorreo{ClaveOperacion: "otra-" + clavePrueba, CorreoRef: refCorreoPrueba, VersionEsperada: 5}); err != nil || len(e.transporte.mensajes) != 1 {
		t.Fatalf("sin activo anterior se envió aviso: %v", err)
	}
}

func TestVerificacionNoEntregaCodigoAlRegistro(t *testing.T) {
	e := prepararServicioCorreos(t)
	peticion := ports.PeticionCorreo{ClaveOperacion: clavePrueba, CorreoRef: refCorreoPrueba, Codigo: "8765 4321", VersionEsperada: 1}
	recibo, err := e.servicio.Verificar(context.Background(), e.orden, peticion)
	if err != nil || !e.registro.comprobacion || e.validador.llamados != 1 || e.registro.peticion.Codigo != "" || len(e.transporte.mensajes) != 0 || recibo.Envio != ports.EnvioSinCorreo {
		t.Fatalf("codigo salió del callback: %v", err)
	}
	peticion.Codigo, peticion.ClaveOperacion = "11112222", "otra-"+clavePrueba
	_, err = e.servicio.Verificar(context.Background(), e.orden, peticion)
	var incorrecto ports.CodigoIncorrecto
	if !errors.Is(err, ports.ErrCorreosCodigoIncorrecto) || !errors.As(err, &incorrecto) || incorrecto.IntentosRestantes != 3 {
		t.Fatalf("código incorrecto mal informado: %v", err)
	}
	e.registro.replay = true
	_, err = e.servicio.Verificar(context.Background(), e.orden, peticion)
	if err != nil || e.validador.llamados != 2 || e.registro.aplicaciones != 2 {
		t.Fatalf("replay volvió a validar código: %v", err)
	}
	for _, codigo := range []string{"1234567", "123456789", "abcd1234", ""} {
		peticion.Codigo = codigo
		if _, err := e.servicio.Verificar(context.Background(), e.orden, peticion); !errors.Is(err, ports.ErrCorreosInvalidos) {
			t.Fatalf("código mal formado %q aceptado: %v", codigo, err)
		}
	}
}

func TestDenegacionPrevieneLecturaYMutacion(t *testing.T) {
	e := prepararServicioCorreos(t)
	e.proveedor.denegar = true
	if _, err := e.servicio.Consultar(context.Background(), e.orden); !errors.Is(err, ports.ErrCorreosProhibido) {
		t.Fatalf("denegación: %v", err)
	}
	peticion := ports.PeticionCorreo{ClaveOperacion: clavePrueba, Direccion: "persona@example.org"}
	if _, err := e.servicio.Anadir(context.Background(), e.orden, peticion); err == nil || e.registro.consultas != 0 || e.registro.recuperaciones != 0 || e.desafio.llamados != 0 || len(e.transporte.mensajes) != 0 {
		t.Fatalf("denegación permitió efectos: %v", err)
	}
}

func TestPeticionesMalFormadasNoLleganAlRegistro(t *testing.T) {
	e := prepararServicioCorreos(t)
	casos := []struct {
		accion string
		p      ports.PeticionCorreo
	}{
		{"anadir", ports.PeticionCorreo{ClaveOperacion: clavePrueba, Direccion: "Nombre <a@example.org>"}},
		{"anadir", ports.PeticionCorreo{ClaveOperacion: clavePrueba, Direccion: "a@example.org", CorreoRef: refCorreoPrueba}},
		{"anadir", ports.PeticionCorreo{ClaveOperacion: "corta", Direccion: "a@example.org"}},
		{"reenviar", ports.PeticionCorreo{ClaveOperacion: clavePrueba, CorreoRef: "correo:uno"}},
		{"activar", ports.PeticionCorreo{ClaveOperacion: clavePrueba, CorreoRef: refCorreoPrueba, Direccion: "a@example.org"}},
		{"retirar", ports.PeticionCorreo{ClaveOperacion: clavePrueba, CorreoRef: refCorreoPrueba, Codigo: "12345678"}},
		{"otra", ports.PeticionCorreo{ClaveOperacion: clavePrueba, CorreoRef: refCorreoPrueba}},
	}
	for _, c := range casos {
		if _, err := e.servicio.Operar(context.Background(), e.orden, c.accion, c.p); !errors.Is(err, ports.ErrCorreosInvalidos) {
			t.Fatalf("%s %+v aceptada: %v", c.accion, c.p, err)
		}
	}
	if len(e.proveedor.materiales) != 0 || e.registro.recuperaciones != 0 {
		t.Fatal("una petición inválida pidió V3 o tocó el registro")
	}
}

func TestAltaSinGeneracionIgualdadFallaCerrada(t *testing.T) {
	e := prepararServicioCorreos(t)
	e.protector.sinClaveIgualdad = true
	_, err := e.servicio.Anadir(context.Background(), e.orden, ports.PeticionCorreo{ClaveOperacion: clavePrueba, Direccion: "persona@example.org"})
	if !errors.Is(err, ports.ErrCorreosNoDisponible) || e.registro.aplicaciones != 0 || e.desafio.llamados != 0 {
		t.Fatalf("alta sin clave HMAC igualdad llegó a efecto: %v", err)
	}
}

func TestReplayConcurrenteConservaReciboOriginal(t *testing.T) {
	e := prepararServicioCorreos(t)
	e.registro.aplicarReplay = true // Otro escritor insertó misma clave+huella entre Recuperar y Aplicar.
	recibo, err := e.servicio.Anadir(context.Background(), e.orden, ports.PeticionCorreo{ClaveOperacion: clavePrueba, Direccion: "persona@example.org"})
	if err != nil || !recibo.Replay || recibo.ReciboRef != "recibo:original" || recibo.CorreoRef != refCorreoPrueba || e.registro.peticion.CorreoRef == recibo.CorreoRef || e.desafio.llamados != 1 || len(e.transporte.mensajes) != 0 {
		t.Fatalf("replay concurrente rechazado o sobrescrito: %+v, %v", recibo, err)
	}
}

func TestReplayTrasRotacionMantieneHuellaHistorica(t *testing.T) {
	e := prepararServicioCorreos(t)
	e.registro.replay = true
	d := e.dependencias()
	d.Sellador = selladorRotadoCorreosPrueba{}
	servicio, err := NuevoServicioCorreos(d)
	if err != nil {
		t.Fatal(err)
	}
	recibo, err := servicio.Anadir(context.Background(), e.orden, ports.PeticionCorreo{ClaveOperacion: clavePrueba, Direccion: "persona@example.org"})
	if err != nil || !recibo.Replay || e.registro.aplicaciones != 0 || e.desafio.llamados != 0 || len(e.registro.material.HuellasPeticion.Retenidas) != 1 || e.registro.material.HuellasPeticion.Retenidas[0].ClaveRef != "clave:huella:v1" {
		t.Fatalf("rotación impidió replay o reservó efecto: %+v, %v", e.registro.material, err)
	}
}

func TestReplayEntreSuperficiesReautorizaYConservaRecibo(t *testing.T) {
	e := prepararServicioCorreos(t)
	peticion := ports.PeticionCorreo{ClaveOperacion: clavePrueba, Direccion: "persona@example.org"}
	primero, err := e.servicio.Anadir(context.Background(), e.orden, peticion)
	if err != nil {
		t.Fatal(err)
	}
	huella := e.proveedor.materiales[0].HuellasPeticion.Activa
	e.registro.replay = true
	actorExterno, vinculoExterno := identidadCorreosPrueba(t, vecdomain.SuperficieAutenticacionExternaPersonalV1, "q")
	ordenExterna, err := NuevaOrdenCorreos(actorExterno, vinculoExterno, vecdomain.SuperficieAutenticacionExternaPersonalV1, e.proveedor)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := e.servicio.Anadir(context.Background(), ordenExterna, peticion)
	if err != nil || !segundo.Replay || segundo.ReciboRef != primero.ReciboRef || segundo.CorreoRef != primero.CorreoRef || e.registro.aplicaciones != 1 || e.desafio.llamados != 1 {
		t.Fatalf("replay cruzado alteró efecto/recibo: %+v, %v", segundo, err)
	}
	if e.proveedor.materiales[2].HuellasPeticion.Activa != huella || e.proveedor.materiales[2].PerfilRef == e.proveedor.materiales[0].PerfilRef || e.proveedor.materiales[2].Superficie == e.proveedor.materiales[0].Superficie || bytes.Equal(e.proveedor.bytes[0], e.proveedor.bytes[2]) {
		t.Fatal("huella semántica mezcló perfil/superficie o V3 no los distinguió")
	}
	if len(e.proveedor.materiales) != 3 {
		t.Fatal("el segundo portal no obtuvo V3 propia")
	}
}

func TestSustitucionDeSuperficieYAudienciaSeDeniega(t *testing.T) {
	actorInterno, vinculoInterno := identidadCorreosPrueba(t, vecdomain.SuperficieAutenticacionInternaCorporativaV1, "p")
	actorExterno, _ := identidadCorreosPrueba(t, vecdomain.SuperficieAutenticacionExternaPersonalV1, "q")
	proveedor := &proveedorCorreosPrueba{}
	if _, err := NuevaOrdenCorreos(actorInterno, vinculoInterno, vecdomain.SuperficieAutenticacionExternaPersonalV1, proveedor); !errors.Is(err, ports.ErrCorreosProhibido) {
		t.Fatalf("ruta sustituida: %v", err)
	}
	if _, err := NuevaOrdenCorreos(actorExterno, vinculoInterno, vecdomain.SuperficieAutenticacionInternaCorporativaV1, proveedor); !errors.Is(err, ports.ErrCorreosNoAutenticado) {
		t.Fatalf("perfil sustituido: %v", err)
	}
	e := prepararServicioCorreos(t)
	e.proveedor.audienciaForzada = ports.AudienciaConsultarCorreosExterna
	if _, err := e.servicio.Consultar(context.Background(), e.orden); !errors.Is(err, ports.ErrCorreosNoDisponible) || e.registro.consultas != 0 {
		t.Fatalf("audiencia cruzada aceptada: %v", err)
	}
}

func TestSHADeMaterialDirectoNoEsHuellaDeEfectoV3(t *testing.T) {
	e := prepararServicioCorreos(t)
	e.proveedor.huellaForzada = "material_directo"
	_, err := e.servicio.Consultar(context.Background(), e.orden)
	if !errors.Is(err, ports.ErrCorreosNoDisponible) || e.registro.consultas != 0 {
		t.Fatalf("SHA material se aceptó como contexto V3: %v", err)
	}
}

func TestVencimientoCanonicoConfiguradoYReservaExacta(t *testing.T) {
	e := prepararServicioCorreos(t)
	instante := time.Now().UTC().Truncate(time.Microsecond).Add(789 * time.Nanosecond)
	d := e.dependencias()
	d.AhoraUTC = func() time.Time { return instante }
	d.Politica = PoliticaDesafioCorreo{Vigencia: 2 * time.Hour}
	servicio, err := NuevoServicioCorreos(d)
	if err != nil {
		t.Fatal(err)
	}
	esperado := instante.Truncate(time.Microsecond).Add(2 * time.Hour)
	_, err = servicio.Anadir(context.Background(), e.orden, ports.PeticionCorreo{ClaveOperacion: clavePrueba, Direccion: "persona@example.org"})
	if err != nil || !e.desafio.venceRecibido.Equal(esperado) || e.desafio.venceRecibido.Nanosecond()%1000 != 0 {
		t.Fatalf("vencimiento no canónico/configurado: %v, %v", e.desafio.venceRecibido, err)
	}
	e.desafio.desfase = time.Nanosecond
	_, err = servicio.Anadir(context.Background(), e.orden, ports.PeticionCorreo{ClaveOperacion: "otra-operacion-1234567890", Direccion: "otra@example.org"})
	if !errors.Is(err, ports.ErrCorreosNoDisponible) || e.registro.aplicaciones != 1 {
		t.Fatalf("reserva con vencimiento distinto aceptada: %v", err)
	}
	d.Politica = PoliticaDesafioCorreo{Vigencia: time.Nanosecond}
	if _, err = NuevoServicioCorreos(d); !errors.Is(err, ports.ErrCorreosNoDisponible) {
		t.Fatalf("política inválida aceptada: %v", err)
	}
	d.Politica, d.Transporte = PoliticaDesafioCorreo{}, nil
	if _, err = NuevoServicioCorreos(d); !errors.Is(err, ports.ErrCorreosNoDisponible) {
		t.Fatalf("servicio sin transporte aceptado: %v", err)
	}
}
