package ports

import (
	"encoding/json"
	"time"
)

// CapturaLecturaNominalRespuestasLlamamiento transporta al dueño de Bolsa la
// autorización puntual y el vínculo emitidos por las autoridades comunes.
// El adaptador debe cotejarla y registrar el acceso en la misma transacción
// que lee las respuestas; su contenido no procede de la petición HTTP.
type CapturaLecturaNominalRespuestasLlamamiento struct {
	ContextoCanonico []byte
	VinculoCanonico  []byte
	Orden            json.RawMessage
	BolsaRef         string
	LlamamientoRef   string
	ValidaHasta      time.Time
}
