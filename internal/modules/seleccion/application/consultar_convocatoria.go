package application

import (
	"context"
	"encoding/hex"
	"errors"
	"reflect"
	"strings"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

// ConsultarConvocatoria comprueba correspondencia y copia la proyección del
// lector. No concede permisos: Bolsa debe autorizar cada lectura por V3.
func ConsultarConvocatoria(ctx context.Context, solicitud ports.SolicitudConsultaConvocatoria, lector ports.LectorConvocatoriaExacta) (ports.LecturaConvocatoria, error) {
	vacio := ports.LecturaConvocatoria{}
	if ctx == nil || lectorConvocatoriaNulo(lector) {
		return vacio, ports.ErrConvocatoriaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	actor, err := solicitud.Actor.Clonar()
	if err != nil || solicitud.Selector.Validar() != nil || solicitud.Correlacion.Validar() != nil {
		return vacio, ports.ErrConsultaConvocatoriaInvalida
	}
	solicitud.Actor = actor
	lectura, err := lector.ConsultarExacta(ctx, solicitud)
	if cancelacion := ctx.Err(); cancelacion != nil {
		return vacio, cancelacion
	}
	if err != nil {
		for _, nominal := range []error{ports.ErrConsultaConvocatoriaDenegada, ports.ErrConvocatoriaNoEncontrada, ports.ErrConvocatoriaNoDisponible, context.Canceled, context.DeadlineExceeded} {
			if errors.Is(err, nominal) {
				return vacio, nominal
			}
		}
		return vacio, ports.ErrConvocatoriaNoDisponible
	}
	if !lecturaConvocatoriaValida(solicitud, lectura) {
		return vacio, ports.ErrRespuestaConvocatoriaInvalida
	}
	lectura.Ficha.Bases = append(lectura.Ficha.Bases[:0:0], lectura.Ficha.Bases...)
	lectura.Ficha.Requisitos = append(lectura.Ficha.Requisitos[:0:0], lectura.Ficha.Requisitos...)
	return lectura, nil
}

func lecturaConvocatoriaValida(s ports.SolicitudConsultaConvocatoria, l ports.LecturaConvocatoria) bool {
	f, e := l.Ficha, l.Evidencia
	if f.ConvocatoriaID != s.Selector.ID || f.Secuencia != s.Selector.Secuencia || f.Revision < 1 ||
		f.FuenteRef != s.Selector.Referencia() || !huellaConsultaValida(f.HuellaVersionSHA256) ||
		f.FlujoProceso.Validar() != nil || f.ReglasBaremacion.Validar() != nil ||
		f.FasesEstado != "pendiente_fuente" || f.ReferenciasCoberturaEstado != "pendiente_fuente" ||
		len(f.Bases) == 0 || len(f.Bases) > 256 || len(f.Requisitos) > 256 ||
		!referenciaConsultaValida(e.ReciboRef) || !referenciaConsultaValida(e.DecisionRef) ||
		!huellaConsultaValida(e.ConsumoHuellaSHA256) || !referenciaConsultaValida(e.AuditoriaRef) ||
		e.CorrelacionRef != correlacionConvocatoria(s) || e.ConsultadaEn.IsZero() || e.ConsultadaEn.Nanosecond()%1000 != 0 {
		return false
	}
	_, zona := e.ConsultadaEn.Zone()
	if zona != 0 {
		return false
	}
	bases, requisitos := map[string]bool{}, map[string]bool{}
	for _, b := range f.Bases {
		if b.Validar() != nil || bases[b.PublicacionRef] {
			return false
		}
		bases[b.PublicacionRef] = true
	}
	for _, r := range f.Requisitos {
		if !referenciaConsultaValida(r.Referencia) || requisitos[r.Referencia] || r.Orden < 1 ||
			!textoConsultaValido(r.Titulo, 180, false) || !textoConsultaValido(r.Descripcion, 3000, true) {
			return false
		}
		requisitos[r.Referencia] = true
	}
	return true
}

func referenciaConsultaValida(s string) bool {
	if len(s) == 0 || len(s) > 512 {
		return false
	}
	for _, r := range s {
		if r < 0x21 || r > 0x7e || r == '*' {
			return false
		}
	}
	return true
}

func huellaConsultaValida(s string) bool {
	if len(s) != 64 || s != strings.ToLower(s) {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

func textoConsultaValido(s string, maximo int, opcional bool) bool {
	return utf8.ValidString(s) && len(s) <= 4*maximo && utf8.RuneCountInString(s) <= maximo && (opcional || strings.TrimSpace(s) != "")
}

func lectorConvocatoriaNulo(lector ports.LectorConvocatoriaExacta) bool {
	if lector == nil {
		return true
	}
	v := reflect.ValueOf(lector)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func correlacionConvocatoria(s ports.SolicitudConsultaConvocatoria) string {
	ref, _ := s.Correlacion.ValorCanonico()
	return ref
}
