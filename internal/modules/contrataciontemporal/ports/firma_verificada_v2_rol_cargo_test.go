package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

func materialRolCargoPrueba() MaterialFirmaVerificadaV2 {
	h := strings.Repeat("a", 64)
	original, firmado, certificado := strings.Repeat("b", 64), strings.Repeat("c", 64), strings.Repeat("d", 64)
	evidencia := []byte(`[{"orden":1}]`)
	suma := sha256.Sum256(evidencia)
	return MaterialFirmaVerificadaV2{MaterialFirmaExterna: MaterialFirmaExterna{
		Via: ViaFirmaCertificadoVEC, OrganizacionRef: "org_prueba", ExpedienteRef: "exp_prueba", VersionExpediente: 1,
		Documento: "informe_definitivo", CatalogoRef: "catalogo_prueba", CatalogoHuella: h,
		PasoRef: "paso_prueba", PasoOrden: 1, Secuencia: 1, HistoriaHuella: h,
		OriginalRef: "original_prueba", OriginalVersion: 1, OriginalHuella: original, FirmadoHuella: firmado,
		CertificadoHuella: certificado, FirmanteRef: "ref:" + certificado,
		FirmantePrincipalRef: "per_prueba", PerfilFirmanteRef: "perfil_prueba", CargoFirmante: "car_prueba",
		UnidadFirmanteRef: "unidad_prueba", PerfilActivoFirmanteRef: "prf_prueba",
		AsignacionFirmanteRef: "asignacion_prueba", AsignacionFirmanteVersion: 1, AsignacionFirmanteHuella: h,
		VersionRolFirmanteRef: "rol_version_prueba", VersionRolFirmanteHuella: h,
		ControlVigenciaFirmanteRef: "rol_version_prueba", ControlVigenciaFirmanteRevision: 1, ControlVigenciaFirmanteHuella: h,
		AsignacionVigenteDesde: "2026-01-01T00:00:00Z", AsignacionVigenteHasta: "2027-01-01T00:00:00Z",
		PoliticaVerificacion: "politica:vec:firma:verificacion-autonoma:v2", RevocacionEstado: "vigente", SelloTiempoEstado: "no_presente",
		ClaveIdempotencia: "clave-prueba-000001", DocumentoCustodiaRef: "custodia_prueba", DocumentoCustodiaVersion: VersionDocumentoCustodiado,
	}, PerfilActivoOperadorRef: "prf_prueba", CuentaFirmanteRef: "cuenta_prueba",
		VinculoCredencialFirmanteRef: "vinculo_prueba", VinculoCredencialFirmanteRevision: 1, VinculoCredencialFirmanteHuella: h,
		RolIDFirmante: "ct_cargo_tecnico_prueba", CatalogoVersion: 1,
		EntradaDocumentoRef: "original_prueba", EntradaDocumentoVersion: 1, EntradaDocumentoLongitud: 100, EntradaDocumentoHuella: original,
		OrdenFirmaPDF: 1, ByteRange: [4]uint64{0, 120, 180, 20}, RevisionHuellaSHA256: firmado,
		ContenidoFirmadoHuellaSHA256: h, RevisionLongitud: 200,
		EvidenciaFirmasCanonica: evidencia, EvidenciaFirmasHuellaSHA256: hex.EncodeToString(suma[:])}
}

func TestMaterialFirmaVerificadaV2ValidaRolYCargoIndependientes(t *testing.T) {
	valido := materialRolCargoPrueba()
	if err := valido.Validar(); err != nil {
		t.Fatalf("rol y cargo distintos del plan: %v", err)
	}
	igual := valido
	igual.CargoFirmante = igual.RolIDFirmante
	if err := igual.Validar(); err != nil {
		t.Fatalf("la forma no impone igualdad ni diferencia: %v", err)
	}
	for nombre, cambiar := range map[string]func(*MaterialFirmaVerificadaV2){
		"rol_vacio":      func(m *MaterialFirmaVerificadaV2) { m.RolIDFirmante = "" },
		"rol_invalido":   func(m *MaterialFirmaVerificadaV2) { m.RolIDFirmante = "rol con espacios" },
		"cargo_vacio":    func(m *MaterialFirmaVerificadaV2) { m.CargoFirmante = "" },
		"cargo_invalido": func(m *MaterialFirmaVerificadaV2) { m.CargoFirmante = "cargo con espacios" },
	} {
		t.Run(nombre, func(t *testing.T) {
			m := valido
			cambiar(&m)
			if !errors.Is(m.Validar(), ErrSolicitudFirmaDocumentoInvalida) {
				t.Fatal("identificador malformado admitido")
			}
		})
	}
}
