package application

import (
	"context"
	"errors"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const correoRefPrueba = "correo:0123456789abcdef0123456789abcdef"

type candidatosAvisosPrueba struct {
	candidatos map[string]string
	err        error
}

func (c candidatosAvisosPrueba) CandidatoParticipacionAvisos(_ context.Context, bolsa, llamamiento, p string) (string, error) {
	if bolsa != "bolsa:encargado:2026-09-17" || llamamiento != "llamamiento:ab" {
		return "", errors.New("la consulta debe llevar la bolsa y el llamamiento del aviso")
	}
	return c.candidatos[p], c.err
}

// fuenteAvisosPrueba imita a Usuarios: si hay correo para el candidato llama
// a usar y devuelve su referencia.
type fuenteAvisosPrueba struct {
	correos  map[string]string
	err      error
	esperar  time.Duration
	pedidas  []puertosbolsa.SolicitudCorreoAvisoPersona
	llamadas int
}

func (f *fuenteAvisosPrueba) ConCorreoAvisoPersona(ctx context.Context, s puertosbolsa.SolicitudCorreoAvisoPersona, usar func(string)) (bool, string, error) {
	f.llamadas++
	f.pedidas = append(f.pedidas, s)
	if f.esperar > 0 {
		select {
		case <-ctx.Done():
			return false, "", ctx.Err()
		case <-time.After(f.esperar):
		}
	}
	if f.err != nil {
		return false, "", f.err
	}
	destino, ok := f.correos[s.CandidatoRef]
	if !ok {
		return false, "", nil
	}
	usar(destino)
	return true, correoRefPrueba, nil
}

type emisorAnotadoPrueba struct {
	destinos []string
	acepta   bool
}

func (e *emisorAnotadoPrueba) EnviarCorreo(_ context.Context, destino, _, _, _ string, _ time.Time) bool {
	e.destinos = append(e.destinos, destino)
	return e.acepta
}

type correoAltaPrueba struct{ err error }

func (c correoAltaPrueba) CorreoParticipacion(context.Context, string) (string, error) {
	return "alta@ejemplo.es", c.err
}

func servicioAvisosPrueba(emisor *emisorAnotadoPrueba, alta correoAltaPrueba) *AvisadorLlamamiento {
	a, err := NuevoAvisadorLlamamiento(alta, emisor)
	if err != nil {
		panic(err)
	}
	return a
}

func avisoPrueba(participacion string) AvisoLlamamiento {
	return AvisoLlamamiento{
		BolsaRef: "bolsa:encargado:2026-09-17", UnidadRef: "unidad:rrhh", AmbitoRef: "ambito:bolsa",
		LlamamientoRef: "llamamiento:" + "ab", ParticipacionRef: participacion,
		Asunto: "Aviso", Cuerpo: "Contenido", MessageID: "<id@vec>", Instante: time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC),
	}
}

func TestAvisoSinB59UsaElAltaSinConstancia(t *testing.T) {
	emisor := &emisorAnotadoPrueba{acepta: true}
	s := servicioAvisosPrueba(emisor, correoAltaPrueba{})
	resultado, fuente := s.Avisar(context.Background(), context.Background(), avisoPrueba("participacion:1"))
	if resultado != "enviado" || fuente != nil || len(emisor.destinos) != 1 || emisor.destinos[0] != "alta@ejemplo.es" {
		t.Fatalf("sin B59 debe comportarse como antes: %s %+v %v", resultado, fuente, emisor.destinos)
	}
}

func TestAvisoVaAlCorreoActivoDeMisCorreos(t *testing.T) {
	emisor := &emisorAnotadoPrueba{acepta: true}
	s := servicioAvisosPrueba(emisor, correoAltaPrueba{})
	fuente := &fuenteAvisosPrueba{correos: map[string]string{"can_candidata": "activo@personal.es"}}
	if err := s.EstablecerCorreoAvisosPersona(candidatosAvisosPrueba{candidatos: map[string]string{"participacion:1": "can_candidata"}}, fuente); err != nil {
		t.Fatal(err)
	}
	presupuesto, cancelar := s.Presupuesto(context.Background())
	defer cancelar()
	resultado, constancia := s.Avisar(context.Background(), presupuesto, avisoPrueba("participacion:1"))
	if resultado != "enviado" || len(emisor.destinos) != 1 || emisor.destinos[0] != "activo@personal.es" {
		t.Fatalf("el aviso debe ir sólo al correo activo: %s %v", resultado, emisor.destinos)
	}
	if constancia == nil || !constancia.Valida() || constancia.Fuente != puertosbolsa.FuenteCorreoMisCorreos || constancia.CorreoRef != correoRefPrueba {
		t.Fatalf("constancia: %+v", constancia)
	}
	p := fuente.pedidas[0]
	if p.CandidatoRef != "can_candidata" || p.BolsaRef != "bolsa:encargado:2026-09-17" || p.UnidadRef != "unidad:rrhh" || p.AmbitoRef != "ambito:bolsa" || p.LlamamientoRef != "llamamiento:ab" {
		t.Fatalf("solicitud a Usuarios: %+v", p)
	}
}

func TestAvisoSinCorreoActivoVaAlAlta(t *testing.T) {
	casos := []struct {
		nombre     string
		candidatos candidatosAvisosPrueba
		fuente     *fuenteAvisosPrueba
		motivo     string
		consultas  int
	}{
		{"sin correo activo", candidatosAvisosPrueba{candidatos: map[string]string{"participacion:1": "can_otra"}}, &fuenteAvisosPrueba{}, puertosbolsa.MotivoFuenteSinCorreoActivo, 1},
		{"sin persona vinculada", candidatosAvisosPrueba{candidatos: map[string]string{}}, &fuenteAvisosPrueba{}, puertosbolsa.MotivoFuenteSinPersonaVinculada, 0},
		{"vínculo ilegible", candidatosAvisosPrueba{err: errors.New("caída")}, &fuenteAvisosPrueba{}, puertosbolsa.MotivoFuenteMisCorreosNoDisponible, 0},
		{"Usuarios caído", candidatosAvisosPrueba{candidatos: map[string]string{"participacion:1": "can_x"}}, &fuenteAvisosPrueba{err: errors.New("caída")}, puertosbolsa.MotivoFuenteMisCorreosNoDisponible, 1},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			emisor := &emisorAnotadoPrueba{acepta: true}
			s := servicioAvisosPrueba(emisor, correoAltaPrueba{})
			if err := s.EstablecerCorreoAvisosPersona(c.candidatos, c.fuente); err != nil {
				t.Fatal(err)
			}
			resultado, constancia := s.Avisar(context.Background(), context.Background(), avisoPrueba("participacion:1"))
			if resultado != "enviado" || len(emisor.destinos) != 1 || emisor.destinos[0] != "alta@ejemplo.es" {
				t.Fatalf("el llamamiento no se bloquea y sale al alta: %s %v", resultado, emisor.destinos)
			}
			if constancia == nil || !constancia.Valida() || constancia.Fuente != puertosbolsa.FuenteCorreoAltaBolsa || constancia.Motivo != c.motivo || constancia.CorreoRef != "" {
				t.Fatalf("constancia: %+v", constancia)
			}
			if c.fuente.llamadas != c.consultas {
				t.Fatalf("consultas a Usuarios: %d", c.fuente.llamadas)
			}
		})
	}
}

func TestAvisoRechazadoPorElRelayNoSeRepiteAlAlta(t *testing.T) {
	emisor := &emisorAnotadoPrueba{acepta: false}
	s := servicioAvisosPrueba(emisor, correoAltaPrueba{})
	if err := s.EstablecerCorreoAvisosPersona(candidatosAvisosPrueba{candidatos: map[string]string{"participacion:1": "can_c"}}, &fuenteAvisosPrueba{correos: map[string]string{"can_c": "activo@personal.es"}}); err != nil {
		t.Fatal(err)
	}
	resultado, constancia := s.Avisar(context.Background(), context.Background(), avisoPrueba("participacion:1"))
	if resultado != "no_enviado" || len(emisor.destinos) != 1 || constancia == nil || constancia.Fuente != puertosbolsa.FuenteCorreoMisCorreos {
		t.Fatalf("un intento a «Mis correos» no se duplica al alta: %s %v %+v", resultado, emisor.destinos, constancia)
	}
}

func TestAvisoConUsuariosLentoAgotaElPresupuestoYSigue(t *testing.T) {
	emisor := &emisorAnotadoPrueba{acepta: true}
	s := servicioAvisosPrueba(emisor, correoAltaPrueba{})
	fuente := &fuenteAvisosPrueba{esperar: time.Hour}
	if err := s.EstablecerCorreoAvisosPersona(candidatosAvisosPrueba{candidatos: map[string]string{"participacion:1": "can_c", "participacion:2": "can_d"}}, fuente); err != nil {
		t.Fatal(err)
	}
	presupuesto, cancelar := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancelar()
	inicio := time.Now()
	for _, p := range []string{"participacion:1", "participacion:2"} {
		resultado, constancia := s.Avisar(context.Background(), presupuesto, avisoPrueba(p))
		if resultado != "enviado" || constancia == nil || constancia.Motivo != puertosbolsa.MotivoFuenteMisCorreosNoDisponible {
			t.Fatalf("%s: %s %+v", p, resultado, constancia)
		}
	}
	if time.Since(inicio) > limiteConsultaCorreoAvisos || fuente.llamadas != 1 {
		t.Fatalf("agotado el presupuesto no se consulta más: %v %d", time.Since(inicio), fuente.llamadas)
	}
	if len(emisor.destinos) != 2 || emisor.destinos[0] != "alta@ejemplo.es" || emisor.destinos[1] != "alta@ejemplo.es" {
		t.Fatalf("destinos: %v", emisor.destinos)
	}
}

func TestAvisoSinCorreoDelAltaQuedaNoEnviado(t *testing.T) {
	emisor := &emisorAnotadoPrueba{acepta: true}
	s := servicioAvisosPrueba(emisor, correoAltaPrueba{err: errors.New("sin datos")})
	if err := s.EstablecerCorreoAvisosPersona(candidatosAvisosPrueba{candidatos: map[string]string{}}, &fuenteAvisosPrueba{}); err != nil {
		t.Fatal(err)
	}
	resultado, constancia := s.Avisar(context.Background(), context.Background(), avisoPrueba("participacion:1"))
	if resultado != "no_enviado" || len(emisor.destinos) != 0 || constancia == nil || constancia.Motivo != puertosbolsa.MotivoFuenteSinPersonaVinculada {
		t.Fatalf("%s %v %+v", resultado, emisor.destinos, constancia)
	}
}

func TestEstablecerCorreoAvisosRechazaNulos(t *testing.T) {
	if _, err := NuevoAvisadorLlamamiento(nil, &emisorAnotadoPrueba{}); err == nil {
		t.Fatal("avisador sin correo del alta admitido")
	}
	if (&ServicioEmisionLlamamiento{}).EstablecerCorreoAvisosPersona(candidatosAvisosPrueba{}, &fuenteAvisosPrueba{}) == nil {
		t.Fatal("servicio sin avisador admitido")
	}
	s := servicioAvisosPrueba(&emisorAnotadoPrueba{}, correoAltaPrueba{})
	if s.EstablecerCorreoAvisosPersona(nil, &fuenteAvisosPrueba{}) == nil || s.EstablecerCorreoAvisosPersona(candidatosAvisosPrueba{}, nil) == nil {
		t.Fatal("dependencias nulas admitidas")
	}
	var fuenteNula *fuenteAvisosPrueba
	if s.EstablecerCorreoAvisosPersona(candidatosAvisosPrueba{}, fuenteNula) == nil {
		t.Fatal("puntero nulo tipado admitido")
	}
}

func TestFuenteCorreoContactoValida(t *testing.T) {
	validas := []puertosbolsa.FuenteCorreoContacto{
		{Fuente: puertosbolsa.FuenteCorreoMisCorreos, Motivo: puertosbolsa.MotivoFuenteCorreoActivo, CorreoRef: correoRefPrueba},
		{Fuente: puertosbolsa.FuenteCorreoAltaBolsa, Motivo: puertosbolsa.MotivoFuenteSinCorreoActivo},
		{Fuente: puertosbolsa.FuenteCorreoAltaBolsa, Motivo: puertosbolsa.MotivoFuenteSinPersonaVinculada},
		{Fuente: puertosbolsa.FuenteCorreoAltaBolsa, Motivo: puertosbolsa.MotivoFuenteMisCorreosNoDisponible},
	}
	for _, f := range validas {
		if !f.Valida() {
			t.Fatalf("debería ser válida: %+v", f)
		}
	}
	invalidas := []puertosbolsa.FuenteCorreoContacto{
		{Fuente: puertosbolsa.FuenteCorreoMisCorreos, Motivo: puertosbolsa.MotivoFuenteCorreoActivo},
		{Fuente: puertosbolsa.FuenteCorreoMisCorreos, Motivo: puertosbolsa.MotivoFuenteCorreoActivo, CorreoRef: "correo:XYZ"},
		{Fuente: puertosbolsa.FuenteCorreoAltaBolsa, Motivo: puertosbolsa.MotivoFuenteCorreoActivo},
		{Fuente: puertosbolsa.FuenteCorreoAltaBolsa, Motivo: puertosbolsa.MotivoFuenteSinCorreoActivo, CorreoRef: correoRefPrueba},
		{Fuente: "otra", Motivo: puertosbolsa.MotivoFuenteSinCorreoActivo},
	}
	for _, f := range invalidas {
		if f.Valida() {
			t.Fatalf("no debería ser válida: %+v", f)
		}
	}
}
