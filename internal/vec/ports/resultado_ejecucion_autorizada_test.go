package ports

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

func datosInformeEjecucionPrueba() DatosResultadoEjecucionAutorizada {
	return DatosResultadoEjecucionAutorizada{
		InformeRef: "inf_ejec_" + strings.Repeat("a", 32), PerfilConsumidor: ResultadoBorradorDietas,
		DecisionRef: "dec_" + strings.Repeat("b", 32), DecisionHuellaSHA256: strings.Repeat("c", 64),
		ContextoRef: "rca_" + strings.Repeat("d", 32), ContextoHuellaSHA256: strings.Repeat("e", 64),
		ActorRef: "per_" + strings.Repeat("f", 32), PerfilRef: "prf_" + strings.Repeat("g", 32),
		CorrelacionRef: "correlacion_" + strings.Repeat("1", 32),
		Accion:         "dietas.borrador.crear_propio", RecursoRef: "dietas:borrador:" + strings.Repeat("h", 32), RecursoHuellaSHA256: strings.Repeat("2", 64),
		Resultado: "fallo_confirmado", Etapa: "transaccion", Causa: "persistencia",
	}
}
func TestInformeResultadoEjecucionContratosNominales(t *testing.T) {
	for _, c := range []struct {
		perfil          PerfilResultadoEjecucion
		accion, recurso string
	}{
		{ResultadoRutasDietas, "dietas.ruta.catalogo.consultar", "dietas:rutas:catalogo_rutas_dietas"},
		{ResultadoRutasDietas, "dietas.ruta.calculo.solicitar", "dietas:rutas:calculo_rutas_dietas"},
		{ResultadoBorradorDietas, "dietas.borrador.crear_propio", "dietas:borrador:" + strings.Repeat("a", 16)},
		{ResultadoBorradorDietas, "dietas.borrador.recuperar_propio", "dietas:borrador:" + strings.Repeat("b", 128)},
		{ResultadoBorradorDietas, "dietas.borrador.listar_propios", "dietas:borradores:propios"},
		{ResultadoMarcajeCronos, "cronos.marcaje.propio.registrar", "marcaje:cronos:" + strings.Repeat("c", 32)},
		{ResultadoMarcajeCronos, "cronos.marcaje.propio.registrar", "marcaje:cronos:" + strings.Repeat("c", 8)},
	} {
		t.Run(c.accion, func(t *testing.T) {
			d := datosInformeEjecucionPrueba()
			d.PerfilConsumidor, d.Accion, d.RecursoRef = c.perfil, c.accion, c.recurso
			i, err := NuevoInformeResultadoEjecucionAutorizada(d)
			if err != nil {
				t.Fatal(err)
			}
			got, err := i.Datos()
			if err != nil || got != d {
				t.Fatal("contrato alterado")
			}
			d.ActorRef = "otra"
			if got, _ = i.Datos(); got.ActorRef == "otra" {
				t.Fatal("informe mutable")
			}
		})
	}
	if _, err := (InformeResultadoEjecucionAutorizada{}).Datos(); err == nil {
		t.Fatal("zero aceptado")
	}
	d := datosInformeEjecucionPrueba()
	// El generador criptográfico real usa decision:, no el prefijo dec_ de
	// los escenarios de prueba. Ambos siguen siendo referencias opacas.
	d.DecisionRef = "decision:" + strings.Repeat("c", 32)
	if _, err := NuevoInformeResultadoEjecucionAutorizada(d); err != nil {
		t.Fatal("referencia del generador productivo rechazada", err)
	}
}
func TestInformeResultadoEjecucionRechazaCrucesYTextoLibre(t *testing.T) {
	casos := map[string]func(*DatosResultadoEjecucionAutorizada){
		"perfil desconocido":         func(d *DatosResultadoEjecucionAutorizada) { d.PerfilConsumidor = "ct" },
		"perfil cruzado":             func(d *DatosResultadoEjecucionAutorizada) { d.PerfilConsumidor = ResultadoMarcajeCronos },
		"accion cruzada":             func(d *DatosResultadoEjecucionAutorizada) { d.Accion = "cronos.marcaje.propio.registrar" },
		"recurso cruzado":            func(d *DatosResultadoEjecucionAutorizada) { d.RecursoRef = "dietas:borradores:propios" },
		"actor DNI":                  func(d *DatosResultadoEjecucionAutorizada) { d.ActorRef = "12345678Z" },
		"perfil libre":               func(d *DatosResultadoEjecucionAutorizada) { d.PerfilRef = "administrador" },
		"contexto vacio":             func(d *DatosResultadoEjecucionAutorizada) { d.ContextoRef = "" },
		"decision libre":             func(d *DatosResultadoEjecucionAutorizada) { d.DecisionRef = "nombre apellido" },
		"correlacion correo":         func(d *DatosResultadoEjecucionAutorizada) { d.CorrelacionRef = "persona@example.invalid" },
		"informe control":            func(d *DatosResultadoEjecucionAutorizada) { d.InformeRef += "\n" },
		"huella decision":            func(d *DatosResultadoEjecucionAutorizada) { d.DecisionHuellaSHA256 = strings.Repeat("0", 64) },
		"huella contexto":            func(d *DatosResultadoEjecucionAutorizada) { d.ContextoHuellaSHA256 = strings.Repeat("z", 64) },
		"huella recurso":             func(d *DatosResultadoEjecucionAutorizada) { d.RecursoHuellaSHA256 += "\x00" },
		"denegacion":                 func(d *DatosResultadoEjecucionAutorizada) { d.Resultado = "denegado" },
		"exito":                      func(d *DatosResultadoEjecucionAutorizada) { d.Resultado = "confirmado" },
		"causa SQL":                  func(d *DatosResultadoEjecucionAutorizada) { d.Causa = "pq error con payload" },
		"etapa arbitraria":           func(d *DatosResultadoEjecucionAutorizada) { d.Etapa = "envio" },
		"commit ambiguo falso fallo": func(d *DatosResultadoEjecucionAutorizada) { d.Etapa = "commit"; d.Causa = "commit_no_confirmado" },
		"rollback no prueba fallo":   func(d *DatosResultadoEjecucionAutorizada) { d.Etapa = "rollback"; d.Causa = "rollback_no_confirmado" },
	}
	for n, f := range casos {
		t.Run(n, func(t *testing.T) {
			d := datosInformeEjecucionPrueba()
			f(&d)
			if _, err := NuevoInformeResultadoEjecucionAutorizada(d); err == nil {
				t.Fatal("entrada inválida aceptada")
			}
		})
	}
}
func TestInformeResultadoEjecucionEtapasYAmbiguedad(t *testing.T) {
	for _, c := range []struct{ resultado, etapa, causa string }{
		{"fallo_confirmado", "preparacion", "material_no_disponible"},
		{"fallo_confirmado", "preparacion", "material_invalido"},
		{"fallo_confirmado", "preparacion", "contexto_cancelado"},
		{"fallo_confirmado", "transaccion", "conflicto"},
		{"fallo_confirmado", "transaccion", "recibo_invalido"},
		{"fallo_confirmado", "transaccion", "contexto_cancelado"},
		{"fallo_confirmado", "transaccion", "consumo_rechazado"},
		{"resultado_indeterminado", "rollback", "rollback_no_confirmado"},
		{"resultado_indeterminado", "commit", "commit_no_confirmado"},
		{"fallo_confirmado", "commit", "commit_revertido"},
		{"resultado_indeterminado", "entrega", "respuesta_no_confirmada"},
		{"resultado_indeterminado", "entrega", "recibo_invalido"},
	} {
		d := datosInformeEjecucionPrueba()
		d.Resultado, d.Etapa, d.Causa = c.resultado, c.etapa, c.causa
		if _, err := NuevoInformeResultadoEjecucionAutorizada(d); err != nil {
			t.Fatalf("%s/%s: %v", c.etapa, c.causa, err)
		}
	}
}
func TestInformeResultadoEjecucionRedactaLogs(t *testing.T) {
	d := datosInformeEjecucionPrueba()
	i, _ := NuevoInformeResultadoEjecucionAutorizada(d)
	r := ReciboResultadoEjecucionAutorizada{InformeRef: d.InformeRef, ReciboRef: "rec_ejec_" + strings.Repeat("9", 32), HuellaSHA256: d.DecisionHuellaSHA256}
	var b bytes.Buffer
	slog.New(slog.NewJSONHandler(&b, nil)).Info("prueba", "datos", d, "informe", i, "recibo", r)
	salida := b.String() + fmt.Sprintf("%v %+v %#v %v %+v %#v %v %+v %#v", d, d, d, i, i, i, r, r, r)
	for _, secreto := range []string{d.InformeRef, d.ActorRef, d.ContextoRef, d.RecursoRef, d.DecisionHuellaSHA256, r.ReciboRef} {
		if strings.Contains(salida, secreto) {
			t.Fatal("referencia filtrada al log")
		}
	}
}
