package domain

import (
	"sort"
	"time"
)

const (
	maximoResultadosHabilitantesCobertura = 3
)

// TratamientoAusenciaCobertura distingue la falta de dato de un resultado
// negativo o no aplicable. Solo una política publicada puede admitirla.
type TratamientoAusenciaCobertura string

const (
	AusenciaCoberturaBloquea  TratamientoAusenciaCobertura = "bloquea"
	AusenciaCoberturaAdmitida TratamientoAusenciaCobertura = "admitida"
)

func (t TratamientoAusenciaCobertura) valido() bool {
	return t == AusenciaCoberturaBloquea ||
		t == AusenciaCoberturaAdmitida
}

// ReglaComprobacionDecisionCobertura declara los resultados que habilitan una
// vía y qué ocurre si la fuente no aporta dato. No contiene nombres de fuentes
// ni lógica específica de Bolsa, SAE o cualquier otra vía.
type ReglaComprobacionDecisionCobertura struct {
	Clave                  ClaveCatalogo                `json:"clave"`
	ResultadosHabilitantes []ResultadoComprobacion      `json:"resultados_habilitantes"`
	TratamientoAusencia    TratamientoAusenciaCobertura `json:"tratamiento_ausencia"`
}

func (r ReglaComprobacionDecisionCobertura) validar() error {
	if !r.Clave.Valida() || !r.TratamientoAusencia.valido() ||
		len(r.ResultadosHabilitantes) == 0 ||
		len(r.ResultadosHabilitantes) >
			maximoResultadosHabilitantesCobertura {
		return ErrDatoInvalido
	}
	vistos := make(map[ResultadoComprobacion]struct{},
		len(r.ResultadosHabilitantes))
	for _, resultado := range r.ResultadosHabilitantes {
		if !resultado.valido() || resultado == ComprobacionNoConsta {
			return ErrDatoInvalido
		}
		if _, repetido := vistos[resultado]; repetido {
			return ErrDatoInvalido
		}
		vistos[resultado] = struct{}{}
	}
	return nil
}

func (r ReglaComprobacionDecisionCobertura) habilita(
	resultado ResultadoComprobacion,
) bool {
	for _, habilitante := range r.ResultadosHabilitantes {
		if habilitante == resultado {
			return true
		}
	}
	return false
}

func (r ReglaComprobacionDecisionCobertura) clonar() ReglaComprobacionDecisionCobertura {
	r.ResultadosHabilitantes = append(
		[]ResultadoComprobacion(nil),
		r.ResultadosHabilitantes...,
	)
	return r
}

// ReglaViaDecisionCobertura aporta una prioridad funcional explícita. No se
// infiere que el orden visual de O4-01 sea una regla de selección.
type ReglaViaDecisionCobertura struct {
	ViaClave       ClaveCatalogo                        `json:"via_clave"`
	Prioridad      uint16                               `json:"prioridad"`
	Comprobaciones []ReglaComprobacionDecisionCobertura `json:"comprobaciones"`
}

func (r ReglaViaDecisionCobertura) clonar() ReglaViaDecisionCobertura {
	clon := r
	clon.Comprobaciones = make(
		[]ReglaComprobacionDecisionCobertura,
		len(r.Comprobaciones),
	)
	for indice := range r.Comprobaciones {
		clon.Comprobaciones[indice] = r.Comprobaciones[indice].clonar()
	}
	return clon
}

// IdentidadPoliticaDecisionCobertura identifica de forma exacta una
// publicación de reglas aplicable a un catálogo O4-01 concreto.
type IdentidadPoliticaDecisionCobertura struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

func (i IdentidadPoliticaDecisionCobertura) Validar() error {
	if !referenciaValida(i.Referencia) || i.Version == 0 ||
		i.Version > maximoEnteroSeguroCatalogoCobertura ||
		!huellaCatalogoValida(i.HuellaSHA256) {
		return ErrDatoInvalido
	}
	return nil
}

func (i IdentidadPoliticaDecisionCobertura) coincide(
	otra IdentidadPoliticaDecisionCobertura,
) bool {
	return i.Validar() == nil && otra.Validar() == nil &&
		i == otra
}

type CanonHuellaPoliticaDecisionCobertura struct {
	Dominio        string `json:"dominio"`
	VersionEsquema uint16 `json:"version_esquema"`
	Algoritmo      string `json:"algoritmo"`
}

func CanonHuellaPoliticaDecisionCoberturaV1() CanonHuellaPoliticaDecisionCobertura {
	return CanonHuellaPoliticaDecisionCobertura{
		Dominio:        "vec.dipgra.contratacion-temporal.politica-decision-cobertura",
		VersionEsquema: 1,
		Algoritmo:      "sha-256",
	}
}

func (c CanonHuellaPoliticaDecisionCobertura) valido() bool {
	return c == CanonHuellaPoliticaDecisionCoberturaV1()
}

type BorradorPoliticaDecisionCobertura struct {
	Referencia      string                         `json:"referencia"`
	Version         uint64                         `json:"version"`
	Catalogo        IdentidadCatalogoViasCobertura `json:"catalogo"`
	OrganizacionRef string                         `json:"organizacion_ref"`
	FinalidadClave  ClaveCatalogo                  `json:"finalidad_clave"`
	FinalidadRef    string                         `json:"finalidad_ref"`
	PublicadaEn     time.Time                      `json:"publicada_en"`
	Vigencia        VigenciaCatalogoCobertura      `json:"vigencia"`
	ProcedenciaRef  string                         `json:"procedencia_ref"`
	Vias            []ReglaViaDecisionCobertura    `json:"vias"`
}

type PublicacionPoliticaDecisionCobertura struct {
	Referencia      string                               `json:"referencia"`
	Version         uint64                               `json:"version"`
	HuellaSHA256    string                               `json:"huella_sha256"`
	Canon           CanonHuellaPoliticaDecisionCobertura `json:"canon"`
	Catalogo        IdentidadCatalogoViasCobertura       `json:"catalogo"`
	OrganizacionRef string                               `json:"organizacion_ref"`
	FinalidadClave  ClaveCatalogo                        `json:"finalidad_clave"`
	FinalidadRef    string                               `json:"finalidad_ref"`
	PublicadaEn     time.Time                            `json:"publicada_en"`
	Vigencia        VigenciaCatalogoCobertura            `json:"vigencia"`
	ProcedenciaRef  string                               `json:"procedencia_ref"`
	Vias            []ReglaViaDecisionCobertura          `json:"vias"`
}

// PoliticaDecisionCobertura es inmutable dentro del proceso. La publicación
// durable y su autorización corresponden a capas posteriores.
type PoliticaDecisionCobertura struct {
	publicacion PublicacionPoliticaDecisionCobertura
}

func PublicarPoliticaDecisionCobertura(
	borrador BorradorPoliticaDecisionCobertura,
	catalogo CatalogoViasCobertura,
) (PoliticaDecisionCobertura, error) {
	normalizado, err := normalizarPoliticaDecisionCobertura(borrador, catalogo)
	if err != nil {
		return PoliticaDecisionCobertura{}, err
	}
	publicacion := PublicacionPoliticaDecisionCobertura{
		Referencia:      normalizado.Referencia,
		Version:         normalizado.Version,
		Canon:           CanonHuellaPoliticaDecisionCoberturaV1(),
		Catalogo:        normalizado.Catalogo,
		OrganizacionRef: normalizado.OrganizacionRef,
		FinalidadClave:  normalizado.FinalidadClave,
		FinalidadRef:    normalizado.FinalidadRef,
		PublicadaEn:     normalizado.PublicadaEn,
		Vigencia:        normalizado.Vigencia,
		ProcedenciaRef:  normalizado.ProcedenciaRef,
		Vias:            normalizado.Vias,
	}
	publicacion.HuellaSHA256, err =
		calcularHuellaPoliticaDecisionCobertura(publicacion)
	if err != nil {
		return PoliticaDecisionCobertura{}, ErrDatoInvalido
	}
	return PoliticaDecisionCobertura{publicacion: publicacion}, nil
}

func RestaurarPoliticaDecisionCobertura(
	publicacion PublicacionPoliticaDecisionCobertura,
	catalogo CatalogoViasCobertura,
) (PoliticaDecisionCobertura, error) {
	if !publicacion.Canon.valido() ||
		!huellaCatalogoValida(publicacion.HuellaSHA256) {
		return PoliticaDecisionCobertura{}, ErrDatoInvalido
	}
	restaurada, err := PublicarPoliticaDecisionCobertura(
		BorradorPoliticaDecisionCobertura{
			Referencia:      publicacion.Referencia,
			Version:         publicacion.Version,
			Catalogo:        publicacion.Catalogo,
			OrganizacionRef: publicacion.OrganizacionRef,
			FinalidadClave:  publicacion.FinalidadClave,
			FinalidadRef:    publicacion.FinalidadRef,
			PublicadaEn:     publicacion.PublicadaEn,
			Vigencia:        publicacion.Vigencia,
			ProcedenciaRef:  publicacion.ProcedenciaRef,
			Vias:            publicacion.Vias,
		},
		catalogo,
	)
	if err != nil ||
		restaurada.publicacion.HuellaSHA256 != publicacion.HuellaSHA256 {
		return PoliticaDecisionCobertura{}, ErrDatoInvalido
	}
	return restaurada, nil
}

func (p PoliticaDecisionCobertura) ValidarPara(
	catalogo CatalogoViasCobertura,
	organizacionRef string,
	finalidadClave ClaveCatalogo,
	finalidadRef string,
	instante time.Time,
) error {
	_, err := RestaurarPoliticaDecisionCobertura(p.Publicacion(), catalogo)
	if err != nil || p.publicacion.OrganizacionRef != organizacionRef ||
		p.publicacion.FinalidadClave != finalidadClave ||
		p.publicacion.FinalidadRef != finalidadRef ||
		!instanteCanonico(instante) ||
		!p.publicacion.Vigencia.contiene(instante) {
		return ErrDatoInvalido
	}
	return nil
}

func (p PoliticaDecisionCobertura) Identidad() IdentidadPoliticaDecisionCobertura {
	return IdentidadPoliticaDecisionCobertura{
		Referencia:   p.publicacion.Referencia,
		Version:      p.publicacion.Version,
		HuellaSHA256: p.publicacion.HuellaSHA256,
	}
}

func (p PoliticaDecisionCobertura) PublicadaEn() time.Time {
	return p.publicacion.PublicadaEn
}

func (p PoliticaDecisionCobertura) Vigencia() VigenciaCatalogoCobertura {
	return p.publicacion.Vigencia
}

func (p PoliticaDecisionCobertura) Finalidad() (
	ClaveCatalogo,
	string,
) {
	return p.publicacion.FinalidadClave, p.publicacion.FinalidadRef
}

func (p PoliticaDecisionCobertura) Vias() []ReglaViaDecisionCobertura {
	return clonarReglasViaDecisionCobertura(p.publicacion.Vias)
}

func (p PoliticaDecisionCobertura) Publicacion() PublicacionPoliticaDecisionCobertura {
	publicacion := p.publicacion
	publicacion.Vias = clonarReglasViaDecisionCobertura(publicacion.Vias)
	return publicacion
}

func normalizarPoliticaDecisionCobertura(
	borrador BorradorPoliticaDecisionCobertura,
	catalogo CatalogoViasCobertura,
) (BorradorPoliticaDecisionCobertura, error) {
	if catalogo.Validar() != nil || !borrador.Catalogo.CoincideExactamente(
		catalogo.Identidad(),
	) || !referenciaValida(borrador.Referencia) ||
		borrador.Version == 0 ||
		borrador.Version > maximoEnteroSeguroCatalogoCobertura ||
		!referenciaValida(borrador.OrganizacionRef) ||
		!borrador.FinalidadClave.Valida() ||
		!referenciaValida(borrador.FinalidadRef) ||
		!instanteCanonico(borrador.PublicadaEn) ||
		borrador.Vigencia.Validar() != nil ||
		borrador.Vigencia.Hasta.IsZero() ||
		borrador.PublicadaEn.After(borrador.Vigencia.Desde) ||
		!vigenciaPoliticaDentroCatalogo(
			borrador.Vigencia,
			catalogo.Vigencia(),
		) ||
		borrador.PublicadaEn.Before(catalogo.PublicadoEn()) ||
		!referenciaValida(borrador.ProcedenciaRef) {
		return BorradorPoliticaDecisionCobertura{}, ErrDatoInvalido
	}
	viasCatalogo := catalogo.Vias()
	if len(borrador.Vias) != len(viasCatalogo) {
		return BorradorPoliticaDecisionCobertura{}, ErrDatoInvalido
	}
	porClave := make(map[ClaveCatalogo]ReglaViaDecisionCobertura,
		len(borrador.Vias))
	prioridades := make(map[uint16]struct{}, len(borrador.Vias))
	for _, via := range borrador.Vias {
		if !via.ViaClave.Valida() || via.Prioridad == 0 {
			return BorradorPoliticaDecisionCobertura{}, ErrDatoInvalido
		}
		if _, existe := porClave[via.ViaClave]; existe {
			return BorradorPoliticaDecisionCobertura{}, ErrDatoInvalido
		}
		if _, existe := prioridades[via.Prioridad]; existe {
			return BorradorPoliticaDecisionCobertura{}, ErrDatoInvalido
		}
		prioridades[via.Prioridad] = struct{}{}
		porClave[via.ViaClave] = via.clonar()
	}
	normalizado := borrador
	normalizado.Vias = make([]ReglaViaDecisionCobertura, 0, len(viasCatalogo))
	for _, viaCatalogo := range viasCatalogo {
		regla, existe := porClave[viaCatalogo.Clave]
		if !existe {
			return BorradorPoliticaDecisionCobertura{}, ErrDatoInvalido
		}
		ordenadas, err := normalizarComprobacionesPolitica(
			regla.Comprobaciones,
			viaCatalogo.Comprobaciones,
		)
		if err != nil {
			return BorradorPoliticaDecisionCobertura{}, err
		}
		regla.Comprobaciones = ordenadas
		normalizado.Vias = append(normalizado.Vias, regla)
	}
	sort.Slice(normalizado.Vias, func(i, j int) bool {
		return normalizado.Vias[i].Prioridad < normalizado.Vias[j].Prioridad
	})
	return normalizado, nil
}

func vigenciaPoliticaDentroCatalogo(
	politica VigenciaCatalogoCobertura,
	catalogo VigenciaCatalogoCobertura,
) bool {
	if politica.Validar() != nil || catalogo.Validar() != nil ||
		politica.Hasta.IsZero() ||
		politica.Desde.Before(catalogo.Desde) {
		return false
	}
	return catalogo.Hasta.IsZero() ||
		!politica.Hasta.After(catalogo.Hasta)
}

func normalizarComprobacionesPolitica(
	reglas []ReglaComprobacionDecisionCobertura,
	comprobaciones []ComprobacionExigibleCobertura,
) ([]ReglaComprobacionDecisionCobertura, error) {
	if len(reglas) != len(comprobaciones) {
		return nil, ErrDatoInvalido
	}
	porClave := make(map[ClaveCatalogo]ReglaComprobacionDecisionCobertura,
		len(reglas))
	for _, regla := range reglas {
		if regla.validar() != nil {
			return nil, ErrDatoInvalido
		}
		if _, existe := porClave[regla.Clave]; existe {
			return nil, ErrDatoInvalido
		}
		sort.Slice(regla.ResultadosHabilitantes, func(i, j int) bool {
			return regla.ResultadosHabilitantes[i] <
				regla.ResultadosHabilitantes[j]
		})
		porClave[regla.Clave] = regla.clonar()
	}
	normalizadas := make([]ReglaComprobacionDecisionCobertura, 0, len(reglas))
	for _, comprobacion := range comprobaciones {
		regla, existe := porClave[comprobacion.Clave]
		if !existe {
			return nil, ErrDatoInvalido
		}
		normalizadas = append(normalizadas, regla)
	}
	return normalizadas, nil
}

func clonarReglasViaDecisionCobertura(
	vias []ReglaViaDecisionCobertura,
) []ReglaViaDecisionCobertura {
	clon := make([]ReglaViaDecisionCobertura, len(vias))
	for indice := range vias {
		clon[indice] = vias[indice].clonar()
	}
	return clon
}
