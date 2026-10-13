package bootstrap

import (
	"context"
	"net/http"
	"time"

	appbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seg "vec-diputacion-granada/internal/vec/adapters/seguridad"
	appvec "vec-diputacion-granada/internal/vec/application"
)

const (
	fronteraAltaBolsaNativaCT      = "bolsa-nativa-ct-alta-interna"
	fronteraCeseBolsaNativaCT      = "bolsa-nativa-ct-cese-interna"
	fronteraSeleccionBolsaNativaCT = "bolsa-nativa-ct-seleccion-interna"
)

// operacionBolsaNativaCT conserva el método y la ruta CT reales. Su catálogo
// privado sólo existe en la composición y no se registra como ruta HTTP.
// El perfil Bolsa procede del manifiesto B-BACK publicado, nunca de HTTP.
type operacionBolsaNativaCT struct {
	ct       *soporteAltaContratacionTemporalDesarrollo
	bolsa    *soporteSesionBorradorBolsaDesarrollo
	publico  catalogoFronterasComunDesarrollo
	privado  catalogoFronterasComunDesarrollo
	sesion   *seguridadComunDesarrollo
	servicio *appbolsa.ServicioSituacionParticipacion
	emision  *appbolsa.ServicioEmisionLlamamiento
}

func nuevaOperacionBolsaNativaCT(
	m *manejadorParticipacionBolsaDesarrollo,
	soporteCT *soporteAltaContratacionTemporalDesarrollo,
	publico catalogoFronterasComunDesarrollo,
	reloj ct.Reloj,
) (*operacionBolsaNativaCT, error) {
	if m == nil || m.preparador == nil || m.preparador.sesion == nil ||
		m.preparador.soporte == nil || m.preparador.soporte.soporteCanal == nil ||
		!m.politicaSituacion.valida() || m.materialSituacion == nil || m.materialEmision == nil ||
		dependenciaBootstrapNula(m.repositorioSituacion) ||
		dependenciaBootstrapNula(m.repositorioEmision) ||
		dependenciaBootstrapNula(m.correoFuente) || dependenciaBootstrapNula(m.correoEmisor) ||
		soporteCT == nil ||
		publico.identidad == nil || dependenciaBootstrapNula(reloj) {
		return nil, ct.ErrComposicionIncorporacionAplicacion
	}
	base, ok := m.preparador.sesion.sesion.(*proveedorSesionConsultaRRHHDesarrollo)
	if !ok || base == nil || !base.fronteras.mismaInstancia(publico) ||
		base.soporte != m.preparador.soporte.soporteCanal ||
		base.registro == nil || base.revalidador == nil || base.resolutor == nil {
		return nil, ct.ErrComposicionIncorporacionAplicacion
	}
	perfil := m.preparador.soporte.soporteCanal.contexto.Resultado.Contexto.PerfilActivoRef
	claves := []struct{ clave, ruta, capacidad string }{
		{fronteraAltaBolsaNativaCT, httpct.RutaConfirmacionB2, claveCapacidadSituacionParticipacionBolsa},
		{fronteraCeseBolsaNativaCT, httpct.RutaCesesNombramiento, claveCapacidadSituacionParticipacionBolsa},
		{fronteraSeleccionBolsaNativaCT, httpct.RutaSeleccionLlamamiento, claveCapacidadEmisionLlamamientoBolsa},
	}
	descriptores := make([]descriptorFronteraComunDesarrollo, 0, len(claves))
	for _, d := range claves {
		descriptores = append(descriptores, descriptorFronteraComunDesarrollo{
			Clave: d.clave, Superficie: superficieInternaSeguridadComunDesarrollo,
			Metodo: http.MethodPost, Ruta: d.ruta, PerfilesActivosRef: []string{perfil},
			ClavePolitica:  clavePoliticaBorradorLlamamientoBolsaDesarrollo,
			ClaveCapacidad: d.capacidad,
		})
	}
	privado, err := nuevoCatalogoFronterasComunDesarrollo(descriptores)
	if err != nil {
		return nil, err
	}
	proveedor, err := nuevoProveedorSesionConsultaRRHHConCatalogoDesarrollo(
		base.soporte, base.registro, base.revalidador, base.reloj, base.resolutor, privado)
	if err != nil {
		return nil, err
	}
	sesion, err := nuevaSeguridadComunDesarrollo(proveedor, reloj)
	if err != nil {
		return nil, err
	}
	catalogoACT, err := nuevoCatalogoAutorizacionComunDesarrollo(privado,
		[]descriptorAutorizacionComunDesarrollo{{Accion: bp.AccionCambiarSituacionParticipacion,
			ClavePolitica:  clavePoliticaBorradorLlamamientoBolsaDesarrollo,
			ClaveCapacidad: claveCapacidadSituacionParticipacionBolsa,
			Fronteras:      []string{fronteraAltaBolsaNativaCT, fronteraCeseBolsaNativaCT},
			Politica:       m.politicaSituacion},
			{Accion: bp.AccionEmitirLlamamiento,
				ClavePolitica:  clavePoliticaBorradorLlamamientoBolsaDesarrollo,
				ClaveCapacidad: claveCapacidadEmisionLlamamientoBolsa,
				Fronteras:      []string{fronteraSeleccionBolsaNativaCT},
				Politica:       m.politicaSituacion}})
	if err != nil {
		return nil, err
	}
	pdp, err := nuevoAutorizadorComunDesarrollo(catalogoACT, reloj,
		seg.GeneradorReferenciasCriptograficas{}, appvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second})
	if err != nil {
		return nil, err
	}
	emisor, err := nuevoEmisorMaterialRenovableCTDesarrollo(pdp, m.materialSituacion)
	if err != nil {
		return nil, err
	}
	emisorB7, err := nuevoEmisorMaterialRenovableCTDesarrollo(pdp, m.materialEmision)
	if err != nil {
		return nil, err
	}
	preparador := *m.preparador
	preparador.sesion = sesion
	servicio, err := appbolsa.NuevoServicioSituacionParticipacion(&preparador,
		emisor, m.repositorioSituacion, reloj.Ahora)
	if err != nil {
		return nil, err
	}
	emision, err := appbolsa.NuevoServicioEmisionLlamamiento(&preparador,
		emisorB7, m.repositorioEmision, m.correoFuente, m.correoEmisor,
		reloj.Ahora, m.correoPersonalizado)
	if err != nil {
		return nil, err
	}
	return &operacionBolsaNativaCT{ct: soporteCT, bolsa: m.preparador.soporte,
		publico: publico, privado: privado, sesion: sesion, servicio: servicio,
		emision: emision}, nil
}

// Contexto devuelve el actor Bolsa sólo después de comprobar la capacidad
// CT y su frontera HTTP originales, el mismo certificado, principal y cuenta.
// La suboperación usa su propio cache nominal; no reaprovecha el del perfil CT.
func (o *operacionBolsaNativaCT) Contexto(ctx context.Context, actorCT *ct.ContextoAutorizacionAltaV3) (
	context.Context, ct.ContextoAutorizacionAltaV3, error,
) {
	var vacio ct.ContextoAutorizacionAltaV3
	if o == nil || o.ct == nil || dependenciaBootstrapNula(o.ct.reloj) || o.bolsa == nil || o.bolsa.soporteCanal == nil ||
		o.sesion == nil || ctx == nil || ctx.Err() != nil {
		return nil, vacio, ct.ErrAutorizacionDenegada
	}
	capacidad, valida := o.ct.capacidadValida(ctx)
	frontera, real := fronteraSeguridadComunDesdeContexto(ctx)
	if !valida || !real || !o.publico.mismaInstancia(frontera.catalogo) ||
		capacidad.metodo != http.MethodPost || frontera.metodo != capacidad.metodo ||
		frontera.ruta != capacidad.ruta || frontera.superficie != superficieInternaSeguridadComunDesarrollo ||
		(capacidad.ruta != httpct.RutaConfirmacionB2 && capacidad.ruta != httpct.RutaCesesNombramiento &&
			capacidad.ruta != httpct.RutaSeleccionLlamamiento) {
		return nil, vacio, ct.ErrAutorizacionDenegada
	}
	var actualCT ct.ContextoAutorizacionAltaV3
	if capacidad.ruta == httpct.RutaConfirmacionB2 {
		// Esta ruta dispone del contexto B2 nominal de ConfirmarOrigen, resuelto
		// desde la misma petición por su autoridad antes de llamar este puente.
		if actorCT == nil {
			return nil, vacio, ct.ErrAutorizacionDenegada
		}
		actualCT = *actorCT
	} else {
		// Selección y cese poseen el resolutor operativo CT de su ruta; no se
		// permite sustituirlo por un perfil recibido como argumento.
		if actorCT != nil {
			return nil, vacio, ct.ErrAutorizacionDenegada
		}
		var err error
		actualCT, err = o.ct.contextoOperativoDesarrollo(ctx)
		if err != nil {
			return nil, vacio, ct.ErrAutorizacionDenegada
		}
	}
	if actualCT.Resultado.Validar() != nil || actualCT.Vinculo.ValidarPara(actualCT.Resultado) != nil ||
		!actualCT.Vinculo.VigenteEn(o.ct.reloj.Ahora(), actualCT.Resultado) ||
		!frontera.descriptor.admitePerfil(actualCT.Resultado.Contexto.PerfilActivoRef) {
		return nil, vacio, ct.ErrAutorizacionDenegada
	}
	bolsa := o.bolsa.soporteCanal
	if bolsa.sello != o.ct.sello || bolsa.principalID != o.ct.principalID ||
		bolsa.certificadoSHA256 != o.ct.certificadoSHA256 ||
		bolsa.contexto.Resultado.Contexto.Instantanea.CuentaRef != actualCT.Resultado.Contexto.Instantanea.CuentaRef {
		return nil, vacio, ct.ErrAutorizacionDenegada
	}
	descriptor, existe := o.privado.resolver(capacidad.metodo, capacidad.ruta)
	if !existe || !descriptor.admitePerfil(bolsa.contexto.Resultado.Contexto.PerfilActivoRef) {
		return nil, vacio, ct.ErrAutorizacionDenegada
	}
	// La capacidad conserva su método, ruta y certificado. Sólo se separan
	// capturas nominales CT que no pertenecen a esta operación Bolsa.
	capacidad.consultaRRHH, capacidad.contextoOperacion, capacidad.original = nil, nil, nil
	hijo := context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	hijo = context.WithValue(hijo, claveContextosNominalesIncorporacion{}, (*capturaContextosNominalesIncorporacion)(nil))
	hijo = context.WithValue(hijo, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
		metodo: capacidad.metodo, ruta: capacidad.ruta,
		superficie: superficieInternaSeguridadComunDesarrollo, catalogo: o.privado, descriptor: descriptor})
	hijo = context.WithValue(hijo, claveCacheSeguridadComunDesarrollo{}, &cacheSeguridadComunDesarrollo{})
	actor, err := o.sesion.ResolverContexto(hijo)
	if err != nil || actor.Resultado.Validar() != nil || actor.Vinculo.ValidarPara(actor.Resultado) != nil {
		return nil, vacio, ct.ErrAutorizacionDenegada
	}
	datos, err := actor.Vinculo.Datos()
	ctDatos, errCT := actualCT.Vinculo.Datos()
	if err != nil || errCT != nil || datos.PrincipalID != ctDatos.PrincipalID ||
		datos.CuentaRef != ctDatos.CuentaRef ||
		actor.Resultado.Contexto.Instantanea.CuentaRef != actualCT.Resultado.Contexto.Instantanea.CuentaRef {
		return nil, vacio, ct.ErrAutorizacionDenegada
	}
	return hijo, ct.ContextoAutorizacionAltaV3{Vinculo: actor.Vinculo, Resultado: actor.Resultado}, nil
}
