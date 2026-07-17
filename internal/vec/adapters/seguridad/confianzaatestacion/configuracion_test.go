package confianzaatestacion

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestConfiguracionConfianzaAtestacionV2ClonaYEsIndependienteDelOrden(t *testing.T) {
	publicadaEn := instanteConfianzaAtestacionV2Prueba()
	publicaPrimera := clavePublicaConfianzaAtestacionV2Prueba(1)
	primera := nuevaRaizConfianzaAtestacionV2Prueba(
		t,
		"clave:atestacion:v2:primera",
		publicaPrimera,
		"vec-diputacion/pruebas/vec/autorizacion-v2",
		EstadoClaveAtestacionAutorizacionV2Activa,
		publicadaEn.Add(-time.Hour),
		publicadaEn.Add(time.Hour),
		time.Time{},
	)
	segunda := nuevaRaizConfianzaAtestacionV2Prueba(
		t,
		"clave:atestacion:v2:segunda",
		clavePublicaConfianzaAtestacionV2Prueba(2),
		"vec-diputacion/pruebas/vec/autorizacion-v2",
		EstadoClaveAtestacionAutorizacionV2Activa,
		publicadaEn.Add(-time.Hour),
		publicadaEn.Add(time.Hour),
		time.Time{},
	)
	configuracion, err := NuevaConfiguracionConfianzaAtestacionAutorizacionV2(
		"confianza:atestacion:v2:revision:1",
		publicadaEn,
		publicadaEn.Add(time.Hour),
		primera,
		segunda,
	)
	if err != nil {
		t.Fatal(err)
	}
	invertida, err := NuevaConfiguracionConfianzaAtestacionAutorizacionV2(
		"confianza:atestacion:v2:revision:1",
		publicadaEn,
		publicadaEn.Add(time.Hour),
		segunda,
		primera,
	)
	if err != nil || configuracion.huellaSHA256 != invertida.huellaSHA256 {
		t.Fatalf("la huella depende del orden: %s / %s / %v", configuracion.huellaSHA256, invertida.huellaSHA256, err)
	}
	if err := configuracion.ValidarHuellaSHA256Esperada(configuracion.huellaSHA256); err != nil {
		t.Fatalf("la huella exacta fue rechazada: %v", err)
	}
	huellaDistinta := strings.Repeat("0", 64)
	if huellaDistinta == configuracion.huellaSHA256 {
		huellaDistinta = strings.Repeat("1", 64)
	}
	if err := configuracion.ValidarHuellaSHA256Esperada(huellaDistinta); !errors.Is(err, ErrConfiguracionConfianzaAtestacionV2Invalida) {
		t.Fatalf("una huella distinta fue aceptada: %v", err)
	}

	publicaPrimera[0] ^= 0xff
	primera.clavePublica[0] ^= 0xff
	if configuracion.validar() != nil {
		t.Fatal("una mutacion del origen altero la configuracion")
	}
	configuracion.raices[0].clavePublica[0] ^= 0xff
	if configuracion.validar() == nil {
		t.Fatal("una mutacion interna no fue detectada")
	}
}

func TestConfiguracionConfianzaAtestacionV2RechazaDuplicadosYLimites(t *testing.T) {
	ahora := instanteConfianzaAtestacionV2Prueba()
	publica := clavePublicaConfianzaAtestacionV2Prueba(3)
	primera := nuevaRaizConfianzaAtestacionV2Prueba(
		t, "clave:atestacion:v2:duplicada", publica,
		"vec-diputacion/pruebas/vec/autorizacion-v2",
		EstadoClaveAtestacionAutorizacionV2Activa,
		ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{},
	)
	mismoID := nuevaRaizConfianzaAtestacionV2Prueba(
		t, primera.claveID, clavePublicaConfianzaAtestacionV2Prueba(4),
		primera.audienciaDespliegue, EstadoClaveAtestacionAutorizacionV2Activa,
		ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{},
	)
	mismaClave := nuevaRaizConfianzaAtestacionV2Prueba(
		t, "clave:atestacion:v2:otro-id", publica,
		primera.audienciaDespliegue, EstadoClaveAtestacionAutorizacionV2Activa,
		ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{},
	)

	casos := []struct {
		nombre    string
		revision  string
		publicada time.Time
		expira    time.Time
		raices    []RaizPublicaAtestacionAutorizacionV2
	}{
		{"sin_raices", "confianza:v2:1", ahora, ahora.Add(time.Hour), nil},
		{"revision_vacia", "", ahora, ahora.Add(time.Hour), []RaizPublicaAtestacionAutorizacionV2{primera}},
		{"revision_con_comodin", "confianza:*", ahora, ahora.Add(time.Hour), []RaizPublicaAtestacionAutorizacionV2{primera}},
		{"expiracion_igual", "confianza:v2:1", ahora, ahora, []RaizPublicaAtestacionAutorizacionV2{primera}},
		{"expiracion_excesiva", "confianza:v2:1", ahora, ahora.Add(24*time.Hour + time.Microsecond), []RaizPublicaAtestacionAutorizacionV2{primera}},
		{"clave_id_duplicada", "confianza:v2:1", ahora, ahora.Add(time.Hour), []RaizPublicaAtestacionAutorizacionV2{primera, mismoID}},
		{"material_duplicado", "confianza:v2:1", ahora, ahora.Add(time.Hour), []RaizPublicaAtestacionAutorizacionV2{primera, mismaClave}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := NuevaConfiguracionConfianzaAtestacionAutorizacionV2(
				caso.revision,
				caso.publicada,
				caso.expira,
				caso.raices...,
			); !errors.Is(err, ErrConfiguracionConfianzaAtestacionV2Invalida) {
				t.Fatalf("configuracion invalida aceptada: %v", err)
			}
		})
	}

	raices := make([]RaizPublicaAtestacionAutorizacionV2, 0, maximoRaicesConfianzaAtestacionV2+1)
	for indice := 0; indice <= maximoRaicesConfianzaAtestacionV2; indice++ {
		raices = append(raices, nuevaRaizConfianzaAtestacionV2Prueba(
			t,
			fmt.Sprintf("clave:atestacion:v2:%03d", indice),
			clavePublicaConfianzaAtestacionV2Prueba(byte(indice+10)),
			"vec-diputacion/pruebas/vec/autorizacion-v2",
			EstadoClaveAtestacionAutorizacionV2Activa,
			ahora.Add(-time.Hour),
			ahora.Add(time.Hour),
			time.Time{},
		))
	}
	if _, err := NuevaConfiguracionConfianzaAtestacionAutorizacionV2(
		"confianza:v2:demasiadas",
		ahora,
		ahora.Add(time.Hour),
		raices...,
	); !errors.Is(err, ErrConfiguracionConfianzaAtestacionV2Invalida) {
		t.Fatalf("demasiadas raices aceptadas: %v", err)
	}
}

func TestRaizConfianzaAtestacionV2ExigePerfilYRevocacionCoherentes(t *testing.T) {
	ahora := instanteConfianzaAtestacionV2Prueba()
	publica := clavePublicaConfianzaAtestacionV2Prueba(5)
	casos := []struct {
		nombre    string
		claveID   string
		publica   ed25519.PublicKey
		audiencia string
		estado    EstadoClaveAtestacionAutorizacionV2
		desde     time.Time
		hasta     time.Time
		revocada  time.Time
	}{
		{"clave_id_vacio", "", publica, "vec-diputacion/pruebas/vec/autorizacion-v2", EstadoClaveAtestacionAutorizacionV2Activa, ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{}},
		{"clave_corta", "clave:v2:1", publica[:8], "vec-diputacion/pruebas/vec/autorizacion-v2", EstadoClaveAtestacionAutorizacionV2Activa, ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{}},
		{"clave_nula", "clave:v2:1", make(ed25519.PublicKey, ed25519.PublicKeySize), "vec-diputacion/pruebas/vec/autorizacion-v2", EstadoClaveAtestacionAutorizacionV2Activa, ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{}},
		{"audiencia_otro_sistema", "clave:v2:1", publica, "otro/pruebas/vec/autorizacion-v2", EstadoClaveAtestacionAutorizacionV2Activa, ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{}},
		{"audiencia_incompleta", "clave:v2:1", publica, "vec-diputacion/pruebas/vec", EstadoClaveAtestacionAutorizacionV2Activa, ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{}},
		{"estado_desconocido", "clave:v2:1", publica, "vec-diputacion/pruebas/vec/autorizacion-v2", "desconocida", ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{}},
		{"ventana_invertida", "clave:v2:1", publica, "vec-diputacion/pruebas/vec/autorizacion-v2", EstadoClaveAtestacionAutorizacionV2Activa, ahora, ahora.Add(-time.Hour), time.Time{}},
		{"activa_revocada", "clave:v2:1", publica, "vec-diputacion/pruebas/vec/autorizacion-v2", EstadoClaveAtestacionAutorizacionV2Activa, ahora.Add(-time.Hour), ahora.Add(time.Hour), ahora},
		{"revocada_sin_fecha", "clave:v2:1", publica, "vec-diputacion/pruebas/vec/autorizacion-v2", EstadoClaveAtestacionAutorizacionV2Revocada, ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{}},
		{"revocada_antes_de_vigencia", "clave:v2:1", publica, "vec-diputacion/pruebas/vec/autorizacion-v2", EstadoClaveAtestacionAutorizacionV2Revocada, ahora.Add(-time.Hour), ahora.Add(time.Hour), ahora.Add(-2 * time.Hour)},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := NuevaRaizPublicaAtestacionAutorizacionV2EdDSA(
				caso.claveID,
				caso.publica,
				caso.audiencia,
				caso.estado,
				caso.desde,
				caso.hasta,
				caso.revocada,
			); !errors.Is(err, ErrConfiguracionConfianzaAtestacionV2Invalida) {
				t.Fatalf("raiz invalida aceptada: %v", err)
			}
		})
	}

	revocada := nuevaRaizConfianzaAtestacionV2Prueba(
		t, "clave:v2:revocada", publica,
		"vec-diputacion/pruebas/vec/autorizacion-v2",
		EstadoClaveAtestacionAutorizacionV2Revocada,
		ahora.Add(-time.Hour), ahora.Add(time.Hour), ahora,
	)
	if _, err := NuevaConfiguracionConfianzaAtestacionAutorizacionV2(
		"confianza:v2:revocacion-futura",
		ahora.Add(-time.Microsecond),
		ahora.Add(time.Hour),
		revocada,
	); !errors.Is(err, ErrConfiguracionConfianzaAtestacionV2Invalida) {
		t.Fatalf("revocacion posterior a publicacion aceptada: %v", err)
	}
}

func TestConfiguracionConfianzaAtestacionV2RedactaYBloqueaCodecs(t *testing.T) {
	ahora := instanteConfianzaAtestacionV2Prueba()
	raiz := nuevaRaizConfianzaAtestacionV2Prueba(
		t, "clave:secreta:no-registrar", clavePublicaConfianzaAtestacionV2Prueba(6),
		"vec-diputacion/pruebas/vec/autorizacion-v2",
		EstadoClaveAtestacionAutorizacionV2Activa,
		ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{},
	)
	configuracion, err := NuevaConfiguracionConfianzaAtestacionAutorizacionV2(
		"revision:secreta:no-registrar", ahora, ahora.Add(time.Hour), raiz,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, valor := range []any{raiz, configuracion} {
		if _, err := json.Marshal(valor); !errors.Is(err, ErrSerializacionConfianzaAtestacionV2Prohibida) {
			t.Fatalf("JSON no bloqueado para %T: %v", valor, err)
		}
		if _, err := xml.Marshal(valor); !errors.Is(err, ErrSerializacionConfianzaAtestacionV2Prohibida) {
			t.Fatalf("XML no bloqueado para %T: %v", valor, err)
		}
		texto := fmt.Sprintf("%v|%+v|%#v", valor, valor, valor)
		if strings.Contains(texto, "secreta") || !strings.Contains(texto, "REDACTADA") {
			t.Fatalf("formato no redactado para %T: %s", valor, texto)
		}
		registro := slog.AnyValue(valor).Resolve().String()
		if strings.Contains(registro, "secreta") || !strings.Contains(registro, "REDACTADA") {
			t.Fatalf("slog no redactado para %T: %s", valor, registro)
		}
	}
}

func nuevaRaizConfianzaAtestacionV2Prueba(
	t *testing.T,
	claveID string,
	publica ed25519.PublicKey,
	audiencia string,
	estado EstadoClaveAtestacionAutorizacionV2,
	desde time.Time,
	hasta time.Time,
	revocada time.Time,
) RaizPublicaAtestacionAutorizacionV2 {
	t.Helper()
	raiz, err := NuevaRaizPublicaAtestacionAutorizacionV2EdDSA(
		claveID, publica, audiencia, estado, desde, hasta, revocada,
	)
	if err != nil {
		t.Fatalf("crear raiz de prueba: %v", err)
	}
	return raiz
}

func clavePublicaConfianzaAtestacionV2Prueba(semilla byte) ed25519.PublicKey {
	material := bytes.Repeat([]byte{semilla}, ed25519.SeedSize)
	privada := ed25519.NewKeyFromSeed(material)
	return append(ed25519.PublicKey(nil), privada.Public().(ed25519.PublicKey)...)
}

func instanteConfianzaAtestacionV2Prueba() time.Time {
	return time.Date(2026, 7, 17, 10, 0, 0, 123_000, time.UTC)
}
