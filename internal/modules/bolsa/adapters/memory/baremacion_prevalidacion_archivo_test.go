package memory

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestHuellasV3PrevalidacionYConfirmacionTienenDominiosEstables(t *testing.T) {
	escenario := nuevoEscenarioManifiestoHistoricoPrueba(t, verificadorHMACMemoriaPrueba{})
	solicitud := escenario.confirmar
	token, err := puertosbolsa.NuevoTokenReservaBaremacion(
		base64.RawURLEncoding.EncodeToString([]byte("abcdefghijklmnopqrstuvwx")),
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud.Token = token
	solicitud = sellarConfirmacionMemoria(solicitud)
	huellaConfirmacion, err := huellaEfectoConfirmacion(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	huellaPrevalidacion, err := huellaEfectoPrevalidacionArchivo(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	const vectorConfirmacion = "85367bef5b3746b001b226829963b74a18e7958e00a0ab4e83d783c806352de0"
	const vectorPrevalidacion = "3817f2cfa437515d01f954456ad1af8b57d42483ed2891df021dd758d9c6171c"
	if huellaConfirmacion != vectorConfirmacion || huellaPrevalidacion != vectorPrevalidacion ||
		huellaConfirmacion == huellaPrevalidacion {
		t.Fatalf("vectores de efecto inesperados: confirmacion=%s prevalidacion=%s",
			huellaConfirmacion, huellaPrevalidacion)
	}
}

func contextoPrevalidacionAlternativoPrueba(
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
	accion puertosbolsa.AccionOperacionBaremacion,
	recurso, principal, sujeto, perfil, finalidad, autorizacion string,
	instante time.Time,
) puertosbolsa.ContextoOperacionBaremacion {
	p := solicitud.ContextoPrevalidacionArchivo.Proyeccion()
	return contextoMemoriaPruebaAutorizacionCompleta(
		accion, recurso, principal, sujeto, perfil, p.AutenticacionRef,
		p.CorrelacionRef, autorizacion, finalidad, instante,
	)
}

func TestSolicitudConfirmacionRechazaPrevalidacionConfundida(t *testing.T) {
	escenario := nuevoEscenarioManifiestoHistoricoPrueba(t, verificadorHMACMemoriaPrueba{})
	base := escenario.confirmar
	p := base.ContextoPrevalidacionArchivo.Proyeccion()
	c := base.Contexto.Proyeccion()
	crear := func(
		accion puertosbolsa.AccionOperacionBaremacion,
		recurso, principal, sujeto, perfil, finalidad, autorizacion string,
	) puertosbolsa.ContextoOperacionBaremacion {
		return contextoPrevalidacionAlternativoPrueba(
			base, accion, recurso, principal, sujeto, perfil, finalidad, autorizacion,
			base.ConfirmadaEn,
		)
	}
	casos := []struct {
		nombre   string
		contexto puertosbolsa.ContextoOperacionBaremacion
	}{
		{"misma referencia que confirmacion", crear(
			puertosbolsa.AccionPrevalidarArchivoProbatorioBaremacion, base.Agregado.ID,
			p.PrincipalRef, p.SujetoRef, p.PerfilActorClave, p.FinalidadClave, c.AutorizacionRef,
		)},
		{"otro actor", crear(
			puertosbolsa.AccionPrevalidarArchivoProbatorioBaremacion, base.Agregado.ID,
			"per_0123456789abcdefghijkm", p.SujetoRef, p.PerfilActorClave, p.FinalidadClave, "autorizacion-prevalidacion-actor-ajeno",
		)},
		{"otro sujeto", crear(
			puertosbolsa.AccionPrevalidarArchivoProbatorioBaremacion, base.Agregado.ID,
			p.PrincipalRef, "sujeto-ajeno", p.PerfilActorClave, p.FinalidadClave, "autorizacion-prevalidacion-sujeto-ajeno",
		)},
		{"otro perfil", crear(
			puertosbolsa.AccionPrevalidarArchivoProbatorioBaremacion, base.Agregado.ID,
			p.PrincipalRef, p.SujetoRef, "prf_0123456789abcdefghijkm", p.FinalidadClave, "autorizacion-prevalidacion-perfil-ajeno",
		)},
		{"otra finalidad", crear(
			puertosbolsa.AccionPrevalidarArchivoProbatorioBaremacion, base.Agregado.ID,
			p.PrincipalRef, p.SujetoRef, p.PerfilActorClave, "finalidad_ajena", "autorizacion-prevalidacion-finalidad-ajena",
		)},
		{"otra baremacion", crear(
			puertosbolsa.AccionPrevalidarArchivoProbatorioBaremacion, "baremacion-ajena",
			p.PrincipalRef, p.SujetoRef, p.PerfilActorClave, p.FinalidadClave, "autorizacion-prevalidacion-baremacion-ajena",
		)},
		{"accion de confirmacion", crear(
			puertosbolsa.AccionConfirmarDecisionBaremacion, base.Agregado.ID,
			p.PrincipalRef, p.SujetoRef, p.PerfilActorClave, p.FinalidadClave, "autorizacion-prevalidacion-accion-ajena",
		)},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			adversaria := base
			adversaria.ContextoPrevalidacionArchivo = caso.contexto
			if adversaria.Validar() == nil {
				t.Fatal("contexto confundido admitido")
			}
			if _, err := escenario.repositorio.ConfirmarCambio(
				context.Background(), adversaria,
			); !errors.Is(err, puertosbolsa.ErrSolicitudBaremacionInvalida) {
				t.Fatalf("repositorio no fallo en cerrado: %v", err)
			}
			comprobarDecisionSinEfectosPrueba(t, escenario.repositorio)
		})
	}
}

func TestSolicitudConfirmacionRechazaPrevalidacionExpirada(t *testing.T) {
	escenario := nuevoEscenarioManifiestoHistoricoPrueba(t, verificadorHMACMemoriaPrueba{})
	adversaria := escenario.confirmar
	p, c := adversaria.ContextoPrevalidacionArchivo.Proyeccion(), adversaria.Contexto.Proyeccion()
	emitida := adversaria.ConfirmadaEn.Add(-time.Minute)
	adversaria.Contexto = contextoMemoriaPruebaAutorizacionCompleta(
		puertosbolsa.AccionConfirmarDecisionBaremacion, adversaria.Agregado.ID,
		c.PrincipalRef, c.SujetoRef, c.PerfilActorClave, c.AutenticacionRef,
		c.CorrelacionRef, c.AutorizacionRef, c.FinalidadClave, emitida,
	)
	adversaria.ContextoPrevalidacionArchivo = contextoMemoriaPruebaAutorizacionCompleta(
		puertosbolsa.AccionPrevalidarArchivoProbatorioBaremacion, adversaria.Agregado.ID,
		p.PrincipalRef, p.SujetoRef, p.PerfilActorClave, p.AutenticacionRef,
		p.CorrelacionRef, p.AutorizacionRef, p.FinalidadClave, emitida,
	)
	if adversaria.Contexto.MismoVinculoAutenticacionQue(adversaria.ContextoPrevalidacionArchivo) != true ||
		adversaria.Validar() == nil {
		t.Fatal("prevalidacion expirada admitida o prueba no aisla la vigencia")
	}
	if _, err := escenario.repositorio.ConfirmarCambio(
		context.Background(), adversaria,
	); !errors.Is(err, puertosbolsa.ErrSolicitudBaremacionInvalida) {
		t.Fatalf("prevalidacion expirada no fallo en cerrado: %v", err)
	}
	comprobarDecisionSinEfectosPrueba(t, escenario.repositorio)
}

func sustituirPrevalidacionTrasSelloPrueba(
	t *testing.T,
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
) puertosbolsa.SolicitudConfirmarCambioBaremacion {
	t.Helper()
	clon, err := solicitud.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	p := clon.ContextoPrevalidacionArchivo.Proyeccion()
	const nuevaRef = "autorizacion-prevalidacion-sustituida"
	clon.ContextoPrevalidacionArchivo = contextoPrevalidacionAlternativoPrueba(
		clon, puertosbolsa.AccionPrevalidarArchivoProbatorioBaremacion, clon.Agregado.ID,
		p.PrincipalRef, p.SujetoRef, p.PerfilActorClave, p.FinalidadClave, nuevaRef,
		clon.ConfirmadaEn,
	)
	manifiesto := clon.Manifiesto.Clonar()
	for indice := range manifiesto.Autorizaciones {
		if manifiesto.Autorizaciones[indice].Accion == puertosbolsa.AccionPrevalidarArchivoProbatorioBaremacion {
			manifiesto.Autorizaciones[indice].AutorizacionRef = nuevaRef
		}
	}
	manifiesto.HuellaManifiestoSHA256, manifiesto.SelloManifiestoHMACSHA256 = "", ""
	preparado, representacion, err := manifiesto.PrepararSellado()
	if err != nil {
		t.Fatal(err)
	}
	manifiesto, err = preparado.IncorporarSello(calcularSelloMemoria(
		puertosbolsa.FinalidadSelloManifiestoProbatorioBaremacionV3, representacion.Revelar(),
	))
	if err != nil {
		t.Fatal(err)
	}
	ultima := &clon.Agregado.Decisiones[len(clon.Agregado.Decisiones)-1]
	firma := ultima.Firma
	firma.HuellaManifiestoProbatorioSHA256 = manifiesto.HuellaManifiestoSHA256
	firma.SelloManifiestoProbatorioHMACSHA256 = manifiesto.SelloManifiestoHMACSHA256
	decision, err := dominiobolsa.ConstituirDecisionFirmada(ultima.Contenido, firma)
	if err != nil {
		t.Fatal(err)
	}
	*ultima, clon.Manifiesto = decision, &manifiesto
	if clon.Validar() != nil {
		t.Fatal("la sustitucion estructural de prueba no quedo bien formada")
	}
	return clon
}

func TestRepositorioRechazaSustituirPrevalidacionDespuesDelSello(t *testing.T) {
	escenario := nuevoEscenarioManifiestoHistoricoPrueba(t, verificadorHMACMemoriaPrueba{})
	adversaria := sustituirPrevalidacionTrasSelloPrueba(t, escenario.confirmar)
	if _, err := escenario.repositorio.ConfirmarCambio(
		context.Background(), adversaria,
	); !errors.Is(err, puertosbolsa.ErrSelloBaremacionNoAutentico) {
		t.Fatalf("sustitucion posterior al HMAC admitida: %v", err)
	}
	comprobarDecisionSinEfectosPrueba(t, escenario.repositorio)
	if _, err := escenario.repositorio.ConfirmarCambio(
		context.Background(), escenario.confirmar,
	); err != nil {
		t.Fatalf("el ataque consumio autorizaciones o reserva: %v", err)
	}
}

func TestRepositorioConsumeDosAutorizacionesYReintentaExactamente(t *testing.T) {
	escenario := nuevoEscenarioManifiestoHistoricoPrueba(t, verificadorHMACMemoriaPrueba{})
	primero, err := escenario.repositorio.ConfirmarCambio(context.Background(), escenario.confirmar)
	if err != nil {
		t.Fatal(err)
	}
	repetido, err := escenario.repositorio.ConfirmarCambio(context.Background(), escenario.confirmar)
	if err != nil || repetido.Evidencia != primero.Evidencia ||
		repetido.Version.Referencia != primero.Version.Referencia {
		t.Fatalf("reintento exacto no recupero el recibo: %+v / %v", repetido, err)
	}
	alterada := escenario.confirmar
	alterada.Trazabilidad.Motivo = "Incorporacion alterada para otro efecto."
	alterada = sellarConfirmacionMemoria(alterada)
	huellaAlterada, err := huellaEfectoConfirmacion(alterada)
	if err != nil {
		t.Fatal(err)
	}
	usosAlterados, err := nuevosUsosAutorizacionConfirmacion(
		alterada, escenario.reloj.Ahora(), huellaAlterada,
	)
	if err != nil {
		t.Fatal(err)
	}
	escenario.repositorio.mu.Lock()
	_, err = escenario.repositorio.comprobarUsosConfirmacionBloqueados(usosAlterados)
	escenario.repositorio.mu.Unlock()
	if !errors.Is(err, puertosbolsa.ErrAutorizacionBaremacionReutilizada) {
		t.Fatalf("la misma decision de prevalidacion habilito otro efecto: %v", err)
	}
	escenario.repositorio.mu.RLock()
	defer escenario.repositorio.mu.RUnlock()
	confirmacion := escenario.confirmar.Contexto.Proyeccion().AutorizacionRef
	prevalidacion := escenario.confirmar.ContextoPrevalidacionArchivo.Proyeccion().AutorizacionRef
	if confirmacion == prevalidacion || escenario.repositorio.usosAutorizacion[confirmacion].DecisionRef == "" ||
		escenario.repositorio.usosAutorizacion[prevalidacion].DecisionRef == "" {
		t.Fatal("consumo dual no quedo persistido de forma diferenciada")
	}
}
