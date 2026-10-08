package bootstrap

import (
	"context"
	"net/http"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/firmaemisorv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// Vía externa de la firma V2 (4c-5, PR 2): piezas para que 4c-8 componga el
// registro de RRHH de una firma hecha en el portafirmas. En la misma petición
// RRHH consulta las firmas R5 V2 y obtiene las decisiones interior y exterior
// de firma_externa.registrar; todas con el perfil fijo de la vía externa
// (PR 1), sólo de organización, y con el motivo propio de la firma V2.

// nuevoPerfilFijoFirmaExternaV2CTDesarrollo prepara el contrato exacto de
// las tres rutas R5. La composición decide si publica o consume esta versión;
// el constructor no escribe ni repara una versión de rol divergente.
func nuevoPerfilFijoFirmaExternaV2CTDesarrollo(principal dominiovec.Principal,
	base ports.ContextoAutorizacionAltaV3, ahora time.Time,
) (*perfilFijoCTDesarrollo, error) {
	p, err := nuevoPerfilFijoCTDesarrollo(principal, base, ahora, clavePerfilFijoFirmaExternaV2CTDesarrollo,
		[]string{httpinterno.RutaOriginalFirmableCT, httpinterno.RutaPreflightFirmaR5, httpinterno.RutaRegistroFirmaExterna},
		func(actor, perfil string) (dominiovec.InstantaneaAutorizacion, error) {
			return nuevaInstantaneaFirmaExternaV2CTDesarrollo(actor, perfil, ahora)
		})
	if err != nil {
		return nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	p.metodo = http.MethodPost
	return p, nil
}

func rutaFirmaExternaV2CTDesarrollo(ruta string) bool {
	return ruta == httpinterno.RutaRegistroFirmaExterna || ruta == httpinterno.RutaPreflightFirmaR5
}

// accionFirmaExternaV2CTDesarrollo: la acción que se audita en la ruta.
func accionFirmaExternaV2CTDesarrollo(ruta string) (string, bool) {
	switch ruta {
	case httpinterno.RutaRegistroFirmaExterna:
		return ports.AccionRegistrarFirmaExterna, true
	case httpinterno.RutaPreflightFirmaR5:
		return ports.AccionConsultarFirmasR5V2, true
	}
	return "", false
}

func motivoFirmaV2CTDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_firma_v2_ct", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("firma-v2-ct-desarrollo-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "firma-v2-ct"),
	}
}

// solicitudAutorizacionFirmaExternaV2CTDesarrolloValida es lo único que el PDP
// de CT admite en la ruta: el motivo propio, la organización del perfil como
// único ámbito y, o bien registrar la firma externa sobre la operación de su
// clave con la huella del material y la del descriptor (interior) o la del
// plan (exterior), o bien la consulta R5 V2 previa de un expediente. AD170,
// AD177 y CT172 recalculan la huella exacta en el consumo.
func solicitudAutorizacionFirmaExternaV2CTDesarrolloValida(datos dominiovec.DatosSolicitudAutorizacionLigadaV3) bool {
	r := datos.Recurso
	if datos.Finalidad != ports.FinalidadFirmaDocumento || datos.ReferenciaMotivo != motivoFirmaV2CTDesarrollo() ||
		r.Validar() != nil || r.ModuloID != ports.ModuloContratacion || len(r.Ambitos) != 1 ||
		r.Ambitos["organizacion_ref"] != organizacionAltaContratacionTemporalDesarrollo ||
		!ctdomain.HuellaSHA256FirmaValida(r.Atributos["material_sha256"]) {
		return false
	}
	switch datos.Accion {
	case ports.AccionRegistrarFirmaExterna:
		clave, conPrefijo := strings.CutPrefix(r.Referencia, ports.PrefijoRecursoFirmaExterna)
		descriptor, plan := r.Atributos["descriptor_firma_sha256"], r.Atributos["plan_firma_sha256"]
		return r.Tipo == ports.TipoRecursoFirmaExterna && conPrefijo && ports.ClaveIdempotenciaFirmaValida(clave) &&
			len(r.Atributos) == 2 && (ctdomain.HuellaSHA256FirmaValida(descriptor) != ctdomain.HuellaSHA256FirmaValida(plan))
	case ports.AccionConsultarFirmasR5V2:
		return r.Tipo == ports.TipoRecursoConsultaFirmasR5 && ctdomain.ReferenciaOpacaValida(r.Referencia) && len(r.Atributos) == 1
	}
	return false
}

// ambitosPerfilFijoCTDesarrollo da al emisor V2 la asignación publicada del
// perfil fijo, la misma que consume el PDP, antes de pedir la decisión: el
// emisor compara con ella los ámbitos del recurso. Sólo responde por ese
// perfil y esa persona; cualquier otra petición es indisponibilidad.
type ambitosPerfilFijoCTDesarrollo struct {
	soporte *soporteAltaContratacionTemporalDesarrollo
	perfil  *perfilFijoCTDesarrollo
}

var _ puertosvec.FuenteAutorizacion = ambitosPerfilFijoCTDesarrollo{}

func (a ambitosPerfilFijoCTDesarrollo) ObtenerInstantaneaAutorizacion(ctx context.Context, principalID, perfilRef string) (dominiovec.InstantaneaAutorizacion, error) {
	if a.soporte == nil || a.perfil == nil || contextoInterfazNulo(ctx) || perfilRef != a.perfil.perfilRef() {
		return dominiovec.InstantaneaAutorizacion{}, puertosvec.ErrFuenteAutorizacionNoDisponible
	}
	i, estado := a.soporte.consumirPerfilFijoCTDesarrolloConEstado(ctx, a.perfil)
	if estado != perfilFijoConsumoVigente || i.AsignacionPerfil.PrincipalID != principalID || i.AsignacionPerfil.PerfilActivoRef != perfilRef {
		return dominiovec.InstantaneaAutorizacion{}, puertosvec.ErrFuenteAutorizacionNoDisponible
	}
	return i, nil
}

// nuevoEmisorFirmaExternaV2CTDesarrollo arma el emisor de la vía externa con
// el perfil fijo ya registrado y los emisores de material de sus dos
// audiencias (consulta R5 V2 y firma_externa.v2). Los ámbitos salen de la
// asignación publicada del perfil (NuevoEmisorConAmbitos).
func nuevoEmisorFirmaExternaV2CTDesarrollo(s *soporteAltaContratacionTemporalDesarrollo, fijo *perfilFijoCTDesarrollo,
	consulta, firmaExterna *emisorMaterialRenovableCTDesarrollo, reloj relojContratacionTemporalDesarrollo, proceso string,
) (*firmaemisorv2.Emisor, *fuenteNominalFirmasR5V2CTDesarrollo, error) {
	if s == nil || fijo == nil || consulta == nil || firmaExterna == nil || proceso == "" ||
		fijo.clave != clavePerfilFijoFirmaExternaV2CTDesarrollo ||
		fijo.metodo != http.MethodPost ||
		!fijo.atiende(httpinterno.RutaOriginalFirmableCT) ||
		!fijo.atiende(httpinterno.RutaPreflightFirmaR5) ||
		!fijo.atiende(httpinterno.RutaRegistroFirmaExterna) ||
		fijo.plantilla.VersionRol.RolID != rolFirmaExternaRegistroCTDesarrollo {
		return nil, nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	esperada, err := nuevaInstantaneaFirmaExternaV2CTDesarrollo(fijo.plantilla.AsignacionPerfil.PrincipalID,
		fijo.perfilRef(), reloj.Ahora())
	if err != nil || !mismasHuellasInstantaneaDesarrollo(esperada, fijo.plantilla) {
		return nil, nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	// La publicación es la fuente de verdad: un rol v1 antiguo de dos
	// concesiones, una asignación revocada o una preimagen divergente no
	// habilitan el emisor ni provocan aquí ningún intento de provisión.
	if i, estado := s.consumirPerfilFijoCTDesarrolloConEstado(context.Background(), fijo); estado != perfilFijoConsumoVigente ||
		i.AsignacionPerfil.PerfilActivoRef != fijo.perfilRef() {
		return nil, nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	fuente := &fuenteNominalFirmasR5V2CTDesarrollo{soporte: s, perfil: fijo, reloj: reloj, proceso: proceso,
		accionDeRuta: accionFirmaExternaV2CTDesarrollo, motivo: motivoFirmaV2CTDesarrollo(), consultaPrevia: true}
	emisor, err := firmaemisorv2.NuevoEmisorConAmbitos(fuente, emisoresFirmaExternaV2CTDesarrollo(consulta, firmaExterna),
		motivoFirmaV2CTDesarrollo(), reloj, ambitosPerfilFijoCTDesarrollo{soporte: s, perfil: fijo})
	if err != nil {
		return nil, nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	return emisor, fuente, nil
}

// emisoresFirmaExternaV2CTDesarrollo: cada acción de la vía externa con el
// material de su audiencia; cualquier otra acción no tiene emisor.
func emisoresFirmaExternaV2CTDesarrollo(consulta, firmaExterna *emisorMaterialRenovableCTDesarrollo) *emisorFirmasR5V2CTDesarrollo {
	return &emisorFirmasR5V2CTDesarrollo{porAccion: map[string]*emisorMaterialRenovableCTDesarrollo{
		ports.AccionConsultarFirmasR5V2: consulta, ports.AccionRegistrarFirmaExterna: firmaExterna}}
}
