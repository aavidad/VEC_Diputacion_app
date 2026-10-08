package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	xls "vec-diputacion-granada/internal/modules/bolsa/adapters/xlsconvoca"
	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	importacionapp "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
)

func TestVistaPreviaCargaConvocaRechazaExtensionAjenaAlContenido(t *testing.T) {
	vista, err := aplicacionbolsa.NuevoPrevisualizadorCargaConvoca(xls.NuevoLector())
	if err != nil {
		t.Fatal(err)
	}
	operador := operadorCargaConvocaBolsa{vista: vista}
	lectura := vistaAutorizadaCargaConvocaBolsa{servicio: &aplicacionbolsa.ServicioVistaPreviaCargaConvocaAutorizada{}}
	for _, caso := range []struct{ ruta, nombre string }{
		{filepath.Join("..", "..", "modules", "bolsa", "application", "testdata", "carga_convoca", "carga_convoca_ejemplo.xlsx"), "acta.xls"},
		{filepath.Join("..", "..", "modules", "bolsa", "adapters", "xlsconvoca", "testdata", "xls_sinteticos", "resumen.xls"), "acta.xlsx"},
	} {
		contenido, err := os.ReadFile(caso.ruta)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := operador.Previsualizar(context.Background(), caso.nombre, contenido); !errors.Is(err, aplicacionbolsa.ErrFicheroCargaConvocaInvalido) {
			t.Fatalf("%s como %s: %v", caso.ruta, caso.nombre, err)
		}
		if _, err := lectura.Preparar(context.Background(), puertosbolsa.SolicitudVistaPreviaCargaConvoca{
			NombreFichero: caso.nombre, Contenido: contenido,
		}); !errors.Is(err, aplicacionbolsa.ErrFicheroCargaConvocaInvalido) {
			t.Fatalf("lectura autorizada %s como %s: %v", caso.ruta, caso.nombre, err)
		}
	}
}

func TestPreparadorVistaPreviaCargaConvocaPermaneceCerradoSinSesion(t *testing.T) {
	p := &preparadorCargaConvocaBolsa{}
	if err := p.PrepararVistaPreviaCargaConvoca(context.Background()); !errors.Is(err, puertosbolsa.ErrCargaConvocaNoDisponible) {
		t.Fatalf("la vista previa se habilitó sin sesión: %v", err)
	}
}

func TestCargaConvocaTieneFronterasAccionYMaterialPropios(t *testing.T) {
	perfil := "prf_bolsa_bback"
	fronteras, err := descriptoresFronterasBorradorLlamamientoBolsaDesarrollo(perfil, false, false, true)
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo(fronteras)
	if err != nil {
		t.Fatal(err)
	}
	politica := politicaDescriptoresBolsaPrueba(t)
	autorizaciones, err := descriptoresAutorizacionBorradorLlamamientoBolsaDesarrollo(politica, false, false, true)
	if err != nil {
		t.Fatal(err)
	}
	accesos, err := nuevoCatalogoAutorizacionComunDesarrollo(catalogo, autorizaciones)
	if err != nil {
		t.Fatal(err)
	}
	for _, ruta := range []string{bolsahttp.RutaVistaPreviaCargaConvoca, bolsahttp.RutaConfirmarCargaConvoca} {
		f, ok := catalogo.resolver(http.MethodPost, ruta)
		if !ok || f.ClaveCapacidad != claveCapacidadCargaConvocaBolsa || len(f.PerfilesActivosRef) != 1 || f.PerfilesActivosRef[0] != perfil {
			t.Fatalf("frontera B1 %s incompleta: %+v", ruta, f)
		}
		if _, ok := accesos.politicaPara(puertosbolsa.AccionConfirmarCargaConvoca, f.Clave, f.ClavePolitica, f.ClaveCapacidad); !ok {
			t.Fatalf("B1 %s sin política nominal", ruta)
		}
		if _, ok := accesos.politicaPara(puertosbolsa.AccionEmitirLlamamiento, f.Clave, f.ClavePolitica, f.ClaveCapacidad); ok {
			t.Fatalf("B1 %s heredó permiso de emisión", ruta)
		}
	}
	d := descriptorMaterialCargaConvocaBolsaDesarrollo()
	if d.Audiencia != puertosbolsa.AudienciaConfirmarCargaConvoca || d.Dominio == dominioMaterialEmisionLlamamientoBolsa || d.Prefijo == prefijoMaterialEmisionLlamamientoBolsa {
		t.Fatal("B1 comparte material con otro efecto")
	}
	if !audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(d.Audiencia) {
		t.Fatal("gobierno CT no puede publicar la audiencia B1")
	}
	if _, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(append(descriptoresMaterialBorradorLlamamientoBolsaDesarrollo(), d)); err != nil {
		t.Fatal(err)
	}
}

func TestCargaConvocaNoSeDeclaraEnCatalogoActivoSinPlantilla(t *testing.T) {
	fronteras, err := descriptoresFronterasBorradorLlamamientoBolsaDesarrollo("prf_bolsa_bback", false, false)
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo(fronteras)
	if err != nil {
		t.Fatal(err)
	}
	for _, ruta := range []string{bolsahttp.RutaVistaPreviaCargaConvoca, bolsahttp.RutaConfirmarCargaConvoca} {
		if _, ok := catalogo.resolver(http.MethodPost, ruta); ok {
			t.Fatalf("B1 expuesta sin plantilla: %s", ruta)
		}
	}
	autorizaciones, err := descriptoresAutorizacionBorradorLlamamientoBolsaDesarrollo(politicaDescriptoresBolsaPrueba(t), false, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, descriptor := range autorizaciones {
		if descriptor.Accion == puertosbolsa.AccionConfirmarCargaConvoca {
			t.Fatal("B1 autoriza sin plantilla")
		}
	}
}

func TestPreparadorCargaConvocaTomaContextoYCategoriaDelServidor(t *testing.T) {
	e := nuevaSesionConsultaPrueba(t)
	directorio := t.TempDir()
	if err := os.Mkdir(filepath.Join(directorio, "identidad"), 0700); err != nil {
		t.Fatal(err)
	}
	escribirManifiestoIdentidadBorradorBolsa(t, directorio, e.principal, e.reloj.Ahora(), nil)
	bolsa, err := nuevoSoporteSesionBorradorBolsaDesarrollo(directorio, e.soporte, e.reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	e.resolutor.base = bolsa.soporteCanal.contexto.Resultado
	fronteras, err := descriptoresFronterasBorradorLlamamientoBolsaDesarrollo(bolsa.soporteCanal.contexto.Resultado.Contexto.PerfilActivoRef, false, false, true)
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo(fronteras)
	if err != nil {
		t.Fatal(err)
	}
	bolsa.soporteCanal.contextoEsperadoRegistrado = bolsa.soporteCanal.contexto.Resultado
	proveedor, err := nuevoProveedorSesionConsultaRRHHConCatalogoDesarrollo(bolsa.soporteCanal, e.registro, e.revalidador, e.reloj, e.resolutor, catalogo)
	if err != nil {
		t.Fatal(err)
	}
	seguridad, err := nuevaSeguridadComunDesarrollo(proveedor, e.reloj)
	if err != nil {
		t.Fatal(err)
	}
	base := &preparadorBorradorLlamamientoDesarrollo{sesion: seguridad, soporte: bolsa, generar: seguridadvec.GeneradorReferenciasCriptograficas{}}
	preparador := &preparadorCargaConvocaBolsa{base: base, cfg: config.Config{RPTCatalogoPath: rutaRPTRepositorioPrueba}}
	ruta := bolsahttp.RutaConfirmarCargaConvoca
	ctx := contextoRutaCoberturaDesarrolloPrueba(bolsa.soporteCanal, e.principal, ruta)
	capacidad := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	capacidad.metodo = http.MethodPost
	capacidad.certificadoVerificadoEn = e.reloj.Ahora().Add(-time.Second)
	capacidad.certificadoValidoHasta = e.reloj.Ahora().Add(time.Minute)
	ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	descriptor, ok := catalogo.resolver(http.MethodPost, ruta)
	if !ok {
		t.Fatal("ruta B1 sin frontera")
	}
	ctx = context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{metodo: http.MethodPost, ruta: ruta, superficie: superficieInternaSeguridadComunDesarrollo, catalogo: catalogo, descriptor: descriptor})
	ctx = context.WithValue(ctx, claveIntentoCargaConvocaBolsa{}, &intentoCargaConvocaBolsa{})
	entrada := bolsahttp.EntradaConfirmarCargaConvoca{CategoriaClave: "auxiliar-administrativo", NombreFichero: filepath.Base("acta.xlsx"), Contenido: []byte("contenido")}
	solicitud, err := preparador.PrepararConfirmacionCargaConvoca(ctx, entrada)
	if err != nil || solicitud.Validar() != nil || solicitud.CategoriaRef != "categoria:rpt:auxiliar-administrativo" ||
		solicitud.ResultadoContexto.Contexto.PerfilActivoRef != bolsa.soporteCanal.contexto.Resultado.Contexto.PerfilActivoRef ||
		solicitud.MotivoAutorizacion != motivoConfirmarCargaConvocaBolsaDesarrollo() || solicitud.BolsaRef != "" {
		t.Fatalf("solicitud B1 no procede del servidor: err=%v categoria=%q bolsa=%q", err, solicitud.CategoriaRef, solicitud.BolsaRef)
	}
	entrada.CategoriaClave = "inventada"
	if _, err := preparador.PrepararConfirmacionCargaConvoca(ctx, entrada); err != bolsahttp.ErrCategoriaCargaConvocaNoValida {
		t.Fatalf("categoría ajena al RPT aceptada: %v", err)
	}
	ctxVista := contextoRutaCoberturaDesarrolloPrueba(bolsa.soporteCanal, e.principal, bolsahttp.RutaVistaPreviaCargaConvoca)
	capacidadVista := ctxVista.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	capacidadVista.metodo = http.MethodPost
	capacidadVista.certificadoVerificadoEn = e.reloj.Ahora().Add(-time.Second)
	capacidadVista.certificadoValidoHasta = e.reloj.Ahora().Add(time.Minute)
	ctxVista = context.WithValue(ctxVista, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidadVista)
	fronteraVista, ok := catalogo.resolver(http.MethodPost, bolsahttp.RutaVistaPreviaCargaConvoca)
	if !ok {
		t.Fatal("vista previa B1 sin frontera")
	}
	ctxVista = context.WithValue(ctxVista, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
		metodo: http.MethodPost, ruta: bolsahttp.RutaVistaPreviaCargaConvoca,
		superficie: superficieInternaSeguridadComunDesarrollo, catalogo: catalogo, descriptor: fronteraVista})
	ctxVista = context.WithValue(ctxVista, claveIntentoCargaConvocaBolsa{}, &intentoCargaConvocaBolsa{})
	if err := preparador.PrepararVistaPreviaCargaConvoca(ctxVista); err != nil {
		t.Fatalf("sesión B1 de lectura no verificada: %v", err)
	}
	qVista, err := preparador.PrepararSolicitudVistaPreviaCargaConvoca(ctxVista, bolsahttp.EntradaVistaPreviaCargaConvoca{
		NombreFichero: "acta.xlsx", Contenido: []byte("contenido"), CategoriaClave: "auxiliar-administrativo",
		Pagina: puertosbolsa.PaginaVistaPreviaCargaConvoca{Filtro: "todas", Limite: 50},
	})
	if err != nil || aplicacionbolsa.ValidarSolicitudVistaPreviaCargaConvoca(qVista) != nil || qVista.CategoriaRef != solicitud.CategoriaRef ||
		qVista.ResultadoContexto.Contexto.Principal.ID == "" {
		t.Fatalf("solicitud de lectura B1 sin identidad/categoría RPT: error=%v valida=%v categoria=%q principal_vacio=%t",
			err, aplicacionbolsa.ValidarSolicitudVistaPreviaCargaConvoca(qVista), qVista.CategoriaRef, qVista.ResultadoContexto.Contexto.Principal.ID == "")
	}
	sumaVista := sha256.Sum256([]byte("contenido"))
	if actaVista, ok := recursoActaCargaConvocaBolsa(ctxVista); !ok ||
		actaVista != importacionapp.ReferenciaActa(hex.EncodeToString(sumaVista[:]), qVista.CategoriaRef) {
		t.Fatalf("recurso de lectura no ligado al acta real: %q", actaVista)
	}
	if _, err := preparador.PrepararSolicitudVistaPreviaCargaConvoca(ctxVista, bolsahttp.EntradaVistaPreviaCargaConvoca{
		NombreFichero: "acta.xlsx", Contenido: []byte("contenido"), CategoriaClave: "inventada",
		Pagina: puertosbolsa.PaginaVistaPreviaCargaConvoca{Filtro: "todas", Limite: 50},
	}); !errors.Is(err, bolsahttp.ErrCategoriaCargaConvocaNoValida) {
		t.Fatalf("la lectura aceptó categoría ajena al RPT: %v", err)
	}
	seguridadCapturada, correlacionCapturada, ok := intentoVerificadoCargaConvocaBolsa(ctx)
	if !ok || seguridadCapturada.Resultado.Validar() != nil || correlacionCapturada.Validar() != nil {
		t.Fatal("el intento perdió identidad o correlación verificadas")
	}
	registrador := &registradorIntentosBaremoPrueba{}
	auditor := &auditorCargaConvocaBolsa{preparador: base, registrador: registrador, proceso: "vec-server"}
	// Simula una revocación posterior: auditar el fallo conserva el contexto
	// verificado y no solicita otra sesión.
	base.sesion = nil
	if err := auditor.RegistrarIntentoFallidoCargaConvoca(ctx, bolsahttp.OperacionConfirmarCargaConvoca, errors.New("fallo de prueba")); err != nil {
		t.Fatalf("auditoría tras revocación: %v", err)
	}
	if len(registrador.ordenes) != 1 {
		t.Fatalf("intentos auditados = %d", len(registrador.ordenes))
	}
	datos, err := registrador.ordenes[0].Datos()
	if err != nil {
		t.Fatal(err)
	}
	correlacion, _ := correlacionCapturada.ValorCanonico()
	huella := sha256.Sum256(entrada.Contenido)
	acta := importacionapp.ReferenciaActa(hex.EncodeToString(huella[:]), solicitud.CategoriaRef)
	if datos.Datos.CorrelacionRef != correlacion || datos.Datos.RecursoRef != acta {
		t.Fatalf("auditoría no conservó recurso/correlación: %+v", datos.Datos)
	}
}
