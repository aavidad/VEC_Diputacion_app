package bootstrap

import (
	"context"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"vec-diputacion-granada/config"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/vec/adapters/conservacion"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestCustodiaFirmaValidaMaterialAntesDeProvisionar(t *testing.T) {
	cfg := config.Config{ExecutionProfile: config.ExecutionProfileDevelopment, AuthMode: config.AuthModeDevelopment,
		DevelopmentGuard: config.DevelopmentGuardAcknowledgement, DevelopmentMaterialDir: t.TempDir()}
	if err := os.Chmod(cfg.DevelopmentMaterialDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(cfg.DevelopmentMaterialDir, "identidad"), 0700); err != nil {
		t.Fatal(err)
	}
	if m, err := configuracionCustodiaFirmaCTDesarrollo(cfg); err != nil || m != nil {
		t.Fatal(m, err)
	}
	cfg.DocumentosEnabled = "true"
	if _, err := configuracionCustodiaFirmaCTDesarrollo(cfg); err == nil {
		t.Fatal("sin material se concedería custodia")
	}
	c := configuracionDocumentosDesarrollo{Version: 1, Autoridad: AutoridadNoAutoritativa,
		Cuentas:         []cuentaRutasDietasDesarrollo{{Sujeto: "sujeto:prueba"}},
		CustodiaFirmado: &custodiaFirmadoConfigDesarrollo{Documentos: map[string]string{"resolucion": "contratacion_temporal.resolucion_firmada.v1"}}}
	c.Motivos.Listar = motivoFirmaDocumentoCTDesarrollo()
	ruta := filepath.Join(cfg.DevelopmentMaterialDir, "identidad", ficheroMaterialDocumentos)
	escribir := func(c configuracionDocumentosDesarrollo) {
		t.Helper()
		b, err := json.Marshal(map[string]any{"version": c.Version, "autoridad": c.Autoridad, "cuentas": c.Cuentas, "motivos": map[string]any{"listar": c.Motivos.Listar}, "custodia_firmado": c.CustodiaFirmado})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(ruta, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	escribir(c)
	m, err := configuracionCustodiaFirmaCTDesarrollo(cfg)
	if err != nil || !maps.Equal(m, c.CustodiaFirmado.Documentos) {
		t.Fatal(m, err)
	}
	for _, invalida := range []string{"contratacion_temporal.borrador.v1", "tipo:desconocido"} {
		c.CustodiaFirmado.Documentos["resolucion"] = invalida
		escribir(c)
		if _, err := configuracionCustodiaFirmaCTDesarrollo(cfg); err == nil {
			t.Fatal("tipo no reservado aceptado")
		}
	}
	if m["resolucion"] != "contratacion_temporal.resolucion_firmada.v1" {
		t.Fatal("mapa validado mutado")
	}
	c.CustodiaFirmado = nil
	escribir(c)
	if m, err := configuracionCustodiaFirmaCTDesarrollo(cfg); err != nil || m != nil {
		t.Fatal(m, err)
	}
	c.Version = 2
	escribir(c)
	if _, err := configuracionCustodiaFirmaCTDesarrollo(cfg); err == nil {
		t.Fatal("versión desconocida aceptada")
	}
	cfg.DocumentosEnabled = "si"
	if _, err := configuracionCustodiaFirmaCTDesarrollo(cfg); err == nil {
		t.Fatal("selector desconocido aceptado")
	}
}

func TestMontajeCustodiaNoMutaPlantillaYRechazaCambioDeConfiguracion(t *testing.T) {
	s, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ahora := s.reloj.Ahora()
	p, err := nuevoPerfilFijoCTDesarrollo(principal, s.contexto, ahora, clavePerfilFijoFirmaCTDesarrollo,
		[]string{httpinterno.RutaFirmaDocumento, httpinterno.RutaConsultaFirmaDocumento},
		func(actor, ref string) (core.InstantaneaAutorizacion, error) {
			return instantaneaPerfilFijoFirmaDocumentoCTDesarrollo(actor, ref, ahora, true)
		})
	if err != nil || s.registrarPerfilFijoCTDesarrollo(p) != nil {
		t.Fatal(err)
	}
	plantilla := clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(p.plantilla)
	catalogo, err := conservacion.NuevoCatalogoProvisional(relojRutasDietas{})
	if err != nil {
		t.Fatal(err)
	}
	c := &custodiaFirmadoConfigDesarrollo{Documentos: map[string]string{"resolucion": "contratacion_temporal.resolucion_firmada.v1"}}
	d, err := nuevaCustodiaDocumentosDesarrollo(c, repositorioCustodiaNoUsado{}, almacenNoUsado{}, catalogo, relojRutasDietas{}, nuevoSeudonimizadorAlmacenDesarrollo([32]byte{1}))
	if err != nil {
		t.Fatal(err)
	}
	s.custodiaFirmaDocumentos = maps.Clone(d.documentos)
	f := &firmaDocumentoCTDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{soporte: s}, servicio: &ctapp.ServicioFirmaDocumento{}}
	if err := f.componerCustodia(&autoridadDocumentosDesarrollo{}); err == nil {
		t.Fatal("configuración de custodia desaparecida tras provisionar admitida")
	}
	d.documentos["resolucion"] = "contratacion_temporal.borrador.v1"
	if err := f.componerCustodia(&autoridadDocumentosDesarrollo{custodia: d, reloj: relojRutasDietas{}}); err == nil {
		t.Fatal("mapa cambiado tras provisionar admitido")
	}
	d.documentos["resolucion"] = s.custodiaFirmaDocumentos["resolucion"]
	if err := f.componerCustodia(&autoridadDocumentosDesarrollo{custodia: d, reloj: relojRutasDietas{}}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(p.plantilla, plantilla) {
		t.Fatal("montaje alteró plantilla fija")
	}
	a := s.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	if a.preparadas != 0 || a.publicadas != 0 {
		t.Fatal("montaje publicó concesiones")
	}
}

func TestCustodiaConsumePerfilFijoSinPublicarYDeniegaRevocado(t *testing.T) {
	for _, revocada := range []bool{false, true} {
		s, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
		ahora := s.reloj.Ahora()
		p, err := nuevoPerfilFijoCTDesarrollo(principal, s.contexto, ahora, clavePerfilFijoFirmaCTDesarrollo,
			[]string{httpinterno.RutaFirmaDocumento, httpinterno.RutaConsultaFirmaDocumento},
			func(actor, ref string) (core.InstantaneaAutorizacion, error) {
				return instantaneaPerfilFijoFirmaDocumentoCTDesarrollo(actor, ref, ahora, true)
			})
		if err != nil || s.registrarPerfilFijoCTDesarrollo(p) != nil {
			t.Fatal(err)
		}
		a := s.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
		i := clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(p.plantilla)
		if revocada {
			i.AsignacionPerfil.Estado = core.EstadoAsignacionPerfilRevocada
			i.AsignacionPerfil.RevocadaEn = ahora
			i.AsignacionPerfil.RevocadaPor = "revocador:prueba"
			i.AsignacionPerfil.RevocacionRef = "revocacion:prueba"
		}
		a.asignaciones = map[string]instantaneaPublicadaDesarrollo{p.perfilRef(): {instantanea: i, actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo}}
		e := esperadoCustodiaPrueba()
		e.fijar(strings.Repeat("a", 64), "")
		preimagen := preimagenCustodiaPrueba(e, nil)
		recurso, err := docports.RecursoV3(docports.AccionCustodiarFirmado, e.documentoRef, preimagen)
		if err != nil {
			t.Fatal(err)
		}
		recurso.Atributos["preimagen_sha256"] = strings.Repeat("a", 64)
		datos := core.DatosSolicitudAutorizacionLigadaV3{Accion: docports.AccionCustodiarFirmado, Finalidad: docports.FinalidadCustodiarFirmado, ReferenciaMotivo: motivoFirmaDocumentoCTDesarrollo(), Recurso: recurso}
		ctx := contextoRutaCoberturaDesarrolloPrueba(s, principal, httpinterno.RutaFirmaDocumento)
		ctx = context.WithValue(ctx, claveCustodiaFirmadoCTDesarrollo{}, e)
		ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
		_, ok := s.instantaneaParaContexto(ctx, httpinterno.RutaFirmaDocumento)
		if ok == revocada {
			t.Fatalf("revocada=%v consumida=%v", revocada, ok)
		}
		if a.preparadas != 0 || a.publicadas != 0 {
			t.Fatal("petición publicó permisos")
		}
		if _, ok := s.instantaneaParaContexto(context.WithValue(ctx, claveCustodiaFirmadoCTDesarrollo{}, (*esperadoCustodiaFirmadoCTDesarrollo)(nil)), httpinterno.RutaFirmaDocumento); ok {
			t.Fatal("custodia sin contexto exacto admitida")
		}
	}
}
