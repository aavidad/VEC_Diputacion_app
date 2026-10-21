package application

import (
	"context"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type sesionSelectorDocumentalPrueba struct {
	t        *testing.T
	entrada  ports.EntradaDetalleExpedienteRRHHMinimizada
	version  uint64
	selector domain.SelectorBolsaPlanB2Nativo
	llamadas int
}

func (s *sesionSelectorDocumentalPrueba) ConsultarCuadroYRegistrar(context.Context, ports.OrdenConsultaCuadroRRHH) (ports.PaginaCuadroRRHH, error) {
	return ports.PaginaCuadroRRHH{}, ports.ErrConsultaRRHHNoDisponible
}
func (s *sesionSelectorDocumentalPrueba) ConsultarDetalleYRegistrar(_ context.Context, o ports.OrdenConsultaDetalleRRHH) (ports.DetalleExpedienteRRHH, error) {
	recibo := reciboConsultaRRHHPrueba(s.t, o.Contexto(), o.Capacidad(), o.Instante(), o.Solicitud().ExpedienteRef(), s.version, 1)
	return ports.NuevoDetalleExpedienteRRHHMinimizado(s.entrada, recibo)
}
func (s *sesionSelectorDocumentalPrueba) ConsultarSelectorYRegistrarAcceso(ctx context.Context, o ports.OrdenConsultaDetalleRRHH) (ports.DetalleExpedienteRRHH, domain.SelectorBolsaPlanB2Nativo, error) {
	s.llamadas++
	d, err := s.ConsultarDetalleYRegistrar(ctx, o)
	return d, s.selector, err
}

func selectorDocumentalPrueba(expediente, organizacion string) domain.SelectorBolsaPlanB2Nativo {
	return domain.SelectorBolsaPlanB2Nativo{
		OrganizacionRef: organizacion, ExpedienteRef: expediente,
		ClaveCT:   "41111111-1111-4111-8111-111111111111",
		UnidadRef: "unidad:prueba", CategoriaRef: "categoria:prueba",
		NecesidadRef: "necesidad:prueba", BolsaRef: "bolsa:prueba",
		LlamamientoRef: "llamamiento:prueba", ReciboEmisionRef: "recibo:emision:prueba",
		ReciboRespuestaRef: "recibo:respuesta:prueba",
	}
}

func TestSelectorDocumentalRRHHReutilizaDetalleNominal(t *testing.T) {
	for _, caso := range []string{"v7", "v8", "v9", "posterior", "sin_propuesta", "expediente_ajeno", "organizacion_ajena", "unidad_ajena", "categoria_ajena", "selector_ausente", "permiso_denegado"} {
		t.Run(caso, func(t *testing.T) {
			e := nuevoEntornoConsultaRRHH(t)
			configurarDetalleCompletoRRHHPrueba(t, e)
			d := e.sesion.detalle.Clonar()
			version := uint64(7)
			switch caso {
			case "v8":
				version = 8
			case "v9":
				version = 9
			case "posterior":
				version = 12
			}
			for n := uint64(5); n <= version; n++ {
				h := d.Hitos[len(d.Hitos)-1]
				h.Secuencia, h.VersionExpediente = n, n
				h.AccionClave = "actuacion_ejercicio"
				h.FaseOrigen = h.FaseDestino
				if n == 7 {
					h.AccionClave, h.FaseDestino = "registrar_propuesta_formalizacion", "nombramiento"
				}
				d.Hitos = append(d.Hitos, h)
			}
			d.Resumen.Version, d.Resumen.FaseClave = version, "nombramiento"
			if caso == "sin_propuesta" {
				d.Hitos[6].AccionClave = "actuacion_ajena"
			}
			ra, _ := ports.NuevaReferenciaHitoAnalisisRRHH(2)
			rc, _ := ports.NuevaReferenciaHitoCoberturaRRHH(3)
			ru, _ := ports.NuevaReferenciaHitoAsignacionRRHH(4)
			entrada, err := ports.NuevaEntradaDetalleExpedienteRRHHMinimizada(d.Resumen, d.Solicitud, d.Analisis, ra, d.Cobertura, rc, d.Asignacion, ru, d.Hitos)
			if err != nil {
				t.Fatal(err)
			}
			s := &sesionSelectorDocumentalPrueba{t: t, entrada: entrada, version: version,
				selector: selectorDocumentalPrueba(d.Resumen.ExpedienteRef, d.Resumen.OrganizacionRef)}
			s.selector.UnidadRef = d.Asignacion.UnidadRef
			s.selector.CategoriaRef = d.Analisis.CategoriaRef
			switch caso {
			case "expediente_ajeno":
				s.selector.ExpedienteRef = "expediente:ajeno"
			case "organizacion_ajena":
				s.selector.OrganizacionRef = "organizacion:ajena"
			case "unidad_ajena":
				s.selector.UnidadRef = "unidad:ajena"
			case "categoria_ajena":
				s.selector.CategoriaRef = "categoria:ajena"
			case "selector_ausente":
				s.selector = domain.SelectorBolsaPlanB2Nativo{}
			case "permiso_denegado":
				e.autoridad.err = ports.ErrConsultaRRHHNoObservable
			}
			servicio, err := NuevoServicioSelectorIdentidadDocumentalRRHH(e.autoridad, e.emisor, s, e.reloj)
			if err != nil {
				t.Fatal(err)
			}
			selector, err := servicio.Consultar(context.Background(), d.Resumen.ExpedienteRef)
			if caso == "v7" || caso == "v8" || caso == "v9" || caso == "posterior" {
				if err != nil || selector != s.selector || s.llamadas != 1 || e.emision.detalle.llamadas != 1 {
					t.Fatal("selector nominal no recuperado", err)
				}
			} else if err == nil || !selector.IsZero() ||
				(caso == "permiso_denegado" && s.llamadas != 0) {
				t.Fatal("selector ajeno o ausente aceptado", err)
			}
		})
	}
}
