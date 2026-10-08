package seudonimizacionpkcs11

import (
	"context"
	"errors"
	"strings"

	registro "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
)

const (
	propositoCertificadoDER = "certificado_der"
	propositoCADer          = "ca_der"
	propositoPresentacionID = "presentacion_nonce"
)

var _ registro.SeudonimizadorPresentacionCertificado = (*Conector)(nil)

// SeudonimizarPresentacionCertificado reutiliza el alta PKCS#11 para los
// cuatro identificadores históricos. Así comparte exactamente clave, epoch,
// etiquetas y preimagen canónica con registrar_sesion_v1. Las tres huellas
// adicionales tienen propósitos fijos y nunca reciben material de clave.
// Las huellas SHA del DER proceden del certificador C4 que ya comprobó TLS,
// registro privado y CRL; este conector no sustituye esa acreditación.
func (c *Conector) SeudonimizarPresentacionCertificado(
	ctx context.Context,
	ids registro.IdentificadoresPresentacionCertificado,
) (registro.SeudonimosPresentacionCertificado, error) {
	var vacia registro.SeudonimosPresentacionCertificado
	if c == nil || ctx == nil {
		return vacia, errors.New("conector PKCS#11 no disponible")
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	if ids.EspacioIdentidad != c.config.EspacioIdentidad ||
		!identificadorValido(ids.AsercionID) ||
		!identificadorValido(ids.SesionIDAfirmada) ||
		!identificadorValido(ids.SujetoID) ||
		!identificadorValido(ids.CuentaID) ||
		ids.NonceID != ids.AsercionID ||
		!huellaDERCanonica(ids.CertificadoSHA256) ||
		!huellaDERCanonica(ids.CASHA256) ||
		ids.CertificadoSHA256 == ids.CASHA256 {
		return vacia, errors.New("identificadores de presentacion invalidos")
	}

	// El puerto de alta es la autoridad del formato HMAC de estos cuatro
	// valores; no reconstruir su preimagen en una segunda implementación.
	alta, err := c.SeudonimizarAlta(ctx, registro.IdentificadoresAlta{
		EspacioIdentidad: ids.EspacioIdentidad,
		AsercionID:       ids.AsercionID,
		SesionID:         ids.SesionIDAfirmada,
		SujetoID:         ids.SujetoID,
		CuentaID:         ids.CuentaID,
	})
	if err != nil {
		return vacia, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cerrado || c.ctx == nil {
		return vacia, errors.New("conector PKCS#11 cerrado")
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	info, err := c.ctx.GetTokenInfo(c.slot)
	if err != nil || strings.TrimSpace(info.Label) != c.config.TokenLabel ||
		strings.TrimSpace(info.SerialNumber) != c.config.TokenSerial {
		return vacia, errors.New("token PKCS#11 retirado")
	}

	salida := registro.SeudonimosPresentacionCertificado{
		Esquema:          alta.Esquema,
		EspacioIdentidad: alta.EspacioIdentidad,
		DominioRef:       alta.DominioRef,
		ClaveID:          alta.ClaveID,
		ClaveVersion:     alta.ClaveVersion,
		AsercionIDHMAC:   alta.AsercionIDHMAC,
		SesionIDHMAC:     alta.SesionIDHMAC,
		SujetoIDHMAC:     alta.SujetoIDHMAC,
		CuentaIDHMAC:     alta.CuentaIDHMAC,
	}
	valores := []struct {
		proposito string
		valor     string
		destino   *[32]byte
	}{
		{propositoCertificadoDER, ids.CertificadoSHA256, &salida.CertificadoDERHMAC},
		{propositoCADer, ids.CASHA256, &salida.CAHMAC},
		{propositoPresentacionID, ids.NonceID, &salida.NonceHMAC},
	}
	for _, valor := range valores {
		if err := ctx.Err(); err != nil {
			return vacia, err
		}
		mensaje := mensajeCanonico(c.config, valor.proposito, valor.valor)
		firma, err := c.firmar(mensaje)
		clear(mensaje)
		if err != nil || firma == [32]byte{} {
			return vacia, errors.New("HMAC PKCS#11 de presentacion fallido")
		}
		*valor.destino = firma
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	return salida, nil
}

func huellaDERCanonica(s string) bool {
	if !strings.HasPrefix(s, "sha256:") || len(s) != len("sha256:")+64 {
		return false
	}
	var distintaDeCero bool
	for _, b := range s[len("sha256:"):] {
		if b >= '1' && b <= '9' || b >= 'a' && b <= 'f' {
			distintaDeCero = true
		}
		if !(b >= '0' && b <= '9' || b >= 'a' && b <= 'f') {
			return false
		}
	}
	return distintaDeCero
}
