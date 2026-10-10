package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	core "vec-diputacion-granada/internal/vec/domain"
)

// funcionNucleoConsumoB2 es la función del núcleo V3 que valida cada decisión
// consumida por las operaciones B2 (campos, tipo, finalidad y módulo).
const funcionNucleoConsumoB2 = "vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna"

// cuerpoNucleoConsumoB2 devuelve el cuerpo de la última migración que define
// la función de consumo: es la que queda instalada.
func cuerpoNucleoConsumoB2(t *testing.T) (string, string) {
	t.Helper()
	dir := filepath.Join("..", "..", "..", "deploy", "postgresql", "autorizacion_atestada_v3", "migraciones")
	ficheros, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil || len(ficheros) == 0 {
		t.Fatalf("sin migraciones del núcleo: %v", err)
	}
	sort.Strings(ficheros)
	cabecera := "CREATE OR REPLACE FUNCTION " + funcionNucleoConsumoB2 + "("
	for i := len(ficheros) - 1; i >= 0; i-- {
		b, err := os.ReadFile(ficheros[i])
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		inicio := strings.Index(s, cabecera)
		if inicio < 0 {
			continue
		}
		cuerpo := s[inicio:]
		apertura := regexp.MustCompile(`AS (\$[a-z_]*\$)`).FindStringSubmatchIndex(cuerpo)
		if apertura == nil {
			t.Fatalf("%s: función sin cuerpo delimitado", ficheros[i])
		}
		marca := cuerpo[apertura[2]:apertura[3]]
		resto := cuerpo[apertura[1]:]
		fin := strings.Index(resto, marca)
		if fin < 0 {
			t.Fatalf("%s: cuerpo sin cierre", ficheros[i])
		}
		return filepath.Base(ficheros[i]), resto[:fin]
	}
	t.Fatal("ninguna migración define la función de consumo")
	return "", ""
}

// contratoNucleoB2 es lo que la SQL exige a la decisión de una operación.
type contratoNucleoB2 struct {
	campos                    []string
	tipo, finalidad, modulo   string
	audiencia                 string
	conTipo, conModulo, conAu bool
}

var (
	reSeparadorBloqueNucleo = regexp.MustCompile(`\n\s+OR \(\n`)
	reCamposLiteral         = regexp.MustCompile(`d->'campos_permitidos'\s*(?:IS NOT DISTINCT FROM|=)\s*'(\[[^']*\])'::jsonb`)
	reCamposCase            = regexp.MustCompile(`d->'campos_permitidos'\s*IS NOT DISTINCT FROM\s*\(?\s*CASE c->>'operacion'((?s:.*?))END`)
	reTipoLiteral           = regexp.MustCompile(`d->>'tipo_recurso'\s*IS NOT DISTINCT FROM\s*'([^']+)'`)
	reTipoCase              = regexp.MustCompile(`d->>'tipo_recurso'\s*IS NOT DISTINCT FROM\s*CASE WHEN c->>'operacion'\s*=\s*'([^']+)'\s*THEN '([^']+)' ELSE '([^']+)' END`)
	reFinalidad             = regexp.MustCompile(`d->>'finalidad'\s*IS NOT DISTINCT FROM\s*'([^']+)'`)
	reModulo                = regexp.MustCompile(`d->>'modulo_id'\s*IS NOT DISTINCT FROM\s*'([^']+)'`)
	reAudiencia             = regexp.MustCompile(`c->>'audiencia_consumo'\s*(?:IS NOT DISTINCT FROM|=)\s*'([^']+)'`)
)

// literalNucleoB2 aplica la regla común: si el bloque fija un único valor,
// ése; si fija varios (una disyunción por operación), el primero tras la
// última mención de la operación.
func literalNucleoB2(re *regexp.Regexp, bloque string, desde int) (string, bool) {
	todos := re.FindAllStringSubmatchIndex(bloque, -1)
	distintos := map[string]bool{}
	for _, m := range todos {
		distintos[bloque[m[2]:m[3]]] = true
	}
	if len(distintos) == 1 {
		return bloque[todos[0][2]:todos[0][3]], true
	}
	for _, m := range todos {
		if m[0] > desde {
			return bloque[m[2]:m[3]], true
		}
	}
	return "", false
}

func contratoNucleoOperacionB2(t *testing.T, cuerpo, accion string) contratoNucleoB2 {
	t.Helper()
	literal := "'" + accion + "'"
	var bloque string
	for _, b := range reSeparadorBloqueNucleo.Split(cuerpo, -1) {
		if strings.Contains(b, literal) {
			if bloque != "" {
				t.Fatalf("%s aparece en más de un bloque del núcleo", accion)
			}
			bloque = b
		}
	}
	if bloque == "" {
		t.Fatalf("%s no aparece en el núcleo", accion)
	}
	ultima := strings.LastIndex(bloque, literal)
	var c contratoNucleoB2
	var lista string
	if m := reCamposCase.FindStringSubmatch(bloque); m != nil {
		ramas := m[1]
		if w := regexp.MustCompile(`WHEN ` + regexp.QuoteMeta(literal) + `\s*THEN\s*'(\[[^']*\])'`).FindStringSubmatch(ramas); w != nil {
			lista = w[1]
		} else if e := regexp.MustCompile(`ELSE\s*'(\[[^']*\])'`).FindStringSubmatch(ramas); e != nil {
			lista = e[1]
		}
	} else if v, ok := literalNucleoB2(reCamposLiteral, bloque, ultima); ok {
		lista = v
	}
	if lista == "" || json.Unmarshal([]byte(lista), &c.campos) != nil {
		t.Fatalf("%s: campos exigidos por el núcleo ilegibles", accion)
	}
	if m := reTipoCase.FindStringSubmatch(bloque); m != nil {
		c.tipo, c.conTipo = m[3], true
		if m[1] == accion {
			c.tipo = m[2]
		}
	} else {
		c.tipo, c.conTipo = literalNucleoB2(reTipoLiteral, bloque, ultima)
	}
	c.finalidad, _ = literalNucleoB2(reFinalidad, bloque, ultima)
	c.modulo, c.conModulo = literalNucleoB2(reModulo, bloque, ultima)
	c.audiencia, c.conAu = literalNucleoB2(reAudiencia, bloque, ultima)
	return c
}

// Cada concesión de los perfiles nominales B2 lleva exactamente los campos,
// tipo, finalidad, módulo y audiencia que exige la SQL del núcleo instalada.
// Los valores se leen de la migración, no se copian a mano, para que un
// cambio en el núcleo rompa aquí y no en PostgreSQL.
func TestIncorporacionB2CamposCoincidenConNucleoSQL(t *testing.T) {
	fichero, cuerpo := cuerpoNucleoConsumoB2(t)
	alta, consultas, _ := escenarioConsultasRRHHDesarrolloPrueba(t)
	soporte := alta.soporte
	vinculo, err := soporte.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	refs := ReferenciasCTIncorporacionDesarrollo{
		PrincipalV3Ref: vinculo.PrincipalID, PerfilV3Ref: vinculo.PerfilActivoRef,
		OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo, UnidadRef: "unidad:desarrollo:rrhh", ActorRef: vinculo.PrincipalID,
	}
	detalle, err := nuevoPerfilNominalIncorporacion(soporte, refs, claveIncorporacionDetalle, nil, soporte.reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	perfiles := &perfilesNominalesIncorporacion{soporte: soporte, consultas: consultas, detalle: detalle}
	config := configuracionB2PuraPrueba().PersonalB2
	if err := extenderPerfilesNominalesB2(perfiles, refs, config, soporte.reloj.Ahora()); err != nil {
		t.Fatal(err)
	}
	revisadas := 0
	for _, d := range operacionesIncorporacionB2() {
		if d.accion == ct.AccionConsultarDetalleRRHH {
			continue // perfil propio del detalle, fuera de este switch
		}
		perfil := perfiles.b2[d.accion]
		if perfil == nil {
			t.Fatalf("%s sin perfil nominal", d.clave)
		}
		var concesiones int
		for _, c := range perfil.plantilla.VersionRol.Concesiones {
			if c.Accion != d.accion {
				continue
			}
			concesiones++
			n := contratoNucleoOperacionB2(t, cuerpo, d.accion)
			if !slices.Equal(c.CamposPermitidos, n.campos) {
				t.Errorf("%s (%s): campos %v; el núcleo %s exige %v", d.clave, d.accion, c.CamposPermitidos, fichero, n.campos)
			}
			if !slices.Equal(c.Finalidades, []string{n.finalidad}) {
				t.Errorf("%s: finalidad %v; el núcleo exige %q", d.clave, c.Finalidades, n.finalidad)
			}
			if n.conTipo && c.TipoRecurso != n.tipo {
				t.Errorf("%s: tipo %q; el núcleo exige %q", d.clave, c.TipoRecurso, n.tipo)
			}
			if n.conModulo && c.ModuloID != n.modulo {
				t.Errorf("%s: módulo %q; el núcleo exige %q", d.clave, c.ModuloID, n.modulo)
			}
			if n.conAu && d.audiencia != n.audiencia {
				t.Errorf("%s: audiencia %q; el núcleo exige %q", d.clave, d.audiencia, n.audiencia)
			}
		}
		if concesiones != 1 {
			t.Fatalf("%s: %d concesiones en su perfil", d.clave, concesiones)
		}
		revisadas++
	}
	if revisadas != len(operacionesIncorporacionB2())-1 {
		t.Fatalf("revisadas %d operaciones", revisadas)
	}

	// Las tres lecturas de Personal con que el plan B2 arma sus opciones
	// (vacantes, régimen y modalidad, clases de ocupación) quedan cubiertas
	// por los ámbitos de su perfil: el PDP compara dimensiones exactas.
	actor, err := soporte.contexto.Resultado.Contexto.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	org := config.OrganismoRef
	cubre := func(accion string, r core.RecursoAutorizable) bool {
		p := perfiles.b2[accion]
		return p != nil && p.plantilla.AsignacionPerfil.Cubre(r)
	}
	vacantes, err := personal.NuevoMaterialVacantesB2(personal.SolicitudVacantesB2{OrganismoRef: org, Corte: personal.CorteEmpleadoB2{VigenteEn: "2026-10-01", ConocidoEn: soporte.reloj.Ahora()}, Limite: 100, Actor: actor})
	if err != nil || !cubre(personal.AccionVacantesB2, vacantes.Recurso()) {
		t.Fatalf("vacantes fuera de su perfil: %v", err)
	}
	for _, tipo := range []string{"regimen", "modalidad"} {
		m, err := personal.NuevoMaterialConsultaCatalogoEmpleadoB2(personal.SolicitudConsultaCatalogoEmpleadoB2{OrganismoRef: org, Tipo: tipo, Estado: "publicada", Limite: 100, Actor: actor})
		if err != nil || !cubre(personal.AccionConsultarCatalogoEmpleadoB2, m.Recurso()) {
			t.Fatalf("catálogo %s fuera de su perfil: %v", tipo, err)
		}
	}
	situacion, err := personal.NuevoMaterialConsultaCatalogoEmpleadoB2(personal.SolicitudConsultaCatalogoEmpleadoB2{OrganismoRef: org, Tipo: "situacion", Estado: "publicada", Limite: 100, Actor: actor})
	if err != nil || cubre(personal.AccionConsultarCatalogoEmpleadoB2, situacion.Recurso()) {
		t.Fatal("el perfil del plan no debe leer otros catálogos de Personal")
	}
	clases, err := personal.NuevoMaterialClasesOcupacionCT(personal.ConsultaClasesOcupacionCT{OrganismoRef: org, Actor: actor})
	if err != nil || !cubre("personal.plan_incorporacion_ct.clases_ocupacion", clases.Recurso()) {
		t.Fatalf("clases de ocupación fuera de su perfil: %v", err)
	}
}
