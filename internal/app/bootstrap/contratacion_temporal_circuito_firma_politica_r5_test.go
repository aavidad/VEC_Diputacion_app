package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	"vec-diputacion-granada/internal/vec/reglas"
)

const rutaCircuitoFirmaRRHHV2Prueba = "../../../data/demo/reglas/ct_circuito_firma.rrhh.v2.json"

func resolverCircuitoFirmaR5Prueba(t *testing.T, ruta string) *reglas.Resolutor {
	t.Helper()
	consulta, err := fichero.NuevaConsultaCatalogos(ruta)
	if err != nil {
		t.Fatal(err)
	}
	resolutor, err := reglas.NuevoResolutor(reglas.Configuracion{
		Consulta: consulta, Metadatos: consulta,
		CatalogoID: reglas.CatalogoCircuitoFirmaCT, ModuloID: reglas.ModuloContratacionTemporal,
		Reloj: relojFijoAltaContratacionTemporalDesarrollo{ahora: time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatal(err)
	}
	return resolutor
}

func TestFuenteCircuitoFirmaR5ConservaPoliticaAlternativasYProcedencia(t *testing.T) {
	resolutor := resolverCircuitoFirmaR5Prueba(t, rutaCircuitoFirmaRRHHV2Prueba)
	origen, err := resolutor.CircuitoFirma(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	obtenido, err := (fuenteCircuitoFirmaReglasDesarrollo{resolutor: resolutor}).CircuitoFirma(t.Context())
	if err != nil || obtenido.CatalogoRef != "vec.contratacion_temporal.circuito_firma:2" ||
		obtenido.CatalogoVersion != uint64(origen.Version) || obtenido.HuellaCatalogo != origen.HuellaCatalogo || obtenido.PermiteMismaPersonaEnPasos ||
		len(obtenido.Documentos) != 2 || len(obtenido.Documentos[1].Pasos) != 2 {
		t.Fatalf("procedencia o política R5 perdida: %+v, %v", obtenido, err)
	}
	paso := obtenido.Documentos[1].Pasos[0]
	if paso.PerfilRef != "perfil:ct:jefatura_servicio_rrhh" || len(paso.PerfilesAlternativos) != 1 ||
		paso.PerfilesAlternativos[0] != "perfil:ct:direccion_rrhh" ||
		paso.Referencia != origen.Documentos[1].Pasos[0].Referencia || obtenido.Documentos[1].Validar() != nil {
		t.Fatalf("alternativa o procedencia del paso perdida: %+v", paso)
	}
}

func TestFuenteCircuitoFirmaR5SoloPermiteMismaPersonaSiCatalogoLoDeclara(t *testing.T) {
	contenido, err := os.ReadFile(rutaCircuitoFirmaRRHHV2Prueba)
	if err != nil {
		t.Fatal(err)
	}
	declaracion := []byte(`"misma_persona_en_dos_pasos": "false"`)
	if bytes.Count(contenido, declaracion) != 3 {
		t.Fatal("el catálogo debe declarar la política en cada paso")
	}
	ruta := filepath.Join(t.TempDir(), "circuito.json")
	if err := os.WriteFile(ruta, bytes.ReplaceAll(contenido, declaracion,
		[]byte(`"misma_persona_en_dos_pasos": "true"`)), 0600); err != nil {
		t.Fatal(err)
	}
	resolutor := resolverCircuitoFirmaR5Prueba(t, ruta)
	origen, err := resolutor.CircuitoFirma(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	obtenido, err := (fuenteCircuitoFirmaReglasDesarrollo{resolutor: resolutor}).CircuitoFirma(t.Context())
	if err != nil || !origen.PermiteMismaPersonaEnPasos || !obtenido.PermiteMismaPersonaEnPasos ||
		obtenido.CatalogoRef != "vec.contratacion_temporal.circuito_firma:2" || obtenido.HuellaCatalogo != origen.HuellaCatalogo {
		t.Fatalf("política publicada no conservada: %+v, %v", obtenido, err)
	}
}

func TestPerfilPasoFirmaR5ExigeAlternativaYProcedenciaExactas(t *testing.T) {
	resolutor := resolverCircuitoFirmaR5Prueba(t, rutaCircuitoFirmaRRHHV2Prueba)
	circuito, err := resolutor.CircuitoFirma(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	paso := circuito.Documentos[1].Pasos[0]
	material := ports.MaterialFirmaDocumento{
		Documento: "resolucion", CatalogoRef: circuito.CatalogoID + ":2",
		CatalogoHuella: circuito.HuellaCatalogo, PasoOrden: paso.Orden, PasoRef: paso.Referencia,
	}
	if !perfilPasoFirmaDocumentoCTDesarrolloCoincide(material, circuito, paso.PerfilRef) ||
		!perfilPasoFirmaDocumentoCTDesarrolloCoincide(material, circuito, paso.PerfilesAlternativos[0]) {
		t.Fatal("un perfil nominal declarado para el paso exacto debe coincidir")
	}
	for _, perfil := range []string{"", "firma_documento_ct_desarrollo", "perfil:ct:otro"} {
		if perfilPasoFirmaDocumentoCTDesarrolloCoincide(material, circuito, perfil) {
			t.Fatalf("perfil no declarado aceptado: %q", perfil)
		}
	}
	mutado := material
	mutado.CatalogoHuella = "otra_huella"
	if perfilPasoFirmaDocumentoCTDesarrolloCoincide(mutado, circuito, paso.PerfilesAlternativos[0]) {
		t.Fatal("alternativa aceptada con otra huella")
	}
	mutado = material
	mutado.PasoRef = "otro_paso"
	if perfilPasoFirmaDocumentoCTDesarrolloCoincide(mutado, circuito, paso.PerfilesAlternativos[0]) {
		t.Fatal("alternativa aceptada para otro paso")
	}
}

type originalCircuitoVersionadoR5Prueba struct{ llamadas int }

func (f *originalCircuitoVersionadoR5Prueba) ObtenerOriginalFirma(context.Context, ports.SolicitudOriginalFirma) (ports.OriginalFirmaAutorizado, error) {
	f.llamadas++
	return ports.OriginalFirmaAutorizado{}, ports.ErrFuenteOriginalFirmaNoDisponible
}

// El servicio usa el catálogo JSON y la misma copia que compone bootstrap.
// La fuente del original marca hasta dónde llega el consumidor V2: antes del
// parche la versión perdida cancela ambas vías antes de esta dependencia.
func TestFuenteCircuitoVersionadoAlcanzaOriginalEnAmbasViasV2(t *testing.T) {
	for _, ruta := range []string{"../../../data/demo/reglas/ct_circuito_firma.ejemplo.demo.json", rutaCircuitoFirmaRRHHV2Prueba} {
		t.Run(filepath.Base(ruta), func(t *testing.T) {
			resolutor := resolverCircuitoFirmaR5Prueba(t, ruta)
			origen, err := resolutor.CircuitoFirma(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			fuente := fuenteCircuitoFirmaReglasDesarrollo{resolutor: resolutor}
			circuito, err := fuente.CircuitoFirma(t.Context())
			if err != nil || circuito.CatalogoVersion != uint64(origen.Version) || circuito.CatalogoVersion == 0 {
				t.Fatalf("versión publicada perdida: %+v, %v", circuito, err)
			}
			p := baseR5ComposicionPrueba{}
			base, err := ctapp.NuevoServicioFirmaDocumento(fuente, p, p, nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := base.ComponerCustodia(dependenciasR5PresentesPrueba{}, map[string]string{"informe_definitivo": "informe_firmado"}); err != nil {
				t.Fatal(err)
			}
			original := &originalCircuitoVersionadoR5Prueba{}
			d := dependenciasCompletasR5Prueba()
			d.original = original
			montaje := &firmaDocumentoCTDesarrollo{servicio: base, custodiaR5Compuesta: true}
			if err := montaje.componerFirmasR5(d); err != nil {
				t.Fatal(err)
			}
			s := ctapp.SolicitudFirmaVec{OrganizacionRef: "organizacion:desarrollo:dipgra", ExpedienteRef: "expediente:ct:versionado", VersionExpediente: 7, Documento: "informe_definitivo", PasoOrden: 1, OriginalRef: "original:ct:versionado", OriginalVersion: 1, PDFFirmado: []byte("%PDF-1.7\nrevision"), ClaveIdempotencia: "clave-circuito-versionado-001"}
			if _, err := montaje.firmaVec.Firmar(t.Context(), s); !errors.Is(err, ports.ErrFuenteOriginalFirmaNoDisponible) {
				t.Fatalf("VEC no alcanzó original: %v", err)
			}
			externa := ctapp.SolicitudFirmaExterna{OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, VersionExpediente: s.VersionExpediente, Documento: s.Documento, PasoOrden: s.PasoOrden, OriginalRef: s.OriginalRef, OriginalVersion: s.OriginalVersion, PDFFirmado: s.PDFFirmado, ClaveIdempotencia: s.ClaveIdempotencia, ReferenciaPortafirmasDeclarada: "PF-2026-001", FechaPortafirmasDeclarada: "2026-10-02T10:00:00Z"}
			if _, err := montaje.firmaExterna.Registrar(t.Context(), externa); !errors.Is(err, ports.ErrFuenteOriginalFirmaNoDisponible) {
				t.Fatalf("externa no alcanzó original: %v", err)
			}
			if original.llamadas != 2 {
				t.Fatalf("dependencia original consultada %d veces", original.llamadas)
			}
		})
	}
}
