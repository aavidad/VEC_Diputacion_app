package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestVerificacionNoMarcaFirmaSinVinculoYRevocacionVigente(t *testing.T) {
	original := []byte("original sintetico")
	firmado := []byte("firmado sintetico")
	base := sha256.Sum256(original)
	firma := sha256.Sum256(firmado)
	s := SolicitudVerificacionFirma{DocumentoID: "ref:" + strings.Repeat("1", 64), Version: 1,
		HuellaOriginalSHA256: hex.EncodeToString(base[:]), ContenidoOriginal: original, ContenidoFirmado: firmado}
	r := ResultadoVerificacionFirma{Estado: EstadoVerificacionValida, VinculoOriginal: true,
		HuellaOriginalSHA256: s.HuellaOriginalSHA256, HuellaFirmadoSHA256: hex.EncodeToString(firma[:]),
		FirmanteRef: "ref:" + strings.Repeat("2", 64), CertificadoHuellaSHA256: strings.Repeat("3", 64),
		SelloTiempoEstado: "valido", RevocacionEstado: "vigente"}
	if err := r.ValidarContra(s); err != nil {
		t.Fatalf("validacion positiva: %v", err)
	}
	r.VinculoOriginal = false
	if r.ValidarContra(s) == nil {
		t.Fatal("acepto firma sin vinculo original")
	}
	r.VinculoOriginal = true
	r.RevocacionEstado = "indeterminada"
	if r.ValidarContra(s) == nil {
		t.Fatal("acepto revocacion indeterminada")
	}
	r.RevocacionEstado = "vigente"
	r.Estado = EstadoVerificacionIndeterminada
	if r.ValidarContra(s) == nil {
		t.Fatal("acepto estado indeterminado")
	}
	proveedor := SinProveedorFirma{}
	estado, err := proveedor.EstadoDeclarado(context.Background(), s.DocumentoID, 1)
	if err != nil || estado != "pendiente_proveedor" {
		t.Fatalf("sin proveedor: %q %v", estado, err)
	}
}
