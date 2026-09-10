package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"testing"
	"time"

	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestRaizInicialV2DesdeOriginalPersonal(t *testing.T) {
	o, fila, ahora := fixtureRaicesV2(t)
	var pub dom.PublicacionDefinicionSeguimiento
	registroV2Exigir(t, json.Unmarshal(fila.publicacion, &pub))
	def, err := dom.RestaurarDefinicionSeguimiento(pub)
	registroV2Exigir(t, err)
	d, err := o.Material().Datos()
	registroV2Exigir(t, err)
	// El fixture legado tiene Personal de antes de la vigencia de definición:
	// no se permite inventar una fecha posterior para hacer encajar esa raíz.
	if _, err := NuevaPreparacionRaizIncorporacionV2(def, fila.referencia, d.Confirmacion.PeriodoIncorporacion, o, ahora); err == nil {
		t.Fatal("original anterior a vigencia aceptado")
	}
	d.Personal.RegistradoEn = ahora.Add(-time.Minute)
	o = ordenCTTransporte(t, d, ahora)
	r, err := NuevaPreparacionRaizIncorporacionV2(def, fila.referencia, d.Confirmacion.PeriodoIncorporacion, o, ahora)
	registroV2Exigir(t, err)
	var estado dom.EstadoPersistidoSeguimiento
	registroV2Exigir(t, json.Unmarshal(r.estado, &estado))
	if estado.RelacionRef != d.Personal.Resultado.RelacionRef || estado.ExpedienteRef != d.Personal.Solicitud.ExpedienteRef ||
		estado.OrganizacionRef != d.Preparacion.OrganizacionRef || !estado.CreadoEn.Equal(d.Personal.RegistradoEn) ||
		!estado.ActualizadoEn.Equal(d.Personal.RegistradoEn) || estado.Version != 0 || len(estado.Actuaciones) != 0 || r.version != 8 {
		t.Fatal("raíz anticipada, no ligada al original")
	}
	_, err = dom.RehidratarSeguimiento(def, estado)
	registroV2Exigir(t, err)
	// Nuevo permiso actual, mismo original y misma raíz técnica: no fecha Now.
	reintento := ordenCTTransporte(t, d, ahora.Add(time.Second))
	repetida, err := NuevaPreparacionRaizIncorporacionV2(def, fila.referencia, d.Confirmacion.PeriodoIncorporacion, reintento, ahora.Add(time.Second))
	registroV2Exigir(t, err)
	if !bytes.Equal(r.estado, repetida.estado) {
		t.Fatal("reintento cambió raíz original")
	}
	evidencia, err := ct.CapturarEvidenciaOrdenOriginalIncorporacionV2(context.Background(), o)
	registroV2Exigir(t, err)
	original := serialTransporte(t, evidencia)
	sobre, err := r.envolverEvidencia(original)
	registroV2Exigir(t, err)
	var wire struct {
		Esquema string          `json:"esquema"`
		Orden   json.RawMessage `json:"orden"`
		Raiz    struct {
			Estado  json.RawMessage `json:"estado"`
			Version uint64          `json:"version_expediente"`
		} `json:"raiz_inicial"`
	}
	registroV2Exigir(t, json.Unmarshal(sobre, &wire))
	if wire.Esquema != "vec.ct.incorporacion.raiz-inicial.v1" || !bytes.Equal(wire.Orden, original) || !bytes.Equal(wire.Raiz.Estado, r.estado) || wire.Raiz.Version != 8 {
		t.Fatal("evidencia original modificada")
	}
	legacy, err := RaizIncorporacionV2Existente(fila.referencia)
	registroV2Exigir(t, err)
	sinSobre, err := legacy.envolverEvidencia(original)
	registroV2Exigir(t, err)
	if !reflect.DeepEqual(sinSobre, original) {
		t.Fatal("contrato legacy alterado")
	}
	cruzado := d.Confirmacion.PeriodoIncorporacion
	cruzado.Hasta = cruzado.Hasta.Add(time.Hour)
	if _, err := NuevaPreparacionRaizIncorporacionV2(def, fila.referencia, cruzado, o, ahora); err == nil {
		t.Fatal("período ajeno aceptado")
	}
	if _, err := NuevaPreparacionRaizIncorporacionV2(def, fila.referencia, d.Confirmacion.PeriodoIncorporacion, ct.OrdenConfirmacionIncorporacionV2{}, ahora); err == nil {
		t.Fatal("orden no acreditada aceptada")
	}
}

type raizInicialRegistroTXPrueba struct{ PreparacionRaizIncorporacionV2 }

func (r raizInicialRegistroTXPrueba) ResolverSeguimientoIncorporacionV2(context.Context, ct.OrdenConfirmacionIncorporacionV2) (string, error) {
	panic("no consultar CT78 para una relación recién creada")
}
func (r raizInicialRegistroTXPrueba) ResolverRaizInicialIncorporacionV2(context.Context, ct.OrdenConfirmacionIncorporacionV2) (PreparacionRaizIncorporacionV2, error) {
	return r.PreparacionRaizIncorporacionV2, nil
}

func TestRaizInicialV2MismaTransaccion(t *testing.T) {
	for _, falla := range []bool{false, true} {
		t.Run(map[bool]string{false: "commit", true: "rollback"}[falla], func(t *testing.T) {
			m, a, _ := setupRegistroTX(t)
			d, err := m.o.Material().Datos()
			registroV2Exigir(t, err)
			d.Personal.RegistradoEn = m.now.Add(-time.Minute)
			m.o = ordenCTTransporte(t, d, m.now)
			m.wire, _ = wireTransporte(t, m.o, m.now)
			def, err := dom.RestaurarDefinicionSeguimiento(m.wire.Historia.Publicacion)
			registroV2Exigir(t, err)
			raiz, err := NuevaPreparacionRaizIncorporacionV2(def, m.wire.Recibo.SeguimientoRef, d.Confirmacion.PeriodoIncorporacion, m.o, m.now)
			registroV2Exigir(t, err)
			a.raices = raizInicialRegistroTXPrueba{raiz}
			ac, err := ct.AcreditarPersonalParaIncorporacion(context.Background(), m.o, acreditadorRegistroTXPrueba(func(context.Context, ct.OrdenConfirmacionIncorporacionV2) (ct.RegistroPersonalEjercicio, error) {
				return d.Personal, nil
			}), a.reloj)
			registroV2Exigir(t, err)
			m.hook = func(paso string) {
				if paso == "query" {
					// Mismo QueryRow23, sólo su argumento evidencia lleva candidato.
					m.params[22], err = raiz.envolverEvidencia(m.params[22].([]byte))
					registroV2Exigir(t, err)
				}
			}
			if falla {
				m.fail = "scan"
			}
			r, err := a.RegistrarORecuperarIncorporacion(context.Background(), m.o, ac)
			if falla {
				if err == nil || slices.Contains(m.pasos, "commit") || !m.rollbackSeguro {
					t.Fatal("error de registro no revirtió la misma TX")
				}
			} else if err != nil || r.ValidarPara(m.o, m.now) != nil || !slices.Contains(m.pasos, "commit") || slices.Contains(m.pasos, "rollback") {
				t.Fatal("registro con raíz inicial no confirmó", err)
			}
		})
	}
}
