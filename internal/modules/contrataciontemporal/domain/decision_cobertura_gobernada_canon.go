package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
)

func calcularHuellaDecisionCobertura(
	publicacion PublicacionDecisionCoberturaGobernada,
) (string, error) {
	material, err := materialCanonicoDecisionCoberturaV1(publicacion)
	if err != nil {
		return "", err
	}
	huella := sha256.Sum256(material)
	return hex.EncodeToString(huella[:]), nil
}

// materialCanonicoDecisionCoberturaV1 usa longitudes uint32, enteros
// big-endian e instantes Unix en microsegundos. No depende de JSON.
func materialCanonicoDecisionCoberturaV1(
	p PublicacionDecisionCoberturaGobernada,
) ([]byte, error) {
	if !p.Canon.valido() || !p.Tipo.valido() ||
		!referenciaValida(p.OrganizacionRef) ||
		!referenciaValida(p.ExpedienteRef) ||
		p.VersionExpedienteOrigen == 0 ||
		p.VersionExpedienteOrigen >= maximoEnteroSeguroCatalogoCobertura ||
		p.VersionExpediente != p.VersionExpedienteOrigen+1 ||
		!referenciaValida(p.ActorRef) || !referenciaValida(p.PerfilRef) ||
		!referenciaValida(p.PropuestaRef) ||
		!huellaValida(p.PropuestaHuellaSHA256) ||
		!referenciaValida(p.PreparacionEvidenciasRef) ||
		!huellaValida(p.PreparacionEvidenciasHuellaSHA256) ||
		!referenciaValida(p.AnalisisRef) ||
		!huellaValida(p.AnalisisHuellaSHA256) ||
		p.Catalogo.Validar() != nil || p.Politica.Validar() != nil ||
		!p.ViaElegida.Valida() || !p.ViaRecomendada.Valida() ||
		!instanteCatalogoCoberturaValido(p.DecididaEn) ||
		p.Actuacion.validar() != nil ||
		p.Actuacion.VersionExpediente != p.VersionExpediente ||
		p.Actuacion.Secuencia != p.VersionExpediente ||
		p.Actuacion.AccionClave != accionEsperadaDecision(p.Tipo) ||
		p.Actuacion.ActorRef != p.ActorRef ||
		!p.Actuacion.RealizadaEn.Equal(p.DecididaEn) ||
		!motivoYPredecesoraDecisionValidos(p) {
		return nil, ErrDatoInvalido
	}
	var destino bytes.Buffer
	e := escritorCanonCatalogo{destino: &destino}
	e.cadena(p.Canon.Dominio)
	e.entero16(p.Canon.VersionEsquema)
	e.cadena(p.Canon.Algoritmo)
	e.cadena(string(p.Tipo))
	e.cadena(p.OrganizacionRef)
	e.cadena(p.ExpedienteRef)
	e.entero64(p.VersionExpedienteOrigen)
	e.entero64(p.VersionExpediente)
	e.cadena(p.ActorRef)
	e.cadena(p.PerfilRef)
	e.cadena(p.PropuestaRef)
	e.cadena(p.PropuestaHuellaSHA256)
	e.cadena(p.PreparacionEvidenciasRef)
	e.cadena(p.PreparacionEvidenciasHuellaSHA256)
	e.cadena(p.AnalisisRef)
	e.cadena(p.AnalisisHuellaSHA256)
	escribirIdentidadCatalogoPropuesta(&e, p.Catalogo)
	e.cadena(p.Politica.Referencia)
	e.entero64(p.Politica.Version)
	e.cadena(p.Politica.HuellaSHA256)
	e.cadena(string(p.ViaElegida))
	e.cadena(string(p.ViaRecomendada))
	e.cadena(p.Motivo.ReferenciaCatalogo.CatalogoID)
	e.entero64(uint64(p.Motivo.ReferenciaCatalogo.CatalogoVersion))
	e.cadena(p.Motivo.ReferenciaCatalogo.CatalogoHuellaSHA256)
	e.cadena(p.Motivo.ReferenciaCatalogo.EntradaClave)
	e.cadena(string(p.Motivo.ClaveI18n))
	e.cadena(p.PredecesoraRef)
	e.cadena(p.PredecesoraHuellaSHA256)
	e.instante(p.DecididaEn)
	e.entero64(p.Actuacion.Secuencia)
	e.entero64(p.Actuacion.VersionExpediente)
	e.cadena(string(p.Actuacion.AccionClave))
	e.cadena(p.Actuacion.ActorRef)
	e.cadena(p.Actuacion.UnidadRef)
	e.instante(p.Actuacion.RealizadaEn)
	e.cadena(string(p.Actuacion.FaseOrigen))
	e.cadena(string(p.Actuacion.FaseDestino))
	e.cadena(string(p.Actuacion.EstadoOrigen))
	e.cadena(string(p.Actuacion.EstadoDestino))
	e.cadena(p.Actuacion.ReciboRef)
	if e.err != nil {
		return nil, ErrDatoInvalido
	}
	return destino.Bytes(), nil
}

func motivoYPredecesoraDecisionValidos(
	p PublicacionDecisionCoberturaGobernada,
) bool {
	if p.Tipo == DecisionCoberturaInicial {
		if p.PredecesoraRef != "" || p.PredecesoraHuellaSHA256 != "" {
			return false
		}
		if p.ViaElegida == p.ViaRecomendada {
			return p.Motivo.vacio()
		}
		return p.Motivo.valido()
	}
	return p.Motivo.valido() &&
		referenciaValida(p.PredecesoraRef) &&
		huellaValida(p.PredecesoraHuellaSHA256)
}
