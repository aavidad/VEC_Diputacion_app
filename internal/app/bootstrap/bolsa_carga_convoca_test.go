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
	importacionapp "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
)

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
