package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

func TestFirmaMultipleEstadoGlobalRechazaAntesDeCompetencia(t *testing.T) {
	casos := []struct {
		nombre   string
		estado   docports.EstadoVerificacionFirma
		motivo   docports.MotivoVerificacionFirma
		esperado docports.MotivoVerificacionFirma
	}{
		{"invalida", docports.EstadoVerificacionNoValida, docports.MotivoIntegridadNoValida, docports.MotivoIntegridadNoValida},
		{"indeterminada", docports.EstadoVerificacionIndeterminada, docports.MotivoRevocacionNoAcreditada, docports.MotivoRevocacionNoAcreditada},
		{"detalle favorable", docports.EstadoVerificacionIndeterminada, docports.MotivoFirmaVerificada, docports.MotivoRespuestaNoInterpretable},
		{"estado desconocido", "otro", docports.MotivoIntegridadNoValida, docports.MotivoRespuestaNoInterpretable},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			s, d, comp, original := prepararServicioMultipleConsumidor(t)
			d.alterarDictamen = func(v *docports.VerificacionFirmasDocumento) { v.Estado, v.Motivo = c.estado, c.motivo }
			_, err := registrarPasoMultipleConsumidor(t, s, d, original, 1)
			var rechazo DictamenRechazado
			if !errors.As(err, &rechazo) || rechazo.Motivo != c.esperado || len(comp.vistas) != 0 || len(d.materiales) != 0 {
				t.Fatalf("dictamen no gobierna decisión: %v", err)
			}
		})
	}
}

func TestServiciosFirmaV2NoNecesitanAutoridadesV1(t *testing.T) {
	s, d, comp, original := prepararServicioMultipleConsumidor(t)
	s.base.verificador = nil
	vec, err := NuevoServicioFirmaVecV2(s.base, d, d, d, d, d, comp, nil)
	if err != nil || vec.registro != nil || vec.consulta != nil || vec.autorizador != nil {
		t.Fatalf("constructor VEC acoplado a V1: %v", err)
	}
	if _, err := registrarPasoMultipleConsumidor(t, vec, d, original, 1); err != nil {
		t.Fatal(err)
	}
	externa, err := NuevoServicioFirmaExternaV2(s.base, d, d, d, d, d, comp, nil)
	if err != nil || externa.registro != nil || externa.consulta != nil || externa.autorizador != nil {
		t.Fatalf("constructor externo acoplado a V1: %v", err)
	}
	if _, err := NuevoServicioFirmaVecV2(s.base, nil, d, d, d, d, comp, nil); !errors.Is(err, ErrCircuitoFirmaNoDisponible) {
		t.Fatal("verificador V2 ausente admitido")
	}
	copia := *s.base
	copia.circuito = nil
	if _, err := NuevoServicioFirmaVecV2(&copia, d, d, d, d, d, comp, nil); !errors.Is(err, ErrCircuitoFirmaNoDisponible) {
		t.Fatal("circuito ausente admitido")
	}
}

func TestPreflightFirmaMultipleLeeActoV2YRecuperaCustodiaAnterior(t *testing.T) {
	for _, alterar := range []bool{false, true} {
		t.Run(fmt.Sprint(alterar), func(t *testing.T) {
			s, d, comp, o := prepararServicioMultipleConsumidor(t)
			if _, err := registrarPasoMultipleConsumidor(t, s, d, o, 1); err != nil {
				t.Fatal(err)
			}
			externa, err := NuevoServicioFirmaExternaV2(s.base, d, d, d, d, d, comp, nil)
			if err != nil {
				t.Fatal(err)
			}
			d.alterarCustodia = alterar
			disponibilidad := &disponibilidadPreflightPrueba{}
			preflight := &ServicioPreflightFirmaR5{firmas: externa, disponibilidad: disponibilidad}
			q := solicitudPreflightPrueba()
			sol := solicitudFirmaVecPrueba(o.contenido, "no-es-operacion")
			q.Canal.OrganizacionRef, q.Canal.ExpedienteRef, q.Canal.VersionObservada = sol.OrganizacionRef, sol.ExpedienteRef, sol.VersionExpediente
			q.OriginalRef, q.OriginalVersion = sol.OriginalRef, sol.OriginalVersion
			r, err := preflight.consultarNominal(context.Background(), q, "per_actual_sintetico_001", strings.Repeat("a", 64))
			if alterar {
				if !errors.Is(err, ports.ErrCadenaFirmaDocumentoRota) || len(disponibilidad.vistas) != 0 {
					t.Fatalf("custodia alterada ofreció vía: %v", err)
				}
			} else if err != nil || r.PasoPendiente != 2 || len(r.ViasDisponibles) != 1 || len(disponibilidad.vistas) != 1 {
				t.Fatalf("preflight V2: %+v %v", r, err)
			}
			if len(d.materiales) != 1 {
				t.Fatal("preflight registró otro efecto")
			}
		})
	}
}

func TestPreflightFirmaMultipleLigaConsultaAlContextoActual(t *testing.T) {
	s, d, comp, o := prepararServicioMultipleConsumidor(t)
	externa, err := NuevoServicioFirmaExternaV2(s.base, d, d, d, d, d, comp, nil)
	if err != nil {
		t.Fatal(err)
	}
	disponibilidad := &disponibilidadPreflightPrueba{}
	externa.multiple.consulta = consultaPreflightR5V2ContextoActual{origen: d, contextoRef: "contexto:ajeno", contextoHuella: strings.Repeat("a", 64)}
	preflight := &ServicioPreflightFirmaR5{firmas: externa, disponibilidad: disponibilidad}
	q := solicitudPreflightPrueba()
	sol := solicitudFirmaVecPrueba(o.contenido, "no-es-operacion")
	q.Canal.OrganizacionRef = sol.OrganizacionRef
	q.OriginalRef, q.OriginalVersion = sol.OriginalRef, sol.OriginalVersion
	if _, err := preflight.consultarNominal(context.Background(), q, "per_actual_sintetico_001", strings.Repeat("a", 64)); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || len(disponibilidad.vistas) != 0 {
		t.Fatalf("capacidad ajena consumida: %v", err)
	}
}

type registroMultipleReciboAlterado struct {
	*dependenciasMultiplePrueba
	alterar func(*ports.ReciboFirmaDocumento)
}

func (r registroMultipleReciboAlterado) RegistrarFirmaVerificadaV2(ctx context.Context, m ports.MaterialFirmaVerificadaV2, c ports.CapacidadFirmaVerificadaV2) (ports.ReciboFirmaDocumento, error) {
	recibo, err := r.dependenciasMultiplePrueba.RegistrarFirmaVerificadaV2(ctx, m, c)
	if err == nil {
		r.alterar(&recibo)
	}
	return recibo, err
}
func TestFirmaMultipleReplayExigeReciboHistoricoExacto(t *testing.T) {
	casos := []struct {
		nombre  string
		alterar func(*ports.ReciboFirmaDocumento)
	}{
		{"firma", func(r *ports.ReciboFirmaDocumento) { r.FirmaRef = "firma:distinta" }},
		{"recibo", func(r *ports.ReciboFirmaDocumento) { r.ReciboRef = "recibo:distinto" }},
		{"fecha", func(r *ports.ReciboFirmaDocumento) { r.RegistradaEn = r.RegistradaEn.Add(time.Second) }},
		{"replay", func(r *ports.ReciboFirmaDocumento) { r.YaRegistrada = false }},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			s, d, _, o := prepararServicioMultipleConsumidor(t)
			if _, err := registrarPasoMultipleConsumidor(t, s, d, o, 1); err != nil {
				t.Fatal(err)
			}
			s.multiple.registro = registroMultipleReciboAlterado{d, c.alterar}
			if _, err := registrarPasoMultipleConsumidor(t, s, d, o, 1); !errors.Is(err, ports.ErrResultadoFirmaDocumentoInvalido) || len(d.materiales) != 1 {
				t.Fatalf("recibo modificado admitido: %v", err)
			}
		})
	}
}

func prepararPreflightEntradaV2(t *testing.T, actos int) (*ServicioPreflightFirmaR5, *dependenciasMultiplePrueba, *originalExternoPrueba, ports.SolicitudPreflightFirmaR5) {
	t.Helper()
	s, d, comp, o := prepararServicioMultipleConsumidor(t)
	for i := 1; i <= actos; i++ {
		if _, err := registrarPasoMultipleConsumidor(t, s, d, o, i); err != nil {
			t.Fatal(err)
		}
	}
	externa, err := NuevoServicioFirmaExternaV2(s.base, d, d, d, d, d, comp, nil)
	if err != nil {
		t.Fatal(err)
	}
	q := solicitudPreflightPrueba()
	sol := solicitudFirmaVecPrueba(o.contenido, "consulta-no-registro")
	q.Canal.OrganizacionRef, q.Canal.ExpedienteRef, q.Canal.VersionObservada = sol.OrganizacionRef, sol.ExpedienteRef, sol.VersionExpediente
	q.OriginalRef, q.OriginalVersion = sol.OriginalRef, sol.OriginalVersion
	return &ServicioPreflightFirmaR5{firmas: externa}, d, o, q
}

func TestPreflightFirmaV2DevuelvePDFEntradaActualYConservaRaiz(t *testing.T) {
	for actos := 0; actos <= 2; actos++ {
		t.Run(fmt.Sprint(actos), func(t *testing.T) {
			s, d, o, q := prepararPreflightEntradaV2(t, actos)
			r, err := s.consultarNominalConEntrada(context.Background(), q, "per_actual_sintetico_001", strings.Repeat("a", 64), true)
			if err != nil || ValidarResultadoPreflightFirmaR5V2(r, q) != nil {
				t.Fatalf("descriptor V2 no válido: %+v %v", r, err)
			}
			if r.OriginalRef != q.OriginalRef || r.OriginalVersion != q.OriginalVersion || len(d.materiales) != actos || len(r.ViasDisponibles) != 0 {
				t.Fatal("raíz, cierre o historia alterados")
			}
			switch actos {
			case 0:
				if r.PasoPendiente != 1 || r.EntradaDocumentoRef != q.OriginalRef || r.EntradaDocumentoVersion != q.OriginalVersion || r.EntradaDocumentoHuella != huella(o.contenido) {
					t.Fatal("entrada del primer paso no procede del original")
				}
			case 1:
				previa := d.lecturas.Firmas[0]
				if r.PasoPendiente != 2 || r.EntradaDocumentoRef != previa.DocumentoCustodiaRef || r.EntradaDocumentoVersion != previa.DocumentoCustodiaVersion || r.EntradaDocumentoHuella != previa.FirmadoHuella || r.EntradaDocumentoRef == q.OriginalRef {
					t.Fatal("el segundo paso no prepara el PDF previo exacto")
				}
			case 2:
				if r.PasoPendiente != 0 || r.EntradaDocumentoRef != "" || r.EntradaDocumentoVersion != 0 || r.EntradaDocumentoHuella != "" {
					t.Fatal("el circuito terminado ofrece un PDF para volver a firmar")
				}
			}
		})
	}
}

func TestPreflightFirmaV2RechazaEntradaAlteradaYCabezaCambiada(t *testing.T) {
	for _, custodia := range []bool{false, true} {
		t.Run(fmt.Sprint(custodia), func(t *testing.T) {
			s, d, o, q := prepararPreflightEntradaV2(t, 1)
			if custodia {
				d.alterarCustodia = true
			} else {
				previo := o.alterar
				o.alterar = func(r *ports.OriginalFirmaAutorizado) { previo(r); d.lecturas.HistoriaRevision++ }
			}
			r, err := s.consultarNominalConEntrada(context.Background(), q, "per_actual_sintetico_001", strings.Repeat("a", 64), true)
			if err == nil || r.EntradaDocumentoRef != "" || r.EntradaDocumentoVersion != 0 || r.EntradaDocumentoHuella != "" || len(d.materiales) != 1 {
				t.Fatalf("se expuso descriptor no acreditado: %+v %v", r, err)
			}
		})
	}
}

func TestPreflightFirmaV2ValidaDescriptorYNoAdmiteServicioV1(t *testing.T) {
	s, _, _, q := prepararPreflightEntradaV2(t, 1)
	r, err := s.consultarNominalConEntrada(context.Background(), q, "per_actual_sintetico_001", strings.Repeat("a", 64), true)
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre  string
		alterar func(*ports.ResultadoPreflightFirmaR5V2)
	}{
		{"ref ausente", func(r *ports.ResultadoPreflightFirmaR5V2) { r.EntradaDocumentoRef = "" }},
		{"version ausente", func(r *ports.ResultadoPreflightFirmaR5V2) { r.EntradaDocumentoVersion = 0 }},
		{"huella ausente", func(r *ports.ResultadoPreflightFirmaR5V2) { r.EntradaDocumentoHuella = "" }},
		{"original en segundo paso", func(r *ports.ResultadoPreflightFirmaR5V2) { r.EntradaDocumentoRef = r.OriginalRef }},
		{"entrada tras terminar", func(r *ports.ResultadoPreflightFirmaR5V2) { r.PasoPendiente = 0 }},
		{"paso no soportado", func(r *ports.ResultadoPreflightFirmaR5V2) { r.PasoPendiente = 3 }},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			alterado := r
			c.alterar(&alterado)
			if ValidarResultadoPreflightFirmaR5V2(alterado, q) == nil {
				t.Fatal("descriptor alterado admitido")
			}
		})
	}
	v1, _, _ := servicioPreflightPrueba(nil)
	if _, err := v1.ConsultarV2(context.Background(), q); !errors.Is(err, ports.ErrPreflightFirmaR5NoDisponible) {
		t.Fatalf("servicio V1 accedió a V2: %v", err)
	}
}
