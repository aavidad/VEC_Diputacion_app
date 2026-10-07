package domain

import (
	"math"
	"strings"
	"testing"
)

func conjuntoAdministracionAccesoPrueba() PreimagenConjuntoCuentasAdministracionAcceso {
	return PreimagenConjuntoCuentasAdministracionAcceso{PersonaRef: "per_" + strings.Repeat("b", 24), PersonaVersion: 3, RevisionContinuidad: 4,
		Cuentas: []PreimagenCuentaAdministracionAcceso{
			{CuentaRef: "cta_" + strings.Repeat("a", 24), Revision: 2, Estado: EstadoCuentaAccesoActiva},
			{CuentaRef: "cta_" + strings.Repeat("b", 24), Revision: 9, Estado: EstadoCuentaAccesoInactiva},
		}}
}

func materialAdministracionAccesoPrueba() MaterialPropuestaAdministracionAcceso {
	return MaterialPropuestaAdministracionAcceso{OperacionRef: "propuesta_admin:" + strings.Repeat("a", 32),
		ProponentePersonaRef: "per_" + strings.Repeat("c", 24), PerfilActivoRef: "prf_" + strings.Repeat("d", 24),
		AsignacionPerfilRef: "asignacion:sintetica:v1",
		Conjunto:            conjuntoAdministracionAccesoPrueba(), EstadoNuevo: EstadoCuentaAccesoInactiva,
		Motivo: ReferenciaEntradaCatalogo{CatalogoID: "motivos_administracion", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("e", 64), EntradaClave: "motivo_" + strings.Repeat("f", 32)}}
}

func TestAdministracionAccesoHuellaLigaComposicionYRevisiones(t *testing.T) {
	original := conjuntoAdministracionAccesoPrueba()
	h, err := original.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []string{"alta", "revision", "estado", "persona", "continuidad"} {
		t.Run(caso, func(t *testing.T) {
			c := conjuntoAdministracionAccesoPrueba()
			switch caso {
			case "alta":
				c.Cuentas = append(c.Cuentas, PreimagenCuentaAdministracionAcceso{CuentaRef: "cta_" + strings.Repeat("c", 24), Revision: 1, Estado: EstadoCuentaAccesoActiva})
			case "revision":
				c.Cuentas[0].Revision++
			case "estado":
				c.Cuentas[0].Estado = EstadoCuentaAccesoInactiva
			case "persona":
				c.PersonaVersion++
			case "continuidad":
				c.RevisionContinuidad++
			}
			hc, err := c.HuellaSHA256()
			if err != nil || hc == h {
				t.Fatal("conjunto alterado no cambia huella")
			}
		})
	}
}

func TestAdministracionAccesoConjuntoRechazaAmbiguedad(t *testing.T) {
	for _, caso := range []string{"vacio", "duplicado", "desordenado", "revision_cero", "estado_desconocido", "comodin", "continuidad_cero"} {
		t.Run(caso, func(t *testing.T) {
			c := conjuntoAdministracionAccesoPrueba()
			switch caso {
			case "vacio":
				c.Cuentas = nil
			case "duplicado":
				c.Cuentas[1] = c.Cuentas[0]
			case "desordenado":
				c.Cuentas[0], c.Cuentas[1] = c.Cuentas[1], c.Cuentas[0]
			case "revision_cero":
				c.Cuentas[0].Revision = 0
			case "estado_desconocido":
				c.Cuentas[0].Estado = "bloqueada"
			case "comodin":
				c.Cuentas[0].CuentaRef = "cta_*"
			case "continuidad_cero":
				c.RevisionContinuidad = 0
			}
			if _, err := c.HuellaSHA256(); err == nil {
				t.Fatal("conjunto ambiguo aceptado")
			}
		})
	}
}

func TestAdministracionAccesoMaterialExigeCambioYRevisionDisponible(t *testing.T) {
	m := materialAdministracionAccesoPrueba()
	if m.Validar() != nil {
		t.Fatal("bloqueo válido rechazado")
	}
	m.EstadoNuevo = EstadoCuentaAccesoActiva
	if m.Validar() != nil {
		t.Fatal("reactivación con cuenta inactiva rechazada")
	}
	m.Conjunto.Cuentas[1].Estado = EstadoCuentaAccesoActiva
	if m.Validar() == nil {
		t.Fatal("otra alta sin cambio de revisión aceptada")
	}
	m.Conjunto.Cuentas[1].Estado, m.Conjunto.Cuentas[1].Revision = EstadoCuentaAccesoInactiva, math.MaxUint64
	if m.Validar() == nil {
		t.Fatal("desbordamiento de revisión aceptado")
	}
	m = materialAdministracionAccesoPrueba()
	m.ProponentePersonaRef = m.Conjunto.PersonaRef
	if m.Validar() == nil {
		t.Fatal("auto-bloqueo aceptado")
	}
}

func TestAdministracionAccesoHuellaIncluyeAsignacionNominal(t *testing.T) {
	m := materialAdministracionAccesoPrueba()
	h, err := m.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	m.AsignacionPerfilRef = "asignacion:otra:v1"
	ho, err := m.HuellaSHA256()
	if err != nil || ho == h {
		t.Fatal("la propuesta no liga su asignación nominal")
	}
}
