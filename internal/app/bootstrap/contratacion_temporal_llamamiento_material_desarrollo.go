package bootstrap

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	puertosct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Misma raíz, gobierno y verificación VEC; el consumo de Bolsa tiene una
// audiencia distinta para impedir usar una capacidad de CT en otro módulo.
func nuevoProveedorMaterialBolsaDesarrollo(ctx context.Context, gobierno *pgxpool.Pool, material materialAtestacionContratacionTemporalDesarrollo, soporte *soporteAltaContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo) (*proveedorMaterialAltaContratacionTemporalDesarrollo, error) {
	return nuevoProveedorMaterialConsumidorDesarrollo(ctx, gobierno, material, soporte, reloj, puertosbolsa.AudienciaIntegracionLlamamientoDesarrollo)
}

// Cada consumidor conserva clave propia y usa el mismo protocolo de gobierno.
// Los dominios anteriores de Bolsa se mantienen byte a byte.
func nuevoProveedorMaterialConsumidorDesarrollo(ctx context.Context, gobierno *pgxpool.Pool, material materialAtestacionContratacionTemporalDesarrollo, soporte *soporteAltaContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo, audiencia string) (*proveedorMaterialAltaContratacionTemporalDesarrollo, error) {
	var dominio, prefijo string
	switch audiencia {
	case puertosbolsa.AudienciaIntegracionLlamamientoDesarrollo:
		dominio, prefijo = "vec.bolsa.desarrollo.capacidad-v3", "clave:capacidad:bolsa:"
	case puertosct.AudienciaConsumoConsultaCuadroRRHHV3:
		dominio, prefijo = "vec.ct.cuadro-rrhh.desarrollo.capacidad-v3", "clave:capacidad:ct-cuadro:"
	case puertosct.AudienciaConsumoConsultaDetalleRRHHV3:
		dominio, prefijo = "vec.ct.detalle-rrhh.desarrollo.capacidad-v3", "clave:capacidad:ct-detalle:"
	default:
		return nil, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	if len(material.claveHMAC) < 32 || !strings.HasPrefix(material.claveHMACID, "clave:capacidad:ct:") {
		return nil, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	mac := hmac.New(sha256.New, material.claveHMAC)
	_, _ = mac.Write([]byte(dominio + ".v1"))
	material.claveHMAC = mac.Sum(nil)
	defer borrarBytes(material.claveHMAC)
	material.claveHMACID = strings.Replace(material.claveHMACID, "clave:capacidad:ct:", prefijo, 1)
	material.audienciaConsumo = audiencia
	huellaSecreto := sha256.Sum256(material.claveHMAC)
	material.claveHMACSecreto = hex.EncodeToString(huellaSecreto[:])
	huellaGobierno := sha256.New()
	_, _ = huellaGobierno.Write([]byte(dominio + ".gobierno.v1\x00"))
	_, _ = huellaGobierno.Write(material.claveHMAC)
	material.claveHMACHuella = hex.EncodeToString(huellaGobierno.Sum(nil))
	// Un solo publicador gobierna las audiencias; no se duplica el protocolo
	// ni se cambia la clave publicada para los cinco pasos anteriores.
	if err := publicarGobiernoAtestacionContratacionTemporalDesarrollo(ctx, gobierno, &material); err != nil {
		return nil, err
	}
	return nuevoProveedorMaterialAltaContratacionTemporalDesarrollo(material, soporte, reloj)
}
