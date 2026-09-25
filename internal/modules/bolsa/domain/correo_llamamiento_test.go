package domain

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func catalogoCorreoPrueba(t *testing.T) CatalogoCorreoLlamamiento {
	t.Helper()
	datos, err := os.ReadFile("../../../../config/bolsa_correo_llamamiento_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	c, err := CatalogoCorreoLlamamientoDesdeJSON(datos)
	if err != nil {
		t.Fatalf("el catálogo versionado debe ser válido: %v", err)
	}
	return c
}

func TestCatalogoCorreoVersionadoEsCoherente(t *testing.T) {
	c := catalogoCorreoPrueba(t)
	if c.PlantillaVigente() != "bolsa-llamamiento-v2" || c.LimiteCaracteres() != 4000 || c.IdiomaDefecto() != "es" {
		t.Fatalf("catálogo inesperado: %s %d %s", c.PlantillaVigente(), c.LimiteCaracteres(), c.IdiomaDefecto())
	}
	if p, ok := c.PlantillaAdmitida("bolsa-llamamiento-v1"); !ok || p {
		t.Fatal("v1 debe seguir admitida como plantilla literal")
	}
	if len(c.Marcadores()) != 12 || c.TextosDefecto("xx").Cuerpo == "" {
		t.Fatal("marcadores o textos por defecto ausentes")
	}
	m := c.Marcadores()
	m[0].Clave = "alterado"
	if c.Marcadores()[0].Clave == "alterado" {
		t.Fatal("los marcadores deben devolverse por copia")
	}
}

func TestCatalogoCorreoRechazaDefectos(t *testing.T) {
	base, err := os.ReadFile("../../../../config/bolsa_correo_llamamiento_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	casos := map[string][2]string{
		"campo desconocido":       {`"version": 1,`, `"version": 1, "extra": true,`},
		"esquema":                 {`vec.bolsa.catalogo_correo_llamamiento.v1`, `otro`},
		"vigente inexistente":     {`"plantilla_vigente": "bolsa-llamamiento-v2"`, `"plantilla_vigente": "bolsa-llamamiento-v9"`},
		"version mal formada":     {`{ "version": "bolsa-llamamiento-v1"`, `{ "version": "llamamiento-v1"`},
		"limite bajo":             {`"limite_caracteres": 4000`, `"limite_caracteres": 10`},
		"formato fecha":           {`"dd/mm/aaaa"`, `"mm-dd"`},
		"zona":                    {`"Europe/Madrid"`, `"Marte/Olimpo"`},
		"fuente desconocida":      {`{ "clave": "nombre", "fuente": "nombre" }`, `{ "clave": "nombre", "fuente": "dni" }`},
		"clave repetida":          {`{ "clave": "apellidos", "fuente": "apellidos" }`, `{ "clave": "nombre", "fuente": "apellidos" }`},
		"clave mal formada":       {`{ "clave": "posicion", "fuente": "posicion" }`, `{ "clave": "Posición", "fuente": "posicion" }`},
		"defecto con desconocido": {`Llamamiento de la bolsa de {bolsa}`, `Llamamiento de {dni}`},
		"idioma sin textos":       {`"idioma_defecto": "es"`, `"idioma_defecto": "en"`},
	}
	for nombre, cambio := range casos {
		datos := strings.Replace(string(base), cambio[0], cambio[1], 1)
		if datos == string(base) {
			t.Fatalf("%s: el caso no altera el catálogo", nombre)
		}
		if _, err := CatalogoCorreoLlamamientoDesdeJSON([]byte(datos)); !errors.Is(err, ErrCatalogoCorreoLlamamientoInvalido) {
			t.Fatalf("%s: se esperaba rechazo, obtenido %v", nombre, err)
		}
	}
	if _, err := CatalogoCorreoLlamamientoDesdeJSON(append(append([]byte(nil), base...), []byte("{}")...)); err == nil {
		t.Fatal("dos documentos JSON deben rechazarse")
	}
}

func valoresCorreoPrueba() ValoresCorreoLlamamiento {
	return ValoresCorreoLlamamiento{Nombre: "Ana", Apellidos: "Pérez Gómez", Bolsa: "Auxiliar administrativo", Posicion: 7,
		Categoria: "C2", Referencia: "NEC-1", Centro: "Residencia", Modalidad: "Sustitución", FechaInicio: "2026-10-01",
		Plazo: "24 horas", FechaEnvio: time.Date(2026, 9, 30, 23, 30, 0, 0, time.UTC)}
}

func TestPersonalizarSustituyePorPersona(t *testing.T) {
	c := catalogoCorreoPrueba(t)
	out, err := c.Personalizar("bolsa-llamamiento-v2", "Bolsa {bolsa}: {nombre_completo}", "Hola {nombre} {apellidos}, posición {posicion}; {referencia} {centro} {modalidad} {categoria} inicio {fecha_inicio}, plazo {plazo}, enviado {fecha_envio}. {sin_cerrar {1} {}", valoresCorreoPrueba())
	if err != nil {
		t.Fatal(err)
	}
	if out.Asunto != "Bolsa Auxiliar administrativo: Ana Pérez Gómez" {
		t.Fatalf("asunto = %q", out.Asunto)
	}
	esperado := "Hola Ana Pérez Gómez, posición 7; NEC-1 Residencia Sustitución C2 inicio 01/10/2026, plazo 24 horas, enviado 01/10/2026. {sin_cerrar {1} {}"
	if out.Cuerpo != esperado {
		t.Fatalf("cuerpo = %q", out.Cuerpo)
	}
}

func TestPersonalizarLiteralNoSustituye(t *testing.T) {
	c := catalogoCorreoPrueba(t)
	out, err := c.Personalizar("bolsa-llamamiento-v1", "Aviso", "Texto con {nombre} literal", ValoresCorreoLlamamiento{})
	if err != nil || out.Cuerpo != "Texto con {nombre} literal" {
		t.Fatalf("v1 debe enviarse literal: %q %v", out.Cuerpo, err)
	}
	if c.RequiereDatosPersonales("bolsa-llamamiento-v1", "{nombre}") || c.ValidarPlantilla("bolsa-llamamiento-v1", "{dni}") != nil {
		t.Fatal("una plantilla literal no interpreta marcadores")
	}
}

func TestPersonalizarRechazos(t *testing.T) {
	c := catalogoCorreoPrueba(t)
	v := valoresCorreoPrueba()
	if _, err := c.Personalizar("bolsa-llamamiento-v2", "Aviso", "Hola {dni}", v); !errors.Is(err, ErrPlantillaCorreoLlamamiento) {
		t.Fatalf("marcador desconocido: %v", err)
	}
	if _, err := c.Personalizar("bolsa-llamamiento-v7", "Aviso", "Hola", v); !errors.Is(err, ErrPlantillaCorreoLlamamiento) {
		t.Fatalf("versión desconocida: %v", err)
	}
	sinPosicion := v
	sinPosicion.Posicion = 0
	if _, err := c.Personalizar("bolsa-llamamiento-v2", "Aviso", "Posición {posicion}", sinPosicion); !errors.Is(err, ErrDatosCorreoLlamamientoIncompletos) {
		t.Fatalf("dato ausente: %v", err)
	}
	largo := v
	largo.Apellidos = strings.Repeat("x", 3990)
	if _, err := c.Personalizar("bolsa-llamamiento-v2", "Aviso", "Hola {nombre} {apellidos} y más texto", largo); !errors.Is(err, ErrCorreoLlamamientoExcedeLimite) {
		t.Fatalf("el límite se aplica tras sustituir: %v", err)
	}
	if _, err := c.Personalizar("bolsa-llamamiento-v2", "Aviso {apellidos}", "Hola", largo); !errors.Is(err, ErrCorreoLlamamientoExcedeLimite) {
		t.Fatalf("el asunto también tiene límite: %v", err)
	}
	if _, err := c.Personalizar("bolsa-llamamiento-v1", "Aviso\r\nBcc: x", "Hola", v); !errors.Is(err, ErrCorreoLlamamientoExcedeLimite) {
		t.Fatalf("un asunto con salto de línea debe rechazarse: %v", err)
	}
}

func TestPersonalizarNeutralizaControles(t *testing.T) {
	c := catalogoCorreoPrueba(t)
	v := valoresCorreoPrueba()
	v.Nombre = "Ana\r\nBcc: intruso@example.org\t"
	out, err := c.Personalizar("bolsa-llamamiento-v2", "Para {nombre}", "Hola {nombre}", v)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(out.Asunto, "\r\n\t") || out.Cuerpo != "Hola Ana Bcc: intruso@example.org" {
		t.Fatalf("controles no neutralizados: %q / %q", out.Asunto, out.Cuerpo)
	}
	if !c.RequiereDatosPersonales("bolsa-llamamiento-v2", "Plazo {plazo}", "Hola {nombre}") || c.RequiereDatosPersonales("bolsa-llamamiento-v2", "Plazo {plazo}") {
		t.Fatal("solo los marcadores personales exigen consultar a la persona")
	}
}
