package composicion

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type relojSubidaOriginalC8 struct{ ahora time.Time }

func (r relojSubidaOriginalC8) AhoraUTC() time.Time { return r.ahora }

type capturaSubidaOriginalC8 struct{ identidad IdentidadRegistradaCronos }

func (c capturaSubidaOriginalC8) CapturarPreparacionSubidaOriginal(_ context.Context,
	actor vecdomain.ResultadoContextoActorRegistradoV2, vinculo vecdomain.VinculoAutenticacionActorV2,
	selector ports.SelectorSubidaOriginalC8) (ports.CapturaLecturaOriginalC8, error) {
	v, _ := json.Marshal(vinculo)
	orden, _ := json.Marshal(map[string]any{
		"accion": "cronos.justificacion.subida.preparar", "recurso_ref": selector.SolicitudRef,
		"registro_contexto_ref": actor.RegistroContextoRef, "version_esperada": selector.VersionEsperada,
	})
	return ports.CapturaLecturaOriginalC8{ContextoCanonico: actor.RepresentacionCanonica,
		VinculoCanonico: v, Orden: orden}, nil
}

type fuenteSubidaOriginalC8 struct {
	estado ports.EstadoPropietarioSubidaOriginalC8
}

func (f fuenteSubidaOriginalC8) ConsultarActoSubidaOriginal(context.Context,
	ports.SelectorSubidaOriginalC8, ports.CapturaLecturaOriginalC8) (ports.EstadoPropietarioSubidaOriginalC8, error) {
	return f.estado, nil
}

func TestSelectorSubidaOriginalC8FijaSolicitudYVersion(t *testing.T) {
	const empleado = "emp_0123456789abcdefghijkl"
	id := identidadPrueba(t, empleado)
	selector := ports.SelectorSubidaOriginalC8{SolicitudRef: "permiso:cronos:solicitud:perm-0001", VersionEsperada: 3}
	estado := ports.EstadoPropietarioSubidaOriginalC8{SolicitudRef: selector.SolicitudRef,
		Version: 3, PersonaRef: id.Contexto.Contexto.PersonaRef, EmpleadoRef: empleado,
		Auditoria: ports.AuditoriaLecturaOriginalC8{Referencia: "auditoria:cronos:uno", HuellaSHA256: "sha", CorrelacionRef: "corr", RegistradaEn: "2026-10-14T20:00:00Z"}}
	r, err := application.NuevoResolutorActoSubidaOriginalC8(selector,
		capturaSubidaOriginalC8{id}, fuenteSubidaOriginalC8{estado}, relojSubidaOriginalC8{id.Contexto.ResueltoEnAutoritativo})
	if err != nil {
		t.Fatal(err)
	}
	acto, err := r.ResolverActoSubidaOriginal(context.Background(), id.Contexto, id.Vinculo, "clave-distinta")
	if err != nil || acto.Recurso.Referencia != selector.SolicitudRef || acto.VersionDueno != 3 ||
		acto.PersonaRef != estado.PersonaRef {
		t.Fatalf("acto incorrecto: %#v, %v", acto, err)
	}
	estado.EmpleadoRef = "emp_otro0123456789abcdefghij"
	r, err = application.NuevoResolutorActoSubidaOriginalC8(selector,
		capturaSubidaOriginalC8{id}, fuenteSubidaOriginalC8{estado}, relojSubidaOriginalC8{id.Contexto.ResueltoEnAutoritativo})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.ResolverActoSubidaOriginal(context.Background(), id.Contexto, id.Vinculo, "clave-distinta"); err == nil {
		t.Fatal("aceptó empleado ajeno")
	}
}
