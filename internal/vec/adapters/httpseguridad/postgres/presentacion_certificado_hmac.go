package postgres

import (
	"context"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

// IdentificadoresPresentacionCertificado solo llegan al conector HSM/KMS
// fijado en composición. El certificado y la CA llegan como huellas SHA-256
// del DER previamente verificado, nunca como DER o secreto en PostgreSQL.
type IdentificadoresPresentacionCertificado struct {
	EspacioIdentidad  string
	AsercionID        string
	SesionIDAfirmada  string
	SujetoID          string
	CuentaID          string
	CertificadoSHA256 string
	CASHA256          string
	NonceID           string
}

// SeudonimosPresentacionCertificado aplica etiquetas de propósito separadas
// para aserción/sesión/sujeto/cuenta, certificado, CA y nonce. El proveedor
// debe emitir todas las huellas bajo UNA coordenada de clave HSM vigente.
type SeudonimosPresentacionCertificado struct {
	Esquema            string
	EspacioIdentidad   string
	DominioRef         string
	ClaveID            string
	ClaveVersion       uint64
	AsercionIDHMAC     [32]byte
	SesionIDHMAC       [32]byte
	SujetoIDHMAC       [32]byte
	CuentaIDHMAC       [32]byte
	CertificadoDERHMAC [32]byte
	CAHMAC             [32]byte
	NonceHMAC          [32]byte
}

type SeudonimizadorPresentacionCertificado interface {
	SeudonimizarPresentacionCertificado(context.Context, IdentificadoresPresentacionCertificado) (SeudonimosPresentacionCertificado, error)
}

// CoordenadasPresentacionCertificado proceden de configuración privada DBA y
// HSM, no del request. El primer corte admite solo este epoch exacto; un
// cambio de clave no coordinado deniega sin tratarlo como certificado nuevo.
type CoordenadasPresentacionCertificado struct {
	EspacioIdentidad string
	DominioRef       string
	ClaveID          string
	ClaveVersion     uint64
}

func (c CoordenadasPresentacionCertificado) valida() bool {
	return espacioIdentidadValido(c.EspacioIdentidad) &&
		referenciaTecnicaValida(c.DominioRef, "idh_") &&
		textoTecnicoValido(c.ClaveID, 128) &&
		c.ClaveVersion > 0 && c.ClaveVersion <= uint64(1<<63-1)
}

func (s SeudonimosPresentacionCertificado) validarPara(c CoordenadasPresentacionCertificado, d httpseguridad.DatosPresentacionCertificado) bool {
	if !c.valida() || s.Esquema != EsquemaHMACSHA256V1 || s.EspacioIdentidad != c.EspacioIdentidad ||
		s.DominioRef != c.DominioRef || s.ClaveID != c.ClaveID || s.ClaveVersion != c.ClaveVersion ||
		d.Emisor != c.EspacioIdentidad {
		return false
	}
	huellas := [][32]byte{s.AsercionIDHMAC, s.SesionIDHMAC, s.SujetoIDHMAC, s.CuentaIDHMAC, s.CertificadoDERHMAC, s.CAHMAC, s.NonceHMAC}
	for i, huella := range huellas {
		if huellaNula(huella) {
			return false
		}
		for j := i + 1; j < len(huellas); j++ {
			if huella == huellas[j] {
				return false
			}
		}
	}
	return true
}
