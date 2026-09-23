package domain

import (
	"errors"
	"time"
)

var ErrOrdenVigenteInvalido = errors.New("bolsa: orden vigente invalido")

const (
	CriterioPuntuacionDescActa  = "puntuacion_desc_acta"
	TipoListaCerrada            = "cerrada"
	TipoListaRotatoria          = "rotatoria"
	ReposicionMismaPosicion     = "misma_posicion"
	ReposicionFinLista          = "fin_lista"
	ReposicionNoDisponibleHasta = "no_disponible_hasta_fecha"
)

// PoliticaOrdenBolsa conserva la regla que explica el orden derivado. La marca
// provisional impide presentar la configuracion inicial como norma aprobada.
type PoliticaOrdenBolsa struct {
	PoliticaRef, BolsaRef, Criterio, TipoLista, Reposicion, Rotulo, Actor string
	Version                                                               uint64
	VigenteDesde                                                          time.Time
	Provisional                                                           bool
}

func (p PoliticaOrdenBolsa) Validar() error {
	if p.PoliticaRef == "" || p.BolsaRef == "" || p.Version == 0 || p.VigenteDesde.IsZero() ||
		p.Criterio != CriterioPuntuacionDescActa || p.Rotulo == "" || p.Actor == "" ||
		(p.TipoLista != TipoListaCerrada && p.TipoLista != TipoListaRotatoria) ||
		(p.Reposicion != ReposicionMismaPosicion && p.Reposicion != ReposicionFinLista && p.Reposicion != ReposicionNoDisponibleHasta) {
		return ErrOrdenVigenteInvalido
	}
	return nil
}

type PosicionOrdenBolsa struct {
	ParticipacionRef string
	OrdenActa        uint64
	OrdenVigente     *uint64
	Situacion        string
	Razon            string
}

type OrdenVigenteBolsa struct {
	Politica   PoliticaOrdenBolsa
	Posiciones []PosicionOrdenBolsa
}

func (o OrdenVigenteBolsa) Validar() error {
	if o.Politica.Validar() != nil || len(o.Posiciones) == 0 {
		return ErrOrdenVigenteInvalido
	}
	vistos := make(map[string]struct{}, len(o.Posiciones))
	for _, posicion := range o.Posiciones {
		if posicion.ParticipacionRef == "" || posicion.OrdenActa == 0 || posicion.Situacion == "" || posicion.Razon == "" {
			return ErrOrdenVigenteInvalido
		}
		if _, existe := vistos[posicion.ParticipacionRef]; existe {
			return ErrOrdenVigenteInvalido
		}
		vistos[posicion.ParticipacionRef] = struct{}{}
		if posicion.OrdenVigente != nil && *posicion.OrdenVigente == 0 {
			return ErrOrdenVigenteInvalido
		}
	}
	return nil
}
