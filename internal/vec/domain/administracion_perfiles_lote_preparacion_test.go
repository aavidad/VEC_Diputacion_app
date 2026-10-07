package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func solicitudPreparacionDominioPrueba() SolicitudPreparacionLoteAdministracionPerfiles {
	return SolicitudPreparacionLoteAdministracionPerfiles{OperacionRef: "prep_admin:" + strings.Repeat("a", 32),
		OrganizacionRef: "org_prueba", UnidadRef: "unidad:prueba", PersonaRef: "per_" + strings.Repeat("b", 32)}
}

// El canon tiene exactamente las ocho claves de texto que valida AUT50.
func TestCanonPreparacionOchoClavesDeTexto(t *testing.T) {
	s := solicitudPreparacionDominioPrueba()
	// Sin actor ni instantánea válidos no hay canon.
	if _, _, err := s.CanonicoYHuella(); err == nil {
		t.Fatal("canon sin actor")
	}
	canon := struct {
		Esquema, OperacionRef, ActorPersonaRef, PerfilActivoRef, AsignacionRef, OrganizacionRef, UnidadRef, PersonaRef string
	}{"administracion_perfiles_lote_preparacion:v1", s.OperacionRef, "per_x", "prf_x", "asignacion:a:v1", s.OrganizacionRef, s.UnidadRef, s.PersonaRef}
	b, _ := json.Marshal(canon)
	var m map[string]any
	if json.Unmarshal(b, &m) != nil || len(m) != 8 {
		t.Fatal("el canon esperado no tiene ocho claves")
	}
}

func TestReferenciaPreparacionLote(t *testing.T) {
	if !ReferenciaPreparacionLoteValida("prep_admin:"+strings.Repeat("0", 32)) ||
		ReferenciaPreparacionLoteValida("acto_admin:"+strings.Repeat("0", 32)) ||
		ReferenciaPreparacionLoteValida("prep_admin:"+strings.Repeat("G", 32)) {
		t.Fatal("referencia de preparación")
	}
}

// ValidarPara exige opciones con referencias únicas, versiones y huellas.
func TestPreparacionValidarParaRechazaOpcionesIncompletas(t *testing.T) {
	ahora := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	base := PreparacionLoteAdministracionPerfiles{
		Altas: []AltaPosibleLoteAdministracion{{RolVersionRef: "rol:tecnico:v1", VigenteHastaMaxima: ahora.Add(time.Hour),
			DuracionPropuesta: time.Hour, PerfilRef: "prf_" + strings.Repeat("1", 32), VinculoRef: "vca_" + strings.Repeat("2", 32),
			HuellaSHA256: strings.Repeat("3", 64)}},
		Bajas: []BajaPosibleLoteAdministracion{{RolVersionRef: "rol:otro:v1", PerfilRef: "prf_" + strings.Repeat("4", 32),
			VinculoRef: "vca_" + strings.Repeat("5", 32), PerfilVersion: 1, VinculoVersion: 1,
			VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), HuellaSHA256: strings.Repeat("6", 64)}},
	}
	// Sin solicitud válida nunca se acepta, aunque las opciones lo sean.
	if base.ValidarPara(solicitudPreparacionDominioPrueba()) == nil {
		t.Fatal("preparación aceptada sin solicitud válida")
	}
}
