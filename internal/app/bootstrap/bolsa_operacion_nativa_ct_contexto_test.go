package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

type sesionOperacionBolsaCTPrueba struct {
	actual contextoSeguridadComunDesarrollo
	usos   int
}

func (s *sesionOperacionBolsaCTPrueba) ResolverContexto(context.Context) (contextoSeguridadComunDesarrollo, error) {
	s.usos++
	return s.actual, nil
}

func TestOperacionPrivadaBolsaCTExigeFronteraPerfilYCertificado(t *testing.T) {
	p, ctx, _, _ := puenteBolsaPrueba(t)
	soporte := p.alta.soporte
	perfilCT := soporte.contexto.Resultado.Contexto.PerfilActivoRef
	principal := core.Principal{ID: soporte.principalID,
		Roles:      []string{rolTecnicoRRHHContratacionTemporalDesarrollo},
		AuthMethod: core.AuthMethodCertificate, AuthAssurance: core.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa,
			"perfil_ejecucion":   config.ExecutionProfileDevelopment,
			"certificate_sha256": soporte.certificadoSHA256}}
	contextoBolsa, err := nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(
		principal, time.Now().UTC().Truncate(time.Microsecond), discriminadorContextoSinteticoBorradorBolsaDesarrollo())
	if err != nil {
		t.Fatal(err)
	}
	contextoFijo, err := nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(
		principal, time.Now().UTC().Truncate(time.Microsecond), discriminadorPerfilFijoCTDesarrollo("incorporacion_b2_ct155"))
	if err != nil || contextoFijo.Resultado.Contexto.PerfilActivoRef == perfilCT {
		t.Fatal("perfil fijo CT de prueba inválido", err)
	}
	perfilBolsa := contextoBolsa.Resultado.Contexto.PerfilActivoRef
	publico, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{{
		Clave: "ct-b2-confirmar-prueba", Metodo: http.MethodPost,
		Ruta: httpct.RutaConfirmacionB2, Superficie: superficieInternaSeguridadComunDesarrollo,
		PerfilesActivosRef: []string{perfilCT, contextoFijo.Resultado.Contexto.PerfilActivoRef}, ClavePolitica: "politica-ct-prueba", ClaveCapacidad: "capacidad-ct-prueba",
	}})
	if err != nil {
		t.Fatal(err)
	}
	privado, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{{
		Clave: fronteraAltaBolsaNativaCT, Metodo: http.MethodPost,
		Ruta: httpct.RutaConfirmacionB2, Superficie: superficieInternaSeguridadComunDesarrollo,
		PerfilesActivosRef: []string{perfilBolsa}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo,
		ClaveCapacidad: claveCapacidadSituacionParticipacionBolsa,
	}})
	if err != nil {
		t.Fatal(err)
	}
	bolsa := &soporteAltaContratacionTemporalDesarrollo{sello: soporte.sello,
		principalID: soporte.principalID, certificadoSHA256: soporte.certificadoSHA256,
		contexto: contextoBolsa, reloj: soporte.reloj}
	sesion := &sesionOperacionBolsaCTPrueba{actual: contextoSeguridadComunDesarrollo{
		Vinculo: bolsa.contexto.Vinculo, Resultado: bolsa.contexto.Resultado}}
	o := &operacionBolsaNativaCT{ct: soporte, bolsa: &soporteSesionBorradorBolsaDesarrollo{soporteCanal: bolsa},
		publico: publico, privado: privado, sesion: &seguridadComunDesarrollo{sesion: sesion, reloj: soporte.reloj}}
	capacidad, ok := soporte.capacidadValida(ctx)
	if !ok {
		t.Fatal("capacidad CT de prueba inválida")
	}
	capacidad.ruta, capacidad.metodo = httpct.RutaConfirmacionB2, http.MethodPost
	ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	ctx = context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
		metodo: http.MethodPost, ruta: httpct.RutaConfirmacionB2,
		superficie: superficieInternaSeguridadComunDesarrollo, catalogo: publico})
	datosBolsa, err := bolsa.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datosCT, err := soporte.contexto.Vinculo.Datos()
	if err != nil || datosBolsa.PrincipalID != datosCT.PrincipalID || datosBolsa.CuentaRef != datosCT.CuentaRef {
		t.Fatal("fixture con cuenta o principal nominal distinto", err)
	}
	if _, _, err := o.Contexto(ctx, &contextoFijo); err != nil || sesion.usos != 1 {
		t.Fatalf("suboperación nominal coherente: usos=%d err=%v", sesion.usos, err)
	}
	if _, _, err := o.Contexto(ctx, nil); !errors.Is(err, ct.ErrAutorizacionDenegada) || sesion.usos != 1 {
		t.Fatal("B2 sin contexto CT nominal llegó a Bolsa")
	}
	otros, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{{
		Clave: "ajeno", Metodo: http.MethodPost, Ruta: httpct.RutaConfirmacionB2,
		Superficie: superficieInternaSeguridadComunDesarrollo, PerfilesActivosRef: []string{perfilCT},
		ClavePolitica: "politica-ajena", ClaveCapacidad: "capacidad-ajena",
	}})
	if err != nil {
		t.Fatal(err)
	}
	ctxAjeno := context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
		metodo: http.MethodPost, ruta: httpct.RutaConfirmacionB2,
		superficie: superficieInternaSeguridadComunDesarrollo, catalogo: otros})
	if _, _, err := o.Contexto(ctxAjeno, &contextoFijo); !errors.Is(err, ct.ErrAutorizacionDenegada) || sesion.usos != 1 {
		t.Fatal("frontera ajena alcanzó el resolutor Bolsa")
	}
	privadoAjeno, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{{
		Clave: fronteraAltaBolsaNativaCT, Metodo: http.MethodPost, Ruta: httpct.RutaConfirmacionB2,
		Superficie: superficieInternaSeguridadComunDesarrollo, PerfilesActivosRef: []string{perfilCT},
		ClavePolitica:  clavePoliticaBorradorLlamamientoBolsaDesarrollo,
		ClaveCapacidad: claveCapacidadSituacionParticipacionBolsa,
	}})
	if err != nil {
		t.Fatal(err)
	}
	o.privado = privadoAjeno
	if _, _, err := o.Contexto(ctx, &contextoFijo); !errors.Is(err, ct.ErrAutorizacionDenegada) || sesion.usos != 1 {
		t.Fatal("perfil CT habilitó suboperación Bolsa")
	}
	o.privado = privado
	bolsa.certificadoSHA256 = strings.Repeat("f", 64)
	if _, _, err := o.Contexto(ctx, &contextoFijo); !errors.Is(err, ct.ErrAutorizacionDenegada) || sesion.usos != 1 {
		t.Fatal("certificado distinto alcanzó el resolutor Bolsa")
	}
	bolsa.certificadoSHA256 = soporte.certificadoSHA256
	sesion.actual = contextoSeguridadComunDesarrollo{Vinculo: soporte.contexto.Vinculo, Resultado: soporte.contexto.Resultado}
	if _, _, err := o.Contexto(ctx, &contextoFijo); !errors.Is(err, ct.ErrAutorizacionDenegada) || sesion.usos != 2 {
		t.Fatal("actor CT fue aceptado como actor de la suboperación Bolsa")
	}
	sesion.actual = contextoSeguridadComunDesarrollo{Vinculo: bolsa.contexto.Vinculo, Resultado: bolsa.contexto.Resultado}
	publicoSeleccion, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{{
		Clave: "ct-seleccion-prueba", Metodo: http.MethodPost, Ruta: httpct.RutaSeleccionLlamamiento,
		Superficie: superficieInternaSeguridadComunDesarrollo, PerfilesActivosRef: []string{perfilCT},
		ClavePolitica: "politica-ct-prueba", ClaveCapacidad: "capacidad-ct-prueba",
	}})
	if err != nil {
		t.Fatal(err)
	}
	privadoSeleccion, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{{
		Clave: fronteraSeleccionBolsaNativaCT, Metodo: http.MethodPost, Ruta: httpct.RutaSeleccionLlamamiento,
		Superficie: superficieInternaSeguridadComunDesarrollo, PerfilesActivosRef: []string{perfilBolsa},
		ClavePolitica:  clavePoliticaBorradorLlamamientoBolsaDesarrollo,
		ClaveCapacidad: claveCapacidadEmisionLlamamientoBolsa,
	}})
	if err != nil {
		t.Fatal(err)
	}
	o.publico, o.privado = publicoSeleccion, privadoSeleccion
	seleccion := capacidad
	seleccion.ruta = httpct.RutaSeleccionLlamamiento
	ctxSeleccion := context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, seleccion)
	ctxSeleccion = context.WithValue(ctxSeleccion, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
		metodo: http.MethodPost, ruta: httpct.RutaSeleccionLlamamiento,
		superficie: superficieInternaSeguridadComunDesarrollo, catalogo: publicoSeleccion})
	if _, _, err := o.Contexto(ctxSeleccion, nil); err != nil || sesion.usos != 3 {
		t.Fatalf("selección CT con perfil nominal vigente: usos=%d err=%v", sesion.usos, err)
	}
	if _, _, err := o.Contexto(ctxSeleccion, &contextoFijo); !errors.Is(err, ct.ErrAutorizacionDenegada) || sesion.usos != 3 {
		t.Fatal("selección admitió perfil aportado")
	}
}
