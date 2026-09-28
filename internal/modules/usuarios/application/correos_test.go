package application

import (
	"context"
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

type desafioCorreosPrueba struct{ llamados int }

type protectorCorreosPrueba struct{ llamados int }

func (p *protectorCorreosPrueba) CifrarDireccionCorreo(_ context.Context, _, ref string, version uint64, claro []byte) (ports.SobreDireccionCorreo, error) {
	p.llamados++
	return ports.SobreDireccionCorreo{CorreoRef: ref, Version: version, ClaveRef: "clave:correo", Nonce: []byte(strings.Repeat("n", 12)), Cifrado: []byte(strings.Repeat("c", 32)), HuellaIgualdad: []byte(strings.Repeat("h", 32))}, nil
}

func (d *desafioCorreosPrueba) PrepararDesafioCorreo(_ context.Context, _, _ string, vence time.Time) (ports.ReservaDesafio, error) {
	d.llamados++
	return ports.ReservaDesafio{DesafioRef: "desafio:uno", Desafio: []byte(strings.Repeat("d", 16)), HuellaCodigo: []byte(strings.Repeat("h", 32)), ClaveRef: "clave:uno", VenceUTC: vence}, nil
}

type validadorCorreosPrueba struct{ llamados int }

func (v *validadorCorreosPrueba) ComprobarCodigoCorreo(_ context.Context, _ ports.MetadatosDesafioCorreo, codigo string) (bool, error) {
	v.llamados++
	return codigo == "CODIGO-1234567890", nil
}

type registroCorreosPrueba struct {
	persona                                 string
	replay                                  bool
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
	servicio, err := NuevoServicioCorreos(registro, &protectorCorreosPrueba{}, desafio, validador, time.Now)
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
	if registro.peticion.Direccion != "" || registro.sobre.CorreoRef != recibo.CorreoRef || registro.sobre.Version != 1 || registro.material.HuellaPeticion == "" || strings.Contains(registro.material.HuellaPeticion, peticion.Direccion) || proveedor.materiales[0].PersonaRef != registro.persona {
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
	if err != nil || !registro.comprobacion || validador.llamados != 1 || registro.peticion.Codigo != "" || strings.Contains(registro.material.HuellaPeticion, peticion.Codigo) {
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
