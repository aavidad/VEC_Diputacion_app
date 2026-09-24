package internactproveedores

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	gocose "github.com/veraison/go-cose"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type firmanteV3 struct {
	claveID, audiencia string
	privada            ed25519.PrivateKey
	reloj              interface{ Ahora() time.Time }
}

func (f *firmanteV3) FirmarAtestacionAutorizacionV3(ctx context.Context, solicitud vecports.SolicitudFirmaAtestacionAutorizacionV3) (vecports.ResultadoFirmaAtestacionAutorizacionV3, error) {
	vacio := vecports.ResultadoFirmaAtestacionAutorizacionV3{}
	if f == nil || ctx == nil || ctx.Err() != nil || len(f.privada) != ed25519.PrivateKeySize || f.reloj == nil {
		return vacio, vecports.ErrFirmaAtestacionNoDisponible
	}
	cabecera, err := solicitud.Cabecera()
	if err != nil || cabecera.ClaveID != f.claveID || cabecera.Audiencia != f.audiencia {
		return vacio, vecports.ErrFirmaAtestacionNoDisponible
	}
	mensaje, err := solicitud.Mensaje()
	if err != nil {
		return vacio, vecports.ErrFirmaAtestacionNoDisponible
	}
	defer clear(mensaje)
	aad, err := confianza.AADExternoAtestacionAutorizacionV3(cabecera.Audiencia)
	if err != nil {
		return vacio, vecports.ErrFirmaAtestacionNoDisponible
	}
	sobre := gocose.NewSign1Message()
	sobre.Headers.Protected.SetAlgorithm(gocose.AlgorithmEdDSA)
	sobre.Headers.Protected[gocose.HeaderLabelKeyID] = []byte(f.claveID)
	sobre.Payload = append([]byte(nil), mensaje...)
	firmante, err := gocose.NewSigner(gocose.AlgorithmEdDSA, f.privada)
	if err != nil || sobre.Sign(rand.Reader, aad, firmante) != nil {
		return vacio, vecports.ErrFirmaAtestacionNoDisponible
	}
	sobre.Payload = nil
	sobre.Headers.RawProtected = nil
	sobre.Headers.RawUnprotected = nil
	firma, err := sobre.MarshalCBOR()
	if err != nil {
		return vacio, vecports.ErrFirmaAtestacionNoDisponible
	}
	defer clear(firma)
	huella := sha256.Sum256(mensaje)
	return vecports.NuevoResultadoFirmaAtestacionAutorizacionV3(solicitud, firma, "evidencia:firma:ct:interno:"+hex.EncodeToString(huella[:8]), f.reloj.Ahora())
}
