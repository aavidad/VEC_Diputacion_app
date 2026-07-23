package seguridad

import (
	"context"
	"crypto/ed25519"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestAutenticadorFuentesAnalisisConservaEvidenciaPublicaHistorica(
	t *testing.T,
) {
	inicio := time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)
	claveRaiz := claveDeterministaAutenticadorPrueba("raiz-rotacion")
	claveK1 := claveDeterministaAutenticadorPrueba("verificador-k1")
	claveK2 := claveDeterministaAutenticadorPrueba("verificador-k2")
	raiz := ports.RaizConfianzaAutoridadFuenteAnalisis{
		ClaveID:             "raiz_rotacion_fuentes_012345",
		ClavePublicaEd25519: claveRaiz.Public().(ed25519.PublicKey),
		Estado:              ports.RaizAutoridadActiva,
		ValidaDesde:         inicio.AddDate(-1, 0, 0),
		ValidaHasta:         inicio.AddDate(1, 0, 0),
		UltimaEmisionPermitida: inicio.Add(
			24 * time.Hour,
		),
	}
	nuevaConfianza := func(
		revocaciones []ports.RevocacionAutoridadFuenteAnalisis,
	) ports.ConfianzaAutoridadesFuenteAnalisis {
		confianza, err := ports.NuevaConfianzaAutoridadesFuenteAnalisis(
			"organizacion_diputacion_granada",
			"audiencia_fuentes_internas_012345",
			[]ports.RaizConfianzaAutoridadFuenteAnalisis{raiz},
			revocaciones,
		)
		if err != nil {
			t.Fatal(err)
		}
		return confianza
	}
	presentador := func(
		clave ed25519.PrivateKey,
		serie uint64,
		emitidaEn time.Time,
		validaHasta time.Time,
	) presentadorFuenteAnalisisPrueba {
		return presentadorFuenteAnalisisPrueba{
			claveRaiz:   claveRaiz,
			clavePrueba: clave,
			datos: ports.DatosCredencialAutoridadFuenteAnalisis{
				RaizClaveID:        raiz.ClaveID,
				AutoridadRef:       "verificador_cobertura_tcb_012345",
				BackendRef:         "backend_verificador_cobertura_01",
				OrganizacionRef:    "organizacion_diputacion_granada",
				Audiencia:          "audiencia_fuentes_internas_012345",
				Rol:                ports.RolVerificadorCobertura,
				Serie:              serie,
				Generacion:         uint32(serie),
				ClavePruebaEd25519: clave.Public().(ed25519.PublicKey),
				EmitidaEn:          emitidaEn,
				ValidaHasta:        validaHasta,
			},
		}
	}
	nuevaEvidencia := func(
		presentador presentadorFuenteAnalisisPrueba,
		comprobadaEn time.Time,
	) ports.EvidenciaPublicaAutoridadFuenteAnalisis {
		desafio, err := ports.NuevoDesafioAutoridadFuenteAnalisis(
			[]byte("peticion-canonica-rotacion"),
			"organizacion_diputacion_granada",
			"audiencia_fuentes_internas_012345",
			ports.RolVerificadorCobertura,
		)
		if err != nil {
			t.Fatal(err)
		}
		presentacion, err := presentador.PresentarAutoridadFuenteAnalisis(
			context.Background(),
			desafio,
		)
		if err != nil {
			t.Fatal(err)
		}
		evidencia, err :=
			ports.NuevaEvidenciaPublicaAutoridadFuenteAnalisis(
				desafio,
				presentacion,
				ports.RolVerificadorCobertura,
				comprobadaEn,
			)
		if err != nil {
			t.Fatal(err)
		}
		return evidencia
	}
	evidenciaK1 := nuevaEvidencia(
		presentador(claveK1, 1, inicio.Add(-time.Hour), inicio.Add(6*time.Second)),
		inicio,
	)
	evidenciaK2 := nuevaEvidencia(
		presentador(
			claveK2,
			2,
			inicio.Add(6*time.Second),
			inicio.Add(20*time.Second),
		),
		inicio.Add(10*time.Second),
	)
	autenticador, err := NuevoAutenticadorFuentesAnalisisConConfianza(
		nuevaConfianza(nil),
	)
	if err != nil {
		t.Fatal(err)
	}
	identidadK1, err := autenticador.
		VerificarEvidenciaPublicaAutoridadFuenteAnalisis(evidenciaK1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := autenticador.
		VerificarEvidenciaPublicaAutoridadFuenteAnalisis(
			evidenciaK2,
		); err != nil {
		t.Fatal(err)
	}
	revalidadaK1, err := autenticador.
		VerificarEvidenciaPublicaAutoridadFuenteAnalisis(evidenciaK1)
	if err != nil ||
		!ports.IdentidadesAutoridadFuenteAnalisisIguales(
			identidadK1,
			revalidadaK1,
		) {
		t.Fatalf("la rotación K2 invalidó la evidencia K1: %v", err)
	}

	revocadaDuranteVentana, err :=
		NuevoAutenticadorFuentesAnalisisConConfianza(
			nuevaConfianza([]ports.RevocacionAutoridadFuenteAnalisis{{
				AutoridadRef: "verificador_cobertura_tcb_012345",
				Serie:        1,
				RevocadaEn:   inicio.Add(4 * time.Second),
			}}),
		)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := revocadaDuranteVentana.
		VerificarEvidenciaPublicaAutoridadFuenteAnalisis(
			evidenciaK1,
		); !errors.Is(err, ports.ErrResultadoFuenteAnalisisNoConfiable) {
		t.Fatalf("revocación durante vigencia no aplicada: %v", err)
	}

	evidenciaEnFinExclusivo := nuevaEvidencia(
		presentador(claveK1, 1, inicio.Add(-time.Hour), inicio.Add(6*time.Second)),
		inicio.Add(6*time.Second),
	)
	if _, err := autenticador.
		VerificarEvidenciaPublicaAutoridadFuenteAnalisis(
			evidenciaEnFinExclusivo,
		); !errors.Is(err, ports.ErrResultadoFuenteAnalisisNoConfiable) {
		t.Fatalf("credencial en fin exclusivo aceptada: %v", err)
	}
}
