package application

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/usuarios/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type proveedorCorreosPrueba struct {
	base       proveedorPrueba
	materiales []ports.MaterialCorreos
}

func (p *proveedorCorreosPrueba) ProveerMaterialCorreos(ctx context.Context, m ports.MaterialCorreos) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.materiales = append(p.materiales, m)
	if p.base.denegar {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrCorreosProhibido
	}
	return p.base.ProveerMaterialPreferencias(ctx, ports.MaterialPreferencias{PersonaRef: m.PersonaRef, Accion: m.Accion})
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
	return ports.ReciboCorreos{ReciboRef: "recibo:uno", PersonaRef: r.persona, Accion: m.Accion, CorreoRef: p.CorreoRef, Version: p.VersionEsperada + 1, FechaUTC: time.Now().UTC()}, nil
}

func prepararServicioCorreos(t *testing.T) (ports.OrdenCorreos, *proveedorCorreosPrueba, *registroCorreosPrueba, *desafioCorreosPrueba, *validadorCorreosPrueba, *ServicioCorreos) {
	t.Helper()
	_, actor := ordenPrueba(t, &proveedorPrueba{})
	proveedor := &proveedorCorreosPrueba{}
	orden, err := ports.NuevaOrdenCorreos(actor, proveedor)
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
	proveedor.base.denegar = true
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
