package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	cronoshttp "vec-diputacion-granada/internal/modules/cronos/adapters/httpinterno"
	cronosapp "vec-diputacion-granada/internal/modules/cronos/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type registroIntentosCronosPrueba struct {
	ordenes  []vecports.OrdenIntentoAuditoria
	err      error
	sinAcuse bool
}

type validadorMotivoDenegadoCronosPrueba struct {
	llamadas int
	err      error
}

func (v *validadorMotivoDenegadoCronosPrueba) ValidarReferenciaMotivoAutorizacionV2(_ context.Context, _ core.ReferenciaEntradaCatalogo, _ time.Time) error {
	v.llamadas++
	return v.err
}

func TestCronosMotivoDenegadoExigeCatalogoPublicado(t *testing.T) {
	motivo := core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 2,
		CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_22222222222222222222222222222222"}
	validador := &validadorMotivoDenegadoCronosPrueba{}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	if err := validarMotivoDenegadoCronos(context.Background(), validador, motivo, motivo.CatalogoID, ahora); err != nil || validador.llamadas != 1 {
		t.Fatal("motivo publicado no habilitó Cronos", err)
	}
	caducado := errors.New("motivo no publicado o caducado")
	validador.err = caducado
	if err := validarMotivoDenegadoCronos(context.Background(), validador, motivo, motivo.CatalogoID, ahora); !errors.Is(err, caducado) || validador.llamadas != 2 {
		t.Fatal("motivo caducado habilitó Cronos", err)
	}
	if err := validarMotivoDenegadoCronos(context.Background(), validador, motivo, "otro_catalogo", ahora); err == nil || validador.llamadas != 2 {
		t.Fatal("motivo de otro catálogo llegó al validador")
	}
}

func (r *registroIntentosCronosPrueba) AppendIntentoAuditoria(_ context.Context, orden vecports.OrdenIntentoAuditoria) (vecports.AcuseIntentoAuditoria, error) {
	r.ordenes = append(r.ordenes, orden)
	if r.err != nil {
		return vecports.AcuseIntentoAuditoria{}, r.err
	}
	if r.sinAcuse {
		return vecports.AcuseIntentoAuditoria{}, nil
	}
	datos, err := orden.Datos()
	if err != nil {
		return vecports.AcuseIntentoAuditoria{}, err
	}
	return vecports.AcuseIntentoAuditoria{AuditoriaRef: "auditoria_cronos_prueba", Secuencia: 1,
		HuellaSHA256: strings.Repeat("a", 64), CorrelacionRef: datos.Datos.CorrelacionRef,
		RegistradaEn: time.Now().UTC().Truncate(time.Microsecond)}, nil
}

func identidadAuditoriaCronosPrueba(t *testing.T) contextoSeguridadComunDesarrollo {
	t.Helper()
	escenario := nuevoEscenarioMaterialRutasDietasPrueba(t, "dietas.ruta.catalogo.consultar")
	resultado := resultadoConVinculoEmpleadoF1(t, escenario.resultado)
	datosSolicitud, err := escenario.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datosVinculo, err := datosSolicitud.VinculoAutenticacionActor.Datos()
	if err != nil {
		t.Fatal(err)
	}
	actor := resultado.Contexto
	cuenta := core.CuentaAutenticadaContextoActor{CuentaRef: actor.Instantanea.CuentaRef,
		Metodo: actor.Principal.AuthMethod, Garantia: actor.Principal.AuthAssurance}
	vinculo, err := core.CrearVinculoAutenticacionActorV2(context.Background(),
		revalidadorMaterialRutasDietasPrueba{resultado: datosVinculo.Autenticacion()},
		core.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: datosVinculo.AutenticacionRef, SesionRef: datosVinculo.SesionRef},
		resolutorMaterialRutasDietasPrueba{resultado: resultado},
		core.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: actor.PerfilActivoRef},
		&relojMaterialRutasDietasPrueba{ahora: escenario.ahora})
	if err != nil || vinculo.ValidarPara(resultado) != nil {
		t.Fatal("identidad nominal de prueba no válida", err)
	}
	return contextoSeguridadComunDesarrollo{Resultado: resultado, Vinculo: vinculo}
}

func TestCronosIntentoComunDenegadoUsaIdentidadHistoricaYAcuseUnico(t *testing.T) {
	holder := identidadAuditoriaCronosPrueba(t)
	destino := &registroIntentosCronosPrueba{}
	motivo := core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 2,
		CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_22222222222222222222222222222222"}
	a := &autoridadCronosEmpleadoDesarrollo{auditoriaIntentos: destino, procesoIntentos: "cronos_portal", motivoDenegado: motivo}
	registro, err := nuevoRegistroDenegacionVinculoCronos(a, holder, cronoshttp.RutaConsultarMovimientosPropios)
	if err != nil || registro == nil {
		t.Fatal("registro común Cronos no compuesto", err)
	}
	actor := holder.Resultado.Contexto
	if err := registro.RegistrarDenegacionVinculo(context.Background(), actor); err != nil {
		t.Fatal(err)
	}
	if err := registro.RegistrarDenegacionVinculo(context.Background(), actor); err != nil || len(destino.ordenes) != 1 {
		t.Fatal("duplicó el intento confirmado", err, len(destino.ordenes))
	}
	datos, err := destino.ordenes[0].Datos()
	if err != nil || datos.Datos.ModuloID != "cronos" || datos.Datos.Accion != cronosapp.AccionConsultarMovimientosPropios ||
		datos.Datos.FinalidadRef != cronosapp.FinalidadConsultarMovimientosPropios || datos.Datos.Resultado != core.ResultadoIntentoAuditoriaDenegado ||
		datos.Datos.Motivo != motivo || strings.Contains(datos.Datos.RecursoRef, "emp_") || datos.ResultadoContexto.Contexto.PersonaRef != actor.PersonaRef {
		t.Fatal("intento común sin alcance o identidad exactos", err)
	}
	if registro.RegistrarDenegacionVinculo(context.Background(), core.ContextoActor{}) == nil || len(destino.ordenes) != 1 {
		t.Fatal("actor no acreditado creó otro intento")
	}
	cruzado, err := actor.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	cruzado.Instantanea.Vinculos[0].Referencia = "emp_" + strings.Repeat("b", 32)
	if cruzado.Validar() != nil || registro.RegistrarDenegacionVinculo(context.Background(), cruzado) == nil || len(destino.ordenes) != 1 {
		t.Fatal("otro empleado con mismo actor/perfil/instante llegó a la auditoría")
	}
	falloComun := errors.New("sin auditoría común")
	fallido := &registroIntentosCronosPrueba{err: falloComun}
	a.auditoriaIntentos = fallido
	registro, err = nuevoRegistroDenegacionVinculoCronos(a, holder, cronoshttp.RutaSolicitarCorreccionPropia)
	if err != nil || registro == nil || !errors.Is(registro.RegistrarDenegacionVinculo(context.Background(), actor), falloComun) || len(fallido.ordenes) != 2 {
		t.Fatal("fallo de auditoría común no cerró la denegación", err)
	}
	primera, _ := fallido.ordenes[0].Datos()
	segunda, _ := fallido.ordenes[1].Datos()
	if primera.IntentoRef != segunda.IntentoRef || primera.Datos != segunda.Datos || primera.Datos.Accion != cronosapp.AccionSolicitarCorreccion {
		t.Fatal("reintento de COMMIT usó otra operación")
	}
	sinAcuse := &registroIntentosCronosPrueba{sinAcuse: true}
	a.auditoriaIntentos = sinAcuse
	registro, err = nuevoRegistroDenegacionVinculoCronos(a, holder, cronoshttp.RutaConsultarMovimientosPropios)
	if err != nil || registro == nil || !errors.Is(registro.RegistrarDenegacionVinculo(context.Background(), actor), vecports.ErrAcuseIntentoAuditoriaInvalido) || len(sinAcuse.ordenes) != 2 {
		t.Fatal("acuse inválido se tomó por COMMIT confirmado", err)
	}
}
