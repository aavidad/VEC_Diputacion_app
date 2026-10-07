package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	cose "github.com/veraison/go-cose"
	"time"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type clock struct{}

func (clock) Ahora() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }

// Only the signer is local synthetic infrastructure; Ed25519/COSE verification
// and the HMAC emitter are the same implementations used by the common service.
type signer struct {
	private ed25519.PrivateKey
	header  vd.CabeceraAtestacionAutorizacionV3
}

func (s signer) FirmarAtestacionAutorizacionV3(ctx context.Context, request vp.SolicitudFirmaAtestacionAutorizacionV3) (vp.ResultadoFirmaAtestacionAutorizacionV3, error) {
	if err := ctx.Err(); err != nil {
		return vp.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	header, err := request.Cabecera()
	if err != nil || header != s.header {
		return vp.ResultadoFirmaAtestacionAutorizacionV3{}, errors.New("signer_header")
	}
	payload, err := request.Mensaje()
	if err != nil {
		return vp.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	aad, err := confianza.AADExternoAtestacionAutorizacionV3(header.Audiencia)
	if err != nil {
		return vp.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	message := cose.NewSign1Message()
	message.Headers.Protected.SetAlgorithm(cose.AlgorithmEdDSA)
	message.Headers.Protected[cose.HeaderLabelKeyID] = []byte(header.ClaveID)
	message.Payload = payload
	provider, err := cose.NewSigner(cose.AlgorithmEdDSA, s.private)
	if err != nil {
		return vp.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	if err = message.Sign(rand.Reader, aad, provider); err != nil {
		return vp.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	message.Payload = nil
	message.Headers.RawProtected = nil
	message.Headers.RawUnprotected = nil
	signed, err := message.MarshalCBOR()
	if err != nil {
		return vp.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	return vp.NuevoResultadoFirmaAtestacionAutorizacionV3(request, signed, "evidencia:rum03:firma-sintetica", clock{}.Ahora())
}

func trust(c cryptoConfig) (confianza.ConfiguracionConfianzaAtestacionAutorizacionV3, ed25519.PrivateKey, error) {
	if len(c.Seed) != ed25519.SeedSize {
		return confianza.ConfiguracionConfianzaAtestacionAutorizacionV3{}, nil, errors.New("seed_size")
	}
	key := ed25519.NewKeyFromSeed(c.Seed)
	root, err := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(c.RootID, c.RootVersion, key.Public().(ed25519.PublicKey), c.Deployment, confianza.EstadoClaveAtestacionAutorizacionV3Activa, c.RootFrom, c.RootUntil, time.Time{})
	if err != nil {
		return confianza.ConfiguracionConfianzaAtestacionAutorizacionV3{}, nil, err
	}
	cfg, err := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(c.Revision, c.Sequence, c.Published, c.Expires, root)
	return cfg, key, err
}
