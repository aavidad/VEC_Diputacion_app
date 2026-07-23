package domain

import (
	"bytes"
	"errors"
	"sort"
	"time"
)

const (
	maximoEstadosSeguimiento                = 128
	maximoMotivosSeguimiento                = 256
	maximoTransicionesSeguimiento           = 512
	maximoDocumentosPorTransicion           = 32
	maximoActuacionesSeguimiento            = 10_000
	maximoResultadosCalendarioPorTransicion = 64
)

var (
	ErrDefinicionSeguimientoInvalida = errors.New(
		"contratacion temporal: definicion de seguimiento invalida",
	)
	ErrSeguimientoInvalido = errors.New(
		"contratacion temporal: seguimiento invalido",
	)
	ErrActuacionSeguimientoEnConflicto = errors.New(
		"contratacion temporal: actuacion de seguimiento en conflicto",
	)
	ErrPublicacionDefinicionSeguimientoEnConflicto = errors.New(
		"contratacion temporal: publicacion de seguimiento en conflicto",
	)
)

// VigenciaSeguimiento representa un intervalo [Desde, Hasta). Hasta cero
// significa ausencia de fin. Todos los demás instantes son UTC canónicos.
type VigenciaSeguimiento struct {
	Desde time.Time `json:"desde"`
	Hasta time.Time `json:"hasta"`
}

func (v VigenciaSeguimiento) Validar() error {
	if !instanteSeguimientoValido(v.Desde) ||
		(!v.Hasta.IsZero() &&
			(!instanteSeguimientoValido(v.Hasta) || !v.Hasta.After(v.Desde))) {
		return ErrDefinicionSeguimientoInvalida
	}
	return nil
}

func (v VigenciaSeguimiento) contiene(instante time.Time) bool {
	return instanteSeguimientoValido(instante) &&
		!instante.Before(v.Desde) &&
		(v.Hasta.IsZero() || instante.Before(v.Hasta))
}

// ReferenciaDefinicionSeguimiento inmoviliza una publicación exacta.
type ReferenciaDefinicionSeguimiento struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

func (r ReferenciaDefinicionSeguimiento) Validar() error {
	if !referenciaValida(r.Referencia) || r.Version == 0 ||
		!huellaSeguimientoValida(r.HuellaSHA256) {
		return ErrDefinicionSeguimientoInvalida
	}
	return nil
}

func (r ReferenciaDefinicionSeguimiento) Coincide(
	otra ReferenciaDefinicionSeguimiento,
) bool {
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

// RequisitoCalendarioSeguimiento gobierna los ámbitos y resultados admitidos.
// El calendario y sus festivos se resuelven fuera del dominio.
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

func (c ClaseTransicionSeguimiento) valida() bool {
	return c == TransicionOrdinaria ||
		c == TransicionRectificacion ||
		c == TransicionReapertura
}

// EfectoPeriodoSeguimiento expresa operaciones técnicas sobre intervalos. La
// publicación decide qué transición laboral utiliza cada operación.
type EfectoPeriodoSeguimiento string

const (
	EfectoPeriodoNinguno EfectoPeriodoSeguimiento = "ninguno"
	EfectoPeriodoAbrir   EfectoPeriodoSeguimiento = "abrir"
	EfectoPeriodoAmpliar EfectoPeriodoSeguimiento = "ampliar"
	EfectoPeriodoCerrar  EfectoPeriodoSeguimiento = "cerrar"
)

func (e EfectoPeriodoSeguimiento) valido() bool {
	return e == EfectoPeriodoNinguno ||
		e == EfectoPeriodoAbrir ||
		e == EfectoPeriodoAmpliar ||
		e == EfectoPeriodoCerrar
}

// TransicionDefinidaSeguimiento contiene requisitos administrables. Las
// claves de estado, transición, motivo y documento no son listas compiladas.
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

func (t TransicionDefinidaSeguimiento) clonar() TransicionDefinidaSeguimiento {
	t.MotivosPermitidos = append([]ClaveCatalogo(nil), t.MotivosPermitidos...)
	t.Documentos = append([]RequisitoDocumentoSeguimiento(nil), t.Documentos...)
	if t.Calendario != nil {
		calendario := *t.Calendario
		calendario.AmbitosPermitidos = append(
			[]ClaveCatalogo(nil),
			t.Calendario.AmbitosPermitidos...,
		)
		calendario.ResultadosPermitidos = append(
			[]ClaveCatalogo(nil),
			t.Calendario.ResultadosPermitidos...,
		)
		t.Calendario = &calendario
	}
	return t
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

// DefinicionSeguimiento mantiene una publicación inmutable y entrega copias.
type DefinicionSeguimiento struct {
	publicacion PublicacionDefinicionSeguimiento
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
	return DefinicionSeguimiento{publicacion: publicacion}, nil
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

func (d DefinicionSeguimiento) Validar() error {
	_, err := RestaurarDefinicionSeguimiento(d.Publicacion())
	return err
}

func (d DefinicionSeguimiento) Referencia() ReferenciaDefinicionSeguimiento {
	return ReferenciaDefinicionSeguimiento{
		Referencia:   d.publicacion.Referencia,
		Version:      d.publicacion.Version,
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
	indice := sort.Search(len(d.publicacion.Transiciones), func(i int) bool {
		return d.publicacion.Transiciones[i].Clave >= clave
	})
	if indice == len(d.publicacion.Transiciones) ||
		d.publicacion.Transiciones[indice].Clave != clave {
		return TransicionDefinidaSeguimiento{}, false
	}
	return d.publicacion.Transiciones[indice].clonar(), true
}

type IntervaloSeguimiento struct {
	Desde time.Time `json:"desde"`
	Hasta time.Time `json:"hasta"`
}

func (p IntervaloSeguimiento) Validar() error {
	if !instanteSeguimientoValido(p.Desde) ||
		!instanteSeguimientoValido(p.Hasta) ||
		!p.Hasta.After(p.Desde) {
		return ErrSeguimientoInvalido
	}
	return nil
}

type DocumentoSeguimiento struct {
	TipoClave  ClaveCatalogo `json:"tipo_clave"`
	Referencia string        `json:"referencia"`
}

type EvidenciaCalendarioSeguimiento struct {
	Referencia             string        `json:"referencia"`
	Version                uint64        `json:"version"`
	HuellaSHA256           string        `json:"huella_sha256"`
	AmbitoTerritorialClave ClaveCatalogo `json:"ambito_territorial_clave"`
	ResultadoClave         ClaveCatalogo `json:"resultado_clave"`
	CalculadoEn            time.Time     `json:"calculado_en"`
}

func (e EvidenciaCalendarioSeguimiento) validar() error {
	if !referenciaValida(e.Referencia) || e.Version == 0 ||
		!huellaSeguimientoValida(e.HuellaSHA256) ||
		!e.AmbitoTerritorialClave.Valida() || !e.ResultadoClave.Valida() ||
		!instanteSeguimientoValido(e.CalculadoEn) {
		return ErrSeguimientoInvalido
	}
	return nil
}

// DatosTransicionSeguimiento no admite texto libre ni datos de identidad. Los
// actores, unidades, documentos y relaciones se conservan por referencia.
type DatosTransicionSeguimiento struct {
	ActuacionRef          string                          `json:"actuacion_ref"`
	TransicionClave       ClaveCatalogo                   `json:"transicion_clave"`
	MotivoClave           ClaveCatalogo                   `json:"motivo_clave,omitempty"`
	ActorRef              string                          `json:"actor_ref"`
	UnidadRef             string                          `json:"unidad_ref"`
	EfectivoEn            time.Time                       `json:"efectivo_en"`
	RegistradaEn          time.Time                       `json:"registrada_en"`
	Documentos            []DocumentoSeguimiento          `json:"documentos"`
	Periodo               *IntervaloSeguimiento           `json:"periodo,omitempty"`
	Calendario            *EvidenciaCalendarioSeguimiento `json:"calendario,omitempty"`
	ReciboRef             string                          `json:"recibo_ref"`
	CorrelacionRef        string                          `json:"correlacion_ref"`
	RectificaActuacionRef string                          `json:"rectifica_actuacion_ref,omitempty"`
}

func (d DatosTransicionSeguimiento) clonar() DatosTransicionSeguimiento {
	d.Documentos = append([]DocumentoSeguimiento(nil), d.Documentos...)
	if d.Periodo != nil {
		periodo := *d.Periodo
		d.Periodo = &periodo
	}
	if d.Calendario != nil {
		calendario := *d.Calendario
		d.Calendario = &calendario
	}
	return d
}

type PeriodoResultanteSeguimiento struct {
	Intervalo    IntervaloSeguimiento `json:"intervalo"`
	ActuacionRef string               `json:"actuacion_ref"`
}

type CeseEfectivoSeguimiento struct {
	EfectivoEn   time.Time `json:"efectivo_en"`
	ActuacionRef string    `json:"actuacion_ref"`
}

type ActuacionSeguimiento struct {
	Secuencia             uint64                          `json:"secuencia"`
	VersionSeguimiento    uint64                          `json:"version_seguimiento"`
	Definicion            ReferenciaDefinicionSeguimiento `json:"definicion"`
	ActuacionRef          string                          `json:"actuacion_ref"`
	TransicionClave       ClaveCatalogo                   `json:"transicion_clave"`
	Clase                 ClaseTransicionSeguimiento      `json:"clase"`
	EstadoOrigen          ClaveCatalogo                   `json:"estado_origen"`
	EstadoDestino         ClaveCatalogo                   `json:"estado_destino"`
	MotivoClave           ClaveCatalogo                   `json:"motivo_clave,omitempty"`
	ActorRef              string                          `json:"actor_ref"`
	UnidadRef             string                          `json:"unidad_ref"`
	EfectivoEn            time.Time                       `json:"efectivo_en"`
	RegistradaEn          time.Time                       `json:"registrada_en"`
	Documentos            []DocumentoSeguimiento          `json:"documentos"`
	Periodo               *IntervaloSeguimiento           `json:"periodo,omitempty"`
	Calendario            *EvidenciaCalendarioSeguimiento `json:"calendario,omitempty"`
	ReciboRef             string                          `json:"recibo_ref"`
	CorrelacionRef        string                          `json:"correlacion_ref"`
	RectificaActuacionRef string                          `json:"rectifica_actuacion_ref,omitempty"`
	HuellaPeticionSHA256  string                          `json:"huella_peticion_sha256"`
	HuellaAnteriorSHA256  string                          `json:"huella_anterior_sha256"`
	HuellaActuacionSHA256 string                          `json:"huella_actuacion_sha256"`
}

func (a ActuacionSeguimiento) clonar() ActuacionSeguimiento {
	datos := a.datos().clonar()
	a.Documentos = datos.Documentos
	a.Periodo = datos.Periodo
	a.Calendario = datos.Calendario
	return a
}

func (a ActuacionSeguimiento) datos() DatosTransicionSeguimiento {
	return DatosTransicionSeguimiento{
		ActuacionRef: a.ActuacionRef, TransicionClave: a.TransicionClave,
		MotivoClave: a.MotivoClave, ActorRef: a.ActorRef, UnidadRef: a.UnidadRef,
		EfectivoEn: a.EfectivoEn, RegistradaEn: a.RegistradaEn,
		Documentos: a.Documentos, Periodo: a.Periodo, Calendario: a.Calendario,
		ReciboRef: a.ReciboRef, CorrelacionRef: a.CorrelacionRef,
		RectificaActuacionRef: a.RectificaActuacionRef,
	}
}

type AltaSeguimiento struct {
	Referencia      string
	OrganizacionRef string
	ExpedienteRef   string
	RelacionRef     string
	PeriodoPrevisto IntervaloSeguimiento
	CreadoEn        time.Time
}

type EstadoPersistidoSeguimiento struct {
	Referencia          string                          `json:"referencia"`
	OrganizacionRef     string                          `json:"organizacion_ref"`
	ExpedienteRef       string                          `json:"expediente_ref"`
	RelacionRef         string                          `json:"relacion_ref"`
	Definicion          ReferenciaDefinicionSeguimiento `json:"definicion"`
	Version             uint64                          `json:"version"`
	EstadoActual        ClaveCatalogo                   `json:"estado_actual"`
	PeriodoPrevisto     IntervaloSeguimiento            `json:"periodo_previsto"`
	PeriodosResultantes []PeriodoResultanteSeguimiento  `json:"periodos_resultantes"`
	CeseEfectivo        *CeseEfectivoSeguimiento        `json:"cese_efectivo,omitempty"`
	CreadoEn            time.Time                       `json:"creado_en"`
	ActualizadoEn       time.Time                       `json:"actualizado_en"`
	HuellaRaizSHA256    string                          `json:"huella_raiz_sha256"`
	Actuaciones         []ActuacionSeguimiento          `json:"actuaciones"`
}

func (e EstadoPersistidoSeguimiento) clonar() EstadoPersistidoSeguimiento {
	e.PeriodosResultantes = append(
		[]PeriodoResultanteSeguimiento(nil),
		e.PeriodosResultantes...,
	)
	if e.CeseEfectivo != nil {
		cese := *e.CeseEfectivo
		e.CeseEfectivo = &cese
	}
	e.Actuaciones = clonarActuacionesSeguimiento(e.Actuaciones)
	return e
}

// Seguimiento es inmutable por valor: aplicar una transición devuelve otro
// agregado. Sus colecciones privadas nunca se comparten con quien llama.
type Seguimiento struct {
	estado EstadoPersistidoSeguimiento
}

func NuevoSeguimiento(
	definicion DefinicionSeguimiento,
	alta AltaSeguimiento,
) (Seguimiento, error) {
	if definicion.Validar() != nil || !definicion.VigenteEn(alta.CreadoEn) ||
		!referenciaValida(alta.Referencia) ||
		!referenciaValida(alta.OrganizacionRef) ||
		!referenciaValida(alta.ExpedienteRef) ||
		!referenciaValida(alta.RelacionRef) ||
		alta.PeriodoPrevisto.Validar() != nil ||
		!instanteSeguimientoValido(alta.CreadoEn) {
		return Seguimiento{}, ErrSeguimientoInvalido
	}
	estado := EstadoPersistidoSeguimiento{
		Referencia: alta.Referencia, OrganizacionRef: alta.OrganizacionRef,
		ExpedienteRef: alta.ExpedienteRef, RelacionRef: alta.RelacionRef,
		Definicion:      definicion.Referencia(),
		EstadoActual:    definicion.publicacion.EstadoInicial,
		PeriodoPrevisto: alta.PeriodoPrevisto,
		CreadoEn:        alta.CreadoEn, ActualizadoEn: alta.CreadoEn,
		PeriodosResultantes: []PeriodoResultanteSeguimiento{},
		Actuaciones:         []ActuacionSeguimiento{},
	}
	var err error
	estado.HuellaRaizSHA256, err = calcularHuellaRaizSeguimiento(estado)
	if err != nil {
		return Seguimiento{}, ErrSeguimientoInvalido
	}
	return Seguimiento{estado: estado}, nil
}

func (s Seguimiento) Version() uint64 {
	return s.estado.Version
}

func (s Seguimiento) EstadoActual() ClaveCatalogo {
	return s.estado.EstadoActual
}

func (s Seguimiento) PeriodosResultantes() []PeriodoResultanteSeguimiento {
	return append([]PeriodoResultanteSeguimiento(nil), s.estado.PeriodosResultantes...)
}

func (s Seguimiento) Actuaciones() []ActuacionSeguimiento {
	return clonarActuacionesSeguimiento(s.estado.Actuaciones)
}

func (s Seguimiento) CeseEfectivo() *CeseEfectivoSeguimiento {
	if s.estado.CeseEfectivo == nil {
		return nil
	}
	cese := *s.estado.CeseEfectivo
	return &cese
}

func (s Seguimiento) Estado() EstadoPersistidoSeguimiento {
	return s.estado.clonar()
}

func (s Seguimiento) Validar(definicion DefinicionSeguimiento) error {
	_, err := RehidratarSeguimiento(definicion, s.Estado())
	return err
}

func (s Seguimiento) Aplicar(
	definicion DefinicionSeguimiento,
	versionEsperada uint64,
	datos DatosTransicionSeguimiento,
) (Seguimiento, error) {
	validado, err := RehidratarSeguimiento(definicion, s.Estado())
	if err != nil {
		return Seguimiento{}, err
	}
	return validado.aplicarSinRehidratar(definicion, versionEsperada, datos)
}

func (s Seguimiento) aplicarSinRehidratar(
	definicion DefinicionSeguimiento,
	versionEsperada uint64,
	datos DatosTransicionSeguimiento,
) (Seguimiento, error) {
	normalizados, err := normalizarDatosTransicionSeguimiento(datos)
	if err != nil {
		return Seguimiento{}, err
	}
	huellaPeticion, err := calcularHuellaPeticionSeguimiento(normalizados)
	if err != nil {
		return Seguimiento{}, ErrSeguimientoInvalido
	}
	for _, existente := range s.estado.Actuaciones {
		if existente.ActuacionRef != normalizados.ActuacionRef {
			continue
		}
		if existente.HuellaPeticionSHA256 == huellaPeticion {
			return Seguimiento{estado: s.estado.clonar()}, nil
		}
		return Seguimiento{}, ErrActuacionSeguimientoEnConflicto
	}
	if versionEsperada != s.estado.Version {
		return Seguimiento{}, ErrVersionEnConflicto
	}
	if len(s.estado.Actuaciones) >= maximoActuacionesSeguimiento ||
		!definicion.Referencia().Coincide(s.estado.Definicion) ||
		!definicion.VigenteEn(normalizados.RegistradaEn) ||
		normalizados.RegistradaEn.Before(s.estado.ActualizadoEn) {
		return Seguimiento{}, ErrTransicionInvalida
	}
	transicion, existe := definicion.transicion(normalizados.TransicionClave)
	if !existe || transicion.Origen != s.estado.EstadoActual ||
		validarRequisitosTransicion(
			transicion,
			normalizados,
			s.estado.Actuaciones,
		) != nil {
		return Seguimiento{}, ErrTransicionInvalida
	}
	siguiente := s.estado.clonar()
	if aplicarEfectoPeriodoSeguimiento(&siguiente, transicion, normalizados) != nil {
		return Seguimiento{}, ErrTransicionInvalida
	}
	anterior := siguiente.HuellaRaizSHA256
	if len(siguiente.Actuaciones) > 0 {
		anterior = siguiente.Actuaciones[len(siguiente.Actuaciones)-1].HuellaActuacionSHA256
	}
	actuacion := ActuacionSeguimiento{
		Secuencia: siguiente.Version + 1, VersionSeguimiento: siguiente.Version + 1,
		Definicion: siguiente.Definicion, ActuacionRef: normalizados.ActuacionRef,
		TransicionClave: transicion.Clave, Clase: transicion.Clase,
		EstadoOrigen: transicion.Origen, EstadoDestino: transicion.Destino,
		MotivoClave: normalizados.MotivoClave, ActorRef: normalizados.ActorRef,
		UnidadRef: normalizados.UnidadRef, EfectivoEn: normalizados.EfectivoEn,
		RegistradaEn: normalizados.RegistradaEn, Documentos: normalizados.Documentos,
		Periodo: normalizados.Periodo, Calendario: normalizados.Calendario,
		ReciboRef: normalizados.ReciboRef, CorrelacionRef: normalizados.CorrelacionRef,
		RectificaActuacionRef: normalizados.RectificaActuacionRef,
		HuellaPeticionSHA256:  huellaPeticion, HuellaAnteriorSHA256: anterior,
	}
	actuacion.HuellaActuacionSHA256, err = calcularHuellaActuacionSeguimiento(actuacion)
	if err != nil {
		return Seguimiento{}, ErrSeguimientoInvalido
	}
	siguiente.Version++
	siguiente.EstadoActual = transicion.Destino
	siguiente.ActualizadoEn = normalizados.RegistradaEn
	siguiente.Actuaciones = append(siguiente.Actuaciones, actuacion.clonar())
	return Seguimiento{estado: siguiente}, nil
}

func normalizarDatosTransicionSeguimiento(
	d DatosTransicionSeguimiento,
) (DatosTransicionSeguimiento, error) {
	if !referenciaValida(d.ActuacionRef) || !d.TransicionClave.Valida() ||
		!referenciaValida(d.ActorRef) || !referenciaValida(d.UnidadRef) ||
		!instanteSeguimientoValido(d.EfectivoEn) ||
		!instanteSeguimientoValido(d.RegistradaEn) ||
		!referenciaValida(d.ReciboRef) || !referenciaValida(d.CorrelacionRef) ||
		len(d.Documentos) > maximoDocumentosPorTransicion ||
		(d.MotivoClave != "" && !d.MotivoClave.Valida()) ||
		(d.RectificaActuacionRef != "" &&
			!referenciaValida(d.RectificaActuacionRef)) ||
		(d.Periodo != nil && d.Periodo.Validar() != nil) ||
		(d.Calendario != nil && d.Calendario.validar() != nil) {
		return DatosTransicionSeguimiento{}, ErrTransicionInvalida
	}
	n := d.clonar()
	sort.Slice(n.Documentos, func(i, j int) bool {
		if n.Documentos[i].TipoClave == n.Documentos[j].TipoClave {
			return n.Documentos[i].Referencia < n.Documentos[j].Referencia
		}
		return n.Documentos[i].TipoClave < n.Documentos[j].TipoClave
	})
	referencias := make(map[string]struct{}, len(n.Documentos))
	for _, documento := range n.Documentos {
		if !documento.TipoClave.Valida() || !referenciaValida(documento.Referencia) {
			return DatosTransicionSeguimiento{}, ErrTransicionInvalida
		}
		if _, repetida := referencias[documento.Referencia]; repetida {
			return DatosTransicionSeguimiento{}, ErrTransicionInvalida
		}
		referencias[documento.Referencia] = struct{}{}
	}
	return n, nil
}

func validarRequisitosTransicion(
	t TransicionDefinidaSeguimiento,
	d DatosTransicionSeguimiento,
	historial []ActuacionSeguimiento,
) error {
	if t.MotivoObligatorio && d.MotivoClave == "" ||
		d.MotivoClave != "" && !contieneClaveSeguimiento(t.MotivosPermitidos, d.MotivoClave) ||
		t.RequierePeriodo != (d.Periodo != nil) ||
		(t.Calendario == nil) != (d.Calendario == nil) {
		return ErrTransicionInvalida
	}
	requisitos := make(map[ClaveCatalogo]bool, len(t.Documentos))
	for _, requisito := range t.Documentos {
		requisitos[requisito.TipoClave] = requisito.Obligatorio
	}
	presentes := make(map[ClaveCatalogo]bool, len(d.Documentos))
	for _, documento := range d.Documentos {
		if _, permitido := requisitos[documento.TipoClave]; !permitido {
			return ErrTransicionInvalida
		}
		presentes[documento.TipoClave] = true
	}
	for tipo, obligatorio := range requisitos {
		if obligatorio && !presentes[tipo] {
			return ErrTransicionInvalida
		}
	}
	if t.Calendario != nil &&
		(!contieneClaveSeguimiento(
			t.Calendario.AmbitosPermitidos,
			d.Calendario.AmbitoTerritorialClave,
		) ||
			!contieneClaveSeguimiento(
				t.Calendario.ResultadosPermitidos,
				d.Calendario.ResultadoClave,
			) ||
			d.Calendario.CalculadoEn.After(d.RegistradaEn)) {
		return ErrTransicionInvalida
	}
	if t.Clase == TransicionRectificacion {
		var rectificada *ActuacionSeguimiento
		for indice := range historial {
			if historial[indice].ActuacionRef == d.RectificaActuacionRef {
				rectificada = &historial[indice]
				break
			}
		}
		if d.RectificaActuacionRef == "" || rectificada == nil ||
			(t.ExigeActorDistinto && rectificada.ActorRef == d.ActorRef) {
			return ErrTransicionInvalida
		}
	} else if d.RectificaActuacionRef != "" {
		return ErrTransicionInvalida
	}
	return nil
}

func aplicarEfectoPeriodoSeguimiento(
	estado *EstadoPersistidoSeguimiento,
	transicion TransicionDefinidaSeguimiento,
	datos DatosTransicionSeguimiento,
) error {
	switch transicion.EfectoPeriodo {
	case EfectoPeriodoNinguno:
		return nil
	case EfectoPeriodoAbrir:
		if len(estado.PeriodosResultantes) != 0 || estado.CeseEfectivo != nil ||
			datos.Periodo == nil || *datos.Periodo != estado.PeriodoPrevisto ||
			!datos.EfectivoEn.Equal(datos.Periodo.Desde) {
			return ErrTransicionInvalida
		}
		estado.PeriodosResultantes = append(
			estado.PeriodosResultantes,
			PeriodoResultanteSeguimiento{
				Intervalo:    *datos.Periodo,
				ActuacionRef: datos.ActuacionRef,
			},
		)
	case EfectoPeriodoAmpliar:
		if len(estado.PeriodosResultantes) == 0 || estado.CeseEfectivo != nil ||
			datos.Periodo == nil {
			return ErrTransicionInvalida
		}
		ultimo := estado.PeriodosResultantes[len(estado.PeriodosResultantes)-1]
		if !datos.Periodo.Desde.Equal(ultimo.Intervalo.Hasta) ||
			!datos.Periodo.Hasta.After(ultimo.Intervalo.Hasta) ||
			!datos.EfectivoEn.Equal(datos.Periodo.Desde) {
			return ErrTransicionInvalida
		}
		estado.PeriodosResultantes = append(
			estado.PeriodosResultantes,
			PeriodoResultanteSeguimiento{
				Intervalo:    *datos.Periodo,
				ActuacionRef: datos.ActuacionRef,
			},
		)
	case EfectoPeriodoCerrar:
		if len(estado.PeriodosResultantes) == 0 || estado.CeseEfectivo != nil ||
			datos.EfectivoEn.Before(
				estado.PeriodosResultantes[0].Intervalo.Desde,
			) ||
			datos.EfectivoEn.Before(datos.RegistradaEn) {
			return ErrTransicionInvalida
		}
		estado.CeseEfectivo = &CeseEfectivoSeguimiento{
			EfectivoEn:   datos.EfectivoEn,
			ActuacionRef: datos.ActuacionRef,
		}
	default:
		return ErrTransicionInvalida
	}
	return nil
}

func RehidratarSeguimiento(
	definicion DefinicionSeguimiento,
	estado EstadoPersistidoSeguimiento,
) (Seguimiento, error) {
	if definicion.Validar() != nil ||
		!definicion.Referencia().Coincide(estado.Definicion) ||
		len(estado.Actuaciones) > maximoActuacionesSeguimiento {
		return Seguimiento{}, ErrSeguimientoInvalido
	}
	actual, err := NuevoSeguimiento(definicion, AltaSeguimiento{
		Referencia: estado.Referencia, OrganizacionRef: estado.OrganizacionRef,
		ExpedienteRef: estado.ExpedienteRef, RelacionRef: estado.RelacionRef,
		PeriodoPrevisto: estado.PeriodoPrevisto, CreadoEn: estado.CreadoEn,
	})
	if err != nil || actual.estado.HuellaRaizSHA256 != estado.HuellaRaizSHA256 {
		return Seguimiento{}, ErrSeguimientoInvalido
	}
	for indice, guardada := range estado.Actuaciones {
		if guardada.Secuencia != uint64(indice+1) ||
			guardada.VersionSeguimiento != uint64(indice+1) ||
			guardada.HuellaAnteriorSHA256 != huellaAnteriorSeguimiento(actual.estado) ||
			!guardada.Definicion.Coincide(estado.Definicion) {
			return Seguimiento{}, ErrSeguimientoInvalido
		}
		normalizados, errNormalizar := normalizarDatosTransicionSeguimiento(guardada.datos())
		if errNormalizar != nil {
			return Seguimiento{}, ErrSeguimientoInvalido
		}
		huellaPeticion, errPeticion := calcularHuellaPeticionSeguimiento(normalizados)
		huellaActuacion, errActuacion := calcularHuellaActuacionSeguimiento(guardada)
		if errPeticion != nil || errActuacion != nil ||
			guardada.HuellaPeticionSHA256 != huellaPeticion ||
			guardada.HuellaActuacionSHA256 != huellaActuacion {
			return Seguimiento{}, ErrSeguimientoInvalido
		}
		actual, err = actual.aplicarSinRehidratar(
			definicion,
			actual.Version(),
			normalizados,
		)
		if err != nil {
			return Seguimiento{}, ErrSeguimientoInvalido
		}
		generada := actual.estado.Actuaciones[len(actual.estado.Actuaciones)-1]
		if generada.HuellaActuacionSHA256 != guardada.HuellaActuacionSHA256 {
			return Seguimiento{}, ErrSeguimientoInvalido
		}
	}
	materialActual, errActual := materialCanonicoEstadoSeguimiento(actual.estado)
	materialGuardado, errGuardado := materialCanonicoEstadoSeguimiento(estado)
	if errActual != nil || errGuardado != nil ||
		!bytes.Equal(materialActual, materialGuardado) {
		return Seguimiento{}, ErrSeguimientoInvalido
	}
	return actual, nil
}
