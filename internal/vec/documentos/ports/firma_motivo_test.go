package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func solicitudMotivadaPrueba() SolicitudVerificacionFirma {
	original := []byte("original sintetico")
	huella := sha256.Sum256(original)
	return SolicitudVerificacionFirma{DocumentoID: "ref:" + strings.Repeat("1", 64), Version: 1,
		HuellaOriginalSHA256: hex.EncodeToString(huella[:]), ContenidoOriginal: original,
		ContenidoFirmado: []byte("firmado sintetico")}
}

func TestMotivoCatalogoCerradoYEstadoUnico(t *testing.T) {
	casos := map[MotivoVerificacionFirma]EstadoVerificacionFirma{
		MotivoFirmaVerificada:             EstadoVerificacionValida,
		MotivoIntegridadNoValida:          EstadoVerificacionNoValida,
		MotivoCertificadoNoValido:         EstadoVerificacionNoValida,
		MotivoConfianzaNoValida:           EstadoVerificacionNoValida,
		MotivoVinculoOriginalNoAcreditado: EstadoVerificacionIndeterminada,
		MotivoFirmanteNoIdentificado:      EstadoVerificacionIndeterminada,
		MotivoCertificadoNoAcreditado:     EstadoVerificacionIndeterminada,
		MotivoConfianzaNoAcreditada:       EstadoVerificacionIndeterminada,
		MotivoRevocacionNoAcreditada:      EstadoVerificacionIndeterminada,
		MotivoSelloTiempoNoAcreditado:     EstadoVerificacionIndeterminada,
		MotivoRechazadaPorValidador:       EstadoVerificacionIndeterminada,
		MotivoCredencialRechazada:         EstadoVerificacionIndeterminada,
		MotivoValidadorNoDisponible:       EstadoVerificacionIndeterminada,
		MotivoRespuestaNoInterpretable:    EstadoVerificacionIndeterminada,
	}
	for motivo, estado := range casos {
		if motivo.EstadoAsociado() != estado {
			t.Fatalf("%s: estado %q", motivo, motivo.EstadoAsociado())
		}
	}
	for _, ajeno := range []MotivoVerificacionFirma{"", "valida", "VERIFICADA", "otro"} {
		if ajeno.EstadoAsociado() != "" {
			t.Fatalf("motivo ajeno aceptado: %q", ajeno)
		}
	}
}

func TestVerificacionMotivadaRechazaIncoherencias(t *testing.T) {
	s := solicitudMotivadaPrueba()
	indeterminada := VerificacionFirmaMotivada{
		Resultado: ResultadoVerificacionFirma{Estado: EstadoVerificacionIndeterminada},
		Motivo:    MotivoRevocacionNoAcreditada,
	}
	if err := indeterminada.ValidarContra(s); err != nil {
		t.Fatalf("indeterminada coherente rechazada: %v", err)
	}
	// Un motivo positivo nunca convierte en firma un estado distinto.
	falsa := VerificacionFirmaMotivada{Resultado: indeterminada.Resultado, Motivo: MotivoFirmaVerificada}
	if falsa.ValidarContra(s) == nil {
		t.Fatal("acepto motivo verificado con estado indeterminado")
	}
	// Estado valido sin las comprobaciones estrictas del puerto: incoherente.
	valida := VerificacionFirmaMotivada{
		Resultado: ResultadoVerificacionFirma{Estado: EstadoVerificacionValida},
		Motivo:    MotivoFirmaVerificada,
	}
	if valida.ValidarContra(s) == nil {
		t.Fatal("acepto valida sin vinculo, sello ni revocacion")
	}
	sinMotivo := VerificacionFirmaMotivada{Resultado: indeterminada.Resultado}
	if sinMotivo.ValidarContra(s) == nil {
		t.Fatal("acepto resultado sin motivo")
	}
}
