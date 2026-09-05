package bootstrap

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func escenarioConsultasRRHHDesarrolloPrueba(t *testing.T) (*dependenciasAltaContratacionTemporalDesarrollo, *autoridadConsultasRRHHDesarrollo, dominiovec.Principal) {
	t.Helper()
	soporte, cobertura, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	delegado, ok := cobertura.autorizador.(autorizadorLigadoContratacionTemporalDesarrollo)
	if !ok {
		t.Fatal("se requiere el autorizador V3 existente")
	}
	alta := &dependenciasAltaContratacionTemporalDesarrollo{soporte: soporte, autorizador: delegado}
	// Dobles de las referencias entregadas por el resolutor SQL. Esta prueba
	// no acredita publicación de catálogos ni persistencia PostgreSQL.
	motivoCuadro := dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_rrhh_prueba", CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("c", 64), EntradaClave: "motivo_0123456789abcdef0123456789abcdef",
	}
	motivoDetalle := motivoCuadro
	motivoDetalle.EntradaClave = "motivo_fedcba9876543210fedcba9876543210"
	anterior := clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(soporte.instantanea)
	analisis := clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(soporte.instantaneaAnalisis)
	autoridad, err := configurarAutoridadConsultasRRHHDesarrollo(alta, soporte.reloj, motivoCuadro, motivoDetalle)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(anterior, soporte.instantanea) || !reflect.DeepEqual(analisis, soporte.instantaneaAnalisis) {
		t.Fatal("se ha alterado un rol anterior")
	}
	// Doble acotado a estas pruebas; no acredita sesión ni persistencia real.
	proveedor := &proveedorContextoConsultasRRHHDesarrolloPrueba{
		resolver: func(context.Context) (ports.ContextoAutorizacionAltaV3, error) {
			resultado, err := soporte.contexto.Resultado.Clonar()
			return ports.ContextoAutorizacionAltaV3{Vinculo: soporte.contexto.Vinculo, Resultado: resultado}, err
		},
	}
	if err := autoridad.configurarProveedorContextoConsultaRRHHDesarrollo(proveedor); err != nil {
		t.Fatal(err)
	}
	return alta, autoridad, principal
}

func datosSolicitudConsultasRRHHDesarrolloPrueba(t *testing.T, s *soporteAltaContratacionTemporalDesarrollo, ruta string) dominiovec.DatosSolicitudAutorizacionLigadaV3 {
	t.Helper()
	accion, finalidad, tipo, referencia, dominio := ports.AccionConsultarCuadroRRHH,
		ports.FinalidadConsultarCuadroRRHH, ports.TipoRecursoCuadroRRHH,
		organizacionAltaContratacionTemporalDesarrollo, ports.DominioHuellaConsultaCuadroRRHH
	cuadro, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 100, "")
	if err != nil {
		t.Fatal(err)
	}
	huella, err := cuadro.HuellaCanonicaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	if ruta == httpinterno.RutaConsultaDetalleRRHH {
		accion, finalidad, tipo, referencia, dominio = ports.AccionConsultarDetalleRRHH,
			ports.FinalidadConsultarDetalleRRHH, ports.TipoRecursoExpediente,
			"expediente:ct:prueba:bandeja", ports.DominioHuellaConsultaDetalleRRHH
		detalle, err := ports.NuevaSolicitudDetalleRRHH(referencia, 6)
		if err != nil {
			t.Fatal(err)
		}
		huella, err = detalle.HuellaCanonicaSHA256()
		if err != nil {
			t.Fatal(err)
		}
	}
	motivo, ok := s.motivoAutorizacionParaRuta(ruta)
	if !ok {
		t.Fatal("motivo no configurado")
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		t.Fatal(err)
	}
	return dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: s.contexto.Vinculo, ReferenciaMotivo: motivo,
		Accion: accion, Finalidad: finalidad, Correlacion: correlacion,
		Recurso: dominiovec.RecursoAutorizable{
			Referencia: referencia, ModuloID: ports.ModuloContratacion, Tipo: tipo,
			Ambitos: map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo,
				"clase_ambito": string(ports.AmbitoOrganizacionRRHH), "ambito_ref": organizacionAltaContratacionTemporalDesarrollo},
			Atributos: map[string]string{"consulta_dominio": dominio, "consulta_huella_sha256": huella},
		},
	}
}

func TestConsultasRRHHDesarrolloContextoSoloCanalCertificado(t *testing.T) {
	alta, autoridad, principal := escenarioConsultasRRHHDesarrolloPrueba(t)
	s := alta.soporte
	for _, ruta := range []string{httpinterno.RutaConsultaCuadroRRHH, httpinterno.RutaConsultaDetalleRRHH} {
		ctx := contextoRutaConsultasRRHHDesarrolloPrueba(s, principal, ruta)
		obtenido, err := autoridad.ResolverContextoConsultaRRHH(ctx)
		if err != nil || obtenido.OrganizacionRef() != organizacionAltaContratacionTemporalDesarrollo {
			t.Fatalf("contexto de consulta no resuelto: %v", err)
		}
	}
	for _, caso := range []struct {
		nombre    string
		modificar func(*dominiovec.Principal)
	}{
		{"otro_actor", func(p *dominiovec.Principal) { p.ID = "certificado:otro" }},
		{"otro_certificado", func(p *dominiovec.Principal) { p.Attributes["certificate_sha256"] = strings.Repeat("e", 64) }},
		{"intervencion", func(p *dominiovec.Principal) { p.Roles = []string{"intervencion"} }},
		{"garantia_baja", func(p *dominiovec.Principal) { p.AuthAssurance = dominiovec.AuthAssuranceLow }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			copia := clonarPrincipalDesarrollo(principal)
			caso.modificar(&copia)
			ctx := contextoRutaConsultasRRHHDesarrolloPrueba(s, copia, httpinterno.RutaConsultaCuadroRRHH)
			if _, err := autoridad.ResolverContextoConsultaRRHH(ctx); err == nil {
				t.Fatal("identidad no autorizada")
			}
		})
	}
	ctxCancelado, cancelar := context.WithCancel(contextoRutaConsultasRRHHDesarrolloPrueba(s, principal, httpinterno.RutaConsultaCuadroRRHH))
	cancelar()
	for _, ctx := range []context.Context{nil, context.Background(), ctxCancelado,
		contextoRutaConsultasRRHHDesarrolloPrueba(s, principal, httpinterno.RutaAltaSolicitudes)} {
		if _, err := autoridad.ResolverContextoConsultaRRHH(ctx); err == nil {
			t.Fatal("se aceptó un canal no válido")
		}
	}
	autoridad.reloj = relojFijoAltaContratacionTemporalDesarrollo{ahora: time.Date(2037, 1, 1, 0, 0, 0, 0, time.UTC)}
	if _, err := autoridad.ResolverContextoConsultaRRHH(contextoRutaConsultasRRHHDesarrolloPrueba(s, principal, httpinterno.RutaConsultaCuadroRRHH)); err == nil {
		t.Fatal("se aceptó un contexto vencido")
	}
}

func TestConsultasRRHHDesarrolloAutorizaV3YUsaRegistroDurable(t *testing.T) {
	alta, autoridad, principal := escenarioConsultasRRHHDesarrolloPrueba(t)
	s := alta.soporte
	roles := make(map[string]bool)
	for _, ruta := range []string{httpinterno.RutaConsultaCuadroRRHH, httpinterno.RutaConsultaDetalleRRHH} {
		ctx := contextoRutaConsultasRRHHDesarrolloPrueba(s, principal, ruta)
		datos := datosSolicitudConsultasRRHHDesarrolloPrueba(t, s, ruta)
		solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(datos)
		if err != nil {
			t.Fatal(err)
		}
		decision, confirmacion, err := autoridad.ExigirSolicitudLigadaV3(ctx, solicitud, s.contexto.Resultado)
		concedida, _, errResultado := decision.Resultado()
		if err != nil || errResultado != nil || !concedida || confirmacion.Validar() != nil || decision.ValidarPara(solicitud) != nil {
			t.Fatalf("consulta V3 no concedida y registrada: %v %v", err, errResultado)
		}
		instantanea, ok := s.instantaneaParaRuta(ruta)
		if !ok || instantanea.Validar() != nil || len(instantanea.VersionRol.Concesiones) != 1 ||
			instantanea.VersionRol.Concesiones[0].Accion != datos.Accion ||
			instantanea.VersionRol.Concesiones[0].TipoRecurso != datos.Recurso.Tipo ||
			!reflect.DeepEqual(instantanea.VersionRol.Concesiones[0].Finalidades, []string{datos.Finalidad}) {
			t.Fatal("concesión no nominal")
		}
		if roles[instantanea.VersionRol.RolID] {
			t.Fatal("cuadro y detalle comparten rol")
		}
		roles[instantanea.VersionRol.RolID] = true
	}
	publicador := s.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	registro := s.registroDecisionesAnalisis.(*registroDecisionesAnalisisContratacionTemporalDesarrolloPrueba)
	if publicador.preparadas != 2 || publicador.publicadas != 2 || registro.concesiones != 2 || len(s.concesiones) != 0 {
		t.Fatalf("no se delegó en publicación/registro durable: preparadas=%d publicadas=%d registros=%d memoria=%d",
			publicador.preparadas, publicador.publicadas, registro.concesiones, len(s.concesiones))
	}
}

func TestConsultasRRHHDesarrolloRechazaCrucesSinDelegar(t *testing.T) {
	alta, autoridad, principal := escenarioConsultasRRHHDesarrolloPrueba(t)
	s := alta.soporte
	for _, ruta := range []string{httpinterno.RutaConsultaCuadroRRHH, httpinterno.RutaConsultaDetalleRRHH} {
		base := datosSolicitudConsultasRRHHDesarrolloPrueba(t, s, ruta)
		for _, caso := range []struct {
			nombre    string
			modificar func(*dominiovec.DatosSolicitudAutorizacionLigadaV3)
		}{
			{"accion", func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) { d.Accion = ports.AccionCrearSolicitud }},
			{"finalidad", func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
				d.Finalidad = "gestionar_contratacion_temporal"
			}},
			{"tipo", func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
				d.Recurso.Tipo = "seleccion_llamamiento_contratacion_temporal"
			}},
			{"organizacion", func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
				d.Recurso.Ambitos["organizacion_ref"] = "organizacion:otra"
			}},
			{"alcance", func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
				d.Recurso.Ambitos["ambito_ref"] = "organizacion:otra"
			}},
			{"clase", func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) { d.Recurso.Ambitos["clase_ambito"] = "centro" }},
			{"atributo_extra", func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) { d.Recurso.Atributos["extra"] = "valor" }},
			{"dominio", func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
				d.Recurso.Atributos["consulta_dominio"] = "dominio:otro"
			}},
			{"huella_nula", func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
				d.Recurso.Atributos["consulta_huella_sha256"] = strings.Repeat("0", 64)
			}},
			{"motivo", func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) { d.ReferenciaMotivo = s.motivo }},
		} {
			t.Run(ruta+"/"+caso.nombre, func(t *testing.T) {
				datos := base
				datos.Recurso.Ambitos = maps.Clone(base.Recurso.Ambitos)
				datos.Recurso.Atributos = maps.Clone(base.Recurso.Atributos)
				caso.modificar(&datos)
				solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(datos)
				if err != nil {
					t.Fatal("la variación debe alcanzar la frontera de composición")
				}
				ctx := contextoRutaConsultasRRHHDesarrolloPrueba(s, principal, ruta)
				if _, _, err := autoridad.ExigirSolicitudLigadaV3(ctx, solicitud, s.contexto.Resultado); err == nil {
					t.Fatal("consulta alterada autorizada")
				}
			})
		}
		otraRuta := httpinterno.RutaConsultaCuadroRRHH
		if ruta == otraRuta {
			otraRuta = httpinterno.RutaConsultaDetalleRRHH
		}
		solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(base)
		if err != nil {
			t.Fatal(err)
		}
		ctx := contextoRutaConsultasRRHHDesarrolloPrueba(s, principal, otraRuta)
		if _, _, err := autoridad.ExigirSolicitudLigadaV3(ctx, solicitud, s.contexto.Resultado); err == nil {
			t.Fatal("se cruzaron las rutas de cuadro y detalle")
		}
	}
	if s.registroDecisionesAnalisis.(*registroDecisionesAnalisisContratacionTemporalDesarrolloPrueba).concesiones != 0 {
		t.Fatal("se delegó una petición rechazada")
	}
}

func TestConsultasRRHHDesarrolloFallaSinRegistroYNoReconfigura(t *testing.T) {
	alta, autoridad, principal := escenarioConsultasRRHHDesarrolloPrueba(t)
	s := alta.soporte
	if _, err := configurarAutoridadConsultasRRHHDesarrollo(alta, s.reloj, s.motivoCuadroRRHH, s.motivoDetalleRRHH); err == nil {
		t.Fatal("se reemplazó una configuración existente")
	}
	for _, motivos := range [][2]dominiovec.ReferenciaEntradaCatalogo{
		{{}, s.motivoDetalleRRHH}, {s.motivoCuadroRRHH, {}}, {s.motivoCuadroRRHH, s.motivoCuadroRRHH},
	} {
		if _, err := configurarAutoridadConsultasRRHHDesarrollo(alta, s.reloj, motivos[0], motivos[1]); err == nil {
			t.Fatal("se aceptó motivo ausente o intercambiable")
		}
	}
	ctx := contextoRutaConsultasRRHHDesarrolloPrueba(s, principal, httpinterno.RutaConsultaCuadroRRHH)
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(datosSolicitudConsultasRRHHDesarrolloPrueba(t, s, httpinterno.RutaConsultaCuadroRRHH))
	if err != nil {
		t.Fatal(err)
	}
	s.registroDecisionesAnalisis = nil
	if _, _, err := autoridad.ExigirSolicitudLigadaV3(ctx, solicitud, s.contexto.Resultado); err == nil {
		t.Fatal("se simuló confirmación sin registro durable")
	}
	if len(s.concesiones) != 0 {
		t.Fatal("se recurrió al registro en memoria")
	}
}

type proveedorContextoConsultasRRHHDesarrolloPrueba struct {
	resolver func(context.Context) (ports.ContextoAutorizacionAltaV3, error)
	llamadas int
}

func (p *proveedorContextoConsultasRRHHDesarrolloPrueba) ResolverContexto(ctx context.Context) (ports.ContextoAutorizacionAltaV3, error) {
	p.llamadas++
	return p.resolver(ctx)
}

func contextoRutaConsultasRRHHDesarrolloPrueba(s *soporteAltaContratacionTemporalDesarrollo, principal dominiovec.Principal, ruta string) context.Context {
	ctx := contextoRutaCoberturaDesarrolloPrueba(s, principal, ruta)
	capacidad := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	capacidad.consultaRRHH = &contextoConsultaRRHHPeticionDesarrollo{}
	capacidad.certificadoVerificadoEn = s.reloj.Ahora()
	capacidad.certificadoValidoHasta = s.reloj.Ahora().Add(time.Hour)
	return context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
}

// Fabrica un par de prueba mediante el dominio, con autenticación breve y
// resultado temporal coherente. No se usa fuera de tests ni escribe sesiones.
func contextoFrescoConsultasRRHHDesarrolloPrueba(t *testing.T, s *soporteAltaContratacionTemporalDesarrollo, clave string) ports.ContextoAutorizacionAltaV3 {
	t.Helper()
	ahora := s.reloj.Ahora()
	base, err := s.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := s.contexto.Resultado.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	cuenta := dominiovec.CuentaAutenticadaContextoActor{
		CuentaRef: base.CuentaRef, Metodo: base.MetodoObservado, Garantia: base.GarantiaObservada,
	}
	resultado.Contexto, err = dominiovec.NuevoContextoActor(cuenta, resultado.Contexto.Instantanea, ahora)
	if err != nil {
		t.Fatal(err)
	}
	resultado.RepresentacionCanonica, err = resultado.Contexto.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	resultado.HuellaSHA256, err = resultado.Contexto.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	resultado.ResueltoEnAutoritativo = ahora
	resultado.RegistroContextoRef = referenciaAltaContratacionTemporalDesarrollo("rca_", "prueba-consulta:"+clave)
	autenticacion := base.Autenticacion()
	autenticacion.AutenticacionRef = referenciaAltaContratacionTemporalDesarrollo("aut_", "prueba-consulta:"+clave)
	autenticacion.AsercionRef = referenciaAltaContratacionTemporalDesarrollo("ase_", "prueba-consulta:"+clave)
	autenticacion.SesionRef = referenciaAltaContratacionTemporalDesarrollo("ses_", "prueba-consulta:"+clave)
	autenticacion.ControlSesionRef = referenciaAltaContratacionTemporalDesarrollo("cse_", "prueba-consulta:"+clave)
	autenticacion.AutenticacionHuellaSHA256 = huellaAltaContratacionTemporalDesarrollo("prueba-consulta:" + clave)
	autenticacion.ControlSesionHuellaSHA256 = huellaAltaContratacionTemporalDesarrollo("prueba-control:" + clave)
	autenticacion.AutenticacionVerificadaEn = ahora.Add(-time.Second)
	autenticacion.SesionEmitidaEn = ahora.Add(-time.Second)
	autenticacion.SesionRevalidadaEn = ahora
	autenticacion.SesionValidaHasta = ahora.Add(time.Minute)
	vinculo, resultado, err := dominiovec.CrearVinculoAutenticacionActorV2ConResultado(
		context.Background(), revalidadorAutenticacionAltaContratacionTemporalDesarrollo{valor: autenticacion},
		dominiovec.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef},
		resolutorContextoAltaContratacionTemporalDesarrollo{valor: resultado},
		dominiovec.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: base.PerfilActivoRef}, s.reloj,
	)
	if err != nil {
		t.Fatal(err)
	}
	return ports.ContextoAutorizacionAltaV3{Vinculo: vinculo, Resultado: resultado}
}

func TestConsultasRRHHDesarrolloSesionFrescaUnicaPorPeticion(t *testing.T) {
	alta, autoridad, principal := escenarioConsultasRRHHDesarrolloPrueba(t)
	s := alta.soporte
	baseAntes, err := s.contexto.Resultado.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	vinculoAntes := s.contexto.Vinculo
	proveedor := autoridad.proveedor.(*proveedorContextoConsultasRRHHDesarrolloPrueba)
	frescos := []ports.ContextoAutorizacionAltaV3{
		contextoFrescoConsultasRRHHDesarrolloPrueba(t, s, "primera"),
		contextoFrescoConsultasRRHHDesarrolloPrueba(t, s, "segunda"),
	}
	proveedor.resolver = func(ctx context.Context) (ports.ContextoAutorizacionAltaV3, error) {
		if _, ok := s.capacidadValida(ctx); !ok {
			return ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
		}
		return frescos[proveedor.llamadas-1], nil
	}
	for i, fresco := range frescos {
		ctx := contextoRutaConsultasRRHHDesarrolloPrueba(s, principal, httpinterno.RutaConsultaCuadroRRHH)
		obtenido, err := autoridad.ResolverContextoConsultaRRHH(ctx)
		datosFrescos, _ := fresco.Vinculo.Datos()
		if err != nil || obtenido.SesionRef() != datosFrescos.SesionRef {
			t.Fatalf("no usa sesión fresca: %v", err)
		}
		datos := datosSolicitudConsultasRRHHDesarrolloPrueba(t, s, httpinterno.RutaConsultaCuadroRRHH)
		vieja, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(datos)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err = autoridad.ExigirSolicitudLigadaV3(ctx, vieja, s.contexto.Resultado); err == nil {
			t.Fatal("acepta vínculo anterior")
		}
		datos.VinculoAutenticacionActor = fresco.Vinculo
		solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(datos)
		if err != nil {
			t.Fatal(err)
		}
		decision, confirmacion, err := autoridad.ExigirSolicitudLigadaV3(ctx, solicitud, fresco.Resultado)
		concedida, _, errDecision := decision.Resultado()
		if err != nil || errDecision != nil || !concedida || confirmacion.Validar() != nil {
			t.Fatalf("deniega contexto fresco coherente: %v", err)
		}
		if proveedor.llamadas != i+1 {
			t.Fatal("repite registro de sesión dentro de la petición")
		}
	}
	if !vinculoAntes.CoincideExactamenteCon(s.contexto.Vinculo) || !reflect.DeepEqual(baseAntes, s.contexto.Resultado) {
		t.Fatal("modifica contexto global")
	}
}

func TestConsultasRRHHDesarrolloProveedorObligatorioYFalloRetenido(t *testing.T) {
	alta, autoridad, principal := escenarioConsultasRRHHDesarrolloPrueba(t)
	s := alta.soporte
	proveedor := autoridad.proveedor.(*proveedorContextoConsultasRRHHDesarrolloPrueba)
	if err := autoridad.configurarProveedorContextoConsultaRRHHDesarrollo(proveedor); err == nil {
		t.Fatal("reconfigura fuente")
	}
	autoridad.proveedor = nil
	var nulo *proveedorContextoConsultasRRHHDesarrolloPrueba
	if err := autoridad.configurarProveedorContextoConsultaRRHHDesarrollo(nulo); err == nil {
		t.Fatal("acepta nulo tipado")
	}
	ctx := contextoRutaConsultasRRHHDesarrolloPrueba(s, principal, httpinterno.RutaConsultaCuadroRRHH)
	if _, err := autoridad.ResolverContextoConsultaRRHH(ctx); err == nil {
		t.Fatal("usa contexto histórico sin fuente")
	}
	if err := autoridad.configurarProveedorContextoConsultaRRHHDesarrollo(proveedor); err != nil {
		t.Fatal(err)
	}
	proveedor.resolver = func(context.Context) (ports.ContextoAutorizacionAltaV3, error) {
		return ports.ContextoAutorizacionAltaV3{}, errors.New("error privado del proveedor")
	}
	for range 2 {
		if _, err := autoridad.ResolverContextoConsultaRRHH(ctx); !errors.Is(err, ports.ErrAutorizacionDenegada) {
			t.Fatalf("no sanea fallo: %v", err)
		}
	}
	if proveedor.llamadas != 1 {
		t.Fatal("repite operación tras error")
	}
}

func TestConsultasRRHHDesarrolloCacheConcurrenteClonadaYNoProlongaSesion(t *testing.T) {
	alta, autoridad, principal := escenarioConsultasRRHHDesarrolloPrueba(t)
	s := alta.soporte
	fresco := contextoFrescoConsultasRRHHDesarrolloPrueba(t, s, "concurrente")
	proveedor := autoridad.proveedor.(*proveedorContextoConsultasRRHHDesarrolloPrueba)
	proveedor.resolver = func(context.Context) (ports.ContextoAutorizacionAltaV3, error) { return fresco, nil }
	ctx := contextoRutaConsultasRRHHDesarrolloPrueba(s, principal, httpinterno.RutaConsultaCuadroRRHH)
	var grupo sync.WaitGroup
	errores := make(chan error, 8)
	for range 8 {
		grupo.Go(func() {
			obtenido, err := autoridad.contextoConsultaRRHHDesarrollo(ctx)
			if err == nil && !obtenido.Vinculo.CoincideExactamenteCon(fresco.Vinculo) {
				err = fmt.Errorf("sesión distinta")
			}
			if err == nil {
				obtenido.Resultado.RepresentacionCanonica[0] = 0
			}
			errores <- err
		})
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatal(err)
		}
	}
	if proveedor.llamadas != 1 {
		t.Fatal("más de una invocación concurrente")
	}
	if _, err := autoridad.ResolverContextoConsultaRRHH(ctx); err != nil {
		t.Fatal("el llamador alteró la copia retenida")
	}
	autoridad.reloj = relojFijoAltaContratacionTemporalDesarrollo{ahora: s.reloj.Ahora().Add(time.Minute)}
	if _, err := autoridad.ResolverContextoConsultaRRHH(ctx); err == nil {
		t.Fatal("prolonga sesión al vencer")
	}
	if proveedor.llamadas != 1 {
		t.Fatal("renueva sesión dentro de petición vencida")
	}
}

func TestConsultasRRHHDesarrolloRechazaProveedorAjenoYSinHolder(t *testing.T) {
	alta, autoridad, principal := escenarioConsultasRRHHDesarrolloPrueba(t)
	s := alta.soporte
	proveedor := autoridad.proveedor.(*proveedorContextoConsultasRRHHDesarrolloPrueba)
	otro := clonarPrincipalDesarrollo(principal)
	otro.ID = "certificado_rrhh_distinto"
	ajeno, err := nuevoContextoAltaContratacionTemporalDesarrollo(otro, s.reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	proveedor.resolver = func(context.Context) (ports.ContextoAutorizacionAltaV3, error) { return ajeno, nil }
	ctxSinHolder := contextoRutaCoberturaDesarrolloPrueba(s, principal, httpinterno.RutaConsultaCuadroRRHH)
	if _, err := autoridad.ResolverContextoConsultaRRHH(ctxSinHolder); err == nil || proveedor.llamadas != 0 {
		t.Fatal("invoca fuente sin holder de petición")
	}
	ctx := contextoRutaConsultasRRHHDesarrolloPrueba(s, principal, httpinterno.RutaConsultaCuadroRRHH)
	if _, err := autoridad.ResolverContextoConsultaRRHH(ctx); err == nil {
		t.Fatal("acepta otro actor/cuenta/perfil")
	}
	if proveedor.llamadas != 1 {
		t.Fatal("no ha probado la frontera del proveedor")
	}
}

func TestConsultasRRHHDesarrolloCapacidadRespetaVentanaCertificado(t *testing.T) {
	alta, _, principal := escenarioConsultasRRHHDesarrolloPrueba(t)
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	// Prueba unitaria del estado TLS recibido por el middleware, no de un
	// handshake ni de la emisión de certificados o sesiones nominales.
	raw := []byte("certificado_sintetico_ventana_consultas")
	huella := sha256.Sum256(raw)
	principal.Attributes["certificate_sha256"] = hex.EncodeToString(huella[:])
	resolvedor, err := nuevoResolvedorIdentidadDesarrollo(identidadCertificadoDesarrollo{
		huella: huella, principal: principal,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		nombre, ruta string
		desde, hasta time.Time
		emite        bool
	}{
		{"vigente_cuadro", httpinterno.RutaConsultaCuadroRRHH, ahora.Add(-time.Hour), ahora.Add(time.Hour), true},
		{"vigente_detalle", httpinterno.RutaConsultaDetalleRRHH, ahora.Add(-time.Hour), ahora.Add(time.Hour), true},
		{"caducado", httpinterno.RutaConsultaCuadroRRHH, ahora.Add(-time.Hour), ahora.Add(-time.Minute), false},
		{"futuro", httpinterno.RutaConsultaDetalleRRHH, ahora.Add(time.Minute), ahora.Add(time.Hour), false},
		{"ruta_previa_intacta", httpinterno.RutaAltaSolicitudes, ahora.Add(-time.Hour), ahora.Add(-time.Minute), true},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			certificado := &x509.Certificate{Raw: raw, NotBefore: caso.desde, NotAfter: caso.hasta}
			peticion := httptest.NewRequest(http.MethodPost, "https://localhost"+caso.ruta, nil)
			peticion.RemoteAddr = "127.0.0.1:41000"
			peticion.TLS = &tls.ConnectionState{
				HandshakeComplete: true, Version: tls.VersionTLS13,
				PeerCertificates: []*x509.Certificate{certificado},
				VerifiedChains:   [][]*x509.Certificate{{certificado, {}}},
			}
			llamadas := 0
			middleware := &revalidadorConsultasContratacionTemporalDesarrollo{
				autoridad: &autoridadConsultasContratacionTemporalDesarrollo{
					sello:      alta.soporte.sello,
					resolvedor: resolvedor,
				},
				siguiente: http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
					llamadas++
					capacidad, existe := r.Context().Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
					if existe != caso.emite {
						t.Fatal("emisión de capacidad incorrecta")
					}
					if existe && rutaConsultaRRHHContratacionTemporalDesarrollo(caso.ruta) {
						if capacidad.consultaRRHH == nil || capacidad.certificadoVerificadoEn.Location() != time.UTC ||
							capacidad.certificadoVerificadoEn.Before(caso.desde) || !capacidad.certificadoVerificadoEn.Before(caso.hasta) ||
							capacidad.certificadoValidoHasta != certificado.NotAfter.UTC() {
							t.Fatal("capacidad sin observación o vigencia del certificado")
						}
					} else if capacidad.consultaRRHH != nil || !capacidad.certificadoVerificadoEn.IsZero() || !capacidad.certificadoValidoHasta.IsZero() {
						t.Fatal("se alteró una ruta anterior")
					}
				}),
			}
			middleware.ServeHTTP(httptest.NewRecorder(), peticion)
			if llamadas != 1 {
				t.Fatal("el siguiente manejador no se ejecutó exactamente una vez")
			}
		})
	}
}
