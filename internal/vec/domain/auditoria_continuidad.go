package domain

import "strconv"

const EsquemaContinuidadCheckpointDesarrollo = "vec.auditoria.continuidad.desarrollo.v1"

type InformeContinuidadCheckpoint struct {
	Continuidad        string `json:"continuidad"`
	Codigo             string `json:"codigo"`
	Clave              string `json:"clave,omitempty"`
	Esperado           string `json:"esperado,omitempty"`
	Obtenido           string `json:"obtenido,omitempty"`
	RecibosVerificados uint64 `json:"recibos_verificados"`
	PrimeraSecuencia   uint64 `json:"primera_secuencia"`
	UltimaSecuencia    uint64 `json:"ultima_secuencia"`
	RegistrosTotal     uint64 `json:"registros_total"`
}

func RechazoContinuidadCheckpoint(clave, esperado, obtenido string) InformeContinuidadCheckpoint {
	return InformeContinuidadCheckpoint{Continuidad: "rechazada", Codigo: "continuidad_rechazada", Clave: clave, Esperado: esperado, Obtenido: obtenido}
}

// CotejarContinuidadCheckpoint comprueba coordenadas y configuración idénticas.
// No verifica firmas ni recalcula eslabones de los registros de auditoría.
func CotejarContinuidadCheckpoint(ancla ReciboCheckpointDesarrollo, recibos []ReciboCheckpointDesarrollo, maxRegistros uint64, maxRecibos int) InformeContinuidadCheckpoint {
	if maxRecibos < 1 || maxRecibos > 256 || len(recibos) < 1 || len(recibos) > maxRecibos || maxRegistros == 0 {
		return RechazoContinuidadCheckpoint("limites", "validos", "invalidos")
	}
	if _, err := ancla.CanonicoParaFirma(); err != nil {
		return RechazoContinuidadCheckpoint("ancla", "valida", "invalida")
	}
	previa := ancla.Checkpoint.Cobertura
	total, primera := previa.Registros, previa.PrimeraSecuencia
	if total > maxRegistros {
		return RechazoContinuidadCheckpoint("registros_total", strconv.FormatUint(maxRegistros, 10), strconv.FormatUint(total, 10))
	}
	for _, recibo := range recibos {
		if _, err := recibo.CanonicoParaFirma(); err != nil {
			return RechazoContinuidadCheckpoint("recibo", "valido", "invalido")
		}
		c := recibo.Checkpoint.Cobertura
		if recibo.Checkpoint.Politica != ancla.Checkpoint.Politica || recibo.PinSPKISHA256 != ancla.PinSPKISHA256 || c.CadenaID != previa.CadenaID {
			return RechazoContinuidadCheckpoint("politica_raiz_cadena", "identicas", "distintas")
		}
		if c.Registros == 0 {
			return RechazoContinuidadCheckpoint("tramo", "no_vacio", "vacio")
		}
		if previa.UltimaSecuencia == 9007199254740991 || c.PrimeraSecuencia != previa.UltimaSecuencia+1 {
			return RechazoContinuidadCheckpoint("primera_secuencia", strconv.FormatUint(previa.UltimaSecuencia+1, 10), strconv.FormatUint(c.PrimeraSecuencia, 10))
		}
		if c.AnteriorSHA256 != previa.CabezaSHA256 {
			return RechazoContinuidadCheckpoint("anterior_sha256", "igual_a_cabeza_previa", "distinto")
		}
		// Restar antes de sumar evita desbordamiento y cuenta también el ancla.
		if c.Registros > maxRegistros-total {
			return RechazoContinuidadCheckpoint("registros_restantes", strconv.FormatUint(maxRegistros-total, 10), strconv.FormatUint(c.Registros, 10))
		}
		total += c.Registros
		if primera == 0 {
			primera = c.PrimeraSecuencia
		}
		previa = c
	}
	return InformeContinuidadCheckpoint{Continuidad: "verificada", Codigo: "continuidad_correcta", RecibosVerificados: uint64(len(recibos)) + 1,
		PrimeraSecuencia: primera, UltimaSecuencia: previa.UltimaSecuencia, RegistrosTotal: total}
}
