package domain

// CanonHuellaSemanticaPropuestaDecisionCobertura identifica un canon separado
// del canon exacto de la propuesta. Su evolución no altera el canon V1
// publicado que liga evidencias, preparación y vigencia.
type CanonHuellaSemanticaPropuestaDecisionCobertura struct {
	Dominio        string `json:"dominio"`
	VersionEsquema uint16 `json:"version_esquema"`
	Algoritmo      string `json:"algoritmo"`
}

func CanonHuellaSemanticaPropuestaDecisionCoberturaV1() CanonHuellaSemanticaPropuestaDecisionCobertura {
	return CanonHuellaSemanticaPropuestaDecisionCobertura{
		Dominio: "vec.dipgra.contratacion-temporal." +
			"propuesta-decision-cobertura-semantica",
		VersionEsquema: 1,
		Algoritmo:      "sha-256",
	}
}

func (c CanonHuellaSemanticaPropuestaDecisionCobertura) valido() bool {
	return c == CanonHuellaSemanticaPropuestaDecisionCoberturaV1()
}

// IdentidadSemanticaPropuestaDecisionCobertura sirve para preview y
// reconfirmación de lo que ve y elige una persona. No es evidencia, no concede
// autorización y no sustituye la propuesta exacta fresca que debe ligar la
// decisión final.
type IdentidadSemanticaPropuestaDecisionCobertura struct {
	Referencia   string                                         `json:"referencia"`
	HuellaSHA256 string                                         `json:"huella_sha256"`
	Canon        CanonHuellaSemanticaPropuestaDecisionCobertura `json:"canon"`
}

// Validar comprueba únicamente forma e integridad nominal de la identidad. No
// demuestra vigencia, evidencia ni autorización.
func (i IdentidadSemanticaPropuestaDecisionCobertura) Validar() error {
	if !i.Canon.valido() || !huellaValida(i.HuellaSHA256) ||
		i.Referencia != referenciaSemanticaPropuestaDecisionCobertura(
			i.HuellaSHA256,
		) {
		return ErrDatoInvalido
	}
	return nil
}

// CoincideExactamente exige dos identidades nominalmente válidas e idénticas.
// C3 debe comparar siempre contra la identidad recién recalculada desde la
// propuesta exacta fresca; este método no sustituye esa recalculación.
func (i IdentidadSemanticaPropuestaDecisionCobertura) CoincideExactamente(
	otra IdentidadSemanticaPropuestaDecisionCobertura,
) bool {
	return i.Validar() == nil && otra.Validar() == nil && i == otra
}

// IdentidadSemantica excluye deliberadamente la identidad de preparación y
// los metadatos de cada evidencia, generación y caducidad. Incluye resultados
// funcionales, evaluaciones y recomendación presentados para decidir.
func (p PropuestaDecisionCobertura) IdentidadSemantica() (
	IdentidadSemanticaPropuestaDecisionCobertura,
	error,
) {
	if p.ValidarPara(p.catalogo, p.politica) != nil {
		return IdentidadSemanticaPropuestaDecisionCobertura{}, ErrDatoInvalido
	}
	canon := CanonHuellaSemanticaPropuestaDecisionCoberturaV1()
	huella, err := calcularHuellaSemanticaPropuestaDecisionCobertura(
		canon,
		p.publicacion,
	)
	if err != nil {
		return IdentidadSemanticaPropuestaDecisionCobertura{}, ErrDatoInvalido
	}
	return IdentidadSemanticaPropuestaDecisionCobertura{
		Referencia:   referenciaSemanticaPropuestaDecisionCobertura(huella),
		HuellaSHA256: huella,
		Canon:        canon,
	}, nil
}

func referenciaSemanticaPropuestaDecisionCobertura(huella string) string {
	return "propuesta-cobertura-semantica:sha256:" + huella
}
