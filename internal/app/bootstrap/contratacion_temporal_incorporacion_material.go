package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"time"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	alta "vec-diputacion-granada/internal/modules/personal/adapters/contrataciontemporal"
	lectura "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
)

// Sólo capacidades YA gobernadas. Firmante, raíz y confianza proceden de la
// composición actual. No genera claves, publica gobierno ni firma actos.
type archivoMaterialIncorporacionV2 struct {
	Alta    archivoCapacidadIncorporacionV2 `json:"alta"`
	Lectura archivoCapacidadIncorporacionV2 `json:"lectura"`
	CT      archivoCapacidadIncorporacionV2 `json:"ct"`
}

type archivoCapacidadIncorporacionV2 struct {
	ClaveID          string    `json:"clave_id"`
	Version          uint64    `json:"version"`
	File             string    `json:"file"`
	SHA256           string    `json:"sha256"`
	EmisorID         string    `json:"emisor_id"`
	Desde            time.Time `json:"desde"`
	Hasta            time.Time `json:"hasta"`
	RevisionGobierno uint64    `json:"revision_gobierno"`
	HuellaGobierno   string    `json:"huella_gobierno"`
}

func cargarMaterialIncorporacionV2(raiz *os.Root, c archivoMaterialIncorporacionV2, raizPublica confianza.RaizPublicaAtestacionAutorizacionV3, reloj relojContratacionTemporalDesarrollo) (inc.EmisionesAutoridad, error) {
	f := ct.ErrComposicionIncorporacionAplicacion
	vacia := inc.EmisionesAutoridad{}
	emisiones := make([]*confianza.EmisorCapacidadesAtestacionAutorizacionV3, 0, 3)
	ids, archivos := map[string]bool{}, map[string]bool{}
	ahora := reloj.Ahora()
	for i, capacidad := range []archivoCapacidadIncorporacionV2{c.Alta, c.Lectura, c.CT} {
		if ids[capacidad.ClaveID] || archivos[capacidad.File] {
			return vacia, f
		}
		ids[capacidad.ClaveID], archivos[capacidad.File] = true, true
		audiencia := []string{alta.AudienciaAltaEjercicio, lectura.AudienciaV2, ct.AudienciaConfirmacionIncorporacionV2}[i]
		secreto, err := leerArchivoIncorporacionV2(raiz, capacidad.File, 256)
		if err != nil {
			return vacia, f
		}
		h := sha256.Sum256(secreto)
		if hex.EncodeToString(h[:]) != capacidad.SHA256 {
			borrarBytes(secreto)
			return vacia, f
		}
		clave, err := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(capacidad.ClaveID, capacidad.Version, secreto, capacidad.EmisorID, audiencia, confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, capacidad.Desde, capacidad.Hasta, time.Time{}, capacidad.RevisionGobierno, capacidad.HuellaGobierno)
		borrarBytes(secreto)
		if err != nil || ahora.Before(capacidad.Desde) || !ahora.Before(capacidad.Hasta) {
			return vacia, f
		}
		emisor, err := confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(clave, reloj)
		if err != nil {
			return vacia, f
		}
		emisiones = append(emisiones, emisor)
	}
	return inc.EmisionesAutoridad{Alta: inc.EmisionAutoridad{Emisor: emisiones[0], Raiz: raizPublica}, Lectura: inc.EmisionAutoridad{Emisor: emisiones[1], Raiz: raizPublica}, CT: inc.EmisionAutoridad{Emisor: emisiones[2], Raiz: raizPublica}}, nil
}
