package httpseguridad

import (
	"errors"
	"testing"
)

func TestCapsulaPresentacionConservaModoSelladoAlVincularYLeer(t *testing.T) {
	servicio, verificador, _, registro, reloj, _ := entornoPresentacionCertificadoPrueba(t)
	estadoInicio, estadoReanudacion := estadosTLSCertificadoCompartidoPresentacion(
		t, sanDNSConfigurado(servicio.identidad.configuracion),
	)
	almacen, err := NuevoRegistroSesionMemoriaCertificado(reloj, politicaSesionMemoriaPrueba(t, 2))
	if err != nil {
		t.Fatal(err)
	}

	ctxInicio, credencialInicio, pruebaInicio := credencialPresentacionCertificadoPrueba(t, servicio, verificador, estadoInicio)
	inicio, err := servicio.IniciarYConsumirPresentacion(ctxInicio, credencialInicio, pruebaInicio)
	if err != nil || inicio.datos == nil || !inicio.datos.inicioExplicito {
		t.Fatalf("inicio no sellado: %v", err)
	}
	ctxInicioVinculado, err := servicio.VincularCapsulaPresentacion(ctxInicio, inicio, credencialInicio.canal)
	if err != nil {
		t.Fatalf("inicio no vinculó: %v", err)
	}
	cuentaInicio, auditoriaInicio, err := servicio.identidad.ExtraerCapsulaIdentidadPeticion(ctxInicioVinculado)
	if err != nil || cuentaInicio.CuentaRef != inicio.datos.resultado.SesionOriginal.CuentaRef ||
		auditoriaInicio.PresentacionOperacionRef() != inicio.datos.resultado.Recibo.OperacionRef {
		t.Fatalf("inicio no entregó cuenta y auditoría: %v", err)
	}
	token, _, err := almacen.EmitirDesdeInicio(ctxInicioVinculado, servicio, pruebaInicio, credencialInicio.canal)
	if err != nil || token == "" {
		t.Fatalf("inicio no emitió token: %v", err)
	}

	registro.original = inicio.datos.resultado.SesionOriginal
	registro.modo = "reanudada"
	ctxReanudacion, credencialReanudacion, pruebaReanudacion := credencialPresentacionCertificadoPrueba(
		t, servicio, verificador, estadoReanudacion,
	)
	actual := verificador.asercion
	actual.ID = "asercion-capsula-modo-reanudacion"
	verificador.fijarAsercion(actual)
	credencialReanudacion = debeCredencial(t, []byte("asercion-capsula-modo-reanudacion-protegida"), credencialReanudacion.canal)
	vinculo, err := almacen.VerificarActual(ctxReanudacion, token, servicio, pruebaReanudacion, credencialReanudacion.canal)
	if err != nil {
		t.Fatalf("token de inicio no verificó en la nueva petición: %v", err)
	}
	reanudacion, err := servicio.ReanudarYConsumirPresentacion(ctxReanudacion, credencialReanudacion, pruebaReanudacion)
	if err != nil || reanudacion.datos == nil || reanudacion.datos.inicioExplicito ||
		reanudacion.datos.resultado.Recibo.ModoInicio != "reanudada" {
		t.Fatalf("reanudación no sellada: %v", err)
	}
	if pruebaInicio.datos.certificadoSHA256 != pruebaReanudacion.datos.certificadoSHA256 ||
		credencialInicio.canal.ReferenciaVinculacion() == credencialReanudacion.canal.ReferenciaVinculacion() {
		t.Fatal("escenario no separa las dos conexiones del mismo certificado")
	}
	if _, err := servicio.VincularCapsulaPresentacion(ctxInicio, reanudacion, credencialInicio.canal); !errors.Is(err, ErrPresentacionCertificadoNoValida) {
		t.Fatalf("otra petición o canal vinculó cápsula: %v", err)
	}
	if _, err := servicio.VincularCapsulaPresentacion(ctxReanudacion, reanudacion, credencialInicio.canal); !errors.Is(err, ErrPresentacionCertificadoNoValida) {
		t.Fatalf("otro canal vinculó cápsula: %v", err)
	}
	ctxAjeno, credencialAjena, pruebaAjena := credencialPresentacionCertificadoPrueba(
		t, servicio, verificador, estadoTLSMutuoReal(t, sanDNSConfigurado(servicio.identidad.configuracion)),
	)
	if pruebaReanudacion.datos.certificadoSHA256 == pruebaAjena.datos.certificadoSHA256 {
		t.Fatal("escenario no usa otro certificado")
	}
	if _, err := servicio.VincularCapsulaPresentacion(ctxAjeno, reanudacion, credencialAjena.canal); !errors.Is(err, ErrPresentacionCertificadoNoValida) {
		t.Fatalf("otro certificado vinculó cápsula: %v", err)
	}

	reanudacion.datos.resultado.Recibo.ModoInicio = "abierta"
	if _, err := servicio.VincularCapsulaPresentacion(ctxReanudacion, reanudacion, credencialReanudacion.canal); !errors.Is(err, ErrPresentacionCertificadoNoValida) {
		t.Fatalf("resultado contrario al modo sellado vinculó: %v", err)
	}
	reanudacion.datos.resultado.Recibo.ModoInicio = "reanudada"
	ctxVinculado, err := servicio.VincularCapsulaPresentacion(ctxReanudacion, reanudacion, credencialReanudacion.canal)
	if err != nil {
		t.Fatalf("GET confirmado no vinculó: %v", err)
	}
	if err := almacen.ConfirmarReanudacion(ctxVinculado, vinculo); err != nil {
		t.Fatalf("GET confirmado no cotejó token: %v", err)
	}
	cuenta, auditoria, err := servicio.identidad.ExtraerCapsulaIdentidadPeticion(ctxVinculado)
	if err != nil || cuenta.CuentaRef != cuentaInicio.CuentaRef ||
		auditoria.SesionRef() != auditoriaInicio.SesionRef() ||
		auditoria.PresentacionOperacionRef() != reanudacion.datos.resultado.Recibo.OperacionRef ||
		auditoria.CanalVinculadoRef() != credencialReanudacion.canal.ReferenciaVinculacion() {
		t.Fatalf("GET confirmado no entregó cuenta y auditoría propias: %v", err)
	}
	if tokenNuevo, _, err := almacen.EmitirDesdeInicio(ctxVinculado, servicio, pruebaReanudacion, credencialReanudacion.canal); err == nil || tokenNuevo != "" {
		t.Fatalf("GET reanudado emitió token de inicio: %v", err)
	}
	if _, err := servicio.VincularCapsulaPresentacion(ctxReanudacion, reanudacion, credencialReanudacion.canal); !errors.Is(err, ErrPresentacionCertificadoNoValida) {
		t.Fatalf("replay de cápsula vinculado: %v", err)
	}
	reanudacion.datos.resultado.Recibo.ModoInicio = "abierta"
	if _, _, err := servicio.identidad.ExtraerCapsulaIdentidadPeticion(ctxVinculado); !errors.Is(err, ErrPresentacionCertificadoNoValida) {
		t.Fatalf("lectura aceptó resultado contrario al modo sellado: %v", err)
	}
}
