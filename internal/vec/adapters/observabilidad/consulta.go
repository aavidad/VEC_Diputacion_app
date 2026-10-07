package observabilidad

import (
	"os"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/catalogoincidencias"
	"vec-diputacion-granada/internal/vec/domain"
)

// RegistroConsulta contiene solo los campos cerrados que necesita una consulta
// local agregada. El mensaje, la correlación y los bytes de origen no salen.
type RegistroConsulta struct {
	Esquema   string
	Instante  time.Time
	Codigo    domain.CodigoIncidenciaTecnica
	Resultado domain.CodigoResultadoTecnico
	Recuento  uint32
}

// ValidarRegistroConsulta reutiliza la validación estricta del recolector sin
// escribir en su almacén ni aceptar una línea que este rechazaría.
func ValidarRegistroConsulta(linea []byte, catalogo *catalogoincidencias.Catalogo) (RegistroConsulta, error) {
	if catalogo == nil || !catalogo.Valido() {
		return RegistroConsulta{}, os.ErrInvalid
	}
	esquema, err := esquemaLineaRecolector(linea)
	if err != nil {
		return RegistroConsulta{}, os.ErrInvalid
	}
	switch esquema {
	case domain.EsquemaIncidenciaTecnica:
		incidencia, err := validarLineaRecolectorConCatalogo(linea, catalogo)
		if err != nil {
			return RegistroConsulta{}, os.ErrInvalid
		}
		instante, err := time.Parse(formatoInstante, incidencia.Instante)
		if err != nil {
			return RegistroConsulta{}, os.ErrInvalid
		}
		return RegistroConsulta{Esquema: esquema, Instante: instante, Codigo: domain.CodigoIncidenciaTecnica(incidencia.Codigo), Recuento: incidencia.Recuento}, nil
	case domain.EsquemaResultadoTecnico:
		resultado, err := validarLineaResultadoRecolector(linea)
		if err != nil {
			return RegistroConsulta{}, os.ErrInvalid
		}
		instante, err := time.Parse(formatoInstante, resultado.Instante)
		if err != nil {
			return RegistroConsulta{}, os.ErrInvalid
		}
		return RegistroConsulta{Esquema: esquema, Instante: instante, Resultado: domain.CodigoResultadoTecnico(resultado.Resultado), Recuento: 1}, nil
	}
	return RegistroConsulta{}, os.ErrInvalid
}
