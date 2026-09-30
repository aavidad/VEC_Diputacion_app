package incorporacionejercicio

import (
	"context"
	"reflect"
	"testing"

	puente "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/personalincorporacion"
	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	lector "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
	core "vec-diputacion-granada/internal/vec/domain"
)

// Reproduce el cambio de perfil tras confirmar Personal y antes de CT.
// Los dobles sólo observan que la alta histórica no se vuelve a emitir.
func TestRecuperacionIncorporacionNoReemiteAltaConOtroPerfil(t *testing.T) {
	c := nuevoCasoPreparacionV2(t)
	_, err := c.app.s.Confirmar(context.Background(), c.app.i)
	registroV2Exigir(t, err)
	datos, err := c.app.ctTX.original.Material().Datos()
	registroV2Exigir(t, err)
	c.parcial = true
	c.app.ctTX.efectos = 0 // CT quedó pendiente; Personal conserva su alta.
	c.app.ctTX.original = ct.OrdenConfirmacionIncorporacionV2{}
	c.app.ctTX.persistido = ct.ResultadoRegistroIncorporacionV2{}
	c.app.reloj.ahora = c.a.ahora
	lectorActual, err := lector.NuevoV2(permisoLecturaApp(func(ctx context.Context, m lector.MaterialV2) (lector.AutorizacionV2, error) {
		contexto, err := m.Contexto()
		registroV2Exigir(t, err)
		datos.Contexto = contexto
		datos.Personal = c.app.alta.registro(t)
		datos.Confirmacion.ResultadoPersonal = datos.Personal.Resultado
		material, err := ct.NuevoMaterialConfirmacionIncorporacionV2(datos, c.app.reloj.ahora)
		registroV2Exigir(t, err)
		recurso, err := m.Recurso()
		registroV2Exigir(t, err)
		a := registroV2Autoridad(t, material, c.app.reloj.ahora, func(x *core.DatosSolicitudAutorizacionLigadaV3) {
			x.Recurso, x.Accion, x.Finalidad = recurso, lector.Accion, lector.Finalidad
		}, lector.AudienciaV2)
		return lector.AutorizacionV2{Solicitud: a.Solicitud, Decision: a.Decision, Confirmacion: a.Confirmacion, Exportacion: a.Exportacion}, nil
	}), txLecturaApp(func(ctx context.Context, sel lector.Selector, o lector.OrdenV2) (lector.Resultado, error) {
		r := c.app.alta.registro(t)
		return lector.Resultado{Registro: r, DecisionLecturaRef: o.Exportacion().ResumenCapacidad().DecisionRef(), ConsumoHuellaSHA256: registroV2Hash([]byte("lectura actual")),
			AuditoriaLecturaRef: "auditoria:lectura:actual", LeidaEn: c.app.reloj.ahora}, nil
	}), c.app.reloj)
	registroV2Exigir(t, err)
	puenteActual, err := puente.NuevoV2(lectorActual, c.app.reloj)
	registroV2Exigir(t, err)
	confirmadorActual, err := appct.NuevoServicioConfirmacionIncorporacionV2(&registroV2Proveedor{t: t, reloj: c.app.reloj}, puenteActual, c.app.ctTX, c.app.reloj)
	registroV2Exigir(t, err)
	altas := c.app.alta.n
	s, err := Nuevo(Configuracion{Preparador: c.p, FuentePersonal: c.app.s.c.FuentePersonal, TernaPersonal: c.app.s.c.TernaPersonal,
		ProveedorAlta: c.app.s.c.ProveedorAlta, TransaccionAlta: c.app.s.c.TransaccionAlta,
		LectorPersonal: lectorActual, Confirmador: confirmadorActual, Reloj: c.app.s.c.Reloj})
	registroV2Exigir(t, err)
	r, err := s.Confirmar(context.Background(), c.app.i)
	registroV2Exigir(t, err)
	if r.ExpedienteRef != c.app.i.ExpedienteRef || c.app.alta.n != altas || c.app.ctTX.efectos != 1 {
		t.Fatal("la recuperación duplicó Personal o no confirmó CT")
	}
	c.confirmada = true
	altas, ctLlamadas := c.app.alta.n, c.app.ctTX.llamadas
	repetido, err := s.Confirmar(context.Background(), c.app.i)
	registroV2Exigir(t, err)
	if !reflect.DeepEqual(r, repetido) || c.app.alta.n != altas || c.app.ctTX.llamadas != ctLlamadas {
		t.Fatal("el replay CT reemitió efectos o alteró el recibo")
	}
}
