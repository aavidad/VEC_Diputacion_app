package ports

import (
	"context"
	"errors"
	"math"
	"strings"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrConsultaFuenteAutoridadInvalida = errors.New("vec: consulta de fuente de autoridad invalida")
	ErrFuenteAutoridadNoEncontrada     = errors.New("vec: fuente de autoridad no encontrada")
)

const MaximoVersionesPaginaFuenteAutoridad uint16 = 100

// SelectorVersionFuenteAutoridad identifica una fila exacta para tareas de
// gobierno. No equivale a una referencia consumible: esta ultima incorpora
// tambien la huella del contenido.
type SelectorVersionFuenteAutoridad struct {
	FuenteID string
	Version  uint64
}

type ConsultaPaginaHistoriaFuenteAutoridad struct {
	FuenteID     string
	DesdeVersion uint64
	Limite       uint16
}

func (s ConsultaPaginaHistoriaFuenteAutoridad) Validar() error {
	if !claveFuenteAutoridadPuertoValida(s.FuenteID) || s.DesdeVersion == 0 ||
		s.Limite == 0 || s.Limite > MaximoVersionesPaginaFuenteAutoridad {
		return ErrConsultaFuenteAutoridadInvalida
	}
	return nil
}

// PaginaHistoriaFuenteAutoridad conserva continuidad explicita. Si HayMas es
// cierto, SiguienteVersion es la primera version de la pagina siguiente; no
// existe cursor implicito ni selector de «ultima» version.
type PaginaHistoriaFuenteAutoridad struct {
	Versiones        []domain.FuenteAutoridadVersionada
	HayMas           bool
	SiguienteVersion uint64
}

func (p PaginaHistoriaFuenteAutoridad) ValidarPara(
	consulta ConsultaPaginaHistoriaFuenteAutoridad,
) error {
	if consulta.Validar() != nil || len(p.Versiones) > int(consulta.Limite) {
		return ErrConsultaFuenteAutoridadInvalida
	}
	if len(p.Versiones) == 0 {
		if p.HayMas || p.SiguienteVersion != 0 {
			return ErrConsultaFuenteAutoridadInvalida
		}
		return nil
	}
	versionEsperada := consulta.DesdeVersion
	for indice, fuente := range p.Versiones {
		if fuente.Validar() != nil || fuente.ID != consulta.FuenteID || fuente.Version != versionEsperada {
			return ErrConsultaFuenteAutoridadInvalida
		}
		if versionEsperada == math.MaxUint64 {
			if p.HayMas || indice != len(p.Versiones)-1 {
				return ErrConsultaFuenteAutoridadInvalida
			}
			break
		}
		versionEsperada++
	}
	if p.HayMas {
		if len(p.Versiones) != int(consulta.Limite) || p.SiguienteVersion != versionEsperada {
			return ErrConsultaFuenteAutoridadInvalida
		}
	} else if p.SiguienteVersion != 0 {
		return ErrConsultaFuenteAutoridadInvalida
	}
	return nil
}

func (p PaginaHistoriaFuenteAutoridad) ClonarPara(
	consulta ConsultaPaginaHistoriaFuenteAutoridad,
) (PaginaHistoriaFuenteAutoridad, error) {
	if p.ValidarPara(consulta) != nil {
		return PaginaHistoriaFuenteAutoridad{}, ErrConsultaFuenteAutoridadInvalida
	}
	clon := p
	clon.Versiones = make([]domain.FuenteAutoridadVersionada, len(p.Versiones))
	for indice, fuente := range p.Versiones {
		canonica, err := fuente.ClonarCanonica()
		if err != nil {
			return PaginaHistoriaFuenteAutoridad{}, ErrConsultaFuenteAutoridadInvalida
		}
		clon.Versiones[indice] = canonica
	}
	return clon, nil
}

func (s SelectorVersionFuenteAutoridad) Validar() error {
	if !claveFuenteAutoridadPuertoValida(s.FuenteID) || s.Version == 0 {
		return ErrConsultaFuenteAutoridadInvalida
	}
	return nil
}

// ConsultaVersionFuenteAutoridad nunca elige la version mas reciente.
// La implementacion debe devolver una copia canonica del agregado.
type ConsultaVersionFuenteAutoridad interface {
	ObtenerVersion(context.Context, SelectorVersionFuenteAutoridad) (domain.FuenteAutoridadVersionada, error)
}

// ConsultaReferenciaFuenteAutoridad resuelve una referencia ya ligada a la
// huella exacta. Una discrepancia de huella se trata como no encontrada.
type ConsultaReferenciaFuenteAutoridad interface {
	ObtenerPorReferencia(context.Context, domain.ReferenciaFuenteAutoridad) (domain.FuenteAutoridadVersionada, error)
}

// ConsultaCitaFuenteAutoridad resuelve solo citas exactas; el puerto no
// completa una lista vacia de preceptos ni sustituye su fuente.
type ConsultaCitaFuenteAutoridad interface {
	ObtenerPorCita(context.Context, domain.CitaFuenteAutoridad) (domain.FuenteAutoridadVersionada, error)
}

// ConsultaHistoriaFuentesAutoridad pagina las versiones por orden ascendente.
// Una pagina vacia es distinta de una consulta invalida.
type ConsultaHistoriaFuentesAutoridad interface {
	ListarVersiones(
		context.Context,
		ConsultaPaginaHistoriaFuenteAutoridad,
	) (PaginaHistoriaFuenteAutoridad, error)
}

func claveFuenteAutoridadPuertoValida(valor string) bool {
	if len(valor) == 0 || len(valor) > 128 || valor != strings.TrimSpace(valor) ||
		valor[0] < 'a' || valor[0] > 'z' {
		return false
	}
	for indice := 1; indice < len(valor); indice++ {
		caracter := valor[indice]
		if (caracter < 'a' || caracter > 'z') && (caracter < '0' || caracter > '9') &&
			caracter != '.' && caracter != '_' && caracter != '-' {
			return false
		}
	}
	return true
}
