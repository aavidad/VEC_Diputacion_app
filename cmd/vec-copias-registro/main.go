// vec-copias-registro records synthetic declared backup progress outside restored roots.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"time"

	fsregistro "vec-diputacion-granada/internal/modules/administracion/adapters/registrocopias"
	app "vec-diputacion-granada/internal/modules/administracion/application/registrocopias"
	"vec-diputacion-granada/internal/modules/administracion/domain/operacionescopias"
	port "vec-diputacion-granada/internal/modules/administracion/ports/registrocopias"
	"vec-diputacion-granada/internal/shared/i18n"
)

type entrada struct {
	Consulta    *port.Consulta               `json:"consulta,omitempty"`
	Sintetica   bool                         `json:"sintetica"`
	Declaracion port.Declaracion             `json:"declaracion"`
	Operacion   string                       `json:"operacion,omitempty"`
	Solicitud   *operacionescopias.Solicitud `json:"solicitud,omitempty"`
	Comando     *operacionescopias.Comando   `json:"comando,omitempty"`
}

var claves = []string{"argumentos_invalidos", "catalogo_invalido", "entrada_invalida", "salida_fallida", "aviso_registro", "listado_registrado", "registro_configuracion_invalida", "registro_entrada_invalida", "registro_historia_incompleta", "registro_no_existe", "registro_no_disponible", "registro_destino_ocupado", "revisar_abandono_confirmado", "operacion_abandono_no_confirmado", "registro_abandono_no_autorizado", "operacion_entrada_invalida", "operacion_conflicto_idempotencia", "operacion_conflicto_version", "operacion_vinculo_distinto", "operacion_transicion_invalida", "operacion_historia_invalida", "revalidar_antes_de_captura", "conciliar_captura_sin_repetir", "revalidar_antes_de_ensayos", "conciliar_ensayos_pendientes", "autenticar_evidencias_antes_de_uso", "revisar_ensayo_fallido", "historia_invalida"}

func main() {
	timer := time.AfterFunc(30*time.Second, func() { os.Exit(2) })
	code := ejecutar(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	timer.Stop()
	os.Exit(code)
}

func ejecutar(args []string, in io.Reader, out, diagnostico io.Writer) int {
	f := flag.NewFlagSet("vec-copias-registro", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	config := f.String("config", "", "")
	textos := f.String("textos", "", "")
	idioma := f.String("idioma", "", "")
	accion := f.String("accion", "", "")
	if f.Parse(args) != nil || f.NArg() != 0 || *config == "" || *textos == "" || *idioma == "" {
		return fallo(diagnostico, nil, *idioma, "argumentos_invalidos", port.Resultado{})
	}
	var mensajes map[string]string
	if leerArchivo(*textos, &mensajes) != nil {
		return fallo(diagnostico, nil, *idioma, "catalogo_invalido", port.Resultado{})
	}
	for _, k := range claves {
		if mensajes[k] == "" {
			return fallo(diagnostico, nil, *idioma, "catalogo_invalido", port.Resultado{})
		}
	}
	c, err := i18n.New(*idioma, map[string]map[string]string{*idioma: mensajes})
	if err != nil {
		return fallo(diagnostico, nil, *idioma, "catalogo_invalido", port.Resultado{})
	}
	var cfg fsregistro.Config
	if leerArchivo(*config, &cfg) != nil {
		return fallo(diagnostico, c, *idioma, "registro_configuracion_invalida", port.Resultado{})
	}
	var e entrada
	if decodificar(in, &e) != nil || !e.Sintetica {
		return fallo(diagnostico, c, *idioma, "entrada_invalida", port.Resultado{})
	}
	valida := false
	if *accion != "listar" && e.Consulta != nil {
		return fallo(diagnostico, c, *idioma, "entrada_invalida", port.Resultado{})
	}
	switch *accion {
	case "reservar":
		valida = e.Solicitud != nil && e.Operacion == "" && e.Comando == nil
	case "aplicar":
		valida = e.Solicitud == nil && e.Operacion != "" && e.Comando != nil
	case "listar":
		valida = e.Solicitud == nil && e.Comando == nil && e.Operacion == "" && e.Consulta != nil
	case "consultar":
		valida = e.Solicitud == nil && e.Operacion != "" && e.Comando == nil
	}
	if !valida {
		return fallo(diagnostico, c, *idioma, "entrada_invalida", port.Resultado{})
	}
	registro, err := fsregistro.Abrir(cfg)
	if err != nil {
		return fallo(diagnostico, c, *idioma, fsregistro.Codigo(err), port.Resultado{})
	}
	s := app.Servicio{Registro: registro}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var res port.Resultado
	switch *accion {
	case "reservar":
		res, err = s.Reservar(ctx, e.Declaracion, *e.Solicitud)
	case "aplicar":
		res, err = s.Aplicar(ctx, e.Declaracion, e.Operacion, *e.Comando)
	case "listar":
		res, err = s.Listar(ctx, e.Declaracion, *e.Consulta)
	case "consultar":
		res, err = s.Consultar(ctx, e.Declaracion, e.Operacion)
	}
	if err != nil {
		return fallo(diagnostico, c, *idioma, fsregistro.Codigo(err), res)
	}
	mensaje := res.Reconciliacion
	if *accion == "listar" {
		mensaje = "listado_registrado"
	}
	salida := struct {
		Resultado            port.Resultado `json:"resultado"`
		Alcance              string         `json:"alcance"`
		Autenticidad         string         `json:"autenticidad"`
		HabilitaCopia        bool           `json:"habilita_copia"`
		HabilitaRestauracion bool           `json:"habilita_restauracion"`
		RegistroDurable      bool           `json:"registro_durable"`
		Mensaje              string         `json:"mensaje"`
		Aviso                string         `json:"aviso"`
	}{Resultado: res, Alcance: "registro_offline_sintetico", Autenticidad: "no_comprobada", RegistroDurable: true, Mensaje: c.T(*idioma, mensaje), Aviso: c.T(*idioma, "aviso_registro")}
	if json.NewEncoder(out).Encode(salida) != nil {
		return fallo(diagnostico, c, *idioma, "salida_fallida", res)
	}
	return 0
}

func leerArchivo(ruta string, v any) error {
	f, err := os.Open(ruta) // #nosec G304 -- Explicit CLI operator file, bounded below; never HTTP input.
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	i, err := f.Stat()
	if err != nil || !i.Mode().IsRegular() || i.Size() > maxBytes {
		return errors.New("entrada_invalida")
	}
	return decodificar(f, v)
}

func fallo(out io.Writer, c *i18n.Catalog, idioma, key string, res port.Resultado) int {
	r := struct {
		Error     string          `json:"error"`
		Mensaje   string          `json:"mensaje,omitempty"`
		Auditoria *port.Auditoria `json:"auditoria,omitempty"`
	}{Error: key}
	if c != nil {
		r.Mensaje = c.T(idioma, key)
	}
	if res.Auditoria.Referencia != "" {
		r.Auditoria = &res.Auditoria
	}
	_ = json.NewEncoder(out).Encode(r)
	return 2
}
