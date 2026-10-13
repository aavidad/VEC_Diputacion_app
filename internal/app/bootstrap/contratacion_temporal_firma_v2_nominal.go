package bootstrap

import (
	"context"
	"net/http"
	"sync"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/firmaemisorv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/consultafirmasv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// Es una selección privada de identidades ya provistas, no una concesión.
// CA/IS/F1 y la asignación vigente contrastan todas sus referencias por uso.
type seleccionNominalFirmaCT struct {
	CertificadoSHA256         string `json:"certificado_sha256"`
	PersonaRef                string `json:"persona_ref"`
	CuentaRef                 string `json:"cuenta_ref"`
	CuentaID                  string `json:"cuenta_id"`
	PerfilRef                 string `json:"perfil_firma_ref"`
	PerfilConsultaCTRef       string `json:"perfil_consulta_ct_ref"`
	PerfilConsultaCircuitoRef string `json:"perfil_consulta_circuito_ref"`
	PerfilDocumentosRef       string `json:"perfil_documentos_ref"`
}

type clavePeticionNominalFirmaCT struct{}
type peticionNominalFirmaCT struct {
	fuente    *fuenteNominalSesionFirmaCT
	solicitud ports.SolicitudSesionFirmanteV2
	sesion    ports.SesionFirmanteV2
	peticion  *http.Request
	extras    map[string]ports.SesionFirmanteV2
	mu        sync.Mutex
}

type fuenteNominalSesionFirmaCT struct {
	autoridad      *autoridadSesionFirmanteV2
	fuente         *fuenteSinteticaSesionFirmanteV2
	asignaciones   vp.FuenteAutorizacion
	porCertificado map[string]seleccionNominalFirmaCT
	reloj          ports.Reloj
	proceso        string
	lecturas       *montajeLecturasFirmaV2
}

func nuevaFuenteNominalSesionFirmaCT(a *autoridadSesionFirmanteV2, asignaciones vp.FuenteAutorizacion,
	selectores []seleccionNominalFirmaCT, reloj ports.Reloj) (*fuenteNominalSesionFirmaCT, error) {
	if a == nil || dependenciaEsNulaContratacionTemporalDesarrollo(asignaciones) || reloj == nil || len(selectores) == 0 || len(selectores) > 64 {
		return nil, errSesionFirmanteV2NoDisponible
	}
	f, ok := a.fuente.(*fuenteSinteticaSesionFirmanteV2)
	if !ok || f == nil || a.garantia != core.AuthAssuranceSubstantial {
		return nil, errSesionFirmanteV2NoDisponible
	}
	porCert := make(map[string]seleccionNominalFirmaCT, len(selectores))
	cuentas := make(map[string]bool, len(selectores))
	perfiles := make(map[string]bool, len(selectores)*4)
	for _, s := range selectores {
		if !huellaSHA256ValidaContratacionTemporalDesarrollo(s.CertificadoSHA256) || s.PersonaRef == "" || s.CuentaRef == "" ||
			s.CuentaID == "" || !perfilActivoSeguridadComunValido(s.PerfilRef) || !perfilActivoSeguridadComunValido(s.PerfilConsultaCTRef) ||
			!perfilActivoSeguridadComunValido(s.PerfilConsultaCircuitoRef) || !perfilActivoSeguridadComunValido(s.PerfilDocumentosRef) ||
			porCert[s.CertificadoSHA256].CuentaRef != "" || cuentas[s.CuentaRef] {
			return nil, errSesionFirmanteV2NoDisponible
		}
		for _, perfil := range []string{s.PerfilRef, s.PerfilConsultaCTRef, s.PerfilConsultaCircuitoRef, s.PerfilDocumentosRef} {
			if perfiles[perfil] {
				return nil, errSesionFirmanteV2NoDisponible
			}
			perfiles[perfil] = true
		}
		porCert[s.CertificadoSHA256] = s
		cuentas[s.CuentaRef] = true
	}
	f.cuentasID = make(map[string]string, len(porCert))
	f.selecciones = make(map[string]seleccionNominalFirmaCT, len(porCert))
	for h, s := range porCert {
		f.cuentasID[h] = s.CuentaID
		f.selecciones[h] = s
	}
	return &fuenteNominalSesionFirmaCT{autoridad: a, fuente: f, asignaciones: asignaciones, porCertificado: porCert, reloj: reloj}, nil
}

// Envolver no publica sesiones ni asignaciones al arrancar. Registra la
// autenticación de esta petición y conserva su garantía sustancial observada.
func (f *fuenteNominalSesionFirmaCT) envolver(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		preparada, err := f.prepararPeticionNominal(r)
		if err != nil || siguiente == nil {
			responderNominalFirmaCT(w, http.StatusForbidden)
			return
		}
		siguiente.ServeHTTP(w, preparada)
	})
}

// También la cabecera del portal usa esta misma sesión F1, sin otra fuente
// de identidad y sin tomar cargos o permisos de la selección JSON.
func (f *fuenteNominalSesionFirmaCT) prepararPeticionNominal(r *http.Request) (*http.Request, error) {
	if f == nil || r == nil {
		return nil, ports.ErrFirmaDocumentoDenegada
	}
	r = r.WithContext(context.WithValue(r.Context(), claveAcreditacionCertificadoFirmaCT{}, &acreditacionCertificadoPorPeticionCT{}))
	if err := f.autoridad.preacreditar(r); err != nil {
		return nil, err
	}
	hasta, huella, ok := certificadoSesionFirmanteV2(r, f.reloj.Ahora())
	seleccion, presente := f.porCertificado[huella]
	if !ok || !presente {
		return nil, ports.ErrFirmaDocumentoDenegada
	}
	snapshot, err := f.asignaciones.ObtenerInstantaneaAutorizacion(r.Context(), seleccion.PersonaRef, seleccion.PerfilRef)
	if err != nil {
		return nil, err
	}
	if snapshot.Validar() != nil || snapshot.AsignacionPerfil.PrincipalID != seleccion.PersonaRef || snapshot.AsignacionPerfil.PerfilActivoRef != seleccion.PerfilRef {
		return nil, ports.ErrFirmaDocumentoDenegada
	}
	q := ports.SolicitudSesionFirmanteV2{CertificadoCanalSHA256: huella, PersonaEsperadaRef: seleccion.PersonaRef,
		CuentaEsperadaRef: seleccion.CuentaRef, PerfilEsperadoRef: seleccion.PerfilRef, RolEsperadoID: snapshot.VersionRol.RolID,
		CertificadoVerificadoEn: f.reloj.Ahora(), CertificadoTLSValidoHasta: hasta}
	sesion, err := f.fuente.AbrirSesionFirmanteV2(r.Context(), q)
	if err != nil {
		return nil, err
	}
	nominal := &peticionNominalFirmaCT{fuente: f, solicitud: q, sesion: sesion, peticion: r, extras: make(map[string]ports.SesionFirmanteV2)}
	ctx := context.WithValue(r.Context(), clavePeticionNominalFirmaCT{}, nominal)
	if _, err := f.contexto(ctx); err != nil {
		return nil, err
	}
	return r.WithContext(ctx), nil
}

func (f *fuenteNominalSesionFirmaCT) admitePrincipal(p core.Principal) bool {
	if f == nil || !principalCertificadoFirmaVecDesarrolloValido(p) {
		return false
	}
	s, ok := f.porCertificado[p.Attributes["certificate_sha256"]]
	return ok && s.PersonaRef == p.ID
}

func (f *fuenteNominalSesionFirmaCT) principalCanal(r *http.Request) (core.Principal, bool) {
	if f == nil || f.fuente == nil {
		return core.Principal{}, false
	}
	p, ok := f.fuente.principalSeleccionado(r)
	return p, ok && f.admitePrincipal(p)
}

func responderNominalFirmaCT(w http.ResponseWriter, estado int) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(estado)
}

func (f *fuenteNominalSesionFirmaCT) contexto(ctx context.Context) (ports.EvidenciaSesionFirmanteV2, error) {
	var cero ports.EvidenciaSesionFirmanteV2
	if f == nil || ctx == nil || ctx.Err() != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	p, ok := ctx.Value(clavePeticionNominalFirmaCT{}).(*peticionNominalFirmaCT)
	if !ok || p == nil || p.fuente != f || p.peticion == nil || dependenciaEsNulaContratacionTemporalDesarrollo(p.sesion) {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	// La auditoría de una descarga puede sobrevivir a la cancelación HTTP.
	// Revalidar la misma hoja TLS y AUT56 con el contexto sin cancelación de
	// esa petición conserva actor y certificado, sin abrir una identidad nueva.
	if f.autoridad.preacreditar(p.peticion.WithContext(ctx)) != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	e, err := p.sesion.RevalidarSesionFirmanteV2(ctx)
	if err != nil || e.Vinculo.ValidarPara(e.Resultado) != nil || !e.Vinculo.VigenteEn(f.reloj.Ahora(), e.Resultado) ||
		e.CertificadoCanalSHA256 != p.solicitud.CertificadoCanalSHA256 || e.Resultado.Contexto.PersonaRef != p.solicitud.PersonaEsperadaRef ||
		e.Resultado.Contexto.PerfilActivoRef != p.solicitud.PerfilEsperadoRef || e.Resultado.Contexto.Instantanea.CuentaRef != p.solicitud.CuentaEsperadaRef {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	datos, err := e.Vinculo.Datos()
	if err != nil || datos.GarantiaObservada != core.AuthAssuranceSubstantial || datos.MetodoObservado != core.AuthMethodCertificate ||
		datos.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 || datos.CuentaPrivilegiada {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	snapshot, err := f.asignaciones.ObtenerInstantaneaAutorizacion(ctx, e.Resultado.Contexto.PersonaRef, e.Resultado.Contexto.PerfilActivoRef)
	if err != nil || snapshot.Validar() != nil || snapshot.VersionRol.RolID != p.solicitud.RolEsperadoID {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	return e, nil
}

func (f *fuenteNominalSesionFirmaCT) RevalidarContextoActorFirmaV2(ctx context.Context) (firmaemisorv2.ContextoActorFirmaV2, error) {
	e, err := f.contexto(ctx)
	return firmaemisorv2.ContextoActorFirmaV2{Resultado: e.Resultado, Vinculo: e.Vinculo, CertificadoCanalSHA256: e.CertificadoCanalSHA256}, err
}

func (f *fuenteNominalSesionFirmaCT) ResolverOrganizacionFirmaVec(ctx context.Context) (string, error) {
	e, err := f.contexto(ctx)
	if err != nil {
		return "", err
	}
	i, err := f.asignaciones.ObtenerInstantaneaAutorizacion(ctx, e.Resultado.Contexto.PersonaRef, e.Resultado.Contexto.PerfilActivoRef)
	if err != nil {
		return "", ports.ErrFirmaDocumentoDenegada
	}
	for _, a := range i.AsignacionPerfil.Ambitos {
		if a.Clave == "organizacion_ref" && len(a.Valores) == 1 {
			return a.Valores[0], nil
		}
	}
	return "", ports.ErrFirmaDocumentoDenegada
}

func (f *fuenteNominalSesionFirmaCT) ResolverContextoConsultaFirmasR5V2(ctx context.Context) (consultafirmasv2.Contexto, error) {
	e, err := f.contexto(ctx)
	if err != nil {
		return consultafirmasv2.Contexto{}, err
	}
	org, unidad, err := f.ResolverAmbitosConsultaFirmaVec(ctx)
	return consultafirmasv2.Contexto{OrganizacionRef: org, UnidadRef: unidad, FirmantePrincipalCandidatoRef: e.Resultado.Contexto.PersonaRef}, err
}

// Los dos ámbitos se resuelven juntos desde la misma asignación vigente.
func (f *fuenteNominalSesionFirmaCT) ResolverAmbitosConsultaFirmaVec(ctx context.Context) (string, string, error) {
	e, err := f.contexto(ctx)
	if err != nil {
		return "", "", err
	}
	i, err := f.asignaciones.ObtenerInstantaneaAutorizacion(ctx, e.Resultado.Contexto.PersonaRef, e.Resultado.Contexto.PerfilActivoRef)
	if err != nil || i.Validar() != nil {
		return "", "", ports.ErrFirmaDocumentoDenegada
	}
	var org, unidad string
	for _, a := range i.AsignacionPerfil.Ambitos {
		if len(a.Valores) != 1 {
			return "", "", ports.ErrFirmaDocumentoDenegada
		}
		switch a.Clave {
		case "organizacion_ref":
			org = a.Valores[0]
		case "unidad_ref":
			unidad = a.Valores[0]
		default:
			return "", "", ports.ErrFirmaDocumentoDenegada
		}
	}
	if org == "" || unidad == "" {
		return "", "", ports.ErrFirmaDocumentoDenegada
	}
	return org, unidad, nil
}

func (f *fuenteNominalSesionFirmaCT) GarantiaAutenticacionFirmaV2(ctx context.Context) (string, error) {
	if _, err := f.contexto(ctx); err != nil {
		return "", err
	}
	return "sustancial", nil
}

// La consulta R5 audita incluso una denegación previa al lector durable.
func (f *fuenteNominalSesionFirmaCT) CrearOrdenIntentoFirma(ctx context.Context, intento, accion, recurso string,
	resultado core.ResultadoIntentoAuditoria) (vp.OrdenIntentoAuditoria, error) {
	var vacia vp.OrdenIntentoAuditoria
	if f == nil || f.proceso == "" || accion != ports.AccionConsultarFirmasR5V2 {
		return vacia, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	e, err := f.contexto(ctx)
	if err != nil {
		return vacia, err
	}
	if recurso == "" {
		recurso = organizacionAltaContratacionTemporalDesarrollo
	}
	ref, err := vp.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return vacia, err
	}
	correlacion, err := ref.ValorCanonico()
	if err != nil {
		return vacia, err
	}
	return vp.NuevaOrdenIntentoAuditoria(intento, e.Resultado, e.Vinculo, core.DatosIntentoAuditoria{
		Accion: accion, ModuloID: ports.ModuloContratacion, RecursoRef: recurso,
		FinalidadRef: ports.FinalidadFirmaDocumento, Resultado: resultado,
		Motivo: motivoFirmaV2CTDesarrollo(), Proceso: f.proceso,
		Canal: canalIntentosFirmasR5V2CTDesarrollo, CorrelacionRef: correlacion,
	})
}

func (f *fuenteNominalSesionFirmaCT) GarantiaActoFirmaV2(ctx context.Context, m ports.MaterialFirmaVerificadaV2) (string, error) {
	e, err := f.contexto(ctx)
	if err != nil {
		return "", err
	}
	if m.Via != ports.ViaFirmaCertificadoVEC || m.FirmantePrincipalRef != e.Resultado.Contexto.PersonaRef ||
		m.CuentaFirmanteRef != e.Resultado.Contexto.Instantanea.CuentaRef || m.PerfilActivoOperadorRef != e.Resultado.Contexto.PerfilActivoRef ||
		m.PerfilActivoFirmanteRef != e.Resultado.Contexto.PerfilActivoRef || m.CertificadoHuella != e.CertificadoCanalSHA256 {
		return "", ports.ErrFirmaDocumentoDenegada
	}
	return "sustancial", nil
}

// La autoridad AUT56 consume la misma sesión ya registrada en la petición.
// Toda selección distinta queda denegada antes de emitir material V3.
func (f *fuenteSinteticaSesionFirmanteV2) sesionNominalExistente(ctx context.Context, q ports.SolicitudSesionFirmanteV2) (ports.SesionFirmanteV2, bool, error) {
	p, ok := ctx.Value(clavePeticionNominalFirmaCT{}).(*peticionNominalFirmaCT)
	if !ok || p == nil {
		return nil, false, nil
	}
	if p == nil || p.fuente == nil || p.fuente.fuente != f || p.solicitud.CertificadoCanalSHA256 != q.CertificadoCanalSHA256 ||
		p.solicitud.PersonaEsperadaRef != q.PersonaEsperadaRef || p.solicitud.CuentaEsperadaRef != q.CuentaEsperadaRef ||
		p.solicitud.PerfilEsperadoRef != q.PerfilEsperadoRef || p.solicitud.RolEsperadoID != q.RolEsperadoID {
		return nil, true, ports.ErrFirmaDocumentoDenegada
	}
	return p.sesion, true, nil
}

// Los perfiles auxiliares son elecciones explícitas del mismo actor y cuenta.
// No se suman sus permisos: cada efecto se emite con su propio contexto F1.
func (f *fuenteNominalSesionFirmaCT) contextoPerfil(ctx context.Context, slot string) (ports.ContextoAutorizacionAltaV3, error) {
	var cero ports.ContextoAutorizacionAltaV3
	principal, err := f.contexto(ctx)
	if err != nil {
		return cero, err
	}
	p, ok := ctx.Value(clavePeticionNominalFirmaCT{}).(*peticionNominalFirmaCT)
	if !ok || p == nil || p.fuente != f {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	seleccion := f.porCertificado[principal.CertificadoCanalSHA256]
	perfil := ""
	switch slot {
	case "consulta_ct":
		perfil = seleccion.PerfilConsultaCTRef
	case "circuito":
		perfil = seleccion.PerfilConsultaCircuitoRef
	case "documentos":
		perfil = seleccion.PerfilDocumentosRef
	default:
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	sesion := p.extras[perfil]
	if sesion == nil {
		snapshot, err := f.asignaciones.ObtenerInstantaneaAutorizacion(ctx, principal.Resultado.Contexto.PersonaRef, perfil)
		if err != nil || snapshot.Validar() != nil {
			return cero, ports.ErrFirmaDocumentoDenegada
		}
		q := p.solicitud
		q.PerfilEsperadoRef = perfil
		q.RolEsperadoID = snapshot.VersionRol.RolID
		// Sólo la selección privada abre el perfil auxiliar; AUT56 permanece ligado
		// al perfil de firma del contenedor original de esta petición.
		sinInicial := context.WithValue(ctx, clavePeticionNominalFirmaCT{}, (*peticionNominalFirmaCT)(nil))
		sesion, err = f.fuente.AbrirSesionFirmanteV2(sinInicial, q)
		if err != nil {
			return cero, err
		}
		p.extras[perfil] = sesion
	}
	e, err := sesion.RevalidarSesionFirmanteV2(ctx)
	if err != nil || e.Resultado.Contexto.PersonaRef != principal.Resultado.Contexto.PersonaRef ||
		e.Resultado.Contexto.Instantanea.CuentaRef != principal.Resultado.Contexto.Instantanea.CuentaRef ||
		e.Resultado.Contexto.PerfilActivoRef != perfil || e.CertificadoCanalSHA256 != principal.CertificadoCanalSHA256 {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	datos, err := e.Vinculo.Datos()
	if err != nil || datos.GarantiaObservada != core.AuthAssuranceSubstantial || datos.CuentaPrivilegiada ||
		!e.Vinculo.VigenteEn(f.reloj.Ahora(), e.Resultado) {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	return ports.ContextoAutorizacionAltaV3{Vinculo: e.Vinculo, Resultado: e.Resultado}, nil
}

func (f *fuenteNominalSesionFirmaCT) ResolverContextoCanalCircuitoRRHH(ctx context.Context) (httpinterno.ContextoCanalCircuitoRRHH, error) {
	e, err := f.contexto(ctx)
	if err != nil {
		return httpinterno.ContextoCanalCircuitoRRHH{}, err
	}
	d, err := e.Vinculo.Datos()
	if err != nil {
		return httpinterno.ContextoCanalCircuitoRRHH{}, err
	}
	org, err := f.ResolverOrganizacionFirmaVec(ctx)
	return httpinterno.ContextoCanalCircuitoRRHH{AutenticacionRef: d.AutenticacionRef, SesionRef: d.SesionRef, PerfilRef: d.PerfilActivoRef, OrganizacionRef: org}, err
}

func (f *fuenteNominalSesionFirmaCT) ResolverContextoAutorizacionAltaV3(ctx context.Context, q ports.SolicitudResolverContextoAutorizacionAltaV3) (ports.ContextoAutorizacionAltaV3, error) {
	e, err := f.contexto(ctx)
	if err != nil {
		return ports.ContextoAutorizacionAltaV3{}, err
	}
	v := ports.ContextoAutorizacionAltaV3{Vinculo: e.Vinculo, Resultado: e.Resultado}
	if err := v.ValidarPara(q, f.reloj.Ahora()); err != nil {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrFirmaDocumentoDenegada
	}
	return v, nil
}

func (f *fuenteNominalSesionFirmaCT) ResolverContextoConsultaRRHH(ctx context.Context) (ports.ContextoConsultaRRHH, error) {
	v, err := f.contextoPerfil(ctx, "consulta_ct")
	if err != nil {
		return ports.ContextoConsultaRRHH{}, err
	}
	org, err := f.ResolverOrganizacionFirmaVec(ctx)
	if err != nil {
		return ports.ContextoConsultaRRHH{}, err
	}
	snapshot, err := f.asignaciones.ObtenerInstantaneaAutorizacion(ctx, v.Resultado.Contexto.PersonaRef, v.Resultado.Contexto.PerfilActivoRef)
	if err != nil || snapshot.Validar() != nil || len(snapshot.AsignacionPerfil.Ambitos) != 3 {
		return ports.ContextoConsultaRRHH{}, ports.ErrFirmaDocumentoDenegada
	}
	ambitos := make(map[string]string, 3)
	for _, a := range snapshot.AsignacionPerfil.Ambitos {
		if len(a.Valores) != 1 {
			return ports.ContextoConsultaRRHH{}, ports.ErrFirmaDocumentoDenegada
		}
		ambitos[a.Clave] = a.Valores[0]
	}
	if len(ambitos) != 3 || ambitos["organizacion_ref"] != org ||
		ambitos["clase_ambito"] != string(ports.AmbitoOrganizacionRRHH) || ambitos["ambito_ref"] != org {
		return ports.ContextoConsultaRRHH{}, ports.ErrFirmaDocumentoDenegada
	}
	return ports.NuevoContextoConsultaRRHH(v, org, f.reloj.Ahora())
}
