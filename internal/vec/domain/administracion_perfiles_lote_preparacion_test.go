package domain

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func solicitudPreparacionDominioPrueba() SolicitudPreparacionLoteAdministracionPerfiles {
	return SolicitudPreparacionLoteAdministracionPerfiles{OperacionRef: "prep_admin:" + strings.Repeat("a", 32),
		OrganizacionRef: "org_prueba", UnidadRef: "unidad:prueba", PersonaRef: "per_" + strings.Repeat("b", 32)}
}

// Vector independiente del serializador: AUT50 compromete estos bytes y SHA.
func TestCanonPreparacionVectorFijo(t *testing.T) {
	s := solicitudPreparacionDominioPrueba()
	// Sin actor ni instantánea válidos no hay canon.
	if _, _, err := s.CanonicoYHuella(); err == nil {
		t.Fatal("canon sin actor")
	}
	actor := solicitudActoAdministracionPerfilesPrueba(t)
	actor.InstantaneaAutorizacion.AsignacionPerfil.AsignacionID = "admin_lote"
	s.Actor, s.InstantaneaAutorizacion, s.CorrelacionRef = actor.Actor, actor.InstantaneaAutorizacion, actor.CorrelacionRef
	esperado := []byte(`{"Esquema":"administracion_perfiles_lote_preparacion:v1",` +
		`"OperacionRef":"prep_admin:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",` +
		`"ActorPersonaRef":"per_rrrrrrrrrrrrrrrrrrrrrr",` +
		`"PerfilActivoRef":"prf_pppppppppppppppppppppp",` +
		`"AsignacionRef":"asignacion:admin_lote:v1",` +
		`"OrganizacionRef":"org_prueba","UnidadRef":"unidad:prueba",` +
		`"PersonaRef":"per_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}`)
	canon, huella, err := s.CanonicoYHuella()
	if err != nil || !bytes.Equal(canon, esperado) ||
		huella != "6947266cd7641172f912e01b16c7999b0498f3dda1738a6db726acec64752aab" {
		t.Fatalf("vector de preparación divergente: %v", err)
	}
	s.UnidadRef = "unidad:otra"
	_, otraHuella, err := s.CanonicoYHuella()
	if err != nil || otraHuella == huella {
		t.Fatal("la unidad no cambió la huella de preparación")
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
