package adminperfiles

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// falloSelector etiqueta un fallo del selector de perfil ADMIN con una clase
// de la lista cerrada. Error(), errors.Is y errors.As siguen siendo los de la
// causa: la clase sólo sirve al registro técnico del 503 y nunca llega al
// cliente ni lleva datos.
type falloSelector struct {
	clase string
	causa error
}

func (f falloSelector) Error() string { return f.causa.Error() }
func (f falloSelector) Unwrap() error { return f.causa }

// clasesFalloSelector es la lista cerrada: cada nombre dice qué comprobación
// falló, no por qué. Las clases SQL pueden llevar detrás el SQLSTATE (cinco
// caracteres [0-9A-Z]) del error de PostgreSQL, nunca su mensaje.
var clasesFalloSelector = map[string]struct{}{
	// Transacción común (postgres.go).
	"selector_sql_transaccion": {}, "selector_sql_configuracion": {}, "selector_sql_commit": {},
	// Consulta auditada (seleccion_auditada_postgres.go).
	"selector_correlacion": {}, "selector_aleatorio": {}, "selector_sql_conexion": {},
	"selector_auditoria_ref": {}, "selector_decodificacion": {}, "selector_denegacion_incoherente": {},
	"selector_motivo_desconocido": {}, "selector_estado_inesperado": {}, "selector_commit_incierto": {},
	"selector_carrera_agotada": {},
	// Validación de la respuesta SQL.
	"selector_lista_invalida": {}, "selector_perfil_distinto": {}, "selector_revision_fuera_de_rango": {},
	"selector_instante": {}, "selector_seleccion_futura": {}, "selector_campos_extra": {},
	// Servicio de aplicación y transporte HTTP.
	"selector_resultado_invalido": {}, "selector_sin_dependencias": {}, "selector_lista_respuesta": {},
	"selector_respuesta_incoherente": {}, "selector_observacion": {}, "selector_auditor_denegacion": {},
}

const prefijoSQLSelector = "selector_sql_"

// ConClaseSelector conserva la clase más interna que ya traiga err; si no
// trae ninguna, le pone clase. Una clase fuera de la lista no se añade.
func ConClaseSelector(clase string, err error) error {
	if err == nil || ClaseFalloSelector(err) != "" || !ClaseSelectorAdmitida(clase) {
		return err
	}
	return falloSelector{clase: clase, causa: err}
}

// conservarClaseSelector traslada la clase de origen a nuevo (por ejemplo,
// tras traducir el error con errorAutoridad, que devuelve un centinela).
func conservarClaseSelector(origen, nuevo error) error {
	if clase := ClaseFalloSelector(origen); clase != "" {
		return ConClaseSelector(clase, nuevo)
	}
	return nuevo
}

// claseSQLSelector compone «selector_sql_<SQLSTATE>» para un error de
// PostgreSQL y «selector_sql_conexion» para cualquier otro error del driver.
func claseSQLSelector(err error) string {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && sqlstateSelectorValido(pg.Code) {
		return prefijoSQLSelector + pg.Code
	}
	return "selector_sql_conexion"
}

// ClaseSelectorAdmitida dice si clase está en la lista cerrada o es una clase
// SQL seguida de un SQLSTATE con formato válido.
func ClaseSelectorAdmitida(clase string) bool {
	if _, ok := clasesFalloSelector[clase]; ok {
		return true
	}
	return len(clase) == len(prefijoSQLSelector)+5 && clase[:len(prefijoSQLSelector)] == prefijoSQLSelector &&
		sqlstateSelectorValido(clase[len(prefijoSQLSelector):])
}

func sqlstateSelectorValido(codigo string) bool {
	if len(codigo) != 5 {
		return false
	}
	for i := 0; i < len(codigo); i++ {
		if c := codigo[i]; (c < '0' || c > '9') && (c < 'A' || c > 'Z') {
			return false
		}
	}
	return true
}

// ClaseFalloSelector devuelve la clase cerrada de err o "" si no la lleva.
// Nunca devuelve la causa.
func ClaseFalloSelector(err error) string {
	var f falloSelector
	if !errors.As(err, &f) || !ClaseSelectorAdmitida(f.clase) {
		return ""
	}
	return f.clase
}
