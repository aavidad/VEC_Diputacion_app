package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

type repositorioCircuitoPrueba struct{ llamadas int }

func (r *repositorioCircuitoPrueba) Decidir(context.Context, dietasports.IdentidadEfectivaCircuito, dietasports.SolicitudDecisionCircuito) (dietasports.ResultadoCircuitoComision, error) {
	r.llamadas++
	return dietasports.ResultadoCircuitoComision{}, nil
}
func (r *repositorioCircuitoPrueba) ListarPendientes(context.Context, dietasports.IdentidadEfectivaCircuito, dietasports.ConsultaBandejaCircuito) (dietasports.PaginaBandejaCircuito, error) {
	r.llamadas++
	return dietasports.PaginaBandejaCircuito{}, nil
}

func (r *repositorioCircuitoPrueba) ConsultarDocumento(context.Context, dietasports.IdentidadEfectivaCircuito, dietasports.SolicitudDocumentoCircuito) (dietasports.DocumentoCircuito, error) {
	r.llamadas++
	return dietasports.DocumentoCircuito{}, nil
}

func TestDocumentoCircuitoContratoPropioYFormaValidada(t *testing.T) {
	ref := "dco_" + strings.Repeat("d", 22)
	s := dietasports.SolicitudDocumentoCircuito{Referencia: ref, Etapa: domain.EtapaLiquidacion, UnidadRef: "unidad:uno"}
	a, r, f := ContratoCircuito(dietasports.SolicitudOperacionCircuito{Operacion: dietasports.OperacionConsultarDocumentoCircuito, Documento: s})
	if a != AccionConsultarDocumentoCircuito || r != ref || f != FinalidadConsultarDocumentoCircuito {
		t.Fatalf("contrato documento: %s %s %s", a, r, f)
	}
	propio := true
	d := dietasports.DocumentoCircuito{Referencia: ref, NumeroDocumento: "VEC-D-2026-000001", FechaApertura: "2026-09-25T08:00:00.000000Z", Estado: domain.EstadoPendienteLiquidacion, Version: 4,
		FechaInicio: "2026-09-25", FechaFin: "2026-09-25", HoraInicio: "08:00", HoraFin: "15:00", Motivo: "Reunión técnica", CodigosRuta: []string{"18087", "18140"},
		VehiculoPropio: &propio, Rutas: json.RawMessage(`[]`), Calculo: json.RawMessage(`{}`), Documento: json.RawMessage(`{"lineas":[]}`)}
	if err := ValidarDocumentoCircuito(d, s); err != nil {
		t.Fatal(err)
	}
	for nombre, cambiar := range map[string]func(*dietasports.DocumentoCircuito){
		"otra etapa":         func(x *dietasports.DocumentoCircuito) { x.Estado = domain.EstadoPendienteAutorizacion },
		"otra referencia":    func(x *dietasports.DocumentoCircuito) { x.Referencia = "dco_" + strings.Repeat("e", 22) },
		"rutas sin vehículo": func(x *dietasports.DocumentoCircuito) { x.VehiculoPropio = nil },
		"documento lista":    func(x *dietasports.DocumentoCircuito) { x.Documento = json.RawMessage(`[]`) },
		"hora":               func(x *dietasports.DocumentoCircuito) { x.HoraFin = "25:00" },
	} {
		x := d
		cambiar(&x)
		if ValidarDocumentoCircuito(x, s) == nil {
			t.Fatalf("%s aceptado", nombre)
		}
	}
	repo := &repositorioCircuitoPrueba{}
	servicio, _ := NuevoServicioCircuitoComision(repo)
	if _, err := servicio.ConsultarDocumento(context.Background(), dietasports.IdentidadEfectivaCircuito{}, s); !errors.Is(err, dietasports.ErrAccesoCircuitoDenegado) || repo.llamadas != 0 {
		t.Fatalf("documento sin V3: %v %d", err, repo.llamadas)
	}
}

func TestCircuitoDistingueAccionesYExigeUnidadInterna(t *testing.T) {
	ref := "dco_" + strings.Repeat("a", 22)
	decision := dietasports.SolicitudDecisionCircuito{Referencia: ref, UnidadRef: "unidad:uno", Etapa: domain.EtapaRevision, Decision: domain.DecisionDevolver, Motivo: "Falta documento", ClaveIdempotencia: "clave_0123456789abcdef", VersionEsperada: 2}
	if err := ValidarSolicitudDecisionCircuito(decision); err != nil {
		t.Fatal(err)
	}
	accion, recurso, finalidad := ContratoCircuito(dietasports.SolicitudOperacionCircuito{Operacion: dietasports.OperacionDecidirCircuito, Decision: decision})
	if accion != "dietas.documento.revisar" || recurso != ref || finalidad != "revisar_documento_dietas" {
		t.Fatalf("contrato D6: %s %s %s", accion, recurso, finalidad)
	}
	consulta := dietasports.ConsultaBandejaCircuito{Etapa: domain.EtapaRevision, UnidadRef: "unidad:uno", Limite: 20, FechaDesde: "2026-09-01", FechaHasta: "2026-09-30"}
	a, r, f := ContratoCircuito(dietasports.SolicitudOperacionCircuito{Operacion: dietasports.OperacionListarBandeja, Consulta: consulta})
	if a != "dietas.bandeja.revision.consultar" || r != "dietas:bandeja:revision" || f != "consultar_bandeja_revision_dietas" {
		t.Fatalf("contrato D8: %s %s %s", a, r, f)
	}
	decision.UnidadRef = ""
	if !errors.Is(ValidarSolicitudDecisionCircuito(decision), domain.ErrDecisionCircuitoInvalida) {
		t.Fatal("decisión sin unidad interna aceptada")
	}
	consulta.FechaDesde = "2026-10-01"
	if !errors.Is(ValidarConsultaBandejaCircuito(consulta), domain.ErrDecisionCircuitoInvalida) {
		t.Fatal("rango invertido aceptado")
	}
}

func TestCircuitoNoLlamaRepositorioSinIdentidadV3(t *testing.T) {
	repo := &repositorioCircuitoPrueba{}
	s, err := NuevoServicioCircuitoComision(repo)
	if err != nil {
		t.Fatal(err)
	}
	decision := dietasports.SolicitudDecisionCircuito{Referencia: "dco_" + strings.Repeat("a", 22), UnidadRef: "unidad:uno", Etapa: domain.EtapaRevision, Decision: domain.DecisionAprobar, ClaveIdempotencia: "clave_0123456789abcdef", VersionEsperada: 1}
	_, err = s.Decidir(context.Background(), dietasports.IdentidadEfectivaCircuito{}, decision)
	if !errors.Is(err, dietasports.ErrAccesoCircuitoDenegado) || repo.llamadas != 0 {
		t.Fatalf("decisión sin V3: %v %d", err, repo.llamadas)
	}
	_, err = s.ListarPendientes(context.Background(), dietasports.IdentidadEfectivaCircuito{}, dietasports.ConsultaBandejaCircuito{Etapa: domain.EtapaRevision, UnidadRef: "unidad:uno", Limite: 20})
	if !errors.Is(err, dietasports.ErrAccesoCircuitoDenegado) || repo.llamadas != 0 {
		t.Fatalf("lista sin V3: %v %d", err, repo.llamadas)
	}
}

// Reenvío: el documento del circuito puede llevar la devolución anterior,
// siempre de una versión previa y con la forma del dominio.
func TestDocumentoCircuitoAceptaDevolucionAnteriorValida(t *testing.T) {
	ref := "dco_" + strings.Repeat("d", 22)
	s := dietasports.SolicitudDocumentoCircuito{Referencia: ref, Etapa: domain.EtapaRevision, UnidadRef: "unidad:uno"}
	d := dietasports.DocumentoCircuito{Referencia: ref, NumeroDocumento: "VEC-D-2026-000001", FechaApertura: "2026-09-25T08:00:00.000000Z", Estado: domain.EstadoEnviadoPendienteRevision, Version: 5,
		FechaInicio: "2026-09-25", FechaFin: "2026-09-25", HoraInicio: "08:00", HoraFin: "15:00", Motivo: "Reunión técnica", CodigosRuta: []string{"18087", "18140"},
		Calculo: json.RawMessage(`{}`), Documento: json.RawMessage(`{"lineas":[]}`),
		Devolucion: &domain.DevolucionComision{Etapa: domain.EtapaAutorizacion, Motivo: "Falta el justificante del taxi", Version: 3, DevueltaEn: "2026-09-23T09:00:00.123456Z"}}
	if err := ValidarDocumentoCircuito(d, s); err != nil {
		t.Fatal(err)
	}
	for nombre, cambiar := range map[string]func(*domain.DevolucionComision){
		"versión actual":  func(x *domain.DevolucionComision) { x.Version = 5 },
		"motivo con BOM":  func(x *domain.DevolucionComision) { x.Motivo += "\ufeff" },
		"etapa inválida":  func(x *domain.DevolucionComision) { x.Etapa = "otra" },
		"fecha sin micro": func(x *domain.DevolucionComision) { x.DevueltaEn = "2026-09-23T09:00:00Z" },
	} {
		x := d
		dev := *d.Devolucion
		cambiar(&dev)
		x.Devolucion = &dev
		if ValidarDocumentoCircuito(x, s) == nil {
			t.Errorf("%s aceptada", nombre)
		}
	}
	x := d
	x.Devolucion = nil
	if err := ValidarDocumentoCircuito(x, s); err != nil {
		t.Fatalf("documento sin reenvío: %v", err)
	}
}

func TestDecisionCircuitoRechazaBordesQueRecortaElNavegador(t *testing.T) {
	ref := "dco_" + strings.Repeat("d", 22)
	for _, motivo := range []string{"\ufeffFalta justificante", "Falta justificante\u0085", "Falta justificante\u00a0"} {
		s := dietasports.SolicitudDecisionCircuito{Referencia: ref, UnidadRef: "unidad:uno", Etapa: domain.EtapaRevision, Decision: domain.DecisionDevolver,
			Motivo: motivo, ClaveIdempotencia: "clave_decision_0001", VersionEsperada: 2}
		if ValidarSolicitudDecisionCircuito(s) == nil {
			t.Errorf("%q aceptado", motivo)
		}
	}
}
