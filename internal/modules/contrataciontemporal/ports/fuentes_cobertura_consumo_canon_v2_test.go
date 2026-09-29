package ports

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func borradorCatalogoConsumoCanonPrueba() domain.BorradorCatalogoViasCobertura {
	publicada := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	return domain.BorradorCatalogoViasCobertura{
		Referencia: "catalogo_cobertura_canon_prueba", Version: 1,
		PublicadoEn:    publicada,
		Vigencia:       domain.VigenciaCatalogoCobertura{Desde: publicada},
		ProcedenciaRef: "resolucion_catalogo_canon_prueba",
		Vias: []domain.DefinicionViaCobertura{{
			Clave: "via_gobernada", Orden: 1,
			Comprobaciones: []domain.ComprobacionExigibleCobertura{{
				Clave: "comprobacion_via", Orden: 1, Obligatoria: true,
				Procedencia: domain.ProcedenciaComprobacionCobertura{
					Clave:               "fuente_gobernada",
					DefinicionFuenteRef: "definicion_fuente_canon_prueba",
				},
			}},
		}},
	}
}

func canonConsumoCatalogoPrueba(
	t *testing.T,
	borrador domain.BorradorCatalogoViasCobertura,
) ([]byte, domain.PublicacionCatalogoViasCobertura) {
	t.Helper()
	catalogo, err := domain.PublicarCatalogoViasCobertura(borrador)
	if err != nil {
		t.Fatal(err)
	}
	publicacion := catalogo.Publicacion()
	confirmacion, err := NuevaConfirmacionPublicacionCobertura(
		"publicador_catalogo_canon_prueba", publicacion,
		time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	canon, err := canonCatalogoConsumoCobertura(confirmacion)
	if err != nil {
		t.Fatal(err)
	}
	return canon, publicacion
}

func TestCanonConsumoCatalogoV1ConservaBytes(t *testing.T) {
	canon, publicacion := canonConsumoCatalogoPrueba(
		t, borradorCatalogoConsumoCanonPrueba(),
	)
	if publicacion.Canon != domain.CanonHuellaCatalogoCoberturaV1() {
		t.Fatal("la publicacion sin metadatos debe conservar V1")
	}
	suma := sha256.Sum256(canon)
	const goldenV1 = "ba47761e5aaff2540a1ba54c09f3a27997262234ea0255a278a7f9e2db675d98"
	if obtenido := hex.EncodeToString(suma[:]); obtenido != goldenV1 {
		t.Fatalf("canon V1 alterado: %s", obtenido)
	}
}

func TestCanonConsumoCatalogoV2SellaDocumentosYDatosExactos(t *testing.T) {
	borrador := borradorCatalogoConsumoCanonPrueba()
	borrador.EsEjemplo = true
	borrador.Vias[0].Documentos = []domain.ElementoPreparacionViaCobertura{
		{Clave: "documento_segundo", Orden: 20, ClaveI18n: "ct.cobertura.documento_segundo"},
		{Clave: "documento_primero", Orden: 10, ClaveI18n: "ct.cobertura.documento_primero"},
	}
	borrador.Vias[0].Datos = []domain.ElementoPreparacionViaCobertura{
		{Clave: "dato_unico", Orden: 7, ClaveI18n: "ct.cobertura.dato_unico"},
	}
	canon, publicacion := canonConsumoCatalogoPrueba(t, borrador)
	if publicacion.Canon != domain.CanonHuellaCatalogoCoberturaV2() {
		t.Fatal("los metadatos de preparacion deben usar V2")
	}
	var sufijo bytes.Buffer
	_ = binary.Write(&sufijo, binary.BigEndian, uint64(2))
	escribirElementoConsumoPrueba(&sufijo, "documento_primero", 10, "ct.cobertura.documento_primero")
	escribirElementoConsumoPrueba(&sufijo, "documento_segundo", 20, "ct.cobertura.documento_segundo")
	_ = binary.Write(&sufijo, binary.BigEndian, uint64(1))
	escribirElementoConsumoPrueba(&sufijo, "dato_unico", 7, "ct.cobertura.dato_unico")
	if !bytes.HasSuffix(canon, sufijo.Bytes()) {
		t.Fatal("el canon V2 no conserva el orden y los bytes exactos de documentos y datos")
	}
	sinMarca := borrador
	sinMarca.EsEjemplo = false
	canonSinMarca, publicacionSinMarca := canonConsumoCatalogoPrueba(t, sinMarca)
	if bytes.Equal(canon, canonSinMarca) ||
		publicacion.HuellaSHA256 == publicacionSinMarca.HuellaSHA256 {
		t.Fatal("la marca de ejemplo no cambió ambas huellas")
	}

	for _, cambiar := range []struct {
		nombre string
		mutar  func(*domain.BorradorCatalogoViasCobertura)
	}{
		{"documento", func(b *domain.BorradorCatalogoViasCobertura) {
			b.Vias[0].Documentos[0].Clave = "documento_alterado"
		}},
		{"dato", func(b *domain.BorradorCatalogoViasCobertura) {
			b.Vias[0].Datos[0].Orden = 8
		}},
		{"clave_i18n", func(b *domain.BorradorCatalogoViasCobertura) {
			b.Vias[0].Documentos[0].ClaveI18n = "ct.cobertura.documento_alterado"
		}},
	} {
		t.Run(cambiar.nombre, func(t *testing.T) {
			modificado := borradorCatalogoConsumoCanonPrueba()
			modificado.EsEjemplo = true
			modificado.Vias[0].Documentos = append(
				[]domain.ElementoPreparacionViaCobertura(nil),
				borrador.Vias[0].Documentos...,
			)
			modificado.Vias[0].Datos = append(
				[]domain.ElementoPreparacionViaCobertura(nil),
				borrador.Vias[0].Datos...,
			)
			cambiar.mutar(&modificado)
			otroCanon, otraPublicacion := canonConsumoCatalogoPrueba(t, modificado)
			if bytes.Equal(canon, otroCanon) ||
				publicacion.HuellaSHA256 == otraPublicacion.HuellaSHA256 {
				t.Fatal("alterar metadatos no cambió ambas huellas")
			}
		})
	}

	marcaAlterada := publicacion
	marcaAlterada.EsEjemplo = false
	if _, err := NuevaConfirmacionPublicacionCobertura(
		"publicador_catalogo_canon_prueba", marcaAlterada,
		time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC),
	); err == nil {
		t.Fatal("la confirmacion aceptó marca de ejemplo adulterada")
	}

	publicacion.Vias[0].Documentos[0].Clave = "documento_alterado"
	if _, err := NuevaConfirmacionPublicacionCobertura(
		"publicador_catalogo_canon_prueba", publicacion,
		time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC),
	); err == nil {
		t.Fatal("la confirmacion aceptó metadatos adulterados sin nueva huella")
	}
}

func escribirElementoConsumoPrueba(
	destino *bytes.Buffer,
	clave string,
	orden uint16,
	claveI18n string,
) {
	_ = binary.Write(destino, binary.BigEndian, uint32(len(clave)))
	_, _ = destino.WriteString(clave)
	_ = binary.Write(destino, binary.BigEndian, orden)
	_ = binary.Write(destino, binary.BigEndian, uint32(len(claveI18n)))
	_, _ = destino.WriteString(claveI18n)
}
