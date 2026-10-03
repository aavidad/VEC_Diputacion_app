package domain

import (
	"errors"
	"time"
)

const EsquemaResultadoTecnico = "vec.resultado_tecnico.v1"

var ErrResultadoTecnicoInvalido = errors.New("vec: resultado tecnico invalido")

// CodigoResultadoTecnico separa el resultado observado del nivel del registro.
// Un resultado correcto nunca se clasifica como incidencia.
type CodigoResultadoTecnico string

const (
	ResultadoTecnicoCorrecto        CodigoResultadoTecnico = "correcto"
	ResultadoTecnicoDenegado        CodigoResultadoTecnico = "denegado"
	ResultadoTecnicoEntradaInvalida CodigoResultadoTecnico = "entrada_invalida"
	ResultadoTecnicoCancelado       CodigoResultadoTecnico = "cancelado"
	ResultadoTecnicoNoDisponible    CodigoResultadoTecnico = "no_disponible"
)

type NivelResultadoTecnico string

const (
	NivelResultadoTecnicoInfo  NivelResultadoTecnico = "info"
	NivelResultadoTecnicoWarn  NivelResultadoTecnico = "warn"
	NivelResultadoTecnicoError NivelResultadoTecnico = "error"
)

// SolicitudResultadoTecnico solo admite códigos técnicos. Nunca transporta
// identidad, recurso, texto libre, error de biblioteca, cabeceras ni cuerpo.
type SolicitudResultadoTecnico struct {
	Resultado  CodigoResultadoTecnico
	Componente ComponenteIncidenciaTecnica
	Etapa      EtapaIncidenciaTecnica
}

type ClasificacionResultadoTecnico struct {
	Resultado  CodigoResultadoTecnico
	Nivel      NivelResultadoTecnico
	Componente ComponenteIncidenciaTecnica
	Etapa      EtapaIncidenciaTecnica
}

// DefinicionResultadoTecnicoDe es el catálogo cerrado de resultado y nivel.
func DefinicionResultadoTecnicoDe(codigo CodigoResultadoTecnico) (NivelResultadoTecnico, bool) {
	switch codigo {
	case ResultadoTecnicoCorrecto, ResultadoTecnicoCancelado:
		return NivelResultadoTecnicoInfo, true
	case ResultadoTecnicoDenegado, ResultadoTecnicoEntradaInvalida:
		return NivelResultadoTecnicoWarn, true
	case ResultadoTecnicoNoDisponible:
		return NivelResultadoTecnicoError, true
	default:
		return "", false
	}
}

func ClasificarResultadoTecnico(s SolicitudResultadoTecnico) (ClasificacionResultadoTecnico, error) {
	nivel, ok := DefinicionResultadoTecnicoDe(s.Resultado)
	if !ok {
		return ClasificacionResultadoTecnico{}, ErrResultadoTecnicoInvalido
	}
	var componente ComponenteIncidenciaTecnica
	var etapa EtapaIncidenciaTecnica
	for _, codigo := range CodigosIncidenciaTecnica() {
		definicion, existe := DefinicionIncidenciaTecnicaDe(codigo)
		if !existe {
			continue
		}
		if canon, valido := canonicoEn(definicion.Componentes, s.Componente); valido {
			componente = canon
		}
		if canon, valido := canonicoEn(definicion.Etapas, s.Etapa); valido {
			etapa = canon
		}
	}
	if componente == "" || etapa == "" {
		return ClasificacionResultadoTecnico{}, ErrResultadoTecnicoInvalido
	}
	resultadoCanonico := s.Resultado
	switch s.Resultado {
	case ResultadoTecnicoCorrecto:
		resultadoCanonico = ResultadoTecnicoCorrecto
	case ResultadoTecnicoDenegado:
		resultadoCanonico = ResultadoTecnicoDenegado
	case ResultadoTecnicoEntradaInvalida:
		resultadoCanonico = ResultadoTecnicoEntradaInvalida
	case ResultadoTecnicoCancelado:
		resultadoCanonico = ResultadoTecnicoCancelado
	case ResultadoTecnicoNoDisponible:
		resultadoCanonico = ResultadoTecnicoNoDisponible
	}
	return ClasificacionResultadoTecnico{
		Resultado: resultadoCanonico, Nivel: nivel, Componente: componente, Etapa: etapa,
	}, nil
}

// ResultadoTecnico es la proyección técnica cerrada que se entrega al emisor.
// La misma correlación enlaza el identificador privado de la petición y la
// referencia canónica V3 consumida por SQL.
type ResultadoTecnico struct {
	Esquema        string
	Instante       time.Time
	Resultado      CodigoResultadoTecnico
	Nivel          NivelResultadoTecnico
	Componente     ComponenteIncidenciaTecnica
	Etapa          EtapaIncidenciaTecnica
	Entorno        EntornoIncidenciaTecnica
	VersionBinario string
	Correlacion    string
	CorrelacionRef string
}

func NuevoResultadoTecnico(
	c ClasificacionResultadoTecnico, instante time.Time,
	entorno EntornoIncidenciaTecnica, version, correlacion string,
	referencia ReferenciaCorrelacionAutorizacionV2,
) (ResultadoTecnico, error) {
	comprobada, err := ClasificarResultadoTecnico(SolicitudResultadoTecnico{
		Resultado: c.Resultado, Componente: c.Componente, Etapa: c.Etapa,
	})
	valor, errReferencia := referencia.ValorCanonico()
	if err != nil || errReferencia != nil || c != comprobada ||
		!EsCorrelacionTecnicaValida(correlacion) || valor != "correlacion_"+correlacion ||
		instante.IsZero() {
		return ResultadoTecnico{}, ErrResultadoTecnicoInvalido
	}
	resultado := ResultadoTecnico{
		Esquema:   EsquemaResultadoTecnico,
		Instante:  instante.UTC().Truncate(time.Millisecond),
		Resultado: c.Resultado, Nivel: c.Nivel,
		Componente: c.Componente, Etapa: c.Etapa,
		Entorno:        NormalizarEntornoIncidenciaTecnica(string(entorno)),
		VersionBinario: NormalizarVersionBinario(version),
		Correlacion:    correlacion, CorrelacionRef: valor,
	}
	if resultado.Validar() != nil {
		return ResultadoTecnico{}, ErrResultadoTecnicoInvalido
	}
	return resultado, nil
}

// Validar permite al recolector aceptar únicamente la proyección cerrada del
// emisor, sin reconstruir una capacidad de autorización desde una línea JSON.
func (r ResultadoTecnico) Validar() error {
	clasificacion, err := ClasificarResultadoTecnico(SolicitudResultadoTecnico{
		Resultado: r.Resultado, Componente: r.Componente, Etapa: r.Etapa,
	})
	if err != nil || r.Esquema != EsquemaResultadoTecnico || r.Nivel != clasificacion.Nivel ||
		r.Entorno != NormalizarEntornoIncidenciaTecnica(string(r.Entorno)) ||
		r.VersionBinario != NormalizarVersionBinario(r.VersionBinario) ||
		!EsCorrelacionTecnicaValida(r.Correlacion) || r.CorrelacionRef != "correlacion_"+r.Correlacion ||
		r.Instante.IsZero() || r.Instante.Location() != time.UTC ||
		r.Instante.Nanosecond()%1_000_000 != 0 ||
		r.Resultado != clasificacion.Resultado || r.Componente != clasificacion.Componente ||
		r.Etapa != clasificacion.Etapa {
		return ErrResultadoTecnicoInvalido
	}
	return nil
}
