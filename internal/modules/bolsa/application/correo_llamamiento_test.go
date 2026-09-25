package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type personalizacionPrueba struct {
	datos     map[string]puertosbolsa.DatosPersonalizacionLlamamiento
	err       error
	consultas int
}

func (p *personalizacionPrueba) DatosPersonalizacionLlamamiento(_ context.Context, _ string, refs []string) (map[string]puertosbolsa.DatosPersonalizacionLlamamiento, error) {
	p.consultas++
	if p.err != nil {
		return nil, p.err
	}
	out := map[string]puertosbolsa.DatosPersonalizacionLlamamiento{}
	for _, r := range refs {
		if d, ok := p.datos[r]; ok {
			out[r] = d
		}
	}
	return out, nil
}

type huellasPrueba struct{}

func (huellasPrueba) RegistrarHuellasCuerpo(context.Context, string, string, string, []byte, []puertosbolsa.HuellaCuerpoContacto) error {
	return nil
}

func servicioCorreoPrueba(t *testing.T, fuente *personalizacionPrueba) *ServicioEmisionLlamamiento {
	t.Helper()
	datos, err := os.ReadFile("../../../../config/bolsa_correo_llamamiento_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := dominiobolsa.CatalogoCorreoLlamamientoDesdeJSON(datos)
	if err != nil {
		t.Fatal(err)
	}
	return &ServicioEmisionLlamamiento{contextoBolsa: contextoContactoPrueba{}, reloj: func() time.Time { return time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC) }, correo: CorreoPersonalizadoLlamamiento{Catalogo: catalogo, Personalizacion: fuente, Huellas: huellasPrueba{}}}
}

func configuracionCorreoPrueba() puertosbolsa.ConfiguracionLlamamiento {
	return puertosbolsa.ConfiguracionLlamamiento{Referencia: "NEC-1", Descripcion: "Cobertura", Categoria: "Auxiliar", Centro: "Centro", Modalidad: "Sustitución",
		FechaInicio: "2026-10-01", Plazo: "48 horas", PlantillaVersion: "bolsa-llamamiento-v2", Asunto: "Llamamiento {bolsa}", Cuerpo: "Hola {nombre} {apellidos}, posición {posicion}. Plazo: {plazo}"}
}

func TestComponerCorreosPersonalizaCadaDestinatario(t *testing.T) {
	fuente := &personalizacionPrueba{datos: map[string]puertosbolsa.DatosPersonalizacionLlamamiento{
		"p:1": {Nombre: "Ana", Apellidos: "Ruiz", Posicion: 1, Bolsa: "Auxiliar"},
		"p:2": {Nombre: "Luis", Apellidos: "Mora", Posicion: 2, Bolsa: "Auxiliar"},
	}}
	s := servicioCorreoPrueba(t, fuente)
	correos, err := s.componerCorreos(context.Background(), "bolsa:1", []string{"p:1", "p:2"}, configuracionCorreoPrueba(), s.reloj())
	if err != nil {
		t.Fatal(err)
	}
	if fuente.consultas != 1 || correos[0].Cuerpo != "Hola Ana Ruiz, posición 1. Plazo: 48 horas" || correos[1].Cuerpo != "Hola Luis Mora, posición 2. Plazo: 48 horas" || correos[1].Asunto != "Llamamiento Auxiliar" {
		t.Fatalf("correos = %#v (consultas %d)", correos, fuente.consultas)
	}
	huellas := huellasCorreos([]string{"p:1", "p:2"}, correos)
	suma := sha256.Sum256([]byte(correos[1].Cuerpo))
	if huellas[1].ParticipacionRef != "p:2" || huellas[1].HuellaCuerpoSHA256 != hex.EncodeToString(suma[:]) || huellas[0].HuellaCuerpoSHA256 == huellas[1].HuellaCuerpoSHA256 || huellas[0].Caracteres != 42 {
		t.Fatalf("huellas = %#v", huellas)
	}
}

func TestComponerCorreosSinMarcadoresPersonalesNoConsultaPersonas(t *testing.T) {
	fuente := &personalizacionPrueba{err: errors.New("no debe consultarse")}
	s := servicioCorreoPrueba(t, fuente)
	c := configuracionCorreoPrueba()
	c.Asunto, c.Cuerpo = "Aviso {referencia}", "Plazo {plazo}"
	correos, err := s.componerCorreos(context.Background(), "bolsa:1", []string{"p:1"}, c, s.reloj())
	if err != nil || fuente.consultas != 0 || correos[0].Asunto != "Aviso NEC-1" {
		t.Fatalf("correos = %#v, err = %v, consultas = %d", correos, err, fuente.consultas)
	}
}

func TestComponerCorreosRechazaSinEfectos(t *testing.T) {
	fuente := &personalizacionPrueba{datos: map[string]puertosbolsa.DatosPersonalizacionLlamamiento{"p:1": {Nombre: "Ana", Apellidos: "Ruiz", Posicion: 1, Bolsa: "Auxiliar"}}}
	s := servicioCorreoPrueba(t, fuente)
	if _, err := s.componerCorreos(context.Background(), "bolsa:1", []string{"p:1", "ajena"}, configuracionCorreoPrueba(), s.reloj()); !errors.Is(err, puertosbolsa.ErrEmisionLlamamientoInvalida) || !errors.Is(err, dominiobolsa.ErrDatosCorreoLlamamientoIncompletos) {
		t.Fatalf("participación ajena: %v", err)
	}
	fuente.err = puertosbolsa.ErrPersonalizacionLlamamientoNoDisponible
	if _, err := s.componerCorreos(context.Background(), "bolsa:1", []string{"p:1"}, configuracionCorreoPrueba(), s.reloj()); !errors.Is(err, puertosbolsa.ErrEmisionLlamamientoNoDisponible) {
		t.Fatalf("fuente caída: %v", err)
	}
	c := configuracionCorreoPrueba()
	c.Cuerpo = "Hola {dni}"
	if err := s.validarConfiguracion(c); !errors.Is(err, dominiobolsa.ErrPlantillaCorreoLlamamiento) {
		t.Fatalf("marcador desconocido: %v", err)
	}
	c = configuracionCorreoPrueba()
	c.PlantillaVersion = "bolsa-llamamiento-v9"
	if err := s.validarConfiguracion(c); !errors.Is(err, puertosbolsa.ErrEmisionLlamamientoInvalida) {
		t.Fatalf("versión ajena al catálogo: %v", err)
	}
}

func TestVistaPreviaYPlantillaCorreo(t *testing.T) {
	fuente := &personalizacionPrueba{datos: map[string]puertosbolsa.DatosPersonalizacionLlamamiento{"p:1": {Nombre: "Ana", Apellidos: "Ruiz", Posicion: 3, Bolsa: "Auxiliar"}}}
	s := servicioCorreoPrueba(t, fuente)
	actor := dominiovec.ContextoActor{PersonaRef: "per_prueba"}
	vista, err := s.VistaPreviaLlamamiento(context.Background(), puertosbolsa.SolicitudVistaPreviaLlamamiento{ContextoActor: actor, BolsaRef: "bolsa:1", ParticipacionRef: "p:1", Configuracion: configuracionCorreoPrueba()})
	if err != nil || vista.Cuerpo != "Hola Ana Ruiz, posición 3. Plazo: 48 horas" || vista.Limite != 4000 || vista.Caracteres != 42 {
		t.Fatalf("vista = %#v, err = %v", vista, err)
	}
	s.contextoBolsa = contextoContactoPrueba{err: dominiovec.ErrAutorizacionDenegada}
	if _, err := s.VistaPreviaLlamamiento(context.Background(), puertosbolsa.SolicitudVistaPreviaLlamamiento{ContextoActor: actor, BolsaRef: "bolsa:1", ParticipacionRef: "p:1", Configuracion: configuracionCorreoPrueba()}); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("sin ámbito de bolsa la vista previa se deniega: %v", err)
	}
	plantilla, err := s.PlantillaCorreoLlamamiento(context.Background(), actor, "es")
	if err != nil || plantilla.PlantillaVersion != "bolsa-llamamiento-v2" || !plantilla.Personalizada || len(plantilla.Marcadores) != 12 || plantilla.Cuerpo == "" {
		t.Fatalf("plantilla = %#v, err = %v", plantilla, err)
	}
	if _, err := s.PlantillaCorreoLlamamiento(context.Background(), dominiovec.ContextoActor{}, "es"); err == nil {
		t.Fatal("sin actor no se entrega la plantilla")
	}
}
