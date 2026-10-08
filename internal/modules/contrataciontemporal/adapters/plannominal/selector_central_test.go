package plannominal

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	firma "vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

type fuenteSeleccionPrueba struct {
	r       ports.SeleccionFirmanteV2
	err     error
	pedidas []ports.SolicitudSeleccionFirmanteV2
}

func (f *fuenteSeleccionPrueba) SeleccionarFirmanteV2(_ context.Context, q ports.SolicitudSeleccionFirmanteV2) (ports.SeleccionFirmanteV2, error) {
	f.pedidas = append(f.pedidas, q)
	return f.r, f.err
}

func seleccionPrueba(m ports.MaterialFirmaVerificadaV2) ports.SeleccionFirmanteV2 {
	h := strings.Repeat("9", 64)
	return ports.SeleccionFirmanteV2{PersonaRef: m.FirmantePrincipalRef, CuentaRef: "cta_prueba", PerfilActivoRef: m.PerfilActivoFirmanteRef,
		RolID: m.RolIDFirmante, CargoRef: "cargo:prueba", EnlaceEjercicioRef: "enc_prueba",
		VinculoCertificado:     ports.ReferenciaVersionadaFirmanteV2{Referencia: "vcc_prueba", Version: 1, HuellaSHA256: h},
		Asignacion:             ports.ReferenciaVersionadaFirmanteV2{Referencia: "asignacion:prueba:v1", Version: 1, HuellaSHA256: h},
		AsignacionVigenteDesde: "2026-01-01T00:00:00Z", AsignacionVigenteHasta: "2027-01-01T00:00:00Z",
		RolRef: "rol:central:v1", RolHuellaSHA256: h, ControlRol: ports.ReferenciaVersionadaFirmanteV2{Referencia: "rol:central:v1", Version: 1, HuellaSHA256: h}}
}

var motivoSelectorPrueba = vd.ReferenciaEntradaCatalogo{CatalogoID: "motivos", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "firma"}

// La selección central da el descriptor que acepta el consumidor V2, con el
// recurso del documento exacto y los datos del paso, y pide a AUT56 el cargo,
// rol, acción, tipo, finalidad, organización y unidad del paso.
func TestSelectorCentralDescriptorDelPaso(t *testing.T) {
	for _, orden := range []int{1, 2} {
		base, m, _, _, _ := descriptorPrueba(t, orden)
		fuente := &fuenteSeleccionPrueba{r: seleccionPrueba(m)}
		sel, err := NuevoSelectorCentralFirmanteV2(fuente, motivoSelectorPrueba)
		if err != nil {
			t.Fatal(err)
		}
		f, err := NuevaFuenteDescriptorFirmaV2(base.plan, sel)
		if err != nil {
			t.Fatal(err)
		}
		d, err := f.DescriptorFirmaV2(t.Context(), m)
		if err != nil {
			t.Fatalf("paso %d: %v", orden, err)
		}
		canon, err := firma.CanonicoDescriptorFirmaVerificadaV2(m, d)
		if err != nil || firma.ValidarDescriptorFirmaVerificadaV2(m, canon) != nil {
			t.Fatalf("descriptor incompatible con el consumidor V2: %v", err)
		}
		q := fuente.pedidas[0]
		if len(fuente.pedidas) != 1 || q.CertificadoHuella != m.CertificadoHuella || q.CargoRef != "cargo:prueba" || q.RolID != m.RolIDFirmante ||
			q.TipoRecurso != "documento" || q.Accion != d.Accion || q.Finalidad != d.Finalidad || q.UnidadRef != m.UnidadFirmanteRef ||
			d.Recurso.DocumentoRef != m.OriginalRef || d.Recurso.RecursoAutorizableRef != m.OriginalRef || d.Seleccion.EnlaceEjercicioRef != "enc_prueba" ||
			(orden == 2) != (d.Recurso.EntradaRevision != nil) || d.Motivo != motivoSelectorPrueba {
			t.Fatalf("paso %d: descriptor o solicitud distintos: %+v %+v", orden, d, q)
		}
	}
}

// La huella del recurso cambia con el documento: misma competencia, otro
// original, otra huella.
func TestHuellaContextoRecursoLigadaAlDocumento(t *testing.T) {
	r := vd.RecursoFirmaHistoricaV1{OrganizacionRef: "org", UnidadRef: "uni", ExpedienteRef: "exp", DocumentoRef: "doc:a", TipoRecurso: "documento",
		Original: vd.ReferenciaHistoricaCompetenciaV1{Referencia: "doc:a", Version: 1, HuellaSHA256: strings.Repeat("c", 64)}}
	a, err := HuellaContextoRecursoFirmaV2("contexto.v1", r)
	r.DocumentoRef, r.Original.Referencia = "doc:b", "doc:b"
	b, err2 := HuellaContextoRecursoFirmaV2("contexto.v1", r)
	if err != nil || err2 != nil || a == b || len(a) != 64 {
		t.Fatal("la huella del recurso no distingue el documento")
	}
	if _, err := HuellaContextoRecursoFirmaV2("", r); err == nil {
		t.Fatal("esquema vacío aceptado")
	}
}

// Una selección de otra persona o de otro perfil, o una fuente que no
// acredita, no dan descriptor.
func TestSelectorCentralRechazaSeleccionAjena(t *testing.T) {
	for caso, cambiar := range map[string]func(*fuenteSeleccionPrueba){
		"otra_persona":  func(f *fuenteSeleccionPrueba) { f.r.PersonaRef = "per_otra" },
		"otro_perfil":   func(f *fuenteSeleccionPrueba) { f.r.PerfilActivoRef = "prf_otro" },
		"otro_cargo":    func(f *fuenteSeleccionPrueba) { f.r.CargoRef = "cargo:otro" },
		"no_acreditada": func(f *fuenteSeleccionPrueba) { f.err = ports.ErrCompetenciaFirmanteNoAcreditada },
	} {
		base, m, _, _, _ := descriptorPrueba(t, 1)
		fuente := &fuenteSeleccionPrueba{r: seleccionPrueba(m)}
		cambiar(fuente)
		sel, _ := NuevoSelectorCentralFirmanteV2(fuente, motivoSelectorPrueba)
		f, _ := NuevaFuenteDescriptorFirmaV2(base.plan, sel)
		if _, err := f.DescriptorFirmaV2(t.Context(), m); err == nil {
			t.Fatalf("%s aceptado", caso)
		}
	}
	if _, err := NuevoSelectorCentralFirmanteV2(nil, motivoSelectorPrueba); err == nil {
		t.Fatal("selector sin fuente")
	}
}

// La fuente de competencia elige el paso del plan con los datos de la
// solicitud y devuelve la evidencia de la selección central.
func TestFuenteCompetenciaFirmantePlan(t *testing.T) {
	base, m, _, _, _ := descriptorPrueba(t, 1)
	fuente := &fuenteSeleccionPrueba{r: seleccionPrueba(m)}
	ahora := time.Date(2026, 10, 6, 10, 0, 0, 0, time.UTC)
	c, err := NuevaFuenteCompetenciaFirmantePlanV2(base.plan, fuente, func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	q := ports.SolicitudCompetenciaFirmante{CatalogoVersion: m.CatalogoVersion, OrganizacionRef: m.OrganizacionRef, ExpedienteRef: m.ExpedienteRef,
		Documento: m.Documento, CatalogoRef: m.CatalogoRef, CatalogoHuella: m.CatalogoHuella, PasoRef: m.PasoRef, PasoOrden: m.PasoOrden,
		PerfilFirmanteRef: m.PerfilFirmanteRef, FirmanteRef: m.FirmanteRef, CertificadoHuella: m.CertificadoHuella}
	e, err := c.AcreditarCompetenciaFirmante(t.Context(), q)
	if err != nil || e.Solicitud != q || !e.Vigente || e.FirmantePrincipalRef != m.FirmantePrincipalRef || e.CuentaFirmanteRef != "cta_prueba" ||
		e.VinculoCredencialFirmanteRef != "vcc_prueba" || e.VinculoCredencialFirmanteRevision != 1 || e.UnidadFirmanteRef != m.UnidadFirmanteRef ||
		e.AsignacionVigenteDesde != "2026-01-01T00:00:00Z" || e.CompetenciaComprobadaEn != "2026-10-06T10:00:00Z" ||
		e.ControlVigenciaFirmanteRevision != 1 || e.CargoFirmante != "cargo:prueba" {
		t.Fatalf("evidencia distinta: %+v %v", e, err)
	}
	otro := q
	otro.PerfilFirmanteRef = "perfil:otro"
	if _, err := c.AcreditarCompetenciaFirmante(t.Context(), otro); !errors.Is(err, ports.ErrCompetenciaFirmanteNoAcreditada) || len(fuente.pedidas) != 1 {
		t.Fatalf("paso inexistente sin denegación o con consulta: %v", err)
	}
	// Las tres fechas deben pasar el validador de la aplicación y de CT170/172
	// (sin ceros finales), con segundos exactos y con fracción.
	for _, ns := range []int{0, 120000000, 123456789} {
		ahora = time.Date(2026, 10, 6, 10, 0, 0, ns, time.UTC)
		fuente.r.AsignacionVigenteDesde = time.Date(2026, 1, 1, 0, 0, 0, ns, time.UTC).Format(time.RFC3339Nano)
		e, err := c.AcreditarCompetenciaFirmante(t.Context(), q)
		for _, v := range []string{e.AsignacionVigenteDesde, e.AsignacionVigenteHasta, e.CompetenciaComprobadaEn} {
			if _, ok := ports.FechaFirmaExternaCanonica(v); err != nil || !ok {
				t.Fatalf("fecha %q no canónica (%d ns): %v", v, ns, err)
			}
		}
	}
	fuente.r = seleccionPrueba(m)
	for caso, cambiar := range map[string]func(*ports.SeleccionFirmanteV2){
		"otro_rol":   func(r *ports.SeleccionFirmanteV2) { r.RolID = "rol_otro" },
		"otro_cargo": func(r *ports.SeleccionFirmanteV2) { r.CargoRef = "cargo:otro" },
		"fecha_mala": func(r *ports.SeleccionFirmanteV2) { r.AsignacionVigenteHasta = "2027-01-01" },
	} {
		fuente.r = seleccionPrueba(m)
		cambiar(&fuente.r)
		if _, err := c.AcreditarCompetenciaFirmante(t.Context(), q); !errors.Is(err, ports.ErrCompetenciaFirmanteNoDisponible) {
			t.Fatalf("%s aceptado: %v", caso, err)
		}
	}
	fuente.r = seleccionPrueba(m)
	fuente.err = ports.ErrCompetenciaFirmanteNoAcreditada
	if _, err := c.AcreditarCompetenciaFirmante(t.Context(), q); !errors.Is(err, ports.ErrCompetenciaFirmanteNoAcreditada) {
		t.Fatalf("no acreditada cambiada: %v", err)
	}
	fuente.err = errors.New("caída")
	if _, err := c.AcreditarCompetenciaFirmante(t.Context(), q); !errors.Is(err, ports.ErrCompetenciaFirmanteNoDisponible) {
		t.Fatalf("caída sin no disponible: %v", err)
	}
	// Dos pasos que sólo difieren en la unidad: la elección es ambigua.
	plan, err := base.plan.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := pasoUnicoDeSolicitud(plan, q); !ok {
		t.Fatal("paso único no elegido")
	}
	otraUnidad := plan.Pasos[0]
	otraUnidad.UnidadRef = "unidad:otra"
	plan.Pasos = append(plan.Pasos, otraUnidad)
	if _, ok := pasoUnicoDeSolicitud(plan, q); ok {
		t.Fatal("dos pasos con otra unidad no ambiguos")
	}
}
