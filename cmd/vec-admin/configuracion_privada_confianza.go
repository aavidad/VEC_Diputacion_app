package main

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

type cabeceraConfianzaPerfilesPrivada struct {
	FormatoVersion uint16 `json:"formato_version"`
	Suite          string `json:"suite"`
	ClaveID        string `json:"clave_id"`
	Audiencia      string `json:"audiencia"`
}
type raizConfianzaPerfilesPrivada struct {
	ClaveID       string    `json:"clave_id"`
	Audiencia     string    `json:"audiencia"`
	Version       uint64    `json:"version"`
	PublicaBase64 string    `json:"publica_base64"`
	Estado        string    `json:"estado"`
	ValidaDesde   time.Time `json:"valida_desde"`
	ValidaHasta   time.Time `json:"valida_hasta"`
	RevocadaEn    time.Time `json:"revocada_en"`
}
type gobiernoConfianzaPerfilesPrivada struct {
	Revision     string    `json:"revision"`
	HuellaSHA256 string    `json:"huella_sha256"`
	Secuencia    uint64    `json:"secuencia"`
	PublicadaEn  time.Time `json:"publicada_en"`
	ExpiraEn     time.Time `json:"expira_en"`
}
type capacidadConfianzaPerfilesPrivada struct {
	Audiencia        string    `json:"audiencia"`
	ClaveID          string    `json:"clave_id"`
	EmisorID         string    `json:"emisor_id"`
	HuellaGobierno   string    `json:"huella_gobierno"`
	Version          uint64    `json:"version"`
	RevisionGobierno uint64    `json:"revision_gobierno"`
	MaterialArchivo  string    `json:"material_archivo"`
	Estado           string    `json:"estado"`
	ValidaDesde      time.Time `json:"valida_desde"`
	ValidaHasta      time.Time `json:"valida_hasta"`
	RevocadaEn       time.Time `json:"revocada_en"`
}
type metadatosConfianzaPerfilesPrivados struct {
	Cabecera          cabeceraConfianzaPerfilesPrivada    `json:"cabecera"`
	Raiz              raizConfianzaPerfilesPrivada        `json:"raiz"`
	Gobierno          gobiernoConfianzaPerfilesPrivada    `json:"gobierno"`
	EntradasCapacidad []capacidadConfianzaPerfilesPrivada `json:"entradas_capacidad"`
}

func decodificarMetadatosConfianzaPerfiles(b json.RawMessage) (metadatosConfianzaPerfilesPrivados, error) {
	var c metadatosConfianzaPerfilesPrivados
	if decodificarConfiguracionPrivada(b, &c) != nil {
		return c, errConfiguracionPrivadaPerfiles
	}
	cabecera := domain.CabeceraAtestacionAutorizacionV3{FormatoVersion: c.Cabecera.FormatoVersion, Suite: c.Cabecera.Suite, ClaveID: c.Cabecera.ClaveID, Audiencia: c.Cabecera.Audiencia}
	publica, err := base64.StdEncoding.Strict().DecodeString(c.Raiz.PublicaBase64)
	if cabecera.Validar() != nil || err != nil || len(publica) != 32 || c.Raiz.ClaveID != c.Cabecera.ClaveID || c.Raiz.Audiencia != c.Cabecera.Audiencia || c.Raiz.Version == 0 || !vigenciaMetadataPerfiles(c.Raiz.Estado, c.Raiz.ValidaDesde, c.Raiz.ValidaHasta, c.Raiz.RevocadaEn) || !textoConfiguracionPerfiles(c.Gobierno.Revision) || !domain.HuellaAdministracionPerfilesValida(c.Gobierno.HuellaSHA256) || c.Gobierno.Secuencia == 0 || !fechaMetadataPerfiles(c.Gobierno.PublicadaEn) || !fechaMetadataPerfiles(c.Gobierno.ExpiraEn) || !c.Gobierno.ExpiraEn.After(c.Gobierno.PublicadaEn) || len(c.EntradasCapacidad) == 0 || len(c.EntradasCapacidad) > 32 {
		return metadatosConfianzaPerfilesPrivados{}, errConfiguracionPrivadaPerfiles
	}
	audiencias := map[string]bool{}
	rutas := map[string]bool{}
	for _, e := range c.EntradasCapacidad {
		if !textoConfiguracionPerfiles(e.Audiencia) || !textoConfiguracionPerfiles(e.ClaveID) || !textoConfiguracionPerfiles(e.EmisorID) || !domain.HuellaAdministracionPerfilesValida(e.HuellaGobierno) || e.Version == 0 || e.RevisionGobierno == 0 || !rutaPrivadaPerfilesValida(e.MaterialArchivo) || audiencias[e.Audiencia] || rutas[e.MaterialArchivo] || !vigenciaCapacidadMetadataPerfiles(e.Estado, e.ValidaDesde, e.ValidaHasta, e.RevocadaEn) {
			return metadatosConfianzaPerfilesPrivados{}, errConfiguracionPrivadaPerfiles
		}
		audiencias[e.Audiencia] = true
		rutas[e.MaterialArchivo] = true
	}
	return c, nil
}
func fechaMetadataPerfiles(t time.Time) bool {
	return !t.IsZero() && t.Location() == time.UTC && t.Nanosecond()%1000 == 0
}
func vigenciaMetadataPerfiles(estado string, desde, hasta, revocada time.Time) bool {
	if !fechaMetadataPerfiles(desde) || !fechaMetadataPerfiles(hasta) || !hasta.After(desde) {
		return false
	}
	if estado == "activa" {
		return revocada.IsZero()
	}
	return estado == "revocada" && fechaMetadataPerfiles(revocada) && !revocada.Before(desde) && revocada.Before(hasta)
}

func vigenciaCapacidadMetadataPerfiles(estado string, desde, hasta, revocada time.Time) bool {
	if estado == "emision" || estado == "verificacion" {
		return vigenciaMetadataPerfiles("activa", desde, hasta, revocada)
	}
	return estado == "revocada" && vigenciaMetadataPerfiles(estado, desde, hasta, revocada)
}
