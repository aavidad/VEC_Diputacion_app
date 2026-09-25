package domain

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestAlcanceProyeccionesContextoActorEsCerrado(t *testing.T) {
	vacio, err := NuevoAlcanceProyeccionesContextoActor()
	if err != nil || !vacio.Vacio() || vacio.IncluyeEmpleado() || vacio != (AlcanceProyeccionesContextoActor{}) ||
		vacio.Proyecciones() == nil || len(vacio.Proyecciones()) != 0 {
		t.Fatal("el alcance vacio no es el valor cero")
	}
	empleado, err := NuevoAlcanceProyeccionesContextoActor(ProyeccionContextoActorEmpleado)
	if err != nil || empleado.Vacio() || !empleado.IncluyeEmpleado() ||
		len(empleado.Proyecciones()) != 1 || empleado.Proyecciones()[0] != ProyeccionContextoActorEmpleado {
		t.Fatal("el alcance empleado no se construyo")
	}
	for _, invalido := range [][]ProyeccionContextoActor{
		{"candidato"}, {""}, {"EMPLEADO"},
		{ProyeccionContextoActorEmpleado, ProyeccionContextoActorEmpleado},
	} {
		if alcance, err := NuevoAlcanceProyeccionesContextoActor(invalido...); !errors.Is(err, ErrAlcanceProyeccionesContextoActorInvalido) || !alcance.Vacio() {
			t.Fatalf("alcance abierto aceptado: %#v", invalido)
		}
	}
}

func TestContextoActorDerivaAlcanceDeLaProyeccionDePersonal(t *testing.T) {
	instante := instanteContextoActorPrueba()
	heredada := instantaneaContextoActorPrueba(instante)
	contexto, err := NuevoContextoActor(solicitudContextoActorPrueba().Cuenta, heredada, instante)
	if err != nil || !contexto.AlcanceProyecciones().Vacio() {
		t.Fatal("un puntero vin_ de empleado heredado no es alcance empleado")
	}
	proyectada := instantaneaContextoActorPrueba(instante)
	proyectada.Vinculos[1].VinculoRef = referenciaContextoActorPrueba("pep_", "e")
	contexto, err = NuevoContextoActor(solicitudContextoActorPrueba().Cuenta, proyectada, instante)
	if err != nil || !contexto.AlcanceProyecciones().IncluyeEmpleado() {
		t.Fatalf("la proyeccion pep_ de Personal no se reconocio: %v", err)
	}
	empleados, err := contexto.Referencias(TipoReferenciaContextoActorEmpleado)
	if err != nil || len(empleados) != 1 || empleados[0] != proyectada.Vinculos[1].Referencia {
		t.Fatal("el empleado proyectado no se entrega como referencia unica")
	}

	// pep_ solo identifica empleados de Personal, nunca candidatos.
	candidato := instantaneaContextoActorPrueba(instante)
	candidato.Vinculos[0].VinculoRef = referenciaContextoActorPrueba("pep_", "c")
	if _, err := NuevoContextoActor(solicitudContextoActorPrueba().Cuenta, candidato, instante); !errors.Is(err, ErrContextoActorInvalido) {
		t.Fatal("un candidato con identidad pep_ fue aceptado")
	}
	// Nunca dos empleados: heredado y proyectado a la vez es ambiguo.
	doble := instantaneaContextoActorPrueba(instante)
	extra := doble.Vinculos[1]
	extra.VinculoRef = referenciaContextoActorPrueba("pep_", "x")
	extra.Referencia = referenciaContextoActorPrueba("emp_", "x")
	doble.Vinculos = append(doble.Vinculos, extra)
	if _, err := NuevoContextoActor(solicitudContextoActorPrueba().Cuenta, doble, instante); !errors.Is(err, ErrContextoActorInvalido) {
		t.Fatal("dos empleados en el mismo contexto")
	}
}

func TestCanonV2ConProyeccionEmpleadoConservaClavesYRehidrata(t *testing.T) {
	instante := instanteContextoActorPrueba()
	proyectada := instantaneaContextoActorPrueba(instante)
	proyectada.CuentaVersion = 2
	proyectada.Vinculos[1].VinculoRef = referenciaContextoActorPrueba("pep_", "e")
	contexto, err := NuevoContextoActor(solicitudContextoActorPrueba().Cuenta, proyectada, instante)
	if err != nil {
		t.Fatal(err)
	}
	canon, err := contexto.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	// Mismas claves que 000006: los consumidores SQL exigen el conjunto exacto.
	if bytes.Contains(canon, []byte("proyecciones")) || bytes.Contains(canon, []byte("alcance")) ||
		!bytes.Contains(canon, []byte(`"vinculo_ref":"`+proyectada.Vinculos[1].VinculoRef+`"`)) {
		t.Fatalf("canon con claves nuevas o sin la entrada de Personal: %s", canon)
	}
	rehidratado, err := RehidratarContextoActorVinculadoV2(canon)
	if err != nil || !rehidratado.AlcanceProyecciones().IncluyeEmpleado() {
		t.Fatalf("el alcance no se reconstruye desde el canon: %v", err)
	}
	// Sustituir pep_ por vin_ cambia alcance y bytes: no hay equivalencia.
	alterado := bytes.Replace(canon, []byte(`"pep_`), []byte(`"vin_`), 1)
	heredado, err := RehidratarContextoActorVinculadoV2(alterado)
	if err != nil || !heredado.AlcanceProyecciones().Vacio() || bytes.Equal(alterado, canon) {
		t.Fatal("el alcance no queda ligado a los bytes del canon")
	}
}

func TestManifiestoAdmitePepSoloParaEmpleado(t *testing.T) {
	manifiesto := manifiestoProcedenciaContextoActorPrueba(1)
	empleado := manifiesto.Vinculos[0]
	empleado.VinculoRef = "pep_" + strings.Repeat("h", 24)
	empleado.Tipo = TipoReferenciaContextoActorEmpleado
	empleado.Referencia = "emp_" + strings.Repeat("i", 24)
	manifiesto.Vinculos = append(manifiesto.Vinculos, empleado)
	if err := manifiesto.Validar(); err != nil {
		t.Fatalf("manifiesto con procedencia de Personal rechazado: %v", err)
	}
	manifiesto.Vinculos[0].VinculoRef = "pep_" + strings.Repeat("e", 24)
	if err := manifiesto.Validar(); !errors.Is(err, ErrManifiestoProcedenciaContextoActorV1Invalido) {
		t.Fatal("candidato pep_ aceptado en el manifiesto")
	}
}
