package domain

import (
	"errors"
	"time"
)

const (
	maximoEstadosSeguimiento                = 128
	maximoMotivosSeguimiento                = 256
	maximoTransicionesSeguimiento           = 512
	maximoDocumentosPorTransicion           = 32
	maximoResultadosCalendarioPorTransicion = 64
)

var (
	ErrDefinicionSeguimientoInvalida               = errors.New("contratacion temporal: definicion de seguimiento invalida")
	ErrPublicacionDefinicionSeguimientoEnConflicto = errors.New("contratacion temporal: publicacion de seguimiento en conflicto")
)

type VigenciaSeguimiento struct {
	Desde time.Time `json:"desde"`
	Hasta time.Time `json:"hasta"`
}

func (v VigenciaSeguimiento) Validar() error {
	if !instanteSeguimientoValido(v.Desde) || (!v.Hasta.IsZero() && (!instanteSeguimientoValido(v.Hasta) || !v.Hasta.After(v.Desde))) {
		return ErrDefinicionSeguimientoInvalida
	}
	return nil
}

func (v VigenciaSeguimiento) contiene(instante time.Time) bool {
	return instanteSeguimientoValido(instante) &&
		!instante.Before(v.Desde) &&
		(v.Hasta.IsZero() || instante.Before(v.Hasta))
}

type ReferenciaDefinicionSeguimiento struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

func (r ReferenciaDefinicionSeguimiento) Validar() error {
	if !referenciaOpacaSeguimientoValida(r.Referencia) || r.Version == 0 || !huellaSeguimientoValida(r.HuellaSHA256) {
		return ErrDefinicionSeguimientoInvalida
	}
	return nil
}

func (r ReferenciaDefinicionSeguimiento) Coincide(otra ReferenciaDefinicionSeguimiento) bool {
	return r.Validar() == nil && otra.Validar() == nil && r == otra
}

type EstadoDefinidoSeguimiento struct {
	Clave ClaveCatalogo `json:"clave"`
	Final bool          `json:"final"`
}

type RequisitoDocumentoSeguimiento struct {
	TipoClave   ClaveCatalogo `json:"tipo_clave"`
	Obligatorio bool          `json:"obligatorio"`
}

type RequisitoCalendarioSeguimiento struct {
	AmbitosPermitidos    []ClaveCatalogo `json:"ambitos_permitidos"`
	ResultadosPermitidos []ClaveCatalogo `json:"resultados_permitidos"`
}

type ClaseTransicionSeguimiento string

const (
	TransicionOrdinaria     ClaseTransicionSeguimiento = "ordinaria"
	TransicionRectificacion ClaseTransicionSeguimiento = "rectificacion"
	TransicionReapertura    ClaseTransicionSeguimiento = "reapertura"
)

type EfectoPeriodoSeguimiento string

const (
	EfectoPeriodoNinguno         EfectoPeriodoSeguimiento = "ninguno"
	EfectoPeriodoAbrir           EfectoPeriodoSeguimiento = "abrir"
	EfectoPeriodoAmpliar         EfectoPeriodoSeguimiento = "ampliar"
	EfectoPeriodoCerrar          EfectoPeriodoSeguimiento = "cerrar"
	EfectoPeriodoRectificarTramo EfectoPeriodoSeguimiento = "rectificar_tramo"
	EfectoPeriodoRectificarCese  EfectoPeriodoSeguimiento = "rectificar_cese"
	EfectoPeriodoReabrir         EfectoPeriodoSeguimiento = "reabrir"
)

func (e EfectoPeriodoSeguimiento) valido() bool {
	return e == EfectoPeriodoNinguno || e == EfectoPeriodoAbrir || e == EfectoPeriodoAmpliar || e == EfectoPeriodoCerrar || e == EfectoPeriodoRectificarTramo || e == EfectoPeriodoRectificarCese || e == EfectoPeriodoReabrir
}

type TransicionDefinidaSeguimiento struct {
	Clave              ClaveCatalogo                   `json:"clave"`
	Origen             ClaveCatalogo                   `json:"origen"`
	Destino            ClaveCatalogo                   `json:"destino"`
	Clase              ClaseTransicionSeguimiento      `json:"clase"`
	MotivosPermitidos  []ClaveCatalogo                 `json:"motivos_permitidos"`
	MotivoObligatorio  bool                            `json:"motivo_obligatorio"`
	Documentos         []RequisitoDocumentoSeguimiento `json:"documentos"`
	Calendario         *RequisitoCalendarioSeguimiento `json:"calendario,omitempty"`
	RequierePeriodo    bool                            `json:"requiere_periodo"`
	EfectoPeriodo      EfectoPeriodoSeguimiento        `json:"efecto_periodo"`
	ExigeActorDistinto bool                            `json:"exige_actor_distinto"`
}

type BorradorDefinicionSeguimiento struct {
	Referencia               string                          `json:"referencia"`
	Version                  uint64                          `json:"version"`
	PublicadoEn              time.Time                       `json:"publicado_en"`
	Vigencia                 VigenciaSeguimiento             `json:"vigencia"`
	EstadoInicial            ClaveCatalogo                   `json:"estado_inicial"`
	ProhibeCiclosSilenciosos bool                            `json:"prohibe_ciclos_silenciosos"`
	Estados                  []EstadoDefinidoSeguimiento     `json:"estados"`
	Motivos                  []ClaveCatalogo                 `json:"motivos"`
	Transiciones             []TransicionDefinidaSeguimiento `json:"transiciones"`
}

type PublicacionDefinicionSeguimiento struct {
	Referencia               string                          `json:"referencia"`
	Version                  uint64                          `json:"version"`
	HuellaSHA256             string                          `json:"huella_sha256"`
	Canon                    CanonSeguimiento                `json:"canon"`
	PublicadoEn              time.Time                       `json:"publicado_en"`
	Vigencia                 VigenciaSeguimiento             `json:"vigencia"`
	EstadoInicial            ClaveCatalogo                   `json:"estado_inicial"`
	ProhibeCiclosSilenciosos bool                            `json:"prohibe_ciclos_silenciosos"`
	Estados                  []EstadoDefinidoSeguimiento     `json:"estados"`
	Motivos                  []ClaveCatalogo                 `json:"motivos"`
	Transiciones             []TransicionDefinidaSeguimiento `json:"transiciones"`
}

type DefinicionSeguimiento struct {
	publicacion  PublicacionDefinicionSeguimiento
	transiciones map[ClaveCatalogo]TransicionDefinidaSeguimiento
}

func PublicarDefinicionSeguimiento(
	borrador BorradorDefinicionSeguimiento,
) (DefinicionSeguimiento, error) {
	normalizado, err := normalizarDefinicionSeguimiento(borrador)
	if err != nil {
		return DefinicionSeguimiento{}, err
	}
	publicacion := PublicacionDefinicionSeguimiento{
		Referencia: normalizado.Referencia, Version: normalizado.Version,
		Canon: CanonDefinicionSeguimientoV1(), PublicadoEn: normalizado.PublicadoEn,
		Vigencia: normalizado.Vigencia, EstadoInicial: normalizado.EstadoInicial,
		ProhibeCiclosSilenciosos: normalizado.ProhibeCiclosSilenciosos,
		Estados:                  normalizado.Estados, Motivos: normalizado.Motivos,
		Transiciones: normalizado.Transiciones,
	}
	publicacion.HuellaSHA256, err = calcularHuellaDefinicionSeguimiento(publicacion)
	if err != nil {
		return DefinicionSeguimiento{}, ErrDefinicionSeguimientoInvalida
	}
	indice := make(map[ClaveCatalogo]TransicionDefinidaSeguimiento, len(publicacion.Transiciones))
	for _, transicion := range publicacion.Transiciones {
		indice[transicion.Clave] = transicion
	}
	return DefinicionSeguimiento{publicacion: publicacion, transiciones: indice}, nil
}

func RestaurarDefinicionSeguimiento(
	publicacion PublicacionDefinicionSeguimiento,
) (DefinicionSeguimiento, error) {
	if !publicacion.Canon.EsDefinicionV1() ||
		!huellaSeguimientoValida(publicacion.HuellaSHA256) {
		return DefinicionSeguimiento{}, ErrDefinicionSeguimientoInvalida
	}
	restaurada, err := PublicarDefinicionSeguimiento(BorradorDefinicionSeguimiento{
		Referencia: publicacion.Referencia, Version: publicacion.Version,
		PublicadoEn: publicacion.PublicadoEn, Vigencia: publicacion.Vigencia,
		EstadoInicial:            publicacion.EstadoInicial,
		ProhibeCiclosSilenciosos: publicacion.ProhibeCiclosSilenciosos,
		Estados:                  publicacion.Estados, Motivos: publicacion.Motivos,
		Transiciones: publicacion.Transiciones,
	})
	if err != nil || restaurada.publicacion.HuellaSHA256 != publicacion.HuellaSHA256 {
		return DefinicionSeguimiento{}, ErrDefinicionSeguimientoInvalida
	}
	return restaurada, nil
}

func (d DefinicionSeguimiento) Validar() (err error) {
	_, err = RestaurarDefinicionSeguimiento(d.Publicacion())
	return
}

func (d DefinicionSeguimiento) Referencia() ReferenciaDefinicionSeguimiento {
	return ReferenciaDefinicionSeguimiento{
		Referencia: d.publicacion.Referencia, Version: d.publicacion.Version,
		HuellaSHA256: d.publicacion.HuellaSHA256,
	}
}

func (d DefinicionSeguimiento) VigenteEn(instante time.Time) bool {
	return d.Validar() == nil && d.publicacion.Vigencia.contiene(instante)
}

func (d DefinicionSeguimiento) Publicacion() PublicacionDefinicionSeguimiento {
	p := d.publicacion
	p.Estados = append([]EstadoDefinidoSeguimiento(nil), p.Estados...)
	p.Motivos = append([]ClaveCatalogo(nil), p.Motivos...)
	p.Transiciones = clonarTransicionesSeguimiento(p.Transiciones)
	return p
}

func (d DefinicionSeguimiento) transicion(
	clave ClaveCatalogo,
) (TransicionDefinidaSeguimiento, bool) {
	transicion, existe := d.transiciones[clave]
	return transicion, existe
}
