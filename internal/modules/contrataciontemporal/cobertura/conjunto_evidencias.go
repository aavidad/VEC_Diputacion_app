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
	dominioConjuntoEvidencias = "VEC-CT-CONJUNTO-EVIDENCIAS-COBERTURA-V1"
	maximoEvidenciasEntrada   = 256
	maximoEvidenciasPorVia    = 32
	redaccionConjunto         = "[CONJUNTO-EVIDENCIAS-COBERTURA-REDACTADO]"
	redaccionCoordenadas      = "[COORDENADAS-EVIDENCIAS-COBERTURA-REDACTADAS]"
)

type CoordenadasConjuntoEvidencias struct {
	OrganizacionRef   string
	ExpedienteRef     string
	VersionExpediente uint64
	Catalogo          domain.IdentidadCatalogoViasCobertura
	Politica          domain.IdentidadPoliticaDecisionCobertura
	FinalidadClave    domain.ClaveCatalogo
	FinalidadRef      string
	ViaClave          domain.ClaveCatalogo
	CategoriaRef      string
	Periodo           domain.PeriodoPrevisto
}

func (c CoordenadasConjuntoEvidencias) validar() error {
	const maximoEnteroSeguro = uint64(1<<53 - 1)
	if !domain.ReferenciaOpacaValida(c.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(c.ExpedienteRef) ||
		c.VersionExpediente == 0 ||
		c.VersionExpediente > maximoEnteroSeguro ||
		c.Catalogo.Validar() != nil || c.Politica.Validar() != nil ||
		!c.FinalidadClave.Valida() ||
		!domain.ReferenciaOpacaValida(c.FinalidadRef) ||
		!c.ViaClave.Valida() ||
		!domain.ReferenciaOpacaValida(c.CategoriaRef) ||
		c.Periodo.Validar() != nil {
		return ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	return nil
}

func (CoordenadasConjuntoEvidencias) String() string {
	return redaccionCoordenadas
}
func (c CoordenadasConjuntoEvidencias) GoString() string { return c.String() }
func (c CoordenadasConjuntoEvidencias) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, c.String())
}
func (c CoordenadasConjuntoEvidencias) LogValue() slog.Value {
	return slog.StringValue(c.String())
}
func (c CoordenadasConjuntoEvidencias) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}
func (c CoordenadasConjuntoEvidencias) MarshalJSON() ([]byte, error) {
	return []byte(`"` + c.String() + `"`), nil
}

// ConjuntoEvidenciasCobertura conserva publicaciones completas restaurables.
// Sus órdenes siguen pendientes hasta la transacción que produzca el efecto.
type ConjuntoEvidenciasCobertura struct {
	coordenadas       CoordenadasConjuntoEvidencias
	catalogoPublicado domain.PublicacionCatalogoViasCobertura
	politicaPublicada domain.PublicacionPoliticaDecisionCobertura
	evidencias        []EvidenciaConsultaCobertura
	huella            string
}

func NuevoConjuntoEvidenciasCobertura(
	coordenadas CoordenadasConjuntoEvidencias,
	catalogo domain.CatalogoViasCobertura,
	politica domain.PoliticaDecisionCobertura,
	evidencias []EvidenciaConsultaCobertura,
	comprobadaEn time.Time,
) (ConjuntoEvidenciasCobertura, error) {
	normalizadas, err := validarYOrdenarEvidencias(
		coordenadas, catalogo, politica, evidencias, comprobadaEn,
	)
	if err != nil {
		return ConjuntoEvidenciasCobertura{}, err
	}
	huella, err := calcularHuellaConjunto(coordenadas, normalizadas)
	if err != nil {
		return ConjuntoEvidenciasCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	return ConjuntoEvidenciasCobertura{
		coordenadas:       coordenadas,
		catalogoPublicado: catalogo.Publicacion(),
		politicaPublicada: politica.Publicacion(),
		evidencias:        normalizadas,
		huella:            huella,
	}, nil
}

func (c ConjuntoEvidenciasCobertura) restaurarGobierno(
	comprobadaEn time.Time,
) (
	domain.CatalogoViasCobertura,
	domain.PoliticaDecisionCobertura,
	error,
) {
	catalogo, errCatalogo := domain.RestaurarCatalogoViasCobertura(
		c.catalogoPublicado,
	)
	politica, errPolitica := domain.RestaurarPoliticaDecisionCobertura(
		c.politicaPublicada,
		catalogo,
	)
	if errCatalogo != nil || errPolitica != nil {
		return domain.CatalogoViasCobertura{},
			domain.PoliticaDecisionCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	if !catalogo.Identidad().CoincideExactamente(c.coordenadas.Catalogo) ||
		politica.Identidad() != c.coordenadas.Politica ||
		!catalogo.VigenteEn(comprobadaEn) ||
		politica.ValidarPara(
			catalogo,
			c.coordenadas.OrganizacionRef,
			c.coordenadas.FinalidadClave,
			c.coordenadas.FinalidadRef,
			comprobadaEn,
		) != nil {
		return domain.CatalogoViasCobertura{},
			domain.PoliticaDecisionCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	return catalogo, politica, nil
}

func (c ConjuntoEvidenciasCobertura) Coordenadas() (
	CoordenadasConjuntoEvidencias,
	error,
) {
	if c.coordenadas.validar() != nil ||
		!huellaValida(c.huella) || len(c.evidencias) == 0 {
		return CoordenadasConjuntoEvidencias{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	return c.coordenadas, nil
}

func (c ConjuntoEvidenciasCobertura) HuellaSHA256() (string, error) {
	if _, err := c.Coordenadas(); err != nil {
		return "", err
	}
	return c.huella, nil
}

func (c ConjuntoEvidenciasCobertura) ComprobacionesEn(
	comprobadaEn time.Time,
) (
	[]domain.ComprobacionCobertura,
	error,
) {
	normalizadas, err := c.evidenciasNormalizadasEn(comprobadaEn)
	if err != nil {
		return nil, err
	}
	resultado := make([]domain.ComprobacionCobertura, len(normalizadas))
	for indice, evidencia := range normalizadas {
		comprobacion, err := evidencia.Comprobacion()
		if err != nil {
			return nil, ports.ErrResultadoFuenteCoberturaNoConfiable
		}
		resultado[indice] = comprobacion
	}
	return resultado, nil
}

func (c ConjuntoEvidenciasCobertura) OrdenesPendientesEn(
	comprobadaEn time.Time,
) ([]ports.OrdenConsumoCobertura, error) {
	normalizadas, err := c.evidenciasNormalizadasEn(comprobadaEn)
	if err != nil {
		return nil, err
	}
	ordenes := make([]ports.OrdenConsumoCobertura, len(normalizadas))
	for indice, evidencia := range normalizadas {
		orden, err := evidencia.OrdenPendienteEn(comprobadaEn)
		if err != nil {
			return nil, ports.ErrResultadoFuenteCoberturaNoConfiable
		}
		ordenes[indice] = orden
	}
	return ordenes, nil
}

func (c ConjuntoEvidenciasCobertura) evidenciasNormalizadasEn(
	comprobadaEn time.Time,
) ([]EvidenciaConsultaCobertura, error) {
	catalogo, politica, err := c.restaurarGobierno(comprobadaEn)
	if err != nil {
		return nil, err
	}
	normalizadas, err := validarYOrdenarEvidencias(
		c.coordenadas, catalogo, politica, c.evidencias, comprobadaEn,
	)
	if err != nil {
		return nil, err
	}
	huella, err := calcularHuellaConjunto(c.coordenadas, normalizadas)
	if err != nil || huella != c.huella {
		return nil, ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	return normalizadas, nil
}

type semanticaEvidencia struct {
	clave, resultado, procedencia, definicion string
	orden                                     uint16
	obligatoria                               bool
}

type identificadorPrueba struct {
	autoridad  string
	generacion uint32
	recibo     string
}

func validarYOrdenarEvidencias(
	coordenadas CoordenadasConjuntoEvidencias,
	catalogo domain.CatalogoViasCobertura,
	politica domain.PoliticaDecisionCobertura,
	evidencias []EvidenciaConsultaCobertura,
	comprobadaEn time.Time,
) ([]EvidenciaConsultaCobertura, error) {
	via, reglas, err := validarGobiernoConjunto(
		coordenadas, catalogo, politica, comprobadaEn,
	)
	if err != nil || len(evidencias) == 0 ||
		len(evidencias) > maximoEvidenciasEntrada {
		return nil, ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	definiciones := make(map[domain.ClaveCatalogo]domain.ComprobacionExigibleCobertura)
	for _, definicion := range via.Comprobaciones {
		definiciones[definicion.Clave] = definicion
	}
	representantes := make(map[domain.ClaveCatalogo]EvidenciaConsultaCobertura)
	semanticas := make(map[domain.ClaveCatalogo]semanticaEvidencia)
	peticiones := make(map[string]semanticaEvidencia)
	respuestas := make(map[string]semanticaEvidencia)
	pruebas := make(map[identificadorPrueba]semanticaEvidencia)
	for _, evidencia := range evidencias {
		if evidencia.ValidarEn(comprobadaEn) != nil {
			return nil, ports.ErrResultadoFuenteCoberturaNoConfiable
		}
		resumen, err := evidencia.Resumen()
		definicion, publicada := definiciones[resumen.Comprobacion.Clave]
		semantica := semanticaDe(resumen)
		prueba := identificadorPrueba{
			resumen.AutoridadRef,
			resumen.Generacion,
			resumen.ReciboRespuestaRef,
		}
		if err != nil || !publicada ||
			!coincideConCoordenadas(resumen, coordenadas) ||
			!coincideConDefinicion(resumen, definicion) ||
			!registrarUso(peticiones, resumen.PeticionRef, semantica) ||
			!registrarUso(respuestas, resumen.HuellaRespuestaSHA256, semantica) ||
			!registrarUso(pruebas, prueba, semantica) {
			return nil, ports.ErrResultadoFuenteCoberturaNoConfiable
		}
		anterior, existe := semanticas[resumen.Comprobacion.Clave]
		if existe && anterior != semantica {
			return nil, ports.ErrResultadoFuenteCoberturaNoConfiable
		}
		semanticas[resumen.Comprobacion.Clave] = semantica
		actual, existe := representantes[resumen.Comprobacion.Clave]
		if !existe || evidenciaPreferible(evidencia, actual) {
			representantes[resumen.Comprobacion.Clave] = evidencia
		}
	}
	if len(representantes) != len(reglas) ||
		len(representantes) > maximoEvidenciasPorVia {
		return nil, ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	salida := make([]EvidenciaConsultaCobertura, 0, len(reglas))
	for _, regla := range reglas {
		evidencia, consta := representantes[regla.Clave]
		if !consta {
			return nil, ports.ErrResultadoFuenteCoberturaNoConfiable
		}
		salida = append(salida, evidencia)
	}
	sort.Slice(salida, func(i, j int) bool {
		izquierda, _ := salida[i].Resumen()
		derecha, _ := salida[j].Resumen()
		if izquierda.OrdenComprobacion != derecha.OrdenComprobacion {
			return izquierda.OrdenComprobacion < derecha.OrdenComprobacion
		}
		return izquierda.Comprobacion.Clave < derecha.Comprobacion.Clave
	})
	return salida, nil
}

func validarGobiernoConjunto(
	coordenadas CoordenadasConjuntoEvidencias,
	catalogo domain.CatalogoViasCobertura,
	politica domain.PoliticaDecisionCobertura,
	comprobadaEn time.Time,
) (
	domain.DefinicionViaCobertura,
	[]domain.ReglaComprobacionDecisionCobertura,
	error,
) {
	if coordenadas.validar() != nil ||
		!domain.InstanteUTCCanonico(comprobadaEn) ||
		catalogo.Validar() != nil ||
		!catalogo.Identidad().CoincideExactamente(coordenadas.Catalogo) ||
		!catalogo.VigenteEn(comprobadaEn) ||
		politica.Identidad() != coordenadas.Politica ||
		politica.ValidarPara(
			catalogo, coordenadas.OrganizacionRef,
			coordenadas.FinalidadClave, coordenadas.FinalidadRef,
			comprobadaEn,
		) != nil {
		return domain.DefinicionViaCobertura{}, nil,
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	via, existe := catalogo.Via(coordenadas.ViaClave)
	if !existe {
		return domain.DefinicionViaCobertura{}, nil,
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	for _, regla := range politica.Vias() {
		if regla.ViaClave == coordenadas.ViaClave &&
			len(regla.Comprobaciones) == len(via.Comprobaciones) {
			return via, regla.Comprobaciones, nil
		}
	}
	return domain.DefinicionViaCobertura{}, nil,
		ports.ErrResultadoFuenteCoberturaNoConfiable
}

func semanticaDe(r ports.ResumenOrdenConsumoCobertura) semanticaEvidencia {
	return semanticaEvidencia{
		clave:       string(r.Comprobacion.Clave),
		resultado:   string(r.Comprobacion.Resultado),
		procedencia: string(r.ProcedenciaClave),
		definicion:  r.DefinicionFuenteRef,
		orden:       r.OrdenComprobacion,
		obligatoria: r.ComprobacionObligatoria,
	}
}

func coincideConDefinicion(
	r ports.ResumenOrdenConsumoCobertura,
	d domain.ComprobacionExigibleCobertura,
) bool {
	return r.Comprobacion.Validar() == nil && r.Comprobacion.Detalle == "" &&
		r.Comprobacion.Clave == d.Clave &&
		r.OrdenComprobacion == d.Orden &&
		r.ComprobacionObligatoria == d.Obligatoria &&
		r.ProcedenciaClave == d.Procedencia.Clave &&
		r.DefinicionFuenteRef == d.Procedencia.DefinicionFuenteRef
}

func registrarUso[K comparable](
	usos map[K]semanticaEvidencia,
	clave K,
	semantica semanticaEvidencia,
) bool {
	anterior, existe := usos[clave]
	if existe {
		return anterior == semantica
	}
	usos[clave] = semantica
	return true
}

func evidenciaPreferible(
	candidata EvidenciaConsultaCobertura,
	actual EvidenciaConsultaCobertura,
) bool {
	c, _ := candidata.Resumen()
	a, _ := actual.Resumen()
	switch {
	case !c.ValidaHasta.Equal(a.ValidaHasta):
		return c.ValidaHasta.After(a.ValidaHasta)
	case !c.EmitidaEn.Equal(a.EmitidaEn):
		return c.EmitidaEn.After(a.EmitidaEn)
	case !c.Comprobacion.EvaluadaEn.Equal(a.Comprobacion.EvaluadaEn):
		return c.Comprobacion.EvaluadaEn.After(a.Comprobacion.EvaluadaEn)
	case !candidata.sueloTemporal().Equal(actual.sueloTemporal()):
		return candidata.sueloTemporal().After(actual.sueloTemporal())
	case c.HuellaRespuestaSHA256 != a.HuellaRespuestaSHA256:
		return c.HuellaRespuestaSHA256 < a.HuellaRespuestaSHA256
	default:
		return c.PeticionRef < a.PeticionRef
	}
}

func coincideConCoordenadas(
	r ports.ResumenOrdenConsumoCobertura,
	c CoordenadasConjuntoEvidencias,
) bool {
	return r.OrganizacionRef == c.OrganizacionRef &&
		r.ExpedienteRef == c.ExpedienteRef &&
		r.VersionExpediente == c.VersionExpediente &&
		r.Catalogo.CoincideExactamente(c.Catalogo) &&
		r.ViaClave == c.ViaClave && r.CategoriaRef == c.CategoriaRef &&
		r.Periodo.Inicio.Equal(c.Periodo.Inicio) &&
		r.Periodo.Fin.Equal(c.Periodo.Fin)
}

func (ConjuntoEvidenciasCobertura) String() string     { return redaccionConjunto }
func (c ConjuntoEvidenciasCobertura) GoString() string { return c.String() }
func (c ConjuntoEvidenciasCobertura) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, c.String())
}
func (c ConjuntoEvidenciasCobertura) LogValue() slog.Value {
	return slog.StringValue(c.String())
}
func (c ConjuntoEvidenciasCobertura) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}
func (c ConjuntoEvidenciasCobertura) MarshalJSON() ([]byte, error) {
	return []byte(`"` + c.String() + `"`), nil
}
