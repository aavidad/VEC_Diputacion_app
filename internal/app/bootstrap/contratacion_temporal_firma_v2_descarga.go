package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/auditoriafirma"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/plannominal"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/consultafirmasv2"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type selectorDescargaFirmaR5V2CT struct {
	nominal *fuenteNominalSesionFirmaCT
	plan    *plannominal.Fuente
}

// El paso para la capacidad de consulta pertenece a quien consulta. La firma
// pedida puede ser de otro paso, pero sólo sale de la historia que R5 le dejó
// leer en esta petición. Ninguna selección de paso llega desde el navegador.
func (s selectorDescargaFirmaR5V2CT) SeleccionarConsultaDescarga(ctx context.Context,
	q consultafirmasv2.SolicitudDescarga) (consultafirmasv2.SeleccionDescarga, error) {
	var vacia consultafirmasv2.SeleccionDescarga
	if ctx == nil || s.nominal == nil || s.plan == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(s.nominal.asignaciones) || q.Validar() != nil {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	documentos, err := s.nominal.contextoPerfil(ctx, "documentos")
	if err != nil {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	firmante, err := s.nominal.contexto(ctx)
	if err != nil || documentos.Resultado.Contexto.PersonaRef != firmante.Resultado.Contexto.PersonaRef {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	instantanea, err := s.nominal.asignaciones.ObtenerInstantaneaAutorizacion(ctx,
		firmante.Resultado.Contexto.PersonaRef, firmante.Resultado.Contexto.PerfilActivoRef)
	if err != nil || instantanea.Validar() != nil || instantanea.VersionRol.RolID == "" ||
		instantanea.AsignacionPerfil.PrincipalID != firmante.Resultado.Contexto.PersonaRef ||
		instantanea.AsignacionPerfil.PerfilActivoRef != firmante.Resultado.Contexto.PerfilActivoRef {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	organizacion, unidad, err := s.nominal.ResolverAmbitosConsultaFirmaVec(ctx)
	if err != nil || organizacion == "" || unidad == "" {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	publicado, err := s.plan.Plan(ctx)
	if err != nil || publicado.Validar() != nil {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	orden, catalogoHuella, err := seleccionarPasoDescargaFirmaV2(publicado.Pasos,
		q.Documento, instantanea.VersionRol.RolID, organizacion, unidad)
	if err != nil {
		return vacia, err
	}
	var aleatorio [16]byte
	if _, err := rand.Read(aleatorio[:]); err != nil {
		return vacia, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	consulta := consultafirmasv2.Solicitud{ExpedienteRef: q.ExpedienteRef,
		VersionExpediente: q.VersionObservada, Documento: q.Documento,
		PasoOrden: orden, ClaveIdempotencia: "consulta-descarga-" + hex.EncodeToString(aleatorio[:]),
		CatalogoHuella: catalogoHuella, Via: ports.ViaFirmaCertificadoVEC}
	if consulta.Validar() != nil || ctx.Err() != nil {
		return vacia, ports.ErrFirmaDocumentoDenegada
	}
	return consultafirmasv2.SeleccionDescarga{OrganizacionRef: organizacion, Consulta: consulta}, nil
}

func seleccionarPasoDescargaFirmaV2(pasos []ctdomain.CompetenciaPasoFirmaV2,
	documento, rol, organizacion, unidad string) (int, string, error) {
	var orden int
	var catalogoHuella string
	coincidencias := 0
	for _, paso := range pasos {
		if paso.Documento == documento && paso.RolID == rol &&
			paso.OrganizacionRef == organizacion && paso.UnidadRef == unidad &&
			paso.PasoOrden >= 1 && paso.PasoOrden <= ports.MaximoPasosFirmasVerificadasV2 {
			orden, catalogoHuella = int(paso.PasoOrden), paso.Circuito.HuellaSHA256
			coincidencias++
		}
	}
	if coincidencias != 1 {
		return 0, "", ports.ErrFirmaDocumentoDenegada
	}
	return orden, catalogoHuella, nil
}

// Se llama desde el montaje R5 existente, una vez provistos el lector, el
// registrador de intentos y la fuente V3 del PDF. No publica por sí sola.
func nuevaRutaDescargaFirmaR5V2CT(m *montajeFirmaVecR5CT, pdf ports.FuentePDFFirmaAnterior,
	registro ports.RegistroFirmasVerificadasV2, intentos vecports.RegistradorIntentosAuditoria) (vechttp.RutaExacta, error) {
	var vacia vechttp.RutaExacta
	if m == nil || m.nominal == nil || m.nominal.proceso == "" || m.plan == nil || m.consulta == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(pdf) || dependenciaEsNulaContratacionTemporalDesarrollo(registro) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(intentos) {
		return vacia, errMontajeFirmaVecR5
	}
	auditado, err := auditoriafirma.Nuevo(registro, intentos, m.nominal)
	if err != nil {
		return vacia, err
	}
	consulta, err := consultafirmasv2.Nuevo(m.nominal, m.consulta, auditado)
	if err != nil {
		return vacia, err
	}
	selector := selectorDescargaFirmaR5V2CT{nominal: m.nominal, plan: m.plan}
	servicio, err := consultafirmasv2.NuevaDescarga(selector, consulta, pdf)
	if err != nil {
		return vacia, err
	}
	h, err := httpinterno.NuevoManejadorDescargaFirmaR5V2(servicio, intentos, m.nominal, m.nominal)
	if err != nil {
		return vacia, err
	}
	return vechttp.RutaExacta{Ruta: httpinterno.RutaDescargaFirmaR5V2, Manejador: m.nominal.envolver(h)}, nil
}

var _ consultafirmasv2.SelectorDescarga = selectorDescargaFirmaR5V2CT{}
