package ports

import (
	"crypto/sha256"
	"errors"
	"testing"
	"time"
)

func TestPreparacionRevalidacionEsLocalExactaYConCopiasDefensivas(
	t *testing.T,
) {
	solicitud, capacidad := capacidadArtefactoAnalisisPrueba(t)
	artefacto := prepararArtefactoAnalisisPrueba(
		t,
		capacidad,
		solicitud,
	)
	comprobadaEn := artefacto.datos.PreparadoEn
	preparacion, err :=
		NuevaPreparacionRevalidacionEvidenciasFuenteAnalisisO3(
			artefacto.pruebas.evidenciaRC,
			artefacto.pruebas.evidenciaCoste,
			comprobadaEn,
		)
	if err != nil {
		t.Fatal(err)
	}
	materialPrimero, err := preparacion.MaterialRC()
	if err != nil || len(materialPrimero) == 0 {
		t.Fatalf("material RC ausente: %v", err)
	}
	original := materialPrimero[0]
	materialPrimero[0] ^= 0xff
	materialSegundo, err := preparacion.MaterialRC()
	if err != nil || materialSegundo[0] != original {
		t.Fatal("MaterialRC devolvió un alias mutable")
	}
	materialCoste, existe, err := preparacion.MaterialCoste()
	if err != nil || !existe || len(materialCoste) == 0 {
		t.Fatalf("material de coste ausente: %v", err)
	}

	resultado := ResultadoRevalidacionEvidenciasFuenteAnalisisO3{
		FuenteRC: nuevaConfirmacionComprobacionAutoridadPrueba(
			preparacion.fuenteRCEsperada,
			preparacion.materialRC,
			RolFuentePresupuestaria,
			preparacion.comprobadaEn,
		),
		VerificadorRC: nuevaConfirmacionComprobacionAutoridadPrueba(
			preparacion.verificadorEsperado,
			preparacion.materialRC,
			RolVerificadorRespuesta,
			preparacion.comprobadaEn,
		),
		PublicadorRC: nuevaConfirmacionComprobacionAutoridadPrueba(
			preparacion.publicadorEsperado,
			preparacion.materialRC,
			RolPublicadorCatalogo,
			preparacion.comprobadaEn,
		),
		FuenteCoste: nuevaConfirmacionComprobacionAutoridadPrueba(
			preparacion.fuenteCosteEsperada,
			preparacion.materialCoste,
			RolCalculadorCoste,
			preparacion.comprobadaEn,
		),
		VerificadorCoste: nuevaConfirmacionComprobacionAutoridadPrueba(
			preparacion.verificadorCoste,
			preparacion.materialCoste,
			RolVerificadorRespuesta,
			preparacion.comprobadaEn,
		),
	}
	if err := preparacion.ValidarResultado(resultado); err != nil {
		t.Fatalf("resultado exacto rechazado: %v", err)
	}
	otraIdentidad := resultado
	otraIdentidad.FuenteRC.vinculo.Serie++
	if err := preparacion.ValidarResultado(otraIdentidad); !errors.Is(
		err,
		ErrResultadoFuenteAnalisisNoConfiable,
	) {
		t.Fatalf("otra identidad fue aceptada: %v", err)
	}
	otroMaterial := resultado
	otroMaterial.FuenteRC.huellaMaterial[0] ^= 0xff
	if err := preparacion.ValidarResultado(otroMaterial); !errors.Is(
		err,
		ErrResultadoFuenteAnalisisNoConfiable,
	) {
		t.Fatalf("una confirmación de otro material fue aceptada: %v", err)
	}
}

func nuevaConfirmacionComprobacionAutoridadPrueba(
	vinculo VinculoAutoridadFuenteAnalisisO3,
	material []byte,
	rol RolAutoridadFuenteAnalisis,
	comprobadaEn time.Time,
) ConfirmacionComprobacionAutoridadFuenteAnalisis {
	return ConfirmacionComprobacionAutoridadFuenteAnalisis{
		vinculo:        vinculo,
		huellaMaterial: sha256.Sum256(material),
		rol:            rol,
		comprobadaEn:   comprobadaEn,
	}
}
