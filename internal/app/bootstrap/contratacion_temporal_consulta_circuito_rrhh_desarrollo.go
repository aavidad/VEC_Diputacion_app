package bootstrap

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const clavePerfilFijoConsultaCircuitoRRHHDesarrollo = "consulta_circuito_rrhh"

// La denegación SQL revierte su propia transacción. La bitácora de frontera
// usa otra conexión y debe quedar confirmada antes de emitir el rechazo.
type auditorConsultaCircuitoRRHHDenegada struct {
	siguiente   http.Handler
	registrador puertosvec.RegistradorAuditoriaFronteraRutaExacta
	soporte     *soporteAltaContratacionTemporalDesarrollo
	reloj       relojContratacionTemporalDesarrollo
}

func (a auditorConsultaCircuitoRRHHDenegada) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if w == nil {
		return
	}
	if r == nil || r.URL == nil || a.siguiente == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(a.registrador) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(a.reloj) ||
		r.URL.Path != httpinterno.RutaConsultaCircuitoRRHH {
		responderConsultaCircuitoAuditoriaNoDisponible(w)
		return
	}
	respuesta := &respuestaConsultaReciboDiferida{cabeceras: make(http.Header)}
	a.siguiente.ServeHTTP(respuesta, r)
	estado := respuesta.estado
	if estado == 0 {
		estado = http.StatusOK
	}
	if estado == http.StatusNotFound || estado == http.StatusForbidden || estado == http.StatusUnauthorized {
		var cuerpo struct {
			Error struct {
				CorrelacionRef string `json:"correlacion_ref"`
			} `json:"error"`
		}
		if json.Unmarshal(respuesta.cuerpo.Bytes(), &cuerpo) != nil {
			responderConsultaCircuitoAuditoriaNoDisponible(w)
			return
		}
		motivo := puertosvec.MotivoAuditoriaFronteraRutaExactaAccesoDenegado
		if estado == http.StatusUnauthorized {
			motivo = puertosvec.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida
		}
		orden := puertosvec.OrdenAuditoriaFronteraRutaExacta{
			CorrelacionRef: cuerpo.Error.CorrelacionRef, Motivo: motivo,
			Superficie: puertosvec.SuperficieAuditoriaFronteraRutaExactaContratacionTemporal,
			Ruta:       httpinterno.RutaConsultaCircuitoRRHH,
		}
		if motivo == puertosvec.MotivoAuditoriaFronteraRutaExactaAccesoDenegado && a.soporte != nil {
			if capacidad, valida := a.soporte.capacidadValida(r.Context()); valida &&
				capacidad.ruta == httpinterno.RutaConsultaCircuitoRRHH &&
				certificadoConsultaReciboRespuestaVigente(capacidad, a.reloj.Ahora()) {
				orden.ActorRef = capacidad.principal.ID
			}
		}
		if orden.Validar() != nil {
			responderConsultaCircuitoAuditoriaNoDisponible(w)
			return
		}
		ctx, cancelar := context.WithTimeout(context.WithoutCancel(r.Context()), plazoarranque.Ampliar(250*time.Millisecond))
		err := a.registrador.RegistrarAuditoriaFronteraRutaExacta(ctx, orden)
		cancelar()
		if err != nil {
			responderConsultaCircuitoAuditoriaNoDisponible(w)
			return
		}
	}
	for clave, valores := range respuesta.cabeceras {
		for _, valor := range valores {
			w.Header().Add(clave, valor)
		}
	}
	w.WriteHeader(estado)
	_, _ = w.Write(respuesta.cuerpo.Bytes())
}

func responderConsultaCircuitoAuditoriaNoDisponible(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusServiceUnavailable)
}

type claveConsultaCircuitoRRHHDesarrollo struct{}

func motivoConsultaCircuitoRRHHDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_consulta_circuito_rrhh", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("consulta-circuito-rrhh-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "consulta-circuito-rrhh"),
	}
}

func descriptorMaterialConsultaCircuitoRRHHDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia:        postgresct.AudienciaConsultaCircuitoRRHH,
		Dominio:          "vec.ct.desarrollo.consulta-circuito-rrhh.capacidad-v3",
		Prefijo:          "clave:capacidad:ct:circuito-rrhh:",
		ProveedorNominal: proveedorMaterialContratacionTemporal,
	}
}

func nuevaInstantaneaConsultaCircuitoRRHHDesarrollo(principalID, perfilRef string, ahora time.Time) (dominiovec.InstantaneaAutorizacion, error) {
	return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(
		principalID, perfilRef, ahora,
		"consulta_circuito_rrhh_desarrollo", "consulta_circuito_rrhh_desarrollo",
		"perfil-consulta-circuito-rrhh-desarrollo",
		[]dominiovec.ConcesionRol{{
			Accion: postgresct.AccionConsultaCircuitoRRHH, ModuloID: ports.ModuloContratacion,
			TipoRecurso:      postgresct.TipoRecursoConsultaCircuitoRRHH,
			Finalidades:      []string{"gestionar_contratacion_temporal"},
			CamposPermitidos: []string{"circuito"}, Obligaciones: []string{"auditar"},
			GarantiaMinima: dominiovec.AuthAssuranceHigh,
		}},
		[]dominiovec.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}},
	)
}

func solicitudAutorizacionConsultaCircuitoRRHHValida(ctx context.Context, datos dominiovec.DatosSolicitudAutorizacionLigadaV3) bool {
	if ctx == nil {
		return false
	}
	s, existe := ctx.Value(claveConsultaCircuitoRRHHDesarrollo{}).(ports.SolicitudConsultaCircuitoRRHH)
	esperado, err := postgresct.RecursoConsultaCircuitoRRHH(s)
	r := datos.Recurso
	return existe && s.Validar() == nil && err == nil && r.Validar() == nil &&
		s.OrganizacionRef == organizacionAltaContratacionTemporalDesarrollo &&
		datos.Accion == postgresct.AccionConsultaCircuitoRRHH &&
		datos.Finalidad == "gestionar_contratacion_temporal" &&
		datos.ReferenciaMotivo == motivoConsultaCircuitoRRHHDesarrollo() &&
		r.Referencia == esperado.Referencia && r.ModuloID == esperado.ModuloID &&
		r.Tipo == esperado.Tipo && maps.Equal(r.Ambitos, esperado.Ambitos) &&
		maps.Equal(r.Atributos, esperado.Atributos)
}

type preparadorConsultaCircuitoRRHHDesarrollo struct {
	alta     *dependenciasAltaContratacionTemporalDesarrollo
	material *proveedorMaterialAltaContratacionTemporalDesarrollo
	reloj    relojContratacionTemporalDesarrollo
}

var _ ports.PreparadorConsultaCircuitoRRHH = (*preparadorConsultaCircuitoRRHHDesarrollo)(nil)
var _ httpinterno.AutoridadContextoCanalCircuitoRRHH = (*preparadorConsultaCircuitoRRHHDesarrollo)(nil)

func (p *preparadorConsultaCircuitoRRHHDesarrollo) ResolverContextoCanalCircuitoRRHH(ctx context.Context) (httpinterno.ContextoCanalCircuitoRRHH, error) {
	vacio := httpinterno.ContextoCanalCircuitoRRHH{}
	if p == nil || p.alta == nil || p.alta.soporte == nil || contextoInterfazNulo(ctx) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.reloj) {
		return vacio, ports.ErrConsultaCircuitoRRHHNoDisponible
	}
	capacidad, valida := p.alta.soporte.capacidadValida(ctx)
	if !valida || capacidad.ruta != httpinterno.RutaConsultaCircuitoRRHH ||
		!certificadoConsultaReciboRespuestaVigente(capacidad, p.reloj.Ahora()) {
		return vacio, ports.ErrConsultaCircuitoRRHHDenegada
	}
	operativo, err := p.alta.soporte.contextoOperativoDesarrollo(ctx)
	if err != nil {
		return vacio, ports.ErrConsultaCircuitoRRHHDenegada
	}
	vinculo, err := operativo.Vinculo.Datos()
	if err != nil {
		return vacio, ports.ErrConsultaCircuitoRRHHDenegada
	}
	return httpinterno.ContextoCanalCircuitoRRHH{
		AutenticacionRef: vinculo.AutenticacionRef, SesionRef: vinculo.SesionRef,
		PerfilRef: vinculo.PerfilActivoRef, OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
	}, nil
}

func (p *preparadorConsultaCircuitoRRHHDesarrollo) PrepararConsultaCircuitoRRHH(ctx context.Context, s ports.SolicitudConsultaCircuitoRRHH) (ports.MaterialConsultaCircuitoRRHH, error) {
	vacio := ports.MaterialConsultaCircuitoRRHH{}
	if p == nil || p.alta == nil || p.alta.soporte == nil || p.alta.autorizador == nil ||
		p.material == nil || s.Validar() != nil {
		return vacio, ports.ErrConsultaCircuitoRRHHNoDisponible
	}
	canal, err := p.ResolverContextoCanalCircuitoRRHH(ctx)
	if err != nil || canal.AutenticacionRef != s.AutenticacionRef || canal.SesionRef != s.SesionRef ||
		canal.PerfilRef != s.PerfilRef || canal.OrganizacionRef != s.OrganizacionRef {
		return vacio, ports.ErrConsultaCircuitoRRHHDenegada
	}
	operativo, err := p.alta.soporte.contextoOperativoDesarrollo(ctx)
	if err != nil {
		return vacio, ports.ErrConsultaCircuitoRRHHDenegada
	}
	vinculo, err := operativo.Vinculo.Datos()
	if err != nil {
		return vacio, ports.ErrConsultaCircuitoRRHHDenegada
	}
	recurso, err := postgresct.RecursoConsultaCircuitoRRHH(s)
	if err != nil {
		return vacio, ports.ErrConsultaCircuitoRRHHInvalida
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, ports.ErrConsultaCircuitoRRHHNoDisponible
	}
	datos := dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: operativo.Vinculo,
		ReferenciaMotivo:          motivoConsultaCircuitoRRHHDesarrollo(),
		Accion:                    postgresct.AccionConsultaCircuitoRRHH, Recurso: recurso,
		Finalidad: "gestionar_contratacion_temporal", Correlacion: correlacion,
	}
	ctx = context.WithValue(ctx, claveConsultaCircuitoRRHHDesarrollo{}, s)
	if !solicitudAutorizacionConsultaCircuitoRRHHValida(ctx, datos) {
		return vacio, ports.ErrConsultaCircuitoRRHHDenegada
	}
	solicitudV3, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		return vacio, ports.ErrConsultaCircuitoRRHHDenegada
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	decision, confirmacion, err := p.alta.autorizador.ExigirSolicitudLigadaV3(ctx, solicitudV3, operativo.Resultado)
	if err != nil {
		return vacio, ports.ErrConsultaCircuitoRRHHDenegada
	}
	autorizacion, err := p.material.proveerMaterialConfirmacion(ctx, solicitudV3, decision, confirmacion,
		motivoConsultaCircuitoRRHHDesarrollo(), operativo.Resultado)
	if err != nil {
		return vacio, ports.ErrConsultaCircuitoRRHHNoDisponible
	}
	material := ports.MaterialConsultaCircuitoRRHH{
		Solicitud: s, ActorRef: vinculo.PrincipalID, PerfilRef: vinculo.PerfilActivoRef,
		Autorizacion: autorizacion,
	}
	if material.ValidarPara(s) != nil {
		return vacio, ports.ErrConsultaCircuitoRRHHDenegada
	}
	return material, nil
}

func nuevoManejadorConsultaCircuitoRRHHDesarrollo(
	cfg config.Config, alta *dependenciasAltaContratacionTemporalDesarrollo,
	derivador *derivadorIdentidadOperacionDesarrollo, reloj relojContratacionTemporalDesarrollo,
) (http.Handler, error) {
	if alta == nil || alta.soporte == nil || alta.postgresql.ejecucion == nil ||
		alta.postgresql.gobierno == nil || alta.postgresql.proveedorMaterial == nil ||
		derivador == nil || !derivador.valido() {
		return nil, ports.ErrConsultaCircuitoRRHHNoDisponible
	}
	s := alta.soporte
	vinculo, err := s.contexto.Vinculo.Datos()
	if err != nil {
		return nil, ports.ErrConsultaCircuitoRRHHNoDisponible
	}
	ahora := reloj.Ahora()
	principal := dominiovec.Principal{
		ID: s.principalID, Roles: []string{rolTecnicoRRHHContratacionTemporalDesarrollo},
		AuthMethod: dominiovec.AuthMethodCertificate, AuthAssurance: dominiovec.AuthAssuranceHigh,
		Attributes: map[string]string{
			"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment,
			"certificate_sha256": s.certificadoSHA256,
		},
	}
	fijo, err := nuevoPerfilFijoCTDesarrollo(principal, s.contexto, ahora,
		clavePerfilFijoConsultaCircuitoRRHHDesarrollo,
		[]string{httpinterno.RutaConsultaCircuitoRRHH},
		func(principalID, perfilRef string) (dominiovec.InstantaneaAutorizacion, error) {
			return nuevaInstantaneaConsultaCircuitoRRHHDesarrollo(principalID, perfilRef, ahora)
		})
	if err != nil || vinculo.PrincipalID != s.principalID {
		return nil, ports.ErrConsultaCircuitoRRHHNoDisponible
	}
	ctx, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(15*time.Second))
	defer cancelar()
	desde, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(ahora)
	if !vigente || publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno,
		[]dominiovec.ReferenciaEntradaCatalogo{motivoConsultaCircuitoRRHHDesarrollo()}, desde) != nil {
		return nil, ports.ErrConsultaCircuitoRRHHNoDisponible
	}
	if err := s.registrarPerfilFijoCTDesarrollo(fijo); err != nil {
		return nil, err
	}
	if err := asegurarPerfilesFijosCTDesarrollo(ctx, alta.postgresql.gobierno, s,
		aprobacionProvisionPerfilesRRHHDesdeConfig(cfg), fijo); err != nil {
		return nil, err
	}
	material, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, ahora)
	if err != nil {
		return nil, ports.ErrConsultaCircuitoRRHHNoDisponible
	}
	defer material.borrarCopiasEfimeras()
	material.fuenteConfianza = alta.postgresql.proveedorMaterial.fuenteConfianza
	proveedor, err := nuevoProveedorMaterialConsumidorConDescriptorDesarrollo(ctx, alta.postgresql.gobierno,
		material, s, reloj, descriptorMaterialConsultaCircuitoRRHHDesarrollo())
	if err != nil {
		return nil, ports.ErrConsultaCircuitoRRHHNoDisponible
	}
	preparador := &preparadorConsultaCircuitoRRHHDesarrollo{alta: alta, material: proveedor, reloj: reloj}
	lector, err := postgresct.NuevoLectorCircuitoRRHHPostgreSQL(alta.postgresql.ejecucion)
	if err != nil {
		return nil, err
	}
	servicio, err := application.NuevoServicioConsultaCircuitoRRHH(preparador, lector)
	if err != nil {
		return nil, err
	}
	return httpinterno.NuevoManejadorConsultaCircuitoRRHH(preparador, servicio)
}
