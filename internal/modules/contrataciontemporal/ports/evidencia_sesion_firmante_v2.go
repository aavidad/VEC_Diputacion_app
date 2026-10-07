package ports

import (
	"context"
	"time"

	vd "vec-diputacion-granada/internal/vec/domain"
)

// SolicitudSesionFirmanteV2 aporta sólo cotejos esperados. La fuente común
// obtiene la identidad de su propia sesión autenticada y registrada; estos
// campos nunca son una autoridad para crear o elegir una cuenta, perfil o rol.
type SolicitudSesionFirmanteV2 struct {
	CertificadoCanalSHA256, PersonaEsperadaRef string
	CanalTLSVinculadoSHA256                    string
	CuentaEsperadaRef, PerfilEsperadoRef       string
	RolEsperadoID                              string
	CertificadoVerificadoEn                    time.Time
	CertificadoTLSValidoHasta                  time.Time
}

// EvidenciaSesionFirmanteV2 procede de la autoridad común de identidad. El
// vínculo conserva política, garantía y control de sesión revalidado; la fuente
// informa además hasta cuándo vale el certificado observado en ese canal.
type EvidenciaSesionFirmanteV2 struct {
	Vinculo                vd.VinculoAutenticacionActorV2
	Resultado              vd.ResultadoContextoActorRegistradoV2
	CertificadoCanalSHA256 string
	CertificadoValidoHasta time.Time
}

// SesionFirmanteV2 reconsulta el registro común en cada uso. El adaptador
// sintético abre una sola alta por petición y revalida ese mismo registro;
// uno corporativo obtiene su sesión ya autenticada sin crear otra identidad.
type SesionFirmanteV2 interface {
	RevalidarSesionFirmanteV2(context.Context) (EvidenciaSesionFirmanteV2, error)
}

// FuenteSesionFirmanteV2 es una dependencia de identidad de CT, no una nueva
// autoridad. La composición inyecta un proveedor común según el canal real.
type FuenteSesionFirmanteV2 interface {
	AbrirSesionFirmanteV2(context.Context, SolicitudSesionFirmanteV2) (SesionFirmanteV2, error)
}
