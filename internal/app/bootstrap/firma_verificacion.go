package bootstrap

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/vec/documentos/adapters/validadorautofirma"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	"vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrComposicionFirmaVerificacionNoDisponible = errors.New("bootstrap: verificacion de firma no disponible")

// nuevoVerificadorFirmaDocumentos compone exclusivamente el puerto interno.
// La presencia del cliente no publica rutas ni habilita una transicion a firmado.
func nuevoVerificadorFirmaDocumentos(cfg config.Config, fuenteResultados ...func() vecports.EmisorResultadosTecnicosConContexto) (docports.VerificadorFirmaMotivado, error) {
	if len(fuenteResultados) > 1 {
		return nil, ErrComposicionFirmaVerificacionNoDisponible
	}
	cfg = cfg.Normalize()
	switch cfg.FirmaVerificacionEnabled {
	case "", "false":
		return nil, nil
	}
	material, err := cargarMaterialFirmaDocumentos(cfg)
	if err != nil {
		return nil, err
	}
	defer material.borrar()
	material.configuracion.Disponibilidad = observadorFirmaDocumentos(fuenteResultados...)
	cliente, err := validadorautofirma.Nuevo(material.configuracion)
	if err != nil {
		return nil, ErrComposicionFirmaVerificacionNoDisponible
	}
	return cliente, nil
}

type materialFirmaDocumentos struct {
	configuracion                 validadorautofirma.Configuracion
	ca, token, certificado, clave []byte
}

func (m *materialFirmaDocumentos) borrar() {
	if m == nil {
		return
	}
	borrarBytes(m.ca)
	borrarBytes(m.token)
	borrarBytes(m.certificado)
	borrarBytes(m.clave)
}

// Una sola lectura privada alimenta el cliente y el comprobante R5. La
// configuración cargada conserva las mismas guardas previas de la vía común.
func cargarMaterialFirmaDocumentos(cfg config.Config) (materialFirmaDocumentos, error) {
	var m materialFirmaDocumentos
	cfg = cfg.Normalize()
	activo, err := cfg.DocumentosDesarrolloActivo()
	if cfg.FirmaVerificacionEnabled != "true" || err != nil || !activo {
		return m, ErrComposicionFirmaVerificacionNoDisponible
	}
	if cfg.FirmaVerificacionURL == "" || cfg.FirmaVerificacionCAFile == "" || cfg.FirmaVerificacionTimeout == "" ||
		(cfg.FirmaVerificacionTokenFile == "" && (cfg.FirmaVerificacionCertFile == "" || cfg.FirmaVerificacionKeyFile == "")) ||
		(cfg.FirmaVerificacionCertFile == "") != (cfg.FirmaVerificacionKeyFile == "") {
		return m, ErrComposicionFirmaVerificacionNoDisponible
	}
	if !nombreServidorTLSValido(cfg.FirmaVerificacionNombreServidorTLS) {
		return m, ErrComposicionFirmaVerificacionNoDisponible
	}
	plazo, err := time.ParseDuration(cfg.FirmaVerificacionTimeout)
	if err != nil || plazo < time.Millisecond || plazo > 60*time.Second {
		return m, ErrComposicionFirmaVerificacionNoDisponible
	}
	m.ca, err = leerFicheroMaterialSeguro(cfg.FirmaVerificacionCAFile, 64<<10)
	if err != nil {
		return m, ErrComposicionFirmaVerificacionNoDisponible
	}
	if cfg.FirmaVerificacionTokenFile != "" {
		m.token, err = leerFicheroMaterialSeguro(cfg.FirmaVerificacionTokenFile, 512)
		if err != nil {
			m.borrar()
			return materialFirmaDocumentos{}, ErrComposicionFirmaVerificacionNoDisponible
		}
	}
	if cfg.FirmaVerificacionCertFile != "" {
		m.certificado, err = leerFicheroMaterialSeguro(cfg.FirmaVerificacionCertFile, 64<<10)
		if err != nil {
			m.borrar()
			return materialFirmaDocumentos{}, ErrComposicionFirmaVerificacionNoDisponible
		}
		m.clave, err = leerFicheroMaterialSeguro(cfg.FirmaVerificacionKeyFile, 64<<10)
		if err != nil {
			m.borrar()
			return materialFirmaDocumentos{}, ErrComposicionFirmaVerificacionNoDisponible
		}
	}
	m.configuracion = validadorautofirma.Configuracion{URL: cfg.FirmaVerificacionURL, CAPEM: m.ca,
		NombreServidorTLS: cfg.FirmaVerificacionNombreServidorTLS, Token: m.token,
		CertificadoClientePEM: m.certificado, ClaveClientePEM: m.clave, Timeout: plazo}
	return m, nil
}

func observadorFirmaDocumentos(fuenteResultados ...func() vecports.EmisorResultadosTecnicosConContexto) func(context.Context, bool) {
	var observar func(context.Context, bool)
	if len(fuenteResultados) == 1 && fuenteResultados[0] != nil {
		observar = func(ctx context.Context, disponible bool) {
			emisor := fuenteResultados[0]()
			if emisor == nil {
				return
			}
			resultado := domain.ResultadoTecnicoNoDisponible
			if disponible {
				resultado = domain.ResultadoTecnicoCorrecto
			}
			emisor.EmitirResultadoConContexto(ctx, domain.SolicitudResultadoTecnico{
				Resultado: resultado, Componente: domain.ComponenteIncidenciaGrxFirma, Etapa: domain.EtapaIncidenciaPeticion})
		}
	}
	return observar
}

// nombreServidorTLSValido admite vacio (se verifica el host de la URL), un
// nombre DNS sin punto final ni puerto o una direccion IP literal.
func nombreServidorTLSValido(nombre string) bool {
	if nombre == "" {
		return true
	}
	if len(nombre) > 253 {
		return false
	}
	if net.ParseIP(nombre) != nil {
		return true
	}
	for _, etiqueta := range strings.Split(nombre, ".") {
		if etiqueta == "" || len(etiqueta) > 63 || etiqueta[0] == '-' || etiqueta[len(etiqueta)-1] == '-' {
			return false
		}
		for i := 0; i < len(etiqueta); i++ {
			c := etiqueta[i]
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	return true
}
