package bootstrap

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"reflect"
	"strings"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// resolverLectoresNominalesInscripcionRRHH añade lectores solo al resolvedor
// de la bandeja de Bolsa. El resolvedor de CT conserva su lista y sus rutas.
// Se usa la misma CA, el mismo formato de identidad y el mismo perfil derivado.
func resolverLectoresNominalesInscripcionRRHH(
	cfg config.Config, base *resolvedorIdentidadDesarrollo,
	entradas []archivoEntradaConsultaRRHHDesarrollo,
) (*resolvedorIdentidadDesarrollo, error) {
	if base == nil || base.lectoresRRHHInvalidos {
		return nil, errMontajeInscripcionBolsa
	}
	if len(entradas) == 0 {
		return base, nil
	}
	caPEM, err := leerFicheroMaterialSeguro(cfg.DevelopmentPaths().CACertificate, tamanoMaximoFicheroMaterialDesarrollo)
	if err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	ca, err := decodificarCertificadoUnico(caPEM)
	if err != nil || ca == nil || !ca.IsCA || ca.KeyUsage&x509.KeyUsageCertSign == 0 {
		return nil, errMontajeInscripcionBolsa
	}
	raices := x509.NewCertPool()
	raices.AddCert(ca)
	local := *base
	local.porHuella = make(map[[sha256.Size]byte]vecdomain.Principal, len(base.porHuella)+len(entradas))
	for huella, principal := range base.porHuella {
		local.porHuella[huella] = principal
	}
	local.lectoresRRHH = make(map[[sha256.Size]byte]identidadConsultaRRHHDesarrollo, len(base.lectoresRRHH)+len(entradas))
	for huella, lector := range base.lectoresRRHH {
		local.lectoresRRHH[huella] = lector
	}
	porSujeto := make(map[string]bool, len(local.porHuella)+len(entradas))
	for _, principal := range local.porHuella {
		porSujeto[principal.ID] = true
	}
	lectores := make([]identidadConsultaRRHHDesarrollo, 0, len(entradas))
	for _, entrada := range entradas {
		certRuta, okCert := rutaMaterialConsultaRRHHDesarrollo(cfg.DevelopmentMaterialDir, entrada.Certificate)
		identidadRuta, okIdentidad := rutaMaterialConsultaRRHHDesarrollo(cfg.DevelopmentMaterialDir, entrada.Identity)
		clase, okClase := claseAmbitoConsultaRRHHDesarrollo(entrada.ClaseAmbito)
		if !okCert || !okIdentidad || !okClase || entrada.Subject == "" ||
			!domain.ReferenciaOpacaValida(entrada.PerfilRef) || !domain.ReferenciaOpacaValida(entrada.OrganizacionRef) ||
			!domain.ReferenciaOpacaValida(entrada.AmbitoRef) ||
			entrada.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
			(clase == ports.AmbitoOrganizacionRRHH && entrada.AmbitoRef != entrada.OrganizacionRef) {
			return nil, errMontajeInscripcionBolsa
		}
		pem, err := leerFicheroMaterialSeguro(certRuta, tamanoMaximoFicheroMaterialDesarrollo)
		if err != nil {
			return nil, errMontajeInscripcionBolsa
		}
		cert, err := decodificarCertificadoUnico(pem)
		if err != nil || cert == nil {
			return nil, errMontajeInscripcionBolsa
		}
		if _, err = cert.Verify(x509.VerifyOptions{Roots: raices, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}); err != nil {
			return nil, errMontajeInscripcionBolsa
		}
		huella := sha256.Sum256(cert.Raw)
		identidad, err := cargarIdentidadDesarrollo(identidadRuta, cert, "lector_rrhh")
		if err != nil || identidad.principal.ID != entrada.Subject ||
			entrada.PerfilRef != referenciaAltaContratacionTemporalDesarrollo("prf_",
				entrada.Subject+"\x00"+hex.EncodeToString(huella[:])+"\x00perfil") {
			return nil, errMontajeInscripcionBolsa
		}
		if existente, presente := local.porHuella[huella]; presente {
			// El cargador global ya pudo registrar el actor, pero nunca lo hizo
			// lector de CT. Solo el mismo certificado y la misma identidad valen.
			if _, lectorCT := base.lectoresRRHH[huella]; lectorCT ||
				!reflect.DeepEqual(existente, identidad.principal) {
				return nil, errMontajeInscripcionBolsa
			}
		} else if porSujeto[entrada.Subject] {
			return nil, errMontajeInscripcionBolsa
		}
		local.porHuella[huella] = identidad.principal
		porSujeto[entrada.Subject] = true
		lectores = append(lectores, identidadConsultaRRHHDesarrollo{
			identidad: identidad, perfilRef: entrada.PerfilRef,
			organizacionRef: entrada.OrganizacionRef, clase: clase, ambitoRef: entrada.AmbitoRef,
		})
	}
	if err := local.registrarLectoresRRHH(lectores); err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	return &local, nil
}

// registrarActoresGlobalesInscripcionRRHH incorpora solo la identidad mTLS
// validada para la carcasa común. La lista nominal de CT permanece intacta;
// las cuentas, sesiones y autorizaciones de Bolsa se componen después.
func registrarActoresGlobalesInscripcionRRHH(
	cfg config.Config, base *resolvedorIdentidadDesarrollo,
) (*resolvedorIdentidadDesarrollo, error) {
	activo, err := cfg.BolsaInscripcionesInternoActivo()
	if err != nil || !activo {
		return base, err
	}
	c, err := cargarConfiguracionInscripcionRRHHDesarrollo(cfg)
	if err != nil {
		return base, errMontajeInscripcionBolsa
	}
	local, err := resolverLectoresNominalesInscripcionRRHH(cfg, base, c.Lectores)
	if err != nil {
		return base, errMontajeInscripcionBolsa
	}
	rrhh, err := cuentasRRHHInscripcionDesarrollo(local, c.Cuentas)
	if err != nil {
		return base, errMontajeInscripcionBolsa
	}
	for _, entrada := range c.Lectores {
		coincidencias := 0
		for i, lector := range rrhh {
			if lector.identidad.principal.ID == entrada.Subject && lector.perfilRef == entrada.PerfilRef {
				cuenta := c.Cuentas[i].CuentaRef
				if !strings.HasPrefix(cuenta, "cta_") || !domain.ReferenciaOpacaValida(cuenta) {
					return base, errMontajeInscripcionBolsa
				}
				coincidencias++
			}
		}
		if coincidencias != 1 {
			return base, errMontajeInscripcionBolsa
		}
	}
	global := *base
	global.porHuella = local.porHuella
	return &global, nil
}
