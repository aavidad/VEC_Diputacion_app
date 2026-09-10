package incorporacionejercicio

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/modules/personal/adapters/fuenteejercicio"
	core "vec-diputacion-granada/internal/vec/domain"
)

func planFuenteV2Prueba() PlanPreparacionDurableV2 {
	huella := strings.Repeat("a", 64)
	return PlanPreparacionDurableV2{
		OrganizacionRef: "organizacion:ejercicio:0001", UnidadRef: "unidad:ejercicio:rrhh",
		SolicitudPersonal: ports.SolicitudAltaPersonalRPT{Esquema: ports.EsquemaAltaPersonalRPT, ContratoVersion: 1,
			SolicitudRef: "solicitud:ejercicio:0001", ExpedienteRef: "expediente:ejercicio:0001", VersionExpediente: 7,
			CapacidadRef: "capacidad:ejercicio:0001", CorrelacionRef: "correlacion:ejercicio:0001", IdempotenciaRef: "idempotencia:ejercicio:0001",
			FuenteRPT: ports.ReferenciaVersionadaPersonalRPT{Referencia: "rpt:ejercicio:0001", Version: 1, HuellaSHA256: huella}, PuestoRef: "puesto:ejercicio:0001", PlazaRef: "plaza:ejercicio:0001"},
		FuentePersonal: fuenteejercicio.TernaEsperada{Referencia: "fuente:ejercicio:0001", Version: 1, HuellaSHA256: huella},
		SeguimientoRef: "seguimiento:ejercicio:0001", RelacionRef: "relacion:ejercicio:0001",
		Definicion: dom.ReferenciaDefinicionSeguimiento{Referencia: "ref:" + huella, Version: 1, HuellaSHA256: huella}, VersionExpedienteRaiz: 7,
		Periodo:     dom.IntervaloSeguimiento{Desde: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC), Hasta: time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)},
		MotivoClave: "revision_ejercicio", Documentos: []dom.DocumentoSeguimiento{{TipoClave: "resolucion", Referencia: "documento:ejercicio:0001"}},
		MotivoV3: core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 2, CatalogoHuellaSHA256: huella, EntradaClave: "motivo_11111111111111111111111111111111"},
	}
}

func documentoPlanesV2Prueba(t *testing.T, planes ...PlanPreparacionDurableV2) []byte {
	t.Helper()
	b, err := json.Marshal(documentoPlanesPreparacionV2{Esquema: esquemaPlanesPreparacionV2, Referencia: "planes:ejercicio:0001", Version: 2, Planes: planes})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func ternaPlanesV2Prueba(b []byte) ports.ReferenciaVersionadaPersonalRPT {
	s := sha256.Sum256(b)
	return ports.ReferenciaVersionadaPersonalRPT{Referencia: "planes:ejercicio:0001", Version: 2, HuellaSHA256: hex.EncodeToString(s[:])}
}

func TestFuentePlanesPreparacionV2ResuelveCopias(t *testing.T) {
	p := planFuenteV2Prueba()
	otro := p.Copia()
	otro.OrganizacionRef = "organizacion:ejercicio:0002"
	otro.SolicitudPersonal.SolicitudRef = "solicitud:ejercicio:0002"
	b := documentoPlanesV2Prueba(t, p, otro)
	f, err := NuevaFuentePlanesPreparacionV2(b, ternaPlanesV2Prueba(b))
	if err != nil {
		t.Fatal(err)
	}
	clear(b)
	for _, esperado := range []PlanPreparacionDurableV2{p, otro} {
		obtenido, err := f.ResolverPlan(context.Background(), esperado.OrganizacionRef, esperado.SolicitudPersonal.ExpedienteRef)
		if err != nil || !reflect.DeepEqual(obtenido, esperado) {
			t.Fatalf("plan alterado: %v", err)
		}
		obtenido.Documentos[0].Referencia = "documento:alterado"
		repetido, err := f.ResolverPlan(context.Background(), esperado.OrganizacionRef, esperado.SolicitudPersonal.ExpedienteRef)
		if err != nil || !reflect.DeepEqual(repetido, esperado) {
			t.Fatalf("copia no aislada: %v", err)
		}
	}
}

func TestFuentePlanesPreparacionV2RechazaDocumento(t *testing.T) {
	p := planFuenteV2Prueba()
	b := documentoPlanesV2Prueba(t, p)
	invalido := p.Copia()
	invalido.SolicitudPersonal.IdempotenciaRef = ""
	casos := map[string][]byte{
		"vacio":               nil,
		"tamano":              []byte(strings.Repeat(" ", maximoBytesPlanesV2+1)),
		"sin_planes":          documentoPlanesV2Prueba(t),
		"duplicado_org_exp":   documentoPlanesV2Prueba(t, p, p),
		"plan_invalido":       documentoPlanesV2Prueba(t, invalido),
		"esquema":             []byte(strings.Replace(string(b), esquemaPlanesPreparacionV2, "otro", 1)),
		"referencia":          []byte(strings.Replace(string(b), "planes:ejercicio:0001", "planes:ejercicio:0002", 1)),
		"version":             []byte(strings.Replace(string(b), `"version":2`, `"version":3`, 1)),
		"desconocida":         append([]byte(`{"permiso":true,`), b[1:]...),
		"duplicada":           append([]byte(`{"version":2,`), b[1:]...),
		"duplicada_escapada":  append([]byte(`{"vers\u0069on":2,`), b[1:]...),
		"mayusculas":          []byte(strings.Replace(string(b), `"organizacion_ref"`, `"Organizacion_ref"`, 1)),
		"anidada_desconocida": []byte(strings.Replace(string(b), `"solicitud_personal":{`, `"solicitud_personal":{"permiso":true,`, 1)),
		"anidada_duplicada":   []byte(strings.Replace(string(b), `"solicitud_personal":{`, `"solicitud_personal":{"contrato_version":1,`, 1)),
		"nulo_escalar":        []byte(strings.Replace(string(b), `"version_seguimiento_esperada":0`, `"version_seguimiento_esperada":null`, 1)),
		"adicional":           append(append([]byte(nil), b...), []byte(` {}`)...),
		"truncado":            b[:len(b)-1],
		"cardinalidad":        documentoPlanesV2Prueba(t, make([]PlanPreparacionDurableV2, maximoPlanesV2+1)...),
	}
	for nombre, contenido := range casos {
		t.Run(nombre, func(t *testing.T) {
			f, err := NuevaFuentePlanesPreparacionV2(contenido, ternaPlanesV2Prueba(contenido))
			if f != nil || !errors.Is(err, ports.ErrComposicionIncorporacionAplicacion) {
				t.Fatalf("documento aceptado: %v", err)
			}
		})
	}
}

func TestFuentePlanesPreparacionV2ExigeTernaExacta(t *testing.T) {
	b := documentoPlanesV2Prueba(t, planFuenteV2Prueba())
	esperada := ternaPlanesV2Prueba(b)
	for _, mutar := range []func(*ports.ReferenciaVersionadaPersonalRPT){
		func(r *ports.ReferenciaVersionadaPersonalRPT) { r.HuellaSHA256 = strings.Repeat("b", 64) },
		func(r *ports.ReferenciaVersionadaPersonalRPT) { r.Referencia = "planes:ejercicio:otra" },
		func(r *ports.ReferenciaVersionadaPersonalRPT) { r.Version++ },
		func(r *ports.ReferenciaVersionadaPersonalRPT) { r.Version = 0 },
	} {
		r := esperada
		mutar(&r)
		if f, err := NuevaFuentePlanesPreparacionV2(b, r); err == nil || f != nil {
			t.Fatal("terna aceptada")
		}
	}
	if f, err := NuevaFuentePlanesPreparacionV2(append(b, '\n'), esperada); err == nil || f != nil {
		t.Fatal("bytes distintos aceptados")
	}
}

func TestFuentePlanesPreparacionV2ConsultaCerrada(t *testing.T) {
	p := planFuenteV2Prueba()
	b := documentoPlanesV2Prueba(t, p)
	f, err := NuevaFuentePlanesPreparacionV2(b, ternaPlanesV2Prueba(b))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	var nula *fuentePlanesPreparacionV2
	for _, caso := range []struct {
		f        *fuentePlanesPreparacionV2
		ctx      context.Context
		org, exp string
		err      error
	}{
		{f, ctx, p.OrganizacionRef, p.SolicitudPersonal.ExpedienteRef, context.Canceled},
		{f, nil, p.OrganizacionRef, p.SolicitudPersonal.ExpedienteRef, ports.ErrComposicionIncorporacionAplicacion},
		{nula, context.Background(), p.OrganizacionRef, p.SolicitudPersonal.ExpedienteRef, ports.ErrComposicionIncorporacionAplicacion},
		{f, context.Background(), "organizacion:ajena", p.SolicitudPersonal.ExpedienteRef, ports.ErrComposicionIncorporacionAplicacion},
		{f, context.Background(), p.OrganizacionRef, "expediente:ajeno", ports.ErrComposicionIncorporacionAplicacion},
		{f, context.Background(), "", p.SolicitudPersonal.ExpedienteRef, ports.ErrComposicionIncorporacionAplicacion},
	} {
		obtenido, err := caso.f.ResolverPlan(caso.ctx, caso.org, caso.exp)
		if !errors.Is(err, caso.err) || !reflect.DeepEqual(obtenido, PlanPreparacionDurableV2{}) {
			t.Fatalf("consulta no cerrada: %v", err)
		}
	}
}
