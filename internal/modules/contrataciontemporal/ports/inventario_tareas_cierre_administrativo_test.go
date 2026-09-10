package ports

import (
	"strings"
	"testing"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func entradaInventarioCierrePrueba() EntradaInventarioTareasCierreAdministrativoEjercicio {
	libro := domain.PublicacionLibroTareasCierreAdministrativoEjercicio{Referencia: "libro:tareas:cierre:ejercicio", Version: 1, Ambito: domain.AmbitoLibroTareasCierreAdministrativoEjercicio, Tareas: []domain.TareaLibroCierreAdministrativoEjercicio{{domain.TareaIncorporacionOriginalAcreditada, domain.EvidenciaIncorporacionOriginal}, {domain.TareaPrimeraAnotacionAcreditada, domain.EvidenciaPrimeraAnotacion}}}
	f := func(tipo domain.TipoEvidenciaTareaCierreAdministrativoEjercicio, ref string, v uint64, h string) domain.EvidenciaTareaCierreAdministrativoEjercicio {
		return domain.EvidenciaTareaCierreAdministrativoEjercicio{Tipo: tipo, Referencia: ref, OrganizacionRef: "organizacion:ejercicio:rrhh", ExpedienteRef: "expediente:ejercicio:001", SeguimientoRef: "seguimiento:ejercicio:001", VersionSeguimientoOriginal: 1, HuellaRaizSeguimientoSHA256: strings.Repeat("a", 64), VersionEvidencia: v, HuellaEvidenciaSHA256: h}
	}
	return EntradaInventarioTareasCierreAdministrativoEjercicio{Libro: libro, OrganizacionRef: "organizacion:ejercicio:rrhh", ExpedienteRef: "expediente:ejercicio:001", SeguimientoRef: "seguimiento:ejercicio:001", Estados: []domain.EstadoTareaCierreAdministrativoEjercicio{{Clave: domain.TareaIncorporacionOriginalAcreditada, Evidencia: f(domain.EvidenciaIncorporacionOriginal, "recibo:incorporacion:001", 1, strings.Repeat("b", 64))}, {Clave: domain.TareaPrimeraAnotacionAcreditada, Pendiente: true, Evidencia: f(domain.EvidenciaPrimeraAnotacion, "recibo:anotacion:001", 2, strings.Repeat("c", 64))}}}
}
func TestEntradaInventarioTareasCierreAdministrativoEjercicioDerivaDTO(t *testing.T) {
	i, e := entradaInventarioCierrePrueba().Preparar()
	if e != nil || !i.Completo || i.Total != 2 || i.Pendientes != 1 {
		t.Fatal("DTO no derivado")
	}
}
func TestEntradaInventarioTareasCierreAdministrativoEjercicioRechazaVersionCruzada(t *testing.T) {
	e := entradaInventarioCierrePrueba()
	e.Estados[1].Evidencia.VersionSeguimientoOriginal = 2
	if _, x := e.Preparar(); x == nil {
		t.Fatal("version cruzada aceptada")
	}
}
