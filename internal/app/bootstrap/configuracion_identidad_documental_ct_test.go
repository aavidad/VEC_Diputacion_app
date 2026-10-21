package bootstrap

import (
	"strings"
	"testing"
)

func configuracionIdentidadDocumentalCTPrueba() archivoIdentidadDocumentalCT {
	return archivoIdentidadDocumentalCT{
		Esquema:           "vec.contratacion-temporal.identidad-documental-servidor.v1",
		PerfilActivoRef:   "prf_0123456789abcdefghijkl",
		VersionRolRef:     "rol:identidad_documental_ct_gestion_dev:v1",
		ActoAsignacionRef: "acto:admin:documental:demo",
		PrincipalID:       "per_0123456789abcdefghijkl", CuentaRef: "cuenta:gestion:demo",
		CertificadoSHA256: strings.Repeat("a", 64), UnidadRef: "unidad:demo",
		Motivo:   motivoOriginalFirmableCTDesarrollo(),
		DSNFiles: map[string]string{"bolsa": "bolsa.dsn", "fuente_autorizacion": "fuente.dsn", "motivos_autorizacion": "motivos.dsn"},
	}
}

func TestIdentidadDocumentalCTConfiguracionSoloActosYArchivosPrivados(t *testing.T) {
	c := configuracionIdentidadDocumentalCTPrueba()
	if !c.valida() {
		t.Fatal("coordenadas válidas rechazadas")
	}
	for _, caso := range []string{"rol", "perfil", "certificado", "dsn", "acto"} {
		otro := c
		switch caso {
		case "rol":
			otro.VersionRolRef = "rol:tecnico_rrhh_desarrollo:v2"
		case "perfil":
			otro.PerfilActivoRef = ""
		case "certificado":
			otro.CertificadoSHA256 = ""
		case "dsn":
			otro.DSNFiles = map[string]string{"bolsa": "bolsa.dsn"}
		case "acto":
			otro.ActoAsignacionRef = ""
		}
		if otro.valida() {
			t.Fatalf("%s abrió configuración incompleta", caso)
		}
	}
	if _, _, err := leerConfiguracionIdentidadDocumentalCT(""); err == nil {
		t.Fatal("archivo opcional ausente pareció instalado")
	}
}
