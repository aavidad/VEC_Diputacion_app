package reglasbaremo

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"
)

// RestaurarConjuntoReglasBaremo reconstruye exclusivamente una
// RepresentacionCanonica V1. Cualquier otra codificacion, aunque json.Unmarshal
// pudiera interpretarla con el mismo resultado aparente, se rechaza.
func RestaurarConjuntoReglasBaremo(contenido []byte) (ConjuntoReglasBaremo, error) {
	if len(contenido) == 0 || len(contenido) > maximoBytesRepresentacion {
		return ConjuntoReglasBaremo{}, nuevoError("representacion_canonica", CodigoFueraDeLimites)
	}
	if !utf8.Valid(contenido) {
		return ConjuntoReglasBaremo{}, nuevoError("representacion_canonica", CodigoValorNoCanonico)
	}

	var material materialConjunto
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&material); err != nil {
		return ConjuntoReglasBaremo{}, nuevoError("representacion_canonica", CodigoValorNoCanonico)
	}
	var sobrante struct{}
	if err := decodificador.Decode(&sobrante); !errors.Is(err, io.EOF) {
		return ConjuntoReglasBaremo{}, nuevoError("representacion_canonica", CodigoValorNoCanonico)
	}
	if material.Esquema != esquemaConjuntoReglasBaremo {
		return ConjuntoReglasBaremo{}, nuevoError("esquema", CodigoEsquemaIncompatible)
	}

	conjunto, err := reconstruirConjunto(material)
	if err != nil {
		return ConjuntoReglasBaremo{}, err
	}
	canonico, err := conjunto.RepresentacionCanonica()
	if err != nil || !bytes.Equal(canonico, contenido) {
		return ConjuntoReglasBaremo{}, nuevoError("representacion_canonica", CodigoValorNoCanonico)
	}
	return conjunto, nil
}

// RestaurarConjuntoReglasBaremoConHuellaSHA256 añade a la restauracion
// canonica la comprobacion en tiempo constante de una huella esperada.
func RestaurarConjuntoReglasBaremoConHuellaSHA256(
	contenido []byte,
	huellaEsperada string,
) (ConjuntoReglasBaremo, error) {
	if !huellaSHA256Valida(huellaEsperada) {
		return ConjuntoReglasBaremo{}, nuevoError("huella_esperada_sha256", CodigoValorNoCanonico)
	}
	conjunto, err := RestaurarConjuntoReglasBaremo(contenido)
	if err != nil {
		return ConjuntoReglasBaremo{}, err
	}
	huellaReal, err := conjunto.HuellaSHA256()
	if err != nil || subtle.ConstantTimeCompare([]byte(huellaReal), []byte(huellaEsperada)) != 1 {
		return ConjuntoReglasBaremo{}, nuevoError("huella_esperada_sha256", CodigoHuellaNoCoincide)
	}
	return conjunto, nil
}

func reconstruirConjunto(material materialConjunto) (ConjuntoReglasBaremo, error) {
	identidad, err := NuevaIdentidadConjuntoReglasBaremo(
		material.Identidad.Referencia,
		material.Identidad.Version,
		material.Identidad.ConvocatoriaRef,
		material.Identidad.ExpedienteRef,
	)
	if err != nil {
		return ConjuntoReglasBaremo{}, err
	}
	bases, err := reconstruirReferencia(material.Bases)
	if err != nil {
		return ConjuntoReglasBaremo{}, err
	}

	secciones := make([]SeccionBaremo, len(material.Secciones))
	for indice, origen := range material.Secciones {
		definicion, err := reconstruirReferencia(origen.Definicion)
		if err != nil {
			return ConjuntoReglasBaremo{}, err
		}
		secciones[indice], err = NuevaSeccionBaremo(
			origen.Clave,
			definicion,
			origen.Orden,
			origen.PuntosMinimos,
			origen.PuntosMaximos,
		)
		if err != nil {
			return ConjuntoReglasBaremo{}, err
		}
	}
	grupos := make([]GrupoConcurrenciaExperiencia, len(material.GruposConcurrencia))
	for indice, origen := range material.GruposConcurrencia {
		grupos[indice], err = reconstruirGrupoConcurrencia(origen)
		if err != nil {
			return ConjuntoReglasBaremo{}, err
		}
	}

	reglas := make([]ReglaExperiencia, len(material.ReglasExperiencia))
	for indice, origen := range material.ReglasExperiencia {
		reglas[indice], err = reconstruirReglaExperiencia(origen)
		if err != nil {
			return ConjuntoReglasBaremo{}, err
		}
	}
	return NuevoConjuntoReglasBaremo(
		identidad,
		bases,
		material.FechaCorteInclusiva,
		secciones,
		grupos,
		reglas,
	)
}

func reconstruirGrupoConcurrencia(
	material materialGrupoConcurrencia,
) (GrupoConcurrenciaExperiencia, error) {
	definicion, err := reconstruirReferencia(material.Definicion)
	if err != nil {
		return GrupoConcurrenciaExperiencia{}, err
	}
	coincidencia, err := NuevaPoliticaCoincidenciaReglas(material.CoincidenciaReglas.Modo)
	if err != nil {
		return GrupoConcurrenciaExperiencia{}, err
	}
	solape, err := reconstruirPoliticaSolape(material.Solape)
	if err != nil {
		return GrupoConcurrenciaExperiencia{}, err
	}
	var reparto *PoliticaRepartoExceso
	if material.RepartoExceso != nil {
		construido, err := NuevaPoliticaRepartoExceso(material.RepartoExceso.Modo)
		if err != nil {
			return GrupoConcurrenciaExperiencia{}, err
		}
		if construido.DesempateEntreReglas() != material.RepartoExceso.DesempateEntreReglas ||
			construido.RepartoDentroMismaRegla() != material.RepartoExceso.RepartoDentroMismaRegla {
			return GrupoConcurrenciaExperiencia{}, nuevoError(
				"politica_reparto_exceso.semantica", CodigoPoliticaIncompleta,
			)
		}
		reparto = &construido
	}
	return NuevoGrupoConcurrenciaExperiencia(
		material.Clave,
		definicion,
		material.Orden,
		coincidencia,
		solape,
		reparto,
	)
}

func reconstruirReferencia(material materialReferencia) (ReferenciaVersionada, error) {
	return NuevaReferenciaVersionada(material.Referencia, material.Version, material.HuellaSHA256)
}

func reconstruirReglaExperiencia(material materialReglaExperiencia) (ReglaExperiencia, error) {
	definicion, err := reconstruirReferencia(material.Definicion)
	if err != nil {
		return ReglaExperiencia{}, err
	}
	criterios := make([]CriterioExperiencia, len(material.Criterios))
	for indice, origen := range material.Criterios {
		catalogo, err := reconstruirReferencia(origen.Catalogo)
		if err != nil {
			return ReglaExperiencia{}, err
		}
		criterios[indice], err = NuevoCriterioExperiencia(origen.Clave, catalogo, origen.Valores)
		if err != nil {
			return ReglaExperiencia{}, err
		}
	}

	temporal, err := NuevaPoliticaUnidadTemporal(
		material.UnidadTemporal.UnidadBase,
		material.UnidadTemporal.UnidadPuntuable,
		material.UnidadTemporal.UnidadesBasePorUnidad,
		material.UnidadTemporal.ExtremoFinal,
	)
	if err != nil {
		return ReglaExperiencia{}, err
	}
	jornada, err := reconstruirPoliticaJornada(material.Jornada)
	if err != nil {
		return ReglaExperiencia{}, err
	}
	restos, err := NuevaPoliticaRestos(material.Restos.Modo)
	if err != nil {
		return ReglaExperiencia{}, err
	}
	redondeo, err := NuevaPoliticaRedondeo(material.Redondeo.Momento, material.Redondeo.Modo)
	if err != nil {
		return ReglaExperiencia{}, err
	}
	maximoUnidades, err := reconstruirLimiteUnidades(material.MaximoUnidades)
	if err != nil {
		return ReglaExperiencia{}, err
	}
	maximoPuntos, err := reconstruirLimitePuntos(material.MaximoPuntos)
	if err != nil {
		return ReglaExperiencia{}, err
	}

	return NuevaReglaExperiencia(
		material.Clave,
		definicion,
		material.SeccionClave,
		material.Orden,
		criterios,
		material.GrupoConcurrenciaClave,
		material.PrioridadConcurrencia,
		temporal,
		jornada,
		restos,
		redondeo,
		material.PuntosPorUnidad,
		maximoUnidades,
		maximoPuntos,
	)
}

func reconstruirPoliticaJornada(material materialPoliticaJornada) (PoliticaJornada, error) {
	if material.Umbral == nil {
		return NuevaPoliticaJornada(material.Modo)
	}
	if material.Modo != JornadaIntegraDesdeUmbral {
		return PoliticaJornada{}, nuevoError("politica_jornada.umbral", CodigoPoliticaIncompleta)
	}
	return NuevaPoliticaJornadaDesdeUmbral(*material.Umbral)
}

func reconstruirPoliticaSolape(material materialPoliticaSolape) (PoliticaSolape, error) {
	if material.Limite == nil {
		return NuevaPoliticaSolape(material.Modo)
	}
	if material.Modo != SolapeAcumularHastaLimite {
		return PoliticaSolape{}, nuevoError("politica_solape.limite", CodigoPoliticaIncompleta)
	}
	return NuevaPoliticaSolapeAcumulable(*material.Limite)
}

func reconstruirLimiteUnidades(material materialLimiteUnidades) (LimiteUnidades, error) {
	switch material.Modo {
	case modoSinLimite:
		if material.Valor != nil {
			return LimiteUnidades{}, nuevoError("maximo_unidades", CodigoPoliticaIncompleta)
		}
		return SinLimiteUnidades(), nil
	case modoLimitado:
		if material.Valor == nil {
			return LimiteUnidades{}, nuevoError("maximo_unidades", CodigoPoliticaIncompleta)
		}
		return NuevoLimiteUnidades(*material.Valor)
	default:
		return LimiteUnidades{}, nuevoError("maximo_unidades", CodigoPoliticaIncompleta)
	}
}

func reconstruirLimitePuntos(material materialLimitePuntos) (LimitePuntos, error) {
	switch material.Modo {
	case modoSinLimite:
		if material.Valor != nil {
			return LimitePuntos{}, nuevoError("maximo_puntos", CodigoPoliticaIncompleta)
		}
		return SinLimitePuntos(), nil
	case modoLimitado:
		if material.Valor == nil {
			return LimitePuntos{}, nuevoError("maximo_puntos", CodigoPoliticaIncompleta)
		}
		return NuevoLimitePuntos(*material.Valor)
	default:
		return LimitePuntos{}, nuevoError("maximo_puntos", CodigoPoliticaIncompleta)
	}
}
