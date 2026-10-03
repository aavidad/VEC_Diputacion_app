package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
)

// El esquema y el dominio son formatos técnicos, no políticas de negocio.
const EsquemaCheckpointDesarrollo = "vec.auditoria.checkpoint.desarrollo.v1"
const DominioFirmaCheckpointDesarrollo = "vec.kms.desarrollo.auditoria.checkpoint.ed25519.v1"

var ErrCheckpointInvalido = errors.New("checkpoint_invalido")
var codigoCheckpoint = regexp.MustCompile(`\A[a-zA-Z0-9][a-zA-Z0-9._:-]{0,159}\z`)

// CoberturaCheckpoint conserva las coordenadas del verificador AD3 existente.
// No contiene registros, identidades ni preimágenes personales.
type CoberturaCheckpoint struct {
	CadenaID         string `json:"cadena_id"`
	PrimeraSecuencia uint64 `json:"primera_secuencia"`
	UltimaSecuencia  uint64 `json:"ultima_secuencia"`
	AnteriorSHA256   string `json:"anterior_sha256"`
	CabezaSHA256     string `json:"cabeza_sha256"`
	Registros        uint64 `json:"registros"`
}

func (c CoberturaCheckpoint) Validar() error {
	if !codigoCheckpoint.MatchString(c.CadenaID) || !SHA256CheckpointValido(c.AnteriorSHA256) || !SHA256CheckpointValido(c.CabezaSHA256) || c.UltimaSecuencia > 9007199254740991 {
		return ErrCheckpointInvalido
	}
	cero := strings.Repeat("0", 64)
	if c.Registros == 0 {
		if c.PrimeraSecuencia != 0 || c.UltimaSecuencia != 0 || c.AnteriorSHA256 != cero || c.CabezaSHA256 != cero {
			return ErrCheckpointInvalido
		}
		return nil
	}
	if c.PrimeraSecuencia < 1 || c.UltimaSecuencia < c.PrimeraSecuencia || c.Registros != c.UltimaSecuencia-c.PrimeraSecuencia+1 || c.PrimeraSecuencia == 1 && c.AnteriorSHA256 != cero {
		return ErrCheckpointInvalido
	}
	return nil
}

type PoliticaCheckpoint struct {
	Version             uint64 `json:"version"`
	PoliticaRef         string `json:"politica_ref"`
	PoliticaVersion     uint64 `json:"politica_version"`
	ClaveRef            string `json:"clave_ref"`
	ClaveVersion        uint64 `json:"clave_version"`
	ProveedorKMS        string `json:"proveedor_kms"`
	ProveedorKMSVersion uint64 `json:"proveedor_kms_version"`
	ProveedorTSA        string `json:"proveedor_tsa"`
	ProveedorTSAVersion uint64 `json:"proveedor_tsa_version"`
	OperacionTSA        string `json:"operacion_tsa"`
	Modo                string `json:"modo"`
}

func (p PoliticaCheckpoint) Validar() error {
	if p.Modo != "DESARROLLO" || p.Version != 1 || p.PoliticaVersion == 0 || p.ClaveVersion == 0 || p.ProveedorKMSVersion == 0 || p.ProveedorTSAVersion == 0 {
		return ErrCheckpointInvalido
	}
	for _, c := range []string{p.PoliticaRef, p.ClaveRef, p.ProveedorKMS, p.ProveedorTSA, p.OperacionTSA} {
		if !codigoCheckpoint.MatchString(c) {
			return ErrCheckpointInvalido
		}
	}
	return nil
}

type CheckpointDesarrollo struct {
	Esquema   string              `json:"esquema"`
	Politica  PoliticaCheckpoint  `json:"politica"`
	Cobertura CoberturaCheckpoint `json:"cobertura"`
}

func (c CheckpointDesarrollo) Canonico() ([]byte, error) {
	if c.Esquema != EsquemaCheckpointDesarrollo || c.Politica.Validar() != nil || c.Cobertura.Validar() != nil {
		return nil, ErrCheckpointInvalido
	}
	return json.Marshal(c)
}

type ReciboTSACheckpoint struct {
	Referencia             string `json:"referencia"`
	HuellaPreimagenSHA256  string `json:"huella_preimagen_sha256"`
	HuellaCheckpointSHA256 string `json:"huella_checkpoint_sha256"`
	Autoridad              string `json:"autoridad"`
	Esquema                string `json:"esquema"`
}

type ReciboCheckpointDesarrollo struct {
	Checkpoint    CheckpointDesarrollo `json:"checkpoint"`
	TSA           ReciboTSACheckpoint  `json:"tsa"`
	PinSPKISHA256 string               `json:"pin_spki_sha256"`
	FirmaBase64   string               `json:"firma_base64"`
}

// CanonicoParaFirma siempre serializa el mismo formato estructurado cerrado.
// La firma no cubre su propio valor; sí cubre checkpoint, versiones, TSA y pin.
func (r ReciboCheckpointDesarrollo) CanonicoParaFirma() ([]byte, error) {
	c, err := r.Checkpoint.Canonico()
	if err != nil {
		return nil, err
	}
	if !SHA256CheckpointValido(r.PinSPKISHA256) || r.TSA.HuellaCheckpointSHA256 != HuellaCheckpoint(c) || !SHA256CheckpointValido(r.TSA.HuellaPreimagenSHA256) || r.TSA.Autoridad != "no_autoritativo" || r.TSA.Esquema != "vec.tsa.desarrollo.v1" || !strings.HasPrefix(r.TSA.Referencia, "tsa-desarrollo:hmac-sha256:") || !SHA256CheckpointValido(strings.TrimPrefix(r.TSA.Referencia, "tsa-desarrollo:hmac-sha256:")) {
		return nil, ErrCheckpointInvalido
	}
	return json.Marshal(struct {
		Dominio    string               `json:"dominio"`
		Checkpoint CheckpointDesarrollo `json:"checkpoint"`
		TSA        ReciboTSACheckpoint  `json:"tsa"`
		Pin        string               `json:"pin_spki_sha256"`
	}{DominioFirmaCheckpointDesarrollo, r.Checkpoint, r.TSA, r.PinSPKISHA256})
}
func HuellaCheckpoint(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func SHA256CheckpointValido(v string) bool {
	if len(v) != 64 {
		return false
	}
	for _, c := range v {
		if c < '0' || c > '9' {
			if c < 'a' || c > 'f' {
				return false
			}
		}
	}
	return true
}
