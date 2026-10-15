package bootstrap

import (
	"context"
	"errors"
	"net/http"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

// El hook de la raíz invoca este montaje después de Documentos y antes de
// publicar la tabla de rutas. Un fallo impide publicar las cinco entradas.
func montarOriginalYFirmaVecR5(alta *dependenciasAltaContratacionTemporalDesarrollo,
	consultas dependenciasConsultasRRHHDesarrollo, render ports.RenderizadorBorradorRRHH,
	documentos *autoridadDocumentosDesarrollo, seudonimos *seudonimizadorAlmacenDesarrollo,
	reloj ports.Reloj, firma *firmaDocumentoCTDesarrollo, montaje *montajeFirmaVecR5CT) ([]vechttp.RutaExacta, error) {
	if alta == nil || firma == nil || montaje == nil || documentos == nil {
		return nil, errMontajeFirmaVecR5
	}
	var original *montajeOriginalFirmaR5Desarrollo
	rutasOriginal, err := nuevaRutaOriginalFirmableCTDesarrollo(alta, consultas, render, documentos, seudonimos, reloj, &original)
	if err != nil || len(rutasOriginal) != 1 || original == nil {
		return nil, errMontajeFirmaVecR5
	}
	// La misma autoridad Documentos acepta sesiones nominales únicamente por
	// contexto de esta petición; el original histórico conserva su perfil.
	original.original.pdp = pdpCTOriginalFirmableDesarrollo{alta: alta, nominal: montaje.nominal}
	firma.nominal = montaje.nominal
	servicioNominal, err := montaje.servicioOriginalNominal(render, original)
	if err != nil {
		return nil, err
	}
	original.servicioOriginal = servicioNominal
	hOriginalNominal, err := httpinterno.NuevoManejadorOriginalFirmableCT(montaje.nominal, servicioNominal)
	if err != nil {
		return nil, err
	}
	hOriginalNominal = montaje.nominal.envolver(hOriginalNominal)
	hOriginalLegacy := rutasOriginal[0].Manejador
	rutasOriginal[0].Manejador = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, huella, ok := certificadoSesionFirmanteV2(r, montaje.reloj.Ahora())
		if ok {
			if _, nominal := montaje.nominal.porCertificado[huella]; nominal {
				hOriginalNominal.ServeHTTP(w, r)
				return
			}
		}
		hOriginalLegacy.ServeHTTP(w, r)
	})
	rutasFirma, err := montaje.rutas(firma, original)
	if err != nil || len(rutasFirma) != 4 {
		return nil, errMontajeFirmaVecR5
	}
	return append(rutasOriginal, rutasFirma...), nil
}

func (p pdpCTOriginalFirmableDesarrollo) solicitarOriginalNominalV3(ctx context.Context, accion, finalidad string,
	recurso core.RecursoAutorizable) (solicitudCustodiaCTDesarrollo, error) {
	var vacia solicitudCustodiaCTDesarrollo
	if p.alta == nil || p.alta.autorizador == nil || p.nominal == nil ||
		!(rutaFirmaVecR5CT(rutaNominalFirmaCT(ctx)) || rutaNominalFirmaCT(ctx) == httpinterno.RutaOriginalFirmableCT ||
			rutaNominalFirmaCT(ctx) == httpinterno.RutaDescargaFirmaR5V2) {
		return vacia, errOriginalFirmableCTDenegado
	}
	operativo, err := p.nominal.contextoPerfil(ctx, "documentos")
	if err != nil {
		return vacia, errOriginalFirmableCTDenegado
	}
	actor, err := operativo.Vinculo.Datos()
	if err != nil {
		return vacia, errOriginalFirmableCTDenegado
	}
	correlacion, err := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacia, errors.Join(docports.ErrCapacidadNoDisponible, err)
	}
	ref, err := correlacion.ValorCanonico()
	if err != nil {
		return vacia, errors.Join(docports.ErrCapacidadNoDisponible, err)
	}
	datos := core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: operativo.Vinculo,
		ReferenciaMotivo: motivoOriginalFirmableCTDesarrollo(), Accion: accion, Recurso: recurso,
		Finalidad: finalidad, Correlacion: correlacion}
	if !solicitudAutorizacionOriginalFirmableCTDesarrolloValida(ctx, datos) {
		return vacia, errOriginalFirmableCTDenegado
	}
	s, err := core.NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		return vacia, errOriginalFirmableCTDenegado
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	d, c, err := p.alta.autorizador.ExigirSolicitudLigadaV3(ctx, s, operativo.Resultado)
	if err != nil {
		if ctx.Err() != nil {
			return vacia, ctx.Err()
		}
		if errors.Is(err, core.ErrAutorizacionDenegada) {
			return vacia, errOriginalFirmableCTDenegado
		}
		return vacia, errors.Join(docports.ErrCapacidadNoDisponible, err)
	}
	return solicitudCustodiaCTDesarrollo{solicitud: s, decision: d, confirmacion: c,
		operativo: operativo, actor: actor, correlacion: ref}, nil
}

func rutaNominalFirmaCT(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	p, ok := ctx.Value(clavePeticionNominalFirmaCT{}).(*peticionNominalFirmaCT)
	if !ok || p == nil || p.peticion == nil || p.peticion.URL == nil {
		return ""
	}
	return p.peticion.URL.Path
}

func (f *firmaDocumentoCTDesarrollo) solicitarCustodiaNominalV3(ctx context.Context,
	recurso core.RecursoAutorizable) (solicitudCustodiaCTDesarrollo, error) {
	var vacia solicitudCustodiaCTDesarrollo
	if f == nil || f.alta == nil || f.alta.autorizador == nil || f.nominal == nil ||
		rutaNominalFirmaCT(ctx) != httpinterno.RutaRegistroFirmaVec {
		return vacia, errCustodiaFirmadoCTDenegada
	}
	operativo, err := f.nominal.contextoPerfil(ctx, "documentos")
	if err != nil {
		return vacia, errCustodiaFirmadoCTDenegada
	}
	actor, err := operativo.Vinculo.Datos()
	if err != nil {
		return vacia, errCustodiaFirmadoCTDenegada
	}
	correlacion, err := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacia, errors.Join(docports.ErrCapacidadNoDisponible, err)
	}
	ref, err := correlacion.ValorCanonico()
	if err != nil {
		return vacia, errors.Join(docports.ErrCapacidadNoDisponible, err)
	}
	datos := core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: operativo.Vinculo,
		ReferenciaMotivo: motivoFirmaDocumentoCTDesarrollo(), Accion: docports.AccionCustodiarFirmado,
		Recurso: recurso, Finalidad: docports.FinalidadCustodiarFirmado, Correlacion: correlacion}
	if !solicitudAutorizacionCustodiaFirmadoCTDesarrolloValida(ctx, datos) {
		return vacia, errCustodiaFirmadoCTDenegada
	}
	s, err := core.NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		return vacia, errCustodiaFirmadoCTDenegada
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	d, c, err := f.alta.autorizador.ExigirSolicitudLigadaV3(ctx, s, operativo.Resultado)
	if err != nil {
		if ctx.Err() != nil {
			return vacia, ctx.Err()
		}
		if errors.Is(err, core.ErrAutorizacionDenegada) {
			return vacia, errCustodiaFirmadoCTDenegada
		}
		return vacia, errors.Join(docports.ErrCapacidadNoDisponible, err)
	}
	return solicitudCustodiaCTDesarrollo{solicitud: s, decision: d, confirmacion: c,
		operativo: operativo, actor: actor, correlacion: ref}, nil
}
