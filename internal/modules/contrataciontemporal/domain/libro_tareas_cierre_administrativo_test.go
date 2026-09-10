package domain

import (
	"strings"
	"testing"
)

func libroCierrePrueba() PublicacionLibroTareasCierreAdministrativoEjercicio {
	return PublicacionLibroTareasCierreAdministrativoEjercicio{Referencia: "libro:tareas:cierre:ejercicio", Version: 1, Ambito: AmbitoLibroTareasCierreAdministrativoEjercicio, Tareas: []TareaLibroCierreAdministrativoEjercicio{{TareaIncorporacionOriginalAcreditada, EvidenciaIncorporacionOriginal}, {TareaPrimeraAnotacionAcreditada, EvidenciaPrimeraAnotacion}}}
}
func evidenciaCierrePrueba(tipo TipoEvidenciaTareaCierreAdministrativoEjercicio, ref string, v uint64, h string) EvidenciaTareaCierreAdministrativoEjercicio {
	return EvidenciaTareaCierreAdministrativoEjercicio{Tipo: tipo, Referencia: ref, OrganizacionRef: "organizacion:ejercicio:rrhh", ExpedienteRef: "expediente:ejercicio:001", SeguimientoRef: "seguimiento:ejercicio:001", VersionSeguimientoOriginal: 1, HuellaRaizSeguimientoSHA256: strings.Repeat("a", 64), VersionEvidencia: v, HuellaEvidenciaSHA256: h}
}
func snapshotCierrePrueba() SnapshotTareasCierreAdministrativoEjercicio {
	return SnapshotTareasCierreAdministrativoEjercicio{Libro: libroCierrePrueba(), OrganizacionRef: "organizacion:ejercicio:rrhh", ExpedienteRef: "expediente:ejercicio:001", SeguimientoRef: "seguimiento:ejercicio:001", Estados: []EstadoTareaCierreAdministrativoEjercicio{{Clave: TareaIncorporacionOriginalAcreditada, Evidencia: evidenciaCierrePrueba(EvidenciaIncorporacionOriginal, "recibo:incorporacion:001", 1, strings.Repeat("b", 64))}, {Clave: TareaPrimeraAnotacionAcreditada, Pendiente: true, Evidencia: evidenciaCierrePrueba(EvidenciaPrimeraAnotacion, "recibo:anotacion:001", 2, strings.Repeat("c", 64))}}}
}
func TestLibroTareasCierreAdministrativoEjercicioPublicaConjuntoExacto(t *testing.T) {
	if _, e := PublicarLibroTareasCierreAdministrativoEjercicio(libroCierrePrueba()); e != nil {
		t.Fatal(e)
	}
	for _, m := range []func(*PublicacionLibroTareasCierreAdministrativoEjercicio){func(p *PublicacionLibroTareasCierreAdministrativoEjercicio) { p.Ambito = "otro" }, func(p *PublicacionLibroTareasCierreAdministrativoEjercicio) { p.Tareas = p.Tareas[:1] }, func(p *PublicacionLibroTareasCierreAdministrativoEjercicio) {
		p.Tareas[1].Clave = TareaIncorporacionOriginalAcreditada
	}} {
		p := libroCierrePrueba()
		m(&p)
		if _, e := PublicarLibroTareasCierreAdministrativoEjercicio(p); e == nil {
			t.Fatal("publicacion invalida aceptada")
		}
	}
}
func TestInventarioTareasCierreAdministrativoEjercicioRechazaFuentesCruzadas(t *testing.T) {
	s, e := NuevoSnapshotTareasCierreAdministrativoEjercicio(snapshotCierrePrueba())
	if e != nil || s.Total() != 2 || s.Pendientes() != 1 {
		t.Fatal("snapshot válido rechazado")
	}
	for _, m := range []func(*SnapshotTareasCierreAdministrativoEjercicio){func(s *SnapshotTareasCierreAdministrativoEjercicio) { s.Estados = s.Estados[:1] }, func(s *SnapshotTareasCierreAdministrativoEjercicio) {
		s.Estados[1].Evidencia.OrganizacionRef = "organizacion:otra"
	}, func(s *SnapshotTareasCierreAdministrativoEjercicio) {
		s.Estados[1].Evidencia.ExpedienteRef = "expediente:otro"
	}, func(s *SnapshotTareasCierreAdministrativoEjercicio) {
		s.Estados[1].Evidencia.SeguimientoRef = "seguimiento:otro"
	}, func(s *SnapshotTareasCierreAdministrativoEjercicio) {
		s.Estados[1].Evidencia.VersionSeguimientoOriginal = 2
	}, func(s *SnapshotTareasCierreAdministrativoEjercicio) {
		s.Estados[1].Evidencia.HuellaRaizSeguimientoSHA256 = strings.Repeat("d", 64)
	}, func(s *SnapshotTareasCierreAdministrativoEjercicio) {
		s.Estados[1].Evidencia.Referencia = s.Estados[0].Evidencia.Referencia
	}, func(s *SnapshotTareasCierreAdministrativoEjercicio) {
		s.Estados[1].Evidencia.HuellaEvidenciaSHA256 = s.Estados[0].Evidencia.HuellaEvidenciaSHA256
	}} {
		x := snapshotCierrePrueba()
		m(&x)
		if _, e := NuevoSnapshotTareasCierreAdministrativoEjercicio(x); e == nil {
			t.Fatal("fuente invalida aceptada")
		}
	}
}
