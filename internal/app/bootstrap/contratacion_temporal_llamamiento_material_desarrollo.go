package bootstrap

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	confianzaatestacion "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
)

// Misma raíz, gobierno y verificación VEC; el consumo de Bolsa tiene una
// audiencia distinta para impedir usar una capacidad de CT en otro módulo.
func nuevoProveedorMaterialBolsaDesarrollo(ctx context.Context, gobierno *pgxpool.Pool, material materialAtestacionContratacionTemporalDesarrollo, soporte *soporteAltaContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo) (*proveedorMaterialAltaContratacionTemporalDesarrollo, error) {
	// CT→Bolsa es un adaptador nominal, no una tercera frontera B-BACK.
	d := descriptorMaterialConsumidorV3Desarrollo{Audiencia: puertosbolsa.AudienciaIntegracionLlamamientoDesarrollo, Dominio: "vec.bolsa.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:bolsa:", ProveedorNominal: proveedorMaterialContratacionTemporal}
	return nuevoProveedorMaterialConsumidorConDescriptorDesarrollo(ctx, gobierno, material, soporte, reloj, d)
}

// Cada consumidor conserva clave propia y usa el mismo protocolo de gobierno.
// Los dominios anteriores de Bolsa se mantienen byte a byte.
func nuevoProveedorMaterialConsumidorDesarrollo(ctx context.Context, gobierno *pgxpool.Pool, material materialAtestacionContratacionTemporalDesarrollo, soporte *soporteAltaContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo, catalogo catalogoMaterialAutorizacionComunDesarrollo, audiencia string) (*proveedorMaterialAltaContratacionTemporalDesarrollo, error) {
	descriptor, ok := catalogo.descriptorPara(audiencia)
	if !ok {
		return nil, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	return nuevoProveedorMaterialConsumidorConDescriptorDesarrollo(ctx, gobierno, material, soporte, reloj, descriptor)
}

func nuevoProveedorMaterialConsumidorConDescriptorDesarrollo(ctx context.Context, gobierno *pgxpool.Pool, material materialAtestacionContratacionTemporalDesarrollo, soporte *soporteAltaContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo, descriptor descriptorMaterialConsumidorV3Desarrollo) (*proveedorMaterialAltaContratacionTemporalDesarrollo, error) {
	material, err := derivarMaterialConsumidorV3Desarrollo(material, descriptor)
	if err != nil {
		return nil, err
	}
	defer borrarBytes(material.claveHMAC)
	// Un solo publicador gobierna las audiencias; no se duplica el protocolo
	// ni se cambia la clave publicada para los cinco pasos anteriores.
	if err := publicarGobiernoAtestacionContratacionTemporalDesarrollo(ctx, gobierno, &material); err != nil {
		return nil, err
	}
	return nuevoProveedorMaterialAltaContratacionTemporalDesarrollo(material, soporte, reloj)
}

func derivarMaterialConsumidorV3Desarrollo(material materialAtestacionContratacionTemporalDesarrollo, descriptor descriptorMaterialConsumidorV3Desarrollo) (materialAtestacionContratacionTemporalDesarrollo, error) {
	const prefijoBase = "clave:capacidad:ct:"
	if descriptor.Audiencia == "" || descriptor.Dominio == "" || descriptor.Prefijo == "" || descriptor.ProveedorNominal == "" || len(material.claveHMAC) < sha256.Size || !strings.HasPrefix(material.claveHMACID, prefijoBase) {
		return materialAtestacionContratacionTemporalDesarrollo{}, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	mac := hmac.New(sha256.New, material.claveHMAC)
	_, _ = mac.Write([]byte(descriptor.Dominio + ".v1"))
	claveDerivada := mac.Sum(nil)
	secreto := sha256.Sum256(claveDerivada)
	huella := sha256.New()
	_, _ = huella.Write([]byte(descriptor.Dominio + ".gobierno.v1\x00"))
	_, _ = huella.Write(claveDerivada)
	material.claveHMAC = claveDerivada
	material.claveHMACID = descriptor.Prefijo + strings.TrimPrefix(material.claveHMACID, prefijoBase)
	material.claveHMACSecreto = hex.EncodeToString(secreto[:])
	material.claveHMACHuella = hex.EncodeToString(huella.Sum(nil))
	material.audienciaConsumo = descriptor.Audiencia
	capacidad, err := confianzaatestacion.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(material.claveHMACID, material.claveHMACVersion, material.claveHMAC, material.emisorID, material.audienciaConsumo, confianzaatestacion.EstadoClaveHMACCapacidadAtestacionV3Emision, material.validaDesde, material.validaHasta, time.Time{}, material.claveHMACRevision, material.claveHMACHuella)
	if err != nil {
		borrarBytes(material.claveHMAC)
		return materialAtestacionContratacionTemporalDesarrollo{}, err
	}
	material.capacidad = capacidad
	return material, nil
}

func nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx context.Context, gobierno *pgxpool.Pool, material materialAtestacionContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo, catalogo catalogoMaterialAutorizacionComunDesarrollo, audiencia string) (*proveedorMaterialAltaContratacionTemporalDesarrollo, error) {
	descriptor, ok := catalogo.descriptorPara(audiencia)
	if !ok {
		return nil, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	material, err := derivarMaterialConsumidorV3Desarrollo(material, descriptor)
	if err != nil {
		return nil, err
	}
	defer borrarBytes(material.claveHMAC)
	if err := publicarGobiernoAtestacionContratacionTemporalDesarrollo(ctx, gobierno, &material); err != nil {
		return nil, err
	}
	return nuevoProveedorMaterialAutorizacionBaseDesarrollo(material, reloj)
}
