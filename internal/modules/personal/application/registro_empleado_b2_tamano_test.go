package application

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

// Historia sintética de una relación: revisiones de actos, ocupaciones,
// situaciones y reconocimientos. Las denominaciones son de tamaño ordinario;
// no se añaden propiedades ajenas al contrato para engordar el JSON.
func fichaB2HistoriaParaTamano(t *testing.T) (domain.FichaEmpleadoB2, domain.MaterialConsultaRegistroEmpleadoB2) {
	t.Helper()
	actor := solicitudP(t).Actor
	corte := domain.CorteEmpleadoB2{VigenteEn: "2026-09-20", ConocidoEn: time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)}
	f := domain.FichaEmpleadoB2{EmpleadoRef: "emp_" + strings.Repeat("e", 24), PersonaRef: "per_" + strings.Repeat("p", 24), OrganismoRef: "organismo:sintetico", Corte: corte, Version: 501}
	m, err := domain.NuevoMaterialFichaEmpleadoB2(domain.SolicitudFichaEmpleadoB2{Actor: actor, EmpleadoRef: f.EmpleadoRef, OrganismoRef: f.OrganismoRef, Corte: corte})
	if err != nil {
		t.Fatal(err)
	}
	refRelacion := "rel_" + strings.Repeat("r", 24)
	snapshot := func(tipo, ref, nombre string) *domain.SnapshotEntradaCatalogoEmpleadoB2 {
		return &domain.SnapshotEntradaCatalogoEmpleadoB2{OrganismoRef: f.OrganismoRef, Tipo: tipo, Ref: ref, Version: 1, Revision: 1,
			Denominacion: nombre, HuellaSHA256: strings.Repeat("a", 64), VigenteDesde: "2004-01-01", Estado: "publicada"}
	}
	regimen := snapshot("regimen", "regimen:funcionario", "Personal funcionario sujeto al régimen administrativo")
	modalidad := snapshot("modalidad", "modalidad:temporal", "Adscripción temporal por necesidades organizativas del servicio")
	situacion := snapshot("situacion", "situacion:servicio_activo", "Servicio activo en la Administración de destino")
	clase := snapshot("clase_servicio", "clase:otras_administraciones", "Servicios reconocidos prestados en otras administraciones públicas")
	traza := func(tipo string, i int, version int64) domain.TrazaEmpleadoB2 {
		return domain.TrazaEmpleadoB2{Desde: "2006-01-01", RegistradaEn: time.Date(2006+i/20, 1, 1, 9, 0, i%20, 0, time.UTC), Version: version,
			ActoRef: fmt.Sprintf("acto:%s:revision:%03d", tipo, i), FuenteRef: "fuente:registro_personal_sintetico", FuenteVersion: 3}
	}
	for i := 0; i < 20; i++ {
		f.Relaciones = append(f.Relaciones, domain.RelacionRegistroEmpleadoB2{RelacionRef: refRelacion, OrganismoRef: f.OrganismoRef, UnidadRef: "unidad:servicios_generales",
			RegimenRef: regimen.Ref, ModalidadRef: modalidad.Ref, Estado: "vigente", Traza: traza("relacion", i, int64(i+1)), CatalogoSnapshot: domain.SnapshotCatalogoEmpleadoB2{Regimen: regimen, Modalidad: modalidad}})
	}
	for i := 0; i < 160; i++ {
		version := int64(i/20 + 1)
		f.Ocupaciones = append(f.Ocupaciones, domain.OcupacionEmpleadoB2{OcupacionRef: fmt.Sprintf("ocupacion:sintetica:%02d", i%20), RelacionRef: refRelacion, PlazaRef: "plaza:administrativo", PuestoRef: "puesto:gestion", UnidadRef: "unidad:servicios_generales", ModalidadRef: modalidad.Ref, Clase: "temporal", Estado: "finalizada", Traza: traza("ocupacion", i, version), CatalogoSnapshot: domain.SnapshotCatalogoEmpleadoB2{Modalidad: modalidad}})
		f.Situaciones = append(f.Situaciones, domain.SituacionEmpleadoB2{SituacionRef: fmt.Sprintf("situacion:sintetica:%02d", i%20), RelacionRef: refRelacion, CodigoRef: situacion.Ref, Estado: "rectificada", Traza: traza("situacion", i, version), CatalogoSnapshot: domain.SnapshotCatalogoEmpleadoB2{Situacion: situacion}})
		f.Servicios = append(f.Servicios, domain.ServicioReconocidoB2{ServicioRef: fmt.Sprintf("servicio:sintetico:%02d", i%20), RelacionRef: refRelacion, Estado: "reconocido", ClaseRef: clase.Ref, PeriodoDesde: domain.FechaCivil(fmt.Sprintf("%04d-01-01", 1980+i%20)), PeriodoHasta: domain.FechaCivil(fmt.Sprintf("%04d-12-31", 1980+i%20)), DiasReconocidos: 365, Traza: traza("servicio", i, version), CatalogoSnapshot: domain.SnapshotCatalogoEmpleadoB2{ClaseServicio: clase}})
	}
	if err := f.ValidarPara(m); err != nil {
		t.Fatalf("la historia sintética no cumple B2: %v", err)
	}
	return f, m
}

func TestFichaB2TamanoConTresPreparaciones(t *testing.T) {
	f, _ := fichaB2HistoriaParaTamano(t)
	e := ports.EvidenciaRegistroEmpleadoB2{ReciboRef: "recibo:sintetico", DecisionRef: "decision:sintetica", EfectoRef: f.EmpleadoRef, ConsumoHuellaSHA256: strings.Repeat("a", 64), AuditoriaRef: "auditoria:sintetica", ConsultadaEn: f.Corte.ConocidoEn}
	anterior, err := json.Marshal(map[string]any{"data": map[string]any{"ficha": f, "evidencia": e}})
	if err != nil {
		t.Fatal(err)
	}
	cer, err := domain.PrepararServiciosParaCertificados(f)
	if err != nil {
		t.Fatal(err)
	}
	rpt, err := domain.PrepararRelacionParaRPT(f)
	if err != nil {
		t.Fatal(err)
	}
	carrera, err := domain.PrepararAntecedentesCarrera(f)
	if err != nil {
		t.Fatal(err)
	}
	nueva, err := json.Marshal(map[string]any{"data": ports.ResultadoFichaEmpleadoB2{Ficha: f, Evidencia: e, PreparacionServicios: &cer, PreparacionRPT: &rpt, PreparacionCarrera: &carrera}})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("filas relación/ocupación/situación/servicio=%d/%d/%d/%d; bytes anteriores=%d; bytes con preparaciones=%d", len(f.Relaciones), len(f.Ocupaciones), len(f.Situaciones), len(f.Servicios), len(anterior), len(nueva))
	if len(anterior) > 512*1024 || len(nueva) <= 512*1024 {
		t.Fatalf("el caso no reproduce un cruce nuevo de 512 KiB: %d -> %d", len(anterior), len(nueva))
	}
	// Recorrer el cliente actual con los bytes realmente producidos por Go,
	// con Content-Length, sin él y en fragmentos de 2048 bytes (294 fragmentos
	// para la respuesta ampliada, por encima del antiguo límite de 256).
	entrada, err := json.Marshal([]string{string(anterior), string(nueva)})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()
	cmd := exec.CommandContext(ctx, "node", "--input-type=module", "-e", `
import assert from 'node:assert/strict';
import {crearClienteRegistroB2} from '../../../../web/static/portal-empleado/modulos/personal/registro-b2-cliente.js';
let stdin=''; for await (const c of process.stdin) stdin+=c;
const [anterior,nueva]=JSON.parse(stdin);
const consulta={empleadoRef:'emp_'+ 'e'.repeat(24),vigenteEn:'2026-09-20',conocidoEn:'2026-09-20T10:00:00.000000Z'};
for (const declarada of [true,false]) for (const fragmentada of [true,false]) for (const contenido of [anterior,nueva]) {
  const bytes=new TextEncoder().encode(contenido);
  const body=fragmentada?new ReadableStream({start(c){for(let i=0;i<bytes.length;i+=2048)c.enqueue(bytes.subarray(i,i+2048));c.close();}}):contenido;
  const cliente=crearClienteRegistroB2({fetchImpl:async()=>new Response(body,{status:200,headers:{'content-type':'application/json; charset=utf-8',...(declarada?{'content-length':String(bytes.length)}:{})}})});
  const resultado=await cliente.consultarFicha(consulta);
  assert.equal(resultado.ficha.empleado_ref,consulta.empleadoRef);
  if(contenido===nueva) assert.ok(resultado.preparacion_servicios&&resultado.preparacion_rpt&&resultado.preparacion_carrera);
}
`)
	cmd.Stdin = bytes.NewReader(entrada)
	if salida, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cliente B2: %v\n%s", err, salida)
	}
}
