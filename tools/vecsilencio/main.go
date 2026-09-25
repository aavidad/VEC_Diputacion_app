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
//
// La línea base congela lo heredado como huellas fichero:función:regla con su
// recuento; la guarda falla si una huella aparece o crece y avisa cuando
// decrece. La base solo puede decrecer: -actualizar-base nunca añade ni
// aumenta. Análisis sintáctico (go/ast), sin dependencias ni compilación.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

func main() {
	os.Exit(ejecutar(os.Args[1:], os.Stdout, os.Stderr))
}

func ejecutar(argumentos []string, salida, errores io.Writer) int {
	opciones := flag.NewFlagSet("vecsilencio", flag.ContinueOnError)
	opciones.SetOutput(errores)
	rutaBase := opciones.String("base", "tools/vecsilencio/base.txt", "fichero de línea base")
	generar := opciones.Bool("generar-base", false, "escribe la línea base completa (solo alta inicial)")
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
	for huella, n := range recuentos {
		porRegla[reglaDeHuella(huella)] += n
	}
	reglas := make([]string, 0, len(porRegla))
	for regla := range porRegla {
		reglas = append(reglas, regla)
	}
	sort.Strings(reglas)
	for _, regla := range reglas {
		modo := "bloqueante"
		if !reglaBloqueante(regla) {
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
