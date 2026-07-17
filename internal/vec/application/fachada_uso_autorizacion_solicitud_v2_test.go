package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type autorizadorFachadaSolicitudV2Prueba struct {
	base  *autorizadorFachadaUsoPrueba
	mutar func(*domain.DecisionAutorizacion, domain.SolicitudAutorizacionLigadaV2)
}

func (a *autorizadorFachadaSolicitudV2Prueba) ExigirSolicitudLigadaV2(
	ctx context.Context,
	solicitud domain.SolicitudAutorizacionLigadaV2,
) (domain.DecisionAutorizacion, error) {
	proyeccion, err := proyectarSolicitudAutorizacionLigadaV2(solicitud)
	if err != nil {
		return domain.DecisionAutorizacion{}, err
	}
	decision, err := a.base.Exigir(ctx, proyeccion)
	if err != nil {
		return decision, err
	}
	huellaSolicitud, err := domain.HuellaSHA256SolicitudAutorizacionV2(solicitud)
	if err != nil {
		return domain.DecisionAutorizacion{}, err
	}
	datosSolicitud, err := solicitud.Datos()
	if err != nil {
		return domain.DecisionAutorizacion{}, err
	}
	huellaMotivo, err := domain.HuellaSHA256MotivoAutorizacionV2(datosSolicitud.ReferenciaMotivo)
	if err != nil {
		return domain.DecisionAutorizacion{}, err
	}
	decision.EsquemaHuellaSolicitud = domain.EsquemaHuellaSolicitudAutorizacionV2
	decision.SolicitudHuellaSHA256 = huellaSolicitud
	decision.EsquemaHuellaMotivo = domain.EsquemaHuellaMotivoAutorizacionV2
	decision.MotivoHuellaSHA256 = huellaMotivo
	if a.mutar != nil {
		a.mutar(&decision, solicitud)
	}
	return decision, nil
}

func TestFachadaSolicitudLigadaV2RechazaMutacionDeSolicitudOMotivo(t *testing.T) {
	ahora := instanteFachadaUsoAutorizacionPrueba()
	actor, vinculo := contextoYVinculoAutenticacionAplicacionPrueba(ahora)
	politica := politicaFachadaUsoPrueba(t, nil, PerfilProteccionUsoAutorizacionInternoAlto)
	motivo := referenciaMotivoAutorizacionV2Prueba(claveMotivoAutorizacionV2Prueba)
	casos := []struct {
		nombre string
		mutar  func(*domain.DecisionAutorizacion, domain.SolicitudAutorizacionLigadaV2)
	}{
		{"valida", nil},
		{"huella solicitud", func(d *domain.DecisionAutorizacion, _ domain.SolicitudAutorizacionLigadaV2) {
			d.SolicitudHuellaSHA256 = strings.Repeat("8", 64)
		}},
		{"huella motivo", func(d *domain.DecisionAutorizacion, _ domain.SolicitudAutorizacionLigadaV2) {
			d.MotivoHuellaSHA256 = strings.Repeat("9", 64)
		}},
		{"solo motivo solicitado", func(d *domain.DecisionAutorizacion, s domain.SolicitudAutorizacionLigadaV2) {
			datos, _ := s.Datos()
			datos.ReferenciaMotivo = referenciaMotivoAutorizacionV2Prueba(claveMotivoAutorizacionV2Alternativa)
			otra, _ := domain.NuevaSolicitudAutorizacionLigadaV2(datos)
			d.SolicitudHuellaSHA256, _ = domain.HuellaSHA256SolicitudAutorizacionV2(otra)
			d.MotivoHuellaSHA256, _ = domain.HuellaSHA256MotivoAutorizacionV2(datos.ReferenciaMotivo)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			base := &autorizadorFachadaUsoPrueba{
				ahora: ahora, garantiaMinima: domain.AuthAssuranceHigh,
			}
			autorizador := &autorizadorFachadaSolicitudV2Prueba{base: base, mutar: caso.mutar}
			fachada, err := NuevaFachadaUsoDecisionAutorizacionSolicitudLigadaV2(
				autorizador,
				relojUsoAutorizacionPrueba{ahora: ahora},
			)
			if err != nil {
				t.Fatal(err)
			}
			evidencia, err := fachada.ExigirEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(
				context.Background(), actor, vinculo, recursoFachadaUsoAutorizacionPrueba(),
				referenciaCorrelacionAutorizacionV2Prueba, motivo, politica,
			)
			if caso.mutar == nil {
				if err != nil || evidencia.ValidarEn(ahora) != nil ||
					evidencia.ValidarMotivo(motivo) != nil {
					t.Fatalf("V2 valida denegada: %v", err)
				}
				return
			}
			if !errors.Is(err, domain.ErrAutorizacionDenegada) || evidencia.ValidarEn(ahora) == nil {
				t.Fatalf("mutacion V2 aceptada: evidencia=%v err=%v", evidencia, err)
			}
		})
	}
}

var _ ports.AutorizadorSolicitudLigadaV2 = (*autorizadorFachadaSolicitudV2Prueba)(nil)
