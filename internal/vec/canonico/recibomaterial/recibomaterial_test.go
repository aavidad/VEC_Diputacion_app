package recibomaterial

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	almacencanonico "vec-diputacion-granada/internal/vec/canonico/almacen"
)

func perfilPrueba() Perfil {
	return Perfil{
		Esquema: EsquemaPerfil, VersionEsquema: EsquemaVersion,
		Referencia: "perfil:capacidades:material:v2:001", Version: 3,
		ConectorLogicoID: "conector:almacen:corporativo", EscrituraEnFlujo: true,
		ReferenciasOpacas: true, IntegridadSHA256: true, Versionado: true,
		Retencion: true, BloqueoLegal: true, CifradoEnTransito: true,
		CifradoEnReposo: true, CifradoPorObjeto: true, PreservaObjetoOriginal: true,
		TamanoMaximoObjeto: 32 << 20,
	}
}

func instantaneaPrueba() Instantanea {
	return Instantanea{
		Esquema: EsquemaInstantanea, VersionEsquema: EsquemaVersion,
		ConectorLogicoID: "conector:almacen:corporativo",
		ObjetoRef:        "objeto:material:v2:001", ObjetoVersion: "version:material:v2:007",
		Zona: almacencanonico.ZonaAdmitida, MIME: "application/pdf", Tamano: 128,
		HuellaContenido: [sha256.Size]byte{1}, EvidenciaCreacionRef: "evidencia:creacion:material:v2:001",
		AlmacenadoEn:         time.Date(2026, 7, 15, 12, 34, 56, 789000000, time.UTC),
		TieneRetencion:       true,
		RetenidoHasta:        time.Date(2027, 7, 15, 12, 34, 56, 789000000, time.UTC),
		EstadoInmovilizacion: EstadoInmovilizado, EstadoObjeto: EstadoObjetoActivo,
	}
}

func reciboPrueba() Recibo {
	return Recibo{
		Esquema: EsquemaRecibo, VersionEsquema: EsquemaVersion,
		ReferenciaDurableOriginal: "recibo:material:durable:001",
		PerfilReferencia:          "perfil:capacidades:material:v2:001", PerfilVersion: 3,
		HuellaPerfil: [sha256.Size]byte{2},
		Hechos: HechosContexto{
			ModuloID: "bolsa", AccionNegocio: "bolsa.decision.custodiar", AccionTecnica: AccionEscribir,
			RecursoRef: "recurso:almacen:001", OperacionRef: "operacion:almacen:001",
			CargaRef: "carga:documental:001", EfectoRef: "efecto:almacen:001",
			Clasificacion: "datos_personales_alta",
		},
		HuellaPlan: HuellaPlan{
			Referencia: "plan:material:v2:001", Version: 5,
			Suma: [sha256.Size]byte{3}, HuellaVinculo: [sha256.Size]byte{4},
		},
		Instantanea: instantaneaPrueba(),
	}
}

func TestPrimitivasRechazanUbicacionesPIIYHuellasNoCanonicas(t *testing.T) {
	for _, valor := range []string{
		"bucket:documentos", "https://almacen.local", "objeto/privado", "12345678Z", "X1234567L",
	} {
		if AliasLogicoValido(valor, 512) {
			t.Fatalf("alias prohibido aceptado: %q", valor)
		}
	}
	if !AliasLogicoValido("objeto:material:v2:001", 512) {
		t.Fatal("alias logico rechazado")
	}
	for _, valor := range []string{strings.Repeat("0", 64), strings.Repeat("A", 64), "abc"} {
		if _, err := DecodificarSHA256(valor); err == nil {
			t.Fatalf("huella prohibida aceptada: %q", valor)
		}
	}
	if _, err := DecodificarSHA256(strings.Repeat("a", 64)); err != nil {
		t.Fatalf("huella valida: %v", err)
	}
}

func TestCanonicoPerfilMantieneVectorPublicado(t *testing.T) {
	huella, err := HuellaPerfil(perfilPrueba())
	if err != nil {
		t.Fatal(err)
	}
	const esperada = "07c9049cbfedd70e4b7b918efb7fe462eef89ecf443549c3bef5ff060fb53fbe"
	if hex.EncodeToString(huella[:]) != esperada {
		t.Fatalf("vector de perfil alterado: %x", huella)
	}
	alterado := perfilPrueba()
	alterado.CifradoEnReposo = false
	if _, err := CanonicoPerfil(alterado); err == nil {
		t.Fatal("perfil sin cifrado aceptado")
	}
}

func TestCanonicosInstantaneaYReciboSonDeterministasYDefensivos(t *testing.T) {
	instantanea := instantaneaPrueba()
	primero, err := CanonicoInstantanea(instantanea)
	if err != nil {
		t.Fatal(err)
	}
	segundo, _ := CanonicoInstantanea(instantanea)
	primero[0] ^= 1
	if string(primero) == string(segundo) {
		t.Fatal("la salida comparte memoria")
	}

	recibo := reciboPrueba()
	canonico, err := CanonicoRecibo(recibo)
	if err != nil {
		t.Fatal(err)
	}
	huella := sha256.Sum256(canonico)
	const esperada = "c853c0d046e7d0166f7e5d1c1e5ea19a221dfad485aa141cb39f7a8307e380a3"
	if hex.EncodeToString(huella[:]) != esperada {
		t.Fatalf("vector de recibo alterado: %x", huella)
	}
	recibo.ReferenciaDurableOriginal = ""
	if _, err := CanonicoRecibo(recibo); err == nil {
		t.Fatal("recibo sin referencia aceptado")
	}
	if _, err := CanonicoIdentidadDurable(recibo); err != nil {
		t.Fatalf("identidad durable: %v", err)
	}
}

func TestAtestacionLigaDominioMensajeYClave(t *testing.T) {
	mensaje := []byte("mensaje material")
	huella := sha256.Sum256(mensaje)
	codigo := make([]byte, sha256.Size)
	if !SolicitudAtestacionValida(DominioRecibo, mensaje, huella) {
		t.Fatal("solicitud valida rechazada")
	}
	if !AtestacionValida(DominioRecibo, mensaje, huella, AlgoritmoHMACSHA256,
		"clave:atestacion:material:v2", 7, DominioRecibo, huella, codigo) {
		t.Fatal("atestacion valida rechazada")
	}
	codigo = codigo[:sha256.Size-1]
	if AtestacionValida(DominioRecibo, mensaje, huella, AlgoritmoHMACSHA256,
		"clave:atestacion:material:v2", 7, DominioRecibo, huella, codigo) {
		t.Fatal("codigo truncado aceptado")
	}
}
