package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
)

func calcularHuellaPropuestaDecisionCobertura(
	publicacion PublicacionPropuestaDecisionCobertura,
) (string, error) {
	material, err := materialCanonicoPropuestaDecisionCoberturaV1(publicacion)
	if err != nil {
		return "", err
	}
	huella := sha256.Sum256(material)
	return hex.EncodeToString(huella[:]), nil
}

func materialCanonicoPropuestaDecisionCoberturaV1(
	publicacion PublicacionPropuestaDecisionCobertura,
) ([]byte, error) {
	if !publicacion.Canon.valido() ||
		!referenciaValida(publicacion.OrganizacionRef) ||
		!referenciaValida(publicacion.ExpedienteRef) ||
		publicacion.VersionExpediente == 0 ||
		!referenciaValida(publicacion.AnalisisRef) ||
		!huellaValida(publicacion.AnalisisHuellaSHA256) ||
		publicacion.Catalogo.Validar() != nil ||
		publicacion.Politica.Validar() != nil ||
		!publicacion.FinalidadClave.Valida() ||
		!referenciaValida(publicacion.FinalidadRef) ||
		!referenciaValida(publicacion.CategoriaRef) ||
		!periodoAnalisisValido(publicacion.Periodo) ||
		!instanteCanonico(publicacion.GeneradaEn) ||
		!instanteCanonico(publicacion.ValidaHasta) ||
		!publicacion.ValidaHasta.After(publicacion.GeneradaEn) ||
		!publicacion.Estado.valido() ||
		!cardinalidadPropuestaDecisionCoberturaValida(publicacion) ||
		(publicacion.Estado == PropuestaCoberturaViable) !=
			publicacion.ViaPropuesta.Valida() {
		return nil, ErrDatoInvalido
	}
	var destino bytes.Buffer
	escritor := escritorCanonCatalogo{destino: &destino}
	escritor.cadena(publicacion.Canon.Dominio)
	escritor.entero16(publicacion.Canon.VersionEsquema)
	escritor.cadena(publicacion.Canon.Algoritmo)
	escritor.cadena(publicacion.OrganizacionRef)
	escritor.cadena(publicacion.ExpedienteRef)
	escritor.entero64(publicacion.VersionExpediente)
	escritor.cadena(publicacion.AnalisisRef)
	escritor.cadena(publicacion.AnalisisHuellaSHA256)
	escribirIdentidadCatalogoPropuesta(&escritor, publicacion.Catalogo)
	escritor.cadena(publicacion.Politica.Referencia)
	escritor.entero64(publicacion.Politica.Version)
	escritor.cadena(publicacion.Politica.HuellaSHA256)
	escritor.cadena(string(publicacion.FinalidadClave))
	escritor.cadena(publicacion.FinalidadRef)
	escritor.cadena(publicacion.CategoriaRef)
	escritor.instante(publicacion.Periodo.Inicio)
	escritor.instante(publicacion.Periodo.Fin)
	escritor.instante(publicacion.GeneradaEn)
	escritor.instante(publicacion.ValidaHasta)
	escritor.cadena(string(publicacion.Estado))
	escritor.cadena(string(publicacion.ViaPropuesta))
	escribirResultadosPropuesta(&escritor, publicacion.Resultados)
	escribirEvaluacionesPropuesta(&escritor, publicacion.Evaluaciones)
	if escritor.err != nil {
		return nil, ErrDatoInvalido
	}
	return destino.Bytes(), nil
}

func cardinalidadPropuestaDecisionCoberturaValida(
	publicacion PublicacionPropuestaDecisionCobertura,
) bool {
	if len(publicacion.Resultados) > maximoComprobacionesCatalogo ||
		len(publicacion.Evaluaciones) > maximoViasCobertura {
		return false
	}
	totalEvidencias := 0
	for _, resultado := range publicacion.Resultados {
		if !resultado.Clave.Valida() || len(resultado.Evidencias) == 0 ||
			len(resultado.Evidencias) >
				maximoEvidenciasPorComprobacionCobertura {
			return false
		}
		totalEvidencias += len(resultado.Evidencias)
		for _, evidencia := range resultado.Evidencias {
			if evidencia.validar() != nil {
				return false
			}
		}
	}
	return totalEvidencias <= maximoResultadosEntradaPropuestaCobertura
}

func escribirIdentidadCatalogoPropuesta(
	escritor *escritorCanonCatalogo,
	identidad IdentidadCatalogoViasCobertura,
) {
	escritor.cadena(identidad.Referencia)
	escritor.entero64(identidad.Version)
	escritor.cadena(identidad.HuellaSHA256)
}

func escribirResultadosPropuesta(
	escritor *escritorCanonCatalogo,
	resultados []ResultadoAgrupadoPropuestaCobertura,
) {
	escritor.entero32(uint32(len(resultados)))
	for _, resultado := range resultados {
		escritor.cadena(string(resultado.Clave))
		escritor.entero32(uint32(len(resultado.Evidencias)))
		for _, evidencia := range resultado.Evidencias {
			escritor.cadena(string(evidencia.Resultado))
			escritor.cadena(evidencia.FuenteRef)
			escritor.cadena(evidencia.ReciboRef)
			escritor.instante(evidencia.EvaluadaEn)
		}
	}
}

func escribirEvaluacionesPropuesta(
	escritor *escritorCanonCatalogo,
	evaluaciones []EvaluacionViaPropuestaCobertura,
) {
	escritor.entero32(uint32(len(evaluaciones)))
	for _, evaluacion := range evaluaciones {
		escritor.cadena(string(evaluacion.ViaClave))
		escritor.entero16(evaluacion.Prioridad)
		escritor.cadena(string(evaluacion.Estado))
		escribirClavesPropuesta(escritor, evaluacion.ResultadosOmitidos)
		escribirClavesPropuesta(escritor, evaluacion.AusenciasBloqueantes)
		escribirClavesPropuesta(escritor, evaluacion.AusenciasAdmitidas)
		escribirClavesPropuesta(escritor, evaluacion.NoHabilitantes)
		escribirClavesPropuesta(escritor, evaluacion.Conflictos)
	}
}

func escribirClavesPropuesta(
	escritor *escritorCanonCatalogo,
	claves []ClaveCatalogo,
) {
	escritor.entero32(uint32(len(claves)))
	for _, clave := range claves {
		escritor.cadena(string(clave))
	}
}
