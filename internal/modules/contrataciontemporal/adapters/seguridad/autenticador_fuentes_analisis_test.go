package seguridad

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type presentadorFuenteAnalisisPrueba struct {
	datos       ports.DatosCredencialAutoridadFuenteAnalisis
	claveRaiz   ed25519.PrivateKey
	clavePrueba ed25519.PrivateKey
}

func (p presentadorFuenteAnalisisPrueba) PresentarAutoridadFuenteAnalisis(
	_ context.Context,
	desafio ports.DesafioAutoridadFuenteAnalisis,
) (ports.PresentacionAutoridadFuenteAnalisis, error) {
	materialCredencial, err :=
		ports.MaterialFirmaCredencialAutoridadFuenteAnalisis(p.datos)
	if err != nil {
		return ports.PresentacionAutoridadFuenteAnalisis{}, err
	}
	credencial, err := ports.NuevaCredencialAutoridadFuenteAnalisis(
		p.datos,
		ed25519.Sign(p.claveRaiz, materialCredencial),
	)
	if err != nil {
		return ports.PresentacionAutoridadFuenteAnalisis{}, err
	}
	materialDesafio, err := desafio.Bytes()
	if err != nil {
		return ports.PresentacionAutoridadFuenteAnalisis{}, err
	}
	return ports.NuevaPresentacionAutoridadFuenteAnalisis(
		credencial,
		ed25519.Sign(p.clavePrueba, materialDesafio),
	)
}

func TestAutenticadorFuentesAnalisisConConfianzaAutentica(t *testing.T) {
	ahora := time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)
	claveRaiz := claveDeterministaAutenticadorPrueba("raiz")
	clavePrueba := claveDeterministaAutenticadorPrueba("prueba")
	confianza, err := ports.NuevaConfianzaAutoridadesFuenteAnalisis(
		"organizacion_diputacion_granada",
		"audiencia_fuentes_internas_012345",
		[]ports.RaizConfianzaAutoridadFuenteAnalisis{{
			ClaveID:             "raiz_institucional_fuentes_012345",
			ClavePublicaEd25519: claveRaiz.Public().(ed25519.PublicKey),
			Estado:              ports.RaizAutoridadActiva,
			ValidaDesde:         ahora.AddDate(-1, 0, 0),
			ValidaHasta:         ahora.AddDate(1, 0, 0),
			UltimaEmisionPermitida: ahora.AddDate(
				0,
				1,
				0,
			),
		}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	autenticador, err := NuevoAutenticadorFuentesAnalisisConConfianza(
		confianza,
	)
	if err != nil {
		t.Fatal(err)
	}
	presentador := presentadorFuenteAnalisisPrueba{
		claveRaiz: claveRaiz, clavePrueba: clavePrueba,
		datos: ports.DatosCredencialAutoridadFuenteAnalisis{
			RaizClaveID:        "raiz_institucional_fuentes_012345",
			AutoridadRef:       "fuente_cobertura_bolsa_012345",
			BackendRef:         "backend_cobertura_bolsa_012345",
			OrganizacionRef:    "organizacion_diputacion_granada",
			Audiencia:          "audiencia_fuentes_internas_012345",
			Rol:                ports.RolFuenteCobertura,
			Serie:              1,
			Generacion:         1,
			ClavePruebaEd25519: clavePrueba.Public().(ed25519.PublicKey),
			EmitidaEn:          ahora.Add(-time.Hour),
			ValidaHasta:        ahora.Add(time.Hour),
		},
	}

	identidad, err := autenticador.AutenticarAutoridadFuenteAnalisis(
		context.Background(),
		presentador,
		[]byte("peticion-canonica-cobertura"),
		ports.RolFuenteCobertura,
		ahora,
	)

	if err != nil {
		t.Fatal(err)
	}
	if identidad.AutoridadRef() != presentador.datos.AutoridadRef ||
		identidad.BackendRef() != presentador.datos.BackendRef ||
		identidad.Rol() != ports.RolFuenteCobertura {
		t.Fatalf("identidad inesperada: %#v", identidad)
	}
}

func TestAutenticadorFuentesAnalisisConConfianzaFallaCerrado(
	t *testing.T,
) {
	if _, err := NuevoAutenticadorFuentesAnalisisConConfianza(
		ports.ConfianzaAutoridadesFuenteAnalisis{},
	); !errors.Is(err, ports.ErrResultadoFuenteAnalisisNoConfiable) {
		t.Fatalf("confianza vacía aceptada: %v", err)
	}
	var autenticador *AutenticadorFuentesAnalisisConConfianza
	if _, err := autenticador.AutenticarAutoridadFuenteAnalisis(
		context.Background(),
		nil,
		[]byte("peticion"),
		ports.RolFuenteCobertura,
		time.Now().UTC(),
	); !errors.Is(err, ports.ErrResultadoFuenteAnalisisNoConfiable) {
		t.Fatalf("adaptador nulo aceptado: %v", err)
	}
}

func claveDeterministaAutenticadorPrueba(
	etiqueta string,
) ed25519.PrivateKey {
	semilla := sha256.Sum256([]byte("VEC-CT-AUTENTICADOR:" + etiqueta))
	return ed25519.NewKeyFromSeed(semilla[:])
}
