package domain

import (
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	ErrBolsaConstituidaInvalida      = errors.New("bolsa: bolsa constituida invalida")
	ErrSituacionBolsaInvalida        = errors.New("bolsa: situacion de participacion invalida")
	ErrParticipacionBolsaInvalida    = errors.New("bolsa: participacion invalida")
	ErrNecesidadCoberturaInvalida    = errors.New("bolsa: necesidad de cobertura invalida")
	ErrPoliticaLlamamientoInvalida   = errors.New("bolsa: politica de llamamiento invalida")
	ErrInstantaneaOrdenBolsaInvalida = errors.New("bolsa: instantanea de orden invalida")
	ErrEvaluacionLlamamientoInvalida = errors.New("bolsa: evaluacion de llamamiento invalida")
	ErrPropuestaLlamamientoInvalida  = errors.New("bolsa: propuesta de llamamiento invalida")
	ErrSinParticipacionElegible      = errors.New("bolsa: no existe participacion elegible evaluada")
)

const (
	maximoSituacionesParticipacion = 4096
	maximoEntradasOrdenBolsa       = 250_000
	maximoRequisitosCobertura      = 512
	maximoMotivosEvaluacion        = 512
	maximoPuestosNecesidad         = 100_000
)

var (
	// El dominio no puede demostrar por si solo que una referencia sea anonima,
	// pero si puede impedir formatos evidentes de documentos de identidad y
	// sus etiquetas.
	// La fuente de identidad debe entregar siempre referencias internas opacas.
	patronDocumentoIdentidadEnReferencia = regexp.MustCompile(`(?i)(([0-9][._:/#-]?){8}|[XYZ][._:/#-]?([0-9][._:/#-]?){7})[A-Z]`)
	patronEtiquetaDocumentoIdentidad     = regexp.MustCompile(`(?i)(^|[._:/#-])(dni|nie|nif|pasaporte|passport)([._:/#-]|$)`)
)

// BolsaConstituida fija la version juridicamente eficaz desde la que puede
// operar una lista. El listado y la resolucion se enlazan por referencia y
// huella; este agregado no acepta una coleccion de candidatos reenviada por un
// adaptador de entrada.
type BolsaConstituida struct {
	BolsaRef                  string     `json:"bolsa_ref"`
	Version                   uint64     `json:"version"`
	ProcesoRef                string     `json:"proceso_ref"`
	CategoriaRef              string     `json:"categoria_ref"`
	ListadoDefinitivoRef      string     `json:"listado_definitivo_ref"`
	VersionListado            uint64     `json:"version_listado"`
	HuellaListadoSHA256       string     `json:"huella_listado_sha256"`
	ResolucionConstitucionRef string     `json:"resolucion_constitucion_ref"`
	HuellaResolucionSHA256    string     `json:"huella_resolucion_sha256"`
	ConstituidaEn             time.Time  `json:"constituida_en"`
	VigenteDesde              time.Time  `json:"vigente_desde"`
	VigenteHasta              *time.Time `json:"vigente_hasta,omitempty"`
}

type AltaBolsaConstituida = BolsaConstituida

func NuevaBolsaConstituida(alta AltaBolsaConstituida) (BolsaConstituida, error) {
	bolsa := BolsaConstituida(alta)
	var err error
	bolsa.ConstituidaEn, err = normalizarInstanteLlamamiento(bolsa.ConstituidaEn)
	if err != nil {
		return BolsaConstituida{}, ErrBolsaConstituidaInvalida
	}
	bolsa.VigenteDesde, err = normalizarInstanteLlamamiento(bolsa.VigenteDesde)
	if err != nil {
		return BolsaConstituida{}, ErrBolsaConstituidaInvalida
	}
	bolsa.VigenteHasta, err = normalizarInstanteOpcionalLlamamiento(bolsa.VigenteHasta)
	if err != nil {
		return BolsaConstituida{}, ErrBolsaConstituidaInvalida
	}
	if err := bolsa.Validar(); err != nil {
		return BolsaConstituida{}, err
	}
	return bolsa, nil
}

func (b BolsaConstituida) Validar() error {
	if !referenciaLlamamientoOpacaValida(b.BolsaRef) || b.Version == 0 ||
		!referenciaLlamamientoOpacaValida(b.ProcesoRef) ||
		!referenciaLlamamientoOpacaValida(b.CategoriaRef) ||
		!referenciaLlamamientoOpacaValida(b.ListadoDefinitivoRef) || b.VersionListado == 0 ||
		!huellaSHA256Valida(b.HuellaListadoSHA256) ||
		!referenciaLlamamientoOpacaValida(b.ResolucionConstitucionRef) ||
		!huellaSHA256Valida(b.HuellaResolucionSHA256) ||
		!instanteLlamamientoCanonico(b.ConstituidaEn) || !instanteLlamamientoCanonico(b.VigenteDesde) ||
		b.VigenteDesde.Before(b.ConstituidaEn) ||
		!intervaloAbiertoLlamamientoValido(b.VigenteDesde, b.VigenteHasta) {
		return ErrBolsaConstituidaInvalida
	}
	return nil
}

func (b BolsaConstituida) VigenteEn(instante time.Time) bool {
	canonico, err := normalizarInstanteLlamamiento(instante)
	if err != nil || b.Validar() != nil || canonico.Before(b.VigenteDesde) {
		return false
	}
	return b.VigenteHasta == nil || canonico.Before(*b.VigenteHasta)
}

func (b BolsaConstituida) ClonarCanonica() (BolsaConstituida, error) {
	if err := b.Validar(); err != nil {
		return BolsaConstituida{}, err
	}
	clon := b
	clon.VigenteHasta = clonarInstanteOpcional(b.VigenteHasta)
	return clon, nil
}

func (b BolsaConstituida) HuellaCanonicaSHA256() (string, error) {
	clon, err := b.ClonarCanonica()
	if err != nil {
		return "", err
	}
	return huellaJSON(clon)
}

// SituacionParticipacionBolsa representa un periodo semicerrado [Desde,
// Hasta). Estado y causa son entradas gobernadas y versionadas; el nucleo no
// incorpora una lista estatica de situaciones ni reglas temporales concretas.
type SituacionParticipacionBolsa struct {
	Secuencia            uint64     `json:"secuencia"`
	EstadoClave          string     `json:"estado_clave"`
	EstadoVersion        uint64     `json:"estado_version"`
	HuellaEstadoSHA256   string     `json:"huella_estado_sha256"`
	CausaClave           string     `json:"causa_clave"`
	CausaVersion         uint64     `json:"causa_version"`
	HuellaCausaSHA256    string     `json:"huella_causa_sha256"`
	DecisionRef          string     `json:"decision_ref"`
	HuellaDecisionSHA256 string     `json:"huella_decision_sha256"`
	Desde                time.Time  `json:"desde"`
	Hasta                *time.Time `json:"hasta,omitempty"`
}

func (s SituacionParticipacionBolsa) Validar() error {
	if s.Secuencia == 0 || !claveLlamamientoValida(s.EstadoClave) || s.EstadoVersion == 0 ||
		!huellaSHA256Valida(s.HuellaEstadoSHA256) || !claveLlamamientoValida(s.CausaClave) ||
		s.CausaVersion == 0 || !huellaSHA256Valida(s.HuellaCausaSHA256) ||
		!referenciaLlamamientoOpacaValida(s.DecisionRef) || !huellaSHA256Valida(s.HuellaDecisionSHA256) ||
		!instanteLlamamientoCanonico(s.Desde) || !intervaloAbiertoLlamamientoValido(s.Desde, s.Hasta) {
		return ErrSituacionBolsaInvalida
	}
	return nil
}

func (s SituacionParticipacionBolsa) clonarCanonica() (SituacionParticipacionBolsa, error) {
	if err := s.Validar(); err != nil {
		return SituacionParticipacionBolsa{}, err
	}
	clon := s
	clon.Hasta = clonarInstanteOpcional(s.Hasta)
	return clon, nil
}

// ParticipacionBolsa mantiene una historia completa y sin solapes. La
// posicion no vive aqui: pertenece a cada InstantaneaOrdenBolsa versionada.
type ParticipacionBolsa struct {
	ParticipacionRef string                        `json:"participacion_ref"`
	BolsaRef         string                        `json:"bolsa_ref"`
	SujetoRef        string                        `json:"sujeto_ref"`
	Version          uint64                        `json:"version"`
	AltaEn           time.Time                     `json:"alta_en"`
	Situaciones      []SituacionParticipacionBolsa `json:"situaciones"`
}

type AltaParticipacionBolsa = ParticipacionBolsa

func NuevaParticipacionBolsa(alta AltaParticipacionBolsa) (ParticipacionBolsa, error) {
	if len(alta.Situaciones) == 0 || len(alta.Situaciones) > maximoSituacionesParticipacion {
		return ParticipacionBolsa{}, ErrParticipacionBolsaInvalida
	}
	participacion := ParticipacionBolsa(alta)
	var err error
	participacion.AltaEn, err = normalizarInstanteLlamamiento(participacion.AltaEn)
	if err != nil {
		return ParticipacionBolsa{}, ErrParticipacionBolsaInvalida
	}
	participacion.Situaciones = make([]SituacionParticipacionBolsa, len(alta.Situaciones))
	for indice := range alta.Situaciones {
		situacion := alta.Situaciones[indice]
		situacion.Desde, err = normalizarInstanteLlamamiento(situacion.Desde)
		if err != nil {
			return ParticipacionBolsa{}, ErrParticipacionBolsaInvalida
		}
		situacion.Hasta, err = normalizarInstanteOpcionalLlamamiento(situacion.Hasta)
		if err != nil {
			return ParticipacionBolsa{}, ErrParticipacionBolsaInvalida
		}
		participacion.Situaciones[indice] = situacion
	}
	sort.Slice(participacion.Situaciones, func(i, j int) bool {
		return participacion.Situaciones[i].Secuencia < participacion.Situaciones[j].Secuencia
	})
	if err := participacion.Validar(); err != nil {
		return ParticipacionBolsa{}, err
	}
	return participacion, nil
}

func (p ParticipacionBolsa) Validar() error {
	if !referenciaLlamamientoOpacaValida(p.ParticipacionRef) ||
		!referenciaLlamamientoOpacaValida(p.BolsaRef) ||
		!referenciaLlamamientoOpacaValida(p.SujetoRef) || p.Version == 0 ||
		!instanteLlamamientoCanonico(p.AltaEn) || len(p.Situaciones) == 0 ||
		len(p.Situaciones) > maximoSituacionesParticipacion {
		return ErrParticipacionBolsaInvalida
	}
	decisiones := make(map[string]struct{}, len(p.Situaciones))
	for indice := range p.Situaciones {
		actual := p.Situaciones[indice]
		if actual.Validar() != nil || actual.Secuencia != uint64(indice+1) {
			return ErrParticipacionBolsaInvalida
		}
		if _, existe := decisiones[actual.DecisionRef]; existe {
			return ErrParticipacionBolsaInvalida
		}
		decisiones[actual.DecisionRef] = struct{}{}
		if indice == 0 {
			if !actual.Desde.Equal(p.AltaEn) {
				return ErrParticipacionBolsaInvalida
			}
			continue
		}
		anterior := p.Situaciones[indice-1]
		if anterior.Hasta == nil || !anterior.Hasta.Equal(actual.Desde) {
			return ErrParticipacionBolsaInvalida
		}
	}
	if p.Situaciones[len(p.Situaciones)-1].Hasta != nil {
		return ErrParticipacionBolsaInvalida
	}
	return nil
}

func (p ParticipacionBolsa) SituacionVigenteEn(instante time.Time) (SituacionParticipacionBolsa, bool) {
	canonico, err := normalizarInstanteLlamamiento(instante)
	if err != nil || p.Validar() != nil || canonico.Before(p.AltaEn) {
		return SituacionParticipacionBolsa{}, false
	}
	indice := sort.Search(len(p.Situaciones), func(i int) bool {
		return p.Situaciones[i].Hasta == nil || canonico.Before(*p.Situaciones[i].Hasta)
	})
	if indice >= len(p.Situaciones) || canonico.Before(p.Situaciones[indice].Desde) {
		return SituacionParticipacionBolsa{}, false
	}
	clon, err := p.Situaciones[indice].clonarCanonica()
	return clon, err == nil
}

func (p ParticipacionBolsa) ClonarCanonica() (ParticipacionBolsa, error) {
	if err := p.Validar(); err != nil {
		return ParticipacionBolsa{}, err
	}
	clon := p
	clon.Situaciones = make([]SituacionParticipacionBolsa, len(p.Situaciones))
	for indice := range p.Situaciones {
		situacion, err := p.Situaciones[indice].clonarCanonica()
		if err != nil {
			return ParticipacionBolsa{}, err
		}
		clon.Situaciones[indice] = situacion
	}
	return clon, nil
}

func (p ParticipacionBolsa) HuellaCanonicaSHA256() (string, error) {
	clon, err := p.ClonarCanonica()
	if err != nil {
		return "", err
	}
	return huellaJSON(clon)
}

// RequisitoCobertura referencia una condicion gobernada. Clave y valor no son
// texto libre; la interpretacion corresponde a la politica versionada.
type RequisitoCobertura struct {
	Clave        string `json:"clave"`
	ValorRef     string `json:"valor_ref"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

func (r RequisitoCobertura) Validar() error {
	if !claveLlamamientoValida(r.Clave) || !referenciaLlamamientoOpacaValida(r.ValorRef) ||
		r.Version == 0 || !huellaSHA256Valida(r.HuellaSHA256) {
		return ErrNecesidadCoberturaInvalida
	}
	return nil
}

// NecesidadCobertura contiene solo datos exactos y referencias gobernadas. La
// version y huella de bolsa impiden reutilizarla con otra constitucion. La
// duracion se representa mediante instantes, no float ni una regla codificada.
type NecesidadCobertura struct {
	NecesidadRef      string               `json:"necesidad_ref"`
	Version           uint64               `json:"version"`
	BolsaRef          string               `json:"bolsa_ref"`
	VersionBolsa      uint64               `json:"version_bolsa"`
	HuellaBolsaSHA256 string               `json:"huella_bolsa_sha256"`
	CategoriaRef      string               `json:"categoria_ref"`
	PuestoRef         string               `json:"puesto_ref"`
	UnidadRef         string               `json:"unidad_ref"`
	TipoCoberturaRef  string               `json:"tipo_cobertura_ref"`
	NumeroPuestos     uint64               `json:"numero_puestos"`
	InicioPrevisto    time.Time            `json:"inicio_previsto"`
	FinPrevisto       time.Time            `json:"fin_previsto"`
	CreadaEn          time.Time            `json:"creada_en"`
	Requisitos        []RequisitoCobertura `json:"requisitos"`
}

type AltaNecesidadCobertura = NecesidadCobertura

func NuevaNecesidadCobertura(alta AltaNecesidadCobertura) (NecesidadCobertura, error) {
	if len(alta.Requisitos) > maximoRequisitosCobertura {
		return NecesidadCobertura{}, ErrNecesidadCoberturaInvalida
	}
	necesidad := NecesidadCobertura(alta)
	var err error
	necesidad.InicioPrevisto, err = normalizarInstanteLlamamiento(necesidad.InicioPrevisto)
	if err != nil {
		return NecesidadCobertura{}, ErrNecesidadCoberturaInvalida
	}
	necesidad.FinPrevisto, err = normalizarInstanteLlamamiento(necesidad.FinPrevisto)
	if err != nil {
		return NecesidadCobertura{}, ErrNecesidadCoberturaInvalida
	}
	necesidad.CreadaEn, err = normalizarInstanteLlamamiento(necesidad.CreadaEn)
	if err != nil {
		return NecesidadCobertura{}, ErrNecesidadCoberturaInvalida
	}
	necesidad.Requisitos = append([]RequisitoCobertura(nil), alta.Requisitos...)
	ordenarRequisitosCobertura(necesidad.Requisitos)
	if err := necesidad.Validar(); err != nil {
		return NecesidadCobertura{}, err
	}
	return necesidad, nil
}

func (n NecesidadCobertura) Validar() error {
	if !referenciaLlamamientoOpacaValida(n.NecesidadRef) || n.Version == 0 ||
		!referenciaLlamamientoOpacaValida(n.BolsaRef) || n.VersionBolsa == 0 ||
		!huellaSHA256Valida(n.HuellaBolsaSHA256) ||
		!referenciaLlamamientoOpacaValida(n.CategoriaRef) ||
		!referenciaLlamamientoOpacaValida(n.PuestoRef) ||
		!referenciaLlamamientoOpacaValida(n.UnidadRef) ||
		!referenciaLlamamientoOpacaValida(n.TipoCoberturaRef) ||
		n.NumeroPuestos == 0 || n.NumeroPuestos > maximoPuestosNecesidad ||
		!instanteLlamamientoCanonico(n.InicioPrevisto) || !instanteLlamamientoCanonico(n.FinPrevisto) ||
		!instanteLlamamientoCanonico(n.CreadaEn) || !n.FinPrevisto.After(n.InicioPrevisto) ||
		!n.CreadaEn.Before(n.FinPrevisto) ||
		len(n.Requisitos) > maximoRequisitosCobertura {
		return ErrNecesidadCoberturaInvalida
	}
	for indice := range n.Requisitos {
		if n.Requisitos[indice].Validar() != nil {
			return ErrNecesidadCoberturaInvalida
		}
		if indice > 0 && n.Requisitos[indice-1].Clave >= n.Requisitos[indice].Clave {
			return ErrNecesidadCoberturaInvalida
		}
	}
	return nil
}

func (n NecesidadCobertura) ClonarCanonica() (NecesidadCobertura, error) {
	if err := n.Validar(); err != nil {
		return NecesidadCobertura{}, err
	}
	clon := n
	clon.Requisitos = append([]RequisitoCobertura(nil), n.Requisitos...)
	return clon, nil
}

func (n NecesidadCobertura) HuellaCanonicaSHA256() (string, error) {
	clon, err := n.ClonarCanonica()
	if err != nil {
		return "", err
	}
	return huellaJSON(clon)
}

// ReferenciaPoliticaLlamamiento fija los bytes y la vigencia de la politica
// usada. Su contenido ejecutable vive fuera de este nucleo puro.
type ReferenciaPoliticaLlamamiento struct {
	PoliticaRef  string     `json:"politica_ref"`
	Clave        string     `json:"clave"`
	Version      uint64     `json:"version"`
	HuellaSHA256 string     `json:"huella_sha256"`
	PublicadaEn  time.Time  `json:"publicada_en"`
	VigenteDesde time.Time  `json:"vigente_desde"`
	VigenteHasta *time.Time `json:"vigente_hasta,omitempty"`
}

func NuevaReferenciaPoliticaLlamamiento(datos ReferenciaPoliticaLlamamiento) (ReferenciaPoliticaLlamamiento, error) {
	politica := datos
	var err error
	politica.PublicadaEn, err = normalizarInstanteLlamamiento(politica.PublicadaEn)
	if err != nil {
		return ReferenciaPoliticaLlamamiento{}, ErrPoliticaLlamamientoInvalida
	}
	politica.VigenteDesde, err = normalizarInstanteLlamamiento(politica.VigenteDesde)
	if err != nil {
		return ReferenciaPoliticaLlamamiento{}, ErrPoliticaLlamamientoInvalida
	}
	politica.VigenteHasta, err = normalizarInstanteOpcionalLlamamiento(politica.VigenteHasta)
	if err != nil {
		return ReferenciaPoliticaLlamamiento{}, ErrPoliticaLlamamientoInvalida
	}
	if err := politica.Validar(); err != nil {
		return ReferenciaPoliticaLlamamiento{}, err
	}
	return politica, nil
}

func (p ReferenciaPoliticaLlamamiento) Validar() error {
	if !referenciaLlamamientoOpacaValida(p.PoliticaRef) || !claveLlamamientoValida(p.Clave) ||
		p.Version == 0 || !huellaSHA256Valida(p.HuellaSHA256) ||
		!instanteLlamamientoCanonico(p.PublicadaEn) || !instanteLlamamientoCanonico(p.VigenteDesde) ||
		p.VigenteDesde.Before(p.PublicadaEn) || !intervaloAbiertoLlamamientoValido(p.VigenteDesde, p.VigenteHasta) {
		return ErrPoliticaLlamamientoInvalida
	}
	return nil
}

func (p ReferenciaPoliticaLlamamiento) VigenteEn(instante time.Time) bool {
	canonico, err := normalizarInstanteLlamamiento(instante)
	if err != nil || p.Validar() != nil || canonico.Before(p.VigenteDesde) {
		return false
	}
	return p.VigenteHasta == nil || canonico.Before(*p.VigenteHasta)
}

func (p ReferenciaPoliticaLlamamiento) ClonarCanonica() (ReferenciaPoliticaLlamamiento, error) {
	if err := p.Validar(); err != nil {
		return ReferenciaPoliticaLlamamiento{}, err
	}
	clon := p
	clon.VigenteHasta = clonarInstanteOpcional(p.VigenteHasta)
	return clon, nil
}

type EntradaOrdenBolsa struct {
	Orden         uint64             `json:"orden"`
	Participacion ParticipacionBolsa `json:"participacion"`
}

func (e EntradaOrdenBolsa) clonarCanonica() (EntradaOrdenBolsa, error) {
	if e.Orden == 0 {
		return EntradaOrdenBolsa{}, ErrInstantaneaOrdenBolsaInvalida
	}
	participacion, err := e.Participacion.ClonarCanonica()
	if err != nil {
		return EntradaOrdenBolsa{}, ErrInstantaneaOrdenBolsaInvalida
	}
	return EntradaOrdenBolsa{Orden: e.Orden, Participacion: participacion}, nil
}

// InstantaneaOrdenBolsa congela el orden completo y la situacion vigente de
// cada participacion en un instante de referencia. La huella se calcula sobre
// una representacion JSON de campos y orden fijos, nunca sobre un map.
type InstantaneaOrdenBolsa struct {
	InstantaneaRef        string              `json:"instantanea_ref"`
	Version               uint64              `json:"version"`
	BolsaRef              string              `json:"bolsa_ref"`
	VersionBolsa          uint64              `json:"version_bolsa"`
	HuellaBolsaSHA256     string              `json:"huella_bolsa_sha256"`
	ListadoDefinitivoRef  string              `json:"listado_definitivo_ref"`
	VersionListado        uint64              `json:"version_listado"`
	HuellaListadoSHA256   string              `json:"huella_listado_sha256"`
	ReferidaEn            time.Time           `json:"referida_en"`
	GeneradaEn            time.Time           `json:"generada_en"`
	Entradas              []EntradaOrdenBolsa `json:"entradas"`
	HuellaContenidoSHA256 string              `json:"huella_contenido_sha256"`
}

type AltaInstantaneaOrdenBolsa struct {
	InstantaneaRef string
	Version        uint64
	Bolsa          BolsaConstituida
	ReferidaEn     time.Time
	GeneradaEn     time.Time
	Entradas       []EntradaOrdenBolsa
}

func NuevaInstantaneaOrdenBolsa(alta AltaInstantaneaOrdenBolsa) (InstantaneaOrdenBolsa, error) {
	if len(alta.Entradas) == 0 || len(alta.Entradas) > maximoEntradasOrdenBolsa {
		return InstantaneaOrdenBolsa{}, ErrInstantaneaOrdenBolsaInvalida
	}
	bolsa, err := alta.Bolsa.ClonarCanonica()
	if err != nil {
		return InstantaneaOrdenBolsa{}, ErrInstantaneaOrdenBolsaInvalida
	}
	referidaEn, err := normalizarInstanteLlamamiento(alta.ReferidaEn)
	if err != nil {
		return InstantaneaOrdenBolsa{}, ErrInstantaneaOrdenBolsaInvalida
	}
	generadaEn, err := normalizarInstanteLlamamiento(alta.GeneradaEn)
	if err != nil {
		return InstantaneaOrdenBolsa{}, ErrInstantaneaOrdenBolsaInvalida
	}
	huellaBolsa, err := bolsa.HuellaCanonicaSHA256()
	if err != nil {
		return InstantaneaOrdenBolsa{}, ErrInstantaneaOrdenBolsaInvalida
	}
	instantanea := InstantaneaOrdenBolsa{
		InstantaneaRef: alta.InstantaneaRef, Version: alta.Version,
		BolsaRef: bolsa.BolsaRef, VersionBolsa: bolsa.Version, HuellaBolsaSHA256: huellaBolsa,
		ListadoDefinitivoRef: bolsa.ListadoDefinitivoRef, VersionListado: bolsa.VersionListado,
		HuellaListadoSHA256: bolsa.HuellaListadoSHA256, ReferidaEn: referidaEn, GeneradaEn: generadaEn,
		Entradas: make([]EntradaOrdenBolsa, len(alta.Entradas)),
	}
	for indice := range alta.Entradas {
		entrada, err := alta.Entradas[indice].clonarCanonica()
		if err != nil {
			return InstantaneaOrdenBolsa{}, ErrInstantaneaOrdenBolsaInvalida
		}
		instantanea.Entradas[indice] = entrada
	}
	sort.Slice(instantanea.Entradas, func(i, j int) bool { return instantanea.Entradas[i].Orden < instantanea.Entradas[j].Orden })
	if !bolsa.VigenteEn(referidaEn) || instantanea.validarContenido(false) != nil {
		return InstantaneaOrdenBolsa{}, ErrInstantaneaOrdenBolsaInvalida
	}
	instantanea.HuellaContenidoSHA256, err = instantanea.calcularHuellaContenidoSHA256()
	if err != nil || instantanea.Validar() != nil {
		return InstantaneaOrdenBolsa{}, ErrInstantaneaOrdenBolsaInvalida
	}
	return instantanea, nil
}

func (i InstantaneaOrdenBolsa) Validar() error {
	return i.validarContenido(true)
}

func (i InstantaneaOrdenBolsa) validarContenido(comprobarHuella bool) error {
	if !referenciaLlamamientoOpacaValida(i.InstantaneaRef) || i.Version == 0 ||
		!referenciaLlamamientoOpacaValida(i.BolsaRef) || i.VersionBolsa == 0 ||
		!huellaSHA256Valida(i.HuellaBolsaSHA256) ||
		!referenciaLlamamientoOpacaValida(i.ListadoDefinitivoRef) || i.VersionListado == 0 ||
		!huellaSHA256Valida(i.HuellaListadoSHA256) || !instanteLlamamientoCanonico(i.ReferidaEn) ||
		!instanteLlamamientoCanonico(i.GeneradaEn) || i.GeneradaEn.Before(i.ReferidaEn) ||
		len(i.Entradas) == 0 || len(i.Entradas) > maximoEntradasOrdenBolsa {
		return ErrInstantaneaOrdenBolsaInvalida
	}
	participaciones := make(map[string]struct{}, len(i.Entradas))
	sujetos := make(map[string]struct{}, len(i.Entradas))
	for indice := range i.Entradas {
		entrada := i.Entradas[indice]
		if entrada.Orden != uint64(indice+1) || entrada.Participacion.Validar() != nil ||
			entrada.Participacion.BolsaRef != i.BolsaRef {
			return ErrInstantaneaOrdenBolsaInvalida
		}
		if _, existe := participaciones[entrada.Participacion.ParticipacionRef]; existe {
			return ErrInstantaneaOrdenBolsaInvalida
		}
		if _, existe := sujetos[entrada.Participacion.SujetoRef]; existe {
			return ErrInstantaneaOrdenBolsaInvalida
		}
		participaciones[entrada.Participacion.ParticipacionRef] = struct{}{}
		sujetos[entrada.Participacion.SujetoRef] = struct{}{}
		if _, vigente := entrada.Participacion.SituacionVigenteEn(i.ReferidaEn); !vigente {
			return ErrInstantaneaOrdenBolsaInvalida
		}
	}
	if comprobarHuella {
		if !huellaSHA256Valida(i.HuellaContenidoSHA256) {
			return ErrInstantaneaOrdenBolsaInvalida
		}
		huella, err := i.calcularHuellaContenidoSHA256()
		if err != nil || huella != i.HuellaContenidoSHA256 {
			return ErrInstantaneaOrdenBolsaInvalida
		}
	}
	return nil
}

func (i InstantaneaOrdenBolsa) calcularHuellaContenidoSHA256() (string, error) {
	contenido := struct {
		InstantaneaRef       string              `json:"instantanea_ref"`
		Version              uint64              `json:"version"`
		BolsaRef             string              `json:"bolsa_ref"`
		VersionBolsa         uint64              `json:"version_bolsa"`
		HuellaBolsaSHA256    string              `json:"huella_bolsa_sha256"`
		ListadoDefinitivoRef string              `json:"listado_definitivo_ref"`
		VersionListado       uint64              `json:"version_listado"`
		HuellaListadoSHA256  string              `json:"huella_listado_sha256"`
		ReferidaEn           time.Time           `json:"referida_en"`
		GeneradaEn           time.Time           `json:"generada_en"`
		Entradas             []EntradaOrdenBolsa `json:"entradas"`
	}{
		i.InstantaneaRef, i.Version, i.BolsaRef, i.VersionBolsa, i.HuellaBolsaSHA256,
		i.ListadoDefinitivoRef, i.VersionListado, i.HuellaListadoSHA256,
		i.ReferidaEn, i.GeneradaEn, i.Entradas,
	}
	return huellaJSON(contenido)
}

func (i InstantaneaOrdenBolsa) ClonarCanonica() (InstantaneaOrdenBolsa, error) {
	if err := i.Validar(); err != nil {
		return InstantaneaOrdenBolsa{}, err
	}
	clon := i
	clon.Entradas = make([]EntradaOrdenBolsa, len(i.Entradas))
	for indice := range i.Entradas {
		entrada, err := i.Entradas[indice].clonarCanonica()
		if err != nil {
			return InstantaneaOrdenBolsa{}, err
		}
		clon.Entradas[indice] = entrada
	}
	return clon, nil
}

type ResultadoElegibilidadLlamamiento string

const (
	ResultadoElegible   ResultadoElegibilidadLlamamiento = "elegible"
	ResultadoNoElegible ResultadoElegibilidadLlamamiento = "no_elegible"
)

func (r ResultadoElegibilidadLlamamiento) Valido() bool {
	return r == ResultadoElegible || r == ResultadoNoElegible
}

type MotivoEvaluacionLlamamiento struct {
	Clave             string `json:"clave"`
	ReglaRef          string `json:"regla_ref"`
	VersionRegla      uint64 `json:"version_regla"`
	HuellaReglaSHA256 string `json:"huella_regla_sha256"`
}

func (m MotivoEvaluacionLlamamiento) Validar() error {
	if !claveLlamamientoValida(m.Clave) || !referenciaLlamamientoOpacaValida(m.ReglaRef) ||
		m.VersionRegla == 0 || !huellaSHA256Valida(m.HuellaReglaSHA256) {
		return ErrEvaluacionLlamamientoInvalida
	}
	return nil
}

// EvaluacionParticipacionLlamamiento es un recibo del motor de politicas. Sus
// referencias y huellas enlazan necesidad, instantanea, politica y situacion;
// EvaluadaEn registra la ejecucion real, posterior a la generacion de la
// instantanea. Un booleano suelto nunca basta para proponer un llamamiento.
type EvaluacionParticipacionLlamamiento struct {
	ParticipacionRef        string                           `json:"participacion_ref"`
	SujetoRef               string                           `json:"sujeto_ref"`
	Orden                   uint64                           `json:"orden"`
	SituacionSecuencia      uint64                           `json:"situacion_secuencia"`
	EstadoClave             string                           `json:"estado_clave"`
	EstadoVersion           uint64                           `json:"estado_version"`
	HuellaEstadoSHA256      string                           `json:"huella_estado_sha256"`
	NecesidadRef            string                           `json:"necesidad_ref"`
	VersionNecesidad        uint64                           `json:"version_necesidad"`
	HuellaNecesidadSHA256   string                           `json:"huella_necesidad_sha256"`
	InstantaneaRef          string                           `json:"instantanea_ref"`
	VersionInstantanea      uint64                           `json:"version_instantanea"`
	HuellaInstantaneaSHA256 string                           `json:"huella_instantanea_sha256"`
	PoliticaRef             string                           `json:"politica_ref"`
	VersionPolitica         uint64                           `json:"version_politica"`
	HuellaPoliticaSHA256    string                           `json:"huella_politica_sha256"`
	Resultado               ResultadoElegibilidadLlamamiento `json:"resultado"`
	Motivos                 []MotivoEvaluacionLlamamiento    `json:"motivos"`
	EntradaEvaluacionRef    string                           `json:"entrada_evaluacion_ref"`
	HuellaEntradaSHA256     string                           `json:"huella_entrada_sha256"`
	ResultadoEvaluacionRef  string                           `json:"resultado_evaluacion_ref"`
	HuellaResultadoSHA256   string                           `json:"huella_resultado_sha256"`
	EvaluadaEn              time.Time                        `json:"evaluada_en"`
}

func (e EvaluacionParticipacionLlamamiento) Validar() error {
	if !referenciaLlamamientoOpacaValida(e.ParticipacionRef) ||
		!referenciaLlamamientoOpacaValida(e.SujetoRef) || e.Orden == 0 || e.SituacionSecuencia == 0 ||
		!claveLlamamientoValida(e.EstadoClave) || e.EstadoVersion == 0 ||
		!huellaSHA256Valida(e.HuellaEstadoSHA256) ||
		!referenciaLlamamientoOpacaValida(e.NecesidadRef) || e.VersionNecesidad == 0 ||
		!huellaSHA256Valida(e.HuellaNecesidadSHA256) ||
		!referenciaLlamamientoOpacaValida(e.InstantaneaRef) || e.VersionInstantanea == 0 ||
		!huellaSHA256Valida(e.HuellaInstantaneaSHA256) ||
		!referenciaLlamamientoOpacaValida(e.PoliticaRef) || e.VersionPolitica == 0 ||
		!huellaSHA256Valida(e.HuellaPoliticaSHA256) || !e.Resultado.Valido() ||
		len(e.Motivos) == 0 || len(e.Motivos) > maximoMotivosEvaluacion ||
		!referenciaLlamamientoOpacaValida(e.EntradaEvaluacionRef) ||
		!huellaSHA256Valida(e.HuellaEntradaSHA256) ||
		!referenciaLlamamientoOpacaValida(e.ResultadoEvaluacionRef) ||
		e.ResultadoEvaluacionRef == e.EntradaEvaluacionRef ||
		!huellaSHA256Valida(e.HuellaResultadoSHA256) || !instanteLlamamientoCanonico(e.EvaluadaEn) {
		return ErrEvaluacionLlamamientoInvalida
	}
	for indice := range e.Motivos {
		if e.Motivos[indice].Validar() != nil {
			return ErrEvaluacionLlamamientoInvalida
		}
		if indice > 0 && claveMotivoEvaluacion(e.Motivos[indice-1]) >= claveMotivoEvaluacion(e.Motivos[indice]) {
			return ErrEvaluacionLlamamientoInvalida
		}
	}
	return nil
}

func (e EvaluacionParticipacionLlamamiento) clonarCanonica() (EvaluacionParticipacionLlamamiento, error) {
	if len(e.Motivos) == 0 || len(e.Motivos) > maximoMotivosEvaluacion {
		return EvaluacionParticipacionLlamamiento{}, ErrEvaluacionLlamamientoInvalida
	}
	clon := e
	instante, err := normalizarInstanteLlamamiento(e.EvaluadaEn)
	if err != nil {
		return EvaluacionParticipacionLlamamiento{}, ErrEvaluacionLlamamientoInvalida
	}
	clon.EvaluadaEn = instante
	clon.Motivos = append([]MotivoEvaluacionLlamamiento(nil), e.Motivos...)
	sort.Slice(clon.Motivos, func(i, j int) bool {
		return claveMotivoEvaluacion(clon.Motivos[i]) < claveMotivoEvaluacion(clon.Motivos[j])
	})
	if err := clon.Validar(); err != nil {
		return EvaluacionParticipacionLlamamiento{}, err
	}
	return clon, nil
}

// PropuestaLlamamiento conserva el prefijo completo del orden hasta la primera
// participacion elegible y la cronologia causal de sus recibos. Asi demuestra
// que no se ha omitido a nadie anterior sin tratar innecesariamente a quienes
// se encuentran despues.
type PropuestaLlamamiento struct {
	PropuestaRef                    string                               `json:"propuesta_ref"`
	BolsaRef                        string                               `json:"bolsa_ref"`
	VersionBolsa                    uint64                               `json:"version_bolsa"`
	HuellaBolsaSHA256               string                               `json:"huella_bolsa_sha256"`
	NecesidadRef                    string                               `json:"necesidad_ref"`
	VersionNecesidad                uint64                               `json:"version_necesidad"`
	HuellaNecesidadSHA256           string                               `json:"huella_necesidad_sha256"`
	InstantaneaRef                  string                               `json:"instantanea_ref"`
	VersionInstantanea              uint64                               `json:"version_instantanea"`
	HuellaInstantaneaSHA256         string                               `json:"huella_instantanea_sha256"`
	PoliticaRef                     string                               `json:"politica_ref"`
	VersionPolitica                 uint64                               `json:"version_politica"`
	HuellaPoliticaSHA256            string                               `json:"huella_politica_sha256"`
	InstanteReferencia              time.Time                            `json:"instante_referencia"`
	InstantaneaGeneradaEn           time.Time                            `json:"instantanea_generada_en"`
	TotalParticipacionesInstantanea uint64                               `json:"total_participaciones_instantanea"`
	Evaluaciones                    []EvaluacionParticipacionLlamamiento `json:"evaluaciones"`
	ParticipacionSeleccionadaRef    string                               `json:"participacion_seleccionada_ref"`
	SujetoSeleccionadoRef           string                               `json:"sujeto_seleccionado_ref"`
	OrdenSeleccionado               uint64                               `json:"orden_seleccionado"`
	GeneradaEn                      time.Time                            `json:"generada_en"`
	HuellaContenidoSHA256           string                               `json:"huella_contenido_sha256"`
}

type OrdenProponerPrimerLlamamiento struct {
	PropuestaRef string
	Bolsa        BolsaConstituida
	Necesidad    NecesidadCobertura
	Instantanea  InstantaneaOrdenBolsa
	Politica     ReferenciaPoliticaLlamamiento
	Evaluaciones []EvaluacionParticipacionLlamamiento
	GeneradaEn   time.Time
}

func ProponerPrimerLlamamiento(orden OrdenProponerPrimerLlamamiento) (PropuestaLlamamiento, error) {
	if len(orden.Evaluaciones) == 0 {
		return PropuestaLlamamiento{}, ErrSinParticipacionElegible
	}
	bolsa, err := orden.Bolsa.ClonarCanonica()
	if err != nil {
		return PropuestaLlamamiento{}, ErrPropuestaLlamamientoInvalida
	}
	necesidad, err := orden.Necesidad.ClonarCanonica()
	if err != nil {
		return PropuestaLlamamiento{}, ErrPropuestaLlamamientoInvalida
	}
	instantanea, err := orden.Instantanea.ClonarCanonica()
	if err != nil {
		return PropuestaLlamamiento{}, ErrPropuestaLlamamientoInvalida
	}
	politica, err := orden.Politica.ClonarCanonica()
	if err != nil {
		return PropuestaLlamamiento{}, ErrPropuestaLlamamientoInvalida
	}
	generadaEn, err := normalizarInstanteLlamamiento(orden.GeneradaEn)
	if err != nil {
		return PropuestaLlamamiento{}, ErrPropuestaLlamamientoInvalida
	}
	if necesidad.BolsaRef != bolsa.BolsaRef || necesidad.VersionBolsa != bolsa.Version ||
		necesidad.CategoriaRef != bolsa.CategoriaRef ||
		instantanea.BolsaRef != bolsa.BolsaRef || instantanea.VersionBolsa != bolsa.Version ||
		instantanea.ListadoDefinitivoRef != bolsa.ListadoDefinitivoRef ||
		instantanea.VersionListado != bolsa.VersionListado ||
		instantanea.HuellaListadoSHA256 != bolsa.HuellaListadoSHA256 ||
		!bolsa.VigenteEn(instantanea.ReferidaEn) || !politica.VigenteEn(instantanea.ReferidaEn) ||
		!bolsa.VigenteEn(generadaEn) || !politica.VigenteEn(generadaEn) ||
		instantanea.ReferidaEn.Before(necesidad.CreadaEn) || !instantanea.ReferidaEn.Before(necesidad.FinPrevisto) ||
		generadaEn.Before(instantanea.GeneradaEn) || generadaEn.Before(necesidad.CreadaEn) ||
		!generadaEn.Before(necesidad.FinPrevisto) {
		return PropuestaLlamamiento{}, ErrPropuestaLlamamientoInvalida
	}
	huellaBolsa, err := bolsa.HuellaCanonicaSHA256()
	if err != nil || huellaBolsa != instantanea.HuellaBolsaSHA256 || huellaBolsa != necesidad.HuellaBolsaSHA256 {
		return PropuestaLlamamiento{}, ErrPropuestaLlamamientoInvalida
	}
	huellaNecesidad, err := necesidad.HuellaCanonicaSHA256()
	if err != nil {
		return PropuestaLlamamiento{}, ErrPropuestaLlamamientoInvalida
	}
	if len(orden.Evaluaciones) > len(instantanea.Entradas) {
		return PropuestaLlamamiento{}, ErrEvaluacionLlamamientoInvalida
	}
	evaluaciones := make([]EvaluacionParticipacionLlamamiento, len(orden.Evaluaciones))
	for indice := range orden.Evaluaciones {
		evaluacion, err := orden.Evaluaciones[indice].clonarCanonica()
		if err != nil {
			return PropuestaLlamamiento{}, ErrEvaluacionLlamamientoInvalida
		}
		evaluaciones[indice] = evaluacion
	}
	sort.Slice(evaluaciones, func(i, j int) bool { return evaluaciones[i].Orden < evaluaciones[j].Orden })
	if err := validarEvaluacionesContraEntradas(
		evaluaciones, instantanea, necesidad, huellaNecesidad, politica, generadaEn,
	); err != nil {
		return PropuestaLlamamiento{}, err
	}
	seleccionada := evaluaciones[len(evaluaciones)-1]
	if seleccionada.Resultado != ResultadoElegible {
		return PropuestaLlamamiento{}, ErrSinParticipacionElegible
	}
	for indice := 0; indice < len(evaluaciones)-1; indice++ {
		if evaluaciones[indice].Resultado != ResultadoNoElegible {
			return PropuestaLlamamiento{}, ErrEvaluacionLlamamientoInvalida
		}
	}
	propuesta := PropuestaLlamamiento{
		PropuestaRef: orden.PropuestaRef,
		BolsaRef:     bolsa.BolsaRef, VersionBolsa: bolsa.Version, HuellaBolsaSHA256: huellaBolsa,
		NecesidadRef: necesidad.NecesidadRef, VersionNecesidad: necesidad.Version, HuellaNecesidadSHA256: huellaNecesidad,
		InstantaneaRef: instantanea.InstantaneaRef, VersionInstantanea: instantanea.Version,
		HuellaInstantaneaSHA256: instantanea.HuellaContenidoSHA256,
		PoliticaRef:             politica.PoliticaRef, VersionPolitica: politica.Version, HuellaPoliticaSHA256: politica.HuellaSHA256,
		InstanteReferencia: instantanea.ReferidaEn, InstantaneaGeneradaEn: instantanea.GeneradaEn,
		TotalParticipacionesInstantanea: uint64(len(instantanea.Entradas)), Evaluaciones: evaluaciones,
		ParticipacionSeleccionadaRef: seleccionada.ParticipacionRef, SujetoSeleccionadoRef: seleccionada.SujetoRef,
		OrdenSeleccionado: seleccionada.Orden, GeneradaEn: generadaEn,
	}
	if propuesta.validarContenido(false) != nil {
		return PropuestaLlamamiento{}, ErrPropuestaLlamamientoInvalida
	}
	propuesta.HuellaContenidoSHA256, err = propuesta.calcularHuellaContenidoSHA256()
	if err != nil || propuesta.Validar() != nil {
		return PropuestaLlamamiento{}, ErrPropuestaLlamamientoInvalida
	}
	return propuesta, nil
}

func validarEvaluacionesContraEntradas(
	evaluaciones []EvaluacionParticipacionLlamamiento,
	instantanea InstantaneaOrdenBolsa,
	necesidad NecesidadCobertura,
	huellaNecesidad string,
	politica ReferenciaPoliticaLlamamiento,
	generadaEn time.Time,
) error {
	if len(evaluaciones) == 0 {
		return ErrSinParticipacionElegible
	}
	if len(evaluaciones) > len(instantanea.Entradas) {
		return ErrEvaluacionLlamamientoInvalida
	}
	for indice := range evaluaciones {
		evaluacion := evaluaciones[indice]
		entrada := instantanea.Entradas[indice]
		situacion, vigente := entrada.Participacion.SituacionVigenteEn(instantanea.ReferidaEn)
		if !vigente || evaluacion.Validar() != nil || evaluacion.Orden != uint64(indice+1) ||
			evaluacion.Orden != entrada.Orden ||
			evaluacion.ParticipacionRef != entrada.Participacion.ParticipacionRef ||
			evaluacion.SujetoRef != entrada.Participacion.SujetoRef ||
			evaluacion.SituacionSecuencia != situacion.Secuencia || evaluacion.EstadoClave != situacion.EstadoClave ||
			evaluacion.EstadoVersion != situacion.EstadoVersion || evaluacion.HuellaEstadoSHA256 != situacion.HuellaEstadoSHA256 ||
			evaluacion.NecesidadRef != necesidad.NecesidadRef || evaluacion.VersionNecesidad != necesidad.Version ||
			evaluacion.HuellaNecesidadSHA256 != huellaNecesidad ||
			evaluacion.InstantaneaRef != instantanea.InstantaneaRef || evaluacion.VersionInstantanea != instantanea.Version ||
			evaluacion.HuellaInstantaneaSHA256 != instantanea.HuellaContenidoSHA256 ||
			evaluacion.PoliticaRef != politica.PoliticaRef || evaluacion.VersionPolitica != politica.Version ||
			evaluacion.HuellaPoliticaSHA256 != politica.HuellaSHA256 ||
			evaluacion.EvaluadaEn.Before(instantanea.GeneradaEn) || evaluacion.EvaluadaEn.After(generadaEn) {
			return ErrEvaluacionLlamamientoInvalida
		}
	}
	return nil
}

func (p PropuestaLlamamiento) Validar() error {
	return p.validarContenido(true)
}

func (p PropuestaLlamamiento) validarContenido(comprobarHuella bool) error {
	if !referenciaLlamamientoOpacaValida(p.PropuestaRef) ||
		!referenciaLlamamientoOpacaValida(p.BolsaRef) || p.VersionBolsa == 0 || !huellaSHA256Valida(p.HuellaBolsaSHA256) ||
		!referenciaLlamamientoOpacaValida(p.NecesidadRef) || p.VersionNecesidad == 0 || !huellaSHA256Valida(p.HuellaNecesidadSHA256) ||
		!referenciaLlamamientoOpacaValida(p.InstantaneaRef) || p.VersionInstantanea == 0 || !huellaSHA256Valida(p.HuellaInstantaneaSHA256) ||
		!referenciaLlamamientoOpacaValida(p.PoliticaRef) || p.VersionPolitica == 0 || !huellaSHA256Valida(p.HuellaPoliticaSHA256) ||
		!instanteLlamamientoCanonico(p.InstanteReferencia) || !instanteLlamamientoCanonico(p.InstantaneaGeneradaEn) ||
		!instanteLlamamientoCanonico(p.GeneradaEn) || p.InstantaneaGeneradaEn.Before(p.InstanteReferencia) ||
		p.GeneradaEn.Before(p.InstantaneaGeneradaEn) || p.TotalParticipacionesInstantanea == 0 ||
		p.TotalParticipacionesInstantanea > maximoEntradasOrdenBolsa ||
		uint64(len(p.Evaluaciones)) > p.TotalParticipacionesInstantanea || len(p.Evaluaciones) == 0 ||
		len(p.Evaluaciones) > maximoEntradasOrdenBolsa ||
		!referenciaLlamamientoOpacaValida(p.ParticipacionSeleccionadaRef) ||
		!referenciaLlamamientoOpacaValida(p.SujetoSeleccionadoRef) || p.OrdenSeleccionado == 0 {
		return ErrPropuestaLlamamientoInvalida
	}
	participaciones := make(map[string]struct{}, len(p.Evaluaciones))
	sujetos := make(map[string]struct{}, len(p.Evaluaciones))
	recibosEvaluacion := make(map[string]struct{}, len(p.Evaluaciones)*2)
	for indice := range p.Evaluaciones {
		evaluacion := p.Evaluaciones[indice]
		if evaluacion.Validar() != nil || evaluacion.Orden != uint64(indice+1) ||
			evaluacion.NecesidadRef != p.NecesidadRef || evaluacion.VersionNecesidad != p.VersionNecesidad ||
			evaluacion.HuellaNecesidadSHA256 != p.HuellaNecesidadSHA256 ||
			evaluacion.InstantaneaRef != p.InstantaneaRef || evaluacion.VersionInstantanea != p.VersionInstantanea ||
			evaluacion.HuellaInstantaneaSHA256 != p.HuellaInstantaneaSHA256 ||
			evaluacion.PoliticaRef != p.PoliticaRef || evaluacion.VersionPolitica != p.VersionPolitica ||
			evaluacion.HuellaPoliticaSHA256 != p.HuellaPoliticaSHA256 ||
			evaluacion.EvaluadaEn.Before(p.InstantaneaGeneradaEn) || evaluacion.EvaluadaEn.After(p.GeneradaEn) {
			return ErrPropuestaLlamamientoInvalida
		}
		if indice < len(p.Evaluaciones)-1 && evaluacion.Resultado != ResultadoNoElegible {
			return ErrPropuestaLlamamientoInvalida
		}
		if _, existe := participaciones[evaluacion.ParticipacionRef]; existe {
			return ErrPropuestaLlamamientoInvalida
		}
		if _, existe := sujetos[evaluacion.SujetoRef]; existe {
			return ErrPropuestaLlamamientoInvalida
		}
		if _, existe := recibosEvaluacion[evaluacion.EntradaEvaluacionRef]; existe {
			return ErrPropuestaLlamamientoInvalida
		}
		recibosEvaluacion[evaluacion.EntradaEvaluacionRef] = struct{}{}
		if _, existe := recibosEvaluacion[evaluacion.ResultadoEvaluacionRef]; existe {
			return ErrPropuestaLlamamientoInvalida
		}
		participaciones[evaluacion.ParticipacionRef] = struct{}{}
		sujetos[evaluacion.SujetoRef] = struct{}{}
		recibosEvaluacion[evaluacion.ResultadoEvaluacionRef] = struct{}{}
	}
	seleccionada := p.Evaluaciones[len(p.Evaluaciones)-1]
	if seleccionada.Resultado != ResultadoElegible || seleccionada.ParticipacionRef != p.ParticipacionSeleccionadaRef ||
		seleccionada.SujetoRef != p.SujetoSeleccionadoRef || seleccionada.Orden != p.OrdenSeleccionado {
		return ErrPropuestaLlamamientoInvalida
	}
	if comprobarHuella {
		if !huellaSHA256Valida(p.HuellaContenidoSHA256) {
			return ErrPropuestaLlamamientoInvalida
		}
		huella, err := p.calcularHuellaContenidoSHA256()
		if err != nil || huella != p.HuellaContenidoSHA256 {
			return ErrPropuestaLlamamientoInvalida
		}
	}
	return nil
}

func (p PropuestaLlamamiento) calcularHuellaContenidoSHA256() (string, error) {
	contenido := struct {
		PropuestaRef                    string                               `json:"propuesta_ref"`
		BolsaRef                        string                               `json:"bolsa_ref"`
		VersionBolsa                    uint64                               `json:"version_bolsa"`
		HuellaBolsaSHA256               string                               `json:"huella_bolsa_sha256"`
		NecesidadRef                    string                               `json:"necesidad_ref"`
		VersionNecesidad                uint64                               `json:"version_necesidad"`
		HuellaNecesidadSHA256           string                               `json:"huella_necesidad_sha256"`
		InstantaneaRef                  string                               `json:"instantanea_ref"`
		VersionInstantanea              uint64                               `json:"version_instantanea"`
		HuellaInstantaneaSHA256         string                               `json:"huella_instantanea_sha256"`
		PoliticaRef                     string                               `json:"politica_ref"`
		VersionPolitica                 uint64                               `json:"version_politica"`
		HuellaPoliticaSHA256            string                               `json:"huella_politica_sha256"`
		InstanteReferencia              time.Time                            `json:"instante_referencia"`
		InstantaneaGeneradaEn           time.Time                            `json:"instantanea_generada_en"`
		TotalParticipacionesInstantanea uint64                               `json:"total_participaciones_instantanea"`
		Evaluaciones                    []EvaluacionParticipacionLlamamiento `json:"evaluaciones"`
		ParticipacionSeleccionadaRef    string                               `json:"participacion_seleccionada_ref"`
		SujetoSeleccionadoRef           string                               `json:"sujeto_seleccionado_ref"`
		OrdenSeleccionado               uint64                               `json:"orden_seleccionado"`
		GeneradaEn                      time.Time                            `json:"generada_en"`
	}{
		p.PropuestaRef, p.BolsaRef, p.VersionBolsa, p.HuellaBolsaSHA256,
		p.NecesidadRef, p.VersionNecesidad, p.HuellaNecesidadSHA256,
		p.InstantaneaRef, p.VersionInstantanea, p.HuellaInstantaneaSHA256,
		p.PoliticaRef, p.VersionPolitica, p.HuellaPoliticaSHA256,
		p.InstanteReferencia, p.InstantaneaGeneradaEn, p.TotalParticipacionesInstantanea, p.Evaluaciones,
		p.ParticipacionSeleccionadaRef, p.SujetoSeleccionadoRef, p.OrdenSeleccionado, p.GeneradaEn,
	}
	return huellaJSON(contenido)
}

func (p PropuestaLlamamiento) ClonarCanonica() (PropuestaLlamamiento, error) {
	if err := p.Validar(); err != nil {
		return PropuestaLlamamiento{}, err
	}
	clon := p
	clon.Evaluaciones = make([]EvaluacionParticipacionLlamamiento, len(p.Evaluaciones))
	for indice := range p.Evaluaciones {
		evaluacion, err := p.Evaluaciones[indice].clonarCanonica()
		if err != nil {
			return PropuestaLlamamiento{}, err
		}
		clon.Evaluaciones[indice] = evaluacion
	}
	return clon, nil
}

func (p PropuestaLlamamiento) EvaluacionesCanonicas() ([]EvaluacionParticipacionLlamamiento, error) {
	clon, err := p.ClonarCanonica()
	if err != nil {
		return nil, err
	}
	return clon.Evaluaciones, nil
}

func ordenarRequisitosCobertura(requisitos []RequisitoCobertura) {
	sort.Slice(requisitos, func(i, j int) bool { return requisitos[i].Clave < requisitos[j].Clave })
}

func claveMotivoEvaluacion(motivo MotivoEvaluacionLlamamiento) string {
	return motivo.ReglaRef + "\x00" + motivo.Clave
}

func referenciaLlamamientoOpacaValida(valor string) bool {
	return referenciaOpacaValida(valor) && !strings.Contains(valor, "*") &&
		!patronDocumentoIdentidadEnReferencia.MatchString(valor) &&
		!patronEtiquetaDocumentoIdentidad.MatchString(valor)
}

func claveLlamamientoValida(valor string) bool {
	return claveNegocioValida(valor) &&
		!patronDocumentoIdentidadEnReferencia.MatchString(valor) &&
		!patronEtiquetaDocumentoIdentidad.MatchString(valor)
}

func normalizarInstanteLlamamiento(instante time.Time) (time.Time, error) {
	if instante.IsZero() || instante.Year() < 1 || instante.Year() > 9999 || instante.Nanosecond()%1_000 != 0 {
		return time.Time{}, ErrPropuestaLlamamientoInvalida
	}
	return instante.UTC(), nil
}

func normalizarInstanteOpcionalLlamamiento(instante *time.Time) (*time.Time, error) {
	if instante == nil {
		return nil, nil
	}
	canonico, err := normalizarInstanteLlamamiento(*instante)
	if err != nil {
		return nil, err
	}
	return &canonico, nil
}

func instanteLlamamientoCanonico(instante time.Time) bool {
	return !instante.IsZero() && instante.Year() >= 1 && instante.Year() <= 9999 &&
		instante.Location() == time.UTC && instante.Nanosecond()%1_000 == 0
}

func intervaloAbiertoLlamamientoValido(desde time.Time, hasta *time.Time) bool {
	return hasta == nil || (instanteLlamamientoCanonico(*hasta) && hasta.After(desde))
}

func clonarInstanteOpcional(instante *time.Time) *time.Time {
	if instante == nil {
		return nil
	}
	clon := *instante
	return &clon
}
