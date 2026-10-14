package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func TestMaterialConsultaPropiaAsignacionConservaContratoAntiguo(t *testing.T) {
	actor := actorAsignacionPrueba(t)
	fecha, _ := NuevaFechaCivil("2026-09-20")
	s := SolicitudAsignacionDietas{Actor: actor, RelacionRef: "rel_" + strings.Repeat("a", 24),
		UnidadRef: "unidad_sintetica", FechaReferencia: fecha, Operacion: ConsultarPropiaAsignacionDietas}
	propio, err := NuevoMaterialAsignacionDietasPropia(s)
	if err != nil {
		t.Fatal(err)
	}
	r := propio.Recurso()
	if r.Referencia != s.RelacionRef || r.ModuloID != "personal" || r.Tipo != "asignacion_dietas_propia" ||
		len(r.Ambitos) != 2 || r.Ambitos["persona_ref"] != actor.PersonaRef ||
		r.Ambitos["empleado_ref"] != "emp_"+strings.Repeat("a", 24) || len(r.Atributos) != 5 ||
		r.Atributos["relacion_ref"] != s.RelacionRef || r.Atributos["unidad_ref"] != s.UnidadRef ||
		r.Atributos["fecha_referencia"] != fecha.Texto() || r.Atributos["operacion"] != "consultar_propia" {
		t.Fatalf("recurso propio incompatible: %+v", r)
	}
	suma := sha256.Sum256(propio.Canonico())
	if r.Atributos["material_sha256"] != hex.EncodeToString(suma[:]) {
		t.Fatal("material de la consulta propia no ligado a su recurso")
	}
	var dto map[string]json.RawMessage
	if err := json.Unmarshal(propio.Canonico(), &dto); err != nil || len(dto) != 17 ||
		string(dto["esquema"]) != `"vec.personal.asignacion-dietas-propia.v1"` ||
		string(dto["operacion"]) != `"consultar_propia"` {
		t.Fatalf("DTO propio incompatible: %s, %v", propio.Canonico(), err)
	}
	if _, err := NuevoMaterialAsignacionDietas(s); err == nil {
		t.Fatal("la consulta antigua aceptó la operación propia nueva")
	}
	s.Operacion = ConsultarAsignacionDietas
	antiguo, err := NuevoMaterialAsignacionDietas(s)
	if err != nil || antiguo.Recurso().Tipo != "asignacion_dietas" || len(antiguo.Recurso().Ambitos) != 4 ||
		len(antiguo.Recurso().Atributos) != 3 {
		t.Fatalf("se alteró la consulta de cuatro ámbitos: %v", err)
	}
	if _, err := NuevoMaterialAsignacionDietasPropia(s); err == nil {
		t.Fatal("la consulta nueva aceptó la operación antigua")
	}
	s.Operacion = ConsultarPropiaAsignacionDietas
	s.PersonaRef = "per_" + strings.Repeat("b", 24)
	if _, err := NuevoMaterialAsignacionDietasPropia(s); err == nil {
		t.Fatal("consulta propia de persona ajena admitida")
	}
}

func actorAsignacionPrueba(t *testing.T) vecdomain.ContextoActor {
	t.Helper()
	z := strings.Repeat("a", 24)
	ahora := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	instantanea := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + z, PersonaVersion: 1, PerfilActivoRef: "prf_" + z, PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), Vinculos: []vecdomain.VinculoReferenciaContextoActor{{VinculoRef: "vin_" + z, Version: 1, Tipo: vecdomain.TipoReferenciaContextoActorEmpleado, Referencia: "emp_" + z, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}}}
	actor, err := vecdomain.NuevoContextoActor(cuenta, instantanea, ahora)
	if err != nil {
		t.Fatal(err)
	}
	return actor
}

func TestAsignacionDietasSeparaSujetoYActorSoloEnGrupo(t *testing.T) {
	actor := actorAsignacionPrueba(t)
	fecha, _ := NuevaFechaCivil("2026-09-20")
	base := SolicitudAsignacionDietas{Actor: actor, RelacionRef: "rel_" + strings.Repeat("a", 24), UnidadRef: "unidad_sintetica", FechaReferencia: fecha, Operacion: ConsultarAsignacionDietas}
	propia, err := NuevoMaterialAsignacionDietas(base)
	if err != nil || propia.Solicitud().PersonaRef != actor.PersonaRef || propia.Solicitud().EmpleadoRef != "emp_"+strings.Repeat("a", 24) {
		t.Fatalf("consulta propia no normalizada: %v", err)
	}
	base.PersonaRef = "per_" + strings.Repeat("b", 24)
	if _, err = NuevoMaterialAsignacionDietas(base); err == nil {
		t.Fatal("consulta propia aceptó persona ajena")
	}
	base.Operacion = CorregirGrupoAsignacionDietas
	base.EmpleadoRef = "emp_" + strings.Repeat("b", 24)
	base.ClaveIdempotencia = "12345678-1234-4123-8123-123456789abc"
	base.VersionEsperada = 1
	base.CentroRef = "centro_sintetico"
	base.AdministrativoPersonaRef = "per_" + strings.Repeat("c", 24)
	base.ResponsablePersonaRef = "per_" + strings.Repeat("d", 24)
	base.GrupoDieta = 2
	base.VigenteDesde = fecha
	base.MotivoRevision = "correccion sintética"
	base.ProcedenciaActoRef = "acto_sintetico"
	grupo, err := NuevoMaterialAsignacionDietas(base)
	if err != nil || grupo.Recurso().Ambitos["persona_ref"] != base.PersonaRef || grupo.Recurso().Ambitos["empleado_ref"] != base.EmpleadoRef {
		t.Fatalf("sujeto grupo no ligado: %v", err)
	}
	base.Operacion = CorregirAsignacionDietas
	if correccion, e := NuevoMaterialAsignacionDietas(base); e != nil || correccion.Recurso().Ambitos["persona_ref"] != base.PersonaRef {
		t.Fatalf("corrección administrativa no ligó sujeto: %v", e)
	}
	base.PersonaRef = actor.PersonaRef
	if _, e := NuevoMaterialAsignacionDietas(base); e == nil {
		t.Fatal("corrección ordinaria aceptó autoadscripción del empleado")
	}
	base.PersonaRef = "per_" + strings.Repeat("b", 24)
	base.Operacion = RegistrarInicialAsignacionDietas
	base.VersionEsperada = 0
	if inicial, e := NuevoMaterialAsignacionDietas(base); e != nil || inicial.Recurso().Ambitos["persona_ref"] != base.PersonaRef {
		t.Fatalf("alta inicial gobernada rechazada: %v", e)
	}
	base.PersonaRef = actor.PersonaRef
	if _, e := NuevoMaterialAsignacionDietas(base); e == nil {
		t.Fatal("alta inicial aceptó actor empleado como administrador")
	}
}
