package seguridad

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/usuarios/ports"
)

type fuentePrueba struct {
	claves   ClavesCorreos
	alCargar func()
}

func (f *fuentePrueba) CargarClavesCorreos(context.Context) (ClavesCorreos, error) {
	if f.alCargar != nil {
		f.alCargar()
	}
	return f.claves, nil
}

func material(b byte) [32]byte {
	var m [32]byte
	for i := range m {
		m[i] = b
	}
	return m
}

func clavesPrueba() ClavesCorreos {
	return ClavesCorreos{
		CifradoActivo:   ClaveCorreo{Ref: "clave:cifrado:v1", Material: material(1)},
		Igualdad:        ClaveCorreo{Ref: "clave:igualdad:v1", Material: material(2)},
		SemanticaActiva: ClaveCorreo{Ref: "clave:semantica:v1", Material: material(3)},
		CodigoActivo:    ClaveCorreo{Ref: "clave:codigo:v1", Material: material(4)},
	}
}

const persona = "persona:1234567890123456"
const correo = "correo:1234567890123456"

func preparar(t *testing.T) (*AdaptadorCorreos, *fuentePrueba, *time.Time) {
	t.Helper()
	ahora := time.Date(2026, 9, 29, 11, 0, 0, 123456000, time.UTC)
	f := &fuentePrueba{claves: clavesPrueba()}
	a, err := NuevoAdaptadorCorreos(f, func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	return a, f, &ahora
}

func TestDireccionAEADYHuellaSeparada(t *testing.T) {
	a, f, ahora := preparar(t)
	ctx := context.Background()
	s, err := a.CifrarDireccionCorreo(ctx, persona, correo, 1, []byte("persona@example.org"))
	if err != nil || len(s.Nonce) != 12 || len(s.HuellaIgualdad) != 32 || s.ClaveIgualdadRef != f.claves.Igualdad.Ref || s.ClaveIgualdadRef == s.ClaveRef || strings.Contains(string(s.Cifrado), "persona@example.org") {
		t.Fatalf("cifrado: %v", err)
	}
	s2, err := a.CifrarDireccionCorreo(ctx, persona, "correo:otro123456789012", 1, []byte("PERSONA@example.org"))
	if err != nil || subtle.ConstantTimeCompare(s.HuellaIgualdad, s2.HuellaIgualdad) != 1 || subtle.ConstantTimeCompare(s.Cifrado, s2.Cifrado) == 1 {
		t.Fatal("huella o nonce incorrectos")
	}
	ver := func(p string, sobre ports.SobreDireccionCorreo) error {
		return a.ConDireccionCorreoDescifrada(ctx, p, sobre, func(b []byte) error {
			if string(b) != "persona@example.org" {
				t.Fatal("claro alterado")
			}
			return nil
		})
	}
	if err := ver(persona, s); err != nil {
		t.Fatal(err)
	}
	for _, cambio := range []func(*ports.SobreDireccionCorreo){
		func(x *ports.SobreDireccionCorreo) { x.CorreoRef = s2.CorreoRef },
		func(x *ports.SobreDireccionCorreo) { x.Version = 2 },
		func(x *ports.SobreDireccionCorreo) { x.Cifrado = s2.Cifrado },
	} {
		otro := s
		cambio(&otro)
		if err := ver(persona, otro); err == nil {
			t.Fatal("intercambio aceptado")
		}
	}
	if err := ver("persona:otro123456789012", s); err == nil {
		t.Fatal("persona intercambiada")
	}
	// La generación anterior permite leer sólo durante su retención.
	f.claves.CifradoRetenidas = []ClaveCorreo{{Ref: f.claves.CifradoActivo.Ref, Material: f.claves.CifradoActivo.Material, RetenerHasta: ahora.Add(24 * time.Hour)}}
	f.claves.CifradoActivo = ClaveCorreo{Ref: "clave:cifrado:v2", Material: material(5)}
	if err := ver(persona, s); err != nil {
		t.Fatal("rotacion:", err)
	}
	// La cripto informa la generación; SQL deniega altas bajo otra generación
	// hasta que se reindexe el conjunto de la persona.
	f.claves.Igualdad = ClaveCorreo{Ref: "clave:igualdad:v2", Material: material(6)}
	rotado, err := a.CifrarDireccionCorreo(ctx, persona, correo, 1, []byte("persona@example.org"))
	if err != nil || rotado.ClaveIgualdadRef != f.claves.Igualdad.Ref || rotado.ClaveIgualdadRef == s.ClaveIgualdadRef || subtle.ConstantTimeCompare(rotado.HuellaIgualdad, s.HuellaIgualdad) == 1 || rotado.ClaveRef != f.claves.CifradoActivo.Ref {
		t.Fatal("rotacion de igualdad sin referencia o huella nueva")
	}
}

func TestHuellaSemanticaReplayConRotacion(t *testing.T) {
	a, f, _ := preparar(t)
	preimagen, _ := json.Marshal(preimagenSemantica{Esquema: "usuarios.correos.peticion.v2", Persona: persona, Accion: ports.AccionAnadirCorreo, Direccion: "persona@example.org"})
	h1, err := a.SellarHuellaCorreo(context.Background(), preimagen)
	if err != nil || !h1.Validar() || strings.Contains(h1.Activa.Valor, "persona@example.org") {
		t.Fatal("huella invalida")
	}
	f.claves.SemanticaRetenidas = []ClaveCorreo{f.claves.SemanticaActiva}
	f.claves.SemanticaActiva = ClaveCorreo{Ref: "clave:semantica:v2", Material: material(6)}
	h2, err := a.SellarHuellaCorreo(context.Background(), preimagen)
	if err != nil || !h2.Validar() || len(h2.Retenidas) != 1 || h2.Retenidas[0] != h1.Activa {
		t.Fatal("replay tras rotacion")
	}
	if h2.Activa.Valor == h1.Activa.Valor {
		t.Fatal("no roto")
	}
	if _, err := a.SellarHuellaCorreo(context.Background(), []byte("oraculo")); err == nil {
		t.Fatal("preimagen arbitraria")
	}
}

func TestDesafioReinicioVinculosYRevocacion(t *testing.T) {
	a, f, ahora := preparar(t)
	vence := ahora.Add(24 * time.Hour)
	r, err := a.PrepararDesafioCorreo(context.Background(), persona, correo, vence)
	if err != nil || len(r.Desafio) != 32 || len(r.HuellaCodigo) != 32 || len(r.DesafioRef) != 72 {
		t.Fatalf("reserva: %v", err)
	}
	r2, err := a.PrepararDesafioCorreo(context.Background(), persona, correo, vence)
	if err != nil || r.DesafioRef == r2.DesafioRef {
		t.Fatal("desafio repetido")
	}
	d := ports.DespachoVerificacionCorreo{OutboxRef: "outbox:1234567890123456", PersonaRef: persona, CorreoRef: correo, DesafioRef: r.DesafioRef, Desafio: r.Desafio, ClaveRef: r.ClaveRef, VenceUTC: vence}
	// Nuevo adaptador/fuente simula reconstrucción tras reinicio.
	reiniciado, _ := NuevoAdaptadorCorreos(&fuentePrueba{claves: f.claves}, func() time.Time { return *ahora })
	codigo, err := reiniciado.DerivarCodigoCorreo(context.Background(), d)
	if err != nil || len(codigo) != 26 {
		t.Fatalf("derivacion: %v", err)
	}
	m := ports.MetadatosDesafioCorreo{PersonaRef: persona, CorreoRef: correo, DesafioRef: r.DesafioRef, HuellaCodigo: r.HuellaCodigo, ClaveRef: r.ClaveRef, VenceUTC: vence}
	if ok, err := reiniciado.ComprobarCodigoCorreo(context.Background(), m, codigo); err != nil || !ok {
		t.Fatalf("codigo valido: %v", err)
	}
	if ok, err := reiniciado.ComprobarCodigoCorreo(context.Background(), m, strings.Repeat("A", 26)); err != nil || ok {
		t.Fatal("codigo distinto aceptado")
	}
	for _, cambio := range []func(*ports.MetadatosDesafioCorreo){
		func(x *ports.MetadatosDesafioCorreo) { x.PersonaRef = "persona:otro123456789012" },
		func(x *ports.MetadatosDesafioCorreo) { x.CorreoRef = "correo:otro123456789012" },
		func(x *ports.MetadatosDesafioCorreo) { x.DesafioRef = r2.DesafioRef },
		func(x *ports.MetadatosDesafioCorreo) { x.VenceUTC = vence.Add(time.Microsecond) },
	} {
		otro := m
		cambio(&otro)
		if ok, _ := reiniciado.ComprobarCodigoCorreo(context.Background(), otro, codigo); ok {
			t.Fatal("sustitucion aceptada")
		}
	}
	f.claves.CodigoRetenidas = []ClaveCorreo{{Ref: f.claves.CodigoActivo.Ref, Material: f.claves.CodigoActivo.Material, RetenerHasta: vence}}
	f.claves.CodigoActivo = ClaveCorreo{Ref: "clave:codigo:v2", Material: material(7)}
	if ok, err := a.ComprobarCodigoCorreo(context.Background(), m, codigo); err != nil || !ok {
		t.Fatal("rotacion invalida pendiente")
	}
	f.claves.CodigoRetenidas[0].Revocada = true
	if ok, err := a.ComprobarCodigoCorreo(context.Background(), m, codigo); err == nil || ok {
		t.Fatal("revocacion ignorada")
	}
	if _, err := a.DerivarCodigoCorreo(context.Background(), d); err == nil {
		t.Fatal("despacho revocado")
	}
}

func TestErroresNoFiltranSecretosYExpiracion(t *testing.T) {
	a, f, ahora := preparar(t)
	vence := ahora.Add(time.Hour)
	r, _ := a.PrepararDesafioCorreo(context.Background(), persona, correo, vence)
	d := ports.DespachoVerificacionCorreo{OutboxRef: "outbox:1234567890123456", PersonaRef: persona, CorreoRef: correo, DesafioRef: r.DesafioRef, Desafio: r.Desafio, ClaveRef: r.ClaveRef, VenceUTC: vence}
	codigo, _ := a.DerivarCodigoCorreo(context.Background(), d)
	*ahora = vence
	_, err := a.DerivarCodigoCorreo(context.Background(), d)
	if err == nil || strings.Contains(err.Error(), codigo) || strings.Contains(err.Error(), "persona@example.org") || strings.Contains(err.Error(), r.ClaveRef) {
		t.Fatal("error con secreto o expiracion admitida")
	}
	f.claves.CodigoActivo.Revocada = true
	if _, err := a.PrepararDesafioCorreo(context.Background(), persona, correo, vence.Add(time.Hour)); err == nil {
		t.Fatal("clave revocada")
	}
}

func TestCaducidadDuranteCargaClaves(t *testing.T) {
	ctx := context.Background()
	for _, operacion := range []string{"preparar", "comprobar", "derivar"} {
		t.Run(operacion, func(t *testing.T) {
			a, fuente, ahora := preparar(t)
			vence := ahora.Add(time.Hour)
			var reserva ports.ReservaDesafio
			var codigo string
			var meta ports.MetadatosDesafioCorreo
			var despacho ports.DespachoVerificacionCorreo
			if operacion != "preparar" {
				var err error
				reserva, err = a.PrepararDesafioCorreo(ctx, persona, correo, vence)
				if err != nil {
					t.Fatal(err)
				}
				despacho = ports.DespachoVerificacionCorreo{OutboxRef: "outbox:1234567890123456", PersonaRef: persona, CorreoRef: correo, DesafioRef: reserva.DesafioRef, Desafio: reserva.Desafio, ClaveRef: reserva.ClaveRef, VenceUTC: vence}
				codigo, err = a.DerivarCodigoCorreo(ctx, despacho)
				if err != nil {
					t.Fatal(err)
				}
				meta = ports.MetadatosDesafioCorreo{PersonaRef: persona, CorreoRef: correo, DesafioRef: reserva.DesafioRef, HuellaCodigo: reserva.HuellaCodigo, ClaveRef: reserva.ClaveRef, VenceUTC: vence}
			}
			fuente.alCargar = func() { *ahora = vence }
			switch operacion {
			case "preparar":
				if r, err := a.PrepararDesafioCorreo(ctx, persona, correo, vence); err == nil || r.DesafioRef != "" {
					t.Fatal("reserva caducada")
				}
			case "comprobar":
				if ok, err := a.ComprobarCodigoCorreo(ctx, meta, codigo); err == nil || ok {
					t.Fatal("codigo caducado aceptado")
				}
			case "derivar":
				if obtenido, err := a.DerivarCodigoCorreo(ctx, despacho); err == nil || obtenido != "" {
					t.Fatal("codigo caducado derivado")
				}
			}
		})
	}
}
