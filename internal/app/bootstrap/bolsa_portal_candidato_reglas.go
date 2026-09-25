package bootstrap

import (
	"context"
	"errors"
	"strings"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

// reglasPortalCandidatoDesarrollo traduce el catálogo de reglas de Bolsa al
// puerto del portal: b29 (modo, contacto efectivo y situaciones), b05 (plazo
// de respuesta), b11 (causas de renuncia justificada) y b18 (pausa máxima).
// Sin b29 vigente el portal no admite acciones.
type reglasPortalCandidatoDesarrollo struct {
	resolutor *reglas.Resolutor
}

var _ puertosbolsa.ReglasPortalCandidato = reglasPortalCandidatoDesarrollo{}

func (r reglasPortalCandidatoDesarrollo) portal(ctx context.Context) (reglas.Regla, error) {
	regla, err := r.resolutor.Regla(ctx, reglas.BolsaPortalCandidato)
	if errors.Is(err, reglas.ErrReglaNoEncontrada) || errors.Is(err, reglas.ErrReglasNoConfiguradas) {
		return reglas.Regla{}, puertosbolsa.ErrReglasPortalCandidatoAusente
	}
	if err != nil {
		return reglas.Regla{}, errors.Join(puertosbolsa.ErrPortalCandidatoNoDisponible, err)
	}
	return regla, nil
}

func (r reglasPortalCandidatoDesarrollo) ModoRespuesta(ctx context.Context) (string, string, error) {
	regla, err := r.portal(ctx)
	if err != nil {
		return "", "", err
	}
	modo := regla.Atributos["modo_respuesta"]
	if modo != puertosbolsa.ModoRespuestaPortalFirme && modo != puertosbolsa.ModoRespuestaPortalPropuesta {
		return "", "", puertosbolsa.ErrPortalCandidatoNoDisponible
	}
	return modo, regla.Referencia, nil
}

func (r reglasPortalCandidatoDesarrollo) ResultadosContactoEfectivo(ctx context.Context) ([]string, error) {
	regla, err := r.portal(ctx)
	if err != nil {
		return nil, err
	}
	return listaAtributoRegla(regla, "contacto_efectivo")
}

func (r reglasPortalCandidatoDesarrollo) SituacionesAdmitidas(ctx context.Context, tipo string) ([]string, string, error) {
	regla, err := r.portal(ctx)
	if err != nil {
		return nil, "", err
	}
	atributo := map[string]string{puertosbolsa.SolicitudPortalPausa: "pausa_desde", puertosbolsa.SolicitudPortalReactivacion: "reactivacion_desde"}[tipo]
	if atributo == "" {
		return nil, "", puertosbolsa.ErrPortalCandidatoInvalido
	}
	lista, err := listaAtributoRegla(regla, atributo)
	return lista, regla.Referencia, err
}

func (r reglasPortalCandidatoDesarrollo) PausaMaxima(ctx context.Context, desde time.Time) (time.Time, string, error) {
	regla, vencimiento, err := r.resolutor.Vencimiento(ctx, reglas.BolsaPausaVoluntaria, desde, "")
	if err != nil {
		return time.Time{}, "", errors.Join(puertosbolsa.ErrPortalCandidatoNoDisponible, err)
	}
	return vencimiento.VenceAntesDe.UTC(), regla.Referencia, nil
}

func (r reglasPortalCandidatoDesarrollo) VencimientoRespuesta(ctx context.Context, contacto time.Time) (time.Time, string, error) {
	regla, vencimiento, err := r.resolutor.Vencimiento(ctx, reglas.BolsaPlazoRespuesta, contacto, "")
	if err != nil {
		return time.Time{}, "", errors.Join(puertosbolsa.ErrPortalCandidatoNoDisponible, err)
	}
	return vencimiento.VenceAntesDe.UTC(), regla.Referencia, nil
}

func (r reglasPortalCandidatoDesarrollo) CausasRenunciaJustificada(ctx context.Context) ([]string, error) {
	regla, err := r.resolutor.Regla(ctx, reglas.BolsaAcreditarRenunciaJustificada)
	if err != nil {
		return nil, errors.Join(puertosbolsa.ErrPortalCandidatoNoDisponible, err)
	}
	return listaAtributoRegla(regla, "causas")
}

// listaAtributoRegla separa por comas un atributo obligatorio de la regla.
func listaAtributoRegla(regla reglas.Regla, atributo string) ([]string, error) {
	valor := regla.Atributos[atributo]
	if valor == "" {
		return nil, puertosbolsa.ErrPortalCandidatoNoDisponible
	}
	lista := strings.Split(valor, ",")
	for _, elemento := range lista {
		if elemento == "" || elemento != strings.TrimSpace(elemento) {
			return nil, puertosbolsa.ErrPortalCandidatoNoDisponible
		}
	}
	return lista, nil
}
