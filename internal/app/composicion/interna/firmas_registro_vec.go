package interna

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"vec-diputacion-granada/internal/app/composicion/identidadordinaria"
	"vec-diputacion-granada/internal/app/composicion/interna/contrataciontemporal/firmavec"
	"vec-diputacion-granada/internal/app/composicion/internactproveedores"
	"vec-diputacion-granada/internal/app/composicion/internagobierno"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

var errRegistroFirmaVecGobernadoNoDisponible = errors.New("composicion interna: registro firma vec no disponible")

// El root aporta sólo autoridades que ya posee. El origen HTTPS procede de
// configuración privada aprobada; vacío deja la ruta sin componer.
type dependenciasRegistroFirmaVecGobernado struct {
	Configuracion                     Configuracion
	OrigenHTTPS                       string
	Identidad                         *httpseguridad.ServicioIdentidad
	Fachada                           *FachadaIdentidadOffline
	Extractor                         *extractorCertificadoPersonalDirecto
	Revalidador                       core.RevalidadorAutenticacionActorV1
	Resolutor                         core.ResolutorContextoActorRegistradoV2
	ProveedoresCT                     *internactproveedores.Proveedores
	Contextos                         map[string]internagobierno.ContextoNominal
	PoliticaRef, PoliticaHuellaSHA256 string
	Auditoria                         vp.RegistradorAuditoriaFronteraRutaExacta
	Reloj                             vp.Reloj
	ManejadorCT                       http.Handler
}

// nuevoPuenteRegistroFirmaVecGobernado utiliza el emisor y registro de la
// misma identidad C4. La fuente V3 es exactamente la del PDP CT ya montado.
// Esta función no crea pools, certificados, perfiles ni concesiones.
func nuevoPuenteRegistroFirmaVecGobernado(d dependenciasRegistroFirmaVecGobernado) (http.Handler, ctports.FuenteSesionFirmanteV2, error) {
	if d.Configuracion.Validar() != nil || d.Identidad == nil || d.Fachada == nil ||
		d.Fachada.servicio != d.Identidad || d.Extractor == nil || d.Extractor.emisor == nil ||
		d.Extractor.registro == nil || d.ProveedoresCT == nil ||
		interfazNulaIdentidadOffline(d.Revalidador) || interfazNulaIdentidadOffline(d.Resolutor) ||
		interfazNulaIdentidadOffline(d.Auditoria) || interfazNulaIdentidadOffline(d.Reloj) ||
		manejadorNulo(d.ManejadorCT) || !origenRegistroFirmaVecGobernadoValido(d.Configuracion, d.OrigenHTTPS) ||
		d.PoliticaRef == "" || !strings.HasPrefix(d.PoliticaHuellaSHA256, "sha256:") ||
		d.Extractor.emisorID != d.Configuracion.EmisorIdentidad ||
		d.Extractor.audiencia != d.Configuracion.Audiencia ||
		!d.Extractor.retirada.Equal(d.Configuracion.RetiradaPoliticaInternaEn) {
		return nil, nil, errRegistroFirmaVecGobernadoNoDisponible
	}
	autorizacion := d.ProveedoresCT.FuenteAutorizacionV3()
	if interfazNulaIdentidadOffline(autorizacion) || len(d.Contextos) == 0 || len(d.Contextos) > 512 {
		return nil, nil, errRegistroFirmaVecGobernadoNoDisponible
	}
	porCuenta := make(map[string]identidadordinaria.PerfilNominalAsignacionVigente, len(d.Contextos))
	for cuenta, nominal := range d.Contextos {
		if cuenta == "" || nominal.PerfilActivoRef == "" || nominal.OrganizacionRef == "" || nominal.UnidadRef == "" {
			return nil, nil, errRegistroFirmaVecGobernadoNoDisponible
		}
		porCuenta[cuenta] = identidadordinaria.PerfilNominalAsignacionVigente{PerfilActivoRef: nominal.PerfilActivoRef}
	}
	entorno, err := firmavec.NuevaIdentidadCertificadoFirmaVecV2(firmavec.ConfiguracionIdentidadCertificadoFirmaVecV2{
		Fuente: identidadordinaria.ConfiguracionFuenteCertificadoTemporalConAsignacionVigente{
			Identidad: d.Identidad, Revalidador: d.Revalidador, Resolutor: d.Resolutor,
			Autorizacion: autorizacion, PorCuenta: porCuenta,
			Politica: identidadordinaria.PoliticaCertificadoTemporal{
				Referencia:   d.PoliticaRef,
				HuellaSHA256: strings.TrimPrefix(d.PoliticaHuellaSHA256, "sha256:"),
				RetiradaEn:   d.Configuracion.RetiradaPoliticaInternaEn,
			},
		},
		Vinculador: d.Fachada, Emisor: d.Extractor.emisor,
		Acreditador: acreditadorRegistroCertificadoFirmaVec{registro: d.Extractor.registro},
		Auditoria:   d.Auditoria, Origen: d.OrigenHTTPS,
		EmisorID: d.Extractor.emisorID, Audiencia: d.Extractor.audiencia,
		Reloj: d.Reloj,
	})
	if err != nil {
		return nil, nil, errRegistroFirmaVecGobernadoNoDisponible
	}
	puente, err := entorno.Envolver(d.ManejadorCT)
	if err != nil {
		return nil, nil, errRegistroFirmaVecGobernadoNoDisponible
	}
	return puente, entorno.FuenteSesionFirmanteV2(), nil
}

func origenRegistroFirmaVecGobernadoValido(cfg Configuracion, origen string) bool {
	if origen == "" || cfg.NombreServidorTLS == "" {
		return false
	}
	u, err := url.Parse(origen)
	return err == nil && u.Scheme == "https" && u.Host != "" &&
		u.User == nil && u.Path == "" && u.RawPath == "" && u.RawQuery == "" &&
		u.Fragment == "" && u.Opaque == "" && u.String() == origen &&
		net.ParseIP(u.Hostname()) == nil && strings.EqualFold(u.Hostname(), cfg.NombreServidorTLS)
}

type acreditadorRegistroCertificadoFirmaVec struct {
	registro *registroCertificadosPersonales
}

func (a acreditadorRegistroCertificadoFirmaVec) AcreditarCertificadoFirmaVecV2(ctx context.Context,
	hoja, autoridad *x509.Certificate, ahora time.Time,
) (firmavec.AcreditacionCertificadoFirmaVecV2, error) {
	var vacia firmavec.AcreditacionCertificadoFirmaVecV2
	if a.registro == nil || ctx == nil || ctx.Err() != nil || hoja == nil || autoridad == nil ||
		len(hoja.Raw) == 0 || a.registro.comprobarCRL(hoja, autoridad, ahora) != nil {
		return vacia, errRegistroFirmaVecGobernadoNoDisponible
	}
	huellaBytes := sha256.Sum256(hoja.Raw)
	huella := "sha256:" + hex.EncodeToString(huellaBytes[:])
	registrado, err := a.registro.resolver(ctx, huella)
	if err != nil || registrado.HuellaSHA256 != huella || registrado.SujetoID == "" || registrado.CuentaID == "" || ctx.Err() != nil {
		return vacia, errRegistroFirmaVecGobernadoNoDisponible
	}
	return firmavec.AcreditacionCertificadoFirmaVecV2{PersonaRef: registrado.SujetoID, CuentaRef: registrado.CuentaID}, nil
}
