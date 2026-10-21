package bootstrap

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
)

// F1-150: la composición del alta registra el perfil fijo de comunicaciones
// solo cuando Bolsa de llamamientos está configurada, antes de abrir
// PostgreSQL (donde el arranque provee todos los perfiles fijos registrados).
//
// El alta completa no se puede montar en un clon desechable sin el material
// de atestación propio de una copia: el gobierno sintético del clon rechaza
// otro material. Por eso esta prueba lee la composición real y comprueba,
// con el soporte que construye, el efecto en las fronteras comunes.
func TestAltaRegistraPerfilFijoComunicacionesConLlamamientos(t *testing.T) {
	archivo, err := parser.ParseFile(token.NewFileSet(), "contratacion_temporal_alta_desarrollo.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var alta *ast.FuncDecl
	for _, d := range archivo.Decls {
		if f, ok := d.(*ast.FuncDecl); ok && f.Recv == nil && f.Name.Name == "nuevasDependenciasAltaContratacionTemporalDesarrollo" {
			alta = f
		}
	}
	if alta == nil || alta.Body == nil {
		t.Fatal("composición del alta no encontrada")
	}
	registros, abrePostgreSQL, posRegistro := 0, token.NoPos, token.NoPos
	ast.Inspect(alta.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			if nombreLlamadaPrueba(x.Fun) == "nuevasDependenciasPostgreSQLContratacionTemporalDesarrollo" && abrePostgreSQL == token.NoPos {
				abrePostgreSQL = x.Pos()
			}
		case *ast.IfStmt:
			if !condicionLlamamientosPrueba(x.Cond) {
				return true
			}
			if fijo := registroComunicacionesPrueba(x.Body); fijo {
				registros++
				posRegistro = x.Pos()
			}
		}
		return true
	})
	if registros != 1 {
		t.Fatalf("el alta registra %d veces el perfil fijo de comunicaciones bajo BolsaLlamamientosConfigurada()", registros)
	}
	if abrePostgreSQL == token.NoPos || posRegistro > abrePostgreSQL {
		t.Fatal("el perfil fijo se registra después de abrir PostgreSQL y no se proveería al arrancar")
	}

	// Lo que esa rama deja en el soporte: la ruta pasa al perfil fijo y la
	// frontera común de la consulta admite solo ese perfil.
	s, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ruta := httpinterno.RutaConsultaComunicacionesExpediente
	dinamico := s.contexto.Resultado.Contexto.PerfilActivoRef
	declaraciones := descriptoresFronterasContratacionTemporalDesarrollo(dinamico, []string{dinamico})
	if antes, err := asignarPerfilesFijosEnFronterasCTDesarrollo(s, dinamico, declaraciones); err != nil ||
		!reflect.DeepEqual(perfilesFronteraPrueba(t, antes, "ct-llamamiento-comunicaciones-consultar"), []string{dinamico}) {
		t.Fatal("sin el perfil fijo la frontera debía seguir en el perfil CT", err)
	}
	fijo, err := nuevoPerfilFijoConsultaComunicacionesExpedienteDesarrollo(principal, s.contexto, s.reloj.Ahora())
	if err != nil || s.registrarPerfilFijoCTDesarrollo(fijo) != nil || s.perfilFijoParaRuta(ruta) != fijo {
		t.Fatal("perfil fijo de comunicaciones no registrado", err)
	}
	despues, err := asignarPerfilesFijosEnFronterasCTDesarrollo(s, dinamico, declaraciones)
	if err != nil || !reflect.DeepEqual(perfilesFronteraPrueba(t, despues, "ct-llamamiento-comunicaciones-consultar"), []string{fijo.perfilRef()}) {
		t.Fatal("la frontera de comunicaciones no quedó solo con el perfil fijo", err)
	}
}

func perfilesFronteraPrueba(t *testing.T, fronteras []descriptorFronteraComunDesarrollo, clave string) []string {
	t.Helper()
	var perfiles []string
	vistas := 0
	for _, d := range fronteras {
		if d.Clave == clave {
			vistas++
			perfiles = d.PerfilesActivosRef
		}
	}
	if vistas != 1 {
		t.Fatalf("frontera %s ausente o repetida", clave)
	}
	return perfiles
}

func nombreLlamadaPrueba(e ast.Expr) string {
	switch f := e.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	}
	return ""
}

// condicionLlamamientosPrueba reconoce exactamente
// cfg.ContratacionTemporalPostgreSQL.BolsaLlamamientosConfigurada().
func condicionLlamamientosPrueba(e ast.Expr) bool {
	llamada, ok := e.(*ast.CallExpr)
	if !ok || len(llamada.Args) != 0 {
		return false
	}
	metodo, ok := llamada.Fun.(*ast.SelectorExpr)
	if !ok || metodo.Sel.Name != "BolsaLlamamientosConfigurada" {
		return false
	}
	campo, ok := metodo.X.(*ast.SelectorExpr)
	if !ok || campo.Sel.Name != "ContratacionTemporalPostgreSQL" {
		return false
	}
	raiz, ok := campo.X.(*ast.Ident)
	return ok && raiz.Name == "cfg"
}

// registroComunicacionesPrueba exige que el bloque construya el perfil fijo
// de comunicaciones y lo registre en el soporte deteniendo el alta si falla.
func registroComunicacionesPrueba(cuerpo *ast.BlockStmt) bool {
	var variable string
	registrado := false
	for _, sentencia := range cuerpo.List {
		switch x := sentencia.(type) {
		case *ast.AssignStmt:
			if len(x.Rhs) == 1 && len(x.Lhs) == 2 {
				if llamada, ok := x.Rhs[0].(*ast.CallExpr); ok &&
					nombreLlamadaPrueba(llamada.Fun) == "nuevoPerfilFijoConsultaComunicacionesExpedienteDesarrollo" {
					if id, ok := x.Lhs[0].(*ast.Ident); ok {
						variable = id.Name
					}
				}
			}
		case *ast.IfStmt:
			ast.Inspect(x.Cond, func(n ast.Node) bool {
				llamada, ok := n.(*ast.CallExpr)
				if ok && nombreLlamadaPrueba(llamada.Fun) == "registrarPerfilFijoCTDesarrollo" && len(llamada.Args) == 1 {
					if id, ok := llamada.Args[0].(*ast.Ident); ok && variable != "" && id.Name == variable {
						if len(x.Body.List) == 1 {
							_, registrado = x.Body.List[0].(*ast.ReturnStmt)
						}
					}
				}
				return true
			})
		}
	}
	return registrado
}
