package cobertura

import (
	"fmt"
	"io"
	"log/slog"
	"sort"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	redaccionDatosConjuntosVias = "[DATOS-CONJUNTOS-VIAS-COBERTURA-REDACTADOS]"
	redaccionConjuntosVias      = "[PREPARACION-CONJUNTOS-VIAS-COBERTURA-REDACTADA]"
)

// DatosPrepararConjuntosViasCobertura contiene exclusivamente autoridad
// resuelta en servidor. Un cliente no debe poder construir estas coordenadas.
type DatosPrepararConjuntosViasCobertura struct {
	AnalisisRef          string
	AnalisisHuellaSHA256 string
	Catalogo             domain.CatalogoViasCobertura
	Politica             domain.PoliticaDecisionCobertura
	Conjuntos            []ConjuntoEvidenciasCobertura
	PreparadaEn          time.Time
}

// PreparacionConjuntosViasCobertura es una capacidad opaca y sin consumo.
// O4-04 volverá a revalidarla y consumirá todas las órdenes en su COMMIT.
type PreparacionConjuntosViasCobertura struct {
	analisisRef          string
	analisisHuellaSHA256 string
	catalogoPublicado    domain.PublicacionCatalogoViasCobertura
	politicaPublicada    domain.PublicacionPoliticaDecisionCobertura
	conjuntos            []ConjuntoEvidenciasCobertura
	preparadaEn          time.Time
	validaHasta          time.Time
	referencia           string
	huella               string
}

type conjuntoViaOrdenado struct {
	via       domain.ClaveCatalogo
	prioridad uint16
	conjunto  ConjuntoEvidenciasCobertura
	huella    string
}

type estadoConjuntosVias struct {
	catalogo      domain.CatalogoViasCobertura
	politica      domain.PoliticaDecisionCobertura
	coordenadas   CoordenadasConjuntoEvidencias
	conjuntos     []conjuntoViaOrdenado
	resultados    []domain.ComprobacionCobertura
	ordenes       []ports.OrdenConsumoCobertura
	validaHasta   time.Time
	materialCanon []parHuellaConjuntoVia
}

// PrepararConjuntosViasCobertura exige exactamente un conjunto por cada vía
// publicada. La operación valida y sella; nunca consume evidencia.
func PrepararConjuntosViasCobertura(
	datos DatosPrepararConjuntosViasCobertura,
) (PreparacionConjuntosViasCobertura, error) {
	if !datosPreparacionConjuntosViasValidos(datos) {
		return PreparacionConjuntosViasCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	catalogo, politica, err := restaurarGobiernoConjuntosVias(
		datos.Catalogo.Publicacion(),
		datos.Politica.Publicacion(),
		datos.PreparadaEn,
	)
	if err != nil {
		return PreparacionConjuntosViasCobertura{}, err
	}
	estado, err := validarConjuntosViasEn(
		datos.AnalisisRef,
		datos.AnalisisHuellaSHA256,
		catalogo,
		politica,
		datos.Conjuntos,
		datos.PreparadaEn,
	)
	if err != nil {
		return PreparacionConjuntosViasCobertura{}, err
	}
	huella, err := calcularHuellaConjuntosVias(
		datos.AnalisisRef,
		datos.AnalisisHuellaSHA256,
		estado.materialCanon,
	)
	if err != nil {
		return PreparacionConjuntosViasCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	return PreparacionConjuntosViasCobertura{
		analisisRef:          datos.AnalisisRef,
		analisisHuellaSHA256: datos.AnalisisHuellaSHA256,
		catalogoPublicado:    catalogo.Publicacion(),
		politicaPublicada:    politica.Publicacion(),
		conjuntos:            clonarConjuntosOrdenados(estado.conjuntos),
		preparadaEn:          datos.PreparadaEn,
		validaHasta:          estado.validaHasta,
		referencia:           referenciaConjuntosVias(huella),
		huella:               huella,
	}, nil
}

func (p PreparacionConjuntosViasCobertura) Referencia() (string, error) {
	if !domain.ReferenciaOpacaValida(p.referencia) ||
		!huellaValida(p.huella) {
		return "", ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	if _, err := p.validarEn(p.preparadaEn); err != nil {
		return "", err
	}
	return p.referencia, nil
}

func (p PreparacionConjuntosViasCobertura) HuellaSHA256() (string, error) {
	if _, err := p.Referencia(); err != nil {
		return "", err
	}
	return p.huella, nil
}

func (p PreparacionConjuntosViasCobertura) ValidaHasta() (time.Time, error) {
	if !domain.InstanteUTCCanonico(p.validaHasta) ||
		!p.validaHasta.After(p.preparadaEn) {
		return time.Time{}, ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	if _, err := p.validarEn(p.preparadaEn); err != nil {
		return time.Time{}, err
	}
	return p.validaHasta, nil
}

// DatosCrearPropuestaEn proyecta los hechos minimizados exactos. El llamador
// no completa catálogo, política, análisis, preparación, vigencia ni ámbito.
func (p PreparacionConjuntosViasCobertura) DatosCrearPropuestaEn(
	instante time.Time,
) (domain.DatosCrearPropuestaDecisionCobertura, error) {
	estado, err := p.validarEn(instante)
	if err != nil {
		return domain.DatosCrearPropuestaDecisionCobertura{}, err
	}
	return domain.DatosCrearPropuestaDecisionCobertura{
		OrganizacionRef:                   estado.coordenadas.OrganizacionRef,
		ExpedienteRef:                     estado.coordenadas.ExpedienteRef,
		VersionExpediente:                 estado.coordenadas.VersionExpediente,
		AnalisisRef:                       p.analisisRef,
		AnalisisHuellaSHA256:              p.analisisHuellaSHA256,
		PreparacionEvidenciasRef:          p.referencia,
		PreparacionEvidenciasHuellaSHA256: p.huella,
		Catalogo:                          estado.catalogo,
		Politica:                          estado.politica,
		FinalidadClave:                    estado.coordenadas.FinalidadClave,
		FinalidadRef:                      estado.coordenadas.FinalidadRef,
		CategoriaRef:                      estado.coordenadas.CategoriaRef,
		Periodo:                           estado.coordenadas.Periodo,
		GeneradaEn:                        instante,
		ValidaHasta:                       estado.validaHasta,
		Resultados:                        clonarComprobaciones(estado.resultados),
	}, nil
}

// OrdenesPendientesEn entrega todas las órdenes en prioridad de política, vía
// y orden de comprobación. No confirma ni consume ninguna.
func (p PreparacionConjuntosViasCobertura) OrdenesPendientesEn(
	instante time.Time,
) ([]ports.OrdenConsumoCobertura, error) {
	estado, err := p.validarEn(instante)
	if err != nil {
		return nil, err
	}
	return append([]ports.OrdenConsumoCobertura(nil), estado.ordenes...), nil
}

func (p PreparacionConjuntosViasCobertura) validarEn(
	instante time.Time,
) (estadoConjuntosVias, error) {
	if !domain.InstanteUTCCanonico(p.preparadaEn) ||
		!domain.InstanteUTCCanonico(instante) ||
		instante.Before(p.preparadaEn) ||
		!instante.Before(p.validaHasta) ||
		!domain.ReferenciaOpacaValida(p.analisisRef) ||
		!huellaValida(p.analisisHuellaSHA256) {
		return estadoConjuntosVias{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	catalogo, politica, err := restaurarGobiernoConjuntosVias(
		p.catalogoPublicado,
		p.politicaPublicada,
		instante,
	)
	if err != nil {
		return estadoConjuntosVias{}, err
	}
	conjuntos := make([]ConjuntoEvidenciasCobertura, len(p.conjuntos))
	for indice := range p.conjuntos {
		conjuntos[indice] = p.conjuntos[indice]
	}
	estado, err := validarConjuntosViasEn(
		p.analisisRef,
		p.analisisHuellaSHA256,
		catalogo,
		politica,
		conjuntos,
		instante,
	)
	if err != nil || !estado.validaHasta.Equal(p.validaHasta) {
		return estadoConjuntosVias{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	huella, err := calcularHuellaConjuntosVias(
		p.analisisRef,
		p.analisisHuellaSHA256,
		estado.materialCanon,
	)
	if err != nil || huella != p.huella ||
		referenciaConjuntosVias(huella) != p.referencia {
		return estadoConjuntosVias{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	return estado, nil
}

func datosPreparacionConjuntosViasValidos(
	datos DatosPrepararConjuntosViasCobertura,
) bool {
	return domain.ReferenciaOpacaValida(datos.AnalisisRef) &&
		huellaValida(datos.AnalisisHuellaSHA256) &&
		domain.InstanteUTCCanonico(datos.PreparadaEn) &&
		len(datos.Conjuntos) > 0 &&
		len(datos.Conjuntos) <= maximoViasPreparacion
}

func restaurarGobiernoConjuntosVias(
	publicacionCatalogo domain.PublicacionCatalogoViasCobertura,
	publicacionPolitica domain.PublicacionPoliticaDecisionCobertura,
	instante time.Time,
) (
	domain.CatalogoViasCobertura,
	domain.PoliticaDecisionCobertura,
	error,
) {
	catalogo, errCatalogo := domain.RestaurarCatalogoViasCobertura(
		publicacionCatalogo,
	)
	politica, errPolitica := domain.RestaurarPoliticaDecisionCobertura(
		publicacionPolitica,
		catalogo,
	)
	finalidadClave, finalidadRef := politica.Finalidad()
	if errCatalogo != nil || errPolitica != nil ||
		!catalogo.VigenteEn(instante) ||
		politica.ValidarPara(
			catalogo,
			publicacionPolitica.OrganizacionRef,
			finalidadClave,
			finalidadRef,
			instante,
		) != nil {
		return domain.CatalogoViasCobertura{},
			domain.PoliticaDecisionCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	return catalogo, politica, nil
}

func validarConjuntosViasEn(
	analisisRef string,
	analisisHuella string,
	catalogo domain.CatalogoViasCobertura,
	politica domain.PoliticaDecisionCobertura,
	conjuntos []ConjuntoEvidenciasCobertura,
	instante time.Time,
) (estadoConjuntosVias, error) {
	if !domain.ReferenciaOpacaValida(analisisRef) ||
		!huellaValida(analisisHuella) ||
		len(conjuntos) != len(catalogo.Vias()) ||
		len(conjuntos) == 0 || len(conjuntos) > maximoViasPreparacion {
		return estadoConjuntosVias{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	prioridades := prioridadesPolitica(politica)
	ordenados := make([]conjuntoViaOrdenado, 0, len(conjuntos))
	vistas := make(map[domain.ClaveCatalogo]struct{}, len(conjuntos))
	var comunes CoordenadasConjuntoEvidencias
	for indice, conjunto := range conjuntos {
		coordenadas, err := conjunto.Coordenadas()
		prioridad, publicada := prioridades[coordenadas.ViaClave]
		_, duplicada := vistas[coordenadas.ViaClave]
		if err != nil || !publicada || duplicada ||
			!coordenadas.Catalogo.CoincideExactamente(
				catalogo.Identidad(),
			) ||
			coordenadas.Politica != politica.Identidad() {
			return estadoConjuntosVias{},
				ports.ErrResultadoFuenteCoberturaNoConfiable
		}
		if indice == 0 {
			comunes = coordenadas
		} else if !coordenadasComunesCoinciden(comunes, coordenadas) {
			return estadoConjuntosVias{},
				ports.ErrResultadoFuenteCoberturaNoConfiable
		}
		huella, err := conjunto.HuellaSHA256()
		if err != nil {
			return estadoConjuntosVias{},
				ports.ErrResultadoFuenteCoberturaNoConfiable
		}
		vistas[coordenadas.ViaClave] = struct{}{}
		ordenados = append(ordenados, conjuntoViaOrdenado{
			via: coordenadas.ViaClave, prioridad: prioridad,
			conjunto: conjunto, huella: huella,
		})
	}
	if !contieneExactamenteVias(vistas, catalogo.Vias()) {
		return estadoConjuntosVias{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	sort.Slice(ordenados, func(i, j int) bool {
		if ordenados[i].prioridad != ordenados[j].prioridad {
			return ordenados[i].prioridad < ordenados[j].prioridad
		}
		return ordenados[i].via < ordenados[j].via
	})
	return proyectarConjuntosVias(
		catalogo, politica, comunes, ordenados, instante,
	)
}

func prioridadesPolitica(
	politica domain.PoliticaDecisionCobertura,
) map[domain.ClaveCatalogo]uint16 {
	reglas := politica.Vias()
	prioridades := make(map[domain.ClaveCatalogo]uint16, len(reglas))
	for _, regla := range reglas {
		prioridades[regla.ViaClave] = regla.Prioridad
	}
	return prioridades
}

func contieneExactamenteVias(
	vistas map[domain.ClaveCatalogo]struct{},
	vias []domain.DefinicionViaCobertura,
) bool {
	if len(vistas) != len(vias) {
		return false
	}
	for _, via := range vias {
		if _, existe := vistas[via.Clave]; !existe {
			return false
		}
	}
	return true
}

func coordenadasComunesCoinciden(
	primera CoordenadasConjuntoEvidencias,
	otra CoordenadasConjuntoEvidencias,
) bool {
	return primera.OrganizacionRef == otra.OrganizacionRef &&
		primera.ExpedienteRef == otra.ExpedienteRef &&
		primera.VersionExpediente == otra.VersionExpediente &&
		primera.Catalogo.CoincideExactamente(otra.Catalogo) &&
		primera.Politica == otra.Politica &&
		primera.FinalidadClave == otra.FinalidadClave &&
		primera.FinalidadRef == otra.FinalidadRef &&
		primera.CategoriaRef == otra.CategoriaRef &&
		primera.Periodo.Inicio.Equal(otra.Periodo.Inicio) &&
		primera.Periodo.Fin.Equal(otra.Periodo.Fin)
}

func clonarConjuntosOrdenados(
	origen []conjuntoViaOrdenado,
) []ConjuntoEvidenciasCobertura {
	clon := make([]ConjuntoEvidenciasCobertura, len(origen))
	for indice := range origen {
		clon[indice] = origen[indice].conjunto
	}
	return clon
}

func clonarComprobaciones(
	origen []domain.ComprobacionCobertura,
) []domain.ComprobacionCobertura {
	return append([]domain.ComprobacionCobertura(nil), origen...)
}

func referenciaConjuntosVias(huella string) string {
	return "preparacion-evidencias-cobertura:sha256:" + huella
}

func (DatosPrepararConjuntosViasCobertura) String() string {
	return redaccionDatosConjuntosVias
}
func (d DatosPrepararConjuntosViasCobertura) GoString() string {
	return d.String()
}
func (d DatosPrepararConjuntosViasCobertura) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, d.String())
}
func (d DatosPrepararConjuntosViasCobertura) LogValue() slog.Value {
	return slog.StringValue(d.String())
}
func (d DatosPrepararConjuntosViasCobertura) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}
func (d DatosPrepararConjuntosViasCobertura) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.String() + `"`), nil
}

func (PreparacionConjuntosViasCobertura) String() string {
	return redaccionConjuntosVias
}
func (p PreparacionConjuntosViasCobertura) GoString() string {
	return p.String()
}
func (p PreparacionConjuntosViasCobertura) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, p.String())
}
func (p PreparacionConjuntosViasCobertura) LogValue() slog.Value {
	return slog.StringValue(p.String())
}
func (p PreparacionConjuntosViasCobertura) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}
func (p PreparacionConjuntosViasCobertura) MarshalJSON() ([]byte, error) {
	return []byte(`"` + p.String() + `"`), nil
}
