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
	funciones            []string
	permitido            bool
	funcionesB96         int
	b96Exacta            bool
	funcionesB104        int
	b104Exacta           bool
	funcionesLecturaB104 int
	lecturaB104Exacta    bool
	funcionesIntentoB104 int
	intentoB104Exacta    bool
	funcionesB138        int
	b138Exacta           bool
	funcionesB146        int
	b146Exacta           bool
	consulta             string
	firma                any
	firmaB104            any
	firmaLecturaB104     any
	firmaIntentoB104     any
	firmaB114            any
	firmaB138            any
	firmaB146            any
}

func (q *consultadorACLMiBolsaInscripcionPrueba) QueryRow(_ context.Context, consulta string, args ...any) pgx.Row {
	q.consulta = consulta
	if len(args) == 8 {
		q.firma = args[0]
		q.firmaB104 = args[2]
		q.firmaLecturaB104 = args[3]
		q.firmaIntentoB104 = args[4]
		q.firmaB114 = args[5]
		q.firmaB138 = args[6]
		q.firmaB146 = args[7]
	}
	return filaACLMiBolsaInscripcionPrueba{q}
}

type filaACLMiBolsaInscripcionPrueba struct {
	q *consultadorACLMiBolsaInscripcionPrueba
}

func (f filaACLMiBolsaInscripcionPrueba) Scan(dest ...any) error {
	if len(dest) != 16 {
		return ErrMaterialV3PortalExternoInvalido
	}
	*dest[0].(*[]string) = slices.Clone(f.q.funciones)
	*dest[1].(*bool) = f.q.permitido
	*dest[2].(*int) = f.q.funcionesB96
	*dest[3].(*bool) = f.q.b96Exacta
	*dest[4].(*int) = f.q.funcionesB104
	*dest[5].(*bool) = f.q.b104Exacta
	*dest[6].(*int) = f.q.funcionesLecturaB104
	*dest[7].(*bool) = f.q.lecturaB104Exacta
	*dest[8].(*int) = f.q.funcionesIntentoB104
	*dest[9].(*bool) = f.q.intentoB104Exacta
	*dest[10].(*int) = 0
	*dest[11].(*bool) = false
	*dest[12].(*int) = f.q.funcionesB138
	*dest[13].(*bool) = f.q.b138Exacta
	*dest[14].(*int) = f.q.funcionesB146
	*dest[15].(*bool) = f.q.b146Exacta
	return nil
}

func TestMaterialInscripcionPortalExternoAudienciaSeparada(t *testing.T) {
	if !consumidorPortalExternoValido(consumidorInscripcionPortalExternoV3) ||
		!slices.Equal(audienciasConsumidorPortalExternoV3(consumidorInscripcionPortalExternoV3), []string{audienciaPresentarInscripcionExternaV3, audienciaPresentarAlegacionExternaV3}) ||
		accionPresentarInscripcionExternaV3 != inscripcion.AccionPresentar ||
		tipoRecursoPresentarInscripcionExternaV3 != "inscripcion_convocatoria" {
		t.Fatal("consumidor de inscripción sin acción, recurso o audiencia exactos")
	}
	catalogo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(descriptoresMaterialPortalExternoV3())
	d, presente := catalogo.descriptorPara(audienciaPresentarInscripcionExternaV3)
	if err != nil || !presente || d != descriptorMaterialPresentarInscripcionExternaV3() {
		t.Fatalf("descriptor de inscripción ausente o compartido: %v", err)
	}
	if d, presente = catalogo.descriptorPara(audienciaPresentarAlegacionExternaV3); !presente || d != descriptorMaterialPresentarAlegacionExternaV3() {
		t.Fatal("descriptor nominal de alegación ausente")
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
	if err != nil || len(inscripcionMaterial) != 2 || inscripcionMaterial[0].audienciaConsumo != audienciaPresentarInscripcionExternaV3 ||
		inscripcionMaterial[1].audienciaConsumo != audienciaPresentarAlegacionExternaV3 ||
		inscripcionMaterial[0].claveHMACID == inscripcionMaterial[1].claveHMACID {
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
	if len(antes) != 12 || len(despues) != 13 ||
		!funcionesMiBolsaPortalExternoExactas(antes, 0, false) ||
		!funcionesMiBolsaPortalExternoExactas(despues, 1, true) {
		t.Fatal("se rompió el arranque B59+B77 anterior o B96 final")
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
	conPreferencias := funcionesMiBolsaPortalExternoConPreferencias(true)
	if len(conPreferencias) != 16 ||
		!funcionesMiBolsaPortalExternoExactasConPreferencias(conPreferencias, 1, true, 1, true, 1, true, 1, true) ||
		funcionesMiBolsaPortalExternoExactasConPreferencias(conPreferencias, 1, true, 1, false, 1, true, 1, true) ||
		funcionesMiBolsaPortalExternoExactasConPreferencias(conPreferencias, 1, true, 2, true, 1, true, 1, true) ||
		funcionesMiBolsaPortalExternoExactasConPreferencias(conPreferencias, 1, true, 1, true, 0, false, 1, true) ||
		funcionesMiBolsaPortalExternoExactasConPreferencias(conPreferencias, 1, true, 1, true, 1, true, 1, false) ||
		funcionesMiBolsaPortalExternoExactasConPreferencias(conPreferencias, 1, true, 0, false, 1, true, 1, true) {
		t.Fatal("B104 sin firma exacta, sobrecarga o GRANT ajeno admitido")
	}
	for _, caso := range []struct {
		nombre                                                                              string
		funciones                                                                           []string
		n, exacta                                                                           int
		nB104, exactaB104, nLecturaB104, exactaLecturaB104, nIntentoB104, exactaIntentoB104 int
		permitido, espera                                                                   bool
	}{
		{"codigo antes de SQL", antes, 0, 0, 0, 0, 0, 0, 0, 0, true, true},
		{"SQL antes de codigo", despues, 1, 1, 0, 0, 0, 0, 0, 0, true, true},
		{"B104 exacta", conPreferencias, 1, 1, 1, 1, 1, 1, 1, 1, true, true},
		{"B104 sin grant", conPreferencias, 1, 1, 1, 0, 1, 1, 1, 1, true, false},
		{"B104 lectura sin grant", conPreferencias, 1, 1, 1, 1, 1, 0, 1, 1, true, false},
		{"B104 intento sin grant", conPreferencias, 1, 1, 1, 1, 1, 1, 1, 0, true, false},
		{"firma vieja", despues, 1, 0, 0, 0, 0, 0, 0, 0, true, false},
		{"grant lateral", conExtraDespues, 1, 1, 0, 0, 0, 0, 0, 0, true, false},
		{"tabla concedida", despues, 1, 1, 0, 0, 0, 0, 0, 0, false, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			q := &consultadorACLMiBolsaInscripcionPrueba{funciones: caso.funciones,
				permitido: caso.permitido, funcionesB96: caso.n, b96Exacta: caso.exacta == 1,
				funcionesB104: caso.nB104, b104Exacta: caso.exactaB104 == 1,
				funcionesLecturaB104: caso.nLecturaB104, lecturaB104Exacta: caso.exactaLecturaB104 == 1,
				funcionesIntentoB104: caso.nIntentoB104, intentoB104Exacta: caso.exactaIntentoB104 == 1}
			if got := comprobarACLMiBolsaPortalExterno(context.Background(), q, false); got != caso.espera {
				t.Fatalf("preflight = %v", got)
			}
			if q.firma != firmaSolicitarInscripcionPortalExterno ||
				q.firmaB104 != firmaOrdenarPreferenciasBolsasPortalExterno ||
				q.firmaLecturaB104 != firmaConsultarPreferenciasBolsasPortalExterno ||
				q.firmaIntentoB104 != firmaIntentoPreferenciasBolsasPortalExterno ||
				q.firmaB114 != firmaPresentarAlegacionPortalExterno ||
				q.firmaB138 != firmaGuardarModalidadesPortalExterno ||
				q.firmaB146 != firmaSolicitarInscripcionConjuntaPortalExterno ||
				!strings.Contains(q.consulta, "array_agg(proname::text") ||
				!strings.Contains(q.consulta, "aclexplode(p.proacl)") ||
				!strings.Contains(q.consulta, "has_table_privilege") {
				t.Fatal("faltó firma exacta o guarda ACL en la consulta")
			}
		})
	}
	// B138 (F1-13) concede la escritura de modalidades al mismo grupo; el
	// arranque la admite solo exacta y junto a las preferencias B104.
	conModalidades := append(slices.Clone(conPreferencias), "guardar_preferencias_modalidades_v1")
	modalidadesSinPreferencias := append(slices.Clone(despues), "guardar_preferencias_modalidades_v1")
	for _, caso := range []struct {
		nombre             string
		funciones          []string
		preferencias       bool
		nB138              int
		exactaB138, espera bool
	}{
		{"B138 exacta con preferencias", conModalidades, true, 1, true, true},
		{"B138 sin firma exacta", conModalidades, true, 1, false, false},
		{"B138 sobrecargada", conModalidades, true, 2, true, false},
		{"B138 sin preferencias B104", modalidadesSinPreferencias, false, 1, true, false},
		{"B138 ausente en la lista", conPreferencias, true, 1, true, false},
		{"lista con B138 sin cotejo", conModalidades, true, 0, false, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			n := 0
			if caso.preferencias {
				n = 1
			}
			q := &consultadorACLMiBolsaInscripcionPrueba{funciones: caso.funciones, permitido: true,
				funcionesB96: 1, b96Exacta: true, funcionesB104: n, b104Exacta: caso.preferencias,
				funcionesLecturaB104: n, lecturaB104Exacta: caso.preferencias,
				funcionesIntentoB104: n, intentoB104Exacta: caso.preferencias,
				funcionesB138: caso.nB138, b138Exacta: caso.exactaB138}
			if got := comprobarACLMiBolsaPortalExterno(context.Background(), q, false); got != caso.espera {
				t.Fatalf("preflight B138 = %v, esperado %v", got, caso.espera)
			}
		})
	}
	for _, caso := range []struct {
		nombre            string
		funciones         []string
		funcionesB96      int
		b96Exacta, espera bool
	}{
		{"sin B96", antes, 0, false, false},
		{"B96 exacta", despues, 1, true, true},
		{"B96 sobrecargada", despues, 2, true, false},
	} {
		t.Run("inscripcion/"+caso.nombre, func(t *testing.T) {
			q := &consultadorACLMiBolsaInscripcionPrueba{funciones: caso.funciones,
				permitido: true, funcionesB96: caso.funcionesB96, b96Exacta: caso.b96Exacta}
			if got := comprobarACLMiBolsaPortalExterno(context.Background(), q, true); got != caso.espera {
				t.Fatalf("inscripcion requiere B96: obtenido=%v esperado=%v", got, caso.espera)
			}
		})
	}

}

// B146: la solicitud conjunta solo se admite exacta y junto a B96.
func TestSolicitudConjuntaPortalExternoSoloJuntoAB96(t *testing.T) {
	con := append(funcionesMiBolsaPortalExternoConInscripcion(), "solicitar_inscripcion_conjunta_v1")
	if quedan, ok := sinSolicitudConjuntaPortalExterno(con, 1, true, 1, true); !ok ||
		!funcionesMiBolsaPortalExternoExactas(quedan, 1, true) {
		t.Fatal("B96 con B146 exacta rechazada")
	}
	if quedan, ok := sinSolicitudConjuntaPortalExterno(funcionesMiBolsaPortalExternoConInscripcion(), 1, true, 0, false); !ok ||
		!funcionesMiBolsaPortalExternoExactas(quedan, 1, true) {
		t.Fatal("B96 sin B146 rechazada")
	}
	for nombre, caso := range map[string]struct {
		funciones []string
		b96, b146 int
		e96, e146 bool
	}{
		"B146 sin B96":       {append(funcionesMiBolsaPortalExternoSinInscripcion(), "solicitar_inscripcion_conjunta_v1"), 0, 1, false, true},
		"B146 inexacta":      {con, 1, 1, true, false},
		"concedida sin cote": {con, 1, 0, true, false},
		"duplicada":          {con, 1, 2, true, true},
	} {
		if _, ok := sinSolicitudConjuntaPortalExterno(caso.funciones, caso.b96, caso.e96, caso.b146, caso.e146); ok {
			t.Fatalf("%s aceptada", nombre)
		}
	}
}

// El preflight completo admite B146 exacta con B96 y rechaza la lista que la
// incluye sin su cotejo.
func TestACLPortalExternoAdmiteB146Exacta(t *testing.T) {
	con := append(funcionesMiBolsaPortalExternoConInscripcion(), "solicitar_inscripcion_conjunta_v1")
	for _, caso := range []struct {
		nombre string
		n      int
		exacta bool
		espera bool
	}{{"exacta", 1, true, true}, {"sin cotejo", 0, false, false}, {"inexacta", 1, false, false}} {
		q := &consultadorACLMiBolsaInscripcionPrueba{funciones: con, permitido: true,
			funcionesB96: 1, b96Exacta: true, funcionesB146: caso.n, b146Exacta: caso.exacta}
		if got := comprobarACLMiBolsaPortalExterno(context.Background(), q, true); got != caso.espera {
			t.Fatalf("%s: preflight %v", caso.nombre, got)
		}
	}
}
