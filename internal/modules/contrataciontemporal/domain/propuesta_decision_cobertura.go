package domain

import (
	"sort"
	"time"
)

const (
	maximoResultadosEntradaPropuestaCobertura = 1024
	maximoEvidenciasPorComprobacionCobertura  = 4
)

type EstadoPropuestaDecisionCobertura string

const (
	PropuestaCoberturaViable      EstadoPropuestaDecisionCobertura = "viable"
	PropuestaCoberturaIncompleta  EstadoPropuestaDecisionCobertura = "incompleta"
	PropuestaCoberturaConflictiva EstadoPropuestaDecisionCobertura = "conflictiva"
	PropuestaCoberturaSinVia      EstadoPropuestaDecisionCobertura = "sin_via"
)

func (e EstadoPropuestaDecisionCobertura) valido() bool {
	return e == PropuestaCoberturaViable ||
		e == PropuestaCoberturaIncompleta ||
		e == PropuestaCoberturaConflictiva ||
		e == PropuestaCoberturaSinVia
}

type EstadoEvaluacionViaCobertura string

const (
	EvaluacionViaCoberturaViable      EstadoEvaluacionViaCobertura = "viable"
	EvaluacionViaCoberturaIncompleta  EstadoEvaluacionViaCobertura = "incompleta"
	EvaluacionViaCoberturaConflictiva EstadoEvaluacionViaCobertura = "conflictiva"
	EvaluacionViaCoberturaNoViable    EstadoEvaluacionViaCobertura = "no_viable"
)

// EvidenciaComprobacionPropuestaCobertura conserva únicamente hechos
// minimizados. El detalle libre de ComprobacionCobertura nunca se copia.
type EvidenciaComprobacionPropuestaCobertura struct {
	Resultado  ResultadoComprobacion `json:"resultado"`
	FuenteRef  string                `json:"fuente_ref"`
	ReciboRef  string                `json:"recibo_ref"`
	EvaluadaEn time.Time             `json:"evaluada_en"`
}

func (e EvidenciaComprobacionPropuestaCobertura) validar() error {
	if !e.Resultado.valido() || !referenciaValida(e.FuenteRef) ||
		!referenciaValida(e.ReciboRef) || !instanteCanonico(e.EvaluadaEn) {
		return ErrDatoInvalido
	}
	return nil
}

// ResultadoAgrupadoPropuestaCobertura deduplica una clave. Varias evidencias
// solo son contradictorias cuando declaran resultados funcionales distintos.
type ResultadoAgrupadoPropuestaCobertura struct {
	Clave      ClaveCatalogo                             `json:"clave"`
	Evidencias []EvidenciaComprobacionPropuestaCobertura `json:"evidencias"`
}

func (r ResultadoAgrupadoPropuestaCobertura) clonar() ResultadoAgrupadoPropuestaCobertura {
	r.Evidencias = append(
		[]EvidenciaComprobacionPropuestaCobertura(nil),
		r.Evidencias...,
	)
	return r
}

func (r ResultadoAgrupadoPropuestaCobertura) conflictivo() bool {
	if len(r.Evidencias) < 2 {
		return false
	}
	primero := r.Evidencias[0].Resultado
	for _, evidencia := range r.Evidencias[1:] {
		if evidencia.Resultado != primero {
			return true
		}
	}
	return false
}

type EvaluacionViaPropuestaCobertura struct {
	ViaClave             ClaveCatalogo                `json:"via_clave"`
	Prioridad            uint16                       `json:"prioridad"`
	Estado               EstadoEvaluacionViaCobertura `json:"estado"`
	ResultadosOmitidos   []ClaveCatalogo              `json:"resultados_omitidos,omitempty"`
	AusenciasBloqueantes []ClaveCatalogo              `json:"ausencias_bloqueantes,omitempty"`
	AusenciasAdmitidas   []ClaveCatalogo              `json:"ausencias_admitidas,omitempty"`
	NoHabilitantes       []ClaveCatalogo              `json:"no_habilitantes,omitempty"`
	Conflictos           []ClaveCatalogo              `json:"conflictos,omitempty"`
}

func (e EvaluacionViaPropuestaCobertura) clonar() EvaluacionViaPropuestaCobertura {
	e.ResultadosOmitidos = append(
		[]ClaveCatalogo(nil), e.ResultadosOmitidos...,
	)
	e.AusenciasBloqueantes = append(
		[]ClaveCatalogo(nil), e.AusenciasBloqueantes...,
	)
	e.AusenciasAdmitidas = append(
		[]ClaveCatalogo(nil), e.AusenciasAdmitidas...,
	)
	e.NoHabilitantes = append(
		[]ClaveCatalogo(nil), e.NoHabilitantes...,
	)
	e.Conflictos = append([]ClaveCatalogo(nil), e.Conflictos...)
	return e
}

type CanonHuellaPropuestaDecisionCobertura struct {
	Dominio        string `json:"dominio"`
	VersionEsquema uint16 `json:"version_esquema"`
	Algoritmo      string `json:"algoritmo"`
}

func CanonHuellaPropuestaDecisionCoberturaV1() CanonHuellaPropuestaDecisionCobertura {
	return CanonHuellaPropuestaDecisionCobertura{
		Dominio:        "vec.dipgra.contratacion-temporal.propuesta-decision-cobertura",
		VersionEsquema: 1,
		Algoritmo:      "sha-256",
	}
}

func (c CanonHuellaPropuestaDecisionCobertura) valido() bool {
	return c == CanonHuellaPropuestaDecisionCoberturaV1()
}

type DatosCrearPropuestaDecisionCobertura struct {
	OrganizacionRef                   string
	ExpedienteRef                     string
	VersionExpediente                 uint64
	AnalisisRef                       string
	AnalisisHuellaSHA256              string
	PreparacionEvidenciasRef          string
	PreparacionEvidenciasHuellaSHA256 string
	Catalogo                          CatalogoViasCobertura
	Politica                          PoliticaDecisionCobertura
	FinalidadClave                    ClaveCatalogo
	FinalidadRef                      string
	CategoriaRef                      string
	Periodo                           PeriodoPrevisto
	GeneradaEn                        time.Time
	ValidaHasta                       time.Time
	Resultados                        []ComprobacionCobertura
}

type PublicacionPropuestaDecisionCobertura struct {
	Referencia                        string                                `json:"referencia"`
	HuellaSHA256                      string                                `json:"huella_sha256"`
	Canon                             CanonHuellaPropuestaDecisionCobertura `json:"canon"`
	OrganizacionRef                   string                                `json:"organizacion_ref"`
	ExpedienteRef                     string                                `json:"expediente_ref"`
	VersionExpediente                 uint64                                `json:"version_expediente"`
	AnalisisRef                       string                                `json:"analisis_ref"`
	AnalisisHuellaSHA256              string                                `json:"analisis_huella_sha256"`
	PreparacionEvidenciasRef          string                                `json:"preparacion_evidencias_ref"`
	PreparacionEvidenciasHuellaSHA256 string                                `json:"preparacion_evidencias_huella_sha256"`
	Catalogo                          IdentidadCatalogoViasCobertura        `json:"catalogo"`
	Politica                          IdentidadPoliticaDecisionCobertura    `json:"politica"`
	FinalidadClave                    ClaveCatalogo                         `json:"finalidad_clave"`
	FinalidadRef                      string                                `json:"finalidad_ref"`
	CategoriaRef                      string                                `json:"categoria_ref"`
	Periodo                           PeriodoPrevisto                       `json:"periodo"`
	GeneradaEn                        time.Time                             `json:"generada_en"`
	ValidaHasta                       time.Time                             `json:"valida_hasta"`
	Estado                            EstadoPropuestaDecisionCobertura      `json:"estado"`
	ViaPropuesta                      ClaveCatalogo                         `json:"via_propuesta,omitempty"`
	Resultados                        []ResultadoAgrupadoPropuestaCobertura `json:"resultados"`
	Evaluaciones                      []EvaluacionViaPropuestaCobertura     `json:"evaluaciones"`
}

// PropuestaDecisionCobertura es una recomendación explicable sin efecto
// jurídico. Solo una operación posterior, autorizada y durable, podrá decidir.
type PropuestaDecisionCobertura struct {
	publicacion PublicacionPropuestaDecisionCobertura
	catalogo    CatalogoViasCobertura
	politica    PoliticaDecisionCobertura
}

func CrearPropuestaDecisionCobertura(
	datos DatosCrearPropuestaDecisionCobertura,
) (PropuestaDecisionCobertura, error) {
	if !referenciaValida(datos.OrganizacionRef) ||
		!referenciaValida(datos.ExpedienteRef) ||
		datos.VersionExpediente == 0 ||
		datos.VersionExpediente > maximoEnteroSeguroCatalogoCobertura ||
		!referenciaValida(datos.AnalisisRef) ||
		!huellaValida(datos.AnalisisHuellaSHA256) ||
		!referenciaValida(datos.PreparacionEvidenciasRef) ||
		!huellaValida(datos.PreparacionEvidenciasHuellaSHA256) ||
		!datos.FinalidadClave.Valida() ||
		!referenciaValida(datos.FinalidadRef) ||
		!referenciaValida(datos.CategoriaRef) ||
		!periodoAnalisisValido(datos.Periodo) ||
		!instanteCanonico(datos.GeneradaEn) ||
		!instanteCanonico(datos.ValidaHasta) ||
		!datos.ValidaHasta.After(datos.GeneradaEn) ||
		datos.Catalogo.Validar() != nil ||
		!datos.Catalogo.VigenteEn(datos.GeneradaEn) ||
		datos.Politica.ValidarPara(
			datos.Catalogo,
			datos.OrganizacionRef,
			datos.FinalidadClave,
			datos.FinalidadRef,
			datos.GeneradaEn,
		) != nil ||
		!intervaloPropuestaDentroVigencias(
			datos.GeneradaEn,
			datos.ValidaHasta,
			datos.Catalogo.Vigencia(),
			datos.Politica.Vigencia(),
		) ||
		datos.GeneradaEn.Before(datos.Politica.PublicadaEn()) ||
		len(datos.Resultados) >
			maximoResultadosEntradaPropuestaCobertura {
		return PropuestaDecisionCobertura{}, ErrDatoInvalido
	}
	resultados, err := agruparResultadosPropuestaCobertura(
		datos.Resultados,
		datos.Catalogo,
		datos.GeneradaEn,
	)
	if err != nil {
		return PropuestaDecisionCobertura{}, err
	}
	evaluaciones, estado, via := evaluarPropuestaCobertura(
		datos.Politica.Vias(),
		resultados,
	)
	publicacion := PublicacionPropuestaDecisionCobertura{
		Canon:                             CanonHuellaPropuestaDecisionCoberturaV1(),
		OrganizacionRef:                   datos.OrganizacionRef,
		ExpedienteRef:                     datos.ExpedienteRef,
		VersionExpediente:                 datos.VersionExpediente,
		AnalisisRef:                       datos.AnalisisRef,
		AnalisisHuellaSHA256:              datos.AnalisisHuellaSHA256,
		PreparacionEvidenciasRef:          datos.PreparacionEvidenciasRef,
		PreparacionEvidenciasHuellaSHA256: datos.PreparacionEvidenciasHuellaSHA256,
		Catalogo:                          datos.Catalogo.Identidad(),
		Politica:                          datos.Politica.Identidad(),
		FinalidadClave:                    datos.FinalidadClave,
		FinalidadRef:                      datos.FinalidadRef,
		CategoriaRef:                      datos.CategoriaRef,
		Periodo:                           datos.Periodo,
		GeneradaEn:                        datos.GeneradaEn,
		ValidaHasta:                       datos.ValidaHasta,
		Estado:                            estado,
		ViaPropuesta:                      via,
		Resultados:                        resultados,
		Evaluaciones:                      evaluaciones,
	}
	publicacion.HuellaSHA256, err =
		calcularHuellaPropuestaDecisionCobertura(publicacion)
	if err != nil {
		return PropuestaDecisionCobertura{}, ErrDatoInvalido
	}
	publicacion.Referencia = referenciaPropuestaDecisionCobertura(
		publicacion.HuellaSHA256,
	)
	return PropuestaDecisionCobertura{
		publicacion: publicacion,
		catalogo:    datos.Catalogo,
		politica:    datos.Politica,
	}, nil
}

func intervaloPropuestaDentroVigencias(
	desde time.Time,
	hasta time.Time,
	catalogo VigenciaCatalogoCobertura,
	politica VigenciaCatalogoCobertura,
) bool {
	if !instanteCanonico(desde) || !instanteCanonico(hasta) ||
		!hasta.After(desde) || !catalogo.contiene(desde) ||
		!politica.contiene(desde) {
		return false
	}
	if !catalogo.Hasta.IsZero() && hasta.After(catalogo.Hasta) {
		return false
	}
	return !politica.Hasta.IsZero() && !hasta.After(politica.Hasta)
}

func RestaurarPropuestaDecisionCobertura(
	publicacion PublicacionPropuestaDecisionCobertura,
	catalogo CatalogoViasCobertura,
	politica PoliticaDecisionCobertura,
) (PropuestaDecisionCobertura, error) {
	huellaDeclarada, errHuella :=
		calcularHuellaPropuestaDecisionCobertura(publicacion)
	if !publicacion.Canon.valido() ||
		!huellaCatalogoValida(publicacion.HuellaSHA256) ||
		errHuella != nil ||
		huellaDeclarada != publicacion.HuellaSHA256 ||
		publicacion.Referencia != referenciaPropuestaDecisionCobertura(
			publicacion.HuellaSHA256,
		) ||
		!publicacion.Catalogo.CoincideExactamente(catalogo.Identidad()) ||
		!publicacion.Politica.coincide(politica.Identidad()) {
		return PropuestaDecisionCobertura{}, ErrDatoInvalido
	}
	resultados := desagruparResultadosPropuestaCobertura(publicacion.Resultados)
	restaurada, err := CrearPropuestaDecisionCobertura(
		DatosCrearPropuestaDecisionCobertura{
			OrganizacionRef:                   publicacion.OrganizacionRef,
			ExpedienteRef:                     publicacion.ExpedienteRef,
			VersionExpediente:                 publicacion.VersionExpediente,
			AnalisisRef:                       publicacion.AnalisisRef,
			AnalisisHuellaSHA256:              publicacion.AnalisisHuellaSHA256,
			PreparacionEvidenciasRef:          publicacion.PreparacionEvidenciasRef,
			PreparacionEvidenciasHuellaSHA256: publicacion.PreparacionEvidenciasHuellaSHA256,
			Catalogo:                          catalogo,
			Politica:                          politica,
			FinalidadClave:                    publicacion.FinalidadClave,
			FinalidadRef:                      publicacion.FinalidadRef,
			CategoriaRef:                      publicacion.CategoriaRef,
			Periodo:                           publicacion.Periodo,
			GeneradaEn:                        publicacion.GeneradaEn,
			ValidaHasta:                       publicacion.ValidaHasta,
			Resultados:                        resultados,
		},
	)
	if err != nil ||
		restaurada.publicacion.HuellaSHA256 != publicacion.HuellaSHA256 {
		return PropuestaDecisionCobertura{}, ErrDatoInvalido
	}
	return restaurada, nil
}

func (p PropuestaDecisionCobertura) ValidarPara(
	catalogo CatalogoViasCobertura,
	politica PoliticaDecisionCobertura,
) error {
	_, err := RestaurarPropuestaDecisionCobertura(
		p.Publicacion(), catalogo, politica,
	)
	return err
}

func (p PropuestaDecisionCobertura) Estado() EstadoPropuestaDecisionCobertura {
	return p.publicacion.Estado
}

func (p PropuestaDecisionCobertura) Referencia() string {
	return p.publicacion.Referencia
}

func (p PropuestaDecisionCobertura) HuellaSHA256() string {
	return p.publicacion.HuellaSHA256
}

func (p PropuestaDecisionCobertura) ViaPropuesta() ClaveCatalogo {
	return p.publicacion.ViaPropuesta
}

func (p PropuestaDecisionCobertura) Evaluaciones() []EvaluacionViaPropuestaCobertura {
	return clonarEvaluacionesViaPropuesta(p.publicacion.Evaluaciones)
}

func (p PropuestaDecisionCobertura) Publicacion() PublicacionPropuestaDecisionCobertura {
	publicacion := p.publicacion
	publicacion.Resultados = clonarResultadosAgrupadosPropuesta(
		publicacion.Resultados,
	)
	publicacion.Evaluaciones = clonarEvaluacionesViaPropuesta(
		publicacion.Evaluaciones,
	)
	return publicacion
}

// VinculoDecisionPropuestaCobertura aporta a O4-03B las coordenadas exactas y
// la evidencia minimizada que deberá sellar una decisión posterior.
type VinculoDecisionPropuestaCobertura struct {
	PropuestaRef                      string                                `json:"propuesta_ref"`
	PropuestaHuella                   string                                `json:"propuesta_huella_sha256"`
	AnalisisRef                       string                                `json:"analisis_ref"`
	AnalisisHuellaSHA256              string                                `json:"analisis_huella_sha256"`
	PreparacionEvidenciasRef          string                                `json:"preparacion_evidencias_ref"`
	PreparacionEvidenciasHuellaSHA256 string                                `json:"preparacion_evidencias_huella_sha256"`
	Catalogo                          IdentidadCatalogoViasCobertura        `json:"catalogo"`
	Politica                          IdentidadPoliticaDecisionCobertura    `json:"politica"`
	ViaClave                          ClaveCatalogo                         `json:"via_clave"`
	ValidaHasta                       time.Time                             `json:"valida_hasta"`
	Resultados                        []ResultadoAgrupadoPropuestaCobertura `json:"resultados"`
}

func (p PropuestaDecisionCobertura) VinculoParaDecision(
	via ClaveCatalogo,
	instante time.Time,
) (VinculoDecisionPropuestaCobertura, error) {
	if p.publicacion.Estado != PropuestaCoberturaViable ||
		!via.Valida() || !instanteCanonico(instante) ||
		instante.Before(p.publicacion.GeneradaEn) ||
		!instante.Before(p.publicacion.ValidaHasta) ||
		p.ValidarPara(p.catalogo, p.politica) != nil ||
		!p.catalogo.VigenteEn(instante) ||
		p.politica.ValidarPara(
			p.catalogo,
			p.publicacion.OrganizacionRef,
			p.publicacion.FinalidadClave,
			p.publicacion.FinalidadRef,
			instante,
		) != nil {
		return VinculoDecisionPropuestaCobertura{}, ErrTransicionInvalida
	}
	for _, evaluacion := range p.publicacion.Evaluaciones {
		if evaluacion.ViaClave == via &&
			evaluacion.Estado == EvaluacionViaCoberturaViable {
			return VinculoDecisionPropuestaCobertura{
				PropuestaRef:                      p.publicacion.Referencia,
				PropuestaHuella:                   p.publicacion.HuellaSHA256,
				AnalisisRef:                       p.publicacion.AnalisisRef,
				AnalisisHuellaSHA256:              p.publicacion.AnalisisHuellaSHA256,
				PreparacionEvidenciasRef:          p.publicacion.PreparacionEvidenciasRef,
				PreparacionEvidenciasHuellaSHA256: p.publicacion.PreparacionEvidenciasHuellaSHA256,
				Catalogo:                          p.publicacion.Catalogo,
				Politica:                          p.publicacion.Politica,
				ViaClave:                          via,
				ValidaHasta:                       p.publicacion.ValidaHasta,
				Resultados: clonarResultadosAgrupadosPropuesta(
					p.publicacion.Resultados,
				),
			}, nil
		}
	}
	return VinculoDecisionPropuestaCobertura{}, ErrTransicionInvalida
}

func clonarResultadosAgrupadosPropuesta(
	resultados []ResultadoAgrupadoPropuestaCobertura,
) []ResultadoAgrupadoPropuestaCobertura {
	clon := make([]ResultadoAgrupadoPropuestaCobertura, len(resultados))
	for indice := range resultados {
		clon[indice] = resultados[indice].clonar()
	}
	return clon
}

func clonarEvaluacionesViaPropuesta(
	evaluaciones []EvaluacionViaPropuestaCobertura,
) []EvaluacionViaPropuestaCobertura {
	clon := make([]EvaluacionViaPropuestaCobertura, len(evaluaciones))
	for indice := range evaluaciones {
		clon[indice] = evaluaciones[indice].clonar()
	}
	return clon
}

func referenciaPropuestaDecisionCobertura(huella string) string {
	return "propuesta-cobertura:sha256:" + huella
}

func ordenarClavesCatalogo(claves []ClaveCatalogo) {
	sort.Slice(claves, func(i, j int) bool { return claves[i] < claves[j] })
}
