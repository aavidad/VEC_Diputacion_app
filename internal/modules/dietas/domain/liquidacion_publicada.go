package domain

import (
	"errors"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	EsquemaPreparacionLiquidacionPublicada = "vec_dietas_preparacion_liquidacion_publicada_v1"
	EsquemaLiquidacionEconomica            = "vec_dietas_liquidacion_economica_v1"
	MaximoLineasLiquidacionPublicada       = 102
)

var (
	ErrLiquidacionPublicada            = errors.New("dietas_liquidacion_publicada_invalida")
	ErrAlojamientoLiquidacionPendiente = errors.New("dietas_alojamiento_liquidacion_pendiente")
	paisISO2Liquidacion                = regexp.MustCompile(`^[A-Z]{2}$`)
	claveIdempotenciaLiquidacion       = regexp.MustCompile(`^[A-Za-z0-9_-]{16,128}$`)
)

// NormaLiquidacionPublicada identifica la fuente y el artículo tal como
// constan en la publicación. Su forma no acredita la vigencia jurídica.
type NormaLiquidacionPublicada struct {
	Referencia string `json:"referencia"`
	Articulo   string `json:"articulo"`
	URL        string `json:"url"`
}

type ReglaLiquidacionPublicada struct {
	ReglaLiquidacionPropuesta
	NormaRef string `json:"norma_ref"`
}

// CatalogoLiquidacionPublicado es una copia inmutable de la versión que
// acredita el repositorio. La huella excluye el propio campo HuellaSHA256.
type CatalogoLiquidacionPublicado struct {
	Referencia                    string                      `json:"referencia"`
	Version                       string                      `json:"version"`
	HuellaSHA256                  string                      `json:"huella_sha256"`
	PublicacionRef                string                      `json:"publicacion_ref"`
	FechaPublicacion              string                      `json:"fecha_publicacion"`
	Normas                        []NormaLiquidacionPublicada `json:"normas"`
	VigenteDesde                  string                      `json:"vigente_desde"`
	VigenteHasta                  string                      `json:"vigente_hasta"`
	Ambito                        string                      `json:"ambito"`
	PaisISO2                      string                      `json:"pais_iso2"`
	Grupo                         int                         `json:"grupo"`
	Vehiculo                      string                      `json:"vehiculo"`
	VersionTarifaRef              string                      `json:"version_tarifa_ref"`
	RotuloClave                   string                      `json:"rotulo_clave"`
	ReglaDevengo                  ReglaDevengoPublicada       `json:"regla_devengo"`
	CatalogoOtrosGastosVersionRef string                      `json:"catalogo_otros_gastos_version_ref,omitempty"`
	Reglas                        []ReglaLiquidacionPublicada `json:"reglas"`
}

func (c CatalogoLiquidacionPublicado) HuellaCalculada() (string, error) {
	c.HuellaSHA256 = ""
	return HuellaDatosLiquidacion(c)
}

func (c CatalogoLiquidacionPublicado) ValidarForma() error {
	if !referenciaJustificante.MatchString(c.Referencia) || !referenciaJustificante.MatchString(c.Version) ||
		!referenciaJustificante.MatchString(c.PublicacionRef) || !VersionTarifaPublicadaValida(c.VersionTarifaRef) ||
		!huellaJustificante.MatchString(c.HuellaSHA256) || !fechaCivilValida(c.FechaPublicacion) ||
		!fechaCivilValida(c.VigenteDesde) || !fechaCivilValida(c.VigenteHasta) || c.VigenteHasta < c.VigenteDesde ||
		c.FechaPublicacion > c.VigenteHasta || !referenciaJustificante.MatchString(c.Ambito) ||
		!paisISO2Liquidacion.MatchString(c.PaisISO2) || c.Grupo < 1 || c.Grupo > 3 ||
		(c.Vehiculo != "propio" && c.Vehiculo != "sin_vehiculo_propio") ||
		len(c.Normas) < 2 || len(c.Normas) > 8 || len(c.Reglas) < 1 || len(c.Reglas) > 32 ||
		strings.HasPrefix(c.Referencia, "propuesta:") || strings.HasPrefix(c.Version, "propuesta:") ||
		strings.HasPrefix(c.VersionTarifaRef, "provisional:") || strings.HasPrefix(c.CatalogoOtrosGastosVersionRef, "provisional:") {
		return ErrLiquidacionPublicada
	}
	h, err := c.HuellaCalculada()
	if err != nil || h != c.HuellaSHA256 {
		return ErrLiquidacionPublicada
	}
	normas := map[string]bool{}
	for _, n := range c.Normas {
		if !referenciaJustificante.MatchString(n.Referencia) || len(n.Articulo) < 1 || len(n.Articulo) > 100 ||
			strings.TrimSpace(n.Articulo) != n.Articulo || strings.ContainsAny(n.Articulo, "\r\n\x00") ||
			len(n.URL) > 500 || !strings.HasPrefix(n.URL, "https://") || strings.ContainsAny(n.URL, "\r\n\x00") || normas[n.Referencia] {
			return ErrLiquidacionPublicada
		}
		normas[n.Referencia] = true
	}
	if c.RotuloClave != "dietas_tarifa_publicada" || c.ReglaDevengo.Validar() != nil || !normas[c.ReglaDevengo.NormaRef] {
		return ErrLiquidacionPublicada
	}
	refs, conceptos := map[string]bool{}, map[string]bool{}
	hayOtros := false
	for _, r := range c.Reglas {
		clave := r.Tipo + ":" + r.Concepto + ":" + strconv.Itoa(r.Grupo)
		if !referenciaJustificante.MatchString(r.Referencia) || strings.HasPrefix(r.Referencia, "propuesta:") ||
			!normas[r.NormaRef] || refs[r.Referencia] || conceptos[clave] || r.TopeCentimos < 0 || r.CentimosPorKM < 0 {
			return ErrLiquidacionPublicada
		}
		refs[r.Referencia], conceptos[clave] = true, true
		switch r.Tipo {
		case "dieta":
			if r.Concepto != "manutencion" || r.Grupo != c.Grupo || r.TopeCentimos < 1 || r.CentimosPorKM != 0 {
				return ErrLiquidacionPublicada
			}
		case "kilometraje":
			if c.Vehiculo != "propio" || r.Concepto != "vehiculo_propio" || r.Grupo != 0 || r.TopeCentimos != 0 || r.CentimosPorKM < 1 || r.CentimosPorKM > 10000 {
				return ErrLiquidacionPublicada
			}
		case ClaseOtroMedio, ClaseOtroGasto:
			hayOtros = true
			if !referenciaJustificante.MatchString(c.CatalogoOtrosGastosVersionRef) || !referenciaJustificante.MatchString(r.Concepto) ||
				r.Grupo != 0 || r.TopeCentimos < 1 || r.CentimosPorKM != 0 {
				return ErrLiquidacionPublicada
			}
		default:
			return ErrLiquidacionPublicada
		}
	}
	if !hayOtros && c.CatalogoOtrosGastosVersionRef != "" {
		return ErrLiquidacionPublicada
	}
	return nil
}

type InstantaneaLiquidacionPublicada struct {
	Esquema           string                       `json:"esquema"`
	ComisionRef       string                       `json:"comision_ref"`
	ComisionVersion   int64                        `json:"comision_version"`
	FechaInicio       string                       `json:"fecha_inicio"`
	HoraInicio        string                       `json:"hora_inicio"`
	FechaFin          string                       `json:"fecha_fin"`
	HoraFin           string                       `json:"hora_fin"`
	DocumentoSHA256   string                       `json:"documento_sha256"`
	CatalogoRef       string                       `json:"catalogo_ref"`
	CatalogoVersion   string                       `json:"catalogo_version"`
	CatalogoSHA256    string                       `json:"catalogo_sha256"`
	Catalogo          CatalogoLiquidacionPublicado `json:"catalogo"`
	Documento         DocumentoComision            `json:"documento"`
	Lineas            []LineaLiquidacionPropuesta  `json:"lineas"`
	Totales           TotalesLiquidacionPropuesta  `json:"totales"`
	DecisionPendiente bool                         `json:"decision_pendiente"`
	SnapshotSHA256    string                       `json:"snapshot_sha256"`
}

type PreparacionLiquidacionPublicada struct {
	instantanea InstantaneaLiquidacionPublicada
}

func (p *PreparacionLiquidacionPublicada) DecisionPendiente() bool {
	return p == nil || p.instantanea.DecisionPendiente
}

func (p *PreparacionLiquidacionPublicada) Instantanea() InstantaneaLiquidacionPublicada {
	if p == nil {
		return InstantaneaLiquidacionPublicada{}
	}
	s := p.instantanea
	s.Catalogo.Normas = append([]NormaLiquidacionPublicada(nil), s.Catalogo.Normas...)
	s.Catalogo.Reglas = append([]ReglaLiquidacionPublicada(nil), s.Catalogo.Reglas...)
	s.Documento = clonarDocumentoLiquidacion(s.Documento)
	s.Lineas = append([]LineaLiquidacionPropuesta(nil), s.Lineas...)
	return s
}

// PrepararLiquidacionPublicada acepta solo datos leídos dentro del servidor.
// SQL debe acreditar que documento y catálogo siguen publicados y vigentes.
func PrepararLiquidacionPublicada(ref string, version int64, desde, horaDesde, hasta, horaHasta, documentoSHA string, d DocumentoComision, c CatalogoLiquidacionPublicado, revisiones []RevisionLineaLiquidacion) (*PreparacionLiquidacionPublicada, error) {
	if d.AlojamientoTopeCentimos != 0 {
		return nil, ErrAlojamientoLiquidacionPendiente
	}
	if !referenciaComision.MatchString(ref) || version < 1 || !fechaCivilValida(desde) || !fechaCivilValida(hasta) || !horaCivil(horaDesde) || !horaCivil(horaHasta) || hasta < desde ||
		!huellaJustificante.MatchString(documentoSHA) || len(d.Lineas) < 1 || len(d.Lineas) > MaximoLineasLiquidacionPublicada || len(d.TramosAceptados) > 62 ||
		(revisiones != nil && len(revisiones) != len(d.Lineas)) || int(d.GrupoDieta) != c.Grupo || d.VehiculoPropio != (c.Vehiculo == "propio") ||
		c.ValidarForma() != nil || d.VersionTarifaAceptada != c.VersionTarifaRef || desde < c.VigenteDesde || hasta > c.VigenteHasta {
		return nil, ErrLiquidacionPublicada
	}
	h, err := HuellaDatosLiquidacion(d)
	if err != nil || h != documentoSHA {
		return nil, ErrLiquidacionPublicada
	}
	for _, l := range d.Lineas {
		if l.Fecha != "" && (l.Fecha < desde || l.Fecha > hasta) {
			return nil, ErrLiquidacionPublicada
		}
	}
	lineas, totales, err := CalcularBasesLiquidacionPublicada(desde, horaDesde, hasta, horaHasta, d, c)
	if err != nil {
		return nil, ErrLiquidacionPublicada
	}
	pendiente := false
	if revisiones == nil {
		revisiones = make([]RevisionLineaLiquidacion, len(lineas))
		for i, linea := range lineas {
			revisiones[i] = RevisionLineaLiquidacion{Indice: i, ReglaRef: linea.ReglaRef,
				ReconocidoPropuestoCentimos: linea.CalculadoPublicadoCentimos}
			if linea.DiferenciaCentimos != 0 {
				pendiente = true
			}
		}
	}
	indices := make(map[int]bool, len(lineas))
	for _, revision := range revisiones {
		if revision.Indice < 0 || revision.Indice >= len(lineas) || indices[revision.Indice] {
			return nil, ErrLiquidacionPublicada
		}
		indices[revision.Indice] = true
		l := &lineas[revision.Indice]
		if revision.ReglaRef != l.ReglaRef || revision.ReconocidoPropuestoCentimos < 0 ||
			revision.ReconocidoPropuestoCentimos > l.CalculadoPublicadoCentimos ||
			(revision.ReconocidoPropuestoCentimos != l.OriginalCentimos && !pendiente && !codigoMotivoLiquidacion.MatchString(revision.MotivoCodigo)) ||
			(revision.ReconocidoPropuestoCentimos == l.OriginalCentimos && revision.MotivoCodigo != "") {
			return nil, ErrLiquidacionPublicada
		}
		l.ReconocidoPropuestoCentimos = revision.ReconocidoPropuestoCentimos
		l.RechazadoCentimos = l.CalculadoPublicadoCentimos - revision.ReconocidoPropuestoCentimos
		l.MotivoCodigo = revision.MotivoCodigo
		if !sumarLiquidacion(&totales.ReconocidoPropuestoCentimos, l.ReconocidoPropuestoCentimos) ||
			!sumarLiquidacion(&totales.RechazadoCentimos, l.RechazadoCentimos) {
			return nil, ErrLiquidacionPublicada
		}
	}
	s := InstantaneaLiquidacionPublicada{Esquema: EsquemaPreparacionLiquidacionPublicada, ComisionRef: ref, ComisionVersion: version,
		FechaInicio: desde, HoraInicio: horaDesde, FechaFin: hasta, HoraFin: horaHasta, DocumentoSHA256: documentoSHA, CatalogoRef: c.Referencia, CatalogoVersion: c.Version,
		CatalogoSHA256: c.HuellaSHA256, Catalogo: c, Documento: clonarDocumentoLiquidacion(d), Lineas: lineas, Totales: totales, DecisionPendiente: pendiente}
	s.Catalogo.Normas = append([]NormaLiquidacionPublicada(nil), c.Normas...)
	s.Catalogo.Reglas = append([]ReglaLiquidacionPublicada(nil), c.Reglas...)
	s.SnapshotSHA256, err = HuellaDatosLiquidacion(s)
	if err != nil {
		return nil, ErrLiquidacionPublicada
	}
	return &PreparacionLiquidacionPublicada{instantanea: s}, nil
}

func validarOtroGastoPublicado(g OtroGastoDeclarado) error {
	if (g.Tipo != ClaseOtroMedio && g.Tipo != ClaseOtroGasto) || len(g.Concepto) < 3 || len(g.Concepto) > 500 ||
		!textoLinea(g.Concepto) || g.ImporteCentimos < 1 || g.ImporteCentimos > 100000000 ||
		!fechaCivilValida(g.Fecha) || !referenciaJustificante.MatchString(g.JustificanteRef) ||
		!huellaJustificante.MatchString(g.JustificanteSHA256) || !referenciaJustificante.MatchString(g.CatalogoVersion) ||
		!referenciaJustificante.MatchString(g.TipoGasto) {
		return ErrLiquidacionPublicada
	}
	return nil
}

func RecuperarPreparacionLiquidacionPublicada(s InstantaneaLiquidacionPublicada) (*PreparacionLiquidacionPublicada, error) {
	if s.Esquema != EsquemaPreparacionLiquidacionPublicada || len(s.Lineas) < 1 || len(s.Lineas) > MaximoLineasLiquidacionPublicada {
		return nil, ErrLiquidacionPublicada
	}
	revisiones := make([]RevisionLineaLiquidacion, len(s.Lineas))
	for i, l := range s.Lineas {
		revisiones[i] = RevisionLineaLiquidacion{Indice: l.Indice, ReglaRef: l.ReglaRef, ReconocidoPropuestoCentimos: l.ReconocidoPropuestoCentimos, MotivoCodigo: l.MotivoCodigo}
	}
	p, err := PrepararLiquidacionPublicada(s.ComisionRef, s.ComisionVersion, s.FechaInicio, s.HoraInicio, s.FechaFin, s.HoraFin, s.DocumentoSHA256, s.Documento, s.Catalogo, revisiones)
	if err != nil || !reflect.DeepEqual(p.Instantanea(), s) {
		return nil, ErrLiquidacionPublicada
	}
	return p, nil
}

// La constancia solo se obtiene del repositorio durable. No expresa firma,
// fiscalización, exportación económica ni pago.
type ConstanciaLiquidacionEconomica struct {
	Referencia        string `json:"referencia"`
	ClaveIdempotencia string `json:"clave_idempotencia"`
	NumeroDocumento   string `json:"numero_documento"`
	VersionEntrada    uint64 `json:"version_entrada"`
	VersionResultante uint64 `json:"version_resultante"`
	TitularPersonaRef string `json:"titular_persona_ref"`
	DecisorPersonaRef string `json:"decisor_persona_ref"`
	UnidadRef         string `json:"unidad_ref"`
	Fecha             string `json:"fecha"`
	ReciboRef         string `json:"recibo_ref"`
	AuditoriaRef      string `json:"auditoria_ref"`
	EventoRef         string `json:"evento_ref"`
}

type InstantaneaLiquidacionEconomica struct {
	Esquema        string                          `json:"esquema"`
	Constancia     ConstanciaLiquidacionEconomica  `json:"constancia"`
	Economia       InstantaneaLiquidacionPublicada `json:"economia"`
	SnapshotSHA256 string                          `json:"snapshot_sha256"`
}

type LiquidacionEconomica struct {
	instantanea InstantaneaLiquidacionEconomica
}

func NuevaLiquidacionEconomica(p *PreparacionLiquidacionPublicada, c ConstanciaLiquidacionEconomica) (*LiquidacionEconomica, error) {
	if p == nil || p.instantanea.DecisionPendiente || !referenciaJustificante.MatchString(c.Referencia) ||
		!claveIdempotenciaLiquidacion.MatchString(c.ClaveIdempotencia) || !referenciaJustificante.MatchString(c.ReciboRef) ||
		!referenciaJustificante.MatchString(c.AuditoriaRef) || !referenciaJustificante.MatchString(c.EventoRef) ||
		!referenciaJustificante.MatchString(c.TitularPersonaRef) || !referenciaJustificante.MatchString(c.DecisorPersonaRef) ||
		!referenciaJustificante.MatchString(c.UnidadRef) || c.TitularPersonaRef == c.DecisorPersonaRef ||
		p.instantanea.ComisionVersion < 1 || c.VersionEntrada >= math.MaxInt64 ||
		strconv.FormatUint(c.VersionEntrada, 10) != strconv.FormatInt(p.instantanea.ComisionVersion, 10) ||
		c.VersionResultante != c.VersionEntrada+1 ||
		!numeroDocumentoEconomicoPublicadoValido(c.NumeroDocumento) {
		return nil, ErrLiquidacionPublicada
	}
	fecha, err := time.Parse(time.RFC3339Nano, c.Fecha)
	if err != nil || fecha.IsZero() {
		return nil, ErrLiquidacionPublicada
	}
	s := InstantaneaLiquidacionEconomica{Esquema: EsquemaLiquidacionEconomica, Constancia: c, Economia: p.Instantanea()}
	s.SnapshotSHA256, err = HuellaDatosLiquidacion(s)
	if err != nil {
		return nil, ErrLiquidacionPublicada
	}
	return &LiquidacionEconomica{instantanea: s}, nil
}

func numeroDocumentoEconomicoPublicadoValido(s string) bool {
	return regexp.MustCompile(`^VEC-D-[0-9]{4}-[0-9]{6,18}$`).MatchString(s)
}

func (l *LiquidacionEconomica) Instantanea() InstantaneaLiquidacionEconomica {
	if l == nil {
		return InstantaneaLiquidacionEconomica{}
	}
	s := l.instantanea
	s.Economia = (&PreparacionLiquidacionPublicada{instantanea: s.Economia}).Instantanea()
	return s
}

func RecuperarLiquidacionEconomica(s InstantaneaLiquidacionEconomica) (*LiquidacionEconomica, error) {
	if s.Esquema != EsquemaLiquidacionEconomica {
		return nil, ErrLiquidacionPublicada
	}
	p, err := RecuperarPreparacionLiquidacionPublicada(s.Economia)
	if err != nil {
		return nil, ErrLiquidacionPublicada
	}
	l, err := NuevaLiquidacionEconomica(p, s.Constancia)
	if err != nil || !reflect.DeepEqual(l.Instantanea(), s) {
		return nil, ErrLiquidacionPublicada
	}
	return l, nil
}
