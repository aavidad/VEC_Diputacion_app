package principal

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// Adaptadores PostgreSQL montados por dietas_rutas.go, dietas_comisiones.go y
// componerBorradoresDietas. Cada fichero usa la cuenta del grupo indicado.
var adaptadoresF4b = map[string][]string{
	"vec_identidad_sesiones_v1_registrador": {
		"internal/vec/adapters/httpseguridad/postgres/registro.go",
	},
	"vec_identidad_sesiones_v1_revalidador": {
		"internal/vec/adapters/httpseguridad/postgres/registro.go",
		"internal/vec/adapters/httpseguridad/postgres/revalidador_autenticacion_actor.go",
	},
	"vec_contexto_actor_v1_runtime": {
		"internal/vec/adapters/contextoactor/postgres/resolutor.go",
	},
	"vec_autorizacion_fuente": {
		"internal/vec/adapters/postgres/autorizacion.go",
	},
	"vec_autorizacion_registro": {
		"internal/vec/adapters/postgres/autorizacion.go",
		"internal/vec/adapters/postgres/autorizacion_solicitud_v3.go",
	},
	"vec_autorizacion_motivos_evaluador": {
		"internal/vec/adapters/postgres/autorizacion_motivos_v2.go",
	},
	"vec_dietas_ejecutor": {
		"internal/modules/dietas/adapters/postgres/acceso_rutas.go",
		"internal/modules/dietas/adapters/postgres/borrador_comision.go",
		"internal/modules/dietas/adapters/postgres/circuito_comision.go",
		"internal/modules/dietas/adapters/postgres/tarifas_provisionales.go",
		"internal/modules/personal/adapters/postgres/relacion_empleado.go",
	},
	"vec_dietas_registrador_frontera": {
		"internal/modules/dietas/adapters/postgres/auditoria_frontera.go",
	},
	"vec_personal_d7_ejecutor": {
		"internal/modules/personal/adapters/postgres/asignacion_dietas.go",
	},
	"vec_personal_registrador_frontera": {
		"internal/modules/personal/adapters/postgres/auditoria_frontera_asignacion_dietas.go",
	},
}

var llamadaF4b = regexp.MustCompile(`\b(vec_[a-z0-9_]+\.[a-z0-9_]+)\s*\(`)
var filaFuncionF4b = regexp.MustCompile(`\('([^']+)','funcion','(vec_[a-z0-9_]+\.[a-z0-9_]+\([^']*\))','EXECUTE'\)`)
var espaciosF4b = regexp.MustCompile(`\s+`)

func funcionesAdaptadoresF4b(t *testing.T, raiz string) map[string]map[string]bool {
	t.Helper()
	resultado := make(map[string]map[string]bool, len(adaptadoresF4b))
	for grupo, archivos := range adaptadoresF4b {
		resultado[grupo] = make(map[string]bool)
		for _, archivo := range archivos {
			contenido, err := os.ReadFile(filepath.Join(raiz, archivo))
			if err != nil {
				t.Fatal(err)
			}
			for _, llamada := range llamadaF4b.FindAllStringSubmatch(string(contenido), -1) {
				resultado[grupo][llamada[1]] = true
			}
		}
	}
	// registro.go y autorizacion.go atienden dos pools nominales. Cada grupo
	// consume solo las funciones de su puerto; el resto no le pertenece.
	delete(resultado["vec_identidad_sesiones_v1_registrador"], "vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1")
	delete(resultado["vec_identidad_sesiones_v1_registrador"], "vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1")
	delete(resultado["vec_identidad_sesiones_v1_revalidador"], "vec_identidad_sesiones_v1.registrar_sesion_v1")
	delete(resultado["vec_identidad_sesiones_v1_revalidador"], "vec_identidad_sesiones_v1.reconciliar_registro_sesion_v1")
	delete(resultado["vec_autorizacion_fuente"], "vec_autorizacion.registrar_decision_si_vigente")
	delete(resultado["vec_autorizacion_registro"], "vec_autorizacion.obtener_instantanea")
	return resultado
}

func compararFuncionesF4b(sql string, llamadas map[string]map[string]bool) error {
	acl := make(map[string]map[string]string)
	for _, fila := range filaFuncionF4b.FindAllStringSubmatch(sql, -1) {
		grupo, firma := fila[1], espaciosF4b.ReplaceAllString(fila[2], "")
		nombre := firma[:strings.IndexByte(firma, '(')]
		if _, ok := llamadas[grupo]; !ok {
			return fmt.Errorf("grupo inesperado: %s", grupo)
		}
		if acl[grupo] == nil {
			acl[grupo] = make(map[string]string)
		}
		if anterior, existe := acl[grupo][nombre]; existe {
			return fmt.Errorf("función duplicada en %s: %s y %s", grupo, anterior, firma)
		}
		acl[grupo][nombre] = firma
	}
	var diferencias []string
	for grupo, funciones := range llamadas {
		for nombre := range funciones {
			if _, ok := acl[grupo][nombre]; !ok {
				diferencias = append(diferencias, "falta "+grupo+" "+nombre)
			}
		}
		for nombre := range acl[grupo] {
			if !funciones[nombre] {
				diferencias = append(diferencias, "sobra "+grupo+" "+nombre)
			}
		}
	}
	slices.Sort(diferencias)
	if len(diferencias) != 0 {
		return fmt.Errorf("ACL F4b y llamadas PostgreSQL difieren: %s", strings.Join(diferencias, "; "))
	}
	return nil
}

func TestACLFuncionesF4bCoincideConAdaptadores(t *testing.T) {
	raiz := filepath.Clean(filepath.Join("..", ".."))
	contenido, err := os.ReadFile(filepath.Join("f4b_acceso_dietas", "transaccion.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sql := string(contenido)
	llamadas := funcionesAdaptadoresF4b(t, raiz)
	if err := compararFuncionesF4b(sql, llamadas); err != nil {
		t.Fatal(err)
	}
	// Las firmas completas de la lista deben figurar en las fuentes SQL
	// canónicas. La normalización solo elimina saltos y espacios de formato.
	var definiciones strings.Builder
	err = filepath.WalkDir(filepath.Join(raiz, "deploy", "postgresql"), func(ruta string, entrada fs.DirEntry, err error) error {
		if err != nil || entrada.IsDir() || !strings.HasSuffix(ruta, ".up.sql") {
			return err
		}
		fuente, err := os.ReadFile(ruta)
		if err == nil {
			definiciones.WriteString(espaciosF4b.ReplaceAllString(string(fuente), ""))
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, fila := range filaFuncionF4b.FindAllStringSubmatch(sql, -1) {
		firma := espaciosF4b.ReplaceAllString(fila[2], "")
		if !strings.Contains(definiciones.String(), firma) {
			t.Errorf("firma ausente en migraciones de origen: %s", firma)
		}
	}

	// Pruebas de sensibilidad sobre la misma comparación que usa el caso real.
	catalogada := "vec_dietas.crear_o_recuperar_comision_catalogada_v2"
	filaCatalogada := filaFuncionF4b.FindString(sql[strings.Index(sql, "('vec_dietas_ejecutor','funcion','"+catalogada):])
	if filaCatalogada == "" {
		t.Fatal("fila catalogada no localizada")
	}
	t.Run("falta catalogada", func(t *testing.T) {
		alterado := strings.Replace(sql, filaCatalogada+",", "", 1)
		if err := compararFuncionesF4b(alterado, llamadas); err == nil || !strings.Contains(err.Error(), "falta vec_dietas_ejecutor "+catalogada) {
			t.Fatalf("no rechazó la retirada de catalogada: %v", err)
		}
	})
	t.Run("sobra calculada antigua", func(t *testing.T) {
		antigua := strings.Replace(filaCatalogada, catalogada, "vec_dietas.crear_o_recuperar_comision_calculada_v1", 1)
		alterado := strings.Replace(sql, filaCatalogada+",", filaCatalogada+",\n "+antigua+",", 1)
		if err := compararFuncionesF4b(alterado, llamadas); err == nil || !strings.Contains(err.Error(), "sobra vec_dietas_ejecutor vec_dietas.crear_o_recuperar_comision_calculada_v1") {
			t.Fatalf("no rechazó la concesión antigua: %v", err)
		}
	})
}
