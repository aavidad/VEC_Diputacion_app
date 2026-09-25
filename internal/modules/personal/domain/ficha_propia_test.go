package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	core "vec-diputacion-granada/internal/vec/domain"
)

func actorFichaPropiaPrueba(t *testing.T, vinculos ...core.VinculoReferenciaContextoActor) core.ContextoActor {
	t.Helper()
	z := strings.Repeat("a", 24)
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	cuenta := core.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceHigh}
	instantanea := core.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 3, CuentaRef: cuenta.CuentaRef, CuentaVersion: 2, PersonaRef: "per_" + z, PersonaVersion: 4, PerfilActivoRef: "prf_" + z, PerfilVersion: 5, Estado: core.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), Vinculos: vinculos}
	actor, err := core.NuevoContextoActor(cuenta, instantanea, ahora)
	if err != nil {
		t.Fatal(err)
	}
	return actor
}

func vinculoEmpleadoPrueba(prefijo, empleado string) core.VinculoReferenciaContextoActor {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	return core.VinculoReferenciaContextoActor{VinculoRef: prefijo + strings.Repeat("e", 24), Version: 1, Tipo: core.TipoReferenciaContextoActorEmpleado, Referencia: empleado, Estado: core.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}
}

func corteFichaPropiaPrueba() CorteEmpleadoB2 {
	return CorteEmpleadoB2{VigenteEn: "2026-09-25", ConocidoEn: time.Date(2026, 9, 25, 9, 59, 59, 123456000, time.UTC)}
}

func TestMaterialFichaPropiaCanonicoYRecurso(t *testing.T) {
	empleado := "emp_" + strings.Repeat("b", 24)
	actor := actorFichaPropiaPrueba(t, vinculoEmpleadoPrueba("pep_", empleado))
	m, err := NuevoMaterialFichaPropia(SolicitudFichaPropia{Corte: corteFichaPropiaPrueba(), Actor: actor})
	if err != nil {
		t.Fatal(err)
	}
	z := strings.Repeat("a", 24)
	esperado := `{"esquema":"vec.personal.ficha-propia.consulta.v1","empleado_ref":"` + empleado + `","vigente_en":"2026-09-25","conocido_en":"2026-09-25T09:59:59.123456Z","actor_ref":"per_` + z + `","contexto_actor_ref":"vca_` + z + `","contexto_version":3,"cuenta_ref":"cta_` + z + `","cuenta_version":2,"perfil_ref":"prf_` + z + `","perfil_version":5,"persona_ref":"per_` + z + `","persona_version":4}`
	if string(m.Canonico()) != esperado {
		t.Fatalf("material no canónico:\n%s\n%s", m.Canonico(), esperado)
	}
	suma := sha256.Sum256([]byte(esperado))
	r := m.Recurso()
	if r.Referencia != empleado || r.Tipo != TipoRecursoFichaPropia || r.ModuloID != "personal" || len(r.Ambitos) != 1 || r.Ambitos["empleado_ref"] != empleado ||
		r.Atributos["material_sha256"] != hex.EncodeToString(suma[:]) || r.Atributos["operacion"] != "ficha_propia" || m.EmpleadoRef() != empleado {
		t.Fatalf("recurso inesperado: %+v", r)
	}
	// El SQL reconstruye la huella con este mismo canon de contexto.
	contexto := `{"ambitos":{"empleado_ref":"` + empleado + `"},"atributos":{"conocido_en":"2026-09-25T09:59:59.123456Z","material_sha256":"` + hex.EncodeToString(suma[:]) + `","operacion":"ficha_propia","vigente_en":"2026-09-25"}}`
	h := sha256.Sum256([]byte(contexto))
	if got, err := m.HuellaSHA256(); err != nil || got != hex.EncodeToString(h[:]) {
		t.Fatalf("huella de contexto divergente del SQL: %s", got)
	}
	// Las copias no comparten mapas con el material.
	r.Ambitos["empleado_ref"] = "otro"
	if m.Recurso().Ambitos["empleado_ref"] != empleado {
		t.Fatal("el recurso comparte mapas")
	}
}

func TestMaterialFichaPropiaExigeEmpleadoProyectadoUnico(t *testing.T) {
	casos := map[string]struct {
		vinculos []core.VinculoReferenciaContextoActor
		err      error
	}{
		"sin_empleado":     {nil, ErrFichaPropiaSinEmpleado},
		"puntero_heredado": {[]core.VinculoReferenciaContextoActor{vinculoEmpleadoPrueba("vin_", "emp_"+strings.Repeat("b", 24))}, ErrFichaPropiaSinEmpleado},
	}
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			_, err := NuevoMaterialFichaPropia(SolicitudFichaPropia{Corte: corteFichaPropiaPrueba(), Actor: actorFichaPropiaPrueba(t, caso.vinculos...)})
			if !errors.Is(err, caso.err) {
				t.Fatalf("error %v, se esperaba %v", err, caso.err)
			}
		})
	}
	actor := actorFichaPropiaPrueba(t, vinculoEmpleadoPrueba("pep_", "emp_"+strings.Repeat("b", 24)))
	if _, err := NuevoMaterialFichaPropia(SolicitudFichaPropia{Corte: CorteEmpleadoB2{VigenteEn: "2026-09-25"}, Actor: actor}); !errors.Is(err, ErrFichaPropiaInvalida) {
		t.Fatal("corte sin instante admitido", err)
	}
	if _, err := NuevoMaterialFichaPropia(SolicitudFichaPropia{Corte: corteFichaPropiaPrueba()}); !errors.Is(err, ErrFichaPropiaInvalida) {
		t.Fatal("actor vacío admitido", err)
	}
}

func fichaPropiaValidaPrueba() FichaPropia {
	return FichaPropia{
		Corte: corteFichaPropiaPrueba(),
		Relaciones: []RelacionFichaPropia{
			{Inicio: "2026-01-01", Fin: "", Estado: "vigente", Regimen: "Funcionario interino", Modalidad: "Vacante", Unidad: "Servicio de Personal", Puesto: "Técnico/a", Situacion: "Servicio activo"},
			{Inicio: "2020-01-01", Fin: "2020-12-31", Estado: "finalizada"},
		},
		Servicios: []ServicioFichaPropia{{Inicio: "2019-01-01", Fin: "2019-12-31", Clase: "Servicios previos", Dias: 365, Estado: "reconocido"}},
	}
}

func TestFichaPropiaValidaParaSuMaterial(t *testing.T) {
	actor := actorFichaPropiaPrueba(t, vinculoEmpleadoPrueba("pep_", "emp_"+strings.Repeat("b", 24)))
	m, err := NuevoMaterialFichaPropia(SolicitudFichaPropia{Corte: corteFichaPropiaPrueba(), Actor: actor})
	if err != nil {
		t.Fatal(err)
	}
	if err := fichaPropiaValidaPrueba().ValidarPara(m); err != nil {
		t.Fatal(err)
	}
	casos := map[string]func(*FichaPropia){
		"corte_ajeno":      func(f *FichaPropia) { f.Corte.VigenteEn = "2026-09-24" },
		"relaciones_nulas": func(f *FichaPropia) { f.Relaciones = nil },
		"estado_libre":     func(f *FichaPropia) { f.Relaciones[0].Estado = "activa" },
		"fin_antes":        func(f *FichaPropia) { f.Relaciones[1].Fin = "2019-12-31" },
		"control":          func(f *FichaPropia) { f.Relaciones[0].Unidad = "a\nb" },
		"texto_largo":      func(f *FichaPropia) { f.Relaciones[0].Puesto = strings.Repeat("x", 301) },
		"servicio_sin_fin": func(f *FichaPropia) { f.Servicios[0].Fin = "" },
		"dias_negativos":   func(f *FichaPropia) { f.Servicios[0].Dias = -1 },
		"estado_servicio":  func(f *FichaPropia) { f.Servicios[0].Estado = "pendiente" },
		"demasiadas_filas": func(f *FichaPropia) { f.Servicios = make([]ServicioFichaPropia, LimiteFilasFichaPropia+1) },
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			f := fichaPropiaValidaPrueba()
			mutar(&f)
			if !errors.Is(f.ValidarPara(m), ErrFichaPropiaInvalida) {
				t.Fatal("ficha inválida admitida")
			}
		})
	}
}
