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

	"vec-diputacion-granada/internal/modules/usuarios/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type proveedorCorreosPrueba struct {
	denegar          bool
	audienciaForzada string
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
	b, err := ports.SerializarMaterialCorreos(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	p.bytes = append(p.bytes, b)
	h := sha256.Sum256(b)
	audiencia, _ := ports.AudienciaCorreos(m.Accion, m.Superficie)
	if p.audienciaForzada != "" {
		audiencia = p.audienciaForzada
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(fmt.Sprintf("dec_correos_%d", len(p.materiales)), strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), m.Accion, m.PersonaRef, hex.EncodeToString(h[:]), audiencia, ahora, ahora.Add(3*time.Second))
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	return vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
}

type desafioCorreosPrueba struct {
	llamados      int
	desfase       time.Duration
	venceRecibido time.Time
}

type protectorCorreosPrueba struct{ llamados int }

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
	return ports.SobreDireccionCorreo{CorreoRef: ref, Version: version, ClaveRef: "clave:correo", Nonce: []byte(strings.Repeat("n", 12)), Cifrado: []byte(strings.Repeat("c", 32)), HuellaIgualdad: []byte(strings.Repeat("h", 32))}, nil
}

func (d *desafioCorreosPrueba) PrepararDesafioCorreo(_ context.Context, _, _ string, vence time.Time) (ports.ReservaDesafio, error) {
	d.llamados++
	d.venceRecibido = vence
	return ports.ReservaDesafio{DesafioRef: "desafio:uno", Desafio: []byte(strings.Repeat("d", 16)), HuellaCodigo: []byte(strings.Repeat("h", 32)), ClaveRef: "clave:uno", VenceUTC: vence.Add(d.desfase)}, nil
}

type validadorCorreosPrueba struct{ llamados int }

func (v *validadorCorreosPrueba) ComprobarCodigoCorreo(_ context.Context, _ ports.MetadatosDesafioCorreo, codigo string) (bool, error) {
	v.llamados++
	return codigo == "CODIGO-1234567890", nil
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
}

func (r *registroCorreosPrueba) ConsultarPropios(_ context.Context, _ ports.OrdenCorreos, m ports.MaterialCorreos, _ vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.VistaCorreos, error) {
	r.consultas++
	r.material = m
	return ports.VistaCorreos{PersonaRef: r.persona}, nil
}
func (r *registroCorreosPrueba) RecuperarOperacion(_ context.Context, _ ports.OrdenCorreos, m ports.MaterialCorreos, _ vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboCorreos, bool, error) {
	r.recuperaciones++
	r.material = m
	if r.replay {
		if r.reciboOriginal.ReciboRef != "" {
			return r.reciboOriginal, true, nil
		}
		return ports.ReciboCorreos{ReciboRef: "recibo:uno", PersonaRef: r.persona, Accion: m.Accion, CorreoRef: "correo:1234567890123456", Version: m.VersionEsperada + 1, FechaUTC: time.Now().UTC()}, true, nil
	}
	return ports.ReciboCorreos{}, false, nil
}
func (r *registroCorreosPrueba) Aplicar(ctx context.Context, _ ports.OrdenCorreos, p ports.PeticionCorreo, m ports.MaterialCorreos, _ vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, sobre ports.SobreDireccionCorreo, reserva ports.ReservaDesafio, comprobador ports.ComprobadorCodigoCorreo) (ports.ReciboCorreos, error) {
	r.aplicaciones++
	r.peticion, r.material, r.sobre, r.reserva = p, m, sobre, reserva
	if m.Accion == ports.AccionVerificarCorreo {
		var err error
		r.comprobacion, err = comprobador.Comprobar(ctx, ports.MetadatosDesafioCorreo{PersonaRef: r.persona, CorreoRef: p.CorreoRef})
		if err != nil || !r.comprobacion {
			return ports.ReciboCorreos{}, ports.ErrCorreosInvalidos
		}
	}
	if r.aplicarReplay {
		return ports.ReciboCorreos{ReciboRef: "recibo:original", PersonaRef: r.persona, Accion: m.Accion, CorreoRef: "correo:1234567890123456", Version: p.VersionEsperada + 1, FechaUTC: time.Now().UTC(), Replay: true}, nil
	}
	r.reciboOriginal = ports.ReciboCorreos{ReciboRef: "recibo:uno", PersonaRef: r.persona, Accion: m.Accion, CorreoRef: p.CorreoRef, Version: p.VersionEsperada + 1, FechaUTC: time.Now().UTC()}
	return r.reciboOriginal, nil
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

func prepararServicioCorreos(t *testing.T) (ports.OrdenCorreos, *proveedorCorreosPrueba, *registroCorreosPrueba, *desafioCorreosPrueba, *validadorCorreosPrueba, *ServicioCorreos) {
	t.Helper()
	actor, vinculo := identidadCorreosPrueba(t, vecdomain.SuperficieAutenticacionInternaCorporativaV1, "p")
	proveedor := &proveedorCorreosPrueba{}
	orden, err := ports.NuevaOrdenCorreos(actor, vinculo, vecdomain.SuperficieAutenticacionInternaCorporativaV1, proveedor)
	if err != nil {
		t.Fatal(err)
	}
	registro := &registroCorreosPrueba{persona: actor.PersonaRef}
	desafio := &desafioCorreosPrueba{}
	validador := &validadorCorreosPrueba{}
	servicio, err := NuevoServicioCorreos(registro, &protectorCorreosPrueba{}, selladorCorreosPrueba{}, desafio, validador, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	return orden, proveedor, registro, desafio, validador, servicio
}

func TestAltaPendienteYReplayNoRegeneranDesafio(t *testing.T) {
	orden, proveedor, registro, desafio, _, servicio := prepararServicioCorreos(t)
	peticion := ports.PeticionCorreo{ClaveOperacion: "operacion-1234567890", Direccion: "persona@example.org"}
	recibo, err := servicio.Anadir(context.Background(), orden, peticion)
	if err != nil {
		t.Fatal(err)
	}
	if recibo.CorreoRef == "" || registro.aplicaciones != 1 || desafio.llamados != 1 || registro.reserva.VenceUTC.Before(time.Now().Add(23*time.Hour)) {
		t.Fatal("alta no reservó desafío de 24 horas")
	}
	if registro.peticion.Direccion != "" || registro.sobre.CorreoRef != recibo.CorreoRef || registro.sobre.Version != 1 || !registro.material.HuellasPeticion.Validar() || strings.Contains(registro.material.HuellasPeticion.Activa.Valor, peticion.Direccion) || proveedor.materiales[0].PersonaRef != registro.persona {
		t.Fatal("material semántico/identidad incorrectos")
	}
	registro.replay = true
	recibo, err = servicio.Anadir(context.Background(), orden, peticion)
	if err != nil || !recibo.Replay || desafio.llamados != 1 || registro.aplicaciones != 1 {
		t.Fatalf("replay regeneró desafío: %v", err)
	}
}

func TestVerificacionNoEntregaCodigoAlRegistro(t *testing.T) {
	orden, _, registro, _, validador, servicio := prepararServicioCorreos(t)
	peticion := ports.PeticionCorreo{ClaveOperacion: "operacion-1234567890", CorreoRef: "correo:1234567890123456", Codigo: "CODIGO-1234567890"}
	_, err := servicio.Verificar(context.Background(), orden, peticion)
	if err != nil || !registro.comprobacion || validador.llamados != 1 || registro.peticion.Codigo != "" || strings.Contains(registro.material.HuellasPeticion.Activa.Valor, peticion.Codigo) {
		t.Fatalf("codigo salió del callback: %v", err)
	}
	registro.replay = true
	_, err = servicio.Verificar(context.Background(), orden, peticion)
	if err != nil || validador.llamados != 1 || registro.aplicaciones != 1 {
		t.Fatalf("replay volvió a validar código: %v", err)
	}
}

func TestDenegacionPrevieneLecturaYMutacion(t *testing.T) {
	orden, proveedor, registro, desafio, _, servicio := prepararServicioCorreos(t)
	proveedor.denegar = true
	if _, err := servicio.Consultar(context.Background(), orden); !errors.Is(err, ports.ErrCorreosProhibido) {
		t.Fatalf("denegación: %v", err)
	}
	peticion := ports.PeticionCorreo{ClaveOperacion: "operacion-1234567890", Direccion: "persona@example.org"}
	if _, err := servicio.Anadir(context.Background(), orden, peticion); err == nil || registro.consultas != 0 || registro.recuperaciones != 0 || desafio.llamados != 0 {
		t.Fatalf("denegación permitió efectos: %v", err)
	}
}

func TestReplayConcurrenteConservaReciboOriginal(t *testing.T) {
	orden, _, registro, desafio, _, servicio := prepararServicioCorreos(t)
	registro.aplicarReplay = true // Otro escritor insertó misma clave+huella entre Recuperar y Aplicar.
	recibo, err := servicio.Anadir(context.Background(), orden, ports.PeticionCorreo{ClaveOperacion: "operacion-1234567890", Direccion: "persona@example.org"})
	if err != nil || !recibo.Replay || recibo.ReciboRef != "recibo:original" || recibo.CorreoRef != "correo:1234567890123456" || registro.peticion.CorreoRef == recibo.CorreoRef || desafio.llamados != 1 {
		t.Fatalf("replay concurrente rechazado o sobrescrito: %+v, %v", recibo, err)
	}
}

func TestReplayTrasRotacionMantieneHuellaHistorica(t *testing.T) {
	orden, _, registro, desafio, validador, _ := prepararServicioCorreos(t)
	registro.replay = true
	servicio, err := NuevoServicioCorreos(registro, &protectorCorreosPrueba{}, selladorRotadoCorreosPrueba{}, desafio, validador, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	recibo, err := servicio.Anadir(context.Background(), orden, ports.PeticionCorreo{ClaveOperacion: "operacion-1234567890", Direccion: "persona@example.org"})
	if err != nil || !recibo.Replay || registro.aplicaciones != 0 || desafio.llamados != 0 || len(registro.material.HuellasPeticion.Retenidas) != 1 || registro.material.HuellasPeticion.Retenidas[0].ClaveRef != "clave:huella:v1" {
		t.Fatalf("rotación impidió replay o reservó efecto: %+v, %v", registro.material, err)
	}
}

func TestReplayEntreSuperficiesReautorizaYConservaRecibo(t *testing.T) {
	ordenInterna, proveedor, registro, desafio, _, servicio := prepararServicioCorreos(t)
	peticion := ports.PeticionCorreo{ClaveOperacion: "operacion-1234567890", Direccion: "persona@example.org"}
	primero, err := servicio.Anadir(context.Background(), ordenInterna, peticion)
	if err != nil {
		t.Fatal(err)
	}
	huella := proveedor.materiales[0].HuellasPeticion.Activa
	registro.replay = true
	actorExterno, vinculoExterno := identidadCorreosPrueba(t, vecdomain.SuperficieAutenticacionExternaPersonalV1, "q")
	ordenExterna, err := ports.NuevaOrdenCorreos(actorExterno, vinculoExterno, vecdomain.SuperficieAutenticacionExternaPersonalV1, proveedor)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := servicio.Anadir(context.Background(), ordenExterna, peticion)
	if err != nil || !segundo.Replay || segundo.ReciboRef != primero.ReciboRef || segundo.CorreoRef != primero.CorreoRef || registro.aplicaciones != 1 || desafio.llamados != 1 {
		t.Fatalf("replay cruzado alteró efecto/recibo: %+v, %v", segundo, err)
	}
	if proveedor.materiales[2].HuellasPeticion.Activa != huella || proveedor.materiales[2].PerfilRef == proveedor.materiales[0].PerfilRef || proveedor.materiales[2].Superficie == proveedor.materiales[0].Superficie || bytes.Equal(proveedor.bytes[0], proveedor.bytes[2]) {
		t.Fatal("huella semántica mezcló perfil/superficie o V3 no los distinguió")
	}
	if len(proveedor.materiales) != 3 {
		t.Fatal("el segundo portal no obtuvo V3 propia")
	}
}

func TestSustitucionDeSuperficieYAudienciaSeDeniega(t *testing.T) {
	actorInterno, vinculoInterno := identidadCorreosPrueba(t, vecdomain.SuperficieAutenticacionInternaCorporativaV1, "p")
	actorExterno, _ := identidadCorreosPrueba(t, vecdomain.SuperficieAutenticacionExternaPersonalV1, "q")
	proveedor := &proveedorCorreosPrueba{}
	if _, err := ports.NuevaOrdenCorreos(actorInterno, vinculoInterno, vecdomain.SuperficieAutenticacionExternaPersonalV1, proveedor); !errors.Is(err, ports.ErrCorreosProhibido) {
		t.Fatalf("ruta sustituida: %v", err)
	}
	if _, err := ports.NuevaOrdenCorreos(actorExterno, vinculoInterno, vecdomain.SuperficieAutenticacionInternaCorporativaV1, proveedor); !errors.Is(err, ports.ErrCorreosNoAutenticado) {
		t.Fatalf("perfil sustituido: %v", err)
	}
	orden, err := ports.NuevaOrdenCorreos(actorInterno, vinculoInterno, vecdomain.SuperficieAutenticacionInternaCorporativaV1, proveedor)
	if err != nil {
		t.Fatal(err)
	}
	proveedor.audienciaForzada = ports.AudienciaConsultarCorreosExterna
	registro := &registroCorreosPrueba{persona: actorInterno.PersonaRef}
	servicio, _ := NuevoServicioCorreos(registro, &protectorCorreosPrueba{}, selladorCorreosPrueba{}, &desafioCorreosPrueba{}, &validadorCorreosPrueba{}, time.Now)
	if _, err := servicio.Consultar(context.Background(), orden); !errors.Is(err, ports.ErrCorreosNoDisponible) || registro.consultas != 0 {
		t.Fatalf("audiencia cruzada aceptada: %v", err)
	}
}

func TestVencimientoCanonicoConfiguradoYReservaExacta(t *testing.T) {
	orden, _, registro, desafio, validador, _ := prepararServicioCorreos(t)
	instante := time.Now().UTC().Truncate(time.Microsecond).Add(789 * time.Nanosecond)
	servicio, err := NuevoServicioCorreosConPolitica(registro, &protectorCorreosPrueba{}, selladorCorreosPrueba{}, desafio, validador, func() time.Time { return instante }, PoliticaDesafioCorreo{Vigencia: 2 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	esperado := instante.Truncate(time.Microsecond).Add(2 * time.Hour)
	_, err = servicio.Anadir(context.Background(), orden, ports.PeticionCorreo{ClaveOperacion: "operacion-1234567890", Direccion: "persona@example.org"})
	if err != nil || !desafio.venceRecibido.Equal(esperado) || desafio.venceRecibido.Nanosecond()%1000 != 0 {
		t.Fatalf("vencimiento no canónico/configurado: %v, %v", desafio.venceRecibido, err)
	}
	desafio.desfase = time.Nanosecond
	_, err = servicio.Anadir(context.Background(), orden, ports.PeticionCorreo{ClaveOperacion: "otra-operacion-1234567890", Direccion: "otra@example.org"})
	if !errors.Is(err, ports.ErrCorreosNoDisponible) || registro.aplicaciones != 1 {
		t.Fatalf("reserva con vencimiento distinto aceptada: %v", err)
	}
	_, err = NuevoServicioCorreosConPolitica(registro, &protectorCorreosPrueba{}, selladorCorreosPrueba{}, desafio, validador, time.Now, PoliticaDesafioCorreo{Vigencia: time.Nanosecond})
	if !errors.Is(err, ports.ErrCorreosNoDisponible) {
		t.Fatalf("política inválida aceptada: %v", err)
	}
}
