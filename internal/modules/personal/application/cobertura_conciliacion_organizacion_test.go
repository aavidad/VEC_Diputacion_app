package application

import (
	"reflect"
	"slices"
	"testing"

	"vec-diputacion-granada/internal/modules/personal/domain"
)

func paqueteCoberturaPrueba(tipo string) PaquetePreparacionOrganizacion {
	p := paqueteRevisionPrueba()
	h := p.Hechos[0]
	if tipo == "plantilla" {
		p.Manifiesto.Tipo = tipo
		h = domain.HechoImportacionOrganizacion{Clase: "plaza", HechoRef: h.HechoRef, Revision: 1, FilaFuenteRef: h.FilaFuenteRef,
			OrganismoRef: h.OrganismoRef, UnidadRef: h.UnidadRef, VigenteDesde: h.VigenteDesde,
			CodigoFuente: "PL-1", ClasificacionRef: "categoria:sintetica", EstadoEstructural: "vigente", DotacionPresupuestaria: "desconocida"}
		p.Hechos = []domain.HechoImportacionOrganizacion{h}
		h.HechoRef, h.FilaFuenteRef, h.CodigoFuente = "33333333-3333-4333-8333-333333333333", "fila:2", "PL-2"
	} else {
		// Un puesto tipo y una unidad comparten origen, pero necesitan decisiones
		// distintas. Una clasificación adicional tampoco sustituye ninguna.
		h.Clase, h.HechoRef = "puesto_tipo", "33333333-3333-4333-8333-333333333333"
		h.CatalogoEntradaClave, h.TipoUnidad = "", ""
		h.CodigoFuente, h.ClasificacionRef = "PT-1", "categoria:sintetica"
	}
	p.Hechos = append(p.Hechos, h)
	return p
}

func TestCoberturaConciliacionSituaHuecosEnPlantillaYRPT(t *testing.T) {
	for _, tipo := range []string{"rpt", "plantilla"} {
		t.Run(tipo, func(t *testing.T) {
			p := paqueteCoberturaPrueba(tipo)
			h := p.Hechos[1]
			p.Decisiones = []domain.DecisionConciliacionOrganizacion{{Clase: h.Clase, FilaFuenteRef: h.FilaFuenteRef,
				Resultado: "vinculada", DestinoRef: "destino:sintetico", EvidenciaRef: "evidencia:sintetica", Motivo: "Correspondencia declarada"}}
			r := RevisarPreparacionOrganizacion(p)
			c := r.CoberturaConciliacion
			if !r.Valido || c == nil || c.Completa || len(c.Hechos) != 2 || c.RecuentosHechos["sin_decision"] != 1 || c.RecuentosHechos["vinculada"] != 1 {
				t.Fatalf("hueco sin identificar: %+v", c)
			}
			faltante, cubierta := c.Hechos[0], c.Hechos[1]
			if faltante.HechoRef != p.Hechos[0].HechoRef || faltante.FilaFuenteRef != "fila:1" || faltante.Resultado != "sin_decision" ||
				cubierta.HechoRef != h.HechoRef || cubierta.Resultado != "vinculada" || cubierta.ClaseDecision != h.Clase {
				t.Fatalf("clases o filas confundidas: %+v", c.Hechos)
			}
			if tipo == "rpt" && faltante.ClaseDecision != "unidad" {
				t.Fatal("nodo sin decisión de unidad")
			}
		})
	}
}

func TestCoberturaConciliacionConservaResultadosYClasificacionAdicional(t *testing.T) {
	for _, caso := range []struct {
		nombre, unidad, clasificacion string
		completa                      bool
	}{
		{"sin decisiones", "", "", false},
		{"pendiente", "pendiente", "", false},
		{"descartada", "descartada", "", false},
		{"vinculada", "vinculada", "", true},
		{"clasificación no cubre unidad", "", "vinculada", false},
		{"clasificación pendiente conserva hueco", "vinculada", "pendiente", false},
		{"clasificación descartada conserva hueco", "vinculada", "descartada", false},
		{"clasificación adicional vinculada", "vinculada", "vinculada", true},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			p := paqueteRevisionPrueba()
			for _, d := range []struct{ clase, resultado string }{{"unidad", caso.unidad}, {"clasificacion", caso.clasificacion}} {
				if d.resultado == "" {
					continue
				}
				decision := domain.DecisionConciliacionOrganizacion{FilaFuenteRef: "fila:1", Clase: d.clase, Resultado: d.resultado,
					Motivo: "Correspondencia declarada", EvidenciaRef: "evidencia:sintetica"}
				if d.resultado == "vinculada" {
					decision.DestinoRef = "destino:sintetico"
				}
				p.Decisiones = append(p.Decisiones, decision)
			}
			r := RevisarPreparacionOrganizacion(p)
			c := r.CoberturaConciliacion
			resultado := caso.unidad
			if resultado == "" {
				resultado = "sin_decision"
			}
			if !r.Valido || c == nil || c.Completa != caso.completa || len(c.Hechos) != 1 || c.Hechos[0].Resultado != resultado || c.RecuentosHechos[resultado] != 1 {
				t.Fatalf("cobertura: %+v", c)
			}
			if caso.clasificacion != "" && (len(c.DecisionesAdicionales) != 1 || c.DecisionesAdicionales[0].Resultado != caso.clasificacion || c.DecisionesAdicionales[0].Clase != "clasificacion") {
				t.Fatalf("clasificación adicional: %+v", c.DecisionesAdicionales)
			}
			if caso.clasificacion == "" && len(c.DecisionesAdicionales) != 0 {
				t.Fatal("decisión adicional inventada")
			}
			if r.Estado != "preparacion_no_autoritativa" || !slices.Contains(r.PendientesPublicacion, "conciliacion_autorizada") ||
				!slices.Contains(r.PendientesPublicacion, "acreditacion_fuente") || !slices.Contains(r.PendientesPublicacion, "publicacion_autorizada") {
				t.Fatal("cobertura elevada a efecto administrativo")
			}
		})
	}
}

func TestCoberturaConciliacionNoAfirmaCoberturaDePaqueteInvalido(t *testing.T) {
	p := paqueteRevisionPrueba()
	p.Decisiones = decisionRevisionPrueba()
	p.Decisiones[0].Clase = "plaza"
	r := RevisarPreparacionOrganizacion(p)
	if r.Valido || r.ClaveError != "decision_invalida" || r.CoberturaConciliacion != nil {
		t.Fatal("decisión de otra clase produjo cobertura")
	}
}

func TestCoberturaConciliacionOrdenYHuellaEstables(t *testing.T) {
	p := paqueteCoberturaPrueba("rpt")
	p.Decisiones = decisionRevisionPrueba()
	p.Decisiones = append(p.Decisiones, domain.DecisionConciliacionOrganizacion{FilaFuenteRef: "fila:1", Clase: "puesto_tipo", Resultado: "descartada",
		Motivo: "Correspondencia sin resolver", EvidenciaRef: "evidencia:sintetica"})
	r := RevisarPreparacionOrganizacion(p)
	slices.Reverse(p.Hechos)
	slices.Reverse(p.Decisiones)
	ro := RevisarPreparacionOrganizacion(p)
	if !r.Valido || !ro.Valido || r.PaqueteHuellaSHA256 != ro.PaqueteHuellaSHA256 || !reflect.DeepEqual(r.CoberturaConciliacion, ro.CoberturaConciliacion) {
		t.Fatal("reordenar cambió cobertura o huella")
	}
	if p.Hechos[0].Clase != "puesto_tipo" || p.Decisiones[0].Clase != "puesto_tipo" {
		t.Fatal("revisión modificó el paquete")
	}
	r.CoberturaConciliacion.Hechos[0].Resultado = "vinculada"
	r.CoberturaConciliacion.RecuentosHechos["vinculada"] = 99
	if despues := RevisarPreparacionOrganizacion(p); !reflect.DeepEqual(despues.CoberturaConciliacion, ro.CoberturaConciliacion) {
		t.Fatal("salidas compartidas entre revisiones")
	}
}
