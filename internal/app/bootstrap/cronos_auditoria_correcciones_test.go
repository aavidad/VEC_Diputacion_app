package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"

	cronoshttp "vec-diputacion-granada/internal/modules/cronos/adapters/httpinterno"
	cronosapp "vec-diputacion-granada/internal/modules/cronos/application"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestDenegacionesCorreccionesCronosUsanContratoNominalYAcuse(t *testing.T) {
	holder := identidadAuditoriaCronosPrueba(t)
	actor := holder.Resultado.Contexto
	motivo := core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 2,
		CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_22222222222222222222222222222222"}
	casos := []struct{ ruta, accion, finalidad string }{
		{cronoshttp.RutaBandejaCorrecciones, cronosapp.AccionBandejaCorrecciones, cronosapp.FinalidadBandejaCorrecciones},
		{cronoshttp.RutaActuacionCorrecciones, cronosapp.AccionDecidirCorreccion, cronosapp.FinalidadDecisionResponsableCorreccion},
		{cronoshttp.RutaActuacionCorrecciones, cronosapp.AccionResolverCorreccionRRHH, cronosapp.FinalidadResolucionRRHHCorreccion},
		{cronoshttp.RutaActuacionCorrecciones, cronosapp.AccionAplicarCorreccion, cronosapp.FinalidadAplicacionCorreccion},
		{cronoshttp.RutaHistoriaCorrecciones, cronosapp.AccionHistoriaCorreccionPropia, cronosapp.FinalidadHistoriaCorreccionPropia},
		{cronoshttp.RutaHistoriaCorrecciones, cronosapp.AccionHistoriaCorreccionEquipo, cronosapp.FinalidadHistoriaCorreccionEquipo},
	}
	for _, caso := range casos {
		t.Run(caso.accion, func(t *testing.T) {
			destino := &registroIntentosCronosPrueba{}
			a := &autoridadCronosEmpleadoDesarrollo{auditoriaIntentos: destino, procesoIntentos: "cronos_portal", motivoDenegado: motivo}
			registro, err := nuevoRegistroDenegacionVinculoCronos(a, holder, caso.ruta)
			if err != nil || registro == nil {
				t.Fatal("ruta sin registro nominal", err)
			}
			if err := registro.RegistrarDenegacionVinculo(context.Background(), actor); err == nil || len(destino.ordenes) != 0 {
				t.Fatal("denegación C5b aceptó acción implícita")
			}
			if err := registro.RegistrarDenegacionCorreccion(context.Background(), actor, caso.accion, caso.finalidad); err != nil {
				t.Fatal("denegación nominal no confirmó acuse", err)
			}
			if err := registro.RegistrarDenegacionCorreccion(context.Background(), actor, caso.accion, caso.finalidad); err != nil || len(destino.ordenes) != 1 {
				t.Fatal("denegación repetida duplicó acuse", err)
			}
			datos, err := destino.ordenes[0].Datos()
			if err != nil || datos.Datos.Accion != caso.accion || datos.Datos.FinalidadRef != caso.finalidad ||
				datos.Datos.Resultado != core.ResultadoIntentoAuditoriaDenegado || strings.Contains(datos.Datos.RecursoRef, "emp_") {
				t.Fatal("acuse con acción, finalidad o recurso incorrectos", err)
			}
			if err := registro.RegistrarDenegacionCorreccion(context.Background(), actor,
				cronosapp.AccionHistoriaCorreccionEquipo, cronosapp.FinalidadHistoriaCorreccionEquipo); caso.accion != cronosapp.AccionHistoriaCorreccionEquipo && err == nil {
				t.Fatal("ruta o acuse aceptó otro contrato")
			}
		})
	}
}

func TestDenegacionCorreccionSinAcuseNoPermite403(t *testing.T) {
	holder := identidadAuditoriaCronosPrueba(t)
	motivo := core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 2,
		CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_22222222222222222222222222222222"}
	destino := &registroIntentosCronosPrueba{err: errors.New("sin acuse")}
	a := &autoridadCronosEmpleadoDesarrollo{auditoriaIntentos: destino, procesoIntentos: "cronos_portal", motivoDenegado: motivo}
	registro, err := nuevoRegistroDenegacionVinculoCronos(a, holder, cronoshttp.RutaHistoriaCorrecciones)
	if err != nil || registro == nil {
		t.Fatal("ruta historia sin registro", err)
	}
	if err := registro.RegistrarDenegacionCorreccion(context.Background(), holder.Resultado.Contexto,
		cronosapp.AccionHistoriaCorreccionPropia, cronosapp.FinalidadHistoriaCorreccionPropia); err == nil || len(destino.ordenes) != 2 {
		t.Fatal("sin acuse, denegación quedó confirmada")
	}
	if err := registro.RegistrarDenegacionCorreccion(context.Background(), holder.Resultado.Contexto,
		cronosapp.AccionHistoriaCorreccionEquipo, cronosapp.FinalidadHistoriaCorreccionEquipo); err == nil || len(destino.ordenes) != 2 {
		t.Fatal("reintento cambió el contrato de la misma petición")
	}
}
