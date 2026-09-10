package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	hist "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/historiaincorporacion"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	lector "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type proveedorTransporteV2 func(context.Context, lector.MaterialV2) (lector.AutorizacionV2, error)

func (f proveedorTransporteV2) AutorizarLecturaIncorporacionV2(c context.Context, m lector.MaterialV2) (lector.AutorizacionV2, error) {
	return f(c, m)
}

type txTransporteV2 func(context.Context, lector.Selector, lector.OrdenV2) (lector.Resultado, error)

func (f txTransporteV2) LeerRegistroPersonalV2(c context.Context, s lector.Selector, o lector.OrdenV2) (lector.Resultado, error) {
	return f(c, s, o)
}

type relojTransporteV2 struct{ t time.Time }

func (r *relojTransporteV2) Ahora() time.Time { return r.t }

func ordenCTTransporte(t *testing.T, d ct.DatosMaterialConfirmacionIncorporacionV2, ahora time.Time) ct.OrdenConfirmacionIncorporacionV2 {
	t.Helper()
	m, e := ct.NuevoMaterialConfirmacionIncorporacionV2(d, ahora)
	registroV2Exigir(t, e)
	r, e := ct.RecursoConfirmacionIncorporacionV2(m)
	registroV2Exigir(t, e)
	s, e := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: d.Contexto.Vinculo,
		Accion: ct.AccionConfirmarIncorporacion, Finalidad: ct.FinalidadConfirmarIncorporacion, Recurso: r, ReferenciaMotivo: d.MotivoV3, Correlacion: d.CorrelacionV3})
	registroV2Exigir(t, e)
	snapshot := instantaneaLectorV2(t, d.Contexto, d.Preparacion.OrganizacionRef, d.Preparacion.UnidadRef, ahora)
	a := permisoFijoV2(t, s, d.Contexto, snapshot, ahora.Add(-time.Millisecond), fmt.Sprintf("dec_%032x", ahora.UnixMicro()), ct.AudienciaConfirmacionIncorporacionV2)
	o, e := ct.NuevaOrdenConfirmacionIncorporacionV2(m, ct.AutorizacionConfirmacionIncorporacionV2{Solicitud: a.Solicitud, Decision: a.Decision, Confirmacion: a.Confirmacion, Exportacion: a.Exportacion}, ahora)
	registroV2Exigir(t, e)
	return o
}
func ordenLectorTransporte(t *testing.T, o ct.OrdenConfirmacionIncorporacionV2, ahora time.Time, mutar func(*lector.Selector, *string, *ct.ContextoAutorizacionAltaV3)) lector.OrdenV2 {
	t.Helper()
	d, e := o.Material().Datos()
	registroV2Exigir(t, e)
	s := lector.Selector{OrganizacionRef: d.Preparacion.OrganizacionRef, SolicitudRef: d.Personal.Solicitud.SolicitudRef,
		ExpedienteRef: d.Personal.Solicitud.ExpedienteRef, VersionExpediente: d.Personal.Solicitud.VersionExpediente,
		ResultadoRef: d.Personal.Resultado.ResultadoRef, ReciboRef: d.Personal.Resultado.ReciboRef, RelacionRef: d.Personal.Resultado.RelacionRef, OcupacionRef: d.Personal.Resultado.OcupacionRef, MaterialSHA256: d.Personal.MaterialSHA256}
	u, cx := d.Preparacion.UnidadRef, d.Contexto
	if mutar != nil {
		mutar(&s, &u, &cx)
	}
	snapshot := instantaneaLectorV2(t, cx, s.OrganizacionRef, u, ahora)
	reloj := &relojTransporteV2{ahora}
	var capturada lector.OrdenV2
	llamadas := 0
	fin := errors.New("TX doble captura sin efecto")
	c, e := lector.NuevoV2(proveedorTransporteV2(func(_ context.Context, m lector.MaterialV2) (lector.AutorizacionV2, error) {
		a := permisoFijoV2(t, solicitudLectorV2(t, m, nil), cx, snapshot, ahora, fmt.Sprintf("dec_%032x", ahora.UnixMicro()+1), lector.AudienciaV2)
		reloj.t = ahora.Add(time.Microsecond)
		return a, nil
	}), txTransporteV2(func(_ context.Context, _ lector.Selector, ord lector.OrdenV2) (lector.Resultado, error) {
		capturada = ord
		llamadas++
		return lector.Resultado{}, fin
	}), reloj)
	registroV2Exigir(t, e)
	_, e = c.Leer(context.Background(), s, u, cx)
	if e == nil || llamadas != 1 || capturada.ValidarEn(reloj.t) != nil {
		t.Fatal("fixture no emitió OrdenV2 legítima")
	}
	return capturada
}
func fixtureTransporte(t *testing.T) (ct.OrdenConfirmacionIncorporacionV2, lector.OrdenV2, time.Time) {
	t.Helper()
	t0 := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	d := registroV2Datos(t, t0)
	d.VersionActualExpediente = 8
	o := ordenCTTransporte(t, d, t0)
	l := ordenLectorTransporte(t, o, t0, nil)
	return o, l, t0.Add(2 * time.Microsecond)
}
func serialTransporte(t *testing.T, v any) []byte {
	t.Helper()
	b, e := json.Marshal(v)
	registroV2Exigir(t, e)
	return b
}
func wireTransporte(t *testing.T, o ct.OrdenConfirmacionIncorporacionV2, now time.Time) (salidaTransporteRegistroV2, ct.ResultadoRegistroIncorporacionV2) {
	t.Helper()
	r := registroV2ResultadoOriginal(t, o, now)
	registroV2Exigir(t, r.ValidarPara(o, now))
	p, a, b := r.Historia.EvidenciaSeguimiento()
	e, err := ct.CapturarEvidenciaOrdenOriginalIncorporacionV2(context.Background(), o)
	registroV2Exigir(t, err)
	return salidaTransporteRegistroV2{r.Recibo, hist.Documento{Esquema: hist.EsquemaDocumento, Evidencia: e, Recibo: r.Recibo, Publicacion: p, Anterior: a, Posterior: b}, false, r.ConsumosActuales}, r
}

func TestRegistroIncorporacionV2TransporteParametros(t *testing.T) {
	o, l, now := fixtureTransporte(t)
	ctx := context.Background()
	ref := registroV2Ref("seguimiento:ejercicio:001")
	llamada, e := PrepararLlamadaRegistroIncorporacionV2(ctx, o, l, ref, now)
	registroV2Exigir(t, e)
	p := llamada.Parametros()
	m, e := o.Material().MaterialCanonico()
	registroV2Exigir(t, e)
	evidencia, e := ct.CapturarEvidenciaOrdenOriginalIncorporacionV2(ctx, o)
	registroV2Exigir(t, e)
	esperado := []any{m, ref}
	for i, x := range []vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{o.Exportacion(), l.Exportacion()} {
		var a, b any
		if i == 0 {
			a = strconv.FormatUint(x.PersonaVersion(), 10)
			b = strconv.FormatUint(x.PerfilVersion(), 10)
		} else {
			a = int64(x.PersonaVersion())
			b = int64(x.PerfilVersion())
		}
		esperado = append(esperado, x.CapacidadCanonica(), x.DecisionCanonica(), x.MotivoCanonico(), x.ContextoActorCanonico(), a, b, x.PayloadVECAD3(), x.SobreCOSESign1(), x.EvidenciaVerificacion(), x.RaizPublicaSPKI())
	}
	esperado = append(esperado, serialTransporte(t, evidencia))
	if len(p) != 23 || !reflect.DeepEqual(p, esperado) {
		t.Fatal("parámetros no coinciden; no imprimir material")
	}
	for i, v := range p {
		if b, ok := v.([]byte); ok {
			b[0] ^= 1
		}
		p[i] = nil
	}
	if !reflect.DeepEqual(llamada.Parametros(), esperado) {
		t.Fatal("Parametros comparte memoria")
	}
	d, _ := o.Material().MaterialCanonico()
	if !bytes.Equal(d, m) {
		t.Fatal("orden modificada")
	}
	if (LlamadaRegistroIncorporacionV2{}).Parametros() != nil {
		t.Fatal("cero no es nil")
	}
	// El límite viene del contrato V3, no de un cast CT a int64.
	x := o.Exportacion()
	construir := func(version uint64) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
		return vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
			x.CapacidadCanonica(), x.ResumenCapacidad(), x.DecisionCanonica(), x.MotivoCanonico(), x.ContextoActorCanonico(), version, version, x.PayloadVECAD3(), x.SobreCOSESign1(), x.EvidenciaVerificacion(), x.RaizPublicaSPKI())
	}
	limite, e := construir(vp.VersionMaximaExactaMaterialConsumoV3)
	registroV2Exigir(t, e)
	z, e := exportacionParametrosRegistroV2(limite, true)
	registroV2Exigir(t, e)
	if z[4] != "9007199254740991" || z[5] != "9007199254740991" {
		t.Fatal("numeric no exacto")
	}
	if _, e = construir(math.MaxUint64); e == nil {
		t.Fatal("fuera del máximo común admitido")
	}
}

func TestRegistroIncorporacionV2TransporteLigaduras(t *testing.T) {
	o, _, now := fixtureTransporte(t)
	t0 := now.Add(-2 * time.Microsecond)
	cambios := map[string]func(*lector.Selector, *string, *ct.ContextoAutorizacionAltaV3){
		"org": func(s *lector.Selector, _ *string, _ *ct.ContextoAutorizacionAltaV3) {
			s.OrganizacionRef = registroV2Ref("otraorg")
		},
		"solicitud": func(s *lector.Selector, _ *string, _ *ct.ContextoAutorizacionAltaV3) { s.SolicitudRef += "x" },
		"expediente": func(s *lector.Selector, _ *string, _ *ct.ContextoAutorizacionAltaV3) {
			s.ExpedienteRef = registroV2Ref("otroexp")
		},
		"version":   func(s *lector.Selector, _ *string, _ *ct.ContextoAutorizacionAltaV3) { s.VersionExpediente++ },
		"resultado": func(s *lector.Selector, _ *string, _ *ct.ContextoAutorizacionAltaV3) { s.ResultadoRef += "x" },
		"recibo":    func(s *lector.Selector, _ *string, _ *ct.ContextoAutorizacionAltaV3) { s.ReciboRef += "x" },
		"relacion": func(s *lector.Selector, _ *string, _ *ct.ContextoAutorizacionAltaV3) {
			s.RelacionRef = registroV2Ref("otrarel")
		},
		"ocupacion": func(s *lector.Selector, _ *string, _ *ct.ContextoAutorizacionAltaV3) { s.OcupacionRef += "x" },
		"material": func(s *lector.Selector, _ *string, _ *ct.ContextoAutorizacionAltaV3) {
			s.MaterialSHA256 = strings.Repeat("c", 64)
		},
		"unidad": func(_ *lector.Selector, u *string, _ *ct.ContextoAutorizacionAltaV3) {
			*u = registroV2Ref("otraunidad")
		},
		"actor": func(_ *lector.Selector, _ *string, c *ct.ContextoAutorizacionAltaV3) {
			*c = registroV2_contextoAutorizacionAltaV3PruebaConMarcas(t, t0, "b", "b")
		},
		"contexto_fresco": func(_ *lector.Selector, _ *string, c *ct.ContextoAutorizacionAltaV3) {
			*c = registroV2_contextoAutorizacionAltaV3Prueba(t, t0.Add(time.Microsecond))
		},
	}
	for n, mutar := range cambios {
		t.Run(n, func(t *testing.T) {
			l := ordenLectorTransporte(t, o, t0.Add(time.Microsecond), mutar)
			r, e := PrepararLlamadaRegistroIncorporacionV2(context.Background(), o, l, registroV2Ref("seguimiento"), now.Add(time.Microsecond))
			if e == nil || r.Parametros() != nil {
				t.Fatal("cruce nominal admitido")
			}
		})
	}
}

func TestRegistroIncorporacionV2TransporteDecoderNominalReplay(t *testing.T) {
	o, _, now := fixtureTransporte(t)
	w, original := wireTransporte(t, o, now)
	b := serialTransporte(t, w)
	r, e := DecodificarRegistroIncorporacionV2(context.Background(), b, o, o, now)
	registroV2Exigir(t, e)
	if !reflect.DeepEqual(r.Recibo, original.Recibo) || r.Recuperado {
		t.Fatal("recibo no exacto")
	}
	r.Recibo.MaterialOriginalCanonico[0] ^= 1
	r.Recibo.Transicion.Documentos[0].Referencia = "ajena"
	var relectura salidaTransporteRegistroV2
	registroV2Exigir(t, json.Unmarshal(b, &relectura))
	if !reflect.DeepEqual(relectura.Recibo, original.Recibo) || !reflect.DeepEqual(r.Historia.ReciboOriginal(), original.Recibo) {
		t.Fatal("retorno comparte memoria")
	}
	// El wire SQL utiliza .US incluso con ceros. No tratarlo como error ni
	// normalizar strings de negocio; solo timestamps tipados y válidos.
	fixed := bytes.ReplaceAll(b, []byte("2026-09-08T12:00:00Z"), []byte("2026-09-08T12:00:00.000000Z"))
	if bytes.Equal(fixed, b) {
		t.Fatal("fixture no ejercita formato SQL")
	}
	_, e = DecodificarRegistroIncorporacionV2(context.Background(), fixed, o, o, now)
	registroV2Exigir(t, e)
	arraysNulos := bytes.Replace(b, []byte(`"actuaciones":[]`), []byte(`"actuaciones":null`), 1)
	if bytes.Equal(arraysNulos, b) {
		t.Fatal("fixture sin array inicial")
	}
	_, e = DecodificarRegistroIncorporacionV2(context.Background(), arraysNulos, o, o, now)
	registroV2Exigir(t, e)
	d, _ := o.Material().Datos()
	futuro := now.Add(24 * time.Hour)
	d.Contexto = registroV2_contextoAutorizacionAltaV3Prueba(t, futuro)
	v, e := d.Contexto.Vinculo.Datos()
	registroV2Exigir(t, e)
	d.SolicitudContexto = ct.SolicitudResolverContextoAutorizacionAltaV3{AutenticacionRef: v.AutenticacionRef, SesionRef: v.SesionRef, PerfilRef: v.PerfilActivoRef}
	fresh := ordenCTTransporte(t, d, futuro)
	if o.ValidarEn(futuro) == nil {
		t.Fatal("orden histórica debería estar caducada")
	}
	w.Recuperado = true
	w.Consumos = registroV2Consumos(t, fresh, futuro)
	rb := serialTransporte(t, w)
	replay, e := DecodificarRegistroIncorporacionV2(context.Background(), rb, fresh, o, futuro)
	registroV2Exigir(t, e)
	if !replay.Recuperado || !reflect.DeepEqual(replay.Recibo, original.Recibo) {
		t.Fatal("historia reetiquetada")
	}
	if _, e = DecodificarRegistroIncorporacionV2(context.Background(), rb, fresh, fresh, futuro); e == nil {
		t.Fatal("orden fresca sustituyó original")
	}
	if _, e = DecodificarRegistroIncorporacionV2(context.Background(), rb, o, o, futuro); e == nil {
		t.Fatal("permiso antiguo como actual")
	}
	clear(rb)
	pub, a, p := replay.Historia.EvidenciaSeguimiento()
	p.Actuaciones[0].Documentos[0].Referencia = "mutada"
	pub.Estados[0].Clave = "mutado"
	a.Referencia = "otra"
	registroV2Exigir(t, replay.ValidarPara(fresh, futuro))
}

func TestRegistroIncorporacionV2TransporteJSONCerrado(t *testing.T) {
	o, _, now := fixtureTransporte(t)
	w, _ := wireTransporte(t, o, now)
	b := serialTransporte(t, w)
	// Todas las mutaciones parten de salida COMPLETA y válida.
	validar := func(t *testing.T, raw []byte) {
		t.Helper()
		r, e := DecodificarRegistroIncorporacionV2(context.Background(), raw, o, o, now)
		if e == nil || !reflect.DeepEqual(r, ct.ResultadoRegistroIncorporacionV2{}) {
			t.Fatal("JSON inválido admitido")
		}
	}
	for _, path := range [][]string{{}, {"historia"}, {"historia", "Evidencia"}, {"recibo"}, {"consumos_actuales"}, {"historia", "Anterior"}, {"historia", "Publicacion"}} {
		label := strings.Join(path, "/")
		base, e := arbolJSONRegistroV2(b)
		registroV2Exigir(t, e)
		node := base.(map[string]any)
		for _, k := range path {
			node = node[k].(map[string]any)
		}
		for key := range node {
			for _, modo := range []string{"ausente", "null", "alias", "tipo"} {
				if _, array := node[key].([]any); modo == "null" && (array || node[key] == nil) {
					continue // null de slices es representable; no es un adversario de tipos.
				}
				t.Run(label+"/"+key+"/"+modo, func(t *testing.T) {
					v, e := arbolJSONRegistroV2(b)
					registroV2Exigir(t, e)
					n := v.(map[string]any)
					for _, k := range path {
						n = n[k].(map[string]any)
					}
					switch modo {
					case "ausente":
						delete(n, key)
					case "null":
						n[key] = nil
					case "alias":
						value := n[key]
						delete(n, key)
						n[strings.ToUpper(key)] = value
					case "tipo":
						n[key] = map[string]any{"invalido": true}
					}
					if modo == "alias" && strings.ToUpper(key) == key {
						t.Fatal("alias no cambia clave")
					}
					validar(t, serialTransporte(t, v))
				})
			}
		}
	}
	casos := map[string][]byte{
		"duplicado_raiz":    append([]byte(`{"recuperado":false,`), b[1:]...),
		"duplicado_anidado": bytes.Replace(b, []byte(`"EvaluadaEn":`), []byte(`"EvaluadaEn":null,"EvaluadaEn":`), 1),
		"desconocido":       append([]byte(`{"nuevo":0,`), b[1:]...),
		"trailing":          append(bytes.Clone(b), []byte("{}")...),
		"utf8":              append(bytes.Clone(b[:len(b)-1]), 0xff, '}'),
	}
	for n, raw := range casos {
		t.Run(n, func(t *testing.T) { validar(t, raw) })
	}
	// Límites aislados con JSON bien formado, antes de decodificar dominio.
	for n, raw := range map[string][]byte{
		"tamano":      append(append([]byte(`"`), bytes.Repeat([]byte("a"), MaximoBytesSalidaRegistroIncorporacionV2)...), '"'),
		"profundidad": []byte(strings.Repeat("[", 34) + "0" + strings.Repeat("]", 34)),
		"nodos":       []byte("[" + strings.Repeat("0,", maxNodosTransporteRegistroV2) + "0]"),
	} {
		t.Run(n, func(t *testing.T) {
			if !json.Valid(raw) {
				t.Fatal("fixture sintáctica inválida")
			}
			if validarJSONRegistroV2(context.Background(), raw) == nil {
				t.Fatal("límite no aislado")
			}
		})
	}
}

func TestRegistroIncorporacionV2TransporteHistoriaCruzada(t *testing.T) {
	o, _, now := fixtureTransporte(t)
	w, _ := wireTransporte(t, o, now)
	b := serialTransporte(t, w)
	cambios := map[string]func(*salidaTransporteRegistroV2){
		"esquema":       func(d *salidaTransporteRegistroV2) { d.Historia.Esquema += "x" },
		"evidencia_sha": func(d *salidaTransporteRegistroV2) { d.Historia.Evidencia.SolicitudSHA256 = strings.Repeat("a", 64) },
		"evidencia_concesion": func(d *salidaTransporteRegistroV2) {
			d.Historia.Evidencia.ConcesionRegistradaEn = d.Historia.Evidencia.ConcesionRegistradaEn.Add(time.Microsecond)
		},
		"evidencia_payload": func(d *salidaTransporteRegistroV2) { d.Historia.Evidencia.PayloadVECAD3[0] ^= 1 },
		"recibo_divergente": func(d *salidaTransporteRegistroV2) { d.Historia.Recibo.AuditoriaCTRef += "x" },
		"raiz":              func(d *salidaTransporteRegistroV2) { d.Historia.Anterior.Referencia = registroV2Ref("otra") },
		"publicacion":       func(d *salidaTransporteRegistroV2) { d.Historia.Publicacion.Version++ },
		"estado":            func(d *salidaTransporteRegistroV2) { d.Historia.Posterior.EstadoActual = "anulado" },
		"consumo":           func(d *salidaTransporteRegistroV2) { d.Consumos.DecisionCTRef += "x" },
		"version":           func(d *salidaTransporteRegistroV2) { d.Historia.Anterior.Version++ },
		"historia_resellada": func(d *salidaTransporteRegistroV2) {
			def, e := dom.RestaurarDefinicionSeguimiento(d.Historia.Publicacion)
			registroV2Exigir(t, e)
			a, e := dom.RehidratarSeguimiento(def, d.Historia.Anterior)
			registroV2Exigir(t, e)
			otra := d.Recibo.Copia().Transicion
			otra.Periodo.Hasta = otra.Periodo.Hasta.Add(time.Hour)
			p, e := a.Aplicar(def, a.Version(), otra)
			registroV2Exigir(t, e)
			canon, e := dom.SerializarEstadoSeguimientoCanonico(def, p.Estado())
			registroV2Exigir(t, e)
			d.Historia.Posterior = p.Estado() // historia válida, pero de otra petición
			d.Recibo.HuellaEstadoResultante = registroV2Hash(canon)
			d.Historia.Recibo.HuellaEstadoResultante = d.Recibo.HuellaEstadoResultante
		},
	}
	for n, mutar := range cambios {
		t.Run(n, func(t *testing.T) {
			var d salidaTransporteRegistroV2
			registroV2Exigir(t, json.Unmarshal(b, &d))
			mutar(&d)
			r, e := DecodificarRegistroIncorporacionV2(context.Background(), serialTransporte(t, d), o, o, now)
			if e == nil || !reflect.DeepEqual(r, ct.ResultadoRegistroIncorporacionV2{}) {
				t.Fatal("cruce de evidencia/estado aceptado")
			}
		})
	}
	for n, fecha := range map[string]string{"offset": "2026-09-08T12:00:00+00:00", "hora": "2026-09-08T24:00:00Z", "dia": "2026-02-30T12:00:00Z", "nanos": "2026-09-08T12:00:00.000000001Z"} {
		t.Run(n, func(t *testing.T) {
			v, e := arbolJSONRegistroV2(b)
			registroV2Exigir(t, e)
			v.(map[string]any)["historia"].(map[string]any)["Evidencia"].(map[string]any)["PreparadoEn"] = fecha
			if _, e = DecodificarRegistroIncorporacionV2(context.Background(), serialTransporte(t, v), o, o, now); e == nil {
				t.Fatal("fecha inválida")
			}
		})
	}
}

func TestRegistroIncorporacionV2TransporteCancelacionCero(t *testing.T) {
	o, l, now := fixtureTransporte(t)
	w, _ := wireTransporte(t, o, now)
	b := serialTransporte(t, w)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, c := range []context.Context{nil, ctx} {
		p, e := PrepararLlamadaRegistroIncorporacionV2(c, o, l, registroV2Ref("seguimiento"), now)
		if e == nil || p.Parametros() != nil {
			t.Fatal("cancel/nil preparación")
		}
		r, e := DecodificarRegistroIncorporacionV2(c, b, o, o, now)
		if e == nil || !reflect.DeepEqual(r, ct.ResultadoRegistroIncorporacionV2{}) {
			t.Fatal("cancel/nil decoder")
		}
		if c != nil && !errors.Is(e, context.Canceled) {
			t.Fatal("prioridad cancelación")
		}
	}
	for _, ref := range []string{"", "seguimiento:opaco", "ref:" + strings.Repeat("0", 64), "ref:" + strings.Repeat("A", 64)} {
		if p, e := PrepararLlamadaRegistroIncorporacionV2(context.Background(), o, l, ref, now); e == nil || p.Parametros() != nil {
			t.Fatal("raíz no SQL")
		}
	}
	for _, fecha := range []time.Time{now.Add(-time.Second), now.Add(time.Minute), {}} {
		if p, e := PrepararLlamadaRegistroIncorporacionV2(context.Background(), o, l, registroV2Ref("seguimiento"), fecha); e == nil || p.Parametros() != nil {
			t.Fatal("tiempo no válido")
		}
		if r, e := DecodificarRegistroIncorporacionV2(context.Background(), b, o, o, fecha); e == nil || !reflect.DeepEqual(r, ct.ResultadoRegistroIncorporacionV2{}) {
			t.Fatal("tiempo decoder")
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, e := PrepararLlamadaRegistroIncorporacionV2(context.Background(), o, l, registroV2Ref("seguimiento"), now)
			if e != nil {
				t.Error("preparar concurrente")
				return
			}
			clear(p.Parametros()[0].([]byte))
			r, e := DecodificarRegistroIncorporacionV2(context.Background(), b, o, o, now)
			if e != nil {
				t.Error("decoder concurrente")
				return
			}
			clear(r.Recibo.MaterialOriginalCanonico)
		}()
	}
	wg.Wait()
}
