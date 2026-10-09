package bootstrap

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
)

type consultadorACLMiBolsaInscripcionPrueba struct {
	funciones    []string
	permitido    bool
	funcionesB96 int
	b96Exacta    bool
	consulta     string
	firma        any
}

func (q *consultadorACLMiBolsaInscripcionPrueba) QueryRow(_ context.Context, consulta string, args ...any) pgx.Row {
	q.consulta = consulta
	if len(args) == 1 {
		q.firma = args[0]
	}
	return filaACLMiBolsaInscripcionPrueba{q}
}

type filaACLMiBolsaInscripcionPrueba struct {
	q *consultadorACLMiBolsaInscripcionPrueba
}

func (f filaACLMiBolsaInscripcionPrueba) Scan(dest ...any) error {
	if len(dest) != 4 {
		return ErrMaterialV3PortalExternoInvalido
	}
	*dest[0].(*[]string) = slices.Clone(f.q.funciones)
	*dest[1].(*bool) = f.q.permitido
	*dest[2].(*int) = f.q.funcionesB96
	*dest[3].(*bool) = f.q.b96Exacta
	return nil
}

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

func TestACLPortalExternoDualExactaAntesYDespuesB96(t *testing.T) {
	if strings.Count(firmaSolicitarInscripcionPortalExterno, ",") != 12 ||
		!strings.HasPrefix(firmaSolicitarInscripcionPortalExterno,
			"vec_bolsa_llamamientos.solicitar_inscripcion_v1(text,jsonb,bytea,") {
		t.Fatal("la firma nominal B96 no tiene trece argumentos")
	}
	antes := funcionesMiBolsaPortalExternoSinInscripcion()
	despues := funcionesMiBolsaPortalExternoConInscripcion()
	if len(antes) != 11 || len(despues) != 12 ||
		!funcionesMiBolsaPortalExternoExactas(antes, 0, false) ||
		!funcionesMiBolsaPortalExternoExactas(despues, 1, true) {
		t.Fatal("se rompió el arranque B59 anterior o B96 final")
	}
	if funcionesMiBolsaPortalExternoExactas(antes, 1, false) ||
		funcionesMiBolsaPortalExternoExactas(despues, 1, false) ||
		funcionesMiBolsaPortalExternoExactas(despues, 2, true) ||
		funcionesMiBolsaPortalExternoExactas(despues, 0, false) {
		t.Fatal("firma B96 vieja, GRANT ausente o sobrecarga aceptados")
	}
	conExtraAntes := append(slices.Clone(antes), "funcion_ajena_v1")
	conExtraDespues := append(slices.Clone(despues), "funcion_ajena_v1")
	if funcionesMiBolsaPortalExternoExactas(conExtraAntes, 0, false) ||
		funcionesMiBolsaPortalExternoExactas(conExtraDespues, 1, true) {
		t.Fatal("GRANT lateral antes o después de B96 aceptado")
	}
	conDuplicado := slices.Clone(despues)
	conDuplicado[0] = conDuplicado[1]
	if funcionesMiBolsaPortalExternoExactas(conDuplicado, 1, true) {
		t.Fatal("sobrecarga repetida aceptada")
	}
	for _, caso := range []struct {
		nombre            string
		funciones         []string
		n, exacta         int
		permitido, espera bool
	}{
		{"codigo antes de SQL", antes, 0, 0, true, true},
		{"SQL antes de codigo", despues, 1, 1, true, true},
		{"firma vieja", despues, 1, 0, true, false},
		{"grant lateral", conExtraDespues, 1, 1, true, false},
		{"tabla concedida", despues, 1, 1, false, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			q := &consultadorACLMiBolsaInscripcionPrueba{funciones: caso.funciones,
				permitido: caso.permitido, funcionesB96: caso.n, b96Exacta: caso.exacta == 1}
			if got := comprobarACLMiBolsaPortalExterno(context.Background(), q); got != caso.espera {
				t.Fatalf("preflight = %v", got)
			}
			if q.firma != firmaSolicitarInscripcionPortalExterno ||
				!strings.Contains(q.consulta, "array_agg(proname::text") ||
				!strings.Contains(q.consulta, "aclexplode(p.proacl)") ||
				!strings.Contains(q.consulta, "has_table_privilege") {
				t.Fatal("faltó firma exacta o guarda ACL en la consulta")
			}
		})
	}
}
