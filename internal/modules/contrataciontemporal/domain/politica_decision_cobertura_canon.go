package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
)

func calcularHuellaPoliticaDecisionCobertura(
	publicacion PublicacionPoliticaDecisionCobertura,
) (string, error) {
	material, err := materialCanonicoPoliticaDecisionCoberturaV1(publicacion)
	if err != nil {
		return "", err
	}
	huella := sha256.Sum256(material)
	return hex.EncodeToString(huella[:]), nil
}

// materialCanonicoPoliticaDecisionCoberturaV1 es un dominio nuevo. No reutiliza
// ni modifica la preimagen histórica del catálogo O4-01.
func materialCanonicoPoliticaDecisionCoberturaV1(
	publicacion PublicacionPoliticaDecisionCobertura,
) ([]byte, error) {
	if !publicacion.Canon.valido() ||
		publicacion.Catalogo.Validar() != nil ||
		!referenciaValida(publicacion.Referencia) ||
		!instanteCanonico(publicacion.PublicadaEn) {
		return nil, ErrDatoInvalido
	}
	var destino bytes.Buffer
	escritor := escritorCanonCatalogo{destino: &destino}
	escritor.cadena(publicacion.Canon.Dominio)
	escritor.entero16(publicacion.Canon.VersionEsquema)
	escritor.cadena(publicacion.Canon.Algoritmo)
	escritor.cadena(publicacion.Referencia)
	escritor.entero64(publicacion.Version)
	escritor.cadena(publicacion.Catalogo.Referencia)
	escritor.entero64(publicacion.Catalogo.Version)
	escritor.cadena(publicacion.Catalogo.HuellaSHA256)
	escritor.cadena(publicacion.OrganizacionRef)
	escritor.cadena(string(publicacion.FinalidadClave))
	escritor.cadena(publicacion.FinalidadRef)
	escritor.instante(publicacion.PublicadaEn)
	escritor.instante(publicacion.Vigencia.Desde)
	escritor.instante(publicacion.Vigencia.Hasta)
	escritor.cadena(publicacion.ProcedenciaRef)
	escritor.entero32(uint32(len(publicacion.Vias)))
	for _, via := range publicacion.Vias {
		escritor.cadena(string(via.ViaClave))
		escritor.entero16(via.Prioridad)
		escritor.entero32(uint32(len(via.Comprobaciones)))
		for _, comprobacion := range via.Comprobaciones {
			escritor.cadena(string(comprobacion.Clave))
			escritor.cadena(string(comprobacion.TratamientoAusencia))
			escritor.entero32(uint32(
				len(comprobacion.ResultadosHabilitantes),
			))
			for _, resultado := range comprobacion.ResultadosHabilitantes {
				escritor.cadena(string(resultado))
			}
		}
	}
	if escritor.err != nil {
		return nil, ErrDatoInvalido
	}
	return destino.Bytes(), nil
}
