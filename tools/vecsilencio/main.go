// Command vecsilencio es la guarda estática de fallos silenciosos de VEC
// (M2a del consenso de supervisión del 25/09/2026: ningún fallo sin
// registro).
//
// Reglas:
//
//	VS000  directiva //vec:silencio-justificado sin motivo de la lista cerrada
//	       (nunca heredable).
//	VS001  `if err != nil` cuyo bloque no usa el error, no registra, no
//	       responde al cliente, no relanza y termina en return nil/false/
//	       vacío o en continue/break.
//	VS002  recover() en una función que no registra ni relanza.
//	VS003  estado HTTP >= 500 escrito en una función que no declara la
//	       incidencia. Solo informe: en vec-server lo cubre el middleware
//	       común y el recuento sirve para seguir otras superficies.
//	VSJ    recuento de directivas //vec:silencio-justificado válidas por
//	       motivo, huella (justificadas):MOTIVO:VSJ. No es una infracción,
//	       pero como la base solo puede decrecer, una justificación nueva
//	       falla igual que un fallo silencioso nuevo.
//
// La línea base congela lo heredado como huellas fichero:función:regla con su
// recuento; la guarda falla si una huella aparece o crece y avisa cuando
// decrece. La base solo puede decrecer, y eso se comprueba en dos niveles:
//
//   - -actualizar-base nunca añade ni aumenta, y -generar-base se niega si la
//     base ya existe (solo sirve para el alta inicial);
//   - -base-anterior RUTA contrasta la base del árbol con la de la rama base
//     (scripts/verificar_calidad.sh la extrae de `git merge-base`): falla si
//     la base del árbol añade una huella o sube un recuento, de modo que
//     editar base.txt a mano tampoco sirve para admitir un fallo nuevo.
//
// Procedimiento:
//
//  1. Un hallazgo nuevo se corrige registrando (log/slog, emisor de
//     incidencias), propagando (return err, motivo cerrado Motivo*/Err*) o
//     respondiendo al cliente con un estado de fallo. Nunca se añade a
//     base.txt.
//  2. Solo si el descarte es inocuo por construcción se justifica con
//     `//vec:silencio-justificado MOTIVO texto` en la línea o la anterior,
//     con MOTIVO de la lista cerrada. La justificación suma VSJ: también
//     falla hasta que Dirección la admite (paso 4).
//  3. Tras corregir, `go run ./tools/vecsilencio -actualizar-base` reduce la
//     base; el commit incluye la base reducida.
//  4. Subir la base (huella nueva, recuento mayor o justificación nueva) es
//     una decisión de Dirección: commit propio que solo cambia base.txt, con
//     motivo en el mensaje, revisado e integrado ejecutando la puerta con
//     VECSILENCIO_RAMA_BASE apuntando a ese commit. Ningún otro camino lo
//     admite.
//  5. El alta inicial (-generar-base) solo funciona sin base.txt.
//
// Análisis sintáctico (go/ast), sin dependencias ni compilación. Límites
// conocidos: sin go/types, una variable de error solo se reconoce por su
// nombre (err, errX, xErr, xError) o por estar declarada con tipo error en
// parámetros, resultados o `var`; el estado HTTP solo se determina cuando es
// una constante de net/http o un literal. Los paquetes de apoyo a pruebas
// (pruebas, *prueba, *pruebas) quedan fuera.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	os.Exit(ejecutar(os.Args[1:], os.Stdout, os.Stderr))
}

func ejecutar(argumentos []string, salida, errores io.Writer) int {
	opciones := flag.NewFlagSet("vecsilencio", flag.ContinueOnError)
	opciones.SetOutput(errores)
	rutaBase := opciones.String("base", "tools/vecsilencio/base.txt", "fichero de línea base")
	generar := opciones.Bool("generar-base", false, "escribe la línea base completa (solo alta inicial; se niega si existe)")
	baseAnterior := opciones.String("base-anterior", "", "línea base de la rama base: la del árbol no puede añadir huellas ni subir recuentos")
	actualizar := opciones.Bool("actualizar-base", false, "reduce la línea base a lo vigente; nunca añade")
	raiz := opciones.String("raiz", ".", "raíz del módulo")
	if err := opciones.Parse(argumentos); err != nil {
		return 2
	}
	directorios := opciones.Args()
	if len(directorios) == 0 {
		// Directorios de producción presentes; uno explícito ausente falla.
		for _, dir := range []string{"cmd", "config", "internal"} {
			if info, err := os.Stat(filepath.Join(*raiz, dir)); err == nil && info.IsDir() {
				directorios = append(directorios, dir)
			}
		}
	}
	hallazgos, err := AnalizarArbol(*raiz, directorios)
	if err != nil {
		fmt.Fprintf(errores, "vecsilencio: %v\n", err)
		return 2
	}
	recuentos := Recontar(hallazgos)
	imprimirResumen(salida, recuentos)
	if *generar {
		if _, err := os.Stat(*rutaBase); !errors.Is(err, fs.ErrNotExist) {
			fmt.Fprintf(errores, "vecsilencio: -generar-base solo sirve para el alta inicial y %s ya existe; la base solo puede decrecer (-actualizar-base)\n", *rutaBase)
			return 2
		}
		if err := EscribirBase(*rutaBase, recuentos); err != nil {
			fmt.Fprintf(errores, "vecsilencio: %v\n", err)
			return 2
		}
		fmt.Fprintf(salida, "vecsilencio: línea base generada en %s\n", *rutaBase)
		return 0
	}
	base, err := LeerBase(*rutaBase)
	if err != nil {
		fmt.Fprintf(errores, "vecsilencio: %v\n", err)
		return 2
	}
	fallo := false
	if *baseAnterior != "" {
		anterior, err := LeerBase(*baseAnterior)
		if err != nil {
			fmt.Fprintf(errores, "vecsilencio: %v\n", err)
			return 2
		}
		if crecidas := Comparar(anterior, base).Nuevas; len(crecidas) > 0 {
			for _, huella := range crecidas {
				fmt.Fprintf(errores, "vecsilencio: la línea base crece respecto a la rama base en %s (%d > %d)\n", huella, base[huella], anterior[huella])
			}
			fmt.Fprintln(errores, "vecsilencio: la línea base solo puede decrecer; corregir el fallo en vez de admitirlo en base.txt")
			fallo = true
		}
	}
	comparacion := Comparar(base, recuentos)
	for _, aviso := range comparacion.Reducibles {
		fmt.Fprintf(salida, "aviso: la base admite más de lo vigente en %s (%d > %d); reducible con -actualizar-base\n", aviso.Huella, aviso.Base, aviso.Vigente)
	}
	if *actualizar {
		if err := EscribirBase(*rutaBase, Reducir(base, recuentos)); err != nil {
			fmt.Fprintf(errores, "vecsilencio: %v\n", err)
			return 2
		}
		fmt.Fprintf(salida, "vecsilencio: línea base reducida en %s\n", *rutaBase)
	}
	if len(comparacion.Nuevas) == 0 {
		if fallo {
			return 1
		}
		fmt.Fprintln(salida, "vecsilencio: sin fallos silenciosos nuevos")
		return 0
	}
	for _, huella := range comparacion.Nuevas {
		for _, h := range hallazgosDe(hallazgos, huella) {
			fmt.Fprintf(errores, "%s:%d: %s en %s (nuevo respecto a la línea base)\n", h.Fichero, h.Linea, h.Regla, h.Funcion)
		}
	}
	fmt.Fprintf(errores, "vecsilencio: %d huellas nuevas o crecientes; registrar o propagar el fallo, o justificar con %s MOTIVO texto\n", len(comparacion.Nuevas), directivaJustificacion)
	return 1
}

func imprimirResumen(salida io.Writer, recuentos map[string]int) {
	porRegla := map[string]int{}
	var justificaciones []string
	for huella, n := range recuentos {
		regla := reglaDeHuella(huella)
		porRegla[regla] += n
		if regla == ReglaJustificacion {
			justificaciones = append(justificaciones, huella)
		}
	}
	sort.Strings(justificaciones)
	for _, huella := range justificaciones {
		fmt.Fprintf(salida, "vecsilencio: justificadas %s %d\n", motivoDeHuella(huella), recuentos[huella])
	}
	reglas := make([]string, 0, len(porRegla))
	for regla := range porRegla {
		reglas = append(reglas, regla)
	}
	sort.Strings(reglas)
	for _, regla := range reglas {
		modo := "bloqueante"
		switch {
		case regla == ReglaJustificacion:
			modo = "justificaciones; solo decrecen"
		case !reglaBloqueante(regla):
			modo = "informe"
		}
		fmt.Fprintf(salida, "vecsilencio: %s %d (%s)\n", regla, porRegla[regla], modo)
	}
}

func hallazgosDe(hallazgos []Hallazgo, huella string) []Hallazgo {
	var r []Hallazgo
	for _, h := range hallazgos {
		if h.Huella() == huella {
			r = append(r, h)
		}
	}
	return r
}

// motivoDeHuella extrae MOTIVO de (justificadas):MOTIVO:VSJ.
func motivoDeHuella(huella string) string {
	partes := strings.Split(huella, ":")
	if len(partes) < 3 {
		return huella
	}
	return partes[len(partes)-2]
}
