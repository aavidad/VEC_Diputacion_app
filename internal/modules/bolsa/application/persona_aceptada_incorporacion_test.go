package application

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

func solicitudPersonaAceptacionPrueba(t *testing.T) ports.SolicitudConsultaPersonaAceptacionCT {
	t.Helper()
	ahora := time.Date(2026, 9, 30, 11, 0, 0, 0, time.UTC)
	actor, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", core.AuthMethodCertificate, core.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	return ports.SolicitudConsultaPersonaAceptacionCT{Selector: ports.SelectorPersonaAceptacionCT{UnidadRef: "unidad:rrhh", CategoriaRef: "categoria:una", NecesidadRef: "necesidad:una", AceptacionOperacionRef: "aceptacion:una", AceptacionRegistroSHA256: strings.Repeat("a", 64), AperturaOperacionRef: "apertura:una", AperturaRegistroSHA256: strings.Repeat("b", 64), LlamamientoRef: "llamamiento:uno", PropuestaRef: "propuesta:una"}, ActorConfiable: ports.ActorConfiablePersonaAceptacionCT{Vinculo: vinculo, Resultado: actor}}
}

func materialPersonaAceptacionPrueba(t *testing.T, p ports.PreparacionConsultaPersonaAceptacionCT) vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	ahora := time.Date(2026, 9, 30, 11, 0, 0, 0, time.UTC)
	h, _ := p.Recurso.HuellaContextoAutorizacionSHA256()
	c := p.Solicitud.ActorConfiable.Resultado
	x, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:prueba", strings.Repeat("c", 64), strings.Repeat("d", 64), c.RegistroContextoRef, c.HuellaSHA256, ports.AccionConsultaPersonaAceptacionCT, p.Recurso.Referencia, h, ports.AudienciaConsultaPersonaAceptacionCT, ahora, ahora.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	m, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), x, []byte("d"), []byte("m"), c.RepresentacionCanonica, c.Contexto.Instantanea.PersonaVersion, c.Contexto.Instantanea.PerfilVersion, []byte("p"), []byte("s"), []byte("e"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

type proveedorPersonaAceptacionPrueba struct {
	t        *testing.T
	llamadas int
	err      error
	cambiar  func(*ports.PreparacionConsultaPersonaAceptacionCT)
}

func (p *proveedorPersonaAceptacionPrueba) AutorizarConsultaPersonaAceptacionCT(_ context.Context, q ports.PreparacionConsultaPersonaAceptacionCT) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.llamadas++
	if p.err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, p.err
	}
	if p.cambiar != nil {
		p.cambiar(&q)
	}
	return materialPersonaAceptacionPrueba(p.t, q), nil
}

type repositorioPersonaAceptacionPrueba struct {
	llamadas int
	cambiar  func(*ports.ResultadoConsultaPersonaAceptacionCT)
}

func (r *repositorioPersonaAceptacionPrueba) ConsultarPersonaAceptacionCT(_ context.Context, o ports.OrdenConsultaPersonaAceptacionCT) (ports.ResultadoConsultaPersonaAceptacionCT, error) {
	r.llamadas++
	v := resultadoPersonaAceptacionPrueba(o)
	if r.cambiar != nil {
		r.cambiar(&v)
	}
	return v, nil
}
func resultadoPersonaAceptacionPrueba(o ports.OrdenConsultaPersonaAceptacionCT) ports.ResultadoConsultaPersonaAceptacionCT {
	s := o.Solicitud.Selector
	x := o.Material.ResumenCapacidad()
	return ports.ResultadoConsultaPersonaAceptacionCT{Estado: "acreditado", Aceptacion: &ports.AceptacionPersonaCT{OperacionRef: s.AceptacionOperacionRef, ReciboRef: "recibo:uno", RegistroSHA256: s.AceptacionRegistroSHA256, AperturaOperacionRef: s.AperturaOperacionRef, AperturaRegistroSHA256: s.AperturaRegistroSHA256, LlamamientoRef: s.LlamamientoRef}, Persona: &ports.PersonaAceptadaCT{Ref: "per_" + strings.Repeat("p", 24), Version: 1}, Vinculo: &ports.VinculoPersonaAceptadaCT{Ref: "vinculo:uno", Version: 1, ProcedenciaRef: "procedencia:una", ProcedenciaVersion: 1, ProcedenciaSHA256: strings.Repeat("d", 64), Poblacion: "externa", VigenteHasta: x.EmitidaEn().Add(time.Hour)}, Evidencia: ports.EvidenciaConsultaPersonaAceptacionCT{DecisionRef: x.DecisionRef(), ConsumoHuellaSHA256: strings.Repeat("e", 64), AuditoriaRef: "auditoria:una", ConsultadaEn: x.EmitidaEn().Add(time.Microsecond)}}
}

func TestPersonaAceptacionCTCanonExactoYSelectorCompleto(t *testing.T) {
	s := solicitudPersonaAceptacionPrueba(t)
	p, err := PrepararConsultaPersonaAceptacionCT(s)
	if err != nil {
		t.Fatal(err)
	}
	esperado := `{"esquema":"vec.bolsa.persona-aceptacion-ct.consulta.v1","unidad_ref":"unidad:rrhh","categoria_ref":"categoria:una","necesidad_ref":"necesidad:una","aceptacion_operacion_ref":"aceptacion:una","aceptacion_registro_sha256":"` + strings.Repeat("a", 64) + `","apertura_operacion_ref":"apertura:una","apertura_registro_sha256":"` + strings.Repeat("b", 64) + `","llamamiento_ref":"llamamiento:uno","propuesta_ref":"propuesta:una"}`
	if string(p.MaterialCanonico) != esperado || len(p.Recurso.Ambitos) != 1 || p.Recurso.Ambitos["unidad_ref"] != s.Selector.UnidadRef || len(p.Recurso.Atributos) != 1 {
		t.Fatal("material o ámbito divergente")
	}
	for _, cambio := range []func(*ports.SelectorPersonaAceptacionCT){func(x *ports.SelectorPersonaAceptacionCT) { x.UnidadRef = "unidad:otra" }, func(x *ports.SelectorPersonaAceptacionCT) { x.CategoriaRef = "categoria:otra" }, func(x *ports.SelectorPersonaAceptacionCT) { x.NecesidadRef = "necesidad:otra" }, func(x *ports.SelectorPersonaAceptacionCT) { x.AceptacionOperacionRef = "aceptacion:otra" }, func(x *ports.SelectorPersonaAceptacionCT) { x.AceptacionRegistroSHA256 = strings.Repeat("c", 64) }, func(x *ports.SelectorPersonaAceptacionCT) { x.AperturaOperacionRef = "apertura:otra" }, func(x *ports.SelectorPersonaAceptacionCT) { x.AperturaRegistroSHA256 = strings.Repeat("d", 64) }, func(x *ports.SelectorPersonaAceptacionCT) { x.LlamamientoRef = "llamamiento:otro" }, func(x *ports.SelectorPersonaAceptacionCT) { x.PropuestaRef = "propuesta:otra" }} {
		otro := s
		cambio(&otro.Selector)
		q, e := PrepararConsultaPersonaAceptacionCT(otro)
		if e != nil || q.Recurso.Atributos["material_sha256"] == p.Recurso.Atributos["material_sha256"] {
			t.Fatal("campo no ligado al material")
		}
	}
}

func TestPersonaAceptacionCTConsultaFrescaYEstadosPendientes(t *testing.T) {
	for _, estado := range []string{"acreditado", "pendiente", "no_encontrada"} {
		t.Run(estado, func(t *testing.T) {
			s := solicitudPersonaAceptacionPrueba(t)
			p := &proveedorPersonaAceptacionPrueba{t: t}
			r := &repositorioPersonaAceptacionPrueba{cambiar: func(v *ports.ResultadoConsultaPersonaAceptacionCT) {
				v.Estado = estado
				if estado != "acreditado" {
					v.Persona = nil
					v.Vinculo = nil
				}
				if estado == "no_encontrada" {
					v.Aceptacion = nil
				}
			}}
			c, _ := NuevoServicioConsultaPersonaAceptacionCT(p, r, func() time.Time { return time.Date(2026, 9, 30, 11, 0, 0, 1000, time.UTC) })
			for i := 0; i < 2; i++ {
				v, e := c.ConsultarPersonaAceptacionCT(context.Background(), s)
				if e != nil || v.Estado != estado {
					t.Fatalf("consulta: %v", e)
				}
			}
			p.err = ports.ErrConsultaPersonaAceptacionCTDenegada
			if _, err := c.ConsultarPersonaAceptacionCT(context.Background(), s); !errors.Is(err, ports.ErrConsultaPersonaAceptacionCTDenegada) || p.llamadas != 3 || r.llamadas != 2 {
				t.Fatal("se reutilizó permiso histórico")
			}
		})
	}
}

func TestPersonaAceptacionCTRechazaResultadoParcialOCruzado(t *testing.T) {
	casos := map[string]func(*ports.ResultadoConsultaPersonaAceptacionCT){"aceptacion": func(v *ports.ResultadoConsultaPersonaAceptacionCT) { v.Aceptacion.OperacionRef = "aceptacion:otra" }, "huella": func(v *ports.ResultadoConsultaPersonaAceptacionCT) {
		v.Aceptacion.RegistroSHA256 = strings.Repeat("f", 64)
	}, "apertura": func(v *ports.ResultadoConsultaPersonaAceptacionCT) {
		v.Aceptacion.AperturaOperacionRef = "apertura:otra"
	}, "llamamiento": func(v *ports.ResultadoConsultaPersonaAceptacionCT) { v.Aceptacion.LlamamientoRef = "llamamiento:otro" }, "persona": func(v *ports.ResultadoConsultaPersonaAceptacionCT) { v.Persona.Ref = "persona:ejercicio:uno" }, "version": func(v *ports.ResultadoConsultaPersonaAceptacionCT) { v.Persona.Version = 0 }, "vigencia": func(v *ports.ResultadoConsultaPersonaAceptacionCT) { v.Vinculo.VigenteHasta = v.Evidencia.ConsultadaEn }, "procedencia": func(v *ports.ResultadoConsultaPersonaAceptacionCT) { v.Vinculo.ProcedenciaSHA256 = "" }, "decision": func(v *ports.ResultadoConsultaPersonaAceptacionCT) { v.Evidencia.DecisionRef = "decision:otra" }, "auditoria": func(v *ports.ResultadoConsultaPersonaAceptacionCT) { v.Evidencia.AuditoriaRef = "" }, "hora": func(v *ports.ResultadoConsultaPersonaAceptacionCT) {
		v.Evidencia.ConsultadaEn = v.Evidencia.ConsultadaEn.Add(time.Hour)
	}, "pendiente con persona": func(v *ports.ResultadoConsultaPersonaAceptacionCT) { v.Estado = "pendiente" }, "ausencia con aceptacion": func(v *ports.ResultadoConsultaPersonaAceptacionCT) { v.Estado = "no_encontrada" }}
	for nombre, cambio := range casos {
		t.Run(nombre, func(t *testing.T) {
			s := solicitudPersonaAceptacionPrueba(t)
			p := &proveedorPersonaAceptacionPrueba{t: t}
			r := &repositorioPersonaAceptacionPrueba{cambiar: cambio}
			c, _ := NuevoServicioConsultaPersonaAceptacionCT(p, r, func() time.Time { return time.Date(2026, 9, 30, 11, 0, 0, 1000, time.UTC) })
			v, e := c.ConsultarPersonaAceptacionCT(context.Background(), s)
			if e == nil || !reflect.DeepEqual(v, ports.ResultadoConsultaPersonaAceptacionCT{}) {
				t.Fatal("exposición parcial")
			}
		})
	}
}

func TestPersonaAceptacionCTMaterialAjenoYNulosAntesRepositorio(t *testing.T) {
	s := solicitudPersonaAceptacionPrueba(t)
	p := &proveedorPersonaAceptacionPrueba{t: t, cambiar: func(x *ports.PreparacionConsultaPersonaAceptacionCT) { x.Recurso.Referencia = "aceptacion:otra" }}
	r := &repositorioPersonaAceptacionPrueba{}
	c, _ := NuevoServicioConsultaPersonaAceptacionCT(p, r, func() time.Time { return time.Date(2026, 9, 30, 11, 0, 0, 1000, time.UTC) })
	if _, e := c.ConsultarPersonaAceptacionCT(context.Background(), s); !errors.Is(e, ports.ErrConsultaPersonaAceptacionCTDenegada) || r.llamadas != 0 {
		t.Fatal("material ajeno llegó al repositorio")
	}
	var pn *proveedorPersonaAceptacionPrueba
	if _, e := NuevoServicioConsultaPersonaAceptacionCT(pn, r, time.Now); !errors.Is(e, ports.ErrConsultaPersonaAceptacionCTNoDisponible) {
		t.Fatal("nil tipado")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := c.ConsultarPersonaAceptacionCT(ctx, s); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	if !errors.Is(ErrorConsultaPersonaAceptacionCT(context.Background(), errors.New("detalle privado")), ports.ErrConsultaPersonaAceptacionCTNoDisponible) {
		t.Fatal("filtración de error")
	}
}

// Este doble fabrica decisiones nominales con el dominio V3 real. El
// almacenamiento y la atestación son dobles; no acreditan SQL ni criptografía.
type emisorNominalPersonaAceptacionPrueba struct {
	t                         *testing.T
	campos, obligaciones      []string
	err                       error
	llamadas                  int
	ultima                    core.SolicitudAutorizacionLigadaV3
	invalida, sinExportador   bool
	denegada, sinConfirmacion bool
}

func (e *emisorNominalPersonaAceptacionPrueba) EmitirMaterialAutorizacionAtestadaV3(_ context.Context, s core.SolicitudAutorizacionLigadaV3, c core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	e.llamadas++
	e.ultima = s
	if e.err != nil {
		return core.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, e.err
	}
	if e.invalida {
		return core.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, nil
	}
	datos, err := s.Datos()
	if err != nil {
		e.t.Fatal(err)
	}
	v, _ := datos.VinculoAutenticacionActor.Datos()
	ahora := time.Date(2026, 9, 30, 11, 0, 0, 0, time.UTC)
	rol := core.VersionRol{RolID: "tecnico_bolsa", Version: 1, Nombre: "Técnico", Estado: core.EstadoVersionRolPublicada, Concesiones: []core.ConcesionRol{{Accion: datos.Accion, ModuloID: datos.Recurso.ModuloID, TipoRecurso: datos.Recurso.Tipo, Finalidades: []string{datos.Finalidad}, GarantiaMinima: core.AuthAssuranceHigh, CamposPermitidos: e.campos, Obligaciones: e.obligaciones}}, PublicadaPor: "seguridad", PublicadaEn: ahora.Add(-time.Hour)}
	if e.denegada {
		rol.Concesiones[0].Accion = "bolsa.otra_operacion.consultar"
	}
	huella, _ := core.HuellaCatalogoPoliticasAutorizacion(nil)
	i := core.InstantaneaAutorizacion{AsignacionPerfil: core.AsignacionPerfil{AsignacionID: "asig-bolsa", Version: 1, PerfilActivoRef: v.PerfilActivoRef, PrincipalID: v.PrincipalID, VersionRolRef: rol.Referencia(), Estado: core.EstadoAsignacionPerfilActiva, Ambitos: []core.AmbitoPerfil{{Clave: "unidad_ref", Valores: []string{datos.Recurso.Ambitos["unidad_ref"]}}}, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), EmitidaPor: "seguridad", EmitidaEn: ahora.Add(-time.Hour)}, VersionRol: rol, ControlVigenciaVersionRol: core.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1, Estado: core.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: "seguridad", ActualizadoEn: ahora.Add(-time.Hour)}, RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huella}
	f, err := core.NuevaEvidenciaEvaluacionAutorizacionV3(s, i, "decision:prueba", ahora, ahora.Add(time.Minute))
	if err != nil {
		e.t.Fatal(err)
	}
	d, err := core.NuevaDecisionAutorizacionLigadaV3(s, f)
	if err != nil {
		e.t.Fatal(err)
	}
	if e.denegada {
		return d, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, nil
	}
	o, err := vecports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(s, d, datos.ReferenciaMotivo, c)
	if err != nil {
		e.t.Fatal(err)
	}
	confirmacion, err := vecports.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(context.Background(), registroConcesionBorradorPrueba{instante: ahora}, o)
	if err != nil {
		e.t.Fatal(err)
	}
	if e.sinConfirmacion {
		return d, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, nil
	}
	dh, _ := core.HuellaSHA256DecisionAutorizacionV3(d)
	mh, _ := core.HuellaSHA256MotivoAutorizacionV2(datos.ReferenciaMotivo)
	rh, _ := datos.Recurso.HuellaContextoAutorizacionSHA256()
	x, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:prueba", dh, mh, c.RegistroContextoRef, c.HuellaSHA256, datos.Accion, datos.Recurso.Referencia, rh, ports.AudienciaConsultaPersonaAceptacionCT, ahora, ahora.Add(3*time.Second))
	if err != nil {
		e.t.Fatal(err)
	}
	dc, _ := core.RepresentacionCanonicaDecisionAutorizacionV3(d)
	mc, _ := core.RepresentacionCanonicaMotivoAutorizacionV2(datos.ReferenciaMotivo)
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	m, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), x, dc, mc, c.RepresentacionCanonica, c.Contexto.Instantanea.PersonaVersion, c.Contexto.Instantanea.PerfilVersion, []byte("p"), []byte("s"), []byte("e"), raiz)
	if err != nil {
		e.t.Fatal(err)
	}
	if e.sinExportador {
		return d, confirmacion, nil, nil
	}
	return d, confirmacion, exportadorBorradorPrueba{material: m}, nil
}

func TestPersonaAceptacionCTProveedorNoConfundeDependenciaIncoherenteConDenegacion(t *testing.T) {
	for _, caso := range []string{"decision inválida", "exportador ausente", "dependencia caída", "denegación", "decisión denegada sin error", "confirmación vacía", "fuente caída y denegación", "registro caído y denegación", "denegación registrada", "cancelación sin detalle"} {
		t.Run(caso, func(t *testing.T) {
			e := &emisorNominalPersonaAceptacionPrueba{t: t, campos: []string{"aceptacion", "persona", "vinculo"}}
			esperado := ports.ErrConsultaPersonaAceptacionCTNoDisponible
			switch caso {
			case "decision inválida":
				e.invalida = true
			case "exportador ausente":
				e.sinExportador = true
			case "dependencia caída":
				e.err = errors.New("detalle privado")
			case "denegación":
				e.err = core.ErrAutorizacionDenegada
				esperado = ports.ErrConsultaPersonaAceptacionCTDenegada
			case "decisión denegada sin error":
				e.denegada = true
				esperado = ports.ErrConsultaPersonaAceptacionCTDenegada
			case "confirmación vacía":
				e.sinConfirmacion = true
			case "fuente caída y denegación":
				e.err = errors.Join(core.ErrAutorizacionDenegada, vecports.ErrFuenteAutorizacionNoDisponible)
			case "registro caído y denegación":
				e.err = errors.Join(core.ErrAutorizacionDenegada, vecports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible)
			case "denegación registrada":
				e.err = vecports.ErrDenegacionExplicitaAutorizacionLigadaV3
				esperado = ports.ErrConsultaPersonaAceptacionCTDenegada
			case "cancelación sin detalle":
				e.err = errors.Join(errors.New("detalle privado"), context.Canceled, core.ErrAutorizacionDenegada)
				esperado = context.Canceled
			}
			p, _ := NuevoProveedorNominalConsultaPersonaAceptacionCT(e, motivoBorradorPrueba(), func(context.Context) (core.ReferenciaCorrelacionAutorizacionV2, error) {
				return correlacionBorradorPrueba(t), nil
			}, func() time.Time { return time.Date(2026, 9, 30, 11, 0, 0, 1000, time.UTC) })
			q, _ := PrepararConsultaPersonaAceptacionCT(solicitudPersonaAceptacionPrueba(t))
			if _, err := p.AutorizarConsultaPersonaAceptacionCT(context.Background(), q); err != esperado {
				t.Fatalf("error nominal: %v", err)
			}
		})
	}
}

func TestPersonaAceptacionCTProveedorNominalReutilizaEmisorYReconstruyeMaterial(t *testing.T) {
	s := solicitudPersonaAceptacionPrueba(t)
	e := &emisorNominalPersonaAceptacionPrueba{t: t, campos: []string{"aceptacion", "persona", "vinculo"}}
	ahora := func() time.Time { return time.Date(2026, 9, 30, 11, 0, 0, 1000, time.UTC) }
	p, err := NuevoProveedorNominalConsultaPersonaAceptacionCT(e, motivoBorradorPrueba(), func(context.Context) (core.ReferenciaCorrelacionAutorizacionV2, error) {
		return correlacionBorradorPrueba(t), nil
	}, ahora)
	if err != nil {
		t.Fatal(err)
	}
	preparada, _ := PrepararConsultaPersonaAceptacionCT(s)
	preparada.Recurso.Referencia = "aceptacion:otra"
	preparada.Recurso.Ambitos["unidad_ref"] = "unidad:otra"
	preparada.MaterialCanonico = []byte(`{"esquema":"otro"}`)
	m, err := p.AutorizarConsultaPersonaAceptacionCT(context.Background(), preparada)
	if err != nil {
		t.Fatal(err)
	}
	datos, _ := e.ultima.Datos()
	if datos.Accion != ports.AccionConsultaPersonaAceptacionCT || datos.Finalidad != ports.FinalidadConsultaPersonaAceptacionCT || datos.Recurso.Referencia != s.Selector.AceptacionOperacionRef || datos.Recurso.Ambitos["unidad_ref"] != s.Selector.UnidadRef || m.ResumenCapacidad().AudienciaConsumo() != ports.AudienciaConsultaPersonaAceptacionCT {
		t.Fatal("material mutable otorgó autoridad")
	}
	r := &repositorioPersonaAceptacionPrueba{}
	c, _ := NuevoServicioConsultaPersonaAceptacionCT(p, r, ahora)
	if _, err := c.ConsultarPersonaAceptacionCT(context.Background(), s); err != nil || r.llamadas != 1 || e.llamadas != 2 {
		t.Fatalf("wrapper→consumidor: %v", err)
	}
}

func TestPersonaAceptacionCTProveedorNominalExigeCamposYObligacionesExactos(t *testing.T) {
	for _, caso := range []struct {
		nombre               string
		campos, obligaciones []string
	}{{"sin campos", nil, nil}, {"menos campos", []string{"persona"}, nil}, {"más campos", []string{"aceptacion", "persona", "vinculo", "otro"}, nil}, {"obligación", []string{"aceptacion", "persona", "vinculo"}, []string{"auditar"}}} {
		t.Run(caso.nombre, func(t *testing.T) {
			e := &emisorNominalPersonaAceptacionPrueba{t: t, campos: caso.campos, obligaciones: caso.obligaciones}
			p, _ := NuevoProveedorNominalConsultaPersonaAceptacionCT(e, motivoBorradorPrueba(), func(context.Context) (core.ReferenciaCorrelacionAutorizacionV2, error) {
				return correlacionBorradorPrueba(t), nil
			}, func() time.Time { return time.Date(2026, 9, 30, 11, 0, 0, 1000, time.UTC) })
			q, _ := PrepararConsultaPersonaAceptacionCT(solicitudPersonaAceptacionPrueba(t))
			if _, err := p.AutorizarConsultaPersonaAceptacionCT(context.Background(), q); !errors.Is(err, ports.ErrConsultaPersonaAceptacionCTDenegada) {
				t.Fatalf("proyección excesiva o incompleta: %v", err)
			}
		})
	}
}
