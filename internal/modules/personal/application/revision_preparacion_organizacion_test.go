package application

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/personal/domain"
)

func paqueteRevisionPrueba() PaquetePreparacionOrganizacion {
	hash := strings.Repeat("a", 64)
	m := domain.ManifiestoImportacionOrganizacion{OrganismoRef: "organismo:sintetico", Tipo: "rpt",
		VersionRef: "11111111-1111-4111-8111-111111111111", VersionRevision: 1, FuenteRef: "fuente:sintetica", FuenteVersion: "v1", FuenteHuellaSHA256: hash,
		CatalogoUnidades:        domain.ReferenciaCatalogoImportacion{ID: "unidades:sinteticas", Version: 1, Revision: 1, HuellaSHA256: hash},
		CatalogoClasificaciones: domain.ReferenciaCatalogoImportacion{ID: "clases:sinteticas", Version: 1, Revision: 1, HuellaSHA256: hash}}
	h := domain.HechoImportacionOrganizacion{Clase: "nodo", HechoRef: "22222222-2222-4222-8222-222222222222", Revision: 1, FilaFuenteRef: "fila:1",
		OrganismoRef: m.OrganismoRef, UnidadRef: "centro:sintetico", VigenteDesde: "2026-01-01", Denominacion: "Centro de Servicios Municipales", CatalogoEntradaClave: "centro:sintetico", TipoUnidad: "centro"}
	return PaquetePreparacionOrganizacion{Manifiesto: m, Hechos: []domain.HechoImportacionOrganizacion{h}}
}

func TestRevisionPreparacionNoElevaDeclaraciones(t *testing.T) {
	p := paqueteRevisionPrueba()
	p.Manifiesto.DocumentoRef, p.Manifiesto.CustodiaRef = "documento:sintetico", "custodia:sintetica"
	p.Manifiesto.DiccionarioRef, p.Manifiesto.ActoRef = "diccionario:sintetico", "acto:sintetico"
	p.Manifiesto.AprobadaEn, p.Manifiesto.PublicadaEn, p.Manifiesto.EfectosDesde = "2026-01-01", "2026-01-02", "2026-01-03"
	if p.Manifiesto.Validar(false) != nil {
		t.Fatal("fixture de manifiesto incompleto")
	}
	r := RevisarPreparacionOrganizacion(p)
	if !r.Valido || r.Estado != "preparacion_no_autoritativa" || len(r.PaqueteHuellaSHA256) != 64 || len(r.ManifiestoHuellaSHA256) != 64 {
		t.Fatalf("informe: %+v", r)
	}
	for _, clave := range []string{"verificacion_catalogos", "acreditacion_fuente", "conciliacion_autorizada", "aprobacion_separada", "publicacion_autorizada"} {
		if !slices.Contains(r.PendientesPublicacion, clave) {
			t.Fatalf("desapareció %s", clave)
		}
	}
}

func TestRevisionPreparacionFalloSituado(t *testing.T) {
	cases := []struct {
		nombre, clave, seccion string
		fila                   int
		cambiar                func(*PaquetePreparacionOrganizacion)
	}{
		{"intervalo manifiesto", "manifiesto_invalido", "manifiesto", 0, func(p *PaquetePreparacionOrganizacion) {
			p.Manifiesto.EfectosDesde = "2026-02-01"
			p.Manifiesto.EfectosHasta = "2026-02-01"
		}},
		{"hechos vacíos", "cantidad_hechos_invalida", "hechos", 0, func(p *PaquetePreparacionOrganizacion) { p.Hechos = nil }},
		{"intervalo hecho", "hecho_invalido", "hechos", 1, func(p *PaquetePreparacionOrganizacion) { p.Hechos[0].VigenteHasta = p.Hechos[0].VigenteDesde }},
		{"fecha imposible", "hecho_invalido", "hechos", 1, func(p *PaquetePreparacionOrganizacion) { p.Hechos[0].VigenteDesde = "2026-02-30" }},
		{"tipo errado", "hecho_invalido", "hechos", 1, func(p *PaquetePreparacionOrganizacion) { p.Hechos[0].Clase = "plaza" }},
		{"organismo ajeno", "hecho_invalido", "hechos", 1, func(p *PaquetePreparacionOrganizacion) { p.Hechos[0].OrganismoRef = "organismo:otro" }},
		{"identidad duplicada", "hecho_duplicado", "hechos", 2, func(p *PaquetePreparacionOrganizacion) {
			p.Hechos = append(p.Hechos, p.Hechos[0])
			p.Hechos[1].FilaFuenteRef = "fila:2"
		}},
		{"origen duplicado", "hecho_duplicado", "hechos", 2, func(p *PaquetePreparacionOrganizacion) {
			p.Hechos = append(p.Hechos, p.Hechos[0])
			p.Hechos[1].HechoRef = "33333333-3333-4333-8333-333333333333"
		}},
		{"destino inventado", "decision_invalida", "decisiones", 1, func(p *PaquetePreparacionOrganizacion) {
			p.Decisiones = decisionRevisionPrueba()
			p.Decisiones[0].DestinoRef = "destino:inventado"
		}},
		{"decisión duplicada", "decision_duplicada", "decisiones", 2, func(p *PaquetePreparacionOrganizacion) {
			p.Decisiones = decisionRevisionPrueba()
			p.Decisiones = append(p.Decisiones, p.Decisiones[0])
		}},
		{"decisión sin fila", "decision_sin_fila", "decisiones", 1, func(p *PaquetePreparacionOrganizacion) {
			p.Decisiones = decisionRevisionPrueba()
			p.Decisiones[0].FilaFuenteRef = "fila:ausente"
		}},
		{"decisión de otra clase sobre una fila existente", "decision_invalida", "decisiones", 1, func(p *PaquetePreparacionOrganizacion) {
			p.Decisiones = decisionRevisionPrueba()
			p.Decisiones[0].Clase = "plaza"
		}},
	}
	for _, c := range cases {
		t.Run(c.nombre, func(t *testing.T) {
			p := paqueteRevisionPrueba()
			c.cambiar(&p)
			r := RevisarPreparacionOrganizacion(p)
			if r.Valido || r.ClaveError != c.clave || r.Seccion != c.seccion || r.FilaFallida != c.fila || r.PaqueteHuellaSHA256 != "" || r.ManifiestoHuellaSHA256 != "" {
				t.Fatalf("informe: %+v", r)
			}
		})
	}
}

func TestRevisionPreparacionAdmiteClasesDeHechosConLaMismaFila(t *testing.T) {
	p := paqueteRevisionPrueba()
	h := p.Hechos[0]
	h.Clase = "puesto_tipo"
	h.HechoRef = "33333333-3333-4333-8333-333333333333"
	h.CatalogoEntradaClave, h.TipoUnidad = "", ""
	h.CodigoFuente, h.ClasificacionRef = "P-1", "categoria:sintetica"
	p.Hechos = append(p.Hechos, h)
	p.Decisiones = []domain.DecisionConciliacionOrganizacion{
		{FilaFuenteRef: "fila:1", Clase: "unidad", Resultado: "pendiente", Motivo: "Unidad por comprobar", EvidenciaRef: "evidencia:sintetica"},
		{FilaFuenteRef: "fila:1", Clase: "puesto_tipo", Resultado: "pendiente", Motivo: "Puesto por comprobar", EvidenciaRef: "evidencia:sintetica"},
		{FilaFuenteRef: "fila:1", Clase: "clasificacion", Resultado: "pendiente", Motivo: "Clasificación por comprobar", EvidenciaRef: "evidencia:sintetica"},
	}
	if r := RevisarPreparacionOrganizacion(p); !r.Valido {
		t.Fatalf("decisiones de la fila sellada: %+v", r)
	}
}

func decisionRevisionPrueba() []domain.DecisionConciliacionOrganizacion {
	return []domain.DecisionConciliacionOrganizacion{{FilaFuenteRef: "fila:1", Clase: "unidad", Resultado: "pendiente", Motivo: "Correspondencia pendiente de acreditar", EvidenciaRef: "evidencia:sintetica"}}
}

func TestRevisionPreparacionHuellaYEntradaConservadas(t *testing.T) {
	p := paqueteRevisionPrueba()
	p.Hechos = append(p.Hechos, p.Hechos[0])
	p.Hechos[1].HechoRef = "33333333-3333-4333-8333-333333333333"
	p.Hechos[1].FilaFuenteRef = "fila:2"
	p.Decisiones = decisionRevisionPrueba()
	p.Decisiones = append(p.Decisiones, p.Decisiones[0])
	p.Decisiones[1].FilaFuenteRef = "fila:2"
	hechos := slices.Clone(p.Hechos)
	decisiones := slices.Clone(p.Decisiones)
	r := RevisarPreparacionOrganizacion(p)
	if !reflect.DeepEqual(p.Hechos, hechos) || !reflect.DeepEqual(p.Decisiones, decisiones) {
		t.Fatal("entrada modificada")
	}
	slices.Reverse(p.Hechos)
	slices.Reverse(p.Decisiones)
	otro := RevisarPreparacionOrganizacion(p)
	if !r.Valido || !otro.Valido || r.PaqueteHuellaSHA256 != otro.PaqueteHuellaSHA256 {
		t.Fatal("orden cambió huella")
	}
	p.Hechos[0].Denominacion = "Centro de Apoyo Territorial"
	cambiado := RevisarPreparacionOrganizacion(p)
	if !cambiado.Valido || cambiado.PaqueteHuellaSHA256 == r.PaqueteHuellaSHA256 {
		t.Fatal("contenido no cambió huella")
	}
	if r.RecuentosClase["nodo"] != 2 || r.RecuentosDecision["pendiente"] != 2 || !slices.Contains(r.PendientesPublicacion, "decisiones_pendientes") {
		t.Fatalf("recuentos: %+v", r)
	}
}

func TestRevisionPreparacionLimites(t *testing.T) {
	p := paqueteRevisionPrueba()
	original := p.Hechos[0]
	for i := 1; i < 3000; i++ {
		h := original
		h.HechoRef = fmt.Sprintf("%08x-2222-4222-8222-222222222222", i)
		h.FilaFuenteRef = fmt.Sprintf("fila:%d", i+1)
		p.Hechos = append(p.Hechos, h)
	}
	if r := RevisarPreparacionOrganizacion(p); !r.Valido || r.Hechos != 3000 {
		t.Fatalf("límite hechos: %+v", r)
	}
	for i := 0; i < 1000; i++ {
		d := decisionRevisionPrueba()[0]
		d.FilaFuenteRef = p.Hechos[i].FilaFuenteRef
		p.Decisiones = append(p.Decisiones, d)
	}
	if r := RevisarPreparacionOrganizacion(p); !r.Valido || r.Decisiones != 1000 {
		t.Fatalf("límite decisiones: %+v", r)
	}
	p.Decisiones = append(p.Decisiones, p.Decisiones[0])
	if r := RevisarPreparacionOrganizacion(p); r.ClaveError != "cantidad_decisiones_invalida" {
		t.Fatalf("exceso decisiones: %+v", r)
	}
	p.Decisiones = nil
	p.Hechos = append(p.Hechos, original)
	if r := RevisarPreparacionOrganizacion(p); r.ClaveError != "cantidad_hechos_invalida" {
		t.Fatalf("exceso hechos: %+v", r)
	}
}

func TestPreparacionExportadaNoModificaNiComparteEntrada(t *testing.T) {
	p := paqueteRevisionPrueba()
	p.Hechos = append(p.Hechos, p.Hechos[0])
	p.Hechos[0].HechoRef = "ffffffff-ffff-4fff-8fff-ffffffffffff"
	p.Hechos[0].FilaFuenteRef = "fila:2"
	p.Decisiones = decisionRevisionPrueba()
	p.Decisiones = append(p.Decisiones, p.Decisiones[0])
	p.Decisiones[0].FilaFuenteRef = "fila:2"
	original, _ := json.Marshal(p)
	normalizado, informe := PrepararPaqueteOrganizacion(p)
	if !informe.Valido || normalizado.Hechos[0].FilaFuenteRef != "fila:1" || normalizado.Decisiones[0].FilaFuenteRef != "fila:1" {
		t.Fatal("normalización ausente")
	}
	material, err := json.Marshal(normalizado)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(material)
	if hex.EncodeToString(h[:]) != informe.PaqueteHuellaSHA256 {
		t.Fatal("huella divergente")
	}
	normalizado.Hechos[0].Denominacion = "Otra denominación"
	normalizado.Decisiones[0].Motivo = "Otro motivo"
	despues, _ := json.Marshal(p)
	if string(original) != string(despues) {
		t.Fatal("entrada modificada o compartida")
	}
	p.Hechos[0].VigenteDesde = "2026-02-30"
	vacio, fallo := PrepararPaqueteOrganizacion(p)
	if fallo.Valido || !reflect.DeepEqual(vacio, PaquetePreparacionOrganizacion{}) || fallo.PaqueteHuellaSHA256 != "" {
		t.Fatal("material inválido retornado")
	}
}
