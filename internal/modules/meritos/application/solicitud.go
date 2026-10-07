package application

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"time"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrSolicitud             = errors.New("meritos.error.solicitud_invalida")
	ErrAcreditacionPendiente = errors.New("meritos.error.acreditacion_pendiente")
)

// Códigos candidatos WIP: no constituyen perfiles, publicación ni autorización.
const (
	accionDeclarar   = "meritos.hecho.declarar"
	accionVerificar  = "meritos.hecho.verificar"
	accionRechazar   = "meritos.hecho.rechazar"
	accionRectificar = "meritos.hecho.rectificar"
)

type Solicitud struct {
	Vinculo           vec.VinculoAutenticacionActorV2
	Contexto          vec.ResultadoContextoActorRegistradoV2
	Correlacion       vec.ReferenciaCorrelacionAutorizacionV2
	Motivo            vec.ReferenciaEntradaCatalogo
	ClaveIdempotencia string
	VersionEsperada   int
	FechaCorte        string
	Hecho             domain.Hecho
}

func (s Solicitud) validarContexto() error {
	if s.Contexto.Validar() != nil || s.Vinculo.ValidarPara(s.Contexto) != nil || s.Correlacion.Validar() != nil {
		return ErrSolicitud
	}
	return nil
}

func (s Solicitud) validarComando() error {
	_, err := time.Parse("2006-01-02", s.FechaCorte)
	if !vec.ReferenciaMotivoAutorizacionV2Valida(s.Motivo) ||
		!domain.ReferenciaValida(s.ClaveIdempotencia) || s.VersionEsperada < 0 ||
		s.VersionEsperada >= 1<<30 || s.Hecho.Validar() != nil || s.Hecho.Version != s.VersionEsperada+1 ||
		s.Hecho.Estado != domain.Declarado || s.Hecho.Revision != nil || err != nil || len(s.FechaCorte) != 10 {
		return ErrSolicitud
	}
	return nil
}

// La huella no incluye correlación ni decisiones efímeras: un reintento conserva
// su significado. Incluye actor, motivo versionado, corte, fuente y evidencias.
func ordenSolicitud(s Solicitud, accion string) (ports.OrdenOperacion, error) {
	h := copiarHecho(s.Hecho)
	sort.Slice(h.Evidencias, func(i, j int) bool {
		if h.Evidencias[i].ID == h.Evidencias[j].ID {
			return h.Evidencias[i].Version < h.Evidencias[j].Version
		}
		return h.Evidencias[i].ID < h.Evidencias[j].ID
	})
	o := ports.OrdenOperacion{Accion: accion, ActorRef: s.Contexto.Contexto.PersonaRef, ClaveIdempotencia: s.ClaveIdempotencia, VersionEsperada: s.VersionEsperada, Hecho: h, Motivo: s.Motivo, FechaCorte: s.FechaCorte}
	preimagen := domain.ComandoHecho{Esquema: domain.EsquemaComandoHecho, Accion: accion, ActorRef: o.ActorRef,
		ClaveIdempotencia: o.ClaveIdempotencia, VersionEsperada: o.VersionEsperada, Hecho: h, Motivo: s.Motivo, FechaCorte: s.FechaCorte}
	b, err := preimagen.RepresentacionCanonica()
	if err != nil {
		return ports.OrdenOperacion{}, ErrSolicitud
	}
	suma := sha256.Sum256(b)
	o.HuellaComando = hex.EncodeToString(suma[:])
	return o, nil
}

func copiarHecho(h domain.Hecho) domain.Hecho {
	h.Evidencias = append([]vec.ReferenciaDocumento{}, h.Evidencias...)
	if h.Horas != nil {
		valor := *h.Horas
		h.Horas = &valor
	}
	if h.Revision != nil {
		valor := *h.Revision
		h.Revision = &valor
	}
	return h
}
