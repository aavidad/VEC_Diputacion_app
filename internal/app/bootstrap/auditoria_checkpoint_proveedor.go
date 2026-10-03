package bootstrap

import (
	"context"
	"crypto/ed25519"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type proveedorCheckpointDesarrollo struct {
	politica domain.PoliticaCheckpoint
	privada  ed25519.PrivateKey
	publica  ed25519.PublicKey
	pin      string
	tsa      ports.TimestampPort
}

// NuevoProveedorCheckpointDesarrollo utiliza el KMS y la TSA locales existentes.
// El material lo aporta el operador; no se genera ni se reutiliza otra privada.
func NuevoProveedorCheckpointDesarrollo(maestra, claveTSA [32]byte, p domain.PoliticaCheckpoint) (*proveedorCheckpointDesarrollo, error) {
	defer clear(maestra[:])
	defer clear(claveTSA[:])
	if maestra == [32]byte{} || claveTSA == [32]byte{} || maestra == claveTSA || p.Validar() != nil {
		return nil, domain.ErrCheckpointInvalido
	}
	envoltura := derivarClaveDesarrollo(maestra, "vec.kms.desarrollo.envoltura.v1")
	defer clear(envoltura[:])
	semilla := derivarClaveDesarrollo(envoltura, domain.DominioFirmaCheckpointDesarrollo)
	defer clear(semilla[:])
	privada := ed25519.NewKeyFromSeed(semilla[:])
	publica := privada.Public().(ed25519.PublicKey)
	der, err := x509.MarshalPKIXPublicKey(publica)
	if err != nil {
		clear(privada)
		return nil, domain.ErrCheckpointInvalido
	}
	return &proveedorCheckpointDesarrollo{politica: p, privada: privada, publica: publica, pin: domain.HuellaCheckpoint(der), tsa: nuevoSelladorTiempoDesarrollo(claveTSA)}, nil
}

// NuevoVerificadorCheckpointDesarrollo recibe la raíz pública por canal separado.
// El recibo no contiene una pública y no puede fijar su propia raíz confiable.
func NuevoVerificadorCheckpointDesarrollo(spkiDER []byte, pin string, p domain.PoliticaCheckpoint) (ports.VerificadorCheckpointDesarrollo, error) {
	k, err := x509.ParsePKIXPublicKey(spkiDER)
	if err != nil || p.Validar() != nil || !domain.SHA256CheckpointValido(pin) || domain.HuellaCheckpoint(spkiDER) != pin {
		return nil, domain.ErrCheckpointInvalido
	}
	pub, ok := k.(ed25519.PublicKey)
	if !ok {
		return nil, domain.ErrCheckpointInvalido
	}
	return &proveedorCheckpointDesarrollo{politica: p, publica: append(ed25519.PublicKey(nil), pub...), pin: pin}, nil
}
func (p *proveedorCheckpointDesarrollo) PinCheckpoint() string {
	if p == nil {
		return ""
	}
	return p.pin
}
func (p *proveedorCheckpointDesarrollo) PublicaCheckpointDER() ([]byte, error) {
	if p == nil || len(p.publica) != ed25519.PublicKeySize {
		return nil, domain.ErrCheckpointInvalido
	}
	return x509.MarshalPKIXPublicKey(p.publica)
}
func (p *proveedorCheckpointDesarrollo) CerrarCheckpoint() {
	if p != nil {
		clear(p.privada)
		if tsa, ok := p.tsa.(*selladorTiempoDesarrollo); ok {
			clear(tsa.clave[:])
		}
		p.privada = nil
		p.tsa = nil
	}
}
func (p *proveedorCheckpointDesarrollo) SellarCheckpoint(ctx context.Context, c domain.CheckpointDesarrollo) (domain.ReciboTSACheckpoint, error) {
	if p == nil || p.tsa == nil || c.Politica != p.politica {
		return domain.ReciboTSACheckpoint{}, domain.ErrCheckpointInvalido
	}
	b, err := c.Canonico()
	if err != nil {
		return domain.ReciboTSACheckpoint{}, err
	}
	h := domain.HuellaCheckpoint(b)
	r, err := p.tsa.Timestamp(ctx, ports.InteropRequest{Operation: p.politica.OperacionTSA, Subject: c.Cobertura.CadenaID, Payload: map[string]string{"huella_checkpoint_sha256": h}})
	if err != nil {
		return domain.ReciboTSACheckpoint{}, err
	}
	return domain.ReciboTSACheckpoint{Referencia: r.Reference, HuellaCheckpointSHA256: h, HuellaPreimagenSHA256: r.Payload["huella_preimagen_sha256"], Autoridad: r.Status, Esquema: r.Payload["esquema"]}, nil
}
func (p *proveedorCheckpointDesarrollo) FirmarCheckpoint(ctx context.Context, r domain.ReciboCheckpointDesarrollo) (domain.ReciboCheckpointDesarrollo, error) {
	if p == nil || ctx == nil || ctx.Err() != nil || len(p.privada) != ed25519.PrivateKeySize || r.Checkpoint.Politica != p.politica || r.PinSPKISHA256 != p.pin || r.FirmaBase64 != "" {
		return domain.ReciboCheckpointDesarrollo{}, domain.ErrCheckpointInvalido
	}
	// La emisión solo firma el recibo del proveedor TSA actual ligado a la carga.
	sello, err := p.SellarCheckpoint(ctx, r.Checkpoint)
	if err != nil || sello != r.TSA {
		return domain.ReciboCheckpointDesarrollo{}, domain.ErrCheckpointInvalido
	}
	b, err := r.CanonicoParaFirma()
	if err != nil {
		return domain.ReciboCheckpointDesarrollo{}, err
	}
	r.FirmaBase64 = base64.StdEncoding.EncodeToString(ed25519.Sign(p.privada, b))
	return r, nil
}
func (p *proveedorCheckpointDesarrollo) VerificarCheckpoint(ctx context.Context, r domain.ReciboCheckpointDesarrollo) error {
	if p == nil || ctx == nil || ctx.Err() != nil || len(p.publica) != ed25519.PublicKeySize || r.Checkpoint.Politica != p.politica || subtle.ConstantTimeCompare([]byte(r.PinSPKISHA256), []byte(p.pin)) != 1 {
		return domain.ErrCheckpointInvalido
	}
	b, err := r.CanonicoParaFirma()
	if err != nil {
		return err
	}
	s, err := base64.StdEncoding.Strict().DecodeString(r.FirmaBase64)
	if err != nil || len(s) != ed25519.SignatureSize || base64.StdEncoding.EncodeToString(s) != r.FirmaBase64 || !ed25519.Verify(p.publica, b, s) {
		return domain.ErrCheckpointInvalido
	}
	return nil
}
