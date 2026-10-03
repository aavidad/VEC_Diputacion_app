package observabilidad

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// validarLineaResultadoRecolector reconstruye un resultado técnico desde el
// catálogo y el esquema exactos. El JSON libre nunca se copia al archivo.
func validarLineaResultadoRecolector(datos []byte) (lineaResultado, error) {
	var l lineaResultado
	if clavesUnicasRecolector(datos) != nil {
		return l, os.ErrInvalid
	}
	d := json.NewDecoder(bytes.NewReader(datos))
	d.DisallowUnknownFields()
	if d.Decode(&l) != nil || d.Decode(new(any)) != io.EOF {
		return lineaResultado{}, os.ErrInvalid
	}
	instante, err := time.Parse(formatoInstante, l.Instante)
	if err != nil || instante.Format(formatoInstante) != l.Instante {
		return lineaResultado{}, os.ErrInvalid
	}
	resultado := domain.ResultadoTecnico{
		Esquema: l.Esquema, Instante: instante,
		Resultado:      domain.CodigoResultadoTecnico(l.Resultado),
		Nivel:          domain.NivelResultadoTecnico(l.Nivel),
		Componente:     domain.ComponenteIncidenciaTecnica(l.Componente),
		Etapa:          domain.EtapaIncidenciaTecnica(l.Etapa),
		Entorno:        domain.EntornoIncidenciaTecnica(l.Entorno),
		VersionBinario: l.VersionBinario,
		Correlacion:    l.Correlacion, CorrelacionRef: l.CorrelacionRef,
	}
	if resultado.Validar() != nil {
		return lineaResultado{}, os.ErrInvalid
	}
	return lineaResultado{
		Esquema: resultado.Esquema, Instante: resultado.Instante.Format(formatoInstante),
		Resultado: string(resultado.Resultado), Nivel: string(resultado.Nivel),
		Componente: string(resultado.Componente), Etapa: string(resultado.Etapa),
		Entorno: string(resultado.Entorno), VersionBinario: resultado.VersionBinario,
		Correlacion: resultado.Correlacion, CorrelacionRef: resultado.CorrelacionRef,
	}, nil
}
