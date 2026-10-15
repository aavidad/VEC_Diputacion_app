package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/app/administracion"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/domain"
)

type relojADMIN struct{}

func (relojADMIN) Ahora() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }

// Las dos entradas heredadas quedan cerradas incluso ante una llamada directa.
// El único montaje permitido requiere el archivo privado metadatos_v1.
func componerProcesoADMIN(_ administracion.Configuracion, _ configuracionPerfilesPrivada) (*http.Server, func(), error) {
	return nil, nil, errorArranque("usuarios_config")
}

func componerProcesoADMINConRuntime(_ administracion.Configuracion, _ configuracionPerfilesPrivada, _ configuracionRuntimeADMIN) (*http.Server, func(), error) {
	return nil, nil, errorArranque("usuarios_config")
}

func confianzaDesdeMetadata(m metadatosConfianzaPerfilesPrivados) (administracion.ConfiguracionConfianzaPerfilesV3, error) {
	var cero administracion.ConfiguracionConfianzaPerfilesV3
	publica, err := base64.StdEncoding.Strict().DecodeString(m.Raiz.PublicaBase64)
	if err != nil {
		return cero, administracion.ErrConfiguracion
	}
	c := administracion.ConfiguracionConfianzaPerfilesV3{
		Cabecera: domain.CabeceraAtestacionAutorizacionV3{FormatoVersion: m.Cabecera.FormatoVersion, Suite: m.Cabecera.Suite, ClaveID: m.Cabecera.ClaveID, Audiencia: m.Cabecera.Audiencia},
		Raiz:     administracion.MaterialRaizPerfilesV3{ClaveID: m.Raiz.ClaveID, Audiencia: m.Raiz.Audiencia, Version: m.Raiz.Version, Publica: ed25519.PublicKey(publica), Estado: confianza.EstadoClaveAtestacionAutorizacionV3(m.Raiz.Estado), ValidaDesde: m.Raiz.ValidaDesde, ValidaHasta: m.Raiz.ValidaHasta, RevocadaEn: m.Raiz.RevocadaEn},
		Gobierno: administracion.GobiernoConfianzaPerfilesV3{Revision: m.Gobierno.Revision, HuellaSHA256: m.Gobierno.HuellaSHA256, Secuencia: m.Gobierno.Secuencia, PublicadaEn: m.Gobierno.PublicadaEn, ExpiraEn: m.Gobierno.ExpiraEn}}
	for _, e := range m.EntradasCapacidad {
		b, err := leerArchivoPrivadoPerfiles(e.MaterialArchivo)
		if err != nil {
			for _, x := range c.EntradasCapacidad {
				clear(x.Material)
			}
			return cero, administracion.ErrConfiguracion
		}
		c.EntradasCapacidad = append(c.EntradasCapacidad, administracion.MaterialCapacidadPerfilesV3{
			Audiencia: e.Audiencia, ClaveID: e.ClaveID, EmisorID: e.EmisorID, HuellaGobierno: e.HuellaGobierno, Version: e.Version, RevisionGobierno: e.RevisionGobierno,
			Material: b, ValidaDesde: e.ValidaDesde, ValidaHasta: e.ValidaHasta, Estado: confianza.EstadoClaveHMACCapacidadAtestacionV3(e.Estado), RevocadaEn: e.RevocadaEn})
	}
	return c, nil
}
