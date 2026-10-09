package bootstrap

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
)

func TestMaterialInscripcionPortalExternoAudienciaSeparada(t *testing.T) {
	if !consumidorPortalExternoValido(consumidorInscripcionPortalExternoV3) ||
		!slices.Equal(audienciasConsumidorPortalExternoV3(consumidorInscripcionPortalExternoV3), []string{audienciaPresentarInscripcionExternaV3}) ||
		accionPresentarInscripcionExternaV3 != inscripcion.AccionPresentar ||
		tipoRecursoPresentarInscripcionExternaV3 != "inscripcion_convocatoria" {
		t.Fatal("consumidor de inscripción sin acción, recurso o audiencia exactos")
	}
	catalogo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(descriptoresMaterialPortalExternoV3())
	d, presente := catalogo.descriptorPara(audienciaPresentarInscripcionExternaV3)
	if err != nil || !presente || d != descriptorMaterialPresentarInscripcionExternaV3() {
		t.Fatalf("descriptor de inscripción ausente o compartido: %v", err)
	}
	publicados := materialesPublicadosPrueba(t, "mi_bolsa", consumidorInscripcionPortalExternoV3)
	directorio := directorioExternoPrueba(t)
	if err := escribirMaterialV3PortalExterno(directorio, publicados); err != nil {
		t.Fatal(err)
	}
	inv, err := leerInventarioV3PortalExterno(directorio)
	if err != nil {
		t.Fatal(err)
	}
	inscripcionMaterial, err := materialesConsumidorV3PortalExterno(directorio, inv, consumidorInscripcionPortalExternoV3, inv.Configuracion)
	if err != nil || len(inscripcionMaterial) != 1 || inscripcionMaterial[0].audienciaConsumo != audienciaPresentarInscripcionExternaV3 {
		t.Fatalf("material de inscripción no recuperado: %v", err)
	}
	miBolsa, err := materialesConsumidorV3PortalExterno(directorio, inv, "mi_bolsa", inv.Configuracion)
	if err != nil || len(miBolsa) != 2 || inscripcionMaterial[0].claveHMACID == miBolsa[0].claveHMACID {
		t.Fatalf("material de inscripción prestado de Mi Bolsa: %v", err)
	}
}

func TestMaterialInscripcionPortalExternoAusenteCierra(t *testing.T) {
	publicados := materialesPublicadosPrueba(t, "mi_bolsa")
	directorio := directorioExternoPrueba(t)
	if err := escribirMaterialV3PortalExterno(directorio, publicados); err != nil {
		t.Fatal(err)
	}
	inv, err := leerInventarioV3PortalExterno(directorio)
	if err != nil {
		t.Fatal(err)
	}
	if material, err := materialesConsumidorV3PortalExterno(directorio, inv, consumidorInscripcionPortalExternoV3, inv.Configuracion); !errors.Is(err, ErrMaterialV3PortalExternoInvalido) || material != nil {
		t.Fatalf("material ausente abrió inscripción: %v", err)
	}
}

func TestACLPortalExternoExigeDoceFuncionesYFirmaTrece(t *testing.T) {
	if strings.Count(firmaSolicitarInscripcionPortalExterno, ",") != 12 ||
		!strings.HasPrefix(firmaSolicitarInscripcionPortalExterno,
			"vec_bolsa_llamamientos.solicitar_inscripcion_v1(text,jsonb,bytea,") {
		t.Fatal("la firma nominal B96 no tiene trece argumentos")
	}
	exactas := funcionesMiBolsaPortalExternoConInscripcion()
	if !funcionesMiBolsaPortalExternoExactas(exactas) {
		t.Fatal("ACL publicada rechazada")
	}
	sinInscripcion := slices.Delete(slices.Clone(exactas), len(exactas)-1, len(exactas))
	if funcionesMiBolsaPortalExternoExactas(sinInscripcion) {
		t.Fatal("faltó B96 y se permitió")
	}
	conExtra := append(slices.Clone(exactas), "funcion_ajena_v1")
	if funcionesMiBolsaPortalExternoExactas(conExtra) {
		t.Fatal("GRANT lateral aceptado")
	}
	conDuplicado := slices.Clone(exactas)
	conDuplicado[0] = conDuplicado[1]
	if funcionesMiBolsaPortalExternoExactas(conDuplicado) {
		t.Fatal("sobrecarga repetida aceptada")
	}
}
