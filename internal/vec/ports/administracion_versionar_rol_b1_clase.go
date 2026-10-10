package ports

import "errors"

// falloVersionBolsa etiqueta un fallo de la ruta B1 con una clase de la lista
// cerrada. Error() y errors.Is siguen siendo los de la causa: la clase sólo
// sirve al registro técnico y nunca llega al cliente ni lleva datos.
type falloVersionBolsa struct {
	clase string
	causa error
}

func (f falloVersionBolsa) Error() string { return f.causa.Error() }
func (f falloVersionBolsa) Unwrap() error { return f.causa }

// clasesFalloVersionBolsa es la lista cerrada. Cada nombre dice qué
// comprobación falló, no por qué: sin referencias ni huellas. La única
// excepción son las clases del intento SQL auditado, que pueden llevar detrás
// el código SQLSTATE (cinco caracteres [0-9A-Z]; ver ClaseIntentoSQLVersionBolsa).
var clasesFalloVersionBolsa = map[string]struct{}{
	// Servicio de aplicación.
	"servicio_no_configurado": {}, "solicitud_invalida": {}, "evidencia_sesion": {},
	"autoridad_sin_puerto": {}, "administrador_instantanea": {}, "administrador_rol": {},
	"plan_servicio": {}, "orden_invalida": {}, "resultado_invalido": {},
	// Fuente del catálogo y rol administrable.
	"catalogo_consulta": {}, "catalogo_canon": {}, "catalogo_paquete": {}, "catalogo_plan_huella": {},
	"rol_administrable_version": {}, "rol_administrable_consulta": {}, "rol_administrable_respuesta": {},
	"rol_administrable_incoherente": {},
	// Adaptador PostgreSQL B1.
	"autoridad_no_disponible": {}, "material_propuesta": {}, "evidencia_o_correlacion": {},
	"recurso_asignacion": {}, "v3_material_incoherente": {}, "sql_transaccion": {},
	"sql_configuracion": {}, "sql_consulta": {}, "sql_intento_incoherente": {}, "sql_intento_error": {},
	"sql_intento_denegado": {}, "sql_respuesta_invalida": {}, "sql_commit": {},
	// Emisor V3 B1.
	"v3_dependencia": {}, "v3_actor_evidencia": {}, "v3_recurso": {}, "v3_vinculo": {},
	"v3_vinculo_privilegiado": {}, "v3_instantanea": {}, "v3_ambito": {}, "v3_sin_concesion": {},
	"v3_correlacion": {}, "v3_solicitud": {}, "v3_emision": {}, "v3_decision": {},
	"v3_exportacion": {}, "v3_material": {},
}

// ConClaseVersionBolsa conserva la clase más interna que ya traiga err; si no
// trae ninguna, le pone clase. Una clase fuera de la lista no se añade.
func ConClaseVersionBolsa(clase string, err error) error {
	if err == nil || ClaseFalloVersionBolsa(err) != "" {
		return err
	}
	if !ClaseVersionBolsaAdmitida(clase) {
		return err
	}
	return falloVersionBolsa{clase: clase, causa: err}
}

// ClaseVersionBolsaAdmitida dice si clase pertenece a la lista cerrada o es
// una clase de intento SQL seguida de un SQLSTATE con formato válido.
func ClaseVersionBolsaAdmitida(clase string) bool {
	if _, ok := clasesFalloVersionBolsa[clase]; ok {
		return true
	}
	for _, base := range basesIntentoSQLVersionBolsa {
		if len(clase) == len(base)+1+5 && clase[:len(base)+1] == base+"_" && SQLSTATEValido(clase[len(base)+1:]) {
			return true
		}
	}
	return false
}

// basesIntentoSQLVersionBolsa son las clases que admiten sufijo SQLSTATE.
var basesIntentoSQLVersionBolsa = [...]string{"sql_intento_error", "sql_intento_denegado"}

// SQLSTATEValido sólo admite el formato de un código SQLSTATE: cinco
// caracteres entre 0-9 y A-Z. No valida que el código exista.
func SQLSTATEValido(codigo string) bool {
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

// ClaseIntentoSQLVersionBolsa compone la clase de un intento B1 auditado por
// AUT63: «sql_intento_<estado>_<sqlstate>» si AUT72 devolvió un código con
// formato válido y «sql_intento_<estado>» si no lo trae (base sin AUT72) o no
// lo tiene. Todo estado distinto de "denegado" cuenta como "error".
func ClaseIntentoSQLVersionBolsa(estado, sqlstate string) string {
	base := "sql_intento_error"
	if estado == "denegado" {
		base = "sql_intento_denegado"
	}
	if SQLSTATEValido(sqlstate) {
		return base + "_" + sqlstate
	}
	return base
}

// ClaseFalloVersionBolsa devuelve la clase cerrada de err o "" si no la lleva.
// Nunca devuelve la causa.
func ClaseFalloVersionBolsa(err error) string {
	var f falloVersionBolsa
	if !errors.As(err, &f) || !ClaseVersionBolsaAdmitida(f.clase) {
		return ""
	}
	return f.clase
}
