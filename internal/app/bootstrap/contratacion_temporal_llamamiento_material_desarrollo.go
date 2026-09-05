package bootstrap

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

// Misma raíz, gobierno y verificación VEC; el consumo de Bolsa tiene una
// audiencia distinta para impedir usar una capacidad de CT en otro módulo.
func nuevoProveedorMaterialBolsaDesarrollo(ctx context.Context, gobierno *pgxpool.Pool, material materialAtestacionContratacionTemporalDesarrollo, soporte *soporteAltaContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo) (*proveedorMaterialAltaContratacionTemporalDesarrollo, error) {
	if len(material.claveHMAC) < 32 || !strings.HasPrefix(material.claveHMACID, "clave:capacidad:ct:") {
		return nil, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	mac := hmac.New(sha256.New, material.claveHMAC)
	_, _ = mac.Write([]byte("vec.bolsa.desarrollo.capacidad-v3.v1"))
	material.claveHMAC = mac.Sum(nil)
	defer borrarBytes(material.claveHMAC)
	material.claveHMACID = strings.Replace(material.claveHMACID, "clave:capacidad:ct:", "clave:capacidad:bolsa:", 1)
	material.audienciaConsumo = puertosbolsa.AudienciaIntegracionLlamamientoDesarrollo
	huellaSecreto := sha256.Sum256(material.claveHMAC)
	material.claveHMACSecreto = hex.EncodeToString(huellaSecreto[:])
	huellaGobierno := sha256.New()
	_, _ = huellaGobierno.Write([]byte("vec.bolsa.desarrollo.capacidad-v3.gobierno.v1\x00"))
	_, _ = huellaGobierno.Write(material.claveHMAC)
	material.claveHMACHuella = hex.EncodeToString(huellaGobierno.Sum(nil))
	// Un solo publicador gobierna ambas audiencias; no se duplica el protocolo
	// ni se cambia la clave publicada para los cinco pasos anteriores.
	if err := publicarGobiernoAtestacionContratacionTemporalDesarrollo(ctx, gobierno, &material); err != nil {
		return nil, err
	}
	return nuevoProveedorMaterialAltaContratacionTemporalDesarrollo(material, soporte, reloj)
}
