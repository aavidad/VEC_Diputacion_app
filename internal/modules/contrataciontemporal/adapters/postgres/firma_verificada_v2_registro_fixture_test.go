package postgres

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type descriptorFirmaV2Prueba struct {
	descriptor DescriptorConstructorFirmaV2
	fallo      error
}

func (f descriptorFirmaV2Prueba) DescriptorFirmaV2(context.Context, ports.MaterialFirmaVerificadaV2) (DescriptorConstructorFirmaV2, error) {
	return f.descriptor, f.fallo
}

func fixtureRegistroFirmaV2(t *testing.T) (ports.MaterialFirmaVerificadaV2, DescriptorConstructorFirmaV2) {
	t.Helper()
	h := strings.Repeat("a", 64)
	cert := strings.Repeat("b", 64)
	original := strings.Repeat("c", 64)
	firmado := strings.Repeat("d", 64)
	evidencia := []byte(`[{"orden":1}]`)
	hash := sha256.Sum256(evidencia)
	m := ports.MaterialFirmaVerificadaV2{MaterialFirmaExterna: ports.MaterialFirmaExterna{
		Via: ports.ViaFirmaCertificadoVEC, OrganizacionRef: "organizacion:prueba", ExpedienteRef: "expediente:prueba", VersionExpediente: 7,
		Documento: "informe_definitivo", CatalogoRef: "catalogo:prueba", CatalogoHuella: h, PasoRef: "paso:primero", PasoOrden: 1, Secuencia: 1,
		HistoriaHuella: h, OriginalRef: "original:prueba", OriginalVersion: 1, OriginalHuella: original, FirmadoHuella: firmado, CertificadoHuella: cert, FirmanteRef: "ref:" + cert,
		FirmantePrincipalRef: "per_firmante_sintetico", PerfilFirmanteRef: "perfil:firmante", CargoFirmante: "ct_cargo_prueba", UnidadFirmanteRef: "unidad:prueba", PerfilActivoFirmanteRef: "prf_firmante_sintetico",
		AsignacionFirmanteRef: "asignacion:prueba", AsignacionFirmanteVersion: 1, AsignacionFirmanteHuella: h, VersionRolFirmanteRef: "rol:ct_cargo_prueba:v1", VersionRolFirmanteHuella: h,
		ControlVigenciaFirmanteRef: "rol:ct_cargo_prueba:v1", ControlVigenciaFirmanteRevision: 1, ControlVigenciaFirmanteHuella: h,
		AsignacionVigenteDesde: "2026-01-01T00:00:00Z", AsignacionVigenteHasta: "2027-01-01T00:00:00Z", PoliticaVerificacion: "politica:vec:firma:verificacion-autonoma:v2",
		RevocacionEstado: "vigente", SelloTiempoEstado: "no_presente", ClaveIdempotencia: "clave-prueba-firma-000001", DocumentoCustodiaRef: "custodia:prueba", DocumentoCustodiaVersion: 1},
		PerfilActivoOperadorRef: "prf_firmante_sintetico", CuentaFirmanteRef: "cuenta:prueba", VinculoCredencialFirmanteRef: "vinculo:prueba", VinculoCredencialFirmanteRevision: 1, VinculoCredencialFirmanteHuella: h,
		RolIDFirmante: "ct_cargo_prueba", CatalogoVersion: 1, EntradaDocumentoRef: "original:prueba", EntradaDocumentoVersion: 1, EntradaDocumentoLongitud: 100, EntradaDocumentoHuella: original,
		OrdenFirmaPDF: 1, ByteRange: [4]uint64{0, 120, 180, 20}, RevisionHuellaSHA256: firmado, ContenidoFirmadoHuellaSHA256: h, RevisionLongitud: 200,
		EvidenciaFirmasCanonica: evidencia, EvidenciaFirmasHuellaSHA256: hex.EncodeToString(hash[:]), ComprobadaEn: time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)}
	if m.Validar() != nil {
		t.Fatal("fixture material no válido")
	}
	d := DescriptorConstructorFirmaV2{Esquema: "vec.competencia-firmante.constructor-ct.v1", CertificadoDERSHA256: cert,
		Seleccion: SeleccionConstructorFirmaV2{PerfilEsperadoRef: m.PerfilFirmanteRef, PerfilActivoRef: m.PerfilActivoFirmanteRef, RolID: m.RolIDFirmante, CargoRef: "cargo:prueba", EnlaceEjercicioRef: "enlace:prueba"},
		Recurso: vd.RecursoFirmaHistoricaV1{OrganizacionRef: m.OrganizacionRef, UnidadRef: m.UnidadFirmanteRef, ExpedienteRef: m.ExpedienteRef, DocumentoRef: m.OriginalRef, RecursoAutorizableRef: m.OriginalRef,
			ModuloID: ports.ModuloContratacion, TipoRecurso: "documento_contratacion_temporal", RecursoContextoSHA256: strings.Repeat("e", 64),
			Original: vd.ReferenciaHistoricaCompetenciaV1{Referencia: m.OriginalRef, Version: 1, HuellaSHA256: original}, PDFRaizSHA256: original,
			Firmado: vd.ReferenciaHistoricaCompetenciaV1{Referencia: m.DocumentoCustodiaRef, Version: 1, HuellaSHA256: firmado}, PDFFirmadoSHA256: firmado, NumeroFirmas: 1},
		Accion: "contratacion_temporal.documento.firmar", Finalidad: "formalizacion", Motivo: vd.ReferenciaEntradaCatalogo{CatalogoID: "motivos", CatalogoVersion: 1, CatalogoHuellaSHA256: h, EntradaClave: "firma"},
		Circuito: vd.ReferenciaHistoricaCompetenciaV1{Referencia: m.CatalogoRef, Version: 1, HuellaSHA256: h}, PasoRef: m.PasoRef, PasoOrden: 1}
	return m, d
}

func capacidadRegistroFirmaV2Prueba(t *testing.T, m ports.MaterialFirmaVerificadaV2) ports.CapacidadFirmaVerificadaV2 {
	t.Helper()
	_, d := fixtureRegistroFirmaV2(t)
	datos, err := ctapp.CanonicoDescriptorFirmaVerificadaV2(m, d)
	if err != nil {
		t.Fatal(err)
	}
	r, err := ctapp.RecursoFirmaVerificadaV2(m, datos)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	h := strings.Repeat("a", 64)
	accion, audiencia := ports.AccionRegistrarFirmaVec, ports.AudienciaFirmaVecV2
	if m.Via == ports.ViaFirmaExternaPortafirmas {
		accion, audiencia = ports.AccionRegistrarFirmaExterna, ports.AudienciaFirmaExternaV2
	}
	resumen, err := vp.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:ct172:registro", h, h, "contexto:ct172:registro", h,
		accion, m.RecursoRef(), huella, audiencia, m.ComprobadaEn, m.ComprobadaEn.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize)).Public())
	if err != nil {
		t.Fatal(err)
	}
	x, err := vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3([]byte(strings.Repeat("x", 512)), resumen, []byte("{}"), []byte("{}"), []byte("{}"), 1, 1, []byte("unidad"), []byte("unidad"), []byte("unidad"), spki)
	if err != nil {
		t.Fatal(err)
	}
	hDescriptor := sha256.Sum256(datos)
	return ports.TransportarMaterialFirmaVerificadaV2ConDescriptor(x, datos, hex.EncodeToString(hDescriptor[:]))
}
