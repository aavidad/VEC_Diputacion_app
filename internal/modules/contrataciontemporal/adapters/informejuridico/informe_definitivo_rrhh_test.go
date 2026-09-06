package informejuridico

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
)

// Fixture de representación, no prueba de autorización ni de tramitación real.
func detalleInformeDefinitivoPrueba() ports.DetalleExpedienteRRHH {
	inicio := time.Date(2026, 9, 6, 1, 0, 0, 0, time.UTC)
	periodo := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	d := ports.DetalleExpedienteRRHH{
		Resumen: ports.ResumenExpedienteRRHH{
			ExpedienteRef: "expediente:ct:sintetico:informe", OrganizacionRef: "organizacion:sintetica:001",
			NumeroVisible: "2026/CT-0001", Version: 7, FlujoRef: "flujo:ct:sintetico", FlujoVersion: 1,
			FlujoHuella: strings.Repeat("a", 64), FaseClave: "nombramiento", EstadoClave: domain.EstadoEnCurso,
			CentroRef: "centro:sintetico:001", CategoriaRef: "categoria:sintetica:c2", ModalidadClave: "sustitucion",
			UnidadRef: "unidad:sintetica:rrhh", CreadoEn: inicio, ActualizadoEn: inicio.Add(6 * time.Minute),
		},
		Solicitud:  ports.SolicitudOperativaRRHH{GrupoSubgrupo: "C2", MotivoClave: "sustitucion", PeriodoInicio: periodo, PeriodoFin: periodo.AddDate(0, 3, 0)},
		Analisis:   &ports.AnalisisOperativoRRHH{ModalidadClave: "sustitucion", CategoriaRef: "categoria:sintetica:c2", CausaClave: "necesidad_temporal", PeriodoInicio: periodo, PeriodoFin: periodo.AddDate(0, 3, 0), PorcentajeJornada: 10000, ResultadoRC: domain.RCValidada},
		Cobertura:  &ports.CoberturaOperativaRRHH{ViaClave: "bolsa_vigente", DecisionGobernada: true},
		Asignacion: &ports.AsignacionOperativaRRHH{UnidadRef: "unidad:sintetica:rrhh", AsignadaEn: inicio},
	}
	for i := uint64(1); i <= 7; i++ {
		h := ports.HitoExpedienteRRHH{Secuencia: i, VersionExpediente: i, AccionClave: "actuacion_sintetica", RealizadaEn: inicio.Add(time.Duration(i-1) * time.Minute), FaseOrigen: "fiscalizacion", FaseDestino: "fiscalizacion", EstadoOrigen: domain.EstadoEnCurso, EstadoDestino: domain.EstadoEnCurso}
		if i == 1 {
			h.FaseOrigen = ""
			h.EstadoOrigen = domain.EstadoPendiente
		}
		if i == 7 {
			h.FaseDestino = "nombramiento"
			h.AccionClave = "registrar_propuesta_formalizacion"
		}
		d.Hitos = append(d.Hitos, h)
	}
	return d
}

func TestInformeDefinitivoPDFRealDeterministaYMarcado(t *testing.T) {
	d := detalleInformeDefinitivoPrueba()
	r := RenderizadorBorradorDesarrollo{PDF: pdf.Renderizador{}}
	primero, err := r.RenderizarBorrador(context.Background(), ports.BorradorInformeDefinitivo, d)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := r.RenderizarBorrador(context.Background(), ports.BorradorInformeDefinitivo, d)
	if err != nil || !bytes.Equal(primero, segundo) || !bytes.HasPrefix(primero, []byte("%PDF-")) {
		t.Fatalf("PDF no determinista o inválido: %v", err)
	}
	contenido := contenidoInformeDefinitivoDesarrollo(d)
	texto := contenido.Titulo + "\n" + strings.Join(contenido.Parrafos, "\n")
	for _, esperado := range []string{"NO FIRMADO NI VALIDADO", "2026/CT-0001", "100,00 %", "Versión de origen: 7", "Pendiente de completar", "no se han inventado"} {
		if !strings.Contains(texto, esperado) {
			t.Fatalf("falta %q", esperado)
		}
	}
	if strings.Contains(texto, "organizacion:sintetica:001") {
		t.Fatal("serialización innecesaria de organización")
	}
}

func TestInformeDefinitivoRechazaAntecedenteAusenteYCancelacion(t *testing.T) {
	r := RenderizadorBorradorDesarrollo{PDF: pdf.Renderizador{}}
	for _, mutar := range []func(*ports.DetalleExpedienteRRHH){
		func(d *ports.DetalleExpedienteRRHH) { d.Resumen.Version = 6 },
		func(d *ports.DetalleExpedienteRRHH) { d.Hitos[6].AccionClave = "otra_actuacion" },
		func(d *ports.DetalleExpedienteRRHH) { d.Analisis = nil },
	} {
		d := detalleInformeDefinitivoPrueba()
		mutar(&d)
		for _, tipo := range []ports.TipoBorradorRRHH{ports.BorradorInformeDefinitivo, ports.BorradorResolucion, ports.BorradorDiligencia, ports.BorradorTomaPosesion} {
			b, err := r.RenderizarBorrador(context.Background(), tipo, d)
			if !errors.Is(err, ports.ErrBorradorRRHHNoDisponible) || len(b) != 0 {
				t.Fatal("antecedente inválido produjo documento")
			}
		}
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if b, err := r.RenderizarBorrador(ctx, ports.BorradorResolucion, detalleInformeDefinitivoPrueba()); !errors.Is(err, context.Canceled) || len(b) != 0 {
		t.Fatal("cancelación produjo documento")
	}
	if b, err := r.RenderizarBorrador(context.Background(), "desconocido", detalleInformeDefinitivoPrueba()); !errors.Is(err, ports.ErrBorradorRRHHNoDisponible) || len(b) != 0 {
		t.Fatal("tipo desconocido produjo documento")
	}
}

func TestResolucionPDFRealDeterministaSinAutoridadInventada(t *testing.T) {
	d := detalleInformeDefinitivoPrueba()
	r := RenderizadorBorradorDesarrollo{PDF: pdf.Renderizador{}}
	primero, err := r.RenderizarBorrador(context.Background(), ports.BorradorResolucion, d)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := r.RenderizarBorrador(context.Background(), ports.BorradorResolucion, d)
	if err != nil || !bytes.Equal(primero, segundo) || !bytes.HasPrefix(primero, []byte("%PDF-")) {
		t.Fatalf("resolución PDF no determinista o inválida: %v", err)
	}
	contenido := contenidoResolucionDesarrollo(d)
	texto := contenido.Titulo + "\n" + strings.Join(contenido.Parrafos, "\n")
	for _, esperado := range []string{"Resolución — borrador", "NO FIRMADO NI VALIDADO", "2026/CT-0001", "100,00 %", "Versión de origen: 7", "Órgano competente: pendiente", "Persona propuesta: identificación autorizada pendiente", "SIN EFECTOS ADMINISTRATIVOS"} {
		if !strings.Contains(texto, esperado) {
			t.Fatalf("falta %q", esperado)
		}
	}
	if strings.Contains(texto, "organizacion:sintetica:001") || strings.Contains(texto, "RESUELVO") {
		t.Fatal("el borrador expone datos innecesarios o aparenta resolver")
	}
}

func TestDiligenciaPDFRealDeterministaSinHechosInventados(t *testing.T) {
	d := detalleInformeDefinitivoPrueba()
	r := RenderizadorBorradorDesarrollo{PDF: pdf.Renderizador{}}
	primero, err := r.RenderizarBorrador(context.Background(), ports.BorradorDiligencia, d)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := r.RenderizarBorrador(context.Background(), ports.BorradorDiligencia, d)
	if err != nil || !bytes.Equal(primero, segundo) || !bytes.HasPrefix(primero, []byte("%PDF-")) {
		t.Fatalf("diligencia PDF no determinista o inválida: %v", err)
	}
	contenido := contenidoDiligenciaDesarrollo(d)
	texto := contenido.Titulo + "\n" + strings.Join(contenido.Parrafos, "\n")
	for _, esperado := range []string{"Diligencia — borrador", "NO FIRMADO NI VALIDADO", "2026/CT-0001", "Versión de origen: 7", "Objeto específico de la diligencia: pendiente", "no la fecha de una comparecencia", "No se afirma que ninguna persona haya comparecido", "SIN EFECTOS ADMINISTRATIVOS"} {
		if !strings.Contains(texto, esperado) {
			t.Fatalf("falta %q", esperado)
		}
	}
	if strings.Contains(texto, "organizacion:sintetica:001") || strings.Contains(texto, "DOY FE") {
		t.Fatal("el borrador expone datos innecesarios o aparenta certificar")
	}
}

func TestTomaPosesionPDFRealDeterministaSinIncorporacionInventada(t *testing.T) {
	d := detalleInformeDefinitivoPrueba()
	r := RenderizadorBorradorDesarrollo{PDF: pdf.Renderizador{}}
	primero, err := r.RenderizarBorrador(context.Background(), ports.BorradorTomaPosesion, d)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := r.RenderizarBorrador(context.Background(), ports.BorradorTomaPosesion, d)
	if err != nil || !bytes.Equal(primero, segundo) || !bytes.HasPrefix(primero, []byte("%PDF-")) {
		t.Fatalf("toma de posesión PDF no determinista o inválida: %v", err)
	}
	contenido := contenidoTomaPosesionDesarrollo(d)
	texto := contenido.Titulo + "\n" + strings.Join(contenido.Parrafos, "\n")
	for _, esperado := range []string{"Toma de posesión — borrador", "NO FIRMADO NI VALIDADO", "2026/CT-0001", "Versión de origen: 7", "Esa fecha no acredita comparecencia", "No se afirma que estos hechos hayan ocurrido", "confirmación de incorporación: pendientes", "SIN EFECTOS ADMINISTRATIVOS"} {
		if !strings.Contains(texto, esperado) {
			t.Fatalf("falta %q", esperado)
		}
	}
	if strings.Contains(texto, "organizacion:sintetica:001") || strings.Contains(texto, "HA TOMADO POSESIÓN") {
		t.Fatal("el borrador expone datos innecesarios o aparenta certificar posesión")
	}
}
